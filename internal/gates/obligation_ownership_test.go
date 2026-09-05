package gates

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/oracle"
	"github.com/RamXX/machinery/internal/pack"
)

// These fixtures exercise the public gates over real files. Oracle IDs come
// from the production generator, not an invented ID-to-owner convention.
type obligationMachine struct {
	stem, guard string
	ids         []string
	guardID     string
}

func writeObligationMachine(t *testing.T, design, stem, guard, declaration string) obligationMachine {
	t.Helper()
	m := obligationMachine{stem: stem, guard: guard}
	path := filepath.Join(design, "machines", stem+".machine.json")
	writeSuiteFile(t, path, fmt.Sprintf(`{"id":%q,"initial":"Ready","states":{"Ready":{"on":{"advance":[{"target":"Done","guard":%q},{"actions":"recordDenied"}]}},"Done":{"type":"final"}}}`, strings.ToLower(stem), guard))
	body, err := oracle.Generate(path)
	if err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, filepath.Join(design, "machines", stem+".oracle.md"), body)
	_, m.ids = oracleTableIDs(body)
	for _, row := range oracleGuardRows(body) {
		if row.guard == guard {
			m.guardID = row.stableID
		}
	}
	if len(m.ids) != 2 || m.guardID == "" {
		t.Fatalf("fixture did not generate two transitions and one guarded row: %s", body)
	}
	writeObligationMatrix(t, design, m, declaration)
	return m
}

func writeObligationMatrix(t *testing.T, design string, m obligationMachine, declaration string) {
	t.Helper()
	writeSuiteFile(t, filepath.Join(design, "machines", m.stem+".matrix.md"),
		"| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n"+
			fmt.Sprintf("| `%s` | guard | `(ctx,evt) -> bool` | true iff permitted %s | - |\n", m.guard, declaration))
}

func appendObligationNarrative(t *testing.T, design, stem, narrative string) {
	t.Helper()
	path := filepath.Join(design, "machines", stem+".matrix.md")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, path, string(body)+"\n"+narrative+"\n")
}

func obligationIDs(m obligationMachine, clauses int) []string {
	ids := append([]string{}, m.ids...)
	for i := range clauses {
		ids = append(ids, m.guardID+string(rune('a'+i)))
	}
	return ids
}

func writeObligationTests(t *testing.T, impl string, ids ...string) {
	t.Helper()
	var body strings.Builder
	body.WriteString("package coverage\nimport \"testing\"\nfunc TestDecisions(t *testing.T) {\n")
	for _, id := range ids {
		fmt.Fprintf(&body, "t.Run(%q, func(t *testing.T) { allowed := true; if !allowed { t.Fatal(\"decision should allow\") } })\n", id)
	}
	body.WriteString("}\n")
	writeSuiteFile(t, filepath.Join(impl, "coverage_test.go"), body.String())
}

func selectedObligationGates(t *testing.T, design, impl, list string) map[string]*Gate {
	t.Helper()
	sel, run, _, err := SelectRunAndNote(design, impl, list, RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*Gate{}
	for _, gate := range run {
		for _, name := range []string{"gt", "gd"} {
			if strings.HasPrefix(strings.ToLower(gate.Title), name+"-") {
				out[name] = gate
			}
		}
	}
	for _, name := range strings.Split(list, ",") {
		if !sel.Run[name] || out[name] == nil {
			t.Fatalf("selected %s did not actually run: selection=%v gates=%v", name, sel.Run, out)
		}
	}
	return out
}

func requireObligationClean(t *testing.T, g *Gate) {
	t.Helper()
	if len(g.Errs)+len(g.Drift)+len(g.Warns) != 0 {
		t.Errorf("%s must be clean: errors=%v drift=%v warnings=%v", g.Title, g.Errs, g.Drift, g.Warns)
	}
}

func requireOwnerError(t *testing.T, g *Gate, owner, guard string) {
	t.Helper()
	for _, diagnostic := range g.Errs {
		if strings.Contains(diagnostic, owner) && strings.Contains(diagnostic, guard) {
			return
		}
	}
	t.Errorf("%s must reject and identify owner %s and guard %s; errors=%v warnings=%v", g.Title, owner, guard, g.Errs, g.Warns)
}

func TestObligationClausesIndependentMachines(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "CLAUSES{owner, balance, fresh}")
	writeObligationTests(t, impl, append(obligationIDs(a, 2), obligationIDs(b, 3)...)...)
	// The shared word 'owner' cannot make Beta's full vocabulary a partial
	// enumeration of Alpha's, or vice versa. Ownership is already in the path.
	appendObligationNarrative(t, design, "Alpha", "guardReady checks owner and open.")
	appendObligationNarrative(t, design, "Beta", "guardReady checks owner, balance and fresh.")
	for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
		requireObligationClean(t, g)
	}
	gt := CheckOracleCoverage(design, impl)
	if got := gt.Counts["falsifying-clause ids covered"]; got != 5 {
		t.Errorf("each of five owner-local clause IDs must count once, got %d: %v", got, gt.Counts)
	}
}

