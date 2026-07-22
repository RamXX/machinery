package oracle

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
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

func TestNestedStatesAppearInEntryExitTable(t *testing.T) {
	// IR-F13: nested states' entry/exit actions were omitted and compound
	// states were labeled "atomic".
	src := `{"id":"m13","initial":"Grp","states":{
		"Grp":{"initial":"Prep","onDone":{"target":"End"},"states":{
			"Prep":{"entry":["armSensor"],"exit":["disarmSensor"],"on":{"GO":{"target":"Fin"}}},
			"Fin":{"type":"final"}}},
		"End":{"type":"final"}}}`
	m, err := ir.LoadMachineJSONStr("w", src)
	if err != nil {
		t.Fatal(err)
	}
	out := Render(m, "w")
	for _, want := range []string{
		"| Grp | compound | - | - |",
		"| Grp.Prep | atomic | armSensor | disarmSensor |",
		"| Grp.Fin | final | - | - |",
		"| End | final | - | - |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing entry/exit row %q in:\n%s", want, out)
		}
	}
}

func TestOracleTagOverride(t *testing.T) {
	// IR-F12: _oracle_tag overrides the derived 4-rune tag so colliding
	// designs (Deal vs DealAggregate) can disambiguate without renaming.
	src := `{"id":"dealAggregate","_oracle_tag":"DEALAGG","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}}},
		"Closed":{"type":"final"}}}`
	m, err := ir.LoadMachineJSONStr("w", src)
	if err != nil {
		t.Fatal(err)
	}
	if tag := Tag(m, "DealAggregate.machine.json"); tag != "DEALAGG" {
		t.Fatalf("Tag override not honored: %q", tag)
	}
	out := Render(m, "DealAggregate.machine.json")
	if !strings.Contains(out, "| DEALAGG-") || !strings.Contains(out, "| T-DEALAGG-01 |") {
		t.Fatalf("stable ids not tagged with the override:\n%s", out)
	}
}

func TestDerivedTagsCollideWithoutOverride(t *testing.T) {
	// The derived tag truncates to 4 runes; Deal and DealAggregate collide.
	// The hash input intentionally stays unchanged (stable-id churn); the
	// oracle CLI hard-errors on the collision instead.
	deal, _ := ir.LoadMachineJSONStr("w", `{"id":"deal","initial":"A","states":{"A":{"type":"final"}}}`)
	agg, _ := ir.LoadMachineJSONStr("w", `{"id":"dealAggregate","initial":"A","states":{"A":{"type":"final"}}}`)
	if Tag(deal, "Deal.machine.json") != Tag(agg, "DealAggregate.machine.json") {
		t.Fatal("expected the derived tags to collide (that is the CLI's job to catch)")
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

// P-F10: the committed oracle records which machinery version generated it,
// as a single markdown comment line the freshness gates strip before diffing.
func TestRenderStampsGeneratorVersion(t *testing.T) {
	text := Render(minimalMachine(), "Widget.machine.json")
	if !strings.Contains(text, version.MarkdownStamp()) {
		t.Fatalf("oracle carries no version stamp:\n%s", text)
	}
	if got := strings.Count(text, "machinery-version:"); got != 1 {
		t.Errorf("oracle carries %d stamp lines, want exactly 1", got)
	}
	if version.StampOf(text) != version.Version {
		t.Errorf("StampOf = %q, want %q", version.StampOf(text), version.Version)
	}
}
