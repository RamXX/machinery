package gates

import (
	"strings"
	"testing"
)

// A group the design declares in the contract's private_groups: list is the
// design's own notation. The projection skips it exactly as Gx-trace does: it
// raises no problem, and nothing inside it becomes a fact. An undeclared
// unknown group on the same row is still a problem, and only that row is
// omitted.
func TestFactsSkipDeclaredPrivateGroups(t *testing.T) {
	matrix := "| name | kind | pre / post |\n|---|---|---|\n" +
		"| `persist` | actor | OWNED-BY{Order.status} WRITES{Order.status} |\n" +
		"| `canPay` | guard | OWNED-BY{Order.total} USES{Order.total} |\n"
	t.Run("declared", func(t *testing.T) {
		design := writeFactsDesign(t, t.TempDir(), map[string]string{
			"ARCHITECTURE.md":             privateContract("private_groups: [OWNED-BY]\n"),
			"machines/Order.machine.json": factsMachine,
			"machines/Order.matrix.md":    matrix,
		})
		rep, err := projectDesignFacts(design)
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.problems) != 0 {
			t.Fatalf("a declared private group is not a projection problem, got %v", rep.problems)
		}
		if got := strings.Join(factRows(rep.facts, "unit_writes"), " "); got != "Order.persist|Order.status" {
			t.Fatalf("the public group beside a private one still projects: %s", got)
		}
		if got := strings.Join(factRows(rep.facts, "unit_uses"), " "); got != "Order.canPay|Order.total" {
			t.Fatalf("the public group beside a private one still projects: %s", got)
		}
		for _, row := range factRows(rep.facts, "unit_declares") {
			if strings.Contains(row, "OWNED-BY") {
				t.Fatalf("a private group must never be projected as a declaration: %s", row)
			}
		}
	})
	t.Run("undeclared", func(t *testing.T) {
		design := writeFactsDesign(t, t.TempDir(), map[string]string{
			"ARCHITECTURE.md":             privateContract(""),
			"machines/Order.machine.json": factsMachine,
			"machines/Order.matrix.md":    matrix,
		})
		rep, err := projectDesignFacts(design)
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.problems) != 2 {
			t.Fatalf("each undeclared unknown group fails its own row, got %v", rep.problems)
		}
		if got := strings.Join(factRows(rep.facts, "unit_writes"), " "); got != "" {
			t.Fatalf("an omitted row projects nothing: %s", got)
		}
	})
}
