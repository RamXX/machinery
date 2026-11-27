package pm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/v2/internal/backlog"
	"github.com/RamXX/machinery/v2/internal/gitx"
)

// ObservationKinds are the kinds of an observation (section 13).
var ObservationKinds = []string{"bug", "security", "debt", "perf", "doc"}

// ParseObservation splits "<kind>: <text>"; ok is false when the kind is not
// one of ObservationKinds or the text is empty.
func ParseObservation(v string) (kind, text string, ok bool) {
	kind, text, found := strings.Cut(v, ":")
	kind, text = strings.TrimSpace(kind), strings.TrimSpace(text)
	if !found || text == "" {
		return "", "", false
	}
	for _, k := range ObservationKinds {
		if k == kind {
			return kind, text, true
		}
	}
	return "", "", false
}

// Observed is an observation line of a backlog item that is not filed.
type Observed struct {
	Item  backlog.Issue
	Kind  string
	Text  string
	Filed string // the item it was filed as, once filed
}

// Unfiled lists the observations of every item that no
// machinery:observation-filed line of the same item marks, sorted by item id
// and then in line order; the same kind and text twice on an item count once
// (section 15.8).
func Unfiled(issues []backlog.Issue) []Observed {
	var res []Observed
	sorted := append([]backlog.Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, is := range sorted {
		t := backlog.ParseTags(&is)
		done := map[string]bool{}
		for _, f := range t.Filed {
			done[f.Kind+"\x00"+f.Text] = true
		}
		for _, o := range t.Observations {
			key := o.Kind + "\x00" + o.Text
			if done[key] {
				continue
			}
			done[key] = true
			res = append(res, Observed{Item: is, Kind: o.Kind, Text: o.Text})
		}
	}
	return res
}

// FileObservation creates the backlog item of one observation and marks the
// line on the item it was seen in.
func FileObservation(c *backlog.Client, o *Observed) error {
	typ := "task"
	if o.Kind == "bug" || o.Kind == "security" {
		typ = "bug"
	}
	id, err := c.CreateItem(backlog.Item{
		Title:       o.Text,
		Type:        typ,
		Description: fmt.Sprintf("Observed in %s (%s): %s: %s", o.Item.ID, OneLine(o.Item.Title), o.Kind, o.Text),
		Labels:      []string{"observation:" + o.Kind},
		Deps:        []string{"discovered-from:" + o.Item.ID},
	})
	if err != nil {
		return err
	}
	o.Filed = id
	return c.Note(o.Item.ID, fmt.Sprintf("machinery:observation-filed %s %s %s", id, o.Kind, o.Text))
}

// LearningGroup holds the learning lines of the closed stories of one intake
// ("" for the stories outside an intake).
type LearningGroup struct {
	Intake string
	Title  string
	Lines  [][2]string // story id, learning
}

// Learnings groups the learning lines of the closed stories by intake: the
// intakes sorted by id, the stories outside an intake last (section 15.8).
// With since set (a full sha), a story counts only when one of its proof
// commits is neither since nor an ancestor of it. stories counts the stories
// that gave a line.
func Learnings(root string, issues []backlog.Issue, since string) (groups []LearningGroup, stories int) {
	byID := map[string]*backlog.Issue{}
	for i := range issues {
		byID[issues[i].ID] = &issues[i]
	}
	at := map[string]int{}
	sorted := append([]backlog.Issue(nil), issues...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	for _, is := range sorted {
		if is.Status != "closed" {
			continue
		}
		t := backlog.ParseTags(&is)
		if len(t.Learnings) == 0 || (since != "" && !provenAfter(root, t.Proofs, since)) {
			continue
		}
		in := storyIntake(&is, t, byID)
		i, ok := at[in]
		if !ok {
			g := LearningGroup{Intake: in}
			if it := byID[in]; it != nil {
				g.Title = it.Title
			}
			groups = append(groups, g)
			i = len(groups) - 1
			at[in] = i
		}
		for _, l := range t.Learnings {
			groups[i].Lines = append(groups[i].Lines, [2]string{is.ID, l})
		}
		stories++
	}
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i].Intake, groups[j].Intake
		if (a == "") != (b == "") {
			return b == ""
		}
		return a < b
	})
	return groups, stories
}

// CountLearnings counts the lines of the groups.
func CountLearnings(groups []LearningGroup) int {
	n := 0
	for _, g := range groups {
		n += len(g.Lines)
	}
	return n
}

// provenAfter reports whether one of the proof commits is neither since nor
// an ancestor of it; a commit this repository does not have counts.
func provenAfter(root string, proofs []backlog.Proof, since string) bool {
	for _, p := range proofs {
		if p.Commit != since && !gitx.IsAncestor(root, p.Commit, since) {
			return true
		}
	}
	return false
}

// storyIntake is the intake a story belongs to: the one its machinery:intake
// line names, else its parent when that is an intake item.
func storyIntake(is *backlog.Issue, t backlog.Tags, byID map[string]*backlog.Issue) string {
	if t.IntakeOf != "" {
		return t.IntakeOf
	}
	if p := byID[is.Parent]; p != nil && backlog.ParseTags(p).Intake {
		return p.ID
	}
	return ""
}

// LastPlanned is the newest commit of a machinery:intake-planned line of any
// intake item that this repository has, or "".
func LastPlanned(root string, issues []backlog.Issue) string {
	seen := map[string]bool{}
	var shas []string
	for i := range issues {
		t := backlog.ParseTags(&issues[i])
		if !t.Intake {
			continue
		}
		for _, m := range t.Planned {
			if !seen[m.Commit] && gitx.Exists(root, m.Commit) {
				seen[m.Commit] = true
				shas = append(shas, m.Commit)
			}
		}
	}
	return gitx.Newest(root, shas)
}

// LearningsReminder is the line intake plan prints when closed stories carry
// learnings the planner has not read since the last plan (section 15.7), or "".
func LearningsReminder(root string, issues []backlog.Issue) string {
	since := LastPlanned(root, issues)
	groups, _ := Learnings(root, issues, since)
	n := CountLearnings(groups)
	switch {
	case n == 0:
		return ""
	case since == "":
		return fmt.Sprintf("learnings: %d unread; machinery learnings", n)
	}
	short := gitx.Short(since)
	return fmt.Sprintf("learnings: %d unread since %s; machinery learnings --since %s", n, short, short)
}
