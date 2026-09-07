package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestObligationParentDeletedRelationalOracleRemainsRequired(t *testing.T) {
	for _, tc := range []struct{ gate, annotation, model, oracle string }{
		{"gp", "policy.relational.yaml", "Policy.als", "Policy.oracle.md"},
		{"gn", "isolation.relational.yaml", "Isolation.als", "Isolation.oracle.md"},
	} {
		t.Run(tc.gate, func(t *testing.T) {
			design, impl := obligationParentFixture(t)
			if err := os.Remove(filepath.Join(design, "checkout.modelith.yaml")); err != nil {
				t.Fatal(err)
			}
			for _, rel := range []string{"domain.modelith.yaml", "formal/" + tc.annotation, "formal/" + tc.model, "formal/" + tc.oracle} {
				body, err := os.ReadFile(filepath.Join(repoRoot(), "examples", "go-crm", "design", rel))
				if err != nil {
					t.Fatal(err)
				}
				writeSuiteFile(t, filepath.Join(design, rel), string(body))
			}
			// Positive relational generation is checked first. The copied
			// parent architecture is not a claim about the CRM domain model.
			check := CheckPolicy
			if tc.gate == "gn" {
				check = CheckIsolation
			}
			requireObligationClean(t, check(design))
			if err := os.Remove(filepath.Join(design, "formal", tc.oracle)); err != nil {
				t.Fatal(err)
			}
			sel, run, _, err := SelectRunAndNote(design, impl, "", RunOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !sel.Run[tc.gate] || !sel.Run["gt"] {
				t.Fatalf("deleting generated output erased required gates: %v", sel.Run)
			}
			found := false
			for _, g := range run {
				if strings.HasPrefix(strings.ToLower(g.Title), tc.gate+"-") {
					for _, drift := range g.Drift {
						found = found || strings.Contains(drift, tc.oracle+" is not committed")
					}
				}
			}
			if !found {
				t.Fatal("default execution must block specifically on the removed required oracle")
			}
		})
	}
}

func TestObligationClausesSharedNarrativeOwnership(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "CLAUSES{owner, balance}")
	writeObligationTests(t, impl, append(obligationIDs(a, 2), obligationIDs(b, 2)...)...)
	writeSuiteFile(t, filepath.Join(design, "BUILD.md"), "Alpha guardReady checks owner and open.\nBeta guardReady checks owner and balance.\n")
	for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
		requireObligationClean(t, g)
	}
	writeSuiteFile(t, filepath.Join(design, "BUILD.md"), "guardReady checks owner.\n")
	gd := selectedObligationGates(t, design, impl, "gd")["gd"]
	if len(gd.Errs)+len(gd.Drift) != 0 || len(gd.Warns) != 1 {
		t.Fatalf("ambiguous narrative must produce one honest ownership warning: %+v", gd)
	}
	for _, required := range []string{"ambiguous machine ownership", "guardReady", "Alpha", "Beta"} {
		if !strings.Contains(gd.Warns[0], required) {
			t.Errorf("ownership warning missing %q: %v", required, gd.Warns)
		}
	}
	// Narrative ambiguity cannot discharge or add a coverage obligation.
	requireObligationClean(t, selectedObligationGates(t, design, impl, "gt")["gt"])
}
