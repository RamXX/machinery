// NFR-record presence (G2). The skill requires the Phase 2 conversation to
// record security posture, capacity assumptions, and observability, "even when
// the answer is out of scope, recorded as such"; the c4 reference states the
// same and until now listed it as attested. The record's CONTENT is judgment
// forever, but its presence is a three-item closed list, so the shell is held
// here: an NFR section exists and each topic is mentioned inside it. "Out of
// scope" satisfies the check by construction, because writing it mentions the
// topic.

package gates

import (
	"strings"
)

// nfrTopics are the three obligations of the NFR record, matched
// case-insensitively inside the section body.
var nfrTopics = []string{"security", "capacity", "observability"}

// sectionBody returns the body of the first markdown ATX section whose
// heading text contains needle (case-insensitive), at any level; the body
// runs to the next heading of the same or a higher level, so subheadings stay
// inside the section.
func sectionBody(text, needle string) (string, bool) {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		level, title := headingText(line)
		if level == 0 || !strings.Contains(strings.ToLower(title), needle) {
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

// checkNFRRecord holds the NFR record's shell: a heading containing "NFR" (or
// "non-functional"), with each of the three topics mentioned in its body.
func checkNFRRecord(g *Gate, text string) {
	body, ok := sectionBody(text, "nfr")
	if !ok {
		body, ok = sectionBody(text, "non-functional")
	}
	if !ok {
		g.Errs = append(g.Errs, "ARCHITECTURE.md has no NFR record (a heading containing 'NFR' or 'non-functional'); record security posture, capacity assumptions, and observability, even as 'out of scope, recorded as such'")
		return
	}
	lower := strings.ToLower(body)
	for _, topic := range nfrTopics {
		if strings.Contains(lower, topic) {
			g.Count("nfr topics recorded")
		} else {
			g.Errs = append(g.Errs, "the NFR record never mentions "+topic+"; record the posture, or record that it is out of scope")
		}
	}
}
