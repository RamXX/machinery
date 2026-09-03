//go:build windows

package designlock

import (
	"errors"
	"fmt"
	"os"
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

func syncRootDirectory(root *os.Root) error {
	directory, err := root.OpenFile(".", os.O_RDONLY|os.O_CREATE|designFileFlagBackupSemantics, 0o755)
	if err != nil {
		return fmt.Errorf("open rooted design directory for durability: %w", err)
	}
	flushed, _, flushErr := designFlushFileBuffers.Call(directory.Fd())
	closeErr := directory.Close()
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush rooted design directory: %w", flushErr))
	}
	if closeErr != nil {
		errs = append(errs, fmt.Errorf("close rooted design directory: %w", closeErr))
	}
	return errors.Join(errs...)
}
