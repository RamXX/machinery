// Prose-count drift (S4 of the dogfood systemic findings, as amended). Rule 15
// (no hand-typed counts) is a house rule, not a gate, and rounds still found
// "53-row count" beside a 52-row table and "the seven runscope-* invariants"
// over eight. The catchable slice: a NUMBER attached to a countable noun in
// the few lines above a markdown table, compared against that table's data
// rows. WARNING tier with a historical exemption ("read 162, then 166"):
// heuristics that flag correct content teach readers to ignore them, so a
// dated history note is never flagged.

package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var spelledNumbers = map[string]int{
	"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6,
	"seven": 7, "eight": 8, "nine": 9, "ten": 10, "eleven": 11, "twelve": 12,
}

// countClaim matches the rule-15 form (b) shapes ONLY: "(N rows" in a
// header beside its table, or "N rows, counted off this table". A first
// calibration with a broad noun vocabulary near any table flagged prose
// about OTHER objects ten times on a clean design; a claim is checkable
// only when it binds itself to the table below.
var countClaim = regexp.MustCompile(`(?i)\((\d+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve) rows?\b|\b(\d+|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve) rows?, counted off this table`)

// historicalCountLine reports whether a line is a rule-15 history note
// ("this read 162, then 166, before landing here") rather than a claim
// about the table below.
func historicalCountLine(line string) bool {
	l := strings.ToLower(line)
	return strings.Contains(l, "read ") || strings.Contains(l, ", then ") ||
		strings.Contains(l, "went stale") || strings.Contains(l, "until ")
}

// checkProseCounts warns where a count claim within five lines above a
// markdown table disagrees with that table's data-row count.
func checkProseCounts(g *Gate, design string) {
	walk := func(path, rel string) error {
		body, err := readDesignFile(design, path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(body), "\n")
		isSep := func(i int) bool {
			if i >= len(lines) {
				return false
			}
			t := strings.TrimSpace(lines[i])
			if !strings.HasPrefix(t, "|") {
				return false
			}
			for _, ch := range t {
				if ch != '|' && ch != '-' && ch != ':' && ch != ' ' {
					return false
				}
			}
			return true
		}
		for i := 0; i < len(lines); i++ {
			// a table starts at a header line whose next line is a separator
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") || !isSep(i+1) {
				continue
			}
			rows := 0
			j := i + 2
			for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
				rows++
				j++
			}
			lo := i - 3
			if lo < 0 {
				lo = 0
			}
			for k := lo; k < i; k++ {
				line := lines[k]
				if strings.HasPrefix(strings.TrimSpace(line), "|") || historicalCountLine(line) {
					continue
				}
				for _, m := range countClaim.FindAllStringSubmatch(line, -1) {
					lit := m[1]
					if lit == "" {
						lit = m[2]
					}
					n, err := strconv.Atoi(lit)
					if err != nil {
						n = spelledNumbers[strings.ToLower(lit)]
					}
					if n == 0 {
						continue
					}
					g.Count("row-count claims beside tables")
					if n != rows {
						g.Warns = append(g.Warns, rel+":"+strconv.Itoa(k+1)+": claims "+lit+" rows but the table below has "+strconv.Itoa(rows)+" data rows; count off the table (rule 15) or fix the figure")
					}
				}
			}
			i = j - 1
		}
		return nil
	}
	err := filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
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
		if idciteSkips(rel) || strings.ToLower(filepath.Ext(fi.Name())) != ".md" {
			return nil
		}
		if base := filepath.Base(rel); base == "DECISIONS.md" || base == "STATE.md" {
			return nil // ledgers narrate history; their figures are of their time
		}
		return walk(path, rel)
	})
	if err != nil {
		g.Errs = append(g.Errs, "prose-count scan failed: "+err.Error())
	}
}
