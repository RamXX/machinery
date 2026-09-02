package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// eventCellFixture builds a design declaring components app and gw plus the
// external store (element db), and runs G2 over the given event section.
func eventCellFixture(t *testing.T, eventSection string) *Gate {
	t.Helper()
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n" +
		"      gw = component \"Gateway\" \"edge\" \"Go\"\n" +
		"      db = container \"Store\" \"state\" \"Postgres\" \"Database\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\n" +
		"externals:\n  - id: external.store\n    element: db\n" +
		"dependency_rules:\n  allow: []\n  deny: []\n```\n" +
		"\n## Mitigations\n\n| dependency | failure modes | mitigation |\n|---|---|---|\n" +
		"| `external.store` | down | fixture posture |\n| `db` | down | fixture posture |\n" +
		"\n## Events\n\nSource: swept the emit call sites and the topic bindings.\n" +
		eventSection + nfrStub
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckC4(design)
}

const fullEventHeader = "\n| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n"

func TestEventCellsCompleteRowPasses(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once | per order | Order.id |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("a complete row must pass cleanly: %v", g.Errs)
	}
	if g.Counts["event-contract cells answered"] != 6 {
		t.Fatalf("answered cells not counted: %+v", g.Counts)
	}
	if g.Counts["event-contract participants resolved"] != 2 {
		t.Fatalf("resolved participants not counted: %+v", g.Counts)
	}
}

func TestEventCellsMissingColumnErrorsOncePerTable(t *testing.T) {
	g := eventCellFixture(t, "\n| event | producer | consumer | payload | delivery |\n|---|---|---|---|---|\n"+
		"| paid | app | gw | Order.id | at-least-once |\n"+
		"| shipped | app | gw | Order.id | at-least-once |\n")
	n := 0
	for _, e := range g.Errs {
		if strings.Contains(e, "has no ordering column") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("a missing column is one finding per table, got %d: %v", n, g.Errs)
	}
	if !hasErr(g, "has no dedupe column") {
		t.Fatalf("every missing column must be reported: %v", g.Errs)
	}
}

func TestEventCellsEmptyCellErrors(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once |  | Order.id |\n")
	if !hasErr(g, "empty ordering cell") {
		t.Fatalf("an empty required cell must error: %v", g.Errs)
	}
}

// The format documents no placeholder token, so an explicit "none" or "n/a"
// is an ANSWER and passes; only an empty cell is an unanswered question. The
// one exception is the dedupe cell under at-least-once delivery, pinned
// separately below.
func TestEventCellsExplicitNoneIsAnAnswer(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | exactly-once | none | n/a |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("an explicit none/n-a is an answer: %v", g.Errs)
	}
}

// At-least-once delivery promises duplicates, so a bare no-answer dedupe cell
// contradicts the row; a reasoned "none (...)" or a named key passes.
func TestEventCellsAtLeastOnceBareDedupe(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once | none | n/a |\n")
	if !hasErr(g, "delivery is at-least-once but dedupe is a bare") {
		t.Fatalf("a bare dedupe under at-least-once must error: %v", g.Errs)
	}
	g = eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once | none | none (idempotent consumer: upsert by Order.id) |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("a reasoned none is an answer: %v", g.Errs)
	}
	g = eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once | none | Order.id |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("a named dedupe key passes: %v", g.Errs)
	}
}

func TestEventCellsUnresolvableParticipantErrors(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | billing | gw | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "producer 'billing' is neither a workspace.dsl element nor a declared external") {
		t.Fatalf("an unresolvable producer must error: %v", g.Errs)
	}
	g = eventCellFixture(t, fullEventHeader+
		"| paid | app | billing | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "consumer 'billing' is neither") {
		t.Fatalf("an unresolvable consumer must error: %v", g.Errs)
	}
}

// The resolution universe is the mitigation row's: workspace.dsl elements plus
// declared externals, by id or by the element they bind.
func TestEventCellsExternalsResolve(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | external.store | Order.id | at-least-once | none | Order.id |\n"+
		"| read | db | app | Order.id | at-least-once | none | Order.id |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("an external id and its element must both resolve: %v", g.Errs)
	}
}

// Annotations live in parentheses and backticks are stripped, the one cell
// grammar every gate here reads a cell by.
func TestEventCellsAnnotationsAndBackticksResolve(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | `app` (via its outbox) | gw (SSE) | Order.id | at-least-once | none | Order.id |\n")
	if len(g.Errs) != 0 {
		t.Fatalf("an annotated, backticked cell must resolve: %v", g.Errs)
	}
}

// Rows are numbered cumulatively across EVERY event-contract table, the way
// pack generation numbers them, so a finding in the second table is
// addressable and no table is silently excused.
func TestEventCellsSecondTableIsRead(t *testing.T) {
	g := eventCellFixture(t, fullEventHeader+
		"| paid | app | gw | Order.id | at-least-once | none | Order.id |\n"+
		"\n### More events\n\nSource: the same sweep.\n"+fullEventHeader+
		"| shipped | app | billing | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "event-contract row 2 (event 'shipped'): consumer 'billing'") {
		t.Fatalf("the second table's rows must be read and numbered cumulatively: %v", g.Errs)
	}
}

// A design with no event-contract table carries no cell obligation at all.
func TestEventCellsNoTableCarriesNoObligation(t *testing.T) {
	g := eventCellFixture(t, "\n| thing | note |\n|---|---|\n| x | y |\n")
	if hasErr(g, "event-contract row") {
		t.Fatalf("a non-event table carries no obligation: %v", g.Errs)
	}
}

// An annotation may itself contain parentheses. Stripping only up to the first
// ")" left a dangling ")" glued to the participant name, so a perfectly
// well-formed cell resolved to nothing and the row failed with "annotations
// only in parentheses" while the annotation WAS in parentheses.
func TestEventCellsNestedAnnotationsResolve(t *testing.T) {
	cases := []struct {
		name string
		row  string
		errs bool
	}{
		{
			name: "a nested annotation on the consumer",
			row:  "| paid | app | gw (delivery lane (durable)) | Order.id | at-least-once | none | Order.id |\n",
		},
		{
			name: "a nested annotation on the producer",
			row:  "| paid | app (the write path (one tx)) | gw | Order.id | at-least-once | none | Order.id |\n",
		},
		{
			name: "two annotations, the second nested",
			row:  "| paid | app | gw (lane) (no machine: an insert (not a transition)) | Order.id | at-least-once | none | Order.id |\n",
		},
		{
			name: "a cell that is annotation all the way down still errors",
			row:  "| paid | app | (nothing but (an annotation)) | Order.id | at-least-once | none | Order.id |\n",
			errs: true,
		},
		{
			name: "an unresolvable name behind a nested annotation still errors",
			row:  "| paid | billing (the biller (v2)) | gw | Order.id | at-least-once | none | Order.id |\n",
			errs: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := eventCellFixture(t, fullEventHeader+tc.row)
			if tc.errs {
				if len(g.Errs) == 0 {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if len(g.Errs) != 0 {
				t.Fatalf("a balanced annotation must resolve: %v", g.Errs)
			}
			if g.Counts["event-contract participants resolved"] != 2 {
				t.Fatalf("both participants must resolve, counted %d", g.Counts["event-contract participants resolved"])
			}
		})
	}
}
