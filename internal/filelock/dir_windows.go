//go:build windows

package filelock

import (
	"fmt"
	"os"
)

func validateLockDir(path string, info os.FileInfo) error {
	// Windows synthesizes POSIX permission bits (commonly 0777), so applying
	// the Unix 0700 assertion makes every valid lock directory impossible.
	// Confinement here is structural: the cache path must be a real directory;
	// the user's cache-directory ACL remains the native access boundary.
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("lock directory %s must be a private real directory", path)
	}
	return nil
}
