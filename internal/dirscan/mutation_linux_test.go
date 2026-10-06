//go:build linux

package dirscan

import (
	"errors"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestMutationChannelRetriesResourceExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		limit error
		watch bool
		hint  string
	}{
		{name: "instances", limit: syscall.EMFILE, hint: "fs.inotify.max_user_instances"},
		{name: "watches", limit: syscall.ENOSPC, watch: true, hint: "fs.inotify.max_user_watches"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			priorInit, priorWatch := inotifyInit, inotifyAddWatch
			t.Cleanup(func() { inotifyInit, inotifyAddWatch = priorInit, priorWatch })
			dir, err := os.Open(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer dir.Close()
			for _, persistent := range []bool{false, true} {
				calls := 0
				inotifyInit, inotifyAddWatch = priorInit, priorWatch
				if tc.watch {
					inotifyAddWatch = func(fd int, path string, mask uint32) (int, error) {
						calls++
						if persistent || calls == 1 {
							return -1, tc.limit
						}
						return priorWatch(fd, path, mask)
					}
				} else {
					inotifyInit = func(flags int) (int, error) {
						calls++
						if persistent || calls == 1 {
							return -1, tc.limit
						}
						return priorInit(flags)
					}
				}
				channel, _, err := watchDirectory(dir)
				if channel != nil {
					if closeErr := channel.Close(); closeErr != nil {
						t.Fatal(closeErr)
					}
				}
				if persistent {
					if !errors.Is(err, tc.limit) || !strings.Contains(err.Error(), tc.hint) || calls < 2 || calls > 8 {
						t.Fatalf("persistent exhaustion: calls=%d err=%v, want bounded retries and %s", calls, err, tc.hint)
					}
				} else if err != nil || calls != 2 {
					t.Errorf("transient exhaustion not retried: calls=%d err=%v", calls, err)
				}
			}
		})
	}
}
