package gates

// Agreement, after the cutover: the rules alone decide. Stage 3 compared the
// subjects of the 0.9.0 Gx prose heuristics with the finding_ subjects and
// wrote down three structural discrepancies. The heuristics are gone; each
// discrepancy is kept here as a fixture whose rule verdict is asserted
// exactly, together with the declared form that settles it, so a rule change
// that moves any of them fails until it is written down.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type agreementCase struct {
	name  string
	files map[string]string
	// rules is the exact set of finding subjects, "<code> <tuple joined by |>".
	rules []string
	// gl is a substring the Gl-ledger warnings must contain ("" for none).
	gl string
	// why records the Stage 3 verdict this case pins.
	why string
}

const agreementMachine = `{"id":"Order","initial":"Open","states":{"Open":{"type":"final"}}}`

var agreementCases = []agreementCase{
	{
		name: "read-only System action stated in prose",
		files: map[string]string{
			"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n" +
				"    actions: [{name: recompute, actor: System, description: \"Recompute the displayed total; writes nothing.\"}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | computes the total |\n",
		},
		rules: []string{"authz_missing Order.recompute"},
		why:   "prose never declares: the description's claim leaves the obligation in place until the unit declares WRITES{}",
	},
	{
		name: "read-only System action declared",
		files: map[string]string{
			"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n" +
				"    actions: [{name: recompute, actor: System}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | computes the total. WRITES{} |\n",
		},
		why: "WRITES{} on the unit of the same id declares the action read-only",
	},
	{
		name: "fact named only in prose",
		files: map[string]string{
			"domain.modelith.yaml":        "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n    actions: [{name: recompute}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | reads `missing_fact` / records result |\n",
		},
		gl:  "undeclared fact reference `missing_fact`",
		why: "a bare backticked token is prose: no fact to resolve, and Gl-ledger warns on it",
	},
	{
		name: "fact declared and unresolved",
		files: map[string]string{
			"domain.modelith.yaml":        "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n    actions: [{name: recompute}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | USES{missing_fact} |\n",
		},
		rules: []string{"fact_unresolved Order.recompute|missing_fact"},
		why:   "a USES{} member is a fact, and one no declaration supplies is a finding",
	},
	{
		name: "unnamed VALUES group on a case-variant unit name",
		files: map[string]string{
			"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nenums:\n  OrderState:\n    values: [{name: Open}, {name: Closed}]\n" +
				"entities:\n  Order:\n    attributes: [{name: state, type: OrderState}]\n    actions: [{name: classify}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `orderState` | guard | VALUES{Open, Closed, Voided} |\n",
		},
		gl:  "differs from enum 'OrderState' only in case",
		why: "binding is by exact name, so the group binds no enum; the intent is caught by the Gl warning, not by a fuzzy join",
	},
	{
		name: "named VALUES group",
		files: map[string]string{
			"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nenums:\n  OrderState:\n    values: [{name: Open}, {name: Closed}]\n" +
				"entities:\n  Order:\n    attributes: [{name: state, type: OrderState}]\n    actions: [{name: classify}]\n",
			"machines/Order.machine.json": agreementMachine,
			"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `orderState` | guard | VALUES OrderState{Open, Closed, Voided} |\n",
		},
		rules: []string{"values_disagree Order.orderState|OrderState|Voided"},
		why:   "the named group binds the enum and its extra member is a finding",
	},
}

func TestRulesAgreementFixtures(t *testing.T) {
	set, err := shippedRules()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range agreementCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.why == "" {
				t.Fatal("every case records its verdict")
			}
			dir := t.TempDir()
			for rel, body := range tc.files {
				path := filepath.Join(dir, filepath.FromSlash(rel))
				must(t, os.MkdirAll(filepath.Dir(path), 0o755))
				must(t, os.WriteFile(path, []byte(body), 0o644))
			}
			facts, err := LoadDesignFacts(dir)
			if err != nil {
				t.Fatal(err)
			}
			findings, errs := evaluateRules(set, facts)
			if len(errs) > 0 {
				t.Fatal(errs)
			}
			var got []string
			for _, f := range findings {
				got = append(got, f.output.code+" "+strings.Join(f.tuple, "|"))
			}
			sort.Strings(got)
			want := append([]string(nil), tc.rules...)
			sort.Strings(want)
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("rules report %v, want %v (%s)", got, want, tc.why)
			}
			warns := strings.Join(CheckLedger(dir).Warns, "\n")
			if tc.gl != "" && !strings.Contains(warns, tc.gl) {
				t.Fatalf("Gl-ledger must warn %q: %s", tc.gl, warns)
			}
		})
	}
}
