package machine

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// tagFailure marks a state that records that something went wrong
// (docs/formats.md section 5.2).
const tagFailure = "failure"

// FailureStates returns the dotted paths of the states tagged failure, in
// document order.
func (m *Machine) FailureStates() []string {
	ro := m.Root.AsObject()
	if ro == nil {
		return nil
	}
	var res []string
	for _, s := range ir.WalkStates(ro.Get2("states"), "") {
		if tags, _ := stateTags(s.Node); tags[tagFailure] {
			res = append(res, s.Path)
		}
	}
	return res
}

// FailureEvents returns the root _failure_events: the events that report a
// failure arriving from outside, in declaration order, without duplicates.
func (m *Machine) FailureEvents() []string { return eventList(m.Root.AsObject(), "_failure_events") }

// eventList reads a root list of event names, keeping the non-empty strings
// in declaration order, each once. Shape errors are lintEventList's.
func eventList(ro *ir.Object, key string) []string {
	if ro == nil {
		return nil
	}
	var res []string
	seen := map[string]bool{}
	for _, e := range ro.Get2(key).AsArray() {
		name := strings.TrimSpace(e.AsString())
		if name != "" && !seen[name] {
			seen[name] = true
			res = append(res, name)
		}
	}
	return res
}

// lintEventList holds a root list of event names (_external_events,
// _failure_events): an array of non-empty names, each of which some state
// handles or ignores. An entry nothing answers is a stale marker.
func lintEventList(base string, ro *ir.Object, states []ir.StateEntry, key, purpose string) (errs []string) {
	ev := ro.Get2(key)
	if ev == nil {
		return nil
	}
	if ev.Kind != ir.KindArray {
		return []string{base + ": " + key + " must be an array of event-name strings " + purpose}
	}
	handled := map[string]bool{}
	for _, s := range states {
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				handled[k] = true
			}
		}
	}
	for _, e := range ev.AsArray() {
		name := ""
		if e != nil && e.Kind == ir.KindString {
			name = strings.TrimSpace(e.AsString())
		}
		if name == "" {
			errs = append(errs, base+": "+key+" entries must be non-empty event-name strings")
			continue
		}
		if !handled[name] {
			errs = append(errs, fmt.Sprintf("%s: %s names %s but no state handles or ignores it; the marker is stale (remove it, or wire the event)", base, key, ir.Repr(name)))
		}
	}
	return errs
}

// Aliases returns the root _aliases: the events that are a model action
// under another name, each with the name of its action. Entries that are not
// a non-empty string are left out; their shape is lintAliases's.
func (m *Machine) Aliases() map[string]string { return eventMap(m.Root.AsObject(), "_aliases") }

// RefusalEvents returns the root _refusal_events: the events by which the
// machine writes that it refused a caller's command, each with the name of
// the action it refuses. Entries of another shape are left out; their shape
// is lintRefusalEvents's.
func (m *Machine) RefusalEvents() map[string]string {
	return eventMap(m.Root.AsObject(), "_refusal_events")
}

// eventMap reads a root object that maps an event to an action name,
// keeping the entries whose value is a non-empty string, trimmed.
func eventMap(o *ir.Object, key string) map[string]string {
	res := map[string]string{}
	if o == nil {
		return res
	}
	al := o.GetObject(key)
	for _, ev := range al.Keys() {
		if v := al.Get2(ev); v != nil && v.Kind == ir.KindString && strings.TrimSpace(v.AsString()) != "" {
			res[ev] = strings.TrimSpace(v.AsString())
		}
	}
	return res
}

// handledEvents returns the events some state handles under on or ignores.
func handledEvents(states []ir.StateEntry) map[string]bool {
	handled := map[string]bool{}
	for _, s := range states {
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				handled[k] = true
			}
		}
	}
	return handled
}

