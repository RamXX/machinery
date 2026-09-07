//go:build unix

package tdd

import (
	"syscall"
	"testing"
)

// mkfifo creates a real FIFO special file on the supported unix platforms.
func mkfifo(t testing.TB, path string) error {
	return syscall.Mkfifo(path, 0o644)
}
