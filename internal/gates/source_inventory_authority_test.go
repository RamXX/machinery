package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceInventoryRejectsRootRenameABA(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "impl")
	mustWrite(t, filepath.Join(root, "safe.go"), "package safe\n")
	initial, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	inventory, _, warns, err := walkSourceFilesBounded(root, nil, 10, 4)
	if err != nil || len(warns) != 0 {
		t.Fatalf("inventory failed before mutation: warns=%v err=%v", warns, err)
	}
	aside := filepath.Join(parent, "aside")
	if err := os.Rename(root, aside); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(aside, root); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(root, initial.ModTime(), initial.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Close(); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("root rename ABA was accepted: %v", err)
	}
}

func TestSourceInventoryRejectsNestedAncestorRenameABA(t *testing.T) {
	root := t.TempDir()
	ancestor := filepath.Join(root, "a")
	mustWrite(t, filepath.Join(ancestor, "b", "safe.go"), "package safe\n")
	initial, err := os.Stat(ancestor)
	if err != nil {
		t.Fatal(err)
	}
	inventory, _, warns, err := walkSourceFilesBounded(root, nil, 10, 4)
	if err != nil || len(warns) != 0 {
		t.Fatalf("inventory failed before mutation: warns=%v err=%v", warns, err)
	}
	aside := filepath.Join(root, "aside")
	if err := os.Rename(ancestor, aside); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(aside, ancestor); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ancestor, initial.ModTime(), initial.ModTime()); err != nil {
		t.Fatal(err)
	}
	if err := inventory.Close(); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("ancestor rename ABA was accepted: %v", err)
	}
}

func TestSourceInventoryReadsThroughRetainedRootAfterPublicReplacement(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "impl")
	mustWrite(t, filepath.Join(root, "safe.go"), "package original\n")
	inventory, _, warns, err := walkSourceFilesBounded(root, nil, 10, 4)
	if err != nil || len(warns) != 0 {
		t.Fatalf("inventory failed before replacement: warns=%v err=%v", warns, err)
	}
	aside := filepath.Join(parent, "original")
	if err := os.Rename(root, aside); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "safe.go"), "package foreign\n")
	body, err := inventory.ReadFile("safe.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "package original\n" {
		t.Fatalf("retained authority read %q, want original source bytes", body)
	}
	if err := inventory.Close(); err == nil || !strings.Contains(err.Error(), "changed identity") {
		t.Fatalf("public root replacement was accepted: %v", err)
	}
}

func TestSourceInventoryRejectsSymlinkRoot(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "impl")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, _, _, err := walkSourceFilesBounded(link, nil, 10, 4); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("symlink root was accepted: %v", err)
	}
}
