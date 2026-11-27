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

	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
	"github.com/RamXX/machinery/v2/internal/stages"
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
	adoption []byte // the project's design/adoption.yaml, nil when it has none
	cfgLabel string
	name     string
	head     string // short sha
	since    []sinceRef
	changed  int
	untrack  int
	outside  []string         // symbolic links of HEAD that leave the project, not in the snapshot
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
	if line := a.outsideLine(); line != "" {
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

// outsideLine names the symbolic links left out of the snapshot (section 18.2).
func (a *assessment) outsideLine() string {
	if len(a.outside) == 0 {
		return ""
	}
	return fmt.Sprintf("outside links: %d symbolic link(s) leave the project and are not in the snapshot: %s", len(a.outside), strings.Join(a.outside, ", "))
}

// inputs resolves the project, the config, the --since refs and the out
// directory, and refuses before anything is written (section 18.1).
func (a *assessment) inputs() error {
	o := a.o
	root, err := ResolveRoot(o.ProjectFlag)
	if err != nil {
		return err
	}
	a.root = root
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
	// the project's design/adoption.yaml goes with either config (section 18.1)
	if a.adoption, err = project.ReadAdoption(a.root); err != nil {
		return err
	}
	if _, perr := project.ParseWith(a.cfgBytes, a.adoption); perr != nil {
		return perr
	}

	if a.head, err = HeadOf(a.root, "assess"); err != nil {
		return err
	}
	for _, ref := range o.Since {
		sha, rerr := gitx.Resolve(a.root, ref)
		if rerr != nil || sha == "" {
			return usagef("--since %s: not a commit of the project", ref)
		}
		a.since = append(a.since, sinceRef{label: ref, sha: sha})
	}
	a.out, err = PrepareOut(a.root, a.o.Out, "assess")
	return err
}

// snapshot extracts HEAD into <out>/snapshot, writes the config there and
// makes it a repository with one commit (section 18.2).
func (a *assessment) snapshot() error {
	var err error
	if a.snap, a.outside, err = TakeSnapshot(a.root, a.out, a.head, a.cfgBytes, a.adoption); err != nil {
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

func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}
