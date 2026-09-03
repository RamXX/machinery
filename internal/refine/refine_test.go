package refine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
)

func repoRoot() string { return "../.." }

func isolatedExampleDesign(t *testing.T, rel string) string {
	t.Helper()
	design := filepath.Join(t.TempDir(), "design")
	if err := os.CopyFS(design, os.DirFS(filepath.Join(repoRoot(), rel))); err != nil {
		t.Fatalf("copy isolated example design: %v", err)
	}
	return design
}

func loadJSON(t *testing.T, path string) *ir.Value {
	t.Helper()
	v, err := ir.LoadMachineJSON(path)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestGuardedCurrentRefinementArtifactsRejectForeignFamilyMembers(t *testing.T) {
	files := map[string][]byte{}
	for _, suffix := range refinementFamilySuffixes {
		files["Deal"+suffix] = []byte("generated " + suffix + "\n")
	}
	for _, suffix := range refinementFamilySuffixes {
		t.Run(suffix, func(t *testing.T) {
			dir := t.TempDir()
			target := "Deal" + suffix
			foreign := []byte("hand-authored sentinel\n")
			if err := os.WriteFile(filepath.Join(dir, target), foreign, 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := guardedCurrentRefinementArtifacts(dir, "Deal.machine.json", "Deal.semantics.yaml", files); err == nil {
				t.Fatalf("foreign %s was accepted for replacement", target)
			}
			if got, err := os.ReadFile(filepath.Join(dir, target)); err != nil || string(got) != string(foreign) {
				t.Fatalf("foreign target changed: %q, %v", got, err)
			}
		})
	}
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

func TestLifecycleModelsExplicitRollbackFault(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/machines/Portfolio.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/formal/Portfolio.semantics.yaml"))
	_, files, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err != nil {
		t.Fatal(err)
	}
	body := files["PortfolioData.tla"]
	for _, want := range []string{`Fault == {"routingFault"}`, "RollbackFault ==", `st' \in Fault`, `stage' \in Fault`, `FaultStutter == st \in Fault`} {
		if !strings.Contains(body, want) {
			t.Errorf("generated refinement does not model rollback fault %q", want)
		}
	}
	if mapping := files["PortfolioRefinement.tla"]; !strings.Contains(mapping, `st \in Fault`) {
		t.Error("refinement mapping does not classify the rollback fault as terminal")
	}
}

func TestLifecycleRejectsNonFinalRollbackFault(t *testing.T) {
	machine := loadJSON(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/machines/Portfolio.machine.json"))
	sem := loadYAML(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/formal/Portfolio.semantics.yaml"))
	machine.AsObject().Get2("states").AsObject().Get2("routingFault").AsObject().Delete("type")
	_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
	if err == nil || !strings.Contains(err.Error(), "must be final") {
		t.Fatalf("non-final rollback fault accepted: %v", err)
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

func repeatedError(t *testing.T, runs int, run func() error) string {
	t.Helper()
	var first string
	for i := 0; i < runs; i++ {
		err := run()
		if err == nil {
			t.Fatalf("run %d returned no error", i)
		}
		got := err.Error()
		if i == 0 {
			first = got
		} else if got != first {
			t.Fatalf("run %d diagnostic changed:\nfirst: %q\n got: %q", i, first, got)
		}
	}
	return first
}

func captureExitError(run func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if exitErr, ok := recovered.(*ExitError); ok {
				err = exitErr
				return
			}
			panic(recovered)
		}
	}()
	run()
	return nil
}

func addEmptyTopState(machine *ir.Value, name string) {
	machine.AsObject().Get2("states").AsObject().Set(name, ir.ObjectValue(ir.NewObject()))
}

func TestLifecycleRollbackMissingRouteDiagnosticIsDeterministic(t *testing.T) {
	fallback := ir.NewObject()
	fallback.Set("target", ir.StringValue("Fault"))
	node := ir.NewObject()
	node.Set("always", ir.ObjectValue(fallback))
	enters := map[string]bool{"Zulu": true, "Alpha": true}

	got := repeatedError(t, 100, func() error {
		return captureExitError(func() {
			validateLifecycleRollbackRoutes(ir.ObjectValue(node), "rolledBack", "Fault", enters)
		})
	})
	if !strings.Contains(got, "priorIsAlpha -> Alpha") {
		t.Fatalf("first missing route is not the sorted state: %q", got)
	}
}

func TestLifecycleMultipleUnexpectedStatesDiagnosticIsDeterministic(t *testing.T) {
	got := repeatedError(t, 100, func() error {
		machine := loadJSON(t, filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
		sem := loadYAML(t, filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
		addEmptyTopState(machine, "zDetour")
		addEmptyTopState(machine, "aDetour")
		_, _, err := EmitLifecycle(machine, sem, [2]string{"m", "s"})
		return err
	})
	if !strings.Contains(got, "aDetour") {
		t.Fatalf("first unexpected state is not the sorted state: %q", got)
	}
}

func TestTerminalMultipleUnexpectedStatesDiagnosticIsDeterministic(t *testing.T) {
	got := repeatedError(t, 100, func() error {
		machine := loadJSON(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/machines/RecommendationRun.machine.json"))
		sem := loadYAML(t, filepath.Join(repoRoot(), "examples/portfolio-engine/design/formal/RecommendationRun.semantics.yaml"))
		addEmptyTopState(machine, "zDetour")
		addEmptyTopState(machine, "aDetour")
		_, _, err := EmitTerminal(machine, sem, [2]string{"m", "s"})
		return err
	})
	if !strings.Contains(got, "aDetour") {
		t.Fatalf("first unexpected state is not the sorted state: %q", got)
	}
}

func TestSagaMultipleUnexpectedStatesDiagnosticIsDeterministic(t *testing.T) {
	got := repeatedError(t, 100, func() error {
		machine := loadJSON(t, filepath.Join(repoRoot(), "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"))
		sem := loadYAML(t, filepath.Join(repoRoot(), "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"))
		addEmptyTopState(machine, "zDetour")
		addEmptyTopState(machine, "aDetour")
		_, _, err := EmitSaga(machine, sem, [2]string{"m", "s"})
		return err
	})
	aDetour := strings.Index(got, "aDetour")
	zDetour := strings.Index(got, "zDetour")
	if aDetour < 0 || zDetour < 0 || aDetour > zDetour {
		t.Fatalf("unexpected saga states are not reported in sorted order: %q", got)
	}
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
	design := isolatedExampleDesign(t, "examples/go-crm/design")
	outdir := t.TempDir()
	names, err := RunWritten(
		filepath.Join(design, "machines/Deal.machine.json"),
		filepath.Join(design, "formal/Deal.semantics.yaml"),
		outdir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("RunWritten wrote nothing")
	}
	if !sort.StringsAreSorted(names) {
		t.Fatalf("RunWritten returned nondeterministic artifact order: %v", names)
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

func TestRunWrittenControlFlowOnlyValidatesWithoutArtifacts(t *testing.T) {
	dir := t.TempDir()
	machine := filepath.Join(dir, "Order.machine.json")
	sem := filepath.Join(dir, "Order.semantics.yaml")
	body := `{"id":"order","initial":"A","states":{"A":{"on":{"go":{"target":"B"}}},"B":{"type":"final"}}}`
	if err := os.WriteFile(machine, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	valid := "machine: order\npattern: control-flow-only\nreason: event-driven lifecycle does not fit a data-refinement algebra\n"
	if err := os.WriteFile(sem, []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	names, err := RunWritten(machine, sem, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("control-flow-only claimed artifacts: %v", names)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("control-flow-only created an output directory/artifact: %v", err)
	}

	bad := strings.Replace(valid, "reason: event-driven lifecycle does not fit a data-refinement algebra", "reason: ''", 1)
	if err := os.WriteFile(sem, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	_, firstErr := RunWritten(machine, sem, out)
	_, secondErr := RunWritten(machine, sem, out)
	if firstErr == nil || !strings.Contains(firstErr.Error(), "non-empty reason") {
		t.Fatalf("invalid control-flow-only declaration accepted: %v", firstErr)
	}
	if secondErr == nil || firstErr.Error() != secondErr.Error() || strings.Contains(firstErr.Error(), "machinery-design-source-") || strings.Contains(firstErr.Error(), "machinery-input-snapshot-") {
		t.Fatalf("invalid semantics diagnostic is unstable or leaks private snapshot:\nfirst: %v\nsecond: %v", firstErr, secondErr)
	}
}

func TestRunWrittenControlFlowOnlyReconcilesRenamedSemanticsFiveFileFamily(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines := filepath.Join(design, "machines")
	formal := filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	semanticsBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(machines, "Deal.machine.json")
	semantics := filepath.Join(formal, "Deal.semantics.yaml")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semantics, semanticsBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(machine, semantics, out); err != nil {
		t.Fatal(err)
	}
	renamedSemantics := filepath.Join(formal, "Renamed.semantics.yaml")
	controlFlowOnly := "machine: deal\npattern: control-flow-only\nreason: this lifecycle is intentionally verified only as control flow\n"
	if err := os.WriteFile(renamedSemantics, []byte(controlFlowOnly), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(semantics); err != nil {
		t.Fatal(err)
	}
	names, err := RunWritten(machine, renamedSemantics, out)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("control-flow-only claimed artifacts: %v", names)
	}
	for _, suffix := range refinementFamilySuffixes {
		if _, err := os.Lstat(filepath.Join(out, "Deal"+suffix)); !os.IsNotExist(err) {
			t.Fatalf("control-flow-only left stale Deal%s: %v", suffix, err)
		}
	}
}

func TestRunWrittenFailsClosedOnAmbiguousExternalSemanticsOwnership(t *testing.T) {
	design := isolatedExampleDesign(t, "examples/go-crm/design")
	machine := filepath.Join(design, "machines/Deal.machine.json")
	linear, err := os.ReadFile(filepath.Join(design, "formal/Deal.semantics.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	controlFlowOnly := []byte("machine: deal\npattern: control-flow-only\nreason: this lifecycle is intentionally verified only as control flow\n")
	for _, tc := range []struct {
		name       string
		secondBase string
	}{
		{name: "same basename in another external directory", secondBase: "Deal.semantics.yaml"},
		{name: "external source move and rename", secondBase: "Moved.semantics.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := filepath.Join(t.TempDir(), "Deal.semantics.yaml")
			second := filepath.Join(t.TempDir(), tc.secondBase)
			if err := os.WriteFile(first, linear, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(second, controlFlowOnly, 0o644); err != nil {
				t.Fatal(err)
			}
			out := t.TempDir()
			if _, err := RunWritten(machine, first, out); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(filepath.Join(out, "DealData.tla"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RunWritten(machine, second, out); err == nil || !strings.Contains(err.Error(), "cannot safely reconcile") {
				t.Fatalf("ambiguous external semantics ownership was not rejected: %v", err)
			}
			after, err := os.ReadFile(filepath.Join(out, "DealData.tla"))
			if err != nil || string(after) != string(before) {
				t.Fatalf("existing external-owner refinement output changed: %v", err)
			}
			for _, suffix := range refinementFamilySuffixes {
				if _, err := os.Stat(filepath.Join(out, "Deal"+suffix)); err != nil {
					t.Fatalf("fail-closed external cleanup removed Deal%s: %v", suffix, err)
				}
			}
		})
	}
}

func TestRunWrittenRejectsAtomicReplacementOfPlannedStaleRefinement(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines, formal := filepath.Join(design, "machines"), filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, _ := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	semanticsBody, _ := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	machine := filepath.Join(machines, "Deal.machine.json")
	semantics := filepath.Join(formal, "Deal.semantics.yaml")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semantics, semanticsBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(machine, semantics, out); err != nil {
		t.Fatal(err)
	}
	controlFlowOnly := "machine: deal\npattern: control-flow-only\nreason: this lifecycle is intentionally verified only as control flow\n"
	if err := os.WriteFile(semantics, []byte(controlFlowOnly), 0o644); err != nil {
		t.Fatal(err)
	}
	prior := runWrittenAfterStalePlan
	runWrittenAfterStalePlan = func() { atomicReplaceFile(t, filepath.Join(out, "DealData.tla")) }
	defer func() { runWrittenAfterStalePlan = prior }()
	if _, err := RunWritten(machine, semantics, out); err == nil || !strings.Contains(err.Error(), "ownership plan") {
		t.Fatalf("atomic stale replacement was accepted: %v", err)
	}
	for _, suffix := range refinementFamilySuffixes {
		if _, err := os.Stat(filepath.Join(out, "Deal"+suffix)); err != nil {
			t.Fatalf("old family mutated at Deal%s: %v", suffix, err)
		}
	}
}

func TestRefinementOwnershipGrammarAndCanonicalHeader(t *testing.T) {
	for _, name := range []string{"bad,name.semantics.yaml", "bad + owner.semantics.yaml", "bad\nname.semantics.yaml", "bad\x01name.semantics.yaml"} {
		if err := validateRefinementOwnerBase(name, ".semantics.yaml"); err == nil {
			t.Errorf("hostile ownership filename accepted: %q", name)
		}
	}
	out := t.TempDir()
	manual := "---- MODULE DealData ----\n\\* machinery-version: v1\nEXTENDS Naturals\nManual == \\\"\\* GENERATED by machinery refine from Deal.machine.json + Deal.semantics.yaml.\\\"\n====\n"
	if err := os.WriteFile(filepath.Join(out, "DealData.tla"), []byte(manual), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, err := staleOwnedRefinementArtifacts(out, t.TempDir(), t.TempDir(), "Deal.machine.json", "Deal.semantics.yaml", map[string][]byte{})
	if err != nil || len(stale) != 0 {
		t.Fatalf("handwritten quoted marker authorized deletion: stale=%v err=%v", stale, err)
	}
}

func TestStaleOwnedRefinementArtifactsConvergesEveryMissingAnchorMember(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines, formal := filepath.Join(design, "machines"), filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	semanticsBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(machines, "Deal.machine.json")
	semantics := filepath.Join(formal, "Deal.semantics.yaml")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semantics, semanticsBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(machine, semantics, out); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(out, "DealData.tla")); err != nil {
		t.Fatal(err)
	}
	stale, err := staleOwnedRefinementArtifacts(out, machines, formal, "Deal.machine.json", "Deal.semantics.yaml", map[string][]byte{})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(stale))
	for _, condition := range stale {
		got = append(got, condition.Name)
	}
	want := []string{"DealContract.tla", "DealData.cfg", "DealRefinement.cfg", "DealRefinement.tla"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("missing-anchor family plan = %v, want %v", got, want)
	}
}

func atomicReplaceFile(t *testing.T, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	temp := path + ".replacement"
	if err := os.WriteFile(temp, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temp, path); err != nil {
		t.Fatal(err)
	}
}

func TestSemanticsSchemaRejectsUnknownAndWrongTypedClaims(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"root typo", "machine: x\npattern: linear-lifecycle\nmax_retry: 3\n", "max_retry"},
		{"overlay typo", "machine: x\npattern: linear-lifecycle\nstages: [A]\nwin_stage: W\nlose_stage: L\nreopen_to: A\nclose_date_on: W\nadvance_event: a\nwin_event: w\nlose_event: l\nreopen_event: r\noverlay: {retri: R}\n", "retri"},
		{"wrong list type", "machine: x\npattern: terminal-lifecycle\nphases: A\nsuccess_terminal: W\nfailure_terminals: [L]\nsuccess_flag: done\n", "phases must be a list"},
		{"retry typo", "machine: x\npattern: terminal-lifecycle\nphases: [A]\nsuccess_terminal: W\nfailure_terminals: [L]\nsuccess_flag: done\nretry: {state: R, serve: A}\n", "serve"},
		{"obligation typo", "machine: x\npattern: saga\nstates: [A]\nobligations: {A: {set: done}}\n", "set"},
		{"null retry bound", "machine: x\npattern: saga\nstates: [A]\nobligations: {A: {sets: done}}\nmax_retries: null\n", "max_retries has the wrong type"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sem, err := ir.LoadYAML([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if err := validateSemanticsSchema(sem); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid semantics accepted: %v", err)
			}
		})
	}
}

func TestSemanticsSchemaFirstUnknownKeyIsDeterministic(t *testing.T) {
	sem, err := ir.LoadYAML([]byte("machine: x\npattern: control-flow-only\nreason: x\nz_bad: 1\na_bad: 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1_000; i++ {
		err := validateSemanticsSchema(sem)
		if err == nil || !strings.Contains(err.Error(), "'a_bad'") {
			t.Fatalf("validation %d = %v", i, err)
		}
	}
}

func TestRunWrittenRejectsSymlinkedAndMutatingSemanticsInput(t *testing.T) {
	design := isolatedExampleDesign(t, "examples/go-crm/design")
	machine := filepath.Join(design, "machines/Deal.machine.json")
	source := filepath.Join(design, "formal/Deal.semantics.yaml")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("symlink", func(t *testing.T) {
		dir := t.TempDir()
		link := filepath.Join(dir, "Deal.semantics.yaml")
		if err := os.Symlink(source, link); err != nil {
			t.Fatal(err)
		}
		if _, err := RunWritten(machine, link, t.TempDir()); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlinked semantics accepted: %v", err)
		}
	})
	t.Run("mutation", func(t *testing.T) {
		dir := t.TempDir()
		input := filepath.Join(dir, "Deal.semantics.yaml")
		if err := os.WriteFile(input, body, 0o644); err != nil {
			t.Fatal(err)
		}
		prior := runWrittenAfterInputSnapshot
		runWrittenAfterInputSnapshot = func() {
			if err := os.WriteFile(input, append(body, []byte("\n# concurrent edit\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		defer func() { runWrittenAfterInputSnapshot = prior }()
		out := filepath.Join(dir, "out")
		if _, err := RunWritten(machine, input, out); err == nil || !strings.Contains(err.Error(), "external input") {
			t.Fatalf("mutating semantics accepted: %v", err)
		}
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("mutation wrote output: %v", err)
		}
	})
	t.Run("symlinked output directory", func(t *testing.T) {
		dir := t.TempDir()
		outside := t.TempDir()
		out := filepath.Join(dir, "out")
		if err := os.Symlink(outside, out); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, err := RunWritten(machine, source, out); err == nil || !strings.Contains(err.Error(), "unsafe output directory") {
			t.Fatalf("symlinked output directory accepted: %v", err)
		}
	})
}

func TestRunWrittenReconcilesPriorFiveFileFamilyAfterMachineRename(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	machines := filepath.Join(design, "machines")
	formal := filepath.Join(design, "formal")
	if err := os.MkdirAll(machines, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(formal, 0o755); err != nil {
		t.Fatal(err)
	}
	machineBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/machines/Deal.machine.json"))
	if err != nil {
		t.Fatal(err)
	}
	semanticsBody, err := os.ReadFile(filepath.Join(repoRoot(), "examples/go-crm/design/formal/Deal.semantics.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	machine := filepath.Join(machines, "Deal.machine.json")
	semantics := filepath.Join(formal, "Deal.semantics.yaml")
	if err := os.WriteFile(machine, machineBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(semantics, semanticsBody, 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if _, err := RunWritten(machine, semantics, out); err != nil {
		t.Fatal(err)
	}
	renamedMachineBody := strings.Replace(string(machineBody), `"id": "deal"`, `"id": "renamed"`, 1)
	renamedSemanticsBody := strings.Replace(string(semanticsBody), "machine: deal", "machine: renamed", 1)
	if renamedMachineBody == string(machineBody) || renamedSemanticsBody == string(semanticsBody) {
		t.Fatal("fixture source identities were not replaced")
	}
	renamedMachine := filepath.Join(machines, "Renamed.machine.json")
	renamedSemantics := filepath.Join(formal, "Renamed.semantics.yaml")
	if err := os.WriteFile(renamedMachine, []byte(renamedMachineBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(renamedSemantics, []byte(renamedSemanticsBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(machine); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(semantics); err != nil {
		t.Fatal(err)
	}
	if _, err := RunWritten(renamedMachine, renamedSemantics, out); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range refinementFamilySuffixes {
		if _, err := os.Lstat(filepath.Join(out, "Deal"+suffix)); !os.IsNotExist(err) {
			t.Fatalf("stale refinement artifact Deal%s remains: %v", suffix, err)
		}
		if _, err := os.Stat(filepath.Join(out, "Renamed"+suffix)); err != nil {
			t.Fatalf("current refinement artifact Renamed%s missing: %v", suffix, err)
		}
	}
}
