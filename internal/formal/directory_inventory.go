package formal

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sort"
)

// formalAfterDirectoryInventoryPass is a deterministic mutation seam between
// the two complete inventory passes. Production code always leaves it nil.
var formalAfterDirectoryInventoryPass func(string)

type formalCapturedDirEntry struct {
	name string
	info os.FileInfo
}

func (e formalCapturedDirEntry) Name() string               { return e.name }
func (e formalCapturedDirEntry) IsDir() bool                { return e.info.IsDir() }
func (e formalCapturedDirEntry) Type() fs.FileMode          { return e.info.Mode().Type() }
func (e formalCapturedDirEntry) Info() (os.FileInfo, error) { return e.info, nil }

type formalInventoryEntry struct {
	entry   formalCapturedDirEntry
	witness string
}

type formalDirectorySnapshot struct {
	info    os.FileInfo
	witness string
	entries []formalInventoryEntry
}

func readFormalRootDirectory(root *os.Root, label string) ([]os.DirEntry, error) {
	if root == nil {
		return nil, fmt.Errorf("formal directory %s has nil root authority", label)
	}
	first, err := snapshotFormalRootDirectory(root, label)
	if err != nil {
		return nil, err
	}
	if formalAfterDirectoryInventoryPass != nil {
		formalAfterDirectoryInventoryPass(label)
	}
	second, err := snapshotFormalRootDirectory(root, label)
	if err != nil {
		return nil, err
	}
	if err := compareFormalDirectorySnapshots(first, second, label); err != nil {
		return nil, err
	}
	entries := make([]os.DirEntry, len(second.entries))
	for i := range second.entries {
		entries[i] = second.entries[i].entry
	}
	return entries, nil
}

func snapshotFormalRootDirectory(root *os.Root, label string) (_ formalDirectorySnapshot, retErr error) {
	dir, err := root.Open(".")
	if err != nil {
		return formalDirectorySnapshot{}, err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	before, err := dir.Stat()
	if err != nil || !before.IsDir() {
		return formalDirectorySnapshot{}, errors.Join(err, fmt.Errorf("formal directory %s must be a real directory", label))
	}
	beforeWitness, err := formalNativeInventoryWitness(dir, before)
	if err != nil {
		return formalDirectorySnapshot{}, fmt.Errorf("witness formal directory %s: %w", label, err)
	}
	raw := make([]os.DirEntry, 0, formalDirPageSize)
	for {
		page, readErr := dir.ReadDir(formalDirPageSize)
		if len(raw) > formalDirEntryMax-len(page) {
			return formalDirectorySnapshot{}, fmt.Errorf("formal directory %s exceeds %d-entry limit", label, formalDirEntryMax)
		}
		raw = append(raw, page...)
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return formalDirectorySnapshot{}, readErr
		}
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].Name() < raw[j].Name() })
	entries := make([]formalInventoryEntry, 0, len(raw))
	for i, item := range raw {
		if i > 0 && raw[i-1].Name() == item.Name() {
			return formalDirectorySnapshot{}, fmt.Errorf("formal directory %s returned duplicate entry %q", label, item.Name())
		}
		captured, err := captureFormalInventoryEntry(root, item.Name(), label)
		if err != nil {
			return formalDirectorySnapshot{}, err
		}
		entries = append(entries, captured)
	}
	after, err := dir.Stat()
	if err != nil || !sameFormalInventoryMetadata(before, after) {
		return formalDirectorySnapshot{}, errors.Join(err, fmt.Errorf("formal directory %s changed during inventory pass", label))
	}
	afterWitness, err := formalNativeInventoryWitness(dir, after)
	if err != nil || beforeWitness != afterWitness {
		return formalDirectorySnapshot{}, errors.Join(err, fmt.Errorf("formal directory %s changed native witness during inventory pass", label))
	}
	return formalDirectorySnapshot{info: after, witness: afterWitness, entries: entries}, nil
}

