package designlock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ExternalTreeSnapshot is a private immutable materialization of an
// implementation tree. Consumers read Path and call Close when their gate
// run finishes; Logical is the caller-facing path used to remap diagnostics.
type ExternalTreeSnapshot struct {
	path    string
	logical string
	cleanup *privateSnapshotCleanup
}

func (s *ExternalTreeSnapshot) Path() string    { return s.path }
func (s *ExternalTreeSnapshot) Logical() string { return s.logical }

func (s *ExternalTreeSnapshot) Close() error {
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

// MaterializeExternalTree copies path through one held no-follow os.Root,
// proves that copy equals the tracked pre/post fingerprints, and returns the
// private stable tree gates must use instead of ambient implementation paths.
func (l *Lock) MaterializeExternalTree(path string) (*ExternalTreeSnapshot, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if inside, rel := l.insideDesign(abs); inside {
		stable := filepath.Join(l.SourceRoot(), rel)
		info, statErr := os.Lstat(stable)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("design source tree %s must be a real directory", path)
		}
		return &ExternalTreeSnapshot{path: stable, logical: path}, nil
	}
	if l.insideSource(abs) {
		info, statErr := os.Lstat(abs)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("design source tree %s must be a real directory", path)
		}
		return &ExternalTreeSnapshot{path: abs, logical: path}, nil
	}
	if err := l.TrackExternalTree(abs); err != nil {
		return nil, err
	}
	cleanup, err := newPrivateSnapshot("machinery-impl-snapshot-")
	if err != nil {
		return nil, fmt.Errorf("create private implementation snapshot: %w", err)
	}
	temp := cleanup.Path()
	snapshot := &ExternalTreeSnapshot{path: temp, logical: path, cleanup: cleanup}
	l.inputAliases = append(l.inputAliases, pathAlias{from: temp, to: abs})
	values, err := l.copyExternalTree(abs, temp, nil, nil)
	if err != nil {
		return nil, errors.Join(err, snapshot.Close())
	}
	if got, want := fingerprintDigest(values), l.externalTrees[abs].digest; got != want {
		return nil, errors.Join(fmt.Errorf("implementation tree changed while materializing its stable snapshot: %s", abs), snapshot.Close())
	}
	if err := l.checkExternalUnchanged(); err != nil {
		return nil, errors.Join(err, snapshot.Close())
	}
	return snapshot, nil
}

// MaterializeDesignWorkspace preserves the common <workspace>/<component>/design
// relative-path topology while keeping the held design bytes from SourceRoot.
// Sibling inputs are independently snapshotted and tracked as external state.
func (l *Lock) MaterializeDesignWorkspace() (*ExternalTreeSnapshot, error) {
	usesParent, err := designUsesParentRelativeInput(l.SourceRoot())
	if err != nil {
		return nil, err
	}
	decomposedWorkspace := false
	if info, statErr := os.Lstat(filepath.Join(l.SourceRoot(), "decomposition.yaml")); statErr == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("decomposition.yaml must be a regular file before widening the retained workspace scope")
		}
		decomposedWorkspace = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}
	if filepath.Base(l.root) != "design" || (!usesParent && !decomposedWorkspace) {
		return &ExternalTreeSnapshot{path: l.SourceRoot(), logical: l.root}, nil
	}
	childWorkspace, err := hasChildPackCapability(l.SourceRoot())
	if err != nil {
		return nil, err
	}
	logicalScope := filepath.Dir(l.root)
	if childWorkspace || decomposedWorkspace {
		logicalScope = filepath.Dir(filepath.Dir(l.root))
	}
	rel, err := filepath.Rel(logicalScope, l.root)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("resolve design workspace topology for %s", l.root)
	}
	workspace, err := l.MaterializeExternalTree(logicalScope)
	if err != nil {
		return nil, err
	}
	designDest := filepath.Join(workspace.Path(), rel)
	if err := os.Mkdir(designDest, 0o700); err != nil {
		return nil, errors.Join(err, workspace.Close())
	}
	if _, err := l.copyExternalTree(l.SourceRoot(), designDest, nil, nil); err != nil {
		return nil, errors.Join(err, workspace.Close())
	}
	if err := os.Chmod(designDest, l.rootInfo.Mode().Perm()); err != nil {
		return nil, errors.Join(err, workspace.Close())
	}
	workspace.path = designDest
	workspace.logical = l.root
	return workspace, nil
}

