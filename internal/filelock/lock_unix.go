//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

package filelock

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// Lock is an OS advisory lock; the kernel releases it on process death.
type Lock struct {
	file     *os.File
	root     *os.Root
	identity string
}

func Acquire(scope string) (*Lock, error) {
	return acquireWithContext(context.Background(), scope, true, acquireHooks{})
}

// AcquireWait takes the same exclusive advisory lock as Acquire, but waits
// until the current holder releases it. Snapshot readers and design writers
// use this form: transient overlap is coordination, not an operator error.
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
	l := &Lock{root: location.root, identity: location.name}
	if err := reserveExclusiveLock(ctx, scope, l.identity, nonBlocking); err != nil {
		return nil, errors.Join(err, location.root.Close())
	}
	f, err := location.openFile()
	if err != nil {
		return nil, errors.Join(err, releaseExclusiveReservation(l, true), location.root.Close())
	}
	l.file = f
	if hooks.afterLockOpen != nil {
		if err := hooks.afterLockOpen(location.dir); err != nil {
			return nil, errors.Join(err, closeUnixLock(l, false))
		}
	}
	tryLock := hooks.tryLock
	if tryLock == nil {
		tryLock = tryExclusiveLock
	}
	if nonBlocking {
		acquired, lockErr := tryLock(l)
		if lockErr != nil {
			return nil, errors.Join(fmt.Errorf("acquire lock for %s: %w", scope, lockErr), closeUnixLock(l, false))
		}
		if !acquired {
			return nil, errors.Join(fmt.Errorf("%w for %s", errLockHeld, scope), closeUnixLock(l, false))
		}
	} else if err := waitForLock(ctx, scope, func() (bool, error) { return tryLock(l) }); err != nil {
		return nil, errors.Join(err, closeUnixLock(l, false))
	}
	if err := location.validatePathIdentity(); err != nil {
		return nil, errors.Join(err, closeUnixLock(l, true))
	}
	return l, nil
}

func (l *Lock) Release() error {
	return closeUnixLock(l, true)
}

func closeUnixLock(lock *Lock, unlock bool) error {
	var unlockErr error
	if unlock {
		unlockErr = unlockExclusiveLock(lock)
	}
	closeErr := lock.file.Close()
	reservationErr := releaseExclusiveReservation(lock, closeErr == nil)
	return errors.Join(unlockErr, closeErr, reservationErr, lock.root.Close())
}
