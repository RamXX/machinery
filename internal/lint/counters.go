// Retry-counter accounting (S3 of the dogfood systemic findings, as amended).
// _delays has a consumed-in-both-directions rule; retry COUNTERS had none,
// so counters accumulated for the life of the row wherever no reset rode the
// deliberate re-entry edge: round 8 found the class in eight machines whose
// matrices promised "a re-issue gets a full budget" over a counter nothing
// cleared. The rule the design converged on: every edge entering a bounded
// leg from outside its retry siblings carries a reset, OR the counter carries
// a proof-carrying one-budget-per-instance annotation (`_counters`), because
// the round-10 audits contain correct no-reset cases ("provably 0 at entry")
// and a lint that flags correct machines teaches readers to ignore it. The
// check is heuristic (naming conventions identify the counter family), so it
// lives in the WARNINGS tier; the `_counters` annotation is validated as an
// error, since a present declaration is a contract.

package lint

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// counterField matches the context fields the convention treats as bounded
// budgets: retry counters, attempt windows, recheck counts, failure streaks.
var counterField = regexp.MustCompile(`(Retries|Attempts|Checks|Streak)$|^retries$`)

// LintCounters returns the counter-accounting findings for one machine:
// errors for a malformed or stale `_counters` declaration, warnings for a
// counter with an increment and an exhaustion guard but no reset anywhere.
func LintCounters(m *ir.Value, base string) (errs, warns []string) {
	ro := m.AsObject()
	ctx := ro.GetObject("context")
	if ctx == nil {
		return nil, nil
	}
	var counters []string
	for _, k := range ctx.Keys() {
		if counterField.MatchString(k) {
			counters = append(counters, k)
		}
	}
	if len(counters) == 0 {
		return nil, nil
	}
	guards, actions, _ := MachineUnitNames(m)
	lowerHas := func(set map[string]bool, pred func(string) bool) bool {
		for n := range set {
			if pred(strings.ToLower(n)) {
				return true
			}
		}
		return false
	}
	waivers := map[string]string{}
	if cv := ro.Get2("_counters"); cv != nil {
		if cv.Kind != ir.KindObject {
			errs = append(errs, base+": _counters must be an object mapping a context counter to its one-budget-per-instance proof")
		} else {
			co := cv.AsObject()
			known := map[string]bool{}
			for _, c := range counters {
				known[c] = true
			}
			for _, k := range co.Keys() {
				reason := strings.TrimSpace(co.GetString(k))
				switch {
				case !known[k]:
					errs = append(errs, fmt.Sprintf("%s: _counters names %s, which is not a counter field of this machine's context", base, ir.Repr(k)))
				case reason == "":
					errs = append(errs, fmt.Sprintf("%s: _counters[%s] carries no proof; the annotation exists to state WHY one budget per instance is correct, not to silence the check", base, ir.Repr(k)))
				default:
					waivers[k] = reason
				}
			}
		}
	}
	sort.Strings(counters)
	for _, c := range counters {
		lc := strings.ToLower(c)
		hasInc := lowerHas(actions, func(n string) bool {
			return strings.Contains(n, lc) && (strings.HasPrefix(n, "inc") || strings.HasPrefix(n, "increment"))
		})
		hasReset := lowerHas(actions, func(n string) bool {
			return strings.Contains(n, lc) && (strings.HasPrefix(n, "clear") || strings.HasPrefix(n, "reset"))
		})
		hasExhaust := lowerHas(guards, func(n string) bool {
			return strings.Contains(n, lc+"exhausted") || strings.HasSuffix(n, lc+"exhausted")
		})
		_, waived := waivers[c]
		switch {
		case hasReset && waived:
			warns = append(warns, fmt.Sprintf("%s: _counters waives %s as one-budget-per-instance, but a reset action for it exists; the waiver is stale, drop it or the reset", base, ir.Repr(c)))
		case hasInc && hasExhaust && !hasReset && !waived:
			warns = append(warns, fmt.Sprintf("%s: counter %s has an increment and an exhaustion guard but no reset action on any edge; if a re-issued entry is promised a fresh budget, wire the reset on the leg-entry edges, and if one budget per instance is the design, say so with its proof in _counters", base, ir.Repr(c)))
		}
	}
	return errs, warns
}
