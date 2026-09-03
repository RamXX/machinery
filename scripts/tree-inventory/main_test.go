package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingDirectoryReader struct {
	requests []int
}

func (reader *recordingDirectoryReader) ReadDir(count int) ([]fs.DirEntry, error) {
	reader.requests = append(reader.requests, count)
	return nil, io.EOF
}

func TestReadDirectoryMaxIntUsesPositiveBoundedPage(t *testing.T) {
	reader := &recordingDirectoryReader{}
	if _, err := readDirectory(reader, math.MaxInt, "test"); err != nil {
		t.Fatal(err)
	}
	if len(reader.requests) != 1 || reader.requests[0] != readDirPage {
		t.Fatalf("ReadDir requests = %v, want one positive page of %d", reader.requests, readDirPage)
	}
}

func testOptions() inventoryOptions {
	return inventoryOptions{maxEntries: 100, maxDepth: 10, maxBytes: 1 << 20, prune: map[string]bool{}}
}

func testSnapshotOptions() snapshotOptions {
	return snapshotOptions{
		inventory:     testOptions(),
		maxFileBytes:  2 << 20,
		maxTotalBytes: 4 << 20,
		excludes:      map[string]bool{},
	}
}

func TestInventoryIsExactAndByteSorted(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "m"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"z", "a", "m/b"} {
		if err := os.WriteFile(filepath.Join(root, rel), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := inventory(t.Context(), []string{root}, nil, testOptions())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{root, filepath.Join(root, "a"), filepath.Join(root, "m"), filepath.Join(root, "m", "b"), filepath.Join(root, "z")}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("inventory = %q, want %q", got, want)
	}
}

func TestInventoryPrunesExactSubtreeAndItsRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "private"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "public"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.prune[".git"] = true
	got, err := inventory(t.Context(), []string{root}, nil, options)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{root, filepath.Join(root, "public")}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("pruned inventory = %q, want %q", got, want)
	}
}

func TestInventoryMatchesExactRegularBaseName(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"workspace.dsl", "nested/workspace.dsl", "notworkspace.dsl"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "directory", "workspace.dsl"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "symlink"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "workspace.dsl"), filepath.Join(root, "symlink", "workspace.dsl")); err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.fileName = "workspace.dsl"
	options.regularFilesOnly = true
	got, err := inventory(t.Context(), []string{root}, nil, options)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(root, "nested", "workspace.dsl"), filepath.Join(root, "workspace.dsl")}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("exact regular-name inventory = %q, want %q", got, want)
	}
}

func TestC4CustomRootInventoryHonorsExpiredDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	options := testOptions()
	options.fileName = "workspace.dsl"
	options.regularFilesOnly = true
	if _, err := inventory(ctx, []string{t.TempDir()}, nil, options); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expired C4 custom-root inventory deadline was ignored: %v", err)
	}
}

func TestInventoryRejectsAggregateEntryOverflow(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	options := testOptions()
	options.maxEntries = 2
	if _, err := inventory(t.Context(), []string{root}, nil, options); err == nil || !strings.Contains(err.Error(), "entry limit") {
		t.Fatalf("high-entry tree was accepted: %v", err)
	}
}

func TestInventoryRejectsExcessiveDepth(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "one", "two"), 0o700); err != nil {
		t.Fatal(err)
	}
	options := testOptions()
	options.maxDepth = 1
	if _, err := inventory(t.Context(), []string{root}, nil, options); err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("deep tree was accepted: %v", err)
	}
}

