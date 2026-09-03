package cachestage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PublishTree durably installs one already-built private tree under base. Every
// file and directory in source is flushed before the rooted rename, and both
// rename parents are flushed afterwards. Callers must hold the cache lock.
func PublishTree(base, source, target string) error {
	return publish(base, source, target, publishHooks{})
}

type publishHooks struct {
	afterTreeSync func() error
	beforeRename  func() error
	afterRename   func() error
}

func publish(base, source, target string, hooks publishHooks) (retErr error) {
	source, err := safeRelativePath(source)
	if err != nil {
		return fmt.Errorf("unsafe cache publication source: %w", err)
	}
	target, err = safeRelativePath(target)
	if err != nil {
		return fmt.Errorf("unsafe cache publication target: %w", err)
	}
	if strings.EqualFold(source, target) || pathContains(source, target) || pathContains(target, source) {
		return fmt.Errorf("cache publication source %s and target %s must be disjoint", source, target)
	}
	before, err := os.Lstat(base)
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return fmt.Errorf("cache publication root %s must be a real directory", base)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return errors.Join(err, fmt.Errorf("cache publication root changed identity while opening"))
	}
	sourceInfo, err := root.Lstat(source)
	if err != nil {
		return fmt.Errorf("inspect cache publication source %s: %w", source, err)
	}
	if sourceInfo.Mode()&os.ModeSymlink != 0 || !sourceInfo.IsDir() || !privateStageMode(sourceInfo.Mode()) {
		return fmt.Errorf("cache publication source %s must be a private real directory", source)
	}
	if err := validateTree(root, source); err != nil {
		return fmt.Errorf("cache publication source %s is unsafe: %w", source, err)
	}
	targetParent := filepath.Dir(target)
	if err := ensurePrivateDir(root, targetParent); err != nil {
		return err
	}
	if info, err := root.Lstat(target); err == nil {
		return fmt.Errorf("cache publication target %s already exists (%s)", target, info.Mode())
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect cache publication target %s: %w", target, err)
	}
	witness, err := syncTreeWitness(root, source)
	if err != nil {
		return fmt.Errorf("sync cache publication tree %s: %w", source, err)
	}
	if hooks.afterTreeSync != nil {
		if err := hooks.afterTreeSync(); err != nil {
			return err
		}
	}
	current, err := syncTreeWitness(root, source)
	if err != nil {
		return fmt.Errorf("revalidate synced cache publication tree %s: %w", source, err)
	}
	if err := compareTreeWitness(witness, current); err != nil {
		return fmt.Errorf("cache publication source %s changed after durable sync: %w", source, err)
	}
	if info, err := root.Lstat(target); err == nil {
		return fmt.Errorf("cache publication target %s appeared before publish (%s)", target, info.Mode())
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reinspect cache publication target %s: %w", target, err)
	}
	if hooks.beforeRename != nil {
		if err := hooks.beforeRename(); err != nil {
			return err
		}
	}
	if info, err := root.Lstat(target); err == nil {
		return fmt.Errorf("cache publication target %s appeared at the publish boundary (%s)", target, info.Mode())
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("reinspect cache publication target %s at the publish boundary: %w", target, err)
	}
	if err := root.Rename(source, target); err != nil {
		return fmt.Errorf("publish cache tree %s: %w", target, err)
	}
	if hooks.afterRename != nil {
		if err := hooks.afterRename(); err != nil {
			return err
		}
	}
	installed, err := syncTreeWitness(root, target)
	if err != nil {
		return fmt.Errorf("verify installed cache tree %s: %w; preserving target for explicit recovery", target, err)
	}
	if err := compareTreeWitness(rebaseTreeWitness(witness, source, target), installed); err != nil {
		return fmt.Errorf("installed cache tree %s does not match the exact synced source: %w; preserving target for explicit recovery", target, err)
	}
	parents := []string{filepath.Dir(source), targetParent}
	sort.Strings(parents)
	prior := ""
	for _, parent := range parents {
		if parent == prior {
			continue
		}
		if err := syncStageDirectory(root, parent); err != nil {
			return fmt.Errorf("sync cache publication parent %s: %w", filepath.ToSlash(parent), err)
		}
		prior = parent
	}
	return nil
}

func rebaseTreeWitness(witness treeWitness, from, to string) treeWitness {
	rebased := treeWitness{entries: make([]treeWitnessEntry, len(witness.entries))}
	for i, entry := range witness.entries {
		rel, err := filepath.Rel(from, entry.path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			rebased.entries[i] = entry
			continue
		}
		entry.path = filepath.Join(to, rel)
		rebased.entries[i] = entry
	}
	return rebased
}

func safeRelativePath(path string) (string, error) {
	clean := filepath.Clean(path)
	if path == "" || filepath.IsAbs(path) || filepath.VolumeName(path) != "" || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q is not a confined relative path", path)
	}
	return clean, nil
}

func pathContains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func ensurePrivateDir(root *os.Root, rel string) error {
	if rel == "." {
		return nil
	}
	clean, err := safeRelativePath(rel)
	if err != nil {
		return err
	}
	current := "."
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		next := component
		if current != "." {
			next = filepath.Join(current, component)
		}
		info, err := root.Lstat(next)
		if os.IsNotExist(err) {
			if err := root.Mkdir(next, 0o700); err != nil {
				return fmt.Errorf("create cache publication directory %s: %w", filepath.ToSlash(next), err)
			}
			if err := syncStageDirectory(root, current); err != nil {
				return fmt.Errorf("sync new cache publication directory %s: %w", filepath.ToSlash(next), err)
			}
			info, err = root.Lstat(next)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !privateStageMode(info.Mode()) {
			return fmt.Errorf("cache publication directory %s must be a private real directory", filepath.ToSlash(next))
		}
		current = next
	}
	return nil
}

