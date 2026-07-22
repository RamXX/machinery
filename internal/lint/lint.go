// Package lint is the Go port of machine_lint.py: structural lint over the
// finite state graph, JSON<->matrix reconciliation, and the CLI entrypoint.
package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// RootKeys / StateKeys / InvokeKeys / StateTypes mirror machine_lint.py exactly.
var RootKeys = map[string]bool{
	"id": true, "initial": true, "context": true, "states": true, "description": true,
	"meta": true, "version": true, "_comment": true, "_delays": true,
	"_lifecycle_of": true, "_role": true, "_component": true, "_max_retries": true,
	"_oracle_tag": true,
}

// TransitionKeys whitelists transition-object members. A typo ("tagret") used
// to silently become an internal self-transition; unknown keys are errors.
var TransitionKeys = map[string]bool{
	"target": true, "guard": true, "actions": true, "description": true, "_comment": true,
}

var StateKeys = map[string]bool{
	"on": true, "after": true, "always": true, "invoke": true, "entry": true, "exit": true,
	"states": true, "initial": true, "type": true, "id": true, "meta": true,
	"description": true, "tags": true, "onDone": true, "output": true, "_comment": true,
	"_exhaustive": true, "_ignores": true,
}

var InvokeKeys = map[string]bool{
	"src": true, "input": true, "id": true, "onDone": true, "onError": true, "_comment": true,
}

var StateTypes = map[string]bool{"": true, "atomic": true, "compound": true, "final": true}

// Result is the output of LintMachine.
type Result struct {
	Errs   []string
	Warns  []string
	Notes  []string
	Counts Counts
}

type Counts struct {
	States      int
	Transitions int
}

// resolver implements XState-v5-correct target resolution over one machine.
//
//   - a bare-name target resolves ONLY among siblings of the source state
//     (the source itself included, for explicit self-transitions);
//   - "#<machineId or custom state id>" resolves to that node, and
//     "#<id>.<path>" resolves every path segment strictly under that node;
//   - ".child" relative targets are out of contract and rejected loudly;
//   - anything else is dangling, never a fallback.
type resolver struct {
	statesVal *ir.Value
	pathSet   map[string]bool
	nodeOf    map[string]*ir.Value
	ids       map[string]string // custom state id -> full dotted path
	rootID    string
	initial   string
	idErrs    []string // id-registration findings (no base prefix)
}

func newResolver(m *ir.Value) *resolver {
	r := &resolver{pathSet: map[string]bool{}, nodeOf: map[string]*ir.Value{}, ids: map[string]string{}}
	ro := m.AsObject()
	r.rootID = ro.GetString("id")
	r.initial = ro.GetString("initial")
	r.statesVal = ro.Get2("states")
	for _, s := range ir.WalkStates(r.statesVal, "") {
		r.pathSet[s.Path] = true
		r.nodeOf[s.Path] = s.Node
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		idv := s.Node.AsObject().Get2("id")
		if idv == nil {
			continue
		}
		if idv.Kind != ir.KindString || idv.AsString() == "" {
			r.idErrs = append(r.idErrs, fmt.Sprintf("state %s id must be a non-empty string", s.Path))
			continue
		}
		id := idv.AsString()
		if !regexpIdent.MatchString(id) {
			r.idErrs = append(r.idErrs, fmt.Sprintf("state %s id %s is not a valid identifier (a dotted or symbolic id can never be referenced by #id.path)", s.Path, ir.Repr(id)))
			continue
		}
		if r.rootID != "" && id == r.rootID {
			r.idErrs = append(r.idErrs, fmt.Sprintf("state %s id %s collides with the machine id", s.Path, ir.Repr(id)))
			continue
		}
		if prev, dup := r.ids[id]; dup {
			r.idErrs = append(r.idErrs, fmt.Sprintf("duplicate state id %s (declared on %s and %s)", ir.Repr(id), prev, s.Path))
			continue
		}
		r.ids[id] = s.Path
	}
	return r
}

// resolve returns (destPath, "") on success or ("", why) on failure. An empty
// target is an internal self-transition and resolves to srcPath.
func (r *resolver) resolve(tgt, srcPath string) (string, string) {
	if tgt == "" {
		return srcPath, ""
	}
	if strings.HasPrefix(tgt, ".") {
		return "", fmt.Sprintf("relative target %s is outside the supported subset (use a sibling name or #id.path.to.state)", ir.Repr(tgt))
	}
	if strings.HasPrefix(tgt, "#") {
		rest := strings.TrimPrefix(tgt, "#")
		if rest == "" {
			return "", fmt.Sprintf("dangling target %s (empty id reference)", ir.Repr(tgt))
		}
		segs := strings.Split(rest, ".")
		var cur string // "" means the machine root
		head := segs[0]
		if p, ok := r.ids[head]; ok {
			cur = p
		} else if r.rootID != "" && head == r.rootID {
			cur = ""
		} else {
			return "", fmt.Sprintf("dangling target %s (%s is neither the machine id nor a declared state id)", ir.Repr(tgt), ir.Repr(head))
		}
		for _, seg := range segs[1:] {
			var children *ir.Value
			if cur == "" {
				children = r.statesVal
			} else if n := r.nodeOf[cur]; n != nil && n.Kind == ir.KindObject {
				children = n.AsObject().Get2("states")
			}
			if children == nil || children.Kind != ir.KindObject || !children.AsObject().Has(seg) {
				at := cur
				if at == "" {
					at = "the machine root"
				}
				return "", fmt.Sprintf("dangling target %s (%s has no child state %s)", ir.Repr(tgt), at, ir.Repr(seg))
			}
			if cur == "" {
				cur = seg
			} else {
				cur = cur + "." + seg
			}
		}
		if cur == "" {
			// bare #machineId: the machine re-enters through its initial state
			if r.initial != "" && r.pathSet[r.initial] {
				return r.initial, ""
			}
			return "", "" // a bogus root initial is reported separately
		}
		return cur, ""
	}
	if strings.Contains(tgt, ".") {
		return "", fmt.Sprintf("dangling target %s (dotted targets must use the #id.path.to.state form)", ir.Repr(tgt))
	}
	cand := tgt
	if i := strings.LastIndex(srcPath, "."); i >= 0 {
		cand = srcPath[:i] + "." + tgt
	}
	if r.pathSet[cand] {
		return cand, ""
	}
	return "", fmt.Sprintf("dangling target %s (no sibling of %s is named %s)", ir.Repr(tgt), srcPath, ir.Repr(tgt))
}

