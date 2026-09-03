// Package dirscan provides bounded, deterministic directory enumeration.
package dirscan

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

const batchSize = 256

// DefaultMaxDepth bounds callers of Walk that do not need a narrower
// application-specific ceiling. The root has depth zero.
const DefaultMaxDepth = 256

// WalkLimits bounds one complete tree inventory before recursion or aggregate
// entry growth can consume unbounded resources.
type WalkLimits struct {
	MaxEntries int
	MaxDepth   int
}

type directoryReader interface {
	ReadDir(int) ([]os.DirEntry, error)
}

var afterEnumeration = func(string) {}

type directoryState struct {
	info     os.FileInfo
	changeID string
}

// Read returns at most maxEntries entries from a real directory. The ceiling
// is enforced while paginating, before an unbounded slice can be allocated.
func Read(path string, maxEntries int) (_ []os.DirEntry, retErr error) {
	if maxEntries < 0 {
		return nil, fmt.Errorf("directory entry limit must be non-negative")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("%s must be a real directory", path)
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	opened, err := dir.Stat()
	if err != nil {
		return nil, err
	}
	if !os.SameFile(before, opened) {
		return nil, fmt.Errorf("directory %s changed identity while opening", path)
	}
	initial, err := captureDirectoryState(dir)
	if err != nil {
		return nil, err
	}
	entries, err := readEntries(dir, path, maxEntries)
	if err != nil {
		return nil, err
	}
	sortEntries(entries)
	afterEnumeration(path)
	afterFirst, err := captureDirectoryState(dir)
	if err != nil {
		return nil, err
	}
	if !sameDirectoryState(initial, afterFirst) {
		return nil, fmt.Errorf("directory %s changed while enumerating", path)
	}

	// A second independently opened pass is required. Metadata alone is not
	// an inventory: a same-directory rename, or a create/delete ABA within a
	// coarse mtime tick, can restore the apparent namespace before the final
	// stat. Retain the first handle as identity authority, bind the second
	// handle to it, and require both the ordered names and the native directory
	// change witness to agree.
	secondDir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, secondDir.Close()) }()
	secondInitial, err := captureDirectoryState(secondDir)
	if err != nil {
		return nil, err
	}
	if !sameDirectoryState(initial, secondInitial) {
		return nil, fmt.Errorf("directory %s changed while opening verification pass", path)
	}
	secondEntries, err := readEntries(secondDir, path, maxEntries)
	if err != nil {
		return nil, err
	}
	sortEntries(secondEntries)
	secondFinal, err := captureDirectoryState(secondDir)
	if err != nil {
		return nil, err
	}
	pathAfter, pathErr := os.Lstat(path)
	if pathErr != nil || !sameDirectoryState(initial, secondFinal) || !stableInfo(initial.info, pathAfter) {
		return nil, errors.Join(pathErr, fmt.Errorf("directory %s changed while enumerating", path))
	}
	if !sameEntryNames(entries, secondEntries) {
		return nil, fmt.Errorf("directory %s changed between inventory passes", path)
	}
	return entries, nil
}

func captureDirectoryState(dir *os.File) (directoryState, error) {
	info, err := dir.Stat()
	if err != nil {
		return directoryState{}, err
	}
	changeID, err := directoryChangeID(dir, info)
	if err != nil {
		return directoryState{}, err
	}
	if changeID == "" {
		return directoryState{}, fmt.Errorf("directory %s has no native change witness", dir.Name())
	}
	return directoryState{info: info, changeID: changeID}, nil
}

// ChangeID returns the native namespace-change witness for an opened
// directory. Callers that enumerate through an os.Root retain their own
// authority but share the same fail-closed platform witness as Read.
func ChangeID(dir *os.File, info os.FileInfo) (string, error) {
	return directoryChangeID(dir, info)
}

func sameDirectoryState(before, after directoryState) bool {
	return stableInfo(before.info, after.info) && before.changeID == after.changeID
}

func sortEntries(entries []os.DirEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
}

func sameEntryNames(left, right []os.DirEntry) bool {
	return slices.EqualFunc(left, right, func(a, b os.DirEntry) bool { return a.Name() == b.Name() })
}

