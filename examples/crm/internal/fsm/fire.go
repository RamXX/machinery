package fsm

import (
	"context"
	"fmt"
)

// Result is what one event did.
type Result struct {
	State     string   // the state after the event
	CaseID    string   // the case that answered the event: the first fired, or the ignore or refusal
	Cases     []string // the transition case ids that fired, in order
	Refused   bool     // no transition fired; the state is unchanged
	Reason    string   // why it was refused
	Internal  bool     // the branch that fired has no target (the state did not move)
	Effects   []string // effects that ran for real, in order
	Simulated []string // effects that are still noops, in order
}

func (r *Result) record(name string, simulated bool) {
	if simulated {
		r.Simulated = append(r.Simulated, name)
	} else {
		r.Effects = append(r.Effects, name)
	}
}

// Fire sends event to a record in state. The first branch of the table, in
// table order, whose guard is empty or holds fires. ctx will bound invoked
// actors; this engine does not run them yet.
func (m *Machine) Fire(ctx context.Context, state, event string, c Ctx) (Result, error) {
	res := Result{State: state, Cases: []string{}, Effects: []string{}, Simulated: []string{}}
	if m.units == nil {
		return res, fmt.Errorf("table %s: not bound", m.t.Machine)
	}
	st, ok := m.states[state]
	if !ok {
		return res, fmt.Errorf("table %s: unknown state %q", m.t.Machine, state)
	}
	if st.Kind == "final" {
		return refuse(res, "", "not accepted in state "+state), nil
	}
	branches := m.branches(state, kindOn, event)
	if len(branches) == 0 {
		for _, ig := range m.t.Ignores {
			if ig.State == state && ig.Event == event {
				return refuse(res, ig.ID, ig.Reason), nil
			}
		}
		return refuse(res, "", "not accepted in state "+state), nil
	}
	t, ok := m.first(branches, c)
	if !ok {
		for _, rf := range m.t.Refusals {
			if rf.State == state && rf.Handler == event {
				return refuse(res, rf.ID, rf.Disposition), nil
			}
		}
		return refuse(res, "", "no branch of "+event+" admits this request in state "+state), nil
	}
	res.Internal = t.Target == ""
	return res, m.take(&res, t, c)
}

func refuse(r Result, id, reason string) Result {
	r.Refused, r.CaseID, r.Reason = true, id, reason
	return r
}

// branches lists the transitions of a source, kind and event in table order.
func (m *Machine) branches(source, kind, event string) []transition {
	var out []transition
	for _, t := range m.t.Transitions {
		if t.Source == source && t.Kind == kind && t.Event == event {
			out = append(out, t)
		}
	}
	return out
}

// first is the first branch whose guard is empty or holds.
func (m *Machine) first(branches []transition, c Ctx) (transition, bool) {
	for _, t := range branches {
		if t.Guard == "" || m.units.Guards[t.Guard](c) {
			return t, true
		}
	}
	return transition{}, false
}

// take runs a transition's effects in order and moves to its target.
func (m *Machine) take(res *Result, t transition, c Ctx) error {
	res.Cases = append(res.Cases, t.ID)
	if res.CaseID == "" {
		res.CaseID = t.ID
	}
	for _, name := range t.Effects {
		a := m.units.Actions[name]
		if a.Run != nil {
			if err := a.Run(c); err != nil {
				return fmt.Errorf("%s: action %s: %w", t.ID, name, err)
			}
		}
		res.record(name, a.Simulated)
	}
	if t.Target != "" {
		res.State = t.Target
	}
	return nil
}
