//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package dirscan

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

type mutationWatchID = uint64

// mutationChannel is one kqueue. Each watched directory contributes an
// EVFILT_VNODE registration on a private duplicate of the caller's
// descriptor, so the watch follows the open directory rather than a path and
// the channel owns the descriptor's lifetime. Notes latch instead of queuing,
// so the channel cannot overflow.
type mutationChannel struct {
	queue int
	held  []int
}

// watchedNotes deliberately excludes NOTE_ATTRIB: the enumeration's own reads
// must never be mistaken for a mutation.
const watchedNotes = syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_DELETE |
	syscall.NOTE_RENAME | syscall.NOTE_REVOKE

func newMutationChannel() (*mutationChannel, error) {
	queue, err := syscall.Kqueue()
	if err != nil {
		return nil, fmt.Errorf("open directory mutation channel: %w", err)
	}
	syscall.CloseOnExec(queue)
	return &mutationChannel{queue: queue}, nil
}

func (c *mutationChannel) watch(dir *os.File) (mutationWatchID, error) {
	held, err := syscall.Dup(int(dir.Fd()))
	if err != nil {
		return 0, fmt.Errorf("watch directory for mutation events: %w", err)
	}
	syscall.CloseOnExec(held)
	change := syscall.Kevent_t{
		Ident:  uint64(held),
		Filter: syscall.EVFILT_VNODE,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
		Fflags: watchedNotes,
	}
	if _, err := syscall.Kevent(c.queue, []syscall.Kevent_t{change}, nil, nil); err != nil {
		return 0, fmt.Errorf("watch directory for mutation events: %w", errors.Join(err, syscall.Close(held)))
	}
	c.held = append(c.held, held)
	return uint64(held), nil
}

// drain polls the queue. An unexpected error leaves the latched state
// unknown, so it reports a mutation for every watch on the channel.
func (c *mutationChannel) drain() (bool, []mutationWatchID) {
	var ids []mutationWatchID
	events := make([]syscall.Kevent_t, 16)
	for {
		n, err := syscall.Kevent(c.queue, nil, events, &syscall.Timespec{})
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return true, ids
		}
		for i := 0; i < n; i++ {
			ids = append(ids, events[i].Ident)
		}
		if n < len(events) {
			return false, ids
		}
	}
}

func (c *mutationChannel) close() error {
	err := syscall.Close(c.queue)
	for _, held := range c.held {
		err = errors.Join(err, syscall.Close(held))
	}
	c.held = nil
	return err
}
