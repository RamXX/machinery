package stages

import (
	"strings"

	"github.com/RamXX/machinery/internal/design"
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
	for _, msg := range d.OwedIssues() {
		s.Errorf("%s", msg)
	}

	// a proposed or retired id is an id of the design, which nothing may claim
	ids := d.DeclaredIDs()
	scenarios := d.ScenarioIDs()
	served, verified := map[string]bool{}, map[string]bool{}
	claimedHeld := map[string]bool{}
	proofs := d.ProofIDs()
	for _, t := range found {
		for _, tok := range t.IDs {
			if !t.Scenario && strings.HasPrefix(tok, "pf:") {
				// a proof obligation is matched against the proof obligations alone (section 8)
				hits := tags.Expand(tok, proofs)
				if len(hits) == 0 {
					s.Errorf("unknown-id: %s:%d: %s", t.File, t.Line, tok)
				}
				for _, id := range hits {
					if key := id + " " + t.File; !claimedHeld[key] {
						claimedHeld[key] = true
						s.Errorf("proof: %s is claimed by %s; tests do not discharge a proof obligation: it awaits a machine-checked proof", id, t.File)
					}
				}
				continue
			}
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
				if st := d.Status(id); st != "" {
					// a pattern claims only the ids that yield an obligation
					if key := id + " " + t.File; !strings.HasSuffix(tok, "*") && !claimedHeld[key] {
						claimedHeld[key] = true
						if st == design.Proposed {
							s.Errorf("proposed: %s is claimed by %s; commit it first", id, t.File)
						} else {
							s.Errorf("retired: %s is claimed by %s; remove the claim, or un-retire it", id, t.File)
						}
					}
					continue
				}
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
