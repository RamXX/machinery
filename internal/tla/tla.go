// Package tla is the Go port of tla_gen.py: translates a machine JSON into a
// TLA+ control-flow model plus a TLC config.
package tla

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

// ExitError carries a hard-error message that maps to Python's sys.exit(msg).
type ExitError struct{ Msg string }

func (e *ExitError) Error() string { return e.Msg }

func die(format string, args ...interface{}) {
	panic(&ExitError{Msg: fmt.Sprintf(format, args...)})
}

// Classify mirrors tla_gen.classify: domain = has `on` or is final; else
// overlay. Retry shape wins over the `on` heuristic: a bounded retry loop
// stays overlay even when it carries event handlers (an invariant may oblige
// an event to act during the backoff), because reclassifying it as domain
// would reset retry counters on its transitions and move it out of the
// overlay liveness class.
func Classify(states []ir.StateEntry) (domain, overlay map[string]bool) {
	domain, overlay = map[string]bool{}, map[string]bool{}
	for _, s := range states {
		o := s.Node.AsObject()
		if retryShaped(o) {
			overlay[s.Name] = true
			continue
		}
		if (o != nil && o.Get2("on") != nil) || o.GetString("type") == "final" {
			domain[s.Name] = true
		} else {
			overlay[s.Name] = true
		}
	}
	return
}

// retryShaped reports the bounded-retry loop shape: a fully guarded always
// plus an after.
func retryShaped(o *ir.Object) bool {
	if o == nil || o.Get2("always") == nil || o.Get2("after") == nil {
		return false
	}
	// Use the authoritative IR normalizer. A string transition is a valid
	// unguarded fallback; the old local object-only parser erased it and
	// misclassified the state as a fully guarded retry loop.
	branches := ir.TransitionsOf(ir.ObjectValue(o), nil, "")
	seenAlways := false
	for _, b := range branches {
		if b.Kind != "always" {
			continue
		}
		seenAlways = true
		if !b.HasGuard {
			return false
		}
	}
	return seenAlways
}

// RetryState is a state with a guarded always plus an after: a bounded retry loop.
type RetryState struct {
	Name string
	Node *ir.Value
}

func RetryStates(states []ir.StateEntry) []RetryState {
	var out []RetryState
	for _, s := range states {
		if retryShaped(s.Node.AsObject()) {
			out = append(out, RetryState{Name: s.Name, Node: s.Node})
		}
	}
	return out
}

func targetsOf(x *ir.Value, what, mid string) []string {
	var items []*ir.Value
	if x == nil {
		items = nil
	} else if x.Kind == ir.KindArray {
		items = x.AsArray()
	} else {
		items = []*ir.Value{x}
	}
	var targets []string
	for _, it := range items {
		if it == nil {
			continue
		}
		switch it.Kind {
		case ir.KindObject:
			if tv := it.AsObject().Get2("target"); tv != nil && tv.Kind == ir.KindString {
				targets = append(targets, tv.AsString())
			}
		case ir.KindString:
			targets = append(targets, it.AsString())
		}
	}
	if len(targets) == 0 {
		die("tla_gen: %s: %s has no target; the retry template needs one", mid, what)
	}
	return targets
}

func stStep(targets []string) string {
	set := map[string]bool{}
	for _, t := range targets {
		set[ir.Simple(t)] = true
	}
	var ts []string
	for t := range set {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	if len(ts) == 1 {
		return fmt.Sprintf(`st' = "%s"`, ts[0])
	}
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = fmt.Sprintf(`"%s"`, t)
	}
	return "st' \\in {" + strings.Join(parts, ", ") + "}"
}

