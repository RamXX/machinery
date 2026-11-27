package affected

import (
	"github.com/RamXX/machinery/v2/internal/evidence"
	"github.com/RamXX/machinery/v2/internal/report"
)

// idFindings are the findings of stage verify that a targeted verify keeps
// only for the given ids (section 12.1).
var idFindings = []string{"unproven: ", "partly run: ", "no claimant can run here: "}

// Restrict narrows an evaluation to a targeted verify (section 12.1): an
// unproven finding, a partly run one and one of an id no claimant of which
// can run here survive only for the given ids. Failures and every other
// finding always stay.
func Restrict(ev *evidence.Evaluation, ids []string) *evidence.Evaluation {
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	var out []report.Finding
	for _, f := range ev.Findings {
		if id, ok := evidence.FindingID(f.Msg, idFindings); ok && f.Sev == report.Error && !keep[id] {
			continue
		}
		out = append(out, f)
	}
	ev.Findings = out
	return ev
}
