package pm

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/backlog"
	"github.com/RamXX/machinery/internal/delegate"
)

// IsQuestion reports whether an item is a question for the owner (section 15.9).
func IsQuestion(is *backlog.Issue) bool { return backlog.ParseTags(is).Question }

// QuestionDescription renders the description of a new question.
func QuestionDescription(actor string, options []string, recommend int, def string, about []string, amount, unit, story string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "machinery:question %s\n", actor)
	for i, o := range options {
		fmt.Fprintf(&b, "machinery:option %d %s\n", i+1, OneLine(o))
	}
	fmt.Fprintf(&b, "machinery:recommend %d\nmachinery:default %s\n", recommend, OneLine(def))
	if len(about) > 0 {
		fmt.Fprintf(&b, "machinery:about %s\n", strings.Join(about, " "))
	}
	if amount != "" {
		fmt.Fprintf(&b, "machinery:amount %s %s\n", amount, unit)
	}
	if story != "" {
		fmt.Fprintf(&b, "machinery:story %s\n", story)
	}
	return b.String()
}

// OwnerAnswer returns the first answer of the owner, or nil.
func OwnerAnswer(t backlog.Tags) *backlog.Answer {
	for i := range t.Answers {
		if !t.Answers[i].Delegated() {
			return &t.Answers[i]
		}
	}
	return nil
}

// DelegateDecision returns the first answer on the owner's behalf that
// decided (an option or a refusal), or nil.
func DelegateDecision(t backlog.Tags) *backlog.Answer {
	for i := range t.Answers {
		if a := t.Answers[i]; a.Delegated() && a.What != delegate.Hold {
			return &t.Answers[i]
		}
	}
	return nil
}

// lastAnswer returns the last answer, or nil.
func lastAnswer(t backlog.Tags) *backlog.Answer {
	if len(t.Answers) == 0 {
		return nil
	}
	return &t.Answers[len(t.Answers)-1]
}

// Held reports whether the last answer of a question is a hold.
func Held(t backlog.Tags) bool {
	a := lastAnswer(t)
	return a != nil && a.Delegated() && a.What == delegate.Hold
}

// WaitsFor says what an open question waits for (section 15.9).
func WaitsFor(t backlog.Tags) string {
	a := lastAnswer(t)
	switch {
	case a == nil:
		return "the owner"
	case !a.Delegated():
		return "nothing: answered by " + a.Actor
	case a.What == delegate.Hold:
		return fmt.Sprintf("the owner; held under clause %s: %s", a.Clause, a.Reason)
	case a.What == delegate.Refuse:
		return "the owner's confirmation of the refusal under clause " + a.Clause
	}
	return fmt.Sprintf("the owner's confirmation of option %s under clause %s", a.What, a.Clause)
}

// OpenQuestions lists the questions that are not closed, sorted by id.
func OpenQuestions(issues []backlog.Issue) []backlog.Issue {
	var res []backlog.Issue
	for i := range issues {
		if issues[i].Status != "closed" && IsQuestion(&issues[i]) {
			res = append(res, issues[i])
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

// Spent sums the amounts of the questions answered with an option on the
// owner's behalf under a clause, in a unit: the ledger of section 15.9.
func Spent(issues []backlog.Issue, clause, unit string) *big.Rat {
	sum := new(big.Rat)
	for i := range issues {
		t := backlog.ParseTags(&issues[i])
		if !t.Question || t.Unit != unit || t.Amount == "" {
			continue
		}
		amount, ok := delegate.ParseAmount(t.Amount)
		if !ok {
			continue
		}
		for _, a := range t.Answers {
			if a.Clause == clause && a.Option() > 0 {
				sum.Add(sum, amount)
			}
		}
	}
	return sum
}
