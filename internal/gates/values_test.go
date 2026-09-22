package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func valuesFixture(t *testing.T, rows string) *Gate {
	t.Helper()
	return CheckTraceability(valuesDesign(t, rows))
}

func valuesDesign(t *testing.T, rows string) string {
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
		"machine: Order\npattern: control-flow-only\nreason: vocabulary fixture\n")
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"),
		"| name | kind | contract (pre / post) | maps to |\n|---|---|---|---|\n"+rows)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"),
		"# A\n\n## Placement\n\n| component (placement) | persistence |\n|---|---|\n| `Order` | in-memory |\n")
	return design
}

func TestValuesH2ClassCFalsePositives(t *testing.T) {
	cases := []struct{ name, prose string }{
		{"pass identity", "with the pass identity (mapping id plus retirement version) as the reason class"},
		{"composite narrative", "stores the pass identity and the refusal class; idempotent by tenant and subject"},
		{"unrelated long cell", "a closed producer is checked first, then the guard verifies provenance. THE CLAUSE VOCABULARY IS UNCHANGED"},
		{"human reason", "records the refused acceptance with the acting reviewer, the reason class (the finding is advisory against pending-effective text)"},
		{"related owner", "the submitted classification is not in the closed vocabulary the item's `RiskMethodology` declares"},
		{"related value", "records the refused value and the closed vocabulary it was checked against"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := valuesDesign(t, "| `h2Unit` | guard | "+tc.prose+" | `order-paid-final` |\n")
			if tc.name == "related owner" || tc.name == "related value" {
				model := strings.Replace(factModel, "entities:\n", "entities:\n  RiskMethodology:\n    definition: the classification vocabulary is declared by the methodology\n", 1)
				model = strings.Replace(model, "  Order:\n", "  Order:\n    definition: the classification vocabulary lives in RiskMethodology definitions\n", 1)
				mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
			}
			g := CheckTraceability(design)
			if hasErr(g, "closed vocabulary has no VALUES") {
				t.Fatalf("H2 reference is not a local closed set: %v", g.Errs)
			}
		})
	}
	t.Run("owned Modelith enum", func(t *testing.T) {
		design := valuesDesign(t, "| `guardRetentionTriggerNamed` | guard | the closed trigger set is intake, supersession, relationship end, asset retirement, or declared event; `trigger` is a NOT NULL enum column | `order-paid-final` |\n")
		model := strings.Replace(factModel, "  OrderState:\n", "  RetentionTrigger:\n    values: [{name: intake}, {name: supersession}]\n  OrderState:\n", 1)
		model = strings.Replace(model, "      - {name: state, type: OrderState}", "      - {name: trigger, type: RetentionTrigger}\n      - {name: state, type: OrderState}", 1)
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
		g := CheckTraceability(design)
		if hasErr(g, "closed vocabulary has no VALUES") {
			t.Fatalf("entity's typed trigger owns the set: %v", g.Errs)
		}
	})
}

func TestValuesRequiresDeclarationForClosedProse(t *testing.T) {
	g := valuesFixture(t, "| `coverageGapReason` | guard | accepts a closed three-value vocabulary | `order-paid-final` |\n")
	if !hasErr(g, "closed vocabulary has no VALUES{...} declaration") {
		t.Fatalf("closed prose without a declaration must fail: %v", g.Errs)
	}
}

func TestValuesMatchesSameNamedModelEnum(t *testing.T) {
	g := valuesFixture(t, "| `orderState` | guard | VALUES{Paid, Placed} | `order-paid-final` |\n")
	if hasErr(g, "VALUES") {
		t.Fatalf("an exact enum twin must pass regardless of member order: %v", g.Errs)
	}
	g = valuesFixture(t, "| `orderState` | guard | VALUES{Placed, Cancelled} | `order-paid-final` |\n")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"VALUES mismatch", "Cancelled", "Paid", "Placed"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("an enum mismatch must report %q and both sets: %v", want, g.Errs)
		}
	}
}

func TestValuesCanBeSoleClosedSourceWithoutEnum(t *testing.T) {
	g := valuesFixture(t, "| `coverageGapReason` | guard | reason class VALUES{missing, stale, conflicting} | `order-paid-final` |\n")
	if hasErr(g, "VALUES") || hasErr(g, "closed vocabulary") {
		t.Fatalf("one declaration closes a vocabulary with no model enum: %v", g.Errs)
	}
}

func TestValuesRejectsMalformedDuplicateAndRepeatedGroups(t *testing.T) {
	cases := []struct{ row, want string }{
		{"VALUES{}", "has no members"},
		{"VALUES{open, open}", "duplicate member 'open'"},
		{"VALUES{open, closed} VALUES{open, closed}", "exactly one VALUES"},
	}
	for _, tc := range cases {
		g := valuesFixture(t, "| `reasonClass` | guard | "+tc.row+" | `order-paid-final` |\n")
		if !hasErr(g, tc.want) {
			t.Fatalf("%s must fail with %q: %v", tc.row, tc.want, g.Errs)
		}
	}
}

func TestValuesWordInProseDoesNotArmDeclaration(t *testing.T) {
	g := valuesFixture(t, "| `saveOrder` | actor | The VALUES of the two new columns are described elsewhere. | `order-paid-final` |\n")
	if hasErr(g, "VALUES") {
		t.Fatalf("uppercase prose is not a declaration: %v", g.Errs)
	}
}

func TestValuesOpenedGroupIsMalformed(t *testing.T) {
	g := valuesFixture(t, "| `saveOrder` | actor | VALUES{open, closed | `order-paid-final` |\n")
	if !hasErr(g, "malformed VALUES declaration") {
		t.Fatalf("opened group must fail closed: %v", g.Errs)
	}
}

func TestValuesNamedVocabularyReconcilesAcrossUnits(t *testing.T) {
	rows := "| `firstUnit` | guard | reason class VALUES refusal_reason{missing, stale} | `order-paid-final` |\n" +
		"| `secondUnit` | guard | reason class VALUES refusal_reason{missing, conflicting} | `order-paid-final` |\n"
	g := valuesFixture(t, rows)
	if !hasErr(g, "conflicting VALUES for 'refusal_reason'") {
		t.Fatalf("one named vocabulary cannot carry two member sets: %v", g.Errs)
	}
}
