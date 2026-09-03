// Gb-plan: the build-plan structure gate. The Build plan section of BUILD.md
// is what a coding agent actually schedules from, so its shape is held like
// any other artifact: the section exists, milestones are marked and uniquely
// numbered, the walking skeleton comes first, every milestone carries a
// definition of done, an optional status line parses, and the skeleton's DoD
// cites at least one committed oracle id. Gx owns the Mode line and the
// Toolchain / State-migration sections; Gb never re-checks them. Ga-accept
// reads the same parsed milestones (accept.go) and owns what a closed one
// must have behind it.

package gates

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
)

// HasBuildDoc reports whether the design has a BUILD.md (Phase 4 produced a
// build document); Gb auto-activates on it.
func HasBuildDoc(design string) bool {
	has, err := probeRegularFile(design, "BUILD.md")
	return has || err != nil
}

var (
	// milestoneRe matches a milestone marker: a bold span opening with
	// M<digits> and a "-" or ":" separator, anchored to the START of a line
	// (optionally after whitespace and a single list bullet), so a bold
	// cross-reference mid-prose ("built by **M0 - Walking skeleton** above")
	// never declares a phantom milestone. The title is the rest of the span,
	// tolerating the trailing period of the prose style ("**M0 - Walking
	// skeleton.** ...").
	milestoneRe = regexp.MustCompile(`(?m)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*\*M(\d+)\s*[-:]\s*([^*]+?)\.?\*\*`)
	// dodPhraseRe is the long form of the DoD token: the phrase immediately
	// followed by a colon, so mid-sentence prose ("the team's definition of
	// done conventions") never counts as a definition of done.
	dodPhraseRe = regexp.MustCompile(`(?i)definition of done:`)
	// skeletonWaiverRe is the explicit skeleton waiver line; the reason is
	// mandatory (an unexplained waiver is an unanswered planning question).
	skeletonWaiverRe = regexp.MustCompile(`(?i)walking skeleton:\s*n/a\s*-\s*\S`)
	// planWaiverRe is the documented section-waiver literal: N/A, a hyphen,
	// a reason (case-sensitive; NG-6).
	planWaiverRe = regexp.MustCompile(`^N/A\s+-\s+\S`)
	// milestoneStatusRe matches a milestone's optional status line. The shape
	// mirrors the DoD line convention (one labeled line inside the milestone
	// block) and tolerates the same bullet and bold decorations the milestone
	// markers use: "Status: closed", "- Status: closed", "**Status:** closed".
	// The value is captured so an unrecognized token fails loudly in Gb
	// instead of silently reading as "not closed" and disarming Ga.
	milestoneStatusRe = regexp.MustCompile(`(?mi)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*{0,2}Status:\*{0,2}[ \t]*([A-Za-z][A-Za-z-]*)`)
	// packetLineRe binds one manifest milestone to one Markdown packet. The
	// destination is intentionally a plain relative path: fragments, URLs,
	// angle destinations, and title suffixes would make inventory comparison
	// ambiguous across Markdown implementations.
	packetLineRe = regexp.MustCompile(`(?mi)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*{0,2}Packet:\*{0,2}[ \t]*\[[^\]\n]+\]\(([^)\n]+)\)[ \t]*$`)
	// demoLineRe makes every manifest milestone promise one independently
	// observable result. The content is a judgment, but presence and
	// non-emptiness are deterministic.
	demoLineRe    = regexp.MustCompile(`(?mi)^[ \t]*(?:[-*][ \t]+|\d+\.[ \t]+)?\*{0,2}Demo:\*{0,2}[ \t]*([^\n]*)$`)
	packetTitleRe = regexp.MustCompile(`^M(\d+)[ \t]*[-:][ \t]*\S`)
	// skeletonNFRRe matches the skeleton milestone's NFR line: the literal
	// token "NFR:" (bold decoration tolerated, like the status line) with the
	// rest of the line captured, so an empty declaration fails loudly. Not
	// anchored to line starts, matching the DoD-token convention.
	skeletonNFRRe = regexp.MustCompile(`\*{0,2}NFR:\*{0,2}[ \t]*([^\n]*)`)
	// planHeadingRe matches a Build plan heading. The phrase "Build plan" may
	// sit anywhere in the heading text, so the template's "N. " section number
	// and any trailing decoration still name the plan section: a real design
	// titled its section "## 9. Build plan (sealed trust layers; user
	// directive 2026-08-04)" and an exact-title match silently found no plan
	// at all (manifest mode reported "7 plans, 7 waived plans" and checked
	// nothing; full mode would have claimed the section was absent).
	// "Build plan" must appear as a WHOLE phrase: "Milestone map" and
	// "Build planning notes" are not plan sections, and neither is "Rebuild
	// plan".
	planHeadingRe = regexp.MustCompile(`(?i)(^|[^a-z])build[ \t]+plan([^a-z]|$)`)
)

