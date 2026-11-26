package checkers

import (
	"fmt"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/report"
)

// FindingLine renders a finding the way the stage and the run command print it.
func FindingLine(checker string, f Finding) string {
	s := fmt.Sprintf("%s: %s: %s", checker, f.Subject, f.Message)
	if f.Rule != "" {
		s += " [" + f.Rule + "]"
	}
	return s
}

// Judge is stage checkers (section 22.7). It is pure: it reads only committed
// files and the design, runs no tool and reads no registry.
func Judge(d *design.Design) *report.Stage {
	s := report.New("checkers")
	p := d.P
	if !Configured(d) {
		return s.NotApplicable("no checkers and no design.facts configured")
	}
	proj, factProblems := Build(d)
	for _, pr := range factProblems {
		s.Errorf("%s: %s", p.Cfg.Design.Facts, pr)
	}
	data, err := proj.Bytes()
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	sha := SHA256(data)
	ids := proj.IDs()
	covered, findings, obligations := 0, 0, 0
	for _, ck := range p.Cfg.Checkers {
		selected := map[string]bool{}
		for _, pat := range ck.Covers {
			matched := false
			for id := range ids {
				if design.MatchID(pat, id) {
					selected[id] = true
					matched = true
				}
			}
			if !matched {
				s.Errorf("%s: covers matches nothing: %s", ck.ID, pat)
			}
		}
		a := Assess(p, ck, sha, ids)
		switch a.Status {
		case Missing:
			s.Errorf("%s: no evidence; run machinery checkers run %s", ck.ID, ck.ID)
			continue
		case Invalid:
			for _, pr := range a.Problems {
				s.Errorf("%s: %s", ck.Evidence, pr)
			}
			continue
		case Stale:
			s.Driftf("%s: stale", ck.Evidence)
		}
		ev := a.Evidence
		for _, f := range ev.Findings {
			if f.Severity == "error" {
				s.Errorf("%s", FindingLine(ck.ID, f))
			} else {
				s.Warnf("%s", FindingLine(ck.ID, f))
			}
		}
		have := map[string]bool{}
		for _, c := range ev.Covers {
			have[c] = true
		}
		covered += len(have)
		findings += len(ev.Findings)
		obligations += len(ev.Obligations)
		if a.Status == Stale {
			continue
		}
		for id := range selected {
			if !have[id] {
				s.Errorf("%s: not covered: %s", ck.ID, id)
			}
		}
	}
	s.Count(len(p.Cfg.Checkers), "checkers").Count(covered, "covered ids").Count(findings, "findings").
		Count(obligations, "obligations").Count(len(proj.Facts), "facts")
	return s
}
