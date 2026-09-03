//go:build windows

package formal

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	formalGenericWrite        = 0x40000000
	formalFileShareRead       = 0x00000001
	formalFileShareWrite      = 0x00000002
	formalFileShareDelete     = 0x00000004
	formalBackupSemanticsFlag = 0x02000000
)

var formalReOpenFile = syscall.NewLazyDLL("kernel32.dll").NewProc("ReOpenFile")

func syncFormalDirectory(root *os.Root) error {
	// ReOpenFile is identity-relative to the held os.Root directory handle.
	// Reopening Root.Name with CreateFileW would race a parent path swap.
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	handle, _, reopenErr := formalReOpenFile.Call(
		dir.Fd(), formalGenericWrite,
		formalFileShareRead|formalFileShareWrite|formalFileShareDelete,
		formalBackupSemanticsFlag,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(fmt.Errorf("reopen formal directory for durability: %w", reopenErr), dir.Close())
	}
	writable := os.NewFile(handle, root.Name())
	return errors.Join(writable.Sync(), writable.Close(), dir.Close())
}

func formalNativeFileWitness(file *os.File, _ os.FileInfo) (string, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("inspect native Windows formal file identity: %w", err)
	}
	return fmt.Sprintf("windows:%x:%08x%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), nil
}
