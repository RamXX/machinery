// Package safefile provides bounded, identity-stable regular-file reads.
package safefile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

var afterInitialRead = func(string) {}

// Read reads exactly the pre-open size of a non-symlink regular file, subject
// to maxBytes, and rejects identity or metadata changes through completion.
func Read(path, kind string, maxBytes int64) (_ []byte, retErr error) {
	if maxBytes < 0 {
		return nil, fmt.Errorf("%s byte limit must be non-negative", kind)
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", kind)
	}
	if before.Size() < 0 || before.Size() > maxBytes {
		return nil, fmt.Errorf("%s size %d exceeds %d-byte limit", kind, before.Size(), maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%s changed identity while opening", kind)
	}
	body, err := io.ReadAll(io.LimitReader(file, before.Size()+1))
	if err != nil {
		return nil, err
	}
	afterInitialRead(path)
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	again, err := io.ReadAll(io.LimitReader(file, before.Size()+1))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	pathAfter, pathErr := os.Lstat(path)
	if pathErr != nil || !os.SameFile(before, after) || !os.SameFile(before, pathAfter) ||
		after.Size() != before.Size() || after.Mode() != before.Mode() || !after.ModTime().Equal(before.ModTime()) ||
		int64(len(body)) != before.Size() || int64(len(again)) != before.Size() || !bytes.Equal(body, again) {
		return nil, errors.Join(pathErr, fmt.Errorf("%s changed while reading", kind))
	}
	return body, nil
}
