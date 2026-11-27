package stages

import (
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/report"
)

func init() { Bind("verify", func(ctx *Context) *report.Stage { return Verify(ctx, false, false) }) }

// Verify is stage verify (section 12). With partly it names every claimant
// without a pass of a partly run id, not only the first ten.
func Verify(ctx *Context, allowDirty, partly bool) *report.Stage {
	s := report.New("verify")
	ev, err := evidence.Evaluate(ctx.P, ctx.Design(), allowDirty)
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	return VerifyStage(ev, nil, partly)
}

// VerifyStage renders an evaluation as stage verify. The claimants without a
// pass are named for the ids of only, every id when it is nil.
func VerifyStage(ev *evidence.Evaluation, only map[string]bool, partly bool) *report.Stage {
	s := report.New("verify")
	s.Findings = append(s.Findings, ev.Findings...)
	s.Findings = append(s.Findings, ev.PartlyFindings(only, partly)...) // section 10.3
	if vas := ev.Design().VitalAreas(); len(vas) > 0 {
		evidence.VitalOf(ev, vas, nil).Warn(s, true) // section 23.2
	}
	if !ev.HasEvidence {
		return s.NotDemonstrated(ev.NoEvidence())
	}
	s.Count(ev.Obligations, "obligations").Count(ev.NProven, "proven")
	evidence.CountAdapter(s, ev.NByFunction, ev.NByFile) // section 10.3
	s.Count(ev.NPartial, "partial")
	if ev.NPartlyRun > 0 {
		s.Count(ev.NPartlyRun, "partly run")
	}
	s.Count(ev.NOwed, "owed").Count(ev.NReceipts, "receipts")
	return s
}
