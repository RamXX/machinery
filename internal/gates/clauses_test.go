package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/oracle"
)

func clauseDesign(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	oracle := "| T-DEAL-01 | DEAL-abc123 | a | b | c | d | - |\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.oracle.md"), []byte(oracle), 0644); err != nil {
		t.Fatal(err)
	}
	matrix := "| name | kind | sig | contract | maps | test | fixture |\n|---|---|---|---|---|---|---|\n" +
		"| `guardCloseEvidenced` | guard | s | true iff an artifact ops holds exists CLAUSES{resolved-task, applied-record} RETIRED{sop-coverage} | inv `x-y` | unit | f |\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(matrix), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestClauseRetiredSurvivorWarns(t *testing.T) {
	d := clauseDesign(t)
	body := "Positive coverage for guardCloseEvidenced needs resolved-task, applied-record, and sop-coverage each independently.\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "RETIRED clause sop-coverage") {
			found = true
		}
	}
	if !found {
		t.Fatalf("retired survivor not flagged: %v", g.Warns)
	}
}

func TestClausePartialEnumerationWarns(t *testing.T) {
	d := clauseDesign(t)
	body := "the falsifying case: guardCloseEvidenced with only resolved-task present\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "applied-record missing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("partial enumeration not flagged: %v", g.Warns)
	}
}

func TestClauseFullOrZeroEnumerationSilent(t *testing.T) {
	d := clauseDesign(t)
	body := "guardCloseEvidenced reads resolved-task and applied-record together.\n" +
		"guardCloseEvidenced refuses a bare close with no evidence.\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "guardCloseEvidenced") {
			t.Fatalf("full or zero enumeration flagged: %v", g.Warns)
		}
	}
}

func TestClauseLedgersExempt(t *testing.T) {
	d := clauseDesign(t)
	body := "the old guardCloseEvidenced read resolved-task alone here\n"
	if err := os.WriteFile(filepath.Join(d, "STATE.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "STATE.md") {
			t.Fatalf("ledger judged for clause drift: %v", g.Warns)
		}
	}
}

// clauseOwnerDesign writes one machine with its generated oracle and the
// matrix under test, so declaration collection runs over real owner artifacts.
func clauseOwnerDesign(t *testing.T, machine, matrix string) string {
	t.Helper()
	d := t.TempDir()
	path := filepath.Join(d, "machines", "Deal.machine.json")
	mustWrite(t, path, machine)
	body, err := oracle.Generate(path)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(d, "machines", "Deal.oracle.md"), body)
	mustWrite(t, filepath.Join(d, "machines", "Deal.matrix.md"), matrix)
	return d
}

const clauseGuardedMachine = `{"id":"deal","initial":"ready","states":{"ready":{"on":{"advance":[{"target":"done","guard":"guardReady"},{"actions":"recordDenied"}]}},"done":{"type":"final"}}}`

// An insert-only machine enforces its invariants on the creation edge; no
// transition carries a guard cell.
const clauseUnguardedMachine = `{"id":"deal","initial":"ready","states":{"ready":{"on":{"give":{"target":"applied"}}},"applied":{"type":"final"}}}`

const clauseGuardRow = "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n" +
	"| `guardReady` | guard | `(ctx,evt) -> bool` | true iff permitted CLAUSES{owner, open} | - |\n"

// A matrix narrates its invariant coverage in prose, and that prose quotes the
// guard's clause vocabulary. v0.6.11 read declarations from table rows only,
// so a sentence was never a declaration; 0.7.0 must not report one as a
// malformed declaration with a sentence fragment for a guard name.
func TestClauseProseMentionIsNotADeclaration(t *testing.T) {
	matrix := clauseGuardRow +
		"\n## Invariant coverage (attested)\n\n" +
		"- `deal-owner-backed` - `guardReady` enforces both of its declared clauses,\n" +
		"  CLAUSES{owner, open}: the owning principal and the open stage, and the first is a\n" +
		"  `guardReady` CLAUSES{owner, open}\n" +
		"  continuation line that wraps the vocabulary across the paragraph.\n"
	g := NewGate("clause prose")
	decls := collectClauseDecls(g, clauseOwnerDesign(t, clauseGuardedMachine, matrix))
	if len(g.Errs) != 0 || len(g.Warns) != 0 {
		t.Fatalf("prose quoting a clause vocabulary is not a declaration: errors=%v warnings=%v", g.Errs, g.Warns)
	}
	if len(decls) != 1 || decls[0].guard != "guardReady" || decls[0].owner != "Deal" {
		t.Fatalf("the guard row is the one declaration: %+v", decls)
	}
}

