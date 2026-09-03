package safefile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRejectsSameSizeMutationAfterInitialRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mutable")
	if err := os.WriteFile(path, []byte("before"), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := afterInitialRead
	t.Cleanup(func() { afterInitialRead = prior })
	afterInitialRead = func(got string) {
		if got == path {
			afterInitialRead = func(string) {}
			if err := os.WriteFile(path, []byte("change"), 0o600); err != nil {
				t.Error(err)
			}
		}
	}
	if _, err := Read(path, "fixture", 16); err == nil || !strings.Contains(err.Error(), "changed while reading") {
		t.Fatalf("Read accepted same-size mutation: %v", err)
	}
}

func TestReadContinuousAppenderCannotExtendReadPastWitnessedSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "growing")
	if err := os.WriteFile(path, []byte("seed"), 0o600); err != nil {
		t.Fatal(err)
	}
	prior := afterInitialRead
	t.Cleanup(func() { afterInitialRead = prior })
	done := make(chan struct{})
	stopped := make(chan struct{})
	afterInitialRead = func(got string) {
		if got != path {
			return
		}
		afterInitialRead = func(string) {}
		first := make(chan struct{})
		go func() {
			defer close(stopped)
			file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
			if err != nil {
				t.Error(err)
				close(first)
				return
			}
			defer func() {
				if err := file.Close(); err != nil {
					t.Error(err)
				}
			}()
			for index := 0; ; index++ {
				if _, err := file.WriteString("growth"); err != nil {
					t.Error(err)
					if index == 0 {
						close(first)
					}
					return
				}
				if index == 0 {
					close(first)
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
		<-first
	}
	_, err := Read(path, "fixture", 16)
	close(done)
	<-stopped
	if err == nil || !strings.Contains(err.Error(), "changed while reading") {
		t.Fatalf("Read accepted growth beyond witnessed size: %v", err)
	}
}

func TestReadRejectsSparseOversizeBeforeAllocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(1025); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path, "fixture", 1024); err == nil || !strings.Contains(err.Error(), "1024-byte limit") {
		t.Fatalf("Read accepted oversized sparse file: %v", err)
	}
}

func TestReadRejectsSymlink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("body"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Read(link, "fixture", 16); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("Read accepted symlink: %v", err)
	}
}