type treeWitness struct {
	entries []treeWitnessEntry
}

type treeWitnessEntry struct {
	path  string
	mode  os.FileMode
	info  os.FileInfo
	hash  string
	isDir bool
}

func syncTreeWitness(root *os.Root, dir string) (treeWitness, error) {
	var witness treeWitness
	if err := syncTreeWitnessDir(root, dir, &witness); err != nil {
		return treeWitness{}, err
	}
	return witness, nil
}

func syncTreeWitnessDir(root *os.Root, dir string, witness *treeWitness) error {
	dirBefore, err := root.Lstat(dir)
	if err != nil {
		return err
	}
	if dirBefore.Mode()&os.ModeSymlink != 0 || !dirBefore.IsDir() || !privateStageMode(dirBefore.Mode()) {
		return fmt.Errorf("%s must be a private real directory", filepath.ToSlash(dir))
	}
	dirIndex := len(witness.entries)
	witness.entries = append(witness.entries, treeWitnessEntry{})
	entries, err := readDir(root, dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := filepath.Join(dir, entry.Name())
		info, err := root.Lstat(child)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s is a symlink", filepath.ToSlash(child))
		case info.IsDir():
			if err := syncTreeWitnessDir(root, child, witness); err != nil {
				return err
			}
		case info.Mode().IsRegular():
			fileWitness, err := syncFileWitness(root, child, info)
			if err != nil {
				return err
			}
			witness.entries = append(witness.entries, fileWitness)
		default:
			return fmt.Errorf("%s is a special entry (%s)", filepath.ToSlash(child), info.Mode())
		}
	}
	afterEntries, err := readDir(root, dir)
	if err != nil {
		return err
	}
	if !sameEntryNames(entries, afterEntries) {
		return fmt.Errorf("cache publication directory %s changed inventory while syncing", filepath.ToSlash(dir))
	}
	if err := syncStageDirectory(root, dir); err != nil {
		return err
	}
	dirAfter, err := root.Lstat(dir)
	if err != nil || !os.SameFile(dirBefore, dirAfter) || dirBefore.Mode() != dirAfter.Mode() || dirBefore.Size() != dirAfter.Size() || !dirBefore.ModTime().Equal(dirAfter.ModTime()) {
		return errors.Join(err, fmt.Errorf("cache publication directory %s changed identity or metadata while syncing", filepath.ToSlash(dir)))
	}
	witness.entries[dirIndex] = treeWitnessEntry{path: dir, mode: dirAfter.Mode(), info: dirAfter, isDir: true}
	return nil
}

func syncFileWitness(root *os.Root, path string, before os.FileInfo) (entry treeWitnessEntry, retErr error) {
	if !privateStageMode(before.Mode()) {
		return entry, fmt.Errorf("%s is not private", filepath.ToSlash(path))
	}
	file, err := root.Open(path)
	if err != nil {
		return entry, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return entry, errors.Join(err, fmt.Errorf("cache publication file %s changed identity or mode while opening", filepath.ToSlash(path)))
	}
	digest, err := hashOpenFile(file)
	if err != nil {
		return entry, err
	}
	if err := file.Sync(); err != nil {
		return entry, err
	}
	openedAfter, err := file.Stat()
	if err != nil || !os.SameFile(opened, openedAfter) || opened.Mode() != openedAfter.Mode() || opened.Size() != openedAfter.Size() || !opened.ModTime().Equal(openedAfter.ModTime()) {
		return entry, errors.Join(err, fmt.Errorf("cache publication file %s changed metadata while syncing", filepath.ToSlash(path)))
	}
	pathAfter, err := root.Lstat(path)
	if err != nil || !os.SameFile(openedAfter, pathAfter) || openedAfter.Mode() != pathAfter.Mode() {
		return entry, errors.Join(err, fmt.Errorf("cache publication file %s changed identity after syncing", filepath.ToSlash(path)))
	}
	return treeWitnessEntry{path: path, mode: pathAfter.Mode(), info: pathAfter, hash: digest}, nil
}

func hashOpenFile(file *os.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func sameEntryNames(left, right []os.DirEntry) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Name() != right[i].Name() {
			return false
		}
	}
	return true
}

func compareTreeWitness(expected, current treeWitness) error {
	if len(expected.entries) != len(current.entries) {
		return fmt.Errorf("inventory count is %d, expected %d", len(current.entries), len(expected.entries))
	}
	for i, want := range expected.entries {
		got := current.entries[i]
		if want.path != got.path || want.isDir != got.isDir {
			return fmt.Errorf("inventory entry %d is %s, expected %s", i, filepath.ToSlash(got.path), filepath.ToSlash(want.path))
		}
		if want.info == nil || got.info == nil || !os.SameFile(want.info, got.info) || want.mode != got.mode || want.info.Size() != got.info.Size() || !want.info.ModTime().Equal(got.info.ModTime()) {
			return fmt.Errorf("%s changed identity or metadata", filepath.ToSlash(want.path))
		}
		if !want.isDir && want.hash != got.hash {
			return fmt.Errorf("%s changed content", filepath.ToSlash(want.path))
		}
	}
	return nil
}
