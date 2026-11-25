package stages

import (
	"fmt"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// waivers applies the waivers of machinery.yaml (section 6.7) to the findings
// of the rules one stage owns.
type waivers struct {
	list []project.Waiver
	used []bool
}

// waiversFor returns the waivers that name one of the rules.
func waiversFor(p *project.Project, rules ...string) *waivers {
	own := map[string]bool{}
	for _, r := range rules {
		own[r] = true
	}
	w := &waivers{}
	for _, wv := range p.Cfg.Waivers {
		if own[wv.Rule] {
			w.list = append(w.list, wv)
		}
	}
	w.used = make([]bool, len(w.list))
	return w
}

// find reports a finding "<rule>: <subject>: <what>". A waiver naming the rule
// and the subject turns it into a warning that carries the reason.
func (w *waivers) find(s *report.Stage, rule, subject, format string, a ...any) {
	msg := fmt.Sprintf("%s: %s: %s", rule, subject, fmt.Sprintf(format, a...))
	for i, wv := range w.list {
		if wv.Rule == rule && wv.ID == subject {
			w.used[i] = true
			s.Warnf("waived: %s (%s)", msg, wv.Reason)
			return
		}
	}
	s.Errorf("%s", msg)
}

// stale reports every waiver that matched no finding.
func (w *waivers) stale(s *report.Stage) {
	for i, wv := range w.list {
		if !w.used[i] {
			s.Errorf("stale waiver: %s %s", wv.Rule, wv.ID)
		}
	}
}
