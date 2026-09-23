// Guard clause-list reconciliation (S17 of the dogfood systemic findings). The
// round-9 BLOCKER was sibling-artifact semantic drift: a recorded guard
// narrowing reached the matrix, the machine comment, the shard narrative and
// the decision ledger, and missed the shard's falsifying-test table and the
// machine's _refusal value; every name resolved, no count drifted, and the
// artifacts were individually well-formed and collectively contradictory.
// The mechanism: a guard's matrix row may declare its clause vocabulary,
//
//	CLAUSES{resolved-task, applied-record} RETIRED{sop-coverage}
//
// and every hand-written line that names the guard is then held to it: a
// line enumerating SOME active clauses but not all is partial enumeration
// (the drift shape), and a line naming a RETIRED clause beside the guard is
// the stale semantics surviving. Proportionality (conductor qualification):
// declarations are opt-in, wanted only where a refusal or falsifying table
// enumerates clauses; an undeclared guard is untouched. Warnings tier.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var clauseDecl = regexp.MustCompile(`CLAUSES\{([^}]*)\}(?:\s*RETIRED\{([^}]*)\})?`)

type clauseSet struct {
	owner   string
	guard   string
	active  []string
	retired []string
	rows    []guardedOracleRow
}

func splitClauses(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// siblingOracleGuards indexes the guard cells of every committed oracle once,
// and answers which OTHER machine's oracle a guard governs. The answer decides
// whether an owner-local declaration with no rows is a silent zero-obligation
// case (no oracle governs the guard at all) or the ownership defect (a sibling
// would have supplied the rows). Stems are visited in sorted order, so the
// machine a diagnostic names is deterministic.
func siblingOracleGuards(design string) func(owner, guard string) string {
	type machineGuards struct {
		stem  string
		cells []string
	}
	var index []machineGuards
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.oracle.md") {
		entry := machineGuards{stem: strings.TrimSuffix(filepath.Base(path), ".oracle.md")}
		for _, row := range oracleGuardRows(readDesignOrEmpty(design, path)) {
			if row.guard != "" {
				entry.cells = append(entry.cells, row.guard)
			}
		}
		if len(entry.cells) > 0 {
			index = append(index, entry)
		}
	}
	return func(owner, guard string) string {
		for _, entry := range index {
			if entry.stem == owner {
				continue
			}
			for _, cell := range entry.cells {
				if tokenIn(guard, cell) {
					return entry.stem
				}
			}
		}
		return ""
	}
}

