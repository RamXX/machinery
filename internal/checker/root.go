package checker

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// designRoot keeps a capability to the design directory open across a complete
// checker operation. All names passed to os.Root remain design-relative, so a
// concurrent replacement of an intermediate directory with a symlink cannot
// redirect a read or mutation outside the directory originally opened.
type designRoot struct {
	abs         string
	root        *os.Root
	syncDirHook func(string) error
}

func openDesignRoot(design string) (*designRoot, error) {
	abs, err := cleanDesignRoot(design)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return nil, fmt.Errorf("open checker design root %s: %w", abs, err)
	}
	return &designRoot{abs: abs, root: root}, nil
}

func (r *designRoot) close() error {
	if r == nil || r.root == nil {
		return nil
	}
	return r.root.Close()
}

func (r *designRoot) rel(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(r.abs, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path %s escapes design %s", path, r.abs)
	}
	if rel == "." {
		return rel, nil
	}
	if err := validatePortableRelativePath(filepath.ToSlash(rel)); err != nil {
		return "", fmt.Errorf("path %s is not portable: %w", path, err)
	}
	return filepath.Clean(rel), nil
}

func (r *designRoot) display(rel string) string {
	return filepath.Join(r.abs, filepath.FromSlash(rel))
}

func (r *designRoot) confinedRel(rel string) (string, error) {
	if err := validatePortableRelativePath(rel); err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%q resolves outside the design directory", rel)
	}
	if err := r.validateNoSymlink(clean); err != nil {
		return "", err
	}
	return clean, nil
}

func (r *designRoot) modelPaths() ([]string, error) {
	dir, err := r.root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var names []string
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".modelith.yaml") {
			continue
		}
		if err := validatePortableComponent(entry.Name()); err != nil {
			return nil, fmt.Errorf("model name %q is not portable: %w", entry.Name(), err)
		}
		exists, err := r.lstatRegular(entry.Name(), "model", false)
		if err != nil {
			return nil, err
		}
		if exists {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (r *designRoot) manifestPaths() ([]string, error) {
	const dirName = "checkers"
	if err := r.validateNoSymlink(dirName); err != nil {
		return nil, err
	}
	info, err := r.root.Lstat(dirName)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect checker directory %s: %w", r.display(dirName), err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("checker directory %s must be a real directory", r.display(dirName))
	}
	dir, err := r.root.Open(dirName)
	if err != nil {
		return nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		return nil, fmt.Errorf("read checker directory %s: %w", r.display(dirName), err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var out []string
	portableNames := map[string]string{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".checker.yaml") {
			continue
		}
		if err := validatePortableComponent(entry.Name()); err != nil {
			return nil, fmt.Errorf("checker manifest name %q is not portable: %w", entry.Name(), err)
		}
		rel := filepath.Join(dirName, entry.Name())
		exists, err := r.lstatRegular(rel, "checker manifest", false)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		folded := strings.ToLower(entry.Name())
		if prior, exists := portableNames[folded]; exists {
			return nil, fmt.Errorf("checker manifests %q and %q collide on a case-insensitive filesystem", prior, entry.Name())
		}
		portableNames[folded] = entry.Name()
		out = append(out, rel)
	}
	sort.Strings(out)
	return out, nil
}

func (r *designRoot) loadModel(rel string) (*Model, []byte, error) {
	data, err := r.readRegular(rel, "model", false)
	if err != nil {
		return nil, nil, err
	}
	model, err := parseModel(r.display(rel), data)
	return model, data, err
}

func (r *designRoot) loadManifest(rel string) (*Manifest, error) {
	data, err := r.readRegular(rel, "checker manifest", false)
	if err != nil {
		return nil, err
	}
	manifest, err := parseManifest(r.display(rel), data)
	if err != nil {
		return nil, err
	}
	for _, output := range []struct{ field, rel string }{
		{"evidence.projection_out", manifest.Evidence.ProjectionOut},
		{"evidence.evidence_in", manifest.Evidence.EvidenceIn},
	} {
		if _, err := r.confinedRel(output.rel); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", manifest.Path, output.field, err)
		}
	}
	return manifest, nil
}

func designIDBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum)
}

// validateNoSymlink rejects every existing symlink and every non-directory
// parent. os.Root supplies the race-safe confinement; this check preserves the
// checker's stricter policy that symlinks are never accepted even when they
// would remain inside the design.
func (r *designRoot) validateNoSymlink(rel string) error {
	if rel == "." {
		return nil
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	parts := strings.Split(clean, string(os.PathSeparator))
	current := ""
	for i, part := range parts {
		if current == "" {
			current = part
		} else {
			current = filepath.Join(current, part)
		}
		info, err := r.root.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("contains symlink component %s", r.display(current))
		}
		if i < len(parts)-1 && !info.IsDir() {
			return fmt.Errorf("contains non-directory component %s", r.display(current))
		}
	}
	return nil
}

func (r *designRoot) lstatRegular(rel, kind string, private bool) (bool, error) {
	if err := r.validateNoSymlink(rel); err != nil {
		return false, err
	}
	info, err := r.root.Lstat(rel)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || (private && !projectionControlPermissionsSafe(info.Mode())) {
		return false, fmt.Errorf("%s %s must be a %sregular, non-symlink file", kind, r.display(rel), map[bool]string{true: "private ", false: ""}[private])
	}
	return true, nil
}

func (r *designRoot) readRegular(rel, kind string, private bool) ([]byte, error) {
	exists, err := r.lstatRegular(rel, kind, private)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &os.PathError{Op: "open", Path: r.display(rel), Err: os.ErrNotExist}
	}
	data, err := r.root.ReadFile(rel)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *designRoot) readRegularBounded(rel, kind string, private bool, maxBytes int64) (data []byte, retErr error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("%s read bound must be positive", kind)
	}
	exists, err := r.lstatRegular(rel, kind, private)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, &os.PathError{Op: "open", Path: r.display(rel), Err: os.ErrNotExist}
	}
	file, err := r.root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %s must remain a regular file while open", kind, r.display(rel))
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, r.display(rel), maxBytes)
	}
	data, err = io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("%s %s exceeds %d-byte limit", kind, r.display(rel), maxBytes)
	}
	return data, nil
}

func (r *designRoot) syncDir(rel string) error {
	if err := r.validateNoSymlink(rel); err != nil {
		return err
	}
	if r.syncDirHook != nil {
		if err := r.syncDirHook(rel); err != nil {
			return err
		}
	}
	return syncRootDirectory(r.root, rel)
}

func closeRoot(retErr *error, root *designRoot) {
	*retErr = errors.Join(*retErr, root.close())
}
