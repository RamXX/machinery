package lint

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var invariantKeys = map[string]bool{
	"id": true, "statement": true, "requires": true, "_comment": true,
}

var invariantRequiresKeys = map[string]bool{
	"event": true, "when": true, "in": true, "except": true, "effect": true, "_comment": true,
}

// lintInvariants validates the optional root-level `_invariants` block and
// enforces the one cross-cutting law it makes checkable: an event an invariant
// obliges to MUTATE CONTEXT in a state must not be declared ignored there,
// because a strict ignore (same state, unchanged context, no actions) forbids
// exactly that mutation.
//
// Why this exists: a design can carry that contradiction invisibly when the
// reconciliation lives in prose or in a test fixture's payload choice. The
// prose does not survive generation; anything generated from the formal
// artifact (skeletons, probes, walks) implements the declared ignore and then
// violates the invariant. Both sides of the conflict are design-side facts,
// so the refusal belongs here, at lint, before any code exists.
//
// Vocabulary (all optional; machines without `_invariants` lint as before):
//
//	"_invariants": [
//	  {"id": "kebab-case-id",
//	   "statement": "the prose law, for humans",
//	   "requires": [
//	     {"event": "eventName",          // must exist in the machine
//	      "when": "guardName",           // optional: the obligation is conditional
//	      "in": "all" | ["StateA", ...], // scope, state paths
//	      "except": ["StateB"],          // only with "all"
//	      "effect": "context"}]}         // v1: the only checkable effect
//	]
func lintInvariants(
	base string,
	ro *ir.Object,
	states []ir.StateEntry,
	onOf map[string]map[string]bool,
	igOf map[string]map[string]string,
	pathSet map[string]bool,
	guards map[string]bool,
	eventUniverse map[string]bool,
) (errs, warns []string) {
	invVal := ro.Get2("_invariants")
	if invVal == nil {
		return nil, nil
	}
	if invVal.Kind != ir.KindArray {
		return []string{base + ": _invariants must be an array of invariant objects"}, nil
	}

	// Effective handling with ancestor credit, mirroring the event-completeness
	// rule: an unhandled event bubbles up in XState, so a handler or an ignore
	// on an ancestor covers the descendant.
	effective := func(p, ev string) (handled, ignored bool) {
		for _, q := range ancestorChain(p) {
			if onOf[q][ev] {
				handled = true
			}
			if _, ok := igOf[q][ev]; ok {
				ignored = true
			}
		}
		return handled, ignored
	}

	seen := map[string]bool{}
	for i, iv := range invVal.AsArray() {
		where := fmt.Sprintf("%s: _invariants[%d]", base, i)
		if iv == nil || iv.Kind != ir.KindObject {
			errs = append(errs, where+" must be an object")
			continue
		}
		io := iv.AsObject()
		for _, k := range io.Keys() {
			if !invariantKeys[k] {
				errs = append(errs, fmt.Sprintf("%s: unsupported key %s (supported: %s)",
					where, ir.Repr(k), strings.Join(sortedKeysMap(invariantKeys), ", ")))
			}
		}
		id := io.GetString("id")
		if !regexpInvariantID.MatchString(id) {
			errs = append(errs, where+": id must be kebab-case matching [a-z][a-z0-9-]*")
		} else {
			where = fmt.Sprintf("%s: invariant %s", base, ir.Repr(id))
			if seen[id] {
				errs = append(errs, where+" is declared twice")
			}
			seen[id] = true
		}
		if strings.TrimSpace(io.GetString("statement")) == "" {
			errs = append(errs, where+": statement must be a non-empty string (the law, for humans)")
		}

		reqVal := io.Get2("requires")
		if reqVal == nil {
			continue // a named prose law with no formal obligations is valid
		}
		if reqVal.Kind != ir.KindArray {
			errs = append(errs, where+": requires must be an array")
			continue
		}
		for j, rq := range reqVal.AsArray() {
			rwhere := fmt.Sprintf("%s: requires[%d]", where, j)
			if rq == nil || rq.Kind != ir.KindObject {
				errs = append(errs, rwhere+" must be an object")
				continue
			}
			qo := rq.AsObject()
			for _, k := range qo.Keys() {
				if !invariantRequiresKeys[k] {
					errs = append(errs, fmt.Sprintf("%s: unsupported key %s (supported: %s)",
						rwhere, ir.Repr(k), strings.Join(sortedKeysMap(invariantRequiresKeys), ", ")))
				}
			}
			if effect := qo.GetString("effect"); effect != "context" {
				errs = append(errs, fmt.Sprintf(
					"%s: effect must be \"context\" (the only checkable effect), got %s",
					rwhere, ir.Repr(effect)))
				continue
			}
			ev := qo.GetString("event")
			if ev == "" || !eventUniverse[ev] {
				errs = append(errs, fmt.Sprintf(
					"%s: event %s is not an event this machine handles or ignores anywhere",
					rwhere, ir.Repr(ev)))
				continue
			}
			if when := qo.GetString("when"); when != "" && !guards[when] {
				warns = append(warns, fmt.Sprintf(
					"%s: when %s names a guard no transition declares yet", rwhere, ir.Repr(when)))
			}

			scope, scopeErrs := invariantScope(rwhere, qo, states, pathSet)
			errs = append(errs, scopeErrs...)

			for _, p := range scope {
				handled, ignored := effective(p, ev)
				if ignored && !handled {
					errs = append(errs, fmt.Sprintf(
						"%s: requires event %s to update context in state %s, but the state "+
							"declares it ignored, and a strict ignore forbids context change; "+
							"declare the handling arm in on: (guarded, with its complement, if "+
							"the obligation is conditional) and drop the _ignores entry, or "+
							"narrow the invariant's scope",
						where, ir.Repr(ev), p))
				}
			}
		}
	}
	return errs, warns
}

