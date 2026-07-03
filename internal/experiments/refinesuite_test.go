package experiments

// Runners for RefineExperiments and ComposeExperiments, against the example
// designs (the same fixtures internal/refine and internal/compose test
// against). Driven by the declared tables so an entry without a runner fails
// loudly, and registered so the completeness test can prove coverage.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/compose"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/refine"
)

func loadExampleMachine(t *testing.T, rel string) *ir.Value {
	t.Helper()
	m, err := ir.LoadMachineJSON(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func loadExampleYAML(t *testing.T, rel string) *ir.Value {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func stringItems(v *ir.Value) []string {
	var out []string
	for _, e := range v.AsArray() {
		if e != nil && e.Kind == ir.KindString {
			out = append(out, e.AsString())
		}
	}
	return out
}

func toValues(xs []string) []*ir.Value {
	out := make([]*ir.Value, len(xs))
	for i, x := range xs {
		out[i] = ir.StringValue(x)
	}
	return out
}

const (
	dealMachinePath = "examples/go-crm/design/machines/Deal.machine.json"
	dealSemPath     = "examples/go-crm/design/formal/Deal.semantics.yaml"
	sagaMachinePath = "examples/fulfillment/design/machines/FulfillmentSaga.machine.json"
	sagaSemPath     = "examples/fulfillment/design/formal/FulfillmentSaga.semantics.yaml"
	runMachinePath  = "examples/portfolio-engine/design/machines/RecommendationRun.machine.json"
	runSemPath      = "examples/portfolio-engine/design/formal/RecommendationRun.semantics.yaml"
	compPath        = "examples/fulfillment/design/formal/checkout.composition.yaml"
)

// refineComposeRunners: experiment name -> a run of the mutated generation
// returning its error; the expected substring comes from the declared table.
var refineComposeRunners = map[string]func(t *testing.T) error{
	"lifecycle-stage-drift": func(t *testing.T) error {
		machine := loadExampleMachine(t, dealMachinePath)
		sem := loadExampleYAML(t, dealSemPath)
		stages := stringItems(sem.AsObject().Get2("stages"))
		sem.AsObject().Set("stages", ir.ArrayValue(toValues(stages[:len(stages)-1])))
		_, _, err := refine.EmitLifecycle(machine, sem, [2]string{"m", "s"})
		return err
	},
	"lifecycle-machine-edit": func(t *testing.T) error {
		machine := loadExampleMachine(t, dealMachinePath)
		sem := loadExampleYAML(t, dealSemPath)
		won := machine.AsObject().Get2("states").AsObject().Get2("Won").AsObject().Get2("on").AsObject()
		won.Set("advanceStage", strObj(`{"target":"persisting","guard":"guardCanAdvance","actions":"setPendingAdvance"}`))
		_, _, err := refine.EmitLifecycle(machine, sem, [2]string{"m", "s"})
		return err
	},
	"lifecycle-stale-rollback": func(t *testing.T) error {
		machine := loadExampleMachine(t, dealMachinePath)
		sem := loadExampleYAML(t, dealSemPath)
		rb := machine.AsObject().Get2("states").AsObject().Get2("rolledBack").AsObject()
		routes := rb.Get2("always").AsArray()
		rb.Set("always", ir.ArrayValue(routes[:len(routes)-1]))
		_, _, err := refine.EmitLifecycle(machine, sem, [2]string{"m", "s"})
		return err
	},
	"lifecycle-missing-event-name": func(t *testing.T) error {
		machine := loadExampleMachine(t, dealMachinePath)
		sem := loadExampleYAML(t, dealSemPath)
		sem.AsObject().Delete("advance_event")
		_, _, err := refine.EmitLifecycle(machine, sem, [2]string{"m", "s"})
		return err
	},
	"saga-step-order-drift": func(t *testing.T) error {
		machine := loadExampleMachine(t, sagaMachinePath)
		sem := loadExampleYAML(t, sagaSemPath)
		// swap the later steps (swapping the first would trip the initial-state
		// check before the forward-chain check this experiment targets)
		states := stringItems(sem.AsObject().Get2("states"))
		states[1], states[2] = states[2], states[1]
		sem.AsObject().Set("states", ir.ArrayValue(toValues(states)))
		_, _, err := refine.EmitSaga(machine, sem, [2]string{"m", "s"})
		return err
	},
	"saga-failure-route-drift": func(t *testing.T) error {
		machine := loadExampleMachine(t, sagaMachinePath)
		sem := loadExampleYAML(t, sagaSemPath)
		paying := machine.AsObject().Get2("states").AsObject().Get2("Paying").AsObject()
		paying.Get2("invoke").AsObject().Get2("onError").AsObject().Set("target", ir.StringValue("Failed"))
		_, _, err := refine.EmitSaga(machine, sem, [2]string{"m", "s"})
		return err
	},
	"saga-missing-undo": func(t *testing.T) error {
		machine := loadExampleMachine(t, sagaMachinePath)
		sem := loadExampleYAML(t, sagaSemPath)
		sem.AsObject().Get2("obligations").AsObject().Get2("Paying").AsObject().Delete("undo")
		_, _, err := refine.EmitSaga(machine, sem, [2]string{"m", "s"})
		return err
	},
	"terminal-phase-order-drift": func(t *testing.T) error {
		// three synthetic phases so the swap drifts the onDone chain without
		// touching the initial state
		machine, err := ir.LoadMachineJSONStr("w", `{"id":"run","initial":"P1","states":{
		  "P1":{"invoke":{"src":"s1","onDone":{"target":"P2"},"onError":{"target":"Failed"}},"after":{"t":{"target":"Failed"}}},
		  "P2":{"invoke":{"src":"s2","onDone":{"target":"P3"},"onError":{"target":"Failed"}},"after":{"t":{"target":"Failed"}}},
		  "P3":{"invoke":{"src":"s3","onDone":{"target":"Done"},"onError":{"target":"Failed"}},"after":{"t":{"target":"Failed"}}},
		  "Done":{"type":"final"},
		  "Failed":{"type":"final"}}}`)
		if err != nil {
			t.Fatal(err)
		}
		sem, err := ir.LoadYAML([]byte("machine: run\npattern: terminal-lifecycle\nphases: [P1, P3, P2]\nsuccess_terminal: Done\nfailure_terminals: [Failed]\n"))
		if err != nil {
			t.Fatal(err)
		}
		_, _, gerr := refine.EmitTerminal(machine, sem, [2]string{"m", "s"})
		return gerr
	},
	"terminal-unserved-phase-retry": func(t *testing.T) error {
		machine := loadExampleMachine(t, runMachinePath)
		sem := loadExampleYAML(t, runSemPath)
		opt := machine.AsObject().Get2("states").AsObject().Get2("Optimizing").AsObject()
		opt.Get2("invoke").AsObject().Get2("onError").AsObject().Set("target", ir.StringValue("collectRetry"))
		_, _, err := refine.EmitTerminal(machine, sem, [2]string{"m", "s"})
		return err
	},
	"compose-step-order-drift": func(t *testing.T) error {
		comp := loadExampleYAML(t, compPath)
		machine := loadExampleMachine(t, sagaMachinePath)
		seq := comp.AsObject().Get2("sequence").AsArray()
		seq[0], seq[1] = seq[1], seq[0]
		_, _, _, err := compose.Generate(comp, machine, "FulfillmentSaga.machine.json")
		return err
	},
	"compose-coordinator-edit": func(t *testing.T) error {
		comp := loadExampleYAML(t, compPath)
		machine := loadExampleMachine(t, sagaMachinePath)
		paying := machine.AsObject().Get2("states").AsObject().Get2("Paying").AsObject()
		paying.Get2("invoke").AsObject().Get2("onError").AsObject().Set("target", ir.StringValue("Failed"))
		_, _, _, err := compose.Generate(comp, machine, "FulfillmentSaga.machine.json")
		return err
	},
	"compose-missing-undo": func(t *testing.T) error {
		comp := loadExampleYAML(t, compPath)
		machine := loadExampleMachine(t, sagaMachinePath)
		seq := comp.AsObject().Get2("sequence").AsArray()
		seq[1].AsObject().Delete("undo")
		_, _, _, err := compose.Generate(comp, machine, "FulfillmentSaga.machine.json")
		return err
	},
}

func init() {
	names := make([]string, 0, len(refineComposeRunners))
	for n := range refineComposeRunners {
		names = append(names, n)
	}
	RegisterRunner("TestRefineAndComposeExperiments", names...)
}

func TestRefineAndComposeExperiments(t *testing.T) {
	all := append(append([]Experiment{}, RefineExperiments...), ComposeExperiments...)
	for _, e := range all {
		t.Run(e.Name, func(t *testing.T) {
			run, ok := refineComposeRunners[e.Name]
			if !ok {
				t.Fatalf("declared experiment %s has no runner", e.Name)
			}
			err := run(t)
			if err == nil || !strings.Contains(err.Error(), e.ExpectSubstr) {
				t.Errorf("expected error containing %q, got %v", e.ExpectSubstr, err)
			}
		})
	}
}
