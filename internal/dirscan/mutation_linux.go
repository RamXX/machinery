//go:build linux

package dirscan

import (
	"errors"
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

var inotifyInit = syscall.InotifyInit1
var inotifyAddWatch = syscall.InotifyAddWatch

func newMutationChannel() (*mutationChannel, error) {
	fd, err := retryMutationResource(func() (int, error) {
		return inotifyInit(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	})
	if err != nil {
		return nil, fmt.Errorf("open directory mutation channel: %w", err)
	}
	return &mutationChannel{fd: fd}, nil
}

func (c *mutationChannel) watch(dir *os.File) (mutationWatchID, error) {
	descriptor := fmt.Sprintf("/proc/self/fd/%d", dir.Fd())
	id, err := retryMutationResource(func() (int, error) {
		return inotifyAddWatch(c.fd, descriptor, watchedEvents)
	})
	if err != nil {
		return 0, fmt.Errorf("watch directory for mutation events: %w", err)
	}
	return int32(id), nil
}

// Resource pressure can clear when another traversal releases its channel.
// Retry only exhaustion, retaining the errno when the bounded wait runs out.
func retryMutationResource(operation func() (int, error)) (int, error) {
	for attempt := 0; ; attempt++ {
		value, err := operation()
		if !errors.Is(err, syscall.EMFILE) && !errors.Is(err, syscall.ENOSPC) {
			return value, err
		}
		if attempt == readAttempts-1 {
			if errors.Is(err, syscall.EMFILE) {
				return value, fmt.Errorf("inotify instance or process file descriptor limit exhausted after %d attempts; raise fs.inotify.max_user_instances or RLIMIT_NOFILE: %w", readAttempts, err)
			}
			return value, fmt.Errorf("inotify watch limit exhausted after %d attempts; raise fs.inotify.max_user_watches: %w", readAttempts, err)
		}
		retryDelay(attempt)
	}
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
