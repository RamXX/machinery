//go:build darwin

package fsatomic

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func renameNoReplaceBase(oldRoot *os.Root, oldname string, newRoot *os.Root, newname string) error {
	oldDir, err := oldRoot.Open(".")
	if err != nil {
		return err
	}
	newDir, err := newRoot.Open(".")
	if err != nil {
		return errors.Join(err, oldDir.Close())
	}
	renameErr := unix.RenameatxNp(int(oldDir.Fd()), oldname, int(newDir.Fd()), newname, unix.RENAME_EXCL)
	return errors.Join(renameErr, newDir.Close(), oldDir.Close())
}

func removePrivateDirectory(root *os.Root, _ *os.Root, name string) error {
	// name is the fresh, identity-revalidated retirement name produced by
	// closeEmpty, never the protocol-visible quarantine name.
	return root.Remove(name)
}
