package stages

import (
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/report"
)

func init() { Bind("verify", func(ctx *Context) *report.Stage { return Verify(ctx, false) }) }

// Verify is stage verify (section 12).
func Verify(ctx *Context, allowDirty bool) *report.Stage {
	s := report.New("verify")
	ev, err := evidence.Evaluate(ctx.P, ctx.Design(), allowDirty)
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	return VerifyStage(ev)
}

// VerifyStage renders an evaluation as stage verify.
func VerifyStage(ev *evidence.Evaluation) *report.Stage {
	s := report.New("verify")
	s.Findings = append(s.Findings, ev.Findings...)
	if !ev.HasEvidence {
		return s.NotDemonstrated(ev.NoEvidence())
	}
	s.Count(ev.Obligations, "obligations").Count(ev.NProven, "proven").Count(ev.NPartial, "partial").
		Count(ev.NOwed, "owed").Count(ev.NReceipts, "receipts")
	return s
}
