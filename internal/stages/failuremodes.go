package stages

import "github.com/RamXX/machinery/internal/report"

func init() { Bind("failure-modes", FailureModes) }

// FailureModes is stage failure-modes (section 6.3).
func FailureModes(ctx *Context) *report.Stage {
	s := report.New("failure-modes")
	d := ctx.Design()
	if !d.HasFailureModes() {
		return s.NotApplicable("no design.failure_modes configured")
	}
	abuse, imported := 0, 0
	for _, fm := range d.FailureModes {
		if fm.Kind == "abuse" {
			abuse++
		}
		if fm.Imported {
			imported++
		}
	}
	s.Count(len(d.FailureModes), "failure modes").Count(abuse, "abuse cases")
	if d.ImportsFM {
		s.Count(imported, "imported")
	}
	// the same account stage xref-failure gives (section 6.6)
	ctx.failureAccount().count(s, d.ImportsFM)
	for _, p := range d.FMProblems {
		s.Errorf("%s", p)
	}
	if len(d.FailureModes) == 0 && len(d.FMProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
