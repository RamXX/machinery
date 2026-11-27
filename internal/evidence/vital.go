package evidence

import (
	"fmt"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/report"
)

// Vital is what keeps the evidence of vital areas from reading as more than
// it is (section 23.2).
type Vital struct {
	Proofs  []string // the proof obligations of the areas, sorted
	Proven  []string // ids a property is over that the evidence proves, sorted
	Assumed []design.UnevidencedAssumption
}

// VitalOf gathers the lines of the given vital areas. When only is not nil,
// Proven keeps the ids it holds.
func VitalOf(ev *Evaluation, areas []design.Area, only map[string]bool) Vital {
	d := ev.d
	v := Vital{Proofs: design.ProofsOf(areas), Assumed: d.Unevidenced(areas)}
	for _, id := range design.OverIDs(areas) {
		if (only == nil || only[id]) && ev.State(id) == Proven {
			v.Proven = append(v.Proven, id)
		}
	}
	return v
}

// ProofsLine counts the proof obligations.
func (v Vital) ProofsLine() string { return fmt.Sprintf("proofs awaiting: %d", len(v.Proofs)) }

// ProvenLine reports an id a property is over as proven, with the wording of
// section 23.2.
func ProvenLine(id string) string { return fmt.Sprintf("vital: %s proven %s", id, design.ProofAwaited) }

// Lines are the report lines: the count, the proven ids, the assumptions.
func (v Vital) Lines() []string {
	lines := []string{v.ProofsLine()}
	for _, id := range v.Proven {
		lines = append(lines, ProvenLine(id))
	}
	for _, u := range v.Assumed {
		lines = append(lines, u.Line())
	}
	return lines
}

// Warn adds the lines to a stage as warnings (sections 12 and 15.6).
func (v Vital) Warn(s *report.Stage, assumed bool) {
	s.Warnf("%s", v.ProofsLine())
	for _, id := range v.Proven {
		s.Warnf("%s", ProvenLine(id))
	}
	if assumed {
		for _, u := range v.Assumed {
			s.Warnf("%s", u.Line())
		}
	}
}
