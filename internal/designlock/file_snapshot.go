package designlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RegularFileSnapshot is a private stable copy of one generator input.
type RegularFileSnapshot struct {
	path    string
	cleanup *privateSnapshotCleanup
}

func (s *RegularFileSnapshot) Path() string { return s.path }

func (s *RegularFileSnapshot) Close() error {
	if s == nil || s.path == "" {
		return nil
	}
	cleanup := s.cleanup
	if cleanup == nil {
		s.path = ""
		return nil
	}
	if err := cleanup.Close(); err != nil {
		return err
	}
	s.path = ""
	s.cleanup = nil
	return nil
}

// MaterializeRegularFile binds one regular input to the held design snapshot
// and copies it through a no-follow os.Root. Inputs outside design are tracked
// by identity, mode, and content through the final CheckUnchanged call.
func (l *Lock) MaterializeRegularFile(path string) (*RegularFileSnapshot, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if inside, rel := l.insideDesign(abs); inside {
		stable := filepath.Join(l.SourceRoot(), rel)
		info, statErr := os.Lstat(stable)
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("generator input %s must be a regular file, not a symlink or special entry", path)
		}
		return &RegularFileSnapshot{path: stable}, nil
	}
	if l.insideSource(abs) {
		info, statErr := os.Lstat(abs)
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("generator input %s must be a regular file, not a symlink or special entry", path)
		}
		return &RegularFileSnapshot{path: abs}, nil
	}
	rootPath, rel, expectedRoot := filepath.Dir(abs), filepath.Base(abs), os.FileInfo(nil)
	parentInfo, statErr := os.Lstat(rootPath)
	if statErr != nil {
		return nil, fmt.Errorf("inspect external input parent: %w", statErr)
	}
	if parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		return nil, fmt.Errorf("external input parent must be a real directory: %s", rootPath)
	}
	rootPath, err = filepath.EvalSymlinks(rootPath)
	if err != nil {
		return nil, err
	}
	if err := validateRealDirectory(rootPath); err != nil {
		return nil, fmt.Errorf("external input parent: %w", err)
	}
	expectedRoot, err = os.Lstat(rootPath)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	openedRoot, err := root.Lstat(".")
	if err != nil || !os.SameFile(expectedRoot, openedRoot) {
		return nil, fmt.Errorf("input root changed identity while opening %s", path)
	}
	info, err := root.Lstat(rel)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("generator input %s must be a regular file, not a symlink or special entry", path)
	}
	src, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	openedInfo, statErr := src.Stat()
	if statErr != nil || !sameFingerprintFile(info, openedInfo) {
		_ = src.Close()
		return nil, fmt.Errorf("generator input %s changed identity while opening", path)
	}
	cleanup, err := newPrivateSnapshot("machinery-input-snapshot-")
	if err != nil {
		_ = src.Close()
		return nil, err
	}
	temp := cleanup.Path()
	snapshot := &RegularFileSnapshot{path: filepath.Join(temp, filepath.Base(abs)), cleanup: cleanup}
	l.inputAliases = append(l.inputAliases, pathAlias{from: snapshot.path, to: abs})
	dst, err := os.OpenFile(snapshot.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		_ = src.Close()
		return nil, errors.Join(err, snapshot.Close())
	}
	digest, copyErr := copySnapshotFile(path, src, dst, openedInfo.Size())
	openedAfter, retainedStatErr := src.Stat()
	after, pathStatErr := root.Lstat(rel)
	err = errors.Join(copyErr, retainedStatErr, pathStatErr, src.Close(), dst.Close())
	if err != nil {
		return nil, errors.Join(err, snapshot.Close())
	}
	if !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
		return nil, errors.Join(fmt.Errorf("generator input %s changed identity while materializing", path), snapshot.Close())
	}
	value := fmt.Sprintf("file:%o:%x", info.Mode().Perm(), digest)
	if inside, _ := l.insideDesign(abs); !inside {
		if l.external == nil {
			l.external = map[string]externalFileState{}
		}
		l.external[abs] = externalFileState{value: value, info: info}
	}
	if err := l.CheckUnchanged(); err != nil {
		return nil, errors.Join(err, snapshot.Close())
	}
	return snapshot, nil
}

func (l *Lock) insideDesign(abs string) (bool, string) {
	rel, err := filepath.Rel(l.root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, ""
	}
	return true, rel
}

// ValidateOutputDir rejects symlinked/special components before a generator
// delegates the actual rooted transaction to artifactset.
func (l *Lock) ValidateOutputDir(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	probe := abs
	for {
		info, statErr := os.Lstat(probe)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("output directory component %s must be a real directory", probe)
			}
			resolved, err := filepath.EvalSymlinks(probe)
			if err != nil {
				return err
			}
			return validateRealDirectory(resolved)
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return statErr
		}
		probe = parent
	}
}

func validateRealDirectory(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	parent := filepath.Dir(abs)
	if parent != abs {
		if err := validateRealDirectory(parent); err != nil {
			return err
		}
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("path component %s is a symlink or non-directory", abs)
	}
	return nil
}
