package install

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/portablepath"
)

// resolvedSource is the only source capability an install renderer receives.
// path is a private immutable materialization; the mutable checkout is retained
// solely so unchanged-through-commit can be verified before publication.
type resolvedSource struct {
	path    string
	verify  func() error
	cleanup func() error
}

type sourceSnapshotEntry struct {
	rel      string
	info     fs.FileInfo
	digest   [sha256.Size]byte
	dir      bool
	changeID string
}

type sourceSnapshot struct {
	original      string
	originalInfo  fs.FileInfo
	root          *os.Root
	materialized  string
	removeScratch func() error
	entries       map[string]sourceSnapshotEntry
	treeRoots     []string
	explicit      []string
}

// sourceSnapshotAfterOpen is a deterministic adversarial test hook. Production
// leaves it inert; tests use it to interleave pathname replacement after the
// retained file capability is open but before bytes are copied.
var (
	sourceSnapshotAfterOpen            = func(string) {}
	sourceSnapshotAfterVerifyDirectory = func(string) {}
)

func acquireInstallSourceSnapshot(src string, targets []string) (*sourceSnapshot, error) {
	abs, err := filepath.Abs(src)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("inspect install source root %s: %w", abs, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("install source root %s must be a real directory, not a symlink or special file", abs)
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("retain install source root %s: %w", abs, err)
	}
	opened, err := root.Open(".")
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open retained install source root: %w", err), root.Close())
	}
	openedInfo, statErr := opened.Stat()
	closeErr := closeInstallFile(opened)
	if statErr != nil || closeErr != nil || !os.SameFile(info, openedInfo) {
		return nil, errors.Join(fmt.Errorf("install source root changed while being retained"), statErr, closeErr, root.Close())
	}
	materialized, removeScratch, err := installScratchDir("source-snapshot")
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	snapshot := &sourceSnapshot{
		original: abs, originalInfo: openedInfo, root: root,
		materialized: materialized, removeScratch: removeScratch, entries: map[string]sourceSnapshotEntry{},
		treeRoots: []string{filepath.FromSlash(skillRel)},
	}
	for _, role := range RoleDocs {
		snapshot.explicit = append(snapshot.explicit, filepath.Join(agentsRel, role))
	}
	set, parseErr := parseTargetsOptional(targets)
	if parseErr != nil {
		return nil, errors.Join(parseErr, snapshot.cleanup())
	}
	if set[TargetOpenCode] {
		for _, command := range openCodeCommands {
			snapshot.explicit = append(snapshot.explicit, filepath.Join("adapters", "opencode", "commands", command))
		}
		snapshot.explicit = append(snapshot.explicit, filepath.Join("adapters", "opencode", "plugins", "machinery.js"))
	}
	for _, rel := range snapshot.treeRoots {
		if err := snapshot.copyEntry(rel, true); err != nil {
			return nil, errors.Join(err, snapshot.cleanup())
		}
	}
	for _, rel := range snapshot.explicit {
		if err := snapshot.copyEntry(rel, false); err != nil {
			return nil, errors.Join(err, snapshot.cleanup())
		}
	}
	if err := snapshot.revalidateDirectoryCensus("while being snapshotted"); err != nil {
		return nil, errors.Join(err, snapshot.cleanup())
	}
	if err := syncTree(materialized); err != nil {
		return nil, errors.Join(fmt.Errorf("sync install source snapshot: %w", err), snapshot.cleanup())
	}
	return snapshot, nil
}

func parseTargetsOptional(targets []string) (map[Target]bool, error) {
	if len(targets) == 0 {
		return map[Target]bool{}, nil
	}
	return parseTargets(targets)
}

