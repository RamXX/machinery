package gates

// Before Gl-ledger warns on a backticked fact-shaped token, it resolves the
// token against what the design declares. A model attribute keeps the
// "declare it in USES{} or WRITES{}" warning; an action, a named unit, an enum
// value, a context key, an event, an invariant id or a file name is not a
// stored fact and never warns; a token naming nothing gets a softer wording.
// Every class is pinned by a positive and a near-neighbour that must warn.

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const resolveModel = `kind: DomainModel
version: v1
enums:
  PaymentOutcome:
    values: [{name: CardDeclined}, {name: settled_late}]
entities:
  Order:
    attributes:
      - {name: total, type: integer}
      - {name: line_item_count, type: integer}
      - {name: outcome, type: PaymentOutcome}
    actions: [{name: markPaid}, {name: cancel_order}]
    invariants:
      - {id: order_total_positive, statement: an order total is positive}
invariants:
  - {id: ledger_balanced, statement: the ledger balances}
`

const resolveArchitecture = "# A\n\n## Events\n\n| event | producer | consumer | payload | delivery |\n|---|---|---|---|---|\n" +
	"| `order_confirmed` | orderSvc | shippingSvc | `Order.id` | at-least-once |\n" +
	"| `Order.shipped` | orderSvc | shippingSvc | `Order.id` | at-least-once |\n"

const resolveHeader = "| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n"

func resolveRow(name, contract string) string {
	return "| `" + name + "` | action | `(ctx) -> ctx` | " + contract + " | - |\n"
}

// resolveDesign writes a design with a model, a machine carrying context
// keys, an event contract, a non-markdown design file, and the given rows
// under a named-unit table (plus a second unit, settle_payment).
func resolveDesign(t *testing.T, rows string) string {
	t.Helper()
	design := t.TempDir()
	must(t, os.MkdirAll(filepath.Join(design, "machines"), 0o755))
	must(t, os.MkdirAll(filepath.Join(design, "schemas"), 0o755))
	machine := strings.Replace(wiringMachine, `"initial": "Placed",`, `"initial": "Placed",
  "context": {"retry_count": 0, "attempts": 0},`, 1)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), machine)
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), resolveModel)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), resolveArchitecture)
	mustWrite(t, filepath.Join(design, "schemas", "Schema.avsc"), "{}\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), "# Order\n\n"+resolveHeader+
		resolveRow("settle_payment", "settles")+resolveRow("persistOrder", "persists")+rows)
	return design
}

func tokenWarns(g *Gate) []string {
	var out []string
	for _, w := range g.Warns {
		if strings.Contains(w, "undeclared fact reference") || strings.Contains(w, "is not a declared fact, action, unit or value") ||
			strings.Contains(w, "backticked tokens outside every declaration group") {
			out = append(out, w)
		}
	}
	return out
}

func TestLedgerResolvesBacktickedTokens(t *testing.T) {
	cases := []struct {
		class, silent, warns string
	}{
		{"model action, qualified", "Order.markPaid", "Order.markPaidd"},
		{"model action, bare", "cancel_order", "cancel_orders"},
		{"named unit, qualified by machine", "Order.persistOrder", "Order.persistOrders"},
		{"named unit, bare, another row", "settle_payment", "settle_payments"},
		{"enum value, snake_case form", "card_declined", "card_declines"},
		{"enum value, its own name", "settled_late", "settled_later"},
		{"context key, bare", "retry_count", "retry_counts"},
		{"context key, qualified by machine", "Order.attempts", "Order.attempt"},
		{"event from the event contract", "order_confirmed", "order_confirmed_v2"},
		{"event, entity-shaped", "Order.shipped", "Order.shippd"},
		{"entity invariant id", "order_total_positive", "order_total_negative"},
		{"model invariant id", "ledger_balanced", "ledger_balance"},
		{"file with a known extension", "ARCHITECTURE.md", "ARCHITECTURE.mdx"},
		{"file under the design", "Schema.avsc", "Other.avsc"},
	}
	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			g := CheckLedger(resolveDesign(t, resolveRow("closeOrder", "closes via `"+tc.silent+"`")))
			if got := tokenWarns(g); len(got) != 0 {
				t.Fatalf("`%s` resolves and must not warn: %q", tc.silent, got)
			}
			g = CheckLedger(resolveDesign(t, resolveRow("closeOrder", "closes via `"+tc.warns+"`")))
			want := "machines/Order.matrix.md:7: row 'closeOrder': `" + tc.warns + "` is not a declared fact, action, unit or value; drop the backticks or declare it"
			if got := tokenWarns(g); len(got) != 1 || got[0] != want {
				t.Fatalf("near-neighbour `%s` must get the unresolved wording:\n got %q\nwant %q", tc.warns, got, want)
			}
			if len(g.Errs) != 0 {
				t.Fatalf("the tier is a warning, never an error: %v", g.Errs)
			}
		})
	}
}

