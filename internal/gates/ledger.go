// Gl-ledger: session-ledger discipline plus house style. The skill specifies
// two ledger formats nothing checked: the STATE.md phase-exit self-review line
// (five verdict keys with a fixed grammar) and the dated DECISIONS.md entry.
// A malformed self-review line silently weakens the discipline it exists to
// record, so the line grammar is held here; the ledgers' CONTENT stays
// unjudged (they narrate history, exactly as Gd/Gc treat them). The same gate
// carries the house-style scan (no em dashes, no emojis in design artifacts):
// a typography rule stated twice in the skill and enforced by nobody. Style
// findings are warnings; the tracked corpus holds them at zero.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// selfReviewKeyRe matches one verdict segment of a self-review line. The
	// grammar (skill, "Phase-exit self-review"): clean, fixed, fixed(<reason>),
	// accepted(<reason>). clean never carries a reason (a clean verdict with an
	// explanation is a contradiction, reported below).
	selfReviewKeyRe = regexp.MustCompile(`^(reality|depth|scope|coverage|consistency)=(clean|fixed|accepted)(\(([^()]*)\))?`)
	selfReviewKeys  = []string{"reality", "depth", "scope", "coverage", "consistency"}
	// decisionDateRe matches the dated-entry opener of a DECISIONS.md line:
	// an optional bullet, then YYYY-MM-DD.
	decisionDateRe = regexp.MustCompile(`^\s*[-*]?\s*(\d{4}-\d{2}-\d{2})\b`)
)

// LedgerActive reports whether the design carries either session ledger. The
// house-style scan runs regardless; this only gates the explicit-request
// error semantics nothing here needs, so Gl activates unconditionally in the
// default suite.
func LedgerActive(design string) bool {
	for _, name := range []string{"STATE.md", "DECISIONS.md"} {
		if fi, err := os.Stat(filepath.Join(design, name)); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
}

// CheckLedger implements Gl-ledger.
func CheckLedger(design string) *Gate {
	g := NewGate("Gl-ledger  session ledgers + house style")
	g.startOrder()
	checkSelfReviewLines(g, design)
	checkDecisionEntries(g, design)
	checkHouseStyle(g, design)
	return g
}

// checkSelfReviewLines holds every `self-review:` line in STATE.md to the
// five-key grammar. Absence of the file, or of any such line, is not a
// finding: the ledger is required by process, not by this gate, and judging
// which phases OWE a line would need a phase parser no free-form ledger can
// satisfy without false positives.
func checkSelfReviewLines(g *Gate, design string) {
	body, ok := readTextOK(filepath.Join(design, "STATE.md"))
	if !ok {
		return
	}
	for lineNo, line := range strings.Split(body, "\n") {
		_, after, found := strings.Cut(line, "self-review:")
		if !found {
			continue
		}
		g.Count("self-review lines")
		loc := "STATE.md:" + strconv.Itoa(lineNo+1)
		rest := strings.TrimSpace(after)
		// the line often sits inside an inline code span; the closing backtick
		// (and any trailing prose punctuation) is not part of the grammar
		rest = strings.TrimRight(rest, "` \t.")
		seen := map[string]bool{}
		bad := false
		for rest != "" {
			m := selfReviewKeyRe.FindStringSubmatch(rest)
			if m == nil {
				g.Errs = append(g.Errs, loc+": self-review segment "+strconv.Quote(firstWord(rest))+
					" does not parse; the grammar is key=clean|fixed|fixed(<reason>)|accepted(<reason>) with keys reality, depth, scope, coverage, consistency")
				bad = true
				break
			}
			key, verdict, parens, reason := m[1], m[2], m[3], m[4]
			if seen[key] {
				g.Errs = append(g.Errs, loc+": self-review states "+key+" twice")
			}
			seen[key] = true
			switch {
			case verdict == "clean" && parens != "":
				g.Errs = append(g.Errs, loc+": self-review "+key+"=clean carries a reason; clean means the pass found nothing (use fixed(<reason>) or accepted(<reason>))")
			case verdict == "accepted" && strings.TrimSpace(reason) == "":
				g.Errs = append(g.Errs, loc+": self-review "+key+"=accepted names no reason; an unexplained waiver is not a verdict")
			}
			rest = strings.TrimLeft(rest[len(m[0]):], " \t")
		}
		if bad {
			continue
		}
		var missing []string
		for _, k := range selfReviewKeys {
			if !seen[k] {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			g.Errs = append(g.Errs, loc+": self-review is missing "+strings.Join(missing, ", ")+"; all five verdicts are stated on one line")
		}
	}
}

// checkDecisionEntries validates the dated openers of DECISIONS.md entries
// (`<date> <who>: <decision>`, per the skill's operating discipline). Only the
// date is mechanically checkable: entry prose is the ledger's own. A heading
// carrying "author-proposed" marks the unconfirmed section; its item count is
// a coverage fact (a note), never a finding, because the items may legally
// belong to a phase still in flight.
func checkDecisionEntries(g *Gate, design string) {
	body, ok := readTextOK(filepath.Join(design, "DECISIONS.md"))
	if !ok {
		return
	}
	lines := strings.Split(body, "\n")
	unconfirmed := 0
	inProposed := false
	for lineNo, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			inProposed = strings.Contains(strings.ToLower(line), "author-proposed")
			continue
		}
		if inProposed {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
				unconfirmed++
			}
		}
		if m := decisionDateRe.FindStringSubmatch(line); m != nil {
			g.Count("dated decision entries")
			if _, err := time.Parse("2006-01-02", m[1]); err != nil {
				g.Errs = append(g.Errs, "DECISIONS.md:"+strconv.Itoa(lineNo+1)+": "+m[1]+" is not a real calendar date")
			}
		}
	}
	if unconfirmed > 0 {
		g.Notes = append(g.Notes, fmt.Sprintf("DECISIONS.md: %d author-proposed, unconfirmed item(s); confirm them (or convert each to a dated open decision with an owner) before recording the phase gate as passed", unconfirmed))
	}
}

