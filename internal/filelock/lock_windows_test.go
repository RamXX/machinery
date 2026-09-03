//go:build windows

package filelock

import (
	"context"
	"errors"
	"testing"
)

func TestWindowsAcquireWaitNeverEntersBlockingKernelLock(t *testing.T) {
	original := filelockLockFile
	t.Cleanup(func() { filelockLockFile = original })

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	filelockLockFile = func(*Lock) (bool, error) {
		calls++
		cancel()
		return false, errorLockViolation
	}

	lock, err := AcquireWaitContext(ctx, t.TempDir())
	if lock != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Windows canceled wait = lock %v, err %v", lock, err)
	}
	if calls != 1 {
		t.Fatalf("Windows nonblocking lock attempts = %d, want 1", calls)
	}
}
