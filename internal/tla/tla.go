// Package tla is the Go port of tla_gen.py: translates a machine JSON into a
// TLA+ control-flow model plus a TLC config.
package tla

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ramirosalas/machinery/internal/ir"
)

// ExitError carries a hard-error message that maps to Python's sys.exit(msg).
type ExitError struct{ Msg string }

func (e *ExitError) Error() string { return e.Msg }

func die(format string, args ...interface{}) {
	panic(&ExitError{Msg: fmt.Sprintf(format, args...)})
}

// Classify mirrors tla_gen.classify: domain = has `on` or is final; else overlay.
func Classify(states []ir.StateEntry) (domain, overlay map[string]bool) {
	domain, overlay = map[string]bool{}, map[string]bool{}
	for _, s := range states {
		o := s.Node.AsObject()
		if (o != nil && o.Get2("on") != nil) || o.GetString("type") == "final" {
			domain[s.Name] = true
		} else {
			overlay[s.Name] = true
		}
	}
	return
}

// RetryStates mirrors tla_gen.retry_states: states with guarded always + after.
type RetryState struct {
	Name string
	Node *ir.Value
}

func RetryStates(states []ir.StateEntry) []RetryState {
	var out []RetryState
	for _, s := range states {
		o := s.Node.AsObject()
		if o.Get2("always") != nil && o.Get2("after") != nil {
			branches := normAlways(o.Get2("always"))
			if len(branches) > 0 {
				allGuarded := true
				for _, b := range branches {
					if !b.HasGuard {
						allGuarded = false
					}
				}
				if allGuarded {
					out = append(out, RetryState{Name: s.Name, Node: s.Node})
				}
			}
		}
	}
	return out
}

type alwaysBranch struct {
	HasGuard bool
	Target   string
	HasTgt   bool
}

func normAlways(v *ir.Value) []alwaysBranch {
	var items []*ir.Value
	if v == nil {
		return nil
	}
	if v.Kind == ir.KindArray {
		items = v.AsArray()
	} else {
		items = []*ir.Value{v}
	}
	var out []alwaysBranch
	for _, it := range items {
		if it == nil || it.Kind != ir.KindObject {
			continue
		}
		o := it.AsObject()
		b := alwaysBranch{}
		if gv := o.Get2("guard"); gv != nil && gv.Kind == ir.KindString && gv.AsString() != "" {
			b.HasGuard = true
		}
		if tv := o.Get2("target"); tv != nil && tv.Kind == ir.KindString {
			b.Target = tv.AsString()
			b.HasTgt = true
		}
		out = append(out, b)
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
		if it.Kind == ir.KindObject {
			if tv := it.AsObject().Get2("target"); tv != nil && tv.Kind == ir.KindString {
				targets = append(targets, tv.AsString())
			}
		} else if it.Kind == ir.KindString {
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

// Generate mirrors tla_gen.generate(path) -> (mid, tla, cfg).
func Generate(path string) (mid, tla, cfg string, err error) {
	m, loadErr := ir.LoadMachineJSON(path)
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
	mid, tla, cfg = generateFromMachine(m, path)
	return mid, tla, cfg, nil
}

func generateFromMachine(m *ir.Value, path string) (string, string, string) {
	ro := m.AsObject()
	mid := ro.GetString("id")
	if mid == "" {
		mid = "machine"
	}
	mid = ir.Title(mid)

	allStates := ir.WalkStates(ro.Get2("states"), "")
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
	for _, s := range states {
		note := strings.TrimSpace(s.Node.AsObject().GetString("_exhaustive"))
		if note != "" {
			exhaustiveNotes = append(exhaustiveNotes, [2]string{s.Name, note})
		}
	}

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
	for _, s := range states {
		if _, ok := rcOf[s.Name]; ok {
			continue
		}
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
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
	}

	for _, r := range retries {
		rn := r.Name
		rnode := r.Node.AsObject()
		var rcVar string = rcOf[rn]
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
	lines = append(lines, fmt.Sprintf("\\* Generated from %s by tools/tla_gen.py. Control-flow model.", filepath.Base(path)))
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
	} else {
		lines = append(lines, "\\*      (none here: every guarded branch list has an unguarded fallback)")
	}
	lines = append(lines, "\\*   2. Every invoke resolves exactly once (onDone or onError; no lost or")
	lines = append(lines, "\\*      duplicated completion) and every after timer eventually fires.")
	lines = append(lines, "\\*   3. Single machine instance; no interleaving with other instances or")
	lines = append(lines, "\\*      machines, no message loss/duplication/reordering between machines.")
	lines = append(lines, "\\*   4. Context data, event payloads, action effects, and real time (the")
	lines = append(lines, "\\*      _delays values) are not modeled at this rung; the data-refined rung")
	lines = append(lines, "\\*      (refine_gen) and the implementation tests carry those.")
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
	lines = append(lines, "Spec == Init /\\ [][Next]_vars /\\ WF_vars(OverlayNext)")
	lines = append(lines, "")
	lines = append(lines, "Live_OverlayResolves == (st \\in Overlay) ~> (st \\in Domain)")
	lines = append(lines, "====")
	tlaOut := strings.Join(lines, "\n") + "\n"

	cfgOut := "CONSTANT MaxRetries = 3\nSPECIFICATION Spec\nINVARIANT TypeOK\nPROPERTY Live_OverlayResolves\n"
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
	mid, tla, cfg, err := Generate(path)
	if err != nil {
		return err
	}
	if outdir == "" {
		outdir = filepath.Dir(path)
	}
	if mkErr := os.MkdirAll(outdir, 0755); mkErr != nil {
		return mkErr
	}
	if wErr := os.WriteFile(filepath.Join(outdir, mid+".tla"), []byte(tla), 0644); wErr != nil {
		return wErr
	}
	if wErr := os.WriteFile(filepath.Join(outdir, mid+".cfg"), []byte(cfg), 0644); wErr != nil {
		return wErr
	}
	fmt.Fprintf(os.Stdout, "wrote %s.tla and %s.cfg to %s\n", mid, mid, outdir)
	return nil
}
