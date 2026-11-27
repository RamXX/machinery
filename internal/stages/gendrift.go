package stages

import (
	"github.com/RamXX/machinery/v2/internal/gen"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("gen-drift", GenDrift) }

// GenDrift is stage gen-drift (section 7).
func GenDrift(ctx *Context) *report.Stage {
	s := report.New("gen-drift")
	d := ctx.Design()
	c := ctx.P.Cfg
	if len(c.Design.Models) == 0 && !c.HasMachines() && !c.HasFailureModes() && c.Design.Risk == "" && len(c.Checkers) == 0 {
		if files, _ := ctx.P.Glob([]string{gen.Dir(d) + "/**"}); len(files) == 0 {
			return s.NotApplicable("no design sources configured")
		}
	}
	diffs, n, err := gen.Compare(d)
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	s.Count(n, "generated files")
	for _, df := range diffs {
		switch df.Kind {
		case gen.Stale:
			s.Driftf("%s: stale", df.Path)
		case gen.Missing:
			s.Driftf("%s: missing", df.Path)
		case gen.Unexpected:
			s.Errorf("%s: unexpected file in gen directory", df.Path)
		}
	}
	return s
}
