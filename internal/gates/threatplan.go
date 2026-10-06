// Threat-driven milestone rules (docs/threat-driven-verification.md, Part A).
// A milestone whose text names a security-relevant subject owes more than a
// happy path: Gb-plan holds paired Accept/Refuse criteria, a RED threat table
// bound to the threat ledger, and a Pass-wrongly line on every OPEN such
// milestone; Ga-accept holds the independent review's threat_review rows on
// every ACCEPTED one dated on or after the ledger's enforced_since. Both read
// the same scope and the same milestone texts here, so the two gates can never
// disagree on which milestone is security-relevant. Gz-threat owns the ledger
// itself: a ledger that does not parse adds no finding here.

package gates

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/threat"
)

var (
	// criterionLineRe matches a standalone Accept: or Refuse: criterion line,
	// tolerating the bullet and bold decorations the other labeled lines do.
	criterionLineRe = regexp.MustCompile(`(?i)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*{0,2}(accept|refuse):\*{0,2}[ \t]*(.*)$`)
	// passWronglyLineRe matches the third RED quality check's answer line.
	passWronglyLineRe = regexp.MustCompile(`(?mi)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*{0,2}Pass-wrongly:\*{0,2}[ \t]*([^\n]*)$`)
)

// threatScope is what the milestone rules need from the ledger and the model:
// which subject names make a milestone security-relevant, and in which mode.
type threatScope struct {
	ledger  *threat.Ledger // nil without a ledger: names come from threat.Detect
	enforce bool
	names   []string            // security-relevant subject names, sorted
	owners  map[string][]string // threat invariant id -> the subjects producing it
}

// loadThreatScope reads design/threats.yaml, or without one detects the
// model's candidate subjects. nil means nothing is security-relevant here: no
// ledger and no model (or no candidate), or a ledger Gz reports as broken.
func loadThreatScope(design string) *threatScope {
	has, err := probeRegularFile(design, threat.FileName)
	if err != nil {
		return nil // Gz owns an unreadable or irregular ledger
	}
	if has {
		data, rerr := readDesignFile(design, filepath.Join(design, threat.FileName))
		if rerr != nil {
			return nil
		}
		l, _ := threat.Parse(data)
		if l == nil {
			return nil
		}
		sc := &threatScope{ledger: l, enforce: l.Enforcing(), owners: map[string][]string{}}
		for _, s := range l.Subjects {
			sc.names = append(sc.names, s.Name)
			for _, r := range s.Rows {
				if r.Invariant != "" {
					sc.owners[r.Invariant] = append(sc.owners[r.Invariant], s.Name)
				}
			}
		}
		sort.Strings(sc.names)
		return sc
	}
	// the model is optional here: its absence is Gx's finding, not Gb's or Ga's
	model := loadModelith(design, NewGate(""))
	if model == nil || model.AsObject() == nil {
		return nil
	}
	sc := &threatScope{}
	for _, c := range threat.Detect(model.AsObject()) {
		sc.names = append(sc.names, c.Subject)
	}
	if len(sc.names) == 0 {
		return nil
	}
	return sc
}

// named returns the subjects text names, sorted and unique: a subject named
// itself as a whole token, or through one of its threat invariant ids.
func (sc *threatScope) named(text string) []string {
	set := map[string]bool{}
	for _, n := range sc.names {
		if subjectTokenIn(n, text) {
			set[n] = true
		}
	}
	for inv, subjects := range sc.owners {
		if tokenIn(inv, text) {
			for _, s := range subjects {
				set[s] = true
			}
		}
	}
	return sortedKeys(set)
}

// report files a finding in the scope's mode: an ERROR when the ledger
// enforces, an "audit:" note otherwise.
func (sc *threatScope) report(g *Gate, msg string) {
	if sc.enforce {
		g.Errs = append(g.Errs, msg)
		return
	}
	g.Notes = append(g.Notes, "audit: "+msg)
}

// subjectTokenIn reports whether subject occurs in text as a whole token. An
// entity matches as a whole word (case-sensitive, so prose "session" is not
// the entity Session); a "." after it is fine, so Session.login names Session.
// An Entity.action subject matches as the whole dotted token: a further
// ".segment" makes it a different path, while a sentence-ending period does
// not. A "." before the subject glues, as does any identifier character.
func subjectTokenIn(subject, text string) bool {
	dotted := strings.Contains(subject, ".")
	idx := 0
	for {
		i := strings.Index(text[idx:], subject)
		if i < 0 {
			return false
		}
		pos := idx + i
		end := pos + len(subject)
		beforeOK := pos == 0 || (!isTokenChar(text[pos-1]) && text[pos-1] != '.')
		afterOK := end == len(text) || !isTokenChar(text[end])
		if afterOK && dotted && end < len(text) && text[end] == '.' && end+1 < len(text) && isTokenChar(text[end+1]) {
			afterOK = false
		}
		if beforeOK && afterOK {
			return true
		}
		idx = pos + 1
	}
}

