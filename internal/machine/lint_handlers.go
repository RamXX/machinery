package machine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// lintHandlers holds the per-state rules over trigger containers: event
// names, branch order, guard-false dispositions (_refusal), the always-list
// exhaustiveness note and strict target resolution.
func lintHandlers(base, p string, node *ir.Value, trs []ir.Transition, res *ir.Resolver) (errs []string) {
	o := node.AsObject()

	// _refusal: what happens when a fully guarded handler admits nothing
	refusal := map[string]string{}
	consumed := map[string]bool{}
	if rv := o.Get2("_refusal"); rv != nil {
		if rv.Kind != ir.KindObject {
			errs = append(errs, base+": state "+p+" _refusal must map handler names to disposition strings")
		} else {
			for _, k := range rv.AsObject().Keys() {
				v := rv.AsObject().Get2(k)
				if v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
					errs = append(errs, fmt.Sprintf("%s: state %s _refusal[%s] has no disposition; state what happens when no guard admits", base, p, ir.Repr(k)))
					continue
				}
				refusal[k] = v.AsString()
			}
		}
	}
	// a fully guarded handler silently absorbs its trigger when every guard
	// is false: the author must say whether that is a refusal or a bug
	checkGuardFalse := func(key, label, trigger string, t *ir.Value) {
		bs := branchesOf(t)
		if len(bs) == 0 {
			return
		}
		for _, b := range bs {
			if !b.HasGuard {
				return
			}
		}
		consumed[key] = true
		if strings.TrimSpace(refusal[key]) == "" {
			errs = append(errs, fmt.Sprintf("%s: state %s %s is fully guarded, so %s that satisfies no guard is silently absorbed. Declare the disposition: _refusal: {%s: \"<what happens when no guard admits>\"}, or add an unguarded branch",
				base, p, label, trigger, ir.Repr(key)))
		}
	}
	// branch shadowing and duplicate guards, for every branch list
	checkShadow := func(label string, t *ir.Value) {
		if t != nil && t.Kind == ir.KindArray && len(t.AsArray()) == 0 {
			errs = append(errs, fmt.Sprintf("%s: state %s %s has no branches (an empty branch list silently swallows the trigger)", base, p, label))
			return
		}
		bs := branchesOf(t)
		for i := 0; i < len(bs)-1; i++ {
			if !bs[i].HasGuard {
				errs = append(errs, fmt.Sprintf("%s: state %s %s branch %d is unguarded but not last; later branches are unreachable", base, p, label, i+1))
			}
		}
		seen := map[string]bool{}
		for i, b := range bs {
			if !b.HasGuard {
				continue
			}
			g := guardOf(t, i)
			if seen[g] {
				errs = append(errs, fmt.Sprintf("%s: state %s %s has two branches with the same guard (%s); the second is unreachable", base, p, label, g))
			}
			seen[g] = true
		}
	}

	if on := o.Get2("on"); on != nil && on.Kind == ir.KindObject {
		for _, ev := range on.AsObject().Keys() {
			if ev == "*" || strings.HasPrefix(ev, "done.") || strings.HasPrefix(ev, "error.") {
				errs = append(errs, fmt.Sprintf("%s: state %s: event name %s is outside the supported subset (wildcard and platform events are not modeled; use explicit events and invoke onDone/onError)", base, p, ir.Repr(ev)))
			} else if !reName.MatchString(ev) {
				errs = append(errs, fmt.Sprintf("%s: state %s: event name %s must match [A-Za-z][A-Za-z0-9_]* (names are hashed into case ids: use letters, digits and underscores)", base, p, ir.Repr(ev)))
			}
			checkShadow("on:"+ev, on.AsObject().Get2(ev))
			checkGuardFalse(ev, "on:"+ev, "an event", on.AsObject().Get2(ev))
		}
	}
	if after := o.Get2("after"); after != nil && after.Kind == ir.KindObject {
		for _, delay := range after.AsObject().Keys() {
			checkShadow("after:"+delay, after.AsObject().Get2(delay))
			checkGuardFalse("after:"+delay, "after:"+delay, "a timer", after.AsObject().Get2(delay))
		}
	}
	if od := o.Get2("onDone"); od != nil {
		checkShadow("onDone", od)
		checkGuardFalse("onDone", "onDone", "a completion", od)
	}
	for _, iv := range ir.InvokesOf(node) {
		if iv == nil || iv.Kind != ir.KindObject {
			continue
		}
		ivObj := iv.AsObject()
		src := ivObj.GetString("src")
		for _, key := range []string{"onDone", "onError"} {
			if ivObj.Get2(key) != nil {
				checkShadow("invoke."+key, ivObj.Get2(key))
				// keyed by src so a state invoking two actors names which one
				checkGuardFalse(src+"."+key, "invoke "+src+"."+key, "a result", ivObj.Get2(key))
			}
		}
	}
	if always := o.Get2("always"); always != nil {
		checkShadow("always", always)
		bs := branchesOf(always)
		fully := len(bs) > 0
		for _, b := range bs {
			if !b.HasGuard {
				fully = false
			}
		}
		// an escape is an actual transition or invoke; empty containers are not
		hasEscape := hasEntries(o.Get2("after")) || hasEntries(o.Get2("on")) ||
			(o.Get2("invoke") != nil && len(ir.InvokesOf(node)) > 0)
		if fully && !hasEscape && strings.TrimSpace(o.GetString("_exhaustive")) == "" {
			errs = append(errs, base+": state "+p+" has a fully guarded always-list and no unguarded escape (after/on/invoke); if no guard is true the machine is stuck. Add an unguarded fallback branch, or an _exhaustive note stating why the guards are total")
		}
	}
	for _, tr := range trs {
		if _, why := res.Resolve(tr.Target, p); why != "" {
			errs = append(errs, base+": "+why+" from "+p+" ("+tr.Kind+":"+tr.Event+")")
		}
	}
	// a disposition for a handler that is not fully guarded (or no longer
	// exists) reads as coverage nobody has
	var stale []string
	for k := range refusal {
		if !consumed[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(stale)
	for _, k := range stale {
		errs = append(errs, fmt.Sprintf("%s: state %s declares _refusal[%s] but no fully guarded handler of that name exists; drop the entry or fix the name", base, p, ir.Repr(k)))
	}
	return errs
}

// lintAlwaysCycles reports one cycle of unguarded eventless transitions.
func lintAlwaysCycles(base string, states []ir.StateEntry, resolve func(string, string) (string, string)) []string {
	next := map[string][]string{}
	for _, s := range states {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
			if tr.Kind == "always" && !tr.HasGuard {
				if dest, _ := resolve(tr.Target, s.Path); dest != "" {
					next[s.Path] = append(next[s.Path], dest)
				}
			}
		}
	}
	color := map[string]int{} // 0 unvisited, 1 on stack, 2 done
	var stack []string
	var found string
	var dfs func(u string) bool
	dfs = func(u string) bool {
		color[u] = 1
		stack = append(stack, u)
		for _, v := range next[u] {
			if color[v] == 1 {
				i := len(stack) - 1
				for i >= 0 && stack[i] != v {
					i--
				}
				found = strings.Join(append(append([]string{}, stack[i:]...), v), " -> ")
				return true
			}
			if color[v] == 0 && dfs(v) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[u] = 2
		return false
	}
	for _, s := range states {
		if color[s.Path] == 0 && len(next[s.Path]) > 0 && dfs(s.Path) {
			return []string{base + ": unguarded always cycle " + found + " (eventless transitions with no guard re-fire forever)"}
		}
	}
	return nil
}

// lintEventCompleteness: every resting leaf state handles or explicitly
// ignores every event the machine reacts to anywhere. Handlers and _ignores on
// the ancestor chain are credited, as an unhandled event bubbles up.
func lintEventCompleteness(base string, states []ir.StateEntry, nodeOf map[string]*ir.Value) (errs []string) {
	allEvents := map[string]bool{}
	onOf := map[string]map[string]bool{}
	igOf := map[string]map[string]string{}
	for _, s := range states {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		o := s.Node.AsObject()
		onSet := map[string]bool{}
		if on := o.Get2("on"); on != nil && on.Kind == ir.KindObject {
			for _, ev := range on.AsObject().Keys() {
				allEvents[ev] = true
				onSet[ev] = true
			}
		}
		onOf[s.Path] = onSet
		ignores := map[string]string{}
		valid := true
		if iv := o.Get2("_ignores"); iv != nil && iv.Kind == ir.KindObject {
			for _, k := range iv.AsObject().Keys() {
				v := iv.AsObject().Get2(k)
				if v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
					valid = false
				} else {
					ignores[k] = v.AsString()
				}
			}
		} else if iv != nil {
			valid = false
		}
		if !valid {
			errs = append(errs, base+": state "+s.Path+" _ignores must map event names to reason strings")
			ignores = map[string]string{}
		}
		for _, ev := range sortedMapKeys(ignores) {
			if onSet[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s both handles and ignores event %s", base, s.Path, ir.Repr(ev)))
			}
		}
		igOf[s.Path] = ignores
	}
	events := sortedSet(allEvents)
	for _, s := range states {
		p, node := s.Path, s.Node
		if node == nil || node.Kind != ir.KindObject {
			continue
		}
		o := node.AsObject()
		if !ir.IsUpperFirst(s.Name) || o.GetString("type") == "final" || o.Get2("states") != nil {
			continue
		}
		chain := ancestorChain(p)
		transient := false
		for _, q := range chain {
			if qn := nodeOf[q]; qn != nil && qn.Kind == ir.KindObject {
				if qo := qn.AsObject(); qo.Get2("invoke") != nil || qo.Get2("always") != nil {
					transient = true
					break
				}
			}
		}
		if transient {
			continue
		}
		handled, ignored := map[string]bool{}, map[string]bool{}
		for _, q := range chain {
			for ev := range onOf[q] {
				handled[ev] = true
			}
			for ev := range igOf[q] {
				ignored[ev] = true
			}
		}
		for _, ev := range sortedMapKeys(igOf[p]) {
			if !onOf[p][ev] && handled[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s ignores event %s but an ancestor handles it (the handler wins; drop the _ignores entry or hoist the reasoning)", base, p, ir.Repr(ev)))
			}
		}
		for _, ev := range events {
			if !handled[ev] && !ignored[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s neither handles nor explicitly ignores event %s (add it to on: or to _ignores: with a reason)", base, p, ir.Repr(ev)))
			}
		}
	}
	return errs
}

