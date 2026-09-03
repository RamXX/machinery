//go:build !windows

package hook

import (
	"errors"
	"fmt"
	"os"
)

func syncStateDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := errors.Join(f.Sync(), f.Close()); err != nil {
		return fmt.Errorf("sync hook state directory %s: %w", dir, err)
	}
	return nil
}

func replaceStateFile(temp, target string) error {
	return replaceStateFileAtomic(temp, target)
}
