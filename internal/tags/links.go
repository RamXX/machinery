package tags

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/project"
)

// Link is a symbolic link under trace.code or trace.tests (section 8): the
// scan does not follow it, so the markers behind it are never read.
type Link struct {
	Path  string   // project-relative
	Lists []string // the trace lists it is under: trace.code, trace.tests
}

// Links lists the symbolic links the scan does not follow and the test
// runner may: a link whose own path a glob of trace.code (inside scope) or
// trace.tests matches, and a link to a directory with a file behind it that
// would match as <link>/<path>. A link to something else, a dangling link and
// a link that leaves the project are found the same way. Sorted by path.
func Links(p *project.Project) ([]Link, error) {
	if p.Cfg.Trace == nil {
		return nil, nil
	}
	lists := []struct {
		name    string
		globs   []string
		inScope bool
	}{{"trace.code", p.Cfg.Trace.Code, true}, {"trace.tests", p.Cfg.Trace.Tests, false}}
	var res []Link
	err := filepath.WalkDir(p.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != p.Root && project.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		rel := p.Rel(path)
		l := Link{Path: rel}
		for _, list := range lists {
			if reaches(p, path, rel, list.globs, list.inScope) {
				l.Lists = append(l.Lists, list.name)
			}
		}
		if len(l.Lists) > 0 {
			res = append(res, l)
		}
		return nil
	})
	sort.Slice(res, func(i, j int) bool { return res[i].Path < res[j].Path })
	return res, err
}

// reaches reports whether a glob list matches the link itself or, when the
// link leads to a directory, a file the directory holds, as it would be named
// below the link.
func reaches(p *project.Project, abs, rel string, globs []string, inScope bool) bool {
	ok := func(f string) bool { return project.Match(globs, f) && (!inScope || p.InScope(f)) }
	if ok(rel) {
		return true
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() || !below(globs, rel) {
		return false
	}
	target, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != target && project.SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		sub, err := filepath.Rel(target, path)
		if err != nil {
			return nil
		}
		if ok(rel + "/" + filepath.ToSlash(sub)) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// below reports whether a glob could match a path below dir: its fixed
// directory prefix leads into dir or out of it. A link to a large tree is
// walked only when a glob can reach below it.
func below(globs []string, dir string) bool {
	for _, g := range globs {
		pre := project.GlobPrefix(g)
		if strings.HasPrefix(pre, dir+"/") || strings.HasPrefix(dir+"/", pre) {
			return true
		}
	}
	return false
}
