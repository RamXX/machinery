// Guard input producibility (S1 of the dogfood systemic findings, as amended).
// The most recurrent BLOCKER class across two dogfood review rounds: a guard
// whose contract demands evidence that only the guarded transition's own
// actions (or a strictly downstream action) can produce, so the first
// instance refuses forever. The _refusal lint forces authors to state the
// guard-false disposition; nothing asked whether the guard-true evidence is
// producible. The mechanism, per the amendment: an opt-in `_io` declaration
// per named unit,
//
//	"_io": {
//	  "guardX":  {"reads":  ["fieldA"]},
//	  "actionY": {"writes": ["fieldA"]},
//	  "create":  {"writes": ["fieldB"]}
//	}
//
// scoped to CONTEXT fields (DB-state reads stay with the actor-contract
// attestation), with CREATION counting as a writer (a non-null initial
// context value, or the create pseudo-unit). The producibility rule: every
// field a guard reads must be writable by some edge reachable WITHOUT
// crossing an edge that guard itself gates; a writer that exists only
// downstream of the guard is the deadlock. Declarations are validated as
// errors (a present declaration is a contract); producibility findings are
// warnings, since the declaration coverage is partial by design.
package lint

import (
	"fmt"
	"sort"

	"github.com/RamXX/machinery/internal/ir"
)

type ioEdge struct {
	source  string
	targets []string
	guard   string
	actions []string
}

// machineEdges enumerates transitions with their guard and action names.
// Entry/exit actions attach to their state's incoming/outgoing edges only
// through execution order the lint does not model; they are treated as
// writers on every edge leaving (exit) or entering (entry) the state.
func machineEdges(m *ir.Value) []ioEdge {
	var out []ioEdge
	actionNames := func(v *ir.Value) []string {
		if v == nil {
			return nil
		}
		var items []*ir.Value
		if v.Kind == ir.KindArray {
			items = v.AsArray()
		} else {
			items = []*ir.Value{v}
		}
		var names []string
		for _, it := range items {
			if it != nil && it.Kind == ir.KindString {
				names = append(names, it.AsString())
			}
		}
		return names
	}
	addBranches := func(source string, v *ir.Value, entryOf map[string][]string, exitActs []string) {
		if v == nil {
			return
		}
		var items []*ir.Value
		if v.Kind == ir.KindArray {
			items = v.AsArray()
		} else {
			items = []*ir.Value{v}
		}
		for _, it := range items {
			if it == nil {
				continue
			}
			e := ioEdge{source: source}
			switch it.Kind {
			case ir.KindString:
				e.targets = []string{it.AsString()}
			case ir.KindObject:
				o := it.AsObject()
				if t := o.GetString("target"); t != "" {
					e.targets = []string{t}
				}
				e.guard = o.GetString("guard")
				e.actions = actionNames(o.Get2("actions"))
			}
			e.actions = append(e.actions, exitActs...)
			for _, t := range e.targets {
				e.actions = append(e.actions, entryOf[t]...)
			}
			out = append(out, e)
		}
	}
	entryOf := map[string][]string{}
	exitOf := map[string][]string{}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		entryOf[s.Name] = actionNames(so.Get2("entry"))
		exitOf[s.Name] = actionNames(so.Get2("exit"))
	}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		so := s.Node.AsObject()
		if on := so.GetObject("on"); on != nil {
			for _, k := range on.Keys() {
				addBranches(s.Name, on.Get2(k), entryOf, exitOf[s.Name])
			}
		}
		addBranches(s.Name, so.Get2("always"), entryOf, exitOf[s.Name])
		if af := so.GetObject("after"); af != nil {
			for _, k := range af.Keys() {
				addBranches(s.Name, af.Get2(k), entryOf, exitOf[s.Name])
			}
		}
		if inv := so.Get2("invoke"); inv != nil && inv.Kind == ir.KindObject {
			io := inv.AsObject()
			addBranches(s.Name, io.Get2("onDone"), entryOf, exitOf[s.Name])
			addBranches(s.Name, io.Get2("onError"), entryOf, exitOf[s.Name])
		}
	}
	return out
}