// ancestorChain returns the paths from the top-level ancestor down to p
// itself: "A.B.C" -> ["A", "A.B", "A.B.C"].
func ancestorChain(p string) []string {
	var out []string
	for i := 0; i < len(p); i++ {
		if p[i] == '.' {
			out = append(out, p[:i])
		}
	}
	return append(out, p)
}

// hasFinalDescendant reports whether any state in node's subtree is final.
func hasFinalDescendant(node *ir.Value) bool {
	if node == nil || node.Kind != ir.KindObject {
		return false
	}
	for _, s := range ir.WalkStates(node.AsObject().Get2("states"), "") {
		if s.Node != nil && s.Node.Kind == ir.KindObject && s.Node.AsObject().GetString("type") == "final" {
			return true
		}
	}
	return false
}

// objHasEntries reports a present, object-typed value with at least one member.
func objHasEntries(v *ir.Value) bool {
	return v != nil && v.Kind == ir.KindObject && v.AsObject().Len() > 0
}

// LintMachine mirrors machine_lint.lint_machine(m, base).
func LintMachine(m *ir.Value, base string) (errs, warns, notes []string, counts Counts) {
	if m == nil || m.Kind != ir.KindObject {
		return []string{base + ": machine is not an object"}, nil, nil, Counts{}
	}
	ro := m.AsObject()

	for _, k := range ro.Keys() {
		if !RootKeys[k] {
			sortedRoots := sortedKeysMap(RootKeys)
			errs = append(errs, fmt.Sprintf("%s: unsupported root key %s (supported: %s)",
				base, ir.Repr(k), strings.Join(sortedRoots, ", ")))
		}
	}
	statesVal := ro.Get2("states")
	if statesVal == nil || statesVal.Kind != ir.KindObject || statesVal.AsObject().Len() == 0 {
		errs = append(errs, base+": machine has no states")
		return errs, warns, notes, counts
	}

	states := ir.WalkStates(statesVal, "")
	counts.States = len(states)
	res := newResolver(m)
	pathSet, nodeOf := res.pathSet, res.nodeOf
	for _, e := range res.idErrs {
		errs = append(errs, base+": "+e)
	}
	resolve := res.resolve

	if tagV := ro.Get2("_oracle_tag"); tagV != nil {
		if tagV.Kind != ir.KindString || !regexpOracleTag.MatchString(tagV.AsString()) {
			errs = append(errs, fmt.Sprintf("%s: _oracle_tag %s must be 2-8 uppercase characters matching [A-Z0-9]", base, ir.Repr(tagV)))
		}
	}
	delaysVal := ro.Get2("_delays")
	declaredDelays := map[string]bool{}
	if delaysVal != nil && delaysVal.Kind == ir.KindObject {
		for _, k := range delaysVal.AsObject().Keys() {
			declaredDelays[k] = true
		}
	} else if delaysVal != nil {
		errs = append(errs, base+": _delays must be an object mapping delay names to \"<ms> - <rationale>\" strings")
	}

	var problems []string
	for _, s := range states {
		p, node := s.Path, s.Node
		if node == nil || node.Kind != ir.KindObject {
			errs = append(errs, base+": state "+p+" is not an object")
			continue
		}
		o := node.AsObject()
		if !regexpName.MatchString(s.Name) {
			errs = append(errs, fmt.Sprintf("%s: state name %s must match [A-Za-z][A-Za-z0-9_]* (state names become oracle stable-id and TLA+ identifiers)",
				base, ir.Repr(s.Name)))
		}
		for _, k := range o.Keys() {
			if !StateKeys[k] {
				errs = append(errs, fmt.Sprintf("%s: unsupported key %s in state %s", base, ir.Repr(k), p))
			}
		}
		// a null (or otherwise non-object/non-array) transition container is
		// malformed input, not "no transitions": error loudly
		for _, k := range []string{"on", "after", "_ignores"} {
			if v := o.Get2(k); v != nil && v.Kind != ir.KindObject {
				errs = append(errs, fmt.Sprintf("%s: state %s: %s must be an object, not null or another type", base, p, ir.Repr(k)))
			}
		}
		if v := o.Get2("invoke"); v != nil && v.Kind != ir.KindObject && v.Kind != ir.KindArray {
			errs = append(errs, fmt.Sprintf("%s: state %s: 'invoke' must be an object or array, not null or another type", base, p))
		}
		stype := o.GetString("type")
		if !StateTypes[stype] {
			errs = append(errs, fmt.Sprintf("%s: unsupported state type %s in %s (parallel/history are not in the supported subset)",
				base, ir.Repr(stype), p))
			continue
		}
		trs := ir.TransitionsOf(node, &problems, p)
		counts.Transitions += len(trs)
		isFinal := stype == "final"
		if isFinal && (len(trs) > 0 || o.Get2("invoke") != nil) {
			errs = append(errs, base+": final state "+p+" declares transitions or an invoke; XState ignores them, so they are dead spec")
		}
		if o.Get2("states") != nil && o.GetString("initial") == "" {
			errs = append(errs, base+": compound state "+p+" has no initial")
		}
		if o.GetString("initial") != "" && o.Get2("states") == nil {
			errs = append(errs, base+": state "+p+" has initial but no child states")
		}
		if init := o.GetString("initial"); init != "" && o.Get2("states") != nil &&
			!o.Get2("states").AsObject().Has(init) {
			errs = append(errs, fmt.Sprintf("%s: compound state %s initial %s is not one of its children", base, p, ir.Repr(init)))
		}
		// onDone fires only when a compound state's final descendant is
		// reached; on an atomic state (or a compound with no final descendant)
		// it is a phantom transition that lints clean and emits a phantom
		// oracle row while masking true unreachability of its target.
		if o.Get2("onDone") != nil {
			if o.Get2("states") == nil {
				errs = append(errs, base+": state "+p+" has onDone but no child states (onDone fires when a compound state reaches a final child; this one never fires)")
			} else if !hasFinalDescendant(node) {
				errs = append(errs, base+": state "+p+" has onDone but no final descendant state, so its onDone can never fire")
			}
		}
		// dead-end / absorbing: a non-final leaf needs at least one transition
		// that LEAVES the state, on itself or on an ancestor; internal and
		// self-targeted transitions do not count.
		if !isFinal && o.Get2("states") == nil {
			exits := false
			for _, q := range ancestorChain(p) {
				qtrs := trs
				if q != p {
					qtrs = ir.TransitionsOf(nodeOf[q], nil, q)
				}
				for _, tr := range qtrs {
					if !tr.HasTgt {
						continue
					}
					dest, why := resolve(tr.Target, q)
					if why != "" || dest != p {
						exits = true
						break
					}
				}
				if exits {
					break
				}
			}
			if !exits {
				if len(trs) == 0 {
					errs = append(errs, base+": dead-end non-final leaf state "+p)
				} else {
					errs = append(errs, base+": absorbing non-final leaf state "+p+" (transitions exist but none leaves the state; add an exiting transition or make it final)")
				}
			}
		}
		for _, iv := range ir.InvokesOf(node) {
			if iv == nil || iv.Kind != ir.KindObject {
				errs = append(errs, fmt.Sprintf("%s: state %s: invoke entries must be objects, got %s", base, p, ir.Repr(iv)))
				continue
			}
			ivObj := iv.AsObject()
			for _, k := range ivObj.Keys() {
				if !InvokeKeys[k] {
					errs = append(errs, fmt.Sprintf("%s: unsupported invoke key %s in state %s", base, ir.Repr(k), p))
				}
			}
			if src := ivObj.Get2("src"); src == nil || src.Kind != ir.KindString || src.AsString() == "" {
				errs = append(errs, fmt.Sprintf("%s: invoke in %s has no src (src is a mandatory string naming the invoked actor)", base, p))
			}
			if ivObj.Get2("onError") == nil {
				src := srcRepr(ivObj)
				errs = append(errs, fmt.Sprintf("%s: invoke %s in %s has no onError", base, src, p))
			}
		}
		if o.Get2("invoke") != nil && len(ir.InvokesOf(node)) > 0 {
			if o.Get2("after") == nil {
				errs = append(errs, base+": invoking state "+p+" has no after/timeout")
			} else if !objHasEntries(o.Get2("after")) && o.Get2("after").Kind == ir.KindObject {
				errs = append(errs, base+": invoking state "+p+" has an empty after block (a timeout delay is required; after: {} declares nothing)")
			}
		}
		if after := o.Get2("after"); after != nil && after.Kind == ir.KindObject {
			for _, delay := range after.AsObject().Keys() {
				if regexpAllDigits.MatchString(delay) {
					errs = append(errs, fmt.Sprintf("%s: state %s after key %s is a raw millisecond value; name the delay and declare its bound in _delays", base, p, ir.Repr(delay)))
				} else if !declaredDelays[delay] {
					errs = append(errs, fmt.Sprintf("%s: state %s after delay %s is not declared in _delays (named delays carry their ms bound and rationale there)", base, p, ir.Repr(delay)))
				}
			}
		}
		ir.ActionNames(o.Get2("entry"), &problems, p+" entry")
		ir.ActionNames(o.Get2("exit"), &problems, p+" exit")

		// branch shadowing + duplicate guards (uniform across every branch list)
		checkShadow := func(label string, t *ir.Value) {
			if t != nil && t.Kind == ir.KindArray && len(t.AsArray()) == 0 {
				// an empty branch list parses clean, counts as handled, and
				// produces zero oracle rows: the event is silently swallowed
				errs = append(errs, fmt.Sprintf("%s: state %s %s has no branches (an empty branch list silently swallows the trigger)", base, p, label))
				return
			}
			branches := normBranches(t)
			for i := 0; i < len(branches)-1; i++ {
				if !branches[i].HasGuard {
					errs = append(errs, fmt.Sprintf("%s: state %s %s branch %d is unguarded but not last; later branches are unreachable",
						base, p, label, i+1))
				}
			}
			// two branches with the same guard: the second can never fire
			seenGuard := map[string]bool{}
			for _, b := range branches {
				key := "unguarded"
				if b.HasGuard {
					key = b.Guard
				}
				if seenGuard[key] && key != "unguarded" { // unguarded dups already reported above
					errs = append(errs, fmt.Sprintf("%s: state %s %s has two branches with the same guard (%s); the second is unreachable",
						base, p, label, key))
				}
				seenGuard[key] = true
			}
		}
		if on := o.Get2("on"); on != nil {
			for _, ev := range on.AsObject().Keys() {
				if ev == "*" || strings.HasPrefix(ev, "done.") || strings.HasPrefix(ev, "error.") {
					errs = append(errs, fmt.Sprintf("%s: state %s: event name %s is outside the supported subset (wildcard and platform events would be modeled as ordinary events; use explicit events and invoke onDone/onError)",
						base, p, ir.Repr(ev)))
				} else if !regexpName.MatchString(ev) {
					errs = append(errs, fmt.Sprintf("%s: state %s: event name %s must match [A-Za-z][A-Za-z0-9_]* (event names become oracle stable-id and TLA+ identifiers)",
						base, p, ir.Repr(ev)))
				}
				checkShadow("on:"+ev, on.AsObject().Get2(ev))
			}
		}
		if after := o.Get2("after"); after != nil {
			for _, delay := range after.AsObject().Keys() {
				checkShadow("after:"+delay, after.AsObject().Get2(delay))
			}
		}
		if od := o.Get2("onDone"); od != nil {
			checkShadow("onDone", od)
		}
		for _, iv := range ir.InvokesOf(node) {
			ivObj := iv.AsObject()
			for _, key := range []string{"onDone", "onError"} {
				if ivObj.Get2(key) != nil {
					checkShadow("invoke."+key, ivObj.Get2(key))
				}
			}
		}
		if always := o.Get2("always"); always != nil {
			checkShadow("always", always)
			branches := normBranches(always)
			fullyGuarded := len(branches) > 0
			for _, b := range branches {
				if !b.HasGuard {
					fullyGuarded = false
				}
			}
			// an escape must be at least one actual transition or invoke entry:
			// empty on:{} / after:{} / invoke:[] containers are not escapes
			hasEscape := objHasEntries(o.Get2("after")) || objHasEntries(o.Get2("on")) ||
				(o.Get2("invoke") != nil && len(ir.InvokesOf(node)) > 0)
			if fullyGuarded && !hasEscape {
				just := o.GetString("_exhaustive")
				if strings.TrimSpace(just) != "" {
					notes = append(notes, base+": state "+p+" always-list is fully guarded; liveness rests on the declared exhaustiveness: "+strings.TrimSpace(just))
				} else {
					errs = append(errs, base+": state "+p+" has a fully guarded always-list and no unguarded escape (after/on/invoke); if no guard is true the machine is stuck. Add an unguarded fallback branch, or an _exhaustive note stating why the guards are total")
				}
			}
		}

		for _, tr := range trs {
			_, why := resolve(tr.Target, p)
			if why != "" {
				errs = append(errs, base+": "+why+" from "+p+" ("+tr.Kind+":"+tr.Event+")")
			}
		}
	}

	for _, pr := range problems {
		errs = append(errs, base+": "+pr)
	}

	// unguarded always cycles loop forever; TLC's liveness property misses
	// them when the loop states classify as domain, so lint must catch them
	alwaysNext := map[string][]string{}
	for _, s := range states {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
			if tr.Kind == "always" && !tr.HasGuard {
				dest, _ := resolve(tr.Target, s.Path)
				if dest != "" {
					alwaysNext[s.Path] = append(alwaysNext[s.Path], dest)
				}
			}
		}
	}
	color := map[string]int{} // 0 unvisited, 1 on stack, 2 done
	var cycStack []string
	var cycles []string
	var dfs func(u string) bool
	dfs = func(u string) bool {
		color[u] = 1
		cycStack = append(cycStack, u)
		for _, v := range alwaysNext[u] {
			if color[v] == 1 {
				i := len(cycStack) - 1
				for i >= 0 && cycStack[i] != v {
					i--
				}
				cyc := append(append([]string{}, cycStack[i:]...), v)
				cycles = append(cycles, strings.Join(cyc, " -> "))
				return true
			}
			if color[v] == 0 && dfs(v) {
				return true
			}
		}
		cycStack = cycStack[:len(cycStack)-1]
		color[u] = 2
		return false
	}
	for _, s := range states { // deterministic source order
		if color[s.Path] == 0 && len(alwaysNext[s.Path]) > 0 {
			if dfs(s.Path) {
				break // one cycle is enough to fail; report deterministically
			}
		}
	}
	for _, c := range cycles {
		errs = append(errs, base+": unguarded always cycle "+c+" (eventless transitions with no guard re-fire forever)")
	}

	// event completeness. It runs over resting LEAF states (nested included):
	// a leaf that is upper-first, non-final, and has no invoke/always on
	// itself or any ancestor sits waiting for external events, wherever it
	// nests. Handlers and _ignores on the ancestor chain are credited (an
	// unhandled event bubbles up in XState).
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
		ignoresVal := o.Get2("_ignores")
		ignores := map[string]string{}
		ignoresValid := true
		if ignoresVal != nil && ignoresVal.Kind == ir.KindObject {
			io := ignoresVal.AsObject()
			for _, k := range io.Keys() {
				v := io.Get2(k)
				if v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
					ignoresValid = false
				} else {
					ignores[k] = v.AsString()
				}
			}
		} else if ignoresVal != nil {
			ignoresValid = false // present but not an object
		}
		if !ignoresValid {
			errs = append(errs, base+": state "+s.Path+" _ignores must map event names to reason strings")
			ignores = map[string]string{} // Python: ignores = {} when invalid
		}
		for _, ev := range sortedKeysStr(ignores) {
			if onSet[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s both handles and ignores event %s", base, s.Path, ir.Repr(ev)))
			}
		}
		igOf[s.Path] = ignores
	}
	sortedAllEvents := sortedSetBool(allEvents)
	for _, s := range states {
		p, n, node := s.Path, s.Name, s.Node
		if node == nil || node.Kind != ir.KindObject {
			continue
		}
		o := node.AsObject()
		if !ir.IsUpperFirst(n) || o.GetString("type") == "final" || o.Get2("states") != nil {
			continue
		}
		chain := ancestorChain(p)
		transient := false
		for _, q := range chain {
			qn := nodeOf[q]
			if qn == nil || qn.Kind != ir.KindObject {
				continue
			}
			qo := qn.AsObject()
			if qo.Get2("invoke") != nil || qo.Get2("always") != nil {
				transient = true
				break
			}
		}
		if transient {
			continue
		}
		handled := map[string]bool{}
		ignored := map[string]bool{}
		for _, q := range chain {
			for ev := range onOf[q] {
				handled[ev] = true
			}
			for ev := range igOf[q] {
				ignored[ev] = true
			}
		}
		for _, ev := range sortedKeysStr(igOf[p]) {
			if !onOf[p][ev] && handled[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s ignores event %s but an ancestor handles it (the handler wins; drop the _ignores entry or hoist the reasoning)",
					base, p, ir.Repr(ev)))
			}
		}
		for _, ev := range sortedAllEvents {
			if !handled[ev] && !ignored[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s neither handles nor explicitly ignores event %s (add it to on: or to _ignores: with a reason)",
					base, p, ir.Repr(ev)))
			}
		}
	}

	// initial + reachability
	init := ro.GetString("initial")
	statesRoot := ro.Get2("states").AsObject()
	if !statesRoot.Has(init) {
		errs = append(errs, fmt.Sprintf("%s: initial %s is not a top-level state", base, ir.Repr(init)))
	} else {
		reached := map[string]bool{}
		var enter func(p string)
		enter = func(p string) {
			if reached[p] {
				return
			}
			reached[p] = true
			node := nodeOf[p]
			if node != nil && node.Kind == ir.KindObject {
				o := node.AsObject()
				if o.Get2("states") != nil {
					childInit := o.GetString("initial")
					if childInit != "" {
						child := p + "." + childInit
						if pathSet[child] {
							enter(child)
						}
					}
				}
			}
		}
		enter(init)
		frontier := true
		for frontier {
			frontier = false
			active := map[string]bool{}
			for p := range reached {
				active[p] = true
				q := p
				for strings.Contains(q, ".") {
					q = q[:strings.LastIndex(q, ".")]
					active[q] = true
				}
			}
			for p := range active {
				for _, tr := range ir.TransitionsOf(nodeOf[p], nil, p) {
					dest, why := resolve(tr.Target, p)
					_ = why
					if dest != "" && !reached[dest] {
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
			// not reachable directly; check if reachable via containment overlap
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
	}

	// Named-unit names (guards, actions, actors) become TLA+/oracle identifiers
	// and are matched against the contract table by the IDENT regex. A hyphen
	// passes every structural check above but can never be extracted from the
	// contract, so G3 reports a phantom "no named-unit contract row". Reject it
	// here, at the source, with an actionable message.
	guards, actions, actors := MachineUnitNames(m)
	for _, u := range []struct {
		kind  string
		names map[string]bool
	}{{"guard", guards}, {"action", actions}, {"actor", actors}} {
		for _, name := range sortedKeysMap(u.names) {
			if !regexpIdent.MatchString(name) {
				errs = append(errs, fmt.Sprintf("%s: %s %s is not a valid identifier (must match [A-Za-z_][A-Za-z0-9_]*); rename to camelCase (named units become TLA+/oracle identifiers, so hyphens are not allowed)",
					base, u.kind, ir.Repr(name)))
			}
		}
	}

	return errs, warns, notes, counts
}

func srcRepr(ivObj *ir.Object) string {
	if s := ivObj.Get2("src"); s != nil {
		return ir.Repr(s.AsString())
	}
	return ir.Repr(nil)
}

type normBranchRec struct {
	HasGuard bool
	Guard    string
	Target   string
	HasTgt   bool
}

func normBranches(t *ir.Value) []normBranchRec {
	var items []*ir.Value
	if t == nil {
		return nil
	}
	if t.Kind == ir.KindArray {
		items = t.AsArray()
	} else {
		items = []*ir.Value{t}
	}
	var out []normBranchRec
	for _, it := range items {
		if it == nil || it.Kind != ir.KindObject {
			out = append(out, normBranchRec{})
			continue
		}
		o := it.AsObject()
		b := normBranchRec{}
		// an empty guard string is NOT a guard (ir.normTransition reports it);
		// counting it as guarded would let a fully-guarded always-list pass
		// the exhaustiveness check on a vacuous guard
		if gv := o.Get2("guard"); gv != nil && gv.Kind == ir.KindString && gv.AsString() != "" {
			b.HasGuard = true
			b.Guard = gv.AsString()
		}
		if tv := o.Get2("target"); tv != nil && tv.Kind == ir.KindString {
			b.Target = tv.AsString()
			b.HasTgt = true
		}
		out = append(out, b)
	}
	return out
}

func sortedKeysMap(m map[string]bool) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedKeysStr(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedSetBool(m map[string]bool) []string { return sortedKeysMap(m) }

// --- matrix reconciliation ---

// MRow is a canonical machine transition row.
type MRow struct {
	Source     string
	Trigger    string
	Guard      string
	HasGuard   bool
	Fallback   bool
	FirstGuard string
	Target     string
	Actions    []string
}

// XRow is a canonical matrix transition row.
type XRow struct {
	Source  string
	Trigger string
	Guard   string
	Target  string
	Actions []string
}

// MachineTransitionRows mirrors machine_lint.machine_transition_rows.
// Sources and targets are FULL dotted paths: two same-named states under
// different parents must never cross-match a matrix row (the matrix may still
// write a simple name when it is unambiguous; ReconcileMatrix handles that).
func MachineTransitionRows(m *ir.Value) []MRow {
	states := ir.WalkStates(m.AsObject().Get2("states"), "")
	res := newResolver(m)
	var rows []MRow
	for _, s := range states {
		groups := map[string][]ir.Transition{}
		var trigOrder []string
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
			var trig string
			switch tr.Kind {
			case "on":
				trig = tr.Event
			case "after":
				trig = "after " + tr.Event
			case "always":
				trig = "always"
			case "stateDone":
				trig = "onDone"
			case "onDone":
				trig = "invoke onDone"
			case "onError":
				trig = "invoke onError"
			}
			if _, ok := groups[trig]; !ok {
				trigOrder = append(trigOrder, trig)
			}
			groups[trig] = append(groups[trig], tr)
		}
		for _, trig := range trigOrder {
			trs := groups[trig]
			for i, tr := range trs {
				var tgt string
				if tr.Target != "" {
					if dest, why := res.resolve(tr.Target, s.Path); why == "" && dest != "" {
						tgt = dest
					} else {
						tgt = ir.Simple(tr.Target) // dangling; lint reports it separately
					}
				} else {
					tgt = "(internal)"
				}
				row := MRow{
					Source:     s.Path,
					Trigger:    trig,
					Guard:      tr.Guard,
					HasGuard:   tr.HasGuard,
					Fallback:   !tr.HasGuard && i > 0,
					FirstGuard: firstGuard(trs),
					Target:     tgt,
					Actions:    append([]string{}, tr.Actions...),
				}
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func firstGuard(trs []ir.Transition) string {
	if len(trs) == 0 {
		return ""
	}
	return trs[0].Guard
}

// MatrixTransitionRows mirrors machine_lint.matrix_transition_rows.
// EVERY table whose header matches the transition-table shape is reconciled
// (rows concatenated); reconciling only the first let contradicting rows in a
// second table pass silently. Table selection uses the hardened FindCol
// label matching, so a prose table whose header cells merely contain
// "resource" / "retarget" / "actions needed" is not a transition table.
func MatrixTransitionRows(text string) ([]XRow, string) {
	var out []XRow
	matched := false
	for _, tbl := range ir.ParseMdTables(text) {
		si := ir.FindCol(tbl.Header, "source")
		ti := ir.FindCol(tbl.Header, "target")
		ai := ir.FindCol(tbl.Header, "actions")
		if si < 0 || ti < 0 || ai < 0 {
			continue
		}
		matched = true
		if out == nil {
			// Non-nil even when every row is filtered out: a present-but-empty
			// table must reconcile (and drift), not read as "no table".
			out = []XRow{}
		}
		ei := ir.FindCol(tbl.Header, "event", "trigger")
		gi := ir.FindCol(tbl.Header, "guard")
		if ei < 0 || gi < 0 {
			return nil, "transition table found but a required column is missing (need source, event/trigger, guard, target, actions)"
		}
		for _, r := range tbl.Rows {
			maxI := si
			for _, idx := range []int{ei, gi, ti, ai} {
				if idx > maxI {
					maxI = idx
				}
			}
			if len(r) <= maxI {
				return nil, "transition table row has too few cells: " + ir.Repr(strings.Join(r, "|"))
			}
			// Documentation-only marker rows (and ONLY those) are excluded:
			// the whole trigger cell is "(final)" or "(any event)". A "(final)"
			// annotation elsewhere in the row still reconciles, so a
			// contradicting row cannot hide behind the marker.
			trigRaw := strings.TrimSpace(r[ei])
			if trigRaw == "(final)" || trigRaw == "(any event)" {
				continue
			}
			src := ir.CleanCell(r[si])
			trig := ir.CleanCell(r[ei])
			guard := ir.CleanCell(r[gi])
			tgtRaw := r[ti]
			var tgt string
			if strings.Contains(tgtRaw, "(internal)") {
				tgt = "(internal)"
			} else {
				tgt = ir.CleanCell(tgtRaw)
			}
			acts := ir.CleanCell(r[ai])
			var actions []string
			if acts != "" && acts != "-" {
				for _, a := range strings.Split(acts, ",") {
					a = strings.TrimSpace(a)
					if a != "" {
						actions = append(actions, a)
					}
				}
			}
			out = append(out, XRow{Source: src, Trigger: trig, Guard: guard, Target: tgt, Actions: actions})
		}
	}
	if matched {
		return out, ""
	}
	// No parsed table matched. If a line still reads as a transition-table
	// HEADER, the table exists but failed to parse (e.g. a malformed separator
	// row that split the block); silence here let contradicting rows pass as
	// "no table". A broken table is a hard error, not an absence. Matching is
	// per-cell label-prefix, never substring-in-line: a prose data row
	// containing "resource", "retarget", or "transactions" is not a header.
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		found := map[string]bool{}
		for _, c := range strings.Split(strings.Trim(trimmed, "|"), "|") {
			cell := strings.ToLower(strings.TrimSpace(ir.CleanCell(c)))
			for _, k := range []string{"source", "target", "actions"} {
				if cell == k || strings.HasPrefix(cell, k+" ") {
					found[k] = true
				}
			}
		}
		if found["source"] && found["target"] && found["actions"] {
			return nil, "a transition-table header is present but no transition table parsed (malformed separator row?); fix the table instead of leaving it unparseable"
		}
	}
	return nil, ""
}

func guardMatches(mr MRow, cell string) bool {
	if mr.HasGuard {
		return cell == mr.Guard
	}
	if mr.Fallback {
		accepted := map[string]bool{"-": true, "(else)": true, "else": true, "": true}
		if mr.FirstGuard != "" {
			accepted["!"+mr.FirstGuard] = true
		}
		return accepted[cell]
	}
	return cell == "-" || cell == ""
}

// ReconcileMatrix mirrors machine_lint.reconcile_matrix. Parse failures (a
// table that looks like a transition table but cannot be reconciled) come
// back as errs; row mismatches come back as drift. Machine rows carry full
// dotted paths; a matrix cell may use a simple state name only when it is
// unambiguous in the machine, otherwise the author must qualify it.
func ReconcileMatrix(m *ir.Value, matrixText, base string) (errs, drift []string, nrows int) {
	mrows := MachineTransitionRows(m)
	xrows, perr := MatrixTransitionRows(matrixText)
	if perr != "" {
		return []string{base + ": " + perr}, nil, 0
	}
	if xrows == nil {
		return nil, nil, 0
	}
	bySimple := map[string][]string{}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		bySimple[s.Name] = append(bySimple[s.Name], s.Path)
	}
	ambig := map[string][]string{}
	for _, x := range xrows {
		for _, cell := range []string{x.Source, x.Target} {
			if cell == "" || cell == "(internal)" || strings.Contains(cell, ".") {
				continue
			}
			if c := bySimple[cell]; len(c) > 1 {
				ambig[cell] = c
			}
		}
	}
	if len(ambig) > 0 {
		var names []string
		for name := range ambig {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			errs = append(errs, fmt.Sprintf("%s: ambiguous state name %s in the matrix (candidates: %s); qualify it with the full dotted path",
				base, ir.Repr(name), strings.Join(ambig[name], ", ")))
		}
		return errs, nil, 0
	}
	stateEq := func(cell, path string) bool {
		if cell == path {
			return true
		}
		c := bySimple[cell]
		return len(c) == 1 && c[0] == path
	}
	unmatched := make([]int, len(xrows))
	for i := range unmatched {
		unmatched[i] = i
	}
	trigEq := func(machineTrig, cell string) bool {
		return cell == machineTrig || cell == "on:"+machineTrig
	}
	take := func(pred func(x XRow) bool) (int, bool) {
		for _, k := range unmatched {
			if pred(xrows[k]) {
				return k, true
			}
		}
		return -1, false
	}
	for _, mr := range mrows {
		mrLocal := mr
		idx, ok := take(func(x XRow) bool {
			tgtOK := false
			if mrLocal.Target == "(internal)" {
				tgtOK = x.Target == "(internal)"
			} else {
				tgtOK = stateEq(x.Target, mrLocal.Target)
			}
			return stateEq(x.Source, mrLocal.Source) && trigEq(mrLocal.Trigger, x.Trigger) &&
				guardMatches(mrLocal, x.Guard) && tgtOK &&
				eqActions(x.Actions, mrLocal.Actions)
		})
		if !ok {
			g := mr.Guard
			if !mr.HasGuard {
				if mr.Fallback {
					g = "else"
				} else {
					g = "-"
				}
			}
			actsStr := "-"
			if len(mr.Actions) > 0 {
				actsStr = strings.Join(mr.Actions, ", ")
			}
			drift = append(drift, fmt.Sprintf("%s: machine transition has no matrix row: %s --%s [%s]--> %s / %s",
				base, mr.Source, mr.Trigger, g, mr.Target, actsStr))
			continue
		}
		// remove idx from unmatched
		for i, k := range unmatched {
			if k == idx {
				unmatched = append(unmatched[:i], unmatched[i+1:]...)
				break
			}
		}
	}
	for _, k := range unmatched {
		x := xrows[k]
		g := x.Guard
		if g == "" {
			g = "-"
		}
		actsStr := "-"
		if len(x.Actions) > 0 {
			actsStr = strings.Join(x.Actions, ", ")
		}
		drift = append(drift, fmt.Sprintf("%s: matrix row has no machine transition: %s --%s [%s]--> %s / %s",
			base, x.Source, x.Trigger, g, x.Target, actsStr))
	}
	return nil, drift, len(mrows)
}

func eqActions(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NamedUnitNames mirrors machine_lint.namedunit_names.
func NamedUnitNames(matrixText string) map[string]bool {
	names := map[string]bool{}
	for _, tbl := range ir.ParseMdTables(matrixText) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if strings.Contains(hl, "signature") || (strings.Contains(hl, "name") && strings.Contains(hl, "kind")) {
			ni := ir.FindCol(tbl.Header, "name")
			if ni < 0 {
				ni = 0
			}
			for _, r := range tbl.Rows {
				cell := ""
				if ni < len(r) {
					cell = r[ni]
				}
				for _, m := range regexpBacktickGroup.FindAllStringSubmatch(cell, -1) {
					names[m[1]] = true
				}
				if !strings.Contains(cell, "`") {
					// unbacktick'd fallback: only a cell that IS a single
					// IDENT token is a name; harvesting every word from a
					// prose cell over-collects (Finding IR-F28)
					if w := strings.TrimSpace(ir.CleanCell(cell)); regexpIdent.MatchString(w) {
						names[w] = true
					}
				}
			}
		}
	}
	return names
}

// MachineUnitNames mirrors machine_lint.machine_unit_names.
func MachineUnitNames(m *ir.Value) (guards, actions, actors map[string]bool) {
	guards, actions, actors = map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
			if tr.HasGuard {
				guards[tr.Guard] = true
			}
			for _, a := range tr.Actions {
				actions[a] = true
			}
		}
		for _, n := range ir.ActionNames(entryExit(s.Node, "entry"), nil, "") {
			actions[n] = true
		}
		for _, n := range ir.ActionNames(entryExit(s.Node, "exit"), nil, "") {
			actions[n] = true
		}
		for _, iv := range ir.InvokesOf(s.Node) {
			if src := iv.AsObject().Get2("src"); src != nil && src.Kind == ir.KindString {
				actors[src.AsString()] = true
			}
		}
	}
	return
}

func entryExit(node *ir.Value, key string) *ir.Value {
	if node == nil || node.Kind != ir.KindObject {
		return nil
	}
	return node.AsObject().Get2(key)
}

// Lint mirrors machine_lint.lint(path): loads machine + matrix, returns summary.
func Lint(path string) (nStates int, errs, warns, drift []string) {
	m, err := ir.LoadMachineJSON(path)
	base := filepath.Base(path)
	if err != nil {
		return 0, []string{err.Error()}, nil, nil
	}
	e, w, notes, counts := LintMachine(m, base)
	mx := path
	mx = strings.TrimSuffix(mx, filepath.Ext(mx)) // remove .json
	mx = strings.TrimSuffix(mx, ".machine") + ".matrix.md"
	if _, statErr := os.Stat(mx); statErr == nil {
		data, readErr := os.ReadFile(mx)
		if readErr != nil {
			// a matrix that exists but cannot be read must never reconcile
			// as "no table present" (that is a silent pass)
			e = append(e, fmt.Sprintf("%s: cannot read matrix %s: %v", base, filepath.Base(mx), readErr))
		} else {
			es, d, _ := ReconcileMatrix(m, string(data), base)
			e = append(e, es...)
			drift = d
		}
	} else {
		w = append(w, base+": no matrix file; named-unit contracts are unchecked (the generated oracle still covers transitions)")
	}
	return counts.States, e, append(w, notes...), drift
}

// Run is the `machinery lint <dir>` entrypoint.
func Run(mdir string, out, errw *os.File) int {
	entries, _ := os.ReadDir(mdir)
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".machine.json") {
			files = append(files, filepath.Join(mdir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(out, "ERROR  no *.machine.json under %s: nothing to lint is a failure, not a pass\n", mdir)
		return 1
	}
	total := 0
	for _, f := range files {
		n, errs, warns, drift := Lint(f)
		fmt.Fprintf(out, "== %s: %d states ==\n", filepath.Base(f), n)
		for _, e := range errs {
			fmt.Fprintf(out, "  ERROR  %s\n", e)
		}
		for _, d := range drift {
			fmt.Fprintf(out, "  DRIFT  %s\n", d)
		}
		for _, w := range warns {
			fmt.Fprintf(out, "  warn   %s\n", w)
		}
		if len(errs) == 0 && len(drift) == 0 && len(warns) == 0 {
			fmt.Fprintln(out, "  ok")
		}
		total += len(errs) + len(drift)
	}
	fmt.Fprintf(out, "\n%d error/drift finding(s) across %d machine(s)\n", total, len(files))
	if total > 0 {
		return 1
	}
	return 0
}
