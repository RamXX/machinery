package stages

import (
	"github.com/RamXX/machinery/internal/checkers"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("checkers", Checkers) }

// Checkers is stage checkers (section 21.7).
func Checkers(ctx *Context) *report.Stage { return checkers.Judge(ctx.Design()) }