func TestObligationClausesUndeclaredSiblingIsNotArmed(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "")
	writeObligationTests(t, impl, append(obligationIDs(a, 2), b.ids...)...)
	appendObligationNarrative(t, design, "Beta", "guardReady checks owner.")
	for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
		requireObligationClean(t, g)
	}
}

func TestObligationClausesMissingLocalCaseNamesOwner(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "CLAUSES{owner, balance, fresh}")
	writeObligationTests(t, impl, append(obligationIDs(a, 2), obligationIDs(b, 2)...)...)
	gt := selectedObligationGates(t, design, impl, "gt")["gt"]
	requireOwnerError(t, gt, "Beta", "guardReady")
	if !hasErr(gt, b.guardID+"c") {
		t.Errorf("missing Beta clause must retain its generated stable ID: %v", gt.Errs)
	}
	if hasErr(gt, a.guardID+"c") {
		t.Errorf("Beta's third clause invented an obligation on Alpha: %v", gt.Errs)
	}
}

func TestObligationClausesOmittedMachineTests(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "CLAUSES{owner, balance}")
	writeObligationTests(t, impl, obligationIDs(a, 2)...)
	gt := selectedObligationGates(t, design, impl, "gt")["gt"]
	requireOwnerError(t, gt, "Beta", "guardReady")
	if !hasErr(gt, b.guardID) || !hasErr(gt, b.guardID+"a") || !hasErr(gt, b.guardID+"b") {
		t.Errorf("one machine's coverage cannot satisfy omitted sibling rows or clauses: %v", gt.Errs)
	}
}

func TestObligationClausesLocalDriftNamesOwner(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open} RETIRED{legacy}")
	b := writeObligationMachine(t, design, "Beta", "guardReady", "CLAUSES{owner, balance}")
	writeObligationTests(t, impl, append(obligationIDs(a, 2), obligationIDs(b, 2)...)...)
	appendObligationNarrative(t, design, "Alpha", "guardReady checks owner and legacy.")
	gd := selectedObligationGates(t, design, impl, "gd")["gd"]
	for _, required := range []string{"open missing", "RETIRED clause legacy"} {
		found := false
		for _, warning := range gd.Warns {
			if strings.Contains(warning, "Alpha") && strings.Contains(warning, "guardReady") && strings.Contains(warning, required) {
				found = true
			}
			if strings.Contains(warning, "balance missing") {
				t.Errorf("Beta's vocabulary leaked into Alpha drift check: %s", warning)
			}
		}
		if !found {
			t.Errorf("owner-local drift must report %q: %v", required, gd.Warns)
		}
	}
}

func TestObligationClausesInvalidDeclarationRejectedByBothGates(t *testing.T) {
	for _, tc := range []struct{ name, declaration string }{
		{"empty", "CLAUSES{}"},
		{"duplicate-clause", "CLAUSES{owner, owner}"},
		{"active-retired-conflict", "CLAUSES{owner, open} RETIRED{owner}"},
		{"missing-close-brace", "CLAUSES{owner, open"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			design, impl := t.TempDir(), t.TempDir()
			a := writeObligationMachine(t, design, "Alpha", "guardReady", tc.declaration)
			writeObligationTests(t, impl, obligationIDs(a, 2)...)
			for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
				requireOwnerError(t, g, "Alpha", "guardReady")
			}
		})
	}
}

func TestObligationClausesDuplicateDeclarationRejected(t *testing.T) {
	for _, second := range []string{"CLAUSES{owner, open}", "CLAUSES{owner, balance}"} {
		t.Run(second, func(t *testing.T) {
			design, impl := t.TempDir(), t.TempDir()
			a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
			path := filepath.Join(design, "machines", "Alpha.matrix.md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			writeSuiteFile(t, path, string(body)+fmt.Sprintf("| `guardReady` | guard | `(ctx,evt) -> bool` | true iff permitted %s | - |\n", second))
			writeObligationTests(t, impl, obligationIDs(a, 2)...)
			for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
				requireOwnerError(t, g, "Alpha", "guardReady")
			}
		})
	}
}

