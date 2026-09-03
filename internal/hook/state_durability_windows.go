//go:build windows

package hook

import (
	"errors"
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	hookGenericWrite        = 0x40000000
	hookFileShareRead       = 0x00000001
	hookFileShareWrite      = 0x00000002
	hookFileShareDelete     = 0x00000004
	hookOpenExisting        = 3
	hookBackupSemantics     = 0x02000000
	hookMoveReplaceExisting = 0x00000001
	hookMoveWriteThrough    = 0x00000008
)

var (
	hookKernel32         = syscall.NewLazyDLL("kernel32.dll")
	hookCreateFileW      = hookKernel32.NewProc("CreateFileW")
	hookFlushFileBuffers = hookKernel32.NewProc("FlushFileBuffers")
	hookCloseHandle      = hookKernel32.NewProc("CloseHandle")
	hookMoveFileExW      = hookKernel32.NewProc("MoveFileExW")
)

func syncStateDirectory(dir string) error {
	wide, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}
	handle, _, createErr := hookCreateFileW.Call(
		uintptr(unsafe.Pointer(wide)), hookGenericWrite,
		hookFileShareRead|hookFileShareWrite|hookFileShareDelete,
		0, hookOpenExisting, hookBackupSemantics, 0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return fmt.Errorf("open hook state directory %s for durability: %w", dir, createErr)
	}
	flushed, _, flushErr := hookFlushFileBuffers.Call(handle)
	closed, _, closeErr := hookCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush hook state directory %s: %w", dir, flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close hook state directory %s: %w", dir, closeErr))
	}
	return errors.Join(errs...)
}

func replaceStateFile(temp, target string) error {
	from, err := syscall.UTF16PtrFromString(temp)
	if err != nil {
		return err
	}
	to, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	moved, _, moveErr := hookMoveFileExW.Call(
		uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)),
		hookMoveReplaceExisting|hookMoveWriteThrough,
	)
	if moved == 0 {
		return fmt.Errorf("durably replace hook state %s with %s: %w", target, temp, moveErr)
	}
	return syncStateDirectory(filepath.Dir(target))
}
