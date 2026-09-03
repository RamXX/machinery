//go:build windows

package main

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const (
	releaseFileFlagBackupSemantics = 0x02000000
)

var (
	releaseKernel32         = syscall.NewLazyDLL("kernel32.dll")
	releaseFlushFileBuffers = releaseKernel32.NewProc("FlushFileBuffers")
)

func replaceArchive(root *os.Root, oldName, newName string) error {
	return root.Rename(oldName, newName)
}

func syncArchiveDirectory(root *os.Root) error {
	// O_CREATE requests a write-capable handle without O_WRONLY/O_RDWR's
	// FILE_NON_DIRECTORY_FILE constraint. The already-existing rooted "."
	// directory is never created or replaced.
	directory, err := root.OpenFile(".", os.O_RDONLY|os.O_CREATE|releaseFileFlagBackupSemantics, 0o755)
	if err != nil {
		return fmt.Errorf("open rooted release output directory for durability: %w", err)
	}
	flushed, _, flushErr := releaseFlushFileBuffers.Call(directory.Fd())
	closeErr := directory.Close()
	var errs []error
	if flushed == 0 {
		errs = append(errs, fmt.Errorf("flush release output directory: %w", flushErr))
	}
	if closeErr != nil {
		errs = append(errs, fmt.Errorf("close release output directory: %w", closeErr))
	}
	return errors.Join(errs...)
}
