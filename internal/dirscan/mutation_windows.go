//go:build windows

package dirscan

import "os"

type mutationWatchID = uint64

// mutationChannel is unused on Windows. The portable half of the witness
// there is the NTFS ChangeTime read from the retained directory handle
// (change_windows.go), which the filesystem stamps per namespace operation
// and which utimes-style restoration does not roll back, so the coarse
// inode-clock blindness this channel exists to cover does not apply. A nil
// implementation means "no event channel needed": Watch succeeds and reports
// no mutations, and callers keep the change-stamp witness they already had.
type mutationChannel struct{}

func newMutationChannel() (*mutationChannel, error) { return nil, nil }

func (c *mutationChannel) watch(*os.File) (mutationWatchID, error) { return 0, nil }

func (c *mutationChannel) drain() (bool, []mutationWatchID) { return false, nil }

func (c *mutationChannel) close() error { return nil }
