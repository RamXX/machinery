package designlock

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type snapshotBudget struct {
	entries    int
	bytes      int64
	maxEntries int
	maxBytes   int64
	maxDepth   int
}

func (budget *snapshotBudget) enterDirectory(label string, depth int) error {
	if budget == nil || budget.maxDepth < 0 || depth < 0 || depth > budget.maxDepth {
		limit := -1
		if budget != nil {
			limit = budget.maxDepth
		}
		return fmt.Errorf("%s exceeds %d-level portable snapshot depth limit", label, limit)
	}
	return nil
}

func readSnapshotRegularRoot(root *os.Root, name, label string, maxBytes int64) ([]byte, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular non-symlink file", label)
	}
	if before.Size() < 0 || before.Size() > maxBytes {
		return nil, fmt.Errorf("%s has invalid size %d (maximum %d)", label, before.Size(), maxBytes)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	opened, statErr := file.Stat()
	if statErr != nil || !sameFingerprintFile(before, opened) {
		_ = file.Close()
		return nil, errors.Join(fmt.Errorf("%s changed identity while opening", label), statErr)
	}
	body, readErr := io.ReadAll(io.LimitReader(file, opened.Size()+1))
	openedAfter, retainedStatErr := file.Stat()
	closeErr := file.Close()
	after, pathStatErr := root.Lstat(name)
	if err := errors.Join(readErr, retainedStatErr, closeErr, pathStatErr); err != nil {
		return nil, err
	}
	if int64(len(body)) != opened.Size() || !sameFingerprintFile(before, openedAfter) || !sameFingerprintFile(before, after) {
		return nil, fmt.Errorf("%s changed while reading exact witnessed bytes", label)
	}
	return body, nil
}

func readSnapshotRegularPath(path, label string, maxBytes int64) ([]byte, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	body, readErr := readSnapshotRegularRoot(root, filepath.Base(path), label, maxBytes)
	return body, errors.Join(readErr, root.Close())
}

func (budget *snapshotBudget) addEntries(label string, count int) error {
	if budget == nil || budget.maxEntries <= 0 || count < 0 || budget.entries > budget.maxEntries-count {
		limit := 0
		if budget != nil {
			limit = budget.maxEntries
		}
		return fmt.Errorf("%s exceeds %d-entry snapshot inventory limit", label, limit)
	}
	budget.entries += count
	return nil
}

func (budget *snapshotBudget) addFile(label string, info fs.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() {
		return nil
	}
	size := info.Size()
	if size < 0 || size > snapshotRegularFileMaxBytes {
		return fmt.Errorf("snapshot file %s has invalid size %d (maximum %d)", label, size, snapshotRegularFileMaxBytes)
	}
	if budget == nil || budget.maxBytes <= 0 || budget.bytes > budget.maxBytes-size {
		limit := int64(0)
		if budget != nil {
			limit = budget.maxBytes
		}
		return fmt.Errorf("snapshot inventory exceeds %d-byte aggregate limit at %s", limit, label)
	}
	budget.bytes += size
	return nil
}

func readSnapshotDir(dir *os.File, label string, budget *snapshotBudget) ([]os.DirEntry, error) {
	var entries []os.DirEntry
	for {
		page, err := dir.ReadDir(snapshotInventoryPageEntries)
		if len(page) > 0 {
			if budgetErr := budget.addEntries(label, len(page)); budgetErr != nil {
				return nil, budgetErr
			}
			entries = append(entries, page...)
		}
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return nil, fmt.Errorf("%s inventory made no enumeration progress", label)
		}
	}
}
