//go:build windows

package gates

import (
	"os"
)

func syncEmbedDirectoryHandle(*os.File) error { return nil }
