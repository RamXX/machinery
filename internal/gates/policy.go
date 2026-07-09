// Gp-policy: the static relational model gate. Deterministic half only (no
// Java, no solver): the policy annotation parses, reconciles against the
// domain model, covers every top-level invariant, and the committed
// Policy.als byte-matches a fresh generation. Running the solver on the
// model is verify-formal's job, exactly as TLC is for the machines.

package gates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RamXX/machinery/internal/alloy"
)

// HasPolicyAnnotation reports whether the design opted into the relational
// layer (the annotation file exists).
func HasPolicyAnnotation(design string) bool {
	fi, err := os.Stat(filepath.Join(design, "formal", alloy.AnnotationName))
	return err == nil && !fi.IsDir()
}

// CheckPolicy implements Gp-policy.
func CheckPolicy(design string) *Gate {
	g := NewGate("Gp-policy static relational model")
	g.startOrder()
	annPath := filepath.Join(design, "formal", alloy.AnnotationName)
	if !HasPolicyAnnotation(design) {
		g.Errs = append(g.Errs, "no formal/"+alloy.AnnotationName+" in the design; the relational layer was requested but never authored (author the annotation, or drop gp from the gate list)")
		return g
	}
	domainPath, _, err := alloy.Paths(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	als, oracleMD, stats, err := alloy.GenerateAll(domainPath, annPath)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	g.Count("roles", stats.Roles)
	g.Count("resources", stats.Resources)
	g.Count("policy rules", stats.Rules)
	if stats.Residuals > 0 {
		g.Count("residuals (waived with reason)", stats.Residuals)
	}
	g.Count("invariants carried", stats.Carried)
	g.Count("solver commands generated", len(stats.Commands))
	g.Count("authz oracle rows generated", stats.OracleRows)

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
	fresh(alloy.OutputName, als, "model")
	fresh(alloy.OracleName, oracleMD, "authz oracle")
	g.RequireNonzero("policy rules", "the annotation compiled no rules")
	return g
}
