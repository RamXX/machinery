package designlock

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeDesignWorkspaceIgnoresIncidentalParentProse(t *testing.T) {
	project := t.TempDir()
	design := filepath.Join(project, "design")
	if err := os.Mkdir(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "README.md"), []byte("The shell example uses ../tmp, but this is not an embed.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "ambient.txt"), []byte("must not be retained\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	workspace, err := lock.MaterializeDesignWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = workspace.Close() }()
	if workspace.Path() != lock.SourceRoot() {
		t.Fatalf("incidental prose widened the retained workspace: %s != %s", workspace.Path(), lock.SourceRoot())
	}
	if _, err := os.Stat(filepath.Join(workspace.Path(), "..", "ambient.txt")); !os.IsNotExist(err) {
		t.Fatalf("ambient project file entered a design-only snapshot: %v", err)
	}
}

func TestMaterializeDesignWorkspaceRetainsDecomposedParentSiblingTopology(t *testing.T) {
	workspaceRoot := t.TempDir()
	design := filepath.Join(workspaceRoot, "parent", "design")
	child := filepath.Join(workspaceRoot, "orders", "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "decomposition.yaml"), []byte("revision: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	childFile := filepath.Join(child, "packmap.yaml")
	if err := os.WriteFile(childFile, []byte("child-v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Release() }()
	stable, err := lock.MaterializeDesignWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = stable.Close() }()
	stableChild := filepath.Clean(filepath.Join(stable.Path(), "..", "..", "orders", "design", "packmap.yaml"))
	if got, err := os.ReadFile(stableChild); err != nil || string(got) != "child-v1\n" {
		t.Fatalf("decomposed sibling topology missing: %q, %v", got, err)
	}
	if err := os.WriteFile(childFile, []byte("child-v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(stableChild); err != nil || string(got) != "child-v1\n" {
		t.Fatalf("stable sibling exposed live mutation: %q, %v", got, err)
	}
	if err := lock.CheckUnchanged(); err == nil {
		t.Fatal("external sibling mutation was not tracked")
	}
}

func TestRetainedWorkspaceScopeWidensOnlyForCommittedChildPack(t *testing.T) {
	root := t.TempDir()
	design := filepath.Join(root, "component", "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := RetainedWorkspaceScope(design); err != nil || got != filepath.Join(root, "component") {
		t.Fatalf("standalone scope = %s, %v", got, err)
	}
	copyPackCapabilityFixture(t, design)
	if got, err := RetainedWorkspaceScope(design); err != nil || got != root {
		t.Fatalf("child-pack scope = %s, %v, want %s", got, err, root)
	}
}

func copyPackCapabilityFixture(t *testing.T, design string) {
	t.Helper()
	source := filepath.Join("..", "..", "examples", "checkout-split", "orders", "design")
	if err := os.MkdirAll(filepath.Join(design, "pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(source, "pack"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(source, "pack", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(design, "pack", entry.Name()), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	body, err := os.ReadFile(filepath.Join(source, "packmap.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "packmap.yaml"), body, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRetainedWorkspaceScopeRejectsMalformedPackAuthority(t *testing.T) {
	root := t.TempDir()
	design := filepath.Join(root, "component", "design")
	if err := os.MkdirAll(filepath.Join(design, "pack"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "pack", "pack.yaml"), []byte("pack_version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RetainedWorkspaceScope(design); err == nil {
		t.Fatal("a handmade one-line pack widened workspace authority")
	}
}
