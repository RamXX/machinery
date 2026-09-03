package designlock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeDesignlockDeepTree(t *testing.T, root string, levels int) string {
	t.Helper()
	current := root
	for level := 0; level < levels; level++ {
		current = filepath.Join(current, fmt.Sprintf("d%02d", level))
		if err := os.Mkdir(current, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(current, "leaf.txt"), []byte("leaf\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return current
}

func requireDesignlockDepthError(t *testing.T, err error) {
	t.Helper()
	want := fmt.Sprintf("%d-level portable snapshot depth limit", snapshotInventoryMaxDepth)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("over-depth tree error = %v; want %q", err, want)
	}
}

func TestFingerprintRootPortableDepthBoundary(t *testing.T) {
	t.Run("at-limit", func(t *testing.T) {
		root := t.TempDir()
		makeDesignlockDeepTree(t, root, snapshotInventoryMaxDepth)
		values, err := fingerprintRoot(root, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(values) != snapshotInventoryMaxDepth+1 {
			t.Fatalf("fingerprint entries = %d, want %d", len(values), snapshotInventoryMaxDepth+1)
		}
	})
	t.Run("over-limit", func(t *testing.T) {
		root := t.TempDir()
		makeDesignlockDeepTree(t, root, snapshotInventoryMaxDepth+1)
		_, err := fingerprintRoot(root, false)
		requireDesignlockDepthError(t, err)
	})
}

func TestCopyExternalTreePortableDepthBoundaryBeforeDestinationMutation(t *testing.T) {
	design := t.TempDir()
	lock, err := AcquireReader(design)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := lock.Release(); err != nil {
			t.Error(err)
		}
	}()

	t.Run("at-limit", func(t *testing.T) {
		source, destination := t.TempDir(), t.TempDir()
		makeDesignlockDeepTree(t, source, snapshotInventoryMaxDepth)
		if _, err := lock.copyExternalTree(source, destination, nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("over-limit-before-mutation", func(t *testing.T) {
		source, destination := t.TempDir(), t.TempDir()
		makeDesignlockDeepTree(t, source, snapshotInventoryMaxDepth+1)
		keep := filepath.Join(destination, "keep.txt")
		if err := os.WriteFile(keep, []byte("keep\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := lock.copyExternalTree(source, destination, nil, nil)
		requireDesignlockDepthError(t, err)
		entries, readErr := os.ReadDir(destination)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if len(entries) != 1 || entries[0].Name() != "keep.txt" {
			t.Fatalf("over-depth preflight mutated destination inventory: %v", entries)
		}
		if body, readErr := os.ReadFile(keep); readErr != nil || string(body) != "keep\n" {
			t.Fatalf("over-depth preflight changed destination sentinel: %q, %v", body, readErr)
		}
	})
}

func TestInterruptedPublicationRecoveryPortableDepthBoundary(t *testing.T) {
	t.Run("at-limit", func(t *testing.T) {
		root := t.TempDir()
		makeDesignlockDeepTree(t, root, snapshotInventoryMaxDepth)
		if found, err := findInterruptedJournal(root); err != nil || found != "" {
			t.Fatalf("at-limit recovery inventory = %q, %v", found, err)
		}
	})
	t.Run("over-limit", func(t *testing.T) {
		root := t.TempDir()
		makeDesignlockDeepTree(t, root, snapshotInventoryMaxDepth+1)
		_, err := findInterruptedJournal(root)
		requireDesignlockDepthError(t, err)
	})
}

func TestPrivateSnapshotCleanupPortableDepthBoundary(t *testing.T) {
	t.Run("at-limit", func(t *testing.T) {
		cleanup, err := newPrivateSnapshot("machinery-depth-at-limit-")
		if err != nil {
			t.Fatal(err)
		}
		makeDesignlockDeepTree(t, cleanup.Path(), snapshotInventoryMaxDepth)
		if err := cleanup.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(cleanup.Path()); !os.IsNotExist(err) {
			t.Fatalf("at-limit private snapshot remained after cleanup: %v", err)
		}
	})
	t.Run("over-limit-preserves-authority", func(t *testing.T) {
		cleanup, err := newPrivateSnapshot("machinery-depth-over-limit-")
		if err != nil {
			t.Fatal(err)
		}
		makeDesignlockDeepTree(t, cleanup.Path(), snapshotInventoryMaxDepth+1)
		err = cleanup.Close()
		requireDesignlockDepthError(t, err)
		if cleanup.quarantine == nil {
			t.Fatal("over-depth private cleanup did not preserve isolated authority")
		}
		if _, err := cleanup.quarantine.Root().Lstat(cleanup.quarantine.Name()); err != nil {
			t.Fatalf("over-depth private cleanup lost isolated tree: %v", err)
		}
		// The production path deliberately preserves over-limit evidence. This
		// test owns the fresh isolated tree, so remove it after proving that fact.
		if err := cleanup.quarantine.RemoveAll(); err != nil {
			t.Fatal(err)
		}
		cleanup.quarantine = nil
		if cleanup.parent != nil {
			if err := cleanup.parent.Close(); err != nil {
				t.Fatal(err)
			}
			cleanup.parent = nil
		}
	})
}
