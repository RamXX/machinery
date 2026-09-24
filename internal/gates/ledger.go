// Gl-ledger: session-ledger discipline plus house style. The skill specifies
// two ledger formats nothing checked: the STATE.md phase-exit self-review line
// (five verdict keys with a fixed grammar) and the dated DECISIONS.md entry.
// A malformed self-review line silently weakens the discipline it exists to
// record, so the line grammar is held here; the ledgers' CONTENT stays
// unjudged (they narrate history, exactly as Gd/Gc treat them). The same gate
// carries the house-style scan (no em dashes, no emojis in design artifacts):
// a typography rule stated twice in the skill and enforced by nobody. Style
// findings are warnings; the tracked corpus holds them at zero. The one
// exception is an em dash in a `*.modelith.md` render: the renderer emits them
// and the post-processing strip is a known, mechanical obligation, so that
// case is an ERROR (the gate now owns the obligation the skill used to spell
// out as a perl one-liner).

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

	"github.com/RamXX/machinery/internal/ir"
)

var (
	// selfReviewKeyRe matches the key=verdict head of one segment of a
	// self-review line. The grammar (skill, "Phase-exit self-review"): clean,
	// fixed, fixed(<reason>), accepted(<reason>). clean never carries a reason
	// (a clean verdict with an explanation is a contradiction, reported below).
	// The optional (<reason>) group is scanned separately with balanced-paren
	// tracking, because real reasons routinely nest parentheses.
	selfReviewKeyRe = regexp.MustCompile(`^(reality|depth|scope|coverage|consistency)=(clean|fixed|accepted)`)
	selfReviewKeys  = []string{"reality", "depth", "scope", "coverage", "consistency"}
	// decisionDateRe matches the dated-entry opener of a DECISIONS.md line:
	// an optional bullet, then YYYY-MM-DD.
	decisionDateRe = regexp.MustCompile(`^\s*[-*]?\s*(\d{4}-\d{2}-\d{2})\b`)
	// htmlCommentRe and machineryMarkerishRe find marker-shaped comments: an
	// HTML comment invoking the machinery marker namespace. A comment that is
	// marker-shaped but matches no known marker grammar arms nothing, silently,
	// which is the failure mode the warn below exists to surface.
	htmlCommentRe        = regexp.MustCompile(`<!--.*?-->`)
	machineryMarkerishRe = regexp.MustCompile(`machinery\s*:`)
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
	checkDuplicateTables(g, design)
	checkUndeclaredFactReferences(g, design)
	checkValuesEnumCase(g, design)
	return g
}

