//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd)

package runtimeclosure

import "os"

// javaMutationSentinel is the no-channel fallback for platforms without a
// portable kernel mutation-event interface (inotify, kqueue). The hybrid
// witness degrades to its stat-metadata comparison, which is exactly the
// guarantee those platforms had before the sentinel existed; nothing is
// weakened.
type javaMutationSentinel struct{}

func newJavaFileMutationSentinel(*os.File) (*javaMutationSentinel, error) {
	return nil, nil
}

func (s *javaMutationSentinel) Mutated() bool { return false }

func (s *javaMutationSentinel) Close() error { return nil }
