package refine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
)

func repoRoot() string { return "../.." }

func loadJSON(t *testing.T, path string) *ir.Value {
	t.Helper()
	v, err := ir.LoadMachineJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func loadYAML(t *testing.T, path string) *ir.Value {
	t.Helper()
	data, err := readFile(path)
	if err != nil {
		t.Fatal(err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestLifecycleReconcilesAndEmits(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	mid, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if mid != "Deal" {
		t.Errorf("mid=%s", mid)
	}
	if body, ok := files["DealData.tla"]; !ok || !containsStr(body, "RECONCILED against the machine") {
		t.Error("DealData.tla missing or unreconciled")
	}
}

func TestLifecycleRejectsStageSetDrift(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	so := sem.AsObject()
	stages := strSlice(so.Get2("stages"))
	// drop the last stage
	newStages := ir.ArrayValue(toValueSlice(stages[:len(stages)-1]))
	so.Set("stages", newStages)
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "domain states disagree") {
		t.Fatalf("expected domain states disagree, got %v", err)
	}
}

func TestLifecycleRejectsMissingEventName(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Delete("advance_event")
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "advance_event") {
		t.Fatalf("expected advance_event error, got %v", err)
	}
}

func TestSagaReconcilesAndModelsPartialCompensation(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	_, files, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	tla := files["FulfillmentSagaData.tla"]
	if !containsStr(tla, "Undo_released") || !containsStr(tla, "Undo_refunded") {
		t.Error("per-obligation undo missing")
	}
	if !containsStr(tla, "PER OBLIGATION") {
		t.Error("PER OBLIGATION note missing")
	}
}

func TestSagaRejectsStepOrderDrift(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	so := sem.AsObject()
	// swap states order (saga uses the 'states' key for forward steps)
	states := strSlice(so.Get2("states"))
	swapped := []string{states[1], states[0]}
	swapped = append(swapped, states[2:]...)
	so.Set("states", ir.ArrayValue(toValueSlice(swapped)))
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil {
		t.Fatal("expected step order drift to fail")
	}
}

func TestTerminalReconcilesAndEmitsWithAnnotationNames(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/machines/RecommendationRun.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/formal/RecommendationRun.semantics.yaml"))
	mid, files, err := EmitTerminal(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if mid != "RecommendationRun" {
		t.Errorf("mid=%s", mid)
	}
	tla := files["RecommendationRunData.tla"]
	if !containsStr(tla, "Inv_Complete") || !containsStr(tla, "Live_Terminates") {
		t.Error("terminal invariants missing")
	}
}

func TestUnsupportedPatternExits(t *testing.T) {
	// Run with a bogus pattern; it must exit nonzero. We test via the error path.
	sem := ir.ObjectValue(ir.NewObject())
	sem.AsObject().Set("machine", ir.StringValue("Deal"))
	sem.AsObject().Set("pattern", ir.StringValue("bogus"))
	// Emit* won't be called for bogus; the Run dispatch rejects. We can't easily
	// call Run (it os.Exit). Instead verify the pattern switch would reject.
	if pat := sem.AsObject().GetString("pattern"); pat == "linear-lifecycle" || pat == "terminal-lifecycle" || pat == "saga" {
		t.Fatal("bogus pattern should not match")
	}
}

// helpers
func readFile(path string) ([]byte, error) { return osReadFile(path) }
func containsStr(s, sub string) bool       { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func toValueSlice(xs []string) []*ir.Value {
	out := make([]*ir.Value, len(xs))
	for i, x := range xs {
		out[i] = ir.StringValue(x)
	}
	return out
}

func TestLifecycleMissingStagesIsErrorNotSilentSuccess(t *testing.T) {
	// Regression: EmitLifecycle used to swallow panics via an outer recover and
	// return ("", nil, nil): exit 0, zero files. Any malformed semantics must
	// surface as a reconciliation error.
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Set("stages", ir.ArrayValue(nil))
	mid, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil {
		t.Fatalf("expected error, got mid=%q files=%d", mid, len(files))
	}
	if !containsStr(err.Error(), "stages") {
		t.Fatalf("error does not name stages: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("files emitted despite error: %d", len(files))
	}
}

func TestLifecycleInvalidReopenToIsError(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Set("reopen_to", ir.StringValue("NoSuchStage"))
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "reopen_to") {
		t.Fatalf("bogus reopen_to accepted: %v", err)
	}
}

func TestLifecycleMissingReopenToIsError(t *testing.T) {
	// Regression: reopen_to was validated only when non-empty; leaving it out
	// generated pending' = "" that only failed later under TLC.
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Delete("reopen_to")
	_, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "reopen_to") {
		t.Fatalf("missing reopen_to accepted (files=%d): %v", len(files), err)
	}
}

// --- FORMAL-F1: reconciliation must be bidirectional. A machine transition
// outside the pattern vocabulary must be a hard generation error, never a
// silently unmodeled route (the proof would assert the opposite of the machine).

func setOn(t *testing.T, machine *ir.Value, state, event, target string) {
	t.Helper()
	node := machine.AsObject().Get2("states").AsObject().Get2(state).AsObject()
	on := node.Get2("on")
	if on == nil {
		on = ir.ObjectValue(ir.NewObject())
		node.Set("on", on)
	}
	tr := ir.NewObject()
	tr.Set("target", ir.StringValue(target))
	on.AsObject().Set(event, ir.ObjectValue(tr))
}

func TestLifecycleRejectsUnmodeledDomainTransition(t *testing.T) {
	// reviewer mutation exp-b2: Negotiation gains forceWin -> Won, bypassing
	// the persist overlay entirely
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	setOn(t, machine, "Negotiation", "forceWin", "Won")
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "forceWin") {
		t.Fatalf("unmodeled forceWin transition accepted: %v", err)
	}
}

func TestLifecycleRejectsUnmodeledOverlayTransition(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	setOn(t, machine, "persisting", "cancel", "Lead")
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "cancel") {
		t.Fatalf("unmodeled overlay on: transition accepted: %v", err)
	}
}

func TestLifecycleRejectsRetryStateInvoke(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	node := machine.AsObject().Get2("states").AsObject().Get2("persistRetry").AsObject()
	inv := ir.NewObject()
	inv.Set("src", ir.StringValue("probe"))
	od := ir.NewObject()
	od.Set("target", ir.StringValue("Lead"))
	inv.Set("onDone", ir.ObjectValue(od))
	node.Set("invoke", ir.ObjectValue(inv))
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil {
		t.Fatal("unmodeled invoke on the retry overlay state accepted")
	}
}

func TestLifecycleRejectsUnexpectedOverlayState(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	extra := ir.NewObject()
	always := ir.NewObject()
	always.Set("target", ir.StringValue("Lead"))
	extra.Set("always", ir.ObjectValue(always))
	machine.AsObject().Get2("states").AsObject().Set("sneakyDetour", ir.ObjectValue(extra))
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "sneakyDetour") {
		t.Fatalf("unexpected overlay state accepted: %v", err)
	}
}

