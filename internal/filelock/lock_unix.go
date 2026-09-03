//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package filelock

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// Lock is an OS advisory lock; the kernel releases it on process death.
type Lock struct {
	file *os.File
	root *os.Root
}

func Acquire(scope string) (*Lock, error) {
	return acquire(scope, true)
}

// AcquireWait takes the same exclusive advisory lock as Acquire, but waits
// until the current holder releases it. Snapshot readers and design writers
// use this form: transient overlap is coordination, not an operator error.
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
	operation := syscall.LOCK_EX
	if nonBlocking {
		operation |= syscall.LOCK_NB
	}
	if err := syscall.Flock(int(f.Fd()), operation); err != nil {
		return nil, errors.Join(fmt.Errorf("another operation holds the lock for %s: %w", scope, err), f.Close(), location.root.Close())
	}
	if err := location.validatePathIdentity(); err != nil {
		unlockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		return nil, errors.Join(err, unlockErr, f.Close(), location.root.Close())
	}
	return &Lock{file: f, root: location.root}, nil
}

func (l *Lock) Release() error {
	return errors.Join(syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN), l.file.Close(), l.root.Close())
}
