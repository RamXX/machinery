package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/testgit"
)

func gitSnapshotFixture(t *testing.T) (string, snapshotOptions) {
	t.Helper()
	root := t.TempDir()
	for path, body := range map[string]string{
		".gitignore":      "cache/\ntracked\n",
		"tracked":         "tracked despite ignore rule",
		"included":        "untracked content",
		"cache/oversized": strings.Repeat("x", 1024),
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-f", "tracked", ".gitignore"}} {
		if output, err := testgit.Run(t.Context(), root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	options := testSnapshotOptions()
	options.inventory.gitTree = true
	options.maxFileBytes = 512
	return root, options
}

func TestGitSnapshotIncludesTrackedAndNonIgnoredFiles(t *testing.T) {
	root, options := gitSnapshotFixture(t)
	got, err := snapshotTree(t.Context(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("snapshot = %q, want three selected files", got)
	}
	for _, path := range []string{".gitignore", "tracked", "included"} {
		if !strings.Contains(strings.Join(got, "\n"), "F\t"+path+"\t") {
			t.Fatalf("missing %s: %q", path, got)
		}
	}
}

func TestGitSnapshotIgnoresCacheEntryChurn(t *testing.T) {
	root, options := gitSnapshotFixture(t)
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	mutations := 0
	snapshotFilePoint = func(_, phase string) error {
		if phase != "after-first-hash" {
			return nil
		}
		mutations++
		return os.WriteFile(filepath.Join(root, "cache", strings.Repeat("x", mutations)), nil, 0o600)
	}
	if _, err := snapshotTree(t.Context(), root, options); err != nil {
		t.Fatalf("ignored cache churn: %v", err)
	}
	if mutations == 0 {
		t.Fatal("cache mutation hook was not exercised")
	}
}

func TestGitSnapshotRejectsNewNonIgnoredFileDuringHashing(t *testing.T) {
	root, options := gitSnapshotFixture(t)
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	snapshotFilePoint = func(_, phase string) error {
		if phase != "after-first-hash" {
			return nil
		}
		return os.WriteFile(filepath.Join(root, "new-file"), nil, 0o600)
	}
	if _, err := snapshotTree(t.Context(), root, options); err == nil || !strings.Contains(err.Error(), "inventory changed while hashing") {
		t.Fatalf("new non-ignored file accepted: %v", err)
	}
}

func TestGitSnapshotRejectsSelectedOversizedFile(t *testing.T) {
	root, options := gitSnapshotFixture(t)
	if err := os.WriteFile(filepath.Join(root, "included"), make([]byte, 1024), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotTree(t.Context(), root, options); err == nil || !strings.Contains(err.Error(), "per-file limit") {
		t.Fatalf("selected oversized file accepted: %v", err)
	}
}

func TestGitSnapshotRejectsSymlinkParent(t *testing.T) {
	root, options := gitSnapshotFixture(t)
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "file"), []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "parent")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := testgit.Run(t.Context(), root, "add", "parent/file"); err != nil {
		t.Fatalf("add: %v\n%s", err, output)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotTree(t.Context(), root, options); err == nil || !strings.Contains(err.Error(), "must be a real directory") {
		t.Fatalf("symlink parent accepted: %v", err)
	}
}

func TestGitSnapshotEnforcesInventoryLimits(t *testing.T) {
	for _, limit := range []string{"entries", "bytes", "depth"} {
		t.Run(limit, func(t *testing.T) {
			root, options := gitSnapshotFixture(t)
			switch limit {
			case "entries":
				options.inventory.maxEntries = 2
			case "bytes":
				options.inventory.maxBytes = 1
			case "depth":
				options.inventory.maxDepth = 0
			}
			if _, err := snapshotTree(t.Context(), root, options); err == nil {
				t.Fatalf("Git snapshot accepted %s overflow", limit)
			}
		})
	}
}
