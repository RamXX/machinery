//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import (
	"errors"
	"fmt"
	"os"
)

func syncCheckerSnapshotDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open checker snapshot directory: %w", err)
	}
	if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
		return fmt.Errorf("sync checker snapshot directory: %w", err)
	}
	return nil
}
