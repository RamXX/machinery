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

// Check refuses what an overlay may not hold, before anything is written: a
// file stage trace would read for tags under cfg (trace.tests, or trace.code
// inside scope), and a path at or below a symbolic link of HEAD. cfg is the
// config of the run, nil when it has none.
func (o *Overlay) Check(root string, cfg *project.Config) error {
	if o == nil {
		return nil
	}
	if cfg == nil {
		return usagef("--overlay %s needs a config (--config, the machinery.yaml of the project or one in the overlay): trace.code and trace.tests say what an overlay may not replace", o.Dir)
	}
	links, err := gitx.HeadLinks(root)
	if err != nil {
		return err
	}
	const why = "an overlay is for design-side files (links, status, risk, config), never for code or tests"
	for _, f := range o.Files {
		if f.Path == project.ConfigName {
			continue // the config of the run, not a file laid over the snapshot
		}
		if t := cfg.Trace; t != nil {
			switch {
			case project.Match(t.Tests, f.Path):
				return o.refuse("%s is under trace.tests; %s", f.Path, why)
			case project.Match(t.Code, f.Path) && project.Match(cfg.Scope, f.Path):
				return o.refuse("%s is under trace.code; %s", f.Path, why)
			}
		}
		for _, l := range links {
			if f.Path == l || strings.HasPrefix(f.Path, l+"/") {
				return o.refuse("%s: %s is a symbolic link at HEAD; an overlay never writes through or over a link", f.Path, l)
			}
		}
	}
	return nil
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
