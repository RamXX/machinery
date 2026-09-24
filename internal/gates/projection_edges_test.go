package gates

// Event edges and per-row degradation (MAC-j39j). The event-contract table is
// one row per producer-consumer edge, so a fan-out event spans several rows:
// the event is defined once, by name, and each row is an edge with its own
// payload. A row the reader cannot project (a malformed declaration group,
// two payloads for one edge) is reported and omitted, and the rest of the
// design still projects for Gy-rules; `machinery project` stays strict.

import (
	"strings"
	"testing"
)

const fanOutHeader = "# Architecture\n\n## Event-contract table\n\n" +
	"| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n"

// Five rows of one event: two producers by two consumers, per-edge payloads
// that differ, and the pricing->ledger row restated verbatim. A sixth row
// names two producers in one cell, which is the cross product of edges.
const fanOutArchitecture = fanOutHeader +
	"| `quote.issued` | `pricing` | `billing` | `Quote.id`, `Quote.total` | at-least-once | none | quote id |\n" +
	"| `quote.issued` | `pricing` | `ledger` | `Quote.id` | at-least-once | none | quote id |\n" +
	"| `quote.issued` | `intake` | `billing` | `Quote.id`, `Quote.channel` | at-least-once | none | quote id |\n" +
	"| `quote.issued` | `intake` | `ledger` | `Quote.id` | at-least-once | none | quote id |\n" +
	"| `quote.issued` | `pricing` | `ledger` | `Quote.id` | at-least-once | none | quote id |\n" +
	"| `quote.voided` | `pricing` + `intake` | `ledger` | `Quote.id` | at-least-once | none | quote id |\n"

func TestFactsFanOutEventIsDefinedOnceWithOneEdgePerPair(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"ARCHITECTURE.md": fanOutArchitecture})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatalf("a fan-out event must project: %v", err)
	}
	events := facts.Rows("event")
	if got := strings.Join(factRows(facts, "event"), " "); got != "quote.issued quote.voided" {
		t.Fatalf("each event is defined once: %s", got)
	}
	if events[0].Source.Line != 7 || events[0].StableID != "event:quote.issued" {
		t.Fatalf("an event's source is the first row naming it: %+v", events[0])
	}
	wantEdges := "quote.issued|intake->billing|quote.issued|intake|billing " +
		"quote.issued|intake->ledger|quote.issued|intake|ledger " +
		"quote.issued|pricing->billing|quote.issued|pricing|billing " +
		"quote.issued|pricing->ledger|quote.issued|pricing|ledger " +
		"quote.voided|intake->ledger|quote.voided|intake|ledger " +
		"quote.voided|pricing->ledger|quote.voided|pricing|ledger"
	if got := strings.Join(factRows(facts, "event_edge"), " "); got != wantEdges {
		t.Fatalf("one edge per producer-consumer pair, the restated row collapsed:\n got %s\nwant %s", got, wantEdges)
	}
	for _, row := range facts.Rows("event_edge") {
		if row.StableID != "edge:"+row.Values[0] {
			t.Fatalf("an edge's stable id is edge:<edge>, content-derived: %+v", row)
		}
		if row.Values[0] == "quote.issued|pricing->ledger" && row.Source.Line != 8 {
			t.Fatalf("a restated edge keeps its first row as source: %+v", row)
		}
	}
	wantPayload := "quote.issued|intake->billing|Quote.channel quote.issued|intake->billing|Quote.id " +
		"quote.issued|intake->ledger|Quote.id " +
		"quote.issued|pricing->billing|Quote.id quote.issued|pricing->billing|Quote.total " +
		"quote.issued|pricing->ledger|Quote.id " +
		"quote.voided|intake->ledger|Quote.id quote.voided|pricing->ledger|Quote.id"
	if got := strings.Join(factRows(facts, "event_edge_payload_field"), " "); got != wantPayload {
		t.Fatalf("each edge carries its own payload:\n got %s\nwant %s", got, wantPayload)
	}
	if got := strings.Join(factRows(facts, "event_payload_field"), " "); got != "quote.issued|Quote.channel quote.issued|Quote.id quote.issued|Quote.total quote.voided|Quote.id" {
		t.Fatalf("event_payload_field is the union over the event's edges: %s", got)
	}
	if got := strings.Join(factRows(facts, "event_producer"), " "); got != "quote.issued|intake quote.issued|pricing quote.voided|intake quote.voided|pricing" {
		t.Fatalf("producers: %s", got)
	}
	if got := strings.Join(factRows(facts, "event_consumer"), " "); got != "quote.issued|billing quote.issued|ledger quote.voided|ledger" {
		t.Fatalf("consumers: %s", got)
	}
}

