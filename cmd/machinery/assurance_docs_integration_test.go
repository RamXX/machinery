//go:build machinery_integration

// MAC-gcrr real registry/bind-path documentation example. These are the
// named runtime cases registered in testdata/integration-lanes/
// consumer-docs.json; the set of Test functions and the fragment must stay in
// exact both-directions sync. The flow is the one documented in
// docs/external-checkers.md, exercised against the real daemon and the pinned
// immutable checker image with production code only: a real temp registry
// whose inputs sit beside it (the documented repo-root workaround), the real
// registry-relative input snapshot, the snapshotted engine, exact-identity
// image verification, and the real /work + read-only /checker binds. A
// missing or unreachable daemon fails the case; nothing is skipped.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/designlock"
)

// consumerDocsAdapter is a real one-file adapter in the documented shape: it
// reads {projection} and {config}, proves its own mounted source is readable
// through the read-only /checker bind, and writes evidence to {out} with zero
// stdout/stderr, per the file-only checker contract.
const consumerDocsAdapter = `import hashlib, json, sys

projection, config, out = sys.argv[1], sys.argv[2], sys.argv[3]
with open(projection, "rb") as f:
    projection_bytes = f.read()
with open(config, "rb") as f:
    config_bytes = f.read()
with open("/checker/consumer-adapter.py", "rb") as f:
    mounted_self = f.read()
evidence = {
    "evidence_schema": "1.0",
    "checker": {"id": "consumer-docs-example", "version": "1"},
    "projection_sha256": hashlib.sha256(projection_bytes).hexdigest(),
    "config_bytes": len(config_bytes),
    "mounted_input_sha256": hashlib.sha256(mounted_self).hexdigest(),
    "verdict": "pass",
}
with open(out, "w") as f:
    json.dump(evidence, f)
`

// consumerDocsProbeWriter proves the mount contract with real filesystem
// semantics inside the container: /work is writable, /checker is not. It
// reports the outcome through /work so the host asserts facts, not hopes.
const consumerDocsProbeWriter = `import json, sys

result = {"work_write": "failed before open", "checker_write": "refused"}
try:
    with open("/work/probe.out", "w") as f:
        f.write("writable\n")
    result["work_write"] = "written"
except OSError as e:
    result["work_write"] = "error: %s" % e
try:
    with open("/checker/probe.out", "w") as f:
        f.write("must not appear\n")
    result["checker_write"] = "unexpectedly succeeded"
except OSError:
    result["checker_write"] = "refused"
with open("/work/probe-result.json", "w") as f:
    json.dump(result, f)
`

