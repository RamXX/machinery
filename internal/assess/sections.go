package assess

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/adopt"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
	"github.com/RamXX/machinery/internal/tags"
	"github.com/RamXX/machinery/internal/vital"
)

// inventory: the facts of the adopt report for the snapshot, and the links
// left out of it.
func (a *assessment) inventory(s *section) {
	for _, l := range a.outside {
		s.own.Warnf("outside-link: %s: leaves the project; not in the snapshot, not assessed", l)
	}
	files, err := a.p.Files()
	if err != nil {
		s.own.Errorf("cannot list files: %v", err)
		return
	}
	r, err := adopt.Describe(a.p)
	if err != nil {
		s.own.Errorf("%v", err)
		return
	}
	s.lines = r.FactLines()
	langs := r.Languages
	if langs == nil {
		langs = []string{}
	}
	s.facts = map[string]any{
		"scope": r.Scope, "languages": langs, "code_files": r.Counts.Code,
		"e2e_test_files": r.Counts.E2E, "unit_test_files": r.Counts.Unit, "double_test_files": r.Counts.Doubles,
		"models": r.Models, "machines": r.Machines, "contracts": r.Contracts,
	}
	s.own.Count(len(files), "files")
	if len(files) == 0 {
		s.own.NotDemonstrated("nothing to check")
	}
}

// design: every lint stage, the ids by status, and how many machines bind
// to the model.
func (a *assessment) design(s *section) {
	s.stages = stages.Run(a.ctx, stages.LintStages)
	// an owed list with a problem is not counted as if it were sound (section 11.1)
	for _, msg := range a.ctx.Design().OwedProblems {
		s.own.Errorf("%s", msg)
	}
	proposed, owed, proven, retired := a.ctx.Design().StatusCounts()
	s.lines = append(s.lines, fmt.Sprintf("ids: %d proposed, %d owed, %d proven, %d retired", proposed, owed, proven, retired))
	s.facts = map[string]any{"statuses": map[string]any{"proposed": proposed, "owed": owed, "proven": proven, "retired": retired}}
	a.vitalLines(s)
	if len(a.ctx.Design().Machines) > 0 {
		bound, unbound := stages.BindingCounts(a.ctx)
		s.own.Count(bound, "bound machines").Count(unbound, "unbound machines")
		s.facts["bound_machines"], s.facts["unbound_machines"] = bound, unbound
	}
}

// vitalLines adds the lines of the vital areas, by committed status, and the
// candidates (sections 18.3, 23.2 and 23.3).
func (a *assessment) vitalLines(s *section) {
	d := a.ctx.Design()
	vas := d.VitalAreas()
	proofs := design.ProofsOf(vas)
	if len(vas) > 0 {
		s.lines = append(s.lines, fmt.Sprintf("proofs awaiting: %d", len(proofs)))
		for _, id := range design.OverIDs(vas) {
			if d.Known(id) && d.CommittedStatus(id) == "proven" {
				s.lines = append(s.lines, evidence.ProvenLine(id))
			}
		}
		for _, u := range d.Unevidenced(vas) {
			s.lines = append(s.lines, u.Line())
		}
	}
	cands, dismissed := vital.Candidates(d)
	s.lines = append(s.lines, vital.Summary(len(cands), dismissed))
	list := []any{}
	for _, c := range cands {
		s.lines = append(s.lines, "  "+c.Line())
		list = append(list, map[string]any{"id": c.ID, "reason": c.Reason})
	}
	s.facts["proofs_awaiting"] = len(proofs)
	s.facts["vital_candidates"] = list
	s.facts["vital_dismissed"] = dismissed
}

// trace: the stage, and the owed ids of the snapshot by claim, counted as
// machinery status counts them (sections 14 and 18.3).
func (a *assessment) trace(s *section) {
	s.stages = stages.Run(a.ctx, []string{"trace"})
	if a.p.Cfg.Trace == nil {
		return
	}
	d := a.ctx.Design()
	claims, err := tags.Read(a.p, d)
	if err != nil {
		s.own.Errorf("cannot read tags: %v", err)
		return
	}
	ev, err := evidence.Evaluate(a.p, d, true)
	if err != nil {
		s.own.Errorf("cannot read the evidence: %v", err)
		return
	}
	split := claims.SplitOwed(d, func(id string) bool { return ev.State(id) == evidence.Proven })
	s.lines = split.Lines("owed, ")
	s.facts = split.Doc()
}

