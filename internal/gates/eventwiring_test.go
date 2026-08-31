package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const wiringModel = `kind: DomainModel
version: v1
enums:
  OrderState:
    values:
      - name: Placed
      - name: Paid
entities:
  Order:
    attributes:
      - name: state
        type: OrderState
    actions:
      - name: markPaid
    invariants:
      - id: order-paid-final
        statement: a paid order stays paid
`

// wiringMachine handles markPaid, ignores a redelivered markPaid on the
// terminal state, and fires emitRequest from an action position.
const wiringMachine = `{
  "id": "Order",
  "initial": "Placed",
  "states": {
    "Placed": {
      "entry": "emitRequest",
      "on": {"markPaid": {"target": "saving"}}
    },
    "saving": {
      "invoke": {"src": "saveOrder", "onDone": {"target": "Paid"}, "onError": {"target": "Placed"}},
      "after": {"5000": {"target": "Placed"}}
    },
    "Paid": {
      "type": "final",
      "_ignores": {"markDeclined": "already Paid; at-least-once redelivery"}
    }
  }
}`

// wiringFixture builds a Gx-ready design (model, machine, placement,
// traceability) whose ARCHITECTURE.md carries the given event section.
func wiringFixture(t *testing.T, eventSection string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), wiringModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	// the named-unit matrix every machine carries; its cells are the emission
	// evidence a produced event is traced to, exactly as on the pack side
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n"+
			"| `saveOrder` | actor | (ctx) -> Order | persists the `markPaid` transition | `order-paid-final` |\n")
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Order` | in-memory |\n\n" + eventSection +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

const wiringHeader = "## Events\n\n| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n"

// The pass case: the event is handled by a machine (the reaction) and named in
// a matrix cell or action (the emission).
func TestEventWiringReconciledRowPasses(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markPaid | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "event-contract row") {
		t.Fatalf("a reconciled row must pass: %v", g.Errs)
	}
	if g.Counts["event-contract reactions traced"] != 1 || g.Counts["event-contract emissions traced"] != 1 {
		t.Fatalf("both directions must be counted: %+v", g.Counts)
	}
}

func TestEventWiringUnhandledEventErrors(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markShipped | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "event 'markShipped' is handled or ignored by no machine") {
		t.Fatalf("a consumed event no machine handles must error: %v", g.Errs)
	}
}

// An `_ignores` entry is handling: the redelivery idiom the xstate reference
// documents, and exactly what the pack side accepts.
func TestEventWiringIgnoresCountAsHandling(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markDeclined | payments | orders | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "handled or ignored by no machine") {
		t.Fatalf("an _ignores entry is explicit handling: %v", g.Errs)
	}
}

func TestEventWiringUnemittedEventErrors(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markPaid | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "appears in no machine action") {
		t.Fatalf("markPaid is an on: event and a matrix-free design still fires it: %v", g.Errs)
	}
	// an event nothing emits and nothing handles reports BOTH directions
	g = wiringFixture(t, wiringHeader+
		"| refunded | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "appears in no machine action") {
		t.Fatalf("a produced event nothing fires must error: %v", g.Errs)
	}
	if !hasErr(g, "handled or ignored by no machine") {
		t.Fatalf("the same row also owes the reaction: %v", g.Errs)
	}
}

// An action position is emission evidence, entry actions included.
func TestEventWiringActionPositionIsEmissionEvidence(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| emitRequest | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "appears in no machine action") {
		t.Fatalf("an entry action is emission evidence: %v", g.Errs)
	}
}

// A matrix cell is emission evidence too, so an emitting action named
// differently is reconciled by naming the event whole-token in its row.
func TestEventWiringMatrixCellIsEmissionEvidence(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), wiringModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | maps to |\n|---|---|---|\n| `saveOrder` | actor | emits `settled` |\n")
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Order` | in-memory |\n\n" + wiringHeader +
		"| settled | orders | payments | Order.id | at-least-once | none | Order.id | \n" +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckTraceability(design)
	if hasErr(g, "appears in no machine action") {
		t.Fatalf("a matrix cell is emission evidence: %v", g.Errs)
	}
}

