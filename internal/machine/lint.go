package machine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// Supported keys of the XState v5 subset (docs/machine-format.md). Any other
// key is an ERROR, never silently skipped.
var (
	rootKeys = setOf("id", "initial", "context", "states", "description", "meta", "version",
		"_comment", "_delays", "_lifecycle_of", "_role", "_oracle_tag", "_guards", "_external_events",
		"_entries")
	stateKeys = setOf("on", "after", "always", "invoke", "entry", "exit", "states", "initial",
		"type", "id", "meta", "description", "tags", "onDone", "output", "_comment",
		"_exhaustive", "_ignores", "_refusal", "_terminal")
	invokeKeys = setOf("src", "input", "id", "onDone", "onError", "_comment")
	stateTypes = setOf("", "atomic", "compound", "final")
)

var (
	reIdent     = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_]*$")
	reName      = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]*$")
	reOracleTag = regexp.MustCompile("^[A-Z0-9]{2,8}$")
	reDigits    = regexp.MustCompile("^[0-9]+$")
	reClause    = regexp.MustCompile("^[a-z][A-Za-z0-9]*$")
)

func setOf(keys ...string) map[string]bool {
	m := map[string]bool{}
	for _, k := range keys {
		m[k] = true
	}
	return m
}

// Counts is what the machines stage reports as checked.
type Counts struct {
	States      int
	Transitions int
}

