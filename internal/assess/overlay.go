package assess

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
)

// The overlay of section 18.2, shared by machinery assess and machinery
// adopt --rehearse: links, status and risk files laid over the snapshot before the run.

// OverlayFile is one file of an overlay.
type OverlayFile struct {
	Path   string // relative to the overlay and to the snapshot, forward slashes
	Role   string // the key under which the config of the run names it, set by Check
	SHA256 string
	data   []byte
	exec   bool
}

// Overlay holds the files of --overlay, read once, sorted by path.
type Overlay struct {
	Dir   string // as given
	Files []OverlayFile
}

// ReadOverlay reads the files of dir. An empty dir is no overlay (nil). It
// refuses a directory that is none or holds no file, a symbolic link, a file
// that is not regular, a path under .git and the config.
func ReadOverlay(dir string) (*Overlay, error) {
	if dir == "" {
		return nil, nil
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, usagef("--overlay %s is not a directory", dir)
	}
	rt, err := os.OpenRoot(dir)
	if err != nil {
		return nil, usagef("--overlay %s is not a directory", dir)
	}
	defer rt.Close()
	o := &Overlay{Dir: dir}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, rerr := filepath.Rel(dir, p)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			return nil
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			return o.refuse("%s is a symbolic link; an overlay holds regular files only", rel)
		case !d.Type().IsRegular():
			return o.refuse("%s is not a regular file; an overlay holds regular files only", rel)
		case strings.EqualFold(strings.SplitN(rel, "/", 2)[0], ".git"):
			return o.refuse("%s: an overlay never writes under .git", rel)
		case strings.EqualFold(rel, project.ConfigName):
			return o.refuse("%s: an overlay never holds the config; the config of the run is the %s of the project or --config", rel, project.ConfigName)
		}
		if cerr := project.CheckPath("--overlay "+dir, rel); cerr != nil {
			return usagef("%s", cerr.Error())
		}
		// read through the root: a link swapped in after the walk is not followed out of it
		info, serr := rt.Lstat(rel)
		if serr != nil || !info.Mode().IsRegular() {
			return o.refuse("%s is not a regular file; an overlay holds regular files only", rel)
		}
		data, derr := rt.ReadFile(rel)
		if derr != nil {
			return o.refuse("%s: cannot read: %s", rel, shortErr(derr))
		}
		sum := sha256.Sum256(data)
		o.Files = append(o.Files, OverlayFile{Path: rel, SHA256: hex.EncodeToString(sum[:]), data: data, exec: info.Mode()&0o111 != 0})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(o.Files) == 0 {
		return nil, usagef("--overlay %s holds no file", dir)
	}
	sort.Slice(o.Files, func(i, j int) bool { return o.Files[i].Path < o.Files[j].Path })
	return o, nil
}

func (o *Overlay) refuse(format string, a ...any) error {
	return usagef("--overlay %s: %s", o.Dir, fmt.Sprintf(format, a...))
}

// The three roles an overlaid file may have: the keys of the config of the
// run under which an overlay source is named (section 18.2).
var overlayRoles = map[string]bool{"import.failure_modes.links": true, "design.status": true, "design.risk": true}

// Check refuses what an overlay may not hold, before anything is written
// (section 18.2), and gives each file its role. cfg is the config of the run,
// labelled label; nil when the run has none. A file must be named by cfg
// under exactly one key, one of the three roles; be spelled as HEAD spells
// it, in ASCII; and not lie at or below a symbolic link of HEAD.
func (o *Overlay) Check(root string, cfg *project.Config, label string) error {
	if o == nil {
		return nil
	}
	if cfg == nil {
		return usagef("--overlay %s needs a config (the %s of the project, or --config): it names the links, status and risk files an overlay may hold", o.Dir, project.ConfigName)
	}
	paths, err := gitx.HeadPaths(root)
	if err != nil {
		return err
	}
	links, err := gitx.HeadLinks(root)
	if err != nil {
		return err
	}
	head := map[string]string{} // folded path or directory of HEAD -> as HEAD spells it
	for _, p := range paths {
		for _, pre := range prefixes(p) {
			head[foldPath(pre)] = pre
		}
	}
	for i := range o.Files {
		f := &o.Files[i]
		if !isASCII(f.Path) {
			return o.refuse("%s is not written in ASCII; an overlay path holds ASCII characters only", f.Path)
		}
		for _, pre := range prefixes(f.Path) {
			if spelled, ok := head[foldPath(pre)]; ok && spelled != pre {
				return o.refuse("%s differs from %s of HEAD only by case; an overlay path is written as HEAD has it", f.Path, spelled)
			}
		}
		for _, l := range links {
			if f.Path == l || strings.HasPrefix(f.Path, l+"/") {
				return o.refuse("%s: %s is a symbolic link at HEAD; an overlay never writes through or over a link", f.Path, l)
			}
		}
		role, other := "", ""
		for _, k := range keysNaming(cfg, f.Path) {
			switch {
			case role == "" && overlayRoles[k]:
				role = k
			case other == "":
				other = k
			}
		}
		switch {
		case role == "":
			return o.refuse("%s is not the links file (import.failure_modes.links), the status file (design.status) or the risk file (design.risk) of %s; an overlay holds nothing else", f.Path, label)
		case other != "":
			return o.refuse("%s is %s and also %s in %s; an overlaid file has exactly one role", f.Path, role, other, label)
		}
		f.Role = role
	}
	return nil
}

