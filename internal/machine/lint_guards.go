package machine

import (
	"fmt"

	"github.com/RamXX/machinery/internal/ir"
)

// lintGuards holds the root _guards declaration (docs/formats.md section 7.2):
// an object mapping a guard the machine uses to a list of at least two
// distinct clause names matching ^[a-z][A-Za-z0-9]*$.
func lintGuards(base string, ro *ir.Object, used []string) (errs []string) {
	gv := ro.Get2("_guards")
	if gv == nil {
		return nil
	}
	if gv.Kind != ir.KindObject {
		return []string{base + ": _guards must be an object mapping a guard name to the list of its clause names"}
	}
	usedSet := map[string]bool{}
	for _, g := range used {
		usedSet[g] = true
	}
	g := gv.AsObject()
	for _, name := range g.Keys() {
		if !usedSet[name] {
			errs = append(errs, fmt.Sprintf("%s: _guards declares %s but no transition uses that guard; drop the entry or fix the name", base, ir.Repr(name)))
		}
		v := g.Get2(name)
		if v == nil || v.Kind != ir.KindArray {
			errs = append(errs, fmt.Sprintf("%s: _guards[%s] must be a list of clause names", base, ir.Repr(name)))
			continue
		}
		clauses := v.AsArray()
		if len(clauses) < 2 {
			errs = append(errs, fmt.Sprintf("%s: _guards[%s] lists %d clause(s); a guard needs at least two (a guard with one check has nothing to separate)", base, ir.Repr(name), len(clauses)))
		}
		seen := map[string]bool{}
		for _, c := range clauses {
			if c == nil || c.Kind != ir.KindString || !reClause.MatchString(c.AsString()) {
				errs = append(errs, fmt.Sprintf("%s: _guards[%s] clause %s must be a string matching ^[a-z][A-Za-z0-9]*$", base, ir.Repr(name), ir.Repr(c)))
				continue
			}
			if seen[c.AsString()] {
				errs = append(errs, fmt.Sprintf("%s: _guards[%s] lists clause %s twice", base, ir.Repr(name), ir.Repr(c.AsString())))
			}
			seen[c.AsString()] = true
		}
	}
	return errs
}

// lintExternalEvents holds the root _external_events marker: the events a
// lifecycle machine receives that are not model actions. Each entry must be a
// non-empty event name that some state handles or ignores; an entry nothing
// answers is a stale marker.
func lintExternalEvents(base string, ro *ir.Object, states []ir.StateEntry) []string {
	return lintEventList(base, ro, states, "_external_events", "this machine receives from outside its component")
}
