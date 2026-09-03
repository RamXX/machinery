package designlock

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func replacePrivateSnapshotDirectory(t *testing.T, path string) string {
	t.Helper()
	parked := path + ".parked"
	if err := os.Rename(path, parked); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "foreign-marker"), []byte("preserve\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return parked
}

func assertForeignPrivateSnapshotPreserved(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(path, "foreign-marker"))
	if err != nil || string(body) != "preserve\n" {
		t.Fatalf("foreign replacement = %q, %v", body, err)
	}
}

func restorePrivateSnapshotDirectory(t *testing.T, path, parked string) {
	t.Helper()
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(parked, path); err != nil {
		t.Fatal(err)
	}
}

func TestRegularFileSnapshotCleanupPreservesReplacementAndRetries(t *testing.T) {
	design := t.TempDir()
	inputDir := t.TempDir()
	input := filepath.Join(inputDir, "input.txt")
	if err := os.WriteFile(input, []byte("input\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	snapshot, err := lock.MaterializeRegularFile(input)
	if err != nil {
		t.Fatal(err)
	}
	cleanupPath := snapshot.cleanup.Path()
	parked := replacePrivateSnapshotDirectory(t, cleanupPath)
	if err := snapshot.Close(); err == nil || !strings.Contains(err.Error(), "was replaced") {
		t.Fatalf("replacement cleanup error = %v", err)
	}
	assertForeignPrivateSnapshotPreserved(t, cleanupPath)
	restorePrivateSnapshotDirectory(t, cleanupPath, parked)
	if err := snapshot.Close(); err != nil {
		t.Fatalf("retry cleanup: %v", err)
	}
	if _, err := os.Lstat(cleanupPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned snapshot remained after retry: %v", err)
	}
}

func TestExternalTreeSnapshotCleanupUsesPrivateAuthority(t *testing.T) {
	design := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "source.txt"), []byte("source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	snapshot, err := lock.MaterializeExternalTree(external)
	if err != nil {
		t.Fatal(err)
	}
	cleanupPath := snapshot.cleanup.Path()
	prior := privateSnapshotAfterQuarantine
	privateSnapshotAfterQuarantine = func(path string) {
		if path != cleanupPath {
			return
		}
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "foreign-marker"), []byte("preserve\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := snapshot.Close(); err != nil {
		t.Fatalf("authority-bound cleanup: %v", err)
	}
	privateSnapshotAfterQuarantine = prior
	t.Cleanup(func() { privateSnapshotAfterQuarantine = prior })
	assertForeignPrivateSnapshotPreserved(t, cleanupPath)
	if err := os.RemoveAll(cleanupPath); err != nil {
		t.Fatal(err)
	}
}

func TestDesignSourceRefreshPreservesReplacementAndReleaseRetries(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "DOMAIN.md"), []byte("domain\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	priorPath := lock.sourceRoot
	parked := replacePrivateSnapshotDirectory(t, priorPath)
	if err := lock.Refresh(); err == nil || !strings.Contains(err.Error(), "was replaced") {
		t.Fatalf("refresh cleanup error = %v", err)
	}
	assertForeignPrivateSnapshotPreserved(t, priorPath)
	restorePrivateSnapshotDirectory(t, priorPath, parked)
	if err := lock.Release(); err != nil {
		t.Fatalf("release did not retry retired source cleanup: %v", err)
	}
	if _, err := os.Lstat(priorPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired source remained after release retry: %v", err)
	}
}

func TestDesignSourceReleasePreservesReplacementAndCanRetryWithoutLock(t *testing.T) {
	design := t.TempDir()
	lock, err := Acquire(design)
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := lock.sourceRoot
	parked := replacePrivateSnapshotDirectory(t, sourcePath)
	if err := lock.Release(); err == nil || !strings.Contains(err.Error(), "was replaced") {
		t.Fatalf("release cleanup error = %v", err)
	}
	assertForeignPrivateSnapshotPreserved(t, sourcePath)
	restorePrivateSnapshotDirectory(t, sourcePath, parked)
	if err := lock.Release(); err != nil {
		t.Fatalf("cleanup retry after lock release: %v", err)
	}
	if _, err := os.Lstat(sourcePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("source remained after cleanup retry: %v", err)
	}
}