// consumerDocsRegistry writes a real registry whose inputs live beside it,
// exactly like the documented place-the-registry-beside-your-inputs pattern,
// and returns the registry path and the directory holding the input sources.
func consumerDocsRegistry(t *testing.T, entries map[string]string) (registryPath string) {
	t.Helper()
	root := t.TempDir()
	for rel, body := range entries {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(root, "checkers.local.yaml")
}

func consumerDocsRegistryYAML(runCommand, inputSource, inputMount string) string {
	return fmt.Sprintf(`checkers:
  consumer-docs-example:
    runtime:
      kind: oci
      engine: [docker]
      image: %s
      platform: %s
      inputs:
        - {source: %s, mount: %s}
    run: %s
    timeout: "120s"
`, lifecycleImage, testRuntimePlatform, inputSource, inputMount, runCommand)
}

// TestAssuranceDocsRegistryBindPathExampleFlow runs the documented example
// flow end to end against the real daemon: registry-relative input snapshot,
// engine snapshot, exact-identity image verification, container execution with
// the /work and read-only /checker binds, byte-identical mounted inputs,
// evidence landing on the host through /work, the zero-output contract, and
// no owned container remaining afterwards.
func TestAssuranceDocsRegistryBindPathExampleFlow(t *testing.T) {
	design := filepath.Join(repoRootDir(t), "examples", "pii-flow", "design")
	snapshot, err := designlock.AcquireReader(design)
	if err != nil {
		t.Fatalf("acquire design snapshot: %v", err)
	}
	defer func() {
		if err := snapshot.CheckUnchanged(); err != nil {
			t.Errorf("design snapshot was not stable: %v", err)
		}
		if err := snapshot.Release(); err != nil {
			t.Errorf("release design snapshot: %v", err)
		}
	}()

	registryPath := consumerDocsRegistry(t, map[string]string{
		"checkers.local.yaml":       consumerDocsRegistryYAML(`["python3", "/checker/consumer-adapter.py", "{projection}", "{config}", "{out}"]`, "tools/consumer-adapter.py", "consumer-adapter.py"),
		"tools/consumer-adapter.py": consumerDocsAdapter,
	})
	registrySnapshot, err := snapshot.MaterializeRegularFile(registryPath)
	if err != nil {
		t.Fatalf("snapshot checker registry: %v", err)
	}
	defer func() {
		if err := registrySnapshot.Close(); err != nil {
			t.Errorf("close registry snapshot: %v", err)
		}
	}()
	reg, err := checker.LoadRegistry(registrySnapshot.Path())
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry, ok := reg.Resolve("consumer-docs-example")
	if !ok {
		t.Fatal("registry must resolve the documented example checker")
	}

	work, err := createCheckerWorkDir(design)
	if err != nil {
		t.Fatalf("create private checker work dir: %v", err)
	}
	defer os.RemoveAll(work)

	tokens := checker.Tokens{
		Projection: "/work/projection.json",
		Config:     "/work/config.json",
		Manifest:   "/work/manifest.checker.yaml",
		Out:        "/work/evidence.json",
	}
	runArgs := tokens.Substitute(entry.Run)

	inputDigests, err := snapshotOCIInputs(work, registryPath, entry.Runtime.Inputs, snapshot)
	if err != nil {
		t.Fatalf("snapshot declared OCI checker inputs (registry-relative resolution): %v", err)
	}
	if len(inputDigests) != 1 || inputDigests["consumer-adapter.py"] == "" {
		t.Fatalf("expected one hashed registry-relative input, got %v", inputDigests)
	}
	runtimeClosure, err := checker.RuntimeClosureDigest(entry.Runtime.Digest, entry.Runtime.Platform, entry.Run, entry.Verify, inputDigests)
	if err != nil {
		t.Fatalf("compute runtime closure: %v", err)
	}
	// The documented engine contract: resolved once from PATH, required to be
	// a regular non-symlink executable, copied and rechecked before use. This
	// host's docker is a Desktop symlink, so publish a snapshot-safe regular
	// copy on PATH exactly like the documented resolution expects.
	engine := lifecycleRealEngine(t)
	t.Setenv("PATH", filepath.Dir(engine)+string(os.PathListSeparator)+os.Getenv("PATH"))
	snapshots, err := snapshotCheckerCommands(work, entry.Runtime.Engine)
	if err != nil {
		t.Fatalf("snapshot engine executable: %v", err)
	}
	engineArgs := snapshots[0]
	lifecycleRequirePinnedImage(t, engineArgs[0])
	if err := verifyLocalOCIImage(engineArgs, entry.Runtime.Image, entry.Runtime.Digest, entry.Runtime.Platform, entry.Timeout, work); err != nil {
		t.Fatalf("verify local OCI image identity: %v", err)
	}
	toolBinding, err := checkerToolClosureHash(work)
	if err != nil {
		t.Fatalf("bind private checker tool closure: %v", err)
	}

	projection := []byte(`{"generated":{"input_hash":"sha256:consumer-docs-example"},"elements":[]}` + "\n")
	if err := os.WriteFile(filepath.Join(work, "projection.json"), projection, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "config.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, runErr := runCheckerOCI(engineArgs, entry.Runtime.Image, entry.Runtime.Platform, runArgs, runtimeClosure, entry.Timeout, work)
	if runErr != nil {
		t.Fatalf("checker run through the documented registry/bind path failed: %v\n%s", runErr, out)
	}
	if out != "" {
		t.Fatalf("checker emitted stdout/stderr despite the file-only evidence contract: %q", out)
	}
	if err := verifyCheckerToolClosure(work, toolBinding); err != nil {
		t.Fatalf("checker mutated its private tool closure: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(work, "evidence.json"))
	if err != nil {
		t.Fatalf("evidence did not land on the host through the /work bind: %v", err)
	}
	var evidence struct {
		Checker          struct{ ID, Version string }
		ProjectionSHA256 string `json:"projection_sha256"`
		ConfigBytes      int    `json:"config_bytes"`
		MountedInputSHA  string `json:"mounted_input_sha256"`
		Verdict          string
	}
	if err := json.Unmarshal(raw, &evidence); err != nil {
		t.Fatalf("evidence is not JSON: %v\n%s", err, raw)
	}
	if evidence.Checker.ID != "consumer-docs-example" || evidence.Verdict != "pass" {
		t.Fatalf("evidence identity/verdict = %+v", evidence)
	}
	if want := fmt.Sprintf("%x", sha256.Sum256(projection)); evidence.ProjectionSHA256 != want {
		t.Fatalf("evidence projection digest %s does not bind the bytes the host mounted (%s)", evidence.ProjectionSHA256, want)
	}
	if evidence.ConfigBytes != 3 {
		t.Fatalf("evidence config length = %d, want the exact mounted bytes", evidence.ConfigBytes)
	}
	if want := strings.TrimPrefix(inputDigests["consumer-adapter.py"], "sha256:"); evidence.MountedInputSHA != want {
		t.Fatalf("mounted /checker input digest %s differs from the registry-relative snapshot digest %s", evidence.MountedInputSHA, want)
	}

	consumerDocsAssertNoOwnedContainer(t, work)
}

// TestAssuranceDocsRuntimeInputsMountReadOnly proves the documented isolation
// contract with real container filesystem semantics: the private /work bind
// is writable by the checker, the read-only /checker input mount refuses
// writes, and results travel back to the host through /work.
func TestAssuranceDocsRuntimeInputsMountReadOnly(t *testing.T) {
	design := filepath.Join(repoRootDir(t), "examples", "pii-flow", "design")
	snapshot, err := designlock.AcquireReader(design)
	if err != nil {
		t.Fatalf("acquire design snapshot: %v", err)
	}
	defer func() {
		if err := snapshot.CheckUnchanged(); err != nil {
			t.Errorf("design snapshot was not stable: %v", err)
		}
		if err := snapshot.Release(); err != nil {
			t.Errorf("release design snapshot: %v", err)
		}
	}()

	registryPath := consumerDocsRegistry(t, map[string]string{
		"checkers.local.yaml":   consumerDocsRegistryYAML(`["python3", "/checker/probe-writer.py"]`, "tools/probe-writer.py", "probe-writer.py"),
		"tools/probe-writer.py": consumerDocsProbeWriter,
	})
	registrySnapshot, err := snapshot.MaterializeRegularFile(registryPath)
	if err != nil {
		t.Fatalf("snapshot checker registry: %v", err)
	}
	defer func() {
		if err := registrySnapshot.Close(); err != nil {
			t.Errorf("close registry snapshot: %v", err)
		}
	}()
	reg, err := checker.LoadRegistry(registrySnapshot.Path())
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	entry, ok := reg.Resolve("consumer-docs-example")
	if !ok {
		t.Fatal("registry must resolve the documented example checker")
	}

	work, err := createCheckerWorkDir(design)
	if err != nil {
		t.Fatalf("create private checker work dir: %v", err)
	}
	defer os.RemoveAll(work)

	inputDigests, err := snapshotOCIInputs(work, registryPath, entry.Runtime.Inputs, snapshot)
	if err != nil {
		t.Fatalf("snapshot declared OCI checker inputs: %v", err)
	}
	runtimeClosure, err := checker.RuntimeClosureDigest(entry.Runtime.Digest, entry.Runtime.Platform, entry.Run, entry.Verify, inputDigests)
	if err != nil {
		t.Fatalf("compute runtime closure: %v", err)
	}
	// The documented engine contract: resolved once from PATH, required to be
	// a regular non-symlink executable, copied and rechecked before use. This
	// host's docker is a Desktop symlink, so publish a snapshot-safe regular
	// copy on PATH exactly like the documented resolution expects.
	engine := lifecycleRealEngine(t)
	t.Setenv("PATH", filepath.Dir(engine)+string(os.PathListSeparator)+os.Getenv("PATH"))
	snapshots, err := snapshotCheckerCommands(work, entry.Runtime.Engine)
	if err != nil {
		t.Fatalf("snapshot engine executable: %v", err)
	}
	engineArgs := snapshots[0]
	lifecycleRequirePinnedImage(t, engineArgs[0])
	if err := verifyLocalOCIImage(engineArgs, entry.Runtime.Image, entry.Runtime.Digest, entry.Runtime.Platform, entry.Timeout, work); err != nil {
		t.Fatalf("verify local OCI image identity: %v", err)
	}

	runArgs := (&checker.Tokens{Out: "/work/probe-result.json"}).Substitute(entry.Run)
	out, runErr := runCheckerOCI(engineArgs, entry.Runtime.Image, entry.Runtime.Platform, runArgs, runtimeClosure, entry.Timeout, work)
	if runErr != nil {
		t.Fatalf("probe run failed: %v\n%s", runErr, out)
	}
	if out != "" {
		t.Fatalf("probe emitted stdout/stderr despite the file-only evidence contract: %q", out)
	}

	raw, err := os.ReadFile(filepath.Join(work, "probe-result.json"))
	if err != nil {
		t.Fatalf("probe result did not land on the host through the /work bind: %v", err)
	}
	var result struct {
		WorkWrite    string `json:"work_write"`
		CheckerWrite string `json:"checker_write"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("probe result is not JSON: %v\n%s", err, raw)
	}
	if result.WorkWrite != "written" {
		t.Fatalf("the documented writable /work bind refused a write: %s", result.WorkWrite)
	}
	if result.CheckerWrite != "refused" {
		t.Fatalf("the documented read-only /checker mount accepted a write: %s", result.CheckerWrite)
	}
	if body, err := os.ReadFile(filepath.Join(work, "probe.out")); err != nil || string(body) != "writable\n" {
		t.Fatalf("the /work write did not reach the host bind (body=%q err=%v)", body, err)
	}

	consumerDocsAssertNoOwnedContainer(t, work)
}

// consumerDocsAssertNoOwnedContainer verifies the documented lifetime
// ownership promise for this example's own container: after runCheckerOCI
// returns, the registered identity no longer exists on the daemon.
func consumerDocsAssertNoOwnedContainer(t *testing.T, work string) {
	t.Helper()
	raw, err := os.ReadFile(lifecycleCIDPath(work))
	if err != nil || len(raw) == 0 {
		t.Fatalf("checker container identity was not registered at %s: %v", lifecycleCIDPath(work), err)
	}
	retryDeadline := time.Now().Add(30 * time.Second)
	for {
		out, _, inspectErr := lifecycleDockerOutput(t, 30*time.Second, "inspect", string(raw))
		if inspectErr != nil || strings.Contains(out, "No such object") {
			return
		}
		if time.Now().After(retryDeadline) {
			t.Fatalf("owned checker container %s still exists after the documented force-remove contract: %s", raw, out)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