func (snapshot *sourceSnapshot) copyEntry(rel string, recursive bool) error {
	rel = filepath.Clean(rel)
	info, err := snapshot.root.Lstat(rel)
	if err != nil {
		return fmt.Errorf("inspect install source entry %s: %w", rel, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("install source entry %s is a symlink", rel)
	}
	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("install source entry %s must be a regular file", rel)
		}
		destination := filepath.Join(snapshot.materialized, rel)
		if err := os.MkdirAll(destination, info.Mode().Perm()); err != nil {
			return err
		}
		dir, err := snapshot.root.Open(rel)
		if err != nil {
			return fmt.Errorf("open install source directory %s: %w", rel, err)
		}
		entries, readErr := dir.ReadDir(-1)
		closeErr := dir.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(readErr, closeErr)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		folded := map[string]string{}
		for _, entry := range entries {
			if err := portablepath.ValidateBase(entry.Name()); err != nil {
				return fmt.Errorf("install source entry %s is not portable: %w", filepath.Join(rel, entry.Name()), err)
			}
			key := strings.ToLower(entry.Name())
			if prior, exists := folded[key]; exists {
				return fmt.Errorf("install source directory %s has case-fold collision %q and %q", rel, prior, entry.Name())
			}
			folded[key] = entry.Name()
			if err := snapshot.copyEntry(filepath.Join(rel, entry.Name()), true); err != nil {
				return err
			}
		}
		current, err := snapshot.root.Lstat(rel)
		if err != nil || !current.IsDir() || !os.SameFile(info, current) || current.Mode() != info.Mode() {
			return errors.Join(fmt.Errorf("install source directory %s changed while being snapshotted", rel), err)
		}
		snapshot.entries[rel] = sourceSnapshotEntry{rel: rel, info: current, dir: true, changeID: installFileChangeID(current)}
		if err := os.Chmod(destination, current.Mode().Perm()); err != nil {
			return err
		}
		return syncDir(destination)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("install source entry %s has unsupported type %s", rel, info.Mode().Type())
	}
	return snapshot.copyFile(rel, info)
}

func (snapshot *sourceSnapshot) copyFile(rel string, before fs.FileInfo) (retErr error) {
	in, err := snapshot.root.Open(rel)
	if err != nil {
		return fmt.Errorf("open install source file %s: %w", rel, err)
	}
	defer func() { retErr = errors.Join(retErr, closeInstallFile(in)) }()
	opened, err := in.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return errors.Join(fmt.Errorf("install source file %s changed before it was read", rel), err)
	}
	sourceSnapshotAfterOpen(rel)
	destination := filepath.Join(snapshot.materialized, rel)
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, opened.Mode().Perm())
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(out, hash), in)
	chmodErr := out.Chmod(opened.Mode().Perm())
	syncErr := out.Sync()
	closeErr := closeInstallFile(out)
	if copyErr != nil || chmodErr != nil || syncErr != nil || closeErr != nil {
		return errors.Join(copyErr, chmodErr, syncErr, closeErr)
	}
	after, statErr := in.Stat()
	pathAfter, pathErr := snapshot.root.Lstat(rel)
	if statErr != nil || pathErr != nil || !os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) ||
		after.Mode() != opened.Mode() || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return errors.Join(fmt.Errorf("install source file %s changed while being snapshotted", rel), statErr, pathErr)
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	snapshot.entries[rel] = sourceSnapshotEntry{rel: rel, info: after, digest: digest, changeID: installFileChangeID(after)}
	return nil
}

func (snapshot *sourceSnapshot) verifyUnchanged() error {
	currentRoot, err := os.Lstat(snapshot.original)
	if err != nil || currentRoot.Mode()&os.ModeSymlink != 0 || !currentRoot.IsDir() || !os.SameFile(snapshot.originalInfo, currentRoot) {
		return errors.Join(fmt.Errorf("install source root changed after the immutable snapshot was acquired"), err)
	}
	seen := map[string]bool{}
	for _, rel := range snapshot.treeRoots {
		if err := snapshot.verifyEntry(rel, true, seen); err != nil {
			return err
		}
	}
	for _, rel := range snapshot.explicit {
		if err := snapshot.verifyEntry(rel, false, seen); err != nil {
			return err
		}
	}
	if len(seen) != len(snapshot.entries) {
		return fmt.Errorf("install source inventory changed after the immutable snapshot was acquired")
	}
	if err := snapshot.revalidateDirectoryCensus("after the immutable snapshot was acquired"); err != nil {
		return err
	}
	return nil
}