// threatDoc is one document a milestone's threat obligations live in.
type threatDoc struct {
	label string // the name findings address it by
	text  string // fence-masked
}

// milestoneThreatDocs returns the texts that carry m's threat obligations: its
// root block in full mode, its packet in pairwise manifest mode, each of its
// shards in matrix mode. A link the structural checks reject yields nothing
// here; those checks already report it.
func milestoneThreatDocs(design string, m planMilestone, mode, linkage string) []threatDoc {
	if mode != "manifest" {
		return []threatDoc{{label: "BUILD.md", text: m.block}}
	}
	re := packetLineRe
	if linkage == "matrix" {
		re = shardLineRe
	}
	matches := re.FindAllStringSubmatch(m.block, -1)
	if linkage != "matrix" && len(matches) != 1 {
		return nil
	}
	var out []threatDoc
	seen := map[string]bool{}
	for _, match := range matches {
		path, ok := portablePacketPath(strings.TrimSpace(match[1]))
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, threatDoc{label: path, text: maskFences(readDesignOrEmpty(design, filepath.Join(design, filepath.FromSlash(path))))})
	}
	return out
}

// planLinkage returns the BUILD.md mode and, in manifest mode, its linkage.
func planLinkage(text string) (mode, linkage string) {
	mode = planMode(text)
	if mode == "manifest" {
		linkage, _ = manifestLinkageMode(text)
	}
	return mode, linkage
}

// threatUnit is one document checked once, with every open milestone it
// serves (a matrix shard may serve several).
type threatUnit struct {
	doc        threatDoc
	milestones []planMilestone
}

func (u threatUnit) who() string {
	var parts []string
	for _, m := range u.milestones {
		parts = append(parts, fmt.Sprintf("M%s (%s)", m.numRaw, m.title))
	}
	if len(parts) == 1 {
		return "milestone " + parts[0]
	}
	return "milestones " + strings.Join(parts, ", ")
}

// checkThreatMilestones is Gb-plan's half: every open milestone whose text
// names a security-relevant subject carries paired criteria, a threat table,
// and a Pass-wrongly line in that text.
func checkThreatMilestones(g *Gate, design, text string) {
	sc := loadThreatScope(design)
	if sc == nil {
		return
	}
	ms, ok := planMilestonesOf(text)
	if !ok {
		return
	}
	mode, linkage := planLinkage(text)
	var order []string
	units := map[string]*threatUnit{}
	for _, m := range ms {
		if m.status != "" && m.status != "open" {
			continue // closed milestones are history
		}
		for _, d := range milestoneThreatDocs(design, m, mode, linkage) {
			// a packet or shard is checked once however many milestones it
			// serves; each full-mode block is its own text in the one root
			key := d.label
			if mode != "manifest" {
				key += "#M" + m.numRaw
			}
			u, ok := units[key]
			if !ok {
				u = &threatUnit{doc: d}
				units[key] = u
				order = append(order, key)
			}
			u.milestones = append(u.milestones, m)
		}
	}
	relevant := map[string]bool{}
	for _, label := range order {
		u := units[label]
		names := sc.named(u.doc.text)
		if len(names) == 0 {
			continue
		}
		for _, m := range u.milestones {
			relevant[m.numRaw] = true
		}
		where := u.doc.label + ": " + u.who()
		checkPairedCriteria(g, sc, where, names, u.doc.text)
		checkThreatTable(g, sc, where, names, u.doc.text)
		checkPassWrongly(g, sc, where, u.doc.text)
	}
	if len(relevant) > 0 {
		g.Count("security-relevant milestones", len(relevant))
	}
}

