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
	for _, rel := range bundledDesigns(t) {
		design := filepath.Join("..", "..", filepath.FromSlash(rel))
		facts, err := LoadDesignFacts(design)
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
	if souffle != "" {
		t.Logf("rules parity: %d rule files x examples ran under both engines", both)
		if want := len(set.files) * len(bundledDesigns(t)); both != want {
			t.Fatalf("%d of %d rule file runs compared", both, want)
		}
	}
}
