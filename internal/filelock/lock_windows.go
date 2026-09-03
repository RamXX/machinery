//go:build windows

package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
	errorLockViolation      = syscall.Errno(33)
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
	filelockLockFile = func(lock *Lock) (bool, error) {
		r1, _, callErr := procLockFileEx.Call(lock.file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0, 1, 0, uintptr(unsafe.Pointer(&lock.overlapped)))
		if r1 != 0 {
			return true, nil
		}
		return false, callErr
	}
)

type Lock struct {
	file       *os.File
	root       *os.Root
	overlapped syscall.Overlapped
}

func Acquire(scope string) (*Lock, error) {
	return acquireWithContext(context.Background(), scope, true, acquireHooks{})
}

// AcquireWait takes the same exclusive advisory lock as Acquire, but waits
// until the current holder releases it.
func AcquireWait(scope string) (*Lock, error) {
	ctx, cancel := defaultAcquireWaitContext()
	defer cancel()
	return AcquireWaitContext(ctx, scope)
}

// AcquireWaitContext takes the same exclusive advisory lock as Acquire and
// waits only until ctx is canceled. It uses bounded nonblocking attempts so a
// stuck holder can never strand an uncancellable kernel wait.
func AcquireWaitContext(ctx context.Context, scope string) (*Lock, error) {
	return acquireWithContext(ctx, scope, false, acquireHooks{})
}

func acquireWithHooks(scope string, nonBlocking bool, hooks acquireHooks) (*Lock, error) {
	ctx := context.Background()
	if !nonBlocking {
		var cancel context.CancelFunc
		ctx, cancel = defaultAcquireWaitContext()
		defer cancel()
	}
	return acquireWithContext(ctx, scope, nonBlocking, hooks)
}

func acquireWithContext(ctx context.Context, scope string, nonBlocking bool, hooks acquireHooks) (*Lock, error) {
	if ctx == nil {
		return nil, fmt.Errorf("acquire lock for %s: nil context", scope)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("acquire lock for %s: %w", scope, err)
	}
	location, err := openLockLocation(scope)
	if err != nil {
		return nil, err
	}
	f, err := location.openFile()
	if err != nil {
		return nil, errors.Join(err, location.root.Close())
	}
	if hooks.afterLockOpen != nil {
		if err := hooks.afterLockOpen(location.dir); err != nil {
			return nil, errors.Join(err, f.Close(), location.root.Close())
		}
	}
	l := &Lock{file: f, root: location.root}
	tryLock := hooks.tryLock
	if tryLock == nil {
		tryLock = tryExclusiveLock
	}
	if nonBlocking {
		acquired, lockErr := tryLock(l)
		if lockErr != nil {
			return nil, errors.Join(fmt.Errorf("acquire lock for %s: %w", scope, lockErr), f.Close(), location.root.Close())
		}
		if !acquired {
			return nil, errors.Join(fmt.Errorf("%w for %s", errLockHeld, scope), f.Close(), location.root.Close())
		}
	} else if err := waitForLock(ctx, scope, func() (bool, error) { return tryLock(l) }); err != nil {
		return nil, errors.Join(err, f.Close(), location.root.Close())
	}
	if err := location.validatePathIdentity(); err != nil {
		unlockResult, _, unlockErr := procUnlockFileEx.Call(f.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&l.overlapped)))
		if unlockResult != 0 {
			unlockErr = nil
		}
		return nil, errors.Join(err, unlockErr, f.Close(), location.root.Close())
	}
	return l, nil
}

func tryExclusiveLock(lock *Lock) (bool, error) {
	acquired, callErr := filelockLockFile(lock)
	if acquired {
		return true, nil
	}
	if errors.Is(callErr, errorLockViolation) {
		return false, nil
	}
	return false, callErr
}

func (l *Lock) Release() error {
	r1, _, callErr := procUnlockFileEx.Call(l.file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&l.overlapped)))
	closeErr := errors.Join(l.file.Close(), l.root.Close())
	if r1 == 0 {
		return errors.Join(callErr, closeErr)
	}
	return closeErr
}
