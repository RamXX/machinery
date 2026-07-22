// Gi-integrity: the static relational INTEGRITY gate. Deterministic half only
// (no Java, no solver): the integrity annotation parses, reconciles against the
// domain model, and the committed Integrity.als byte-matches a fresh
// generation. Running the admissibility runs on the model is verify-formal's
// job, exactly as TLC is for the machines and the solver is for Gp-policy.

package gates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/version"
)

// HasIntegrityAnnotation reports whether the design opted into the integrity
// layer (the annotation file exists).
func HasIntegrityAnnotation(design string) bool {
	fi, err := os.Stat(filepath.Join(design, "formal", alloy.IntegrityAnnotationName))
	return err == nil && !fi.IsDir()
}

// CheckIntegrity implements Gi-integrity.
func CheckIntegrity(design string) *Gate {
	g := NewGate("Gi-integrity structural relational model")
	g.startOrder()
	annPath := filepath.Join(design, "formal", alloy.IntegrityAnnotationName)
	if !HasIntegrityAnnotation(design) {
		g.Errs = append(g.Errs, "no formal/"+alloy.IntegrityAnnotationName+" in the design; the integrity layer was requested but never authored (author the annotation, or drop gi from the gate list)")
		return g
	}
	domainPath, _, err := alloy.Paths(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	als, stats, err := alloy.GenerateIntegrity(domainPath, annPath)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	g.Count("entities", stats.Entities)
	if stats.Relationships > 0 {
		g.Count("relationships", stats.Relationships)
	}
	if stats.Uniques > 0 {
		g.Count("unique keys", stats.Uniques)
	}
	if stats.Singletons > 0 {
		g.Count("singleton flags", stats.Singletons)
	}
	if stats.Residuals > 0 {
		g.Count("residuals (waived with reason)", stats.Residuals)
	}
	g.Count("invariants carried", stats.Carried)
	g.Count("solver commands generated", len(stats.Commands))

	committed := filepath.Join(design, "formal", alloy.IntegrityOutputName)
	raw, rerr := os.ReadFile(committed)
	if rerr == nil {
		g.recordStamp(string(raw))
	}
	switch {
	case rerr != nil:
		g.Drift = append(g.Drift, fmt.Sprintf("formal/%s is not committed; the annotation compiles but the model was never generated. Run 'machinery alloy %s' and commit the output.", alloy.IntegrityOutputName, design))
	case version.Strip(string(raw)) != version.Strip(als):
		// version stamps stripped from both sides: skew is a note, not DRIFT
		g.Drift = append(g.Drift, fmt.Sprintf("formal/%s is stale: it does not match a fresh generation from the domain model + annotation. Run 'machinery alloy %s' and commit the regenerated file; never hand-edit it.", alloy.IntegrityOutputName, design))
	default:
		g.Count("committed artifacts fresh (byte-identical)")
	}
	g.RequireNonzero("invariants carried", "the annotation carried no invariants")
	return g
}
