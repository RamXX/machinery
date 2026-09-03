//go:build windows

package modelithtx

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	modelithGenericRead     = 0x80000000
	modelithGenericWrite    = 0x40000000
	modelithShareRead       = 0x00000001
	modelithShareWrite      = 0x00000002
	modelithShareDelete     = 0x00000004
	modelithBackupSemantics = 0x02000000
)

var (
	modelithKernel32         = syscall.NewLazyDLL("kernel32.dll")
	modelithReOpenFile       = modelithKernel32.NewProc("ReOpenFile")
	modelithFlushFileBuffers = modelithKernel32.NewProc("FlushFileBuffers")
	modelithCloseHandle      = modelithKernel32.NewProc("CloseHandle")
)

func syncModelithDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	handle, _, reopenErr := modelithReOpenFile.Call(
		dir.Fd(), modelithGenericRead|modelithGenericWrite,
		modelithShareRead|modelithShareWrite|modelithShareDelete,
		modelithBackupSemantics,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(fmt.Errorf("reopen Modelith directory for durability: %w", reopenErr), dir.Close())
	}
	flushed, _, flushErr := modelithFlushFileBuffers.Call(handle)
	closed, _, closeErr := modelithCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush Modelith directory: %w", flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close Modelith directory: %w", closeErr))
	}
	errs = append(errs, dir.Close())
	return errors.Join(errs...)
}

func modelithNativeEntryWitness(file *os.File, _ os.FileInfo) (string, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("inspect native Windows entry identity: %w", err)
	}
	return fmt.Sprintf("windows:%x:%08x%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), nil
}
