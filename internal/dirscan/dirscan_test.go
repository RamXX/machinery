package dirscan

import (
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingDirectoryReader struct {
	requests []int
}

type failingDirectoryReader struct {
	err error
}

func (r failingDirectoryReader) ReadDir(int) ([]os.DirEntry, error) {
	return nil, r.err
}

func (r *recordingDirectoryReader) ReadDir(count int) ([]os.DirEntry, error) {
	r.requests = append(r.requests, count)
	return nil, io.EOF
}

func TestReadEntriesMaxIntNeverIssuesUnboundedRequest(t *testing.T) {
	reader := &recordingDirectoryReader{}
	if _, err := readEntries(reader, "test", math.MaxInt); err != nil {
		t.Fatal(err)
	}
	if len(reader.requests) != 1 || reader.requests[0] != batchSize {
		t.Fatalf("ReadDir requests = %v, want one positive bounded page of %d", reader.requests, batchSize)
	}
}

func TestReadEntriesPropagatesEnumerationError(t *testing.T) {
	want := errors.New("injected directory read failure")
	if _, err := readEntries(failingDirectoryReader{err: want}, "test", 10); !errors.Is(err, want) {
		t.Fatalf("readEntries error = %v, want injected error", err)
	}
}

func TestReadIsSortedAndEnforcesCeilingDuringPagination(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"z", "a", "m", "b", "x"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Read(dir, 4); err == nil || !strings.Contains(err.Error(), "4-entry limit") {
		t.Fatalf("Read accepted over-limit inventory: %v", err)
	}
	entries, err := Read(dir, 5)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"a", "b", "m", "x", "z"} {
		if entries[i].Name() != want {
			t.Fatalf("entry %d = %q, want %q", i, entries[i].Name(), want)
		}
	}
}

func TestReadRejectsSymlinkDirectory(t *testing.T) {
	target := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Read(link, 1); err == nil || !strings.Contains(err.Error(), "real directory") {
		t.Fatalf("Read accepted symlink directory: %v", err)
	}
}

func TestReadRejectsConcurrentInventoryMutation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "before"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	prior := afterEnumeration
	t.Cleanup(func() { afterEnumeration = prior })
	afterEnumeration = func(path string) {
		if path != dir {
			return
		}
		afterEnumeration = func(string) {}
		if err := os.WriteFile(filepath.Join(dir, "after"), nil, 0o600); err != nil {
			t.Error(err)
		}
	}
	if _, err := Read(dir, 10); err == nil || !strings.Contains(err.Error(), "changed while enumerating") {
		t.Fatalf("Read accepted concurrent inventory mutation: %v", err)
	}
}

func TestReadRejectsSameDirectoryCreateDeleteABA(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stable"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	prior := afterEnumeration
	t.Cleanup(func() { afterEnumeration = prior })
	afterEnumeration = func(path string) {
		if path != dir {
			return
		}
		afterEnumeration = func(string) {}
		transient := filepath.Join(dir, "transient")
		if err := os.WriteFile(transient, nil, 0o600); err != nil {
			t.Error(err)
			return
		}
		if err := os.Remove(transient); err != nil {
			t.Error(err)
		}
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Error(err)
		}
	}
	if _, err := Read(dir, 10); err == nil || !strings.Contains(err.Error(), "changed while enumerating") {
		t.Fatalf("Read accepted create/delete ABA: %v", err)
	}
}

func TestReadRejectsSameDirectoryRenameABA(t *testing.T) {
	dir := t.TempDir()
	stable := filepath.Join(dir, "stable")
	transient := filepath.Join(dir, "transient")
	if err := os.WriteFile(stable, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	initial, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	prior := afterEnumeration
	t.Cleanup(func() { afterEnumeration = prior })
	afterEnumeration = func(path string) {
		if path != dir {
			return
		}
		afterEnumeration = func(string) {}
		if err := os.Rename(stable, transient); err != nil {
			t.Error(err)
			return
		}
		if err := os.Rename(transient, stable); err != nil {
			t.Error(err)
		}
		if err := os.Chtimes(dir, initial.ModTime(), initial.ModTime()); err != nil {
			t.Error(err)
		}
	}
	if _, err := Read(dir, 10); err == nil || !strings.Contains(err.Error(), "changed while enumerating") {
		t.Fatalf("Read accepted rename ABA: %v", err)
	}
}

func TestWalkEnforcesAggregateCeilingAcrossDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, rel := range []string{"a/one", "b/two"} {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	err := Walk(dir, 3, func(string, os.DirEntry, error) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "-entry limit") {
		t.Fatalf("Walk accepted aggregate overflow: %v", err)
	}
}

func TestWalkBoundedEnforcesDepthBeforeVisitingEntry(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "one", "two")
	if err := os.MkdirAll(deep, 0o700); err != nil {
		t.Fatal(err)
	}
	visitedDeep := false
	err := WalkBounded(dir, WalkLimits{MaxEntries: 10, MaxDepth: 1}, func(path string, _ os.DirEntry, err error) error {
		if path == deep {
			visitedDeep = true
		}
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "depth limit") {
		t.Fatalf("WalkBounded accepted excessive depth: %v", err)
	}
	if visitedDeep {
		t.Fatal("WalkBounded visited an entry beyond the depth ceiling")
	}
}