// One edge stated twice with two payloads is a real contradiction: the
// strict reader fails naming both rows, and the partial reader keeps the
// first statement, omits the second row, and names both.
func TestFactsContradictoryEdgeIsARowProblemNamingBothRows(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"ARCHITECTURE.md": fanOutHeader +
		"| `quote.issued` | `pricing` | `billing` | `Quote.id`, `Quote.total` | at-least-once | none | quote id |\n" +
		"| `quote.issued` | `pricing` | `billing` | `Quote.id` | at-least-once | none | quote id |\n" +
		"| `quote.issued` | `pricing` | `ledger` | `Quote.id` | at-least-once | none | quote id |\n"})
	if _, err := LoadDesignFacts(design); err == nil || !strings.Contains(err.Error(), "ARCHITECTURE.md:7") || !strings.Contains(err.Error(), "ARCHITECTURE.md:8") {
		t.Fatalf("a contradictory edge must fail the strict projection naming both rows, got %v", err)
	}
	rep, err := projectDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.problems) != 1 || !strings.Contains(rep.problems[0], "quote.issued|pricing->billing") ||
		!strings.Contains(rep.problems[0], "ARCHITECTURE.md:7") || !strings.Contains(rep.problems[0], "ARCHITECTURE.md:8") {
		t.Fatalf("one problem naming the edge and both rows, got %v", rep.problems)
	}
	if got := strings.Join(factRows(rep.facts, "event_edge_payload_field"), " "); got != "quote.issued|pricing->billing|Quote.id quote.issued|pricing->billing|Quote.total quote.issued|pricing->ledger|Quote.id" {
		t.Fatalf("the first statement stays, the contradicting row is omitted, the rest projects: %s", got)
	}
}

// A malformed declaration group on one matrix row omits that row's unit and
// every fact it states; the other rows of the same matrix still project.
func TestFactsMalformedMatrixRowOmitsOnlyThatRow(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | pre / post |\n|---|---|---|\n" +
			"| `persist` | actor | WRITES{Order.status} |\n" +
			"| `canPay` | guard | USES{Order.total, Order.total} |\n",
	})
	if _, err := LoadDesignFacts(design); err == nil || !strings.Contains(err.Error(), "machines/Order.matrix.md:4") {
		t.Fatalf("a malformed row must fail the strict projection naming it, got %v", err)
	}
	rep, err := projectDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.problems) != 1 || !strings.Contains(rep.problems[0], "machines/Order.matrix.md:4") {
		t.Fatalf("one problem naming the row, got %v", rep.problems)
	}
	if got := strings.Join(factRows(rep.facts, "unit"), " "); got != "Order.persist|Order|persist|actor" {
		t.Fatalf("the malformed row's unit is omitted, the other projects: %s", got)
	}
	if got := strings.Join(factRows(rep.facts, "unit_uses"), " "); got != "" {
		t.Fatalf("no fact of the malformed row survives: %s", got)
	}
	if got := strings.Join(factRows(rep.facts, "unit_writes"), " "); got != "Order.persist|Order.status" {
		t.Fatalf("the well-formed row's facts project: %s", got)
	}
}

const parallelRelationshipsModel = `kind: DomainModel
version: v1
entities:
  Norm:
    attributes:
      - {name: code, type: string}
  Link:
    attributes:
      - {name: kind, type: string}
    relationships:
      - {entity: Norm, cardinality: "n:1"}
      - {entity: Norm, cardinality: "n:1"}
      - {entity: Norm, cardinality: "n:1", role: target}
`

// Two relationships between one entity pair with one cardinality and no role
// or name share an id. That is a model finding naming both, never a failed
// projection: the relation is a set, so they project as one tuple, and a
// named sibling keeps its own id.
func TestFactsParallelUnnamedRelationshipsAreAModelFinding(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"domain.modelith.yaml": parallelRelationshipsModel})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatalf("parallel unnamed relationships must not fail the projection: %v", err)
	}
	if got := strings.Join(factRows(facts, "relationship"), " "); got != "Link->Norm:n:1|Link|Norm|n:1 Link->Norm:n:1:target|Link|Norm|n:1" {
		t.Fatalf("relationships: %s", got)
	}
	rep, err := projectDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.problems) != 0 || len(rep.modelFindings) != 1 {
		t.Fatalf("one model finding and no projection problem, got problems %v findings %v", rep.problems, rep.modelFindings)
	}
	f := rep.modelFindings[0]
	for _, want := range []string{"domain.modelith.yaml:11", "domain.modelith.yaml:12", "Link->Norm:n:1", "role"} {
		if !strings.Contains(f, want) {
			t.Fatalf("the finding must name %s: %s", want, f)
		}
	}
}