// Lint holds one machine to the rules of docs/machine-format.md. Every finding
// starts with the machine's project-relative path.
func (m *Machine) Lint() (errs []string, counts Counts) {
	base := m.Path
	ro := m.Root.AsObject()
	if ro == nil {
		return []string{base + ": machine is not an object"}, counts
	}
	for _, k := range ro.Keys() {
		if !rootKeys[k] {
			errs = append(errs, fmt.Sprintf("%s: unsupported root key %s (supported: %s)",
				base, ir.Repr(k), strings.Join(sortedSet(rootKeys), ", ")))
		}
	}
	statesVal := ro.Get2("states")
	if statesVal == nil || statesVal.Kind != ir.KindObject || statesVal.AsObject().Len() == 0 {
		return append(errs, base+": machine has no states"), counts
	}
	states := ir.WalkStates(statesVal, "")
	counts.States = len(states)
	res := m.Resolver
	nodeOf := res.NodeOf
	for _, e := range res.IDErrs {
		errs = append(errs, base+": "+e)
	}
	resolve := res.Resolve

	if tagV := ro.Get2("_oracle_tag"); tagV != nil {
		if tagV.Kind != ir.KindString || !reOracleTag.MatchString(tagV.AsString()) {
			errs = append(errs, fmt.Sprintf("%s: _oracle_tag %s must be 2-8 uppercase characters matching [A-Z0-9]", base, ir.Repr(tagV)))
		}
	}
	errs = append(errs, lintExternalEvents(base, ro, states)...)
	errs = append(errs, lintEntries(base, ro)...)
	declaredDelays, usedDelays := map[string]bool{}, map[string]bool{}
	if dv := ro.Get2("_delays"); dv != nil && dv.Kind == ir.KindObject {
		for _, k := range dv.AsObject().Keys() {
			declaredDelays[k] = true
		}
	} else if dv != nil {
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
		if !reName.MatchString(s.Name) {
			errs = append(errs, fmt.Sprintf("%s: state name %s must match [A-Za-z][A-Za-z0-9_]* (names are hashed into case ids: use letters, digits and underscores)",
				base, ir.Repr(s.Name)))
		}
		for _, k := range o.Keys() {
			if !stateKeys[k] {
				errs = append(errs, fmt.Sprintf("%s: unsupported key %s in state %s", base, ir.Repr(k), p))
			}
		}
		// a null or mistyped container is malformed input, not "no transitions"
		for _, k := range []string{"on", "after", "_ignores"} {
			if v := o.Get2(k); v != nil && v.Kind != ir.KindObject {
				errs = append(errs, fmt.Sprintf("%s: state %s: %s must be an object, not null or another type", base, p, ir.Repr(k)))
			}
		}
		if v := o.Get2("invoke"); v != nil && v.Kind != ir.KindObject && v.Kind != ir.KindArray {
			errs = append(errs, fmt.Sprintf("%s: state %s: 'invoke' must be an object or array, not null or another type", base, p))
		}
		stype := o.GetString("type")
		if !stateTypes[stype] {
			errs = append(errs, fmt.Sprintf("%s: unsupported state type %s in %s (parallel and history are not in the supported subset)",
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
		if init := o.GetString("initial"); init != "" && o.Get2("states") != nil && !o.Get2("states").AsObject().Has(init) {
			errs = append(errs, fmt.Sprintf("%s: compound state %s initial %s is not one of its children", base, p, ir.Repr(init)))
		}
		// onDone fires only when a compound state reaches a final descendant
		if o.Get2("onDone") != nil {
			if o.Get2("states") == nil {
				errs = append(errs, base+": state "+p+" has onDone but no child states (onDone fires when a compound state reaches a final child; this one never fires)")
			} else if !hasFinalDescendant(node) {
				errs = append(errs, base+": state "+p+" has onDone but no final descendant state, so its onDone can never fire")
			}
		}
		// dead end or absorbing: a non-final leaf needs a transition that
		// leaves it, on itself or on an ancestor
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
					if dest, why := resolve(tr.Target, q); why != "" || dest != p {
						exits = true
						break
					}
				}
				if exits {
					break
				}
			}
			_, terminal := o.Get("_terminal")
			switch {
			case !exits && len(trs) == 0 && terminal:
				errs = append(errs, base+": dead-end non-final leaf state "+p+" (_terminal is declared, but a terminal state still answers events and this one has no transition)")
			case !exits && len(trs) == 0:
				errs = append(errs, base+": dead-end non-final leaf state "+p)
			case !exits && !terminal:
				errs = append(errs, base+": absorbing non-final leaf state "+p+" (transitions exist but none leaves the state; add an exiting transition, make it final, or declare _terminal with the reason)")
			case exits && terminal:
				errs = append(errs, base+": state "+p+" declares _terminal but a transition leaves it; the claim is stale (drop _terminal, or the exiting transition)")
			}
		}
		errs = append(errs, lintTerminal(base, p, o, isFinal)...)
		for _, iv := range ir.InvokesOf(node) {
			if iv == nil || iv.Kind != ir.KindObject {
				errs = append(errs, fmt.Sprintf("%s: state %s: invoke entries must be objects, got %s", base, p, ir.Repr(iv)))
				continue
			}
			ivObj := iv.AsObject()
			for _, k := range ivObj.Keys() {
				if !invokeKeys[k] {
					errs = append(errs, fmt.Sprintf("%s: unsupported invoke key %s in state %s", base, ir.Repr(k), p))
				}
			}
			if src := ivObj.Get2("src"); src == nil || src.Kind != ir.KindString || src.AsString() == "" {
				errs = append(errs, fmt.Sprintf("%s: invoke in %s has no src (src is a mandatory string naming the invoked actor)", base, p))
			}
			if ivObj.Get2("onError") == nil {
				errs = append(errs, fmt.Sprintf("%s: invoke %s in %s has no onError", base, srcRepr(ivObj), p))
			}
		}
		if o.Get2("invoke") != nil && len(ir.InvokesOf(node)) > 0 {
			if o.Get2("after") == nil {
				errs = append(errs, base+": invoking state "+p+" has no after/timeout")
			} else if !hasEntries(o.Get2("after")) && o.Get2("after").Kind == ir.KindObject {
				errs = append(errs, base+": invoking state "+p+" has an empty after block (a timeout delay is required; after: {} declares nothing)")
			}
		}
		if after := o.Get2("after"); after != nil && after.Kind == ir.KindObject {
			for _, delay := range after.AsObject().Keys() {
				usedDelays[delay] = true
				if reDigits.MatchString(delay) {
					errs = append(errs, fmt.Sprintf("%s: state %s after key %s is a raw millisecond value; name the delay and declare its bound in _delays", base, p, ir.Repr(delay)))
				} else if !declaredDelays[delay] {
					errs = append(errs, fmt.Sprintf("%s: state %s after delay %s is not declared in _delays (named delays carry their ms bound and rationale there)", base, p, ir.Repr(delay)))
				}
			}
		}
		ir.ActionNames(o.Get2("entry"), &problems, p+" entry")
		ir.ActionNames(o.Get2("exit"), &problems, p+" exit")
		errs = append(errs, lintHandlers(base, p, node, trs, res)...)
	}

	// a declared delay no after edge consumes is a cadence nothing implements
	for _, d := range sortedSet(declaredDelays) {
		if !usedDelays[d] {
			errs = append(errs, fmt.Sprintf("%s: _delays declares %s but no after edge consumes it; a declared cadence nothing implements is not a bound (wire it to an after, or drop it)", base, ir.Repr(d)))
		}
	}
	for _, pr := range problems {
		errs = append(errs, base+": "+pr)
	}
	errs = append(errs, lintAlwaysCycles(base, states, resolve)...)
	errs = append(errs, lintEventCompleteness(base, states, nodeOf)...)
	errs = append(errs, lintReachability(base, ro, states, res, m.Entries())...)

	// guards, actions and actors are implemented in code under these names
	for _, u := range []struct {
		kind  string
		names []string
	}{{"guard", m.Table.Units.Guards}, {"action", m.Table.Units.Actions}, {"actor", m.Table.Units.Actors}} {
		for _, name := range u.names {
			if !reIdent.MatchString(name) {
				errs = append(errs, fmt.Sprintf("%s: %s %s is not a valid identifier (must match [A-Za-z_][A-Za-z0-9_]*); rename to camelCase, because named units are implemented in code",
					base, u.kind, ir.Repr(name)))
			}
		}
	}
	errs = append(errs, lintGuards(base, ro, m.Table.Units.Guards)...)
	errs = append(errs, lintFailClosed(base, ro, states, res)...)
	return errs, counts
}

