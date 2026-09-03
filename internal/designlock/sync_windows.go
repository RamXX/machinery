//go:build windows

package designlock

import (
	"os"
)

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return dir.Close()
}

// Windows reports synthesized POSIX permission bits; ACL confinement is
// enforced by the private directory and retained-root transaction protocol.
func publishSentinelPermissionsSafe(os.FileMode) bool { return true }
