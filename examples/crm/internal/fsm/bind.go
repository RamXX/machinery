package fsm

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Ctx is passed through to every guard, action and actor untouched.
type Ctx any

// Action is the implementation of a named action. A simulated action is
// still a noop and is reported as such.
type Action struct {
	Run       func(Ctx) error
	Simulated bool
}

// Actor is the implementation of an invoked actor. Its context carries the
// deadline of the state's after delay, when there is one.
type Actor struct {
	Run       func(context.Context, Ctx) error
	Simulated bool
}

// Units maps the names the table uses to their implementations.
type Units struct {
	Guards  map[string]func(Ctx) bool
	Actions map[string]Action
	Actors  map[string]Actor
}

// Bind checks that every guard, action and actor the table names has an
// implementation and keeps them. The error names every missing one.
func (m *Machine) Bind(u Units) error {
	guards, actions, actors := m.names()
	var missing []string
	for _, g := range guards {
		if _, ok := u.Guards[g]; !ok {
			missing = append(missing, "guard "+g)
		}
	}
	for _, a := range actions {
		if _, ok := u.Actions[a]; !ok {
			missing = append(missing, "action "+a)
		}
	}
	for _, a := range actors {
		if _, ok := u.Actors[a]; !ok {
			missing = append(missing, "actor "+a)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("table %s: no implementation for %s; implement them or fix the table", m.t.Machine, strings.Join(missing, ", "))
	}
	m.units = &u
	return nil
}

// names is every unit name the table lists or uses, each set sorted.
func (m *Machine) names() (guards, actions, actors []string) {
	g, a, r := set(m.t.Units.Guards), set(m.t.Units.Actions), set(m.t.Units.Actors)
	for _, t := range m.t.Transitions {
		if t.Guard != "" {
			g[t.Guard] = true
		}
		for _, n := range t.Effects {
			a[n] = true
		}
	}
	for _, s := range m.t.States {
		for _, n := range s.Invoke {
			r[n] = true
		}
	}
	return sorted(g), sorted(a), sorted(r)
}

func set(names []string) map[string]bool {
	s := map[string]bool{}
	for _, n := range names {
		s[n] = true
	}
	return s
}

func sorted(s map[string]bool) []string {
	out := make([]string, 0, len(s))
	for n := range s {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
