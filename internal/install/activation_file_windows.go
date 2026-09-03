//go:build windows

package install

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	installGenericRead         = 0x80000000
	installFileAttributeNormal = 0x00000080
)

func openActivationExecutable(path string) (*os.File, error) {
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, _, callErr := installCreateFileW.Call(
		uintptr(unsafe.Pointer(wide)), installGenericRead, installFileShareRead,
		0, installOpenExisting, installFileAttributeNormal, 0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return nil, callErr
	}
	return os.NewFile(handle, path), nil
}
