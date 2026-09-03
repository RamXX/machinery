//go:build !windows

package main

import (
	"errors"
	"os"
)

func replaceArchive(root *os.Root, oldName, newName string) error {
	return root.Rename(oldName, newName)
}

func syncArchiveDirectory(root *os.Root) error {
	directory, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(directory.Sync(), directory.Close())
}
