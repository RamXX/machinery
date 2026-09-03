//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func nativeWitness(file *os.File, _ os.FileInfo) (string, error) {
	if file == nil {
		return "", fmt.Errorf("native Windows witness requires an opened entry")
	}
	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(syscall.Handle(file.Fd()), &info); err != nil {
		return "", fmt.Errorf("inspect native Windows filesystem identity: %w", err)
	}
	return fmt.Sprintf("windows:%x:%08x%08x:birth:%08x%08x:change:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow, info.CreationTime.HighDateTime, info.CreationTime.LowDateTime, info.LastWriteTime.HighDateTime, info.LastWriteTime.LowDateTime), nil
}
