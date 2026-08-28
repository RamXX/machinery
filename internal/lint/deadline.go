// Absolute-deadline idiom checks (S9 of the dogfood systemic findings, as
// amended). A bound declared as a per-state `after` is silently cancelled
// when the state exits; two dogfood rounds hit the same defect (a per-state
// answerBound, then a trial window) before the stamped-absolute-deadline
// idiom was invented: stamp the instant once, derive the REMAINING delay,
// and consume the derived delay on every non-final state between the stamp
// and the terminal the bound governs. The checkable shape: a derived delay
// (name ending in Remaining, or a _delays description saying so) must leave
// no HOLE in its span: a non-final state that sits between two consuming
// states but does not itself consume the delay is a window where the
// deadline silently stops ticking. Warnings tier: span membership is
// derived from the state graph, and the idiom is a convention.

package lint

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// derivedDeadlineDelay reports whether a _delays entry names the derived
// remaining-time form of a stamped absolute deadline.
func derivedDeadlineDelay(name, desc string) bool {
	if strings.HasSuffix(strings.ToLower(name), "remaining") {
		return true
	}
	d := strings.ToLower(desc)
	return strings.Contains(d, "derived") && (strings.Contains(d, "deadline") || strings.Contains(d, "absolute"))
}

// stateGraph returns adjacency over state names: every on/always/after/invoke
// transition target.
func stateGraph(m *ir.Value) map[string][]string {
	adj := map[string][]string{}
	addTargets := func(from string, v *ir.Value) {
		adj[from] = append(adj[from], transitionTargets(v)...)
	}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		if on := so.GetObject("on"); on != nil {
			for _, k := range on.Keys() {
				addTargets(s.Name, on.Get2(k))
			}
		}
		addTargets(s.Name, so.Get2("always"))
		if af := so.GetObject("after"); af != nil {
			for _, k := range af.Keys() {
				addTargets(s.Name, af.Get2(k))
			}
		}
		if inv := so.Get2("invoke"); inv != nil && inv.Kind == ir.KindObject {
			io := inv.AsObject()
			addTargets(s.Name, io.Get2("onDone"))
			addTargets(s.Name, io.Get2("onError"))
		}
	}
	return adj
}

func transitionTargets(v *ir.Value) []string {
	if v == nil {
		return nil
	}
	var items []*ir.Value
	if v.Kind == ir.KindArray {
		items = v.AsArray()
	} else {
		items = []*ir.Value{v}
	}
	var out []string
	for _, it := range items {
		if it == nil {
			continue
		}
		switch it.Kind {
		case ir.KindString:
			out = append(out, it.AsString())
		case ir.KindObject:
			if t := it.AsObject().GetString("target"); t != "" {
				out = append(out, t)
			}
		}
	}
	return out
}

func reachableFrom(adj map[string][]string, starts map[string]bool) map[string]bool {
	seen := map[string]bool{}
	var stack []string
	for s := range starts {
		stack = append(stack, s)
		seen[s] = true
	}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, t := range adj[s] {
			if !seen[t] {
				seen[t] = true
				stack = append(stack, t)
			}
		}
	}
	return seen
}

func invert(adj map[string][]string) map[string][]string {
	out := map[string][]string{}
	for s, ts := range adj {
		for _, t := range ts {
			out[t] = append(out[t], s)
		}
	}
	return out
}

// LintDerivedDeadlines warns for every hole in a derived deadline's span.
func LintDerivedDeadlines(m *ir.Value, base string) (warns []string) {
	ro := m.AsObject()
	delays := ro.GetObject("_delays")
	if delays == nil {
		return nil
	}
	finals := map[string]bool{}
	consumers := map[string]map[string]bool{}
	exempt := map[string]bool{}
	for _, s := range ir.WalkStates(ro.Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		if so.GetString("type") == "final" {
			finals[s.Name] = true
		}
		hasAfter := false
		if af := so.GetObject("after"); af != nil {
			hasAfter = af.Len() > 0
			for _, k := range af.Keys() {
				if consumers[k] == nil {
					consumers[k] = map[string]bool{}
				}
				consumers[k][s.Name] = true
			}
		}
		// A state already carrying an after (a retry sibling's backoff) can
		// take no second one: the TLA retry template admits exactly one, and
		// the idiom covers those re-entries with a guard-first deadline
		// check instead. A pure-always state is transient and never dwells.
		if hasAfter {
			exempt[s.Name] = true
		}
		if so.Get2("always") != nil && so.Get2("on") == nil && so.Get2("invoke") == nil && !hasAfter {
			exempt[s.Name] = true
		}
	}
	adj := stateGraph(m)
	radj := invert(adj)
	for _, name := range delays.Keys() {
		if !derivedDeadlineDelay(name, delays.GetString(name)) {
			continue
		}
		cons := consumers[name]
		if len(cons) < 2 {
			continue // a single consumer defines no span to hole
		}
		fromCons := reachableFrom(adj, cons)
		toCons := reachableFrom(radj, cons)
		var holes []string
		for s := range fromCons {
			if toCons[s] && !cons[s] && !finals[s] && !exempt[s] {
				holes = append(holes, s)
			}
		}
		sort.Strings(holes)
		for _, h := range holes {
			warns = append(warns, fmt.Sprintf("%s: state %s sits inside the span of derived deadline %s but carries no after consuming it; the deadline silently stops ticking there (the stamped-absolute-deadline idiom wants the derived delay on every non-final state between the stamp and the terminal it governs)", base, h, ir.Repr(name)))
		}
	}
	return warns
}
