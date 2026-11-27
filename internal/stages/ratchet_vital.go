package stages

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/design"
	"github.com/RamXX/machinery/v2/internal/report"
)

// ratchetVital is rule vital-weakened of stage ratchet (section 11.3): a
// vital area of the base may not state less now unless the risk file records
// the change under lowered:. It returns the number of vital areas of the base.
func ratchetVital(s *report.Stage, d *design.Design, base baseRisk, lowered []design.Lowering, inv baseInventory) int {
	weakened := func(area, what string) {
		for _, l := range lowered {
			if l.Area == area && strings.TrimSpace(l.What) == what {
				s.Warnf("ratchet: vital weakened: %s: %s (%s)", area, what, strings.TrimSpace(l.Reason))
				return
			}
		}
		s.Errorf("ratchet: vital weakened: %s: %s; record it under lowered: in the risk file with a reason", area, what)
	}
	now := map[string]design.Area{}
	for _, a := range d.Areas {
		if _, dup := now[a.Name]; !dup {
			now[a.Name] = a
		}
	}
	n := 0
	var vitals []design.Area
	for _, old := range base.areas {
		if old.Tier != design.Vital {
			continue
		}
		n++
		vitals = append(vitals, old)
		cur, ok := now[old.Name]
		switch {
		case !ok:
			weakened(old.Name, "area removed")
		case cur.Tier != design.Vital:
			// a lowering: rule tier-lowered holds it
		default:
			for _, what := range vitalLosses(old, cur) {
				weakened(old.Name, what)
			}
		}
	}
	if len(vitals) == 0 {
		return n
	}
	had := map[string]bool{}
	for _, dm := range base.notVital {
		had[strings.TrimSpace(dm.ID)] = true
	}
	ids := append(d.DeclaredIDs(), baseIDs(inv)...)
	sort.Strings(ids)
	for _, dm := range d.NotVital {
		pattern := strings.TrimSpace(dm.ID)
		if pattern == "" || had[pattern] {
			continue
		}
		had[pattern] = true
		if area := heldBy(vitals, pattern, ids); area != "" {
			weakened(area, "not_vital "+pattern+" added")
		}
	}
	return n
}

// heldBy names the first vital area of the base that held an id the pattern
// matches, or "".
func heldBy(vitals []design.Area, pattern string, ids []string) string {
	for _, a := range vitals {
		for _, id := range ids {
			if design.MatchID(pattern, id) && design.MatchAny(a.IDs, id) {
				return a.Name
			}
		}
	}
	return ""
}

// vitalLosses lists what a vital area of the base states that the same area
// no longer states, in the order of section 11.3.
func vitalLosses(old, cur design.Area) []string {
	var res []string
	props := map[string]design.Property{}
	for _, p := range cur.Property {
		props[p.Name] = p
	}
	for _, p := range old.Property {
		c, ok := props[p.Name]
		if !ok {
			res = append(res, fmt.Sprintf("property %s removed", p.Name))
			continue
		}
		if strings.TrimSpace(p.Statement) != strings.TrimSpace(c.Statement) {
			res = append(res, fmt.Sprintf("property %s: statement changed", p.Name))
		}
		if p.Kind != c.Kind {
			res = append(res, fmt.Sprintf("property %s: kind %s -> %s", p.Name, p.Kind, c.Kind))
		}
		for _, id := range missing(p.Over, c.Over) {
			res = append(res, fmt.Sprintf("property %s: over %s removed", p.Name, id))
		}
	}
	assumes := map[string]design.Assumption{}
	for _, a := range cur.Assumes {
		assumes[a.Name] = a
	}
	for _, a := range old.Assumes {
		c, ok := assumes[a.Name]
		if !ok {
			res = append(res, fmt.Sprintf("assumption %s removed", a.Name))
			continue
		}
		for _, id := range missing(a.Evidence, c.Evidence) {
			res = append(res, fmt.Sprintf("assumption %s: evidence %s removed", a.Name, id))
		}
	}
	if old.Core != nil {
		var paths, entries []string
		if cur.Core != nil {
			paths, entries = cur.Core.Paths, cur.Core.Entry
		}
		for _, g := range missing(old.Core.Paths, paths) {
			res = append(res, "core path "+g+" removed")
		}
		for _, e := range missing(old.Core.Entry, entries) {
			res = append(res, "core entry "+e+" removed")
		}
	}
	return res
}

// missing lists the items of was that are not in now, in the order of was.
func missing(was, now []string) []string {
	have := set(now)
	var res []string
	for _, x := range was {
		if !have[x] {
			res = append(res, x)
			have[x] = true
		}
	}
	return res
}
