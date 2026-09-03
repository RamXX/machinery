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
