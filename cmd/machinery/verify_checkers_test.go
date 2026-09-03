package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/processcontrol"
	machversion "github.com/RamXX/machinery/internal/version"
)

type recordingCheckerToolDirectoryReader struct {
	requests []int
}

func (r *recordingCheckerToolDirectoryReader) ReadDir(count int) ([]fs.DirEntry, error) {
	r.requests = append(r.requests, count)
	return nil, io.EOF
}

func TestCheckerToolInventoryMaxIntNeverIssuesUnboundedRequest(t *testing.T) {
	reader := &recordingCheckerToolDirectoryReader{}
	if _, err := readCheckerToolDir(reader, math.MaxInt); err != nil {
		t.Fatal(err)
	}
	if len(reader.requests) != 1 || reader.requests[0] != checkerToolReadDirPage {
		t.Fatalf("ReadDir requests = %v, want one positive bounded page of %d", reader.requests, checkerToolReadDirPage)
	}
}

const testRuntimeClosure = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
const testRuntimeImage = "example.invalid/machinery/checker-test@" + testRuntimeClosure
const testRuntimePlatform = "linux/amd64"

const (
	checkerScratchCrashEnv       = "MACHINERY_CHECKER_SCRATCH_CRASH_HELPER"
	checkerScratchCrashDesignEnv = "MACHINERY_CHECKER_SCRATCH_CRASH_DESIGN"
	checkerScratchCrashRegistry  = "MACHINERY_CHECKER_SCRATCH_CRASH_REGISTRY"
)

func TestCheckerToolInventoryRejectsEntryBeyondFixedCeiling(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"c", "a", "b"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck // test cleanup
	if _, err := readCheckerToolDir(f, 2); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("high-entry checker inventory was accepted: %v", err)
	}
}