func readEntries(dir directoryReader, path string, maxEntries int) ([]os.DirEntry, error) {
	entries := make([]os.DirEntry, 0, min(maxEntries, batchSize))
	for {
		remaining := maxEntries - len(entries)
		count := min(batchSize, remaining)
		if count == 0 {
			// Probe for exactly one excess entry. In particular, do not express
			// this as maxEntries+1: MaxInt is a valid ceiling and must not turn
			// into ReadDir's unbounded non-positive request through overflow.
			count = 1
		}
		batch, readErr := dir.ReadDir(count)
		if len(batch) == 0 && readErr == nil {
			return nil, fmt.Errorf("directory %s returned an empty page without EOF", path)
		}
		entries = append(entries, batch...)
		if len(entries) > maxEntries {
			return nil, fmt.Errorf("directory %s exceeds %d-entry limit", path, maxEntries)
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return entries, nil
}

// Walk visits one real tree without following symlinks. maxEntries is one
// aggregate ceiling across all descendants (the root is not counted).
func Walk(base string, maxEntries int, fn fs.WalkDirFunc) error {
	return WalkBounded(base, WalkLimits{MaxEntries: maxEntries, MaxDepth: DefaultMaxDepth}, fn)
}

// WalkBounded visits one real tree without following symlinks. Both ceilings
// apply to the complete traversal; the root is not included in MaxEntries and
// has depth zero.
func WalkBounded(base string, limits WalkLimits, fn fs.WalkDirFunc) error {
	if limits.MaxEntries < 0 {
		return fmt.Errorf("tree entry limit must be non-negative")
	}
	if limits.MaxDepth < 0 {
		return fmt.Errorf("tree depth limit must be non-negative")
	}
	rootInfo, err := os.Lstat(base)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("%s must be a real directory", base)
	}
	if err := fn(base, fs.FileInfoToDirEntry(rootInfo), nil); err != nil {
		if errors.Is(err, fs.SkipDir) || errors.Is(err, fs.SkipAll) {
			return nil
		}
		return err
	}
	seen := 0
	var walk func(string, int) error
	walk = func(dir string, depth int) (retErr error) {
		before, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		authority, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, authority.Close()) }()
		initial, err := captureDirectoryState(authority)
		if err != nil {
			return err
		}
		if !stableInfo(before, initial.info) {
			return fmt.Errorf("directory %s changed identity while opening", dir)
		}
		entries, err := Read(dir, limits.MaxEntries-seen)
		if err != nil {
			return err
		}
		for _, listed := range entries {
			if depth >= limits.MaxDepth {
				return fmt.Errorf("tree %s exceeds %d-level depth limit at %s", base, limits.MaxDepth, filepath.Join(dir, listed.Name()))
			}
			entryDepth := depth + 1
			seen++
			path := filepath.Join(dir, listed.Name())
			info, statErr := os.Lstat(path)
			if statErr != nil {
				if err := fn(path, listed, statErr); err != nil {
					return err
				}
				continue
			}
			entry := fs.FileInfoToDirEntry(info)
			walkErr := fn(path, entry, nil)
			if errors.Is(walkErr, fs.SkipAll) {
				return fs.SkipAll
			}
			if walkErr != nil && !errors.Is(walkErr, fs.SkipDir) {
				return walkErr
			}
			if info.IsDir() && walkErr == nil {
				if err := walk(path, entryDepth); err != nil {
					return err
				}
			}
		}
		heldAfter, err := captureDirectoryState(authority)
		if err != nil || !sameDirectoryState(initial, heldAfter) {
			return errors.Join(err, fmt.Errorf("directory %s changed while walking", dir))
		}
		after, err := os.Lstat(dir)
		if err != nil || !stableInfo(initial.info, after) {
			return errors.Join(err, fmt.Errorf("directory %s changed while walking", dir))
		}
		return nil
	}
	err = walk(base, 0)
	if errors.Is(err, fs.SkipAll) {
		return nil
	}
	return err
}

func stableInfo(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}