func TestInventoryRejectsContinuousDirectoryGrowth(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "before"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	prior := inventoryAfterEnumeration
	t.Cleanup(func() { inventoryAfterEnumeration = prior })
	stop := make(chan struct{})
	stopped := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-stopped:
		default:
			close(stop)
			<-stopped
		}
	})
	inventoryAfterEnumeration = func(path string) {
		if path != root {
			return
		}
		inventoryAfterEnumeration = func(string) {}
		started := make(chan struct{})
		go func() {
			defer close(stopped)
			for i := 0; ; i++ {
				if err := os.WriteFile(filepath.Join(root, "growth-"+strings.Repeat("x", i%32)), nil, 0o600); err != nil {
					t.Error(err)
					return
				}
				if i == 0 {
					close(started)
				}
				select {
				case <-stop:
					return
				default:
				}
			}
		}()
		<-started
	}
	_, err := inventory(t.Context(), []string{root}, nil, testOptions())
	close(stop)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed while enumerating") {
		t.Fatalf("growing tree was accepted: %v", err)
	}
}

func TestInventoryRejectsSameDirectoryABADespiteRestoredMtime(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "entry")
	parked := filepath.Join(root, "parked")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	directoryInfo, err := os.Stat(root)
	if err != nil {
		t.Fatal(err)
	}
	prior := inventoryBetweenPasses
	t.Cleanup(func() { inventoryBetweenPasses = prior })
	inventoryBetweenPasses = func() {
		inventoryBetweenPasses = func() {}
		// Restore the exact same child inode, size, mode, and mtime after a
		// directory-entry ABA. Only the held directory's native change witness
		// can distinguish the two otherwise-identical ordered inventories.
		if err := os.Rename(path, parked); err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(path, []byte("transient"), 0o600); err != nil {
			t.Error(err)
			return
		}
		if err := os.Remove(path); err != nil {
			t.Error(err)
			return
		}
		if err := os.Rename(parked, path); err != nil {
			t.Error(err)
			return
		}
		if err := os.Chtimes(root, directoryInfo.ModTime(), directoryInfo.ModTime()); err != nil {
			t.Error(err)
		}
	}
	if _, err := inventory(t.Context(), []string{root}, nil, testOptions()); err == nil || !strings.Contains(err.Error(), "between bounded passes") {
		t.Fatalf("same-directory ABA with restored mtime was accepted: %v", err)
	}
}

func TestInventoryHonorsExpiredDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inventory(ctx, []string{t.TempDir()}, nil, testOptions()); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expired traversal deadline was ignored: %v", err)
	}
}

func TestInventoryRejectsAggregatePathByteOverflow(t *testing.T) {
	root := t.TempDir()
	options := testOptions()
	options.maxBytes = int64(len(root))
	if _, err := inventory(t.Context(), []string{root}, nil, options); err == nil || !strings.Contains(err.Error(), "byte path limit") {
		t.Fatalf("path-byte overflow was accepted: %v", err)
	}
}

func TestSnapshotIsExactAndExcludesDeclaredFile(t *testing.T) {
	root := t.TempDir()
	body := []byte("content")
	if err := os.WriteFile(filepath.Join(root, "included"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "excluded"), []byte("render"), 0o600); err != nil {
		t.Fatal(err)
	}
	options := testSnapshotOptions()
	options.excludes["excluded"] = true
	got, err := snapshotTree(t.Context(), root, options)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	want := []string{fmt.Sprintf("F\tincluded\t%d\tsha256:%x", len(body), digest)}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("snapshot = %q, want %q", got, want)
	}
}

func TestSnapshotRejectsSparseFileBeforeHashing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sparse")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(1025); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	options := testSnapshotOptions()
	options.maxFileBytes = 1024
	if _, err := snapshotTree(t.Context(), root, options); err == nil || !strings.Contains(err.Error(), "per-file limit") {
		t.Fatalf("oversized sparse file was accepted: %v", err)
	}
}

func TestSnapshotRejectsAggregateFileBytesBeforeHashing(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("123456"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	options := testSnapshotOptions()
	options.maxTotalBytes = 10
	if _, err := snapshotTree(t.Context(), root, options); err == nil || !strings.Contains(err.Error(), "aggregate limit") {
		t.Fatalf("aggregate file overflow was accepted: %v", err)
	}
}

func TestSnapshotRejectsAmbiguousPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "ambiguous\tpath"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotTree(t.Context(), root, testSnapshotOptions()); err == nil || !strings.Contains(err.Error(), "cannot be represented safely") {
		t.Fatalf("ambiguous snapshot path was accepted: %v", err)
	}
}

