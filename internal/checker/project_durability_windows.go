//go:build windows

package checker

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	checkerGenericRead         = 0x80000000
	checkerGenericWrite        = 0x40000000
	checkerFileShareRead       = 0x00000001
	checkerFileShareWrite      = 0x00000002
	checkerFileShareDelete     = 0x00000004
	checkerBackupSemanticsFlag = 0x02000000
)

var (
	checkerKernel32         = syscall.NewLazyDLL("kernel32.dll")
	checkerReOpenFile       = checkerKernel32.NewProc("ReOpenFile")
	checkerFlushFileBuffers = checkerKernel32.NewProc("FlushFileBuffers")
	checkerCloseHandle      = checkerKernel32.NewProc("CloseHandle")
	checkerGetFileInfo      = checkerKernel32.NewProc("GetFileInformationByHandle")
)

// syncRootDirectory durably flushes directory metadata on Windows. Root.Open
// first acquires the exact directory under the design capability; ReOpenFile
// upgrades that same object to a write-capable backup-semantics handle without
// resolving its ambient path again.
func syncRootDirectory(root *os.Root, rel string) error {
	rooted, err := root.Open(rel)
	if err != nil {
		return fmt.Errorf("open rooted checker transaction directory %s: %w", rel, err)
	}
	handle, _, reopenErr := checkerReOpenFile.Call(
		rooted.Fd(),
		checkerGenericRead|checkerGenericWrite,
		checkerFileShareRead|checkerFileShareWrite|checkerFileShareDelete,
		checkerBackupSemanticsFlag,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(rooted.Close(), fmt.Errorf("reopen rooted checker transaction directory %s for durability: %w", rel, reopenErr))
	}
	flushed, _, flushErr := checkerFlushFileBuffers.Call(handle)
	closed, _, closeErr := checkerCloseHandle.Call(handle)
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush rooted checker transaction directory %s: %w", rel, flushErr))
	}
	if closed == 0 {
		errs = append(errs, fmt.Errorf("close rooted checker transaction directory %s: %w", rel, closeErr))
	}
	errs = append(errs, rooted.Close())
	return errors.Join(errs...)
}

// Windows Go file modes do not expose ACL privacy and commonly report 0666 or
// 0777 even when the requested creation mode was 0600/0700. Safety is instead
// enforced by the non-symlink regular-file/directory checks and confinement.
func projectionControlPermissionsSafe(os.FileMode) bool {
	return true
}

func projectionFileIdentity(file *os.File, _ os.FileInfo) (string, error) {
	var info syscall.ByHandleFileInformation
	ok, _, infoErr := checkerGetFileInfo.Call(file.Fd(), uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return "", fmt.Errorf("identify rooted checker transaction file: %w", infoErr)
	}
	return fmt.Sprintf("%d:%d:%d", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
}
