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
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", "")
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
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if !hasErr(g, "matrix producer 'Order.reindex' has no authorization row") {
		t.Fatalf("a named cascade producer must be admitted independently: %v", g.Errs)
	}
	g = authzFixture(t, authzHeader+
		"| `Order.markPaid` | `orders.write` |\n| `Order.reindex` | `orders.reindex` |\n", matrix)
	if hasErr(g, "authorization") {
		t.Fatalf("both obligations are admitted: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsDuplicateAndOrphanRows(t *testing.T) {
	g := authzFixture(t, authzHeader+
		"| `Order.markPaid` | `orders.write` |\n| `Order.markPaid` | `orders.write` |\n| `Order.ghost` | `orders.ghost` |\n", "")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"duplicate authorization row", "names no System action or matrix producer"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("inventory closure must report %q: %v", want, g.Errs)
		}
	}
}

func TestAuthorizationInventoryIgnoresCombinedProducerConsumerProse(t *testing.T) {
	matrix := "\n| cascade | producer / consumer | outcome |\n|---|---|---|\n| persist | `Order.reindex` and `Order.repair` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if hasErr(g, "matrix producer") {
		t.Fatalf("combined prose column must not arm producer obligations: %v", g.Errs)
	}
}

func TestAuthorizationInventoryUsesOneSubjectPerProducerCell(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` (via `Order.repair`) | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n| `Order.reindex` | `orders.reindex` |\n", matrix)
	if hasErr(g, "matrix producer 'Order.repair'") || hasErr(g, "matrix producer 'Order.reindex'") {
		t.Fatalf("one producer cell names one cleaned subject: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsTwoProducerSubjectsInOneCell(t *testing.T) {
	matrix := "\n| cascade | producer | outcome |\n|---|---|---|\n| persist | `Order.reindex` and `Order.repair` | emitted |\n"
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | `orders.write` |\n", matrix)
	if !hasErr(g, "producer cell must name one subject") {
		t.Fatalf("two subjects must not turn into a phantom obligation: %v", g.Errs)
	}
}

func TestAuthorizationInventoryMalformedMarkersAndAdmissions(t *testing.T) {
	cases := []struct{ name, inventory, want string }{
		{"no marker", "", "no <!-- machinery:authorization-inventory --> marker"},
		{"two markers", authzHeader + "| `Order.markPaid` | cap |\n<!-- machinery:authorization-inventory -->\n", "marker appears 2 times"},
		{"no table", "<!-- machinery:authorization-inventory -->\n", "marker has no table"},
		{"empty admission", authzHeader + "| `Order.markPaid` | |\n", "admission is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := authzFixture(t, tc.inventory, "")
			if !hasErr(g, tc.want) {
				t.Fatalf("want %q: %v", tc.want, g.Errs)
			}
		})
	}
}

func TestAuthorizationInventoryFindingNamesSourcePath(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.ghost` | stale |\n", "")
	if !hasErr(g, "ARCHITECTURE.md:") {
		t.Fatalf("row finding must name its source artifact: %v", g.Errs)
	}
}

func TestAuthorizationInventoryRejectsProseAdmission(t *testing.T) {
	g := authzFixture(t, authzHeader+"| `Order.markPaid` | TODO |\n", "")
	if !hasErr(g, "admission must name one capability") {
		t.Fatalf("free text cannot prove an admitting capability: %v", g.Errs)
	}
}