// checkDuplicateTables warns when the same table (whitespace-collapsed) sits
// in two hand-written documents with no machinery:embed marker sanctioning
// the copy. The doctrine ("a copied table is a promise until it is marked")
// left remembering the marker to the author; this makes the forgotten case
// visible. Warn tier: the copy may be mid-edit, but the tracked corpus holds
// warnings at zero. Ledgers are exempt (they narrate history), generated
// artifacts are skipped as Gd skips them, and fenced examples are masked.
func checkDuplicateTables(g *Gate, design string) {
	type occ struct {
		where  string
		marked bool
		gen    bool // the table sits in a generated *.modelith.md render
	}
	seen := map[string][]occ{}
	var order []string
	files, err := markdownFiles(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return
	}
	for _, path := range files {
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		base := filepath.Base(rel)
		if idciteSkips(rel) || base == "STATE.md" || base == "DECISIONS.md" {
			continue
		}
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, rel+": unreadable: "+err.Error())
			continue
		}
		lines := strings.Split(maskFences(string(body)), "\n")
		start := -1
		flush := func(end int) {
			if start < 0 {
				return
			}
			s := start
			start = -1
			if end-s < 3 { // header, separator, at least one data row
				return
			}
			var norm []string
			for _, l := range lines[s:end] {
				norm = append(norm, collapseWS(l))
			}
			marked := false
			for j := s - 1; j >= 0 && j >= s-3; j-- {
				t := strings.TrimSpace(lines[j])
				if embedMarker.MatchString(t) {
					marked = true
					break
				}
				if t != "" && !strings.HasPrefix(t, "<!--") {
					break
				}
			}
			key := strings.Join(norm, "\n")
			if _, dup := seen[key]; !dup {
				order = append(order, key)
			}
			seen[key] = append(seen[key], occ{
				where:  rel + ":" + strconv.Itoa(s+1),
				marked: marked,
				gen:    strings.HasSuffix(strings.ToLower(base), ".modelith.md"),
			})
		}
		for i, l := range lines {
			if strings.HasPrefix(strings.TrimLeft(l, " \t"), "|") {
				if start < 0 {
					start = i
				}
			} else {
				flush(i)
			}
		}
		flush(len(lines))
	}
	for _, key := range order {
		occs := seen[key]
		if len(occs) < 2 {
			continue
		}
		marked := 0
		allGen := true
		var wheres []string
		for _, o := range occs {
			if o.marked {
				marked++
			}
			if !o.gen {
				allGen = false
			}
			wheres = append(wheres, o.where)
		}
		// one original plus marked copies is the sanctioned shape; anything
		// less marked means at least one copy nobody declared
		if marked >= len(occs)-1 {
			g.Count("duplicate tables embed-marked")
			continue
		}
		// a generated render is never hand-edited, so "mark it" is advice the
		// author cannot legally take: two entities render byte-identical
		// tables, and the fix lives in the modelith YAML source
		if allGen {
			g.Warns = append(g.Warns, "the same table appears at "+strings.Join(wheres, " and ")+
				"; both sit in a generated *.modelith.md render, so this is two entities rendering byte-identical tables, not a copy anyone made: differentiate them in the modelith YAML source (for example, give distinguishing attribute descriptions), then re-render (never hand-edit the render, and no machinery:embed marker belongs there)")
			continue
		}
		g.Warns = append(g.Warns, "the same table appears at "+strings.Join(wheres, " and ")+
			" with no machinery:embed marker on the copy; a copied table is a promise until it is marked (mark it, or let one side own the rows)")
	}
}

