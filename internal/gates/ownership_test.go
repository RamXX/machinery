package gates

import (
	"path/filepath"
	"testing"
)

const ownershipModel = `kind: DomainModel
version: v1
entities:
  Deal:
    actions:
      - name: create
        actor: Rep
      - name: close
        actor: Rep
  Task:
    actions:
      - name: open
`

// ownershipFixture builds a design with a two-entity model, a two-component
// DSL, and the given action-ownership table section.
func ownershipFixture(t *testing.T, table string, withModel bool) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      domain = component \"domain\" \"logic\" \"Go\"\n" +
		"      tasks = component \"tasks\" \"logic\" \"Go\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: p.domain\n    element: domain\n    code: [\"domain/**\"]\n```\n" + table + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	if withModel {
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), ownershipModel)
	}
	return CheckC4(design)
}

const ownershipHeader = "\n## Action ownership\n\n| action | owning component |\n|---|---|\n"

func TestOwnershipCompleteTablePasses(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+
		"| `Deal.create`, `Deal.close` | `domain` |\n"+
		"| `Task.open` | `tasks` |\n", true)
	if len(g.Errs) != 0 {
		t.Fatalf("a complete ownership table must pass: %v", g.Errs)
	}
	if g.Counts["actions owned"] != 3 {
		t.Fatalf("actions owned = %d, want 3: %+v", g.Counts["actions owned"], g.Counts)
	}
}

func TestOwnershipAbsentTableCarriesNoObligation(t *testing.T) {
	g := ownershipFixture(t, "", true)
	if hasErr(g, "action-ownership") || hasErr(g, "ownership row") {
		t.Fatalf("no table means no obligation: %v", g.Errs)
	}
}

func TestOwnershipMissingActionErrors(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+"| `Deal.create`, `Deal.close` | `domain` |\n", true)
	if !hasErr(g, "'Task.open' appears in no action-ownership row") {
		t.Fatalf("a missing action must error: %v", g.Errs)
	}
}

func TestOwnershipUnknownActionAndOwnerError(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+
		"| `Deal.create`, `Deal.close`, `Task.open` | `domain` |\n"+
		"| `Deal.reopen` | `ghost` |\n", true)
	if !hasErr(g, "unknown action 'Deal.reopen'") {
		t.Fatalf("an unknown action must error: %v", g.Errs)
	}
	if !hasErr(g, "owner `ghost`") {
		t.Fatalf("an unresolvable owner must error: %v", g.Errs)
	}
}

func TestOwnershipDuplicateActionErrors(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+
		"| `Deal.create` | `domain` |\n| `Deal.create`, `Deal.close`, `Task.open` | `domain` |\n", true)
	if !hasErr(g, "more than once") {
		t.Fatalf("a doubly-owned action must error: %v", g.Errs)
	}
}

func TestOwnershipWaiverNeedsReason(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+
		"| `Deal.create`, `Deal.close` | `domain` |\n"+
		"| `Task.open` | (unowned: ) |\n", true)
	if !hasErr(g, "names no reason") {
		t.Fatalf("an empty waiver reason must error: %v", g.Errs)
	}
	g = ownershipFixture(t, ownershipHeader+
		"| `Deal.create`, `Deal.close` | `domain` |\n"+
		"| `Task.open` | (unowned: a pure notification with no home component yet) |\n", true)
	if len(g.Errs) != 0 {
		t.Fatalf("a reasoned waiver must pass: %v", g.Errs)
	}
	if g.Counts["actions ownership-waived"] != 1 {
		t.Fatalf("waived action not counted: %+v", g.Counts)
	}
}

func TestOwnershipBoundaryIDAsOwner(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+
		"| `Deal.create`, `Deal.close`, `Task.open` | `p.domain` |\n", true)
	if len(g.Errs) != 0 {
		t.Fatalf("a contract boundary id is a valid owner: %v", g.Errs)
	}
}

func TestOwnershipTableWithoutModelErrors(t *testing.T) {
	g := ownershipFixture(t, ownershipHeader+"| `Deal.create` | `domain` |\n", false)
	if !hasErr(g, "domain model cannot be read") {
		t.Fatalf("a table with no model must error: %v", g.Errs)
	}
}
