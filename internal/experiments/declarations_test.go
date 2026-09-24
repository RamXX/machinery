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
		"malformed-writes-declaration", "unknown-declaration-group", "undeclared-fact-reference",
		"private-group-names-public-group", "misspelled-group-beside-private", "unresolved-backticked-token")
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
		"| atomic persist CARRIES{column:Widget.status} |", "| "+contract+" |")
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
	mutateSaveWidgetContract(t, design, "atomic persist. OWNED-BY{Widget.status}")
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

// declarePrivateGroups adds a private_groups: list to the fixture's contract.
func declarePrivateGroups(t *testing.T, design, list string) {
	t.Helper()
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "contract_version: 2\n", "contract_version: 2\nprivate_groups: "+list+"\n")
}

func TestPrivateGroupNamingPublicGroupIsError(t *testing.T) {
	e := experimentNamed(t, "private-group-names-public-group")
	design, _ := fixture(t)
	declarePrivateGroups(t, design, "[WRITES]")
	if g := gates.CheckC4(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped G2: %v", e.Name, g.Errs)
	}

	near, _ := fixture(t)
	declarePrivateGroups(t, near, "[OWNED-BY]")
	mutateSaveWidgetContract(t, near, "atomic persist. OWNED-BY{Widget.status} CARRIES{column:Widget.status}")
	if g := gates.CheckC4(near); containsAny(g.Errs, "private_groups") {
		t.Fatalf("a valid private_groups list must pass G2: %v", g.Errs)
	}
	g := gates.CheckTraceability(near)
	if len(g.Errs) != 0 || g.Counts["private declaration groups skipped"] != 1 {
		t.Fatalf("a declared private group must be skipped, visibly: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestMisspelledGroupBesidePrivateIsError(t *testing.T) {
	e := experimentNamed(t, "misspelled-group-beside-private")
	design, _ := fixture(t)
	declarePrivateGroups(t, design, "[OWNED-BY]")
	mutateSaveWidgetContract(t, design, "atomic persist. OWNED-BY{Widget.status} WRITE{Widget.status}")
	if g := gates.CheckTraceability(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gx: %v", e.Name, g.Errs)
	}

	near, _ := fixture(t)
	declarePrivateGroups(t, near, "[OWNED-BY]")
	mutateSaveWidgetContract(t, near, "atomic persist. OWNED-BY{Widget.status} WRITES{Widget.status} CARRIES{column:Widget.status}")
	if g := gates.CheckTraceability(near); len(g.Errs) != 0 {
		t.Fatalf("a declared private group beside a public one must parse clean: %v", g.Errs)
	}
}

func TestUnresolvedBacktickedTokenIsSofterWarning(t *testing.T) {
	e := experimentNamed(t, "unresolved-backticked-token")
	design, _ := fixture(t)
	mutateSaveWidgetContract(t, design, "atomic persist within `retry_budget`")
	g := gates.CheckLedger(design)
	if !containsAny(g.Warns, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gl: %v", e.Name, g.Warns)
	}
	if containsAny(g.Warns, "undeclared fact reference `retry_budget`") {
		t.Fatalf("a token naming no model attribute must not get the attribute wording: %v", g.Warns)
	}

	for _, quoted := range []string{"`Widget.publish`", "`Widget.saveWidget`", "`ARCHITECTURE.md`"} {
		near, _ := fixture(t)
		mutateSaveWidgetContract(t, near, "atomic persist, see "+quoted)
		if w := gates.CheckLedger(near).Warns; len(w) != 0 {
			t.Fatalf("%s names an action, a unit or a file and must not warn: %v", quoted, w)
		}
	}
}