// checkSelfReviewLines holds every `self-review:` line in STATE.md to the
// five-key grammar. Absence of the file, or of any such line, is not a
// finding: the ledger is required by process, not by this gate, and judging
// which phases OWE a line would need a phase parser no free-form ledger can
// satisfy without false positives.
func checkSelfReviewLines(g *Gate, design string) {
	body, ok := readTextOK(design, filepath.Join(design, "STATE.md"))
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
		// the line often sits inside an inline code span, itself often inside a
		// markdown table cell; the closing backtick, the cell's trailing pipe,
		// and any trailing prose punctuation are not part of the grammar (a raw
		// pipe cannot appear inside a table cell anyway: it would split the cell)
		rest = strings.TrimRight(rest, "`| \t.")
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
			key, verdict := m[1], m[2]
			consumed := len(m[0])
			hasReason := false
			reason := ""
			if consumed < len(rest) && rest[consumed] == '(' {
				interior, n, balanced := balancedParen(rest[consumed:])
				if !balanced {
					g.Errs = append(g.Errs, loc+": self-review segment "+strconv.Quote(firstWord(rest))+
						" does not parse (its reason's parentheses never balance); the grammar is key=clean|fixed|fixed(<reason>)|accepted(<reason>)")
					bad = true
					break
				}
				hasReason = true
				reason = interior
				consumed += n
			}
			if seen[key] {
				g.Errs = append(g.Errs, loc+": self-review states "+key+" twice")
			}
			seen[key] = true
			switch {
			case verdict == "clean" && hasReason:
				g.Errs = append(g.Errs, loc+": self-review "+key+"=clean carries a reason; clean means the pass found nothing (use fixed(<reason>) or accepted(<reason>))")
			case verdict == "accepted" && strings.TrimSpace(reason) == "":
				g.Errs = append(g.Errs, loc+": self-review "+key+"=accepted names no reason; an unexplained waiver is not a verdict")
			}
			rest = strings.TrimLeft(rest[consumed:], " \t")
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
	body, ok := readTextOK(design, filepath.Join(design, "DECISIONS.md"))
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
// post-processing step this check exists to catch. There the finding is an
// ERROR, not a warning: the render is generated from a known renderer with a
// known mechanical fix, so a surviving em dash is a skipped step rather than a
// style opinion. Every other file, and every emoji anywhere, stays at the warn
// tier.
func checkHouseStyle(g *Gate, design string) {
	type finding struct {
		rel   string
		line  int
		msg   string
		isErr bool
	}
	var findings []finding
	ignored := 0
	walkErr := walkTreeBounded(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if fi.IsDir() {
			return nil
		}
		if idciteSkips(rel) || !idciteScannable(fi.Name()) {
			return nil
		}
		// Gl walks INTO an ignored subtree instead of pruning it, so the
		// count it reports is files, not directories: "3 paths ignored" for a
		// vendored tree of 300 documents would understate what was skipped,
		// and the number exists to make the ignoring visible.
		if ignoredHere(design, rel) {
			ignored++
			return nil
		}
		body, ok := readTextOK(design, path)
		if !ok {
			g.Errs = append(g.Errs, "house-style scan incomplete: "+rel+" is unreadable")
			return nil
		}
		g.Count("files style-scanned")
		render := isModelithRender(fi.Name())
		for lineNo, line := range strings.Split(body, "\n") {
			if strings.ContainsRune(line, '\u2014') {
				if render {
					findings = append(findings, finding{rel, lineNo + 1, "em dash (U+2014) in a generated modelith render; the post-render strip was skipped: perl -CSD -i -pe 's/\\x{2014}/-/g' " + filepath.ToSlash(rel), true})
				} else {
					findings = append(findings, finding{rel, lineNo + 1, "em dash (U+2014); house style forbids it (use a hyphen, colon, or parentheses)", false})
				}
			}
			for _, r := range line {
				if emojiRune(r) {
					findings = append(findings, finding{rel, lineNo + 1, fmt.Sprintf("emoji %q; house style forbids emojis in design artifacts (plain Unicode symbols are fine)", r), false})
					break
				}
			}
			// near-miss opt-in markers: a marker-shaped comment that parses as
			// no known marker grammar arms nothing, and the tier it meant to
			// arm silently never runs
			for _, c := range htmlCommentRe.FindAllString(line, -1) {
				if !machineryMarkerishRe.MatchString(c) {
					continue
				}
				if embedMarker.MatchString(c) || readsCompleteMarker.MatchString(c) || authorizationMarker.MatchString(c) {
					continue
				}
				findings = append(findings, finding{rel, lineNo + 1, "marker-shaped comment " + strconv.Quote(c) + " matches no known machinery marker grammar (machinery:embed, machinery:reads-complete, machinery:authorization-inventory); a near-miss marker arms nothing", false})
			}
		}
		return nil
	})
	if walkErr != nil {
		g.Errs = append(g.Errs, "house-style scan incomplete: "+walkErr.Error())
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].rel != findings[j].rel {
			return findings[i].rel < findings[j].rel
		}
		return findings[i].line < findings[j].line
	})
	// the ignore count is emitted verbatim, zero included, whenever the design
	// carries the file: a reader must be able to see that paths are being
	// ignored, and that a list has stopped matching anything.
	if designIgnoreFor(design).present {
		g.CheckedExtra(strconv.Itoa(ignored) + " paths ignored (" + IgnoreFileName + ")")
	}
	for _, f := range findings {
		text := f.rel + ":" + strconv.Itoa(f.line) + ": " + f.msg
		if f.isErr {
			g.Errs = append(g.Errs, text)
		} else {
			g.Warns = append(g.Warns, text)
		}
	}
}

// isModelithRender reports whether a file name is a generated modelith render
// (`<name>.modelith.md`). The render is the one hand-committed file in the
// design whose em dashes come from a generator rather than an author, which is
// what earns it the error tier above.
func isModelithRender(base string) bool {
	return strings.HasSuffix(strings.ToLower(base), ".modelith.md")
}

// balancedParen scans a string beginning with '(' to its balanced closing
// paren. It returns the interior (the reason text, nested parens intact), the
// byte length of the whole group including both delimiters, and whether a
// balanced close was found at all.
func balancedParen(s string) (interior string, length int, balanced bool) {
	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], i + 1, true
			}
		}
	}
	return "", 0, false
}

// firstWord returns the first whitespace-delimited token of s, for error
// messages that quote the offending segment without dumping the line.
func firstWord(s string) string {
	if i := strings.IndexAny(s, " \t"); i >= 0 {
		return s[:i]
	}
	return s
}

