package tags

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Link is a symbolic link a trace list reaches (section 8): the scan does not
// follow it, so the markers behind it are never read.
type Link struct {
	Path  string   // project-relative, as the runner sees it
	Lists []string // the trace lists that reach it: trace.code, trace.tests
}

// traceList is one list of globs that can reach a link.
type traceList struct {
	name    string
	globs   []string
	inScope bool // trace.code reaches only links inside scope
}

// metLink is a link the walk met. back is set for a link that leads to a
// directory the walk was inside: the path of that directory, whose files
// stand for the files below the link.
type metLink struct {
	path  string
	cycle bool
	back  string
}

// linkWalk finds links at any depth. It follows a link to a directory to
// find links and files below it, never to read them, and walks no directory
// twice in one chain, so a cycle ends.
type linkWalk struct {
	globs  []string          // every glob of every list
	inside map[string]string // real path of each directory the walk is in -> its path
	links  []metLink
	files  []string // every regular file met, by the path the runner sees
}

// Links lists the symbolic links the scan does not follow and the test
// runner may, at any depth (section 8, "Symbolic links"): a link whose own
// path a glob of trace.code (inside scope) or trace.tests matches, and a link
// to a directory with a file found below it that would match as
// <link>/<path>. A link to something else, a dangling link and a link that
// leaves the project are found the same way. Nothing under a directory the
// scan never walks is found, nor a link of such a name. Sorted by path.
func Links(p *project.Project) ([]Link, error) {
	if p.Cfg.Trace == nil {
		return nil, nil
	}
	lists := []traceList{{"trace.code", p.Cfg.Trace.Code, true}, {"trace.tests", p.Cfg.Trace.Tests, false}}
	root, err := filepath.EvalSymlinks(p.Root)
	if err != nil {
		return nil, err
	}
	w := &linkWalk{inside: map[string]string{}}
	for _, l := range lists {
		w.globs = append(w.globs, l.globs...)
	}
	if err := w.walk(root, "", true); err != nil {
		return nil, err
	}
	sort.Strings(w.files)
	var res []Link
	for _, l := range w.links {
		found := Link{Path: l.path}
		for _, list := range lists {
			if w.reaches(p, l, list) {
				found.Lists = append(found.Lists, list.name)
			}
		}
		if len(found.Lists) > 0 {
			res = append(res, found)
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].Path < res[j].Path })
	return res, nil
}

// walk lists dir, named rel. In the tree (not behind a link) a directory
// that cannot be read is an error, as it is for the scan; behind a link it
// shows nothing below the link.
func (w *linkWalk) walk(dir, rel string, tree bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if tree {
			return err
		}
		return nil
	}
	w.inside[dir] = rel
	defer delete(w.inside, dir)
	for _, e := range entries {
		link := e.Type()&fs.ModeSymlink != 0
		if (link || e.IsDir()) && project.SkipDir(e.Name()) {
			continue // never walked by the scan, so never a finding
		}
		abs, sub := filepath.Join(dir, e.Name()), join(rel, e.Name())
		switch {
		case link:
			if err := w.link(abs, sub); err != nil {
				return err
			}
		case e.IsDir():
			if err := w.walk(abs, sub, tree); err != nil {
				return err
			}
		case e.Type().IsRegular():
			w.files = append(w.files, sub)
		}
	}
	return nil
}

// link records a link and, when it leads to a directory a glob can reach
// below, walks that directory under the name of the link.
func (w *linkWalk) link(abs, rel string) error {
	w.links = append(w.links, metLink{path: rel})
	if !below(w.globs, rel) {
		return nil
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return nil
	}
	target, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil
	}
	if at, ok := w.inside[target]; ok {
		w.links[len(w.links)-1] = metLink{path: rel, cycle: true, back: at}
		return nil
	}
	return w.walk(target, rel, false)
}

// reaches reports whether a list matches the link itself or a file found
// below it, as named below the link; below a link that closes a cycle, the
// files below the directory it leads back to.
func (w *linkWalk) reaches(p *project.Project, l metLink, list traceList) bool {
	ok := func(f string) bool { return project.Match(list.globs, f) && (!list.inScope || p.InScope(f)) }
	if ok(l.path) {
		return true
	}
	from := l.path
	if l.cycle {
		from = l.back
	}
	pre := join(from, "")
	for i := sort.SearchStrings(w.files, pre); i < len(w.files) && strings.HasPrefix(w.files[i], pre); i++ {
		if ok(l.path + "/" + w.files[i][len(pre):]) {
			return true
		}
	}
	return false
}

// join names an entry of a directory named rel ("" for the root).
func join(rel, name string) string {
	if rel == "" {
		return name
	}
	return rel + "/" + name
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
