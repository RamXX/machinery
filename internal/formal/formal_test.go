package formal

import (
	"os"
	"path/filepath"
	"testing"
)

// Neither test needs java: both fail before any TLC invocation.

func TestVerifyFormalFailsWhenNothingToCheck(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := VerifyFormal(design); got != 1 {
		t.Fatalf("empty formal suite returned %d, want 1 (nothing to check is a failure)", got)
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
	if got := VerifyFormal(design); got != 1 {
		t.Fatalf("generator failure returned %d, want 1", got)
	}
}
