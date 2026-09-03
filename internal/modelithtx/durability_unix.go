//go:build !windows

package modelithtx

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func syncModelithDirectory(root *os.Root, rel string) error {
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func modelithNativeEntryWitness(_ *os.File, info os.FileInfo) (string, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return "", fmt.Errorf("entry lacks a stable native Unix identity")
	}
	return fmt.Sprintf("unix:%x:%x", stat.Dev, stat.Ino), nil
}