// keysNaming lists, in a fixed order, the keys of cfg that name rel: the
// paths and directories under design and import, and the globs of import,
// trace, tests and constraints, whatever the scope. A path is compared as
// written; a glob as written and with case folded.
func keysNaming(cfg *project.Config, rel string) []string {
	var res []string
	file := func(key, p string) {
		if p != "" && path.Clean(p) == rel {
			res = append(res, key)
		}
	}
	dir := func(key, p string) {
		if p != "" && strings.HasPrefix(rel, path.Clean(p)+"/") {
			res = append(res, key)
		}
	}
	globs := func(key string, gs []string) {
		folded := make([]string, len(gs))
		for i, g := range gs {
			folded[i] = foldPath(g)
		}
		if project.Match(gs, rel) || project.Match(folded, foldPath(rel)) {
			res = append(res, key)
		}
	}
	d := cfg.Design
	for _, m := range d.Models {
		file("design.models", m)
	}
	dir("design.machines", d.Machines)
	file("design.failure_modes", d.FailureModes)
	file("design.risk", d.Risk)
	file("design.status", d.Status)
	dir("design.formal", d.Formal)
	file("design.facts", d.Facts)
	file("design.owed", d.Owed)
	dir("design.gen", d.Gen)
	if im := cfg.ImportedMachines(); im != nil {
		globs("import.machines.files", im.Files)
		file("import.machines.register.file", im.RegisterFile())
	}
	if im := cfg.ImportedFailureModes(); im != nil {
		file("import.failure_modes.links", im.Links)
		file("import.failure_modes.file", im.File)
	}
	if t := cfg.Trace; t != nil {
		globs("trace.code", t.Code)
		globs("trace.tests", t.Tests)
	}
	if t := cfg.Tests; t != nil {
		globs("tests.e2e", t.E2E)
		globs("tests.files", t.Files)
		file("tests.baseline", t.Baseline)
	}
	if c := cfg.Constraints; c != nil {
		for _, l := range c.Layers {
			globs("constraints.layers", l.Paths)
		}
		if c.Size != nil {
			globs("constraints.size.paths", c.Size.Paths)
		}
		file("constraints.baseline", c.Baseline)
	}
	return res
}

// prefixes lists a path and every directory above it, shortest first.
func prefixes(rel string) []string {
	segs := strings.Split(rel, "/")
	res := make([]string, 0, len(segs))
	for i := range segs {
		res = append(res, strings.Join(segs[:i+1], "/"))
	}
	return res
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// asciiTwins are the characters whose canonical form is one ASCII character:
// a file system that normalises names takes each for its twin.
var asciiTwins = map[rune]rune{'\u212a': 'K', '\u037e': ';', '\u1fef': '`'}

// foldPath is the key two paths share when a file system that folds case and
// normalises names would take them for one, given that one of them is ASCII:
// every character becomes the smallest of its case orbit.
func foldPath(p string) string {
	return strings.Map(func(r rune) rune {
		if t, ok := asciiTwins[r]; ok {
			r = t
		}
		least := r
		for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
			if f < least {
				least = f
			}
		}
		return least
	}, p)
}

// Summary says how many files were laid over the snapshot, or "".
func (o *Overlay) Summary() string {
	if o == nil {
		return ""
	}
	return fmt.Sprintf("overlay: %d file(s) laid over the snapshot; this is not the result of HEAD", len(o.Files))
}

// Line is the stdout line of an overlay: the summary and the paths, or "".
func (o *Overlay) Line() string {
	if o == nil {
		return ""
	}
	var named []string
	for _, f := range o.Files {
		named = append(named, fmt.Sprintf("%s (%s)", f.Path, f.Role))
	}
	return o.Summary() + ": " + strings.Join(named, ", ")
}

// lay writes the files over the snapshot through a root bound to it. A path that
// meets a symbolic link or a file where a directory is needed stops it.
func (o *Overlay) lay(snap string) error {
	if o == nil {
		return nil
	}
	rt, err := os.OpenRoot(snap)
	if err != nil {
		return err
	}
	defer rt.Close()
	for _, f := range o.Files {
		segs := strings.Split(f.Path, "/")
		for i := range segs {
			at := path.Join(segs[:i+1]...)
			info, serr := rt.Lstat(at)
			if serr != nil {
				break // nothing there yet: created below
			}
			last := i == len(segs)-1
			switch {
			case info.Mode()&fs.ModeSymlink != 0:
				return fmt.Errorf("overlay: %s: %s is a symbolic link in the snapshot; nothing is written through or over a link", f.Path, at)
			case !last && !info.IsDir():
				return fmt.Errorf("overlay: %s: %s is a file in the snapshot, not a directory", f.Path, at)
			case last && !info.Mode().IsRegular():
				return fmt.Errorf("overlay: %s is not a regular file in the snapshot", f.Path)
			}
		}
		if err := rt.MkdirAll(path.Dir(f.Path), 0o755); err != nil {
			return err
		}
		if err := rt.Remove(f.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
		mode := os.FileMode(0o644)
		if f.exec {
			mode = 0o755
		}
		file, err := rt.OpenFile(f.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, werr := file.Write(f.data)
		if cerr := file.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
	}
	return nil
}
