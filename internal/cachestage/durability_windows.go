//go:build windows

package cachestage

import (
	"os"
)

func syncStageDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	return dir.Close()
}
