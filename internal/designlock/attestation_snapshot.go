package designlock

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// AttestationTreeEntry describes one observed portable implementation subject.
// Identity witnesses are deliberately private and never persisted in receipts.
type AttestationTreeEntry struct {
	Path      string
	Directory bool
	Mode      uint32
	Size      int64
	SHA256    [32]byte
}

// AttestationTreeSnapshot retains the ORIGINAL rooted authorities until Close.
// Path is an immutable copy compatible with generic implementation consumers;
// Entries also includes the retained design overlay when design is in scope.
type AttestationTreeSnapshot struct {
	lock                 *Lock
	path, logical        string
	design, impl         *os.Root
	designInfo, implInfo os.FileInfo
	entries              []AttestationTreeEntry
	witnesses            map[string]os.FileInfo
	cleanup              *privateSnapshotCleanup
	closed               bool
	closeErr             error
}

var newAttestationSnapshotBudget = func() snapshotBudget {
	return snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
}

type attestationInventoryPolicy struct {
	strict    bool
	copySkip  string
	witnesses map[string]os.FileInfo
}

func (s *AttestationTreeSnapshot) Path() string    { return s.path }
func (s *AttestationTreeSnapshot) Logical() string { return s.logical }

// Entries returns a defensive copy, not mutable capture authority.
func (s *AttestationTreeSnapshot) Entries() []AttestationTreeEntry {
	if s == nil || s.closed {
		return nil
	}
	return append([]AttestationTreeEntry(nil), s.entries...)
}

func attestationWithin(parent, child string) (string, bool) {
	rel, err := filepath.Rel(parent, child)
	return rel, err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func openAttestationRoot(logical string) (*os.Root, os.FileInfo, error) {
	before, err := os.Lstat(logical)
	if err != nil {
		return nil, nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return nil, nil, fmt.Errorf("%s must be a real non-symlink directory", logical)
	}
	root, err := os.OpenRoot(logical)
	if err != nil {
		return nil, nil, err
	}
	inside, err := root.Lstat(".")
	if err != nil || !attestationSameDirectory(before, inside) {
		return nil, nil, errors.Join(fmt.Errorf("%s root changed while opening", logical), err, root.Close())
	}
	return root, before, nil
}

// Descendant opening is relative to a retained authority. Validate each name
// before and after opening, including ancestors of the final root leaf.
func openAttestationChild(parent *os.Root, rel, logical string) (*os.Root, os.FileInfo, error) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	witnesses := map[string]os.FileInfo{}
	for i := range parts {
		name := filepath.FromSlash(strings.Join(parts[:i+1], "/"))
		info, err := parent.Lstat(name)
		if err != nil || info == nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, nil, errors.Join(fmt.Errorf("%s ancestor %s must be a real non-symlink directory", logical, name), err)
		}
		witnesses[name] = info
	}
	root, err := parent.OpenRoot(rel)
	if err != nil {
		return nil, nil, err
	}
	inside, err := root.Lstat(".")
	if err != nil || !attestationSameDirectory(witnesses[rel], inside) {
		return nil, nil, errors.Join(fmt.Errorf("%s changed while opening rooted child", logical), err, root.Close())
	}
	for name, before := range witnesses {
		after, err := parent.Lstat(name)
		if err != nil || !attestationSameDirectory(before, after) {
			return nil, nil, errors.Join(fmt.Errorf("%s ancestor %s changed", logical, name), err, root.Close())
		}
	}
	return root, inside, nil
}

func attestationSameDirectory(a, b os.FileInfo) bool {
	return a != nil && b != nil && a.IsDir() && b.IsDir() && a.Mode() == b.Mode() && os.SameFile(a, b)
}

