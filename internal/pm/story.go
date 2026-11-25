package pm

import (
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
)

// StoryTier returns the tier of a story: the highest tier among its ids,
// else its tier label, else routine (section 15.1). areas are the areas
// giving the tier.
func StoryTier(d *design.Design, is *backlog.Issue, ids []string) (tier string, areas []string) {
	if len(ids) > 0 {
		return evidence.HighestTier(d, ids)
	}
	if l := LabelTier(is); l != "" {
		return l, nil
	}
	return design.Routine, nil
}

// LabelTier returns the tier of the story's tier:<tier> label, or "".
func LabelTier(is *backlog.Issue) string {
	best := ""
	for _, l := range is.Labels {
		t, ok := strings.CutPrefix(l, "tier:")
		if !ok || (t != design.Routine && t != design.Consequential && t != design.Critical) {
			continue
		}
		if best == "" || design.TierRank(t) > design.TierRank(best) {
			best = t
		}
	}
	return best
}

// Route returns the routing text of a tier (section 15.5).
func Route(d *design.Design, tier string) string { return d.P.Cfg.Routing[tier] }

// ObligationIDs keeps the ids that are obligations, sorted.
func ObligationIDs(d *design.Design, ids []string) []string {
	var res []string
	for _, id := range ids {
		if d.Obligation(id) != nil {
			res = append(res, id)
		}
	}
	sort.Strings(res)
	return res
}

// CriticalAreas returns the names of the critical areas the ids belong to,
// an fv id naming its own area, sorted.
func CriticalAreas(d *design.Design, ids []string) []string {
	set := map[string]bool{}
	for _, id := range ids {
		_, in := d.Tier(id)
		for _, a := range in {
			if a.Tier == design.Critical {
				set[a.Name] = true
			}
		}
		if name, ok := strings.CutPrefix(id, "fv:"); ok {
			for _, a := range d.Areas {
				if a.Name == name && a.Tier == design.Critical {
					set[name] = true
				}
			}
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]bool) []string {
	var res []string
	for k := range set {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

func setOf(list []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range list {
		m[s] = true
	}
	return m
}
