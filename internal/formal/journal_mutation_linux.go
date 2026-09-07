//go:build linux

package formal

import (
	"os"
	"strconv"
	"syscall"
)

// formalJournalMutationSentinel observes kernel mutation events for one
// already-open journal file across the retained-authority witness window.
// The Linux implementation is an inotify watch attached through the
// descriptor's /proc path, so the watch stays bound to the opened inode even
// if the journal path is later replaced. Any queued event proves the journal
// content was rewritten during the window even when the kernel's coarse inode
// timestamps make the before and after states compare equal; an external
// replace-restore cannot un-queue these events. Overflow or an ambiguous
// drain fails closed (reports a mutation).
type formalJournalMutationSentinel struct {
	watch   int
	latched bool
}

func newFormalJournalMutationSentinel(file *os.File) (*formalJournalMutationSentinel, error) {
	watch, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}
	procPath := "/proc/self/fd/" + strconv.Itoa(int(file.Fd()))
	// IN_ATTRIB is deliberately excluded: access-time updates caused by the
	// authority's own reads must not be mistaken for mutations. IN_MOVE_SELF
	// and IN_DELETE_SELF are excluded because recovery legitimately renames
	// the retained journal (isolation, restore, quarantine) before finally
	// unlinking it. IN_MODIFY covers truncate/rewrite through the name or any
	// retained descriptor, the same-inode ABA class this sentinel exists for.
	const mask = syscall.IN_MODIFY
	if _, err := syscall.InotifyAddWatch(watch, procPath, mask); err != nil {
		_ = syscall.Close(watch)
		return nil, err
	}
	return &formalJournalMutationSentinel{watch: watch}, nil
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
	buf := make([]byte, 8192)
	for {
		n, err := syscall.Read(s.watch, buf)
		if n > 0 {
			// The queue serves exactly one watch, so any record is either a
			// watched mutation or a queue-overflow notice.
			s.latched = true
			return true
		}
		if err == syscall.EINTR {
			continue
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return false
		}
		s.latched = true
		return true
	}
}

func (s *formalJournalMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return syscall.Close(s.watch)
}
