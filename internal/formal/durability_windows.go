//go:build windows

package formal

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func syncFormalDirectory(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return dir.Close()
}

func formalNativeFileWitness(file *os.File, _ os.FileInfo) (string, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("inspect native Windows formal file identity: %w", err)
	}
	return fmt.Sprintf("windows:%x:%08x%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), nil
}

type formalWindowsBasicInfo struct {
	CreationTime   int64
	LastAccessTime int64
	LastWriteTime  int64
	ChangeTime     int64
	FileAttributes uint32
}

func formalNativeInventoryWitness(file *os.File, info os.FileInfo) (string, error) {
	identity, err := formalNativeFileWitness(file, info)
	if err != nil {
		return "", err
	}
	var basic formalWindowsBasicInfo
	if err := windows.GetFileInformationByHandleEx(
		windows.Handle(file.Fd()),
		windows.FileBasicInfo,
		(*byte)(unsafe.Pointer(&basic)),
		uint32(unsafe.Sizeof(basic)),
	); err != nil {
		return "", fmt.Errorf("inspect native Windows formal inventory change identity: %w", err)
	}
	return fmt.Sprintf("%s:change:%x", identity, basic.ChangeTime), nil
}
