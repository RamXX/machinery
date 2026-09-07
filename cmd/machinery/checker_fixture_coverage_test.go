package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These are protocol fixtures, not evidence of Docker lifecycle behavior.
// The inner selection deliberately excludes this compiling regression.
var checkerCoverageLeaves = []string{
	"TestVerifyCheckersReproducible",
	"TestCheckerFixtureProtocolControls",
	"TestRunCheckerBoundsOutput",
	"TestRunCheckerReportsStreamsInDeterministicOrder",
	"TestRunCheckerBoundsDescendantPipeWait",
	"TestRunCheckerTimeoutDiagnostic",
	"TestVerifyLocalOCIImageBoundsUnresponsiveEngine",
}

func TestCheckerFixtureCoverageRegression(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	defer cancel()
	started := time.Now()
	t.Logf("host=%s/%s toolchain=%s", runtime.GOOS, runtime.GOARCH, runtime.Version())
	for _, mode := range []string{"native", "covered"} {
		t.Run(mode, func(t *testing.T) {
			owned := t.TempDir()
			binary := filepath.Join(owned, "checker.test")
			buildArgs := []string{"test", "-c", "-o", binary}
			if mode == "covered" {
				buildArgs = append(buildArgs, "-cover")
			}
			buildArgs = append(buildArgs, ".")
			stdout, stderr, err := checkerCoverageCommand(t, ctx, nil, "go", buildArgs...)
			if err != nil {
				t.Fatalf("SETUP compiler failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
			}
			selector := "^(" + strings.Join(checkerCoverageLeaves, "|") + ")$"
			args := []string{"-test.v", "-test.count=1", "-test.timeout=60s", "-test.run=" + selector}
			profile := filepath.Join(owned, "coverage.out")
			if mode == "covered" {
				args = append(args, "-test.coverprofile="+profile)
			}
			// Own the golden harness's temporary build as well as this binary.
			env := environmentWithOverrides(os.Environ(), []string{"TMPDIR=" + owned})
			stdout, stderr, err = checkerCoverageCommand(t, ctx, env, binary, args...)
			t.Logf("%s child stdout:\n%s\n%s child stderr:\n%s", mode, stdout, mode, stderr)
			for _, leaf := range checkerCoverageLeaves {
				if !strings.Contains(stdout, "=== RUN   "+leaf+"\n") {
					t.Errorf("INVENTORY %s did not execute %s", mode, leaf)
				}
			}
			for _, leaf := range []string{"correct", "wrong-digest", "wrong-platform", "extra-data"} {
				if !strings.Contains(stdout, "=== RUN   TestCheckerFixtureProtocolControls/"+leaf+"\n") {
					t.Errorf("INVENTORY %s did not execute protocol control %s", mode, leaf)
				}
			}
			if strings.Contains(stdout, "--- SKIP:") {
				t.Error("INVENTORY selected subprocess tests must not skip")
			}
			if mode == "covered" {
				checkerCoverageProfile(t, ctx, profile)
			}
			if err != nil {
				t.Errorf("BEHAVIOR %s selected checker assertions failed: %v", mode, err)
			}
		})
	}
	t.Logf("paired regression elapsed=%s", time.Since(started))
}

// TestCheckerFixtureProtocolControls calls unchanged production verification
// after observing the actual helper's stdout and stderr independently. Errors
// on the raw observation do not prevent the strict parser diagnosis being run.
func TestCheckerFixtureProtocolControls(t *testing.T) {
	cases := []struct {
		name, mode, diagnosis string
		extra                 bool
	}{
		{name: "correct", mode: "oci-engine"},
		{name: "wrong-digest", mode: "oci-engine-wrong-digest", diagnosis: "do not contain exact reference"},
		{name: "wrong-platform", mode: "oci-engine-wrong-platform", diagnosis: "does not match required platform"},
		{name: "extra-data", mode: "oci-engine", diagnosis: "OCI RepoDigests response has trailing data", extra: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			work := t.TempDir()
			var engine []string
			if err := json.Unmarshal([]byte(checkerFixtureEngineArgs(t, tc.mode)), &engine); err != nil {
				t.Fatalf("SETUP existing engine arguments: %v", err)
			}
			if tc.extra {
				// Execute the real fixture first, propagate its exit status, then
				// add intentional fourth JSON data. This is malformed protocol
				// input, not a replacement compiler/process result.
				var quoted []string
				for _, arg := range engine {
					quoted = append(quoted, "'"+strings.ReplaceAll(arg, "'", "'\"'\"'")+"'")
				}
				script := writeScript(t, strings.Join(quoted, " ")+" \"$@\" || exit $?\nprintf '\"extra-protocol-data\"\\n'\n")
				engine = []string{script}
			}
			args := append(append([]string(nil), engine...), "image", "inspect", "--format", "{{json .RepoDigests}}\n{{json .Os}}\n{{json .Architecture}}", testRuntimeImage)
			env, err := deterministicCheckerEnv(work)
			if err != nil {
				t.Fatalf("SETUP deterministic environment: %v", err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, args[0], args[1:]...)
			cmd.Dir, cmd.Env = work, env
			cmd.WaitDelay = checkerWaitDelay
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("SETUP raw fixture execution: %v stdout=%q stderr=%q", err, stdout.String(), stderr.String())
			}
			t.Logf("RAW stdout(%d)=%q stderr(%d)=%q", stdout.Len(), stdout.String(), stderr.Len(), stderr.String())
			if stderr.Len() != 0 {
				t.Errorf("PROTOCOL incidental fixture stderr contaminates inspection: %q", stderr.String())
			}
			dec := json.NewDecoder(strings.NewReader(stdout.String()))
			var digests []string
			var imageOS, arch string
			for _, value := range []any{&digests, &imageOS, &arch} {
				if err := dec.Decode(value); err != nil {
					t.Fatalf("PROTOCOL missing inspection value: %v", err)
				}
			}
			wantImage, wantPlatform := testRuntimeImage, testRuntimePlatform
			if tc.name == "wrong-digest" {
				wantImage = strings.SplitN(testRuntimeImage, "@", 2)[0] + "@sha256:" + strings.Repeat("f", 64)
			}
			if tc.name == "wrong-platform" {
				wantPlatform = "linux/arm64"
			}
			if len(digests) != 1 || digests[0] != wantImage || imageOS+"/"+arch != wantPlatform {
				t.Fatalf("PROTOCOL fixture did not provide intended input: digests=%v platform=%s/%s", digests, imageOS, arch)
			}
			if tc.extra {
				var extra string
				if err := dec.Decode(&extra); err != nil || extra != "extra-protocol-data" {
					t.Fatalf("PROTOCOL deliberate extra input missing: %q %v", extra, err)
				}
			}
			var trailing any
			if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
				t.Errorf("PROTOCOL unexpected stdout suffix: %v", err)
			}
			// Verify production still reports streams in deterministic order.
			combined, runErr := runChecker(args, 5*time.Second, work)
			if runErr != nil {
				t.Fatalf("SETUP production runner: %v", runErr)
			}
			if combined != stdout.String()+stderr.String() {
				t.Errorf("PROTOCOL production stream ordering changed: %q", combined)
			}
			err = verifyLocalOCIImage(engine, testRuntimeImage, testRuntimeClosure, testRuntimePlatform, 5*time.Second, work)
			if tc.diagnosis == "" {
				if err != nil {
					t.Errorf("BEHAVIOR valid inspection rejected: %v", err)
				} else {
					t.Log("REACHED correct digest/platform accepted")
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.diagnosis) {
				t.Errorf("UNREACHED intended diagnosis %q; actual=%v", tc.diagnosis, err)
			} else {
				t.Logf("REACHED intended diagnosis %q", tc.diagnosis)
			}
		})
	}
}

