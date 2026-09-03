package gates

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/portablepath"
)

const designArtifactMaxBytes int64 = 16 << 20
const designInventoryMaxEntries = 100_000
const designInventoryMaxDepth = 64
const designInventoryReadPage = 256

var rootReadAfterInitial = func(string) {}
var rootInventoryAfterFirst = func(string) {}

type rootDirectoryWitness struct {
	info     os.FileInfo
	changeID string
}

func probeRegularFile(design, rel string) (bool, error) {
	return inspectDesignPath(design, rel, false)
}

func probeRealDir(design, rel string) (bool, error) {
	return inspectDesignPath(design, rel, true)
}

// inspectDesignPath classifies one authored path without following symlinks.
// Every existing path component must be a real directory and the leaf must
// have the requested kind. This keeps gate reads inside the design tree even
// when a checkout contains a symlinked parent directory.
func inspectDesignPath(design, rel string, wantDir bool) (bool, error) {
	root, rootPath, err := openRealRoot(design)
	if err != nil {
		return false, err
	}
	defer root.Close()
	_ = rootPath
	return inspectRootPath(root, rel, wantDir)
}

func inspectRootPath(root *os.Root, rel string, wantDir bool) (bool, error) {
	clean := filepath.Clean(rel)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false, fmt.Errorf("unsafe design-relative path %q", rel)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	for i := range parts {
		prefix := filepath.Join(parts[:i+1]...)
		info, statErr := root.Lstat(prefix)
		if os.IsNotExist(statErr) {
			return false, nil
		}
		if statErr != nil {
			return false, fmt.Errorf("inspect %s: %w", prefix, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("%s must stay inside the design as a real %s; symlinks are rejected", filepath.ToSlash(rel), pathKind(wantDir))
		}
		if i < len(parts)-1 {
			if !info.IsDir() {
				return false, fmt.Errorf("%s must be a real directory", filepath.ToSlash(strings.Join(parts[:i+1], string(filepath.Separator))))
			}
			continue
		}
		if wantDir && !info.IsDir() {
			return false, fmt.Errorf("%s must be a real directory", filepath.ToSlash(rel))
		}
		if !wantDir && info.IsDir() {
			return false, fmt.Errorf("%s is a directory; expected a regular file", filepath.ToSlash(rel))
		}
		if !wantDir && !info.Mode().IsRegular() {
			return false, fmt.Errorf("%s must be a regular file, not a special file", filepath.ToSlash(rel))
		}
	}
	return true, nil
}

func pathKind(dir bool) string {
	if dir {
		return "directory"
	}
	return "file"
}

func readDesignFile(design, path string) ([]byte, error) {
	root, rootPath, err := openRealRoot(design)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(rootPath, absPath)
	if err != nil {
		return nil, err
	}
	ok, err := inspectRootPath(root, rel, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, os.ErrNotExist
	}
	return readRootRegularFile(root, rel)
}

func readRegularFile(path string) ([]byte, error) {
	root, _, err := openRealRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	base := filepath.Base(path)
	ok, err := inspectRootPath(root, base, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, os.ErrNotExist
	}
	return readRootRegularFile(root, base)
}

func openRealRoot(dir string) (*os.Root, string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, "", err
	}
	before, err := os.Lstat(abs)
	if err != nil {
		return nil, "", fmt.Errorf("inspect design root %s: %w", abs, err)
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, "", fmt.Errorf("design root %s must be a real directory, not a symlink or special file", abs)
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, "", err
	}
	after, err := root.Stat(".")
	if err != nil {
		root.Close()
		return nil, "", err
	}
	if !os.SameFile(before, after) {
		root.Close()
		return nil, "", fmt.Errorf("design root %s changed while it was being opened", abs)
	}
	return root, abs, nil
}

func readRootRegularFile(root *os.Root, rel string) (_ []byte, retErr error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a special file", rel)
	}
	if info.Size() < 0 || info.Size() > designArtifactMaxBytes {
		return nil, fmt.Errorf("%s size %d exceeds %d-byte limit", rel, info.Size(), designArtifactMaxBytes)
	}
	body, err := io.ReadAll(io.LimitReader(f, info.Size()+1))
	if err != nil {
		return nil, err
	}
	rootReadAfterInitial(rel)
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	again, err := io.ReadAll(io.LimitReader(f, info.Size()+1))
	if err != nil {
		return nil, err
	}
	after, err := f.Stat()
	if err != nil {
		return nil, err
	}
	pathAfter, pathErr := root.Lstat(rel)
	if pathErr != nil || !os.SameFile(info, after) || !os.SameFile(info, pathAfter) ||
		after.Size() != info.Size() || after.Mode() != info.Mode() || !after.ModTime().Equal(info.ModTime()) ||
		int64(len(body)) != info.Size() || int64(len(again)) != info.Size() || !bytes.Equal(body, again) {
		return nil, errors.Join(pathErr, fmt.Errorf("%s changed while reading", rel))
	}
	return body, nil
}

