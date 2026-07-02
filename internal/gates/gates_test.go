package gates

import (
	"path/filepath"
	"testing"
)

// repoRoot resolves to the repo root (the test runs from internal/gates).
func repoRoot() string { return "../.." }

func TestCheckC4CleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckC4(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G2 errors: %v", ex, g.Errs)
			}
			if g.Counts["boundaries"] == 0 {
				t.Errorf("%s: no boundaries parsed", ex)
			}
		})
	}
}

func TestCheckMachinesCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckMachines(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G3 errors: %v", ex, g.Errs)
			}
			if len(g.Drift) != 0 {
				t.Errorf("%s: G3 drift: %v", ex, g.Drift)
			}
			if g.Counts["machines"] == 0 {
				t.Errorf("%s: no machines parsed", ex)
			}
		})
	}
}

func TestCheckTraceabilityCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckTraceability(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: Gx errors: %v", ex, g.Errs)
			}
		})
	}
}

func TestCheckImportsCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	impl := filepath.Join(repoRoot(), "examples", "go-crm", "impl")
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Errorf("G4 errors: %v", g.Errs)
	}
	if g.Counts["go files checked"] == 0 {
		t.Error("no go files checked")
	}
}

func TestTokenInWholeToken(t *testing.T) {
	if !tokenIn("inv-1", "foo inv-1 bar") {
		t.Error("inv-1 should match standalone")
	}
	if tokenIn("inv-1", "foo inv-12 bar") {
		t.Error("inv-1 must not match inside inv-12")
	}
}

func TestGateEmitFormatting(t *testing.T) {
	g := NewGate("Test")
	g.Count("widgets")
	g.Count("widgets")
	g.Count("gadgets", 3)
	if g.Counts["widgets"] != 2 || g.Counts["gadgets"] != 3 {
		t.Errorf("counts wrong: %+v", g.Counts)
	}
	// order must be insertion order
	if g.countOrder[0] != "widgets" || g.countOrder[1] != "gadgets" {
		t.Errorf("count order wrong: %v", g.countOrder)
	}
}