// emojiRune reports whether r sits in an emoji block. Deliberately narrow:
// the house rule permits Unicode (check marks, arrows, box drawing), so only
// the emoji-proper planes are flagged and the warning stays holdable at zero.
func emojiRune(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1F6FF: // symbols, pictographs, emoticons, transport
		return true
	case r >= 0x1F900 && r <= 0x1F9FF: // supplemental symbols and pictographs
		return true
	case r >= 0x1FA70 && r <= 0x1FAFF: // symbols and pictographs extended-A
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF: // regional indicators (flags)
		return true
	}
	return false
}

// checkHouseStyle warns on em dashes (U+2014) and emojis in the hand-written
// design surface, ledgers included: the drift exemption for ledgers is about
// judging historical content, and typography is not content. Generated
// artifacts are skipped exactly as Gd skips them; the committed modelith
// render is scanned deliberately, because stripping its em dashes is the
// post-processing step this warning exists to catch.
func checkHouseStyle(g *Gate, design string) {
	type finding struct {
		rel  string
		line int
		msg  string
	}
	var findings []finding
	_ = filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // keep walking; the audit covers what is readable
		}
		if fi.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if idciteSkips(rel) || !idciteScannable(fi.Name()) {
			return nil
		}
		body, ok := readTextOK(path)
		if !ok {
			return nil
		}
		g.Count("files style-scanned")
		for lineNo, line := range strings.Split(body, "\n") {
			if strings.ContainsRune(line, '\u2014') {
				findings = append(findings, finding{rel, lineNo + 1, "em dash (U+2014); house style forbids it (use a hyphen, colon, or parentheses; for a fresh modelith render, strip with perl -CSD -i -pe 's/\\x{2014}/-/g')"})
			}
			for _, r := range line {
				if emojiRune(r) {
					findings = append(findings, finding{rel, lineNo + 1, fmt.Sprintf("emoji %q; house style forbids emojis in design artifacts (plain Unicode symbols are fine)", r)})
					break
				}
			}
		}
		return nil
	})
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].rel != findings[j].rel {
			return findings[i].rel < findings[j].rel
		}
		return findings[i].line < findings[j].line
	})
	for _, f := range findings {
		g.Warns = append(g.Warns, f.rel+":"+strconv.Itoa(f.line)+": "+f.msg)
	}
}

// firstWord returns the first whitespace-delimited token of s, for error
// messages that quote the offending segment without dumping the line.
func firstWord(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}
