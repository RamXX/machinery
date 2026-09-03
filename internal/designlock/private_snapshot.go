package designlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/RamXX/machinery/internal/fsatomic"
)

// privateSnapshotCleanup retains both the temporary directory itself and its
// parent namespace. Cleanup first moves the exact directory into a private
// quarantine, then removes it through that opened authority. A replacement at
// the public temporary name is therefore never traversed or deleted.
type privateSnapshotCleanup struct {
	path       string
	name       string
	parent     *os.Root
	object     *os.Root
	info       os.FileInfo
	quarantine *fsatomic.Quarantined
	closed     bool
}

var privateSnapshotAfterQuarantine = func(string) {}

func newPrivateSnapshot(prefix string) (*privateSnapshotCleanup, error) {
	temp, err := os.MkdirTemp("", prefix)
	if err != nil {
		return nil, err
	}
	parentPath, name := filepath.Dir(temp), filepath.Base(temp)
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		// The fresh directory has not escaped this function, but without a
		// retained parent authority path cleanup would be unsafe. Preserve it.
		return nil, fmt.Errorf("open private snapshot parent; preserving %s: %w", temp, err)
	}
	info, err := parent.Lstat(name)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(fmt.Errorf("inspect private snapshot directory; preserving %s", temp), err, parent.Close())
	}
	object, err := parent.OpenRoot(name)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("retain private snapshot directory; preserving %s: %w", temp, err), parent.Close())
	}
	inside, err := object.Lstat(".")
	if err != nil || !os.SameFile(info, inside) {
		return nil, errors.Join(fmt.Errorf("private snapshot changed while retaining its authority; preserving %s", temp), err, object.Close(), parent.Close())
	}
	return &privateSnapshotCleanup{path: temp, name: name, parent: parent, object: object, info: inside}, nil
}

func (c *privateSnapshotCleanup) Path() string {
	if c == nil {
		return ""
	}
	return c.path
}

func (c *privateSnapshotCleanup) Close() error {
	if c == nil || c.closed {
		return nil
	}
	if c.quarantine == nil {
		quarantine, err := fsatomic.Quarantine(c.parent, c.name, c.name+".retired-")
		if err != nil {
			return fmt.Errorf("isolate private snapshot %s: %w", c.path, err)
		}
		held, heldErr := c.object.Lstat(".")
		isolated, isolatedErr := quarantine.Root().Lstat(quarantine.Name())
		if err := errors.Join(heldErr, isolatedErr); err != nil || !held.IsDir() || !isolated.IsDir() ||
			!os.SameFile(c.info, held) || !os.SameFile(held, isolated) {
			restoreErr := quarantine.Restore()
			return errors.Join(fmt.Errorf("private snapshot name %s was replaced; preserving the isolated object", c.path), err, restoreErr, quarantine.Close())
		}
		c.quarantine = quarantine
		privateSnapshotAfterQuarantine(c.path)
	}
	if c.object != nil {
		err := c.object.Close()
		c.object = nil
		if err != nil {
			return fmt.Errorf("close retained private snapshot authority: %w", err)
		}
	}
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
	if err := emptyPrivateSnapshotTree(c.quarantine.Root(), c.quarantine.Name(), &budget, 0, false); err != nil {
		return fmt.Errorf("validate and empty isolated private snapshot %s: %w", c.path, err)
	}
	if err := c.quarantine.Remove(); err != nil {
		return fmt.Errorf("remove isolated private snapshot %s: %w", c.path, err)
	}
	c.quarantine = nil
	err := c.parent.Close()
	c.parent = nil
	c.closed = true
	if err != nil {
		return fmt.Errorf("close private snapshot parent authority: %w", err)
	}
	return nil
}

func emptyPrivateSnapshotTree(root *os.Root, name string, budget *snapshotBudget, depth int, removeSelf bool) error {
	label := "private snapshot cleanup directory " + filepath.ToSlash(name)
	if err := budget.enterDirectory(label, depth); err != nil {
		return err
	}
	before, err := root.Lstat(name)
	if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return errors.Join(err, fmt.Errorf("%s must be a real directory", label))
	}
	directory, err := root.Open(name)
	if err != nil {
		return err
	}
	opened, statErr := directory.Stat()
	if statErr != nil || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return errors.Join(statErr, fmt.Errorf("%s changed while opening", label), directory.Close())
	}
	entries, readErr := readSnapshotDir(directory, label, budget)
	closeErr := directory.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		rel := filepath.Join(name, entry.Name())
		info, err := root.Lstat(rel)
		if err != nil {
			return err
		}
		if err := budget.addFile(filepath.ToSlash(rel), info); err != nil {
			return err
		}
		switch {
		case info.IsDir() && info.Mode()&os.ModeSymlink == 0:
			if err := emptyPrivateSnapshotTree(root, rel, budget, depth+1, true); err != nil {
				return err
			}
		case info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0:
			current, err := root.Lstat(rel)
			if err != nil || !sameFingerprintFile(info, current) {
				return errors.Join(err, fmt.Errorf("private snapshot cleanup entry %s changed before removal", filepath.ToSlash(rel)))
			}
			if err := root.Remove(rel); err != nil {
				return err
			}
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("private snapshot cleanup entry %s is a symlink; preserving authority", filepath.ToSlash(rel))
		default:
			return fmt.Errorf("private snapshot cleanup entry %s is a special file (%s); preserving authority", filepath.ToSlash(rel), info.Mode())
		}
	}
	after, err := root.Lstat(name)
	if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
		return errors.Join(err, fmt.Errorf("%s changed while emptying", label))
	}
	if removeSelf {
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	return nil
}
