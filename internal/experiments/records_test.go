package experiments

// Contract-only records (consistency layer Stage 5, NEXT.md entry 13). An
// immutable, append-only record has no lifecycle, so it has no machine; its
// named-unit matrix is its contract, and its ARCHITECTURE.md placement row
// declares that with '(no machine: <reason>)'. G3, Gd, Gx and Gy all read
// that one waiver (gates.NoMachineWaivers): the declared record passes each
// of them, and a matrix with no machine and no waiver is still an orphan.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

func init() {
	RegisterRunner("records_test.go",
		"record-orphan-matrix", "record-orphan-matrix-rule", "record-waiver-empty-reason")
}

const recordModel = `  ErasureRecord:
    attributes:
      - name: subject
        type: string
    actions:
      - name: append
    invariants:
      - id: erasure-append-only
`

const recordMatrix = "# ErasureRecord - contract-only record\n\n" +
	"## (a) Named-unit contract table\n\n" +
	"| name | kind | signature | pre / post | maps to |\n" +
	"|---|---|---|---|---|\n" +
	"| `appendErasure` | action | `(input) -> row \\| err` | append once, never update. WRITES{ErasureRecord.subject} CARRIES{column:ErasureRecord.subject} | inv `erasure-append-only` |\n" +
	"| `guardErasable` | guard | `(input) -> bool` | the subject is known and its scope is closed. USES{ErasureRecord.subject} CLAUSES{subject-known, scope-closed} | inv `erasure-append-only` |\n"

const recordPlacementWaived = "| `ErasureRecord` (no machine: immutable append-only record; contract in machines/ErasureRecord.matrix.md) | in-process | db row, insert only | single writer |\n"

// contractOnlyRecord adds an ErasureRecord entity, its contract-only matrix,
// and its placement row to the widget fixture. placement is the row as
// written ("" adds no row, which leaves the entity unplaced).
func contractOnlyRecord(t *testing.T, design, placement string) {
	t.Helper()
	editFile(t, filepath.Join(design, "widget.modelith.yaml"), "      - id: widget-owned\n", "      - id: widget-owned\n"+recordModel)
	mustWrite(t, filepath.Join(design, "machines", "ErasureRecord.matrix.md"), recordMatrix)
	if placement != "" {
		editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "| single writer |\n", "| single writer |\n"+placement)
	}
}

// gateFindings returns a gate's ERROR and DRIFT lines.
func gateFindings(g *gates.Gate) []string {
	return append(append([]string(nil), g.Errs...), g.Drift...)
}

// The declared record passes every gate that reads a matrix, without a fake
// machine and without an oracle.
func TestContractOnlyRecordPassesTheGates(t *testing.T) {
	design, _ := fixture(t)
	contractOnlyRecord(t, design, recordPlacementWaived)
	for _, g := range []*gates.Gate{
		gates.CheckMachines(design), gates.CheckIDCitations(design),
		gates.CheckTraceability(design), gates.CheckRules(design, false),
	} {
		if f := gateFindings(g); len(f) != 0 {
			t.Errorf("%s: %v", g.Title, f)
		}
	}
	if n := gates.CheckMachines(design).Counts["contract-only matrices (no machine: waived)"]; n != 1 {
		t.Fatalf("G3 must count the waived record, got %d", n)
	}
	for _, g := range []*gates.Gate{gates.CheckIDCitations(design), gates.CheckTraceability(design), gates.CheckRules(design, false)} {
		if len(g.Warns) != 0 {
			t.Errorf("%s warns on the declared record: %v", g.Title, g.Warns)
		}
	}
}

// The waiver declares the record, not its references: a stale invariant
// reference or an unresolved fact in the contract-only matrix still fails.
func TestContractOnlyRecordReferencesStillResolve(t *testing.T) {
	design, _ := fixture(t)
	contractOnlyRecord(t, design, recordPlacementWaived)
	matrix := filepath.Join(design, "machines", "ErasureRecord.matrix.md")
	editFile(t, matrix, "never update. WRITES{ErasureRecord.subject} CARRIES{column:ErasureRecord.subject} | inv `erasure-append-only`",
		"never update. WRITES{ErasureRecord.subjekt} CARRIES{column:ErasureRecord.subject} | inv `erasure-appendonly`")
	if g := gates.CheckTraceability(design); !containsAny(g.Drift, "`erasure-appendonly`, which is not a declared invariant") {
		t.Fatalf("Gx must report the stale invariant reference: %v", g.Drift)
	}
	if got := rulesFindings(t, design); !containsAny(got, "row 'ErasureRecord.appendErasure': fact_unresolved (fact 'ErasureRecord.subjekt')") {
		t.Fatalf("Gy-rules must report the unresolved fact: %v", got)
	}
}

