// Event-contract enumeration sources (G2). The skill requires every
// event-contract table to state where its rows were enumerated FROM
// (emit/publish call sites, broker or infra configuration, API specs): a
// table with no named enumeration source is a completeness claim with no
// evidence. Gs already holds the legacy surface ledger to exactly this rule
// (`source:` per class); this is the same rule applied to the forward event
// tables. A `machinery:embed` marker above the table satisfies it, because
// the marker names the source document the rows were carried from.

package gates

import (
	"regexp"
	"strconv"
	"strings"
)

// eventSourceNoteRe matches an enumeration-source note line: a "Source:" or
// "Sources:" label, or the phrase "enumerated from".
var eventSourceNoteRe = regexp.MustCompile(`(?i)(?:\bsources?\s*:\s*\S|\benumerated from\s+\S)`)

func validEventEmbed(line string) bool {
	m := embedMarker.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	attrs := map[string]string{}
	for _, a := range embedAttr.FindAllStringSubmatch(m[1], -1) {
		attrs[a[1]] = strings.TrimSpace(a[2])
	}
	if attrs["from"] == "" || attrs["table"] == "" || attrs["claims"] == "" {
		return false
	}
	for key := range attrs {
		if key != "from" && key != "table" && key != "where" && key != "claims" {
			return false
		}
	}
	return true
}

// eventSourceLookback is how many lines above the table header the note may
// sit (the prose-count check uses the same neighborhood idiom).
const eventSourceLookback = 5

// isTableSeparatorLine reports whether line i of lines is a markdown table
// separator (|---|---|).
func isTableSeparatorLine(lines []string, i int) bool {
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

// checkEventTableSources errors on every event-contract table (header naming
// producer, consumer, and delivery) with no enumeration-source evidence in
// the lines directly above it.
func checkEventTableSources(g *Gate, text string) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") || !isTableSeparatorLine(lines, i+1) {
			continue
		}
		hl := strings.ToLower(t)
		if !strings.Contains(hl, "producer") || !strings.Contains(hl, "consumer") || !strings.Contains(hl, "delivery") {
			continue
		}
		sourced := false
		for k := max(0, i-eventSourceLookback); k < i; k++ {
			if validEventEmbed(lines[k]) || eventSourceNoteRe.MatchString(lines[k]) {
				sourced = true
				break
			}
		}
		if sourced {
			g.Count("event tables with sources")
		} else {
			g.Errs = append(g.Errs, "ARCHITECTURE.md:"+strconv.Itoa(i+1)+": event-contract table names no enumeration source; state where the rows came from (emit/publish call sites, broker or infra configuration, API specs) in a 'Source:' note directly above the table, or carry the rows with a machinery:embed marker")
		}
	}
}
