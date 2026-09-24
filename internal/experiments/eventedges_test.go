package experiments

// Event edges (MAC-j39j). The event-contract table is one row per
// producer-consumer edge, so one event spans several rows. The synthetic
// fan-out design below has one event on four rows (two producers, two
// consumers, per-edge payloads that differ by producer) plus a fifth row
// restating one edge verbatim. It must project, and each matrix payload {}
// binds to the edges its unit's component takes part in: the app unit's
// payload matches its own edges and not intake's, and passes.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func init() {
	RegisterRunner("eventedges_test.go",
		"rules-payload-edge-disagrees", "rules-payload-no-edge", "rules-payload-unknown-event",
		"event-edge-contradiction", "relationship-parallel-unnamed", "rules-partial-projection")
}

const fanOutModel = `  Lead:
    attributes:
      - name: channel
        type: string
    actions:
      - name: capture
  Audit:
    attributes:
      - name: note
        type: string
    actions:
      - name: record
`

// fanOutEdges is the event-contract table: widget.priced from app and from
// intake to storelib and to ledger, app's edges carrying the price and
// intake's the channel; the last row restates app->ledger.
const fanOutEdges = "\n## 5. Event contracts\n\n" +
	"| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n" +
	"| `widget.priced` | `app` | `storelib` | `Widget.id`, `Widget.price` | at-least-once | none | widget id |\n" +
	"| `widget.priced` | `app` | `ledger` | `Widget.id`, `Widget.price` | at-least-once | none | widget id |\n" +
	"| `widget.priced` | `intake` | `storelib` | `Widget.id`, `Widget.channel` | at-least-once | none | widget id |\n" +
	"| `widget.priced` | `intake` | `ledger` | `Widget.id`, `Widget.channel` | at-least-once | none | widget id |\n" +
	"| `widget.priced` | `app` | `ledger` | `Widget.id`, `Widget.price` | at-least-once | none | widget id |\n"

const fanOutOwnership = "\n## 5a. Action ownership\n\n| action | owning component |\n|---|---|\n" +
	"| `Widget.publish` | `app` |\n| `Lead.capture` | `intake` |\n| `Audit.record` | `archive` |\n"

const finalMachine = `{"id":"%s","initial":"Open","states":{"Open":{"type":"final"}}}`

func emittedEvents(rows ...string) string {
	return "\n## (b) Emitted events\n\n| name | kind | event | pre / post |\n|---|---|---|---|\n" + strings.Join(rows, "\n") + "\n"
}

// fanOutFixture writes the fan-out design over the widget fixture. leadPayload
// is the payload the intake component's emitting unit declares; extra rows
// are appended to the event-contract table.
func fanOutFixture(t *testing.T, leadPayload string, extraRows ...string) string {
	t.Helper()
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "widget.modelith.yaml"), "      - id: widget-owned\n", "      - id: widget-owned\n"+fanOutModel)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "\n## 6. Dependency mitigation posture",
		fanOutEdges+strings.Join(extraRows, "")+fanOutOwnership+"\n## 6. Dependency mitigation posture")
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"), "\n## (c) Transition matrix",
		emittedEvents("| `announce` | action | `widget.priced` | payload {Widget.id, Widget.price} |")+"\n## (c) Transition matrix")
	for _, m := range []string{"Lead", "Audit"} {
		mustWrite(t, filepath.Join(design, "machines", m+".machine.json"), strings.Replace(finalMachine, "%s", strings.ToLower(m), 1))
	}
	mustWrite(t, filepath.Join(design, "machines", "Lead.matrix.md"), "# Lead\n"+
		emittedEvents("| `emitPriced` | action | `widget.priced` | payload {"+leadPayload+"} |"))
	return design
}

func auditPayload(t *testing.T, design, event string) {
	t.Helper()
	mustWrite(t, filepath.Join(design, "machines", "Audit.matrix.md"), "# Audit\n"+
		emittedEvents("| `logPriced` | action | `"+event+"` | payload {Widget.id} |"))
}

// refuteProjectionErrors fails on any Gy-rules projection error or model
// finding.
func refuteProjectionErrors(t *testing.T, design string) {
	t.Helper()
	for _, f := range rulesFindings(t, design) {
		if strings.HasPrefix(f, "projection error") || strings.Contains(f, "no distinguishing role or name") {
			t.Fatalf("the design must project whole: %s", f)
		}
	}
}

// The fan-out design projects, the restated row collapses, and the app
// unit's payload, which matches app's edges and not intake's, passes; the
// intake unit declaring app's payload disagrees with its own edges.
func TestRulesPayloadEdgeDisagrees(t *testing.T) {
	design := fanOutFixture(t, "Widget.id, Widget.price")
	requireRuleFinding(t, "rules-payload-edge-disagrees", design)
	got := rulesFindings(t, design)
	if containsAny(got, "row 'Widget.announce': payload_") {
		t.Fatalf("the app unit matches its own edges and must pass: %v", got)
	}

	near := fanOutFixture(t, "Widget.id, Widget.channel")
	refuteProjectionErrors(t, near)
	refuteRuleFinding(t, near, "payload_twin")
	refuteRuleFinding(t, near, "payload_no_edge")
}