func captureFormalInventoryEntry(root *os.Root, name, label string) (_ formalInventoryEntry, retErr error) {
	before, err := root.Lstat(name)
	if err != nil {
		return formalInventoryEntry{}, fmt.Errorf("inspect formal directory %s entry %q: %w", label, name, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() && !before.IsDir() {
		return formalInventoryEntry{}, fmt.Errorf("formal directory %s entry %q must be a regular non-symlink file or real directory", label, name)
	}
	file, err := root.Open(name)
	if err != nil {
		return formalInventoryEntry{}, fmt.Errorf("open formal directory %s entry %q: %w", label, name, err)
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !sameFormalInventoryMetadata(before, opened) {
		return formalInventoryEntry{}, errors.Join(err, fmt.Errorf("formal directory %s entry %q changed while opening", label, name))
	}
	witness, err := formalNativeInventoryWitness(file, opened)
	if err != nil {
		return formalInventoryEntry{}, fmt.Errorf("witness formal directory %s entry %q: %w", label, name, err)
	}
	after, err := root.Lstat(name)
	if err != nil || !sameFormalInventoryMetadata(opened, after) {
		return formalInventoryEntry{}, errors.Join(err, fmt.Errorf("formal directory %s entry %q changed while witnessing", label, name))
	}
	finalOpened, err := file.Stat()
	if err != nil || !sameFormalInventoryMetadata(after, finalOpened) {
		return formalInventoryEntry{}, errors.Join(err, fmt.Errorf("formal directory %s entry %q changed after witnessing", label, name))
	}
	finalWitness, err := formalNativeInventoryWitness(file, finalOpened)
	if err != nil || witness != finalWitness {
		return formalInventoryEntry{}, errors.Join(err, fmt.Errorf("formal directory %s entry %q changed native witness during inventory pass", label, name))
	}
	return formalInventoryEntry{entry: formalCapturedDirEntry{name: name, info: finalOpened}, witness: finalWitness}, nil
}

func compareFormalDirectorySnapshots(first, second formalDirectorySnapshot, label string) error {
	if !sameFormalInventoryMetadata(first.info, second.info) || first.witness != second.witness {
		return fmt.Errorf("formal directory %s changed between inventory passes", label)
	}
	if len(first.entries) != len(second.entries) {
		return fmt.Errorf("formal directory %s inventory changed between passes", label)
	}
	for i := range first.entries {
		left, right := first.entries[i], second.entries[i]
		if left.entry.name != right.entry.name || !sameFormalInventoryMetadata(left.entry.info, right.entry.info) || left.witness != right.witness {
			return fmt.Errorf("formal directory %s entry %q changed between inventory passes", label, left.entry.name)
		}
	}
	return nil
}

func sameFormalInventoryMetadata(left, right os.FileInfo) bool {
	return left != nil && right != nil && os.SameFile(left, right) && left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}

func readFormalDirectory(path string) (_ []os.DirEntry, retErr error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("formal directory %s must be a real directory", path)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	opened, openedWitness, err := snapshotFormalRootIdentity(root)
	if err != nil || !sameFormalInventoryMetadata(before, opened) {
		return nil, errors.Join(err, fmt.Errorf("formal directory %s changed identity while opening", path))
	}
	entries, err := readFormalRootDirectory(root, path)
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil || !sameFormalInventoryMetadata(before, after) {
		return nil, errors.Join(err, fmt.Errorf("formal directory %s changed while enumerating", path))
	}
	finalOpened, finalWitness, err := snapshotFormalRootIdentity(root)
	if err != nil || !sameFormalInventoryMetadata(after, finalOpened) || openedWitness != finalWitness {
		return nil, errors.Join(err, fmt.Errorf("formal directory %s changed native identity while enumerating", path))
	}
	return entries, nil
}

func snapshotFormalRootIdentity(root *os.Root) (_ os.FileInfo, _ string, retErr error) {
	dir, err := root.Open(".")
	if err != nil {
		return nil, "", err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	info, err := dir.Stat()
	if err != nil || !info.IsDir() {
		return nil, "", errors.Join(err, fmt.Errorf("formal root authority is not a directory"))
	}
	witness, err := formalNativeInventoryWitness(dir, info)
	if err != nil {
		return nil, "", err
	}
	return info, witness, nil
}
