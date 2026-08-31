package gates

import (
	"os"
	"path/filepath"
	"testing"
)

const clauseCovOracle = `# Oracle: Deal

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-DEAL-01 | DEAL-abc123 | Lead | on:win | guardCanWin | Won | commit |
| T-DEAL-02 | DEAL-def456 | Lead | on:lose | - | Lost | commit |
`

// clauseCovFixture builds a design whose matrix declares guardCanWin's two
// clauses, plus an impl whose one test file carries the given text.
func clauseCovFixture(t *testing.T, testBody string) (string, string) {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "machines", "Deal.oracle.md"), clauseCovOracle)
	mustWrite(t, filepath.Join(design, "machines", "Deal.matrix.md"),
		"| `guardCanWin` | guard | CLAUSES{owned-record, open-stage} |\n")
	impl := t.TempDir()
	mustWrite(t, filepath.Join(impl, "deal_test.go"), testBody)
	return design, impl
}

func TestClauseCoverageEmptyDeclarationErrors(t *testing.T) {
	// CLAUSES{} announces the falsifying-test obligation and then arms
	// nothing; silently skipping it once dropped the obligation without a
	// trace. The declaration must either list clauses or go.
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "machines", "Deal.oracle.md"), clauseCovOracle)
	mustWrite(t, filepath.Join(design, "machines", "Deal.matrix.md"),
		"| `guardCanWin` | guard | CLAUSES{} |\n")
	impl := t.TempDir()
	mustWrite(t, filepath.Join(impl, "deal_test.go"), "package x\n// covers DEAL-abc123 DEAL-def456\n")
	g := CheckOracleCoverage(design, impl)
	if !hasErr(g, "declares CLAUSES{} with no clauses") {
		t.Fatalf("an empty CLAUSES declaration must error: %v", g.Errs)
	}
}

func TestClauseCoverageComplete(t *testing.T) {
	design, impl := clauseCovFixture(t,
		"package x\n// covers DEAL-abc123 DEAL-def456 DEAL-abc123a DEAL-abc123b\n")
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("complete clause coverage must pass: %v", g.Errs)
	}
	if g.Counts["falsifying-clause ids covered"] != 2 {
		t.Fatalf("suffixed ids not counted: %+v", g.Counts)
	}
	if g.Counts["clause-declared guards checked"] != 1 {
		t.Fatalf("declared guard not counted: %+v", g.Counts)
	}
}

func TestClauseCoverageMissingSuffixErrors(t *testing.T) {
	design, impl := clauseCovFixture(t,
		"package x\n// covers DEAL-abc123 DEAL-def456 DEAL-abc123a\n")
	g := CheckOracleCoverage(design, impl)
	if !hasErr(g, "misses falsifying test(s) DEAL-abc123b") {
		t.Fatalf("a missing suffixed id must error: %v", g.Errs)
	}
}

func TestClauseCoverageWholesaleParseDoesNotDischarge(t *testing.T) {
	// a conformance parse covers the base rows by construction, but the
	// falsifying-clause tests are exactly what the table cannot derive
	design, impl := clauseCovFixture(t,
		"package x\nvar oraclePath = \"Deal.oracle.md\"\nvar row = \"a | b\"\n")
	g := CheckOracleCoverage(design, impl)
	if !hasErr(g, "misses falsifying test(s) DEAL-abc123a, DEAL-abc123b") {
		t.Fatalf("wholesale citation must not discharge clause coverage: %v", g.Errs)
	}
}

func TestClauseCoverageUndeclaredGuardCarriesNoObligation(t *testing.T) {
	design, impl := clauseCovFixture(t,
		"package x\n// covers DEAL-abc123 DEAL-def456\n")
	// drop the declaration: no CLAUSES, no obligation
	mustWrite(t, filepath.Join(design, "machines", "Deal.matrix.md"),
		"| `guardCanWin` | guard | owned and open |\n")
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("an undeclared guard carries no clause obligation: %v", g.Errs)
	}
}

func TestClauseCoverageUnguardedRowsExempt(t *testing.T) {
	// DEAL-def456's row has no guard, so guardCanWin's clauses never bind it
	design, impl := clauseCovFixture(t,
		"package x\n// covers DEAL-abc123 DEAL-def456 DEAL-abc123a DEAL-abc123b\n")
	g := CheckOracleCoverage(design, impl)
	for _, e := range g.Errs {
		if idTokenIn("DEAL-def456", e) {
			t.Fatalf("an unguarded row must carry no clause obligation: %v", g.Errs)
		}
	}
}