// A unit whose component takes part in no edge of the event binds to
// nothing; the same unit on an event of its own component would pass.
func TestRulesPayloadNoEdge(t *testing.T) {
	design := fanOutFixture(t, "Widget.id, Widget.channel")
	auditPayload(t, design, "widget.priced")
	requireRuleFinding(t, "rules-payload-no-edge", design)

	near := fanOutFixture(t, "Widget.id, Widget.channel",
		"| `widget.archived` | `archive` | `ledger` | `Widget.id` | at-least-once | none | widget id |\n")
	auditPayload(t, near, "widget.archived")
	refuteProjectionErrors(t, near)
	refuteRuleFinding(t, near, "payload_")
}

func TestRulesPayloadUnknownEvent(t *testing.T) {
	design := fanOutFixture(t, "Widget.id, Widget.channel")
	auditPayload(t, design, "widget.retired")
	requireRuleFinding(t, "rules-payload-unknown-event", design)

	near := fanOutFixture(t, "Widget.id, Widget.channel",
		"| `widget.retired` | `archive` | `ledger` | `Widget.id` | at-least-once | none | widget id |\n")
	auditPayload(t, near, "widget.retired")
	refuteRuleFinding(t, near, "payload_")
}

// One edge stated twice with two payloads is a projection error naming both
// rows; the rest still projects and the rules still run on it.
func TestEventEdgeContradiction(t *testing.T) {
	design := fanOutFixture(t, "Widget.id, Widget.price",
		"| `widget.priced` | `app` | `storelib` | `Widget.id` | at-least-once | none | widget id |\n")
	e := experimentNamed(t, "event-edge-contradiction")
	got := rulesFindings(t, design)
	var problem string
	for _, f := range got {
		if strings.Contains(f, e.ExpectSubstr) {
			problem = f
		}
	}
	if problem == "" || !strings.HasPrefix(problem, "projection error") {
		t.Fatalf("%s escaped Gy-rules: %v", e.Name, got)
	}
	if first, second := rowLine(t, design, 0), rowLine(t, design, 5); !strings.Contains(problem, first) || !strings.Contains(problem, second) {
		t.Fatalf("the problem must name both rows (%s, %s): %s", first, second, problem)
	}
	if !containsAny(got, "row 'Lead.emitPriced': payload_twin") {
		t.Fatalf("the rules must still run on the rows that projected: %v", got)
	}
}

// rowLine returns ARCHITECTURE.md:<line> of the n-th widget.priced row.
func rowLine(t *testing.T, design string, n int) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(design, "ARCHITECTURE.md"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(body), "\n")
	seen := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "| `widget.priced` |") {
			if seen == n {
				return "ARCHITECTURE.md:" + strconv.Itoa(i+1)
			}
			seen++
		}
	}
	t.Fatalf("fixture drift: no widget.priced row %d", n)
	return ""
}

// Two unnamed relationships between one entity pair with one cardinality are
// a Gy model finding naming both; named, they are two relationships.
func TestRelationshipParallelUnnamed(t *testing.T) {
	parallel := func(roles ...string) string {
		design := fanOutFixture(t, "Widget.id, Widget.channel")
		rels := "    relationships:\n"
		for _, role := range roles {
			rels += "      - entity: Lead\n        cardinality: \"n:1\"\n"
			if role != "" {
				rels += "        role: " + role + "\n"
			}
		}
		editFile(t, filepath.Join(design, "widget.modelith.yaml"), "    invariants:\n      - id: widget-owned\n", rels+"    invariants:\n      - id: widget-owned\n")
		return design
	}
	requireRuleFinding(t, "relationship-parallel-unnamed", parallel("", ""))
	refuteProjectionErrors(t, parallel("source", "target"))
}

// A malformed group on one row is a projection error, the rules still run on
// the other rows, and every finding they print says the projection was
// partial.
func TestRulesPartialProjection(t *testing.T) {
	design, _ := fixture(t)
	contractOf(t, design, "actor may publish", "actor may publish. USES{Widget.stat}")
	contractOf(t, design, "stash pending", "stash pending. USES{Widget.status, Widget.status}")
	requireRuleFinding(t, "rules-partial-projection", design)
	got := rulesFindings(t, design)
	if !containsAny(got, "row 'Widget.guardCanPublish': fact_unresolved (fact 'Widget.stat') [projection partial") {
		t.Fatalf("the finding on the projected row must be reported and marked: %v", got)
	}
}
