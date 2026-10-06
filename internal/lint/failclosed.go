package lint

import (
	"fmt"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// Fail closed by contract. A component that reports a verdict reports
// success only when every applicable check passed; "not checked" is visible
// and never yields caller-facing success. In a machine the author says which
// states are which with XState tags:
//
//   - accepting: the state reports caller-facing success;
//   - unchecked / omitted: an applicable check has not run, or an input it
//     needs was absent.
//
// The lint then holds three structural rules. A machine with none of these
// tags is unaffected; any other tag is inert here.
const (
	tagAccepting = "accepting"
	tagUnchecked = "unchecked"
	tagOmitted   = "omitted"
)

// stateTags reads a state's XState `tags` (a string or an array of strings).
// ok is false when the value is present but unreadable: a tag the lint cannot
// read is a fail-closed rule it cannot hold, so that is an error, not a skip.
func stateTags(node *ir.Value) (tags map[string]bool, ok bool) {
	tags = map[string]bool{}
	if node == nil || node.Kind != ir.KindObject {
		return tags, true
	}
	v := node.AsObject().Get2("tags")
	if v == nil {
		return tags, true
	}
	switch v.Kind {
	case ir.KindString:
		if strings.TrimSpace(v.AsString()) == "" {
			return tags, false
		}
		tags[v.AsString()] = true
	case ir.KindArray:
		for _, e := range v.AsArray() {
			if e == nil || e.Kind != ir.KindString || strings.TrimSpace(e.AsString()) == "" {
				return map[string]bool{}, false
			}
			tags[e.AsString()] = true
		}
	default:
		return tags, false
	}
	return tags, true
}

// notCheckedTag returns the not-checked tag a state carries ("" if none).
func notCheckedTag(tags map[string]bool) string {
	for _, t := range []string{tagUnchecked, tagOmitted} {
		if tags[t] {
			return t
		}
	}
	return ""
}

// transitionLabel names an edge the way the author wrote it.
func transitionLabel(tr ir.Transition) string {
	switch tr.Kind {
	case "on", "after":
		return tr.Kind + ":" + tr.Event
	case "stateDone":
		return "onDone"
	case "onDone", "onError":
		return "invoke " + tr.Event + "." + tr.Kind
	default:
		return tr.Kind
	}
}

// lintFailClosed holds the three fail-closed rules over one machine:
//
//	(a) no transition of any kind whose source is an unchecked/omitted state
//	    (or a descendant of one) enters an accepting state;
//	(b) no accepting state is entered at machine start (the root initial and
//	    its initial child chain);
//	(c) no state is tagged both accepting and unchecked/omitted.
//
// "Enters" follows XState: a transition from q to dest enters every ancestor
// of dest that is not already an ancestor of q, dest itself, and dest's
// initial child chain when dest is compound.
func lintFailClosed(base string, ro *ir.Object, states []ir.StateEntry, r *resolver) []string {
	var errs []string
	tagsOf := map[string]map[string]bool{}
	anyRelevant := false
	for _, s := range states {
		tags, ok := stateTags(s.Node)
		if !ok {
			errs = append(errs, fmt.Sprintf("%s: state %s tags must be a string or an array of non-empty strings", base, s.Path))
		}
		tagsOf[s.Path] = tags
		if tags[tagAccepting] || notCheckedTag(tags) != "" {
			anyRelevant = true
		}
	}
	if !anyRelevant {
		return errs
	}

	// (c) one state cannot both report success and say a check did not run
	for _, s := range states {
		tags := tagsOf[s.Path]
		if nc := notCheckedTag(tags); nc != "" && tags[tagAccepting] {
			errs = append(errs, fmt.Sprintf("%s: fail-closed: state %s is tagged both accepting and %s; a state where a check has not run cannot report caller-facing success", base, s.Path, nc))
		}
	}

	// initialChain returns p and, while the node is compound, its initial
	// child, grandchild, and so on.
	initialChain := func(p string) []string {
		var out []string
		for p != "" && r.pathSet[p] {
			out = append(out, p)
			n := r.nodeOf[p]
			if n == nil || n.Kind != ir.KindObject || n.AsObject().Get2("states") == nil {
				break
			}
			ci := n.AsObject().GetString("initial")
			if ci == "" {
				break
			}
			p = p + "." + ci
		}
		return out
	}

	// (b) success before any check has run
	for _, p := range initialChain(ro.GetString("initial")) {
		if tagsOf[p][tagAccepting] {
			errs = append(errs, fmt.Sprintf("%s: fail-closed: accepting state %s is entered at machine start (the initial state chain); success before any check has run is not fail-closed", base, p))
			break
		}
	}

	// (a) not-checked never reaches success, by any edge kind
	seen := map[string]bool{}
	for _, s := range states {
		q := s.Path
		// the nearest not-checked state on q's ancestor chain (q included)
		chain := ancestorChain(q)
		ncState, ncTag := "", ""
		for i := len(chain) - 1; i >= 0; i-- {
			if t := notCheckedTag(tagsOf[chain[i]]); t != "" {
				ncState, ncTag = chain[i], t
				break
			}
		}
		if ncState == "" {
			continue
		}
		from := ncTag + " state " + q
		if ncState != q {
			from = q + " (inside " + ncTag + " state " + ncState + ")"
		}
		srcAnc := map[string]bool{}
		for _, a := range chain {
			srcAnc[a] = true
		}
		for _, tr := range ir.TransitionsOf(s.Node, nil, q) {
			if !tr.HasTgt {
				continue // internal: nothing is entered
			}
			dest, why := r.resolve(tr.Target, q)
			if why != "" || dest == "" {
				continue // a dangling target is reported by the main lint
			}
			var entered []string
			for _, a := range ancestorChain(dest) {
				if !srcAnc[a] || a == dest {
					entered = append(entered, a)
				}
			}
			entered = append(entered, initialChain(dest)[1:]...)
			for _, e := range entered {
				if !tagsOf[e][tagAccepting] {
					continue
				}
				msg := fmt.Sprintf("%s: fail-closed: %s transition from %s enters accepting state %s; a state where a check has not run or an input was absent must never yield caller-facing success (route it to a failing or explicitly not-checked outcome)",
					base, transitionLabel(tr), from, e)
				if !seen[msg] {
					seen[msg] = true
					errs = append(errs, msg)
				}
				break
			}
		}
	}
	return errs
}