// The house waiver token, with the house rule: a reason is mandatory, and the
// waived cell is the one that owes the obligation.
func TestEventWiringWaiverNeedsAReason(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markShipped | orders | payments (no machine: shipping is a downstream service with no machine here) | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "handled or ignored by no machine") {
		t.Fatalf("a reasoned consumer waiver discharges the reaction: %v", g.Errs)
	}
	if g.Counts["event-contract reactions waived"] != 1 {
		t.Fatalf("the waiver must stay visible in the counts: %+v", g.Counts)
	}
	g = wiringFixture(t, wiringHeader+
		"| markShipped | orders | payments (no machine: ) | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "waiver names no reason") {
		t.Fatalf("an empty reason is an unanswered question, not a waiver: %v", g.Errs)
	}
}

func TestEventWiringProducerWaiverDischargesEmissionOnly(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| markPaid | orders (no machine: emitted by the peer service's outbox) | payments | Order.id | at-least-once | none | Order.id |\n")
	if g.Counts["event-contract emissions waived"] != 1 {
		t.Fatalf("the producer waiver must be counted: %+v", g.Counts)
	}
	if g.Counts["event-contract reactions traced"] != 1 {
		t.Fatalf("the reaction obligation still binds: %+v", g.Counts)
	}
}

// A prose event cell names its events backticked, the same idiom the
// mitigation and placement rows use; each named event is reconciled.
func TestEventWiringReadsEveryBacktickedEventInAProseCell(t *testing.T) {
	g := wiringFixture(t, wiringHeader+
		"| `markPaid` / `markShipped` events | orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if !hasErr(g, "event 'markShipped' is handled or ignored by no machine") {
		t.Fatalf("every backticked event in the cell is reconciled: %v", g.Errs)
	}
	if hasErr(g, "event 'markPaid' is handled") {
		t.Fatalf("the handled event in the same cell must pass: %v", g.Errs)
	}
}

// A table with no event column names no event to reconcile: a stated
// non-check, visible in the counts rather than a silent pass.
func TestEventWiringWithoutAnEventColumnIsAStatedNonCheck(t *testing.T) {
	g := wiringFixture(t, "## Events\n\n| producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|\n"+
		"| orders | payments | Order.id | at-least-once | none | Order.id |\n")
	if hasErr(g, "event-contract row") {
		t.Fatalf("a table with no event column reconciles nothing: %v", g.Errs)
	}
	if g.Counts["event-contract rows without an event column"] != 1 {
		t.Fatalf("the unreconciled row must stay visible: %+v", g.Counts)
	}
}

// A design that carries a pack is the pack gate's: G5 reconciles its boundary
// events from the generated events.md, where direction is explicit. One defect
// never earns two findings from two gates.
func TestEventWiringSkippedWhenAPackGovernsTheDesign(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), wiringModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "pack", "pack.yaml"), "pack_revision: 1\n")
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Order` | in-memory |\n\n" + wiringHeader +
		"| markShipped | orders | payments | Order.id | at-least-once | none | Order.id |\n" +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckTraceability(design)
	if hasErr(g, "event-contract row") {
		t.Fatalf("a packed design is G5's; Gx must not double-report: %v", g.Errs)
	}
}

// The reverse sweep, armed by _external_events: a machine event declared
// externally sourced owes an event-contract row, even when no table exists.
func TestEventWiringExternalDeclarationReverseSweep(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"),
		`{"id":"order","_external_events":["markPaid"],"initial":"A","states":{
			"A":{"on":{"markPaid":{"target":"B"}}},"B":{"type":"final"}}}`)
	g := NewGate("test")
	checkEventWiring(g, design, "# arch, no event table\n")
	if !hasErr(g, "declared externally sourced (_external_events) but no event-contract row names it") {
		t.Fatalf("a declared external event without a row must error: %v", g.Errs)
	}
	g = NewGate("test")
	checkEventWiring(g, design,
		"| event | producer | consumer | payload | delivery | ordering | dedupe |\n"+
			"|---|---|---|---|---|---|---|\n"+
			"| `markPaid` | payments (no machine: peer subsystem) | orders | Payment.id | at-least-once | none | Payment.id |\n")
	for _, e := range g.Errs {
		if strings.Contains(e, "externally sourced") {
			t.Fatalf("a contract row must satisfy the sweep: %v", g.Errs)
		}
	}
	if g.Counts["externally-sourced events with contract rows"] != 1 {
		t.Fatalf("the satisfied sweep must be counted: %+v", g.Counts)
	}
}
