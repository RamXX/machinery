package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
	machversion "github.com/RamXX/machinery/internal/version"
)

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

const vcManifest = `checker: {id: test}
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
	p := filepath.Join(t.TempDir(), "checkers.local.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// runVC runs verify-checkers with captured IO, returning stdout, stderr, and
// the exit code (0 when exitFunc was never called).
func runVC(t *testing.T, design, registry string, extra ...string) (string, string, int) {
	t.Helper()
	out, errB, codes := withCapturedIO(t)
	cmd := newVerifyCheckersCmd()
	cmd.SetArgs(append([]string{design, "--registry", registry}, extra...))
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	code := 0
	if len(*codes) > 0 {
		code = (*codes)[len(*codes)-1]
	}
	return out.String(), errB.String(), code
}

// TestVerifyCheckersReproducible: a stub that reproduces the committed evidence
// yields ok and exit 0, and machinery hands the adapter the manifest config as
// JSON via {config}.
func TestVerifyCheckersReproducible(t *testing.T) {
	d := setupVerifyDesign(t)
	configSink := filepath.Join(t.TempDir(), "seen-config.json")
	stub := writeScript(t, "cp \"$1\" \""+configSink+"\"\ncp \""+d.evPath+"\" \"$2\"\n")
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
	fixture := filepath.Join(t.TempDir(), "fresh.json")
	writeJSON(t, fixture, ev)

	stub := writeScript(t, "cp \""+fixture+"\" \"$1\"\n")
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
	stub := writeScript(t, "cp \""+d.evPath+"\" \"$1\"\n")
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
	runStub := writeScript(t, "cp \""+d.evPath+"\" \"$1\"\n")
	verifyStub := writeScript(t, "exit 1\n")
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

	stub := writeScript(t, "cp \""+fixture+"\" \"$1\"\n")
	reg := writeRegistryFile(t, "checkers:\n  test:\n    run: [\""+stub+"\", \"{out}\"]\n")

	out, errS, code := runVC(t, d.dir, reg)
	if code != 1 {
		t.Fatalf("an input_hash mismatch must exit 1, got %d\nstdout: %s", code, out)
	}
	if !strings.Contains(errS, "not reproducible") || !strings.Contains(errS, "input_hash") {
		t.Fatalf("expected an input_hash reproducibility ERROR, got:\n%s", errS)
	}
}

// TestVerifyCheckersCheckerFilter: --checker selects a single checker; an
// unknown id is an error.
func TestVerifyCheckersCheckerFilter(t *testing.T) {
	d := setupVerifyDesign(t)
	stub := writeScript(t, "cp \""+d.evPath+"\" \"$1\"\n")
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

	// A registry with one present (sh) and one missing adapter.
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(filepath.Join(dir, ".machinery"), 0o755); err != nil {
		t.Fatal(err)
	}
	reg := "checkers:\n  present:\n    run: [\"sh\", \"{out}\"]\n  absent:\n    run: [\"machinery-no-such-adapter-xyz\", \"{out}\"]\n"
	if err := os.WriteFile(checker.DefaultRegistryPath, []byte(reg), 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	reportCheckerBinaries(&buf)
	got := buf.String()
	if !strings.Contains(got, "ok       checker present adapter sh") {
		t.Errorf("present adapter not reported ok:\n%s", got)
	}
	if !strings.Contains(got, "MISSING  checker absent adapter machinery-no-such-adapter-xyz") {
		t.Errorf("missing adapter not reported MISSING:\n%s", got)
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
