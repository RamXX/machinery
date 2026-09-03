//go:build windows

package formal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsFormalDirectoryDurabilityAndRealRecovery(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := syncFormalDirectory(root); err != nil {
		t.Fatalf("native directory flush failed: %v", err)
	}
	files := map[string]generatedArtifact{"A.tla": {body: []byte("new"), owner: "windows-test"}}
	if err := commitGeneratedArtifacts(dir, files); err != nil {
		t.Fatalf("native durable commit failed: %v", err)
	}
	entries := []formalJournalEntry{formalRecoveryEntry("A.tla", true, "new", "interrupted")}
	if err := os.Rename(filepath.Join(dir, "A.tla"), filepath.Join(dir, entries[0].Backup)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, entries[0].Stage), []byte("interrupted"), 0o644); err != nil {
		t.Fatal(err)
	}
	hydrateFormalRecoveryWitnesses(t, dir, entries)
	if err := createFormalJournal(root, entries); err != nil {
		t.Fatal(err)
	}
	if err := appendFormalPhase(root, "parking"); err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireDesignLock(dir)
	if err != nil {
		t.Fatalf("native recovery failed: %v", err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "A.tla"))
	if err != nil || string(got) != "new" {
		t.Fatalf("native recovery restored %q, %v", got, err)
	}
}
