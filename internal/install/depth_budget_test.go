package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeInstallDepthTree creates a directory root plus descendants. The returned
// leaf is exactly depth edges below root.
func makeInstallDepthTree(t *testing.T, parent, name string, depth int) (string, string) {
	t.Helper()
	root := filepath.Join(parent, name)
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	leaf := root
	for index := 0; index < depth; index++ {
		leaf = filepath.Join(leaf, "d")
		if err := os.Mkdir(leaf, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root, leaf
}

func requireInstallDepthError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "traversal depth limit") {
		t.Fatalf("expected traversal depth error, got %v", err)
	}
}

func requireInstallPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("over-depth traversal mutated %s: %v", path, err)
	}
}

func TestInstallArtifactDigestDepthBudget(t *testing.T) {
	parent := t.TempDir()
	exact, _ := makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
	if _, err := stableArtifactPostImageDigest(exact); err != nil {
		t.Fatalf("exact-limit path digest failed: %v", err)
	}
	over, _ := makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
	_, err := stableArtifactPostImageDigest(over)
	requireInstallDepthError(t, err)
}

func TestInstallRootedArtifactDigestDepthBudget(t *testing.T) {
	parent := t.TempDir()
	makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
	makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
	root, err := os.OpenRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup
	if _, err := stableArtifactDigestRoot(root, "exact"); err != nil {
		t.Fatalf("exact-limit rooted digest failed: %v", err)
	}
	_, err = stableArtifactDigestRoot(root, "over")
	requireInstallDepthError(t, err)
}

func TestInstallCopyDepthBudgetBeforeMutation(t *testing.T) {
	t.Run("path to path", func(t *testing.T) {
		t.Setenv("MACHINERY_CONFIG_DIR", privateConfigDir(t))
		parent := t.TempDir()
		exact, _ := makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
		exactDst := filepath.Join(parent, "exact-dst")
		if err := copyEntryNoFollow(exact, exactDst); err != nil {
			t.Fatalf("exact-limit copy failed: %v", err)
		}
		over, _ := makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
		overDst := filepath.Join(parent, "over-dst")
		requireInstallDepthError(t, copyEntryNoFollow(over, overDst))
		requireInstallPathAbsent(t, overDst)
	})

	t.Run("path to root", func(t *testing.T) {
		parent := t.TempDir()
		exact, _ := makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
		over, _ := makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
		destination := t.TempDir()
		root, err := os.OpenRoot(destination)
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close() //nolint:errcheck // test cleanup
		if err := copyEntryToRoot(exact, root, "exact"); err != nil {
			t.Fatalf("exact-limit rooted destination copy failed: %v", err)
		}
		requireInstallDepthError(t, copyEntryToRoot(over, root, "over"))
		requireInstallPathAbsent(t, filepath.Join(destination, "over"))
	})

	t.Run("root to path", func(t *testing.T) {
		source := t.TempDir()
		makeInstallDepthTree(t, source, "exact", installMaxTraversalDepth)
		makeInstallDepthTree(t, source, "over", installMaxTraversalDepth+1)
		root, err := os.OpenRoot(source)
		if err != nil {
			t.Fatal(err)
		}
		defer root.Close() //nolint:errcheck // test cleanup
		destination := t.TempDir()
		if err := copyEntryFromRoot(root, "exact", filepath.Join(destination, "exact")); err != nil {
			t.Fatalf("exact-limit rooted source copy failed: %v", err)
		}
		overDst := filepath.Join(destination, "over")
		requireInstallDepthError(t, copyEntryFromRoot(root, "over", overDst))
		requireInstallPathAbsent(t, overDst)
	})
}

