//go:build windows

package cachestage

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	stageGenericRead     = 0x80000000
	stageGenericWrite    = 0x40000000
	stageShareRead       = 0x00000001
	stageShareWrite      = 0x00000002
	stageShareDelete     = 0x00000004
	stageBackupSemantics = 0x02000000
)

var (
	stageKernel32         = syscall.NewLazyDLL("kernel32.dll")
	stageReOpenFile       = stageKernel32.NewProc("ReOpenFile")
	stageFlushFileBuffers = stageKernel32.NewProc("FlushFileBuffers")
	stageCloseHandle      = stageKernel32.NewProc("CloseHandle")
)

func syncStageDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	handle, _, reopenErr := stageReOpenFile.Call(
		dir.Fd(), stageGenericRead|stageGenericWrite,
		stageShareRead|stageShareWrite|stageShareDelete,
		stageBackupSemantics,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(fmt.Errorf("reopen cache directory for durability: %w", reopenErr), dir.Close())
	}
	flushed, _, flushErr := stageFlushFileBuffers.Call(handle)
	closed, _, closeErr := stageCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush cache directory: %w", flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close cache directory: %w", closeErr))
	}
	errs = append(errs, dir.Close())
	return errors.Join(errs...)
}