func srcRepr(ivObj *ir.Object) string {
	if s := ivObj.Get2("src"); s != nil {
		return ir.Repr(s.AsString())
	}
	return ir.Repr(nil)
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

// hasFinalDescendant reports whether any state below node is final.
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

// hasEntries reports a present object with at least one member.
func hasEntries(v *ir.Value) bool {
	return v != nil && v.Kind == ir.KindObject && v.AsObject().Len() > 0
}

type branch struct {
	HasGuard bool
}

func branchesOf(t *ir.Value) []branch {
	if t == nil {
		return nil
	}
	items := []*ir.Value{t}
	if t.Kind == ir.KindArray {
		items = t.AsArray()
	}
	var out []branch
	for _, it := range items {
		b := branch{}
		if it != nil && it.Kind == ir.KindObject {
			// an empty guard string is not a guard (the normalizer reports it)
			if gv := it.AsObject().Get2("guard"); gv != nil && gv.Kind == ir.KindString && gv.AsString() != "" {
				b.HasGuard = true
			}
		}
		out = append(out, b)
	}
	return out
}

func guardOf(t *ir.Value, i int) string {
	items := []*ir.Value{t}
	if t.Kind == ir.KindArray {
		items = t.AsArray()
	}
	if i < len(items) && items[i] != nil && items[i].Kind == ir.KindObject {
		return items[i].AsObject().GetString("guard")
	}
	return ""
}

func sortedSet(m map[string]bool) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedMapKeys(m map[string]string) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
