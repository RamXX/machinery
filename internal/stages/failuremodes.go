package stages

import "github.com/RamXX/machinery/internal/report"

func init() { Bind("failure-modes", FailureModes) }

// FailureModes is stage failure-modes (section 6.3).
func FailureModes(ctx *Context) *report.Stage {
	s := report.New("failure-modes")
	d := ctx.Design()
	if !d.HasFM {
		return s.NotApplicable("no design.failure_modes configured")
	}
	abuse := 0
	for _, fm := range d.FailureModes {
		if fm.Kind == "abuse" {
			abuse++
		}
	}
	s.Count(len(d.FailureModes), "failure modes").Count(abuse, "abuse cases")
	for _, p := range d.FMProblems {
		s.Errorf("%s", p)
	}
	if len(d.FailureModes) == 0 && len(d.FMProblems) == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
