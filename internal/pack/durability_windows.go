//go:build windows

package pack

import (
	"os"
)

func syncPackDirectory(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return dir.Close()
}

func packJournalPermissionsSafe(os.FileMode) bool { return true }
