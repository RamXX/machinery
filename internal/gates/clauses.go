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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var clauseDecl = regexp.MustCompile(`CLAUSES\{([^}]*)\}(?:\s*RETIRED\{([^}]*)\})?`)

type clauseSet struct {
	guard   string
	active  []string
	retired []string
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

// collectClauseDecls finds CLAUSES declarations on matrix guard rows.
func collectClauseDecls(design string) []clauseSet {
	var out []clauseSet
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(body), "\n") {
			m := clauseDecl.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			cells := strings.Split(line, "|")
			if len(cells) < 3 {
				continue
			}
			name := strings.Trim(strings.TrimSpace(cells[1]), "`")
			if name == "" {
				continue
			}
			out = append(out, clauseSet{guard: name, active: splitClauses(m[1]), retired: splitClauses(m[2])})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].guard < out[j].guard })
	return out
}

// checkClauseDrift scans every hand-written design file for lines naming a
// declared guard and holds them to its clause vocabulary.
func checkClauseDrift(g *Gate, design string) {
	decls := collectClauseDecls(design)
	if len(decls) == 0 {
		return
	}
	g.Count("guards with clause declarations", len(decls))
	_ = filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if idciteSkips(rel) || !idciteScannable(fi.Name()) {
			return nil
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for lineNo, line := range strings.Split(string(body), "\n") {
			for _, d := range decls {
				if !tokenIn(d.guard, line) || clauseDecl.MatchString(line) {
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
				loc := rel + ":" + strconv.Itoa(lineNo+1)
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
}
