package experiments

// Runners for the consistency-layer Stage 1 experiments: the declaration
// grammar (WRITES, USES, CARRIES, SUPERSEDES) is closed and parsed, and a
// backticked fact outside every group is visible. Each mutation is applied to
// the synthetic fixture design and paired with a near-neighbour that must not
// fire, so an over-eager rule fails here as loudly as a missing one.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func init() {
	RegisterRunner("declarations_test.go",
		"malformed-writes-declaration", "unknown-declaration-group", "undeclared-fact-reference")
}

func experimentNamed(t *testing.T, name string) Experiment {
	t.Helper()
	for _, e := range All() {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("experiment %s is not declared", name)
	return Experiment{}
}

// mutateSaveWidgetContract replaces the saveWidget contract cell.
func mutateSaveWidgetContract(t *testing.T, design, contract string) {
	t.Helper()
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"| atomic persist |", "| "+contract+" |")
}

func TestMalformedWritesDeclarationIsError(t *testing.T) {
	e := experimentNamed(t, "malformed-writes-declaration")
	design, _ := fixture(t)
	mutateSaveWidgetContract(t, design, "atomic persist. WRITES{Widget.status, Widget.status}")
	if g := gates.CheckTraceability(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gx: %v", e.Name, g.Errs)
	}

	near, _ := fixture(t)
	mutateSaveWidgetContract(t, near, "atomic persist. WRITES{Widget.status}")
	if g := gates.CheckTraceability(near); len(g.Errs) != 0 || g.Counts["declaration groups parsed"] != 1 {
		t.Fatalf("a well-formed WRITES group must parse clean: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestUnknownDeclarationGroupIsError(t *testing.T) {
	e := experimentNamed(t, "unknown-declaration-group")
	design, _ := fixture(t)
	mutateSaveWidgetContract(t, design, "atomic persist. MACHINE-WRITTEN{Widget.status}")
	if g := gates.CheckTraceability(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gx: %v", e.Name, g.Errs)
	}

	near, _ := fixture(t)
	mutateSaveWidgetContract(t, near, "admits the row shape `WidgetRow{id, status}`; CARRIES{column:Widget.status}")
	if g := gates.CheckTraceability(near); len(g.Errs) != 0 {
		t.Fatalf("a CamelCase literal and a known group must not trip the rule: %v", g.Errs)
	}
}

func TestUndeclaredFactReferenceIsWarning(t *testing.T) {
	e := experimentNamed(t, "undeclared-fact-reference")
	design, _ := fixture(t)
	mutateSaveWidgetContract(t, design, "atomic persist of `Widget.status`")
	g := gates.CheckLedger(design)
	if !containsAny(g.Warns, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gl: %v", e.Name, g.Warns)
	}
	if len(g.Errs) != 0 {
		t.Fatalf("the tier is a warning, never an error: %v", g.Errs)
	}

	near, _ := fixture(t)
	mutateSaveWidgetContract(t, near, "atomic persist of `Widget.status`. WRITES{Widget.status}")
	for _, w := range gates.CheckLedger(near).Warns {
		if strings.Contains(w, "undeclared fact reference") {
			t.Fatalf("a fact the row declares is not undeclared: %s", w)
		}
	}
}