// RetainedWorkspaceScope returns the lexical capability available to authored
// design references. A normal project/design may reach its project root only.
// A generated child pack is the explicit, committed proof that the design is
// one component inside a parent decomposition workspace, so only that shape
// may reach the shared grandparent containing sibling components.
func RetainedWorkspaceScope(design string) (string, error) {
	design = filepath.Clean(design)
	if filepath.Base(design) != "design" {
		return design, nil
	}
	project := filepath.Dir(design)
	childWorkspace, err := hasChildPackCapability(design)
	if err != nil {
		return "", err
	}
	decomposedWorkspace := false
	if info, statErr := os.Lstat(filepath.Join(design, "decomposition.yaml")); statErr == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("decomposition.yaml must be a regular file before widening the retained workspace scope")
		}
		decomposedWorkspace = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	if childWorkspace || decomposedWorkspace {
		return filepath.Dir(project), nil
	}
	return project, nil
}

func designUsesParentRelativeInput(root string) (bool, error) {
	values, err := fingerprint(root)
	if err != nil {
		return false, err
	}
	var names []string
	for name, value := range values {
		if strings.HasPrefix(value, "file:") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if !strings.HasSuffix(strings.ToLower(name), ".md") && !strings.EqualFold(filepath.Base(name), "workspace.dsl") {
			continue
		}
		body, err := readSnapshotRegularPath(filepath.Join(root, filepath.FromSlash(name)), "design reference document "+name, 64<<20)
		if err != nil {
			return false, err
		}
		for _, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasSuffix(strings.ToLower(name), ".md") {
				if !strings.Contains(trimmed, "machinery:embed") {
					continue
				}
				const marker = `from="`
				start := strings.Index(trimmed, marker)
				if start < 0 {
					continue
				}
				value := trimmed[start+len(marker):]
				if end := strings.IndexByte(value, '"'); end >= 0 {
					value = value[:end]
				}
				if value == ".." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))), "../") {
					return true, nil
				}
				continue
			}
			lower := strings.ToLower(trimmed)
			for _, directive := range []string{"!include", "!docs", "!adrs"} {
				if lower == directive || strings.HasPrefix(lower, directive+" ") || strings.HasPrefix(lower, directive+"\t") {
					args := strings.Fields(trimmed[len(directive):])
					if len(args) == 0 {
						continue
					}
					value := strings.Trim(strings.TrimSpace(args[0]), `"'`)
					if value == ".." || strings.HasPrefix(filepath.ToSlash(filepath.Clean(filepath.FromSlash(value))), "../") {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func (l *Lock) externalDesignRel(root string) string {
	rel, err := filepath.Rel(root, l.root)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return filepath.Clean(rel)
}

func (l *Lock) copyExternalTree(source, dest string, beforeOpen, afterRead func(string)) (map[string]string, error) {
	before, err := os.Lstat(source)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("implementation root must be a real directory")
	}
	if beforeOpen != nil {
		beforeOpen(".")
	}
	root, err := os.OpenRoot(source)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	inside, err := root.Lstat(".")
	if err != nil || !os.SameFile(before, inside) {
		return nil, fmt.Errorf("implementation root changed identity while opening stable snapshot")
	}
	exclude := l.externalDesignRel(source)
	if err := validateExternalTreeForCopy(root, exclude); err != nil {
		return nil, err
	}
	values := map[string]string{}
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		if err := budget.enterDirectory("implementation directory "+filepath.ToSlash(dir), depth); err != nil {
			return err
		}
		handle, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readSnapshotDir(handle, "implementation directory "+filepath.ToSlash(dir), &budget)
		closeErr := handle.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			name := entry.Name()
			if name == ".git" {
				continue
			}
			rel := name
			if dir != "." {
				rel = filepath.Join(dir, name)
			}
			if exclude != "" && (rel == exclude || strings.HasPrefix(rel, exclude+string(filepath.Separator))) {
				continue
			}
			info, err := root.Lstat(rel)
			if err != nil {
				return err
			}
			display := filepath.ToSlash(rel)
			if err := budget.addFile(display, info); err != nil {
				return err
			}
			destPath := filepath.Join(dest, rel)
			switch {
			case info.IsDir():
				values[display+"/"] = fmt.Sprintf("dir:%o", info.Mode().Perm())
				if err := os.Mkdir(destPath, 0o700); err != nil {
					return err
				}
				if err := walk(rel, depth+1); err != nil {
					return err
				}
				if err := os.Chmod(destPath, info.Mode().Perm()); err != nil {
					return err
				}
			case info.Mode().IsRegular():
				if beforeOpen != nil {
					beforeOpen(rel)
				}
				src, err := root.Open(rel)
				if err != nil {
					return err
				}
				openedInfo, statErr := src.Stat()
				if statErr != nil || !sameFingerprintFile(info, openedInfo) {
					_ = src.Close()
					if afterRead != nil {
						afterRead(rel)
					}
					return fmt.Errorf("implementation entry %s changed identity while opening", display)
				}
				dst, err := os.OpenFile(destPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
				if err != nil {
					_ = src.Close()
					return err
				}
				digest, copyErr := copySnapshotFile(display, src, dst, openedInfo.Size())
				openedAfter, retainedStatErr := src.Stat()
				err = errors.Join(copyErr, retainedStatErr, src.Close(), dst.Close())
				if err != nil {
					return err
				}
				if err := os.Chmod(destPath, info.Mode().Perm()); err != nil {
					return err
				}
				after, statErr := root.Lstat(rel)
				if statErr != nil || !sameFingerprintFile(info, openedAfter) || !sameFingerprintFile(info, after) {
					return fmt.Errorf("implementation entry %s changed identity while reading", display)
				}
				values[display] = fmt.Sprintf("file:%o:%x", info.Mode().Perm(), digest)
				if afterRead != nil {
					afterRead(rel)
				}
			case info.Mode()&os.ModeSymlink != 0:
				return fmt.Errorf("implementation inventory entry %s is a symlink", display)
			default:
				return fmt.Errorf("implementation inventory entry %s is a special file (%s)", display, info.Mode())
			}
		}
		return nil
	}
	if err := walk(".", 0); err != nil {
		return nil, err
	}
	current, err := l.fingerprintExternalTree(source)
	if err != nil {
		return nil, fmt.Errorf("recheck implementation inventory: %w", err)
	}
	if changed := firstFingerprintChange(values, current); changed != "" {
		return nil, fmt.Errorf("implementation inventory changed while materializing at %s", changed)
	}
	copied, err := fingerprint(dest)
	if err != nil {
		return nil, fmt.Errorf("fingerprint materialized tree: %w", err)
	}
	if got, want := fingerprintDigest(copied), fingerprintDigest(values); got != want {
		return nil, fmt.Errorf("materialized tree metadata or contents differ from its source")
	}
	return copied, nil
}

func validateExternalTreeForCopy(root *os.Root, exclude string) error {
	budget := snapshotBudget{maxEntries: snapshotInventoryMaxEntries, maxBytes: snapshotAggregateMaxBytes, maxDepth: snapshotInventoryMaxDepth}
	caseFolded := map[string]string{}
	var walk func(string, int) error
	walk = func(dir string, depth int) error {
		label := "implementation preflight directory " + filepath.ToSlash(dir)
		if err := budget.enterDirectory(label, depth); err != nil {
			return err
		}
		before, err := root.Lstat(dir)
		if err != nil || !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
			return errors.Join(err, fmt.Errorf("%s must remain a real directory", label))
		}
		handle, err := root.Open(dir)
		if err != nil {
			return err
		}
		entries, readErr := readSnapshotDir(handle, label, &budget)
		closeErr := handle.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return err
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			name := entry.Name()
			if name == ".git" {
				continue
			}
			rel := name
			if dir != "." {
				rel = filepath.Join(dir, name)
			}
			if exclude != "" && (rel == exclude || strings.HasPrefix(rel, exclude+string(filepath.Separator))) {
				continue
			}
			display := filepath.ToSlash(rel)
			if err := validateInventoryPath(display, caseFolded); err != nil {
				return err
			}
			info, err := root.Lstat(rel)
			if err != nil {
				return err
			}
			if err := budget.addFile(display, info); err != nil {
				return err
			}
			switch {
			case info.IsDir() && info.Mode()&os.ModeSymlink == 0:
				if err := walk(rel, depth+1); err != nil {
					return err
				}
			case info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0:
			case info.Mode()&os.ModeSymlink != 0:
				return fmt.Errorf("implementation inventory entry %s is a symlink", display)
			default:
				return fmt.Errorf("implementation inventory entry %s is a special file (%s)", display, info.Mode())
			}
		}
		after, err := root.Lstat(dir)
		if err != nil || !os.SameFile(before, after) || before.Mode() != after.Mode() {
			return errors.Join(err, fmt.Errorf("%s changed while reading", label))
		}
		return nil
	}
	return walk(".", 0)
}
