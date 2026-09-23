package gates

import (
	"fmt"
	"strings"
	"testing"
)

// NoMachineWaivers is the one reader of the contract-only declaration: the
// waiver in the component cell or in the machine-placement cell, the row's
// first component, an empty reason kept as "" (the caller decides it is no
// waiver), and a '(not placed: ...)' row ignored.
func TestNoMachineWaivers(t *testing.T) {
	arch := "# Architecture\n\n## Placement\n\n" +
		"| component | machine placement | persistence |\n|---|---|---|\n" +
		"| `Order` | in-process | row |\n" +
		"| `Receipt` (no machine: append-only record) | - | row |\n" +
		"| `Ledger` + `Entry` | (no machine: rows written with their `Order`) | row |\n" +
		"| `Blank` (no machine: ) | - | row |\n" +
		"| `Value` (not placed: a value object; no machine: n/a) | - | - |\n" +
		"| prose row (no machine: nothing named) | - | - |\n\n" +
		"| event | producer | consumer |\n|---|---|---|\n| `Stray` (no machine: not a placement table) | a | b |\n"
	var got []string
	for _, w := range NoMachineWaivers(arch) {
		got = append(got, fmt.Sprintf("%s|%s|%d", w.Component, w.Reason, w.Line))
	}
	want := "Receipt|append-only record|8,Ledger|rows written with their `Order`|9,Blank||10"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v, want %s", got, want)
	}
}

// The projection reads the same waivers: one no_machine_waiver per waiver
// with a reason, and one matrix row per matrix file, machine or not.
func TestProjectionNoMachineWaiverAndMatrix(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md":    "| name | kind | pre / post |\n|---|---|---|\n| `canPay` | guard | - |\n",
		"machines/Receipt.matrix.md":  "| name | kind | pre / post |\n|---|---|---|\n| `checkReceipt` | guard | - |\n",
		"ARCHITECTURE.md": "# Architecture\n\n| component | placement | persistence |\n|---|---|---|\n" +
			"| `Receipt` (no machine: append-only record) | in-process | row |\n| `Blank` (no machine:) | in-process | row |\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factRows(facts, "no_machine_waiver"), ","); got != "Receipt" {
		t.Fatalf("no_machine_waiver = %s", got)
	}
	if got := strings.Join(factRows(facts, "matrix"), ","); got != "Order,Receipt" {
		t.Fatalf("matrix = %s", got)
	}
	if got := contractOnlyMatrices(design); len(got) != 1 || !got["Receipt"] {
		t.Fatalf("contractOnlyMatrices = %v", got)
	}
}

// RESERVED{type:X} projects reserved(X, row) and gives the reserving row no
// ownership; a slices.yaml row: citation projects packet_cites(slice, key),
// and a malformed row citation (Gw-packet's to report) projects nothing.
func TestProjectionReservedAndPacketCites(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"ARCHITECTURE.md": "# Architecture\n\n## Types\n\n| type | note |\n|---|---|\n" +
			"| WireDraft | RESERVED{type:Receipt, type:Manifest} |\n| Receipt | SUPERSEDES{type:ReceiptV0} |\n",
		"slices.yaml": "milestones:\n  - id: M1\n    slices:\n      - id: M1-S1\n        cites:\n" +
			"          - row:ARCHITECTURE.md#types#ReceiptV0\n          - row:ARCHITECTURE.md#types\n          - section:ARCHITECTURE.md#types\n",
	})
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	for rel, want := range map[string]string{
		"reserved":     "Manifest|WireDraft,Receipt|WireDraft",
		"type_owner":   "Receipt|ARCHITECTURE.md",
		"packet_cites": "M1-S1|ReceiptV0",
	} {
		if got := strings.Join(factRows(facts, rel), ","); got != want {
			t.Fatalf("%s = %s, want %s", rel, got, want)
		}
	}
}

// The binding table parses into milestone_says_*: a test id normalizes to
// its stable id, backticks are tolerated, several paths split on commas.
func TestProjectionMilestoneBindings(t *testing.T) {
	files := map[string]string{
		"machines/Order.oracle.md": "| test id | stable id |\n|---|---|\n| T-ORD-01 | ORD-aaaaaa |\n",
		"BUILD.md":                 "# BUILD\n\n| oracle | bound at |\n|---|---|\n| T-ORD-01 | `a_test.go`, b/c_test.go |\n| ORD-cccccc | Unbound |\n| | skipped |\n",
	}
	facts, err := LoadDesignFacts(writeFactsDesign(t, t.TempDir(), files))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(factRows(facts, "milestone_says_bound"), ","); got != "ORD-aaaaaa|a_test.go,ORD-aaaaaa|b/c_test.go" {
		t.Fatalf("milestone_says_bound = %s", got)
	}
	if got := strings.Join(factRows(facts, "milestone_says_unbound"), ","); got != "ORD-cccccc" {
		t.Fatalf("milestone_says_unbound = %s", got)
	}
	if n := len(facts.Rows("bound_at")) + len(facts.Rows("test_file")); n != 0 {
		t.Fatalf("without --impl the implementation relations are empty, got %d rows", n)
	}
	for _, cell := range []string{"bound in a_test.go", "../escape_test.go", "/abs/a_test.go", "a_test.go,"} {
		files["BUILD.md"] = "# BUILD\n\n| oracle | bound at |\n|---|---|\n| ORD-aaaaaa | " + cell + " |\n"
		if _, err := LoadDesignFacts(writeFactsDesign(t, t.TempDir(), files)); err == nil || !strings.Contains(err.Error(), "must be the literal unbound or test file paths") {
			t.Fatalf("%q: got %v", cell, err)
		}
	}
	files["BUILD.md"] = "# BUILD\n\n| oracle | bound at |\n|---|---|\n| ORD aaaaaa | unbound |\n"
	if _, err := LoadDesignFacts(writeFactsDesign(t, t.TempDir(), files)); err == nil || !strings.Contains(err.Error(), "is not one oracle id") {
		t.Fatalf("a malformed row key must fail: %v", err)
	}
}

// The suite hands --impl to Gy-rules: with it the binding twins fire, and
// without it the same design is silent on them.
func TestRulesMilestoneBindingsNeedImpl(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), everyRuleFires)
	impl := writeFactsDesign(t, t.TempDir(), everyRuleFiresImpl)
	codes := func(impl string) string {
		_, run, _, err := SelectRunAndNote(design, impl, "gy", RunOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, e := range run[0].Errs {
			if strings.Contains(e, "milestone_binding") {
				got = append(got, e)
			}
		}
		return strings.Join(got, "\n")
	}
	want := "BUILD.md:7: row 'ORD-aaaaaa': milestone_binding_stale\n" +
		"BUILD.md:8: row 'ORD-bbbbbb': milestone_binding_phantom (path 'order_test.go')"
	if got := codes(impl); got != want {
		t.Fatalf("with --impl:\n%s\nwant:\n%s", got, want)
	}
	if got := codes(""); got != "" {
		t.Fatalf("without --impl the binding rules must stay silent: %s", got)
	}
}