// lintRefusalEvents holds the root _refusal_events: an object that maps an
// event some state handles or ignores to a non-empty action name. The
// machine writes a refusal itself, in answer to a caller, so the event is
// not external; it is declared and no failure, so it is no failure event;
// and it refuses an action, so it is not that action under another name.
func lintRefusalEvents(base string, ro *ir.Object, states []ir.StateEntry) (errs []string) {
	rv := ro.Get2("_refusal_events")
	if rv == nil {
		return nil
	}
	if rv.Kind != ir.KindObject {
		return []string{base + ": _refusal_events must be an object that maps an event name to the name of the model action it refuses"}
	}
	handled := handledEvents(states)
	other := map[string]string{}
	for _, e := range eventList(ro, "_external_events") {
		other[e] = "_external_events; the machine writes a refusal itself, so it does not arrive from outside"
	}
	for _, e := range eventList(ro, "_failure_events") {
		other[e] = "_failure_events; a refusal is declared, not a failure"
	}
	for _, e := range ro.GetObject("_aliases").Keys() {
		if other[e] == "" {
			other[e] = "_aliases; an event refuses an action or is one under another name"
		}
	}
	for _, ev := range rv.AsObject().Keys() {
		if v := rv.AsObject().Get2(ev); v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
			errs = append(errs, fmt.Sprintf("%s: _refusal_events[%s] must be a non-empty action name", base, ir.Repr(ev)))
			continue
		}
		if !handled[ev] {
			errs = append(errs, fmt.Sprintf("%s: _refusal_events names %s but no state handles or ignores it; the marker is stale (remove it, or wire the event)", base, ir.Repr(ev)))
			continue
		}
		if why := other[ev]; why != "" {
			errs = append(errs, fmt.Sprintf("%s: event %s is in both _refusal_events and %s (keep one)", base, ir.Repr(ev), why))
		}
	}
	return errs
}

// lintAliases holds the root _aliases: an object that maps an event some
// state handles or ignores to a non-empty action name. An aliased event is
// a caller's action, so it is not in _external_events. It may be a failure
// event: the action it stands for is then itself the failing outcome.
func lintAliases(base string, ro *ir.Object, states []ir.StateEntry) (errs []string) {
	av := ro.Get2("_aliases")
	if av == nil {
		return nil
	}
	if av.Kind != ir.KindObject {
		return []string{base + ": _aliases must be an object that maps an event name to the name of the model action it is"}
	}
	handled := map[string]bool{}
	for _, s := range states {
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				handled[k] = true
			}
		}
	}
	outside := map[string]string{}
	for _, e := range eventList(ro, "_external_events") {
		outside[e] = "_external_events"
	}
	for _, ev := range av.AsObject().Keys() {
		if v := av.AsObject().Get2(ev); v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
			errs = append(errs, fmt.Sprintf("%s: _aliases[%s] must be a non-empty action name", base, ir.Repr(ev)))
			continue
		}
		if !handled[ev] {
			errs = append(errs, fmt.Sprintf("%s: _aliases names %s but no state handles or ignores it; the marker is stale (remove it, or wire the event)", base, ir.Repr(ev)))
			continue
		}
		if key := outside[ev]; key != "" {
			errs = append(errs, fmt.Sprintf("%s: event %s is in both _aliases and %s; an alias says the event is a caller's action, so it does not arrive from outside (keep one)", base, ir.Repr(ev), key))
		}
	}
	return errs
}

// notFailures returns the root _not_failures: the event names and state
// paths whose name hint is dismissed, each with a non-empty reason.
func notFailures(ro *ir.Object) map[string]bool {
	res := map[string]bool{}
	nf := ro.GetObject("_not_failures")
	for _, n := range nf.Keys() {
		if v := nf.Get2(n); v != nil && v.Kind == ir.KindString && strings.TrimSpace(v.AsString()) != "" {
			res[n] = true
		}
	}
	return res
}

// lintNotFailures holds the root _not_failures: an object that maps an
// event some state handles or ignores, or the dotted path of a state, to
// the reason it is no failure. A declared failure is not dismissed.
func lintNotFailures(base string, ro *ir.Object, states []ir.StateEntry) (errs []string) {
	nv := ro.Get2("_not_failures")
	if nv == nil {
		return nil
	}
	if nv.Kind != ir.KindObject {
		return []string{base + ": _not_failures must be an object that maps an event name or a state path to the reason it is no failure"}
	}
	handled, isState, tagged := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range states {
		isState[s.Path] = true
		if tags, _ := stateTags(s.Node); tags[tagFailure] {
			tagged[s.Path] = true
		}
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				handled[k] = true
			}
		}
	}
	failure := map[string]bool{}
	for _, e := range eventList(ro, "_failure_events") {
		failure[e] = true
	}
	refusal := eventMap(ro, "_refusal_events")
	for _, n := range nv.AsObject().Keys() {
		v := nv.AsObject().Get2(n)
		switch {
		case v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "":
			errs = append(errs, fmt.Sprintf("%s: _not_failures[%s] must be a non-empty reason", base, ir.Repr(n)))
		case !handled[n] && !isState[n]:
			errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which is no event a state handles or ignores and no state; the marker is stale (remove it, or correct the name)", base, ir.Repr(n)))
		case !looksLikeFailure(n[strings.LastIndex(n, ".")+1:]):
			// no hint is printed for the name, so the entry dismisses nothing
			errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which does not look like a failure (%s), so there is no hint to dismiss (remove it)", base, ir.Repr(n), ir.NoFailureWord))
		case handled[n] && !isState[n] && refusal[n] != "":
			errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which is in _refusal_events and is never hinted, so there is no hint to dismiss (remove it)", base, ir.Repr(n)))
		default:
			if handled[n] && failure[n] {
				errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which is in _failure_events; a declared failure is not dismissed (keep one)", base, ir.Repr(n)))
			}
			if tagged[n] {
				errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which is a state tagged failure; a declared failure is not dismissed (keep one)", base, ir.Repr(n)))
			}
		}
	}
	return errs
}

