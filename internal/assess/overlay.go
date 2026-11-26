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
// adopt --rehearse: design-side files laid over the snapshot before the run.

// OverlayFile is one file of an overlay.
type OverlayFile struct {
	Path   string // relative to the overlay and to the snapshot, forward slashes
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
// that is not regular and a path under .git.
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

// Config returns the machinery.yaml of the overlay, or nil.
func (o *Overlay) Config() []byte {
	if o == nil {
		return nil
	}
	for _, f := range o.Files {
		if f.Path == project.ConfigName {
			return f.data
		}
	}
	return nil
}

// others lists the files of the overlay but the config, which is the config
// of the run and is not laid over the snapshot.
func (o *Overlay) others() []OverlayFile {
	var res []OverlayFile
	for _, f := range o.Files {
		if f.Path != project.ConfigName {
			res = append(res, f)
		}
	}
	return res
}

// Check refuses what an overlay may not hold, before anything is written
// (section 18.2). The judge is the machinery.yaml of HEAD when HEAD has one,
// else external (the --config file, labelled externalLabel, nil for none):
// never a config of the overlay. A file of the overlay must be a design
// source the judge names, spelled as HEAD spells it, in ASCII, not at or
// below a symbolic link of HEAD, and matched by no code or test glob of the
// judge or of run, the config of the run (labelled runLabel, nil for none).
func (o *Overlay) Check(root string, external *project.Config, externalLabel string, run *project.Config, runLabel string) error {
	if o == nil {
		return nil
	}
	judge, label := external, externalLabel
	var atHead *project.Config
	if data, ok := gitx.Show(root, "HEAD", project.ConfigName); ok {
		cfg, err := project.Parse(data)
		if err != nil {
			return project.Errf("%s at HEAD, which judges the overlay: %v", project.ConfigName, err)
		}
		atHead, judge, label = cfg, cfg, project.ConfigName+" at HEAD"
	}
	if o.Config() != nil && run != nil {
		// a waiver is no overlay source: an overlaid config carries only those HEAD has
		has := map[project.Waiver]bool{}
		if atHead != nil {
			for _, w := range atHead.Waivers {
				has[project.Waiver{Rule: w.Rule, ID: w.ID}] = true
			}
		}
		for _, w := range run.Waivers {
			if !has[project.Waiver{Rule: w.Rule, ID: w.ID}] {
				return o.refuse("%s carries the waiver {rule: %s, id: %q} that HEAD does not have; an overlay never adds a waiver (commit it, or pass the config with --config)", project.ConfigName, w.Rule, w.ID)
			}
		}
	}
	others := o.others()
	if len(others) == 0 {
		return nil
	}
	if judge == nil {
		return usagef("--overlay %s needs a config to be judged by (the %s of HEAD, or --config): it says which files are design sources, and a config in the overlay never does", o.Dir, project.ConfigName)
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
	for _, f := range others {
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
		if !designSource(judge, f.Path) {
			return o.refuse("%s is not a design source of %s: an overlay holds only the config and the design sources it names (design.risk, design.status, import.failure_modes.links)", f.Path, label)
		}
		for _, c := range []struct {
			cfg   *project.Config
			label string
		}{{judge, label}, {run, runLabel}} {
			if key := evidenceKey(c.cfg, f.Path); key != "" {
				return o.refuse("%s is under %s of %s; an overlay never replaces what is read as code or tests", f.Path, key, c.label)
			}
		}
	}
	return nil
}

// designSource reports whether cfg names rel as a design source an overlay
// may hold: design.risk, design.status or import.failure_modes.links. The
// spelling must be the config's. Models, machines, the failure-mode file,
// facts and the owed list are not: an overlay never changes what is owed.
func designSource(cfg *project.Config, rel string) bool {
	named := []string{cfg.Design.Risk, cfg.Design.Status}
	if im := cfg.ImportedFailureModes(); im != nil {
		named = append(named, im.Links)
	}
	for _, n := range named {
		if n != "" && path.Clean(n) == rel {
			return true
		}
	}
	return false
}

// evidenceKey names the first key of cfg whose globs match rel, compared as
// written and with case folded, whatever the scope: the globs under which
// files are read as code or as tests. "" when none does, or cfg is nil.
func evidenceKey(cfg *project.Config, rel string) string {
	if cfg == nil {
		return ""
	}
	type set struct {
		key   string
		globs []string
	}
	var sets []set
	if t := cfg.Trace; t != nil {
		sets = append(sets, set{"trace.code", t.Code}, set{"trace.tests", t.Tests})
	}
	if t := cfg.Tests; t != nil {
		sets = append(sets, set{"tests.e2e", t.E2E}, set{"tests.files", t.Files})
	}
	if c := cfg.Constraints; c != nil {
		for _, l := range c.Layers {
			sets = append(sets, set{"constraints.layers", l.Paths})
		}
		if c.Size != nil {
			sets = append(sets, set{"constraints.size.paths", c.Size.Paths})
		}
	}
	for _, s := range sets {
		folded := make([]string, len(s.globs))
		for i, g := range s.globs {
			folded[i] = foldPath(g)
		}
		if project.Match(s.globs, rel) || project.Match(folded, foldPath(rel)) {
			return s.key
		}
	}
	return ""
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

// Paths lists the overlaid files.
func (o *Overlay) Paths() []string {
	var res []string
	if o != nil {
		for _, f := range o.Files {
			res = append(res, f.Path)
		}
	}
	return res
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
	return o.Summary() + ": " + strings.Join(o.Paths(), ", ")
}

// lay writes the files over the snapshot through a root bound to it. The
// config is not laid: TakeSnapshot writes the config of the run. A path that
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
		if f.Path == project.ConfigName {
			continue
		}
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
