//go:build windows

package install

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	installGenericWrite        = 0x40000000
	installFileShareRead       = 0x00000001
	installFileShareWrite      = 0x00000002
	installFileShareDelete     = 0x00000004
	installOpenExisting        = 3
	installBackupSemantics     = 0x02000000
	installMoveReplaceExisting = 0x00000001
	installMoveWriteThrough    = 0x00000008
)

var (
	installKernel32         = syscall.NewLazyDLL("kernel32.dll")
	installCreateFileW      = installKernel32.NewProc("CreateFileW")
	installFlushFileBuffers = installKernel32.NewProc("FlushFileBuffers")
	installCloseHandle      = installKernel32.NewProc("CloseHandle")
	installMoveFileExW      = installKernel32.NewProc("MoveFileExW")
	installGetFileInfo      = installKernel32.NewProc("GetFileInformationByHandle")
	installReOpenFile       = installKernel32.NewProc("ReOpenFile")
)

// Windows synthesizes POSIX permission bits. Confinement is structural and
// relies on the user's config/cache ACL rather than a Unix mode assertion.
func privateFilePermissionsOK(os.FileInfo) bool           { return true }
func installDirectoryOwnedByCurrentUser(os.FileInfo) bool { return true }

func syncDirectoryPath(path string) error {
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	handle, _, createErr := installCreateFileW.Call(
		uintptr(unsafe.Pointer(wide)), installGenericWrite,
		installFileShareRead|installFileShareWrite|installFileShareDelete,
		0, installOpenExisting, installBackupSemantics, 0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return fmt.Errorf("open install directory %s for durability: %w", path, createErr)
	}
	flushed, _, flushErr := installFlushFileBuffers.Call(handle)
	closed, _, closeErr := installCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush install directory %s: %w", path, flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close install directory %s: %w", path, closeErr))
	}
	return errors.Join(errs...)
}

func syncRootDirectoryFile(f *os.File, path string) error {
	handle, _, reopenErr := installReOpenFile.Call(
		f.Fd(), installGenericWrite,
		installFileShareRead|installFileShareWrite|installFileShareDelete,
		installBackupSemantics,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return fmt.Errorf("reopen rooted install directory %s for durability: %w", path, reopenErr)
	}
	flushed, _, flushErr := installFlushFileBuffers.Call(handle)
	closed, _, closeErr := installCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush rooted install directory %s: %w", path, flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close rooted install directory %s: %w", path, closeErr))
	}
	return errors.Join(errs...)
}

func renameInstallPath(oldPath, newPath string) error {
	oldWide, err := syscall.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newWide, err := syscall.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	moved, _, moveErr := installMoveFileExW.Call(
		uintptr(unsafe.Pointer(oldWide)), uintptr(unsafe.Pointer(newWide)),
		installMoveReplaceExisting|installMoveWriteThrough,
	)
	if moved == 0 {
		return fmt.Errorf("durably rename install path %s to %s: %w", oldPath, newPath, moveErr)
	}
	return nil
}

func stableInstallDirIdentity(path string, _ os.FileInfo) (string, error) {
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	handle, _, createErr := installCreateFileW.Call(
		uintptr(unsafe.Pointer(wide)), installGenericWrite,
		installFileShareRead|installFileShareWrite|installFileShareDelete,
		0, installOpenExisting, installBackupSemantics, 0,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return "", fmt.Errorf("open install anchor %s: %w", path, createErr)
	}
	var info syscall.ByHandleFileInformation
	ok, _, infoErr := installGetFileInfo.Call(handle, uintptr(unsafe.Pointer(&info)))
	closed, _, closeErr := installCloseHandle.Call(handle)
	if ok == 0 {
		return "", fmt.Errorf("identify install anchor %s: %w", path, infoErr)
	}
	if closed == 0 {
		return "", fmt.Errorf("close install anchor %s: %w", path, closeErr)
	}
	return fmt.Sprintf("%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
}
