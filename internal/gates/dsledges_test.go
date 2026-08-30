package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// drawnEdgeFixture builds a design whose workspace.dsl holds the given model
// body (declarations and relationships) and whose contract binds boundaries
// alpha -> element a and beta -> element b, external ext -> element x, under
// the given dependency_rules block. Everything else G2 asks of a design (the
// interface contracts for the allow edges, the mitigation row for the
// external, the NFR record) is supplied, so a fixture built to exercise the
// drawn-edge check reports only that check's findings.
func drawnEdgeFixture(t *testing.T, modelBody, rules string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n" + modelBody + "  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    element: a\n    code: [\"a/**\"]\n" +
		"  - id: beta\n    element: b\n    code: [\"b/**\"]\n" +
		"externals:\n  - id: ext\n    element: x\n" +
		rules + "```\n" +
		"\n## Mitigations\n\n| dependency | failure modes | mitigation |\n|---|---|---|\n" +
		"| `ext` | down | fixture posture |\n| `x` | down | fixture posture |\n" +
		coveringInterfaceTable(rules) + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

// twoBoundaryModel declares a, b, and x with one drawn relationship a -> b.
const twoBoundaryModel = "    sys = softwareSystem \"S\" \"sys\" {\n" +
	"      a = container \"A\" \"one\" \"Go\"\n" +
	"      b = container \"B\" \"two\" \"Go\"\n" +
	"      x = container \"X\" \"store\" \"Postgres\" \"Database\"\n    }\n"

func TestDrawnEdgeAllowedPasses(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow:\n    - alpha -> beta\n  deny: []\n")
	if len(g.Errs) != 0 {
		t.Fatalf("an allowed drawn edge must pass cleanly: %v", g.Errs)
	}
	if g.Counts["drawn edges verified"] != 1 {
		t.Fatalf("verified edge not counted: %+v", g.Counts)
	}
}

func TestDrawnEdgeDeniedErrors(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow: []\n  deny:\n    - alpha -> beta\n")
	if !hasErr(g, "which the contract denies") {
		t.Fatalf("a denied drawn edge must error: %v", g.Errs)
	}
	if !hasErr(g, "workspace.dsl:8") {
		t.Fatalf("the finding must name the line the edge is drawn on: %v", g.Errs)
	}
}

// A wildcard deny is matched the way G4 matches it, so the diagram cannot
// escape a rule the code could not.
func TestDrawnEdgeDeniedByWildcardErrors(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> x \"writes\"\n",
		"dependency_rules:\n  allow: []\n  deny:\n    - \"* -> ext\"\n")
	if !hasErr(g, "which the contract denies") {
		t.Fatalf("a wildcard deny must catch the drawn edge: %v", g.Errs)
	}
}

func TestDrawnEdgeUndeclaredErrors(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow: []\n  deny: []\n")
	if !hasErr(g, "which no dependency rule covers") {
		t.Fatalf("a drawn edge no rule covers must error: %v", g.Errs)
	}
}

// baseline is the ratchet's amnesty in G4 and it is the same amnesty here; the
// ratchet itself has nothing to say about a drawn edge (it snapshots offender
// files, and a diagram has none).
func TestDrawnEdgeBaselinedIsToleratedNotAnError(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow: []\n  deny:\n    - alpha -> beta\n  baseline:\n    - alpha -> beta\n")
	if len(g.Errs) != 0 {
		t.Fatalf("a baselined drawn edge is tolerated debt, not a finding: %v", g.Errs)
	}
	if g.Counts["drawn edges baselined"] != 1 {
		t.Fatalf("baselined edge not counted: %+v", g.Counts)
	}
}

// allow+baseline is already ONE G2 finding on the rules; the drawn edge must
// not turn it into a second.
func TestDrawnEdgeAllowedAndBaselinedReportsOnce(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow:\n    - alpha -> beta\n  deny: []\n  baseline:\n    - alpha -> beta\n")
	n := 0
	for _, e := range g.Errs {
		if strings.Contains(e, "both allowed and baselined") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the contradiction must be reported exactly once, got %d: %v", n, g.Errs)
	}
	if hasErr(g, "the diagram draws") {
		t.Fatalf("the drawn edge must not add a second finding: %v", g.Errs)
	}
}

func TestDrawnEdgeUnknownEndpointErrors(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> ghost \"calls\"\n",
		"dependency_rules:\n  allow:\n    - alpha -> beta\n  deny: []\n")
	if !hasErr(g, "names 'ghost', which no element declares") {
		t.Fatalf("an endpoint no element declares must error: %v", g.Errs)
	}
}

