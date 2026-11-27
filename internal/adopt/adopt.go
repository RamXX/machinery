// Package adopt brings an existing project under machinery without touching
// its code (docs/formats.md section 17.2).
package adopt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/baseline"
	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/gitx"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
	"github.com/RamXX/machinery/v2/internal/stages"
	"github.com/RamXX/machinery/v2/internal/tags"
)

// Options are the arguments of machinery adopt.
type Options struct {
	Root  string   // the directory to adopt
	Paths []string // --paths
	E2E   []string // --e2e
	Write bool     // --write
	// NoBacklog never runs the backlog tool (a rehearsal, section 17.3).
	NoBacklog bool
	// SkipBacklog never runs the backlog tool either: --no-backlog or
	// MACHINERY_NO_BACKLOG=1 (section 15.1).
	SkipBacklog bool
	// Commit is the short sha the report and every reason name; empty, the
	// HEAD of Root. A rehearsal gives the HEAD of the project it snapshotted,
	// never the snapshot's own commit (section 17.3).
	Commit string
	// Gen writes the generated files (machinery gen). It refuses, returning
	// the blocking stages, when the design has errors. A rendering beside a
	// model that may not be written is left as it is, and returned in left.
	Gen GenFunc
}

// GenFunc generates the design, writing only the renderings may allows.
type GenFunc func(ctx *stages.Context, may func(rel string) bool) (written []string, left []Left, blocked []*report.Stage, err error)

// Change is one file adopt would write, and what it would put in it.
type Change struct{ Path, What string }

// Left is a rendering beside a model that generation would write and adopt
// leaves to the project (section 17.2).
type Left struct {
	Path    string
	Missing bool
}

// Line is what the report says of it.
func (l Left) Line() string {
	if l.Missing {
		return l.Path + " is missing; the project generates it"
	}
	return l.Path + " is stale; the project regenerates it"
}

// Result is what adopt found and did.
type Result struct {
	Name      string
	Sha       string
	Scope     []string
	Languages []string
	Counts    counts
	Models    int
	Machines  int
	Contracts int
	Backlog   Backlog
	Would     []Change // dry run
	Written   []string // --write, sorted
	Left      []Left   // renderings left to the project, sorted
	Blocked   []*report.Stage
	Next      []string
	Write     bool
}