// The action-ownership table projects each owned action with its component;
// an (unowned: <reason>) row projects nothing.
func TestFactsActionOwnershipProjectsActionOwner(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"ARCHITECTURE.md": "# Architecture\n\n## Action ownership\n\n" +
		"| action | owning component |\n|---|---|\n" +
		"| `Order.pay` | `billing` |\n" +
		"| `Order.refund`, `Order.view` | `billing` (the ledger reads it) |\n" +
		"| `Order.archive` | (unowned: a batch job with no single owner) |\n"})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factRows(facts, "action_owner"), " "); got != "Order.pay|billing Order.refund|billing Order.view|billing" {
		t.Fatalf("action_owner: %s", got)
	}
}

// A malformed contract group omits every declaration of its row, a
// well-formed sibling group included; the other rows still project.
func TestFactsMalformedContractRowOmitsOnlyThatRow(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{"ARCHITECTURE.md": "# Architecture\n\n" +
		"| type | replaces |\n|---|---|\n" +
		"| TypeX | SUPERSEDES{type:OldX} |\n" +
		"| TypeY | SUPERSEDES{OldY} RESERVED{type:TypeZ} |\n"})
	rep, err := projectDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.problems) != 1 || !strings.Contains(rep.problems[0], "ARCHITECTURE.md:6") {
		t.Fatalf("one problem naming the row, got %v", rep.problems)
	}
	if got := strings.Join(factRows(rep.facts, "supersedes"), " ") + "/" + strings.Join(factRows(rep.facts, "reserved"), " "); got != "TypeX|OldX/" {
		t.Fatalf("only the well-formed row projects: %s", got)
	}
}

// Gy-rules degrades per row: a projection error is an ERROR naming the row,
// the rules still run on everything else, the gate stays red, and every
// finding printed beside a projection error says the projection was partial.
func TestRulesRunOverAPartialProjectionAndMarkEveryFinding(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | pre / post |\n|---|---|---|\n" +
			"| `persist` | actor | USES{Order.stat} CARRIES{column:Order.status} |\n" +
			"| `canPay` | guard | USES{Order.total, Order.total} |\n",
	})
	g := CheckRules(design, false)
	var projection, findings []string
	for _, e := range g.Errs {
		if strings.HasPrefix(e, "projection error") {
			projection = append(projection, e)
		} else {
			findings = append(findings, e)
		}
	}
	if len(projection) != 1 || !strings.Contains(projection[0], "machines/Order.matrix.md:4") {
		t.Fatalf("one projection ERROR naming the row, got %v", g.Errs)
	}
	if !containsSub(findings, "row 'Order.persist': fact_unresolved (fact 'Order.stat')") {
		t.Fatalf("the rules must run on the rows that projected: %v", g.Errs)
	}
	for _, f := range findings {
		if !strings.Contains(f, partialProjectionNote(1)) {
			t.Fatalf("a finding printed beside a projection error must say the projection was partial: %s", f)
		}
	}
}

// Parallel unnamed relationships are one Gy ERROR naming both and asking for
// a name; no fact is omitted, so the other findings carry no partial note.
func TestRulesReportParallelUnnamedRelationships(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"domain.modelith.yaml":       parallelRelationshipsModel,
		"machines/Link.matrix.md":    "| name | kind | pre / post |\n|---|---|---|\n| `checkLink` | guard | USES{Link.knd} |\n",
		"machines/Link.machine.json": factsMachine,
	})
	g := CheckRules(design, false)
	if !containsSub(g.Errs, "domain.modelith.yaml:12: relationship 'Link->Norm:n:1'") || !containsSub(g.Errs, "domain.modelith.yaml:11") {
		t.Fatalf("the parallel relationships must be one ERROR naming both: %v", g.Errs)
	}
	if !containsSub(g.Errs, "fact_unresolved (fact 'Link.knd')") || containsSub(g.Errs, "projection partial") {
		t.Fatalf("the rules run on complete facts, with no partial note: %v", g.Errs)
	}
}

func containsSub(lines []string, sub string) bool {
	for _, l := range lines {
		if strings.Contains(l, sub) {
			return true
		}
	}
	return false
}
