package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func writeFile(t *testing.T, path, body string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
}

func TestCopyTreeReproducesTheManifest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks and permission bits are POSIX here")
	}
	src := filepath.Join(t.TempDir(), "src")
	writeFile(t, filepath.Join(src, "a.md"), "alpha\n", 0o644)
	writeFile(t, filepath.Join(src, "bin", "run"), "#!/bin/sh\n", 0o755)
	writeFile(t, filepath.Join(src, "ro", "frozen.txt"), "frozen\n", 0o444)
	if err := os.Symlink("../a.md", filepath.Join(src, "bin", "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(src, "ro"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeTree(src) })

	before, err := hashTree(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { removeTree(dst) })
	after, err := hashTree(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("copy differs: %v", diffManifests(before, after))
	}
	if before["bin/link"] != "link ../a.md" {
		t.Fatalf("symlink recorded as %q, want its target, never followed", before["bin/link"])
	}
}

func TestDiffManifestsSeesEveryKindOfChange(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep.md"), "same\n", 0o644)
	writeFile(t, filepath.Join(root, "edit.md"), "before\n", 0o644)
	writeFile(t, filepath.Join(root, "gone.md"), "x\n", 0o644)
	writeFile(t, filepath.Join(root, "mode.sh"), "x\n", 0o644)
	before, err := hashTree(root)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "edit.md"), "after\n", 0o644)
	if err := os.Remove(filepath.Join(root, "gone.md")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "new", "added.md"), "x\n", 0o644)
	want := []string{"edit.md", "gone.md", "new", "new/added.md"}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(root, "mode.sh"), 0o755); err != nil {
			t.Fatal(err)
		}
		want = []string{"edit.md", "gone.md", "mode.sh", "new", "new/added.md"}
	}
	after, err := hashTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := diffManifests(before, after); !reflect.DeepEqual(got, want) {
		t.Fatalf("changed = %v, want %v", got, want)
	}
}

func TestFindCopyRootPrefersTheGitWorkTree(t *testing.T) {
	repo := t.TempDir()
	design := filepath.Join(repo, "docs", "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := findCopyRoot(design); got != design {
		t.Fatalf("outside git: root = %s, want the design itself", got)
	}
	// a worktree or submodule carries .git as a file; either form marks the root
	writeFile(t, filepath.Join(repo, ".git"), "gitdir: /elsewhere\n", 0o644)
	if got := findCopyRoot(design); got != repo {
		t.Fatalf("inside git: root = %s, want %s", got, repo)
	}
}