// checkPairedCriteria holds rule 1: at least one Accept: line, each followed
// (next non-blank line) by its Refuse: line, and no Refuse: without an Accept.
func checkPairedCriteria(g *Gate, sc *threatScope, where string, names []string, text string) {
	var labels, lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		label := ""
		if m := criterionLineRe.FindStringSubmatch(line); m != nil {
			label = strings.ToLower(m[1])
		}
		labels = append(labels, label)
		lines = append(lines, strings.TrimSpace(line))
	}
	accepts, pairs := 0, 0
	var unpaired, orphans []string
	for i, label := range labels {
		switch label {
		case "accept":
			accepts++
			if i+1 < len(labels) && labels[i+1] == "refuse" {
				pairs++
			} else {
				unpaired = append(unpaired, lines[i])
			}
		case "refuse":
			if i == 0 || labels[i-1] != "accept" {
				orphans = append(orphans, lines[i])
			}
		}
	}
	subjects := strings.Join(names, ", ")
	switch {
	case accepts == 0:
		sc.report(g, fmt.Sprintf("%s names security-relevant subject(s) %s but states no paired acceptance criteria; write each property as an 'Accept:' line directly followed by the 'Refuse:' line naming what must be refused", where, subjects))
	case pairs == 0:
		sc.report(g, fmt.Sprintf("%s names security-relevant subject(s) %s but states only positive criteria; follow each 'Accept:' line with the 'Refuse:' line naming what must be refused", where, subjects))
	default:
		for _, l := range unpaired {
			sc.report(g, fmt.Sprintf("%s has an Accept: line not followed by its Refuse: line (%q); put the matching 'Refuse:' line directly after it", where, l))
		}
	}
	for _, l := range orphans {
		sc.report(g, fmt.Sprintf("%s has a Refuse: line with no Accept: line before it (%q); pair it with the Accept: line it refutes", where, l))
	}
}

// threatTableRow is one data row of a milestone's threat table.
type threatTableRow struct {
	adversary, subject, negative, risk string
	hasSubject                         bool
}

// threatTables returns the rows of every table whose header has an
// adversary column and a negative test column, and whether any such table
// exists at all.
func threatTables(text string) ([]threatTableRow, [][]string, bool) {
	var rows []threatTableRow
	var cells [][]string
	found := false
	for _, t := range ir.ParseMdTables(text) {
		adv := ir.FindCol(t.Header, "adversary")
		neg := ir.FindCol(t.Header, "negative test", "negative_test", "negative tests")
		if adv < 0 || neg < 0 {
			continue
		}
		found = true
		risk := ir.FindCol(t.Header, "accepted risk", "accepted_risk")
		subj := ir.FindCol(t.Header, "subject")
		cell := func(row []string, i int) string {
			if i < 0 || i >= len(row) {
				return ""
			}
			return row[i]
		}
		for _, r := range t.Rows {
			rows = append(rows, threatTableRow{
				adversary: cell(r, adv), negative: cell(r, neg), risk: cell(r, risk),
				subject: cell(r, subj), hasSubject: subj >= 0,
			})
			cells = append(cells, r)
		}
	}
	return rows, cells, found
}

// checkThreatTable holds rule 2. With a ledger, one row per adversary row of
// every named subject except not_applicable ones, bound to its negative test
// or its accepted risk's owner; without one, a table with at least one row.
// A cell that closes a threat by pointing at prose, or by agreement with a
// reference, is an ERROR in every mode: what is written is always held.
func checkThreatTable(g *Gate, sc *threatScope, where string, names []string, text string) {
	rows, cells, found := threatTables(text)
	if !found {
		sc.report(g, fmt.Sprintf("%s names security-relevant subject(s) %s but carries no threat table; add a Markdown table with an 'adversary' column and a 'negative test' column (an 'accepted risk' column is optional), one row per adversary the ledger does not mark not_applicable", where, strings.Join(names, ", ")))
		return
	}
	for _, r := range cells {
		for _, c := range r {
			switch {
			case threat.Documented(c):
				g.Errs = append(g.Errs, fmt.Sprintf("%s: threat table cell %q: \"documented\" is not a closure; name the locked negative test, or the owner-signed accepted risk design/threats.yaml records", where, c))
			case threat.AgreementOnly(c):
				g.Errs = append(g.Errs, fmt.Sprintf("%s: threat table cell %q only claims agreement; agreement with a reference is not correctness, name an independent negative test", where, c))
			}
		}
	}
	if len(rows) == 0 {
		sc.report(g, fmt.Sprintf("%s: the threat table has no rows; list each adversary the milestone faces with its locked negative test", where))
		return
	}
	if sc.ledger == nil {
		return
	}
	for _, s := range sc.ledger.Subjects {
		if !containsString(names, s.Name) {
			continue
		}
		for _, lr := range s.Rows {
			if !lr.Covered() && lr.Risk == nil {
				continue // not_applicable: no attack surface, no row owed
			}
			var candidates []threatTableRow
			for _, tr := range rows {
				if tokenIn(lr.Adversary, tr.adversary) && (!tr.hasSubject || subjectTokenIn(s.Name, tr.subject)) {
					candidates = append(candidates, tr)
				}
			}
			if len(candidates) == 0 {
				if lr.Covered() {
					sc.report(g, fmt.Sprintf("%s: the threat table has no row for %s adversary %s; add one naming its locked negative test %s (design/threats.yaml)", where, s.Name, lr.Adversary, lr.NegativeTest))
				} else {
					sc.report(g, fmt.Sprintf("%s: the threat table has no row for %s adversary %s; add one naming the accepted risk owned by %s (design/threats.yaml)", where, s.Name, lr.Adversary, lr.Risk.Owner))
				}
				continue
			}
			satisfied := false
			for _, tr := range candidates {
				if lr.Covered() && idTokenIn(lr.NegativeTest, tr.negative) {
					satisfied = true
				}
				if lr.Risk != nil && lr.Risk.Owner != "" {
					owner := strings.ToLower(lr.Risk.Owner)
					if strings.Contains(strings.ToLower(tr.risk), owner) || strings.Contains(strings.ToLower(tr.negative), owner) {
						satisfied = true
					}
				}
			}
			switch {
			case satisfied:
				g.Count("threat table rows bound")
			case lr.Covered():
				sc.report(g, fmt.Sprintf("%s: the threat table row for %s adversary %s does not name its locked negative test %s; design/threats.yaml binds that adversary to it", where, s.Name, lr.Adversary, lr.NegativeTest))
			default:
				sc.report(g, fmt.Sprintf("%s: the threat table row for %s adversary %s does not name the accepted risk's owner %s; the table carries the owner-signed risk design/threats.yaml records", where, s.Name, lr.Adversary, lr.Risk.Owner))
			}
		}
	}
}

