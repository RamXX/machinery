// Every-match table locators. The event-contract and interface-contract scans
// have taken EVERY header-matching table since the PACK-1 lesson; the
// mitigation and placement locators still took the first and broke, so a
// second table of either kind carried no obligation at all. These tests pin
// both halves of the fix: a later table's rows are read (they can fail), and
// its coverage counts (they can satisfy).

package gates

import (
	"os"
	"path/filepath"
	"testing"
)

// twoMitigationTableFixture runs G2 over a design declaring two Database
// containers, whose mitigation posture is split across two tables.
func twoMitigationTableFixture(t *testing.T, second string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n" +
		"      db = container \"Store\" \"state\" \"Postgres\" \"Database\"\n" +
		"      legacydb = container \"Legacy Store\" \"old state\" \"MySQL\" \"Database\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\ndependency_rules:\n  allow: []\n  deny: []\n```\n" +
		"\n## Mitigations\n\n| dependency | failure modes | mitigation |\n|---|---|---|\n" +
		"| `db` | down | retry with backoff |\n" +
		"\n## Transition-only mitigations\n\n" + second + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

// The coverage half: a dependency mitigated only in the SECOND table is
// covered. Under the first-match locator it was reported as unmitigated.
func TestMitigationSecondTableCovers(t *testing.T) {
	g := twoMitigationTableFixture(t,
		"| dependency | failure modes | mitigation |\n|---|---|---|\n| `legacydb` | down | read-only fallback |\n")
	if hasErr(g, "infrastructure dependency `legacydb` has no mitigation row") {
		t.Fatalf("a second mitigation table must cover its dependency: %v", g.Errs)
	}
	if g.Counts["mitigation rows"] != 2 {
		t.Fatalf("rows must aggregate across tables: %+v", g.Counts)
	}
}

// The obligation half: a row in the second table is READ, so a name it cannot
// resolve is a finding. Under the first-match locator its rows carried none.
func TestMitigationSecondTableRowsCarryObligations(t *testing.T) {
	g := twoMitigationTableFixture(t,
		"| dependency | failure modes | mitigation |\n|---|---|---|\n"+
			"| `legacydb` | down | read-only fallback |\n| `ghostdb` | down | none |\n")
	if !hasErr(g, "mitigation row names `ghostdb`") {
		t.Fatalf("the second table's rows must carry the resolution obligation: %v", g.Errs)
	}
}

// placementFixture runs Gx over a design with two entities whose placement is
// split across two tables.
func placementFixture(t *testing.T, second string) *Gate {
	return placementFixtureWithBuild(t, second, "")
}

// placementFixtureWithBuild is placementFixture plus an optional BUILD.md.
func placementFixtureWithBuild(t *testing.T, second, buildDoc string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	model := `kind: DomainModel
version: v1
enums:
  DealStage:
    values:
      - name: Open
      - name: Won
entities:
  Deal:
    attributes:
      - name: stage
        type: DealStage
    actions:
      - name: win
    invariants:
      - id: deal-won-final
        statement: a won deal stays won
  Note:
    attributes:
      - name: body
        type: String
`
	machine := `{
  "id": "Deal",
  "initial": "Open",
  "states": {
    "Open": {"on": {"win": {"target": "saving"}}},
    "saving": {
      "invoke": {"src": "saveDeal", "onDone": {"target": "Won"}, "onError": {"target": "Open"}},
      "after": {"5000": {"target": "Open"}}
    },
    "Won": {"type": "final"}
  }
}`
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	mustWrite(t, filepath.Join(design, "machines", "Deal.machine.json"), machine)
	mustWrite(t, filepath.Join(design, "machines", "Deal.matrix.md"),
		"| name | kind | maps to |\n|---|---|---|\n| `saveDeal` | actor | `deal-won-final` |\n")
	arch := "# A\n\n## Placement (aggregates)\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Deal` | in-memory |\n\n## Placement (the rest)\n\n" + second
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	if buildDoc != "" {
		mustWrite(t, filepath.Join(design, "BUILD.md"), buildDoc)
	}
	return CheckTraceability(design)
}

// The coverage half: an entity placed only in the SECOND table is placed.
func TestPlacementSecondTablePlaces(t *testing.T) {
	g := placementFixture(t, "| component (placement) | persistence |\n|---|---|\n"+
		"| `Note` (no machine: value-like, written with its Deal) | persisted rows |\n")
	if hasErr(g, "entity `Note` appears in no persistence-and-placement row") {
		t.Fatalf("a second placement table must place its entity: %v", g.Errs)
	}
	if g.Counts["entities placed"] != 2 {
		t.Fatalf("placements must aggregate across tables: %+v", g.Counts)
	}
}

// The obligation half: a row in the second table is READ, so a component with
// no machine and no waiver is a finding.
func TestPlacementSecondTableRowsCarryObligations(t *testing.T) {
	g := placementFixture(t, "| component (placement) | persistence |\n|---|---|\n"+
		"| `Note` | persisted rows |\n")
	if !hasErr(g, "placement row component `Note` has no machine") {
		t.Fatalf("the second table's rows must carry the machine obligation: %v", g.Errs)
	}
}

// The persisted count is the BUILD.md state-migration trigger, and it counts
// the second table's rows too: a persisted row there once demanded nothing.
func TestPlacementPersistedCountAggregatesAcrossTables(t *testing.T) {
	build := "Mode: full\n\n# Build\n\n## Toolchain\n\nGo 1.22.\n"
	// the first table's only row is in-memory, so the persisted row in the
	// SECOND table is the sole trigger for the state-migration obligation
	g := placementFixtureWithBuild(t, "| component (placement) | persistence |\n|---|---|\n"+
		"| `Note` (no machine: value-like, written with its Deal) | persisted rows |\n", build)
	if !hasErr(g, "placement row(s) persist machine state but BUILD.md has no State migration heading") {
		t.Fatalf("a persisted row in the second table must trigger the obligation: %v", g.Errs)
	}
	g = placementFixtureWithBuild(t, "| component (placement) | persistence |\n|---|---|\n"+
		"| `Note` (no machine: value-like, written with its Deal) | in-memory |\n", build)
	if hasErr(g, "State migration heading") {
		t.Fatalf("no persisted row means no obligation: %v", g.Errs)
	}
}
