// Package compose is the Go port of compose_gen.py: generates a composition
// spec validated against the coordinator machine, modeling full branching.
package compose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
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
			// a targeted route from any other trigger (on:, always, state-level
			// onDone) is outside the composition vocabulary; silently dropping
			// it would leave the model proving the opposite of the coordinator
			if tr.Kind != "onDone" && tr.Kind != "onError" && tr.Kind != "after" && tr.Target != "" {
				trig := tr.Kind
				if tr.Event != "" {
					trig = tr.Kind + ":" + tr.Event
				}
				die("coordinator step %s declares a transition the composition model does not carry (%s -> %s); remove it or extend the composition template",
					ir.Repr(cur), trig, ir.Repr(ir.Simple(tr.Target)))
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
	if schemaErr := validateCompositionSchema(comp); schemaErr != nil {
		return "", "", "", schemaErr
	}
	name, tla, cfg = generateImpl(comp, machine, machineName)
	return
}

func validateCompositionSchema(comp *ir.Value) error {
	if comp == nil || comp.Kind != ir.KindObject {
		return composeSchemaError("composition file must be a mapping")
	}
	o := comp.AsObject()
	if err := composeRejectUnknown(o, map[string]bool{"composition": true, "coordinator": true, "aggregates": true, "sequence": true, "invariants": true}, "composition"); err != nil {
		return err
	}
	for _, key := range []string{"composition", "coordinator"} {
		if err := composeRequireString(o, key, "composition"); err != nil {
			return err
		}
	}
	aggregates := o.Get2("aggregates")
	if aggregates == nil || aggregates.Kind != ir.KindObject {
		return composeSchemaError("composition.aggregates must be a mapping")
	}
	for _, name := range aggregates.AsObject().Keys() {
		entry := aggregates.AsObject().Get2(name)
		where := "composition.aggregates." + name
		if entry == nil || entry.Kind != ir.KindObject {
			return composeSchemaError(where + " must be a mapping")
		}
		if err := composeRejectUnknown(entry.AsObject(), map[string]bool{"states": true, "initial": true}, where); err != nil {
			return err
		}
		if err := composeRequireStringArray(entry.AsObject(), "states", true, where); err != nil {
			return err
		}
		if err := composeRequireString(entry.AsObject(), "initial", where); err != nil {
			return err
		}
	}
	sequence := o.Get2("sequence")
	if sequence == nil || sequence.Kind != ir.KindArray {
		return composeSchemaError("composition.sequence must be a list")
	}
	for i, item := range sequence.AsArray() {
		where := fmt.Sprintf("composition.sequence[%d]", i)
		if item == nil || item.Kind != ir.KindObject {
			return composeSchemaError(where + " must be a mapping")
		}
		if err := composeRejectUnknown(item.AsObject(), map[string]bool{"step": true, "aggregate": true, "to": true, "undo": true}, where); err != nil {
			return err
		}
		for _, key := range []string{"step", "aggregate", "to"} {
			if err := composeRequireString(item.AsObject(), key, where); err != nil {
				return err
			}
		}
		if undo := item.AsObject().Get2("undo"); undo != nil {
			undoWhere := where + ".undo"
			if undo.Kind != ir.KindObject {
				return composeSchemaError(undoWhere + " must be a mapping")
			}
			if err := composeRejectUnknown(undo.AsObject(), map[string]bool{"to": true}, undoWhere); err != nil {
				return err
			}
			if err := composeRequireString(undo.AsObject(), "to", undoWhere); err != nil {
				return err
			}
		}
	}
	if invariants := o.Get2("invariants"); invariants != nil {
		if invariants.Kind != ir.KindObject {
			return composeSchemaError("composition.invariants must be a mapping of names to strings")
		}
		for _, name := range invariants.AsObject().Keys() {
			if err := composeRequireString(invariants.AsObject(), name, "composition.invariants"); err != nil {
				return err
			}
		}
	}
	return nil
}

func composeSchemaError(message string) error { return &ExitError{Msg: "compose_gen: " + message} }

