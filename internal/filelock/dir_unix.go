//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package filelock

import (
	"fmt"
	"os"
)

func validateLockDir(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("lock directory %s must be a private real directory", path)
	}
	return nil
}
