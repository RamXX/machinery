//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package runtimeclosure

import (
	"os"
	"syscall"
)

// javaMutationSentinel observes kernel mutation events for one already-open
// launcher file across a witness window. The kqueue implementation attaches
// an EVFILT_VNODE note to the broker's own open descriptor, so any rewrite of
// that inode latches an event even when timestamp granularity hides the
// difference between the observed states. An ambiguous drain fails closed
// (reports a mutation).
type javaMutationSentinel struct {
	queue int
}

func newJavaFileMutationSentinel(file *os.File) (*javaMutationSentinel, error) {
	queue, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	// NOTE_ATTRIB is deliberately excluded: access-time updates caused by the
	// broker's own reads must not be mistaken for mutations. NOTE_WRITE and
	// NOTE_EXTEND cover truncate/rewrite, the same-inode ABA class this
	// sentinel exists for.
	change := syscall.Kevent_t{
		Ident:  uint64(file.Fd()),
		Filter: syscall.EVFILT_VNODE,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
		Fflags: syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_DELETE | syscall.NOTE_RENAME | syscall.NOTE_REVOKE,
	}
	if _, err := syscall.Kevent(queue, []syscall.Kevent_t{change}, nil, nil); err != nil {
		_ = syscall.Close(queue)
		return nil, err
	}
	return &javaMutationSentinel{queue: queue}, nil
}

// Mutated reports whether any watched mutation event was observed since the
// sentinel was created. It must be called at most once, at the end of the
// witness window.
func (s *javaMutationSentinel) Mutated() bool {
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

func (s *javaMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return syscall.Close(s.queue)
}