func setExpr(s map[string]bool) string {
	var xs []string
	for x := range s {
		xs = append(xs, x)
	}
	sort.Strings(xs)
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = fmt.Sprintf(`"%s"`, x)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func declaresRefusalHandler(state *ir.Value, handler string) bool {
	o := state.AsObject()
	if o == nil {
		return false
	}
	if o.GetObject("on").Get2(handler) != nil {
		return true
	}
	if strings.HasPrefix(handler, "after:") {
		return o.GetObject("after").Get2(strings.TrimPrefix(handler, "after:")) != nil
	}
	for _, inv := range ir.InvokesOf(state) {
		io := inv.AsObject()
		if io == nil {
			continue
		}
		for _, branch := range []string{"onDone", "onError"} {
			if handler == io.GetString("src")+"."+branch && io.Get2(branch) != nil {
				return true
			}
		}
	}
	return false
}

// Check runs the generator's full admissibility pass over one machine and
// reports the refusal without writing anything: the G3 gate calls it so the
// structural lint and the formal generator admit the SAME machine subset
// (S11 of the dogfood systemic findings: G3 once passed a retry state carrying
// two after entries that verify-formal then refused, so a wave went
// G3-green and formally ungenerable). Deliberately a thin wrapper over
// Generate rather than a copied constraint list: one validator, one truth;
// the copies were the bug one level up.
func Check(path string) error {
	_, _, _, err := Generate(path)
	return err
}

// Generate mirrors tla_gen.generate(path) -> (mid, tla, cfg).
func Generate(path string) (mid, tla, cfg string, err error) {
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return "", "", "", &ExitError{Msg: "tla_gen: " + readErr.Error()}
	}
	return GenerateBytes(path, raw)
}

// GenerateBytes generates a TLA+ model from an authoritative byte snapshot.
// source is used only as the stable logical identity in diagnostics.
func GenerateBytes(source string, raw []byte) (mid, tla, cfg string, err error) {
	m, loadErr := ir.LoadMachineJSONBytes(source, raw)
	if loadErr != nil {
		return "", "", "", &ExitError{Msg: "tla_gen: " + loadErr.Error()}
	}
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
			} else {
				panic(r)
			}
		}
	}()
	mid, tla, cfg = generateFromMachine(m, source)
	return mid, tla, cfg, nil
}

