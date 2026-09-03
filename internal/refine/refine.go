// Package refine is the Go port of refine_gen.py: generates the data-refined
// model, abstract contract, and refinement mapping for a machine from a
// declarative semantics annotation, AFTER reconciling it against the machine.
package refine

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/artifactset"
	"github.com/RamXX/machinery/internal/designlock"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/portablepath"
	"github.com/RamXX/machinery/internal/version"
)

// ExitError carries a hard-error (maps to Python sys.exit).
type ExitError struct{ Msg string }

func (e *ExitError) Error() string { return e.Msg }

func die(format string, args ...interface{}) {
	panic(&ExitError{Msg: "refine_gen: RECONCILIATION FAILED: " + fmt.Sprintf(format, args...)})
}

func topStates(m *ir.Value) map[string]*ir.Value {
	out := map[string]*ir.Value{}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		if !strings.Contains(s.Path, ".") {
			out[s.Name] = s.Node
		}
	}
	return out
}

func onTargets(node *ir.Value, event string) []string {
	var out []string
	for _, tr := range ir.TransitionsOf(node, nil, "") {
		if tr.Kind == "on" && tr.Event == event && tr.Target != "" {
			out = append(out, ir.Simple(tr.Target))
		}
	}
	return out
}

func invokeBranchTargets(node *ir.Value, key string) []string {
	var out []string
	for _, tr := range ir.TransitionsOf(node, nil, "") {
		if tr.Kind == key && tr.Target != "" {
			out = append(out, ir.Simple(tr.Target))
		}
	}
	return out
}

func afterTargets(node *ir.Value) map[string]bool {
	out := map[string]bool{}
	for _, tr := range ir.TransitionsOf(node, nil, "") {
		if tr.Kind == "after" && tr.Target != "" {
			out[ir.Simple(tr.Target)] = true
		}
	}
	return out
}

func alwaysTargets(node *ir.Value) map[string]bool {
	out := map[string]bool{}
	o := node.AsObject()
	if always := o.Get2("always"); always != nil {
		items := []*ir.Value{always}
		if always.Kind == ir.KindArray {
			items = always.AsArray()
		}
		for _, it := range items {
			if it != nil && it.Kind == ir.KindObject {
				if tv := it.AsObject().Get2("target"); tv != nil && tv.Kind == ir.KindString {
					out[ir.Simple(tv.AsString())] = true
				}
			}
		}
	}
	return out
}

// validateLifecycleRollbackRoutes proves more than target-set equality.  A
// rollback decision is ordered control flow: every remembered domain state
// needs its own guarded branch and the list needs one final unguarded fallback
// for a corrupt or otherwise unrecognized prior value.  Collapsing the routes
// to target names loses that fact when the fallback intentionally shares a
// target with a guarded branch.
func validateLifecycleRollbackRoutes(node *ir.Value, rollback, fault string, enters map[string]bool) map[string]bool {
	targetsSeen := map[string]bool{}
	guardsSeen := map[string]bool{}
	fallbacks := 0
	fallbackSeen := false
	for _, tr := range ir.TransitionsOf(node, nil, rollback) {
		if tr.Kind != "always" || tr.Target == "" {
			continue
		}
		target := ir.Simple(tr.Target)
		targetsSeen[target] = true
		if !tr.HasGuard {
			fallbacks++
			fallbackSeen = true
			if fault != "" && target != fault {
				die("%s rollback fallback routes to %s; overlay.fault requires %s", rollback, ir.Repr(target), ir.Repr(fault))
			}
			continue
		}
		if fallbackSeen {
			die("%s rollback routing is incomplete or stale: the unguarded fallback must be the final route", rollback)
		}
		const prefix = "priorIs"
		if !strings.HasPrefix(tr.Guard, prefix) {
			die("%s rollback guard %s is not a prior-state guard", rollback, ir.Repr(tr.Guard))
		}
		state := strings.TrimPrefix(tr.Guard, prefix)
		if state == "" || !enters[state] || target != state {
			die("%s rollback guard %s routes to %s; expected priorIs<State> -> <State> for a state that enters the overlay", rollback, ir.Repr(tr.Guard), ir.Repr(target))
		}
		if guardsSeen[state] {
			die("%s declares duplicate rollback guard %s", rollback, ir.Repr(tr.Guard))
		}
		guardsSeen[state] = true
	}
	for _, state := range sortedSet(enters) {
		if !guardsSeen[state] {
			die("%s rollback routing is incomplete or stale: missing guarded route priorIs%s -> %s", rollback, state, state)
		}
	}
	if fallbacks != 1 {
		die("%s rollback routing is incomplete or stale: expected exactly one final unguarded fallback, found %d", rollback, fallbacks)
	}
	return targetsSeen
}

// requireModeled enumerates the FULL transition set of a pattern-relevant
// state and dies on the first targeted transition the emitted model does not
// carry. allowed maps a trigger key to the set of targets the model represents
// from this state; keys are "on:<event>" for on: handlers and the bare kind
// ("after", "always", "onDone", "onError", "stateDone") otherwise. A trigger
// absent from the map admits no targeted transition. Mirrors the tla_gen
// retry-state rule: a silently dropped route leaves every proof green while
// asserting the opposite of the real machine. Internal transitions (no
// target) do not move the state and are not modeled by construction.
func requireModeled(node *ir.Value, state, pattern string, allowed map[string]map[string]bool) {
	for _, tr := range ir.TransitionsOf(node, nil, state) {
		if tr.Target == "" {
			continue
		}
		tgt := ir.Simple(tr.Target)
		key := tr.Kind
		if tr.Kind == "on" {
			key = "on:" + tr.Event
		}
		if allowed[key][tgt] {
			continue
		}
		die("state %s declares a transition the emitted model does not carry (%s -> %s); the %s pattern cannot model it, so the proof would silently assert the opposite of the machine; remove the transition or extend the pattern",
			ir.Repr(state), key, ir.Repr(tgt), pattern)
	}
}

// targets builds an allowed-target set literal.
func targets(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// machineEffectiveMaxRetries mirrors tla_gen's _max_retries handling: the
// machine's declared retry bound, defaulting to 3. declared reports whether
// the machine carries the annotation.
func machineEffectiveMaxRetries(machine *ir.Value) (value int, declared bool) {
	v := machine.AsObject().Get2("_max_retries")
	if v == nil {
		return 3, false
	}
	n, err := v.AsNumber().Int64()
	if v.Kind != ir.KindNumber || err != nil || n < 1 {
		die("machine _max_retries must be a positive integer")
	}
	return int(n), true
}

// reconcileMaxRetries resolves the retry bound BOTH rungs must share. The
// control-flow rung (tla_gen) proves the machine at its effective bound
// (_max_retries, default 3); a data rung proving a different bound proves a
// different system. Absent semantics max_retries inherits the machine's
// effective value, never 0; an explicit mismatch is a hard generation error.
// The returned source string is stated in the generated header.
func reconcileMaxRetries(machine, sem *ir.Value) (int, string) {
	mv, declared := machineEffectiveMaxRetries(machine)
	machineSrc := "the machine default (no _max_retries declared)"
	if declared {
		machineSrc = "machine _max_retries"
	}
	sv := sem.AsObject().Get2("max_retries")
	if sv == nil {
		return mv, "inherited from " + machineSrc
	}
	n, err := sv.AsNumber().Int64()
	if sv.Kind != ir.KindNumber || err != nil || n < 1 {
		die("semantics max_retries must be a positive integer; omit it to inherit the machine's effective bound")
	}
	if int(n) != mv {
		die("semantics max_retries = %d but the machine's effective bound is %d (%s); the two rungs would prove different systems; make them agree or drop max_retries from the semantics", n, mv, machineSrc)
	}
	return mv, "semantics max_retries, matching " + machineSrc
}

// --- lifecycle pattern ---

func lifecycleOverlay(sem *ir.Value) (busy, retry, rollback string) {
	busy, retry, rollback = "persisting", "persistRetry", "rolledBack"
	if ov := sem.AsObject().Get2("overlay"); ov != nil && ov.Kind == ir.KindObject {
		oo := ov.AsObject()
		if v := oo.Get2("busy"); v != nil && v.Kind == ir.KindString {
			busy = v.AsString()
		}
		if v := oo.Get2("retry"); v != nil && v.Kind == ir.KindString {
			retry = v.AsString()
		}
		if v := oo.Get2("rollback"); v != nil && v.Kind == ir.KindString {
			rollback = v.AsString()
		}
	}
	return
}

// lifecycleFault returns the optional final state used when rollback routing
// cannot recover the remembered domain state. Keeping it explicit in the
// semantics prevents a catch-all machine branch from disappearing from the
// refinement proof.
func lifecycleFault(sem *ir.Value) string {
	if ov := sem.AsObject().Get2("overlay"); ov != nil && ov.Kind == ir.KindObject {
		if v := ov.AsObject().Get2("fault"); v != nil && v.Kind == ir.KindString {
			return v.AsString()
		}
	}
	return ""
}

func strSlice(v *ir.Value) []string {
	var out []string
	if v == nil || v.Kind != ir.KindArray {
		return out
	}
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			out = append(out, e.AsString())
		}
	}
	return out
}

func requireStr(sem *ir.Object, key string) string {
	v := sem.Get2(key)
	if v == nil || v.Kind != ir.KindString || v.AsString() == "" {
		die("semantics must declare %s (the Modelith action name) so the machine's transition structure can be verified", key)
	}
	return v.AsString()
}

func sortedSet(m map[string]bool) []string {
	var xs []string
	for x := range m {
		xs = append(xs, x)
	}
	sort.Strings(xs)
	return xs
}

func sortedTopStateNames(m map[string]*ir.Value) []string {
	xs := make([]string, 0, len(m))
	for name := range m {
		xs = append(xs, name)
	}
	sort.Strings(xs)
	return xs
}

func contains(xs []string, x string) bool {
	for _, e := range xs {
		if e == x {
			return true
		}
	}
	return false
}

