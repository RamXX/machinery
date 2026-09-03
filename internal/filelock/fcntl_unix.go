//go:build aix || (solaris && !illumos)

package filelock

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"syscall"
)

var fcntlProcessLocks = struct {
	sync.Mutex
	held map[string]struct{}
}{held: make(map[string]struct{})}

// AIX and Solaris expose process-scoped fcntl record locks rather than
// descriptor-scoped flock locks. Reserve the identity before opening the lock
// file: closing a second descriptor for the same inode would otherwise release
// this process's existing record lock and permit cross-process split brain.
func reserveExclusiveLock(ctx context.Context, scope, identity string, nonBlocking bool) error {
	tryReserve := func() (bool, error) {
		fcntlProcessLocks.Lock()
		defer fcntlProcessLocks.Unlock()
		if _, held := fcntlProcessLocks.held[identity]; held {
			return false, nil
		}
		fcntlProcessLocks.held[identity] = struct{}{}
		return true, nil
	}
	if nonBlocking {
		reserved, err := tryReserve()
		if err != nil {
			return err
		}
		if !reserved {
			return fmt.Errorf("%w for %s", errLockHeld, scope)
		}
		return nil
	}
	return waitForLock(ctx, scope, tryReserve)
}

func releaseExclusiveReservation(lock *Lock, descriptorClosed bool) error {
	if !descriptorClosed {
		return fmt.Errorf("retain in-process lock reservation for %s because its descriptor did not close", lock.identity)
	}
	fcntlProcessLocks.Lock()
	defer fcntlProcessLocks.Unlock()
	if _, held := fcntlProcessLocks.held[lock.identity]; !held {
		return fmt.Errorf("lock %s has no in-process reservation", lock.identity)
	}
	delete(fcntlProcessLocks.held, lock.identity)
	return nil
}

func tryExclusiveLock(lock *Lock) (bool, error) {
	record := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: int16(io.SeekStart)}
	if err := syscall.FcntlFlock(lock.file.Fd(), syscall.F_SETLK, &record); err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EAGAIN) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func unlockExclusiveLock(lock *Lock) error {
	record := syscall.Flock_t{Type: syscall.F_UNLCK, Whence: int16(io.SeekStart)}
	return syscall.FcntlFlock(lock.file.Fd(), syscall.F_SETLK, &record)
}