// Run plans, and with Write applies, the adoption of the project at o.Root.
func Run(o Options) (*Result, error) {
	root, err := filepath.Abs(o.Root)
	if err != nil {
		return nil, err
	}
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	for _, g := range append(append([]string{}, o.Paths...), o.E2E...) {
		if err := project.CheckPath("glob", g); err != nil {
			return nil, err
		}
		if !project.ValidGlob(g) {
			return nil, project.Errf("%q is not a valid glob", g)
		}
	}
	// machinery.yaml is read through the YAML decoder, as every command reads
	// it, and never written when it exists (section 17.2)
	cfgPath := filepath.Join(root, project.ConfigName)
	_, serr := os.Lstat(cfgPath)
	existing := serr == nil
	adoption, err := project.ReadAdoption(root)
	if err != nil {
		return nil, err
	}
	pl := &plan{}
	var cfgBytes []byte
	var old project.Adoption
	var oldP *project.Project
	switch {
	case existing:
		if oldP, err = project.Load(root); err != nil {
			return nil, err
		}
		if lines := design.OwedRefusal(oldP); len(lines) > 0 {
			return nil, fmt.Errorf("%s", strings.Join(lines, "\n"))
		}
		if cfgBytes, err = os.ReadFile(cfgPath); err != nil {
			return nil, project.Errf("%v", err)
		}
		raw, rerr := project.Raw(cfgBytes, adoption)
		if rerr != nil {
			return nil, rerr
		}
		pl.view = *raw
		if adoption != nil {
			a, perr := project.ParseAdoption(adoption)
			if perr != nil {
				return nil, perr
			}
			old = *a
		}
	case adoption != nil:
		return nil, project.Errf("%s exists and %s does not: it holds what adopt added to a %s; restore that file or remove this one",
			project.AdoptionName, project.ConfigName, project.ConfigName)
	}
	base := &project.Project{Root: root}
	files, err := base.Files()
	if err != nil {
		return nil, err
	}
	genDir := "design/gen"
	if oldP != nil {
		genDir = oldP.Cfg.Design.Gen
	}
	exe := "bd"
	if oldP != nil {
		exe = oldP.Cfg.Tools.BD
	}
	res := &Result{Name: filepath.Base(root), Write: o.Write, Sha: "no commit", Backlog: readBacklog(root, exe, o.NoBacklog, o.SkipBacklog)}
	head := o.Commit
	if h, err := gitx.HeadIfAny(root); err != nil {
		return nil, err
	} else if head == "" {
		head = gitx.Short(h)
	}
	if head != "" {
		res.Sha = head
	}

	pl.step1(existing, o, discover(files, genDir), res.Backlog.Found)
	cfg, data, err := pl.config(existing, cfgBytes, adoption, &old)
	if err != nil {
		return nil, err
	}
	p := &project.Project{Root: root, Cfg: *cfg}

	// step 4: the failure states and events the design already has are
	// unmodeled, not new; waive them so day one passes
	// and step 5: what the design already has that binds to nothing is
	// unbound, not new
	sha := head
	if sha == "" {
		sha = "no commit"
	}
	var waiverNotes []string
	for _, step := range []struct {
		gaps   func(*stages.Context) []project.Waiver
		reason string
		note   string
	}{
		{stages.UnmodeledFailures, "unmodeled at adoption", "new waivers: %d"},
		{stages.UnboundDesign, "unbound at adoption", "unbound waivers: %d"},
	} {
		gaps := step.gaps(&stages.Context{P: p})
		if len(gaps) == 0 {
			continue
		}
		pl.waive(gaps, step.reason+" ("+sha+")", step.note)
		waiverNotes = append(waiverNotes, fmt.Sprintf(step.note, len(gaps)))
		if cfg, data, err = pl.config(existing, cfgBytes, adoption, &old); err != nil {
			return nil, err
		}
		p = &project.Project{Root: root, Cfg: *cfg}
	}

	// a planned mark that names no story id is refused before anything is written
	var planned map[string]string
	if len(p.Cfg.ImportedPlanned()) > 0 {
		var problems []string
		if planned, problems = design.Load(p).PlannedMarks(); len(problems) > 0 {
			return nil, fmt.Errorf("%s", strings.Join(problems, "\n"))
		}
	}
	if err := describe(res, p, files); err != nil {
		return nil, err
	}
	w := &worker{planned: planned, o: o, res: res, oldP: oldP, p: p, head: head, written: map[string]bool{}}
	switch {
	case data == nil:
	case existing:
		if err := w.adoption(data, "adds: "+strings.Join(pl.notes, "; ")); err != nil {
			return nil, err
		}
	default:
		what := "new config"
		if len(waiverNotes) > 0 {
			what += "; " + strings.Join(waiverNotes, "; ")
		}
		if err := w.newConfig(data, what); err != nil {
			return nil, err
		}
	}
	if err := w.baselines(); err != nil {
		return nil, err
	}
	if err := w.owed(); err != nil {
		return nil, err
	}
	res.Next = next(res, o)
	for f := range w.written {
		res.Written = append(res.Written, f)
	}
	sort.Strings(res.Written)
	sort.Slice(res.Would, func(i, j int) bool { return res.Would[i].Path < res.Would[j].Path })
	return res, nil
}

// Describe returns the facts of the adopt report for a configured project,
// planning nothing (section 18.3: the inventory of machinery assess).
func Describe(p *project.Project) (*Result, error) {
	files, err := p.Files()
	if err != nil {
		return nil, err
	}
	res := &Result{Name: filepath.Base(p.Root)}
	if err := describe(res, p, files); err != nil {
		return nil, err
	}
	return res, nil
}

