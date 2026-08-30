package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// The boundary-event direction vocabulary. G5 reads a pack's events.md and
// holds each row to the rule its direction names; a row whose direction is
// neither consumes nor produces used to fall out of the switch in silence,
// checked by nobody and counted by nobody.

// packDirectionMachine handles the consumed event and fires the produced one,
// so the two well-formed rows pass and the third row is alone in the output.
const packDirectionMachine = `{
  "id": "Order",
  "initial": "New",
  "states": {
    "New": {"on": {"markPaid": {"target": "Paid", "actions": ["request"]}}},
    "Paid": {}
  }
}`

// packDirectionEvents carries one legal row per direction plus the offender.
const packDirectionEvents = `# Boundary event contracts: orders

| event | direction | peer | payload | delivery | ordering | dedupe |
|---|---|---|---|---|---|---|
| markPaid | consumes | payments | Payment.orderId | at-least-once | none | Payment.id |
| request | produces | payments | Payment.orderId | at-least-once | none | Payment.orderId |
| refunded | emits | payments | Payment.orderId | at-least-once | none | Payment.id |

Boundary events: 3
`

func writePackDirectionDesign(t *testing.T) string {
	t.Helper()
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "machines", "Order.machine.json"), packDirectionMachine)
	return design
}

// A direction outside the vocabulary is an ERROR naming the value and the
// vocabulary, and the two legal rows still pass on their own rules.
func TestBoundaryEventUnknownDirection(t *testing.T) {
	g := NewGate("G5")
	checkBoundaryEvents(writePackDirectionDesign(t), packDirectionEvents, g)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "boundary event 'refunded' has direction 'emits'") {
		t.Fatalf("the offending row must be named: %v", g.Errs)
	}
	if !strings.Contains(joined, "the direction vocabulary is exactly consumes or produces") {
		t.Errorf("the finding must name the accepted vocabulary: %v", g.Errs)
	}
	if len(g.Errs) != 1 {
		t.Errorf("only the offending row may be reported: %v", g.Errs)
	}
	if g.Counts["consumed events handled"] != 1 || g.Counts["produced events emitted"] != 1 {
		t.Errorf("the legal rows must still be checked: %+v", g.Counts)
	}
}

// An empty direction cell is the same failure with a blanker face; it must
// not read as a row that was checked.
func TestBoundaryEventEmptyDirection(t *testing.T) {
	events := strings.Replace(packDirectionEvents, "| refunded | emits |", "| refunded |  |", 1)
	g := NewGate("G5")
	checkBoundaryEvents(writePackDirectionDesign(t), events, g)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "boundary event 'refunded' has direction ''") {
		t.Fatalf("an empty direction must be a finding: %v", g.Errs)
	}
}

// The vocabulary is exact, not case-folded or pluralized: the pack generator
// emits one spelling, so anything else is an authoring mistake and saying so
// is more useful than quietly accepting a variant.
func TestBoundaryEventDirectionVocabularyIsExact(t *testing.T) {
	for _, dir := range []string{"Consumes", "produce", "in", "consumes/produces"} {
		t.Run(dir, func(t *testing.T) {
			events := strings.Replace(packDirectionEvents, "| refunded | emits |", "| refunded | "+dir+" |", 1)
			g := NewGate("G5")
			checkBoundaryEvents(writePackDirectionDesign(t), events, g)
			if !strings.Contains(strings.Join(g.Errs, "\n"), "has direction "+"'"+dir+"'") {
				t.Fatalf("%q must not pass as a direction: %v", dir, g.Errs)
			}
		})
	}
}

// The clean case: nothing but the two legal directions leaves no finding.
func TestBoundaryEventLegalDirectionsOnly(t *testing.T) {
	events := strings.Replace(packDirectionEvents,
		"| refunded | emits | payments | Payment.orderId | at-least-once | none | Payment.id |\n", "", 1)
	g := NewGate("G5")
	checkBoundaryEvents(writePackDirectionDesign(t), events, g)
	if len(g.Errs) != 0 {
		t.Fatalf("a well-formed event table is not a finding: %v", g.Errs)
	}
}
