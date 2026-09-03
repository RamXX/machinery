//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package designlock

import "os"

func syncDirectory(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	return errorsJoin(f.Sync(), f.Close())
}

func syncRootDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errorsJoin(directory.Sync(), directory.Close())
}