func TestCheckerToolInventoryRejectsExcessiveDepth(t *testing.T) {
	work := t.TempDir()
	for _, dir := range []string{"tool-assets", "tool-path", "tool-snapshots"} {
		if err := os.Mkdir(filepath.Join(work, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(work, "tool-assets", "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(work)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	if _, _, err := inventoryCheckerToolClosureBounded(root, 10, 1); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("deep checker tool inventory was accepted: %v", err)
	}
}

func TestReadEvidenceTraceRejectsHighEntryInventory(t *testing.T) {
	root := t.TempDir()
	generated := filepath.Join(root, "checkers", "test", "generated")
	if err := os.MkdirAll(generated, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= checkerTraceInventoryLimit; i++ {
		name := fmt.Sprintf("trace-%04d.bin", i)
		if err := os.WriteFile(filepath.Join(generated, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	evidence := &checker.Evidence{TraceRef: "generated/trace-0000.bin"}
	if _, err := readEvidenceTrace(root, "checkers/test/evidence.json", evidence); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("high-entry generated trace inventory was accepted: %v", err)
	}
}

func TestCheckerToolClosureRejectsContinuousAppender(t *testing.T) {
	work := t.TempDir()
	for _, dir := range []string{"tool-assets", "tool-path", "tool-snapshots"} {
		if err := os.Mkdir(filepath.Join(work, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	targetRel := filepath.Join("tool-snapshots", "tool")
	target := filepath.Join(work, targetRel)
	if err := os.WriteFile(target, []byte(strings.Repeat("x", 1<<20)), 0o700); err != nil {
		t.Fatal(err)
	}
	old := checkerToolClosurePoint
	t.Cleanup(func() { checkerToolClosurePoint = old })
	done := make(chan struct{})
	stopped := make(chan struct{})
	checkerToolClosurePoint = func(path, phase string) error {
		if path != targetRel || phase != "after-open" {
			return nil
		}
		checkerToolClosurePoint = func(string, string) error { return nil }
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			f, openErr := os.OpenFile(target, os.O_WRONLY|os.O_APPEND, 0)
			if openErr != nil {
				close(first)
				return
			}
			defer f.Close() //nolint:errcheck // test mutation
			for i := 0; ; i++ {
				_, _ = f.Write([]byte("growth"))
				if i == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
		return nil
	}
	_, err := checkerToolClosureHash(work)
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed size while hashing") {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

func TestCheckerSnapshotRejectsContinuousAppender(t *testing.T) {
	source := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(source, []byte(strings.Repeat("x", 1<<20)), 0o700); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(source)
	if err != nil {
		t.Fatal(err)
	}
	old := checkerSnapshotPoint
	t.Cleanup(func() { checkerSnapshotPoint = old })
	done := make(chan struct{})
	stopped := make(chan struct{})
	checkerSnapshotPoint = func(path, phase string) error {
		if path != source || phase != "after-open" {
			return nil
		}
		checkerSnapshotPoint = func(string, string) error { return nil }
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			f, openErr := os.OpenFile(source, os.O_WRONLY|os.O_APPEND, 0)
			if openErr != nil {
				close(first)
				return
			}
			defer f.Close() //nolint:errcheck // test mutation
			for i := 0; ; i++ {
				_, _ = f.Write([]byte("growth"))
				if i == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
		return nil
	}
	err = snapshotCheckerFile(checkerSnapshotSource{path: source, info: info}, filepath.Join(t.TempDir(), "snapshot"), 0o700)
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed size") {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

const vcModel = `kind: DomainModel
version: v1
title: T
entities:
  DataSubject:
    attributes:
      - {name: email, type: string}
    relationships:
      - {entity: Export, cardinality: 1:n}
    invariants:
      - {id: priv-consent, statement: "Consent required."}
      - {id: priv-retention, statement: "Retention bounded."}
  Export:
    attributes:
      - {name: name, type: string}
`

const vcManifest = `checker: {id: test, runtime_closure: ` + testRuntimeClosure + `}
projection: {include: [model, invariants, relationships]}
coverage:
  claim: ["priv-*"]
config:
  sensitive: [email]
evidence:
  projection_out: checkers/test/projection.json
  evidence_in: checkers/test/evidence.json
`

// vcDesign is a complete, by-default-reproducible checker design plus the paths
// a test mutates per case.
type vcDesign struct {
	dir      string
	projPath string
	evPath   string
}

// setupVerifyDesign builds a temp design the way internal/gates builds one: a
// model, a manifest, a committed projection generated from the model, and a
// committed evidence whose input_hash binds to that projection.
func setupVerifyDesign(t *testing.T) vcDesign {
	t.Helper()
	design := t.TempDir()
	modelPath := filepath.Join(design, "d.modelith.yaml")
	if err := os.WriteFile(modelPath, []byte(vcModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers", "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	manPath := filepath.Join(design, "checkers", "test.checker.yaml")
	if err := os.WriteFile(manPath, []byte(vcManifest), 0o644); err != nil {
		t.Fatal(err)
	}

	man, err := checker.LoadManifest(manPath)
	if err != nil {
		t.Fatal(err)
	}
	model, err := checker.LoadModel(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	did, err := checker.DesignID(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	proj, err := checker.Generate(model, man, did, machversion.Version)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := proj.Render()
	if err != nil {
		t.Fatal(err)
	}
	projPath := filepath.Join(design, "checkers", "test", "projection.json")
	if err := os.WriteFile(projPath, rendered, 0o644); err != nil {
		t.Fatal(err)
	}
	hash, err := proj.InputHash()
	if err != nil {
		t.Fatal(err)
	}

	ev := checker.Evidence{
		EvidenceSchema: checker.SchemaVersion,
		InputHash:      hash,
		RuntimeClosure: testRuntimeClosure,
		Verdict:        "pass",
		Coverage: []checker.CoverageRow{
			{Element: "inv:priv-consent", Verdict: "pass"},
			{Element: "inv:priv-retention", Verdict: "pass"},
		},
	}
	ev.Checker.ID = "test"
	ev.Checker.Version = "t"
	evPath := filepath.Join(design, "checkers", "test", "evidence.json")
	writeJSON(t, evPath, ev)

	return vcDesign{dir: design, projPath: projPath, evPath: evPath}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeScript writes an executable /bin/sh stub and returns its path.
func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stub.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

// writeRegistryFile writes a registry YAML file and returns its path.
func writeRegistryFile(t *testing.T, body string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	engineJSON, err := json.Marshal([]string{executable, "-test.run=^TestCheckerProcessFixture$", "--", checkerProcessFixtureMarker, "oci-engine"})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(body, "\n")
	var augmented []string
	for _, line := range lines {
		augmented = append(augmented, line)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(strings.TrimSpace(line), ":") {
			augmented = append(augmented,
				"    runtime:",
				"      kind: oci",
				"      engine: "+string(engineJSON),
				"      image: "+testRuntimeImage,
				"      platform: "+testRuntimePlatform,
			)
		}
	}
	p := filepath.Join(t.TempDir(), "checkers.local.yaml")
	if err := os.WriteFile(p, []byte(strings.Join(augmented, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func checkerFixtureEngineArgs(t *testing.T, mode string) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal([]string{executable, "-test.run=^TestCheckerProcessFixture$", "--", checkerProcessFixtureMarker, mode})
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func writeRawRegistryFile(t *testing.T, dir, body string) string {
	t.Helper()
	if dir == "" {
		dir = t.TempDir()
	}
	p := filepath.Join(dir, "checkers.local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runVC runs verify-checkers with captured IO, returning stdout, stderr, and
// the exit code (0 when exitFunc was never called).
func runVC(t *testing.T, design, registry string, extra ...string) (string, string, int) {
	t.Helper()
	syncTestRuntimeClosure(t, design, registry)
	return runVCRaw(t, design, registry, extra...)
}

func runVCRaw(t *testing.T, design, registry string, extra ...string) (string, string, int) {
	t.Helper()
	out, errB, codes := withCapturedIO(t)
	cmd := newVerifyCheckersCmd()
	cmd.SetArgs(append([]string{design, "--registry", registry}, extra...))
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	code := 0
	if len(*codes) > 0 {
		code = (*codes)[len(*codes)-1]
	}
	return out.String(), errB.String(), code
}

func syncTestRuntimeClosure(t *testing.T, design, registryPath string) {
	t.Helper()
	reg, err := checker.LoadRegistry(registryPath)
	if err != nil {
		return
	}
	entry, ok := reg.Resolve("test")
	if !ok {
		return
	}
	inputDigests := make(map[string]string, len(entry.Runtime.Inputs))
	registryAbs, err := filepath.Abs(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range entry.Runtime.Inputs {
		source := input.Source
		if !filepath.IsAbs(source) {
			source = filepath.Join(filepath.Dir(registryAbs), filepath.FromSlash(source))
		}
		body, readErr := os.ReadFile(source)
		if readErr != nil {
			t.Fatal(readErr)
		}
		inputDigests[input.Mount] = fmt.Sprintf("sha256:%x", sha256.Sum256(body))
	}
	closure, err := checker.RuntimeClosureDigest(entry.Runtime.Digest, entry.Runtime.Platform, entry.Run, entry.Verify, inputDigests)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(design, "checkers", "test.checker.yaml")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return
	}
	start := strings.Index(string(body), "runtime_closure: ")
	if start >= 0 {
		valueAt := start + len("runtime_closure: ")
		if len(body) >= valueAt+len(testRuntimeClosure) {
			updated := append([]byte(nil), body...)
			copy(updated[valueAt:valueAt+len(testRuntimeClosure)], closure)
			if err := os.WriteFile(manifestPath, updated, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	evidencePath := filepath.Join(design, "checkers", "test", "evidence.json")
	evidenceBody, err := os.ReadFile(evidencePath)
	if err != nil {
		return
	}
	var evidence checker.Evidence
	if json.Unmarshal(evidenceBody, &evidence) == nil {
		evidence.RuntimeClosure = closure
		writeJSON(t, evidencePath, evidence)
	}
}

// TestVerifyCheckersReproducible: a stub that reproduces the committed evidence
// yields ok and exit 0, and machinery hands the adapter the manifest config as
// JSON via {config}.
func TestVerifyCheckersReproducible(t *testing.T) {
	d := setupVerifyDesign(t)
	configSink := filepath.Join(t.TempDir(), "seen-config.json")
	stub := writeScript(t, "/bin/cp \"$1\" \""+configSink+"\"\n/bin/cp \""+d.evPath+"\" \"$2\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{config}\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 0 {
		t.Fatalf("reproducible design should exit 0, got %d\nstderr: %s", code, errS)
	}
	if !strings.Contains(out, "verify-checkers test: ok (verdict=pass, reproduced)") {
		t.Fatalf("expected ok result line, got:\n%s", out)
	}
	if !strings.Contains(out, "1 checker(s) verified") {
		t.Fatalf("expected summary line, got:\n%s", out)
	}

	// {config} carried the manifest config block as JSON.
	seen, err := os.ReadFile(configSink)
	if err != nil {
		t.Fatalf("adapter never received {config}: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(seen, &cfg); err != nil {
		t.Fatalf("{config} was not valid JSON: %v\n%s", err, seen)
	}
	if _, ok := cfg["sensitive"]; !ok {
		t.Fatalf("{config} JSON missing the manifest config: %s", seen)
	}
}

// TestVerifyCheckersDifferentVerdictFails: a stub that writes a different
// verdict is not reproducible -> ERROR, exit 1.
func TestVerifyCheckersDifferentVerdictFails(t *testing.T) {
	d := setupVerifyDesign(t)

	// A fresh-evidence fixture identical to committed except the verdict.
	var ev checker.Evidence
	raw, err := os.ReadFile(d.evPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	ev.Verdict = "fail"
	ev.Findings = []checker.Finding{{Severity: "blocking", Message: "injected failure"}}
	fixture := filepath.Join(t.TempDir(), "fresh.json")
	writeJSON(t, fixture, ev)

	stub := writeScript(t, "/bin/cp \""+fixture+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("a different verdict must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "not reproducible") {
		t.Fatalf("expected a reproducibility ERROR, got:\n%s", errS)
	}
}

// TestVerifyCheckersMissingRegistryEntry: registry has no entry for the
// manifest's id -> ERROR, exit 1.
func TestVerifyCheckersMissingRegistryEntry(t *testing.T) {
	d := setupVerifyDesign(t)
	reg := writeRegistryFile(t, "checkers:\n  other:\n    run: [\"/bin/true\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("a missing registry entry must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "no registry entry for checker 'test'") {
		t.Fatalf("expected a missing-entry ERROR, got:\n%s", errS)
	}
}

// TestVerifyCheckersMissingProjection: the committed projection is absent ->
// ERROR, exit 1.
func TestVerifyCheckersMissingProjection(t *testing.T) {
	d := setupVerifyDesign(t)
	if err := os.Remove(d.projPath); err != nil {
		t.Fatal(err)
	}
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("a missing committed projection must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "projection not committed") {
		t.Fatalf("expected a missing-projection ERROR, got:\n%s", errS)
	}
}

// TestVerifyCheckersReplayFails: the run reproduces, but the verify/replay
// command exits nonzero -> ERROR, exit 1.
func TestVerifyCheckersReplayFails(t *testing.T) {
	d := setupVerifyDesign(t)
	runStub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	verifyStub := writeScript(t, "exit 1\n")
	uniqueVerifyStub := filepath.Join(filepath.Dir(verifyStub), "verify-stub.sh")
	if err := os.Rename(verifyStub, uniqueVerifyStub); err != nil {
		t.Fatal(err)
	}
	verifyStub = uniqueVerifyStub
	reg := writeRegistryFile(t,
		"checkers:\n  test:\n    run: [\""+runStub+"\", \"{out}\"]\n    verify: [\""+verifyStub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("a failing verify/replay must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "replay/verify failed for 'test'") {
		t.Fatalf("expected a replay ERROR, got:\n%s", errS)
	}
}

func TestVerifyCheckersRejectsSuccessfulRunOutput(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "printf 'run warning\\n'\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 || !strings.Contains(errS, "run emitted stdout/stderr despite the file-only evidence contract") || !strings.Contains(errS, `run warning\n`) {
		t.Fatalf("successful run output was not rejected and surfaced: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
}

func TestVerifyCheckersRejectsSuccessfulReplayOutput(t *testing.T) {
	d := setupVerifyDesign(t)
	runStub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	verifyStub := writeScript(t, "printf 'replay warning\\n' >&2\n")
	reg := writeRegistryFile(t,
		"checkers:\n  test:\n    run: [\""+runStub+"\", \"{out}\"]\n    verify: [\""+verifyStub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 || !strings.Contains(errS, "replay/verify for 'test' emitted stdout/stderr despite the file-only evidence contract") || !strings.Contains(errS, `replay warning\n`) {
		t.Fatalf("successful replay output was not rejected and surfaced: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
}

func TestVerifyCheckersRejectsNondeterministicFreshTrace(t *testing.T) {
	for _, withVerify := range []bool{false, true} {
		name := "without verify"
		if withVerify {
			name = "with verify"
		}
		t.Run(name, func(t *testing.T) {
			d := setupVerifyDesign(t)
			tracePath := filepath.Join(d.dir, "checkers", "test", "generated", "trace.bin")
			if err := os.MkdirAll(filepath.Dir(tracePath), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(tracePath, []byte("committed-trace\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			var evidence checker.Evidence
			body, err := os.ReadFile(d.evPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body, &evidence); err != nil {
				t.Fatal(err)
			}
			evidence.TraceRef = "generated/trace.bin"
			writeJSON(t, d.evPath, evidence)

			runStub := writeScript(t, "/bin/mkdir -p \"${1%/*}/generated\"\nprintf 'fresh-trace\\n' > \"${1%/*}/generated/trace.bin\"\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
			registryBody := "checkers:\n  test:\n    run: [\"" + runStub + "\", \"{out}\"]\n"
			verifyMarker := filepath.Join(t.TempDir(), "verify-ran")
			if withVerify {
				verifyStub := writeScript(t, "/usr/bin/touch \""+verifyMarker+"\"\n")
				registryBody += "    verify: [\"" + verifyStub + "\", \"{out}\"]\n"
			}
			registry := writeRegistryFile(t, registryBody)

			out, errS, code := runVC(t, d.dir, registry)
			if code != 1 || !strings.Contains(errS, "trace content differs") || !strings.Contains(errS, "committed sha256:") || !strings.Contains(errS, "fresh sha256:") {
				t.Fatalf("nondeterministic trace was not rejected: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
			}
			if _, err := os.Lstat(verifyMarker); !os.IsNotExist(err) {
				t.Fatalf("replay ran before fresh trace content was authenticated: %v", err)
			}
		})
	}
}

func TestVerifyCheckersRejectsFreshTraceInventoryExtraArtifact(t *testing.T) {
	d := setupVerifyDesign(t)
	tracePath := filepath.Join(d.dir, "checkers", "test", "generated", "trace.bin")
	if err := os.MkdirAll(filepath.Dir(tracePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracePath, []byte("stable-trace\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var evidence checker.Evidence
	body, err := os.ReadFile(d.evPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	evidence.TraceRef = "generated/trace.bin"
	writeJSON(t, d.evPath, evidence)
	runStub := writeScript(t, "/bin/mkdir -p \"${1%/*}/generated\"\nprintf 'stable-trace\\n' > \"${1%/*}/generated/trace.bin\"\nprintf 'extra\\n' > \"${1%/*}/generated/extra.bin\"\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
	registry := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+runStub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, registry)
	if code != 1 || !strings.Contains(errS, "fresh trace inventory is invalid") || !strings.Contains(errS, "undeclared artifact generated/extra.bin") {
		t.Fatalf("extra fresh trace artifact was not rejected: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
}

// TestVerifyCheckersInputHashMismatch: same verdict but a different input_hash
// (the verdict was computed over a different design) -> not reproducible.
func TestVerifyCheckersInputHashMismatch(t *testing.T) {
	d := setupVerifyDesign(t)
	var ev checker.Evidence
	raw, err := os.ReadFile(d.evPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	ev.InputHash = "sha256:" + strings.Repeat("0", 64)
	fixture := filepath.Join(t.TempDir(), "fresh.json")
	writeJSON(t, fixture, ev)

	stub := writeScript(t, "/bin/cp \""+fixture+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("an input_hash mismatch must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "not reproducible") || !strings.Contains(errS, "input_hash") {
		t.Fatalf("expected an input_hash reproducibility ERROR, got:\n%s", errS)
	}
}

func TestVerifyCheckersRejectsMatchingEvidenceOverWrongCurrentInput(t *testing.T) {
	d := setupVerifyDesign(t)
	var ev checker.Evidence
	raw, err := os.ReadFile(d.evPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	ev.InputHash = "sha256:" + strings.Repeat("0", 64)
	writeJSON(t, d.evPath, ev)
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	_, errS, code := runVC(t, d.dir, reg)
	if code != 1 || !strings.Contains(errS, "committed evidence input_hash does not bind") {
		t.Fatalf("matching wrong evidence was accepted: code=%d\n%s", code, errS)
	}
}

func TestVerifyCheckersRejectsProjectionStaleAgainstCurrentModel(t *testing.T) {
	d := setupVerifyDesign(t)
	modelPath := filepath.Join(d.dir, "d.modelith.yaml")
	raw, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, append(raw, []byte("\n# source changed\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	_, errS, code := runVC(t, d.dir, reg)
	if code != 1 || !strings.Contains(errS, "committed projection is stale") {
		t.Fatalf("stale committed projection was accepted: code=%d\n%s", code, errS)
	}
}

func TestVerifyCheckersRunsInIsolatedWorkingDirectoryAndEnvironment(t *testing.T) {
	d := setupVerifyDesign(t)
	sinkDir := t.TempDir()
	cwdSink := filepath.Join(sinkDir, "cwd")
	envSink := filepath.Join(sinkDir, "env")
	t.Setenv("MACHINERY_SHOULD_NOT_REACH_CHECKER", "secret")
	stub := writeScript(t, "pwd > \"$1\"\n/usr/bin/env | /usr/bin/sort > \"$2\"\n/bin/cp \""+d.evPath+"\" \"$3\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \""+cwdSink+"\", \""+envSink+"\", \"{out}\"]\n")

	_, errS, code := runVC(t, d.dir, reg)
	if code != 0 {
		t.Fatalf("isolated checker failed: code=%d\n%s", code, errS)
	}
	cwd, err := os.ReadFile(cwdSink)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(cwd)) == d.dir || !strings.Contains(string(cwd), "machinery-verify-checkers") {
		t.Fatalf("checker did not run in its explicit temp cwd: %q", cwd)
	}
	if _, err := os.Stat(strings.TrimSpace(string(cwd))); !os.IsNotExist(err) {
		t.Fatalf("private checker workspace was not removed after verification: %v", err)
	}
	env, err := os.ReadFile(envSink)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(env), "MACHINERY_SHOULD_NOT_REACH_CHECKER") {
		t.Fatalf("ambient environment leaked to checker:\n%s", env)
	}
	for _, required := range []string{"HOME=", "LANG=C", "LC_ALL=C", "TMPDIR=", "TZ=UTC"} {
		if !strings.Contains(string(env), required) {
			t.Errorf("deterministic environment missing %s:\n%s", required, env)
		}
	}
}

func TestSnapshotCheckerCommandsPreservesModeAndRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable permissions and symlink fixture")
	}
	toolDir := t.TempDir()
	tool := filepath.Join(toolDir, "checker")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'pinned\\n'\n"), 0o751); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	snapshots, err := snapshotCheckerCommands(work, []string{tool}, []string{tool, "verify"})
	if err != nil {
		t.Fatal(err)
	}
	if snapshots[0][0] != snapshots[1][0] {
		t.Fatalf("one executable was copied more than once: %q != %q", snapshots[0][0], snapshots[1][0])
	}
	info, err := os.Stat(snapshots[0][0])
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o751 {
		t.Fatalf("snapshot mode = %o, want 751", got)
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf 'replaced\\n'\n"), 0o751); err != nil {
		t.Fatal(err)
	}
	out, err := runChecker(snapshots[0], time.Second, work)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "pinned" {
		t.Fatalf("snapshot changed with source executable: %q", out)
	}

	symlink := filepath.Join(t.TempDir(), "checker-link")
	if err := os.Symlink(tool, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotCheckerCommands(t.TempDir(), []string{symlink}); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("executable symlink was not rejected explicitly: %v", err)
	}
	if _, err := snapshotCheckerCommands(t.TempDir(), []string{t.TempDir()}); err == nil {
		t.Fatal("non-regular executable was accepted")
	}
}

func writeSparseCheckerTool(t *testing.T, path string, size int64, mode os.FileMode) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		t.Fatal(err)
	}
	err = errors.Join(file.Truncate(size), file.Close())
	if err != nil {
		t.Fatal(err)
	}
}

func writeCheckerSnapshotTestExecutable(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "checker")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSnapshotCheckerCommandsRejectsOversizedExecutableAndAssetBeforeCopy(t *testing.T) {
	oversizedExecutable := filepath.Join(t.TempDir(), "oversized-checker")
	writeSparseCheckerTool(t, oversizedExecutable, checkerToolMaxFileBytes+1, 0o755)
	if _, err := snapshotCheckerCommands(t.TempDir(), []string{oversizedExecutable}); err == nil || !strings.Contains(err.Error(), "per-file limit") {
		t.Fatalf("oversized checker executable was accepted: %v", err)
	}

	tool := writeCheckerSnapshotTestExecutable(t)
	oversizedAsset := filepath.Join(t.TempDir(), "oversized-asset")
	writeSparseCheckerTool(t, oversizedAsset, checkerToolMaxFileBytes+1, 0o600)
	if _, err := snapshotCheckerCommands(t.TempDir(), []string{tool, oversizedAsset}); err == nil || !strings.Contains(err.Error(), "per-file limit") {
		t.Fatalf("oversized checker asset was accepted: %v", err)
	}
}

func TestSnapshotCheckerCommandsRejectsOversizedAggregateDuringMetadataPreflight(t *testing.T) {
	tool := writeCheckerSnapshotTestExecutable(t)
	command := []string{tool}
	for i := range 5 {
		asset := filepath.Join(t.TempDir(), fmt.Sprintf("asset-%d", i))
		writeSparseCheckerTool(t, asset, 110<<20, 0o600)
		command = append(command, asset)
	}
	work := t.TempDir()
	if _, err := snapshotCheckerCommands(work, command); err == nil || !strings.Contains(err.Error(), "total limit") {
		t.Fatalf("oversized checker tool aggregate was accepted: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(work, "tool-snapshots")); !os.IsNotExist(err) {
		t.Fatalf("aggregate preflight copied data before rejecting inventory: %v", err)
	}
}

func TestSnapshotCheckerCommandsRejectsSourceGrowthAndPathABA(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "growth",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, writeErr := file.WriteString("growth")
				if err := errors.Join(writeErr, file.Close()); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "path ABA",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				replacement := path + ".replacement"
				if err := os.WriteFile(replacement, body, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			tool := writeCheckerSnapshotTestExecutable(t)
			prior := checkerSnapshotPoint
			checkerSnapshotPoint = func(path, point string) error {
				if path == tool && point == "after-open" {
					test.mutate(t, path)
				}
				return nil
			}
			t.Cleanup(func() { checkerSnapshotPoint = prior })
			_, err := snapshotCheckerCommands(t.TempDir(), []string{tool})
			checkerSnapshotPoint = prior
			if err == nil || !strings.Contains(err.Error(), "changed") {
				t.Fatalf("checker source %s was accepted: %v", test.name, err)
			}
		})
	}
}

func TestCheckerToolClosureHashRejectsOversizedSparseFileGrowthAndABA(t *testing.T) {
	makeClosure := func(t *testing.T) (string, string) {
		t.Helper()
		root := t.TempDir()
		for _, dir := range []string{"tool-assets", "tool-path", "tool-snapshots"} {
			if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		path := filepath.Join(root, "tool-assets", "asset")
		if err := os.WriteFile(path, []byte("stable"), 0o600); err != nil {
			t.Fatal(err)
		}
		return root, path
	}
	t.Run("oversized sparse file", func(t *testing.T) {
		root, path := makeClosure(t)
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		writeSparseCheckerTool(t, path, checkerToolMaxFileBytes+1, 0o600)
		if _, err := checkerToolClosureHash(root); err == nil || !strings.Contains(err.Error(), "per-file limit") {
			t.Fatalf("oversized closure file was accepted: %v", err)
		}
	})
	t.Run("oversized sparse aggregate", func(t *testing.T) {
		root, path := makeClosure(t)
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		for i := range 5 {
			writeSparseCheckerTool(t, filepath.Join(root, "tool-assets", fmt.Sprintf("asset-%d", i)), 110<<20, 0o600)
		}
		if _, err := checkerToolClosureHash(root); err == nil || !strings.Contains(err.Error(), "total limit") {
			t.Fatalf("oversized closure aggregate was accepted: %v", err)
		}
	})
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "growth",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
				if err != nil {
					t.Fatal(err)
				}
				_, writeErr := file.WriteString("growth")
				if err := errors.Join(writeErr, file.Close()); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "path ABA",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				replacement := path + ".replacement"
				if err := os.WriteFile(replacement, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, path := makeClosure(t)
			rel, err := filepath.Rel(root, path)
			if err != nil {
				t.Fatal(err)
			}
			prior := checkerToolClosurePoint
			checkerToolClosurePoint = func(got, point string) error {
				if got == rel && point == "after-open" {
					test.mutate(t, path)
				}
				return nil
			}
			t.Cleanup(func() { checkerToolClosurePoint = prior })
			_, err = checkerToolClosureHash(root)
			checkerToolClosurePoint = prior
			if err == nil || !strings.Contains(err.Error(), "changed") {
				t.Fatalf("closure %s was accepted: %v", test.name, err)
			}
		})
	}
}

func TestPiiFlowRuntimeClosureGolden(t *testing.T) {
	root := repoRootDir(t)
	registryPath := filepath.Join(root, "examples", "pii-flow", "checkers.local.example.yaml")
	reg, err := checker.LoadRegistry(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := reg.Resolve("pii-flow")
	if !ok {
		t.Fatal("pii-flow registry entry is missing")
	}
	inputDigests := make(map[string]string, len(entry.Runtime.Inputs))
	for _, input := range entry.Runtime.Inputs {
		body, err := os.ReadFile(filepath.Join(filepath.Dir(registryPath), filepath.FromSlash(input.Source)))
		if err != nil {
			t.Fatal(err)
		}
		inputDigests[input.Mount] = fmt.Sprintf("sha256:%x", sha256.Sum256(body))
	}
	closure, err := checker.RuntimeClosureDigest(entry.Runtime.Digest, entry.Runtime.Platform, entry.Run, entry.Verify, inputDigests)
	if err != nil {
		t.Fatal(err)
	}
	design := filepath.Join(root, "examples", "pii-flow", "design")
	manifest, err := checker.LoadManifest(filepath.Join(design, "checkers", "pii-flow.checker.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := checker.LoadEvidence(filepath.Join(design, "checkers", "pii-flow", "evidence.json"))
	if err != nil {
		t.Fatal(err)
	}
	if closure != manifest.Checker.RuntimeClosure || closure != evidence.RuntimeClosure {
		t.Fatalf("pii-flow closure = %s, manifest = %s, evidence = %s", closure, manifest.Checker.RuntimeClosure, evidence.RuntimeClosure)
	}
}

func TestVerifyCheckersPiiFlowEngineGolden(t *testing.T) {
	required := os.Getenv("MACHINERY_REQUIRE_OCI_GOLDEN") == "1"
	if !required {
		t.Skip("real OCI golden runs only in the explicitly provisioned engine lane")
	}
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Fatalf("required OCI golden has no Docker engine: %v", err)
	}
	docker, err = filepath.EvalSymlinks(docker)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(docker)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("Docker does not resolve to a snapshot-safe regular file: %v", err)
	}
	engineDir := t.TempDir()
	engineName := "docker"
	if runtime.GOOS == "windows" {
		engineName += ".exe"
	}
	engine := filepath.Join(engineDir, engineName)
	if err := os.Link(docker, engine); err != nil {
		body, readErr := os.ReadFile(docker)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if writeErr := os.WriteFile(engine, body, info.Mode().Perm()); writeErr != nil {
			t.Fatalf("copy Docker executable after hard-link failure %v: %v", err, writeErr)
		}
	}
	if os.Getenv("DOCKER_HOST") == "" {
		endpoint, inspectErr := dockerContextEndpoint(docker)
		if inspectErr != nil {
			t.Fatalf("required OCI golden cannot resolve Docker endpoint: %v", inspectErr)
		}
		t.Setenv("DOCKER_HOST", endpoint)
	}
	const image = "python@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc"
	root := repoRootDir(t)
	digest, err := checker.OCIImageDigest(image)
	if err != nil {
		t.Fatal(err)
	}
	if inspectErr := verifyLocalOCIImage([]string{docker}, image, digest, testRuntimePlatform, checkerOCIControlPlaneTimeout, root); inspectErr != nil {
		t.Fatalf("required OCI golden image is not locally provisioned: %v", inspectErr)
	}
	t.Chdir(root)
	t.Setenv("PATH", engineDir)
	registry := filepath.Join(root, "examples", "pii-flow", "checkers.local.example.yaml")

	out, errS, code := runVC(t, filepath.Join(root, "examples", "pii-flow", "design"), registry)
	if code != 0 || errS != "" {
		t.Fatalf("pii-flow engine verification failed: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
	want := "verify-checkers pii-flow: ok (verdict=pass, reproduced)\n\n1 checker(s) verified\n"
	if out != want {
		t.Fatalf("pii-flow engine output drifted:\ngot:  %q\nwant: %q", out, want)
	}

	// The pinned interpreter must not see a host-only module or child tool even
	// when both are advertised through the invoking host's ambient search paths.
	if err := os.WriteFile(filepath.Join(engineDir, "machinery_ambient_only.py"), []byte("HOST = True\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(engineDir, "machinery-host-only-tool"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PYTHONPATH", engineDir)
	isolationWork, err := os.MkdirTemp(root, ".oci-isolation-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(isolationWork); err != nil {
			t.Error(err)
		}
	})
	if err := os.Mkdir(filepath.Join(isolationWork, "runtime-inputs"), 0o700); err != nil {
		t.Fatal(err)
	}
	probe := "import importlib.util, shutil, sys; sys.exit(0 if importlib.util.find_spec('machinery_ambient_only') is None and shutil.which('machinery-host-only-tool') is None else 9)"
	if output, err := runCheckerOCI([]string{engine}, image, testRuntimePlatform, []string{"python3", "-c", probe}, testRuntimeClosure, 30*time.Second, isolationWork); err != nil {
		t.Fatalf("pinned OCI userspace resolved an ambient host dependency: %v\n%s", err, output)
	}
}

func dockerContextEndpoint(docker string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, docker, "context", "inspect", "--format", "{{(index .Endpoints \"docker\").Host}}")
	cmd.Env = os.Environ()
	stdout := boundedCheckerOutput{limit: 8 * 1024}
	stderr := boundedCheckerOutput{limit: 8 * 1024}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := processcontrol.Run(ctx, cmd)
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Docker context inspect timed out after 5s: %w", errors.Join(context.DeadlineExceeded, err))
	}
	if err != nil {
		return "", fmt.Errorf("Docker context inspect failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	endpoint := strings.TrimSpace(stdout.String())
	if endpoint == "" {
		return "", fmt.Errorf("Docker context inspect returned an empty endpoint")
	}
	return endpoint, nil
}

func TestVerifyCheckersRuntimeClosureRejectsHostDrift(t *testing.T) {
	d := setupVerifyDesign(t)
	registryDir := t.TempDir()
	asset := filepath.Join(registryDir, "rules.txt")
	if err := os.WriteFile(asset, []byte("host-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	engine := checkerFixtureEngineArgs(t, "oci-engine")
	registryBody := func(inputs string) string {
		return "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: " + engine + "\n      image: " + testRuntimeImage + "\n      platform: " + testRuntimePlatform + "\n      inputs: " + inputs + "\n    run: [\"" + stub + "\", \"{out}\"]\n"
	}
	registry := writeRawRegistryFile(t, registryDir, registryBody("[{source: rules.txt, mount: rules.txt}]"))
	if out, errS, code := runVC(t, d.dir, registry); code != 0 {
		t.Fatalf("host A fixture did not establish a reproducible closure: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}

	cases := []struct {
		name   string
		inputs string
		asset  string
	}{
		{"changed input bytes", "[{source: rules.txt, mount: rules.txt}]", "host-b\n"},
		{"missing declared input", "[]", "host-a\n"},
		{"extra declared input", "[{source: rules.txt, mount: rules.txt}, {source: extra.txt, mount: extra.txt}]", "host-a\n"},
	}
	if err := os.WriteFile(filepath.Join(registryDir, "extra.txt"), []byte("extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(asset, []byte(tc.asset), 0o644); err != nil {
				t.Fatal(err)
			}
			writeRawRegistryFile(t, registryDir, registryBody(tc.inputs))
			_, errS, code := runVCRaw(t, d.dir, registry)
			if code != 1 || !strings.Contains(errS, "local registry runtime closure") || !strings.Contains(errS, "refusing to execute a different image, command, or checker input") {
				t.Fatalf("runtime closure drift was not rejected deterministically: code=%d\nstderr=%s", code, errS)
			}
		})
	}
}

func TestVerifyCheckersRejectsWrongLocalImageIdentity(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	registry := writeRawRegistryFile(t, "", "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: "+checkerFixtureEngineArgs(t, "oci-engine-wrong-digest")+"\n      image: "+testRuntimeImage+"\n      platform: "+testRuntimePlatform+"\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, registry)
	_, errS, code := runVCRaw(t, d.dir, registry)
	if code != 1 || !strings.Contains(errS, "RepoDigests") || !strings.Contains(errS, "do not contain exact reference") {
		t.Fatalf("wrong local image identity was not rejected: code=%d\nstderr=%s", code, errS)
	}
}

func TestVerifyCheckersRejectsWrongLocalImagePlatform(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	registry := writeRawRegistryFile(t, "", "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: "+checkerFixtureEngineArgs(t, "oci-engine-wrong-platform")+"\n      image: "+testRuntimeImage+"\n      platform: "+testRuntimePlatform+"\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, registry)
	_, errS, code := runVCRaw(t, d.dir, registry)
	if code != 1 || !strings.Contains(errS, "OCI image platform linux/arm64 does not match required platform linux/amd64") {
		t.Fatalf("wrong local image platform was not rejected: code=%d\nstderr=%s", code, errS)
	}
}

func TestVerifyCheckersMissingOCIEngineIsHardError(t *testing.T) {
	d := setupVerifyDesign(t)
	registry := writeRawRegistryFile(t, "", "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: [machinery-no-such-oci-engine]\n      image: "+testRuntimeImage+"\n      platform: "+testRuntimePlatform+"\n    run: [checker, \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, registry)
	_, errS, code := runVCRaw(t, d.dir, registry)
	if code != 1 || !strings.Contains(errS, "OCI engine executable is unsafe or cannot be snapshotted") || !strings.Contains(errS, "machinery-no-such-oci-engine") {
		t.Fatalf("missing OCI engine was not a deterministic hard error: code=%d\nstderr=%s", code, errS)
	}
}

func TestVerifyCheckersCleansPrivateOCIWorkspace(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	registry := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")
	parent := filepath.Dir(d.dir)
	assertNoWorkspaces := func() {
		t.Helper()
		matches, err := filepath.Glob(filepath.Join(parent, ".machinery-verify-checkers-*"))
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 0 {
			t.Fatalf("private OCI checker workspace was not cleaned: %v", matches)
		}
	}
	assertNoWorkspaces()
	if out, errS, code := runVC(t, d.dir, registry); code != 0 {
		t.Fatalf("successful cleanup fixture failed: code=%d\nstdout=%s\nstderr=%s", code, out, errS)
	}
	assertNoWorkspaces()

	badRegistry := writeRawRegistryFile(t, "", "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: "+checkerFixtureEngineArgs(t, "oci-engine-wrong-digest")+"\n      image: "+testRuntimeImage+"\n      platform: "+testRuntimePlatform+"\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, badRegistry)
	if _, _, code := runVCRaw(t, d.dir, badRegistry); code != 1 {
		t.Fatalf("failure cleanup fixture returned %d, want 1", code)
	}
	assertNoWorkspaces()
}

func TestSnapshotOCIInputsRejectsSymlinkSpecialAndMissingSources(t *testing.T) {
	snapshot, err := designlock.AcquireReader(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := snapshot.Release(); err != nil {
			t.Error(err)
		}
	})
	registryDir := t.TempDir()
	regular := filepath.Join(registryDir, "regular.txt")
	if err := os.WriteFile(regular, []byte("pinned\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(registryDir, "link.txt")
	if err := os.Symlink(regular, symlink); err != nil {
		t.Fatal(err)
	}
	realDir := filepath.Join(registryDir, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(realDir, "nested.txt"), []byte("nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realDir, filepath.Join(registryDir, "linked-dir")); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(registryDir, "checkers.local.yaml")
	for _, tc := range []struct {
		name   string
		source string
		want   string
	}{
		{"symlink", "link.txt", "regular file, not a symlink"},
		{"symlink parent", "linked-dir/nested.txt", "parent must be a real directory"},
		{"directory", ".", "must be a regular file"},
		{"missing", "missing.txt", "no such file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var diagnostics []string
			for range 3 {
				_, err := snapshotOCIInputs(t.TempDir(), registry, []checker.OCIInput{{Source: tc.source, Mount: "input.txt"}}, snapshot)
				if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
					t.Fatalf("unsafe OCI input diagnostic = %v, want %q", err, tc.want)
				}
				diagnostics = append(diagnostics, err.Error())
			}
			if diagnostics[0] != diagnostics[1] || diagnostics[1] != diagnostics[2] {
				t.Fatalf("OCI input diagnostic varied across runs: %q", diagnostics)
			}
		})
	}
}

func TestSnapshotRootedCheckerFileSurvivesParentPathSwap(t *testing.T) {
	parent := t.TempDir()
	registryDir := filepath.Join(parent, "registry")
	if err := os.Mkdir(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(registryDir, "input.txt"), []byte("declared\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := root.Close(); err != nil {
			t.Error(err)
		}
	})
	moved := filepath.Join(parent, "registry-opened")
	if err := os.Rename(registryDir, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(registryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(registryDir, "input.txt")
	if err := os.WriteFile(outside, []byte("outside-sentinel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(t.TempDir(), "snapshot.txt")
	if err := snapshotRootedCheckerFile(root, "input.txt", destination, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "declared\n" {
		t.Fatalf("parent path swap redirected rooted checker read: %q", got)
	}
	sentinel, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(sentinel) != "outside-sentinel\n" {
		t.Fatalf("outside sentinel changed: %q", sentinel)
	}
}

const checkerProcessFixtureMarker = "--machinery-checker-process-fixture"

func TestCheckerProcessFixture(t *testing.T) {
	mode := ""
	for i := range os.Args {
		if os.Args[i] == checkerProcessFixtureMarker && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			break
		}
	}
	switch mode {
	case "oci-engine":
		runCheckerOCIEngineFixture(false, false)
	case "oci-engine-wrong-digest":
		runCheckerOCIEngineFixture(true, false)
	case "oci-engine-wrong-platform":
		runCheckerOCIEngineFixture(false, true)
	case "oversize":
		chunk := bytes.Repeat([]byte{'x'}, 64*1024)
		for range 16 {
			_, _ = os.Stdout.Write(chunk)
		}
		os.Exit(17)
	case "hold":
		time.Sleep(10 * time.Second)
	case "hold-ready":
		ready := checkerProcessFixtureArgument("hold-ready")
		if ready == "" {
			os.Exit(18)
		}
		if err := os.WriteFile(ready, []byte("ready\n"), 0o600); err != nil {
			os.Exit(18)
		}
		time.Sleep(10 * time.Second)
	case "fork":
		ready := filepath.Join(".", ".checker-descendant-ready")
		_ = os.Remove(ready)
		child := exec.CommandContext(context.Background(), os.Args[0], "-test.run=^TestCheckerProcessFixture$", "--", checkerProcessFixtureMarker, "hold-ready", ready)
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			os.Exit(18)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := os.Lstat(ready); err == nil {
				break
			} else if !errors.Is(err, os.ErrNotExist) || time.Now().After(deadline) {
				os.Exit(18)
			}
			time.Sleep(5 * time.Millisecond)
		}
	case "echo-runtime":
		cwd, _ := os.Getwd()
		fmt.Printf("cwd=%s\nargv0=%s\nhome=%s\ntmp=%s\n", cwd, os.Args[0], os.Getenv("HOME"), os.Getenv("TMPDIR"))
	case "both-streams":
		_, _ = fmt.Fprintln(os.Stderr, "stderr-marker")
		_, _ = fmt.Fprintln(os.Stdout, "stdout-marker")
	case "copy-file":
		for i := range os.Args {
			if os.Args[i] == "copy-file" && i+2 < len(os.Args) {
				data, err := os.ReadFile(os.Args[i+1])
				if err != nil {
					os.Exit(19)
				}
				if err := os.WriteFile(os.Args[i+2], data, 0o600); err != nil {
					os.Exit(20)
				}
				break
			}
		}
	case "adapter-phase":
		for i := range os.Args {
			if os.Args[i] != "adapter-phase" || i+3 >= len(os.Args) {
				continue
			}
			phase, adapter := os.Args[i+1], os.Args[i+2]
			lines := strings.Split(strings.TrimSpace(string(mustReadProcessFile(adapter))), "\n")
			if len(lines) != 2 || lines[0] != "good" {
				os.Exit(21)
			}
			if phase == "run" {
				if i+4 >= len(os.Args) {
					os.Exit(22)
				}
				data := mustReadProcessFile(os.Args[i+3])
				if err := os.WriteFile(os.Args[i+4], data, 0o600); err != nil {
					os.Exit(23)
				}
				if err := os.WriteFile(lines[1], []byte("changed\n"), 0o644); err != nil {
					os.Exit(24)
				}
			}
			break
		}
	}
}

func checkerProcessFixtureArgument(mode string) string {
	for i := range os.Args {
		if os.Args[i] == checkerProcessFixtureMarker && i+2 < len(os.Args) && os.Args[i+1] == mode {
			return os.Args[i+2]
		}
	}
	return ""
}

func runCheckerOCIEngineFixture(wrongDigest, wrongPlatform bool) {
	args := os.Args
	imageAt := -1
	work := ""
	platform := ""
	env := map[string]string{}
	for i := range args {
		if args[i] == "--platform" && i+1 < len(args) {
			platform = args[i+1]
		}
		if args[i] == "--mount" && i+1 < len(args) && strings.Contains(args[i+1], "dst=/work") {
			for _, field := range strings.Split(args[i+1], ",") {
				if strings.HasPrefix(field, "src=") {
					work = strings.TrimPrefix(field, "src=")
				}
			}
		}
		if args[i] == "--env" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
		if strings.Contains(args[i], "@sha256:") {
			imageAt = i
			break
		}
	}
	if imageAt >= 0 && imageAt > 1 && args[imageAt-2] == "--format" {
		repoDigest := args[imageAt]
		if wrongDigest {
			repoDigest = strings.SplitN(repoDigest, "@", 2)[0] + "@sha256:" + strings.Repeat("f", 64)
		}
		encoded, _ := json.Marshal([]string{repoDigest})
		_, _ = os.Stdout.Write(append(encoded, '\n'))
		imagePlatform := platform
		if wrongPlatform {
			imagePlatform = "linux/arm64"
		}
		imageOS, architecture, ok := strings.Cut(imagePlatform, "/")
		if !ok {
			os.Exit(31)
		}
		_, _ = fmt.Fprintf(os.Stdout, "%q\n%q\n", imageOS, architecture)
		os.Exit(0)
	}
	if imageAt < 0 || imageAt+1 >= len(args) || work == "" || platform != testRuntimePlatform {
		os.Exit(31)
	}
	inner := append([]string(nil), args[imageAt+1:]...)
	for i, arg := range inner {
		if arg == "/work" {
			inner[i] = work
		} else if strings.HasPrefix(arg, "/work/") {
			inner[i] = filepath.Join(work, filepath.FromSlash(strings.TrimPrefix(arg, "/work/")))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, inner[0], inner[1:]...)
	cmd.Dir = work
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	blocked := map[string]bool{}
	for _, key := range keys {
		blocked[key] = true
	}
	for _, pair := range os.Environ() {
		key := strings.SplitN(pair, "=", 2)[0]
		if !blocked[key] {
			cmd.Env = append(cmd.Env, pair)
		}
	}
	for _, key := range keys {
		cmd.Env = append(cmd.Env, key+"="+env[key])
	}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(32)
	}
	os.Exit(0)
}

func mustReadProcessFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		os.Exit(25)
	}
	return data
}

func checkerProcessFixtureCommand(t *testing.T, mode string) []string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return []string{executable, "-test.run=^TestCheckerProcessFixture$", "--", checkerProcessFixtureMarker, mode}
}

func TestRunCheckerBoundsOutput(t *testing.T) {
	out, err := runChecker(checkerProcessFixtureCommand(t, "oversize"), 5*time.Second, t.TempDir())
	if err == nil {
		t.Fatal("oversized failing checker unexpectedly succeeded")
	}
	if !strings.Contains(out, "checker output truncated") || !strings.Contains(out, "byte(s) omitted") {
		t.Fatalf("truncation was not diagnosed: len=%d suffix=%q", len(out), out[max(0, len(out)-100):])
	}
	if len(out) > checkerOutputLimit+128 {
		t.Fatalf("captured output exceeded bound: %d", len(out))
	}
}

func TestRunCheckerReportsStreamsInDeterministicOrder(t *testing.T) {
	out, err := runChecker(checkerProcessFixtureCommand(t, "both-streams"), 5*time.Second, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	stdoutAt, stderrAt := strings.Index(out, "stdout-marker"), strings.Index(out, "stderr-marker")
	if stdoutAt < 0 || stderrAt < 0 || stdoutAt > stderrAt {
		t.Fatalf("stdout and stderr were not reported in canonical order: %q", out)
	}
}

func TestRunCheckerBoundsDescendantPipeWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix process-group termination assertion")
	}
	started := time.Now()
	_, err := runChecker(checkerProcessFixtureCommand(t, "fork"), 5*time.Second, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "descendant held output pipes open") {
		t.Fatalf("descendant pipe retention was not diagnosed: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("descendant pipe retention exceeded bounded cleanup delay: %s", elapsed)
	}
}

func TestRunCheckerTimeoutDiagnostic(t *testing.T) {
	started := time.Now()
	_, err := runChecker(checkerProcessFixtureCommand(t, "hold"), 100*time.Millisecond, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "timed out after 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("timeout diagnostic was incomplete: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("checker timeout exceeded bounded cleanup: %s", elapsed)
	}
}

func TestVerifyLocalOCIImageBoundsUnresponsiveEngine(t *testing.T) {
	started := time.Now()
	err := verifyLocalOCIImage(checkerProcessFixtureCommand(t, "hold"), testRuntimeImage, testRuntimeClosure, testRuntimePlatform, 100*time.Millisecond, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "timed out after 100ms") || !strings.Contains(err.Error(), "process tree was terminated") {
		t.Fatalf("unresponsive OCI image inspection was not diagnosed: %v", err)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("unresponsive OCI image inspection exceeded bounded cleanup: %s", elapsed)
	}
}

func TestRunCheckerRedactsPrivateNonceAndUsesLogicalEnvironment(t *testing.T) {
	var outputs []string
	for range 2 {
		work := t.TempDir()
		command := checkerProcessFixtureCommand(t, "echo-runtime")
		snapshots, err := snapshotCheckerCommands(work, command)
		if err != nil {
			t.Fatal(err)
		}
		out, err := runChecker(snapshots[0], 5*time.Second, work)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, work) || !strings.Contains(out, "cwd=.") || !strings.Contains(out, "home=home") || !strings.Contains(out, "tmp=tmp") {
			t.Fatalf("checker runtime exposed its private nonce or unstable environment:\n%s", out)
		}
		outputs = append(outputs, out)
	}
	if outputs[0] != outputs[1] {
		t.Fatalf("equivalent checker runtimes produced different observable paths:\nfirst: %q\nnext:  %q", outputs[0], outputs[1])
	}
}

func TestVerifyCheckerScratchCrashHelper(t *testing.T) {
	if os.Getenv(checkerScratchCrashEnv) != "1" {
		return
	}
	if code := verifyCheckers(os.Getenv(checkerScratchCrashDesignEnv), os.Getenv(checkerScratchCrashRegistry), ""); code != 0 {
		t.Fatalf("verify-checkers helper exited %d", code)
	}
}

func TestCreateCheckerWorkDirRejectsTempRootInsideDesign(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows resolves its system temp root independently of TMPDIR")
	}
	design := t.TempDir()
	t.Setenv("TMPDIR", design)
	work, err := createCheckerWorkDir(design)
	if err == nil || !strings.Contains(err.Error(), "must be outside the design root") {
		if work != "" {
			_ = os.RemoveAll(work)
		}
		t.Fatalf("design-local temp root was accepted: work=%q err=%v", work, err)
	}
}

func TestVerifyCheckerScratchSurvivesCrashOutsideDesignSnapshots(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process kill/orphan inspection uses Unix subprocess semantics")
	}
	d := setupVerifyDesign(t)
	control := t.TempDir()
	started := filepath.Join(control, "started")
	release := filepath.Join(control, "release")
	childDone := filepath.Join(control, "child-done")
	stub := writeScript(t, "/usr/bin/touch \""+started+"\"\nwhile [ ! -f \""+release+"\" ]; do /bin/sleep 0.01; done\n/bin/cp \""+d.evPath+"\" \"$1\"\n/usr/bin/touch \""+childDone+"\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, reg)

	report := filepath.Join(control, "helper-control-root")
	fakeUserRoot := filepath.Join(control, "fake-user")
	if err := os.Mkdir(fakeUserRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestVerifyCheckerScratchCrashHelper$", "-test.count=1")
	command.Env = append(cmdTestSubprocessEnvironment(fakeUserRoot, report),
		checkerScratchCrashEnv+"=1",
		checkerScratchCrashDesignEnv+"="+d.dir,
		checkerScratchCrashRegistry+"="+reg,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, report)
	waitForFile(t, started)

	helperRootBytes, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	helperRoot := strings.TrimSpace(string(helperRootBytes))
	t.Cleanup(func() {
		if err := os.RemoveAll(helperRoot); err != nil {
			t.Error(err)
		}
	})
	scratch, err := filepath.Glob(filepath.Join(helperRoot, "temp", "machinery-verify-checkers-*"))
	if err != nil || len(scratch) != 1 {
		t.Fatalf("active checker scratch = %v, %v", scratch, err)
	}
	designParent := filepath.Dir(d.dir)
	if err := filepath.WalkDir(designParent, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.Contains(entry.Name(), "machinery-verify-checkers-") {
			return fmt.Errorf("checker scratch entered sibling snapshot at %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("crash helper exited successfully after kill")
	}
	if err := os.WriteFile(release, []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, childDone)
	if _, err := os.Lstat(scratch[0]); err != nil {
		t.Fatalf("SIGKILL scratch orphan was not available for isolation proof: %v", err)
	}
	if adjacent, err := filepath.Glob(filepath.Join(designParent, ".machinery-verify-checkers-*")); err != nil || len(adjacent) != 0 {
		t.Fatalf("checker scratch leaked beside design after crash: %v, %v", adjacent, err)
	}
}

func TestVerifyCheckersHoldsSnapshotUntilAdapterCompletes(t *testing.T) {
	d := setupVerifyDesign(t)
	control := t.TempDir()
	started := filepath.Join(control, "started")
	release := filepath.Join(control, "release")
	stub := writeScript(t, "/usr/bin/touch \""+started+"\"\nwhile [ ! -f \""+release+"\" ]; do /bin/sleep 0.01; done\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, reg)

	_, _, _ = withCapturedIO(t)
	verifyDone := make(chan int, 1)
	go func() { verifyDone <- verifyCheckers(d.dir, reg, "") }()
	waitForFile(t, started)

	projectDone := make(chan error, 1)
	go func() {
		_, err := checker.ProjectAll(d.dir, machversion.Version)
		projectDone <- err
	}()
	select {
	case err := <-projectDone:
		t.Fatalf("ProjectAll completed while verifier held the reader snapshot: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
	if err := os.WriteFile(release, []byte("release\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-verifyDone:
		if code != 0 {
			t.Fatalf("verifier failed after releasing adapter: %d", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("verifier did not complete")
	}
	select {
	case err := <-projectDone:
		if err != nil {
			t.Fatalf("ProjectAll failed after reader released: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ProjectAll remained blocked after verifier released")
	}
}

func TestVerifyCheckersReadsImmutableSnapshotAcrossABAMutation(t *testing.T) {
	d := setupVerifyDesign(t)
	modelPath := filepath.Join(d.dir, "d.modelith.yaml")
	manifestPath := filepath.Join(d.dir, "checkers", "test.checker.yaml")
	toolPath := filepath.Join(d.dir, "tool-input.txt")
	if err := os.WriteFile(toolPath, []byte("tool-a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	control := t.TempDir()
	started := filepath.Join(control, "started")
	release := filepath.Join(control, "release")
	stub := writeScript(t, "/usr/bin/touch \""+started+"\"\nwhile [ ! -f \""+release+"\" ]; do /bin/sleep 0.01; done\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
	inputSource := filepath.ToSlash(filepath.Join(filepath.Base(d.dir), filepath.Base(toolPath)))
	registry := writeRawRegistryFile(t, filepath.Dir(d.dir), "checkers:\n  test:\n    runtime:\n      kind: oci\n      engine: "+checkerFixtureEngineArgs(t, "oci-engine")+"\n      image: "+testRuntimeImage+"\n      platform: "+testRuntimePlatform+"\n      inputs: [{source: "+inputSource+", mount: tool-input.txt}]\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, registry)

	originals := map[string][]byte{}
	for _, path := range []string{manifestPath, modelPath, d.evPath, toolPath} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		originals[path] = body
	}
	snapshotReady := make(chan struct{})
	mutationReady := make(chan struct{})
	priorHook := verifyCheckersAfterRegistrySnapshot
	verifyCheckersAfterRegistrySnapshot = func() {
		close(snapshotReady)
		<-mutationReady
	}
	t.Cleanup(func() { verifyCheckersAfterRegistrySnapshot = priorHook })

	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- verifyCheckersTo(d.dir, registry, "", &stdout, &stderr) }()
	<-snapshotReady
	for path, body := range map[string][]byte{
		manifestPath: []byte("not: [a valid checker manifest\n"),
		modelPath:    []byte("not: [a valid domain model\n"),
		d.evPath:     []byte("{not valid evidence\n"),
		toolPath:     []byte("tool-b\n"),
	} {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	close(mutationReady)
	select {
	case code := <-done:
		t.Fatalf("verify-checkers failed before adapter under ABA mutation: code=%d\nstdout=%s\nstderr=%s", code, &stdout, &stderr)
	case <-time.After(100 * time.Millisecond):
	}
	waitForFile(t, started)
	for path, body := range originals {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(release, []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "verify-checkers test: ok") {
			t.Fatalf("ABA mutation escaped immutable snapshot: code=%d\nstdout=%s\nstderr=%s", code, &stdout, &stderr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("verify-checkers did not finish after restoring ABA mutation")
	}
}

func TestVerifyCheckersReadsMaterializedExternalRegistryAcrossABAMutation(t *testing.T) {
	d := setupVerifyDesign(t)
	control := t.TempDir()
	started := filepath.Join(control, "started")
	release := filepath.Join(control, "release")
	stub := writeScript(t, "/usr/bin/touch \""+started+"\"\nwhile [ ! -f \""+release+"\" ]; do /bin/sleep 0.01; done\n/bin/cp \""+d.evPath+"\" \"$1\"\n")
	registry := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")
	syncTestRuntimeClosure(t, d.dir, registry)
	original, err := os.ReadFile(registry)
	if err != nil {
		t.Fatal(err)
	}

	registryReady := make(chan struct{})
	mutationReady := make(chan struct{})
	priorHook := verifyCheckersAfterRegistrySnapshot
	verifyCheckersAfterRegistrySnapshot = func() {
		close(registryReady)
		<-mutationReady
	}
	t.Cleanup(func() { verifyCheckersAfterRegistrySnapshot = priorHook })

	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	go func() { done <- verifyCheckersTo(d.dir, registry, "", &stdout, &stderr) }()
	<-registryReady
	if err := os.WriteFile(registry, []byte("not: [a valid registry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	close(mutationReady)
	waitForFile(t, started)
	if err := os.WriteFile(registry, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(release, []byte("release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {
	case code := <-done:
		if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "verify-checkers test: ok") {
			t.Fatalf("external registry ABA escaped stable materialization: code=%d\nstdout=%s\nstderr=%s", code, &stdout, &stderr)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("verify-checkers did not finish after restoring external registry ABA")
	}
}

func TestVerifyCheckersRejectsNoncooperativeEditDuringAdapter(t *testing.T) {
	d := setupVerifyDesign(t)
	model := filepath.Join(d.dir, "d.modelith.yaml")
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\nprintf '\\n# noncooperative edit\\n' >> \""+model+"\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("noncooperative edit escaped final snapshot check: code=%d\nstderr=%s", code, errS)
	}
	if !strings.Contains(errS, "design snapshot was not stable") || !strings.Contains(errS, "d.modelith.yaml") {
		t.Fatalf("snapshot mutation diagnostic is incomplete:\n%s", errS)
	}
	if strings.Contains(out, "verify-checkers test: ok") || strings.Contains(out, "checker(s) verified") {
		t.Fatalf("unstable snapshot emitted success output:\n%s", out)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func TestReproDiffCoversCompleteEvidenceSemantics(t *testing.T) {
	base := &checker.Evidence{
		EvidenceSchema: "1.0",
		InputHash:      "sha256:" + strings.Repeat("a", 64),
		Verdict:        "pass",
		Coverage:       []checker.CoverageRow{{Element: "inv:x", Verdict: "pass", Detail: "checked"}},
		Findings:       []checker.Finding{{Severity: "info", Code: "I1", Element: "inv:x", Message: "ok", Locator: "r:1"}},
		Attestation:    json.RawMessage(`{"proof":"abc","nested":{"n":1}}`),
		TraceRef:       "trace.json",
	}
	base.Checker.ID, base.Checker.Version = "test", "1.2.3"
	clone := func() *checker.Evidence {
		b, _ := json.Marshal(base)
		var out checker.Evidence
		_ = json.Unmarshal(b, &out)
		return &out
	}
	mutations := map[string]func(*checker.Evidence){
		"schema":          func(e *checker.Evidence) { e.EvidenceSchema = "2.0" },
		"checker id":      func(e *checker.Evidence) { e.Checker.ID = "other" },
		"checker version": func(e *checker.Evidence) { e.Checker.Version = "9" },
		"coverage detail": func(e *checker.Evidence) { e.Coverage[0].Detail = "different" },
		"finding locator": func(e *checker.Evidence) { e.Findings[0].Locator = "r:2" },
		"attestation":     func(e *checker.Evidence) { e.Attestation = json.RawMessage(`{"proof":"changed"}`) },
		"trace ref":       func(e *checker.Evidence) { e.TraceRef = "other.json" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			fresh := clone()
			mutate(fresh)
			if diff := reproDiff(base, fresh); diff == "" {
				t.Fatal("semantic evidence mutation reproduced")
			}
		})
	}
	// Object key order is JSON syntax, not semantics.
	fresh := clone()
	fresh.Attestation = json.RawMessage(`{"nested":{"n":1},"proof":"abc"}`)
	if diff := reproDiff(base, fresh); diff != "" {
		t.Fatalf("attestation object reordering moved semantics: %s", diff)
	}
	fresh = clone()
	base.Attestation = json.RawMessage(`{"integer":9007199254740992}`)
	fresh.Attestation = json.RawMessage(`{"integer":9007199254740993}`)
	if diff := reproDiff(base, fresh); diff == "" {
		t.Fatal("attestation integers beyond float64 precision falsely reproduced")
	}
}

// TestVerifyCheckersCheckerFilter: --checker selects a single checker; an
// unknown id is an error.
func TestVerifyCheckersCheckerFilter(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "/bin/cp \""+d.evPath+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg, "--checker", "test")
	if code != 0 {
		t.Fatalf("--checker test should exit 0, got %d\nstderr: %s", code, errS)
	}
	if !strings.Contains(out, "verify-checkers test: ok") {
		t.Fatalf("expected the selected checker to verify, got:\n%s", out)
	}

	_, errS, code = runVC(t, d.dir, reg, "--checker", "nope")
	if code != 1 {
		t.Fatalf("--checker nope should exit 1, got %d", code)
	}
	if !strings.Contains(errS, "no checker with id \"nope\"") {
		t.Fatalf("expected an unknown-checker ERROR, got:\n%s", errS)
	}
}

// TestVerifyCheckersNoManifests: a design with no checker manifests is an error.
func TestVerifyCheckersNoManifests(t *testing.T) {
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\"/bin/true\"]\n")
	_, errS, code := runVC(t, t.TempDir(), reg)
	if code != 1 {
		t.Fatalf("no manifests must exit 1, got %d", code)
	}
	if !strings.Contains(errS, "no checkers/*.checker.yaml") {
		t.Fatalf("expected a no-manifests ERROR, got:\n%s", errS)
	}
}

// TestReportCheckerBinaries covers the doctor probe: silent with no registry
// (the byte-for-byte invariant), present/missing lines with one, and the
// unreadable and empty cases.
func TestReportCheckerBinaries(t *testing.T) {
	// No registry in cwd: the probe must emit nothing.
	t.Chdir(t.TempDir())
	var buf strings.Builder
	reportCheckerBinaries(&buf)
	if buf.String() != "" {
		t.Fatalf("with no registry the probe must be silent, got:\n%s", buf.String())
	}

	// A registry with one present and one missing OCI engine. Container commands
	// are deliberately not resolved on the host.
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("PATH", filepath.Dir(goldenBin(t))+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.MkdirAll(filepath.Join(dir, ".machinery"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := "checkers:\n  present:\n    runtime: {kind: oci, engine: [machinery], image: example.invalid/present@" + testRuntimeClosure + ", platform: " + testRuntimePlatform + "}\n    run: [checker, \"{out}\"]\n  absent:\n    runtime: {kind: oci, engine: [machinery-no-such-engine-xyz], image: example.invalid/absent@" + testRuntimeClosure + ", platform: " + testRuntimePlatform + "}\n    run: [checker, \"{out}\"]\n    verify: [checker, verify]\n"
	if err := os.WriteFile(checker.DefaultRegistryPath, []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	reportCheckerBinaries(&buf)
	got := buf.String()
	if !strings.Contains(got, "present  checker present OCI engine machinery is snapshot-safe") || !strings.Contains(got, "not executed by doctor") {
		t.Errorf("present OCI engine not reported ok:\n%s", got)
	}
	if !strings.Contains(got, "MISSING  checker absent OCI engine machinery-no-such-engine-xyz") {
		t.Errorf("missing OCI engine not reported MISSING:\n%s", got)
	}

	// A malformed registry reports MISSING/unreadable, never a silent skip.
	if err := os.WriteFile(checker.DefaultRegistryPath, []byte("checkers:\n  bad:\n    verify: [x]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	reportCheckerBinaries(&buf)
	if !strings.Contains(buf.String(), "is unreadable") {
		t.Errorf("an unreadable registry must be reported:\n%s", buf.String())
	}
}

// TestVerifyCheckersMissingRegistryFile: no registry at the given path -> ERROR,
// exit 1, with a message that points at the missing file.
func TestVerifyCheckersMissingRegistryFile(t *testing.T) {
	d := setupVerifyDesign(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist.yaml")

	out, errS, code := runVC(t, d.dir, missing)
	if code != 1 {
		t.Fatalf("a missing registry must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "no checker registry at") {
		t.Fatalf("expected a missing-registry ERROR, got:\n%s", errS)
	}
}
