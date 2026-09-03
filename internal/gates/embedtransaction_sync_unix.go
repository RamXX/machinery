//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package gates

import "os"

func syncEmbedDirectoryHandle(f *os.File) error { return f.Sync() }
