package stages

import (
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/xstate"
)

func init() { Bind("xstate", XState) }

// XState is stage xstate (section 6.8).
func XState(ctx *Context) *report.Stage {
	s := report.New("xstate")
	p := ctx.P
	if !p.Cfg.Tools.XState {
		return s.NotApplicable("tools.xstate is not enabled")
	}
	pkg, reason := xstate.Package(p)
	if reason != "" {
		return s.NotDemonstrated(reason)
	}
	if reason := xstate.Node(p); reason != "" {
		return s.NotDemonstrated(reason)
	}
	ms := ctx.Design().Machines
	if len(ms) == 0 {
		return s.NotDemonstrated("nothing to check")
	}
	results, err := xstate.Run(p, pkg, ms)
	if err != nil {
		return s.NotDemonstrated(err.Error())
	}
	cases := 0
	for _, m := range ms {
		findings, n := xstate.Compare(m, results)
		cases += n
		for _, f := range findings {
			s.Errorf("xstate: %s", f)
		}
	}
	return s.Count(len(ms), "machines").Count(cases, "cases")
}
