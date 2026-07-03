// Package compose is the Go port of compose_gen.py: generates a composition
// spec validated against the coordinator machine, modeling full branching.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// ExitError carries a hard-error (maps to Python sys.exit).
type ExitError struct{ Msg string }

func (e *ExitError) Error() string { return e.Msg }

func die(format string, args ...interface{}) {
	panic(&ExitError{Msg: "compose_gen: VALIDATION FAILED: " + fmt.Sprintf(format, args...)})
}

// ForwardChainResult is the coordinator's forward path + per-step failures.
type ForwardChainResult struct {
	Chain    []string
	Fails    map[string]map[string]bool
	Terminal string
}

// ForwardChain mirrors compose_gen.forward_chain.
func ForwardChain(machine *ir.Value) ForwardChainResult {
	top := map[string]*ir.Value{}
	for _, s := range ir.WalkStates(machine.AsObject().Get2("states"), "") {
		if !strings.Contains(s.Path, ".") {
			top[s.Name] = s.Node
		}
	}
	var chain []string
	fails := map[string]map[string]bool{}
	cur := machine.AsObject().GetString("initial")
	seen := map[string]bool{}
	for {
		node, ok := top[cur]
		if !ok || node.AsObject().GetString("type") == "final" {
			break
		}
		if seen[cur] {
			die("forward chain loops at %s", ir.Repr(cur))
		}
		seen[cur] = true
		dones := map[string]bool{}
		errs := map[string]bool{}
		for _, tr := range ir.TransitionsOf(node, nil, "") {
			if tr.Kind == "onDone" && tr.Target != "" {
				dones[ir.Simple(tr.Target)] = true
			}
			if (tr.Kind == "onError" || tr.Kind == "after") && tr.Target != "" {
				errs[ir.Simple(tr.Target)] = true
			}
		}
		if len(dones) != 1 {
			break
		}
		chain = append(chain, cur)
		fails[cur] = errs
		cur = firstKey(dones)
	}
	return ForwardChainResult{Chain: chain, Fails: fails, Terminal: cur}
}

func firstKey(m map[string]bool) string {
	for k := range m {
		return k
	}
	return ""
}

type SeqEntry struct {
	Step      string
	Aggregate string
	To        string
	Undo      *UndoSpec
}

type UndoSpec struct {
	To string
}

// Generate mirrors compose_gen.generate(comp, machine, machine_name).
func Generate(comp, machine *ir.Value, machineName string) (name, tla, cfg string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
			} else {
				panic(r)
			}
		}
	}()
	name, tla, cfg = generateImpl(comp, machine, machineName)
	return
}