// A guard no oracle in the design governs carries zero falsifying-clause
// obligations, exactly as in v0.6.11. The ownership rule that survives is the
// one MAC-olrx closed: a sibling machine's oracle cannot supply the rows.
func TestClauseGuardGovernedByNoOracleIsAccepted(t *testing.T) {
	g := NewGate("clause unguarded owner")
	decls := collectClauseDecls(g, clauseOwnerDesign(t, clauseUnguardedMachine, clauseGuardRow))
	if len(g.Errs) != 0 || len(g.Warns) != 0 {
		t.Fatalf("a guard on a creation edge no oracle row governs must not be a finding: errors=%v warnings=%v", g.Errs, g.Warns)
	}
	if len(decls) != 1 || len(decls[0].rows) != 0 {
		t.Fatalf("the declaration stands with no oracle rows to owe: %+v", decls)
	}
}

// The sibling-resolution defect stays closed: Deal declares a guard only
// Other's oracle governs, and that is still an ownership error.
func TestClauseGuardGovernedOnlyBySiblingStillFails(t *testing.T) {
	design := clauseOwnerDesign(t, clauseUnguardedMachine, clauseGuardRow)
	path := filepath.Join(design, "machines", "Other.machine.json")
	mustWrite(t, path, clauseGuardedMachine)
	body, err := oracle.Generate(path)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "machines", "Other.oracle.md"), body)
	g := NewGate("clause sibling owner")
	collectClauseDecls(g, design)
	if !hasErr(g, "another machine cannot supply it") {
		t.Fatalf("a declaration must not resolve through a sibling's oracle: %v", g.Errs)
	}
}

// Contract cells are English. A row that retires nothing but describes a
// RETIRED site, or names the CLAUSES declaration without writing a second
// group, declares exactly what its one group says. A non-guard unit may carry
// a clause vocabulary too, as it could in v0.6.11.
func TestClauseDeclarationRowAcceptsProseAroundIt(t *testing.T) {
	const header = "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n"
	rows := map[string]string{
		"english_retired_in_prose": "| `guardReady` | guard | s | CLAUSES{owner, open}. One negative per clause: a RETIRED site, a missing owner | - |\n",
		"bare_word_in_same_cell":   "| `guardReady` | guard | s | reconciled to the CLAUSES declaration here. CLAUSES{owner, open} | - |\n",
		"non_guard_unit_declares": "| `guardReady` | guard | s | CLAUSES{owner, open} | - |\n" +
			"| `resolveApplicable` | actor | s | CLAUSES{assets-pinned, version-pinned} | - |\n",
	}
	for name, row := range rows {
		t.Run(name, func(t *testing.T) {
			g := NewGate("clause row prose")
			decls := collectClauseDecls(g, clauseOwnerDesign(t, clauseGuardedMachine, header+row))
			if len(g.Errs) != 0 || len(g.Warns) != 0 {
				t.Fatalf("prose beside one declaration is not a malformed declaration: errors=%v warnings=%v", g.Errs, g.Warns)
			}
			if len(decls) == 0 || decls[0].guard != "guardReady" {
				t.Fatalf("the declaration must still be collected: %+v", decls)
			}
		})
	}
}

// Two declaration groups on one row are ambiguous and stay rejected, as does a
// RETIRED group the declaration does not carry.
func TestClauseAmbiguousRowStillFails(t *testing.T) {
	const header = "| name | kind | signature | contract | maps to |\n|---|---|---|---|---|\n"
	rows := map[string]string{
		"two_groups":       "| `guardReady` | guard | s | CLAUSES{owner, open} and formerly CLAUSES{} | - |\n",
		"detached_retired": "| `guardReady` | guard | s | CLAUSES{owner, open} and RETIRED{legacy} was dropped RETIRED{older} | - |\n",
	}
	for name, row := range rows {
		t.Run(name, func(t *testing.T) {
			g := NewGate("clause ambiguity")
			collectClauseDecls(g, clauseOwnerDesign(t, clauseGuardedMachine, header+row))
			if !hasErr(g, "malformed CLAUSES declaration") {
				t.Fatalf("an ambiguous row must stay rejected: %v", g.Errs)
			}
		})
	}
}
