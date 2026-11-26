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
	if !d.HasRisk && !d.HasStatus {
		return s.NotApplicable("no design.risk configured")
	}
	for _, p := range d.RiskProblems {
		s.Errorf("%s", p)
	}
	for _, p := range d.StatusProblems {
		s.Errorf("%s", p)
	}
	for _, p := range d.VitalProblems() {
		s.Errorf("%s", p)
	}
	ids := d.DeclaredIDs()
	cons, crit, vital := 0, 0, 0
	for _, id := range ids {
		if !tierable[kindOf(id)] {
			continue
		}
		switch t, _ := d.Tier(id); t {
		case design.Consequential:
			cons++
		case design.Critical:
			crit++
		case design.Vital:
			vital++
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
	if d.HasRisk {
		s.Count(len(d.Areas), "areas").Count(cons, "consequential ids").Count(crit, "critical ids")
		if vas := d.VitalAreas(); len(vas) > 0 {
			props, assumes := 0, 0
			for _, a := range vas {
				props += len(a.Property)
				assumes += len(a.Assumes)
			}
			s.Count(vital, "vital ids").Count(props, "properties").Count(assumes, "assumptions")
		}
		if len(d.NotVital) > 0 {
			s.Count(len(d.NotVital), "dismissals")
		}
	}
	if d.HasStatus {
		s.Count(len(d.WithStatus(design.Proposed)), "proposed ids").Count(len(d.WithStatus(design.Retired)), "retired ids")
	}
	if len(d.Areas) == 0 && d.StatusList.Entries() == 0 && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
