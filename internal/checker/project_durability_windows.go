//go:build windows

package checker

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	checkerKernel32    = syscall.NewLazyDLL("kernel32.dll")
	checkerGetFileInfo = checkerKernel32.NewProc("GetFileInformationByHandle")
)

// syncRootDirectory verifies the retained directory capability. Windows does
// not list directory handles as supported FlushFileBuffers targets; transaction
// files are flushed individually and the journal makes recovery convergent.
func syncRootDirectory(root *os.Root, rel string) error {
	rooted, err := root.Open(rel)
	if err != nil {
		return fmt.Errorf("open rooted checker transaction directory %s: %w", rel, err)
	}
	return rooted.Close()
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
