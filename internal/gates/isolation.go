// Gn-isolation: the static relational ISOLATION gate. Deterministic half only
// (no Java, no solver): the isolation annotation parses, reconciles against the
// domain model, and the committed Isolation.als and Isolation.oracle.md
// byte-match a fresh generation. Running the checks on the model is
// verify-formal's job, exactly as for Gp-policy and Gi-integrity.

package gates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/alloy"
)

// HasIsolationAnnotation reports whether the design opted into the isolation
// layer (the annotation file exists).
func HasIsolationAnnotation(design string) bool {
	fi, err := os.Stat(filepath.Join(design, "formal", alloy.IsolationAnnotationName))
	return err == nil && !fi.IsDir()
}

// CheckIsolation implements Gn-isolation.
func CheckIsolation(design string) *Gate {
	g := NewGate("Gn-isolation multi-tenant relational model")
	g.startOrder()
	annPath := filepath.Join(design, "formal", alloy.IsolationAnnotationName)
	if !HasIsolationAnnotation(design) {
		g.Errs = append(g.Errs, "no formal/"+alloy.IsolationAnnotationName+" in the design; the isolation layer was requested but never authored (author the annotation, or drop gn from the gate list)")
		return g
	}
	domainPath, _, err := alloy.Paths(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	als, oracleMD, stats, err := alloy.GenerateIsolation(domainPath, annPath)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	g.Count("records", stats.Records)
	g.Count("references", stats.References)
	if stats.Residuals > 0 {
		g.Count("residuals (waived with reason)", stats.Residuals)
	}
	g.Count("invariants carried", stats.Carried)
	g.Count("solver commands generated", len(stats.Commands))
	g.Count("tenant oracle rows generated", stats.OracleRows)

	fresh := func(name, want, label string) {
		committed := filepath.Join(design, "formal", name)
		raw, rerr := os.ReadFile(committed)
		switch {
		case rerr != nil:
			g.Drift = append(g.Drift, fmt.Sprintf("formal/%s is not committed; the annotation compiles but the %s was never generated. Run 'machinery alloy %s' and commit the output.", name, label, design))
		case string(raw) != want:
			g.Drift = append(g.Drift, fmt.Sprintf("formal/%s is stale: it does not match a fresh generation from the domain model + annotation. Run 'machinery alloy %s' and commit the regenerated file; never hand-edit it.", name, design))
		default:
			g.Count("committed artifacts fresh (byte-identical)")
		}
	}
	fresh(alloy.IsolationOutputName, als, "model")
	fresh(alloy.IsolationOracleName, oracleMD, "tenant oracle")
	g.RequireNonzero("references", "the annotation compiled no references")
	return g
}
