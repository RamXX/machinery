// Package assess reports, read-only, where an existing project stands
// against its own design (docs/formats.md section 18). It writes only below
// its out directory: the project, its .git directory and its tracker are
// read and never written.
package assess

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
)

// Options are the arguments of machinery assess.
type Options struct {
	ProjectFlag string   // -C
	Config      string   // --config
	Out         string   // --out
	Since       []string // --since
	Stdout      io.Writer
	// Gen writes the generated files (machinery gen). It refuses, returning
	// the blocking stages, when the design has errors.
	Gen func(*stages.Context) (written []string, blocked []*report.Stage, err error)
}

// UsageError is a refusal of section 18.1 that is not a config error. It
// exits 2, like a config error.
type UsageError struct{ Msg string }

func (e *UsageError) Error() string { return e.Msg }

func usagef(format string, a ...any) error { return &UsageError{fmt.Sprintf(format, a...)} }

// sinceRef is one comparison of the history section.
type sinceRef struct {
	label string // the ref as given, or "first commit of <model>"
	sha   string
}

type assessment struct {
	o        Options
	root     string // the project, absolute and resolved
	out      string // absolute
	snap     string
	cfgBytes []byte
	cfgLabel string
	name     string
	head     string // short sha
	since    []sinceRef
	changed  int
	untrack  int
	p        *project.Project // the snapshot
	ctx      *stages.Context
	sections []*section
}

// Run assesses the project. A *UsageError or a config error means nothing
// was written; any other error means the snapshot could not be taken.
func Run(o Options) error {
	// no git command assess runs may take an optional lock: git status would
	// otherwise refresh the index of the project
	os.Setenv("GIT_OPTIONAL_LOCKS", "0")
	a := &assessment{o: o}
	if err := a.inputs(); err != nil {
		return err
	}
	w := o.Stdout
	fmt.Fprintf(w, "out: %s\n", a.out)
	if err := a.snapshot(); err != nil {
		return fmt.Errorf("assess: cannot take the snapshot: %v", err)
	}
	fmt.Fprintf(w, "project: %s at %s\n", a.name, a.head)
	if line := a.dirtyLine(); line != "" {
		fmt.Fprintln(w, line)
	}
	for _, def := range sectionDefs {
		s := a.run(def)
		a.sections = append(a.sections, s)
		fmt.Fprintln(w, s.summaryLine())
	}
	if err := a.writeReports(); err != nil {
		return fmt.Errorf("assess: cannot write the report: %v", err)
	}
	fmt.Fprintln(w, a.totalLine())
	return nil
}

func (a *assessment) dirtyLine() string {
	if a.changed == 0 && a.untrack == 0 {
		return ""
	}
	return fmt.Sprintf("dirty: %d changed, %d untracked file(s) are not assessed", a.changed, a.untrack)
}

// inputs resolves the project, the config, the --since refs and the out
// directory, and refuses before anything is written (section 18.1).
func (a *assessment) inputs() error {
	o := a.o
	root := o.ProjectFlag
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		if r, ferr := project.Find(wd); ferr == nil {
			root = r
		} else if top, terr := gitx.TopLevel(wd); terr == nil {
			root = top
		} else {
			root = wd
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if info, serr := os.Stat(abs); serr != nil || !info.IsDir() {
		return usagef("%s is not a directory", root)
	}
	a.root = resolve(abs)
	a.name = filepath.Base(a.root)

	if o.Config != "" {
		data, rerr := os.ReadFile(o.Config)
		if rerr != nil {
			return project.Errf("--config %s: cannot read: %v", o.Config, shortErr(rerr))
		}
		a.cfgBytes, a.cfgLabel = data, "--config "+filepath.Base(o.Config)
	} else {
		data, rerr := os.ReadFile(filepath.Join(a.root, project.ConfigName))
		if rerr != nil {
			return project.Errf("no %s in %s and no --config: write one by hand (docs/brownfield.md, step 1) and pass it with --config", project.ConfigName, a.root)
		}
		a.cfgBytes, a.cfgLabel = data, project.ConfigName
	}
	if _, perr := project.Parse(a.cfgBytes); perr != nil {
		return perr
	}

	head, herr := gitx.Head(a.root)
	if herr != nil || head == "" {
		return usagef("%s is not a git repository with a commit; assess reads its HEAD", a.root)
	}
	a.head = gitx.Short(head)
	for _, ref := range o.Since {
		sha, rerr := gitx.Resolve(a.root, ref)
		if rerr != nil || sha == "" {
			return usagef("--since %s: not a commit of the project", ref)
		}
		a.since = append(a.since, sinceRef{label: ref, sha: sha})
	}
	return a.outDir()
}

// outDir checks --out, or makes the default one under the system temp
// directory, outside the project and the repository that holds it.
func (a *assessment) outDir() error {
	containers := []string{a.root}
	if top, err := gitx.TopLevel(a.root); err == nil && top != "" {
		containers = append(containers, resolve(top))
	}
	if gd, err := gitx.CommonDir(a.root); err == nil {
		containers = append(containers, resolve(gd))
	}
	inside := func(p string) string {
		rp := resolve(p)
		for _, c := range containers {
			if within(rp, c) {
				return c
			}
		}
		return ""
	}
	if a.o.Out == "" {
		tmp := os.TempDir()
		if c := inside(tmp); c != "" {
			return usagef("the system temp directory %s is inside %s; assess never writes in the project: name a directory outside it with --out", tmp, c)
		}
		dir, err := os.MkdirTemp("", "machinery-assess-")
		if err != nil {
			return err
		}
		a.out = dir
		return nil
	}
	abs, err := filepath.Abs(a.o.Out)
	if err != nil {
		return err
	}
	if c := inside(abs); c != "" {
		return usagef("--out %s is inside %s; assess never writes in the project: name a directory outside it", a.o.Out, c)
	}
	if info, serr := os.Lstat(abs); serr == nil {
		entries, rerr := os.ReadDir(abs)
		if !info.IsDir() || rerr != nil || len(entries) > 0 {
			return usagef("--out %s is not empty; name an absent or empty directory", a.o.Out)
		}
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return err
	}
	a.out = abs
	return nil
}

// snapshot extracts HEAD into <out>/snapshot, writes the config there and
// makes it a repository with one commit (section 18.2).
func (a *assessment) snapshot() error {
	a.snap = filepath.Join(a.out, "snapshot")
	if err := os.MkdirAll(a.snap, 0o755); err != nil {
		return err
	}
	if _, err := gitx.Archive(a.root, a.snap); err != nil {
		return err
	}
	cfg := filepath.Join(a.snap, project.ConfigName)
	if err := os.RemoveAll(cfg); err != nil {
		return err
	}
	if err := os.WriteFile(cfg, a.cfgBytes, 0o644); err != nil {
		return err
	}
	if err := gitx.InitCommit(a.snap, "snapshot of "+a.head); err != nil {
		return err
	}
	p, err := project.Load(a.snap)
	if err != nil {
		return err
	}
	if d := p.Cfg.Tools.Dir; d != "" && !filepath.IsAbs(d) {
		p.Cfg.Tools.Dir = filepath.Join(a.root, filepath.FromSlash(d))
	}
	a.p = p
	a.ctx = &stages.Context{P: p}
	a.changed, a.untrack, err = gitx.StatusCounts(a.root)
	return err
}

// resolve returns an absolute path with the symbolic links of its longest
// existing prefix resolved.
func resolve(p string) string {
	cur, rest := filepath.Clean(p), ""
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return filepath.Clean(p)
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// within reports whether path is dir or below it.
func within(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}