func generateFromMachine(m *ir.Value, path string) (string, string, string) {
	ro := m.AsObject()
	mid, nameErr := ir.TLAModuleName(m)
	if nameErr != nil {
		die("tla_gen: %s: %v", filepath.Base(path), nameErr)
	}

	allStates := ir.WalkStates(ro.Get2("states"), "")
	if len(allStates) == 0 {
		die("tla_gen: %s: machine has no states", mid)
	}
	var nested []string
	for _, s := range allStates {
		if strings.Contains(s.Path, ".") {
			nested = append(nested, s.Path)
		}
	}
	if len(nested) > 0 {
		sortedN := append([]string{}, nested...)
		sort.Strings(sortedN)
		die("tla_gen: %s: nested states are not supported at rung 3 (%s); flatten the machine or extend the generator",
			mid, strings.Join(sortedN, ", "))
	}
	if problems := ir.TransitionProblems(m); len(problems) > 0 {
		die("tla_gen: %s: malformed transition IR: %s", mid, strings.Join(problems, "; "))
	}
	for _, s := range allStates {
		stype := s.Node.AsObject().GetString("type")
		if stype != "" && stype != "atomic" && stype != "compound" && stype != "final" {
			die("tla_gen: %s: unsupported state type %s in %s", mid, ir.Repr(stype), s.Name)
		}
	}

	states := allStates
	var names []string
	for _, s := range states {
		names = append(names, s.Name)
	}
	domain, overlay := Classify(states)
	retries := RetryStates(states)
	rcOf := map[string]string{}
	var counters []string
	for i, r := range retries {
		v := fmt.Sprintf("rc%d", i+1)
		rcOf[r.Name] = v
		counters = append(counters, v)
	}
	initial := ro.GetString("initial")
	if initial == "" {
		die("tla_gen: %s: machine has no initial state", mid)
	}
	var finalStates []string
	for _, s := range states {
		if s.Node.AsObject().GetString("type") == "final" {
			finalStates = append(finalStates, s.Name)
		}
	}
	sort.Strings(finalStates)
	finalSet := map[string]bool{}
	for _, f := range finalStates {
		finalSet[f] = true
	}

	var exhaustiveNotes [][2]string
	var refusalNotes [][3]string
	for _, s := range states {
		note := strings.TrimSpace(s.Node.AsObject().GetString("_exhaustive"))
		if note != "" {
			exhaustiveNotes = append(exhaustiveNotes, [2]string{s.Name, note})
		}
		if refusal := s.Node.AsObject().Get2("_refusal"); refusal != nil {
			if refusal.Kind != ir.KindObject || refusal.AsObject().Len() == 0 {
				die("tla_gen: %s: state %s _refusal must be a non-empty object mapping declared handlers to non-empty reasons", mid, s.Name)
			}
			for _, handler := range refusal.AsObject().Keys() {
				v := refusal.AsObject().Get2(handler)
				if strings.TrimSpace(handler) == "" || v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
					die("tla_gen: %s: state %s _refusal entry %s must have a non-empty string reason", mid, s.Name, ir.Repr(handler))
				}
				if !declaresRefusalHandler(s.Node, handler) {
					die("tla_gen: %s: state %s _refusal names handler %s, but the state declares no matching on:, after:, or invoke branch", mid, s.Name, ir.Repr(handler))
				}
				refusalNotes = append(refusalNotes, [3]string{s.Name, handler, v.AsString()})
			}
		}
	}
	claimLiveness := len(refusalNotes) == 0

	counterUpdates := func(src, tgt string) map[string]string {
		ups := map[string]string{}
		for _, v := range rcOf {
			if domain[src] {
				ups[v] = "0"
			} else if domain[tgt] {
				ups[v] = "0"
			} else {
				ups[v] = v
			}
		}
		return ups
	}

	var domActions, ovlActions, defs, comments []string
	idx := 0
	emit := func(s ir.StateEntry, tr ir.Transition) {
		idx++
		tgt := ir.Simple(tr.Target)
		if tgt == "" {
			tgt = s.Name
		}
		name := fmt.Sprintf("T%d", idx)
		ups := counterUpdates(s.Name, tgt)
		parts := []string{fmt.Sprintf(`st = "%s"`, s.Name), fmt.Sprintf(`st' = "%s"`, tgt)}
		var upKeys []string
		for k := range ups {
			upKeys = append(upKeys, k)
		}
		sort.Strings(upKeys)
		for _, v := range upKeys {
			parts = append(parts, fmt.Sprintf("%s' = %s", v, ups[v]))
		}
		defs = append(defs, name+" == "+strings.Join(parts, " /\\ "))
		var trig string
		if tr.Event != "" {
			trig = tr.Kind + ":" + tr.Event
		} else {
			trig = tr.Kind
		}
		comments = append(comments, fmt.Sprintf("  \\* %s: %s -%s-> %s", name, s.Name, trig, tgt))
		if domain[s.Name] {
			domActions = append(domActions, name)
		} else {
			ovlActions = append(ovlActions, name)
		}
	}
	for _, s := range states {
		if _, ok := rcOf[s.Name]; ok {
			// a retry-shaped state's transitions are generated from its loop
			// shape below; any other transition source would be silently
			// dropped from the model, so each is an unsupported shape, not a
			// skip: on: handlers, invoke onDone/onError, and state-level onDone
			so := s.Node.AsObject()
			// Event handlers on a retry state are legitimate (an invariant may
			// oblige an event to act during the backoff: the ignore-consistency
			// law makes handling, not ignoring, the required shape). They emit
			// as ordinary actions: a self-loop leaves every retry counter
			// unchanged and an exit follows the domain rule, both by the same
			// counterUpdates the rest of the model uses. The loop shape
			// (always + after) stays with the retry template below.
			for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
				if tr.Kind == "on" {
					emit(s, tr)
				}
			}
			if so.Get2("invoke") != nil {
				die("tla_gen: %s: retry state %s declares an invoke, whose onDone/onError transitions rung 3 would silently drop; route the completion through a non-retry state or extend the generator", mid, s.Name)
			}
			if so.Get2("onDone") != nil {
				die("tla_gen: %s: retry state %s declares a state-level onDone, which rung 3 would silently drop; route it through a non-retry state or extend the generator", mid, s.Name)
			}
			continue
		}
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
			emit(s, tr)
		}
	}

	for _, r := range retries {
		rn := r.Name
		rnode := r.Node.AsObject()
		var rcVar = rcOf[rn]
		aStep := stStep(targetsOf(rnode.Get2("always"), fmt.Sprintf("retry state %s always", rn), mid))
		afterObj := rnode.Get2("after").AsObject()
		if afterObj.Len() != 1 {
			die("tla_gen: %s: retry state %s has %d after entries; the retry template needs exactly one",
				mid, rn, afterObj.Len())
		}
		var afterKey string
		for _, k := range afterObj.Keys() {
			afterKey = k
		}
		fStep := stStep(targetsOf(afterObj.Get2(afterKey), fmt.Sprintf("retry state %s after", rn), mid))
		var others []string
		for _, v := range counters {
			if v != rcVar {
				others = append(others, v)
			}
		}
		unch := ""
		if len(others) > 0 {
			parts := make([]string, len(others))
			for i, v := range others {
				parts[i] = fmt.Sprintf("%s' = %s", v, v)
			}
			unch = " /\\ " + strings.Join(parts, " /\\ ")
		}
		defs = append(defs, fmt.Sprintf(`RetryExhausted_%s == st = "%s" /\ %s >= MaxRetries /\ %s /\ %s' = %s%s`,
			rn, rn, rcVar, aStep, rcVar, rcVar, unch))
		defs = append(defs, fmt.Sprintf(`RetryAgain_%s == st = "%s" /\ %s < MaxRetries /\ %s /\ %s' = %s + 1%s`,
			rn, rn, rcVar, fStep, rcVar, rcVar, unch))
		ovlActions = append(ovlActions, fmt.Sprintf("RetryExhausted_%s", rn), fmt.Sprintf("RetryAgain_%s", rn))
	}

	if len(finalStates) > 0 {
		defs = append(defs, "Terminated == st \\in Final /\\ UNCHANGED vars")
	}

	varlist := "st"
	if len(counters) > 0 {
		varlist += ", " + strings.Join(counters, ", ")
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("---- MODULE %s ----", mid))
	lines = append(lines, `EXTENDS Naturals`)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("\\* Generated from %s by machinery tla. Control-flow model.", filepath.Base(path)))
	lines = append(lines, "\\*")
	lines = append(lines, "\\* ASSUMPTIONS (what this abstraction erases; the proof is conditional on them):")
	lines = append(lines, "\\*   1. Guards are erased to nondeterminism: SOUND for safety. For LIVENESS this")
	lines = append(lines, "\\*      is conditional on every fully guarded branch list being exhaustive.")
	lines = append(lines, "\\*      machine_lint requires an unguarded fallback or an _exhaustive note; where")
	lines = append(lines, "\\*      an _exhaustive note is used TLC CANNOT verify it, so the liveness result")
	lines = append(lines, "\\*      below is only as sound as these hand-checked, UNVERIFIED claims:")
	if len(exhaustiveNotes) > 0 {
		for _, n := range exhaustiveNotes {
			lines = append(lines, fmt.Sprintf("\\*      - UNVERIFIED, state %s: %s", n[0], n[1]))
		}
	} else if claimLiveness {
		// Preserve the established generated bytes for machines whose proof
		// posture did not change. Refusal-bearing machines use the precise
		// always-list wording below because guarded handlers intentionally lack
		// a fallback there.
		lines = append(lines, "\\*      (none here: every guarded branch list has an unguarded fallback)")
	} else {
		lines = append(lines, "\\*      (none here: every guarded always-list has an unguarded fallback)")
	}
	if !claimLiveness {
		lines = append(lines, "\\*      Handler refusal is permitted in this machine. A refused trigger leaves")
		lines = append(lines, "\\*      the state unchanged, so this rung checks safety only and makes no")
		lines = append(lines, "\\*      fairness or overlay-resolution liveness claim:")
		for _, n := range refusalNotes {
			lines = append(lines, fmt.Sprintf("\\*      - state %s, handler %s: %s", n[0], n[1], n[2]))
		}
	}
	lines = append(lines, "\\*   2. Every invoke resolves exactly once (onDone or onError; no lost or")
	lines = append(lines, "\\*      duplicated completion) and every after timer eventually fires.")
	lines = append(lines, "\\*   3. Single machine instance; no interleaving with other instances or")
	lines = append(lines, "\\*      machines, no message loss/duplication/reordering between machines.")
	lines = append(lines, "\\*   4. Context data, event payloads, action effects, and real time (the")
	lines = append(lines, "\\*      _delays values) are not modeled at this rung; the data-refined rung")
	lines = append(lines, "\\*      (refine_gen) and the implementation tests carry those.")
	if len(counters) > 0 {
		lines = append(lines, "\\*   5. Retry counters (rc*) reset to 0 on every transition that leaves from")
		lines = append(lines, "\\*      or lands on a domain state; a counter surviving a domain hop is not")
		lines = append(lines, "\\*      representable at this rung.")
		lines = append(lines, "\\*   6. Retry-shaped states (fully guarded always + after) are modeled as the")
		lines = append(lines, "\\*      concrete bounded loop: the guarded always list is replaced by the")
		lines = append(lines, "\\*      exhaustion test rc >= MaxRetries and the after timer by the retry step")
		lines = append(lines, "\\*      rc < MaxRetries; the guards themselves are erased (see 1).")
	}
	lines = append(lines, "CONSTANT MaxRetries")
	lines = append(lines, fmt.Sprintf("VARIABLES %s", varlist))
	lines = append(lines, fmt.Sprintf("vars == << %s >>", varlist))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("States == %s", setExpr(setOf(names))))
	lines = append(lines, fmt.Sprintf("Domain == %s", setExpr(domain)))
	lines = append(lines, fmt.Sprintf("Overlay == %s", setExpr(overlay)))
	if len(finalStates) > 0 {
		lines = append(lines, fmt.Sprintf("Final == %s", setExpr(finalSet)))
	}
	lines = append(lines, "")
	tycounts := "TRUE"
	if len(counters) > 0 {
		var tps []string
		for _, v := range counters {
			tps = append(tps, fmt.Sprintf("%s \\in 0..MaxRetries", v))
		}
		tycounts = strings.Join(tps, " /\\ ")
	}
	lines = append(lines, fmt.Sprintf("TypeOK == st \\in States /\\ %s", tycounts))
	initCounts := ""
	for _, v := range counters {
		initCounts += fmt.Sprintf(" /\\ %s = 0", v)
	}
	lines = append(lines, fmt.Sprintf(`Init == st = "%s"%s`, initial, initCounts))
	lines = append(lines, "")
	lines = append(lines, comments...)
	lines = append(lines, "")
	lines = append(lines, defs...)
	lines = append(lines, "")
	if len(domActions) > 0 {
		lines = append(lines, "DomainNext == "+strings.Join(domActions, " \\/ "))
	} else {
		lines = append(lines, "DomainNext == FALSE")
	}
	if len(ovlActions) > 0 {
		lines = append(lines, "OverlayNext == "+strings.Join(ovlActions, " \\/ "))
	} else {
		lines = append(lines, "OverlayNext == FALSE")
	}
	next := "Next == DomainNext \\/ OverlayNext"
	if len(finalStates) > 0 {
		next += " \\/ Terminated"
	}
	lines = append(lines, next)
	lines = append(lines, "")
	if claimLiveness {
		lines = append(lines, "Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
	} else {
		lines = append(lines, "Spec == Init /\\ [][Next]_vars")
	}
	lines = append(lines, "")
	if !claimLiveness {
		lines = append(lines, "\\* Live_OverlayResolves intentionally omitted: _refusal permits persistent stuttering.")
	} else if len(domain) == 0 {
		// A perpetual envelope (timer-driven breaker, poller, health monitor)
		// has no resting domain state and no final: Overlay ~> Domain would be
		// unsatisfiable on a correct machine. Liveness reduces to deadlock
		// freedom, which TLC checks by default; the property stays declared so
		// the .cfg keeps one shape.
		lines = append(lines, "\\* Perpetual envelope: no domain state to resolve into; liveness is deadlock freedom.")
		lines = append(lines, "Live_OverlayResolves == TRUE")
	} else {
		lines = append(lines, "Live_OverlayResolves == (st \\in Overlay) ~> (st \\in Domain)")
	}
	lines = append(lines, "====")
	tlaOut := strings.Join(lines, "\n") + "\n"

	maxRetries := 3
	if v := ro.Get2("_max_retries"); v != nil {
		n, err := v.AsNumber().Int64()
		if v.Kind != ir.KindNumber || err != nil || n < 1 {
			die("tla_gen: %s: _max_retries must be a positive integer", mid)
		}
		maxRetries = int(n)
	}
	cfgOut := fmt.Sprintf("CONSTANT MaxRetries = %d\nSPECIFICATION Spec\nINVARIANT TypeOK\n", maxRetries)
	if claimLiveness {
		cfgOut += "PROPERTY Live_OverlayResolves\n"
	}
	return mid, tlaOut, cfgOut
}