// describe fills the facts of the report from the configuration that adoption
// leaves behind.
func describe(res *Result, p *project.Project, files []string) error {
	res.Scope = p.Cfg.Scope
	res.Languages = languages(p, files)
	c, err := count(p, files)
	if err != nil {
		return err
	}
	res.Counts = c
	res.Models = len(p.Cfg.Design.Models)
	res.Contracts = len(p.Cfg.Contracts)
	if dir := p.Cfg.Design.Machines; dir != "" {
		ms, _ := p.Glob([]string{strings.TrimSuffix(dir, "/") + "/*.machine.json"})
		res.Machines = len(ms)
	}
	if im := p.Cfg.ImportedMachines(); im != nil {
		gen := strings.TrimSuffix(p.Cfg.Design.Gen, "/") + "/"
		ms, _ := p.Glob(im.Files)
		for _, f := range ms {
			if !strings.HasPrefix(f, gen) && f != im.RegisterFile() {
				res.Machines++
			}
		}
	}
	return nil
}

// worker applies steps 1 to 3 of section 17.2 or, without Write, records
// what they would do.
type worker struct {
	o       Options
	res     *Result
	oldP, p *project.Project
	head    string
	written map[string]bool
	planned map[string]string // the project's planned marks: id to story
}

func (w *worker) record(rel, what string) {
	if w.o.Write {
		w.written[rel] = true
		return
	}
	w.res.Would = append(w.res.Would, Change{rel, what})
}

