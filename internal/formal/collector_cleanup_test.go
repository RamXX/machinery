package formal

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyFormalFailsClosedWhenCollectorCleanupFails(t *testing.T) {
	design := t.TempDir()
	machineDir := filepath.Join(design, "machines")
	if err := os.MkdirAll(machineDir, 0o755); err != nil {
		t.Fatal(err)
	}
	machine := `{"id":"Toy","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(machineDir, "Toy.machine.json"), []byte(machine), 0o644); err != nil {
		t.Fatal(err)
	}

	previousRemove := removeArtifactCollectorRoot
	var retainedRoot string
	removeArtifactCollectorRoot = func(root string) error {
		retainedRoot = root
		return errors.New("injected cleanup failure")
	}
	t.Cleanup(func() {
		removeArtifactCollectorRoot = previousRemove
		if retainedRoot != "" {
			if err := os.RemoveAll(retainedRoot); err != nil {
				t.Errorf("remove retained formal staging directory: %v", err)
			}
		}
	})

	var stdout, stderr bytes.Buffer
	if got := VerifyFormalTo(design, true, &stdout, &stderr); got != 1 {
		t.Fatalf("VerifyFormalTo returned %d after staging cleanup failure, want 1; stdout=%q stderr=%q", got, stdout.String(), stderr.String())
	}
	if got := stderr.String(); got != "verify-formal: remove generator staging directory: injected cleanup failure\n" {
		t.Fatalf("unexpected cleanup diagnostic: %q", got)
	}
	if retainedRoot == "" {
		t.Fatal("collector cleanup hook was not called")
	}
	if strings.Contains(stderr.String(), retainedRoot) {
		t.Fatalf("cleanup diagnostic exposed private staging path %q: %q", retainedRoot, stderr.String())
	}
}
