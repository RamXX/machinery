package stages

import (
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("gen-drift", GenDrift) }

// GenDrift is stage gen-drift (section 7).
func GenDrift(ctx *Context) *report.Stage {
	s := report.New("gen-drift")
	d := ctx.Design()
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