// lintFailure holds the failure states and events: no state is tagged both
// failure and accepting, _failure_events is a well-formed list of answered
// events, and no event is in both _external_events and _failure_events.
func lintFailure(base string, ro *ir.Object, states []ir.StateEntry) (errs []string) {
	for _, s := range states {
		if tags, ok := stateTags(s.Node); ok && tags[tagFailure] && tags[tagAccepting] {
			errs = append(errs, fmt.Sprintf("%s: state %s is tagged both failure and accepting; a state that records that something went wrong cannot report caller-facing success", base, s.Path))
		}
	}
	errs = append(errs, lintEventList(base, ro, states, "_failure_events", "that report a failure arriving from outside")...)
	external := map[string]bool{}
	for _, e := range eventList(ro, "_external_events") {
		external[e] = true
	}
	for _, e := range eventList(ro, "_failure_events") {
		if external[e] {
			errs = append(errs, fmt.Sprintf("%s: event %s is in both _external_events and _failure_events; a failure event is external already (keep it in _failure_events only)", base, ir.Repr(e)))
		}
	}
	return errs
}

func looksLikeFailure(name string) bool { return ir.LooksLikeFailure(name) }

// IdleDismissals returns, as findings, the entries of _not_failures that
// dismiss nothing because the name is one of actions, the actions of the
// machine's lifecycle entity, which are never hinted. entity is its name.
func (m *Machine) IdleDismissals(entity string, actions map[string]bool) (errs []string) {
	ro := m.Root.AsObject()
	if ro == nil {
		return nil
	}
	isState := map[string]bool{}
	for _, s := range ir.WalkStates(ro.Get2("states"), "") {
		isState[s.Path] = true
	}
	for _, n := range ro.GetObject("_not_failures").Keys() {
		if actions[n] && !isState[n] && looksLikeFailure(n) {
			errs = append(errs, fmt.Sprintf("%s: _not_failures names %s, which is an action of %s and is never hinted, so there is no hint to dismiss (remove it)", m.Path, ir.Repr(n), entity))
		}
	}
	return errs
}

// Hints returns the name hints of section 6.2, warnings that never fail the
// stage: a state whose name looks like a failure and is not tagged failure,
// and an event the machine answers whose name looks like a failure and that
// is neither a failure event nor one of actions (the actions of the
// machine's lifecycle entity). An event listed as external or given an alias
// is still hinted: it may report a failure. _not_failures dismisses a hint
// by name.
func (m *Machine) Hints(actions map[string]bool) (hints []string) {
	ro := m.Root.AsObject()
	if ro == nil {
		return nil
	}
	states := ir.WalkStates(ro.Get2("states"), "")
	events := map[string]bool{}
	dismissed := notFailures(ro)
	for _, s := range states {
		if tags, _ := stateTags(s.Node); looksLikeFailure(s.Name) && !tags[tagFailure] && !dismissed[s.Path] {
			hints = append(hints, fmt.Sprintf("%s: state %s looks like a failure state and is not tagged failure", m.Path, s.Path))
		}
		if o := s.Node.AsObject(); o != nil {
			for _, k := range o.GetObject("on").Keys() {
				events[k] = true
			}
			for _, k := range o.GetObject("_ignores").Keys() {
				events[k] = true
			}
		}
	}
	declared := map[string]bool{}
	for _, e := range m.FailureEvents() {
		declared[e] = true
	}
	for e := range dismissed {
		declared[e] = true
	}
	for e := range m.RefusalEvents() {
		declared[e] = true // a declared refusal, not a failure
	}
	for _, ev := range sortedSet(events) {
		if looksLikeFailure(ev) && !declared[ev] && !actions[ev] {
			hints = append(hints, fmt.Sprintf("%s: event %s looks like a failure event and is not in _failure_events", m.Path, ev))
		}
	}
	return hints
}
