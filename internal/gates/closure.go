// Adoption-closure table (G2, opt-in by table presence). A technology choice
// is a closure, not a node: adopting X adopts X's stateful backends,
// sidecars, operators, credentials, and egress. DISCOVERING the closure is
// judgment forever (the universe is open); CARRYING a discovered member is
// not, because a declared member owes exactly what every dependency owes: a
// declaration and a mitigation row. The table gives the closure an artifact:
// a header naming technology and closure columns, one row per adopted
// technology. An optional scorecard column holds the dated OpenSSF Scorecard
// evidence the skill asks for, in a checkable shape.

package gates

import (
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	// noClosureWaiverRe waives one technology's closure: it genuinely brings
	// nothing with it. The reason is mandatory, as with every waiver.
	noClosureWaiverRe = regexp.MustCompile(`\(no closure:\s*([^)]*)\)`)
	// scorecardCellRe is the checkable evidence shape: a score with the date
	// it was read ("7.5 (2026-08-01)"), or an explained n/a.
	scorecardCellRe = regexp.MustCompile(`^(?:\d+(?:\.\d+)?\s*\(\d{4}-\d{2}-\d{2}\)|(?i:n/a)\s*-\s*\S.*)$`)
	// licenseCellRe is the checkable license shape: an SPDX id or expression
	// ("Apache-2.0", "GPL-3.0-or-later OR MIT"), or an explained n/a. The id
	// grammar is deliberately loose (the SPDX list is open); what is held is
	// that the cell is an identifier-shaped answer or a reasoned waiver, never
	// prose and never blank.
	licenseCellRe = regexp.MustCompile(`^(?:[A-Za-z0-9][A-Za-z0-9.+-]*(?:\s+(?:OR|AND|WITH)\s+[A-Za-z0-9][A-Za-z0-9.+-]*)*|(?i:n/a)\s*-\s*\S.*)$`)
)

// checkAdoptionClosure holds every adoption-closure table in ARCHITECTURE.md:
// each closure member resolves to a declared element or external AND has a
// mitigation row; required and covered are G2's already-computed external map
// and mitigation-row token set.
func checkAdoptionClosure(g *Gate, text string, els map[string]dslEl, required map[string]string, covered map[string]bool) {
	for _, tbl := range ir.ParseMdTables(text) {
		ti := colContaining(tbl.Header, "technology")
		ci := colContaining(tbl.Header, "closure")
		if ti < 0 || ci < 0 {
			continue
		}
		si := colContaining(tbl.Header, "scorecard")
		li := colContaining(tbl.Header, "license")
		for _, r := range tbl.Rows {
			if len(r) == 0 {
				continue
			}
			g.Count("adoption rows")
			tech := strings.TrimSpace(strings.ReplaceAll(r[min(ti, len(r)-1)], "`", ""))
			if tech == "" {
				tech = "(unnamed technology)"
			}
			cell := ""
			if ci < len(r) {
				cell = r[ci]
			}
			if m := noClosureWaiverRe.FindStringSubmatch(cell); m != nil {
				if strings.TrimSpace(m[1]) == "" {
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" waives its closure with no reason; write '(no closure: <reason>)'")
				} else {
					g.Count("closures waived")
				}
			} else {
				members := mitTokRe.FindAllStringSubmatch(parenAnnotationRe.ReplaceAllString(cell, " "), -1)
				if len(members) == 0 {
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" names no closure member in backticks and no '(no closure: <reason>)' waiver; enumerate what the technology brings with it")
				}
				for _, m := range members {
					tok := m[1]
					_, isEl := els[tok]
					alt, isExt := required[tok]
					if !isEl && !isExt {
						g.Errs = append(g.Errs, "closure member `"+tok+"` of "+ir.Repr(tech)+" is neither a workspace.dsl element nor a declared external; a closure member is a first-class dependency and must be declared")
						continue
					}
					if covered[tok] || (alt != "" && covered[alt]) {
						g.Count("closure members with mitigation rows")
					} else {
						g.Errs = append(g.Errs, "closure member `"+tok+"` of "+ir.Repr(tech)+" has no mitigation row; every closure member gets the full dependency treatment")
					}
				}
			}
			if si >= 0 {
				score := ""
				if si < len(r) {
					score = strings.TrimSpace(r[si])
				}
				switch {
				case score == "":
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" leaves scorecard empty; record the dated OpenSSF Scorecard score ('7.5 (2026-08-01)') or 'n/a - <reason>'")
				case scorecardCellRe.MatchString(score):
					g.Count("scorecard entries")
				default:
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" scorecard "+ir.Repr(score)+" is neither '<score> (YYYY-MM-DD)' nor 'n/a - <reason>'")
				}
			}
			// the license check the skill asks for per closure member, in a
			// checkable shape; opt-in by column presence, like scorecard
			if li >= 0 {
				lic := ""
				if li < len(r) {
					lic = strings.TrimSpace(strings.ReplaceAll(r[li], "`", ""))
				}
				switch {
				case lic == "":
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" leaves license empty; record the SPDX id ('Apache-2.0') or 'n/a - <reason>'")
				case licenseCellRe.MatchString(lic):
					g.Count("license entries")
				default:
					g.Errs = append(g.Errs, "adoption row "+ir.Repr(tech)+" license "+ir.Repr(lic)+" is neither an SPDX id/expression nor 'n/a - <reason>'")
				}
			}
		}
	}
}