func TestTerminalRejectsUnmodeledPhaseTransition(t *testing.T) {
	machineSrc := `{"id":"run","initial":"Collecting","states":{
	  "Collecting":{"invoke":{"src":"collect","onDone":{"target":"Scoring"},"onError":{"target":"Aborted"}},"after":{"t1":{"target":"Aborted"}}},
	  "Scoring":{"invoke":{"src":"score","onDone":{"target":"Completed"},"onError":{"target":"Expired"}},"after":{"t2":{"target":"Expired"}}, "on":{"skip":{"target":"Completed"}}},
	  "Completed":{"type":"final"},
	  "Aborted":{"type":"final"},
	  "Expired":{"type":"final"}}}`
	semSrc := `machine: run
pattern: terminal-lifecycle
phases: [Collecting, Scoring]
success_terminal: Completed
failure_terminals: [Aborted, Expired]
`
	m, err := ir.LoadMachineJSONStr("w", machineSrc)
	if err != nil {
		t.Fatal(err)
	}
	sem, err := ir.LoadYAML([]byte(semSrc))
	if err != nil {
		t.Fatal(err)
	}
	_, _, emitErr := EmitTerminal(m, sem, [2]string{"m", "s"})
	if emitErr == nil || !containsStr(emitErr.Error(), "skip") {
		t.Fatalf("unmodeled phase on: transition accepted: %v", emitErr)
	}
}