func checkerCoverageCommand(t *testing.T, parent context.Context, env []string, name string, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	started := time.Now()
	err := cmd.Run()
	t.Logf("COMMAND %s %q elapsed=%s err=%v", name, args, time.Since(started), err)
	if ctx.Err() != nil {
		t.Fatalf("DEADLINE command did not complete: %v", ctx.Err())
	}
	return stdout.String(), stderr.String(), err
}

func checkerCoverageProfile(t *testing.T, ctx context.Context, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("COVERAGE profile missing: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 || lines[0] != "mode: set" {
		t.Fatalf("COVERAGE expected nonempty statement profile: %q", data)
	}
	block := regexp.MustCompile(`^(.+):([0-9]+)\.([0-9]+),([0-9]+)\.([0-9]+) ([0-9]+) ([0-9]+)$`)
	statements, executed, productionExecuted := 0, 0, 0
	for _, line := range lines[1:] {
		parts := block.FindStringSubmatch(line)
		if parts == nil {
			t.Fatalf("COVERAGE invalid statement block: %q", line)
		}
		if !strings.Contains(parts[1], "/cmd/machinery/") {
			t.Fatalf("COVERAGE unexpected package: %q", parts[1])
		}
		n, _ := strconv.Atoi(parts[6])
		count, _ := strconv.Atoi(parts[7])
		statements += n
		if count > 0 {
			executed += n
			if strings.HasSuffix(parts[1], "/verify_checkers.go") {
				productionExecuted += n
			}
		}
	}
	if statements == 0 || executed == 0 || productionExecuted == 0 {
		t.Fatalf("COVERAGE no real package/production execution: statements=%d executed=%d verify_checkers=%d", statements, executed, productionExecuted)
	}
	stdout, stderr, err := checkerCoverageCommand(t, ctx, nil, "go", "tool", "cover", "-func="+path)
	if err != nil {
		t.Fatalf("COVERAGE Go rejected profile: %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	t.Logf("COVERAGE valid blocks=%d statements=%d executed=%d verify_checkers=%d (%.1f%%)", len(lines)-1, statements, executed, productionExecuted, 100*float64(executed)/float64(statements))
	if !strings.Contains(stdout, "total:") {
		t.Errorf("COVERAGE Go report missing total: %s", stdout)
	}
}
