package gates

// Gx-trace reports the shape errors of the VALUES, payload, derived: and
// authorization-inventory declarations. What they mean (a VALUES group
// against its enum, a payload twin against its contract row, an admission
// against the System writes) is Gy-rules'; these tests pin only the grammar.

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

// shapeDesign writes a one-machine design whose Order.matrix.md is matrix,
// plus any extra files.
func shapeDesign(t *testing.T, matrix string, extra map[string]string) string {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", "formal"} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), factModel)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"),
		"machine: Order\npattern: control-flow-only\nreason: declaration shape fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), matrix)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"),
		"# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n")
	for rel, body := range extra {
		path := filepath.Join(design, filepath.FromSlash(rel))
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		mustWrite(t, path, body)
	}
	return design
}

// valuesDesign is a design whose named-unit table carries rows.
func valuesDesign(t *testing.T, rows string) string {
	t.Helper()
	return shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"+rows, nil)
}

const eventMatrixHeader = "| event | reacting unit | payload contract | maps to |\n|---|---|---|---|\n"

func TestDeclarationShapeErrors(t *testing.T) {
	cases := []struct {
		name, rows, want string
	}{
		{"values opened", "| `classify` | guard | VALUES{Placed, Paid | `order-paid-final` |\n", "Order.matrix.md:3: malformed VALUES declaration; write VALUES{a, b, c}"},
		{"values duplicate", "| `classify` | guard | VALUES{Placed, Placed} | `order-paid-final` |\n", "Order.matrix.md:3: VALUES declaration has duplicate member 'Placed'"},
		{"values empty", "| `classify` | guard | VALUES{} | `order-paid-final` |\n", "Order.matrix.md:3: VALUES declaration has no members"},
		{"values twice", "| `classify` | guard | VALUES{Placed} and VALUES kind{a} | `order-paid-final` |\n", "Order.matrix.md:3: a row carries at most one VALUES{a, b, c} declaration"},
		{"derived malformed", "| `classify` | guard | derived: pay_window | `order-paid-final` |\n", "Order.matrix.md:3: malformed derived waiver; write derived: fact_name (<reason>)"},
		{"derived reasonless", "| `classify` | guard | derived: pay_window ( ) | `order-paid-final` |\n", "Order.matrix.md:3: derived waiver for 'pay_window' names no reason"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if g := CheckTraceability(valuesDesign(t, tc.rows)); !hasErr(g, tc.want) {
				t.Fatalf("want %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestPayloadDeclarationShapes(t *testing.T) {
	cases := []struct {
		name, row, want string
	}{
		{"unclosed", "| `markPaid` | `applyPayment` | payload {Order.id | `order-paid-final` |\n", "exactly one complete payload {field, ...}"},
		{"malformed field", "| `markPaid` | `applyPayment` | payload {Order id} | `order-paid-final` |\n", "payload declaration field 'Order id' is not an identifier"},
		{"duplicate field", "| `markPaid` | `applyPayment` | payload {Order.id, Order.id} | `order-paid-final` |\n", "payload declaration has duplicate field 'Order.id'"},
		{"two events", "| `markPaid`, `markVoid` | `applyPayment` | payload {Order.id} | `order-paid-final` |\n", "payload declaration names 2 events; one closed payload set binds to exactly one event row"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if g := CheckTraceability(shapeDesign(t, eventMatrixHeader+tc.row, nil)); !hasErr(g, tc.want) {
				t.Fatalf("want %q, got %v", tc.want, g.Errs)
			}
		})
	}
	// An escaped pipe inside the payload cell keeps the row's event cell.
	g := CheckTraceability(shapeDesign(t, eventMatrixHeader+"| `markPaid` | `applyPayment` | payload {Order.id}; demote \\| retire | `order-paid-final` |\n", nil))
	if hasErr(g, "payload") || g.Counts["payload declarations parsed"] != 1 {
		t.Fatalf("escaped pipe must preserve the declaring row: %v, %+v", g.Errs, g.Counts)
	}
	// Payload prose beside another grammar is not a declaration.
	g = CheckTraceability(shapeDesign(t, eventMatrixHeader+"| `markPaid` | `applyPayment` | prose about payload then READS{Order.id} | `order-paid-final` |\n", nil))
	if hasErr(g, "payload") || g.Counts["payload declarations parsed"] != 0 {
		t.Fatalf("payload prose is not a declaration: %v", g.Errs)
	}
}

const inventoryHead = "# Authorization\n\n<!-- machinery:authorization-inventory -->\n\n"

func TestAuthorizationInventoryShapes(t *testing.T) {
	table := "| authorization subject | admission |\n|---|---|\n"
	cases := []struct {
		name, body, want string
	}{
		{"two markers", inventoryHead + "<!-- machinery:authorization-inventory -->\n\n" + table + "| Order.markPaid | `app` |\n", "authorization inventory marker appears 2 times"},
		{"marker without table", inventoryHead + "nothing here\n", "authorization inventory marker has no table with authorization subject and admission columns"},
		{"missing column", inventoryHead + "| authorization subject | owner |\n|---|---|\n| Order.markPaid | app |\n", "AUTHORIZATION.md:7: authorization inventory table must have authorization subject and admission columns"},
		{"empty subject", inventoryHead + table + "|  | `app` |\n", "AUTHORIZATION.md:7: authorization row names no subject"},
		{"bad subject", inventoryHead + table + "| mark paid | `app` |\n", "AUTHORIZATION.md:7: authorization subject 'mark paid' is not one Entity.action identifier"},
		{"empty admission", inventoryHead + table + "| Order.markPaid |  |\n", "AUTHORIZATION.md:7: authorization admission is empty"},
		{"prose admission", inventoryHead + table + "| Order.markPaid | the app decides |\n", "AUTHORIZATION.md:7: admission must name one capability as a backticked identifier"},
		{"reasonless waiver", inventoryHead + table + "| Order.markPaid | (no authorization: ) |\n", "AUTHORIZATION.md:7: authorization waiver names no reason"},
		{"duplicate subject", inventoryHead + table + "| Order.markPaid | `app` |\n| Order.markPaid | `app` |\n", "AUTHORIZATION.md:8: duplicate authorization row for 'Order.markPaid' (first at AUTHORIZATION.md:7)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n", map[string]string{"AUTHORIZATION.md": tc.body})
			if g := CheckTraceability(design); !hasErr(g, tc.want) {
				t.Fatalf("want %q, got %v", tc.want, g.Errs)
			}
		})
	}
	design := shapeDesign(t, "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n", map[string]string{
		"AUTHORIZATION.md": inventoryHead + table + "| Order.markPaid | `app` |\n| Order.markVoid | (no authorization: batch job) |\n",
	})
	g := CheckTraceability(design)
	for _, e := range g.Errs {
		if strings.Contains(e, "authorization") {
			t.Fatalf("a well-formed inventory has no shape error: %s", e)
		}
	}
	if g.Counts["authorization rows read"] != 2 {
		t.Fatalf("both rows read: %v", g.Counts)
	}
}

func deprecationWarnings(g *Gate) []string {
	var out []string
	for _, w := range g.Warns {
		if strings.Contains(w, "0.9.0 inferred") {
			out = append(out, w)
		}
	}
	return out
}

// A design that never migrated (no WRITES, USES or PRODUCES anywhere) and
// quotes a fact-shaped token in prose on a row with no group gets exactly one
// deprecation warning, naming the migration note. A design that declares any
// of the three has migrated and gets none, even with a prose backtick left;
// a row that carries a group is a declared row, not prose inference.
func TestPreCutoverInferenceWarning(t *testing.T) {
	head := "| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"
	g := CheckTraceability(valuesDesign(t, "| `markPaid` | action | records `Order.state` and `order_total` | `order-paid-final` |\n"+
		"| `settle` | action | reads `order_total` | `order-paid-final` |\n"))
	got := deprecationWarnings(g)
	if len(got) != 1 || !strings.HasPrefix(got[0], "Order.matrix.md:3: row 'markPaid': `Order.state` is quoted in prose") ||
		!strings.Contains(got[0], MigrationNote) || !strings.Contains(got[0], "removed in the release after next") {
		t.Fatalf("one warning at the first prose token, naming the note: %v", got)
	}
	if len(g.Errs) != 0 && hasErr(g, "0.9.0") {
		t.Fatalf("the deprecation is a warning, never an error: %v", g.Errs)
	}
	migrated := CheckTraceability(shapeDesign(t, head+
		"| `markPaid` | action | records `Order.state` | `order-paid-final` |\n"+
		"| `settle` | action | WRITES{Order.state} | `order-paid-final` |\n", nil))
	if got := deprecationWarnings(migrated); len(got) != 0 {
		t.Fatalf("a migrated design with one prose backtick left must not warn: %v", got)
	}
	grouped := CheckTraceability(shapeDesign(t, head+
		"| `classify` | guard | reads `order_total`. VALUES{Placed, Paid} | `order-paid-final` |\n", nil))
	if got := deprecationWarnings(grouped); len(got) != 0 {
		t.Fatalf("a row with a declaration group is not prose inference: %v", got)
	}
	plain := CheckTraceability(shapeDesign(t, head+"| `markPaid` | action | records the payment | `order-paid-final` |\n", nil))
	if got := deprecationWarnings(plain); len(got) != 0 {
		t.Fatalf("no fact-shaped prose token, no warning: %v", got)
	}
}

// The deprecation fires on no bundled example.
func TestPreCutoverInferenceSilentOnExamples(t *testing.T) {
	for _, rel := range bundledDesigns(t) {
		design := filepath.Join("..", "..", filepath.FromSlash(rel))
		if !HasMachines(design) {
			continue
		}
		if got := deprecationWarnings(CheckTraceability(design)); len(got) != 0 {
			t.Fatalf("%s: %v", rel, got)
		}
	}
}
