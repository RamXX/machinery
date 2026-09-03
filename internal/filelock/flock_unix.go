//go:build darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd

package filelock

import (
	"context"
	"errors"
	"syscall"
)

func reserveExclusiveLock(context.Context, string, string, bool) error {
	return nil
}

func releaseExclusiveReservation(*Lock, bool) error {
	return nil
}

func tryExclusiveLock(lock *Lock) (bool, error) {
	err := syscall.Flock(int(lock.file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return false, nil
	}
	return false, err
}

func unlockExclusiveLock(lock *Lock) error {
	return syscall.Flock(int(lock.file.Fd()), syscall.LOCK_UN)
}