// The undeclared-fact-reference tier (consistency layer, Stage 1). A
// backticked token shaped like a stored fact, snake_case or Entity.attr, in a
// matrix row's contract, clause, or payload cell, and outside every
// declaration group, is prose quoting a fact nobody declared. It is a WARNING
// and never resolves anything: prose never declares. The author either
// declares the fact (USES{} for what the unit reads or names, WRITES{} for
// what it stores) or drops the backticks.
//
// Not a reference, by construction rather than by guess: a token inside any
// group (CLAUSES, READS, VALUES, ORACLESET, WRITES, USES, CARRIES, payload,
// and an unknown group, which Gx already reports), a token inside a
// `derived: x (reason)` form, the row's own unit name, a fact the same row
// declares (see rowDeclaredTokens), and a token naming a machine, a state, or
// a Machine.state pair. Fenced code is never a row.
var (
	snakeFactToken      = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)+$`)
	entityAttrFactToken = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*\.[a-z][A-Za-z0-9]*$`)
	factBacktickSpan    = regexp.MustCompile("`([^`]*)`")
	derivedFormRe       = regexp.MustCompile("(?i)\\bderived:\\s*`?[A-Za-z][A-Za-z0-9_.]*`?\\s*\\([^)]*\\)")
)

// factReferenceCells returns the contract, clause, and payload columns of a
// matrix table header: the pre/post column of a named-unit table, and any
// column whose label names a payload or a clause.
func factReferenceCells(header []string) []int {
	var cols []int
	_, _, prepost, named := namedUnitCols(header)
	for i, h := range header {
		label := strings.ToLower(h)
		if (named && i == prepost) || strings.Contains(label, "payload") || strings.Contains(label, "clause") {
			cols = append(cols, i)
		}
	}
	return cols
}

// designNameTokens collects the tokens that name a machine or a state (bare,
// by path, and qualified by its machine). Oracle ids need no entry: a test id
// (`T-ORDE-01`) or stable id (`ORDE-eb2d3b`) carries a hyphen and upper-case
// letters, so neither fact-token shape can ever match one.
func designNameTokens(design string) map[string]bool {
	names := map[string]bool{}
	mdir := filepath.Join(design, "machines")
	for _, path := range sortedGlob(mdir, "*.machine.json") {
		stem := strings.TrimSuffix(filepath.Base(path), ".machine.json")
		machines := []string{stem}
		names[stem] = true
		m, err := loadDesignMachine(design, path)
		if err != nil {
			continue
		}
		if id := strings.TrimSpace(m.AsObject().GetString("id")); id != "" {
			names[id] = true
			machines = append(machines, id)
		}
		for _, st := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			names[st.Name], names[st.Path] = true, true
			for _, mn := range machines {
				names[mn+"."+st.Name], names[mn+"."+st.Path] = true, true
			}
		}
	}
	return names
}

// declaredSpans returns the byte ranges of a cell that sit inside a
// declaration group or a derived form.
func declaredSpans(cell string) [][2]int {
	var spans [][2]int
	for _, s := range scanGroups(cell) {
		end := s.end
		if end < 0 {
			end = len(cell) // unterminated: Gx reports it; nothing after it is prose to judge
		}
		spans = append(spans, [2]int{s.start, end})
	}
	for _, re := range []*regexp.Regexp{payloadDeclaration, valuesGroup, derivedFormRe} {
		for _, loc := range re.FindAllStringIndex(cell, -1) {
			spans = append(spans, [2]int{loc[0], loc[1]})
		}
	}
	return spans
}

// rowDeclaredTokens returns the facts a row itself declares: WRITES and
// USES members and `derived:` names, from any cell. A backticked quotation of
// a fact the same row declares is a declared reference, not an undeclared
// one; without this the warning's own advice ("declare it in USES{}") could
// never be taken, because the quotation stays outside the group it points to.
// Scope is the row: a declaration on another unit's row says nothing about
// what this unit reads or writes.
func rowDeclaredTokens(cells []string) map[string]bool {
	out := map[string]bool{}
	for _, cell := range cells {
		for _, span := range scanGroups(cell) {
			if span.end < 0 || span.nested || (span.name != GroupWrites && span.name != GroupUses) {
				continue
			}
			members, why := parseDottedMembers(span.name, span.body, true)
			if why != "" {
				continue // Gx reports the malformed group
			}
			for _, m := range members {
				out[m] = true
			}
		}
		for _, at := range derivedWord.FindAllStringIndex(cell, -1) {
			if m := derivedFact.FindStringSubmatch(cell[at[0]:]); m != nil {
				out[m[1]] = true
			}
		}
	}
	return out
}