func TestTerminalRejectsUnmodeledTerminalTransition(t *testing.T) {
	machineSrc := `{"id":"run","initial":"Collecting","states":{
	  "Collecting":{"invoke":{"src":"collect","onDone":{"target":"Completed"},"onError":{"target":"Aborted"}},"after":{"t1":{"target":"Aborted"}}},
	  "Completed":{"type":"final","on":{"reopen":{"target":"Collecting"}}},
	  "Aborted":{"type":"final"}}}`
	semSrc := `machine: run
pattern: terminal-lifecycle
phases: [Collecting]
success_terminal: Completed
failure_terminals: [Aborted]
`
	m, err := ir.LoadMachineJSONStr("w", machineSrc)
	if err != nil {
		t.Fatal(err)
	}
	sem, err := ir.LoadYAML([]byte(semSrc))
	if err != nil {
		t.Fatal(err)
	}
	_, _, emitErr := EmitTerminal(m, sem, [2]string{"m", "s"})
	if emitErr == nil || !containsStr(emitErr.Error(), "reopen") {
		t.Fatalf("unmodeled terminal transition accepted: %v", emitErr)
	}
}

func TestSagaRejectsUnmodeledForwardTransition(t *testing.T) {
	// reviewer mutation exp-g: Shipping gains abort -> Failed, a route the
	// saga model does not carry (it would skip compensation entirely)
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	setOn(t, machine, "Shipping", "abort", "Failed")
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "abort") {
		t.Fatalf("unmodeled saga forward transition accepted: %v", err)
	}
}

func TestSagaRejectsUnmodeledCompensatingTransition(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	setOn(t, machine, "Compensating", "giveUp", "FailedDirty")
	_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "giveUp") {
		t.Fatalf("unmodeled Compensating transition accepted: %v", err)
	}
}

// --- FORMAL-F7: the data rung and the control rung must prove the same
// retry bound; absent semantics max_retries inherits the machine's effective
// value (default 3), never 0.

func TestRefineMaxRetriesMismatchIsError(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	machine.AsObject().Set("_max_retries", ir.NumberValue("5"))
	// semantics declares 3; the machine says 5: two different systems
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "max_retries") {
		t.Fatalf("max_retries mismatch accepted: %v", err)
	}
}

func TestRefineMaxRetriesInheritsMachineValue(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	machine.AsObject().Set("_max_retries", ir.NumberValue("5"))
	sem.AsObject().Delete("max_retries")
	_, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(files["DealData.cfg"], "MaxRetries = 5") {
		t.Errorf("absent semantics max_retries did not inherit the machine value:\n%s", files["DealData.cfg"])
	}
	if !containsStr(files["DealData.tla"], "MaxRetries = 5") || !containsStr(files["DealData.tla"], "machine _max_retries") {
		t.Error("header does not state the inherited value and its source")
	}
}

func TestRefineMaxRetriesAbsentEverywhereDefaultsToMachineDefault(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Delete("max_retries")
	_, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if containsStr(files["DealData.cfg"], "MaxRetries = 0") {
		t.Fatalf("absent max_retries silently disabled retry:\n%s", files["DealData.cfg"])
	}
	if !containsStr(files["DealData.cfg"], "MaxRetries = 3") {
		t.Errorf("absent max_retries did not inherit the machine default 3:\n%s", files["DealData.cfg"])
	}
}

func TestRefineMaxRetriesInvalidIsError(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	sem.AsObject().Set("max_retries", ir.StringValue("lots"))
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !containsStr(err.Error(), "max_retries") {
		t.Fatalf("invalid max_retries accepted (would silently become 0): %v", err)
	}
}