func setOf(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// Run is the `machinery tla <machine.json> [out-dir]` entrypoint.
func Run(path, outdir string) error {
	return RunTo(path, outdir, os.Stdout)

}

// RunTo is Run with an explicit status-output sink.
func RunTo(path, outdir string, out io.Writer) error {
	_, err := RunWrittenTo(path, outdir, out)
	return err
}

// RunWritten is Run, reporting the basenames of the files it wrote so callers
// (verify-formal) can distinguish freshly generated pairs from committed
// orphans.
func RunWritten(path, outdir string) ([]string, error) {
	return RunWrittenTo(path, outdir, os.Stdout)
}

// RunWrittenTo is RunWritten with an explicit status-output sink.
func RunWrittenTo(path, outdir string, out io.Writer) ([]string, error) {
	snapshot, err := designlock.Acquire(filepath.Dir(filepath.Dir(path)))
	if err != nil {
		return nil, err
	}
	written, retErr := RunWrittenInSnapshotTo(snapshot, path, outdir, out)
	retErr = errors.Join(retErr, snapshot.Release())
	retErr = snapshot.LogicalError(retErr)
	if retErr != nil {
		return nil, retErr
	}
	return written, nil
}

// RunWrittenInSnapshot is RunWritten for an orchestrator (verify-formal)
// which already holds the design snapshot lock.
var runWrittenAfterSourceSnapshot = func() {}
var runWrittenAfterStalePlan = func() {}

func RunWrittenInSnapshot(snapshot *designlock.Lock, path, outdir string) ([]string, error) {
	return RunWrittenInSnapshotTo(snapshot, path, outdir, os.Stdout)
}

// RunWrittenInSnapshotTo is RunWrittenInSnapshot with an explicit output sink.
func RunWrittenInSnapshotTo(snapshot *designlock.Lock, path, outdir string, out io.Writer) ([]string, error) {
	if err := snapshot.ResumeExpected("tla", "rerun `machinery tla` with the same arguments"); err != nil {
		return nil, err
	}
	sourcePath, err := snapshot.SourcePath(path)
	if err != nil {
		return nil, fmt.Errorf("tla_gen: resolve immutable machine source: %w", err)
	}
	runWrittenAfterSourceSnapshot()
	if err := ir.ValidateTLAModuleInventory(filepath.Dir(sourcePath)); err != nil {
		return nil, fmt.Errorf("tla_gen: %w", err)
	}
	mid, tla, cfg, err := Generate(sourcePath)
	if err != nil {
		return nil, err
	}
	if outdir == "" {
		outdir = filepath.Dir(path)
	}
	// Stamp at write time, not in Generate: the pack generator embeds
	// Generate's output in hash-covered pack files, where a version stamp
	// would churn the content hash on every release (P-F10).
	files := map[string][]byte{
		mid + ".tla": []byte(version.StampTLAModule(tla)),
		mid + ".cfg": []byte(version.StampCfg(cfg)),
	}
	replacements, err := guardedCurrentTLAArtifacts(outdir, filepath.Base(sourcePath), files)
	if err != nil {
		return nil, err
	}
	stale, err := staleOwnedTLAArtifacts(outdir, filepath.Dir(sourcePath), files)
	if err != nil {
		return nil, err
	}
	if len(stale) > 0 {
		runWrittenAfterStalePlan()
	}
	expected := []designlock.OutputExpectation{
		designlock.ExpectFile(filepath.Join(outdir, mid+".tla"), files[mid+".tla"], 0o644),
		designlock.ExpectFile(filepath.Join(outdir, mid+".cfg"), files[mid+".cfg"], 0o644),
	}
	for _, name := range stale {
		expected = append(expected, designlock.ExpectAbsent(filepath.Join(outdir, name.Name)))
	}
	if wErr := snapshot.PublishExpectedRooted("tla", "rerun `machinery tla` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
		return outputs.WithRoot(outdir, func(root *os.Root) error {
			return artifactset.ReconcileGuardedRooted(outdir, root, files, stale, replacements)
		})
	}); wErr != nil {
		return nil, wErr
	}
	if _, err := fmt.Fprintf(out, "wrote %s.tla and %s.cfg to %s\n", mid, mid, outdir); err != nil {
		return nil, fmt.Errorf("tla_gen: write status output: %w", err)
	}
	return []string{mid + ".tla", mid + ".cfg"}, nil
}

