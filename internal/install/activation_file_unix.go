//go:build !windows

package install

import "os"

func openActivationExecutable(path string) (*os.File, error) {
	return os.Open(path)
}
