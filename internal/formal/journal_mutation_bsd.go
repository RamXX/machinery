//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package formal

import (
	"os"
	"syscall"
)

// formalJournalMutationSentinel observes kernel mutation events for one
// already-open journal file across the retained-authority witness window.
// The kqueue implementation attaches an EVFILT_VNODE note to the authority's
// own open descriptor, so the watch stays bound to the opened inode even if
// the journal path is later replaced: any latched event proves the journal
// content was rewritten during the window even when timestamp granularity
// hides the difference between the observed states. Overflow-free by
// construction (the note latches, it does not queue); an ambiguous drain
// fails closed.
type formalJournalMutationSentinel struct {
	queue   int
	latched bool
}

func newFormalJournalMutationSentinel(file *os.File) (*formalJournalMutationSentinel, error) {
	queue, err := syscall.Kqueue()
	if err != nil {
		return nil, err
	}
	// NOTE_ATTRIB is deliberately excluded: access-time updates caused by the
	// authority's own reads must not be mistaken for mutations. NOTE_DELETE
	// and NOTE_RENAME are excluded because recovery legitimately renames the
	// retained journal (isolation, restore, quarantine) before finally
	// unlinking it. NOTE_WRITE and NOTE_EXTEND cover truncate/rewrite through
	// the name or any retained descriptor, the same-inode ABA class this
	// sentinel exists for.
	change := syscall.Kevent_t{
		Ident:  uint64(file.Fd()),
		Filter: syscall.EVFILT_VNODE,
		Flags:  syscall.EV_ADD | syscall.EV_CLEAR,
		Fflags: syscall.NOTE_WRITE | syscall.NOTE_EXTEND | syscall.NOTE_REVOKE,
	}
	if _, err := syscall.Kevent(queue, []syscall.Kevent_t{change}, nil, nil); err != nil {
		_ = syscall.Close(queue)
		return nil, err
	}
	return &formalJournalMutationSentinel{queue: queue}, nil
}

// Mutated reports whether any watched mutation event was observed since the
// sentinel was created or last consulted. Observation latches: once a
// mutation is seen, every later call reports it, so a window that consumed
// the event cannot be re-entered as if nothing had happened.
func (s *formalJournalMutationSentinel) Mutated() bool {
	if s == nil {
		return false
	}
	if s.latched {
		return true
	}
	events := make([]syscall.Kevent_t, 8)
	for {
		n, err := syscall.Kevent(s.queue, nil, events, &syscall.Timespec{})
		if n > 0 {
			s.latched = true
			return true
		}
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			s.latched = true
			return true
		}
		return false
	}
}

func (s *formalJournalMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return syscall.Close(s.queue)
}
