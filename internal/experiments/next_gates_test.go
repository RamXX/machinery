package experiments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func writeNextGateFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The H2 class-A mutation: a Modelith System write exists, every legacy gate
// input is otherwise coherent, and the authorization inventory is absent.
func TestNextGateSystemWriteWithoutAuthorizationRow(t *testing.T) {
	design := t.TempDir()
	writeNextGateFile(t, filepath.Join(design, "domain.modelith.yaml"), `kind: DomainModel
version: v1
enums:
  OrderState:
    values: [{name: Open}]
entities:
  Order:
    attributes: [{name: state, type: OrderState}]
    actions: [{name: recompute, actor: System}]
    invariants: [{id: order-held, statement: held}]
`)
	writeNextGateFile(t, filepath.Join(design, "machines", "Order.machine.json"),
		`{"id":"Order","initial":"Open","states":{"Open":{"type":"final"}}}`)
	writeNextGateFile(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | maps to |\n|---|---|---|\n| `recompute` | action | `order-held` |\n")
	writeNextGateFile(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: experiment\n")
	writeNextGateFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"| component (placement) | persistence |\n|---|---|\n| `Order` | memory |\n")

	g := gates.CheckTraceability(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "System action 'Order.recompute' has no authorization row") {
		t.Fatalf("class-A mutation escaped Gx: %v", g.Errs)
	}
}
