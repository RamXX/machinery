package experiments

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/lint"
	"github.com/RamXX/machinery/internal/oracle"
)

// minimalMachine reproduces tests/conftest.minimal_machine as an ordered *Value.
func minimalMachine() *ir.Value {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","context":{"widgetId":null},"states":{
		"Draft":{"on":{"publish":[
			{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},
			{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},
		                          "onError":{"target":"Draft","actions":"recordError"}},
		              "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
	return m
}

func strObj(s string) *ir.Value { v, _ := ir.LoadMachineJSONStr("x", s); return v }

// containsAny reports whether any of findings contains sub.
func containsAny(findings []string, sub string) bool {
	for _, f := range findings {
		if strings.Contains(f, sub) {
			return true
		}
	}
	return false
}

// machineLintRunners maps every declared MachineLintExperiments entry to the
// mutation that exercises it. The expected finding comes from the declared
// table (single-sourced), and the registry test fails when a declared
// experiment has no runner here.
var machineLintRunners = map[string]func(*ir.Value){
	"unknown-root-key": func(m *ir.Value) {
		m.AsObject().Set("fancyExtension", ir.BoolValue(true))
	},
	"parallel-state": func(m *ir.Value) {
		m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("type", ir.StringValue("parallel"))
	},
	"dangling-target": func(m *ir.Value) {
		m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Get2("on").AsObject().Get2("publish").AsArray()[0].AsObject().Set("target", ir.StringValue("NoSuchState"))
	},
	"state-level-ondone": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}],"wrap":{"target":"Wrapper"}}`))
		st.Set("Wrapper", strObj(`{"initial":"Inner","states":{"Inner":{"type":"final"}},"onDone":{"target":"NoSuchState","actions":"ghostAction"}}`))
	},
	"dead-end-leaf": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		on, _ := ir.LoadMachineJSONStr("x", `{"park":{"target":"Parked"},"pub2":{"target":"Published"}}`)
		st.Get2("Draft").AsObject().Set("on", on)
		st.Set("Parked", ir.ObjectValue(ir.NewObject()))
	},
	"invoke-no-onerror": func(m *ir.Value) {
		m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Get2("invoke").AsObject().Delete("onError")
	},
	"invoke-no-after": func(m *ir.Value) {
		m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Delete("after")
	},
	"final-with-transitions": func(m *ir.Value) {
		pub := m.AsObject().Get2("states").AsObject().Get2("Published").AsObject()
		pub.Set("on", strObj(`{"poke":{"target":"Draft"}}`))
	},
	"compound-no-initial": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}],"wrap":{"target":"Wrapper"}}`))
		st.Set("Wrapper", strObj(`{"states":{"Inner":{"type":"final"}}}`))
	},
	"shadowed-branch": func(m *ir.Value) {
		m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("on",
			strObj(`{"publish":[{"target":"persisting","actions":"x"},{"actions":"y"}]}`))
	},
	"guarded-always-no-escape": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"route":{"target":"router"},"pub2":{"target":"Published"}}`))
		st.Set("router", strObj(`{"always":[{"target":"Draft","guard":"priorIsDraft"}]}`))
	},
	"ambiguous-target": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"a":{"target":"A"},"b":{"target":"B"}}`))
		st.Set("A", strObj(`{"initial":"Dup","states":{"Dup":{"type":"final"}},"on":{"x":{"target":"Dup"}}}`))
		st.Set("B", strObj(`{"initial":"Dup","states":{"Dup":{"type":"final"}}}`))
	},
	"bad-initial": func(m *ir.Value) {
		m.AsObject().Set("initial", ir.StringValue("Nowhere"))
	},
	"resting-missing-event": func(m *ir.Value) {
		st := m.AsObject().Get2("states").AsObject()
		st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"park":{"target":"Parked"},"pub2":{"target":"Published"}}`))
		st.Set("Parked", strObj(`{"on":{"unpark":{"target":"Draft"}}}`))
	},
	"both-handles-ignores": func(m *ir.Value) {
		ig := strObj(`{"publish":"never mind"}`)
		m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("_ignores", ig)
	},
}

func init() {
	names := make([]string, 0, len(machineLintRunners))
	for n := range machineLintRunners {
		names = append(names, n)
	}
	RegisterRunner("TestMachineLintExperiments", names...)
}

// TestMachineLintExperiments runs every declared machine_lint mutation
// experiment; the declared table drives the loop so an entry without a
// runner fails here as well as in the registry test.
func TestMachineLintExperiments(t *testing.T) {
	for _, e := range MachineLintExperiments {
		t.Run(e.Name, func(t *testing.T) {
			mutate, ok := machineLintRunners[e.Name]
			if !ok {
				t.Fatalf("declared experiment %s has no runner", e.Name)
			}
			m := minimalMachine()
			mutate(m)
			errs, _, _, _ := lint.LintMachine(m, "w")
			if !containsAny(errs, e.ExpectSubstr) {
				t.Errorf("expected finding containing %q, got %v", e.ExpectSubstr, errs)
			}
		})
	}
}

// TestEveryDeclaredExperimentHasARunner enforces the completeness contract:
// a declared experiment nobody runs is an unenforceable promise.
func TestEveryDeclaredExperimentHasARunner(t *testing.T) {
	if missing := UncoveredExperiments(); len(missing) > 0 {
		t.Fatalf("declared experiments with no registered runner: %v", missing)
	}
}

// TestOracleStableIDExperiments: the bit-parity guarantees these; spot-check.
func TestOracleStableIDExperiments(t *testing.T) {
	t.Run("stable_id_survives_insertion", func(t *testing.T) {
		before := oracle.Render(minimalMachine(), "w")
		m2, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
			"Draft":{"on":{"archive":{"target":"Published"},"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}]}},
			"Published":{"type":"final"},
			"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
		after := oracle.Render(m2, "w")
		// every WIDG- stable id in before must appear in after (insertion adds one)
		for _, line := range strings.Split(before, "\n") {
			if strings.Contains(line, "| WIDG-") {
				sid := strings.Split(strings.Split(line, "| WIDG-")[1], " |")[0]
				if !strings.Contains(after, "WIDG-"+sid+" ") && !strings.Contains(after, "WIDG-"+sid+"|") {
					t.Errorf("stable id WIDG-%s lost after insertion", sid)
				}
			}
		}
	})
}

// TestGateAccumulatorAbsenceIsFailure asserts the Gate accumulator's
// absence-is-failure discipline directly: an empty gate must report "nothing
// checked". The full MachineryCheckExperiments run against a filesystem
// fixture in gatesuite_test.go.
func TestGateAccumulatorAbsenceIsFailure(t *testing.T) {
	g := gates.NewGate("probe")
	g.RequireNonzero("machines", "no machines parsed")
	if !containsAny(g.Errs, "nothing checked") {
		t.Fatal("empty gate must report 'nothing checked'")
	}
}
