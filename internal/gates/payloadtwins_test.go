package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func payloadTwinFixture(t *testing.T, archPayload, matrixRows string) *Gate {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), wiringModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	if err := os.MkdirAll(filepath.Join(design, "formal"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: fixture checks payload declarations only\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| event | reacting unit | payload contract | maps to |\n|---|---|---|---|\n"+matrixRows)
	arch := "# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n" +
		"| `Order` | in-memory |\n\n" + wiringHeader +
		"| markPaid | orders | payments | " + archPayload + " | at-least-once | none | Order.id |\n" +
		"\n## Traceability\n\n| invariant | where |\n|---|---|\n| order-paid-final | `Order` guard |\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	return CheckTraceability(design)
}

func TestPayloadTwinsExactSetPassesRegardlessOfOrder(t *testing.T) {
	g := payloadTwinFixture(t, "`Order.id`, `Order.paidAt`",
		"| `markPaid` | `applyPayment` | payload is exactly {Order.paidAt, Order.id} | `order-paid-final` |\n")
	if hasErr(g, "payload declaration") || hasErr(g, "payload twin") {
		t.Fatalf("equal payload sets must pass: %v", g.Errs)
	}
	if g.Counts["event payload twins reconciled"] != 1 {
		t.Fatalf("the compared twin must be counted: %+v", g.Counts)
	}
}

func TestPayloadTwinsMismatchReportsBothSpellings(t *testing.T) {
	g := payloadTwinFixture(t, "`Order.id`, `Order.paidAt`",
		"| `markPaid` | `applyPayment` | payload {Order.id, Order.currency} | `order-paid-final` |\n")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"payload twin", "Order.currency", "Order.paidAt", "Order.id"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("mismatch must report %q and both spellings: %v", want, g.Errs)
		}
	}
}

func TestPayloadTwinsFailClosedOnMalformedDeclaration(t *testing.T) {
	cases := []struct {
		name, declaration, want string
	}{
		{"empty", "payload {}", "no fields"},
		{"empty member", "payload {Order.id, }", "empty member"},
		{"duplicate", "payload {Order.id, Order.id}", "duplicate field"},
		{"two groups", "payload {Order.id} payload {Order.paidAt}", "exactly one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := payloadTwinFixture(t, "`Order.id`",
				"| `markPaid` | `applyPayment` | "+tc.declaration+" | `order-paid-final` |\n")
			if !hasErr(g, tc.want) {
				t.Fatalf("malformed declaration must fail with %q: %v", tc.want, g.Errs)
			}
		})
	}
}

func TestPayloadTwinsOneDeclarationCannotCoverSeveralEvents(t *testing.T) {
	g := payloadTwinFixture(t, "`Order.id`",
		"| `markPaid` / `markDeclined` events | `applyPayment` | payload {Order.id} | `order-paid-final` |\n")
	if !hasErr(g, "names 2 events") {
		t.Fatalf("one payload declaration must bind to exactly one event: %v", g.Errs)
	}
}
