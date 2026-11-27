package stages

import (
	"bytes"
	"strings"

	"github.com/RamXX/machinery/v2/internal/modelith"
	"github.com/RamXX/machinery/v2/internal/project"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("model", Model) }

// Model is stage model (section 6.1).
func Model(ctx *Context) *report.Stage {
	s := report.New("model")
	p := ctx.P
	if len(p.Cfg.Design.Models) == 0 {
		return s.NotApplicable("no design.models configured")
	}
	if _, reason := modelith.Check(p); reason != "" {
		return s.NotDemonstrated(reason)
	}
	d := ctx.Design()
	for _, prob := range d.ModelProblems {
		s.Errorf("%s", prob)
	}
	for _, rel := range p.Cfg.Design.Models {
		if !p.Exists(rel) {
			continue // reported as a model problem
		}
		findings, err := modelith.Lint(p, rel)
		if err != nil {
			s.Errorf("%s: %v", rel, err)
			continue
		}
		for _, f := range findings {
			msg := strings.ReplaceAll(f.Message, "\u2014", "-")
			if f.Severity == "warn" {
				s.Warnf("%s: %s", rel, msg)
			} else {
				s.Errorf("%s: %s", rel, msg)
			}
		}
		if !p.Cfg.ToolRenders() {
			continue // the rendering is the project's (design.rendering, section 5.1)
		}
		want, err := modelith.Render(p, rel)
		if err != nil {
			s.Errorf("%s: %v", rel, err)
			continue
		}
		md := modelith.RenderedPath(rel)
		got, rerr := p.Read(md)
		switch {
		case rerr != nil:
			s.Driftf("%s: missing; run machinery gen", md)
		case !bytes.Equal(got, want):
			s.Driftf("%s: stale; run machinery gen", md)
		}
	}
	if p.Cfg.Design.Rendering == project.RenderingProject {
		s.Warnf("rendering: kept by the project")
	}
	var entities, actions, invs, scs int
	for _, m := range d.Models {
		entities += len(m.Entities)
		invs += len(m.Invariants)
		scs += len(m.Scenarios)
		for _, e := range m.Entities {
			actions += len(e.Actions)
			invs += len(e.Invariants)
		}
	}
	for _, id := range d.Duplicates {
		if k := kindOf(id); k == "act" || k == "inv" || k == "sc" {
			s.Errorf("duplicate id %s", id)
		}
	}
	s.Count(len(d.Models), "models").Count(entities, "entities").Count(actions, "actions").Count(invs, "invariants").Count(scs, "scenarios")
	return s
}
