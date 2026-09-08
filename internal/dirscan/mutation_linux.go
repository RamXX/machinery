//go:build linux

package dirscan

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type mutationWatchID = int32

// mutationChannel is one inotify instance. Watches are added through
// /proc/self/fd so the kernel resolves the caller's descriptor rather than a
// path, which keeps an os.Root enumeration inside its own confinement.
type mutationChannel struct {
	fd int
}

// watchedEvents deliberately excludes IN_ATTRIB and IN_ACCESS: the
// enumeration's own reads must never be mistaken for a mutation.
const watchedEvents = syscall.IN_MODIFY | syscall.IN_CREATE | syscall.IN_DELETE |
	syscall.IN_MOVED_FROM | syscall.IN_MOVED_TO | syscall.IN_MOVE_SELF | syscall.IN_DELETE_SELF

func newMutationChannel() (*mutationChannel, error) {
	fd, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("open directory mutation channel: %w", err)
	}
	return &mutationChannel{fd: fd}, nil
}

func (c *mutationChannel) watch(dir *os.File) (mutationWatchID, error) {
	descriptor := fmt.Sprintf("/proc/self/fd/%d", dir.Fd())
	id, err := syscall.InotifyAddWatch(c.fd, descriptor, watchedEvents)
	if err != nil {
		return 0, fmt.Errorf("watch directory for mutation events: %w", err)
	}
	return int32(id), nil
}

// drain reads every queued record. A queue overflow carries no usable watch
// descriptor, and an unexpected read error leaves the queue in an unknown
// state, so both report a mutation for every watch on the channel.
func (c *mutationChannel) drain() (bool, []mutationWatchID) {
	var ids []mutationWatchID
	buf := make([]byte, 16*syscall.SizeofInotifyEvent+syscall.NAME_MAX+1)
	for {
		n, err := syscall.Read(c.fd, buf)
		if err == syscall.EINTR {
			continue
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return false, ids
		}
		if err != nil {
			return true, ids
		}
		if n <= 0 {
			return false, ids
		}
		for offset := 0; offset+syscall.SizeofInotifyEvent <= n; {
			event := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			if event.Mask&syscall.IN_Q_OVERFLOW != 0 || event.Wd < 0 {
				return true, ids
			}
			ids = append(ids, event.Wd)
			offset += syscall.SizeofInotifyEvent + int(event.Len)
		}
	}
}

func (c *mutationChannel) close() error {
	return syscall.Close(c.fd)
}