func TestInstallWalkAndCopyTreeDepthBudgetBeforeMutation(t *testing.T) {
	parent := t.TempDir()
	exact, _ := makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
	visited := 0
	if err := walkInstallTreeBounded(exact, installArtifactMaxEntries, func(string, os.FileInfo) error {
		visited++
		return nil
	}); err != nil {
		t.Fatalf("exact-limit walk failed: %v", err)
	}
	if want := installMaxTraversalDepth + 1; visited != want {
		t.Fatalf("exact-limit walk visited %d entries, want %d", visited, want)
	}

	over, _ := makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
	visited = 0
	err := walkInstallTreeBounded(over, installArtifactMaxEntries, func(string, os.FileInfo) error {
		visited++
		return nil
	})
	requireInstallDepthError(t, err)
	if visited != 0 {
		t.Fatalf("over-depth walk invoked visitor %d times before rejecting inventory", visited)
	}

	exactDst := filepath.Join(parent, "exact-copy")
	if err := copyTree(exact, exactDst); err != nil {
		t.Fatalf("exact-limit copyTree failed: %v", err)
	}
	overDst := filepath.Join(parent, "over-copy")
	requireInstallDepthError(t, copyTree(over, overDst))
	requireInstallPathAbsent(t, overDst)
}

func TestInstallSourceSnapshotDepthBudget(t *testing.T) {
	exactSource := fakeSource(t)
	exactBase := filepath.Join(exactSource, skillRel)
	_, _ = makeInstallDepthTree(t, exactBase, "deep", installMaxTraversalDepth-installRelativeTraversalDepth(skillRel)-1)
	exact, err := acquireInstallSourceSnapshot(exactSource, nil)
	if err != nil {
		t.Fatalf("exact-limit source snapshot failed: %v", err)
	}
	if err := exact.verifyUnchanged(); err != nil {
		t.Fatalf("exact-limit source verification failed: %v", err)
	}
	if err := exact.cleanup(); err != nil {
		t.Fatal(err)
	}

	overSource := fakeSource(t)
	overBase := filepath.Join(overSource, skillRel)
	_, _ = makeInstallDepthTree(t, overBase, "deep", installMaxTraversalDepth-installRelativeTraversalDepth(skillRel))
	_, err = acquireInstallSourceSnapshot(overSource, nil)
	requireInstallDepthError(t, err)
}

func TestInstallPluginTraversalDepthBudget(t *testing.T) {
	parent := t.TempDir()
	makeInstallDepthTree(t, parent, "exact", installMaxTraversalDepth)
	makeInstallDepthTree(t, parent, "over", installMaxTraversalDepth+1)
	if err := os.Mkdir(filepath.Join(parent, "topology-exact"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parent, "topology-over"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close() //nolint:errcheck // test cleanup

	if err := walkPluginCacheTopology(root, "topology-exact", installMaxTraversalDepth, 1, map[string]pluginCacheTopologyEntry{}, nil); err != nil {
		t.Fatalf("exact-limit topology walk failed: %v", err)
	}
	requireInstallDepthError(t, walkPluginCacheTopology(root, "topology-over", installMaxTraversalDepth+1, 1, map[string]pluginCacheTopologyEntry{}, nil))

	expected := map[string]bool{}
	expectedDirectories := map[string]bool{}
	path := "exact"
	for depth := 0; depth <= installMaxTraversalDepth; depth++ {
		expectedDirectories[path] = true
		path = filepath.Join(path, "d")
	}
	if err := walkCachedPluginInventory(root, "exact", expected, expectedDirectories, map[string]bool{}, map[string]cachedPluginInventoryEntry{}, nil, 0); err != nil {
		t.Fatalf("exact-limit cached inventory failed: %v", err)
	}
	expectedDirectories = map[string]bool{}
	path = "over"
	for depth := 0; depth <= installMaxTraversalDepth+1; depth++ {
		expectedDirectories[path] = true
		path = filepath.Join(path, "d")
	}
	requireInstallDepthError(t, walkCachedPluginInventory(root, "over", expected, expectedDirectories, map[string]bool{}, map[string]cachedPluginInventoryEntry{}, nil, 0))
}
