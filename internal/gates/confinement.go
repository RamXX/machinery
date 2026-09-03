package gates

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/portablepath"
)

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

func readRootRegularFile(root *os.Root, rel string) ([]byte, error) {
	f, err := root.Open(rel)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s must be a regular file, not a special file", rel)
	}
	return io.ReadAll(f)
}

// validateDesignInventory is the universal read-side boundary for a suite
// run. Authored designs are portable regular files in real directories;
// symlinks and special entries are never valid evidence, even when a narrow
// explicit gate would otherwise happen not to read them.
func validateDesignInventory(design string) error {
	root, rootPath, err := openRealRoot(design)
	if err != nil {
		return err
	}
	defer root.Close()
	caseFolded := map[string]string{}
	return fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := filepath.FromSlash(path)
		if path == "." {
			return nil
		}
		if ignoredHere(rootPath, rel) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		portableRel := filepath.ToSlash(rel)
		if err := validatePortableInventoryPath(caseFolded, portableRel); err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: design artifacts must not be symlinks", filepath.ToSlash(rel))
		}
		if !entry.IsDir() {
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if !entryInfo.Mode().IsRegular() {
				return fmt.Errorf("%s: design artifacts must be regular files", filepath.ToSlash(rel))
			}
		}
		return nil
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
