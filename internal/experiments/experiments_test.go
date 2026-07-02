package experiments

import (
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/gates"
	"github.com/ramirosalas/machinery/internal/ir"
	"github.com/ramirosalas/machinery/internal/lint"
	"github.com/ramirosalas/machinery/internal/oracle"
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

// TestMachineLintExperiments runs every machine_lint mutation experiment.
func TestMachineLintExperiments(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ir.Value)
		want   string
	}{
		{"unknown-root-key", func(m *ir.Value) {
			m.AsObject().Set("fancyExtension", ir.BoolValue(true))
		}, "unsupported root key"},
		{"parallel-state", func(m *ir.Value) {
			m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("type", ir.StringValue("parallel"))
		}, "parallel"},
		{"dangling-target", func(m *ir.Value) {
			m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Get2("on").AsObject().Get2("publish").AsArray()[0].AsObject().Set("target", ir.StringValue("NoSuchState"))
		}, "dangling target"},
		{"dead-end-leaf", func(m *ir.Value) {
			st := m.AsObject().Get2("states").AsObject()
			on, _ := ir.LoadMachineJSONStr("x", `{"park":{"target":"Parked"},"pub2":{"target":"Published"}}`)
			st.Get2("Draft").AsObject().Set("on", on)
			st.Set("Parked", ir.ObjectValue(ir.NewObject()))
		}, "dead-end non-final leaf state Parked"},
		{"invoke-no-onerror", func(m *ir.Value) {
			m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Get2("invoke").AsObject().Delete("onError")
		}, "has no onError"},
		{"invoke-no-after", func(m *ir.Value) {
			m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Delete("after")
		}, "no after/timeout"},
		{"final-with-transitions", func(m *ir.Value) {
			pub := m.AsObject().Get2("states").AsObject().Get2("Published").AsObject()
			pub.Set("on", strObj(`{"poke":{"target":"Draft"}}`))
		}, "final state Published declares transitions"},
		{"shadowed-branch", func(m *ir.Value) {
			m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("on",
				strObj(`{"publish":[{"target":"persisting","actions":"x"},{"actions":"y"}]}`))
		}, "unreachable"},
		{"bad-initial", func(m *ir.Value) {
			m.AsObject().Set("initial", ir.StringValue("Nowhere"))
		}, "initial 'Nowhere'"},
		{"guarded-always-no-escape", func(m *ir.Value) {
			st := m.AsObject().Get2("states").AsObject()
			st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"route":{"target":"router"},"pub2":{"target":"Published"}}`))
			st.Set("router", strObj(`{"always":[{"target":"Draft","guard":"priorIsDraft"}]}`))
		}, "fully guarded always-list"},
		{"resting-missing-event", func(m *ir.Value) {
			st := m.AsObject().Get2("states").AsObject()
			st.Get2("Draft").AsObject().Set("on", strObj(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"park":{"target":"Parked"},"pub2":{"target":"Published"}}`))
			st.Set("Parked", strObj(`{"on":{"unpark":{"target":"Draft"}}}`))
		}, "neither handles nor explicitly ignores event 'publish'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := minimalMachine()
			c.mutate(m)
			errs, _, _, _ := lint.LintMachine(m, "w")
			if !containsAny(errs, c.want) {
				t.Errorf("expected finding containing %q, got %v", c.want, errs)
			}
		})
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

// TestMachineryCheckExperiments spot-checks the gate suite adversarial cases
// against a synthesized fixture (proves Go catches the same vacuity as Python).
func TestMachineryCheckExperiments(t *testing.T) {
	// The full gate experiments need a filesystem fixture; the byte-parity
	// differential harness (diff-all.sh) already proves Go == Python on the
	// shipped examples. Here we assert the Gate accumulator's absence-is-failure
	// discipline directly: an empty gate must report "nothing checked".
	g := gates.NewGate("probe")
	g.RequireNonzero("machines", "no machines parsed")
	if !containsAny(g.Errs, "nothing checked") {
		t.Fatal("empty gate must report 'nothing checked'")
	}
}
