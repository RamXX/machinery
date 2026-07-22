package formal

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureOutput runs f with os.Stdout and os.Stderr redirected and returns
// what was written to each.
func captureOutput(t *testing.T, f func()) (stdout, stderr string) {
	t.Helper()
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = ow, ew
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	outCh := make(chan string)
	errCh := make(chan string)
	go func() { b, _ := io.ReadAll(or); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(er); errCh <- string(b) }()
	f()
	ow.Close()
	ew.Close()
	return <-outCh, <-errCh
}

// None of these tests needs java: the failure cases fail before any TLC
// invocation, and gen-only never invokes TLC at all.

func TestVerifyFormalFailsWhenNothingToCheck(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, false); got != 1 {
		t.Fatalf("empty formal suite returned %d, want 1 (nothing to check is a failure)", got)
	}
	if got := VerifyFormal(design, true); got != 1 {
		t.Fatalf("empty formal suite returned %d in gen-only mode, want 1 (nothing to generate is a failure)", got)
	}
}

func TestVerifyFormalFailsOnGeneratorError(t *testing.T) {
	// A machine tla_gen cannot translate (nested states) must fail the run,
	// never be skipped while stale committed specs are checked as fresh.
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	nested := `{"id":"broken","initial":"A","states":{"A":{"initial":"B","states":{"B":{"type":"final"}}}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Broken.machine.json"), []byte(nested), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, false); got != 1 {
		t.Fatalf("generator failure returned %d, want 1", got)
	}
	if got := VerifyFormal(design, true); got != 1 {
		t.Fatalf("generator failure returned %d in gen-only mode, want 1", got)
	}
}

func TestVerifyFormalGenOnlyRegeneratesWithoutTLC(t *testing.T) {
	// gen-only must succeed with no java in the loop: specs regenerated, TLC
	// skipped, exit 0. This is the nightly regen gate's code path.
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design, true); got != 0 {
		t.Fatalf("gen-only on a valid machine returned %d, want 0", got)
	}
	for _, f := range []string{"Toy.tla", "Toy.cfg"} {
		if _, err := os.Stat(filepath.Join(design, "formal", f)); err != nil {
			t.Fatalf("gen-only did not regenerate %s: %v", f, err)
		}
	}
}

// FORMAL-F5: gen-only counted every committed .tla/.cfg pair as "regenerated
// from source"; an orphan pair with no source inflated the count and a
// zero-machine design exited 0 claiming a pair was regenerated.

func TestVerifyFormalGenOnlyReportsOnlyWrittenPairs(t *testing.T) {
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	// an orphan committed pair no generator produces (reviewer fixture exp-c)
	stale := "---- MODULE Stale ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var rc int
	stdout, stderr := captureOutput(t, func() { rc = VerifyFormal(design, true) })
	if rc != 0 {
		t.Fatalf("gen-only returned %d, want 0 (an orphan pair warns, it does not fail)", rc)
	}
	if !strings.Contains(stdout, "1 spec pair(s) regenerated") {
		t.Errorf("gen-only counted committed pairs as regenerated:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Stale") || !strings.Contains(stderr, "not regenerated (no source)") {
		t.Errorf("orphan pair not warned about:\nstdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestVerifyFormalNothingToGenerateIsHardError(t *testing.T) {
	// zero machines AND zero relational annotations: even with a committed
	// orphan pair on disk, there is nothing to generate; exit hard.
	design := t.TempDir()
	fdir := filepath.Join(design, "formal")
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fdir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := "---- MODULE Stale ----\nVARIABLE x\nInit == x = 0\nNext == x' = x\nSpec == Init /\\ [][Next]_x\n====\n"
	if err := os.WriteFile(filepath.Join(fdir, "Stale.tla"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fdir, "Stale.cfg"), []byte("SPECIFICATION Spec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, genOnly := range []bool{true, false} {
		var rc int
		_, stderr := captureOutput(t, func() { rc = VerifyFormal(design, genOnly) })
		if rc != 1 {
			t.Fatalf("genOnly=%v: returned %d, want 1", genOnly, rc)
		}
		if !strings.Contains(stderr, "nothing to generate") {
			t.Errorf("genOnly=%v: stderr does not say nothing to generate:\n%s", genOnly, stderr)
		}
	}
}

// FORMAL-F6: runTLC's error was discarded; an infrastructure failure (missing
// java, missing jar, timeout) printed a bare FAIL with zero diagnostics.
func TestVerifyFormalPrintsTLCInfrastructureError(t *testing.T) {
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	flat := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(mdir, "Toy.machine.json"), []byte(flat), 0o644); err != nil {
		t.Fatal(err)
	}
	// a dummy jar so ensureJar succeeds without a network fetch, and an empty
	// PATH so the java invocation itself fails: the pure infrastructure case
	dummyJar := filepath.Join(design, "dummy.jar")
	if err := os.WriteFile(dummyJar, []byte("not a jar"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TLA_TOOLS_JAR", dummyJar)
	t.Setenv("PATH", filepath.Join(design, "empty-path"))
	var rc int
	stdout, _ := captureOutput(t, func() { rc = VerifyFormal(design, false) })
	if rc != 1 {
		t.Fatalf("returned %d, want 1", rc)
	}
	if !strings.Contains(stdout, "FAIL") {
		t.Fatalf("no FAIL line:\n%s", stdout)
	}
	if !strings.Contains(stdout, "error:") || !strings.Contains(stdout, "java") {
		t.Errorf("infrastructure error not printed alongside the FAIL:\n%s", stdout)
	}
}
