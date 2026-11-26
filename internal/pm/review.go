package pm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/report"
)

// The kinds of review (section 15.10).
const (
	ReviewBacklog = "backlog"
	ReviewDesign  = "design"
	ReviewIntake  = "intake"
)

// ReviewLabel is the label of the review item that holds the backlog reviews.
const ReviewLabel = "machinery-review"

// Review verdicts.
const (
	Approved = "approved"
	Rejected = "rejected"
)

// Who is a context: an actor, with the session and the agent when recorded.
type Who struct {
	Actor   string
	Session string
	Agent   string
}

// Part renders the session and agent words of a tag line.
func (w Who) Part() string { return backlog.Who{Session: w.Session, Agent: w.Agent}.Part() }

// ReviewLine renders a machinery:review line.
func ReviewLine(kind, verdict, commit, contract string, me Who, stories, issues []string) string {
	line := fmt.Sprintf("machinery:review %s %s commit=%s contract=%s actor=%s%s", kind, verdict, commit, contract,
		backlog.Value(me.Actor), me.Part())
	if len(stories) > 0 {
		s := append([]string(nil), stories...)
		sort.Strings(s)
		line += " stories=" + strings.Join(s, ",")
	}
	for i, is := range issues {
		line += fmt.Sprintf(" issue%d=%s", i+1, backlog.Quote(OneLine(is)))
	}
	return line
}

