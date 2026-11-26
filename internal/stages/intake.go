package stages

import (
	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/pm"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("intake", Intake) }

// Intake is stage intake (section 15.7): warnings about open intake items
// whose design changed without planned stories, and about delivered ones
// still open. It reads the backlog once, and only when bd can run.
func Intake(ctx *Context) *report.Stage {
	s := report.New("intake")
	if ctx.P.Cfg.Backlog == nil {
		return s.NotApplicable("no backlog configured")
	}
	if ctx.NoBacklog != "" {
		return s.NotApplicable(ctx.NoBacklog)
	}
	c, err := backlog.New(ctx.P, "")
	if err != nil {
		return s.NotApplicable("bd not runnable")
	}
	issues, err := c.List()
	if err != nil {
		return s.NotApplicable("the backlog cannot be read")
	}
	open := pm.OpenIntakes(issues)
	if len(open) == 0 {
		return s.NotApplicable("no open intake item")
	}
	contract := pm.ContractOf(ctx.Design()).Hash
	for _, in := range open {
		for _, msg := range pm.IntakeFindings(in, contract, issues) {
			s.Warnf("%s", msg)
		}
	}
	return s.Count(len(open), "intake items")
}