const maxExecutionPacketBytes = 64 << 10

var executionPacketSections = []string{
	"Outcome",
	"Domain context",
	"Architecture context",
	"Behavior and oracles",
	"TDD and implementation",
	"Risks and recovery",
	"Acceptance",
}

// idTokenIn reports whether an oracle or test id occurs in text as a whole
// token. The boundaries differ from tokenIn (the invariant matcher): a
// hyphen still glues, because the ids are themselves hyphenated ("X-DEAL-eb0c40"
// does not contain "DEAL-eb0c40"), but an underscore is a boundary, because
// test frameworks join names with it (a Go subtest literal
// "T-DEAL-01_DEAL-eb0c40" cites both ids).
func idTokenIn(token, text string) bool {
	idx := 0
	for {
		i := strings.Index(text[idx:], token)
		if i < 0 {
			return false
		}
		pos := idx + i
		beforeOK := pos == 0 || !isIDTokenChar(text[pos-1])
		afterOK := pos+len(token) == len(text) || !isIDTokenChar(text[pos+len(token)])
		if beforeOK && afterOK {
			return true
		}
		idx = pos + 1
	}
}

func isIDTokenChar(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9') || b == '-'
}

// maskFences blanks fenced code blocks (``` and ~~~) line by line, preserving
// the line structure so byte offsets stay valid. The CommonMark run-length
// rule applies: a fence opened with N delimiter characters closes only on a
// line of >= N of the SAME character, so a 4-backtick documentation fence
// swallows its inner ``` lines instead of leaking example content as plan
// structure (NG-5). Every Gb scan runs on the masked text: a bash "# comment"
// inside a fence is not a heading, and a fenced fake "**M9 - ...**" or
// "DoD:" is not plan structure. Neither ir.ParseMdTables nor Gx's BUILD
// scans are fence-aware; this helper is Gb's own.
func maskFences(text string) string {
	lines := strings.Split(text, "\n")
	fenceChar := byte(0) // the active fence character, 0 when outside a fence
	fenceLen := 0
	for i, line := range lines {
		t := strings.TrimLeft(line, " \t")
		switch {
		case fenceChar == 0 && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")):
			fenceChar = t[0]
			fenceLen = leadingRun(t, fenceChar)
			lines[i] = ""
		case fenceChar != 0:
			if t != "" && t[0] == fenceChar && leadingRun(t, fenceChar) >= fenceLen {
				fenceChar, fenceLen = 0, 0
			}
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// leadingRun counts the leading run of c in s.
func leadingRun(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}

// headingText parses a markdown ATX heading line into level and text; level 0
// means the line is not a heading.
func headingText(line string) (int, string) {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n >= len(line) || line[n] != ' ' {
		return 0, ""
	}
	return n, strings.TrimSpace(line[n:])
}

// buildPlanSection returns the body of the Build plan section: a ## or ###
// heading whose text carries the phrase "Build plan", whatever the section
// number and whatever trailing decoration the heading adds. The body runs to
// the next heading of the same or higher level, so subheadings stay inside
// the section. When a document declares more than one such section only the
// first body is returned; checkPlanDoc ERRORs on the count, so a milestone
// can never hide in a second section a first-match read would skip (the
// PACK-1 every-match rule, discharged here by refusing the ambiguity).
func buildPlanSection(text string) (string, bool) {
	bodies := buildPlanSections(text)
	if len(bodies) == 0 {
		return "", false
	}
	return bodies[0], true
}

// buildPlanSections returns the body of EVERY Build plan section in document
// order. A matching subheading inside an already-collected section is part of
// that section's body, not a section of its own, so the scan resumes past
// each collected body.
func buildPlanSections(text string) []string {
	lines := strings.Split(text, "\n")
	var bodies []string
	for i := 0; i < len(lines); i++ {
		level, title := headingText(lines[i])
		if (level != 2 && level != 3) || !planHeadingRe.MatchString(title) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l, _ := headingText(lines[j]); l > 0 && l <= level {
				end = j
				break
			}
		}
		bodies = append(bodies, strings.Join(lines[i+1:end], "\n"))
		i = end - 1
	}
	return bodies
}

// planOracleIDs collects BOTH id columns (sequential test ids and stable
// ids) of every committed machines/*.oracle.md table, as read from the
// files: the id shapes are never assumed. The committed files are the source
// here; G3 separately holds them fresh against the machines.
func planOracleIDs(design string, g *Gate) []string {
	var ids []string
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.oracle.md") {
		testIDs, stableIDs := oracleTableIDs(readDesignFileOrErr(design, path, g))
		ids = append(ids, testIDs...)
		ids = append(ids, stableIDs...)
	}
	return ids
}

// planMode returns the BUILD.md mode declaration ("full" or "manifest").
// Gx owns findings about the Mode line itself; an absent declaration falls
// back to full so a pre-Gx draft still gets its plan checked. The sniff runs
// on fence-masked text: a fenced example "Mode:" line must not override the
// real declaration (NG-4).
func planMode(text string) string {
	if m := modeRe.FindStringSubmatch(maskFences(text)); m != nil {
		return m[1]
	}
	return "full"
}

// planPackets lists a manifest design's execution packets under BUILD/ plus
// the number of navigation index files exempted. README.md and index.md carry
// no packet obligation.
func planPackets(design string) (packets []string, indexFiles int, err error) {
	paths, err := sortedGlobExt(filepath.Join(design, "BUILD"), ".md")
	if err != nil {
		return nil, 0, err
	}
	for _, packet := range paths {
		switch strings.ToLower(filepath.Base(packet)) {
		case "readme.md", "index.md":
			indexFiles++
		default:
			packets = append(packets, packet)
		}
	}
	return packets, indexFiles, nil
}

// portablePacketPath accepts only a direct BUILD/*.md destination. Keeping
// the grammar smaller than general Markdown makes the root manifest and the
// on-disk inventory exactly comparable on every supported OS.
func portablePacketPath(raw string) (string, bool) {
	if raw == "" || strings.ContainsAny(raw, `\\?#`) || filepath.IsAbs(raw) {
		return "", false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(raw)))
	if clean != raw || filepath.ToSlash(filepath.Dir(filepath.FromSlash(raw))) != "BUILD" {
		return "", false
	}
	base := filepath.Base(filepath.FromSlash(raw))
	if strings.EqualFold(base, "README.md") || strings.EqualFold(base, "index.md") || !strings.EqualFold(filepath.Ext(base), ".md") {
		return "", false
	}
	return clean, true
}

