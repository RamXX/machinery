//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd || windows)

package dirscan

import (
	"fmt"
	"os"
)

type mutationWatchID = uint64

// mutationChannel has no implementation on platforms without a kernel
// directory mutation-event interface. Opening one fails, and every caller
// that needs a granularity-independent witness refuses the enumeration rather
// than accepting it on a change stamp whose resolution is unknown. This is
// the fail-closed direction: a platform machinery does not build for cannot
// silently degrade the ABA defense.
type mutationChannel struct{}

func newMutationChannel() (*mutationChannel, error) {
	return nil, fmt.Errorf("this platform has no directory mutation-event channel")
}

func (c *mutationChannel) watch(*os.File) (mutationWatchID, error) {
	return 0, fmt.Errorf("this platform has no directory mutation-event channel")
}

func (c *mutationChannel) drain() (bool, []mutationWatchID) { return true, nil }

func (c *mutationChannel) close() error { return nil }
