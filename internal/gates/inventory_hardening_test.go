package gates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSortedGlobExtPropagatesDirectoryReadFailure(t *testing.T) {
	want := errors.New("injected inventory failure")
	prior := readGateDirectory
	t.Cleanup(func() { readGateDirectory = prior })
	readGateDirectory = func(string, int) ([]os.DirEntry, error) { return nil, want }

	if _, err := sortedGlobExt(t.TempDir(), ".md"); !errors.Is(err, want) {
		t.Fatalf("sortedGlobExt error = %v, want injected failure", err)
	}
}

func TestFinalHandoffReportsDirectoryReadFailure(t *testing.T) {
	design := t.TempDir()
	want := errors.New("injected inventory failure")
	prior := readGateDirectory
	t.Cleanup(func() { readGateDirectory = prior })
	readGateDirectory = func(string, int) ([]os.DirEntry, error) { return nil, want }

	g := CheckFinalHandoff(design)
	if !containsInventoryFinding(g.Errs, "injected inventory failure") {
		t.Fatalf("CheckFinalHandoff errors = %v, want inventory failure", g.Errs)
	}
}

func TestReadRootDirectoryRejectsCreateDeleteABA(t *testing.T) {
	testReadRootDirectoryABA(t, func(t *testing.T, dir string) {
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(transient); err != nil {
			t.Fatal(err)
		}
	})
}

func TestReadRootDirectoryRejectsRenameABA(t *testing.T) {
	testReadRootDirectoryABA(t, func(t *testing.T, dir string) {
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

func testReadRootDirectoryABA(t *testing.T, mutate func(*testing.T, string)) {
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
	prior := rootInventoryAfterFirst
	t.Cleanup(func() { rootInventoryAfterFirst = prior })
	rootInventoryAfterFirst = func(rel string) {
		if rel != "." {
			return
		}
		rootInventoryAfterFirst = func(string) {}
		mutate(t, dir)
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Fatal(err)
		}
	}

	if _, _, err := readRootDirectory(root, ".", 10); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("readRootDirectory accepted same-directory ABA: %v", err)
	}
}

func containsInventoryFinding(findings []string, needle string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, needle) {
			return true
		}
	}
	return false
}
