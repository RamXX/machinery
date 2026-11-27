package xstate

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/v2/internal/machine"
)

// maxSteps bounds the eventless steps followed after a case fires.
const maxSteps = 64

// Expect is what the table says XState does for a case: the leaf it settles
// in and the named actions it executes on the way.
type Expect struct {
	State   string
	Actions []string
}

// Settle follows, from leaf, what XState runs after a step with only guard
// true: eventless (always) rows, innermost state first, then the onDone rows
// of a compound state whose final child was entered. It returns the leaf the
// machine rests in and the effects of the rows followed.
func Settle(m *machine.Machine, leaf, guard string) (string, []string) {
	t := m.Table
	enabled := func(kind, source string) *machine.Transition {
		for i := range t.Transitions {
			tr := &t.Transitions[i]
			if tr.Kind == kind && tr.Source == source && (tr.Guard == "" || tr.Guard == guard) {
				return tr
			}
		}
		return nil
	}
	var effects []string
	var done []string
	if p := machine.Parent(leaf); p != "" && isFinal(t, leaf) {
		done = append(done, p)
	}
	for step := 0; step < maxSteps; step++ {
		if !strings.Contains(leaf, ".") && isFinal(t, leaf) {
			break
		}
		var tr *machine.Transition
		for s := leaf; s != "" && tr == nil; s = machine.Parent(s) {
			tr = enabled("always", s)
		}
		for tr == nil && len(done) > 0 {
			p := done[0]
			done = done[1:]
			if p == leaf || machine.IsDescendant(leaf, p) {
				tr = enabled("stateDone", p)
			}
		}
		if tr == nil {
			break
		}
		effects = append(effects, m.Step(leaf, tr.Source, tr.Target, tr.Actions)...)
		if tr.Target != "" {
			leaf = m.Resolver.Leaf(tr.Target)
			if p := machine.Parent(leaf); p != "" && isFinal(t, leaf) {
				done = append(done, p)
			}
		}
	}
	return leaf, effects
}

func isFinal(t *machine.Table, path string) bool {
	for _, s := range t.States {
		if s.Path == path {
			return s.Kind == "final"
		}
	}
	return false
}

// ExpectCase is the table's answer for a transition row.
func ExpectCase(m *machine.Machine, tr machine.Transition) Expect {
	leaf := m.Resolver.Leaf(tr.Source)
	if tr.Target != "" {
		leaf = m.Resolver.Leaf(tr.Target)
	}
	state, more := Settle(m, leaf, tr.Guard)
	return Expect{State: state, Actions: dedup(append(append([]string{}, tr.Effects...), more...))}
}

// Compare checks the results of one machine against its table and returns the
// findings, without the "xstate: " prefix. compared counts the cases compared.
func Compare(m *machine.Machine, results []Result) (findings []string, compared int) {
	byCase := map[string]Result{}
	var top *Result
	for i := range results {
		r := results[i]
		if r.Machine != m.ID {
			continue
		}
		if r.Case == "" {
			top = &results[i]
			continue
		}
		byCase[r.Case] = r
	}
	if top == nil {
		return []string{m.ID + ": XState gave no answer for the machine"}, 0
	}
	if top.Error != "" {
		return []string{fmt.Sprintf("%s: XState rejects the machine: %s", m.ID, top.Error)}, 0
	}
	if want, _ := Settle(m, m.Resolver.Leaf(m.Table.Initial), ""); top.Initial != want {
		findings = append(findings, fmt.Sprintf("%s initial: XState starts in %s, the table starts in %s", m.ID, top.Initial, want))
	}
	for _, tr := range m.Table.Transitions {
		if tr.Kind != "on" {
			continue
		}
		compared++
		r, ok := byCase[tr.ID]
		switch {
		case !ok:
			findings = append(findings, fmt.Sprintf("%s %s: XState gave no answer", m.ID, tr.ID))
			continue
		case r.Error != "":
			findings = append(findings, fmt.Sprintf("%s %s: XState fails: %s", m.ID, tr.ID, r.Error))
			continue
		}
		want := ExpectCase(m, tr)
		if r.State != want.State {
			findings = append(findings, fmt.Sprintf("%s %s: target: XState enters %s, the table expects %s", m.ID, tr.ID, r.State, want.State))
		}
		if got := dedup(r.Actions); strings.Join(got, ",") != strings.Join(want.Actions, ",") {
			findings = append(findings, fmt.Sprintf("%s %s: actions: XState executes %s, the table expects %s", m.ID, tr.ID, list(got), list(want.Actions)))
		}
	}
	return findings, compared
}

func dedup(names []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

func list(names []string) string { return "[" + strings.Join(names, " ") + "]" }
