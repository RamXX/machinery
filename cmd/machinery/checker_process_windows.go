//go:build windows

package main

import (
	"errors"
	"fmt"
	"syscall"
)

const (
	checkerSnapshotGenericRead     = 0x80000000
	checkerSnapshotGenericWrite    = 0x40000000
	checkerSnapshotShareRead       = 0x00000001
	checkerSnapshotShareWrite      = 0x00000002
	checkerSnapshotShareDelete     = 0x00000004
	checkerSnapshotOpenExisting    = 3
	checkerSnapshotBackupSemantics = 0x02000000
)

func syncCheckerSnapshotDir(path string) error {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("encode checker snapshot directory: %w", err)
	}
	handle, err := syscall.CreateFile(
		name,
		checkerSnapshotGenericRead|checkerSnapshotGenericWrite,
		checkerSnapshotShareRead|checkerSnapshotShareWrite|checkerSnapshotShareDelete,
		nil,
		checkerSnapshotOpenExisting,
		checkerSnapshotBackupSemantics,
		0,
	)
	if err != nil {
		return fmt.Errorf("open checker snapshot directory: %w", err)
	}
	return errors.Join(syscall.FlushFileBuffers(handle), syscall.CloseHandle(handle))
}