func stageSection(name string) func(a *assessment, s *section) {
	return func(a *assessment, s *section) {
		s.stages = stages.Run(a.ctx, []string{name})
	}
}

// generation: machinery gen in the snapshot.
func (a *assessment) generation(s *section) {
	written, blocked, err := a.o.Gen(a.ctx)
	if err != nil {
		s.own.Errorf("gen: %v", err)
		return
	}
	if len(blocked) > 0 {
		var b strings.Builder
		var names []string
		for _, st := range blocked {
			report.Print(&b, st)
			names = append(names, st.Name)
		}
		b.WriteString("gen: refused: the design has errors; nothing was written\n")
		s.text = b.String()
		s.lines = []string{"gen: refused: " + strings.Join(names, ", ") + " have errors"}
		s.own.NotDemonstrated("gen refused: " + strings.Join(names, ", ") + " have errors")
		s.facts = map[string]any{"files": 0, "written": 0}
		return
	}
	s.lines = append(s.lines, fmt.Sprintf("gen: wrote %d file(s)", len(written)))
	for _, f := range written {
		s.lines = append(s.lines, "  "+f)
	}
	diffs, n, err := gen.Compare(a.ctx.Design())
	if err != nil {
		s.own.Errorf("gen: %v", err)
		return
	}
	for _, df := range diffs {
		s.own.Errorf("%s: not current after gen", df.Path)
	}
	s.own.Count(n, "generated files").Count(len(written), "written")
	s.facts = map[string]any{"files": n, "written": len(written)}
}

// formal: the sanity run of section 18.5.
func (a *assessment) formal(s *section) {
	d := a.ctx.Design()
	if len(d.Machines) == 0 && len(d.Models) == 0 {
		s.own.NotApplicable("no machine and no model")
		return
	}
	ms, mods, err := formal.Sanity(a.p, d, filepath.Join(a.out, "formal"), 4)
	var te *formal.ToolError
	if errors.As(err, &te) {
		s.own.NotDemonstrated(te.Reason)
		return
	}
	if err != nil {
		s.own.Errorf("%v", err)
		return
	}
	states := 0
	machines := []any{}
	// a run that did not complete gives no verdict (sections 16.9 and 18.5)
	var incomplete []string
	notComplete := func(name, reason string) {
		msg := fmt.Sprintf("%s: the sanity run did not complete: %s", name, reason)
		s.own.Warnf("%s", msg)
		incomplete = append(incomplete, msg)
	}
	for _, m := range ms {
		states += m.States
		for _, u := range m.Unreachable {
			s.own.Errorf("%s: unreachable state %s", m.Machine, u)
		}
		if m.Deadlock != "" {
			s.own.Errorf("%s: deadlock in state %s: neither final nor _terminal, and no transition applies", m.Machine, m.Deadlock)
		}
		result := "passed"
		line := fmt.Sprintf("%s: %d states, %d distinct states found, ", m.Machine, m.States, m.Distinct)
		switch {
		case m.Incomplete != "":
			notComplete(m.Machine, m.Incomplete)
			result = "incomplete"
			line = fmt.Sprintf("%s: %d states, did not complete", m.Machine, m.States)
		case m.Failed():
			result = "failed"
			line += result
		default:
			line += result
		}
		s.lines = append(s.lines, line)
		unreach := m.Unreachable
		if unreach == nil {
			unreach = []string{}
		}
		machines = append(machines, map[string]any{"machine": m.Machine, "states": m.States, "distinct": m.Distinct,
			"result": result, "reason": m.Incomplete, "unreachable": unreach, "deadlock": m.Deadlock})
	}
	models := []any{}
	for _, m := range mods {
		result, line := "instance found", "instance found"
		switch {
		case m.Incomplete != "":
			notComplete(m.Model, m.Incomplete)
			result, line = "incomplete", "did not complete"
		case !m.Instance:
			result, line = "no instance", "no instance"
			s.own.Errorf("%s: no instance in which every entity exists (for 4)", m.Model)
		}
		s.lines = append(s.lines, fmt.Sprintf("%s: %s", m.Model, line))
		models = append(models, map[string]any{"model": m.Model, "result": result, "reason": m.Incomplete})
	}
	s.own.Count(len(ms), "machines").Count(states, "states").Count(len(mods), "models")
	s.facts = map[string]any{"machines": machines, "models": models}
	if len(incomplete) > 0 {
		s.own.NotDemonstrated(incomplete[0] + "; fix the cause and run machinery assess again")
	}
}
