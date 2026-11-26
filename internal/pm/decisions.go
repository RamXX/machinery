package pm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
)

// Decision is a backlog item that carries a machinery:decision line
// (section 15.8).
type Decision struct {
	ID         string
	Title      string
	Actor      string
	IDs        []string // the ids and id patterns it governs, sorted
	Vetoable   bool
	Confirmed  []string
	Supersedes []string
	By         string // the decision that supersedes it, "" while current
}

// Decisions reads every decision of the backlog, sorted by id, with the
// decision that supersedes each.
func Decisions(issues []backlog.Issue) []Decision {
	var res []Decision
	by := map[string]string{}
	for i := range issues {
		t := backlog.ParseTags(&issues[i])
		if !t.Decision {
			continue
		}
		res = append(res, Decision{ID: issues[i].ID, Title: issues[i].Title, Actor: t.DecidedBy, IDs: t.IDs,
			Vetoable: t.Vetoable, Confirmed: t.Confirmed, Supersedes: t.Supersedes})
		for _, old := range t.Supersedes {
			if by[old] == "" || issues[i].ID < by[old] {
				by[old] = issues[i].ID
			}
		}
	}
	for i := range res {
		res[i].By = by[res[i].ID]
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

// CurrentDecisions keeps the decisions no other decision supersedes.
func CurrentDecisions(issues []backlog.Issue) []Decision {
	var res []Decision
	for _, d := range Decisions(issues) {
		if d.By == "" {
			res = append(res, d)
		}
	}
	return res
}

// IsDecision reports whether an item is a decision.
func IsDecision(is *backlog.Issue) bool { return backlog.ParseTags(is).Decision }

// OpenVeto reports whether the owner may still overturn the decision: it is
// vetoable and nobody confirmed it.
func (d Decision) OpenVeto() bool { return d.Vetoable && len(d.Confirmed) == 0 }

// Governs reports whether the decision governs one of the ids or patterns.
func (d Decision) Governs(ids []string) bool {
	for _, mine := range d.IDs {
		for _, id := range ids {
			if Overlap(mine, id) {
				return true
			}
		}
	}
	return false
}

// Overlap reports whether two ids or id patterns name a common id: equal
// ids, an id with the prefix of a pattern, or two patterns one prefix of
// which contains the other.
func Overlap(a, b string) bool {
	pa, wa := strings.CutSuffix(a, "*")
	pb, wb := strings.CutSuffix(b, "*")
	switch {
	case wa && wb:
		return strings.HasPrefix(pa, pb) || strings.HasPrefix(pb, pa)
	case wa:
		return strings.HasPrefix(b, pa)
	case wb:
		return strings.HasPrefix(a, pb)
	}
	return a == b
}

// DecisionDescription renders the description of a new decision.
func DecisionDescription(why, actor string, ids []string, supersedes string, vetoable bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Why:\n%s\n\nmachinery:decision %s\n", OneLine(why), actor)
	if len(ids) > 0 {
		s := append([]string(nil), ids...)
		sort.Strings(s)
		fmt.Fprintf(&b, "machinery:ids %s\n", strings.Join(s, " "))
	}
	if supersedes != "" {
		fmt.Fprintf(&b, "machinery:supersedes %s\n", supersedes)
	}
	if vetoable {
		b.WriteString("machinery:vetoable\n")
	}
	return b.String()
}
