//go:build machinery_integration

package main

// MAC-hlae required-lane saga-closure CLI cases: a real isolated machinery
// binary against a real isolated copy of the fulfillment design, with the
// lane-provisioned pinned Java/TLC closure.
//
//   - Positive control: the unchanged design's refine, compose, and
//     verify-formal paths stay green (real TLC, 11 passed).
//   - Unsafe mutations (timeout-to-clean-Failed while cleanup may be
//     outstanding; compensation completion redirect) are rejected with a
//     precise nonzero diagnostic BEFORE any successful proof or publication.
//
// Missing pinned runtimes are hard failures, never skips: this file is a
// registered member of the required integration lane
// (testdata/integration-lanes/saga.json).

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/runtimeclosure"
)

func sagaClosureRepo(t *testing.T) string {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

// sagaClosureBinary builds one isolated candidate binary; nothing here touches
// any installed machinery asset.
func sagaClosureBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "machinery")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/machinery")
	cmd.Dir = sagaClosureRepo(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("isolated candidate build failed: %v %s", err, out)
	}
	return bin
}

// sagaClosureDesign copies the shared fulfillment design into private
// scratch; the shared example is never mutated.
func sagaClosureDesign(t *testing.T) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "design")
	if err := os.CopyFS(dest, os.DirFS(filepath.Join(sagaClosureRepo(t), "examples/fulfillment/design"))); err != nil {
		t.Fatalf("copy isolated fulfillment design: %v", err)
	}
	return dest
}