func (snapshot *sourceSnapshot) verifyEntry(rel string, recursive bool, seen map[string]bool) error {
	info, err := snapshot.root.Lstat(rel)
	if err != nil {
		return fmt.Errorf("verify install source entry %s: %w", rel, err)
	}
	want, ok := snapshot.entries[rel]
	if !ok || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != want.dir || !os.SameFile(want.info, info) || info.Mode() != want.info.Mode() {
		return fmt.Errorf("install source entry %s changed after the immutable snapshot was acquired", rel)
	}
	seen[rel] = true
	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("install source entry %s changed from a file to a directory", rel)
		}
		dir, err := snapshot.root.Open(rel)
		if err != nil {
			return err
		}
		entries, readErr := dir.ReadDir(-1)
		closeErr := dir.Close()
		if readErr != nil || closeErr != nil {
			return errors.Join(readErr, closeErr)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			child := filepath.Join(rel, entry.Name())
			if _, expected := snapshot.entries[child]; !expected {
				return fmt.Errorf("install source gained unexpected entry %s after the immutable snapshot was acquired", child)
			}
			if err := snapshot.verifyEntry(child, true, seen); err != nil {
				return err
			}
		}
		sourceSnapshotAfterVerifyDirectory(rel)
		return nil
	}
	in, err := snapshot.root.Open(rel)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, readErr := io.Copy(hash, in)
	after, statErr := in.Stat()
	closeErr := closeInstallFile(in)
	if readErr != nil || statErr != nil || closeErr != nil {
		return errors.Join(readErr, statErr, closeErr)
	}
	if !os.SameFile(want.info, after) || after.Size() != want.info.Size() || !after.ModTime().Equal(want.info.ModTime()) || !equalDigest(hash.Sum(nil), want.digest[:]) ||
		(want.changeID != "" && installFileChangeID(after) != "" && want.changeID != installFileChangeID(after)) {
		return fmt.Errorf("install source file %s changed after the immutable snapshot was acquired", rel)
	}
	return nil
}

func (snapshot *sourceSnapshot) revalidateDirectoryCensus(when string) error {
	directChildren := make(map[string]map[string]sourceSnapshotEntry)
	var directories []string
	for rel, entry := range snapshot.entries {
		if entry.dir {
			directories = append(directories, rel)
		}
		parent := filepath.Dir(rel)
		if _, tracked := snapshot.entries[parent]; !tracked {
			continue
		}
		if directChildren[parent] == nil {
			directChildren[parent] = map[string]sourceSnapshotEntry{}
		}
		directChildren[parent][rel] = entry
	}
	sort.Sort(sort.Reverse(sort.StringSlice(directories)))
	for _, rel := range directories {
		want := snapshot.entries[rel]
		before, err := snapshot.root.Lstat(rel)
		if err != nil || !sameSourceSnapshotEntry(want, before) {
			return errors.Join(fmt.Errorf("install source directory %s changed %s", rel, when), err)
		}
		dir, err := snapshot.root.Open(rel)
		if err != nil {
			return fmt.Errorf("retain install source directory %s for final census: %w", rel, err)
		}
		openedInfo, statErr := dir.Stat()
		entries, readErr := dir.ReadDir(-1)
		closeErr := closeInstallFile(dir)
		if statErr != nil || readErr != nil || closeErr != nil || !sameSourceSnapshotEntry(want, openedInfo) {
			return errors.Join(fmt.Errorf("install source directory %s changed during final census", rel), statErr, readErr, closeErr)
		}
		expected := directChildren[rel]
		if len(entries) != len(expected) {
			return fmt.Errorf("install source directory %s inventory changed %s", rel, when)
		}
		for _, entry := range entries {
			child := filepath.Join(rel, entry.Name())
			wantChild, ok := expected[child]
			if !ok {
				return fmt.Errorf("install source gained unexpected entry %s %s", child, when)
			}
			info, err := snapshot.root.Lstat(child)
			if err != nil || !sameSourceSnapshotEntry(wantChild, info) {
				return errors.Join(fmt.Errorf("install source entry %s changed %s", child, when), err)
			}
		}
		after, err := snapshot.root.Lstat(rel)
		if err != nil || !sameSourceSnapshotEntry(want, after) {
			return errors.Join(fmt.Errorf("install source directory %s changed during final census", rel), err)
		}
	}
	return nil
}

func sameSourceSnapshotEntry(want sourceSnapshotEntry, got fs.FileInfo) bool {
	if got == nil || got.Mode()&os.ModeSymlink != 0 || got.IsDir() != want.dir || !os.SameFile(want.info, got) ||
		got.Mode() != want.info.Mode() || got.Size() != want.info.Size() || !got.ModTime().Equal(want.info.ModTime()) {
		return false
	}
	gotChangeID := installFileChangeID(got)
	return want.changeID == "" || gotChangeID == "" || want.changeID == gotChangeID
}

func equalDigest(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var different byte
	for i := range left {
		different |= left[i] ^ right[i]
	}
	return different == 0
}

func (snapshot *sourceSnapshot) cleanup() error {
	return errors.Join(snapshot.root.Close(), snapshot.removeScratch())
}
