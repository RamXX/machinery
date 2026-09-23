package gates

// The load-bearing test of the shipped rules: every rule file, over every
// bundled example's facts, yields byte-identical output relations under
// native Soufflé and under internal/datalog. The facts are written to disk
// exactly as `machinery project --facts` writes them, plus an empty file for
// every catalog relation of a layer the design lacks (the gate supplies those
// relations empty in process; Soufflé needs the file). Soufflé does not fix
// the row order of its outputs, so both sides are compared after sorting
// their lines. Without souffle on PATH the Soufflé half is skipped with a
// message; the in-process half still runs.

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/datalog"
	"github.com/RamXX/machinery/rules"
)

// writeRuleFacts writes a design's facts directory as the gate's rules see it.
func writeRuleFacts(t *testing.T, facts *checker.DesignFacts, dir string) {
	t.Helper()
	files, err := facts.FactsFiles()
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range checker.RelationCatalog() {
		if _, ok := files[spec.Name+".facts"]; !ok {
			files[spec.Name+".facts"] = nil
		}
	}
	for name, body := range files {
		must(t, os.WriteFile(filepath.Join(dir, name), body, 0o644))
	}
}

func sortedOutputLines(s string) string {
	lines := strings.SplitAfter(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	sort.Strings(lines)
	return strings.Join(lines, "")
}

func TestRulesParity(t *testing.T) {
	souffle, _ := exec.LookPath("souffle")
	if souffle == "" {
		t.Log("souffle not on PATH: the Soufflé half of the rules parity is skipped; install souffle (or run the CI datalog-parity job) to run it")
	}
	set, err := shippedRules()
	if err != nil {
		t.Fatal(err)
	}
	both := 0
	type target struct{ name, path string }
	var targets []target
	for _, rel := range bundledDesigns(t) {
		targets = append(targets, target{rel, filepath.Join("..", "..", filepath.FromSlash(rel))})
	}
	// The examples fire few rules; the synthetic design fires the rest, so
	// parity also covers populated outputs of every rule body.
	targets = append(targets, target{"synthetic/every-rule-fires", writeFactsDesign(t, t.TempDir(), everyRuleFires)})
	nonEmpty := map[string]bool{}
	for _, tg := range targets {
		rel := tg.name
		facts, err := LoadDesignFacts(tg.path)
		if err != nil {
			t.Fatal(err)
		}
		factsDir := t.TempDir()
		writeRuleFacts(t, facts, factsDir)
		fromFiles, err := datalog.ReadFacts(factsDir)
		if err != nil {
			t.Fatal(err)
		}
		inProcess := ruleInputs(facts)
		for _, rf := range set.files {
			t.Run(rel+"/"+rf.name, func(t *testing.T) {
				viaFiles, err := rf.prog.Run(fromFiles, datalog.Options{})
				if err != nil {
					t.Fatal(err)
				}
				viaGate, err := rf.prog.Run(inProcess, datalog.Options{Explain: true})
				if err != nil {
					t.Fatal(err)
				}
				for _, out := range rf.prog.OutputRelations() {
					if viaFiles.FormatCSV(out) != viaGate.FormatCSV(out) {
						t.Fatalf("%s: the facts files and the in-process facts disagree", out)
					}
					if viaGate.FormatCSV(out) != "" {
						nonEmpty[out] = true
					}
				}
				if souffle == "" {
					return
				}
				outDir := t.TempDir()
				src, err := rules.Consistency.ReadFile(path.Join(rules.ConsistencyDir, rf.name))
				if err != nil {
					t.Fatal(err)
				}
				program := filepath.Join(t.TempDir(), rf.name)
				must(t, os.WriteFile(program, src, 0o644))
				cmd := exec.CommandContext(t.Context(), souffle, "-F", factsDir, "-D", outDir, program)
				if msg, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("souffle rejected %s: %v\n%s", rf.name, err, msg)
				}
				for _, out := range rf.prog.OutputRelations() {
					theirs, err := os.ReadFile(filepath.Join(outDir, out+".csv"))
					if err != nil {
						t.Fatal(err)
					}
					if got, want := sortedOutputLines(viaGate.FormatCSV(out)), sortedOutputLines(string(theirs)); got != want {
						t.Fatalf("%s differs\n internal/datalog:\n%s\n souffle:\n%s", out, got, want)
					}
				}
				both++
			})
		}
	}
	for _, set := range set.files {
		for _, o := range set.outputs {
			if !nonEmpty[o.relation] {
				t.Errorf("parity never compared a populated %s; extend everyRuleFires", o.relation)
			}
		}
	}
	if souffle != "" {
		t.Logf("rules parity: %d rule file runs (%d files x %d designs) compared under both engines", both, len(set.files), len(targets))
		if want := len(set.files) * len(targets); both != want {
			t.Fatalf("%d of %d rule file runs compared", both, want)
		}
	}
}

