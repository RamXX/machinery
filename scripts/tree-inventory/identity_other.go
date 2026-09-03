//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

package main

import (
	"fmt"
	"os"
)

func nativeWitness(*os.File, os.FileInfo) (string, error) {
	return "", fmt.Errorf("native filesystem witnesses are unavailable on this platform")
}
