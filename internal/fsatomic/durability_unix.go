//go:build !windows

package fsatomic

import (
	"errors"
	"os"
)

func syncDirectory(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
