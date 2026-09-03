//go:build windows

package pack

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	packGenericWrite        = 0x40000000
	packFileShareRead       = 0x00000001
	packFileShareWrite      = 0x00000002
	packFileShareDelete     = 0x00000004
	packBackupSemanticsFlag = 0x02000000
)

var packReOpenFile = syscall.NewLazyDLL("kernel32.dll").NewProc("ReOpenFile")

func syncPackDirectory(root *os.Root) error {
	// ReOpenFile preserves the identity held by os.Root across a parent path
	// swap while granting the GENERIC_WRITE access FlushFileBuffers requires.
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	handle, _, reopenErr := packReOpenFile.Call(
		dir.Fd(), packGenericWrite,
		packFileShareRead|packFileShareWrite|packFileShareDelete,
		packBackupSemanticsFlag,
	)
	if handle == uintptr(syscall.InvalidHandle) {
		return errors.Join(fmt.Errorf("reopen pack directory for durability: %w", reopenErr), dir.Close())
	}
	writable := os.NewFile(handle, root.Name())
	return errors.Join(writable.Sync(), writable.Close(), dir.Close())
}