// collectClauseDecls finds and validates CLAUSES declarations on matrix guard
// rows. The matrix filename owns the declaration: Alpha.matrix.md can bind
// only Alpha.machine.json and Alpha.oracle.md, even when another machine uses
// the same guard name. Gt and Gd share this validation so missing ownership
// never silently removes an obligation from either gate.
func collectClauseDecls(g *Gate, design string) []clauseSet {
	var out []clauseSet
	seen := map[[2]string]bool{}
	siblings := siblingOracleGuards(design)
	contractOnly := contractOnlyMatrices(design)
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "clause matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable clause matrix: "+err.Error())
			continue
		}
		owner := strings.TrimSuffix(filepath.Base(path), ".matrix.md")
		for lineNo, line := range strings.Split(string(body), "\n") {
			// A declaration is a GROUP, CLAUSES{...}, on a row of the
			// named-unit contract table. Both halves matter, because a matrix
			// also NARRATES its units: a contract cell says "reconciled to the
			// CLAUSES declaration here", an invariant-coverage bullet quotes a
			// guard's vocabulary, and a long enumeration wraps across lines.
			// Prose is not a declaration; judging it as one reports a sentence
			// fragment as the unit's name and holds the author to a row never
			// written. A line carrying fewer than two pipes is not a row.
			if !strings.Contains(line, "CLAUSES{") && !strings.Contains(line, "RETIRED{") {
				continue
			}
			if strings.Count(line, "|") < 2 {
				continue
			}
			cells := splitTableRow(line)
			name := strings.Trim(strings.TrimSpace(cellAt(cells, 0)), "`")
			loc := fmt.Sprintf("%s:%d: machine %q guard %q", filepath.Base(path), lineNo+1, owner, clipText(name))
			m := clauseDecl.FindStringSubmatch(line)
			switch {
			case len(cells) < 2 || name == "":
				g.Errs = append(g.Errs, loc+": CLAUSES declaration is not a named row of the named-unit contract table; declare the clauses on the unit's own row (| `name` | kind | ... CLAUSES{...} |)")
				continue
			case m == nil || strings.Count(line, "CLAUSES{") != 1 ||
				strings.Contains(clauseDecl.ReplaceAllString(line, ""), "RETIRED{"):
				g.Errs = append(g.Errs, loc+": malformed CLAUSES declaration; require one named row with a single CLAUSES{...} and optional RETIRED{...}")
				continue
			}
			d := clauseSet{owner: owner, guard: name, active: splitClauses(m[1]), retired: splitClauses(m[2])}
			key := [2]string{owner, name}
			if seen[key] {
				g.Errs = append(g.Errs, loc+": duplicate or conflicting CLAUSES declaration for this machine and guard")
				continue
			}
			seen[key] = true
			if len(d.active) == 0 {
				g.Errs = append(g.Errs, loc+": declares CLAUSES{} with no clauses; list the falsifying clauses or drop the declaration")
			}
			if len(d.active) > 26 {
				g.Errs = append(g.Errs, loc+": declares more than 26 clauses; the stable-id suffix scheme supports only a-z (split the guard)")
			}
			vocabulary := map[string]bool{}
			for _, clause := range append(append([]string{}, d.active...), d.retired...) {
				if vocabulary[clause] {
					g.Errs = append(g.Errs, loc+": duplicate or active/retired conflicting clause "+clause)
				}
				vocabulary[clause] = true
			}
			if strings.Contains(m[1], ",") && len(strings.Split(m[1], ",")) != len(d.active) ||
				strings.Contains(m[2], ",") && len(strings.Split(m[2], ",")) != len(d.retired) {
				g.Errs = append(g.Errs, loc+": malformed clause list contains an empty member")
			}
			switch {
			case owner == "":
				g.Errs = append(g.Errs, loc+": empty matrix owner; name the matrix after its machine")
			case contractOnly[owner]:
				// A contract-only record (its placement row waives the
				// machine): the clause set is the record's contract and
				// governs no transition, so it binds no oracle row and owes
				// no suffixed transition id. Its obligation is the
				// assurance inventory's guard-clause key, one per active
				// clause (owner = the matrix, id = guard:clause). A sibling
				// oracle governing the guard is still the ownership defect.
				if other := siblings(owner, name); other != "" {
					g.Errs = append(g.Errs, loc+": contract-only matrix declares clauses for a guard "+other+"'s oracle governs; another machine cannot supply it")
				}
			default:
				if _, err := readDesignFile(design, filepath.Join(design, "machines", owner+".machine.json")); err != nil {
					g.Errs = append(g.Errs, loc+": missing or unreadable owning machine: "+err.Error())
				}
				oracle, err := readDesignFile(design, filepath.Join(design, "machines", owner+".oracle.md"))
				if err != nil {
					g.Errs = append(g.Errs, loc+": missing or unreadable owning oracle: "+err.Error())
				} else {
					for _, row := range oracleGuardRows(string(oracle)) {
						if tokenIn(name, row.guard) {
							d.rows = append(d.rows, row)
						}
					}
					// A guard NO oracle governs owes nothing and is not a
					// finding: v0.6.11 resolved rows across every machine, so
					// an insert-only machine could declare the clauses of a
					// guard that sits on its creation edge rather than on a
					// guarded transition. What must never resolve is a
					// declaration a SIBLING's oracle would have supplied,
					// which is the ownership defect this validation closed.
					if other := siblings(owner, name); len(d.rows) == 0 && other != "" {
						g.Errs = append(g.Errs, loc+": owning oracle has no transition governed by this guard, but "+other+"'s does; another machine cannot supply it")
					}
				}
			}
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].owner != out[j].owner {
			return out[i].owner < out[j].owner
		}
		return out[i].guard < out[j].guard
	})
	return out
}

