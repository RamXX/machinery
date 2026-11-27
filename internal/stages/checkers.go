package stages

import (
	"github.com/RamXX/machinery/v2/internal/checkers"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("checkers", Checkers) }

// Checkers is stage checkers (section 22.7).
func Checkers(ctx *Context) *report.Stage { return checkers.Judge(ctx.Design()) }
