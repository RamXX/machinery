//go:build windows

package hook

import "os"

func syncStateDirectory(dir string) error {
	opened, err := os.Open(dir)
	if err != nil {
		return err
	}
	return opened.Close()
}

func replaceStateFile(temp, target string) error {
	return replaceStateFileAtomic(temp, target)
}
