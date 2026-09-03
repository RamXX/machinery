//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	runSafeGenericRead      = 0x80000000
	runSafeShareRead        = 0x00000001
	runSafeShareWrite       = 0x00000002
	runSafeShareDelete      = 0x00000004
	runSafeOpenExisting     = 3
	runSafeOpenReparsePoint = 0x00200000
	runSafeBackupSemantics  = 0x02000000
)

func nativeExecutableWitness(file *os.File, _ os.FileInfo) (identity, change string, err error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", "", fmt.Errorf("inspect native Windows executable identity: %w", err)
	}
	identity = fmt.Sprintf("windows:%x:%08x%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime)
	change = fmt.Sprintf("mtime:%08x%08x", info.LastWriteTime.HighDateTime, info.LastWriteTime.LowDateTime)
	return identity, change, nil
}

func nativeSymlinkWitness(path string, _ os.FileInfo) (identity, change string, retErr error) {
	pointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", "", err
	}
	handle, err := syscall.CreateFile(pointer, runSafeGenericRead, runSafeShareRead|runSafeShareWrite|runSafeShareDelete, nil,
		runSafeOpenExisting, runSafeOpenReparsePoint|runSafeBackupSemantics, 0)
	if err != nil {
		return "", "", fmt.Errorf("open native Windows executable symlink identity: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, syscall.CloseHandle(handle)) }()
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(handle, &info); err != nil {
		return "", "", fmt.Errorf("inspect native Windows executable symlink identity: %w", err)
	}
	identity = fmt.Sprintf("windows:%x:%08x%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime)
	change = fmt.Sprintf("mtime:%08x%08x", info.LastWriteTime.HighDateTime, info.LastWriteTime.LowDateTime)
	return identity, change, nil
}
