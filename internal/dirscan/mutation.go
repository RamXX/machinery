package dirscan

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// MutationChannel is the granularity-independent half of a directory
// consistency witness.
//
// The portable half of the witness is the native change stamp
// (directoryChangeID): a Unix inode ctime, or the NTFS ChangeTime. On Linux
// before the multigrain-timestamp work (mainline 6.13) every inode timestamp
// is taken from the kernel's coarse clock, whose resolution is one timer tick
// (1 ms at CONFIG_HZ=1000, 4 ms at 250). A create-then-delete, or a
// rename-and-back, that completes inside one tick leaves ctime, mtime, size
// and link count all identical, so the stamp cannot tell an ABA mutation from
// a quiescent directory. STATX_CHANGE_COOKIE, which would expose i_version,
// is stripped from userspace requests by those kernels, so there is no finer
// stamp to fall back to.
//
// A kernel mutation-event channel does not depend on any clock. Every watched
// namespace operation queues an event, and no later operation can un-queue it,
// so an ABA that restores the directory's apparent state is still reported.
// One channel serves a whole traversal: the per-directory watches are cheap,
// while the channel itself is the scarce resource (an inotify instance).
//
// A MutationChannel is safe for concurrent use.
type MutationChannel struct {
	mu      sync.Mutex
	impl    *mutationChannel
	closed  bool
	all     bool
	mutated map[mutationWatchID]bool
}

// MutationWatch is one directory's view of a MutationChannel. It stays valid
// until the channel is closed.
type MutationWatch struct {
	channel *MutationChannel
	id      mutationWatchID
	armed   bool
}

// NewMutationChannel opens one kernel mutation-event channel. It fails when
// the platform has an event interface that could not be opened, so a caller
// that needs a granularity-independent witness can refuse the enumeration
// instead of accepting it on the strength of a stamp that may be blind.
func NewMutationChannel() (*MutationChannel, error) {
	impl, err := newMutationChannel()
	if err != nil {
		return nil, err
	}
	return &MutationChannel{impl: impl, mutated: map[mutationWatchID]bool{}}, nil
}

// Watch arms the channel for one already-opened directory. The watch is
// established through the caller's descriptor, never through a path, so it
// cannot resolve to a different directory than the one being enumerated.
func (c *MutationChannel) Watch(dir *os.File) (*MutationWatch, error) {
	if c == nil {
		return nil, fmt.Errorf("directory mutation channel is not open")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, fmt.Errorf("directory mutation channel is closed")
	}
	if c.impl == nil {
		return &MutationWatch{}, nil
	}
	id, err := c.impl.watch(dir)
	if err != nil {
		return nil, err
	}
	return &MutationWatch{channel: c, id: id, armed: true}, nil
}

// hasEventChannel reports whether this platform backs the witness with a
// kernel event channel. Windows does not: its NTFS ChangeTime stamp is not
// clock-coarse in the way a pre-multigrain Unix inode ctime is, so the
// channel there is a no-op.
func (c *MutationChannel) hasEventChannel() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.impl != nil
}

// Close releases the channel and every watch on it.
func (c *MutationChannel) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.impl == nil {
		return nil
	}
	impl := c.impl
	c.impl = nil
	return impl.close()
}

// Mutated reports whether the kernel observed a namespace mutation of the
// watched directory since the watch was armed. Draining is cumulative: each
// call folds the pending events into the per-watch record, so an earlier
// observation is never lost by a later call, and the answer never regresses
// from true back to false. A drain that cannot be trusted (a queue overflow,
// or an unexpected read error) reports a mutation for every watch.
func (w *MutationWatch) Mutated() bool {
	if w == nil || !w.armed {
		return false
	}
	return w.channel.observed(w.id)
}

func (c *MutationChannel) observed(id mutationWatchID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.impl == nil {
		return c.all || c.mutated[id]
	}
	all, ids := c.impl.drain()
	if all {
		c.all = true
	}
	for _, seen := range ids {
		c.mutated[seen] = true
	}
	return c.all || c.mutated[id]
}

// watchDirectory is the single-directory convenience used by enumerations
// whose witness window does not outlive one function call. It is a variable
// so a test can reproduce a host that cannot arm the event channel.
var watchDirectory = func(dir *os.File) (*MutationChannel, *MutationWatch, error) {
	channel, err := NewMutationChannel()
	if err != nil {
		return nil, nil, err
	}
	watch, err := channel.Watch(dir)
	if err != nil {
		return nil, nil, errors.Join(err, channel.Close())
	}
	return channel, watch, nil
}