func TestObligationClausesOwnerCannotResolveThroughSibling(t *testing.T) {
	for _, mutation := range []string{"orphan-matrix", "empty-owner", "wrong-owner-guard", "removed-oracle"} {
		t.Run(mutation, func(t *testing.T) {
			design, impl := t.TempDir(), t.TempDir()
			a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
			b := writeObligationMachine(t, design, "Beta", "guardOther", "CLAUSES{balance, fresh}")
			writeObligationTests(t, impl, append(obligationIDs(a, 2), obligationIDs(b, 2)...)...)
			owner, guard := "Alpha", "guardReady"
			switch mutation {
			case "orphan-matrix", "empty-owner":
				owner = "Absent"
				if mutation == "empty-owner" {
					owner = ""
				}
				if err := os.Rename(filepath.Join(design, "machines", "Alpha.matrix.md"), filepath.Join(design, "machines", owner+".matrix.md")); err != nil {
					t.Fatal(err)
				}
				owner += ".matrix.md"
			case "wrong-owner-guard":
				a.guard, guard = "guardOther", "guardOther"
				writeObligationMatrix(t, design, a, "CLAUSES{balance, fresh}")
			case "removed-oracle":
				if err := os.Remove(filepath.Join(design, "machines", "Alpha.oracle.md")); err != nil {
					t.Fatal(err)
				}
			}
			for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
				requireOwnerError(t, g, owner, guard)
			}
		})
	}
}

func TestObligationClausesSingleMachinePositiveControl(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open} RETIRED{legacy}")
	writeObligationTests(t, impl, obligationIDs(a, 2)...)
	appendObligationNarrative(t, design, "Alpha", "guardReady checks owner and open.")
	for _, g := range selectedObligationGates(t, design, impl, "gt,gd") {
		requireObligationClean(t, g)
	}
}

func TestObligationClausesEveryLocalClauseRequired(t *testing.T) {
	for _, missing := range []string{"a", "b"} {
		t.Run(missing, func(t *testing.T) {
			design, impl := t.TempDir(), t.TempDir()
			a := writeObligationMachine(t, design, "Alpha", "guardReady", "CLAUSES{owner, open}")
			var ids []string
			for _, id := range obligationIDs(a, 2) {
				if id != a.guardID+missing {
					ids = append(ids, id)
				}
			}
			writeObligationTests(t, impl, ids...)
			gt := selectedObligationGates(t, design, impl, "gt")["gt"]
			requireOwnerError(t, gt, "Alpha", "guardReady")
			if !hasErr(gt, a.guardID+missing) {
				t.Errorf("missing local clause must retain generated ID %s: %v", a.guardID+missing, gt.Errs)
			}
			// Checking coverage must not rewrite IDs or the generated oracle.
			path := filepath.Join(design, "machines", "Alpha.machine.json")
			fresh, err := oracle.Generate(path)
			if err != nil {
				t.Fatal(err)
			}
			committed, err := os.ReadFile(filepath.Join(design, "machines", "Alpha.oracle.md"))
			if err != nil || string(committed) != fresh {
				t.Fatalf("coverage changed stable oracle bytes: %v", err)
			}
		})
	}
}

// Copy the complete recursive fixture, including child packs, so parent
// selection is not tested with a malformed decomposition marker.
func obligationParentFixture(t *testing.T) (string, string) {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("..", "..", "examples", "checkout-split")
	if err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		writeSuiteFile(t, filepath.Join(dst, rel), string(body))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	design, impl := filepath.Join(dst, "parent", "design"), filepath.Join(dst, "parent", "impl")
	writeSuiteFile(t, filepath.Join(impl, "go.mod"), "module example.com/parent\n\ngo 1.27.0\n\nrequire example.com/busdriver v0.0.0\nreplace example.com/busdriver => ../busdriver\n")
	// A real local dependency and mapped source exercise G4's allowed edge.
	// Keeping that gate clean prevents an unrelated empty-import failure
	// from satisfying the negative CLI assertion.
	writeSuiteFile(t, filepath.Join(dst, "parent", "busdriver", "go.mod"), "module example.com/busdriver\n\ngo 1.27.0\n")
	writeSuiteFile(t, filepath.Join(dst, "parent", "busdriver", "driver.go"), "package busdriver\nimport \"encoding/json\"\nfunc Encode(value any) ([]byte, error) { return json.Marshal(value) }\n")
	writeSuiteFile(t, filepath.Join(impl, "orders", "orders.go"), "package orders\nimport \"example.com/busdriver\"\nfunc EncodeOrder(id string) ([]byte, error) { return busdriver.Encode(map[string]string{\"id\": id}) }\n")
	if _, err := pack.LoadDecomposition(design); err != nil {
		t.Fatalf("complete decomposition fixture invalid: %v", err)
	}
	requireObligationClean(t, CheckPack(design))
	return design, impl
}

