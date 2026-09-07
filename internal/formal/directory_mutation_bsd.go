//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package formal

import (
	"errors"
	"os"
	"syscall"
)

// formalMutationSentinel observes kernel mutation events for one directory
// across a witness window. The kqueue implementation attaches an EVFILT_VNODE
// note to the directory: any latched event proves the directory (or its entry
// set) was mutated during the window even when timestamp granularity hides
// the difference between the observed states. Overflow-free by construction
// (the note latches, it does not queue); an ambiguous drain fails closed.
type formalMutationSentinel struct {
	queue int
	dir   *os.File
}

func newFormalDirectoryMutationSentinel(path string) (*formalMutationSentinel, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	queue, err := syscall.Kqueue()
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	// NOTE_ATTRIB is deliberately excluded: access-time updates caused by the
	// inventory's own reads must not be mistaken for mutations.
	change := syscall.Kevent_t{
		Ident:  uint64(dir.Fd()),
		Filter: syscall.EVFILT_VNODE,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
		Fflags: syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_DELETE | syscall.NOTE_RENAME | syscall.NOTE_REVOKE,
	}
	if _, err := syscall.Kevent(queue, []syscall.Kevent_t{change}, nil, nil); err != nil {
		_ = syscall.Close(queue)
		_ = dir.Close()
		return nil, err
	}
	return &formalMutationSentinel{queue: queue, dir: dir}, nil
}

// Mutated reports whether any watched mutation event was observed since the
// sentinel was created. It must be called at most once, at the end of the
// witness window.
func (s *formalMutationSentinel) Mutated() bool {
	if s == nil {
		return false
	}
	events := make([]syscall.Kevent_t, 8)
	for {
		n, err := syscall.Kevent(s.queue, nil, events, &syscall.Timespec{})
		if n > 0 {
			return true
		}
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return true
		}
		return false
	}
}

func (s *formalMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return errors.Join(syscall.Close(s.queue), s.dir.Close())
}