func checkUndeclaredFactReferences(g *Gate, design string) {
	paths := sortedGlob(filepath.Join(design, "machines"), "*.matrix.md")
	if len(paths) == 0 {
		return
	}
	names := designNameTokens(design)
	for _, path := range paths {
		body, ok := readTextOK(design, path)
		if !ok {
			continue // Gx and G3 report an unreadable matrix
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		for _, r := range walkTableRows(normalizeNewlines([]byte(body))) {
			cols := factReferenceCells(r.header)
			if len(cols) == 0 {
				continue
			}
			row := rowIdentity(r)
			own := rowDeclaredTokens(r.cells)
			for _, n := range strings.Split(row, "/") {
				own[strings.TrimSpace(n)] = true
			}
			seen := map[string]bool{}
			for _, ci := range cols {
				cell := cellAt(r.cells, ci)
				spans := declaredSpans(cell)
			tokens:
				for _, loc := range factBacktickSpan.FindAllStringSubmatchIndex(cell, -1) {
					for _, sp := range spans {
						if loc[0] >= sp[0] && loc[1] <= sp[1] {
							continue tokens
						}
					}
					tok := strings.TrimSpace(cell[loc[2]:loc[3]])
					if !snakeFactToken.MatchString(tok) && !entityAttrFactToken.MatchString(tok) {
						continue
					}
					if own[tok] || names[tok] || seen[tok] {
						continue
					}
					seen[tok] = true
					g.Warns = append(g.Warns, rel+":"+strconv.Itoa(r.line)+": row "+ir.Repr(row)+": undeclared fact reference `"+tok+"`; declare it in USES{} or WRITES{} or drop the backticks")
				}
			}
		}
	}
}

// checkValuesEnumCase warns on an unnamed VALUES{...} group whose unit name
// matches a Modelith enum name only when case is ignored. The consistency
// rules bind a group to an enum by exact name, and an unnamed group's name is
// its unit's name, so VALUES{...} on unit orderState binds no enum
// OrderState: the agreement the author meant is never checked. The binding
// rule stays exact (no case folding, no fuzzy join); this warning tells the
// author to name the group, VALUES OrderState{...}, which binds it.
func checkValuesEnumCase(g *Gate, design string) {
	paths := sortedGlob(filepath.Join(design, "machines"), "*.matrix.md")
	if len(paths) == 0 {
		return
	}
	dm := loadModelith(design, NewGate("values enum case"))
	if dm == nil {
		return // Gx reports the model
	}
	byLower := map[string][]string{}
	if enums := dm.AsObject().GetObject("enums"); enums != nil {
		for _, name := range enums.Keys() {
			byLower[strings.ToLower(name)] = append(byLower[strings.ToLower(name)], name)
		}
	}
	if len(byLower) == 0 {
		return
	}
	for _, path := range paths {
		body, ok := readTextOK(design, path)
		if !ok {
			continue
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		for _, r := range walkTableRows(normalizeNewlines([]byte(body))) {
			ni, _, _, named := namedUnitCols(r.header)
			if !named {
				continue
			}
			unnamed := false
			for _, cell := range r.cells {
				for _, m := range valuesGroup.FindAllStringSubmatch(cell, -1) {
					if m[1] == "" {
						unnamed = true
					}
				}
			}
			if !unnamed {
				continue
			}
			for _, unit := range unitNames(cellAt(r.cells, ni)) {
				for _, enum := range byLower[strings.ToLower(unit)] {
					if enum == unit {
						continue
					}
					g.Warns = append(g.Warns, rel+":"+strconv.Itoa(r.line)+": row "+ir.Repr(unit)+": unnamed VALUES{...} takes the unit name "+ir.Repr(unit)+
						", which differs from enum "+ir.Repr(enum)+" only in case and so binds no enum; name the group (VALUES "+enum+"{...}) to bind it")
				}
			}
		}
	}
}