func addParentObligation(t *testing.T, design, name string) []string {
	t.Helper()
	// The committed decision tables are copied verbatim, preserving their
	// generated IDs and full allow/deny/unreachable vocabulary.
	body, err := os.ReadFile(filepath.Join("..", "..", "examples", "go-crm", "design", "formal", name))
	if err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, filepath.Join(design, "formal", name), string(body))
	_, ids := oracleTableIDs(string(body))
	if len(ids) == 0 {
		t.Fatal("relational fixture has no decision rows")
	}
	return ids
}

func TestObligationParentSelectionRetainsRelationalCoverage(t *testing.T) {
	for _, name := range formalOracleNames {
		t.Run(name, func(t *testing.T) {
			design, impl := obligationParentFixture(t)
			ids := addParentObligation(t, design, name)
			writeObligationTests(t, impl)
			sel, err := Select(design, "", impl)
			if err != nil {
				t.Fatal(err)
			}
			if !sel.Run["gt"] {
				t.Errorf("parent-owned %s obligations were erased by default selection: %v; %s", name, sel.Run, sel.Note)
			}
			gt := selectedObligationGates(t, design, impl, "gt")["gt"]
			if !hasErr(gt, ids[0]) {
				t.Errorf("explicit Gt must reject missing parent decision tests: %v", gt.Errs)
			}
			writeObligationTests(t, impl, ids...)
			requireObligationClean(t, selectedObligationGates(t, design, impl, "gt")["gt"])
		})
	}
}

func TestObligationParentRealCLI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "machinery")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, "./cmd/machinery")
	build.Dir = filepath.Join("..", "..")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real CLI: %v\n%s", err, out)
	}
	run := func(t *testing.T, design, impl, gates string) (string, error) {
		t.Helper()
		args := []string{"check", design, "--impl", impl}
		if gates != "" {
			args = append(args, "--gate", gates)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, bin, args...)
		out, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			t.Fatalf("CLI deadline exceeded: %v\n%s", ctx.Err(), out)
		}
		return string(out), err
	}
	t.Run("obligation-free-parent-control", func(t *testing.T) {
		design, impl := obligationParentFixture(t)
		out, err := run(t, design, impl, "")
		if err != nil {
			t.Fatalf("complete obligation-free parent must remain valid: %v\n%s", err, out)
		}
		out, err = run(t, design, impl, "gt")
		if err != nil || !strings.Contains(out, "0 machines") || !strings.Contains(out, "0 test files scanned") {
			t.Errorf("explicit zero-obligation result must be honest: %v\n%s", err, out)
		}
	})
	for _, name := range formalOracleNames {
		t.Run(name, func(t *testing.T) {
			design, impl := obligationParentFixture(t)
			ids := addParentObligation(t, design, name)
			writeObligationTests(t, impl, ids...)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			executeTests := exec.CommandContext(ctx, "go", "test", "-count=1", "./...")
			executeTests.Dir = impl
			if out, err := executeTests.CombinedOutput(); err != nil {
				t.Fatalf("parent positive test fixture must compile and execute: %v\n%s", err, out)
			}
			for _, gates := range []string{"gt", ""} {
				out, err := run(t, design, impl, gates)
				if err != nil {
					t.Errorf("covered parent must pass gate=%q: %v\n%s", gates, err, out)
				}
			}
			// Remove a real decision ID, retaining a valid executable test
			// file. Neither a compile failure nor zero test-file detection
			// can masquerade as the expected coverage rejection.
			writeObligationTests(t, impl, ids[1:]...)
			for _, gates := range []string{"gt", ""} {
				out, err := run(t, design, impl, gates)
				if err == nil || !strings.Contains(out, "Gt-tests") || !strings.Contains(out, ids[0]) || !strings.Contains(out, name) {
					t.Errorf("missing parent decision must reject gate=%q with oracle/ID evidence: %v\n%s", gates, err, out)
				}
				if bytes.Contains([]byte(out), []byte("platform-green")) {
					t.Errorf("missing parent tests must never report platform-green: %s", out)
				}
			}
		})
	}
}
