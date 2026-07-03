package formal

import (
	"os"
	"path/filepath"
	"testing"
)

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