func generateImpl(comp, machine *ir.Value, machineName string) (string, string, string) {
	co := comp.AsObject()
	name := ir.Title(co.GetString("composition"))
	var tla, cfg string
	aggsVal := co.Get2("aggregates").AsObject()
	seqVal := co.Get2("sequence").AsArray()
	invsVal := co.Get2("invariants")
	if invsVal == nil {
		invsVal = ir.ObjectValue(ir.NewObject())
	}
	invs := invsVal.AsObject()
	aggnames := aggsVal.Keys()

	// validate against the coordinator machine
	fc := ForwardChain(machine)
	chain := fc.Chain
	fails := fc.Fails
	terminal := fc.Terminal
	if len(chain) == 0 {
		die("coordinator %s has no forward chain (no invoking step states); nothing to compose", machineName)
	}

	var declared []string
	var seq []SeqEntry
	for _, s := range seqVal {
		so := s.AsObject()
		step := so.GetString("step")
		declared = append(declared, step)
		e := SeqEntry{Step: step, Aggregate: so.GetString("aggregate"), To: so.GetString("to")}
		if u := so.Get2("undo"); u != nil && u.Kind == ir.KindObject {
			e.Undo = &UndoSpec{To: u.AsObject().GetString("to")}
		}
		seq = append(seq, e)
	}
	for _, d := range declared {
		if d == "" {
			die("every sequence entry needs step: <coordinator state>")
		}
	}
	if !sliceEq(declared, chain) {
		die("sequence steps %s do not match the coordinator's forward chain %s (from %s)",
			bracketStr(declared), bracketStr(chain), machineName)
	}
	for i, s := range seq {
		var expectedFail map[string]bool
		if i == 0 {
			expectedFail = map[string]bool{"Failed": true}
		} else {
			expectedFail = map[string]bool{"Compensating": true}
		}
		if !setEq(fails[s.Step], expectedFail) {
			die("step %s failure paths in the machine go to %s, expected %s",
				ir.Repr(s.Step), brSorted(fails[s.Step]), brSorted(expectedFail))
		}
		agg := aggsVal.Get2(s.Aggregate)
		if agg == nil {
			die("step %s names unknown aggregate %s", ir.Repr(s.Step), ir.Repr(s.Aggregate))
		}
		aggStates := setOfStates(agg.AsObject().Get2("states"))
		if !aggStates[s.To] {
			die("step %s commits %s to unknown state %s", ir.Repr(s.Step), ir.Repr(s.Aggregate), ir.Repr(s.To))
		}
		if i < len(seq)-1 && s.Undo == nil {
			die("step %s needs an undo: (its compensating obligation); only the completing step may omit it", ir.Repr(s.Step))
		}
		if s.Undo != nil && !aggStates[s.Undo.To] {
			die("step %s undo names unknown state %s", ir.Repr(s.Step), ir.Repr(s.Undo.To))
		}
	}
	// two steps compensating the same aggregate would emit duplicate Undo_<agg>
	// definitions (a TLA parse error at best, a wrong model at worst)
	undoAgg := map[string]string{}
	for _, s := range seq {
		if s.Undo == nil {
			continue
		}
		if prev, ok := undoAgg[s.Aggregate]; ok {
			die("steps %s and %s both declare an undo for aggregate %s; each aggregate may be compensated by exactly one step", ir.Repr(prev), ir.Repr(s.Step), ir.Repr(s.Aggregate))
		}
		undoAgg[s.Aggregate] = s.Step
	}

	top := map[string]*ir.Value{}
	for _, st := range ir.WalkStates(machine.AsObject().Get2("states"), "") {
		if !strings.Contains(st.Path, ".") {
			top[st.Name] = st.Node
		}
	}
	for _, needed := range []string{"Compensating", "Completed", "Failed", "FailedDirty"} {
		if _, ok := top[needed]; !ok {
			die("coordinator has no %s state; the composition template expects the saga pattern", ir.Repr(needed))
		}
	}

	sagaStates := append(append([]string{}, chain...), "Compensating", "Completed", "Failed", "FailedDirty")
	var obligations [][3]string // (aggregate, to, undoTo)
	for _, s := range seq {
		if s.Undo != nil {
			obligations = append(obligations, [3]string{s.Aggregate, s.To, s.Undo.To})
		}
	}

	var L []string
	L = append(L, fmt.Sprintf("---- MODULE %s ----", name))
	L = append(L, fmt.Sprintf(`\* GENERATED by machinery compose from %s.composition.yaml,`, co.GetString("composition")))
	L = append(L, fmt.Sprintf(`\* VALIDATED against %s: the step order below IS the coordinator's`, machineName))
	L = append(L, `\* forward onDone chain, and every failure route matches the machine.`)
	L = append(L, `\* Models the full branching: step failures, per-obligation compensation`)
	L = append(L, `\* in any order, and the FailedDirty stall with obligations still held.`)
	L = append(L, `\* Residual assumption: each aggregate conforms to its abstract states,`)
	L = append(L, `\* discharged per aggregate by its own machine, oracle, and tests.`)
	L = append(L, "")
	for _, a := range aggnames {
		vals := strSliceStates(aggsVal.Get2(a).AsObject().Get2("states"))
		quoted := make([]string, len(vals))
		for i, v := range vals {
			quoted[i] = fmt.Sprintf(`"%s"`, v)
		}
		L = append(L, fmt.Sprintf("%sStates == {%s}", ir.Title(a), strings.Join(quoted, ", ")))
	}
	quotedSaga := make([]string, len(sagaStates))
	for i, v := range sagaStates {
		quotedSaga[i] = fmt.Sprintf(`"%s"`, v)
	}
	L = append(L, fmt.Sprintf("SagaStates == {%s}", strings.Join(quotedSaga, ", ")))
	varlist := "saga, " + strings.Join(aggnames, ", ")
	L = append(L, fmt.Sprintf("VARIABLES %s", varlist))
	L = append(L, fmt.Sprintf("vars == << %s >>", varlist))
	L = append(L, "")
	typeok := `TypeOK == saga \in SagaStates`
	for _, a := range aggnames {
		typeok += fmt.Sprintf(` /\ %s \in %sStates`, a, ir.Title(a))
	}
	L = append(L, typeok)
	init := fmt.Sprintf(`Init == saga = "%s"`, chain[0])
	for _, a := range aggnames {
		init += fmt.Sprintf(` /\ %s = "%s"`, a, aggsVal.Get2(a).AsObject().GetString("initial"))
	}
	L = append(L, init)
	L = append(L, "")

	unch := func(exclude []string) string {
		exSet := map[string]bool{}
		for _, e := range exclude {
			exSet[e] = true
		}
		var keep []string
		for _, a := range aggnames {
			if !exSet[a] {
				keep = append(keep, a)
			}
		}
		if len(keep) == 0 {
			return "TRUE"
		}
		return "UNCHANGED << " + strings.Join(keep, ", ") + " >>"
	}

	var acts []string
	for i, s := range seq {
		var nxt string
		if i+1 < len(chain) {
			nxt = chain[i+1]
		} else {
			nxt = terminal
		}
		L = append(L, fmt.Sprintf(`Done_%s == saga = "%s" /\ saga' = "%s" /\ %s' = "%s" /\ %s`,
			s.Step, s.Step, nxt, s.Aggregate, s.To, unch([]string{s.Aggregate})))
		var failTo string
		if i == 0 {
			failTo = "Failed"
		} else {
			failTo = "Compensating"
		}
		L = append(L, fmt.Sprintf(`Fail_%s == saga = "%s" /\ saga' = "%s" /\ %s`,
			s.Step, s.Step, failTo, unch(nil)))
		acts = append(acts, "Done_"+s.Step, "Fail_"+s.Step)
	}
	for _, ob := range obligations {
		a, to, undoTo := ob[0], ob[1], ob[2]
		L = append(L, fmt.Sprintf(`Undo_%s == saga = "Compensating" /\ %s = "%s" /\ %s' = "%s" /\ saga' = saga /\ %s`,
			a, a, to, a, undoTo, unch([]string{a})))
		acts = append(acts, "Undo_"+a)
	}
	var cleanParts, dirtyParts []string
	for _, ob := range obligations {
		a, to := ob[0], ob[1]
		cleanParts = append(cleanParts, fmt.Sprintf(`%s # "%s"`, a, to))
		dirtyParts = append(dirtyParts, fmt.Sprintf(`%s = "%s"`, a, to))
	}
	clean := strings.Join(cleanParts, " /\\ ")
	dirty := strings.Join(dirtyParts, " \\/ ")
	L = append(L, fmt.Sprintf(`CompensateDone == saga = "Compensating" /\ (%s) /\ saga' = "Failed" /\ %s`, clean, unch(nil)))
	L = append(L, fmt.Sprintf(`CompensateStall == saga = "Compensating" /\ (%s) /\ saga' = "FailedDirty" /\ %s`, dirty, unch(nil)))
	acts = append(acts, "CompensateDone", "CompensateStall")
	L = append(L, `Done == saga \in {"Completed", "Failed", "FailedDirty"} /\ UNCHANGED vars`)
	acts = append(acts, "Done")
	L = append(L, "Next == "+strings.Join(acts, " \\/ "))
	L = append(L, "Spec == Init /\\ [][Next]_vars /\\ WF_vars(Next)")
	L = append(L, "")

	cn := func(iname string) string {
		var parts []string
		for _, w := range strings.Split(iname, "-") {
			parts = append(parts, capitalize(w))
		}
		return "Inv_" + strings.Join(parts, "")
	}
	L = append(L, `\* auto-generated: a clean Failed end has undone every committed obligation;`)
	L = append(L, `\* only the explicit FailedDirty residual may still hold one`)
	L = append(L, fmt.Sprintf(`Inv_CleanCompensation == (saga = "Failed") => (%s)`, clean))
	for _, iname := range invs.Keys() {
		expr := invs.Get2(iname).AsString()
		L = append(L, fmt.Sprintf("%s == %s", cn(iname), expr))
	}
	L = append(L, `Live_Terminates == TRUE ~> (saga \in {"Completed", "Failed", "FailedDirty"})`)
	L = append(L, "====")
	tla = strings.Join(L, "\n") + "\n"

	cfgParts := []string{"SPECIFICATION Spec", "INVARIANT TypeOK", "INVARIANT Inv_CleanCompensation"}
	for _, iname := range invs.Keys() {
		cfgParts = append(cfgParts, "INVARIANT "+cn(iname))
	}
	cfgParts = append(cfgParts, "PROPERTY Live_Terminates")
	cfg = strings.Join(cfgParts, "\n") + "\n"

	fmt.Fprintf(os.Stdout, "compose_gen: validated %s against %s: %d forward steps, %d obligations, %d declared invariants\n",
		name, machineName, len(chain), len(obligations), len(invs.Keys()))
	return name, tla, cfg
}

