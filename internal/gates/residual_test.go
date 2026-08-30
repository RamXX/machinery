package gates

import (
	"os"
	"path/filepath"
	"testing"
)

const residualModel = `kind: DomainModel
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
`

const residualMachine = `{
  "id": "Deal",
  "initial": "Open",
  "states": {
    "Open": {
      "on": {"win": {"target": "saving"}}
    },
    "saving": {
      "invoke": {"src": "saveDeal", "onDone": {"target": "Won"}, "onError": {"target": "Open"}},
      "after": {"5000": {"target": "Open"}}
    },
    "Won": {"type": "final"}
  }
}`

// residualFixture builds a Gx-ready design (model, machine, placement) whose
// mitigation table carries the given handled-by cells.
func residualFixture(t *testing.T, mitigationTable string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), residualModel)
	mustWrite(t, filepath.Join(design, "machines", "Deal.machine.json"), residualMachine)
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Deal` | in-memory |\n\n" + mitigationTable +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| deal-won-final | `Deal` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

func TestResidualHandlerColumnResolves(t *testing.T) {
	g := residualFixture(t, "## Mitigations\n\n| dependency | failure modes | mitigation | handled by |\n|---|---|---|---|\n"+
		"| `store` | unavailable | retry | `saveDeal` |\n"+
		"| `queue` | redelivery | dedupe | `Deal` |\n")
	if hasErr(g, "residual handler") {
		t.Fatalf("resolvable handlers must pass: %v", g.Errs)
	}
	if g.Counts["mitigation rows with residual handlers"] != 2 {
		t.Fatalf("handler rows not counted: %+v", g.Counts)
	}
}

func TestResidualHandlerColumnAbsentCarriesNoObligation(t *testing.T) {
	g := residualFixture(t, "## Mitigations\n\n| dependency | failure modes | mitigation |\n|---|---|---|\n"+
		"| `store` | unavailable | retry |\n")
	if hasErr(g, "residual") {
		t.Fatalf("no handled-by column means no obligation: %v", g.Errs)
	}
}

func TestResidualHandlerUnresolvableErrors(t *testing.T) {
	g := residualFixture(t, "## Mitigations\n\n| dependency | failure modes | mitigation | handled by |\n|---|---|---|---|\n"+
		"| `store` | unavailable | retry | `saveOrder` |\n")
	if !hasErr(g, "residual handler `saveOrder`") {
		t.Fatalf("an unresolvable handler must error: %v", g.Errs)
	}
}

func TestResidualHandlerEmptyCellNeedsWaiver(t *testing.T) {
	g := residualFixture(t, "## Mitigations\n\n| dependency | failure modes | mitigation | handled by |\n|---|---|---|---|\n"+
		"| `store` | corrupt | restore from backup |  |\n")
	if !hasErr(g, "names no residual handler") {
		t.Fatalf("an empty handler cell must error: %v", g.Errs)
	}
	g = residualFixture(t, "## Mitigations\n\n| dependency | failure modes | mitigation | handled by |\n|---|---|---|---|\n"+
		"| `store` | corrupt | restore from backup | (no residual: fatal and loud; the process exits) |\n")
	if hasErr(g, "residual") {
		t.Fatalf("a reasoned waiver must pass: %v", g.Errs)
	}
	if g.Counts["residual handlers waived"] != 1 {
		t.Fatalf("waiver not counted: %+v", g.Counts)
	}
}