func composeRejectUnknown(o *ir.Object, allowed map[string]bool, where string) error {
	var unknown []string
	for _, key := range o.Keys() {
		if !allowed[key] {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return composeSchemaError(fmt.Sprintf("%s has unsupported key %s", where, ir.Repr(unknown[0])))
	}
	return nil
}

func composeRequireString(o *ir.Object, key, where string) error {
	v := o.Get2(key)
	if v == nil {
		return composeSchemaError(fmt.Sprintf("%s.%s is required", where, key))
	}
	if v.Kind != ir.KindString {
		return composeSchemaError(fmt.Sprintf("%s.%s has the wrong type", where, key))
	}
	return nil
}

func composeRequireStringArray(o *ir.Object, key string, required bool, where string) error {
	v := o.Get2(key)
	if v == nil {
		if required {
			return composeSchemaError(fmt.Sprintf("%s.%s is required", where, key))
		}
		return nil
	}
	if v.Kind != ir.KindArray {
		return composeSchemaError(fmt.Sprintf("%s.%s must be a list of strings", where, key))
	}
	for i, item := range v.AsArray() {
		if item == nil || item.Kind != ir.KindString {
			return composeSchemaError(fmt.Sprintf("%s.%s[%d] must be a string", where, key, i))
		}
	}
	return nil
}

func generateImpl(comp, machine *ir.Value, machineName string) (string, string, string) {
	co := comp.AsObject()
	name := ir.Title(co.GetString("composition"))
	coordinator := co.GetString("coordinator")
	machineStem := strings.TrimSuffix(filepath.Base(machineName), ".machine.json")
	machineID := machine.AsObject().GetString("id")
	if !strings.EqualFold(coordinator, machineStem) && !strings.EqualFold(coordinator, machineID) {
		die("composition coordinator %s does not match machine %s (id %s)", ir.Repr(coordinator), ir.Repr(machineStem), ir.Repr(machineID))
	}
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
	return RunTo(compPath, machinePath, outdir, os.Stdout)
}

// RunTo is Run with an explicit status-output sink.
func RunTo(compPath, machinePath, outdir string, out io.Writer) error {
	_, err := RunWrittenTo(compPath, machinePath, outdir, out)
	return err
}

// RunWritten is Run, reporting the basenames of the files it wrote so callers
// (verify-formal) can distinguish freshly generated pairs from committed
// orphans.
func RunWritten(compPath, machinePath, outdir string) ([]string, error) {
	return RunWrittenTo(compPath, machinePath, outdir, os.Stdout)
}

// RunWrittenTo is RunWritten with an explicit status-output sink.
func RunWrittenTo(compPath, machinePath, outdir string, out io.Writer) ([]string, error) {
	snapshot, err := designlock.Acquire(filepath.Dir(filepath.Dir(machinePath)))
	if err != nil {
		return nil, err
	}
	written, retErr := RunWrittenInSnapshotTo(snapshot, compPath, machinePath, outdir, out)
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

func RunWrittenInSnapshot(snapshot *designlock.Lock, compPath, machinePath, outdir string) (written []string, retErr error) {
	return RunWrittenInSnapshotTo(snapshot, compPath, machinePath, outdir, os.Stdout)
}

// RunWrittenInSnapshotTo is RunWrittenInSnapshot with an explicit output sink.
func RunWrittenInSnapshotTo(snapshot *designlock.Lock, compPath, machinePath, outdir string, out io.Writer) (written []string, retErr error) {
	compositionSource := filepath.Base(compPath)
	if err := validateComposeOwnerBase(compositionSource); err != nil {
		return nil, &ExitError{Msg: "compose_gen: invalid composition input filename: " + err.Error()}
	}
	compositionSourceDir := ""
	if stableDir, err := snapshot.SourcePath(filepath.Dir(compPath)); err != nil {
		return nil, &ExitError{Msg: "compose_gen: resolve composition inventory: " + err.Error()}
	} else if stableDir != filepath.Dir(compPath) || pathWithin(snapshot.SourceRoot(), stableDir) {
		compositionSourceDir = stableDir
	}
	if outdir == "" {
		outdir = filepath.Dir(compPath)
	}
	if err := snapshot.ValidateOutputDir(outdir); err != nil {
		return nil, &ExitError{Msg: "compose_gen: unsafe output directory: " + err.Error()}
	}
	machines, err := snapshot.MaterializeExternalTree(filepath.Dir(machinePath))
	if err != nil {
		return nil, &ExitError{Msg: "compose_gen: snapshot machine inventory: " + err.Error()}
	}
	defer func() { retErr = errors.Join(retErr, machines.Close()) }()
	machinePath = filepath.Join(machines.Path(), filepath.Base(machinePath))
	composition, err := snapshot.MaterializeRegularFile(compPath)
	if err != nil {
		return nil, &ExitError{Msg: "compose_gen: snapshot composition: " + err.Error()}
	}
	defer func() { retErr = errors.Join(retErr, composition.Close()) }()
	compPath = composition.Path()
	if err := snapshot.ResumeExpected("compose", "rerun `machinery compose` with the same arguments"); err != nil {
		return nil, err
	}
	runWrittenAfterInputSnapshot()
	if err := ir.ValidateTLAModuleInventory(filepath.Dir(machinePath)); err != nil {
		return nil, &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	data, err := os.ReadFile(compPath)
	if err != nil {
		return nil, &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	comp, err := ir.LoadYAML(data)
	if err != nil {
		return nil, &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	if comp.Kind != ir.KindObject {
		return nil, &ExitError{Msg: "compose_gen: composition file is not a mapping"}
	}
	machine, err := ir.LoadMachineJSON(machinePath)
	if err != nil {
		return nil, &ExitError{Msg: "compose_gen: " + err.Error()}
	}
	name, tla, cfg, genErr := Generate(comp, machine, filepath.Base(machinePath))
	if genErr != nil {
		return nil, genErr
	}
	tla, err = bindCompositionSource(tla, comp.AsObject().GetString("composition")+".composition.yaml", compositionSource)
	if err != nil {
		return nil, err
	}
	// Stamp at write time (P-F10): the committed artifact records which
	// machinery version produced it; freshness diffs strip the line.
	files := map[string][]byte{
		name + ".tla": []byte(version.StampTLAModule(tla)),
		name + ".cfg": []byte(version.StampCfg(cfg)),
	}
	replacements, err := guardedCurrentComposeArtifacts(outdir, compositionSource, files)
	if err != nil {
		return nil, err
	}
	stale, err := staleOwnedComposeArtifacts(outdir, compositionSourceDir, compositionSource, files)
	if err != nil {
		return nil, err
	}
	if len(stale) > 0 {
		runWrittenAfterStalePlan()
	}
	expected := []designlock.OutputExpectation{
		designlock.ExpectFile(filepath.Join(outdir, name+".tla"), files[name+".tla"], 0o644),
		designlock.ExpectFile(filepath.Join(outdir, name+".cfg"), files[name+".cfg"], 0o644),
	}
	for _, artifact := range stale {
		expected = append(expected, designlock.ExpectAbsent(filepath.Join(outdir, artifact.Name)))
	}
	if wErr := snapshot.PublishExpectedRooted("compose", "rerun `machinery compose` with the same arguments", expected, func(outputs *designlock.OutputScope) error {
		return outputs.WithRoot(outdir, func(root *os.Root) error {
			return artifactset.ReconcileGuardedRooted(outdir, root, files, stale, replacements)
		})
	}); wErr != nil {
		return nil, wErr
	}
	if _, err := fmt.Fprintf(out, "generated %s.tla + %s.cfg\n", name, name); err != nil {
		return nil, fmt.Errorf("compose_gen: write status output: %w", err)
	}
	return []string{name + ".tla", name + ".cfg"}, nil
}

func guardedCurrentComposeArtifacts(outdir, compositionSource string, files map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
	info, err := os.Lstat(outdir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, fmt.Errorf("compose_gen: output directory must be a real directory")
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
			owner, generated, err := canonicalComposeOwner(name, body)
			if err != nil {
				return nil, err
			}
			if !generated || owner != compositionSource {
				return nil, fmt.Errorf("compose_gen: refusing to replace foreign or manual artifact %s", name)
			}
		case ".cfg":
			anchor := strings.TrimSuffix(name, ".cfg") + ".tla"
			anchorBody, ok := inspected[anchor]
			if !ok || !canonicalComposeConfig(body) {
				return nil, fmt.Errorf("compose_gen: refusing to replace unowned config %s", name)
			}
			owner, generated, err := canonicalComposeOwner(anchor, anchorBody)
			if err != nil || !generated || owner != compositionSource {
				return nil, fmt.Errorf("compose_gen: refusing to replace config %s without a same-owner generated module", name)
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

func bindCompositionSource(tla, declared, actual string) (string, error) {
	declaredHeader := `\* GENERATED by machinery compose from ` + declared + `,`
	actualHeader := `\* GENERATED by machinery compose from ` + actual + `,`
	if !strings.Contains(tla, declaredHeader) {
		return "", &ExitError{Msg: "compose_gen: generated model has no canonical source-ownership header"}
	}
	return strings.Replace(tla, declaredHeader, actualHeader, 1), nil
}

func staleOwnedComposeArtifacts(outdir, sourceDir, source string, keep map[string][]byte) ([]artifactset.RemovalPrecondition, error) {
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
		owner, generated, headerErr := canonicalComposeOwner(name, body)
		if headerErr != nil {
			return nil, headerErr
		}
		if !generated {
			continue
		}
		if sourceDir == "" {
			return nil, fmt.Errorf("compose_gen: cannot safely reconcile non-current generated artifact %s for external composition input %s; remove the stale generated pair explicitly and rerun", name, source)
		}
		owned := owner == source
		if !owned && sourceDir != "" && filepath.Base(owner) == owner && strings.HasSuffix(owner, ".composition.yaml") {
			_, statErr := os.Lstat(filepath.Join(sourceDir, owner))
			owned = errors.Is(statErr, os.ErrNotExist)
			if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
				return nil, statErr
			}
		}
		if !owned {
			continue
		}
		stale = append(stale, condition)
		cfg := strings.TrimSuffix(name, ".tla") + ".cfg"
		if _, current := keep[cfg]; current {
			continue
		}
		_, cfgCondition, inspectErr := artifactset.InspectRemovalCandidate(outdir, cfg)
		if inspectErr == nil {
			stale = append(stale, cfgCondition)
		} else if !errors.Is(inspectErr, os.ErrNotExist) {
			return nil, inspectErr
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
		if canonicalComposeConfig(body) {
			stale = append(stale, condition)
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].Name < stale[j].Name })
	return stale, nil
}

func canonicalComposeConfig(body []byte) bool {
	lines := strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
	if len(lines) < 5 || !strings.HasPrefix(lines[0], `\* machinery-version: `) ||
		lines[1] != "SPECIFICATION Spec" || lines[2] != "INVARIANT TypeOK" ||
		lines[3] != "INVARIANT Inv_CleanCompensation" || lines[len(lines)-1] != "PROPERTY Live_Terminates" {
		return false
	}
	for _, line := range lines[4 : len(lines)-1] {
		if !strings.HasPrefix(line, "INVARIANT Inv_") || strings.TrimSpace(line) != line {
			return false
		}
	}
	return true
}

func canonicalComposeOwner(name string, body []byte) (string, bool, error) {
	lines := bytes.SplitN(body, []byte("\n"), 4)
	module := strings.TrimSuffix(name, ".tla")
	if len(lines) < 3 || string(lines[0]) != "---- MODULE "+module+" ----" || !bytes.HasPrefix(lines[1], []byte(`\* machinery-version: `)) {
		return "", false, nil
	}
	const prefix = `\* GENERATED by machinery compose from `
	header := string(lines[2])
	if !strings.HasPrefix(header, prefix) || !strings.HasSuffix(header, ",") {
		return "", false, nil
	}
	owner := strings.TrimSuffix(strings.TrimPrefix(header, prefix), ",")
	if filepath.Base(owner) != owner {
		return "", true, fmt.Errorf("compose_gen: generated artifact %s has invalid source-ownership header", name)
	}
	if err := validateComposeOwnerBase(owner); err != nil {
		return "", true, fmt.Errorf("compose_gen: generated artifact %s has non-portable source owner: %w", name, err)
	}
	return owner, true, nil
}

func validateComposeOwnerBase(name string) error {
	if !strings.HasSuffix(name, ".composition.yaml") {
		return fmt.Errorf("must end in .composition.yaml")
	}
	if strings.Contains(name, ",") || strings.Contains(name, " + ") {
		return fmt.Errorf("contains a reserved ownership-header delimiter")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("contains a control character")
		}
	}
	return portablepath.ValidateBase(name)
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
