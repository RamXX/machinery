package project

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// skipDirs are never walked: they hold tools, tracker data and other checkouts.
var skipDirs = map[string]bool{".git": true, "node_modules": true, ".machinery": true, ".beads": true, ".worktrees": true, ".bin": true}

// SkipDir reports whether a directory of this name is never walked.
func SkipDir(name string) bool { return skipDirs[name] }

// Match reports whether a project-relative path matches any of the globs.
func Match(globs []string, rel string) bool {
	for _, g := range globs {
		if ok, _ := doublestar.Match(g, rel); ok {
			return true
		}
		// a bare directory glob "dir/**" also matches files directly under dir
	}
	return false
}

// ValidGlob reports whether a pattern is well formed.
func ValidGlob(g string) bool { return doublestar.ValidatePattern(g) }

// Files lists every regular file under the root as sorted project-relative paths.
func (p *Project) Files() ([]string, error) {
	var res []string
	err := filepath.WalkDir(p.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != p.Root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			res = append(res, p.Rel(path))
		}
		return nil
	})
	sort.Strings(res)
	return res, err
}

// Glob lists the files matching any of the globs, sorted.
func (p *Project) Glob(globs []string) ([]string, error) {
	if len(globs) == 0 {
		return nil, nil
	}
	all, err := p.Files()
	if err != nil {
		return nil, err
	}
	var res []string
	for _, f := range all {
		if Match(globs, f) {
			res = append(res, f)
		}
	}
	return res, nil
}

// InScope reports whether a project-relative path is inside the scope.
func (p *Project) InScope(rel string) bool { return Match(p.Cfg.Scope, rel) }

// Read reads a project-relative file.
func (p *Project) Read(rel string) ([]byte, error) { return os.ReadFile(p.Abs(rel)) }

// Exists reports whether a project-relative path exists.
func (p *Project) Exists(rel string) bool {
	_, err := os.Stat(p.Abs(rel))
	return err == nil
}

// WriteAtomic writes a project-relative file through a temporary file and a rename.
func (p *Project) WriteAtomic(rel string, data []byte) error {
	return WriteFileAtomic(p.Abs(rel), data)
}

// WriteFileAtomic writes a file through a temporary file and a rename.
func WriteFileAtomic(abs string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), "."+filepath.Base(abs)+".*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, abs)
}

// GlobPrefix returns the fixed directory prefix of a glob (before any wildcard).
func GlobPrefix(g string) string {
	i := strings.IndexAny(g, "*?[{")
	if i < 0 {
		return g
	}
	g = g[:i]
	if j := strings.LastIndex(g, "/"); j >= 0 {
		return g[:j+1]
	}
	return ""
}
