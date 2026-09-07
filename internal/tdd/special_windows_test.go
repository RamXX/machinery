//go:build windows

package tdd

import "errors"

// mkfifo is unavailable on the unsupported windows platform; special-file
// rejection is exercised on the supported darwin/arm64 and linux/amd64
// native lanes where FIFOs exist.
func mkfifo(t testing.TB, path string) error {
	return errors.New("no FIFO special files on this platform")
}
