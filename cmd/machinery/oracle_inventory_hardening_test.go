package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadOracleRootDirRejectsCreateDeleteABA(t *testing.T) {
	testReadOracleRootDirABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadOracleRootDirRejectsRenameABA(t *testing.T) {
	testReadOracleRootDirABA(t, func(t *testing.T, dir string) {
		stable := filepath.Join(dir, "stable")
		transient := filepath.Join(dir, "transient")
		if err := os.Rename(stable, transient); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(transient, stable); err != nil {
			t.Fatal(err)
		}
	})
}

func testReadOracleRootDirABA(t *testing.T, mutate func(*testing.T, string)) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stable"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	prior := oracleInventoryAfterFirst
	t.Cleanup(func() { oracleInventoryAfterFirst = prior })
	oracleInventoryAfterFirst = func(rel string) {
		if rel != "." {
			return
		}
		oracleInventoryAfterFirst = func(string) {}
		mutate(t, dir)
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := readOracleRootDir(root, ".", 10); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("readOracleRootDir accepted same-directory ABA: %v", err)
	}
}
