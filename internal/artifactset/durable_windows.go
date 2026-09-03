//go:build windows

package artifactset

import (
	"fmt"
	"os"
	"syscall"
)

const (
	fileFlagBackupSemantics = 0x02000000
)

var (
	artifactKernel32        = syscall.NewLazyDLL("kernel32.dll")
	artifactFlushFileBuffer = artifactKernel32.NewProc("FlushFileBuffers")
)

func txOpenRoot(path string) (*os.Root, *os.File, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("inspect artifact root before opening: %w", err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, nil, fmt.Errorf("artifact root must remain a real directory while opening")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open artifact directory root: %w", err)
	}
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		_ = root.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("inspect opened artifact directory root: %w", err)
		}
		return nil, nil, fmt.Errorf("artifact directory changed identity while opening root")
	}
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		_ = root.Close()
		return nil, nil, fmt.Errorf("encode directory sync path: %w", err)
	}
	handle, err := syscall.CreateFile(pathPtr, syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, fileFlagBackupSemantics, 0)
	if err != nil {
		_ = root.Close()
		return nil, nil, fmt.Errorf("open directory sync handle: %w", err)
	}
	syncFile := os.NewFile(uintptr(handle), path)
	syncInfo, statErr := syncFile.Stat()
	if statErr != nil || !os.SameFile(inside, syncInfo) {
		_ = syncFile.Close()
		_ = root.Close()
		if statErr != nil {
			return nil, nil, fmt.Errorf("inspect directory sync handle: %w", statErr)
		}
		return nil, nil, fmt.Errorf("artifact directory changed identity while opening sync handle")
	}
	return root, syncFile, nil
}

func txOpenSyncRoot(root *os.Root) (*os.File, error) {
	inside, err := root.Lstat(".")
	if err != nil || !inside.IsDir() {
		return nil, fmt.Errorf("inspect retained artifact root: %w", err)
	}
	pathPtr, err := syscall.UTF16PtrFromString(root.Name())
	if err != nil {
		return nil, fmt.Errorf("encode retained directory sync path: %w", err)
	}
	handle, err := syscall.CreateFile(pathPtr, syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, fileFlagBackupSemantics, 0)
	if err != nil {
		return nil, fmt.Errorf("open retained directory sync handle: %w", err)
	}
	syncFile := os.NewFile(uintptr(handle), root.Name())
	opened, statErr := syncFile.Stat()
	if statErr != nil || !os.SameFile(inside, opened) {
		_ = syncFile.Close()
		return nil, fmt.Errorf("retained artifact root changed while opening sync handle")
	}
	return syncFile, nil
}

func txSyncHeld(syncFile *os.File) error {
	result, _, flushErr := artifactFlushFileBuffer.Call(syncFile.Fd())
	if result == 0 {
		return fmt.Errorf("flush rooted directory buffers: %w", flushErr)
	}
	return nil
}