func TestSnapshotRejectsContinuousAppender(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 1<<20)), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	stop := make(chan struct{})
	stopped := make(chan struct{})
	snapshotFilePoint = func(rel, phase string) error {
		if rel != "file" || phase != "after-open" {
			return nil
		}
		snapshotFilePoint = func(string, string) error { return nil }
		started := make(chan struct{})
		go func() {
			defer close(stopped)
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				close(started)
				return
			}
			defer file.Close() //nolint:errcheck // adversarial writer
			for index := 0; ; index++ {
				_, _ = file.Write([]byte("growth"))
				if index == 0 {
					close(started)
				}
				select {
				case <-stop:
					return
				default:
				}
			}
		}()
		<-started
		return nil
	}
	_, err := snapshotTree(t.Context(), root, testSnapshotOptions())
	close(stop)
	<-stopped
	if err == nil || (!strings.Contains(err.Error(), "grew") && !strings.Contains(err.Error(), "changed")) {
		t.Fatalf("continuous appender was accepted: %v", err)
	}
}

func TestSnapshotRejectsSameIdentityContentABA(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	snapshotFilePoint = func(rel, phase string) error {
		if rel != "file" || phase != "after-first-hash" {
			return nil
		}
		snapshotFilePoint = func(string, string) error { return nil }
		if err := os.WriteFile(path, []byte("after!"), 0o600); err != nil {
			return err
		}
		return os.Chtimes(path, info.ModTime(), info.ModTime())
	}
	if _, err := snapshotTree(t.Context(), root, testSnapshotOptions()); err == nil || !strings.Contains(err.Error(), "changed while hashing") {
		t.Fatalf("same-identity content ABA was accepted: %v", err)
	}
}

func TestSnapshotRejectsLateRewriteOfPreviouslyHashedFile(t *testing.T) {
	root := t.TempDir()
	earlyPath := filepath.Join(root, "a-early")
	latePath := filepath.Join(root, "z-late")
	if err := os.WriteFile(earlyPath, []byte("AAAA"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latePath, []byte("later"), 0o600); err != nil {
		t.Fatal(err)
	}
	earlyInfo, err := os.Stat(earlyPath)
	if err != nil {
		t.Fatal(err)
	}
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	snapshotFilePoint = func(rel, phase string) error {
		if rel != "z-late" || phase != "after-first-hash" {
			return nil
		}
		snapshotFilePoint = func(string, string) error { return nil }
		if err := os.WriteFile(earlyPath, []byte("BBBB"), 0o600); err != nil {
			return err
		}
		return os.Chtimes(earlyPath, earlyInfo.ModTime(), earlyInfo.ModTime())
	}
	if _, err := snapshotTree(t.Context(), root, testSnapshotOptions()); err == nil || !strings.Contains(err.Error(), "between bounded content passes") {
		t.Fatalf("late same-size rewrite of previously hashed file was accepted: %v", err)
	}
}

func TestSnapshotRejectsPathReplacementDuringHash(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := snapshotFilePoint
	t.Cleanup(func() { snapshotFilePoint = prior })
	snapshotFilePoint = func(rel, phase string) error {
		if rel != "file" || phase != "after-open" {
			return nil
		}
		snapshotFilePoint = func(string, string) error { return nil }
		if err := os.Rename(path, path+".old"); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("before"), 0o600)
	}
	if _, err := snapshotTree(t.Context(), root, testSnapshotOptions()); err == nil || !strings.Contains(err.Error(), "changed while hashing") {
		t.Fatalf("path replacement during hash was accepted: %v", err)
	}
}

func TestSnapshotHonorsExpiredDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := snapshotTree(ctx, t.TempDir(), testSnapshotOptions()); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expired snapshot deadline was ignored: %v", err)
	}
}