func ReconcileLifecycle(machine, sem *ir.Value) map[string]bool {
	so := sem.AsObject()
	busy, retry, rollback := lifecycleOverlay(sem)
	fault := lifecycleFault(sem)
	stages := strSlice(so.Get2("stages"))
	win := so.GetString("win_stage")
	lose := so.GetString("lose_stage")
	if len(stages) == 0 {
		die("linear-lifecycle must declare stages (the ordered forward states)")
	}
	if win == "" || lose == "" {
		die("linear-lifecycle must declare win_stage and lose_stage")
	}
	for _, key := range []string{"advance_event", "win_event", "lose_event", "reopen_event"} {
		requireStr(so, key)
	}
	adv := so.GetString("advance_event")
	wev := so.GetString("win_event")
	lev := so.GetString("lose_event")
	rev := so.GetString("reopen_event")
	top := topStates(machine)
	domainExpected := map[string]bool{}
	for _, s := range stages {
		domainExpected[s] = true
	}
	domainExpected[win] = true
	domainExpected[lose] = true
	domainActual := map[string]bool{}
	for _, n := range sortedTopStateNames(top) {
		if ir.IsUpperFirst(n) {
			domainActual[n] = true
		}
	}
	if !setEq(domainActual, domainExpected) {
		die("domain states disagree: machine has %s, semantics declare %s",
			bracket(sortedSet(domainActual)), bracket(sortedSet(domainExpected)))
	}
	if machine.AsObject().GetString("initial") != stages[0] {
		die("machine initial is %s, semantics stage order starts at %s",
			ir.Repr(machine.AsObject().GetString("initial")), ir.Repr(stages[0]))
	}
	for _, ov := range []string{busy, retry, rollback} {
		if _, ok := top[ov]; !ok {
			die("overlay state %s missing from the machine (declared under overlay:)", ir.Repr(ov))
		}
	}
	if fault != "" {
		node, ok := top[fault]
		if !ok {
			die("overlay fault state %s missing from the machine (declared under overlay:)", ir.Repr(fault))
		}
		if node.AsObject().GetString("type") != "final" {
			die("overlay fault state %s must be final; rollback failure is a terminal outcome", ir.Repr(fault))
		}
	}
	// every top-level state must be part of the pattern: a state outside the
	// vocabulary would be entirely unmodeled
	expectedTop := map[string]bool{busy: true, retry: true, rollback: true}
	if fault != "" {
		expectedTop[fault] = true
	}
	for s := range domainExpected {
		expectedTop[s] = true
	}
	for _, n := range sortedTopStateNames(top) {
		if !expectedTop[n] {
			die("machine state %s is outside the linear-lifecycle vocabulary (stages, terminals, and the busy/retry/rollback overlay); the emitted model would not carry it", ir.Repr(n))
		}
	}
	for _, s := range stages[:len(stages)-1] {
		if !contains(onTargets(top[s], adv), busy) {
			die("stage %s has no %s transition into %s", ir.Repr(s), ir.Repr(adv), ir.Repr(busy))
		}
	}
	if contains(onTargets(top[stages[len(stages)-1]], adv), busy) {
		die("last open stage %s must not advance (win/lose only)", ir.Repr(stages[len(stages)-1]))
	}
	for _, s := range stages {
		for _, pair := range [][2]string{{wev, "win"}, {lev, "lose"}} {
			if !contains(onTargets(top[s], pair[0]), busy) {
				die("open stage %s has no %s (%s) transition into %s", ir.Repr(s), pair[1], ir.Repr(pair[0]), ir.Repr(busy))
			}
		}
		if contains(onTargets(top[s], rev), busy) {
			die("open stage %s must not reopen (terminals only)", ir.Repr(s))
		}
	}
	for _, tt := range []string{win, lose} {
		if !contains(onTargets(top[tt], rev), busy) {
			die("terminal %s has no reopen (%s) transition into %s", ir.Repr(tt), ir.Repr(rev), ir.Repr(busy))
		}
		for _, ev := range []string{adv, wev, lev} {
			if contains(onTargets(top[tt], ev), busy) {
				die("terminal %s must reject %s, not persist it", ir.Repr(tt), ir.Repr(ev))
			}
		}
	}
	ondone := setOf(invokeBranchTargets(top[busy], "onDone"))
	expectedCommits := map[string]bool{}
	for _, s := range stages[1:] {
		expectedCommits[s] = true
	}
	expectedCommits[win] = true
	expectedCommits[lose] = true
	if !subset(expectedCommits, ondone) {
		die("%s onDone commits to %s; expected at least %s (every advance/win/lose target)",
			busy, bracket(sortedSet(ondone)), bracket(sortedSet(expectedCommits)))
	}
	allowed := map[string]bool{}
	for k := range expectedCommits {
		allowed[k] = true
	}
	allowed[rollback] = true
	if !subset(ondone, allowed) {
		die("%s onDone reaches unexpected states %s", busy, brSorted(sub(ondone, allowed)))
	}
	onerror := setOf(invokeBranchTargets(top[busy], "onError"))
	retryRollback := map[string]bool{retry: true, rollback: true}
	if !subset(onerror, retryRollback) {
		die("%s onError reaches unexpected states %s", busy, brSorted(onerror))
	}
	retryAlways := alwaysTargets(top[retry])
	for k := range invokeBranchTargetsSet(top[retry], "always") {
		retryAlways[k] = true
	}
	expRB := map[string]bool{rollback: true}
	if !setEq(retryAlways, expRB) {
		die("%s always must go to %s (found %s)", retry, rollback, brSorted(retryAlways))
	}
	enters := map[string]bool{}
	for _, s := range sortedSet(domainActual) {
		for _, tr := range ir.TransitionsOf(top[s], nil, "") {
			if tr.Target != "" && ir.Simple(tr.Target) == busy {
				enters[s] = true
			}
		}
	}
	expectedRollbackTargets := map[string]bool{}
	for _, state := range sortedSet(enters) {
		expectedRollbackTargets[state] = true
	}
	if fault != "" {
		expectedRollbackTargets[fault] = true
	}
	rbTargets := validateLifecycleRollbackRoutes(top[rollback], rollback, fault, enters)
	if !setEq(rbTargets, expectedRollbackTargets) {
		die("%s routes to %s but the overlay is entered from %s; the rollback routing is incomplete or stale",
			rollback, brSorted(rbTargets), brSorted(expectedRollbackTargets))
	}
	closeOn := so.GetString("close_date_on")
	if !domainExpected[closeOn] {
		die("close_date_on %s is not a domain state", ir.Repr(closeOn))
	}
	// the pattern requires reopen routes (checked above), so reopen_to is
	// mandatory; leaving it empty used to generate pending' = "" and only
	// fail later under TLC
	reopenTo := so.GetString("reopen_to")
	if reopenTo == "" {
		die("linear-lifecycle must declare reopen_to (the machine's terminals declare reopen routes; the generated model needs their target stage)")
	}
	if !contains(stages, reopenTo) {
		die("reopen_to %s is not a declared stage (%s)", ir.Repr(reopenTo), strings.Join(stages, ", "))
	}
	// bidirectional closure: the checks above prove every transition the
	// pattern REQUIRES exists; now prove the machine carries NOTHING ELSE.
	// A transition outside this vocabulary (an extra on:, after, invoke, or
	// always route) would be silently unmodeled and every proof would stay
	// green while asserting the opposite of the real machine.
	for i, s := range stages {
		allow := map[string]map[string]bool{
			"on:" + wev: targets(busy),
			"on:" + lev: targets(busy),
		}
		if i < len(stages)-1 {
			allow["on:"+adv] = targets(busy)
		}
		requireModeled(top[s], s, "linear-lifecycle", allow)
	}
	for _, tt := range []string{win, lose} {
		requireModeled(top[tt], tt, "linear-lifecycle", map[string]map[string]bool{
			"on:" + rev: targets(busy),
		})
	}
	busyCommits := targets(rollback)
	for k := range expectedCommits {
		busyCommits[k] = true
	}
	requireModeled(top[busy], busy, "linear-lifecycle", map[string]map[string]bool{
		"onDone":  busyCommits,
		"onError": targets(retry, rollback),
		"after":   targets(retry, rollback),
	})
	requireModeled(top[retry], retry, "linear-lifecycle", map[string]map[string]bool{
		"always": targets(rollback),
		"after":  targets(busy),
	})
	requireModeled(top[rollback], rollback, "linear-lifecycle", map[string]map[string]bool{
		"always": expectedRollbackTargets,
	})
	return enters
}

func invokeBranchTargetsSet(node *ir.Value, key string) map[string]bool {
	out := map[string]bool{}
	for _, t := range invokeBranchTargets(node, key) {
		out[t] = true
	}
	return out
}

// EmitLifecycle mirrors refine_gen.emit_lifecycle. Returns (mid, files).
func EmitLifecycle(machine, sem *ir.Value, sourceNames [2]string) (string, map[string]string, error) {
	var files map[string]string
	var mid string
	var refineErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if ee, ok := r.(*ExitError); ok {
					refineErr = ee
				} else {
					panic(r)
				}
			}
		}()
		mid, files = emitLifecycleImpl(machine, sem, sourceNames)
	}()
	if refineErr != nil {
		return "", nil, refineErr
	}
	return mid, files, nil
}

