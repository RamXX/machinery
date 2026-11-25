package affected

import (
	"strings"

	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/report"
)

// Restrict narrows an evaluation to a targeted verify (section 12.1): an
// unproven finding survives only for the given ids. Failures and every other
// finding always stay.
func Restrict(ev *evidence.Evaluation, ids []string) *evidence.Evaluation {
	keep := map[string]bool{}
	for _, id := range ids {
		keep[id] = true
	}
	var out []report.Finding
	for _, f := range ev.Findings {
		if id, ok := strings.CutPrefix(f.Msg, "unproven: "); ok && f.Sev == report.Error && !keep[id] {
			continue
		}
		out = append(out, f)
	}
	ev.Findings = out
	return ev
}