// everyRuleFires is a design on which every shipped output relation has at
// least one tuple. TypeD is claimed twice: the contract row declaring it and
// the migration.yaml disposition treating it as a legacy type.
var everyRuleFires = map[string]string{
	"domain.modelith.yaml": `kind: DomainModel
version: v1
enums:
  OrderStatus:
    values: [{name: Placed}, {name: Paid}]
entities:
  Order:
    attributes:
      - {name: status, type: OrderStatus}
      - {name: total, type: integer}
    actions:
      - {name: pay, actor: System}
      - {name: refund, actor: System}
      - {name: view}
`,
	"machines/Order.machine.json": factsMachine,
	"machines/Order.matrix.md": "| name | kind | event | pre / post |\n|---|---|---|---|\n" +
		"| `canPay` | guard | - | USES{Order.stat} VALUES OrderStatus{Placed, Voided} CARRIES{signal:paid} |\n" +
		"| `persist` | actor | - | WRITES{Order.status} PRODUCES{Order.settle} |\n" +
		"| `announce` | action | `order.paid` | payload {Order.id} |\n" +
		"| `reasonA` | guard | - | VALUES reason{late, lost} |\n" +
		"| `reasonB` | guard | - | VALUES reason{late, early} |\n",
	"ARCHITECTURE.md": "# Architecture\n\n| event | producer | consumer | delivery | payload |\n|---|---|---|---|---|\n" +
		"| `order.paid` | app | ledger | at-least-once | `Order.id`, `Order.total` |\n\n" +
		"| type | replaces |\n|---|---|\n" +
		"| TypeA | SUPERSEDES{type:TypeB} |\n| TypeB | SUPERSEDES{type:TypeC} |\n| TypeC | SUPERSEDES{type:TypeA} |\n" +
		"| TypeD | SUPERSEDES{type:Ghost} |\n| DraftContract | RESERVED{type:TypeA} |\n\n" +
		"| component | placement | persistence |\n|---|---|---|\n" +
		"| `Order` (no machine: a stale waiver) | in-process | row |\n" +
		"| `Receipt` (no machine: an append-only record) | in-process | row |\n",
	"slices.yaml":                "milestones:\n  - id: M1\n    slices:\n      - id: M1-S1\n        cites:\n          - row:ARCHITECTURE.md#types#TypeB\n          - row:ARCHITECTURE.md#types#TypeD\n",
	"machines/Ledger.matrix.md":  "| name | kind | pre / post |\n|---|---|---|\n| `checkLedger` | guard | - |\n",
	"machines/Receipt.matrix.md": "| name | kind | pre / post |\n|---|---|---|\n| `checkReceipt` | guard | - |\n",
	"migration.yaml":             "contract_version: 1\nmode: rebuild\ndispositions:\n  - legacy: TypeD\n    target: Order\n    strategy: replace\n    rationale: r\n",
	"AUTHORIZATION.md": "<!-- machinery:authorization-inventory -->\n\n| authorization subject | admission |\n|---|---|\n" +
		"| Order.pay | `nowhere` |\n| Order.view | `nowhere` |\n",
}