// checkClauseDrift binds machine artifacts by filename. Shared narrative can
// name an owner explicitly; an unqualified guard is resolved only when its
// owner is unambiguous, including machines that have no CLAUSES declaration.
// Ambiguous clause enumerations warn; a guard mention with no clause tokens
// remains unjudged, just as it does for a uniquely owned guard.
// Narrative resolution cannot change the owner-local test obligations.
func checkClauseDrift(g *Gate, design string) {
	decls := collectClauseDecls(g, design)
	if len(decls) == 0 {
		return
	}
	g.Count("guards with clause declarations", len(decls))
	owners := map[string]map[string]bool{}
	for _, d := range decls {
		if owners[d.guard] == nil {
			owners[d.guard] = map[string]bool{}
		}
		owners[d.guard][d.owner] = true
	}
	oraclePaths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.oracle.md", "clause oracle")
	for _, path := range oraclePaths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, "clause owner scan: "+err.Error())
			continue
		}
		owner := strings.TrimSuffix(filepath.Base(path), ".oracle.md")
		for _, row := range oracleGuardRows(string(body)) {
			for guard, set := range owners {
				if tokenIn(guard, row.guard) {
					set[owner] = true
				}
			}
		}
	}
	err := walkTreeBounded(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if ignoredHere(design, rel) {
			if fi.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if idciteSkips(rel) || !idciteScannable(fi.Name()) {
			return nil
		}
		// DECISIONS.md and STATE.md narrate history: an entry describing the
		// pre-re-cut world beside a guard's name is the record, not drift,
		// exactly as with stable-id citations.
		if base := filepath.Base(rel); base == "DECISIONS.md" || base == "STATE.md" {
			return nil
		}
		body, readErr := readDesignFile(design, path)
		if readErr != nil {
			return readErr
		}
		for lineNo, line := range strings.Split(string(body), "\n") {
			ambiguous := map[string]bool{}
			for _, d := range decls {
				if !tokenIn(d.guard, line) || clauseDecl.MatchString(line) {
					continue
				}
				artifactOwner := ""
				if filepath.Dir(rel) == "machines" {
					for _, suffix := range []string{".matrix.md", ".machine.json", ".oracle.md"} {
						if strings.HasSuffix(rel, suffix) {
							artifactOwner = strings.TrimSuffix(filepath.Base(rel), suffix)
						}
					}
				}
				if artifactOwner != "" && artifactOwner != d.owner {
					continue
				}
				if artifactOwner == "" && len(owners[d.guard]) > 1 && !tokenIn(d.owner, line) {
					var names []string
					named := false
					enumerates := false
					for _, clause := range append(append([]string{}, d.active...), d.retired...) {
						enumerates = enumerates || tokenIn(clause, line)
					}
					for owner := range owners[d.guard] {
						names = append(names, owner)
						named = named || tokenIn(owner, line)
					}
					if !named && enumerates && !ambiguous[d.guard] {
						sort.Strings(names)
						g.Warns = append(g.Warns, fmt.Sprintf("%s:%d: guard %s has ambiguous machine ownership (%s); name its owner to check narrative clause drift", rel, lineNo+1, d.guard, strings.Join(names, ", ")))
						ambiguous[d.guard] = true
					}
					continue
				}
				var present, missing, stale []string
				for _, c := range d.active {
					if tokenIn(c, line) {
						present = append(present, c)
					} else {
						missing = append(missing, c)
					}
				}
				for _, c := range d.retired {
					if tokenIn(c, line) {
						stale = append(stale, c)
					}
				}
				loc := rel + ":" + strconv.Itoa(lineNo+1) + ": machine " + d.owner
				if len(stale) > 0 {
					g.Warns = append(g.Warns, loc+": names "+d.guard+" beside its RETIRED clause "+strings.Join(stale, ", ")+"; the old semantics survive here after the guard's re-cut")
				}
				if len(present) > 0 && len(missing) > 0 {
					g.Warns = append(g.Warns, loc+": enumerates "+strconv.Itoa(len(present))+" of "+d.guard+"'s "+strconv.Itoa(len(d.active))+" clauses ("+strings.Join(missing, ", ")+" missing); a partial enumeration beside the guard is the sibling-drift shape, enumerate all or none")
				}
			}
		}
		return nil
	})
	if err != nil {
		g.Errs = append(g.Errs, "clause drift scan failed: "+err.Error())
	}
}