// newConfig creates machinery.yaml. It never replaces one: the file is
// opened to be created, and a file that is there is an error.
func (w *worker) newConfig(data []byte, what string) error {
	if w.o.Write {
		f, err := os.OpenFile(filepath.Join(w.p.Root, project.ConfigName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return project.Errf("%s: adopt never replaces it: %v", project.ConfigName, err)
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	w.record(project.ConfigName, what)
	return nil
}

// adoption writes design/adoption.yaml whole (section 2.1).
func (w *worker) adoption(data []byte, what string) error {
	if w.o.Write {
		if err := w.p.WriteAtomic(project.AdoptionName, data); err != nil {
			return err
		}
	}
	w.record(project.AdoptionName, what)
	return nil
}

// subjectFile is the file a baseline subject is about.
func subjectFile(v baseline.Violation) string {
	s := v.Subject
	switch v.Rule {
	case "layer":
		s, _, _ = strings.Cut(s, " -> ")
	case "test-double":
		s, _, _ = strings.Cut(s, ": ")
	case "func-size":
		if i := strings.LastIndex(s, ":"); i > 0 {
			s = s[:i]
		}
	}
	return s
}

// merge is step 2 for one baseline file: existing entries are kept, and
// violations are added only for paths the old configuration did not cover.
func (w *worker) merge(rel string, viol []baseline.Violation, oldCovers func(*project.Project, string) bool) error {
	entries, problems := baseline.Load(w.p, rel)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	exists := w.p.Exists(rel)
	merged := make([]baseline.Violation, 0, len(entries))
	listed := map[baseline.Entry]bool{}
	for _, e := range entries {
		listed[e] = true
		merged = append(merged, baseline.Violation{Rule: e.Rule, Subject: e.Subject})
	}
	added := 0
	for _, v := range viol {
		if listed[baseline.Entry{Rule: v.Rule, Subject: v.Subject}] {
			continue
		}
		if exists && w.oldP != nil && oldCovers(w.oldP, subjectFile(v)) {
			continue
		}
		merged = append(merged, v)
		added++
	}
	if exists && added == 0 {
		return nil
	}
	if w.o.Write {
		if err := baseline.Write(w.p, rel, merged); err != nil {
			return err
		}
	}
	w.record(rel, fmt.Sprintf("baseline, new entries: %d", added))
	return nil
}

// owed is step 3: gen, then the obligations and unserved act ids that are not
// owed yet.
func (w *worker) owed() error {
	p := w.p
	ctx := &stages.Context{P: p}
	d := ctx.Design()
	for _, name := range []string{"model", "machines", "failure-modes"} {
		s := stages.RunOne(ctx, name)
		if errs, _, _ := s.Count3(); errs > 0 || (name == "model" && s.Outcome() == report.NotDemonstrated) {
			w.res.Blocked = append(w.res.Blocked, s)
		}
	}
	if len(w.res.Blocked) > 0 {
		return nil
	}
	if len(d.OwedProblems) > 0 {
		return fmt.Errorf("%s", strings.Join(d.OwedProblems, "; "))
	}
	if w.o.Write {
		written, left, blocked, err := w.o.Gen(ctx, w.may)
		if err != nil {
			return err
		}
		w.res.Blocked = blocked
		if len(blocked) > 0 {
			return nil
		}
		for _, f := range written {
			w.written[f] = true
		}
		w.res.Left = left
		d = ctx.Design()
	} else {
		pending, left := pendingGen(d, w.may)
		for _, f := range pending {
			w.record(f, "generated by machinery gen")
		}
		w.res.Left = left
	}
	return w.addOwed(d)
}

// may reports a file adopt may write (section 17.2): a file under design/
// or the gen directory, the owed list and the baselines.
func (w *worker) may(rel string) bool {
	c := w.p.Cfg
	gen := strings.TrimSuffix(c.Design.Gen, "/") + "/"
	switch {
	case strings.HasPrefix(rel, DesignDir), strings.HasPrefix(rel, gen), rel == c.Design.Owed:
		return true
	}
	if c.Tests != nil && rel == c.Tests.Baseline {
		return true
	}
	return c.Constraints != nil && rel == c.Constraints.Baseline
}

func (w *worker) addOwed(d *design.Design) error {
	have := d.OwedSet()
	at := " (" + w.reasonSha() + ")"
	reason := design.ReasonUnproven + at
	entries := append([]design.OwedEntry(nil), d.Owed...)
	added := 0
	claims, err := tags.Read(w.p, d)
	if err != nil {
		return err
	}
	served := claims.Served
	planned := w.planned
	// Only what is new is owed: an id the committed obligation list already
	// has was owed, or proven and pruned, before. Owing it again would undo
	// the ratchet.
	known, err := committedIDs(w.p)
	if err != nil {
		return err
	}
	for _, o := range d.Obligations {
		if _, ok := have[o.ID]; !ok && !known[o.ID] {
			// what the project planned, what a test already claims, what no test names
			e := design.OwedEntry{ID: o.ID, Reason: reason}
			switch {
			case planned[o.ID] != "":
				e.Reason, e.Story = design.ReasonPlanned+at, planned[o.ID]
			case len(claims.Tests[o.ID]) > 0:
				e.Reason = design.ReasonClaimed + at
			}
			entries = append(entries, e)
			added++
		}
	}
	for _, s := range d.Served {
		if _, ok := have[s.ID]; !ok && !served[s.ID] && !known[s.ID] {
			entries = append(entries, design.OwedEntry{ID: s.ID, Reason: reason})
			added++
		}
	}
	if added == 0 {
		return nil
	}
	if w.o.Write {
		if err := design.WriteOwed(w.p, entries); err != nil {
			return err
		}
	}
	w.record(w.p.Cfg.Design.Owed, fmt.Sprintf("new owed entries: %d", added))
	return nil
}

// committedIDs returns the ids of the obligation list at HEAD (empty when
// there is none, as on a first adoption, outside git or before the first
// commit).
func committedIDs(p *project.Project) (map[string]bool, error) {
	res := map[string]bool{}
	if head, err := gitx.HeadIfAny(p.Root); err != nil || head == "" {
		return res, err
	}
	data, ok, err := gitx.Show(p.Root, "HEAD", strings.TrimSuffix(p.Cfg.Design.Gen, "/")+"/obligations.json")
	if err != nil || !ok {
		return res, err
	}
	var doc struct {
		Obligations []struct {
			ID string `json:"id"`
		} `json:"obligations"`
		Served []struct {
			ID string `json:"id"`
		} `json:"served"`
	}
	if json.Unmarshal(data, &doc) != nil {
		return res, nil
	}
	for _, o := range doc.Obligations {
		res[o.ID] = true
	}
	for _, s := range doc.Served {
		res[s.ID] = true
	}
	return res, nil
}

func (w *worker) reasonSha() string {
	if w.head == "" {
		return "no commit"
	}
	return w.head
}
