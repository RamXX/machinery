//go:build linux

package formal

import (
	"syscall"
)

// formalMutationSentinel observes kernel mutation events for one directory
// across a witness window. The Linux implementation is an inotify watch: any
// queued event proves the directory (or its entry set) was mutated during the
// window even when the kernel's coarse inode timestamps make the before and
// after states compare equal. An external replace-restore cannot un-queue
// these events; they are the granularity-independent half of the hybrid
// witness. Overflow or an ambiguous drain fails closed (reports a mutation).
type formalMutationSentinel struct {
	watch int
}

func newFormalDirectoryMutationSentinel(path string) (*formalMutationSentinel, error) {
	watch, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}
	// IN_ATTRIB is deliberately excluded: access-time updates caused by the
	// inventory's own reads must not be mistaken for mutations.
	const mask = syscall.IN_MODIFY | syscall.IN_CREATE | syscall.IN_DELETE |
		syscall.IN_MOVED_FROM | syscall.IN_MOVED_TO | syscall.IN_MOVE_SELF | syscall.IN_DELETE_SELF
	if _, err := syscall.InotifyAddWatch(watch, path, mask); err != nil {
		_ = syscall.Close(watch)
		return nil, err
	}
	return &formalMutationSentinel{watch: watch}, nil
}

// Mutated reports whether any watched mutation event was observed since the
// sentinel was created. It must be called at most once, at the end of the
// witness window.
func (s *formalMutationSentinel) Mutated() bool {
	if s == nil {
		return false
	}
	buf := make([]byte, 8192)
	for {
		n, err := syscall.Read(s.watch, buf)
		if n > 0 {
			// The queue serves exactly one watch, so any record is either a
			// watched mutation or a queue-overflow notice.
			return true
		}
		if err == syscall.EINTR {
			continue
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return false
		}
		return true
	}
}

func (s *formalMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return syscall.Close(s.watch)
}
