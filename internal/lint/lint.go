// Package lint is the Go port of machine_lint.py: structural lint over the
// finite state graph, JSON<->matrix reconciliation, and the CLI entrypoint.
package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ramirosalas/machinery/internal/ir"
)

// RootKeys / StateKeys / InvokeKeys / StateTypes mirror machine_lint.py exactly.
var RootKeys = map[string]bool{
	"id": true, "initial": true, "context": true, "states": true, "description": true,
	"meta": true, "version": true, "_comment": true, "_delays": true,
	"_lifecycle_of": true, "_role": true, "_component": true,
}

var StateKeys = map[string]bool{
	"on": true, "after": true, "always": true, "invoke": true, "entry": true, "exit": true,
	"states": true, "initial": true, "type": true, "id": true, "meta": true,
	"description": true, "tags": true, "onDone": true, "output": true, "_comment": true,
	"_exhaustive": true, "_ignores": true,
}

var InvokeKeys = map[string]bool{
	"src": true, "input": true, "id": true, "onDone": true, "onError": true, "_comment": true,
}

var StateTypes = map[string]bool{"": true, "atomic": true, "compound": true, "final": true}

// Result is the output of LintMachine.
type Result struct {
	Errs   []string
	Warns  []string
	Notes  []string
	Counts Counts
}

type Counts struct {
	States      int
	Transitions int
}