func emitLifecycleImpl(machine, sem *ir.Value, sourceNames [2]string) (string, map[string]string) {
	so := sem.AsObject()
	mid := ir.Title(so.GetString("machine"))
	busy, retry, rollback := lifecycleOverlay(sem)
	fault := lifecycleFault(sem)
	ReconcileLifecycle(machine, sem)
	stages := strSlice(so.Get2("stages"))
	win := so.GetString("win_stage")
	lose := so.GetString("lose_stage")
	reopenTo := so.GetString("reopen_to")
	closeOn := so.GetString("close_date_on")
	maxr, maxrSrc := reconcileMaxRetries(machine, sem)
	initial := stages[0]

	terminal := []string{win, lose}
	domain := append(append([]string{}, stages...), terminal...)
	faults := []string{}
	if fault != "" {
		faults = append(faults, fault)
	}
	advanceable := stages[:len(stages)-1]
	rank := map[string]int{}
	for i, s := range stages {
		rank[s] = i
	}
	rank[win] = len(stages)
	rank[lose] = len(stages)
	nxt := map[string]string{}
	for i := 0; i < len(stages)-1; i++ {
		nxt[stages[i]] = stages[i+1]
	}

	q := func(xs []string) string {
		ps := make([]string, len(xs))
		for i, x := range xs {
			ps[i] = fmt.Sprintf(`"%s"`, x)
		}
		return "{" + strings.Join(ps, ", ") + "}"
	}
	rankParts := make([]string, 0, len(domain))
	for _, s := range domain {
		rankParts = append(rankParts, fmt.Sprintf("%s |-> %d", s, rank[s]))
	}
	rankf := "[" + strings.Join(rankParts, ", ") + "]"
	nextParts := make([]string, 0, len(nxt))
	for _, s := range stages[:len(stages)-1] {
		nextParts = append(nextParts, fmt.Sprintf(`%s |-> "%s"`, s, nxt[s]))
	}
	nextf := "[" + strings.Join(nextParts, ", ") + "]"

	header := fmt.Sprintf(`\* GENERATED by machinery refine from %s + %s.
\* Data-refined model: proves the real domain invariants, not just control flow.
\*
\* RECONCILED against the machine before emission: domain states, initial, the
\* advance/win/lose/reopen transition structure, the overlay shape, and the
\* rollback routing all match the machine JSON; a drifted semantics file is a
\* hard generation error.
\* STILL ASSUMED (outside the machine JSON, carried by the named-unit contracts
\* and the implementation tests): the pending/prior context updates the actions
\* perform, and single-instance execution.
\* MaxRetries = %d (source: %s).`, sourceNames[0], sourceNames[1], maxr, maxrSrc)

	data := fmt.Sprintf(`---- MODULE %sData ----
%s
EXTENDS Naturals

CONSTANT MaxRetries

Open == %s
Terminal == %s
Domain == Open \cup Terminal
Fault == %s
Resting == Domain \cup Fault
Overlay == {"%s", "%s", "%s"}
None == "none"
Rank == %s
NextStage == %s

VARIABLES st, rc, stage, pending, prior, closeSet
vars == << st, rc, stage, pending, prior, closeSet >>

TypeOK ==
  /\ st \in (Resting \cup Overlay)
  /\ rc \in 0..MaxRetries
  /\ stage \in Resting
  /\ pending \in (Domain \cup {None})
  /\ prior \in (Domain \cup {None})
  /\ closeSet \in BOOLEAN

Init ==
  /\ st = "%s" /\ stage = "%s"
  /\ rc = 0 /\ pending = None /\ prior = None /\ closeSet = FALSE

StartAdvance ==
  /\ st \in %s
  /\ st' = "%s" /\ pending' = NextStage[st] /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartWin ==
  /\ st \in Open
  /\ st' = "%s" /\ pending' = "%s" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartLose ==
  /\ st \in Open
  /\ st' = "%s" /\ pending' = "%s" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

StartReopen ==
  /\ st \in Terminal
  /\ st' = "%s" /\ pending' = "%s" /\ prior' = st
  /\ rc' = 0 /\ stage' = stage /\ closeSet' = closeSet

SaveDone ==
  /\ st = "%s"
  /\ st' = pending /\ stage' = pending
  /\ closeSet' = (closeSet \/ (pending = "%s"))
  /\ pending' = None /\ prior' = None /\ rc' = 0

SaveLocked ==
  /\ st = "%s" /\ st' = "%s"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

SaveFail ==
  /\ st = "%s" /\ st' = "%s"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryExhausted ==
  /\ st = "%s" /\ rc >= MaxRetries /\ st' = "%s"
  /\ UNCHANGED << rc, stage, pending, prior, closeSet >>

RetryAgain ==
  /\ st = "%s" /\ rc < MaxRetries /\ st' = "%s" /\ rc' = rc + 1
  /\ UNCHANGED << stage, pending, prior, closeSet >>

RolledBack ==
  /\ st = "%s"
  /\ st' = prior /\ stage' = prior
  /\ pending' = None /\ prior' = None /\ rc' = 0 /\ closeSet' = closeSet

RollbackFault ==
  /\ st = "%s" /\ st' \in Fault /\ stage' = st'
  /\ pending' = None /\ prior' = None /\ rc' = 0 /\ closeSet' = closeSet

FaultStutter == st \in Fault /\ UNCHANGED vars

Domain_Next == StartAdvance \/ StartWin \/ StartLose \/ StartReopen \/ FaultStutter
Overlay_Next == SaveDone \/ SaveLocked \/ SaveFail \/ RetryExhausted \/ RetryAgain \/ RolledBack \/ RollbackFault
Next == Domain_Next \/ Overlay_Next

Spec == Init /\ [][Next]_vars /\ WF_vars(Overlay_Next)

Inv_StageValid == stage \in Resting
Inv_Atomic == (st \in Overlay) => (stage = prior)
Inv_DomainConsistent == (st \in Resting) => (st = stage /\ pending = None /\ prior = None)
Inv_CloseDate == (stage = "%s") => closeSet

StageForward ==
  [][ (stage' # stage) =>
        \/ stage' \in Fault
        \/ /\ stage \in Domain /\ stage' \in Domain
           /\ \/ Rank[stage'] > Rank[stage]
              \/ (stage \in Terminal /\ stage' = "%s") ]_stage

Live_OverlayResolves == (st \in Overlay) ~> (st \in Resting)
====
`,
		mid, header, q(stages), q(terminal), q(faults), busy, retry, rollback, rankf, nextf,
		initial, initial, q(advanceable), busy, busy, win, busy, lose, busy, reopenTo,
		busy, closeOn, busy, retry, busy, rollback, retry, rollback, retry, busy, rollback, rollback, closeOn, reopenTo)

	dataCfg := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_StageValid\nINVARIANT Inv_Atomic\nINVARIANT Inv_DomainConsistent\nINVARIANT Inv_CloseDate\nPROPERTY StageForward\nPROPERTY Live_OverlayResolves\n", maxr)

	contract := fmt.Sprintf(`---- MODULE %sContract ----
\* GENERATED. The abstract contract the big picture assumes of the %s
\* aggregate: resting or busy, atomic while busy, and every busy period terminates.
VARIABLES phase, kind
cvars == << phase, kind >>

Phases == {"resting", "busy"}
Kinds == {"open", "terminal"}

CTypeOK == phase \in Phases /\ kind \in Kinds
CInit == phase = "resting" /\ kind = "open"

Begin == phase = "resting" /\ phase' = "busy" /\ kind' = kind
Finish == phase = "busy" /\ phase' = "resting" /\ kind' \in Kinds
Churn == phase = "busy" /\ phase' = "busy" /\ kind' = kind
RestStutter == phase = "resting" /\ UNCHANGED cvars

CNext == Begin \/ Finish \/ Churn \/ RestStutter
CSpec == CInit /\ [][CNext]_cvars /\ WF_cvars(Finish)
CTermination == (phase = "busy") ~> (phase = "resting")
====
`, mid, so.GetString("machine"))

	refinement := fmt.Sprintf(`---- MODULE %sRefinement ----
\* GENERATED. Proof that %sData refines %sContract under a refinement mapping.
EXTENDS %sData

phaseBar == IF st \in Resting THEN "resting" ELSE "busy"
kindBar == IF st \in Fault \/ stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE %sContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
`, mid, mid, mid, mid, mid)

	refCfg := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT RefTypeOK\nPROPERTY RefSpec\nPROPERTY RefTermination\n", maxr)

	return mid, map[string]string{
		mid + "Data.tla":       data,
		mid + "Data.cfg":       dataCfg,
		mid + "Contract.tla":   contract,
		mid + "Refinement.tla": refinement,
		mid + "Refinement.cfg": refCfg,
	}
}

// --- terminal-lifecycle pattern ---

type retrySpec struct {
	State  string
	Serves string
}

func retriesOf(sem *ir.Value) []retrySpec {
	so := sem.AsObject()
	var out []retrySpec
	if r := so.Get2("retries"); r != nil && r.Kind == ir.KindArray {
		for _, e := range r.AsArray() {
			out = append(out, retrySpec{State: e.AsObject().GetString("state"), Serves: e.AsObject().GetString("serves")})
		}
	} else if r := so.Get2("retry"); r != nil && r.Kind == ir.KindObject {
		out = append(out, retrySpec{State: r.AsObject().GetString("state"), Serves: r.AsObject().GetString("serves")})
	}
	return out
}

// ReconcileTerminal additionally returns failRoutes: the machine's ACTUAL
// failure target(s) per phase (sorted), so emission models what the machine
// does instead of assuming failures[0].
func ReconcileTerminal(machine, sem *ir.Value) (phases []string, success string, failures []string, retries []retrySpec, failRoutes map[string][]string, err error) {
	failRoutes = map[string][]string{}
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
			} else {
				panic(r)
			}
		}
	}()
	so := sem.AsObject()
	phases = strSlice(so.Get2("phases"))
	if len(phases) == 0 {
		die("terminal-lifecycle must declare phases (the ordered forward states)")
	}
	success = so.GetString("success_terminal")
	failures = strSlice(so.Get2("failure_terminals"))
	if success == "" || len(failures) == 0 {
		die("terminal-lifecycle must declare success_terminal and failure_terminals")
	}
	retries = retriesOf(sem)
	top := topStates(machine)
	domainExpected := map[string]bool{success: true}
	for _, p := range phases {
		domainExpected[p] = true
	}
	for _, f := range failures {
		domainExpected[f] = true
	}
	domainActual := map[string]bool{}
	for _, n := range sortedTopStateNames(top) {
		if ir.IsUpperFirst(n) {
			domainActual[n] = true
		}
	}
	if !setEq(domainActual, domainExpected) {
		die("domain states disagree: machine has %s, semantics declare %s", brSorted(domainActual), brSorted(domainExpected))
	}
	if machine.AsObject().GetString("initial") != phases[0] {
		die("machine initial is %s, first phase is %s", ir.Repr(machine.AsObject().GetString("initial")), ir.Repr(phases[0]))
	}
	for _, t := range append([]string{success}, failures...) {
		if top[t].AsObject().GetString("type") != "final" {
			die("terminal %s must be a final state", ir.Repr(t))
		}
	}
	retryOf := map[string]string{}
	for _, r := range retries {
		if _, ok := top[r.State]; !ok {
			die("retry state %s missing from the machine", ir.Repr(r.State))
		}
		if !contains(phases, r.Serves) {
			die("retry %s serves unknown phase %s", ir.Repr(r.State), ir.Repr(r.Serves))
		}
		retryOf[r.Serves] = r.State
	}
	// every top-level state must be part of the pattern: an undeclared state
	// would be entirely unmodeled
	expectedTop := map[string]bool{}
	for s := range domainExpected {
		expectedTop[s] = true
	}
	for _, r := range retries {
		expectedTop[r.State] = true
	}
	for _, n := range sortedTopStateNames(top) {
		if !expectedTop[n] {
			die("machine state %s is outside the terminal-lifecycle vocabulary (phases, terminals, and the declared retry overlays); the emitted model would not carry it", ir.Repr(n))
		}
	}
	for i, p := range phases {
		node := top[p]
		no := node.AsObject()
		if no.GetString("type") == "final" {
			die("phase %s must not be final", ir.Repr(p))
		}
		if no.Get2("invoke") == nil {
			die("phase %s must invoke an effect (its onDone advances the pipeline)", ir.Repr(p))
		}
		var nxt string
		if i+1 < len(phases) {
			nxt = phases[i+1]
		} else {
			nxt = success
		}
		ondone := setOf(invokeBranchTargets(node, "onDone"))
		expNxt := map[string]bool{nxt: true}
		if !setEq(ondone, expNxt) {
			die("phase %s onDone goes to %s, expected %s", ir.Repr(p), brSorted(ondone), ir.Repr(nxt))
		}
		failTargets := map[string]bool{}
		for _, t := range invokeBranchTargets(node, "onError") {
			failTargets[t] = true
		}
		for t := range afterTargets(node) {
			failTargets[t] = true
		}
		if len(failTargets) == 0 {
			die("phase %s has no failure path (onError/after); a phase must be able to fail or time out", ir.Repr(p))
		}
		allowed := map[string]bool{}
		if rs, ok := retryOf[p]; ok {
			allowed[rs] = true
		} else {
			for _, f := range failures {
				allowed[f] = true
			}
		}
		if !subset(failTargets, allowed) {
			die("phase %s failure paths %s are not within %s (a served phase fails into its retry state; an unserved phase fails into a failure terminal)", ir.Repr(p), brSorted(failTargets), brSorted(allowed))
		}
		var routes []string
		for t := range failTargets {
			routes = append(routes, t)
		}
		sort.Strings(routes)
		failRoutes[p] = routes
	}
	for _, r := range retries {
		rs := top[r.State]
		exhaust := alwaysTargets(rs)
		if len(exhaust) == 0 || !subset(exhaust, setOf(failures)) {
			die("retry %s exhaustion (always) must go to a failure terminal, found %s", ir.Repr(r.State), brSorted(exhaust))
		}
		back := afterTargets(rs)
		expBack := map[string]bool{r.Serves: true}
		if !setEq(back, expBack) {
			die("retry %s backoff (after) must return to %s, found %s", ir.Repr(r.State), ir.Repr(r.Serves), brSorted(back))
		}
	}
	// bidirectional closure: the machine must carry NOTHING beyond the
	// vocabulary just verified; an extra route would be silently unmodeled
	for i, p := range phases {
		var nxt string
		if i+1 < len(phases) {
			nxt = phases[i+1]
		} else {
			nxt = success
		}
		fail := targets()
		if rs, ok := retryOf[p]; ok {
			fail[rs] = true
		} else {
			for _, f := range failures {
				fail[f] = true
			}
		}
		requireModeled(top[p], p, "terminal-lifecycle", map[string]map[string]bool{
			"onDone":  targets(nxt),
			"onError": fail,
			"after":   fail,
		})
	}
	for _, r := range retries {
		requireModeled(top[r.State], r.State, "terminal-lifecycle", map[string]map[string]bool{
			"always": alwaysTargets(top[r.State]),
			"after":  targets(r.Serves),
		})
	}
	for _, t := range append([]string{success}, failures...) {
		requireModeled(top[t], t, "terminal-lifecycle", nil)
	}
	return phases, success, failures, retries, failRoutes, nil
}