// ReviewsOf returns the review lines of a kind on the items, the items in id
// order and the lines in their order.
func ReviewsOf(issues []backlog.Issue, kind string) []backlog.Review {
	items := append([]backlog.Issue(nil), issues...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	var res []backlog.Review
	for i := range items {
		for _, r := range backlog.ParseTags(&items[i]).Reviews {
			if r.Kind == kind {
				res = append(res, r)
			}
		}
	}
	return res
}

// LatestFor returns the latest review for a contract hash, and for a backlog
// review the same wave-1 ids; nil when there is none.
func LatestFor(reviews []backlog.Review, contract string, stories []string, withStories bool) *backlog.Review {
	want := append([]string(nil), stories...)
	sort.Strings(want)
	var last *backlog.Review
	for i := range reviews {
		r := &reviews[i]
		if r.Contract != contract {
			continue
		}
		if withStories && strings.Join(r.Stories, ",") != strings.Join(want, ",") {
			continue
		}
		last = r
	}
	return last
}

// SameContext compares a reviewer with a planner (section 15.10). same is the
// reason they are one context; warn is set when independence rests on the
// actor names alone.
func SameContext(r, p Who) (same, warn string) {
	which := func(rHas, pHas bool) string {
		switch {
		case !rHas && !pHas:
			return "the review and the plan"
		case !rHas:
			return "the review"
		}
		return "the plan"
	}
	if r.Session != "" && p.Session != "" {
		if r.Session != p.Session {
			return "", ""
		}
		if r.Agent != "" && p.Agent != "" {
			if r.Agent == p.Agent {
				return fmt.Sprintf("same session %s and agent %s", r.Session, r.Agent), ""
			}
			return "", ""
		}
		if r.Actor == p.Actor {
			return fmt.Sprintf("same actor %s in session %s, and no agent tells them apart", r.Actor, r.Session), ""
		}
		return "", fmt.Sprintf("reviewer and planner share session %s and no agent is recorded for %s: independence rests on the actor name",
			r.Session, which(r.Agent != "", p.Agent != ""))
	}
	if r.Actor == p.Actor {
		return fmt.Sprintf("same actor %s, and no session tells them apart", r.Actor), ""
	}
	return "", fmt.Sprintf("no session recorded for %s: independence rests on the actor name", which(r.Session != "", p.Session != ""))
}

// Compare compares a reviewer with every planner: the first reason they are
// one context, and the warnings, sorted and once each.
func Compare(r Who, planners []Who) (same string, warns []string) {
	seen := map[string]bool{}
	for _, p := range planners {
		s, w := SameContext(r, p)
		if s != "" && same == "" {
			same = s
		}
		if w != "" && !seen[w] {
			seen[w] = true
			warns = append(warns, w)
		}
	}
	sort.Strings(warns)
	return same, warns
}

// IntakePlanners returns the planners of an intake: the planning lines of the
// item that record an actor.
func IntakePlanners(t backlog.Tags) []Who {
	var res []Who
	marks := append([]backlog.Mark(nil), t.Planned...)
	if t.Base != nil {
		marks = append([]backlog.Mark{*t.Base}, marks...)
	}
	for _, m := range marks {
		if m.Actor != "" {
			res = append(res, Who{m.Actor, m.Who.Session, m.Who.Agent})
		}
	}
	return res
}

// BacklogPlanners returns the planners of the intakes that have a story in
// wave 1.
func BacklogPlanners(issues []backlog.Issue, wave1 []string) []Who {
	in := setOf(wave1)
	intakes := map[string]bool{}
	for i := range issues {
		if in[issues[i].ID] {
			if id := IntakeOf(&issues[i]); id != "" {
				intakes[id] = true
			}
		}
	}
	var res []Who
	for i := range issues {
		if intakes[issues[i].ID] {
			res = append(res, IntakePlanners(backlog.ParseTags(&issues[i]))...)
		}
	}
	return res
}

// ReviewWho is the context of a review line.
func ReviewWho(r *backlog.Review) Who { return Who{r.Actor, r.Who.Session, r.Who.Agent} }

// HoldBacklogReview adds to the waves stage of a freeze what section 15.4
// asks of the backlog review: an approved latest review for the contract and
// the wave-1 ids, from another context than the planner who freezes.
func HoldBacklogReview(s *report.Stage, issues []backlog.Issue, c *Contract, wave1 []string, planner Who) {
	reviews := ReviewsOf(issues, ReviewBacklog)
	s.Count(len(reviews), "reviews")
	r := LatestFor(reviews, c.Hash, wave1, true)
	if r == nil || r.Verdict != Approved {
		s.Errorf("no approved review of this backlog at %s: the checker runs machinery review backlog", c.Short())
		return
	}
	same, warn := SameContext(ReviewWho(r), planner)
	if same != "" {
		s.Errorf("the review of this backlog at %s is the planner's: %s; the checker reviews in a fresh context", c.Short(), same)
	}
	if warn != "" {
		s.Warnf("%s", warn)
	}
}

// DesignReviewGate is what intake plan --apply asks of the design review of
// an intake (section 15.7): the refusal, else the warnings.
func DesignReviewGate(in *Intake, c *Contract, planner Who) (refusal string, warns []string) {
	r := LatestFor(designReviews(in), c.Hash, nil, false)
	if r == nil || r.Verdict != Approved {
		return fmt.Sprintf("no approved design review of %s at %s: the checker runs machinery review design %s", in.ID, c.Short(), in.ID), nil
	}
	same, warn := SameContext(ReviewWho(r), planner)
	if same != "" {
		return fmt.Sprintf("the design review of %s at %s is the planner's: %s; the checker reviews in a fresh context", in.ID, c.Short(), same), nil
	}
	if warn != "" {
		warns = append(warns, warn)
	}
	return "", warns
}

// DesignReviewApproved reports whether the latest design review of an intake
// for the contract is approved, and how many design reviews it has.
func DesignReviewApproved(in *Intake, contract string) (bool, int) {
	all := designReviews(in)
	r := LatestFor(all, contract, nil, false)
	return r != nil && r.Verdict == Approved, len(all)
}

func designReviews(in *Intake) []backlog.Review {
	var res []backlog.Review
	for _, r := range in.Tags.Reviews {
		if r.Kind == ReviewDesign {
			res = append(res, r)
		}
	}
	return res
}

// ReviewStatus is the review lines of machinery status (section 14): the
// latest backlog review, then the latest design and intake review of each
// open intake item.
func ReviewStatus(issues []backlog.Issue) []string {
	var res []string
	if all := ReviewsOf(issues, ReviewBacklog); len(all) > 0 {
		res = append(res, "review backlog: "+reviewSummary(all[len(all)-1]))
	}
	for _, in := range OpenIntakes(issues) {
		for _, kind := range []string{ReviewDesign, ReviewIntake} {
			var last *backlog.Review
			for i := range in.Tags.Reviews {
				if in.Tags.Reviews[i].Kind == kind {
					last = &in.Tags.Reviews[i]
				}
			}
			if last != nil {
				res = append(res, fmt.Sprintf("review %s %s: %s", kind, in.ID, reviewSummary(*last)))
			}
		}
	}
	return res
}

func reviewSummary(r backlog.Review) string {
	line := fmt.Sprintf("%s at %s by %s", r.Verdict, short12(r.Contract), r.Actor)
	if n := len(r.Issues); n > 0 {
		line += fmt.Sprintf(", %d issue(s)", n)
	}
	return line
}

// ReviewItem returns the item that holds the backlog reviews: the first, by
// id, with the review label; nil when there is none.
func ReviewItem(issues []backlog.Issue) *backlog.Issue {
	var best *backlog.Issue
	for i := range issues {
		if issues[i].HasLabel(ReviewLabel) && (best == nil || issues[i].ID < best.ID) {
			best = &issues[i]
		}
	}
	return best
}

// ParseIssue splits an issue of a rejected review into its rule, scope and
// instances; ok is false when the rule or the scope is empty.
func ParseIssue(s string) (rule, scope, instances string, ok bool) {
	parts := strings.SplitN(OneLine(s), ": ", 3)
	if len(parts) < 2 {
		return "", "", "", false
	}
	rule, scope = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if len(parts) == 3 {
		instances = strings.TrimSpace(parts[2])
	}
	return rule, scope, instances, rule != "" && scope != ""
}
