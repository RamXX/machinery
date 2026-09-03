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