// checkPassWrongly holds rule 3: one standalone, non-empty Pass-wrongly line.
func checkPassWrongly(g *Gate, sc *threatScope, where, text string) {
	lines := passWronglyLineRe.FindAllStringSubmatch(text, -1)
	switch {
	case len(lines) == 0:
		sc.report(g, fmt.Sprintf("%s has no Pass-wrongly: line; answer \"what would make this pass wrongly?\" on one standalone 'Pass-wrongly: <answer>' line", where))
	case len(lines) > 1:
		sc.report(g, fmt.Sprintf("%s has %d Pass-wrongly: lines; keep exactly one standalone 'Pass-wrongly: <answer>' line", where, len(lines)))
	case strings.TrimSpace(lines[0][1]) == "":
		sc.report(g, fmt.Sprintf("%s has an empty Pass-wrongly: line; answer \"what would make this pass wrongly?\" on it", where))
	}
}

// exitStatusWords are the words a reason that only reports an exit status is
// made of. A reason left with any other word names why the probe was refused.
var exitStatusWords = map[string]bool{
	"exit": true, "exited": true, "exits": true, "code": true, "status": true, "rc": true,
	"non": true, "nonzero": true, "zero": true, "with": true, "a": true, "an": true, "the": true,
	"returned": true, "returns": true, "return": true, "failed": true, "fails": true, "fail": true,
	"failure": true, "error": true, "errored": true, "process": true, "command": true, "value": true, "of": true,
}

