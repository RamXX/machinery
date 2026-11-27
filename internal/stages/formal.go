package stages

import (
	"github.com/RamXX/machinery/v2/internal/formal"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("formal", Formal) }

// Formal is stage formal (section 16.9). It is pure: it runs no tool.
func Formal(ctx *Context) *report.Stage {
	j, err := formal.Judge(ctx.Design())
	if err != nil {
		s := report.New("formal")
		s.Errorf("%v", err)
		return s
	}
	return j.Stage()
}