// MaterializeAttestationTree binds the full supplied root and the original
// design generation without using generic ambient external-tree authority.
func (l *Lock) MaterializeAttestationTree(path string) (result *AttestationTreeSnapshot, retErr error) {
	if l == nil || l.lock == nil || l.sourceRoot == "" {
		return nil, fmt.Errorf("GV_SCOPE_CUSTODY: design snapshot released or unavailable")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if rel, inside := attestationWithin(l.sourceRoot, abs); inside {
		abs = filepath.Join(l.root, rel)
	}
	for _, alias := range l.sourceAliases {
		if _, inside := attestationWithin(alias, abs); inside {
			return nil, fmt.Errorf("GV_SCOPE_CUSTODY: unrecognized private source path")
		}
	}
	s := &AttestationTreeSnapshot{lock: l, logical: abs, witnesses: map[string]os.FileInfo{}}
	defer func() {
		if retErr != nil {
			retErr = l.LogicalError(errors.Join(fmt.Errorf("GV_SCOPE_CUSTODY: %w", retErr), s.Close()))
		}
	}()
	if rel, inside := attestationWithin(abs, l.root); inside {
		s.impl, s.implInfo, err = openAttestationRoot(abs)
		if err != nil {
			return nil, err
		}
		if rel == "." {
			s.design, s.designInfo = s.impl, s.implInfo
		} else {
			s.design, s.designInfo, err = openAttestationChild(s.impl, rel, l.root)
		}
	} else {
		s.design, s.designInfo, err = openAttestationRoot(l.root)
		if err != nil {
			return nil, err
		}
		if rel, inside := attestationWithin(l.root, abs); inside {
			s.impl, s.implInfo, err = openAttestationChild(s.design, rel, abs)
		} else {
			s.impl, s.implInfo, err = openAttestationRoot(abs)
		}
	}
	if err != nil {
		return nil, err
	}
	if !attestationSameDirectory(l.rootInfo, s.designInfo) {
		return nil, fmt.Errorf("original design root generation changed: %s", l.root)
	}
	if err := s.checkDesign(); err != nil {
		return nil, err
	}
	policy := attestationInventoryPolicy{strict: true, witnesses: s.witnesses}
	copyTo := ""
	if rel, inside := attestationWithin(l.root, abs); inside {
		s.path = filepath.Join(l.SourceRoot(), rel)
	} else {
		s.cleanup, err = newPrivateSnapshot("machinery-attestation-")
		if err != nil {
			return nil, err
		}
		s.path, copyTo = s.cleanup.Path(), s.cleanup.Path()
		l.inputAliases = append(l.inputAliases, pathAlias{from: s.path, to: abs})
		if rel, inside := attestationWithin(abs, l.root); inside {
			policy.copySkip = rel
		}
	}
	s.entries, err = captureAttestationRoot(s.impl, abs, policy, copyTo)
	if err != nil {
		return nil, err
	}
	if err := s.CheckUnchanged(); err != nil {
		return nil, err
	}
	if err := s.checkCopy(policy.copySkip); err != nil {
		return nil, err
	}
	return s, nil
}

func attestationFingerprint(entries []AttestationTreeEntry) map[string]string {
	values := map[string]string{}
	for _, e := range entries {
		if e.Path == "." {
			continue
		}
		if e.Directory {
			values[e.Path+"/"] = fmt.Sprintf("dir:%o", e.Mode)
		} else {
			values[e.Path] = fmt.Sprintf("file:%o:%x", e.Mode, e.SHA256)
		}
	}
	return values
}

func (s *AttestationTreeSnapshot) checkDesign() error {
	entries, err := captureAttestationRoot(s.design, s.lock.root, attestationInventoryPolicy{}, "")
	if err != nil {
		return err
	}
	if change := firstFingerprintChange(s.lock.snapshot, attestationFingerprint(entries)); change != "" {
		return fmt.Errorf("original design generation changed at %s", change)
	}
	copyRoot, err := os.OpenRoot(s.lock.SourceRoot())
	if err != nil {
		return err
	}
	copied, readErr := captureAttestationRoot(copyRoot, s.lock.root, attestationInventoryPolicy{}, "")
	if err := errors.Join(readErr, copyRoot.Close()); err != nil {
		return err
	}
	if change := firstFingerprintChange(s.lock.snapshot, attestationFingerprint(copied)); change != "" {
		return fmt.Errorf("retained design generation differs at %s", change)
	}
	return nil
}

func (s *AttestationTreeSnapshot) checkCopy(skip string) error {
	root, err := os.OpenRoot(s.path)
	if err != nil {
		return err
	}
	entries, readErr := captureAttestationRoot(root, s.logical, attestationInventoryPolicy{}, "")
	if err := errors.Join(readErr, root.Close()); err != nil {
		return err
	}
	var want []AttestationTreeEntry
	for _, entry := range s.entries {
		if skip != "" && (entry.Path == filepath.ToSlash(skip) || strings.HasPrefix(entry.Path, filepath.ToSlash(skip)+"/")) {
			continue
		}
		want = append(want, entry)
	}
	if change := firstFingerprintChange(attestationFingerprint(want), attestationFingerprint(entries)); change != "" {
		return fmt.Errorf("stable implementation copy differs at %s", change)
	}
	return nil
}

// CheckUnchanged rehashes the retained original roots, checks name witnesses,
// and preserves the exact design generation from acquisition.
func (s *AttestationTreeSnapshot) CheckUnchanged() error {
	if s == nil || s.closed {
		return fmt.Errorf("GV_SCOPE_CUSTODY: attestation snapshot released")
	}
	var errs []error
	for _, w := range []struct {
		name string
		root *os.Root
		info os.FileInfo
	}{{s.logical, s.impl, s.implInfo}, {s.lock.root, s.design, s.designInfo}} {
		named, ne := os.Lstat(w.name)
		held, he := w.root.Lstat(".")
		if ne != nil || he != nil || !attestationSameDirectory(w.info, named) || !attestationSameDirectory(w.info, held) {
			errs = append(errs, errors.Join(fmt.Errorf("original root changed identity: %s", w.name), ne, he))
		}
	}
	entries, err := captureAttestationRoot(s.impl, s.logical, attestationInventoryPolicy{strict: true, witnesses: s.witnesses}, "")
	errs = append(errs, err)
	if err == nil && !reflect.DeepEqual(entries, s.entries) {
		errs = append(errs, fmt.Errorf("original implementation inventory changed at %s", firstFingerprintChange(attestationFingerprint(s.entries), attestationFingerprint(entries))))
	}
	errs = append(errs, s.checkDesign())
	if err := errors.Join(errs...); err != nil {
		return s.lock.LogicalError(fmt.Errorf("GV_SCOPE_CUSTODY: %w", err))
	}
	return nil
}

// Close joins all original-handle and owned-copy cleanup errors exactly once.
func (s *AttestationTreeSnapshot) Close() error {
	if s == nil {
		return nil
	}
	if s.closed {
		return s.closeErr
	}
	s.closed = true
	var errs []error
	if s.impl != nil {
		errs = append(errs, s.impl.Close())
	}
	if s.design != nil && s.design != s.impl {
		errs = append(errs, s.design.Close())
	}
	if s.cleanup != nil {
		errs = append(errs, s.cleanup.Close())
	}
	s.closeErr = s.lock.LogicalError(errors.Join(errs...))
	return s.closeErr
}

func captureAttestationRoot(root *os.Root, logical string, policy attestationInventoryPolicy, copyTo string) ([]AttestationTreeEntry, error) {
	budget := newAttestationSnapshotBudget()
	if err := budget.addEntries(logical, 1); err != nil {
		return nil, err
	}
	var out []AttestationTreeEntry
	folded := map[string]string{}
	identities := map[string][]struct {
		path string
		info os.FileInfo
	}{}
	var walk func(string, int) error
	walk = func(name string, depth int) error {
		label := filepath.Join(logical, name)
		before, err := root.Lstat(name)
		if err != nil {
			return err
		}
		if before.Mode()&os.ModeSymlink != 0 || (!before.IsDir() && !before.Mode().IsRegular()) {
			return fmt.Errorf("%s is a symlink or special file", label)
		}
		if previous := policy.witnesses[name]; previous != nil {
			stable := sameFingerprintFile(previous, before)
			if before.IsDir() {
				stable = attestationSameDirectory(previous, before)
			}
			if !stable {
				return fmt.Errorf("%s changed identity or metadata since capture", label)
			}
		} else if policy.witnesses != nil {
			policy.witnesses[name] = before
		}
		entry := AttestationTreeEntry{Path: filepath.ToSlash(name), Directory: before.IsDir(), Mode: uint32(before.Mode().Perm())}
		copyEntry := copyTo != "" && !(policy.copySkip != "" && (name == policy.copySkip || strings.HasPrefix(name, policy.copySkip+string(filepath.Separator))))
		if before.IsDir() {
			if err := budget.enterDirectory(label, depth); err != nil {
				return err
			}
			if copyEntry && name != "." {
				if err := os.Mkdir(filepath.Join(copyTo, name), 0o700); err != nil {
					return err
				}
			}
			file, err := root.Open(name)
			if err != nil {
				return err
			}
			opened, statErr := file.Stat()
			if statErr != nil || !attestationSameDirectory(before, opened) {
				return errors.Join(fmt.Errorf("%s changed while opening directory", label), statErr, file.Close())
			}
			children, readErr := readSnapshotDir(file, label, &budget)
			if err := errors.Join(readErr, file.Close()); err != nil {
				return err
			}
			out = append(out, entry)
			sort.Slice(children, func(i, j int) bool { return children[i].Name() < children[j].Name() })
			for _, child := range children {
				rel := filepath.Join(name, child.Name())
				if child.Name() == ".git" {
					if policy.strict {
						if name != "." {
							return fmt.Errorf("GV_SCOPE_UNSUPPORTED_METADATA: nested metadata %s", filepath.Join(logical, rel))
						}
						info, err := root.Lstat(rel)
						if err != nil || info == nil || info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
							return errors.Join(fmt.Errorf("%s metadata must be a real directory or regular gitfile", filepath.Join(logical, rel)), err)
						}
					}
					continue
				}
				if err := validateInventoryPath(filepath.ToSlash(rel), folded); err != nil {
					return fmt.Errorf("GV_SCOPE_PATH: %w", err)
				}
				if err := walk(rel, depth+1); err != nil {
					return err
				}
			}
			if copyEntry && name != "." {
				if err := os.Chmod(filepath.Join(copyTo, name), before.Mode().Perm()); err != nil {
					return err
				}
			}
			after, err := root.Lstat(name)
			if err != nil || !attestationSameDirectory(before, after) {
				return errors.Join(fmt.Errorf("%s changed while reading directory", label), err)
			}
			return nil
		}
		if err := budget.addFile(label, before); err != nil {
			return err
		}
		if policy.strict {
			// Unix Dev/Ino accelerate SameFile without persisting identities.
			// Other platforms conservatively use the full SameFile comparison.
			key := "other-platform"
			value := reflect.Indirect(reflect.ValueOf(before.Sys()))
			if value.IsValid() && value.Kind() == reflect.Struct {
				dev, ino := value.FieldByName("Dev"), value.FieldByName("Ino")
				if dev.IsValid() && ino.IsValid() {
					key = fmt.Sprint(dev, ":", ino)
				}
			}
			for _, prior := range identities[key] {
				if os.SameFile(prior.info, before) {
					return fmt.Errorf("GV_SCOPE_ALIAS: %s and %s alias the same file identity", prior.path, label)
				}
			}
			identities[key] = append(identities[key], struct {
				path string
				info os.FileInfo
			}{label, before})
		}
		file, err := root.Open(name)
		if err != nil {
			return err
		}
		opened, statErr := file.Stat()
		if statErr != nil || !sameFingerprintFile(before, opened) {
			return errors.Join(fmt.Errorf("%s changed while opening file", label), statErr, file.Close())
		}
		var writer io.Writer = io.Discard
		var dest *os.File
		if copyEntry {
			dest, err = os.OpenFile(filepath.Join(copyTo, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
			if err != nil {
				return errors.Join(err, file.Close())
			}
			writer = dest
		}
		digest, readErr := copySnapshotFile(label, file, writer, before.Size())
		held, heldErr := file.Stat()
		closeErr := file.Close()
		if dest != nil {
			closeErr = errors.Join(closeErr, dest.Close(), os.Chmod(filepath.Join(copyTo, name), before.Mode().Perm()))
		}
		after, pathErr := root.Lstat(name)
		if err := errors.Join(readErr, heldErr, closeErr, pathErr); err != nil {
			return err
		}
		if !sameFingerprintFile(before, held) || !sameFingerprintFile(before, after) {
			return fmt.Errorf("%s changed while reading file", label)
		}
		entry.Size, entry.SHA256 = before.Size(), digest
		out = append(out, entry)
		return nil
	}
	if err := walk(".", 0); err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
