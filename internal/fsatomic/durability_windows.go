//go:build windows

package fsatomic

import (
	"os"
)

func syncDirectory(root *os.Root) error {
	// Windows does not expose directory handles as supported FlushFileBuffers
	// targets. File payloads and journals are flushed before every metadata
	// transition; the recovery protocol therefore supplies convergence after a
	// crash, while this retained-root check preserves confinement.
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return dir.Close()
}
