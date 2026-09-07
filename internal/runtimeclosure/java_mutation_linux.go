//go:build linux

package runtimeclosure

import (
	"os"
	"strconv"
	"syscall"
)

// javaMutationSentinel observes kernel mutation events for one already-open
// launcher file across a witness window. The Linux implementation is an
// inotify watch attached through the descriptor's /proc path, so the watch
// stays bound to the opened inode even if the path is later replaced. Any
// queued event proves the launcher was rewritten or moved during the window
// even when the kernel's coarse inode timestamps make the before and after
// states compare equal; an external replace-restore cannot un-queue these
// events. Overflow or an ambiguous drain fails closed (reports a mutation).
type javaMutationSentinel struct {
	watch int
}

func newJavaFileMutationSentinel(file *os.File) (*javaMutationSentinel, error) {
	watch, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		return nil, err
	}
	procPath := "/proc/self/fd/" + strconv.Itoa(int(file.Fd()))
	// IN_ATTRIB is deliberately excluded: access-time updates caused by the
	// broker's own reads must not be mistaken for mutations. IN_MODIFY covers
	// truncate/rewrite, the same-inode ABA class this sentinel exists for.
	const mask = syscall.IN_MODIFY | syscall.IN_MOVE_SELF | syscall.IN_DELETE_SELF
	if _, err := syscall.InotifyAddWatch(watch, procPath, mask); err != nil {
		_ = syscall.Close(watch)
		return nil, err
	}
	return &javaMutationSentinel{watch: watch}, nil
}

// Mutated reports whether any watched mutation event was observed since the
// sentinel was created. It must be called at most once, at the end of the
// witness window.
func (s *javaMutationSentinel) Mutated() bool {
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

func (s *javaMutationSentinel) Close() error {
	if s == nil {
		return nil
	}
	return syscall.Close(s.watch)
}