// EmitTerminal mirrors refine_gen.emit_terminal.
func EmitTerminal(machine, sem *ir.Value, sourceNames [2]string) (mid string, files map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				mid, files, err = "", nil, ee
			} else {
				panic(r)
			}
		}
	}()
	so := sem.AsObject()
	mid = ir.Title(so.GetString("machine"))
	phases, success, failures, retries, failRoutes, err := ReconcileTerminal(machine, sem)
	if err != nil {
		return "", nil, err
	}
	maxr, maxrSrc := reconcileMaxRetries(machine, sem)
	flag := so.GetString("success_flag")
	if flag == "" {
		flag = "completed"
	}
	retryOf := map[string]string{}
	for _, r := range retries {
		retryOf[r.Serves] = r.State
	}
	rcOf := map[string]string{}
	var counters []string
	for i, r := range retries {
		var v string
		if len(retries) > 1 {
			v = fmt.Sprintf("rc%d", i+1)
		} else {
			v = "rc"
		}
		rcOf[r.State] = v
		counters = append(counters, v)
	}
	// every reconciled exhaustion target, not just the first: with several
	// failure terminals the model must keep the machine's nondeterminism
	exhaustTo := map[string][]string{}
	top := topStates(machine)
	for _, r := range retries {
		exhaustTo[r.State] = sortedSet(alwaysTargets(top[r.State]))
	}

	q := func(xs []string) string {
		ps := make([]string, len(xs))
		for i, x := range xs {
			ps[i] = fmt.Sprintf(`"%s"`, x)
		}
		return "{" + strings.Join(ps, ", ") + "}"
	}
	nonSt := append(append([]string{}, counters...), flag)
	unch := func(changed []string) string {
		chSet := map[string]bool{}
		for _, c := range changed {
			chSet[c] = true
		}
		var keep []string
		for _, v := range nonSt {
			if !chSet[v] {
				keep = append(keep, v)
			}
		}
		if len(keep) == 0 {
			return "TRUE"
		}
		return "UNCHANGED << " + strings.Join(keep, ", ") + " >>"
	}

	header := fmt.Sprintf(`\* GENERATED by machinery refine (terminal-lifecycle) from %s + %s.
\* Data-refined model of a forward pipeline: invoking phases advance to a success
\* terminal or fail (directly or after bounded retries) to a failure terminal.
\*
\* RECONCILED against the machine before emission: the phase order, the onDone
\* forward chain, every failure route, the terminal states, and each retry
\* overlay all match the machine JSON. All state names come from the annotation;
\* nothing is hardcoded to a domain.
\* Proves: completeness (a success terminal implies its completion flag; there is
\* no partial success), terminal absorption, and termination. The domain-progress
\* proof is separate from the persistence mechanism: no persist overlay is baked in.
\* STILL ASSUMED: the effect the completion flag stands for is really established
\* on the success path, and single-instance execution.
\* MaxRetries = %d (source: %s).`, sourceNames[0], sourceNames[1], maxr, maxrSrc)

	var L []string
	L = append(L, fmt.Sprintf("---- MODULE %sData ----", mid))
	L = append(L, header)
	L = append(L, "EXTENDS Naturals")
	L = append(L, "")
	L = append(L, "CONSTANT MaxRetries")
	L = append(L, fmt.Sprintf("Phases == %s", q(phases)))
	L = append(L, fmt.Sprintf("Success == %s", q([]string{success})))
	L = append(L, fmt.Sprintf("Failure == %s", q(failures)))
	L = append(L, "Terminal == Success \\cup Failure")
	var stset string
	if len(retries) > 0 {
		var rs []string
		for _, r := range retries {
			rs = append(rs, r.State)
		}
		L = append(L, fmt.Sprintf("Retry == %s", q(rs)))
		stset = "(Phases \\cup Retry \\cup Terminal)"
	} else {
		stset = "(Phases \\cup Terminal)"
	}
	L = append(L, fmt.Sprintf("VARIABLES st, %s", strings.Join(nonSt, ", ")))
	L = append(L, fmt.Sprintf("vars == << st, %s >>", strings.Join(nonSt, ", ")))
	L = append(L, "")
	tyctr := ""
	for _, c := range counters {
		tyctr += fmt.Sprintf(" /\\ %s \\in 0..MaxRetries", c)
	}
	L = append(L, fmt.Sprintf("TypeOK == st \\in %s%s /\\ %s \\in BOOLEAN", stset, tyctr, flag))
	initctr := ""
	for _, c := range counters {
		initctr += fmt.Sprintf(" /\\ %s = 0", c)
	}
	L = append(L, fmt.Sprintf(`Init == st = "%s"%s /\ %s = FALSE`, phases[0], initctr, flag))
	L = append(L, "")
	var acts []string
	for i, p := range phases {
		var nxt string
		if i+1 < len(phases) {
			nxt = phases[i+1]
		} else {
			nxt = success
		}
		var setFlag string
		var changed []string
		if nxt == success {
			setFlag = fmt.Sprintf(" /\\ %s' = TRUE", flag)
			changed = []string{flag}
		}
		L = append(L, fmt.Sprintf(`Done_%s == st = "%s" /\ st' = "%s"%s /\ %s`, p, p, nxt, setFlag, unch(changed)))
		acts = append(acts, "Done_"+p)
		if ft, ok := retryOf[p]; ok {
			L = append(L, fmt.Sprintf(`Fail_%s == st = "%s" /\ st' = "%s" /\ %s`, p, p, ft, unch(nil)))
		} else if routes := failRoutes[p]; len(routes) == 1 {
			// the machine's reconciled failure target, not failures[0] by fiat
			L = append(L, fmt.Sprintf(`Fail_%s == st = "%s" /\ st' = "%s" /\ %s`, p, p, routes[0], unch(nil)))
		} else {
			// several reconciled failure terminals: model the nondeterminism
			var quoted []string
			for _, r := range routes {
				quoted = append(quoted, `"`+r+`"`)
			}
			L = append(L, fmt.Sprintf(`Fail_%s == st = "%s" /\ st' \in {%s} /\ %s`, p, p, strings.Join(quoted, ", "), unch(nil)))
		}
		acts = append(acts, "Fail_"+p)
	}
	for _, r := range retries {
		rs, serves := r.State, r.Serves
		ctr := rcOf[r.State]
		et := exhaustTo[r.State]
		L = append(L, fmt.Sprintf(`RetryAgain_%s == st = "%s" /\ %s < MaxRetries /\ st' = "%s" /\ %s' = %s + 1 /\ %s`, rs, rs, ctr, serves, ctr, ctr, unch([]string{ctr})))
		if len(et) == 1 {
			L = append(L, fmt.Sprintf(`RetryExhausted_%s == st = "%s" /\ %s >= MaxRetries /\ st' = "%s" /\ %s`, rs, rs, ctr, et[0], unch(nil)))
		} else {
			var quoted []string
			for _, t := range et {
				quoted = append(quoted, `"`+t+`"`)
			}
			L = append(L, fmt.Sprintf(`RetryExhausted_%s == st = "%s" /\ %s >= MaxRetries /\ st' \in {%s} /\ %s`, rs, rs, ctr, strings.Join(quoted, ", "), unch(nil)))
		}
		acts = append(acts, "RetryAgain_"+rs, "RetryExhausted_"+rs)
	}
	L = append(L, "Terminated == st \\in Terminal /\\ UNCHANGED vars")
	L = append(L, "Prog == "+strings.Join(acts, " \\/ "))
	L = append(L, "Next == Prog \\/ Terminated")
	L = append(L, "Spec == Init /\\ [][Next]_vars /\\ WF_vars(Prog)")
	L = append(L, "")
	L = append(L, fmt.Sprintf("Inv_Complete == (st \\in Success) => %s", flag))
	L = append(L, "Inv_TerminalAbsorbing == [][ (st \\in Terminal) => (st' = st) ]_st")
	L = append(L, "Live_Terminates == (st \\notin Terminal) ~> (st \\in Terminal)")
	L = append(L, "====")
	tla := strings.Join(L, "\n") + "\n"
	cfg := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_Complete\nPROPERTY Inv_TerminalAbsorbing\nPROPERTY Live_Terminates\n", maxr)
	return mid, map[string]string{
		mid + "Data.tla": tla,
		mid + "Data.cfg": cfg,
	}, nil
}

