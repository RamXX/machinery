// Gb-plan: the build-plan structure gate. The Build plan section of BUILD.md
// is what a coding agent actually schedules from, so its shape is held like
// any other artifact: the section exists, milestones are marked and uniquely
// numbered, the walking skeleton comes first, every milestone carries a
// definition of done, and the skeleton's DoD cites at least one committed
// oracle id. Gx owns the Mode line and the Toolchain / State-migration
// sections; Gb never re-checks them.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/pack"
)

// HasBuildDoc reports whether the design has a BUILD.md (Phase 4 produced a
// build document); Gb auto-activates on it.
func HasBuildDoc(design string) bool {
	fi, err := os.Stat(filepath.Join(design, "BUILD.md"))
	return err == nil && !fi.IsDir()
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
	// planHeadingNumRe strips the template's "N. " section-number prefix.
	planHeadingNumRe = regexp.MustCompile(`^\d+\.\s+`)
)

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

// maskFences blanks fenced code blocks (``` and ~~~, tracking open/close)
// line by line, preserving the line structure so byte offsets stay valid.
// Every Gb scan runs on the masked text: a bash "# comment" inside a fence is
// not a heading, and a fenced fake "**M9 - ...**" or "DoD:" is not plan
// structure. Neither ir.ParseMdTables nor Gx's BUILD scans are fence-aware;
// this helper is Gb's own.
func maskFences(text string) string {
	lines := strings.Split(text, "\n")
	var fence string // the active fence delimiter, "" when outside a fence
	for i, line := range lines {
		t := strings.TrimLeft(line, " \t")
		switch {
		case fence == "" && (strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")):
			fence = t[:3]
			lines[i] = ""
		case fence != "":
			if strings.HasPrefix(t, fence) {
				fence = ""
			}
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
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
// heading whose text, minus an optional "N. " prefix, is exactly "Build
// plan" (case-insensitive). The body runs to the next heading of the same or
// higher level, so subheadings stay inside the section.
func buildPlanSection(text string) (string, bool) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		level, title := headingText(line)
		if level != 2 && level != 3 {
			continue
		}
		if !strings.EqualFold(planHeadingNumRe.ReplaceAllString(title, ""), "Build plan") {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l, _ := headingText(lines[j]); l > 0 && l <= level {
				end = j
				break
			}
		}
		return strings.Join(lines[i+1:end], "\n"), true
	}
	return "", false
}

// planOracleIDs collects BOTH id columns (sequential test ids and stable
// ids) of every committed machines/*.oracle.md table, as read from the
// files: the id shapes are never assumed. The committed files are the source
// here; G3 separately holds them fresh against the machines.
func planOracleIDs(design string) []string {
	var ids []string
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.oracle.md") {
		testIDs, stableIDs := oracleTableIDs(readOrEmpty(path))
		ids = append(ids, testIDs...)
		ids = append(ids, stableIDs...)
	}
	return ids
}

// CheckBuildPlan implements Gb-plan.
func CheckBuildPlan(design string) *Gate {
	g := NewGate("Gb-plan  build plan structure")
	g.startOrder()
	if !HasBuildDoc(design) {
		g.Errs = append(g.Errs, "no BUILD.md in the design; the build-plan gate was requested but Phase 4 never produced a build document (author BUILD.md, or drop gb from the gate list)")
		return g
	}
	text := readOrEmpty(filepath.Join(design, "BUILD.md"))
	// Gx owns findings about the Mode line itself; an absent declaration
	// falls back to full so a pre-Gx draft still gets its plan checked.
	mode := "full"
	if m := modeRe.FindStringSubmatch(text); m != nil {
		mode = m[1]
	}
	if mode == "manifest" {
		// README.md and index.md under BUILD/ are navigation for humans, not
		// plan shards; they carry no plan obligation, and the exemption stays
		// visible in the checked line
		var shards []string
		indexFiles := 0
		for _, shard := range sortedGlobExt(filepath.Join(design, "BUILD"), ".md") {
			switch strings.ToLower(filepath.Base(shard)) {
			case "readme.md", "index.md":
				indexFiles++
			default:
				shards = append(shards, shard)
			}
		}
		if indexFiles > 0 {
			g.CheckedExtra(fmt.Sprintf("%d index files exempt", indexFiles))
		}
		if len(shards) == 0 {
			if pack.HasDecomposition(design) {
				// the checkout-split parent shape: the manifest fixes the
				// shared artifacts and the children carry the buildable
				// plans; the zero must stay visible in every run
				g.CheckedExtra("0 local plans (decomposed parent; the children carry the plans)")
				return g
			}
			g.Errs = append(g.Errs, "BUILD.md declares manifest mode but BUILD/ holds no shards and the design has no decomposition; a manifest with nothing behind it plans nothing")
			return g
		}
		// each shard gets the full check, skeleton citation included, against
		// the design-wide committed oracle ids: a shard plans work on the same
		// machines the root design committed. The manifest ROOT itself carries
		// no plan obligation.
		ids := planOracleIDs(design)
		for _, shard := range shards {
			checkPlanDoc(g, filepath.Base(shard), readOrEmpty(shard), ids)
		}
		return g
	}
	checkPlanDoc(g, "BUILD.md", text, planOracleIDs(design))
	return g
}

// checkPlanDoc runs the structural checks on one build-plan document (the
// root BUILD.md or a manifest shard). oracleIDs is the committed-oracle id
// corpus for the skeleton-citation check.
func checkPlanDoc(g *Gate, name, text string, oracleIDs []string) {
	g.Count("plans")
	// every scan below runs on the fence-masked text: fence content is
	// neither headings nor milestones nor DoD lines nor citations
	text = maskFences(text)
	body, ok := buildPlanSection(text)
	if !ok {
		g.Errs = append(g.Errs, name+": no Build plan section (need a ## or ### heading titled 'Build plan'; a numeric 'N. ' prefix is fine)")
		return
	}
	// section waiver: "N/A - <reason>" as the first non-blank line waives
	// the structural checks; a bare N/A explains nothing and stays an error
	if first := firstNonBlankLine(body); strings.HasPrefix(strings.ToUpper(first), "N/A") {
		if strings.TrimLeft(first[len("N/A"):], " \t-:.,") == "" {
			g.Errs = append(g.Errs, name+": the build plan is waived with a bare N/A; a waiver needs a reason (N/A - <reason>)")
		} else {
			g.Count("waived plans")
		}
		return
	}

	matches := milestoneRe.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		g.Errs = append(g.Errs, name+": the build plan has no milestone markers (**M<n> - <title>**); without milestones there is no walking skeleton and no DoD to hold")
		return
	}
	g.Count("milestones", len(matches))
	type milestone struct{ num, title, block string }
	var ms []milestone
	var nums []string
	// a milestone block ends at the next milestone marker, the next heading
	// of ANY level (a trailing "### Notes" subsection is not part of the last
	// milestone), or the section end, whichever comes first
	headings := headingOffsets(body)
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
		ms = append(ms, milestone{
			num:   body[m[2]:m[3]],
			title: strings.TrimSpace(body[m[4]:m[5]]),
			block: body[m[0]:end],
		})
		nums = append(nums, body[m[2]:m[3]])
	}
	for _, n := range uniqueDuplicates(nums) {
		if c := countStr(nums, n); c > 1 {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s is declared %d times; milestone numbers must be unique", name, n, c))
		}
	}

	for _, m := range ms {
		if dodIndex(m.block) >= 0 {
			g.Count("DoD-bearing milestones")
		} else {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: milestone M%s (%s) states no definition of done; add 'DoD:' to its block", name, m.num, m.title))
		}
	}

	skeletonFirst := strings.Contains(strings.ToLower(ms[0].title), "walking skeleton")
	skeletonWaived := skeletonWaiverRe.MatchString(body)
	switch {
	case skeletonFirst:
	case skeletonWaived:
		g.Count("skeleton waivers")
	default:
		g.Errs = append(g.Errs, fmt.Sprintf("%s: the first milestone (M%s - %s) is not the walking skeleton; plan the skeleton first, or waive it with 'walking skeleton: N/A - <reason>'", name, ms[0].num, ms[0].title))
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
		corpus := ""
		if d := dodIndex(ms[0].block); d >= 0 {
			corpus = ms[0].block[d:]
		}
		found := 0
		for _, id := range oracleIDs {
			if idTokenIn(id, corpus) {
				found++
			}
		}
		if found == 0 {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: the walking-skeleton milestone (M%s) cites no committed oracle id at or after its DoD (no test id or stable id from machines/*.oracle.md appears whole-token there); the skeleton's DoD must name the transitions it proves", name, ms[0].num))
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
