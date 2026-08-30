package gates

import (
	"path/filepath"
	"testing"
)

// closureFixture builds a design with one Database-tagged element (store), a
// declared external, a mitigation table covering both, and the given
// adoption-closure section.
func closureFixture(t *testing.T, closureSection string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n" +
		"      store = container \"Store\" \"state\" \"PG\" \"Database\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: p.app\n    element: app\n    code: [\"app/**\"]\n" +
		"externals:\n  - id: external.vault\n    imports: [\"vault\"]\n```\n\n" +
		"## Mitigations\n\n| dependency | failure modes | mitigation |\n|---|---|---|\n" +
		"| `store` | unavailable | retry |\n| `external.vault` | unavailable | cache |\n" +
		closureSection + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

func TestClosureCoveredMembersPass(t *testing.T) {
	g := closureFixture(t, "\n## Adoption closures\n\n| technology | closure members | scorecard |\n|---|---|---|\n"+
		"| `PG` operator | `store`, `external.vault` | 7.5 (2026-08-01) |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("declared and mitigated members must pass: %v", g.Errs)
	}
	if g.Counts["closure members with mitigation rows"] != 2 {
		t.Fatalf("members not counted: %+v", g.Counts)
	}
	if g.Counts["scorecard entries"] != 1 {
		t.Fatalf("scorecard not counted: %+v", g.Counts)
	}
}

func TestClosureAbsentTableCarriesNoObligation(t *testing.T) {
	g := closureFixture(t, "")
	if hasErr(g, "closure") {
		t.Fatalf("no table means no obligation: %v", g.Errs)
	}
}

func TestClosureUndeclaredMemberErrors(t *testing.T) {
	g := closureFixture(t, "\n## Adoption closures\n\n| technology | closure members |\n|---|---|\n"+
		"| something | `sidecar` |\n")
	if !hasFinding(g.Errs, "closure member `sidecar`", "must be declared") {
		t.Fatalf("an undeclared member must error: %v", g.Errs)
	}
}

func TestClosureUnmitigatedMemberErrors(t *testing.T) {
	// app is a declared element but has no mitigation row
	g := closureFixture(t, "\n## Adoption closures\n\n| technology | closure members |\n|---|---|\n"+
		"| something | `app` |\n")
	if !hasFinding(g.Errs, "closure member `app`", "no mitigation row") {
		t.Fatalf("an unmitigated member must error: %v", g.Errs)
	}
}

func TestClosureWaiverAndScorecardShapes(t *testing.T) {
	g := closureFixture(t, "\n## Adoption closures\n\n| technology | closure members | scorecard |\n|---|---|---|\n"+
		"| lib-a | (no closure: a pure library, brings nothing) | n/a - internal code, not an OSS adoption |\n"+
		"| lib-b | `store` | 9 (2026-02-30) is not checked for calendar truth but shape |\n")
	if hasErr(g, "'lib-a'") {
		t.Fatalf("a reasoned waiver and n/a scorecard must pass: %v", g.Errs)
	}
	if !hasFinding(g.Errs, "'lib-b'", "neither") {
		t.Fatalf("a malformed scorecard cell must error: %v", g.Errs)
	}
	g = closureFixture(t, "\n## Adoption closures\n\n| technology | closure members | scorecard |\n|---|---|---|\n"+
		"| lib-c | `store` |  |\n")
	if !hasErr(g, "leaves scorecard empty") {
		t.Fatalf("an empty scorecard cell must error: %v", g.Errs)
	}
	g = closureFixture(t, "\n## Adoption closures\n\n| technology | closure members |\n|---|---|\n"+
		"| lib-d | (no closure: ) |\n")
	if !hasErr(g, "waives its closure with no reason") {
		t.Fatalf("an empty waiver reason must error: %v", g.Errs)
	}
}