// A declared element the contract never claimed (a person, a context box, a
// container outside the contract) is outside the dependency vocabulary: no
// rule could speak about it, so it carries no obligation. The count keeps the
// unjudged remainder visible instead of silent.
func TestDrawnEdgeOutsideContractVocabularyCarriesNoObligation(t *testing.T) {
	model := "    user = person \"User\" \"someone\"\n" + twoBoundaryModel + "    user -> a \"uses\"\n"
	g := drawnEdgeFixture(t, model, "dependency_rules:\n  allow: []\n  deny: []\n")
	if len(g.Errs) != 0 {
		t.Fatalf("an unbound endpoint carries no obligation: %v", g.Errs)
	}
	if g.Counts["drawn edges outside the contract vocabulary"] != 1 {
		t.Fatalf("the unjudged edge must stay visible in the counts: %+v", g.Counts)
	}
}

// Two elements of the SAME boundary are not a crossing, exactly as in G4.
func TestDrawnEdgeWithinOneBoundaryIsNotACrossing(t *testing.T) {
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      a = container \"A\" \"one\" \"Go\"\n    }\n    a -> a \"self\"\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    element: a\n    code: [\"a/**\"]\ndependency_rules:\n  allow: []\n  deny: []\n```\n" + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckC4(design)
	if hasErr(g, "the diagram draws") {
		t.Fatalf("an edge inside one boundary is no crossing: %v", g.Errs)
	}
}

// The relationship parser reads relationship lines and nothing else: a view
// expression that happens to contain an arrow is not a drawn edge, and `this`
// is the DSL's self-reference keyword rather than a name.
func TestDslRelationshipParsingIsAnchored(t *testing.T) {
	text := "  model {\n    a = container \"A\" \"one\" \"Go\"\n" +
		"    a -> b \"calls\"\n" +
		"    b->c\n" +
		"    this -> d \"from inside a block\"\n" +
		"  }\n  views {\n    include a -> b\n    exclude x -> y\n    autoLayout lr\n  }\n"
	got := dslRelationships(text)
	var pairs []string
	for _, r := range got {
		pairs = append(pairs, r.Src+"->"+r.Dst)
	}
	want := []string{"a->b", "b->c", "this->d"}
	if strings.Join(pairs, ",") != strings.Join(want, ",") {
		t.Fatalf("relationship parse = %v, want %v", pairs, want)
	}
	if got[0].Line != 3 {
		t.Fatalf("line number = %d, want 3", got[0].Line)
	}
}

func TestDrawnEdgeThisKeywordCarriesNoObligation(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    this -> b \"calls\"\n",
		"dependency_rules:\n  allow: []\n  deny: []\n")
	if len(g.Errs) != 0 {
		t.Fatalf("`this` names no element and carries no obligation: %v", g.Errs)
	}
}

// A hierarchical identifier names the same element as its last segment, the
// fallback the boundary binding already uses.
func TestDrawnEdgeHierarchicalIdentifierResolves(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    sys.a -> sys.b \"calls\"\n",
		"dependency_rules:\n  allow:\n    - alpha -> beta\n  deny: []\n")
	if len(g.Errs) != 0 {
		t.Fatalf("a hierarchical identifier must resolve: %v", g.Errs)
	}
	if g.Counts["drawn edges verified"] != 1 {
		t.Fatalf("verified edge not counted: %+v", g.Counts)
	}
}

// The converse is deliberately NOT required: a diagram is legitimately
// partial, so an allow rule nothing draws is no finding.
func TestAllowRuleWithNoDrawnEdgeIsNotAFinding(t *testing.T) {
	g := drawnEdgeFixture(t, twoBoundaryModel+"    a -> b \"calls\"\n",
		"dependency_rules:\n  allow:\n    - alpha -> beta\n    - beta -> ext\n  deny: []\n")
	if hasErr(g, "the diagram draws") || hasErr(g, "no drawn") {
		t.Fatalf("an undrawn allow edge is no finding: %v", g.Errs)
	}
}
