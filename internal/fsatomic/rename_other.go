//go:build !darwin && !linux && !windows

package fsatomic

import (
	"fmt"
	"os"
)

func renameNoReplaceBase(_ *os.Root, oldname string, _ *os.Root, newname string) error {
	return fmt.Errorf("atomic no-replace rename is unsupported on this platform")
}

func removePrivateDirectory(root *os.Root, _ *os.Root, name string) error {
	// name is the fresh, identity-revalidated retirement name produced by
	// closeEmpty, never the protocol-visible quarantine name.
	return root.Remove(name)
}
