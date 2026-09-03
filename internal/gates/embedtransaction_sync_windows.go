//go:build windows

package gates

import (
	"fmt"
	"os"
	"syscall"
)

var embedFlushFileBuffers = syscall.NewLazyDLL("kernel32.dll").NewProc("FlushFileBuffers")

func syncEmbedDirectoryHandle(f *os.File) error {
	result, _, callErr := embedFlushFileBuffers.Call(f.Fd())
	if result == 0 {
		return fmt.Errorf("flush embed transaction directory: %w", callErr)
	}
	return nil
}
