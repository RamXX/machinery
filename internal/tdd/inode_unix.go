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
	return inodeIdentity{dev: widenIdentityField(st.Dev), ino: widenIdentityField(st.Ino)}, true
}

// widenIdentityField widens one fixed-width stat identity field to the
// comparable key width. syscall.Stat_t does not agree on those widths across
// the unix targets this file builds for: Dev is a signed 32-bit value on
// darwin and an unsigned 64-bit value on linux/amd64. Widening through a type
// parameter keeps one correct expression for every target instead of a
// per-target copy of inodeKey, and it is never an identity conversion in the
// generic body, so it needs no linter suppression on either platform.
func widenIdentityField[T ~int16 | ~int32 | ~int64 | ~uint16 | ~uint32 | ~uint64](field T) uint64 {
	return uint64(field)
}

type inodeIdentity struct{ dev, ino uint64 }
