package stages

import (
	"github.com/RamXX/machinery/internal/probe"
	"github.com/RamXX/machinery/internal/report"
)

func init() {
	Bind("probe", func(ctx *Context) *report.Stage {
		return report.New("probe").NotDemonstrated("no command; run machinery probe -- <command>")
	})
}

// Probe is stage probe (section 20) for one test command.
func Probe(ctx *Context, opts probe.Options) *report.Stage {
	return probe.Stage(ctx.P, ctx.Design(), opts)
}