func TestLedgerAttributeKeepsTheDeclareWording(t *testing.T) {
	g := CheckLedger(resolveDesign(t, resolveRow("closeOrder", "reads `Order.total` and `line_item_count`")))
	want := []string{
		"machines/Order.matrix.md:7: row 'closeOrder': undeclared fact reference `Order.total`; declare it in USES{} or WRITES{} or drop the backticks",
		"machines/Order.matrix.md:7: row 'closeOrder': undeclared fact reference `line_item_count`; declare it in USES{} or WRITES{} or drop the backticks",
	}
	if got := tokenWarns(g); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("got %q", got)
	}
	if g.Counts["undeclared fact references"] != 2 {
		t.Fatalf("the attribute class is counted: %v", g.Counts)
	}
}

func TestLedgerResolvedClassesAreCounted(t *testing.T) {
	g := CheckLedger(resolveDesign(t, resolveRow("closeOrder",
		"`Order.markPaid`, `Order.persistOrder`, `card_declined`, `retry_count`, `order_confirmed`, `ledger_balanced`, `ARCHITECTURE.md`, `nothing_here`")))
	for label, n := range map[string]int{
		"backticked actions": 1, "backticked named units": 1, "backticked enum values": 1, "backticked context keys": 1,
		"backticked events": 1, "backticked invariant ids": 1, "backticked file names": 1, "unresolved backticked tokens": 1,
	} {
		if g.Counts[label] != n {
			t.Fatalf("count %q = %d, want %d (all: %v)", label, g.Counts[label], n, g.Counts)
		}
	}
}

// manyUnresolved returns n rows, each quoting one distinct unresolved token.
func manyUnresolved(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(resolveRow("step"+strconv.Itoa(i), "uses `ghost_token_"+strconv.Itoa(i)+"`"))
	}
	return b.String()
}

func TestLedgerSummarizesAFileAboveTheThreshold(t *testing.T) {
	t.Run("at the threshold every line prints", func(t *testing.T) {
		g := CheckLedger(resolveDesign(t, manyUnresolved(UndeclaredSummaryThreshold)))
		if got := tokenWarns(g); len(got) != UndeclaredSummaryThreshold {
			t.Fatalf("want %d lines, got %d: %q", UndeclaredSummaryThreshold, len(got), got)
		}
	})
	t.Run("above it one summary line per file", func(t *testing.T) {
		n := UndeclaredSummaryThreshold + 1
		g := CheckLedger(resolveDesign(t, manyUnresolved(n)+resolveRow("closeOrder", "reads `Order.total`")))
		got := tokenWarns(g)
		want := "machines/Order.matrix.md: 22 backticked tokens outside every declaration group (1 names a model attribute, to declare in USES{} or WRITES{}; 21 name no declared fact, action, unit or value), first `ghost_token_0`, `ghost_token_1`, `ghost_token_2`; machinery check --verbose lists each"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("got %q\nwant %q", got, want)
		}
		if g.Counts["unresolved backticked tokens"] != n || g.Counts["undeclared fact references"] != 1 {
			t.Fatalf("the checked line keeps the exact counts: %v", g.Counts)
		}
	})
	t.Run("--verbose prints every line", func(t *testing.T) {
		n := UndeclaredSummaryThreshold + 1
		g := CheckLedgerWith(resolveDesign(t, manyUnresolved(n)), LedgerOptions{Verbose: true})
		if got := tokenWarns(g); len(got) != n {
			t.Fatalf("want %d lines, got %d", n, len(got))
		}
	})
}
