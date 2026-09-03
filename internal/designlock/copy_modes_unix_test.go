//go:build !windows

package designlock

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestCopyExternalTreePreservesModesDespiteUmaskAndReadonlyDirectories(t *testing.T) {
	design := t.TempDir()
	source := t.TempDir()
	readonly := filepath.Join(source, "readonly")
	if err := os.Mkdir(readonly, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(readonly, "source.yaml")
	if err := os.WriteFile(file, []byte("stable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(readonly, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(readonly, 0o755)

	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	dest := t.TempDir()
	prior := syscall.Umask(0o077)
	defer syscall.Umask(prior)
	values, err := lock.copyExternalTree(source, dest, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Join(dest, "readonly"), 0o755)
	if info, err := os.Stat(filepath.Join(dest, "readonly")); err != nil || info.Mode().Perm() != 0o555 {
		t.Fatalf("destination directory mode = %v, %v; want 0555", info, err)
	}
	destFile := filepath.Join(dest, "readonly", "source.yaml")
	if info, err := os.Stat(destFile); err != nil || info.Mode().Perm() != 0o444 {
		t.Fatalf("destination file mode = %v, %v; want 0444", info, err)
	}
	if err := os.Chmod(destFile, 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := fingerprint(dest)
	if err != nil {
		t.Fatal(err)
	}
	if fingerprintDigest(changed) == fingerprintDigest(values) {
		t.Fatal("destination mode change did not alter the materialized fingerprint")
	}
}
