//go:build windows

package hook

import (
	"fmt"
	"os"
	"syscall"
)

func hookNativeDirectoryWitness(file *os.File, _ os.FileInfo) (string, error) {
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("inspect native Windows hook state directory identity: %w", err)
	}
	return fmt.Sprintf("windows:%x:%08x%08x:birth:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime), nil
}
