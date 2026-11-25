package fsm

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// maxSteps bounds the walk through invoked states.
const maxSteps = 16

// Result is what one event did.
type Result struct {
	State     string   // the state after the event
	CaseID    string   // the case that answered the event: the first fired, or the ignore or refusal
	Cases     []string // the transition case ids that fired, in order
	Refused   bool     // no transition fired; the state is unchanged
	Reason    string   // why it was refused
	Internal  bool     // the branch that fired has no target (the state did not move)
	Failed    bool     // an onError or after branch fired: the invoked actor failed or timed out
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
// table order, whose guard is empty or holds fires; then, while the machine
// is in a state that invokes an actor, the actor runs and its onDone,
// onError or after branch fires. ctx bounds the actors.
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
	if err := m.take(&res, t, c); err != nil || res.Internal {
		return res, err
	}
	return res, m.walk(ctx, &res, c)
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

// walk runs the actor of each invoking state it reaches and fires the
// branch the outcome selects: after when the deadline passed, onError when
// the actor failed, onDone otherwise.
func (m *Machine) walk(ctx context.Context, res *Result, c Ctx) error {
	for step := 0; len(m.states[res.State].Invoke) > 0; step++ {
		if step == maxSteps {
			return fmt.Errorf("table %s: more than %d invoked states in one event", m.t.Machine, maxSteps)
		}
		source := res.State
		kind, event := m.invoke(ctx, res, m.states[source], c)
		t, ok := m.first(m.branches(source, kind, event), c)
		if !ok {
			return fmt.Errorf("table %s: state %s has no %s branch for %s that admits", m.t.Machine, source, kind, event)
		}
		res.Failed = res.Failed || kind != kindOnDone
		if err := m.take(res, t, c); err != nil {
			return err
		}
		if t.Target == "" {
			return fmt.Errorf("table %s: %s keeps the machine in invoking state %s", m.t.Machine, t.ID, source)
		}
	}
	return nil
}

// invoke runs the state's actor under the state's after deadline and
// returns the kind and event of the branch to take.
func (m *Machine) invoke(ctx context.Context, res *Result, st state, c Ctx) (string, string) {
	name := st.Invoke[0]
	actor := m.units.Actors[name]
	var timer []transition
	for _, t := range m.t.Transitions {
		if t.Source == st.Path && t.Kind == kindAfter {
			timer = append(timer, t)
		}
	}
	run := ctx
	if len(timer) > 0 {
		var cancel context.CancelFunc
		run, cancel = context.WithTimeout(ctx, time.Duration(m.t.Delays[timer[0].Event])*time.Millisecond)
		defer cancel()
	}
	var err error
	if actor.Run != nil {
		err = actor.Run(run, c)
	}
	res.record(name, actor.Simulated)
	switch {
	case len(timer) > 0 && errors.Is(run.Err(), context.DeadlineExceeded):
		return kindAfter, timer[0].Event
	case err != nil:
		return kindOnError, name
	default:
		return kindOnDone, name
	}
}