func guardedCurrentTLAArtifacts(outdir, machineSource string, files map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
	info, err := os.Lstat(outdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("tla_gen: output directory must be a real directory")
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
	for _, name := range names {
		body, exists := inspected[name]
		if !exists || bytes.Equal(body, files[name]) {
			continue
		}
		switch filepath.Ext(name) {
		case ".tla":
			owner, generated, err := canonicalTLAOwner(name, body)
			if err != nil {
				return nil, err
			}
			if !generated || owner != machineSource {
				return nil, fmt.Errorf("tla_gen: refusing to replace foreign or manual artifact %s", name)
			}
		case ".cfg":
			anchor := strings.TrimSuffix(name, ".cfg") + ".tla"
			anchorBody, ok := inspected[anchor]
			if !ok || !canonicalTLAConfig(body) {
				return nil, fmt.Errorf("tla_gen: refusing to replace unowned config %s", name)
			}
			owner, generated, err := canonicalTLAOwner(anchor, anchorBody)
			if err != nil || !generated || owner != machineSource {
				return nil, fmt.Errorf("tla_gen: refusing to replace config %s without a same-owner generated module", name)
			}
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

func staleOwnedTLAArtifacts(outdir, machineDir string, keep map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
	entries, err := os.ReadDir(outdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var stale []artifactset.RemovalPrecondition
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".tla") {
			continue
		}
		if _, current := keep[name]; current {
			continue
		}
		body, condition, err := artifactset.InspectRemovalCandidate(outdir, name)
		if err != nil {
			return nil, err
		}
		source, generated, headerErr := canonicalTLAOwner(name, body)
		if headerErr != nil {
			return nil, headerErr
		}
		if !generated {
			continue
		}
		if _, err := os.Lstat(filepath.Join(machineDir, source)); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		stale = append(stale, condition)
		cfg := strings.TrimSuffix(name, ".tla") + ".cfg"
		if _, kept := keep[cfg]; !kept {
			_, cfgCondition, err := artifactset.InspectRemovalCandidate(outdir, cfg)
			if err == nil {
				stale = append(stale, cfgCondition)
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".cfg") {
			continue
		}
		if _, current := keep[name]; current {
			continue
		}
		anchor := strings.TrimSuffix(name, ".cfg") + ".tla"
		if _, err := os.Lstat(filepath.Join(outdir, anchor)); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		body, condition, err := artifactset.InspectRemovalCandidate(outdir, name)
		if err != nil {
			return nil, err
		}
		if !canonicalTLAConfig(body) {
			continue
		}
		source := strings.TrimSuffix(name, ".cfg") + ".machine.json"
		if _, err := os.Lstat(filepath.Join(machineDir, source)); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		stale = append(stale, condition)
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Name < stale[j].Name })
	return stale, nil
}

func canonicalTLAConfig(body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) != 4 && len(lines) != 5 {
		return false
	}
	if !strings.HasPrefix(lines[0], `\* machinery-version: `) || lines[1] == "" || !strings.HasPrefix(lines[1], "CONSTANT MaxRetries = ") ||
		lines[2] != "SPECIFICATION Spec" || lines[3] != "INVARIANT TypeOK" {
		return false
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(lines[1], "CONSTANT MaxRetries = ")); err != nil {
		return false
	}
	return len(lines) == 4 || lines[4] == "PROPERTY Live_OverlayResolves"
}

func canonicalTLAOwner(name string, body []byte) (string, bool, error) {
	lines := bytes.SplitN(body, []byte("\n"), 6)
	module := strings.TrimSuffix(name, ".tla")
	if len(lines) < 5 || string(lines[0]) != "---- MODULE "+module+" ----" ||
		!bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) || string(lines[2]) != "EXTENDS Naturals" || len(lines[3]) != 0 {
		return "", false, nil
	}
	const prefix = `\* Generated from `
	const suffix = ` by machinery tla. Control-flow model.`
	header := string(lines[4])
	if !strings.HasPrefix(header, prefix) || !strings.HasSuffix(header, suffix) {
		return "", false, nil
	}
	source := strings.TrimSuffix(strings.TrimPrefix(header, prefix), suffix)
	if filepath.Base(source) != source || !strings.HasSuffix(source, ".machine.json") {
		return "", true, fmt.Errorf("tla_gen: generated artifact %s has invalid source-ownership header", name)
	}
	if err := portablepath.ValidateBase(source); err != nil {
		return "", true, fmt.Errorf("tla_gen: generated artifact %s has non-portable source owner: %w", name, err)
	}
	if strings.TrimSuffix(source, ".machine.json") != module {
		return "", true, fmt.Errorf("tla_gen: generated artifact %s module does not match source owner %s", name, source)
	}
	return source, true, nil
}
