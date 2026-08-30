// Falsifying-clause coverage (Gt, opt-in via CLAUSES declarations). The
// build template requires one test per falsifying clause of each conjunction
// guard (a guard `A AND B AND C` needs one test with only A false, one with
// only B false, one with only C false), keyed as the base stable id plus a
// lowercase letter (the N3 suffix convention Gd already resolves design-side).
// Nothing held the SUITE to that rule: a guard could declare its clauses,
// every design artifact could cite them, and the tests could still cover none
// of them. Once a matrix row declares CLAUSES{...}, the obligation is closed:
// every committed oracle row that guard governs owes one suffixed id per
// active clause, in the test corpus, whole-token. Wholesale conformance
// parsing does not discharge it, because falsifying-clause tests are exactly
// what the oracle table cannot derive.

package gates

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// guardedOracleRow is one committed transition-oracle row with its guard
// cell.
type guardedOracleRow struct {
	stableID string
	guard    string
}

// oracleGuardRows parses a committed oracle's transition table into
// (stable id, guard) pairs; rows without a guard come back with "".
func oracleGuardRows(text string) []guardedOracleRow {
	var out []guardedOracleRow
	for _, tbl := range ir.ParseMdTables(text) {
		si := ir.FindCol(tbl.Header, "stable id")
		gi := ir.FindCol(tbl.Header, "guard")
		if si < 0 || gi < 0 {
			continue
		}
		for _, r := range tbl.Rows {
			if si >= len(r) {
				continue
			}
			id := strings.TrimSpace(r[si])
			if id == "" || id == "-" {
				continue
			}
			guard := ""
			if gi < len(r) {
				guard = strings.TrimSpace(r[gi])
			}
			out = append(out, guardedOracleRow{stableID: id, guard: guard})
		}
	}
	return out
}

// checkClauseCoverage holds the suite to the falsifying-clause obligation of
// every CLAUSES-declared guard: for each committed oracle row the guard
// governs, one suffixed stable id per active clause (base+a, base+b, ...)
// appears whole-token in some test file.
func checkClauseCoverage(g *Gate, design string, corpus testCorpusData) {
	decls := collectClauseDecls(design)
	if len(decls) == 0 {
		return
	}
	var rows []guardedOracleRow
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.oracle.md") {
		rows = append(rows, oracleGuardRows(readOrEmpty(path))...) // read errors reported by the coverage pass
	}
	for _, d := range decls {
		n := len(d.active)
		if n == 0 || n > 26 {
			continue // an empty or absurd declaration is its own review finding
		}
		g.Count("clause-declared guards checked")
		for _, row := range rows {
			if row.guard == "" || !tokenIn(d.guard, row.guard) {
				continue
			}
			var missing []string
			for i := range n {
				suffixed := row.stableID + string(rune('a'+i))
				if idTokenIn(suffixed, corpus.joined) {
					g.Count("falsifying-clause ids covered")
				} else {
					missing = append(missing, suffixed)
				}
			}
			if len(missing) > 0 {
				g.Errs = append(g.Errs, fmt.Sprintf("guard %s declares %d clause(s) (%s) but the suite misses falsifying test(s) %s for oracle row %s; one test per clause with only that clause false (the conformance parse cannot derive these)",
					d.guard, n, strings.Join(d.active, ", "), strings.Join(missing, ", "), row.stableID))
			}
		}
	}
}
