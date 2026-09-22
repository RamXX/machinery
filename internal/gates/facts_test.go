package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const factModel = `kind: DomainModel
version: v1
enums:
  OrderState:
    values: [{name: Placed}, {name: Paid}]
entities:
  Order:
    attributes:
      - {name: state, type: OrderState}
      - {name: order_total, type: integer}
    actions: [{name: markPaid}]
    invariants:
      - {id: order-paid-final, statement: a paid order stays paid}
`

func factFixture(t *testing.T, contract, payload string) *Gate {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", "formal"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	machine := strings.Replace(wiringMachine, `"initial": "Placed",`, `"initial": "Placed", "context": {"retry_count": 0},`, 1)
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), factModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), machine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: fact fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"+
			"| `saveOrder` | actor | "+contract+" | `order-paid-final` |\n")
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n\n" +
		wiringHeader + "| markPaid | orders | payments | " + payload + " | at-least-once | none | event_id |\n" +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

func TestFactsResolveAcrossModelContextAndEventPayload(t *testing.T) {
	g := factFixture(t, "reads `order_total`, increments `retry_count`, records `event_id`", "`event_id`")
	if hasErr(g, "unresolved fact") {
		t.Fatalf("facts declared by the model, context, or event contract must resolve: %v", g.Errs)
	}
}

func TestFactsRejectUnknownNamedUnitFact(t *testing.T) {
	g := factFixture(t, "persists `ghost_fact`", "`event_id`")
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("a named unit fact with no declaration must fail: %v", g.Errs)
	}
}

func TestFactsRejectUnknownEventPayloadFact(t *testing.T) {
	g := factFixture(t, "persists `order_total`", "`ghost_fact`")
	if !hasErr(g, "unresolved fact 'ghost_fact'") {
		t.Fatalf("an event payload fact with no declaration must fail: %v", g.Errs)
	}
}

func TestFactsDerivedWaiverRequiresReason(t *testing.T) {
	g := factFixture(t, "computes `risk_score`; derived: risk_score (weighted fraud inputs)", "`event_id`")
	if hasErr(g, "unresolved fact 'risk_score'") {
		t.Fatalf("a reasoned row-local derived waiver must close the fact: %v", g.Errs)
	}
	g = factFixture(t, "computes `risk_score`; derived: risk_score ()", "`event_id`")
	if !hasErr(g, "derived waiver for 'risk_score' names no reason") {
		t.Fatalf("an empty derived reason must fail: %v", g.Errs)
	}
}