// validateDesignInventory is the universal read-side boundary for a suite
// run. Authored designs are portable regular files in real directories;
// symlinks and special entries are never valid evidence, even when a narrow
// explicit gate would otherwise happen not to read them.
func validateDesignInventory(design string) error {
	return validateDesignInventoryBounded(design, designInventoryMaxEntries, designInventoryMaxDepth)
}

func validateDesignInventoryBounded(design string, maxEntries, maxDepth int) error {
	if maxEntries <= 0 {
		return fmt.Errorf("design inventory entry limit must be positive")
	}
	if maxDepth < 0 {
		return fmt.Errorf("design inventory depth limit must be non-negative")
	}
	root, rootPath, err := openRealRoot(design)
	if err != nil {
		return err
	}
	defer root.Close()
	caseFolded := map[string]string{}
	entriesSeen := 0
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		entries, witness, err := readRootDirectory(root, dir, maxEntries-entriesSeen)
		if err != nil {
			return err
		}
		// Reserve the complete enumerated page before descending. Sibling
		// entries are already resident and part of this traversal's authority;
		// charging them one-by-one would let an early subtree consume their
		// budget and then accept the already-enumerated siblings for free.
		entriesSeen += len(entries)
		for _, entry := range entries {
			if depth >= maxDepth {
				rel := filepath.Join(dir, entry.Name())
				return fmt.Errorf("design inventory exceeds %d-level depth limit at %s", maxDepth, filepath.ToSlash(rel))
			}
			rel := entry.Name()
			if dir != "." {
				rel = filepath.Join(dir, rel)
			}
			entryInfo, err := root.Lstat(rel)
			if err != nil {
				return err
			}
			if ignoredHere(rootPath, rel) {
				continue
			}
			portableRel := filepath.ToSlash(rel)
			if err := validatePortableInventoryPath(caseFolded, portableRel); err != nil {
				return err
			}
			if entryInfo.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s: design artifacts must not be symlinks", portableRel)
			}
			if entryInfo.IsDir() {
				if err := walk(rel, depth+1); err != nil {
					return err
				}
				continue
			}
			if !entryInfo.Mode().IsRegular() {
				return fmt.Errorf("%s: design artifacts must be regular files", portableRel)
			}
		}
		return revalidateRootDirectory(root, dir, witness)
	}
	return walk(".", 0)
}

func readRootDirectory(root *os.Root, rel string, maxEntries int) (_ []fs.DirEntry, witness rootDirectoryWitness, retErr error) {
	if maxEntries < 0 {
		return nil, rootDirectoryWitness{}, fmt.Errorf("design inventory exceeds %d-entry limit", designInventoryMaxEntries)
	}
	before, err := root.Lstat(rel)
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, rootDirectoryWitness{}, fmt.Errorf("%s must be a real directory", filepath.ToSlash(rel))
	}
	dir, err := root.Open(rel)
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	opened, err := dir.Stat()
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	if !sameInventoryInfo(before, opened) {
		return nil, rootDirectoryWitness{}, fmt.Errorf("design inventory directory %s changed identity while opening", filepath.ToSlash(rel))
	}
	changeID, err := dirscan.ChangeID(dir, opened)
	if err != nil || changeID == "" {
		return nil, rootDirectoryWitness{}, errors.Join(err, fmt.Errorf("design inventory directory %s has no native change witness", filepath.ToSlash(rel)))
	}
	witness = rootDirectoryWitness{info: opened, changeID: changeID}
	entries, err := readRootDirEntries(dir, maxEntries)
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	rootInventoryAfterFirst(rel)
	firstAfter, err := dir.Stat()
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	firstChange, err := dirscan.ChangeID(dir, firstAfter)
	if err != nil || !sameInventoryInfo(opened, firstAfter) || firstChange != changeID {
		return nil, rootDirectoryWitness{}, errors.Join(err, fmt.Errorf("design inventory directory %s changed while enumerating", filepath.ToSlash(rel)))
	}
	verifyDir, err := root.Open(rel)
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	defer func() { retErr = errors.Join(retErr, verifyDir.Close()) }()
	verifyBefore, err := verifyDir.Stat()
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	verifyChange, err := dirscan.ChangeID(verifyDir, verifyBefore)
	if err != nil || !sameInventoryInfo(opened, verifyBefore) || verifyChange != changeID {
		return nil, rootDirectoryWitness{}, errors.Join(err, fmt.Errorf("design inventory directory %s changed before verification pass", filepath.ToSlash(rel)))
	}
	verifyEntries, err := readRootDirEntries(verifyDir, maxEntries)
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	sort.Slice(verifyEntries, func(i, j int) bool { return verifyEntries[i].Name() < verifyEntries[j].Name() })
	verifyAfter, err := verifyDir.Stat()
	if err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	verifyAfterChange, err := dirscan.ChangeID(verifyDir, verifyAfter)
	if err != nil || !sameInventoryInfo(opened, verifyAfter) || verifyAfterChange != changeID || !sameDirEntryNames(entries, verifyEntries) {
		return nil, rootDirectoryWitness{}, errors.Join(err, fmt.Errorf("design inventory directory %s changed between inventory passes", filepath.ToSlash(rel)))
	}
	if err := revalidateRootDirectory(root, rel, witness); err != nil {
		return nil, rootDirectoryWitness{}, err
	}
	return entries, witness, nil
}

