package oracle

import (
	"regexp"
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/ir"
)

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

func rowsOf(t *testing.T, text string) map[string]string {
	out := map[string]string{}
	rowRe := regexp.MustCompile(`\|\s*(T-\S+)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*\|\s*(\S+)\s*\|`)
	for _, line := range strings.Split(text, "\n") {
		if m := rowRe.FindStringSubmatch(line); m != nil {
			out[m[2]] = line // stable id -> row
		}
	}
	return out
}

func TestRenderHasStableAndSequentialIDs(t *testing.T) {
	text := Render(minimalMachine(), "Widget.machine.json")
	if !strings.Contains(text, "| test id | stable id |") {
		t.Fatalf("missing header")
	}
	rows := rowsOf(t, text)
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows, got %d", len(rows))
	}
	re := regexp.MustCompile(`WIDG-[0-9a-f]{6}`)
	for sid := range rows {
		if !re.MatchString(sid) {
			t.Errorf("stable id %q does not match WIDG-hex6", sid)
		}
	}
}

func TestStableIDsSurviveUnrelatedInsertion(t *testing.T) {
	before := rowsOf(t, Render(minimalMachine(), "w"))
	m2, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"archive":{"target":"Published"},"publish":[
			{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},
			{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},
		                          "onError":{"target":"Draft","actions":"recordError"}},
		              "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
	after := rowsOf(t, Render(m2, "w"))
	// before ⊆ after
	for sid := range before {
		if _, ok := after[sid]; !ok {
			t.Errorf("stable id %q lost after insertion", sid)
		}
	}
	lost := 0
	for sid := range after {
		if _, ok := before[sid]; !ok {
			lost++
		}
	}
	if lost != 1 {
		t.Fatalf("expected exactly 1 new id, got %d", lost)
	}
}

func TestStableIDChangesWhenStimulusChanges(t *testing.T) {
	before := rowsOf(t, Render(minimalMachine(), "w"))
	m2, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[
			{"target":"persisting","guard":"guardCanShip","actions":"setPending"},
			{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},
		                          "onError":{"target":"Draft","actions":"recordError"}},
		              "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
	after := rowsOf(t, Render(m2, "w"))
	// Python asserts set(before) != set(after): at least one id changed.
	same := true
	for sid := range before {
		if _, ok := after[sid]; !ok {
			same = false
			break
		}
	}
	for sid := range after {
		if _, ok := before[sid]; !ok {
			same = false
			break
		}
	}
	if same {
		t.Fatal("stimulus changed but stable id set unchanged")
	}
}

func TestStableIDConstantWhenOnlyExpectationChanges(t *testing.T) {
	before := rowsOf(t, Render(minimalMachine(), "w"))
	m2, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[
			{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},
			{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commitAndLog"},
		                          "onError":{"target":"Draft","actions":"recordError"}},
		              "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
	after := rowsOf(t, Render(m2, "w"))
	if len(before) != len(after) {
		t.Fatalf("row count changed")
	}
	changed := 0
	for sid := range before {
		if before[sid] != after[sid] {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("expected exactly 1 changed row, got %d", changed)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	m := minimalMachine()
	if Render(m, "w") != Render(m, "w") {
		t.Fatal("render not deterministic")
	}
}