func packetHeadingCounts(text string) map[string]int {
	counts := make(map[string]int, len(executionPacketSections))
	for _, line := range strings.Split(maskFences(text), "\n") {
		level, title := headingText(line)
		if level != 2 {
			continue
		}
		for _, required := range executionPacketSections {
			if strings.EqualFold(strings.TrimSpace(title), required) {
				counts[required]++
			}
		}
	}
	return counts
}

// checkManifestPackets holds the small-model execution boundary: the root is
// the sole milestone/demo manifest, and every non-index BUILD/*.md file is a
// bounded packet owned by exactly one milestone.
func checkManifestPackets(g *Gate, design, root string, packets []string) {
	body, ok := buildPlanSection(maskFences(root))
	if !ok {
		g.Errs = append(g.Errs, "BUILD.md: manifest mode requires the root to carry the single Build plan; each root milestone links to one execution packet")
		return
	}
	if first := firstNonBlankLine(body); strings.HasPrefix(strings.ToUpper(first), "N/A") {
		g.Errs = append(g.Errs, "BUILD.md: a manifest root cannot waive its Build plan; it is the single milestone, demo, packet, and acceptance authority")
		return
	}
	milestones := parsePlanMilestones(body)
	if len(milestones) == 0 {
		return // checkPlanDoc reports the missing milestone structure
	}
	refs := make(map[string]int, len(milestones))
	owners := make(map[string]string, len(milestones))
	for _, m := range milestones {
		matches := packetLineRe.FindAllStringSubmatch(m.block, -1)
		switch len(matches) {
		case 0:
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has no Packet line; link exactly one direct BUILD/*.md execution packet", m.numRaw, m.title))
		case 1:
			path, valid := portablePacketPath(strings.TrimSpace(matches[0][1]))
			if !valid {
				g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has invalid packet path %s; use one portable direct path such as BUILD/M%s-name.md", m.numRaw, m.title, ir.Repr(strings.TrimSpace(matches[0][1])), m.numRaw))
				break
			}
			if !strings.HasPrefix(filepath.Base(filepath.FromSlash(path)), "M"+m.numRaw+"-") {
				g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) links %s; its packet basename must start M%s-", m.numRaw, m.title, ir.Repr(path), m.numRaw))
			}
			refs[path]++
			owners[path] = m.numRaw
		default:
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has %d Packet lines; each milestone owns exactly one packet", m.numRaw, m.title, len(matches)))
		}

		demos := demoLineRe.FindAllStringSubmatch(m.block, -1)
		switch len(demos) {
		case 0:
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has no Demo line; state one observable milestone result", m.numRaw, m.title))
		case 1:
			if strings.TrimSpace(demos[0][1]) == "" {
				g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has an empty Demo line; state one observable milestone result", m.numRaw, m.title))
			} else {
				g.Count("milestone demos")
			}
		default:
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: milestone M%s (%s) has %d Demo lines; state exactly one observable result", m.numRaw, m.title, len(demos)))
		}
	}

	inventory := make(map[string]string, len(packets))
	for _, packet := range packets {
		rel := "BUILD/" + filepath.Base(packet)
		inventory[rel] = packet
	}
	for _, ref := range sortedKeys(refs) {
		count := refs[ref]
		if count != 1 {
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: packet %s is linked by %d milestones; every packet must be owned by exactly one milestone", ref, count))
		}
		if _, exists := inventory[ref]; !exists {
			g.Errs = append(g.Errs, fmt.Sprintf("BUILD.md: packet %s linked by M%s is not a regular non-index file in BUILD/", ref, owners[ref]))
		}
	}
	for _, rel := range sortedKeys(inventory) {
		path := inventory[rel]
		count := refs[rel]
		if count == 0 {
			g.Errs = append(g.Errs, rel+": packet is not linked by any root milestone; every packet must be inventoried exactly once")
			continue
		}
		text := readDesignFileOrErr(design, path, g)
		if len([]byte(text)) > maxExecutionPacketBytes {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: execution packet is %d bytes; the maximum is %d bytes (split the milestone at a user-demonstrable boundary)", rel, len([]byte(text)), maxExecutionPacketBytes))
		}
		masked := maskFences(text)
		if len(buildPlanSections(masked)) > 0 || len(milestoneRe.FindAllStringIndex(masked, -1)) > 0 {
			g.Errs = append(g.Errs, rel+": execution packets contain context for one root milestone; keep milestone declarations and the Build plan only in BUILD.md")
		}
		var h1s []string
		for _, line := range strings.Split(masked, "\n") {
			if level, title := headingText(line); level == 1 {
				h1s = append(h1s, title)
			}
		}
		var title []string
		if len(h1s) == 1 {
			title = packetTitleRe.FindStringSubmatch(h1s[0])
		}
		if len(h1s) != 1 || title == nil || title[1] != owners[rel] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: packet must have exactly one first-level title identifying its owner as '# M%s - <title>'", rel, owners[rel]))
		}
		counts := packetHeadingCounts(masked)
		for _, section := range executionPacketSections {
			switch counts[section] {
			case 1:
				g.Count("packet sections")
			case 0:
				g.Errs = append(g.Errs, fmt.Sprintf("%s: missing required '## %s' section", rel, section))
			default:
				g.Errs = append(g.Errs, fmt.Sprintf("%s: required '## %s' section appears %d times; each packet section is unique", rel, section, counts[section]))
			}
		}
		g.Count("execution packets")
	}
}

// buildTemplateSections is the template's closed section list. Each entry is
// located by a heading phrase, at any level, in the root or (manifest mode)
// any packet; the needles accept the corpus's real spellings (the v0.3.13/14
// locator lesson). A section that does not apply keeps its heading and waives
// its body with 'N/A - <reason>', which satisfies presence by construction.
var buildTemplateSections = []struct {
	name    string
	needles []string
}{
	{"Purpose and scope", []string{"purpose"}},
	{"Glossary", []string{"glossary"}},
	{"Domain model", []string{"domain model"}},
	{"Architecture", []string{"architecture"}},
	{"Behavior (the state machines)", []string{"behavior"}},
	{"Traceability matrix", []string{"traceability"}},
	{"Test specification", []string{"test spec", "test suite"}},
	{"State migration", []string{"state migration"}},
	{"Build plan", []string{"build plan"}},
	{"Language realization notes", []string{"language realization", "realization notes", "toolchain"}},
	{"Hard-TDD protocol", []string{"hard-tdd", "hard tdd"}},
	{"Open questions and residual risks", []string{"open questions", "residual risk"}},
}

// gatesDisclaimerText is the template's "What the gates do not verify" block,
// owed verbatim by every root BUILD.md so a green check is never read as more
// than it is. Compared whitespace-collapsed: reflowed line breaks are fine,
// wording changes are not. TestGatesDisclaimerMatchesTemplate pins this
// constant to references/build-md-template.md.
const gatesDisclaimerText = `Not covered by any deterministic check or proof, by construction: whether the interrogation
extracted the RIGHT invariants (a shallow domain model gates clean); guard and action semantics in
code (the named-unit contracts carry them into tests; a wrong implementation of a correctly-named
guard is caught by tests, not proofs); races between concurrent machine instances, and message
loss, duplication, or reordering between machines (the models are single-instance; the
event-contract table and the idempotency contracts govern those seams, and the tests exercise
them); whether migration transformations preserve real production data (Gm proves decision
coverage, not the implementation or a database run); coupling through shared database tables or
bus topics (invisible to import analysis; the event-contract table governs it); and security,
capacity, and observability beyond what the Phase 2 NFR record captures.`

// collapseWS reduces every whitespace run to one space, the equivalence under
// which "verbatim" is judged (reflow is formatting, wording is content).
func collapseWS(s string) string { return strings.Join(strings.Fields(s), " ") }

// checkTemplateSections holds the union of the root's and packets' headings to
// the template's closed section list.
func checkTemplateSections(g *Gate, maskedDocs []string) {
	for _, s := range buildTemplateSections {
		found := false
		for _, text := range maskedDocs {
			for _, n := range s.needles {
				if headingContains(text, n) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			g.Count("template sections present")
		} else {
			g.Errs = append(g.Errs, "BUILD.md (root plus any packets) has no '"+s.name+"' section; fill every template section, or keep its heading and waive the body with 'N/A - <reason>'")
		}
	}
}

// disclaimerSource names where the canonical block lives, in every finding
// about it: the skill's installed reference, whose text this binary pins.
const disclaimerSource = "the block under 'What the gates do not verify' in references/build-md-template.md (the machinery skill's installed reference, pinned in this binary)"

// divergenceWindow is how many words of each side a divergence report shows.
const divergenceWindow = 8

// wordWindow renders up to divergenceWindow words of words starting at from,
// with an ellipsis when more follow; a from past the end says so in words,
// because "" would read as a missing report rather than a missing tail.
func wordWindow(words []string, from int) string {
	if from >= len(words) {
		return "(nothing; the text ends here)"
	}
	end := from + divergenceWindow
	tail := ""
	if end < len(words) {
		tail = " ..."
	} else {
		end = len(words)
	}
	return strings.Join(words[from:end], " ") + tail
}

// firstWordDivergence reports the 1-based word position where want and got
// first differ under whitespace collapse, plus a short window of each side
// from that position. ok is false only when the two word sequences are
// identical, which the containment check has already ruled out by the time
// this runs. The compare starts at word 1 of each: when the found body is a
// different text entirely, the first words diverge and the report says so.
func firstWordDivergence(want, got string) (pos int, wantWin, gotWin string, ok bool) {
	w, f := strings.Fields(want), strings.Fields(got)
	i := 0
	for i < len(w) && i < len(f) && w[i] == f[i] {
		i++
	}
	if i >= len(w) && i >= len(f) {
		return 0, "", "", false
	}
	return i + 1, wordWindow(w, i), wordWindow(f, i), true
}

// checkGatesDisclaimer holds the root BUILD.md to the template's verbatim
// "What the gates do not verify" block. Only the root is checked: a manifest
// design's packets carry no disclaimer obligation.
func checkGatesDisclaimer(g *Gate, maskedRoot string) {
	body, ok := sectionBody(maskedRoot, "what the gates do not verify")
	if !ok {
		g.Errs = append(g.Errs, "BUILD.md has no 'What the gates do not verify' section; copy "+disclaimerSource+" verbatim, so a green check is never read as more than it is (only the root BUILD.md is checked; packets are not)")
		return
	}
	if !strings.Contains(collapseWS(body), collapseWS(gatesDisclaimerText)) {
		msg := "the 'What the gates do not verify' block in BUILD.md is not the template's text; the canonical wording is " + disclaimerSource
		if pos, wantWin, gotWin, diverged := firstWordDivergence(gatesDisclaimerText, body); diverged {
			msg += fmt.Sprintf("; first divergence at word %d: expected %q, found %q", pos, wantWin, gotWin)
		}
		msg += "; include the block verbatim (reflowed line breaks are fine, wording changes are not), and note that only the root BUILD.md is checked, packets are not"
		g.Errs = append(g.Errs, msg)
		return
	}
	g.Count("gates-disclaimer verbatim")
}

// checkDataDictionaryIdentity errors when more than one HEADING carrying the
// phrase "data dictionary" exists across the root and packets: the dictionary
// is the one canonical schema, and two copies are two schemas the moment one
// is edited. The check keys on heading titles alone, never on the rows below
// them, so a derived or per-packet slice clears it by being titled as one. An
// embed-marked copy is the other sanctioned exception, and it fits only a
// byte-identical copy (Ge holds its fidelity).
func checkDataDictionaryIdentity(g *Gate, docs []planNamedDoc) {
	var wheres []string
	for _, d := range docs {
		lines := strings.Split(d.text, "\n")
		for i, line := range lines {
			_, title := headingText(line)
			if title == "" || !strings.Contains(strings.ToLower(title), "data dictionary") {
				continue
			}
			if i > 0 && embedMarker.MatchString(lines[i-1]) {
				continue // a marked embed is the sanctioned copy; Ge holds it
			}
			wheres = append(wheres, d.name+":"+strconv.Itoa(i+1))
		}
	}
	switch {
	case len(wheres) > 1:
		g.Errs = append(g.Errs, strconv.Itoa(len(wheres))+" headings contain the phrase 'data dictionary' ("+strings.Join(wheres, ", ")+"); the dictionary is the one canonical schema, so keep the phrase on exactly one heading: retitle derived or per-packet slices (e.g. 'schema slice'), or put a machinery:embed marker on the line before a byte-identical copy (Ge then holds its fidelity)")
	case len(wheres) == 1:
		g.Count("data dictionary unique")
	}
}

// planNamedDoc is one build document with the name findings address it by.
type planNamedDoc struct {
	name string
	text string // fence-masked
}

// CheckBuildPlan implements Gb-plan.
func CheckBuildPlan(design string) *Gate {
	g := NewGate("Gb-plan  build plan structure")
	g.startOrder()
	has, probeErr := probeRegularFile(design, "BUILD.md")
	if probeErr != nil {
		g.Errs = append(g.Errs, probeErr.Error())
		return g
	}
	if !has {
		g.Errs = append(g.Errs, "no BUILD.md in the design; the build-plan gate was requested but Phase 4 never produced a build document (author BUILD.md, or drop gb from the gate list)")
		return g
	}
	text := readDesignFileOrErr(design, filepath.Join(design, "BUILD.md"), g)
	var packets []string
	indexFiles := 0
	if planMode(text) == "manifest" {
		var inventoryErr error
		packets, indexFiles, inventoryErr = planPackets(design)
		if inventoryErr != nil {
			g.Errs = append(g.Errs, inventoryErr.Error())
		}
	}
	docs := []planNamedDoc{{name: "BUILD.md", text: maskFences(text)}}
	for _, packet := range packets {
		docs = append(docs, planNamedDoc{name: filepath.Base(packet), text: maskFences(readDesignFileOrErr(design, packet, g))})
	}
	// A machine-less decomposed parent's manifest is a table of contents over
	// the children, not a buildable plan: the template sections and the
	// disclaimer live in the child BUILDs, exactly as the plans do.
	parentManifest := planMode(text) == "manifest" && len(packets) == 0 && pack.HasDecomposition(design)
	if !parentManifest {
		var masked []string
		for _, d := range docs {
			masked = append(masked, d.text)
		}
		checkTemplateSections(g, masked)
		checkGatesDisclaimer(g, docs[0].text)
	}
	checkDataDictionaryIdentity(g, docs)
	if planMode(text) == "manifest" {
		if indexFiles > 0 {
			g.CheckedExtra(fmt.Sprintf("%d index files exempt", indexFiles))
		}
		if len(packets) == 0 {
			if pack.HasDecomposition(design) {
				// the checkout-split parent shape: the manifest fixes the
				// shared artifacts and the children carry the buildable
				// plans; the zero must stay visible in every run
				g.CheckedExtra("0 local plans (decomposed parent; the children carry the plans)")
				return g
			}
		}
		checkPlanDoc(g, "BUILD.md", text, planOracleIDs(design, g))
		checkManifestPackets(g, design, text, packets)
		return g
	}
	checkPlanDoc(g, "BUILD.md", text, planOracleIDs(design, g))
	return g
}

// planMilestone is one parsed milestone block of a build-plan document. Gb
// holds its shape; Ga reads the same parse to decide what a closed milestone
// owes in acceptance evidence.
type planMilestone struct {
	numRaw string // the number exactly as written (M01 stays M01 in messages)
	num    int    // the parsed number; numOK reports whether it parsed
	numOK  bool
	title  string
	block  string
	status string // the lowercased Status: value; "" when the block states none
	dod    int    // offset of the DoD token in block; -1 when the block states none
}

// closed reports whether the milestone block marks the milestone closed.
func (m planMilestone) closed() bool { return m.status == "closed" }

// dodText returns the milestone's definition of done: the block from the DoD
// token to its end ("" when the block states none). Pre-DoD prose never
// counts, in Gb's skeleton citation and in Ga's id coverage alike.
func (m planMilestone) dodText() string {
	if m.dod < 0 {
		return ""
	}
	return m.block[m.dod:]
}

// parsePlanMilestones parses the milestone blocks of a Build plan section
// body (already fence-masked). A milestone block ends at the next milestone
// marker, the next heading of ANY level (a trailing "### Notes" subsection is
// not part of the last milestone), or the section end, whichever comes first.
func parsePlanMilestones(body string) []planMilestone {
	matches := milestoneRe.FindAllStringSubmatchIndex(body, -1)
	headings := headingOffsets(body)
	var ms []planMilestone
	for i, m := range matches {
		end := len(body)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		for _, h := range headings {
			if h > m[0] && h < end {
				end = h
				break
			}
		}
		block := body[m[0]:end]
		pm := planMilestone{
			numRaw: body[m[2]:m[3]],
			title:  strings.TrimSpace(body[m[4]:m[5]]),
			block:  block,
			dod:    dodIndex(block),
		}
		// \d+ can only overflow; an absurd number is its own review finding
		if v, err := strconv.Atoi(pm.numRaw); err == nil {
			pm.num, pm.numOK = v, true
		}
		if sm := milestoneStatusRe.FindStringSubmatch(block); sm != nil {
			pm.status = strings.ToLower(sm[1])
		}
		ms = append(ms, pm)
	}
	return ms
}

// planMilestonesOf parses one plan-bearing document: its milestones and
// whether the document declares a checkable plan at all. ok is false when the
// document has no Build plan section or waives it (a waived section declares
// no milestones by definition).
func planMilestonesOf(text string) (ms []planMilestone, ok bool) {
	body, found := buildPlanSection(maskFences(text))
	if !found {
		return nil, false
	}
	if first := firstNonBlankLine(body); strings.HasPrefix(strings.ToUpper(first), "N/A") {
		return nil, false
	}
	return parsePlanMilestones(body), true
}

// planDoc is one plan-bearing document of a design with its milestones.
type planDoc struct {
	name       string
	milestones []planMilestone
}

// planDocuments returns the one plan-bearing document of the design. In
// manifest mode BUILD/*.md files are execution packets, never competing
// milestone authorities; acceptance remains keyed to the root manifest.
func planDocuments(design string, g *Gate) []planDoc {
	if !HasBuildDoc(design) {
		return nil
	}
	text := readDesignFileOrErr(design, filepath.Join(design, "BUILD.md"), g)
	var out []planDoc
	if ms, ok := planMilestonesOf(text); ok {
		out = append(out, planDoc{name: "BUILD.md", milestones: ms})
	}
	return out
}

// checkPlanDoc runs the structural checks on one build-plan document (the
// root BUILD.md or a manifest shard). oracleIDs is the committed-oracle id
// corpus for the skeleton-citation check.
func checkPlanDoc(g *Gate, name, text string, oracleIDs []string) {
	g.Count("plans")
	// every scan below runs on the fence-masked text: fence content is
	// neither headings nor milestones nor DoD lines nor citations
	text = maskFences(text)
	sections := buildPlanSections(text)
	if len(sections) == 0 {
		g.Errs = append(g.Errs, name+": no Build plan section (need a ## or ### heading carrying the phrase 'Build plan'; a section number and trailing decoration are fine)")
		return
	}
	if len(sections) > 1 {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: %d 'Build plan' sections; the plan lives in exactly one section so every milestone is held (merge them)", name, len(sections)))
	}
	body := sections[0]
	// section waiver: ONLY the documented literal form "N/A - <reason>" as
	// the first non-blank line waives the structural checks (NG-6: any line
	// merely starting with N/A once waived everything). A first line that
	// looks like a waiver attempt but misses the form fails loudly instead
	// of silently un-waiving into confusing structural errors.
	if first := firstNonBlankLine(body); strings.HasPrefix(strings.ToUpper(first), "N/A") {
		switch {
		case planWaiverRe.MatchString(first):
			g.Count("waived plans")
		case strings.TrimLeft(first[len("N/A"):], " \t-:.,") == "":
			g.Errs = append(g.Errs, name+": the build plan is waived with a bare N/A; a waiver needs a reason (N/A - <reason>)")
		default:
			g.Errs = append(g.Errs, name+": the build plan starts with N/A but not in the waiver form; a waiver is the literal 'N/A - <reason>' (or write the plan)")
		}
		return
	}

	ms := parsePlanMilestones(body)
	if len(ms) == 0 {
		g.Errs = append(g.Errs, name+": the build plan has no milestone markers (**M<n> - <title>**); without milestones there is no walking skeleton and no DoD to hold")
		return
	}
	g.Count("milestones", len(ms))
	// numbers compare numerically: M1 and M01 are the same milestone (NG-8)
	numCount := map[int]int{}
	var numOrder []int
	for _, m := range ms {
		if !m.numOK {
			continue
		}
		if numCount[m.num] == 0 {
			numOrder = append(numOrder, m.num)
		}
		numCount[m.num]++
	}
	for _, v := range numOrder {
		if c := numCount[v]; c > 1 {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%d is declared %d times; milestone numbers must be unique", name, v, c))
		}
	}

	for _, m := range ms {
		if m.dod >= 0 {
			g.Count("DoD-bearing milestones")
		} else {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s (%s) states no definition of done; add 'DoD:' to its block", name, m.numRaw, m.title))
		}
		// the status line is optional, but a typo in it must not read as
		// "not closed" and silently disarm Ga-accept
		switch m.status {
		case "", "open", "closed":
		default:
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s (%s) has an unrecognized status %s; the milestone status line is 'Status: open' or 'Status: closed' (omit it for open)", name, m.numRaw, m.title, ir.Repr(m.status)))
		}
		if m.closed() {
			g.Count("closed milestones")
		}
	}

	skeletonFirst := strings.Contains(strings.ToLower(ms[0].title), "walking skeleton")
	skeletonWaived := skeletonWaiverRe.MatchString(body)
	switch {
	case skeletonFirst:
	case skeletonWaived:
		g.Count("skeleton waivers")
	default:
		g.Errs = append(g.Errs, fmt.Sprintf("%s: the first milestone (M%s - %s) is not the walking skeleton; plan the skeleton first, or waive it with 'walking skeleton: N/A - <reason>'", name, ms[0].numRaw, ms[0].title))
	}

	// the skeleton is the pattern template every later milestone copies, so
	// its block names the NFR-record mechanisms it instantiates on an `NFR:`
	// line (error envelope, config registration, observability hooks, auth
	// posture; 'NFR: none - <reason>' when the record leaves nothing to
	// instantiate). The template stated this requirement and, until now,
	// stated that Gb does not check it.
	if skeletonFirst {
		if m := skeletonNFRRe.FindStringSubmatch(ms[0].block); m == nil {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: the walking-skeleton milestone (M%s) has no 'NFR:' line; name the NFR-record mechanisms the skeleton instantiates (or 'NFR: none - <reason>')", name, ms[0].numRaw))
		} else if strings.TrimSpace(m[1]) == "" {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: the walking-skeleton milestone (M%s) has an empty 'NFR:' line; name the mechanisms, or 'NFR: none - <reason>'", name, ms[0].numRaw))
		} else {
			g.Count("skeleton NFR lines")
		}
	}

	switch {
	case !skeletonFirst:
		// waived: nothing to cite. Not waived: the skeleton-first error
		// above already blocks, and there is no skeleton block to look in.
		if skeletonWaived {
			g.CheckedExtra("skeleton citation skipped (skeleton waived)")
		}
	case len(oracleIDs) == 0:
		g.CheckedExtra("skeleton citation skipped (no committed oracles; G3/Gx own that absence)")
	default:
		// the DoD is what cites the id: pre-DoD prose does not count, so the
		// search runs from the first DoD token to the block end
		corpus := ms[0].dodText()
		found := 0
		for _, id := range oracleIDs {
			if idTokenIn(id, corpus) {
				found++
			}
		}
		if found == 0 {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: the walking-skeleton milestone (M%s) cites no committed oracle id at or after its DoD (no test id or stable id from machines/*.oracle.md appears whole-token there); the skeleton's DoD must name the transitions it proves", name, ms[0].numRaw))
		} else {
			g.Count("skeleton citations", found)
		}
	}
}

// dodIndex returns the offset of the first DoD token in block: the literal
// case-sensitive "DoD:", or the case-insensitive phrase "definition of done"
// immediately followed by a colon, whichever comes first; -1 when the block
// states no definition of done. Deliberately NOT anchored to line starts: the
// bundled examples legitimately state "DoD:" mid-paragraph after a sentence.
func dodIndex(block string) int {
	i := strings.Index(block, "DoD:")
	j := -1
	if loc := dodPhraseRe.FindStringIndex(block); loc != nil {
		j = loc[0]
	}
	switch {
	case i < 0:
		return j
	case j < 0 || i < j:
		return i
	default:
		return j
	}
}

// headingOffsets returns the byte offset of every ATX heading line in text,
// ascending.
func headingOffsets(text string) []int {
	var offs []int
	off := 0
	for _, line := range strings.Split(text, "\n") {
		if l, _ := headingText(line); l > 0 {
			offs = append(offs, off)
		}
		off += len(line) + 1
	}
	return offs
}

// firstNonBlankLine returns the first non-blank line of text, trimmed; ""
// when every line is blank.
func firstNonBlankLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}
