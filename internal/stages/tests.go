package stages

import (
	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/testpolicy"
)

func init() { Bind("tests", Tests) }

// Tests is stage tests (section 9.4).
func Tests(ctx *Context) *report.Stage {
	s := report.New("tests")
	p := ctx.P
	if !testpolicy.Applies(p) {
		return s.NotApplicable("no tests or trace.tests configured")
	}
	vs, c, err := testpolicy.Violations(p)
	if err != nil {
		s.Errorf("cannot read test files: %v", err)
		return s
	}
	entries, problems := baseline.Load(p, testpolicy.BaselinePath(p))
	for _, msg := range problems {
		s.Errorf("%s", msg)
	}
	baseline.Apply(s, entries, vs)
	s.Count(c.TestFiles, "test files").Count(c.E2EFiles, "files under e2e roots").Count(c.Markers, "markers")
	if c.TestFiles == 0 && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