// LintMachine mirrors machine_lint.lint_machine(m, base).
func LintMachine(m *ir.Value, base string) (errs, warns, notes []string, counts Counts) {
	if m == nil || m.Kind != ir.KindObject {
		return []string{base + ": machine is not an object"}, nil, nil, Counts{}
	}
	ro := m.AsObject()

	for _, k := range ro.Keys() {
		if !RootKeys[k] {
			sortedRoots := sortedKeysMap(RootKeys)
			errs = append(errs, fmt.Sprintf("%s: unsupported root key %s (supported: %s)",
				base, ir.Repr(k), strings.Join(sortedRoots, ", ")))
		}
	}
	statesVal := ro.Get2("states")
	if statesVal == nil || statesVal.Kind != ir.KindObject || statesVal.AsObject().Len() == 0 {
		errs = append(errs, base+": machine has no states")
		return errs, warns, notes, counts
	}

	states := ir.WalkStates(statesVal, "")
	counts.States = len(states)
	pathSet := map[string]bool{}
	bySimple := map[string][]string{}
	nodeOf := map[string]*ir.Value{}
	for _, s := range states {
		pathSet[s.Path] = true
		bySimple[s.Name] = append(bySimple[s.Name], s.Path)
		nodeOf[s.Path] = s.Node
	}

	resolve := func(tgt, srcPath string) (string, string) {
		if tgt == "" {
			return srcPath, "" // internal (self) transition (Python: tgt is None)
		}
		t := strings.TrimLeft(tgt, "#")
		if pathSet[t] {
			return t, ""
		}
		// simple-name lookup: last segment of t
		simple := t
		if i := strings.LastIndex(t, "."); i >= 0 {
			simple = t[i+1:]
		}
		cands := bySimple[simple]
		if len(cands) == 1 {
			return cands[0], ""
		}
		if len(cands) > 1 {
			sortedC := append([]string{}, cands...)
			sort.Strings(sortedC)
			return "", fmt.Sprintf("ambiguous target %s (candidates: %s)", ir.Repr(tgt), strings.Join(sortedC, ", "))
		}
		return "", fmt.Sprintf("dangling target %s", ir.Repr(tgt))
	}

	var problems []string
	for _, s := range states {
		p, node := s.Path, s.Node
		if node == nil || node.Kind != ir.KindObject {
			errs = append(errs, base+": state "+p+" is not an object")
			continue
		}
		o := node.AsObject()
		for _, k := range o.Keys() {
			if !StateKeys[k] {
				errs = append(errs, fmt.Sprintf("%s: unsupported key %s in state %s", base, ir.Repr(k), p))
			}
		}
		stype := o.GetString("type")
		if !StateTypes[stype] {
			errs = append(errs, fmt.Sprintf("%s: unsupported state type %s in %s (parallel/history are not in the supported subset)",
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
		if !isFinal && o.Get2("states") == nil && len(trs) == 0 {
			errs = append(errs, base+": dead-end non-final leaf state "+p)
		}
		for _, iv := range ir.InvokesOf(node) {
			ivObj := iv.AsObject()
			for _, k := range ivObj.Keys() {
				if !InvokeKeys[k] {
					errs = append(errs, fmt.Sprintf("%s: unsupported invoke key %s in state %s", base, ir.Repr(k), p))
				}
			}
			if ivObj.Get2("onError") == nil {
				src := srcRepr(ivObj)
				errs = append(errs, fmt.Sprintf("%s: invoke %s in %s has no onError", base, src, p))
			}
		}
		if o.Get2("invoke") != nil && o.Get2("after") == nil {
			errs = append(errs, base+": invoking state "+p+" has no after/timeout")
		}
		ir.ActionNames(o.Get2("entry"), &problems, p+" entry")
		ir.ActionNames(o.Get2("exit"), &problems, p+" exit")

		// branch shadowing
		checkShadow := func(label string, t *ir.Value) {
			branches := normBranches(t)
			for i := 0; i < len(branches)-1; i++ {
				if !branches[i].HasGuard {
					errs = append(errs, fmt.Sprintf("%s: state %s %s branch %d is unguarded but not last; later branches are unreachable",
						base, p, label, i+1))
				}
			}
		}
		if on := o.Get2("on"); on != nil {
			for _, ev := range on.AsObject().Keys() {
				checkShadow("on:"+ev, on.AsObject().Get2(ev))
			}
		}
		if after := o.Get2("after"); after != nil {
			for _, delay := range after.AsObject().Keys() {
				checkShadow("after:"+delay, after.AsObject().Get2(delay))
			}
		}
		if od := o.Get2("onDone"); od != nil {
			checkShadow("onDone", od)
		}
		for _, iv := range ir.InvokesOf(node) {
			ivObj := iv.AsObject()
			for _, key := range []string{"onDone", "onError"} {
				if ivObj.Get2(key) != nil {
					checkShadow("invoke."+key, ivObj.Get2(key))
				}
			}
		}
		if always := o.Get2("always"); always != nil {
			checkShadow("always", always)
			branches := normBranches(always)
			fullyGuarded := len(branches) > 0
			for _, b := range branches {
				if !b.HasGuard {
					fullyGuarded = false
				}
			}
			hasEscape := o.Get2("after") != nil || o.Get2("on") != nil || o.Get2("invoke") != nil
			if fullyGuarded && !hasEscape {
				just := o.GetString("_exhaustive")
				if strings.TrimSpace(just) != "" {
					notes = append(notes, base+": state "+p+" always-list is fully guarded; liveness rests on the declared exhaustiveness: "+strings.TrimSpace(just))
				} else {
					errs = append(errs, base+": state "+p+" has a fully guarded always-list and no unguarded escape (after/on/invoke); if no guard is true the machine is stuck. Add an unguarded fallback branch, or an _exhaustive note stating why the guards are total")
				}
			}
		}

		for _, tr := range trs {
			_, why := resolve(tr.Target, p)
			if why != "" {
				errs = append(errs, base+": "+why+" from "+p+" ("+tr.Kind+":"+tr.Event+")")
			}
		}
	}

	for _, pr := range problems {
		errs = append(errs, base+": "+pr)
	}

	// event completeness
	allEvents := map[string]bool{}
	for _, s := range states {
		if s.Node == nil || s.Node.Kind != ir.KindObject {
			continue
		}
		if on := s.Node.AsObject().Get2("on"); on != nil {
			for _, ev := range on.AsObject().Keys() {
				allEvents[ev] = true
			}
		}
	}
	sortedAllEvents := sortedSetBool(allEvents)
	for _, s := range states {
		p, n, node := s.Path, s.Name, s.Node
		if strings.Contains(p, ".") || node == nil || node.Kind != ir.KindObject {
			continue
		}
		o := node.AsObject()
		if !ir.IsUpperFirst(n) || o.GetString("type") == "final" || o.Get2("states") != nil {
			continue
		}
		if o.Get2("invoke") != nil || o.Get2("always") != nil {
			continue
		}
		ignoresVal := o.Get2("_ignores")
		ignores := map[string]string{}
		ignoresValid := true
		if ignoresVal != nil && ignoresVal.Kind == ir.KindObject {
			io := ignoresVal.AsObject()
			for _, k := range io.Keys() {
				v := io.Get2(k)
				if v == nil || v.Kind != ir.KindString || strings.TrimSpace(v.AsString()) == "" {
					ignoresValid = false
				} else {
					ignores[k] = v.AsString()
				}
			}
		} else if ignoresVal != nil {
			ignoresValid = false // present but not an object
		}
		if !ignoresValid {
			errs = append(errs, base+": state "+p+" _ignores must map event names to reason strings")
			ignores = map[string]string{} // Python: ignores = {} when invalid
		}
		// both handles and ignores
		var onEvents []string
		if on := o.Get2("on"); on != nil {
			onEvents = on.AsObject().Keys()
		}
		onEventSet := map[string]bool{}
		for _, e := range onEvents {
			onEventSet[e] = true
		}
		sortedIgnores := sortedKeysStr(ignores)
		for _, ev := range sortedIgnores {
			if onEventSet[ev] {
				errs = append(errs, fmt.Sprintf("%s: state %s both handles and ignores event %s", base, p, ir.Repr(ev)))
			}
		}
		for _, ev := range sortedAllEvents {
			if !onEventSet[ev] && ignores[ev] == "" {
				errs = append(errs, fmt.Sprintf("%s: state %s neither handles nor explicitly ignores event %s (add it to on: or to _ignores: with a reason)",
					base, p, ir.Repr(ev)))
			}
		}
	}

	// initial + reachability
	init := ro.GetString("initial")
	statesRoot := ro.Get2("states").AsObject()
	if !statesRoot.Has(init) {
		errs = append(errs, fmt.Sprintf("%s: initial %s is not a top-level state", base, ir.Repr(init)))
	} else {
		reached := map[string]bool{}
		var enter func(p string)
		enter = func(p string) {
			if reached[p] {
				return
			}
			reached[p] = true
			node := nodeOf[p]
			if node != nil && node.Kind == ir.KindObject {
				o := node.AsObject()
				if o.Get2("states") != nil {
					childInit := o.GetString("initial")
					if childInit != "" {
						child := p + "." + childInit
						if pathSet[child] {
							enter(child)
						}
					}
				}
			}
		}
		enter(init)
		frontier := true
		for frontier {
			frontier = false
			active := map[string]bool{}
			for p := range reached {
				active[p] = true
				q := p
				for strings.Contains(q, ".") {
					q = q[:strings.LastIndex(q, ".")]
					active[q] = true
				}
			}
			for p := range active {
				for _, tr := range ir.TransitionsOf(nodeOf[p], nil, p) {
					dest, why := resolve(tr.Target, p)
					_ = why
					if dest != "" && !reached[dest] {
						enter(dest)
						frontier = true
					}
				}
			}
		}
		for _, s := range states {
			p := s.Path
			if reached[p] {
				continue
			}
			// not reachable directly; check if reachable via containment overlap
			hidden := false
			for r := range reached {
				if strings.HasPrefix(p, r+".") || strings.HasPrefix(r, p+".") {
					hidden = true
					break
				}
			}
			if !hidden {
				errs = append(errs, base+": unreachable state "+p)
			}
		}
	}

	return errs, warns, notes, counts
}

func srcRepr(ivObj *ir.Object) string {
	if s := ivObj.Get2("src"); s != nil {
		return ir.Repr(s.AsString())
	}
	return ir.Repr(nil)
}

type normBranchRec struct {
	HasGuard bool
	Target   string
	HasTgt   bool
}

func normBranches(t *ir.Value) []normBranchRec {
	var items []*ir.Value
	if t == nil {
		return nil
	}
	if t.Kind == ir.KindArray {
		items = t.AsArray()
	} else {
		items = []*ir.Value{t}
	}
	var out []normBranchRec
	for _, it := range items {
		if it == nil || it.Kind != ir.KindObject {
			out = append(out, normBranchRec{})
			continue
		}
		o := it.AsObject()
		b := normBranchRec{}
		if gv := o.Get2("guard"); gv != nil && gv.Kind == ir.KindString {
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

func sortedKeysMap(m map[string]bool) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedKeysStr(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedSetBool(m map[string]bool) []string { return sortedKeysMap(m) }

// --- matrix reconciliation ---

// MRow is a canonical machine transition row.
type MRow struct {
	Source     string
	Trigger    string
	Guard      string
	HasGuard   bool
	Fallback   bool
	FirstGuard string
	Target     string
	Actions    []string
}

// XRow is a canonical matrix transition row.
type XRow struct {
	Source  string
	Trigger string
	Guard   string
	Target  string
	Actions []string
}

// MachineTransitionRows mirrors machine_lint.machine_transition_rows.
func MachineTransitionRows(m *ir.Value) []MRow {
	states := ir.WalkStates(m.AsObject().Get2("states"), "")
	var rows []MRow
	for _, s := range states {
		groups := map[string][]ir.Transition{}
		var trigOrder []string
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
			var trig string
			switch tr.Kind {
			case "on":
				trig = tr.Event
			case "after":
				trig = "after " + tr.Event
			case "always":
				trig = "always"
			case "stateDone":
				trig = "onDone"
			case "onDone":
				trig = "invoke onDone"
			case "onError":
				trig = "invoke onError"
			}
			if _, ok := groups[trig]; !ok {
				trigOrder = append(trigOrder, trig)
			}
			groups[trig] = append(groups[trig], tr)
		}
		for _, trig := range trigOrder {
			trs := groups[trig]
			for i, tr := range trs {
				var tgt string
				if tr.Target != "" {
					tgt = ir.Simple(tr.Target)
				} else {
					tgt = "(internal)"
				}
				row := MRow{
					Source:     s.Name,
					Trigger:    trig,
					Guard:      tr.Guard,
					HasGuard:   tr.HasGuard,
					Fallback:   !tr.HasGuard && i > 0,
					FirstGuard: firstGuard(trs),
					Target:     tgt,
					Actions:    append([]string{}, tr.Actions...),
				}
				rows = append(rows, row)
			}
		}
	}
	return rows
}

func firstGuard(trs []ir.Transition) string {
	if len(trs) == 0 {
		return ""
	}
	return trs[0].Guard
}

// MatrixTransitionRows mirrors machine_lint.matrix_transition_rows.
func MatrixTransitionRows(text string) ([]XRow, string) {
	for _, tbl := range ir.ParseMdTables(text) {
		joined := strings.ToLower(strings.Join(tbl.Header, " "))
		if strings.Contains(joined, "source") && strings.Contains(joined, "target") && strings.Contains(joined, "actions") {
			si := ir.FindCol(tbl.Header, "source")
			ei := ir.FindCol(tbl.Header, "event", "trigger")
			gi := ir.FindCol(tbl.Header, "guard")
			ti := ir.FindCol(tbl.Header, "target")
			ai := ir.FindCol(tbl.Header, "actions")
			if si < 0 || ei < 0 || gi < 0 || ti < 0 || ai < 0 {
				return nil, "transition table found but a required column is missing (need source, event/trigger, guard, target, actions)"
			}
			var out []XRow
			for _, r := range tbl.Rows {
				maxI := si
				for _, idx := range []int{ei, gi, ti, ai} {
					if idx > maxI {
						maxI = idx
					}
				}
				if len(r) <= maxI {
					return nil, "transition table row has too few cells: " + ir.Repr(strings.Join(r, "|"))
				}
				if strings.Contains(r[ei], "(final)") || strings.Contains(r[ti], "(final)") || strings.Contains(r[ei], "(any event)") {
					continue
				}
				src := ir.CleanCell(r[si])
				trig := ir.CleanCell(r[ei])
				guard := ir.CleanCell(r[gi])
				tgtRaw := r[ti]
				var tgt string
				if strings.Contains(tgtRaw, "(internal)") {
					tgt = "(internal)"
				} else {
					tgt = ir.CleanCell(tgtRaw)
				}
				acts := ir.CleanCell(r[ai])
				var actions []string
				if acts != "" && acts != "-" {
					for _, a := range strings.Split(acts, ",") {
						a = strings.TrimSpace(a)
						if a != "" {
							actions = append(actions, a)
						}
					}
				}
				out = append(out, XRow{Source: src, Trigger: trig, Guard: guard, Target: tgt, Actions: actions})
			}
			return out, ""
		}
	}
	return nil, ""
}

func guardMatches(mr MRow, cell string) bool {
	if mr.HasGuard {
		return cell == mr.Guard
	}
	if mr.Fallback {
		accepted := map[string]bool{"-": true, "(else)": true, "else": true, "": true}
		if mr.FirstGuard != "" {
			accepted["!"+mr.FirstGuard] = true
		}
		return accepted[cell]
	}
	return cell == "-" || cell == ""
}

// ReconcileMatrix mirrors machine_lint.reconcile_matrix.
func ReconcileMatrix(m *ir.Value, matrixText, base string) ([]string, int) {
	var drift []string
	mrows := MachineTransitionRows(m)
	xrows, err := MatrixTransitionRows(matrixText)
	if err != "" {
		return []string{base + ": " + err}, 0
	}
	if xrows == nil {
		return nil, 0
	}
	unmatched := make([]int, len(xrows))
	for i := range unmatched {
		unmatched[i] = i
	}
	trigEq := func(machineTrig, cell string) bool {
		return cell == machineTrig || cell == "on:"+machineTrig
	}
	take := func(pred func(x XRow) bool) (int, bool) {
		for _, k := range unmatched {
			if pred(xrows[k]) {
				return k, true
			}
		}
		return -1, false
	}
	for _, mr := range mrows {
		mrLocal := mr
		idx, ok := take(func(x XRow) bool {
			return x.Source == mrLocal.Source && trigEq(mrLocal.Trigger, x.Trigger) &&
				guardMatches(mrLocal, x.Guard) && x.Target == mrLocal.Target &&
				eqActions(x.Actions, mrLocal.Actions)
		})
		if !ok {
			g := mr.Guard
			if !mr.HasGuard {
				if mr.Fallback {
					g = "else"
				} else {
					g = "-"
				}
			}
			actsStr := "-"
			if len(mr.Actions) > 0 {
				actsStr = strings.Join(mr.Actions, ", ")
			}
			drift = append(drift, fmt.Sprintf("%s: machine transition has no matrix row: %s --%s [%s]--> %s / %s",
				base, mr.Source, mr.Trigger, g, mr.Target, actsStr))
			continue
		}
		// remove idx from unmatched
		for i, k := range unmatched {
			if k == idx {
				unmatched = append(unmatched[:i], unmatched[i+1:]...)
				break
			}
		}
	}
	for _, k := range unmatched {
		x := xrows[k]
		g := x.Guard
		if g == "" {
			g = "-"
		}
		actsStr := "-"
		if len(x.Actions) > 0 {
			actsStr = strings.Join(x.Actions, ", ")
		}
		drift = append(drift, fmt.Sprintf("%s: matrix row has no machine transition: %s --%s [%s]--> %s / %s",
			base, x.Source, x.Trigger, g, x.Target, actsStr))
	}
	return drift, len(mrows)
}

func eqActions(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// NamedUnitNames mirrors machine_lint.namedunit_names.
func NamedUnitNames(matrixText string) map[string]bool {
	names := map[string]bool{}
	for _, tbl := range ir.ParseMdTables(matrixText) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if strings.Contains(hl, "signature") || (strings.Contains(hl, "name") && strings.Contains(hl, "kind")) {
			ni := ir.FindCol(tbl.Header, "name")
			if ni < 0 {
				ni = 0
			}
			for _, r := range tbl.Rows {
				cell := ""
				if ni < len(r) {
					cell = r[ni]
				}
				for _, m := range regexpBacktickGroup.FindAllStringSubmatch(cell, -1) {
					names[m[1]] = true
				}
				if !strings.Contains(cell, "`") {
					for _, m := range regexpWord.FindAllStringSubmatch(cell, -1) {
						names[m[1]] = true
					}
				}
			}
		}
	}
	return names
}

// MachineUnitNames mirrors machine_lint.machine_unit_names.
func MachineUnitNames(m *ir.Value) (guards, actions, actors map[string]bool) {
	guards, actions, actors = map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Name) {
			if tr.HasGuard {
				guards[tr.Guard] = true
			}
			for _, a := range tr.Actions {
				actions[a] = true
			}
		}
		for _, n := range ir.ActionNames(entryExit(s.Node, "entry"), nil, "") {
			actions[n] = true
		}
		for _, n := range ir.ActionNames(entryExit(s.Node, "exit"), nil, "") {
			actions[n] = true
		}
		for _, iv := range ir.InvokesOf(s.Node) {
			if src := iv.AsObject().Get2("src"); src != nil && src.Kind == ir.KindString {
				actors[src.AsString()] = true
			}
		}
	}
	return
}

func entryExit(node *ir.Value, key string) *ir.Value {
	if node == nil || node.Kind != ir.KindObject {
		return nil
	}
	return node.AsObject().Get2(key)
}

// Lint mirrors machine_lint.lint(path): loads machine + matrix, returns summary.
func Lint(path string) (nStates int, errs, warns, drift []string) {
	m, err := ir.LoadMachineJSON(path)
	base := filepath.Base(path)
	if err != nil {
		return 0, []string{err.Error()}, nil, nil
	}
	e, w, notes, counts := LintMachine(m, base)
	mx := path
	mx = strings.TrimSuffix(mx, filepath.Ext(mx)) // remove .json
	mx = strings.TrimSuffix(mx, ".machine") + ".matrix.md"
	if _, statErr := os.Stat(mx); statErr == nil {
		data, _ := os.ReadFile(mx)
		d, _ := ReconcileMatrix(m, string(data), base)
		drift = d
	} else {
		w = append(w, base+": no matrix file; named-unit contracts are unchecked (the generated oracle still covers transitions)")
	}
	return counts.States, e, append(w, notes...), drift
}

// Run is the `machinery lint <dir>` entrypoint.
func Run(mdir string, out, errw *os.File) int {
	entries, _ := os.ReadDir(mdir)
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".machine.json") {
			files = append(files, filepath.Join(mdir, e.Name()))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		fmt.Fprintf(out, "ERROR  no *.machine.json under %s: nothing to lint is a failure, not a pass\n", mdir)
		return 1
	}
	total := 0
	for _, f := range files {
		n, errs, warns, drift := Lint(f)
		fmt.Fprintf(out, "== %s: %d states ==\n", filepath.Base(f), n)
		for _, e := range errs {
			fmt.Fprintf(out, "  ERROR  %s\n", e)
		}
		for _, d := range drift {
			fmt.Fprintf(out, "  DRIFT  %s\n", d)
		}
		for _, w := range warns {
			fmt.Fprintf(out, "  warn   %s\n", w)
		}
		if len(errs) == 0 && len(drift) == 0 && len(warns) == 0 {
			fmt.Fprintln(out, "  ok")
		}
		total += len(errs) + len(drift)
	}
	fmt.Fprintf(out, "\n%d error/drift finding(s) across %d machine(s)\n", total, len(files))
	if total > 0 {
		return 1
	}
	return 0
}
