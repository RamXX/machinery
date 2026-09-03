//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package checker

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func syncRootDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	if err := errors.Join(dir.Sync(), dir.Close()); err != nil {
		return fmt.Errorf("sync rooted directory %s: %w", rel, err)
	}
	return nil
}

func projectionControlPermissionsSafe(mode os.FileMode) bool {
	return mode.Perm()&0o077 == 0
}

func projectionFileIdentity(_ *os.File, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "", fmt.Errorf("file has no native stat identity")
	}
	return fmt.Sprintf("%d:%d", stat.Dev, stat.Ino), nil
}
