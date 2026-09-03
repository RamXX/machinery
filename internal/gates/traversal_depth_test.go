package gates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func linearTraversalFixture(t *testing.T, fileName string) (string, string) {
	t.Helper()
	root := t.TempDir()
	leaf := filepath.Join(root, "a", "b", fileName)
	if err := os.MkdirAll(filepath.Dir(leaf), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(leaf, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, leaf
}

func broadTraversalFixture(t *testing.T, fileName string) string {
	t.Helper()
	root := t.TempDir()
	nested := filepath.Join(root, "a", "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"one-", "two-", "three-"} {
		if err := os.WriteFile(filepath.Join(nested, prefix+fileName), []byte(prefix), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "root-"+fileName), []byte("root"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestValidateDesignInventoryEnforcesDepthAndAggregateLimits(t *testing.T) {
	root, _ := linearTraversalFixture(t, "leaf.md")
	if err := validateDesignInventoryBounded(root, 3, 3); err != nil {
		t.Fatalf("inventory exactly at entry/depth limits failed: %v", err)
	}
	if err := validateDesignInventoryBounded(root, 3, 2); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("inventory beyond depth limit was accepted: %v", err)
	}

	broad := broadTraversalFixture(t, "leaf.md")
	if err := validateDesignInventoryBounded(broad, 6, 3); err != nil {
		t.Fatalf("broad inventory exactly at aggregate limit failed: %v", err)
	}
	if err := validateDesignInventoryBounded(broad, 5, 3); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("broad multi-directory aggregate overflow was accepted: %v", err)
	}
}

func TestWalkTreeDirBoundedEnforcesDepthBeforeCallbackAndAggregateLimit(t *testing.T) {
	root, leaf := linearTraversalFixture(t, "leaf.md")
	visited := map[string]bool{}
	visit := func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		visited[path] = true
		return nil
	}
	if err := walkTreeDirBounded(root, 3, 3, visit); err != nil {
		t.Fatalf("walk exactly at entry/depth limits failed: %v", err)
	}
	visited = map[string]bool{}
	if err := walkTreeDirBounded(root, 3, 2, visit); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("walk beyond depth limit was accepted: %v", err)
	}
	if visited[leaf] {
		t.Fatal("over-depth callback ran before the traversal failed closed")
	}

	broad := broadTraversalFixture(t, "leaf.md")
	if err := walkTreeDirBounded(broad, 6, 3, visit); err != nil {
		t.Fatalf("broad walk exactly at aggregate limit failed: %v", err)
	}
	if err := walkTreeDirBounded(broad, 5, 3, visit); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("broad multi-directory aggregate overflow was accepted: %v", err)
	}
}

func TestWalkSourceFilesEnforcesDepthAndOneAggregateLimit(t *testing.T) {
	walkPaths := func(root string, maxEntries, maxDepth int) (files, warns []string, err error) {
		inventory, _, warns, err := walkSourceFilesBounded(root, nil, maxEntries, maxDepth)
		if inventory == nil {
			return nil, warns, err
		}
		return inventory.Paths(), warns, errors.Join(err, inventory.Close())
	}
	root, leaf := linearTraversalFixture(t, "leaf.go")
	files, warns, err := walkPaths(root, 3, 3)
	if err != nil || len(warns) != 0 || len(files) != 1 || files[0] != leaf {
		t.Fatalf("source walk exactly at entry/depth limits failed: files=%v warns=%v err=%v", files, warns, err)
	}
	files, warns, err = walkPaths(root, 3, 2)
	if err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("source walk beyond depth limit was accepted: files=%v warns=%v err=%v", files, warns, err)
	}
	if len(files) != 0 {
		t.Fatalf("over-depth source was returned before failure: %v", files)
	}

	broad := broadTraversalFixture(t, "leaf.go")
	files, warns, err = walkPaths(broad, 6, 3)
	if err != nil || len(warns) != 0 || len(files) != 4 {
		t.Fatalf("broad source walk exactly at aggregate limit failed: files=%v warns=%v err=%v", files, warns, err)
	}
	files, warns, err = walkPaths(broad, 5, 3)
	if err != nil || len(warns) == 0 || !strings.Contains(strings.Join(warns, "\n"), "entry limit") {
		t.Fatalf("broad source aggregate overflow was not reported deterministically: files=%v warns=%v err=%v", files, warns, err)
	}
	if len(files) >= 4 {
		t.Fatalf("source overflow returned a falsely complete inventory: %v", files)
	}
}

func TestEmbedResidueInventoryEnforcesDepthAndAggregateLimits(t *testing.T) {
	check := func(t *testing.T, root string, maxEntries, maxDepth int) error {
		t.Helper()
		tx, err := openEmbedRootTransaction(root)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := tx.Close(); err != nil {
				t.Errorf("close transaction root: %v", err)
			}
		}()
		_, err = tx.embedResiduePathsBounded(maxEntries, maxDepth)
		return err
	}

	root, _ := linearTraversalFixture(t, "leaf.md")
	if err := check(t, root, 3, 3); err != nil {
		t.Fatalf("residue inventory exactly at entry/depth limits failed: %v", err)
	}
	if err := check(t, root, 3, 2); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("residue inventory beyond depth limit was accepted: %v", err)
	}

	broad := broadTraversalFixture(t, "leaf.md")
	if err := check(t, broad, 6, 3); err != nil {
		t.Fatalf("broad residue inventory exactly at aggregate limit failed: %v", err)
	}
	if err := check(t, broad, 5, 3); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("broad residue aggregate overflow was accepted: %v", err)
	}
}