// exitStatusOnly reports a reason that says only that the process failed:
// "exit 1", "non-zero exit", "exit code", "failed". A non-zero exit is not a
// reason; the probe asserts why it was refused.
func exitStatusOnly(reason string) bool {
	words := strings.FieldsFunc(strings.ToLower(reason), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	if len(words) == 0 {
		return false
	}
	for _, w := range words {
		if exitStatusWords[w] || strings.Trim(w, "0123456789") == "" {
			continue
		}
		return false
	}
	return true
}

// threatReviewRow is one row of an acceptance file's threat_review.
type threatReviewRow struct {
	adversary, subject, probe, expected, observed string
}

// threatReviewKeys is the closed key set of one threat_review row.
var threatReviewKeys = []string{"adversary", "subject", "probe", "expected_reason", "observed_reason"}

// acceptThreatReview reads the optional threat_review list. Malformed rows
// are errors in every mode: an evidence row nobody can read binds nothing.
func acceptThreatReview(g *Gate, label string, root *ir.Object) []threatReviewRow {
	v := root.Get2("threat_review")
	if v == nil || v.Kind == ir.KindNull {
		return nil
	}
	if v.Kind != ir.KindArray {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: threat_review must be a list of mappings, each with exactly %s", label, strings.Join(threatReviewKeys, ", ")))
		return nil
	}
	var out []threatReviewRow
	for i, e := range v.AsArray() {
		o := e.AsObject()
		if o == nil {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: threat_review[%d] is not a mapping; a row carries exactly %s", label, i, strings.Join(threatReviewKeys, ", ")))
			continue
		}
		bad := false
		for _, k := range o.Keys() {
			if !containsString(threatReviewKeys, k) {
				g.Errs = append(g.Errs, fmt.Sprintf("%s: threat_review[%d] has unknown key %q; a row carries exactly %s", label, i, k, strings.Join(threatReviewKeys, ", ")))
				bad = true
			}
		}
		vals := map[string]string{}
		for _, k := range threatReviewKeys {
			vals[k] = strings.TrimSpace(o.GetString(k))
			if vals[k] == "" {
				g.Errs = append(g.Errs, fmt.Sprintf("%s: threat_review[%d] needs a non-empty %s", label, i, k))
				bad = true
			}
		}
		if a := vals["adversary"]; a != "" && !containsString(threat.Adversaries, a) {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: threat_review[%d] adversary %q is not a standard adversary (expected one of %s)", label, i, a, strings.Join(threat.Adversaries, ", ")))
			bad = true
		}
		if bad {
			continue
		}
		out = append(out, threatReviewRow{adversary: vals["adversary"], subject: vals["subject"], probe: vals["probe"], expected: vals["expected_reason"], observed: vals["observed_reason"]})
	}
	return out
}

// checkThreatReviews is Ga-accept's half: an ACCEPTED acceptance of a
// security-relevant milestone, dated on or after the ledger's enforced_since,
// records a threat_review row for every ledger adversary row (except
// not_applicable) of every subject the milestone names, each refused for the
// reason the threat table expects.
func checkThreatReviews(g *Gate, design string, records map[int]*acceptRecord, byNum map[int]milestoneRef) {
	sc := loadThreatScope(design)
	if sc == nil {
		return
	}
	mode, linkage := planLinkage(readDesignOrEmpty(design, filepath.Join(design, "BUILD.md")))
	for _, num := range sortedRecordNums(records) {
		rec := records[num]
		ref, ok := byNum[num]
		if !ok || rec.verdict != "ACCEPTED" {
			continue
		}
		if sc.ledger != nil && sc.ledger.EnforcedSince != "" && rec.date < sc.ledger.EnforcedSince {
			continue // accepted before adoption: history
		}
		var texts []string
		for _, d := range milestoneThreatDocs(design, ref.m, mode, linkage) {
			texts = append(texts, d.text)
		}
		names := sc.named(strings.Join(texts, "\n"))
		if len(names) == 0 {
			continue
		}
		g.Count("security-relevant acceptances")
		for _, r := range rec.threatReview {
			switch {
			case exitStatusOnly(r.expected) || exitStatusOnly(r.observed):
				sc.report(g, fmt.Sprintf("%s: threat_review row for %s adversary %s gives expected_reason %q and observed_reason %q; a non-zero exit is not a reason, record the failure reason the probe asserted", rec.label, r.subject, r.adversary, r.expected, r.observed))
			case r.expected != r.observed:
				sc.report(g, fmt.Sprintf("%s: threat_review row for %s adversary %s expected_reason %q but observed_reason %q; the probe was refused for a different reason than the threat table expects, so the threat is not shown refused", rec.label, r.subject, r.adversary, r.expected, r.observed))
			}
		}
		if sc.ledger == nil {
			if len(rec.threatReview) == 0 {
				sc.report(g, fmt.Sprintf("%s: milestone M%d names security-relevant subject(s) %s but records no threat_review; the independent review writes its own threat table first, probes each row with a real process, and records the probe with its expected and observed failure reason", rec.label, num, strings.Join(names, ", ")))
			}
			continue
		}
		for _, s := range sc.ledger.Subjects {
			if !containsString(names, s.Name) {
				continue
			}
			for _, lr := range s.Rows {
				if !lr.Covered() && lr.Risk == nil {
					continue
				}
				var found *threatReviewRow
				for i := range rec.threatReview {
					if rec.threatReview[i].adversary == lr.Adversary && rec.threatReview[i].subject == s.Name {
						found = &rec.threatReview[i]
						break
					}
				}
				switch {
				case found == nil:
					sc.report(g, fmt.Sprintf("%s: records no threat_review row for %s adversary %s; probe it with a real process and record the probe, the expected failure reason, and the observed one", rec.label, s.Name, lr.Adversary))
				case !exitStatusOnly(found.expected) && !exitStatusOnly(found.observed) && found.expected == found.observed:
					g.Count("threat reviews bound")
				}
			}
		}
	}
}

// containsString reports whether xs holds x.
func containsString(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}
