package oracle

import (
	"fmt"
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
	a := Render(m, "w")
	b := Render(m, "w")
	if a != b {
		t.Fatal("render not deterministic")
	}
}

func TestStableIDPrefixCollisionIsExtended(t *testing.T) {
	// Brute-force two event names whose stimulus hashes share a 6-hex prefix;
	// distinct transitions must never render as duplicate-branch suffixes.
	tag := "WIDG"
	base := stableHash(tag, "Draft", "on:evbase", "")
	var collide string
	for i := 0; ; i++ {
		cand := fmt.Sprintf("ev%d", i)
		if cand == "evbase" {
			continue
		}
		if stableHash(tag, "Draft", "on:"+cand, "")[:6] == base[:6] {
			collide = cand
			break
		}
		if i > 3_000_000 {
			t.Skip("no collision found in budget")
		}
	}
	src := fmt.Sprintf(`{"id":"widget","initial":"Draft","states":{
	  "Draft":{"on":{"evbase":{"target":"Done"},"%s":{"target":"Done"}}},
	  "Done":{"type":"final"}}}`, collide)
	m, err := ir.LoadMachineJSONStr("w", src)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(m, "w")
	if strings.Contains(out, base[:6]+".2") {
		t.Fatalf("distinct transitions rendered as duplicate suffix:\n%s", out)
	}
	if !strings.Contains(out, "WIDG-"+base[:8]) {
		t.Fatalf("expected extended 8-hex prefix for colliding rows:\n%s", out)
	}
}

func TestTagTruncationIsRuneSafe(t *testing.T) {
	src := `{"id":"багаж","initial":"A","states":{"A":{"on":{"x":{"target":"B"}}},"B":{"type":"final"}}}`
	m, err := ir.LoadMachineJSONStr("w", src)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(m, "w") // must not split a rune mid-byte
	if !strings.Contains(out, "БАГА-") {
		t.Fatalf("expected rune-safe 4-rune tag, got:\n%s", out)
	}
}
