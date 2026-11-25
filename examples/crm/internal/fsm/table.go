// Package fsm drives a state machine from its generated runtime table
// (docs/formats.md section 7.3). The table decides every transition; the
// caller supplies the named guards, actions and actors. It knows nothing
// about HTTP or SQL.
package fsm

import (
	"encoding/json"
	"fmt"
)

// The table of section 7.3, as far as the engine reads it. Other keys, such
// as clauses, are ignored.
type table struct {
	Version     int            `json:"version"`
	Machine     string         `json:"machine"`
	Initial     string         `json:"initial"`
	States      []state        `json:"states"`
	Transitions []transition   `json:"transitions"`
	Ignores     []ignore       `json:"ignores"`
	Refusals    []refusal      `json:"refusals"`
	Units       units          `json:"units"`
	Delays      map[string]int `json:"delays"`
}

type state struct {
	Path   string   `json:"path"`
	Kind   string   `json:"kind"`
	Entry  []string `json:"entry"`
	Exit   []string `json:"exit"`
	Invoke []string `json:"invoke"`
}

type transition struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Kind    string   `json:"kind"`
	Event   string   `json:"event"`
	Guard   string   `json:"guard"`
	Target  string   `json:"target"`
	Actions []string `json:"actions"`
	Effects []string `json:"effects"`
}

type ignore struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Event  string `json:"event"`
	Reason string `json:"reason"`
}

type refusal struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Handler     string `json:"handler"`
	Disposition string `json:"disposition"`
}

type units struct {
	Guards  []string `json:"guards"`
	Actions []string `json:"actions"`
	Actors  []string `json:"actors"`
}

// Transition kinds the engine drives.
const (
	kindOn      = "on"
	kindAfter   = "after"
	kindOnDone  = "onDone"
	kindOnError = "onError"
)

// Machine is a loaded table. Bind it before firing events.
type Machine struct {
	t      table
	states map[string]state
	units  *Units
}

// Load parses a runtime table and rejects what this engine cannot drive.
func Load(tableJSON []byte) (*Machine, error) {
	var t table
	if err := json.Unmarshal(tableJSON, &t); err != nil {
		return nil, fmt.Errorf("parse table: %w", err)
	}
	if t.Version != 1 {
		return nil, fmt.Errorf("table %s: version %d, want 1", t.Machine, t.Version)
	}
	m := &Machine{t: t, states: map[string]state{}}
	for _, s := range t.States {
		if s.Kind != "atomic" && s.Kind != "final" {
			return nil, fmt.Errorf("table %s: state %s is %s; this engine drives atomic and final states only", t.Machine, s.Path, s.Kind)
		}
		if len(s.Invoke) > 1 {
			return nil, fmt.Errorf("table %s: state %s invokes %d actors; this engine runs one per state", t.Machine, s.Path, len(s.Invoke))
		}
		m.states[s.Path] = s
	}
	if _, ok := m.states[t.Initial]; !ok {
		return nil, fmt.Errorf("table %s: initial state %q is not a state of the table", t.Machine, t.Initial)
	}
	for _, tr := range t.Transitions {
		if err := m.checkTransition(tr); err != nil {
			return nil, fmt.Errorf("table %s: %s: %w", t.Machine, tr.ID, err)
		}
	}
	for _, ig := range t.Ignores {
		if _, ok := m.states[ig.State]; !ok {
			return nil, fmt.Errorf("table %s: ignore %s names unknown state %q", t.Machine, ig.ID, ig.State)
		}
	}
	return m, nil
}

func (m *Machine) checkTransition(tr transition) error {
	src, ok := m.states[tr.Source]
	if !ok {
		return fmt.Errorf("unknown source state %q", tr.Source)
	}
	if _, ok := m.states[tr.Target]; tr.Target != "" && !ok {
		return fmt.Errorf("unknown target state %q", tr.Target)
	}
	switch tr.Kind {
	case kindOn:
	case kindOnDone, kindOnError:
		if len(src.Invoke) == 0 || src.Invoke[0] != tr.Event {
			return fmt.Errorf("%s of %s, which state %s does not invoke", tr.Kind, tr.Event, tr.Source)
		}
	case kindAfter:
		if len(src.Invoke) == 0 {
			return fmt.Errorf("after on state %s, which invokes nothing; this engine runs timers only around an actor", tr.Source)
		}
		if _, ok := m.t.Delays[tr.Event]; !ok {
			return fmt.Errorf("after %s has no entry in delays", tr.Event)
		}
	default:
		return fmt.Errorf("transition kind %q is not supported by this engine", tr.Kind)
	}
	return nil
}

// Name is the machine id of the table.
func (m *Machine) Name() string { return m.t.Machine }

// Initial is the state a new record starts in.
func (m *Machine) Initial() string { return m.t.Initial }
