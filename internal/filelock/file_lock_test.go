package filelock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func privateLockTestDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestAcquireFileWaitContextLivesBesideTheResource(t *testing.T) {
	dir := privateLockTestDir(t)
	first, err := AcquireFileWaitContext(context.Background(), dir, ".resource.lock")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(dir, ".resource.lock"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("lock file is not a regular file inside the guarded directory: %v, %v", info, err)
	}
	// A second test binary resolves a different scope-lock namespace; the
	// in-directory lock must not depend on it.
	t.Setenv(filelockTestRootEnv, t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	if lock, err := AcquireFileWaitContext(ctx, dir, ".resource.lock"); lock != nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second holder from another lock namespace = lock %v, err %v; want to wait on the one file", lock, err)
	}
	acquired := make(chan *Lock, 1)
	errs := make(chan error, 1)
	go func() {
		lock, err := AcquireFileWaitContext(context.Background(), dir, ".resource.lock")
		if err != nil {
			errs <- err
			return
		}
		acquired <- lock
	}()
	select {
	case <-acquired:
		t.Fatal("waiter acquired while the first lock was held")
	case err := <-errs:
		t.Fatalf("waiter failed instead of waiting: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	select {
	case lock := <-acquired:
		if err := lock.Release(); err != nil {
			t.Fatal(err)
		}
	case err := <-errs:
		t.Fatal(err)
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not acquire after release")
	}
}

func TestAcquireFileWaitContextHonorsCancellationWhileWaiting(t *testing.T) {
	dir := privateLockTestDir(t)
	holder, err := AcquireFileWaitContext(context.Background(), dir, "held.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Release()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		lock, err := AcquireFileWaitContext(ctx, dir, "held.lock")
		if lock != nil {
			err = errors.Join(errors.New("acquired a held lock"), lock.Release())
		}
		done <- err
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled waiter = %v; want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("canceled waiter did not return")
	}
}

func TestAcquireFileWaitContextRejectsUnsafeInputsBeforeOpen(t *testing.T) {
	originalOpen := filelockOpenFile
	t.Cleanup(func() { filelockOpenFile = originalOpen })
	filelockOpenFile = func(*os.Root, string) (*os.File, error) {
		t.Fatal("rejected acquisition opened a lock file")
		return nil, errors.New("unreachable")
	}
	dir := privateLockTestDir(t)
	var nilContext context.Context
	if lock, err := AcquireFileWaitContext(nilContext, dir, "x.lock"); lock != nil || err == nil || !strings.Contains(err.Error(), "nil context") {
		t.Fatalf("nil context = lock %v, err %v", lock, err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if lock, err := AcquireFileWaitContext(canceled, dir, "x.lock"); lock != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled = lock %v, err %v", lock, err)
	}
	for _, name := range []string{"", ".", "..", "a/b", `a\b`} {
		if lock, err := AcquireFileWaitContext(context.Background(), dir, name); lock != nil || err == nil || !strings.Contains(err.Error(), "one path segment") {
			t.Fatalf("name %q = lock %v, err %v; want path-segment rejection", name, lock, err)
		}
	}
	if lock, err := AcquireFileWaitContext(context.Background(), filepath.Join(dir, "missing"), "x.lock"); lock != nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing directory = lock %v, err %v", lock, err)
	}
	if runtime.GOOS != "windows" {
		open := filepath.Join(dir, "open")
		if err := os.Mkdir(open, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(open, 0o755); err != nil {
			t.Fatal(err)
		}
		if lock, err := AcquireFileWaitContext(context.Background(), open, "x.lock"); lock != nil || err == nil || !strings.Contains(err.Error(), "private real directory") {
			t.Fatalf("non-private directory = lock %v, err %v", lock, err)
		}
		link := filepath.Join(dir, "link")
		if err := os.Symlink(t.TempDir(), link); err != nil {
			t.Fatal(err)
		}
		if lock, err := AcquireFileWaitContext(context.Background(), link, "x.lock"); lock != nil || err == nil || !strings.Contains(err.Error(), "private real directory") {
			t.Fatalf("symlinked directory = lock %v, err %v", lock, err)
		}
	}
}