// invariantScope resolves a requires entry's `in`/`except` to state paths.
func invariantScope(
	rwhere string, qo *ir.Object, states []ir.StateEntry, pathSet map[string]bool,
) (scope []string, errs []string) {
	inVal := qo.Get2("in")
	exceptVal := qo.Get2("except")

	switch {
	case inVal != nil && inVal.Kind == ir.KindString && inVal.AsString() == "all":
		excluded := map[string]bool{}
		if exceptVal != nil {
			if exceptVal.Kind != ir.KindArray {
				errs = append(errs, rwhere+": except must be an array of state paths")
			} else {
				for _, e := range exceptVal.AsArray() {
					if e == nil || e.Kind != ir.KindString || !pathSet[e.AsString()] {
						errs = append(errs, fmt.Sprintf("%s: except entry %s is not a state in this machine",
							rwhere, ir.Repr(reprOrNil(e))))
						continue
					}
					excluded[e.AsString()] = true
				}
			}
		}
		for _, s := range states {
			if !excluded[s.Path] {
				scope = append(scope, s.Path)
			}
		}
	case inVal != nil && inVal.Kind == ir.KindArray:
		if exceptVal != nil {
			errs = append(errs, rwhere+": except is only valid with in: \"all\"")
		}
		if len(inVal.AsArray()) == 0 {
			errs = append(errs, rwhere+": in must not be an empty list")
		}
		for _, e := range inVal.AsArray() {
			if e == nil || e.Kind != ir.KindString || !pathSet[e.AsString()] {
				errs = append(errs, fmt.Sprintf("%s: in entry %s is not a state in this machine",
					rwhere, ir.Repr(reprOrNil(e))))
				continue
			}
			scope = append(scope, e.AsString())
		}
	default:
		errs = append(errs, rwhere+": in must be \"all\" or a list of state paths")
	}
	return scope, errs
}

func reprOrNil(v *ir.Value) string {
	if v == nil {
		return "null"
	}
	if v.Kind == ir.KindString {
		return v.AsString()
	}
	return "non-string"
}
