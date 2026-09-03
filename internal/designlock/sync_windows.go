//go:build windows

package designlock

import (
	"fmt"
	"syscall"
)

const designFileFlagBackupSemantics = 0x02000000

var designFlushFileBuffers = syscall.NewLazyDLL("kernel32.dll").NewProc("FlushFileBuffers")

func syncDirectory(path string) error {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	h, err := syscall.CreateFile(ptr, syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, designFileFlagBackupSemantics, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(h)
	result, _, flushErr := designFlushFileBuffers.Call(uintptr(h))
	if result == 0 {
		return fmt.Errorf("flush design directory: %w", flushErr)
	}
	return nil
}