// lintReachability: every state is reachable from the initial state.
func lintReachability(base string, ro *ir.Object, states []ir.StateEntry, res *ir.Resolver) (errs []string) {
	init := ro.GetString("initial")
	if !ro.Get2("states").AsObject().Has(init) {
		return []string{fmt.Sprintf("%s: initial %s is not a top-level state", base, ir.Repr(init))}
	}
	pathSet, nodeOf := res.PathSet, res.NodeOf
	reached := map[string]bool{}
	var enter func(p string)
	enter = func(p string) {
		if reached[p] {
			return
		}
		reached[p] = true
		if node := nodeOf[p]; node != nil && node.Kind == ir.KindObject {
			o := node.AsObject()
			if o.Get2("states") != nil {
				if ci := o.GetString("initial"); ci != "" && pathSet[p+"."+ci] {
					enter(p + "." + ci)
				}
			}
		}
	}
	enter(init)
	for frontier := true; frontier; {
		frontier = false
		active := map[string]bool{}
		for p := range reached {
			active[p] = true
			for q := p; strings.Contains(q, "."); {
				q = q[:strings.LastIndex(q, ".")]
				active[q] = true
			}
		}
		for p := range active {
			for _, tr := range ir.TransitionsOf(nodeOf[p], nil, p) {
				if dest, _ := res.Resolve(tr.Target, p); dest != "" && !reached[dest] {
					enter(dest)
					frontier = true
				}
			}
		}
	}
	for _, s := range states {
		p := s.Path
		if reached[p] {
			continue
		}
		// reachable through containment overlap
		hidden := false
		for r := range reached {
			if strings.HasPrefix(p, r+".") || strings.HasPrefix(r, p+".") {
				hidden = true
				break
			}
		}
		if !hidden {
			errs = append(errs, base+": unreachable state "+p)
		}
	}
	return errs
}
