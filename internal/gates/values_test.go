package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func valuesFixture(t *testing.T, rows string) *Gate {
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
	return CheckTraceability(design)
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
