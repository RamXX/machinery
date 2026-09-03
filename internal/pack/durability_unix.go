//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package pack

import (
	"errors"
	"os"
)

func syncPackDirectory(root *os.Root) error {
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(f.Sync(), f.Close())
}

func packJournalPermissionsSafe(mode os.FileMode) bool { return mode.Perm() == 0o600 }
