//go:build windows

package artifactset

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestWindowsNativeCommitAndSeededRecovery(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{"A": "old-a", "B": "old-b"})
	if err := Commit(dir, map[string][]byte{"A": []byte("first-a"), "B": []byte("first-b")}); err != nil {
		t.Fatalf("native Windows commit: %v", err)
	}
	assertFile(t, filepath.Join(dir, "A"), "first-a")
	assertFile(t, filepath.Join(dir, "B"), "first-b")

	ops := txDefaultOps(nil)
	ops.faultAfter = func(point string) error {
		if point == "install:B" {
			return errors.New("seed recovery")
		}
		return nil
	}
	if err := txCommit(dir, map[string][]byte{"A": []byte("second-a"), "B": []byte("second-b")}, ops); err == nil {
		t.Fatal("seeded Windows crash did not fire")
	}
	if err := Commit(dir, map[string][]byte{}); err != nil {
		t.Fatalf("native Windows recovery: %v", err)
	}
	assertFile(t, filepath.Join(dir, "A"), "first-a")
	assertFile(t, filepath.Join(dir, "B"), "first-b")
	assertNoTransactionFiles(t, dir)
}
