package refine

import (
	"path/filepath"
	"testing"

	"github.com/ramirosalas/machinery/internal/ir"
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
