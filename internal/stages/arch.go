package stages

import (
	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("arch", Arch) }

// Arch is stage arch (section 9).
func Arch(ctx *Context) *report.Stage {
	s := report.New("arch")
	p := ctx.P
	if !arch.Applies(p) {
		return s.NotApplicable("no constraints configured")
	}
	vs, c, err := arch.Violations(p)
	if err != nil {
		s.Errorf("cannot read files: %v", err)
		return s
	}
	entries, problems := baseline.Load(p, arch.BaselinePath(p))
	for _, msg := range problems {
		s.Errorf("%s", msg)
	}
	baseline.Apply(s, entries, vs)
	s.Count(c.Files, "files").Count(c.Imports, "imports").Count(c.Funcs, "functions")
	if (!arch.HasRules(p) || c.Files == 0) && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
