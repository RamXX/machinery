//go:build windows

package fsatomic

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	fsatomicGenericRead     = 0x80000000
	fsatomicGenericWrite    = 0x40000000
	fsatomicShareRead       = 0x00000001
	fsatomicShareWrite      = 0x00000002
	fsatomicShareDelete     = 0x00000004
	fsatomicBackupSemantics = 0x02000000
)

var (
	fsatomicKernel32         = syscall.NewLazyDLL("kernel32.dll")
	fsatomicReOpenFile       = fsatomicKernel32.NewProc("ReOpenFile")
	fsatomicFlushFileBuffers = fsatomicKernel32.NewProc("FlushFileBuffers")
	fsatomicCloseHandle      = fsatomicKernel32.NewProc("CloseHandle")
)

func syncDirectory(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	handle, _, reopenErr := fsatomicReOpenFile.Call(
		dir.Fd(), fsatomicGenericRead|fsatomicGenericWrite,
		fsatomicShareRead|fsatomicShareWrite|fsatomicShareDelete,
		fsatomicBackupSemantics,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(fmt.Errorf("reopen atomic directory for durability: %w", reopenErr), dir.Close())
	}
	flushed, _, flushErr := fsatomicFlushFileBuffers.Call(handle)
	closed, _, closeErr := fsatomicCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush atomic directory: %w", flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close atomic directory: %w", closeErr))
	}
	errs = append(errs, dir.Close())
	return errors.Join(errs...)
}
