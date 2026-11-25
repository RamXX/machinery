package fsm

import (
	"sort"
	"unicode"
)

// PersistedEdges lists, sorted, the pairs (state a record is loaded in,
// state it is saved in) the machine allows, where both are domain states (a
// name starting with an upper-case letter). A transition into a transient
// state is followed through that state's onDone, onError and after branches
// to the domain states it can end in. Pairs with equal source and target
// are left out.
func (m *Machine) PersistedEdges() [][2]string {
	seen := map[[2]string]bool{}
	for _, t := range m.t.Transitions {
		if !isDomain(t.Source) || t.Target == "" {
			continue
		}
		for _, end := range m.ends(t.Target, map[string]bool{}) {
			if end != t.Source {
				seen[[2]string{t.Source, end}] = true
			}
		}
	}
	out := make([][2]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// ends is the domain states a walk entering state can rest in.
func (m *Machine) ends(state string, visiting map[string]bool) []string {
	if isDomain(state) {
		return []string{state}
	}
	if visiting[state] {
		return nil
	}
	visiting[state] = true
	var out []string
	for _, t := range m.t.Transitions {
		if t.Source == state && t.Target != "" && (t.Kind == kindOnDone || t.Kind == kindOnError || t.Kind == kindAfter) {
			out = append(out, m.ends(t.Target, visiting)...)
		}
	}
	return out
}

func isDomain(state string) bool {
	for _, r := range state {
		return unicode.IsUpper(r)
	}
	return false
}
