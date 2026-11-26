package assess

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/adopt"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/stages"
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

// design: every lint stage.
func (a *assessment) design(s *section) {
	s.stages = stages.Run(a.ctx, stages.LintStages)
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
	for _, m := range ms {
		states += m.States
		for _, u := range m.Unreachable {
			s.own.Errorf("%s: unreachable state %s", m.Machine, u)
		}
		if m.Deadlock != "" {
			s.own.Errorf("%s: deadlock in state %s: neither final nor _terminal, and no transition applies", m.Machine, m.Deadlock)
		}
		if m.Error != "" {
			s.own.Errorf("%s: TLC could not check the sanity spec: %s", m.Machine, m.Error)
		}
		result := "passed"
		if m.Failed() {
			result = "failed"
		}
		s.lines = append(s.lines, fmt.Sprintf("%s: %d states, %d distinct states found, %s", m.Machine, m.States, m.Distinct, result))
		unreach := m.Unreachable
		if unreach == nil {
			unreach = []string{}
		}
		machines = append(machines, map[string]any{"machine": m.Machine, "states": m.States, "distinct": m.Distinct,
			"result": result, "unreachable": unreach, "deadlock": m.Deadlock})
	}
	models := []any{}
	for _, m := range mods {
		result := "instance found"
		switch {
		case m.Error != "":
			result = "not checked"
			s.own.Errorf("%s: Alloy could not check the skeleton: %s", m.Model, m.Error)
		case !m.Instance:
			result = "no instance"
			s.own.Errorf("%s: no instance in which every entity exists (for 4)", m.Model)
		}
		s.lines = append(s.lines, fmt.Sprintf("%s: %s", m.Model, result))
		models = append(models, map[string]any{"model": m.Model, "result": result})
	}
	s.own.Count(len(ms), "machines").Count(states, "states").Count(len(mods), "models")
	s.facts = map[string]any{"machines": machines, "models": models}
}
