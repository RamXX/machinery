package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const authzModel = `kind: DomainModel
version: v1
enums:
  OrderState:
    values: [{name: Placed}, {name: Paid}]
entities:
  Order:
    attributes: [{name: state, type: OrderState}]
    actions:
      - {name: markPaid, actor: System}
    invariants:
      - {id: order-paid-final, statement: a paid order stays paid}
`

func authzFixture(t *testing.T, inventory, extraMatrix string) *Gate {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", "formal"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), authzModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: authorization fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | maps to |\n|---|---|---|\n| `saveOrder` | actor | `order-paid-final` |\n"+extraMatrix)
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n\n" + inventory
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

const authzHeader = `<!-- machinery:authorization-inventory -->

| authorization subject | admission |
|---|---|
`

func TestAuthorizationInventoryMissingSystemActionErrors(t *testing.T) {
	g := authzFixture(t, "", "")
	if !hasErr(g, "System action 'Order.markPaid' has no authorization row") {
		t.Fatalf("a System write without an inventory row must fail: %v", g.Errs)
	}
}

func TestAuthorizationInventoryAdmitsSystemAction(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | internal machine capability `orders.write` |\n", "")
	if hasErr(g, "authorization") {
		t.Fatalf("an admitted System action must pass: %v", g.Errs)
	}
	if g.Counts["authorization obligations admitted"] != 1 {
		t.Fatalf("the admission must be counted: %+v", g.Counts)
	}
}

func TestAuthorizationInventoryWaiverRequiresReason(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | (no authorization: immutable internal replay) |\n", "")
	if hasErr(g, "authorization") {
		t.Fatalf("a reasoned waiver must pass: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+"| `Order.markPaid` | (no authorization: ) |\n", "")
	if !hasErr(g, "waiver names no reason") {
		t.Fatalf("an empty waiver must fail: %v", g.Errs)
	}
}

func TestAuthorizationInventoryProducerArmCreatesObligation(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | internal capability |\n", matrix)
	if !hasErr(g, "matrix producer 'Order.reindex' has no authorization row") {
		t.Fatalf("a named cascade producer must be admitted independently: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+
		"| `Order.markPaid` | internal capability |\n| `Order.reindex` | producer capability |\n", matrix)
	if hasErr(g, "authorization") {
		t.Fatalf("both obligations are admitted: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsDuplicateAndOrphanRows(t *testing.T) {
	g := authzFixture(t, authzHeader+
		"| `Order.markPaid` | first |\n| `Order.markPaid` | second |\n| `Order.ghost` | stale |\n", "")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"duplicate authorization row", "names no System action or matrix producer"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("inventory closure must report %q: %v", want, g.Errs)
		}
	}
}
