//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd)

package formal

import "os"

// formalJournalMutationSentinel is the no-channel fallback for platforms
// without a portable kernel mutation-event interface (inotify, kqueue) for an
// already-open file. The hybrid retained-authority witness degrades to its
// stat-metadata comparison, which is exactly the guarantee those platforms
// had before the sentinel existed; nothing is weakened.
type formalJournalMutationSentinel struct{}

func newFormalJournalMutationSentinel(*os.File) (*formalJournalMutationSentinel, error) {
	return nil, nil
}

func (s *formalJournalMutationSentinel) Mutated() bool { return false }

func (s *formalJournalMutationSentinel) Close() error { return nil }
