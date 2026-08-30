package gates

import (
	"os"
	"path/filepath"
	"testing"
)

// readsFixture builds the same Gx-ready design the wiring tests use (model,
// machine, placement, traceability), with the given event section in
// ARCHITECTURE.md and the given extra rows in the named-unit matrix, which is
// where a consumer declares what it READS.
func readsFixture(t *testing.T, eventSection, matrixRows string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), wiringModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n"+
			"| `saveOrder` | actor | (ctx) -> Order | persists the `markPaid` transition | `order-paid-final` |\n"+
			matrixRows)
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Order` | in-memory |\n\n" + eventSection +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

// readsArmed is the completeness marker, on its own line above the table.
const readsArmed = "<!-- machinery:reads-complete -->\n\n"

// readsRow's payload carries two fields; the consumer reads one of them.
const readsRow = "| markPaid | orders | payments | Order.id, Order.paidAt | at-least-once | none | Order.id |\n"

// readsDeclRow is the consuming matrix row, declaring what it reads.
const readsDeclRow = "| `applyPayment` | action | (ctx,evt) -> ctx | reacts to `markPaid` READS{Order.paidAt} | - |\n"

// An unarmed design carries no obligation whatsoever: today's behavior, byte
// for byte. A consumer row with no READS declaration is exactly as silent as
// it has always been.
func TestReadsCompleteUnarmedDesignCarriesNoObligation(t *testing.T) {
	g := readsFixture(t, wiringHeader+readsRow, "")
	if hasErr(g, "READS") {
		t.Fatalf("an unarmed design owes no reads declaration: %v", g.Errs)
	}
	for _, c := range []string{
		"event-contract consumer reads declared",
		"event-contract consumer reads missing",
		"event-contract consumer reads waived",
	} {
		if _, ok := g.Counts[c]; ok {
			t.Fatalf("an unarmed design must not count %q: %+v", c, g.Counts)
		}
	}
}

// Armed and missing: the row nobody declared reads for is an ERROR naming the
// row and the event, not a silent no-obligation row. This is the hole the
// motivating defect fell through.
func TestReadsCompleteArmedRowWithoutDeclarationErrors(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+readsRow, "")
	if !hasErr(g, "event-contract row 1 (event 'markPaid'): no consumer READS declaration") {
		t.Fatalf("an armed row with no declaration must error, naming row and event: %v", g.Errs)
	}
	if g.Counts["event-contract consumer reads missing"] != 1 {
		t.Fatalf("the missing declaration must stay visible in the counts: %+v", g.Counts)
	}
}

// Armed and declared clean: the declaration resolves against the row's own
// payload cell, and both the row and the field are counted.
func TestReadsCompleteArmedDeclaredRowPasses(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+readsRow, readsDeclRow)
	if hasErr(g, "READS") {
		t.Fatalf("a declared, carried read must pass: %v", g.Errs)
	}
	if g.Counts["event-contract consumer reads declared"] != 1 {
		t.Fatalf("the declaration must be counted: %+v", g.Counts)
	}
	if g.Counts["declared read fields carried"] != 1 {
		t.Fatalf("the resolved field must be counted: %+v", g.Counts)
	}
	if g.Counts["event-contract consumer reads missing"] != 0 {
		t.Fatalf("nothing is missing here: %+v", g.Counts)
	}
}

// The house waiver idiom, with the house rule: a reason is mandatory. A pure
// signal (the consumer reads nothing off the payload and refetches) says so
// in the consumer cell, and the rule stays satisfiable honestly.
func TestReadsCompleteWaiverDischargesTheRow(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+
		"| markPaid | orders | payments (no reads: pure signal; the consumer refetches the order by id) | Order.id | at-least-once | none | Order.id |\n", "")
	if hasErr(g, "READS") {
		t.Fatalf("a reasoned waiver discharges the row: %v", g.Errs)
	}
	if g.Counts["event-contract consumer reads waived"] != 1 {
		t.Fatalf("the waiver must stay visible in the counts: %+v", g.Counts)
	}
	g = readsFixture(t, readsArmed+wiringHeader+
		"| markPaid | orders | payments (no reads: ) | Order.id | at-least-once | none | Order.id |\n", "")
	if !hasErr(g, "waiver names no reason") {
		t.Fatalf("an empty reason is an unanswered question, not a waiver: %v", g.Errs)
	}
}

// A `(no machine: <reason>)` waiver answers a different question (nothing
// reacts as a machine event) and never discharges the reads obligation: the
// motivating defect's consumer reacted through an invoke actor, which is
// exactly where the payload insufficiency lived.
func TestReadsCompleteNoMachineWaiverDoesNotWaiveReads(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+
		"| markPaid | orders | payments (no machine: the peer service consumes it through its invoke actor) | Order.id | at-least-once | none | Order.id |\n", "")
	if !hasErr(g, "no consumer READS declaration") {
		t.Fatalf("one waiver must not answer another question: %v", g.Errs)
	}
}

// The regression, modeled on the motivating defect: the consumer declares it
// reads a field the payload does not carry. Armed, that is an ERROR on the
// declaring line, naming the row whose payload is short.
func TestReadsCompleteDeclaredFieldThePayloadDoesNotCarryErrors(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+
		"| markPaid | orders | payments | Order.id | at-least-once | none | Order.id |\n",
		"| `draftNorm` | action | (ctx,evt) -> ctx | reacts to `markPaid` READS{Order.documentVersion} | - |\n")
	if !hasErr(g, "reads Order.documentVersion") {
		t.Fatalf("a declared read the payload does not carry must error: %v", g.Errs)
	}
	if !hasErr(g, "payload-sufficiency") {
		t.Fatalf("the finding must name the defect class it belongs to: %v", g.Errs)
	}
}

// A near-miss field name must not resolve: the check is whole-token, the same
// rule the opt-in warn tier has always applied.
func TestReadsCompleteFieldResolutionIsWholeToken(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+
		"| markPaid | orders | payments | Order.paidAtUtc | at-least-once | none | Order.id |\n",
		readsDeclRow)
	if !hasErr(g, "reads Order.paidAt") {
		t.Fatalf("a prefix of a payload field is not the field: %v", g.Errs)
	}
}

// Every event a prose cell names owes its own declaration, the same reading
// the wiring check gives the cell.
func TestReadsCompleteReadsEveryBacktickedEventInAProseCell(t *testing.T) {
	g := readsFixture(t, readsArmed+wiringHeader+
		"| `markPaid` / `markDeclined` events | orders | payments | Order.id | at-least-once | none | Order.id |\n",
		"| `applyPayment` | action | (ctx,evt) -> ctx | reacts to `markPaid` READS{Order.id} | - |\n")
	if !hasErr(g, "no consumer READS declaration for event 'markDeclined'") {
		t.Fatalf("each event in the cell owes a declaration: %v", g.Errs)
	}
	if hasErr(g, "no consumer READS declaration for event 'markPaid'") {
		t.Fatalf("the declared event in the same cell must pass: %v", g.Errs)
	}
}

// Arming a table whose rows name no event is an authoring error, not a silent
// pass: the declarations are keyed by event, so a table with no event column
// can never satisfy the tier.
func TestReadsCompleteArmedTableWithoutAnEventColumnErrors(t *testing.T) {
	g := readsFixture(t, readsArmed+"## Events\n\n| producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|\n"+
		"| orders | payments | Order.id | at-least-once | none | Order.id |\n", "")
	if !hasErr(g, "no event column") {
		t.Fatalf("an armed table with no event column must say so: %v", g.Errs)
	}
}

// Arming a design that carries no event-contract table at all is a claim with
// no subject; it fails loudly rather than passing empty.
func TestReadsCompleteArmedWithNoEventTableErrors(t *testing.T) {
	g := readsFixture(t, readsArmed+"## Events\n\nNone yet.\n", "")
	if !hasErr(g, "no event-contract table") {
		t.Fatalf("an armed design with no table must fail loudly: %v", g.Errs)
	}
}
