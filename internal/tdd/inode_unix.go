//go:build unix

package tdd

import (
	"os"
	"syscall"
)

// inodeKey extracts a comparable device/inode identity for hardlink-alias
// detection on the supported unix platforms.
func inodeKey(fi os.FileInfo) (inodeIdentity, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return inodeIdentity{}, false
	}
	return inodeIdentity{dev: uint64(st.Dev), ino: st.Ino}, true
}

type inodeIdentity struct{ dev, ino uint64 }
