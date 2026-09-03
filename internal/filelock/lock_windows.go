//go:build windows

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileFailImmediately = 0x00000001
	lockfileExclusiveLock   = 0x00000002
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

type Lock struct {
	file       *os.File
	root       *os.Root
	overlapped syscall.Overlapped
}

func Acquire(scope string) (*Lock, error) {
	return acquire(scope, true)
}

// AcquireWait takes the same exclusive advisory lock as Acquire, but waits
// until the current holder releases it.
func AcquireWait(scope string) (*Lock, error) {
	return acquire(scope, false)
}

func acquire(scope string, nonBlocking bool) (*Lock, error) {
	return acquireWithHooks(scope, nonBlocking, acquireHooks{})
}

func acquireWithHooks(scope string, nonBlocking bool, hooks acquireHooks) (*Lock, error) {
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
	flags := uintptr(lockfileExclusiveLock)
	if nonBlocking {
		flags |= lockfileFailImmediately
	}
	r1, _, callErr := procLockFileEx.Call(f.Fd(), flags, 0, 1, 0, uintptr(unsafe.Pointer(&l.overlapped)))
	if r1 == 0 {
		return nil, errors.Join(fmt.Errorf("another operation holds the lock for %s: %w", scope, callErr), f.Close(), location.root.Close())
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

func (l *Lock) Release() error {
	r1, _, callErr := procUnlockFileEx.Call(l.file.Fd(), 0, 1, 0, uintptr(unsafe.Pointer(&l.overlapped)))
	closeErr := errors.Join(l.file.Close(), l.root.Close())
	if r1 == 0 {
		return errors.Join(callErr, closeErr)
	}
	return closeErr
}
