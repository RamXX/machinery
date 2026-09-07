//go:build windows

package tdd

import "os"

// inodeKey has no portable windows equivalent at the os.FileInfo level, so
// hardlink-alias detection reports no key there; the supported native
// assurance platforms (darwin/arm64, linux/amd64) always detect aliases.
func inodeKey(fi os.FileInfo) (inodeIdentity, bool) {
	return inodeIdentity{}, false
}

type inodeIdentity struct{ dev, ino uint64 }