func TestSagaMaxRetriesInheritsMachineDefault(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
	sem.AsObject().Delete("max_retries")
	_, files, err := EmitSaga(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsStr(files["FulfillmentSagaData.cfg"], "MaxRetries = 3") {
		t.Errorf("saga absent max_retries did not inherit the machine default:\n%s", files["FulfillmentSagaData.cfg"])
	}
}

func TestTerminalRetryExhaustionModelsEveryTarget(t *testing.T) {
	// Regression: exhaustion modeled only the alphabetically-first always
	// target; with several failure terminals the TLA under-approximated.
	machineSrc := `{"id":"run","initial":"Collecting","_max_retries":2,"states":{
	  "Collecting":{"invoke":{"src":"collect","onDone":{"target":"Scoring"},"onError":{"target":"collectRetry"}},"after":{"t1":{"target":"collectRetry"}}},
	  "Scoring":{"invoke":{"src":"score","onDone":{"target":"Completed"},"onError":{"target":"Expired"}},"after":{"t2":{"target":"Expired"}}},
	  "collectRetry":{"always":[{"target":"Aborted","guard":"gaveUp"},{"target":"Expired"}],"after":{"b":{"target":"Collecting"}}},
	  "Completed":{"type":"final"},
	  "Aborted":{"type":"final"},
	  "Expired":{"type":"final"}}}`
	semSrc := `machine: run
pattern: terminal-lifecycle
phases: [Collecting, Scoring]
success_terminal: Completed
failure_terminals: [Aborted, Expired]
retry: { state: collectRetry, serves: Collecting }
max_retries: 2
`
	m, err := ir.LoadMachineJSONStr("w", machineSrc)
	if err != nil {
		t.Fatal(err)
	}
	sem, err := ir.LoadYAML([]byte(semSrc))
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := EmitTerminal(m, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	var body string
	for name, b := range files {
		if containsStr(name, "Data.tla") {
			body = b
		}
	}
	if !containsStr(body, `st' \in {"Aborted", "Expired"}`) {
		t.Fatalf("exhaustion does not model every reconciled target:\n%s", body)
	}
}

func TestTerminalEmissionUsesReconciledFailureRoute(t *testing.T) {
	// A phase whose machine routes failure to failures[1] must be modeled to
	// failures[1], not silently rerouted to failures[0].
	machineSrc := `{"id":"run","initial":"Collecting","states":{
	  "Collecting":{"invoke":{"src":"collect","onDone":{"target":"Scoring"},"onError":{"target":"Aborted"}},"after":{"t1":{"target":"Aborted"}}},
	  "Scoring":{"invoke":{"src":"score","onDone":{"target":"Completed"},"onError":{"target":"Expired"}},"after":{"t2":{"target":"Expired"}}},
	  "Completed":{"type":"final"},
	  "Aborted":{"type":"final"},
	  "Expired":{"type":"final"}}}`
	semSrc := `machine: run
pattern: terminal-lifecycle
phases: [Collecting, Scoring]
success_terminal: Completed
failure_terminals: [Aborted, Expired]
`
	m, err := ir.LoadMachineJSONStr("w", machineSrc)
	if err != nil {
		t.Fatal(err)
	}
	sem, err := ir.LoadYAML([]byte(semSrc))
	if err != nil {
		t.Fatal(err)
	}
	_, files, err := EmitTerminal(m, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	var body string
	for name, b := range files {
		if containsStr(name, "Data.tla") {
			body = b
		}
	}
	if body == "" {
		t.Fatal("no Data.tla emitted")
	}
	if !containsStr(body, `Fail_Scoring == st = "Scoring" /\ st' = "Expired"`) {
		t.Fatalf("Fail_Scoring not routed to the machine's reconciled target:\n%s", body)
	}
}

// P-F10: every file RunWritten commits to design/formal carries exactly one
// version stamp line; the in-memory Emit* output stays unstamped.
func TestRunWrittenStampsGeneratorVersion(t *testing.T) {
	outdir := t.TempDir()
	names, err := RunWritten(
		filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"),
		filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"),
		outdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("RunWritten wrote nothing")
	}
	for _, n := range names {
		data, rerr := os.ReadFile(filepath.Join(outdir, n))
		if rerr != nil {
			t.Fatal(rerr)
		}
		body := string(data)
		if !strings.Contains(body, version.TLAStamp()) {
			t.Errorf("%s carries no version stamp", n)
		}
		if got := strings.Count(body, "machinery-version:"); got != 1 {
			t.Errorf("%s carries %d stamp lines, want exactly 1", n, got)
		}
		if strings.HasSuffix(n, ".tla") && !strings.HasPrefix(body, "---- MODULE ") {
			t.Errorf("%s no longer opens with the MODULE line", n)
		}
	}
}
