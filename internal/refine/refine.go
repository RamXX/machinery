// Package refine is the Go port of refine_gen.py: generates the data-refined
// model, abstract contract, and refinement mapping for a machine from a
// declarative semantics annotation, AFTER reconciling it against the machine.
package refine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ramirosalas/machinery/internal/ir"
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
	for n, node := range top {
		if ir.IsUpperFirst(n) {
			domainActual[n] = true
		}
		_ = node
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
	rbTargets := alwaysTargets(top[rollback])
	enters := map[string]bool{}
	for s := range domainActual {
		for _, tr := range ir.TransitionsOf(top[s], nil, "") {
			if tr.Target != "" && ir.Simple(tr.Target) == busy {
				enters[s] = true
			}
		}
	}
	if !setEq(rbTargets, enters) {
		die("%s routes to %s but the overlay is entered from %s; the rollback routing is incomplete or stale",
			rollback, brSorted(rbTargets), brSorted(enters))
	}
	closeOn := so.GetString("close_date_on")
	if !domainExpected[closeOn] {
		die("close_date_on %s is not a domain state", ir.Repr(closeOn))
	}
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
	reconciledFrom := ReconcileLifecycle(machine, sem)
	stages := strSlice(so.Get2("stages"))
	win := so.GetString("win_stage")
	lose := so.GetString("lose_stage")
	reopenTo := so.GetString("reopen_to")
	closeOn := so.GetString("close_date_on")
	maxr := intNum(so.Get2("max_retries"))
	initial := stages[0]
	if reopenTo != "" && !contains(stages, reopenTo) {
		die("reopen_to %s is not a declared stage (%s)", ir.Repr(reopenTo), strings.Join(stages, ", "))
	}

	terminal := []string{win, lose}
	domain := append(append([]string{}, stages...), terminal...)
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
\* perform, the retry bound MaxRetries = %d, and single-instance execution.`, sourceNames[0], sourceNames[1], maxr)

	data := fmt.Sprintf(`---- MODULE %sData ----
%s
EXTENDS Naturals

CONSTANT MaxRetries

Open == %s
Terminal == %s
Domain == Open \cup Terminal
Overlay == {"%s", "%s", "%s"}
None == "none"
Rank == %s
NextStage == %s

VARIABLES st, rc, stage, pending, prior, closeSet
vars == << st, rc, stage, pending, prior, closeSet >>

TypeOK ==
  /\ st \in (Domain \cup Overlay)
  /\ rc \in 0..MaxRetries
  /\ stage \in Domain
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

Domain_Next == StartAdvance \/ StartWin \/ StartLose \/ StartReopen
Overlay_Next == SaveDone \/ SaveLocked \/ SaveFail \/ RetryExhausted \/ RetryAgain \/ RolledBack
Next == Domain_Next \/ Overlay_Next

Spec == Init /\ [][Next]_vars /\ WF_vars(Overlay_Next)

Inv_StageValid == stage \in Domain
Inv_Atomic == (st \in Overlay) => (stage = prior)
Inv_DomainConsistent == (st \in Domain) => (st = stage /\ pending = None /\ prior = None)
Inv_CloseDate == (stage = "%s") => closeSet

StageForward ==
  [][ (stage' # stage) =>
        \/ Rank[stage'] > Rank[stage]
        \/ (stage \in Terminal /\ stage' = "%s") ]_stage

Live_OverlayResolves == (st \in Overlay) ~> (st \in Domain)
====
`,
		mid, header, q(stages), q(terminal), busy, retry, rollback, rankf, nextf,
		initial, initial, q(advanceable), busy, busy, win, busy, lose, busy, reopenTo,
		busy, closeOn, busy, retry, busy, rollback, retry, rollback, retry, busy, rollback, closeOn, reopenTo)

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

phaseBar == IF st \in Domain THEN "resting" ELSE "busy"
kindBar == IF stage \in Terminal THEN "terminal" ELSE "open"

DC == INSTANCE %sContract WITH phase <- phaseBar, kind <- kindBar

RefTypeOK == DC!CTypeOK
RefSpec == DC!CSpec
RefTermination == DC!CTermination
====
`, mid, mid, mid, mid, mid)

	refCfg := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT RefTypeOK\nPROPERTY RefSpec\nPROPERTY RefTermination\n", maxr)

	fmt.Fprintf(os.Stdout, "refine_gen: reconciled %s against the machine: %d domain states, overlay entered from %d states\n",
		mid, len(stages)+2, len(reconciledFrom))
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
	for n := range top {
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
	return phases, success, failures, retries, failRoutes, nil
}

// EmitTerminal mirrors refine_gen.emit_terminal.
func EmitTerminal(machine, sem *ir.Value, sourceNames [2]string) (string, map[string]string, error) {
	so := sem.AsObject()
	mid := ir.Title(so.GetString("machine"))
	phases, success, failures, retries, failRoutes, err := ReconcileTerminal(machine, sem)
	if err != nil {
		return "", nil, err
	}
	maxr := intNum(so.Get2("max_retries"))
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
	exhaustTo := map[string]string{}
	top := topStates(machine)
	for _, r := range retries {
		exhaustTo[r.State] = sortedSet(alwaysTargets(top[r.State]))[0]
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
\* on the success path, MaxRetries = %d, and single-instance execution.`, sourceNames[0], sourceNames[1], maxr)

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
		L = append(L, fmt.Sprintf(`RetryExhausted_%s == st = "%s" /\ %s >= MaxRetries /\ st' = "%s" /\ %s`, rs, rs, ctr, et, unch(nil)))
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
	fmt.Fprintf(os.Stdout, "refine_gen: reconciled %s against the machine: %d phases, %d retry overlay(s), %d failure terminal(s)\n",
		mid, len(phases), len(retries), len(failures))
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
	for n := range top {
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
func EmitSaga(machine, sem *ir.Value, sourceNames [2]string) (string, map[string]string, error) {
	if err := ReconcileSaga(machine, sem); err != nil {
		return "", nil, err
	}
	so := sem.AsObject()
	mid := ir.Title(so.GetString("machine"))
	states := strSlice(so.Get2("states"))
	oblObj := so.Get2("obligations").AsObject()
	maxr := intNum(so.Get2("max_retries"))
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
	L = append(L, fmt.Sprintf("\\* actors commit and undo, the retry bound MaxRetries = %d, single instance.", maxr))
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
	fmt.Fprintf(os.Stdout, "refine_gen: reconciled %s against the machine: %d forward steps, %d compensating obligations\n",
		mid, len(states), len(obligations))
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

func intNum(v *ir.Value) int {
	if v == nil {
		return 0
	}
	n := v.AsNumber()
	var i int
	if _, err := fmt.Sscanf(string(n), "%d", &i); err != nil {
		return 0
	}
	return i
}

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
	machine, err := ir.LoadMachineJSON(machinePath)
	if err != nil {
		return &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	data, err := os.ReadFile(semPath)
	if err != nil {
		return &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	sem, err := ir.LoadYAML(data)
	if err != nil {
		return &ExitError{Msg: "refine_gen: " + err.Error()}
	}
	if sem.Kind != ir.KindObject {
		return &ExitError{Msg: "refine_gen: semantics file is not a mapping"}
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
	default:
		return &ExitError{Msg: fmt.Sprintf("refine_gen: unsupported pattern %s (linear-lifecycle, terminal-lifecycle, saga)", ir.Repr(pat))}
	}
	if genErr != nil {
		return genErr
	}
	if len(files) == 0 {
		return &ExitError{Msg: "refine_gen: " + mid + ": generation produced no files"}
	}
	if outdir == "" {
		outdir = filepath.Dir(semPath)
	}
	if mkErr := os.MkdirAll(outdir, 0755); mkErr != nil {
		return mkErr
	}
	for name, body := range files {
		if wErr := os.WriteFile(filepath.Join(outdir, name), []byte(body), 0644); wErr != nil {
			return wErr
		}
	}
	fmt.Fprintf(os.Stdout, "generated %d files for %s (%s)\n", len(files), mid, pat)
	return nil
}
