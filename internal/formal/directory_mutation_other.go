//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd)

package formal

// formalMutationSentinel is the no-channel fallback for platforms without a
// portable kernel mutation-event interface (inotify, kqueue). The hybrid
// witness degrades to its stat-metadata comparison, which is exactly the
// guarantee those platforms had before the sentinel existed; nothing is
// weakened.
type formalMutationSentinel struct{}

func newFormalDirectoryMutationSentinel(string) (*formalMutationSentinel, error) {
	return nil, nil
}

func (s *formalMutationSentinel) Mutated() bool { return false }

func (s *formalMutationSentinel) Close() error { return nil }