// LintIO validates a machine's _io declaration and reports guard reads with
// no producible writer.
func LintIO(m *ir.Value, base string) (errs, warns []string) {
	ro := m.AsObject()
	iov := ro.Get2("_io")
	if iov == nil {
		return nil, nil
	}
	if iov.Kind != ir.KindObject {
		return []string{base + ": _io must be an object mapping unit names to {reads, writes} field lists"}, nil
	}
	guards, actions, actors := MachineUnitNames(m)
	isUnit := func(n string) bool { return n == "create" || guards[n] || actions[n] || actors[n] }
	ctx := ro.GetObject("context")
	ctxHas := func(f string) bool { return ctx != nil && ctx.Get2(f) != nil }
	ctxNonNull := func(f string) bool {
		if ctx == nil {
			return false
		}
		v := ctx.Get2(f)
		return v != nil && v.Kind != ir.KindNull
	}
	io := iov.AsObject()
	reads := map[string][]string{} // guard -> fields
	fieldWriters := map[string]map[string]bool{}
	list := func(o *ir.Object, key string) []string {
		var out []string
		v := o.Get2(key)
		if v == nil {
			return nil
		}
		if v.Kind != ir.KindArray {
			return nil
		}
		for _, it := range v.AsArray() {
			if it != nil && it.Kind == ir.KindString {
				out = append(out, it.AsString())
			}
		}
		return out
	}
	for _, unit := range io.Keys() {
		uo := io.Get2(unit)
		if uo == nil || uo.Kind != ir.KindObject {
			errs = append(errs, fmt.Sprintf("%s: _io[%s] must be an object with reads and/or writes lists", base, ir.Repr(unit)))
			continue
		}
		if !isUnit(unit) {
			errs = append(errs, fmt.Sprintf("%s: _io names %s, which is not a unit of this machine (or the create pseudo-unit)", base, ir.Repr(unit)))
			continue
		}
		u := uo.AsObject()
		for _, f := range list(u, "reads") {
			if !ctxHas(f) {
				errs = append(errs, fmt.Sprintf("%s: _io[%s].reads names %s, which is not a context field (declare context reads only; DB-state reads stay with the actor contract)", base, ir.Repr(unit), ir.Repr(f)))
				continue
			}
			if guards[unit] {
				reads[unit] = append(reads[unit], f)
			}
		}
		for _, f := range list(u, "writes") {
			if !ctxHas(f) {
				errs = append(errs, fmt.Sprintf("%s: _io[%s].writes names %s, which is not a context field", base, ir.Repr(unit), ir.Repr(f)))
				continue
			}
			if fieldWriters[f] == nil {
				fieldWriters[f] = map[string]bool{}
			}
			fieldWriters[f][unit] = true
		}
	}
	if len(reads) == 0 {
		return errs, warns
	}
	edges := machineEdges(m)
	initial := ro.GetString("initial")
	var guardNames []string
	for gname := range reads {
		guardNames = append(guardNames, gname)
	}
	sort.Strings(guardNames)
	for _, gname := range guardNames {
		// reachability with every edge gated by THIS guard removed
		adj := map[string][]string{}
		for _, e := range edges {
			if e.guard == gname {
				continue
			}
			adj[e.source] = append(adj[e.source], e.targets...)
		}
		reach := map[string]bool{initial: true}
		stack := []string{initial}
		for len(stack) > 0 {
			s := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, t := range adj[s] {
				if !reach[t] {
					reach[t] = true
					stack = append(stack, t)
				}
			}
		}
		fields := append([]string(nil), reads[gname]...)
		sort.Strings(fields)
		for _, f := range fields {
			if ctxNonNull(f) || fieldWriters[f]["create"] {
				continue // creation counts as a writer
			}
			producible := false
			for _, e := range edges {
				if e.guard == gname || !reach[e.source] {
					continue
				}
				for _, a := range e.actions {
					if fieldWriters[f][a] {
						producible = true
					}
				}
			}
			if !producible {
				warns = append(warns, fmt.Sprintf("%s: guard %s reads %s, and no declared writer of it rides an edge reachable without crossing that guard (creation writes nothing into it either); if the only writers are downstream, the first instance refuses forever, the S1 deadlock class", base, ir.Repr(gname), ir.Repr(f)))
			}
		}
	}
	return errs, warns
}
