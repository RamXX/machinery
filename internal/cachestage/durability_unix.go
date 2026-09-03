//go:build !windows

package cachestage

import (
	"errors"
	"os"
)

func syncStageDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
