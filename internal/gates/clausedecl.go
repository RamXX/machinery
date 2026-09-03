// Clause-declaration completeness (Gd, warnings tier).
//
// The clause-drift check holds a DECLARED vocabulary against every mention of
// its guard. It never asked the reverse question: a guard whose contract cell
// reads "true iff A and B" and declares no `CLAUSES{...}` at all passes in
// silence, so the drift shape the declaration exists to catch simply never
// arms. A dogfood hardening wave asserted "every compound guard declares its
// clauses" twice and was falsified twice by hand-judgment, because the claim
// had no mechanical carrier. This is the carrier.
//
// What is read is the guard's CONTRACT STATEMENT, not the whole cell: the
// sentence carrying `iff`, else the cell's first sentence. The rest of a
// pre/post cell is rationale prose, and the word "and" in a sentence
// explaining WHY a guard exists is narration, not a second clause; scanning
// it produced two false findings for every true one on the calibration
// corpus. A row may answer with `(single-clause: <reason>)` instead of a
// declaration, which is the same waiver grammar the rest of the suite uses.

package gates

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// compoundContractRe matches the conjunction and disjunction vocabulary of a
// guard contract, whole-token and case-insensitive. The surrounding groups
// keep the match whole-token under RE2's lack of lookaround, the same shape
// stableIDToken uses; "at-or-after" therefore does not match on "or".
var compoundContractRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_-])(and|or|any|either|all of|one of)($|[^A-Za-z0-9_-])`)

// singleClauseRe is the row-level waiver, with its reason captured.
var singleClauseRe = regexp.MustCompile(`\(single-clause:\s*([^)]*)\)`)

// iffRe locates the biconditional a contract statement is written around.
var iffRe = regexp.MustCompile(`(?i)(^|[^A-Za-z0-9_-])iff($|[^A-Za-z0-9_-])`)

// guardContractStatement returns the part of a pre/post cell that STATES the
// contract: the first sentence containing `iff`, else the first sentence.
// Sentences end at a period followed by a space or the cell's end, and a
// period inside a backticked code span never ends one (`v1.2` is not two
// sentences).
func guardContractStatement(cell string) string {
	var parts []string
	var cur strings.Builder
	inTick := false
	runes := []rune(cell)
	for i, ch := range runes {
		if ch == '`' {
			inTick = !inTick
		}
		cur.WriteRune(ch)
		if !inTick && ch == '.' && (i+1 >= len(runes) || runes[i+1] == ' ') {
			parts = append(parts, cur.String())
			cur.Reset()
		}
	}
	if strings.TrimSpace(cur.String()) != "" {
		parts = append(parts, cur.String())
	}
	if len(parts) == 0 {
		return cell
	}
	for _, p := range parts {
		if iffRe.MatchString(p) {
			return p
		}
	}
	return parts[0]
}

// namedUnitCols locates the name, kind, and pre/post columns of a named-unit
// contract table HEADER, returning ok=false for any other row. The name and
// kind labels must match exactly and the pre/post label by substring (a
// design writes either "pre / post" or "contract (pre / post)"); an exact
// label rule is what keeps a data row whose cells happen to contain the words
// from being read as a header.
func namedUnitCols(header []string) (name, kind, prepost int, ok bool) {
	name, kind, prepost = -1, -1, -1
	for i, h := range header {
		switch label := strings.ToLower(strings.TrimSpace(h)); {
		case label == "name" && name < 0:
			name = i
		case label == "kind" && kind < 0:
			kind = i
		case strings.Contains(label, "pre /") && prepost < 0:
			prepost = i
		}
	}
	return name, kind, prepost, name >= 0 && kind >= 0 && prepost >= 0
}

// checkClauseCompleteness warns for every guard row whose contract statement
// is a conjunction or a disjunction and which declares neither a clause
// vocabulary nor a single-clause waiver.
func checkClauseCompleteness(g *Gate, design string) {
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, ok := readTextOK(design, path)
		if !ok {
			continue
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		var cols []int
		for lineNo, line := range strings.Split(body, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				cols = nil // a table ended; the next one declares its own columns
				continue
			}
			cells := splitTableRow(line)
			if ni, ki, pi, isHdr := namedUnitCols(cells); isHdr {
				cols = []int{ni, ki, pi}
				continue
			}
			if cols == nil {
				continue
			}
			kind := strings.TrimSpace(cellAt(cells, cols[1]))
			if kind != "guard" {
				continue
			}
			name := strings.Trim(strings.TrimSpace(cellAt(cells, cols[0])), "`")
			if name == "" {
				continue
			}
			g.Count("guard rows read for clause completeness")
			if clauseDecl.MatchString(line) {
				g.Count("guard rows with a clause declaration")
				continue
			}
			stmt := guardContractStatement(cellAt(cells, cols[2]))
			m := compoundContractRe.FindStringSubmatch(stmt)
			if m == nil {
				continue
			}
			loc := rel + ":" + strconv.Itoa(lineNo+1)
			if w := singleClauseRe.FindStringSubmatch(line); w != nil {
				if strings.TrimSpace(w[1]) == "" {
					g.Warns = append(g.Warns, loc+": guard `"+name+"` waives its clause declaration but names no reason; write '(single-clause: <reason>)'")
					continue
				}
				g.Count("single-clause waivers")
				continue
			}
			g.Warns = append(g.Warns, loc+": guard `"+name+"` states a compound contract ("+ir.Repr(strings.ToLower(m[2]))+" in "+ir.Repr(clipText(stmt))+") and declares no CLAUSES{...}; declare the clause vocabulary so every enumeration of it is held to one list, or waive with '(single-clause: <reason>)'")
		}
	}
}

// clipText shortens a quoted excerpt for a finding: enough to recognize what
// is meant, never a whole paragraph on one warn line. Shared by every finding
// that quotes an author's cell back to them.
func clipText(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 90
	if len([]rune(s)) <= max {
		return s
	}
	return string([]rune(s)[:max]) + "..."
}

// splitTableRow splits a markdown table row into cells the way the rest of
// the matrix line-scanners do (split on the raw pipes, keeping the leading
// empty cell so column indices line up with the header's own split).
func splitTableRow(line string) []string {
	cells := strings.Split(strings.TrimSpace(line), "|")
	if len(cells) > 0 && strings.TrimSpace(cells[0]) == "" {
		cells = cells[1:]
	}
	if n := len(cells); n > 0 && strings.TrimSpace(cells[n-1]) == "" {
		cells = cells[:n-1]
	}
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}
