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

	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
	"github.com/RamXX/machinery/internal/tags"
)

// Options are the arguments of machinery adopt.
type Options struct {
	Root  string   // the directory to adopt
	Paths []string // --paths
	E2E   []string // --e2e
	Write bool     // --write
	// NoBacklog never runs the backlog tool (a rehearsal, section 17.3).
	NoBacklog bool
	// Gen writes the generated files (machinery gen). It refuses, returning
	// the blocking stages, when the design has errors.
	Gen func(*stages.Context) (written []string, blocked []*report.Stage, err error)
}

// Change is one file adopt would write, and what it would put in it.
type Change struct{ Path, What string }

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
	cfgBytes, rerr := os.ReadFile(filepath.Join(root, project.ConfigName))
	existing := rerr == nil
	var oldP *project.Project
	if existing {
		if oldP, err = project.Load(root); err != nil {
			return nil, err
		}
		if msg := design.OwedUnreadable(oldP); msg != "" {
			return nil, fmt.Errorf("%s", msg)
		}
	} else {
		cfgBytes = nil
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
	res := &Result{Name: filepath.Base(root), Write: o.Write, Sha: "no commit", Backlog: readBacklog(root, exe, o.NoBacklog)}
	var head string
	if h, herr := gitx.Head(root); herr == nil {
		head = gitx.Short(h)
		res.Sha = head
	}

	ed, err := newEditor(cfgBytes)
	if err != nil {
		return nil, err
	}
	applyEdits(ed, existing, o, discover(files, genDir), res.Backlog.Found)
	newBytes, err := ed.bytes()
	if err != nil {
		return nil, err
	}
	final := cfgBytes
	if newBytes != nil {
		final = newBytes
	}
	cfg, err := project.Parse(final)
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
		ed.addWaivers(gaps, step.reason+" ("+sha+")", step.note)
		waiverNotes = append(waiverNotes, fmt.Sprintf(step.note, len(gaps)))
		if newBytes, err = ed.bytes(); err != nil {
			return nil, err
		}
		if cfg, err = project.Parse(newBytes); err != nil {
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
	if newBytes != nil {
		what := "new config"
		if existing {
			what = "edit: " + strings.Join(ed.changes, "; ")
		} else if len(waiverNotes) > 0 {
			what += "; " + strings.Join(waiverNotes, "; ")
		}
		if err := w.config(newBytes, what); err != nil {
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

// applyEdits is step 1 of section 17.2: it adds what adoption needs to the
// configuration and changes nothing else.
func applyEdits(ed *editor, existing bool, o Options, src Sources, backlogFound bool) {
	root := ed.root
	switch {
	case !existing:
		paths := o.Paths
		if len(paths) == 0 {
			paths = []string{defaultScopeGlob}
		}
		ed.addSeq(root, "scope", paths, true, "scope")
	case get(root, "scope") != nil:
		ed.addSeq(root, "scope", covered(seqValues(get(root, "scope")), o.Paths), true, "scope")
	}
	effScope := seqValues(get(root, "scope"))
	if len(effScope) == 0 {
		effScope = []string{defaultScopeGlob}
	}

	// design sources found now, including ones written after the first adoption
	var newModels []string
	have := map[string]bool{}
	for _, m := range seqValues(get(get(root, "design"), "models")) {
		have[m] = true
	}
	for _, m := range src.Models {
		if !have[m] {
			newModels = append(newModels, m)
		}
	}
	if len(newModels) > 0 {
		ed.addSeq(ed.mapping(root, "design"), "models", newModels, false, "design.models")
	}
	imported := get(get(root, "import"), "machines") != nil
	if len(src.Machines) > 0 && get(get(root, "design"), "machines") == nil && !imported {
		dir := src.machinesDir()
		ed.set(ed.mapping(root, "design"), "machines", dir, "design.machines +"+dir)
	}
	haveContract := ed.contractPaths()
	for _, c := range src.Contracts {
		if !haveContract[c] {
			ed.addContract(c)
		}
	}
	design := get(root, "design")
	hasDesign := len(seqValues(get(design, "models"))) > 0 || get(design, "machines") != nil || imported

	traceExisted := get(root, "trace") != nil
	tests := get(root, "tests")
	e2e := o.E2E
	if len(e2e) == 0 {
		e2e = seqValues(get(tests, "e2e"))
	}
	if len(e2e) == 0 {
		e2e = seqValues(get(get(root, "trace"), "tests"))
	}
	if len(e2e) == 0 {
		e2e = []string{DefaultE2E}
	}
	switch trace := get(root, "trace"); {
	case trace == nil && hasDesign:
		t := ed.mapping(root, "trace")
		ed.addSeq(t, "code", effScope, true, "trace.code")
		ed.addSeq(t, "tests", e2e, true, "trace.tests")
	case trace != nil:
		if get(trace, "code") != nil {
			ed.addSeq(trace, "code", covered(seqValues(get(trace, "code")), o.Paths), true, "trace.code")
		}
		if get(trace, "tests") != nil {
			ed.addSeq(trace, "tests", covered(seqValues(get(trace, "tests")), o.E2E), true, "trace.tests")
		}
	}

	if tests == nil {
		tests = ed.mapping(root, "tests")
		if !traceExisted {
			ed.addSeq(tests, "e2e", e2e, true, "tests.e2e")
		}
	} else if len(o.E2E) > 0 {
		switch {
		case get(tests, "e2e") != nil:
			ed.addSeq(tests, "e2e", covered(seqValues(get(tests, "e2e")), o.E2E), true, "tests.e2e")
		case get(get(root, "trace"), "tests") == nil:
			ed.addSeq(tests, "e2e", o.E2E, true, "tests.e2e")
		}
	}
	ed.set(tests, "baseline", TestsBaseline, "tests.baseline")

	if cons := get(root, "constraints"); cons != nil {
		ed.set(cons, "baseline", ArchBaseline, "constraints.baseline")
	}
	if backlogFound && get(root, "backlog") == nil {
		ed.set(ed.mapping(root, "backlog"), "tool", "bd", "backlog.tool: bd")
	}
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

func (w *worker) config(data []byte, what string) error {
	if w.o.Write {
		if err := project.WriteFileAtomic(filepath.Join(w.p.Root, project.ConfigName), data); err != nil {
			return err
		}
	}
	w.record(project.ConfigName, what)
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
		written, blocked, err := w.o.Gen(ctx)
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
		d = ctx.Design()
	} else {
		for _, f := range pendingGen(d) {
			w.record(f, "generated by machinery gen")
		}
	}
	return w.addOwed(d)
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
	known := committedIDs(w.p)
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
// there is none, as on a first adoption).
func committedIDs(p *project.Project) map[string]bool {
	res := map[string]bool{}
	data, ok := gitx.Show(p.Root, "HEAD", strings.TrimSuffix(p.Cfg.Design.Gen, "/")+"/obligations.json")
	if !ok {
		return res
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
		return res
	}
	for _, o := range doc.Obligations {
		res[o.ID] = true
	}
	for _, s := range doc.Served {
		res[s.ID] = true
	}
	return res
}

func (w *worker) reasonSha() string {
	if w.head == "" {
		return "no commit"
	}
	return w.head
}