func readRootDirEntries(dir *os.File, maxEntries int) ([]fs.DirEntry, error) {
	entries := make([]fs.DirEntry, 0, min(maxEntries, designInventoryReadPage))
	for {
		remaining := maxEntries - len(entries)
		count := min(designInventoryReadPage, remaining)
		if count == 0 {
			count = 1
		}
		page, readErr := dir.ReadDir(count)
		entries = append(entries, page...)
		if len(entries) > maxEntries {
			return nil, fmt.Errorf("design inventory exceeds %d-entry limit", designInventoryMaxEntries)
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

func revalidateRootDirectory(root *os.Root, rel string, before rootDirectoryWitness) (retErr error) {
	after, err := root.Lstat(rel)
	if err != nil || !sameInventoryInfo(before.info, after) {
		return errors.Join(err, fmt.Errorf("design inventory directory %s changed while enumerating", filepath.ToSlash(rel)))
	}
	dir, err := root.Open(rel)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, dir.Close()) }()
	opened, err := dir.Stat()
	if err != nil {
		return err
	}
	changeID, err := dirscan.ChangeID(dir, opened)
	if err != nil || !sameInventoryInfo(before.info, opened) || changeID != before.changeID {
		return errors.Join(err, fmt.Errorf("design inventory directory %s changed while walking", filepath.ToSlash(rel)))
	}
	return nil
}

func sameDirEntryNames(left, right []fs.DirEntry) bool {
	return slices.EqualFunc(left, right, func(a, b fs.DirEntry) bool { return a.Name() == b.Name() })
}

func sameInventoryInfo(before, after os.FileInfo) bool {
	return before != nil && after != nil && os.SameFile(before, after) && before.Mode() == after.Mode() && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

// walkTreeDirBounded is the common exact-inventory walker for public gate
// helpers. It holds one root handle, paginates every directory, enforces one
// aggregate entry ceiling, a fixed portable depth ceiling, and revalidates
// each directory after its subtree.
func walkTreeDirBounded(base string, maxEntries, maxDepth int, fn fs.WalkDirFunc) (retErr error) {
	if maxEntries <= 0 {
		return fmt.Errorf("tree entry limit must be positive")
	}
	if maxDepth < 0 {
		return fmt.Errorf("tree depth limit must be non-negative")
	}
	root, rootPath, err := openRealRoot(base)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, root.Close()) }()
	rootInfo, err := root.Lstat(".")
	if err != nil {
		return err
	}
	if err := fn(rootPath, fs.FileInfoToDirEntry(rootInfo), nil); err != nil {
		if errors.Is(err, fs.SkipDir) || errors.Is(err, fs.SkipAll) {
			return nil
		}
		return err
	}
	entriesSeen := 0
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		entries, witness, err := readRootDirectory(root, dir, maxEntries-entriesSeen)
		if err != nil {
			return err
		}
		entriesSeen += len(entries)
		for _, listed := range entries {
			rel := listed.Name()
			if dir != "." {
				rel = filepath.Join(dir, rel)
			}
			if depth >= maxDepth {
				return fmt.Errorf("tree %s exceeds %d-level depth limit at %s", base, maxDepth, filepath.Join(rootPath, rel))
			}
			info, err := root.Lstat(rel)
			if err != nil {
				return fn(filepath.Join(rootPath, rel), listed, err)
			}
			entry := fs.FileInfoToDirEntry(info)
			walkErr := fn(filepath.Join(rootPath, rel), entry, nil)
			if errors.Is(walkErr, fs.SkipAll) {
				return fs.SkipAll
			}
			if walkErr != nil && !errors.Is(walkErr, fs.SkipDir) {
				return walkErr
			}
			if info.IsDir() && walkErr == nil {
				if err := walk(rel, depth+1); err != nil {
					return err
				}
			}
		}
		return revalidateRootDirectory(root, dir, witness)
	}
	err = walk(".", 0)
	if errors.Is(err, fs.SkipAll) {
		return nil
	}
	return err
}

func walkTreeBounded(base string, fn filepath.WalkFunc) error {
	return walkTreeDirBounded(base, designInventoryMaxEntries, designInventoryMaxDepth, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fn(path, nil, walkErr)
		}
		info, err := entry.Info()
		if err != nil {
			return fn(path, nil, err)
		}
		return fn(path, info, nil)
	})
}

func validatePortableInventoryPath(caseFolded map[string]string, rel string) error {
	if err := portablepath.ValidateRelative(rel); err != nil {
		return fmt.Errorf("%s: non-portable design path: %w", rel, err)
	}
	fold := strings.ToLower(rel)
	if prior, exists := caseFolded[fold]; exists && prior != rel {
		return fmt.Errorf("portable design-path collision: %q and %q alias on case-insensitive filesystems", prior, rel)
	}
	caseFolded[fold] = rel
	return nil
}
