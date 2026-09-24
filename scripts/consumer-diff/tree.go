package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// manifest maps a slash path relative to a tree root to a digest of what is
// there: the entry type, its permission bits, and the file content or the
// symlink target. Modification times are deliberately absent: a copy must
// compare equal to its source.
type manifest map[string]string

// findCopyRoot returns the git work tree holding dir (the nearest ancestor
// with a .git entry), so the copy keeps the repository history Ga-accept
// reads; outside git it is dir itself.
func findCopyRoot(dir string) string {
	for d := dir; ; {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return dir
		}
		d = parent
	}
}

// hashTree walks root without following symlinks. Sockets, pipes and devices
// carry no content and are skipped, in the manifest and in the copy alike.
func hashTree(root string) (manifest, error) {
	m := manifest{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode.IsDir():
			m[rel] = fmt.Sprintf("dir %o", mode.Perm())
		case mode&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			m[rel] = "link " + target
		case mode.IsRegular():
			sum, err := fileDigest(path)
			if err != nil {
				return err
			}
			m[rel] = fmt.Sprintf("file %o %s", mode.Perm(), sum)
		}
		return nil
	})
	return m, err
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// diffManifests lists every path added, removed, or changed, sorted.
func diffManifests(before, after manifest) []string {
	var changed []string
	for p, v := range before {
		if after[p] != v {
			changed = append(changed, p)
		}
	}
	for p := range after {
		if _, ok := before[p]; !ok {
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	return changed
}

// copyTree reproduces src at dst: files with their bytes and permission bits,
// symlinks as symlinks (never followed), directories with their permission
// bits applied last so a read-only directory can still be filled.
func copyTree(src, dst string) error {
	type dirMode struct {
		path string
		perm fs.FileMode
	}
	var dirs []dirMode
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		switch {
		case mode.IsDir():
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			dirs = append(dirs, dirMode{target, mode.Perm()})
		case mode&fs.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case mode.IsRegular():
			return copyFile(path, target, mode.Perm())
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := os.Chmod(dirs[i].path, dirs[i].perm); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, perm)
}

// removeTree deletes a copy even when it holds read-only directories.
func removeTree(root string) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(root)
}
