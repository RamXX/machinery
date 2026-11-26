package stages

import (
	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/vital"
)

func init() { Bind("arch", Arch) }

// ArchViolations returns the violations of stage arch, the cores of the
// vital areas included, and what it counted.
func ArchViolations(ctx *Context) ([]arch.Violation, arch.Counts, []arch.Core, error) {
	cores, err := vital.Cores(ctx.Design())
	if err != nil {
		return nil, arch.Counts{}, nil, err
	}
	vs, c, err := arch.Violations(ctx.P, cores)
	return vs, c, cores, err
}

// Arch is stage arch (section 9).
func Arch(ctx *Context) *report.Stage {
	s := report.New("arch")
	p := ctx.P
	vs, c, cores, err := ArchViolations(ctx)
	if !arch.Applies(p) && len(cores) == 0 {
		return s.NotApplicable("no constraints configured")
	}
	if err != nil {
		s.Errorf("cannot read files: %v", err)
		return s
	}
	if p.Cfg.Constraints == nil || p.Cfg.Constraints.Cores == nil {
		for _, core := range cores {
			s.Errorf("cores: %s is vital and constraints.cores is not set; set max_file_lines and max_func_lines for its core", core.Area)
		}
	}
	entries, problems := baseline.Load(p, arch.BaselinePath(p))
	for _, msg := range problems {
		s.Errorf("%s", msg)
	}
	baseline.Apply(s, entries, vs)
	s.Count(c.Files, "files").Count(c.Imports, "imports").Count(c.Funcs, "functions")
	for _, cc := range c.Cores {
		s.Count(cc.Files, "core files of "+cc.Area).Count(cc.Lines, "core lines of "+cc.Area).Count(cc.Entries, "entry points of "+cc.Area)
	}
	for _, msg := range c.Undemonstrated {
		s.Warnf("%s", msg)
	}
	if len(c.Undemonstrated) > 0 {
		s.NotDemonstrated(c.Undemonstrated[0])
	}
	if (!arch.HasRules(p) || c.Files == 0) && len(c.Cores) == 0 && s.Blocking() == 0 {
		s.NotDemonstrated("nothing to check")
	}
	return s
}
