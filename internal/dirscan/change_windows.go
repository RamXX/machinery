//go:build windows

package dirscan

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type fileBasicInfo struct {
	CreationTime   int64
	LastAccessTime int64
	LastWriteTime  int64
	ChangeTime     int64
	FileAttributes uint32
}

// directoryChangeID asks Windows for ChangeTime through the retained
// directory handle. LastWriteTime alone is not an adequate namespace ABA
// witness and may be restored or coarsened by a filesystem.
func directoryChangeID(dir *os.File, _ os.FileInfo) (string, error) {
	var info fileBasicInfo
	if err := windows.GetFileInformationByHandleEx(
		windows.Handle(dir.Fd()),
		windows.FileBasicInfo,
		(*byte)(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		return "", fmt.Errorf("read directory change time: %w", err)
	}
	return fmt.Sprintf("%d", info.ChangeTime), nil
}