// capitalize mirrors Python str.capitalize: first rune upper, rest lower.
func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	for i := 1; i < len(r); i++ {
		r[i] = []rune(strings.ToLower(string(r[i])))[0]
	}
	return string(r)
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

func sliceEq(a, b []string) bool {
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

func setOfStates(v *ir.Value) map[string]bool {
	out := map[string]bool{}
	if v == nil {
		return out
	}
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			out[e.AsString()] = true
		}
	}
	return out
}

func strSliceStates(v *ir.Value) []string {
	var out []string
	if v == nil {
		return out
	}
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			out = append(out, e.AsString())
		}
	}
	return out
}

func brSorted(m map[string]bool) string {
	var xs []string
	for x := range m {
		xs = append(xs, x)
	}
	sort.Strings(xs)
	return bracketStr(xs)
}

func bracketStr(xs []string) string {
	quoted := make([]string, len(xs))
	for i, x := range xs {
		quoted[i] = fmt.Sprintf(`"%s"`, x)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

// Run is the `machinery compose <composition.yaml> <coordinator.machine.json> [out-dir]` entrypoint.
func Run(compPath, machinePath, outdir string) error {
	data, err := os.ReadFile(compPath)
	if err != nil {
		return &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	comp, err := ir.LoadYAML(data)
	if err != nil {
		return &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	if comp.Kind != ir.KindObject {
		return &ExitError{Msg: "compose_gen: composition file is not a mapping"}
	}
	machine, err := ir.LoadMachineJSON(machinePath)
	if err != nil {
		return &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	name, tla, cfg, genErr := Generate(comp, machine, filepath.Base(machinePath))
	if genErr != nil {
		return genErr
	}
	if outdir == "" {
		outdir = filepath.Dir(compPath)
	}
	if mkErr := os.MkdirAll(outdir, 0755); mkErr != nil {
		return mkErr
	}
	if wErr := os.WriteFile(filepath.Join(outdir, name+".tla"), []byte(tla), 0644); wErr != nil {
		return wErr
	}
	if wErr := os.WriteFile(filepath.Join(outdir, name+".cfg"), []byte(cfg), 0644); wErr != nil {
		return wErr
	}
	fmt.Fprintf(os.Stdout, "generated %s.tla + %s.cfg\n", name, name)
	return nil
}
