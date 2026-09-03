//go:build windows

package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsPackDirectoryDurabilityAndRealRecovery(t *testing.T) {
	design := t.TempDir()
	if _, err := writePacksWithRename(design, map[string]map[string]string{
		"orders": {"pack.yaml": "committed"},
	}, nil); err != nil {
		t.Fatalf("native durable pack commit failed: %v", err)
	}
	root := filepath.Join(design, "packs")
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := syncPackDirectory(rootHandle); err != nil {
		t.Fatalf("native directory flush failed: %v", err)
	}
	target := "orders.pack"
	entry := packJournalEntry{
		Target: target, Stage: packScratchName("stage", target),
		Backup: packScratchName("backup", target), Retire: packScratchName("retire", target), Existed: true,
	}
	backup := filepath.Join(root, entry.Backup)
	if err := os.Rename(filepath.Join(root, target), backup); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, entry.Stage)
	if err := os.Mkdir(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	before, err := capturePackTree(rootHandle, entry.Backup)
	if err != nil {
		t.Fatal(err)
	}
	after, err := capturePackTree(rootHandle, entry.Stage)
	if err != nil {
		t.Fatal(err)
	}
	entry.Before = before.witness
	entry.AfterTree = after.witness.Tree
	if err := createPackJournal(rootHandle, []packJournalEntry{entry}); err != nil {
		t.Fatal(err)
	}
	if err := appendPackStaged(rootHandle, []packStagedEntry{{Target: target, After: after.witness}}); err != nil {
		t.Fatal(err)
	}
	if err := appendPackPhase(rootHandle, "parking"); err != nil {
		t.Fatal(err)
	}
	if err := rootHandle.Close(); err != nil {
		t.Fatal(err)
	}
	lock, err := acquirePackWriteLock(root)
	if err != nil {
		t.Fatalf("native recovery failed: %v", err)
	}
	if err := lock.releaseAll(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, target, "pack.yaml"))
	if err != nil || string(got) != "committed" {
		t.Fatalf("native recovery restored %q, %v", got, err)
	}
}