// --- saga pattern ---

func ReconcileSaga(machine, sem *ir.Value) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
			} else {
				panic(r)
			}
		}
	}()
	so := sem.AsObject()
	states := strSlice(so.Get2("states"))
	oblObj := so.Get2("obligations").AsObject()
	top := topStates(machine)
	expected := map[string]bool{}
	for _, s := range states {
		expected[s] = true
	}
	for _, s := range []string{"Compensating", "compensateRetry", "Completed", "Failed", "FailedDirty"} {
		expected[s] = true
	}
	actual := map[string]bool{}
	for _, n := range sortedTopStateNames(top) {
		actual[n] = true
	}
	if !setEq(actual, expected) {
		die("saga states disagree: machine has %s, semantics imply %s", brSorted(actual), brSorted(expected))
	}
	if machine.AsObject().GetString("initial") != states[0] {
		die("machine initial is %s, first forward step is %s", ir.Repr(machine.AsObject().GetString("initial")), ir.Repr(states[0]))
	}
	for i, s := range states {
		var nxt string
		if i+1 < len(states) {
			nxt = states[i+1]
		} else {
			nxt = "Completed"
		}
		ondone := setOf(invokeBranchTargets(top[s], "onDone"))
		expNxt := map[string]bool{nxt: true}
		if !setEq(ondone, expNxt) {
			die("forward step %s onDone goes to %s, expected %s", ir.Repr(s), brSorted(ondone), ir.Repr(nxt))
		}
		var failTo string
		if i == 0 {
			failTo = "Failed"
		} else {
			failTo = "Compensating"
		}
		onerr := setOf(invokeBranchTargets(top[s], "onError"))
		after := afterTargets(top[s])
		expFail := map[string]bool{failTo: true}
		if !setEq(onerr, expFail) || !setEq(after, expFail) {
			die("forward step %s failure paths go to onError=%s, after=%s; expected %s (first step fails clean, later steps compensate)", ir.Repr(s), brSorted(onerr), brSorted(after), ir.Repr(failTo))
		}
	}
	comp := top["Compensating"]
	if !setEq(setOf(invokeBranchTargets(comp, "onDone")), map[string]bool{"Failed": true}) {
		die("Compensating onDone must reach Failed (compensation complete)")
	}
	if !setEq(setOf(invokeBranchTargets(comp, "onError")), map[string]bool{"compensateRetry": true}) {
		die("Compensating onError must reach compensateRetry")
	}
	cr := top["compensateRetry"]
	crAlways := map[string]bool{}
	for _, b := range alwaysBranchTargets(cr) {
		crAlways[b] = true
	}
	crAfter := afterTargets(cr)
	expAlways := map[string]bool{"FailedDirty": true}
	expAfter := map[string]bool{"Compensating": true}
	if !setEq(crAlways, expAlways) || !setEq(crAfter, expAfter) {
		die("compensateRetry must exhaust to FailedDirty and back off to Compensating")
	}
	for _, f := range []string{"Completed", "Failed", "FailedDirty"} {
		if top[f].AsObject().GetString("type") != "final" {
			die("%s must be a final state", f)
		}
	}
	// bidirectional closure: the machine must carry NOTHING beyond the saga
	// vocabulary just verified; an extra route (e.g. an on: escape hatch from
	// a forward step) would be silently unmodeled and the no-silent-loss proof
	// would assert the opposite of the machine
	for i, s := range states {
		var nxt string
		if i+1 < len(states) {
			nxt = states[i+1]
		} else {
			nxt = "Completed"
		}
		var failTo string
		if i == 0 {
			failTo = "Failed"
		} else {
			failTo = "Compensating"
		}
		requireModeled(top[s], s, "saga", map[string]map[string]bool{
			"onDone":  targets(nxt),
			"onError": targets(failTo),
			"after":   targets(failTo),
		})
	}
	requireModeled(top["Compensating"], "Compensating", "saga", map[string]map[string]bool{
		"onDone":  targets("Failed"),
		"onError": targets("compensateRetry"),
		"after":   targets("compensateRetry", "Failed"),
	})
	requireModeled(top["compensateRetry"], "compensateRetry", "saga", map[string]map[string]bool{
		"always": targets("FailedDirty"),
		"after":  targets("Compensating"),
	})
	for _, f := range []string{"Completed", "Failed", "FailedDirty"} {
		requireModeled(top[f], f, "saga", nil)
	}
	for _, s := range states[:len(states)-1] {
		oo := oblObj.Get2(s).AsObject()
		if oo.GetString("sets") == "" || oo.GetString("undo") == "" {
			die("forward step %s must declare sets: and undo: (its compensating obligation); only the completing step may omit undo", ir.Repr(s))
		}
	}
	lastOO := oblObj.Get2(states[len(states)-1]).AsObject()
	if lastOO.GetString("sets") == "" {
		die("completing step %s must declare sets:", ir.Repr(states[len(states)-1]))
	}
	var unknown []string
	for _, k := range oblObj.Keys() {
		if !contains(states, k) {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		die("obligations declared for unknown steps: %s", bracket(unknown))
	}
	return nil
}

func alwaysBranchTargets(node *ir.Value) []string {
	var out []string
	o := node.AsObject()
	if always := o.Get2("always"); always != nil {
		items := []*ir.Value{always}
		if always.Kind == ir.KindArray {
			items = always.AsArray()
		}
		for _, it := range items {
			if it != nil && it.Kind == ir.KindObject {
				if tv := it.AsObject().Get2("target"); tv != nil && tv.Kind == ir.KindString {
					out = append(out, ir.Simple(tv.AsString()))
				}
			}
		}
	}
	return out
}

// EmitSaga mirrors refine_gen.emit_saga.
func EmitSaga(machine, sem *ir.Value, sourceNames [2]string) (mid string, files map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				mid, files, err = "", nil, ee
			} else {
				panic(r)
			}
		}
	}()
	if err := ReconcileSaga(machine, sem); err != nil {
		return "", nil, err
	}
	so := sem.AsObject()
	mid = ir.Title(so.GetString("machine"))
	states := strSlice(so.Get2("states"))
	oblObj := so.Get2("obligations").AsObject()
	maxr, maxrSrc := reconcileMaxRetries(machine, sem)
	initial := states[0]

	var flags []string
	flagsSeen := map[string]bool{}
	obligationOf := func(s string) (sets, undo string) {
		oo := oblObj.Get2(s).AsObject()
		sets = oo.GetString("sets")
		undo = oo.GetString("undo")
		return
	}
	for _, s := range states {
		sets, undo := obligationOf(s)
		for _, v := range []string{sets, undo} {
			if v != "" && !flagsSeen[v] {
				flagsSeen[v] = true
				flags = append(flags, v)
			}
		}
	}
	var obligations [][2]string
	for _, s := range states {
		sets, undo := obligationOf(s)
		if sets != "" && undo != "" {
			obligations = append(obligations, [2]string{sets, undo})
		}
	}
	varlist := "st, rc"
	if len(flags) > 0 {
		varlist += ", " + strings.Join(flags, ", ")
	}
	unch := func(exclude []string) string {
		exSet := map[string]bool{}
		for _, e := range exclude {
			exSet[e] = true
		}
		var keep []string
		keep = append(keep, "rc")
		for _, f := range flags {
			if !exSet[f] {
				keep = append(keep, f)
			}
		}
		if exSet["rc"] {
			keep = filterOut(keep, "rc")
		}
		return "<< " + strings.Join(keep, ", ") + " >>"
	}
	_ = unch

	var L []string
	L = append(L, fmt.Sprintf("---- MODULE %sData ----", mid))
	L = append(L, fmt.Sprintf(`\* GENERATED by machinery refine (saga pattern) from %s + %s.`, sourceNames[0], sourceNames[1]))
	L = append(L, "\\* Proves money and stock are never silently lost: a terminal saga has undone")
	L = append(L, "\\* every obligation it committed, or ends FailedDirty as an explicit residual.")
	L = append(L, "\\*")
	L = append(L, "\\* RECONCILED against the machine before emission: step order, failure routing,")
	L = append(L, "\\* the compensation loop, and the final states all match the machine JSON.")
	L = append(L, "\\* Compensation here is PER OBLIGATION (each undo its own step), refining the")
	L = append(L, "\\* machine's single idempotent compensate invoke, so partial compensation is")
	L = append(L, "\\* representable. STILL ASSUMED: the obligation flags mirror what the real")
	L = append(L, "\\* actors commit and undo, and single-instance execution.")
	L = append(L, fmt.Sprintf("\\* MaxRetries = %d (source: %s).", maxr, maxrSrc))
	L = append(L, "EXTENDS Naturals")
	L = append(L, "")
	L = append(L, "CONSTANT MaxRetries")
	stepsQuoted := make([]string, len(states))
	for i, s := range states {
		stepsQuoted[i] = "\"" + s + "\""
	}
	stepsSet := "{" + strings.Join(stepsQuoted, ", ") + ", \"Compensating\", \"compensateRetry\"}"
	L = append(L, "Steps == "+stepsSet)
	L = append(L, `Final == {"Completed", "Failed", "FailedDirty"}`)
	L = append(L, fmt.Sprintf("VARIABLES %s", varlist))
	L = append(L, fmt.Sprintf("vars == << %s >>", varlist))
	L = append(L, "")
	typeok := "TypeOK == st \\in (Steps \\cup Final) /\\ rc \\in 0..MaxRetries"
	for _, f := range flags {
		typeok += fmt.Sprintf(" /\\ %s \\in BOOLEAN", f)
	}
	L = append(L, typeok)
	init := fmt.Sprintf(`Init == st = "%s" /\ rc = 0`, initial)
	for _, f := range flags {
		init += fmt.Sprintf(" /\\ %s = FALSE", f)
	}
	L = append(L, init)
	L = append(L, "")
	var overlay []string
	for i, s := range states {
		var nxt string
		if i+1 < len(states) {
			nxt = states[i+1]
		} else {
			nxt = "Completed"
		}
		sets, _ := obligationOf(s)
		var eff string
		var excl []string
		if sets != "" {
			eff = fmt.Sprintf(" /\\ %s' = TRUE", sets)
			excl = []string{sets}
		}
		L = append(L, fmt.Sprintf(`Done_%s == st = "%s" /\ st' = "%s"%s /\ UNCHANGED %s`, s, s, nxt, eff, sagaUnch(flags, excl)))
		var ft string
		if i == 0 {
			ft = "Failed"
		} else {
			ft = "Compensating"
		}
		L = append(L, fmt.Sprintf(`Fail_%s == st = "%s" /\ st' = "%s" /\ UNCHANGED %s`, s, s, ft, sagaUnch(flags, nil)))
		overlay = append(overlay, "Done_"+s, "Fail_"+s)
	}
	// per-obligation compensation
	var openOblParts, allCleanParts []string
	for _, ob := range obligations {
		openOblParts = append(openOblParts, fmt.Sprintf("(%s /\\ ~%s)", ob[0], ob[1]))
		allCleanParts = append(allCleanParts, fmt.Sprintf("(%s => %s)", ob[0], ob[1]))
	}
	openObl := strings.Join(openOblParts, " \\/ ")
	allClean := strings.Join(allCleanParts, " /\\ ")
	for _, ob := range obligations {
		u := ob[1]
		L = append(L, fmt.Sprintf(`Undo_%s == st = "Compensating" /\ %s /\ ~%s /\ %s' = TRUE /\ st' = st /\ UNCHANGED %s`, u, ob[0], u, u, sagaUnch(flags, []string{u})))
		overlay = append(overlay, "Undo_"+u)
	}
	L = append(L, fmt.Sprintf(`CompensateDone == st = "Compensating" /\ (%s) /\ st' = "Failed" /\ UNCHANGED %s`, allClean, sagaUnch(flags, nil)))
	L = append(L, fmt.Sprintf(`CompensateErr == st = "Compensating" /\ (%s) /\ st' = "compensateRetry" /\ UNCHANGED %s`, openObl, sagaUnch(flags, nil)))
	L = append(L, fmt.Sprintf(`RetryExhausted == st = "compensateRetry" /\ rc >= MaxRetries /\ st' = "FailedDirty" /\ UNCHANGED %s`, sagaUnch(flags, nil)))
	L = append(L, fmt.Sprintf(`RetryAgain == st = "compensateRetry" /\ rc < MaxRetries /\ st' = "Compensating" /\ rc' = rc + 1 /\ UNCHANGED %s`, sagaUnch(flags, []string{"rc"})))
	overlay = append(overlay, "CompensateDone", "CompensateErr", "RetryExhausted", "RetryAgain")
	L = append(L, "")
	L = append(L, "OverlayNext == "+strings.Join(overlay, " \\/ "))
	L = append(L, "Terminated == st \\in Final /\\ UNCHANGED vars")
	L = append(L, "Next == OverlayNext \\/ Terminated")
	L = append(L, "Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
	L = append(L, "")
	var nslParts []string
	for _, ob := range obligations {
		nslParts = append(nslParts, fmt.Sprintf(`((%s /\ st # "Completed") => (%s \/ st = "FailedDirty"))`, ob[0], ob[1]))
	}
	nsl := strings.Join(nslParts, " /\\ ")
	L = append(L, fmt.Sprintf("Inv_NoSilentLoss == (st \\in Final) => (%s)", nsl))
	L = append(L, fmt.Sprintf(`Inv_CleanCompensation == (st = "Failed") => (%s)`, allClean))
	L = append(L, "Live_Terminates == (st \\notin Final) ~> (st \\in Final)")
	L = append(L, "====")
	tla := strings.Join(L, "\n") + "\n"
	cfg := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_NoSilentLoss\nINVARIANT Inv_CleanCompensation\nPROPERTY Live_Terminates\n", maxr)
	return mid, map[string]string{
		mid + "Data.tla": tla,
		mid + "Data.cfg": cfg,
	}, nil
}

