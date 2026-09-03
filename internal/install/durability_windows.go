//go:build windows

package install

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	installReadAttributes  = 0x00000080
	installFileShareRead   = 0x00000001
	installFileShareWrite  = 0x00000002
	installFileShareDelete = 0x00000004
	installOpenExisting    = 3
	installBackupSemantics = 0x02000000
	installOpenReparse     = 0x00200000
)

var (
	installKernel32    = syscall.NewLazyDLL("kernel32.dll")
	installCreateFileW = installKernel32.NewProc("CreateFileW")
	installCloseHandle = installKernel32.NewProc("CloseHandle")
	installGetFileInfo = installKernel32.NewProc("GetFileInformationByHandle")
)

// Windows synthesizes POSIX permission bits. Confinement is structural and
// relies on the user's config/cache ACL rather than a Unix mode assertion.
func privateFilePermissionsOK(os.FileInfo) bool           { return true }
func installDirectoryOwnedByCurrentUser(os.FileInfo) bool { return true }

func syncDirectoryPath(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return dir.Close()
}

func syncRootDirectoryFile(*os.File, string) error { return nil }

func stableInstallDirIdentity(path string, _ os.FileInfo) (string, error) {
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	handle, _, createErr := installCreateFileW.Call(
		uintptr(unsafe.Pointer(wide)), installReadAttributes,
		installFileShareRead|installFileShareWrite|installFileShareDelete,
		0, installOpenExisting, installBackupSemantics|installOpenReparse, 0,
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
