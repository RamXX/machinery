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

func packGeneratedDirectoryMode() os.FileMode { return 0o777 }
func packGeneratedFileMode() os.FileMode      { return 0o666 }