// sagaClosureMutateMachine applies one exact byte-level semantic mutation to
// the isolated machine copy and fails if the anchor bytes were not unique.
func sagaClosureMutateMachine(t *testing.T, design, old, new string) {
	t.Helper()
	path := filepath.Join(design, "machines", "FulfillmentSaga.machine.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(body), old); n != 1 {
		t.Fatalf("mutation anchor %q matched %d times in %s", old, n, path)
	}
	if err := os.WriteFile(path, bytes.ReplaceAll(body, []byte(old), []byte(new)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sagaClosureRun executes the isolated binary with exactly the lane-provisioned
// pinned Java/TLC closure; ambient engine overrides are never inherited.
func sagaClosureRun(t *testing.T, bin, workDir string, timeout time.Duration, args ...string) (int, string) {
	t.Helper()
	java := os.Getenv("MACHINERY_INTEGRATION_JAVA")
	jar := os.Getenv("MACHINERY_INTEGRATION_TLC_JAR")
	if java == "" || jar == "" {
		t.Fatal("pinned Java/TLC closure not provisioned (MACHINERY_INTEGRATION_JAVA / MACHINERY_INTEGRATION_TLC_JAR); the required lane provisions it")
	}
	digest, err := runtimeclosure.JavaClosureDigest(java)
	if err != nil {
		t.Fatalf("digest pinned Java closure: %v", err)
	}
	tmp := t.TempDir()
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		switch key {
		case "MACHINERY_JAVA", "MACHINERY_JAVA_CLOSURE_SHA256", "TLA_TOOLS_JAR", "TLA_TOOLS_JAR_SHA256", "JAVA_TOOL_OPTIONS", "JDK_JAVA_OPTIONS", "CLASSPATH", "TMPDIR":
			continue
		}
		env = append(env, item)
	}
	env = append(env,
		"MACHINERY_JAVA="+java,
		"MACHINERY_JAVA_CLOSURE_SHA256="+digest,
		"TLA_TOOLS_JAR="+jar,
		"TMPDIR="+tmp,
	)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = workDir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		exitErr, ok := runErr.(*exec.ExitError)
		if !ok {
			t.Fatalf("real CLI invocation failed to start: %v %s", runErr, out.String())
		}
		code = exitErr.ExitCode()
	}
	return code, out.String()
}

// TestSagaClosureCLIPositiveControlStaysGreen: the unchanged fulfillment
// design keeps its intended success / clean-failure / FailedDirty proofs on
// the real CLI path, including the real pinned TLC engine.
func TestSagaClosureCLIPositiveControlStaysGreen(t *testing.T) {
	bin := sagaClosureBinary(t)
	design := sagaClosureDesign(t)
	if code, out := sagaClosureRun(t, bin, design, 2*time.Minute,
		"refine", filepath.Join("machines", "FulfillmentSaga.machine.json"), filepath.Join("formal", "FulfillmentSaga.semantics.yaml"), filepath.Join("formal")); code != 0 {
		t.Fatalf("positive-control refine failed (%d): %s", code, out)
	}
	if code, out := sagaClosureRun(t, bin, design, 2*time.Minute,
		"compose", filepath.Join("formal", "checkout.composition.yaml"), filepath.Join("machines", "FulfillmentSaga.machine.json"), filepath.Join("formal")); code != 0 {
		t.Fatalf("positive-control compose failed (%d): %s", code, out)
	}
	code, out := sagaClosureRun(t, bin, design, 8*time.Minute, "verify-formal", design)
	if code != 0 {
		t.Fatalf("positive-control verify-formal failed (%d): %s", code, out)
	}
	if !strings.Contains(out, "11 passed, 0 failed") {
		t.Fatalf("positive control lost its intended proofs: %s", out)
	}
}

// TestSagaClosureCLIRejectsUnsafeCompensationRoutes: the assessed mutations
// must be rejected with a precise nonzero diagnostic before any successful
// proof or publication — refine, compose, and the verify-formal gate.
func TestSagaClosureCLIRejectsUnsafeCompensationRoutes(t *testing.T) {
	bin := sagaClosureBinary(t)

	// M1: timeout-to-clean-Failed while cleanup may be outstanding.
	design := sagaClosureDesign(t)
	sagaClosureMutateMachine(t, design,
		`"compensateTimeout": { "target": "compensateRetry"`,
		`"compensateTimeout": { "target": "Failed"`)
	if code, out := sagaClosureRun(t, bin, design, 2*time.Minute,
		"refine", filepath.Join("machines", "FulfillmentSaga.machine.json"), filepath.Join("formal", "FulfillmentSaga.semantics.yaml"), filepath.Join("formal")); code == 0 {
		t.Fatalf("refine accepted timeout-to-clean-Failed route (%d): %s", code, out)
	} else if !strings.Contains(out, "after:compensateTimeout") || !strings.Contains(out, "outstanding") {
		t.Fatalf("refine diagnostic lost the intended reason: %s", out)
	}
	code, out := sagaClosureRun(t, bin, design, 8*time.Minute, "verify-formal", design)
	if code == 0 || strings.Contains(out, "11 passed, 0 failed") {
		t.Fatalf("verify-formal proved the unsafe mutation green (%d): %s", code, out)
	}
	if !strings.Contains(out, "RECONCILIATION FAILED") && !strings.Contains(out, "VALIDATION FAILED") {
		t.Fatalf("verify-formal did not reject at generation (before proof): %s", out)
	}

	// M2: compensation completion redirect (Compensating onDone Failed ->
	// Completed), the compose-side assessed mutation.
	design2 := sagaClosureDesign(t)
	sagaClosureMutateMachine(t, design2,
		`"onDone": { "target": "Failed", "actions": "recordCompensated" }`,
		`"onDone": { "target": "Completed", "actions": "recordCompensated" }`)
	if code, out := sagaClosureRun(t, bin, design2, 2*time.Minute,
		"compose", filepath.Join("formal", "checkout.composition.yaml"), filepath.Join("machines", "FulfillmentSaga.machine.json"), filepath.Join("formal")); code == 0 {
		t.Fatalf("compose accepted compensation completion redirect (%d): %s", code, out)
	} else if !strings.Contains(out, "Compensating onDone must reach Failed") {
		t.Fatalf("compose diagnostic lost the intended reason: %s", out)
	}
	code, out = sagaClosureRun(t, bin, design2, 8*time.Minute, "verify-formal", design2)
	if code == 0 || strings.Contains(out, "11 passed, 0 failed") {
		t.Fatalf("verify-formal proved the completion redirect green (%d): %s", code, out)
	}
}
