package stages

import (
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/tags"
)

func init() { Bind("trace", Trace) }

// claimed are the obligation kinds a test must verify (section 8). fv ids are
// proven by the formal stage.
var claimed = map[string]bool{"inv": true, "sc": true, "tc": true, "fm": true, "cx": true, "ck": true}

// Trace is stage trace (section 8).
func Trace(ctx *Context) *report.Stage {
	s := report.New("trace")
	p := ctx.P
	if p.Cfg.Trace == nil {
		return s.NotApplicable("no trace configured")
	}
	d := ctx.Design()
	code, err := tags.CodeFiles(p)
	if err != nil {
		s.Errorf("cannot list code files: %v", err)
	}
	tests, err := tags.TestFiles(p)
	if err != nil {
		s.Errorf("cannot list test files: %v", err)
	}
	if len(p.Cfg.Trace.Code) > 0 && len(code) == 0 {
		s.Errorf("trace.code matches no file")
	}
	if len(p.Cfg.Trace.Tests) > 0 && len(tests) == 0 {
		s.Errorf("trace.tests matches no file")
	}
	found, err := tags.Scan(p, append(append([]string{}, code...), tests...))
	if err != nil {
		s.Errorf("cannot read tags: %v", err)
	}

	for _, msg := range d.OwedProblems {
		s.Errorf("%s", msg)
	}
	owed := d.OwedSet()
	for id := range owed {
		if !d.Known(id) {
			s.Errorf("unknown owed id: %s", id)
		}
	}

	ids := d.AllIDs()
	scenarios := d.ScenarioIDs()
	served, verified := map[string]bool{}, map[string]bool{}
	for _, t := range found {
		for _, tok := range t.IDs {
			hits := tags.Resolve(t, tok, ids, scenarios)
			if len(hits) == 0 {
				if t.Scenario {
					s.Errorf("unknown-id: %s:%d: scenario %q", t.File, t.Line, tok)
				} else {
					s.Errorf("unknown-id: %s:%d: %s", t.File, t.Line, tok)
				}
				continue
			}
			for _, id := range hits {
				if t.Kind == tags.Serves {
					served[id] = true
				} else {
					verified[id] = true
				}
			}
		}
	}
	for _, a := range d.Served {
		if _, isOwed := owed[a.ID]; !served[a.ID] && !isOwed {
			s.Errorf("unserved: %s", a.ID)
		}
	}
	for _, o := range d.Obligations {
		if _, isOwed := owed[o.ID]; claimed[o.Kind] && !verified[o.ID] && !isOwed {
			s.Errorf("unverified: %s", o.ID)
		}
	}

	s.Count(len(code), "code files").Count(len(tests), "test files").
		Count(len(d.Served), "act ids").Count(len(d.Obligations), "obligations")
	if len(d.Served)+len(d.Obligations) == 0 && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
