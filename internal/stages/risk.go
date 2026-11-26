package stages

import (
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("risk", Risk) }

// tierable are the id kinds a risk area may name (section 23).
var tierable = map[string]bool{"act": true, "inv": true, "sc": true, "tc": true, "fm": true}

func kindOf(id string) string {
	for i := 0; i < len(id); i++ {
		if id[i] == ':' {
			return id[:i]
		}
	}
	return ""
}

// Risk is stage risk (section 23).
func Risk(ctx *Context) *report.Stage {
	s := report.New("risk")
	d := ctx.Design()
	if !d.HasRisk {
		return s.NotApplicable("no design.risk configured")
	}
	for _, p := range d.RiskProblems {
		s.Errorf("%s", p)
	}
	ids := d.AllIDs()
	cons, crit := 0, 0
	for _, id := range ids {
		if !tierable[kindOf(id)] {
			continue
		}
		switch t, _ := d.Tier(id); t {
		case design.Consequential:
			cons++
		case design.Critical:
			crit++
		}
	}
	rel := ctx.P.Cfg.Design.Risk
	for _, a := range d.Areas {
		for _, pat := range a.IDs {
			matched := false
			for _, id := range ids {
				if tierable[kindOf(id)] && design.MatchID(pat, id) {
					matched = true
					break
				}
			}
			if !matched {
				s.Errorf("%s: %s: %s matches no act, inv, sc, tc or fm id", rel, a.Name, pat)
			}
		}
	}
	s.Count(len(d.Areas), "areas").Count(cons, "consequential ids").Count(crit, "critical ids")
	if len(d.Areas) == 0 && len(d.RiskProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