// A matrix with no machine and no waiver is a stale orphan: G3 and Gy-rules
// both report it, and Gx reports the unwaived placement row.
func TestUndeclaredOrphanMatrixStillFails(t *testing.T) {
	design, _ := fixture(t)
	contractOnlyRecord(t, design, "| `ErasureRecord` | in-process | db row, insert only | single writer |\n")
	e := experimentNamed(t, "record-orphan-matrix")
	if g := gates.CheckMachines(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped G3: %v", e.Name, g.Errs)
	}
	requireRuleFinding(t, "record-orphan-matrix-rule", design)
	if g := gates.CheckTraceability(design); !containsAny(g.Errs, "placement row component `ErasureRecord` has no machine") {
		t.Fatalf("Gx must report the unwaived placement row: %v", g.Errs)
	}
	// Gd still demands the owning machine and oracle of an undeclared matrix.
	if g := gates.CheckIDCitations(design); !containsAny(g.Errs, "missing or unreadable owning machine") {
		t.Fatalf("Gd must report the missing owner of an undeclared matrix: %v", g.Errs)
	}
}

// A waiver with an empty reason is no waiver: Gx reports the row, and the
// matrix stays an orphan for G3 and Gy-rules.
func TestContractOnlyWaiverWithEmptyReason(t *testing.T) {
	design, _ := fixture(t)
	contractOnlyRecord(t, design, "| `ErasureRecord` (no machine: ) | in-process | db row, insert only | single writer |\n")
	e := experimentNamed(t, "record-waiver-empty-reason")
	if g := gates.CheckTraceability(design); !containsAny(g.Errs, e.ExpectSubstr) {
		t.Fatalf("%s escaped Gx: %v", e.Name, g.Errs)
	}
	if g := gates.CheckMachines(design); !containsAny(g.Errs, "ErasureRecord.matrix.md: orphan matrix") {
		t.Fatalf("an empty-reason waiver must leave the matrix an orphan: %v", g.Errs)
	}
	requireRuleFinding(t, "record-orphan-matrix-rule", design)
}

// A '(no machine: ...)' placement beside a small machine of the same name is
// an accepted convention (an envelope machine for a record-only entity), which
// 0.9.0 accepted: Gy-rules raises nothing on it. The waiver still declares
// nothing there (the matrix has its machine), so G3 counts no contract-only
// matrix for it.
func TestWaiverBesideMachineIsAccepted(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "| `Widget` | in-process |", "| `Widget` (no machine: envelope for a record-only entity) | in-process |")
	if got := rulesFindings(t, design); len(got) != 0 {
		t.Fatalf("Gy-rules must accept a waiver beside a machine: %v", got)
	}
	if n := gates.CheckMachines(design).Counts["contract-only matrices (no machine: waived)"]; n != 0 {
		t.Fatalf("a waiver beside a machine declares no contract-only matrix, got %d", n)
	}
}

// A contract-only clause set governs no transition, so it binds no oracle row
// and owes no suffixed transition id (no phantom ids). Its coverage
// obligation is stated in the assurance inventory: one guard-clause key per
// active clause, owned by the matrix, id guard:clause.
func TestContractOnlyClausesOweGuardClauseKeysNotTransitionIDs(t *testing.T) {
	design, impl := fixture(t)
	contractOnlyRecord(t, design, recordPlacementWaived)
	inv, err := gates.AssuranceInventory(design)
	if err != nil {
		t.Fatal(err)
	}
	var clauses []string
	for _, o := range inv.Obligations {
		switch {
		case o.Key.Kind == protocol.KindOracleRow && strings.Contains(o.Key.Owner, "ErasureRecord"):
			t.Fatalf("a contract-only record owes no oracle row: %+v", o.Key)
		case o.Key.Kind == protocol.KindGuardClause && o.Key.Owner == "ErasureRecord":
			clauses = append(clauses, o.Key.ID)
		}
	}
	if strings.Join(clauses, ",") != "guardErasable:scope-closed,guardErasable:subject-known" {
		t.Fatalf("guard-clause obligations = %v", clauses)
	}
	g := gates.CheckOracleCoverage(design, impl)
	for _, e := range g.Errs {
		if strings.Contains(e, "guardErasable") {
			t.Fatalf("Gt must not demand transition ids of a contract-only clause set: %s", e)
		}
	}
}