// sagaUnch builds "<< rc, flag1, flag2 >>" excluding the given flags (order: rc, then flags).
func sagaUnch(flags []string, exclude []string) string {
	exSet := map[string]bool{}
	for _, e := range exclude {
		exSet[e] = true
	}
	var keep []string
	if !exSet["rc"] {
		keep = append(keep, "rc")
	}
	for _, f := range flags {
		if !exSet[f] {
			keep = append(keep, f)
		}
	}
	return "<< " + strings.Join(keep, ", ") + " >>"
}

// --- helpers ---

func setEq(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func subset(a, b map[string]bool) bool {
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sub(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

func setOf(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

func brSorted(m map[string]bool) string {
	return bracket(sortedSet(m))
}

func bracket(xs []string) string {
	ps := make([]string, len(xs))
	for i, x := range xs {
		ps[i] = fmt.Sprintf(`"%s"`, x)
	}
	return "{" + strings.Join(ps, ", ") + "}"
}

func filterOut(xs []string, x string) []string {
	var out []string
	for _, e := range xs {
		if e != x {
			out = append(out, e)
		}
	}
	return out
}

// Run is the `machinery refine <machine.json> <semantics.yaml> [out-dir]` entrypoint.
func Run(machinePath, semPath, outdir string) error {
	return RunTo(machinePath, semPath, outdir, os.Stdout)
}

// RunTo is Run with an explicit status-output sink.
func RunTo(machinePath, semPath, outdir string, out io.Writer) error {
	_, err := RunWrittenTo(machinePath, semPath, outdir, out)
	return err
}

// ValidateControlFlowOnly validates the deliberately shallow semantics
// declaration for a lifecycle whose control flow is already represented by
// the rung-3 TLA generator but does not truthfully fit a data-refinement
// algebra. Its closed three-key shape prevents the annotation from implying a
// deeper proof through ignored fields.
func ValidateControlFlowOnly(machinePath, semPath string) error {
	machine, err := ir.LoadMachineJSON(machinePath)
	if err != nil {
		return &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	data, err := os.ReadFile(semPath)
	if err != nil {
		return &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	sem, err := ir.LoadYAML(data)
	if err != nil || sem.Kind != ir.KindObject {
		return &ExitError{Msg: "refine_gen: control-flow-only semantics must be a yaml mapping"}
	}
	o := sem.AsObject()
	allowed := map[string]bool{"machine": true, "pattern": true, "reason": true}
	for _, key := range o.Keys() {
		if !allowed[key] {
			return &ExitError{Msg: fmt.Sprintf("refine_gen: control-flow-only semantics has unsupported key %s (only machine, pattern, reason are allowed)", ir.Repr(key))}
		}
	}
	if o.Len() != 3 {
		return &ExitError{Msg: "refine_gen: control-flow-only semantics requires exactly machine, pattern, and reason"}
	}
	want := machine.AsObject().GetString("id")
	if got := o.GetString("machine"); got == "" || got != want {
		return &ExitError{Msg: fmt.Sprintf("refine_gen: control-flow-only machine must exactly equal the machine id %s, got %s", ir.Repr(want), ir.Repr(got))}
	}
	if o.GetString("pattern") != "control-flow-only" {
		return &ExitError{Msg: "refine_gen: pattern must be control-flow-only"}
	}
	reason := o.Get2("reason")
	if reason == nil || reason.Kind != ir.KindString || strings.TrimSpace(reason.AsString()) == "" {
		return &ExitError{Msg: "refine_gen: control-flow-only semantics requires a non-empty reason"}
	}
	return nil
}

func validateSemanticsSchema(sem *ir.Value) error {
	if sem == nil || sem.Kind != ir.KindObject {
		return &ExitError{Msg: "refine_gen: semantics file is not a mapping"}
	}
	o := sem.AsObject()
	pattern := o.GetString("pattern")
	common := map[string]bool{"machine": true, "pattern": true, "max_retries": true}
	allowed := map[string]bool{}
	for key := range common {
		allowed[key] = true
	}
	switch pattern {
	case "linear-lifecycle":
		for _, key := range []string{"stages", "win_stage", "lose_stage", "reopen_to", "close_date_on", "advance_event", "win_event", "lose_event", "reopen_event", "overlay"} {
			allowed[key] = true
		}
	case "terminal-lifecycle":
		for _, key := range []string{"phases", "success_terminal", "failure_terminals", "retry", "retries", "success_flag"} {
			allowed[key] = true
		}
	case "saga":
		allowed["states"], allowed["obligations"] = true, true
	case "control-flow-only":
		allowed = map[string]bool{"machine": true, "pattern": true, "reason": true}
	default:
		// The caller emits the canonical unsupported-pattern diagnostic.
		return nil
	}
	if err := rejectUnknownKeys(o, allowed, "semantics"); err != nil {
		return err
	}
	if err := requireSchemaKind(o, "machine", ir.KindString, true, "semantics"); err != nil {
		return err
	}
	if err := requireSchemaKind(o, "pattern", ir.KindString, true, "semantics"); err != nil {
		return err
	}
	if err := requireSchemaKind(o, "max_retries", ir.KindNumber, false, "semantics"); err != nil {
		return err
	}
	switch pattern {
	case "linear-lifecycle":
		if err := requireStringArray(o, "stages"); err != nil {
			return err
		}
		for _, key := range []string{"win_stage", "lose_stage", "reopen_to", "close_date_on", "advance_event", "win_event", "lose_event", "reopen_event"} {
			if err := requireSchemaKind(o, key, ir.KindString, true, "semantics"); err != nil {
				return err
			}
		}
		if overlay := o.Get2("overlay"); overlay != nil {
			if overlay.Kind != ir.KindObject {
				return schemaError("semantics.overlay must be a mapping")
			}
			if err := rejectUnknownKeys(overlay.AsObject(), map[string]bool{"busy": true, "retry": true, "rollback": true, "fault": true}, "semantics.overlay"); err != nil {
				return err
			}
			for _, key := range overlay.AsObject().Keys() {
				if err := requireSchemaKind(overlay.AsObject(), key, ir.KindString, true, "semantics.overlay"); err != nil {
					return err
				}
			}
		}
	case "terminal-lifecycle":
		if err := requireStringArray(o, "phases"); err != nil {
			return err
		}
		if err := requireStringArray(o, "failure_terminals"); err != nil {
			return err
		}
		for _, key := range []string{"success_terminal", "success_flag"} {
			if err := requireSchemaKind(o, key, ir.KindString, true, "semantics"); err != nil {
				return err
			}
		}
		if o.Get2("retry") != nil && o.Get2("retries") != nil {
			return schemaError("semantics may declare retry or retries, not both")
		}
		if retry := o.Get2("retry"); retry != nil {
			if err := validateRetrySchema(retry, "semantics.retry"); err != nil {
				return err
			}
		}
		if retries := o.Get2("retries"); retries != nil {
			if retries.Kind != ir.KindArray {
				return schemaError("semantics.retries must be a list")
			}
			for i, retry := range retries.AsArray() {
				if err := validateRetrySchema(retry, fmt.Sprintf("semantics.retries[%d]", i)); err != nil {
					return err
				}
			}
		}
	case "saga":
		if err := requireStringArray(o, "states"); err != nil {
			return err
		}
		obligations := o.Get2("obligations")
		if obligations == nil || obligations.Kind != ir.KindObject {
			return schemaError("semantics.obligations must be a mapping")
		}
		for _, state := range obligations.AsObject().Keys() {
			entry := obligations.AsObject().Get2(state)
			where := "semantics.obligations." + state
			if entry == nil || entry.Kind != ir.KindObject {
				return schemaError(where + " must be a mapping")
			}
			if err := rejectUnknownKeys(entry.AsObject(), map[string]bool{"sets": true, "undo": true}, where); err != nil {
				return err
			}
			if err := requireSchemaKind(entry.AsObject(), "sets", ir.KindString, true, where); err != nil {
				return err
			}
			if err := requireSchemaKind(entry.AsObject(), "undo", ir.KindString, false, where); err != nil {
				return err
			}
		}
	case "control-flow-only":
		if err := requireSchemaKind(o, "reason", ir.KindString, true, "semantics"); err != nil {
			return err
		}
	}
	return nil
}

func schemaError(message string) error { return &ExitError{Msg: "refine_gen: " + message} }

func rejectUnknownKeys(o *ir.Object, allowed map[string]bool, where string) error {
	var unknown []string
	for _, key := range o.Keys() {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return schemaError(fmt.Sprintf("%s has unsupported key %s", where, ir.Repr(unknown[0])))
	}
	return nil
}

func requireSchemaKind(o *ir.Object, key string, kind ir.Kind, required bool, where string) error {
	v := o.Get2(key)
	if v == nil {
		if required {
			return schemaError(fmt.Sprintf("%s.%s is required", where, key))
		}
		return nil
	}
	if v.Kind != kind {
		return schemaError(fmt.Sprintf("%s.%s has the wrong type", where, key))
	}
	return nil
}

func requireStringArray(o *ir.Object, key string) error {
	v := o.Get2(key)
	if v == nil {
		return schemaError(fmt.Sprintf("semantics.%s is required", key))
	}
	if v.Kind != ir.KindArray {
		return schemaError(fmt.Sprintf("semantics.%s must be a list of strings", key))
	}
	for i, item := range v.AsArray() {
		if item == nil || item.Kind != ir.KindString {
			return schemaError(fmt.Sprintf("semantics.%s[%d] must be a string", key, i))
		}
	}
	return nil
}

func validateRetrySchema(v *ir.Value, where string) error {
	if v == nil || v.Kind != ir.KindObject {
		return schemaError(where + " must be a mapping")
	}
	if err := rejectUnknownKeys(v.AsObject(), map[string]bool{"state": true, "serves": true}, where); err != nil {
		return err
	}
	for _, key := range []string{"state", "serves"} {
		if err := requireSchemaKind(v.AsObject(), key, ir.KindString, true, where); err != nil {
			return err
		}
	}
	return nil
}

// RunWritten is Run, reporting the basenames of the files it wrote so callers
// (verify-formal) can distinguish freshly generated pairs from committed
// orphans.
func RunWritten(machinePath, semPath, outdir string) ([]string, error) {
	return RunWrittenTo(machinePath, semPath, outdir, os.Stdout)
}

// RunWrittenTo is RunWritten with an explicit status-output sink.
func RunWrittenTo(machinePath, semPath, outdir string, out io.Writer) ([]string, error) {
	snapshot, err := designlock.Acquire(filepath.Dir(filepath.Dir(machinePath)))
	if err != nil {
		return nil, err
	}
	written, retErr := RunWrittenInSnapshotTo(snapshot, machinePath, semPath, outdir, out)
	retErr = errors.Join(retErr, snapshot.Release())
	retErr = snapshot.LogicalError(retErr)
	if retErr != nil {
		return nil, retErr
	}
	return written, nil
}

// RunWrittenInSnapshot is RunWritten for an orchestrator which already owns
// the design snapshot lock.
var runWrittenAfterInputSnapshot = func() {}
var runWrittenAfterStalePlan = func() {}

func RunWrittenInSnapshot(snapshot *designlock.Lock, machinePath, semPath, outdir string) (written []string, retErr error) {
	return RunWrittenInSnapshotTo(snapshot, machinePath, semPath, outdir, os.Stdout)
}

// RunWrittenInSnapshotTo is RunWrittenInSnapshot with an explicit output sink.
func RunWrittenInSnapshotTo(snapshot *designlock.Lock, machinePath, semPath, outdir string, out io.Writer) (written []string, retErr error) {
	machineSource := filepath.Base(machinePath)
	semanticsSource := filepath.Base(semPath)
	if err := validateRefinementOwnerBase(machineSource, ".machine.json"); err != nil {
		return nil, &ExitError{Msg: "refine_gen: invalid machine input filename: " + err.Error()}
	}
	if err := validateRefinementOwnerBase(semanticsSource, ".semantics.yaml"); err != nil {
		return nil, &ExitError{Msg: "refine_gen: invalid semantics input filename: " + err.Error()}
	}
	semanticsSourceDir := ""
	if stableDir, err := snapshot.SourcePath(filepath.Dir(semPath)); err != nil {
		return nil, &ExitError{Msg: "refine_gen: resolve semantics inventory: " + err.Error()}
	} else if stableDir != filepath.Dir(semPath) || refinementPathWithin(snapshot.SourceRoot(), stableDir) {
		semanticsSourceDir = stableDir
	}
	if outdir == "" {
		outdir = filepath.Dir(semPath)
	}
	if err := snapshot.ValidateOutputDir(outdir); err != nil {
		return nil, &ExitError{Msg: "refine_gen: unsafe output directory: " + err.Error()}
	}
	machines, err := snapshot.MaterializeExternalTree(filepath.Dir(machinePath))
	if err != nil {
		return nil, &ExitError{Msg: "refine_gen: snapshot machine inventory: " + err.Error()}
	}
	defer func() { retErr = errors.Join(retErr, machines.Close()) }()
	machinePath = filepath.Join(machines.Path(), filepath.Base(machinePath))
	semantics, err := snapshot.MaterializeRegularFile(semPath)
	if err != nil {
		return nil, &ExitError{Msg: "refine_gen: snapshot semantics: " + err.Error()}
	}
	defer func() { retErr = errors.Join(retErr, semantics.Close()) }()
	semPath = semantics.Path()
	if err := snapshot.ResumeExpected("refine", "rerun `machinery refine` with the same arguments"); err != nil {
		return nil, err
	}
	runWrittenAfterInputSnapshot()
	if err := ir.ValidateTLAModuleInventory(filepath.Dir(machinePath)); err != nil {
		return nil, &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	machine, err := ir.LoadMachineJSON(machinePath)
	if err != nil {
		return nil, &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	data, err := os.ReadFile(semPath)
	if err != nil {
		return nil, &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	sem, err := ir.LoadYAML(data)
	if err != nil {
		return nil, &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	if sem.Kind != ir.KindObject {
		return nil, &ExitError{Msg: "refine_gen: semantics file is not a mapping"}
	}
	if err := validateSemanticsSchema(sem); err != nil {
		return nil, err
	}
	names := [2]string{filepath.Base(machinePath), filepath.Base(semPath)}
	pat := sem.AsObject().GetString("pattern")
	var mid string
	var files map[string]string
	var genErr error
	switch pat {
	case "linear-lifecycle":
		mid, files, genErr = EmitLifecycle(machine, sem, names)
	case "terminal-lifecycle":
		mid, files, genErr = EmitTerminal(machine, sem, names)
	case "saga":
		mid, files, genErr = EmitSaga(machine, sem, names)
	case "control-flow-only":
		if err := ValidateControlFlowOnly(machinePath, semPath); err != nil {
			return nil, err
		}
		stale, err := staleOwnedRefinementArtifacts(outdir, filepath.Dir(machinePath), semanticsSourceDir, machineSource, semanticsSource, nil)
		if err != nil {
			return nil, err
		}
		if len(stale) > 0 {
			runWrittenAfterStalePlan()
			expected := make([]designlock.OutputExpectation, 0, len(stale))
			for _, name := range stale {
				expected = append(expected, designlock.ExpectAbsent(filepath.Join(outdir, name.Name)))
			}
			if err := snapshot.PublishExpectedRooted("refine", "rerun `machinery refine` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
				return outputs.WithRoot(outdir, func(root *os.Root) error {
					return artifactset.ReconcilePlannedRooted(outdir, root, map[string][]byte{}, stale)
				})
			}); err != nil {
				return nil, err
			}
		}
		if _, err := fmt.Fprintf(out, "validated control-flow-only semantics for %s; no data-refinement artifacts claimed\n", machine.AsObject().GetString("id")); err != nil {
			return nil, fmt.Errorf("refine_gen: write status output: %w", err)
		}
		if err := snapshot.CheckUnchanged(); err != nil {
			return nil, err
		}
		return []string{}, nil
	default:
		return nil, &ExitError{Msg: fmt.Sprintf("refine_gen: unsupported pattern %s (linear-lifecycle, terminal-lifecycle, saga, control-flow-only)", ir.Repr(pat))}
	}
	if genErr != nil {
		return nil, genErr
	}
	if len(files) == 0 {
		return nil, &ExitError{Msg: "refine_gen: " + mid + ": generation produced no files"}
	}
	var fileNames []string
	for name := range files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)
	committed := make(map[string][]byte, len(fileNames))
	for _, name := range fileNames {
		body := files[name]
		// Stamp at write time (P-F10): the committed artifact records which
		// machinery version produced it; freshness diffs strip the line.
		switch {
		case strings.HasSuffix(name, ".tla"):
			body = version.StampTLAModule(body)
		case strings.HasSuffix(name, ".cfg"):
			body = version.StampCfg(body)
		}
		committed[name] = []byte(body)
		written = append(written, name)
	}
	replacements, err := guardedCurrentRefinementArtifacts(outdir, machineSource, semanticsSource, committed)
	if err != nil {
		return nil, err
	}
	stale, err := staleOwnedRefinementArtifacts(outdir, filepath.Dir(machinePath), semanticsSourceDir, machineSource, semanticsSource, committed)
	if err != nil {
		return nil, err
	}
	if len(stale) > 0 {
		runWrittenAfterStalePlan()
	}
	expected := make([]designlock.OutputExpectation, 0, len(fileNames)+len(stale))
	for _, name := range fileNames {
		expected = append(expected, designlock.ExpectFile(filepath.Join(outdir, name), committed[name], 0o644))
	}
	for _, name := range stale {
		expected = append(expected, designlock.ExpectAbsent(filepath.Join(outdir, name.Name)))
	}
	if wErr := snapshot.PublishExpectedRooted("refine", "rerun `machinery refine` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
		return outputs.WithRoot(outdir, func(root *os.Root) error {
			return artifactset.ReconcileGuardedRooted(outdir, root, committed, stale, replacements)
		})
	}); wErr != nil {
		return nil, wErr
	}
	if _, err := fmt.Fprintf(out, "generated %d files for %s (%s)\n", len(files), mid, pat); err != nil {
		return nil, fmt.Errorf("refine_gen: write status output: %w", err)
	}
	return written, nil
}

func guardedCurrentRefinementArtifacts(outdir, machineSource, semanticsSource string, files map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
	info, err := os.Lstat(outdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("refine_gen: output directory must be a real directory")
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	inspected := map[string][]byte{}
	conditions := map[string]artifactset.RemovalPrecondition{}
	for _, name := range names {
		body, condition, err := artifactset.InspectRemovalCandidate(outdir, name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		inspected[name], conditions[name] = body, condition
	}
	prefix := strings.TrimSuffix(machineSource, ".machine.json")
	anchor := prefix + "Data.tla"
	anchorOwned := false
	if body, ok := inspected[anchor]; ok {
		machineOwner, semanticsOwner, generated, err := refinementSourceOwner(anchor, body)
		if err != nil {
			return nil, err
		}
		anchorOwned = generated && machineOwner == machineSource && semanticsOwner == semanticsSource
	}
	for _, name := range names {
		body, exists := inspected[name]
		if !exists || bytes.Equal(body, files[name]) {
			continue
		}
		if name == anchor {
			if !anchorOwned {
				return nil, fmt.Errorf("refine_gen: refusing to replace foreign or manual artifact %s", name)
			}
			continue
		}
		if !anchorOwned || !canonicalRefinementFamilyMember(name, prefix, body) {
			return nil, fmt.Errorf("refine_gen: refusing to replace artifact %s without a canonical same-owner refinement family", name)
		}
	}
	out := make([]artifactset.RemovalPrecondition, 0, len(conditions))
	for _, name := range names {
		if condition, ok := conditions[name]; ok {
			out = append(out, condition)
		}
	}
	return out, nil
}

var refinementFamilySuffixes = []string{"Data.tla", "Data.cfg", "Contract.tla", "Refinement.tla", "Refinement.cfg"}

func staleOwnedRefinementArtifacts(outdir, machineDir, semanticsDir, machineSource, semanticsSource string, keep map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
	entries, err := os.ReadDir(outdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	staleSet := map[string]artifactset.RemovalPrecondition{}
entryLoop:
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, "Data.tla") {
			continue
		}
		body, _, err := artifactset.InspectRemovalCandidate(outdir, name)
		if err != nil {
			return nil, err
		}
		machineOwner, semanticsOwner, generated, ownerErr := refinementSourceOwner(name, body)
		if ownerErr != nil {
			return nil, fmt.Errorf("refine_gen: invalid source-ownership header in %s: %w", name, ownerErr)
		}
		if !generated {
			continue
		}
		prefix := strings.TrimSuffix(name, "Data.tla")
		if semanticsDir == "" {
			for _, suffix := range refinementFamilySuffixes {
				candidate := prefix + suffix
				if _, current := keep[candidate]; current {
					continue
				}
				_, _, inspectErr := artifactset.InspectRemovalCandidate(outdir, candidate)
				switch {
				case errors.Is(inspectErr, os.ErrNotExist):
				case inspectErr != nil:
					return nil, inspectErr
				default:
					return nil, fmt.Errorf("refine_gen: cannot safely reconcile non-current generated artifact %s for external semantics input %s; remove the stale generated family explicitly and rerun", candidate, semanticsSource)
				}
			}
			continue entryLoop
		}
		owned := machineOwner == machineSource && semanticsOwner == semanticsSource
		if !owned {
			_, statErr := os.Lstat(filepath.Join(machineDir, machineOwner))
			owned = errors.Is(statErr, os.ErrNotExist)
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return nil, statErr
			}
		}
		if !owned && semanticsDir != "" {
			_, statErr := os.Lstat(filepath.Join(semanticsDir, semanticsOwner))
			owned = errors.Is(statErr, os.ErrNotExist)
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return nil, statErr
			}
		}
		if !owned {
			continue
		}
		for _, suffix := range refinementFamilySuffixes {
			candidate := prefix + suffix
			if _, current := keep[candidate]; current {
				continue
			}
			_, condition, inspectErr := artifactset.InspectRemovalCandidate(outdir, candidate)
			if inspectErr == nil {
				staleSet[candidate] = condition
			} else if !errors.Is(inspectErr, os.ErrNotExist) {
				return nil, inspectErr
			}
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if _, current := keep[name]; current || strings.HasSuffix(name, "Data.tla") {
			continue
		}
		prefix, recognized := refinementFamilyPrefix(name)
		if !recognized {
			continue
		}
		anchor := prefix + "Data.tla"
		if _, err := os.Lstat(filepath.Join(outdir, anchor)); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		body, condition, err := artifactset.InspectRemovalCandidate(outdir, name)
		if err != nil {
			return nil, err
		}
		if !canonicalRefinementFamilyMember(name, prefix, body) {
			continue
		}
		if prefix != strings.TrimSuffix(machineSource, ".machine.json") {
			if _, err := os.Lstat(filepath.Join(machineDir, prefix+".machine.json")); err == nil {
				continue
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		staleSet[name] = condition
	}
	stale := make([]artifactset.RemovalPrecondition, 0, len(staleSet))
	for _, condition := range staleSet {
		stale = append(stale, condition)
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Name < stale[j].Name })
	return stale, nil
}

func refinementFamilyPrefix(name string) (string, bool) {
	for _, suffix := range refinementFamilySuffixes {
		if strings.HasSuffix(name, suffix) && len(name) > len(suffix) {
			return strings.TrimSuffix(name, suffix), true
		}
	}
	return "", false
}

func canonicalRefinementFamilyMember(name, prefix string, body []byte) bool {
	switch name {
	case prefix + "Contract.tla":
		lines := bytes.SplitN(body, []byte("\n"), 4)
		return len(lines) >= 3 && string(lines[0]) == "---- MODULE "+prefix+"Contract ----" &&
			bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) && bytes.HasPrefix(lines[2], []byte(`\* GENERATED. `))
	case prefix + "Refinement.tla":
		lines := bytes.SplitN(body, []byte("\n"), 4)
		want := `\* GENERATED. Proof that ` + prefix + `Data refines ` + prefix + `Contract under a refinement mapping.`
		return len(lines) >= 3 && string(lines[0]) == "---- MODULE "+prefix+"Refinement ----" &&
			bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) && string(lines[2]) == want
	case prefix + "Data.cfg":
		return canonicalRefinementDataConfig(body)
	case prefix + "Refinement.cfg":
		return canonicalRefinementProofConfig(body)
	default:
		return false
	}
}

func canonicalRefinementDataConfig(body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) < 4 || !strings.HasPrefix(lines[0], `\* machinery-version: `) || !strings.HasPrefix(lines[1], "CONSTANT MaxRetries = ") {
		return false
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(lines[1], "CONSTANT MaxRetries = ")); err != nil {
		return false
	}
	rest := strings.Join(lines[2:], "\n")
	for _, known := range []string{
		"SPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_StageValid\nINVARIANT Inv_Atomic\nINVARIANT Inv_DomainConsistent\nINVARIANT Inv_CloseDate\nPROPERTY StageForward\nPROPERTY Live_OverlayResolves",
		"SPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_Complete\nPROPERTY Inv_TerminalAbsorbing\nPROPERTY Live_Terminates",
		"SPECIFICATION Spec\nINVARIANT TypeOK\nINVARIANT Inv_NoSilentLoss\nINVARIANT Inv_CleanCompensation\nPROPERTY Live_Terminates",
	} {
		if rest == known {
			return true
		}
	}
	return false
}

func canonicalRefinementProofConfig(body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) != 6 || !strings.HasPrefix(lines[0], `\* machinery-version: `) || !strings.HasPrefix(lines[1], "CONSTANT MaxRetries = ") {
		return false
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(lines[1], "CONSTANT MaxRetries = ")); err != nil {
		return false
	}
	return strings.Join(lines[2:], "\n") == "SPECIFICATION Spec\nINVARIANT RefTypeOK\nPROPERTY RefSpec\nPROPERTY RefTermination"
}

func refinementSourceOwner(name string, body []byte) (machine, semantics string, generated bool, err error) {
	lines := bytes.SplitN(body, []byte("\n"), 4)
	module := strings.TrimSuffix(name, ".tla")
	if len(lines) < 3 || string(lines[0]) != "---- MODULE "+module+" ----" || !bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) {
		return "", "", false, nil
	}
	header := string(lines[2])
	var owner string
	for _, prefix := range []string{
		`\* GENERATED by machinery refine from `,
		`\* GENERATED by machinery refine (terminal-lifecycle) from `,
		`\* GENERATED by machinery refine (saga pattern) from `,
	} {
		if strings.HasPrefix(header, prefix) && strings.HasSuffix(header, ".") {
			owner = strings.TrimSuffix(strings.TrimPrefix(header, prefix), ".")
			break
		}
	}
	if owner == "" {
		return "", "", false, nil
	}
	machine, semantics, ok := strings.Cut(owner, " + ")
	if !ok || filepath.Base(machine) != machine || filepath.Base(semantics) != semantics ||
		!strings.HasSuffix(machine, ".machine.json") || !strings.HasSuffix(semantics, ".semantics.yaml") {
		return "", "", true, fmt.Errorf("expected portable machine and semantics basenames")
	}
	if err := validateRefinementOwnerBase(machine, ".machine.json"); err != nil {
		return "", "", true, fmt.Errorf("invalid machine owner: %w", err)
	}
	if err := validateRefinementOwnerBase(semantics, ".semantics.yaml"); err != nil {
		return "", "", true, fmt.Errorf("invalid semantics owner: %w", err)
	}
	machineStem := strings.TrimSuffix(machine, ".machine.json")
	if !strings.HasPrefix(module, machineStem) || module != machineStem+"Data" {
		return "", "", true, fmt.Errorf("data module %s does not match machine owner %s", module, machine)
	}
	return machine, semantics, true, nil
}

func validateRefinementOwnerBase(name, suffix string) error {
	if !strings.HasSuffix(name, suffix) {
		return fmt.Errorf("must end in %s", suffix)
	}
	if strings.Contains(name, " + ") || strings.Contains(name, ",") {
		return fmt.Errorf("contains a reserved ownership-header delimiter")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("contains a control character")
		}
	}
	return portablepath.ValidateBase(name)
}

func refinementPathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
