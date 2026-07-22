package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func minimalMachine() *ir.Value {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","context":{"widgetId":null},
		"_delays":{"persistTimeout":"10000 ms - store write timeout"},"states":{
		"Draft":{"on":{"publish":[
			{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},
			{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},
		                          "onError":{"target":"Draft","actions":"recordError"}},
		              "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`)
	return m
}

func errsOf(t *testing.T, m *ir.Value, base string) []string {
	t.Helper()
	if base == "" {
		base = "Widget.machine.json"
	}
	errs, _, _, _ := LintMachine(m, base)
	return errs
}

func contains(s []string, sub string) bool {
	for _, e := range s {
		if strings.Contains(e, sub) {
			return true
		}
	}
	return false
}

func TestMinimalMachineIsClean(t *testing.T) {
	errs, _, notes, counts := LintMachine(minimalMachine(), "w")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	_ = notes
	if counts.States != 3 {
		t.Errorf("states: %d", counts.States)
	}
	if counts.Transitions != 5 {
		t.Errorf("transitions: %d", counts.Transitions)
	}
}

func TestKebabCaseUnitNameIsError(t *testing.T) {
	// A hyphenated guard/action passes every structural check but can never be
	// extracted from the contract table, so G3 reports a phantom "no named-unit
	// contract row". Lint must reject it at the source (Finding 1).
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[
			{"target":"Published","guard":"guard-can-publish","actions":"set-pending"}]}},
		"Published":{"type":"final"}}}`)
	errs := errsOf(t, m, "w")
	if !contains(errs, "guard 'guard-can-publish' is not a valid identifier") {
		t.Errorf("expected the kebab-case guard to be rejected, got: %v", errs)
	}
	if !contains(errs, "action 'set-pending' is not a valid identifier") {
		t.Errorf("expected the kebab-case action to be rejected, got: %v", errs)
	}
}

func TestCamelCaseUnitNamesAreClean(t *testing.T) {
	// The reference examples are camelCase; the identifier check must not flag them.
	for _, e := range errsOf(t, minimalMachine(), "w") {
		if strings.Contains(e, "is not a valid identifier") {
			t.Errorf("camelCase unit wrongly rejected: %s", e)
		}
	}
}

func TestUnknownRootKeyIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Set("fancyExtension", ir.BoolValue(true))
	if !contains(errsOf(t, m, "w"), "unsupported root key 'fancyExtension'") {
		t.Fail()
	}
}

func TestUnknownStateKeyIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("onExit", ir.StringValue("x"))
	if !contains(errsOf(t, m, "w"), "unsupported key 'onExit'") {
		t.Fail()
	}
}

func TestParallelStateIsErrorNotSilence(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("type", ir.StringValue("parallel"))
	if !contains(errsOf(t, m, "w"), "parallel") {
		t.Fail()
	}
}

func TestDanglingTargetIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Get2("on").AsObject().Get2("publish").AsArray()[0].AsObject().Set("target", ir.StringValue("NoSuchState"))
	if !contains(errsOf(t, m, "w"), "dangling target") {
		t.Fail()
	}
}

func TestStateLevelOnDoneIsSeen(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}],
		                 "wrap":{"target":"Wrapper"}}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}},
		"Wrapper":{"initial":"Inner","states":{"Inner":{"type":"final"}},"onDone":{"target":"NoSuchState","actions":"ghostAction"}}}}`)
	if !contains(errsOf(t, m, "w"), "dangling target 'NoSuchState'") {
		t.Fail()
	}
}

func TestParameterizedActionsAreSeen(t *testing.T) {
	m := minimalMachine()
	ann, _ := ir.LoadMachineJSONStr("x", `[{"type":"announce"}]`)
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("entry", ann)
	_, actions, _ := MachineUnitNames(m)
	if !actions["announce"] {
		t.Fatal("announce not seen")
	}
}

func TestBogusActionValueIsError(t *testing.T) {
	m := minimalMachine()
	ann, _ := ir.LoadMachineJSONStr("x", `[42]`)
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("entry", ann)
	if !contains(errsOf(t, m, "w"), "unsupported action value") {
		t.Fail()
	}
}

func TestUnreachableStateIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Orphan":{"on":{"poke":{"target":"Draft"}}}}}`)
	if !contains(errsOf(t, m, "w"), "unreachable state Orphan") {
		t.Fail()
	}
}

func TestReachabilityThroughCompoundInitial(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"wrap":{"target":"Wrapper"}},
		         "publish2":{"target":"Wrapper"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Wrapper":{"initial":"Inner","states":{"Inner":{"on":{"go":{"target":"Deep"}}},"Deep":{"type":"final"}}}}}`)
	errs := errsOf(t, m, "w")
	if contains(errs, "unreachable") {
		t.Fatalf("expected no unreachable, got %v", errs)
	}
}

func TestDeadEndLeafIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"park":{"target":"Parked"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Parked":{}}}`)
	if !contains(errsOf(t, m, "w"), "dead-end non-final leaf state Parked") {
		t.Fail()
	}
}

func TestInvokeWithoutOnErrorIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Get2("invoke").AsObject().Delete("onError")
	e := errsOf(t, m, "w")
	if !contains(e, "has no onError") {
		t.Fatalf("expected 'has no onError' in %v", e)
	}
}

func TestInvokeWithoutAfterIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Delete("after")
	if !contains(errsOf(t, m, "w"), "no after/timeout") {
		t.Fail()
	}
}

func TestFinalStateWithTransitionsIsError(t *testing.T) {
	m := minimalMachine()
	pub := m.AsObject().Get2("states").AsObject().Get2("Published").AsObject()
	on, _ := ir.LoadMachineJSONStr("x", `{"poke":{"target":"Draft"}}`)
	pub.Set("on", on)
	if !contains(errsOf(t, m, "w"), "final state Published declares transitions") {
		t.Fail()
	}
}

func TestCompoundWithoutInitialIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"wrap":{"target":"Wrapper"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Wrapper":{"states":{"Inner":{"type":"final"}}}}}`)
	if !contains(errsOf(t, m, "w"), "compound state Wrapper has no initial") {
		t.Fail()
	}
}

func TestShadowedBranchIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","actions":"setPending"},{"actions":"recordDenied"}]}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}}}}`)
	errs := errsOf(t, m, "w")
	found := false
	for _, e := range errs {
		if strings.Contains(e, "unreachable") && strings.Contains(e, "branch") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected shadowed branch, got %v", errs)
	}
}

func TestFullyGuardedAlwaysWithoutEscapeIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"route":{"target":"router"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"router":{"always":[{"target":"Draft","guard":"priorIsDraft"}]}}}`)
	if !contains(errsOf(t, m, "w"), "fully guarded always-list") {
		t.Fail()
	}
}

func TestExhaustiveAnnotationDischargesGuardedAlways(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"route":{"target":"router"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"router":{"always":[{"target":"Draft","guard":"priorIsDraft"}],"_exhaustive":"priorStage ranges over {Draft} by construction"}}}`)
	errs, _, notes, _ := LintMachine(m, "w")
	if contains(errs, "fully guarded") {
		t.Fatalf("should be discharged")
	}
	if !contains(notes, "exhaustiveness") {
		t.Fatalf("missing note, got %v", notes)
	}
}

func TestChildTargetByBareNameIsDangling(t *testing.T) {
	// IR-F05/F06 decision: a bare-name target resolves ONLY among siblings of
	// the source. "Dup" from A names A's child, which XState would not resolve;
	// the old global fallback silently picked one of the two Dup states.
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"a":{"target":"A"},"b":{"target":"B"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"A":{"initial":"Dup","states":{"Dup":{"type":"final"}},"on":{"x":{"target":"Dup"}}},
		"B":{"initial":"Dup","states":{"Dup":{"type":"final"}}}}}`)
	if !contains(errsOf(t, m, "w"), "dangling target 'Dup'") {
		t.Fail()
	}
}

func TestBadInitialIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Set("initial", ir.StringValue("Nowhere"))
	if !contains(errsOf(t, m, "w"), "initial 'Nowhere'") {
		t.Fail()
	}
}

func TestLoadMachineReportsJSONError(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "Bad.machine.json")
	os.WriteFile(p, []byte("{not json"), 0644)
	_, err := ir.LoadMachineJSON(p)
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected invalid JSON error, got %v", err)
	}
}

// --- matrix reconciliation ---

const matrixOK = `
## Transition matrix

| # | source | event / after / always | guard | target | actions |
|---|---|---|---|---|---|
| 1 | Draft | publish | guardCanPublish | persisting | setPending |
| 2 | Draft | publish | !guardCanPublish | Draft (internal) | recordDenied |
| 3 | persisting | invoke onDone | - | Published | commit |
| 4 | persisting | invoke onError | - | Draft | recordError |
| 5 | persisting | after persistTimeout | - | Draft | recordTimeout |
`

func TestMatchingMatrixReconciles(t *testing.T) {
	errs, drift, n := ReconcileMatrix(minimalMachine(), matrixOK, "w")
	if len(errs) != 0 || len(drift) != 0 {
		t.Fatalf("errs=%v drift=%v", errs, drift)
	}
	if n != 5 {
		t.Errorf("n=%d", n)
	}
}

func TestRetargetedTransitionIsDrift(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("persisting").AsObject().Get2("invoke").AsObject().Get2("onDone").AsObject().Set("target", ir.StringValue("Draft"))
	_, drift, _ := ReconcileMatrix(m, matrixOK, "w")
	if !contains(drift, "machine transition has no matrix row") || !contains(drift, "matrix row has no machine transition") {
		t.Fatalf("expected both drift kinds, got %v", drift)
	}
}

func TestMatrixWithoutTransitionTableIsNotDrift(t *testing.T) {
	text := "## Named units\n\n| name | kind | signature |\n|---|---|---|\n| `x` | guard | f |\n"
	errs, drift, n := ReconcileMatrix(minimalMachine(), text, "w")
	if len(errs) != 0 || len(drift) != 0 || n != 0 {
		t.Fatalf("errs=%v drift=%v n=%d", errs, drift, n)
	}
}

func TestNamedUnitNamesParsesBacktickedGroups(t *testing.T) {
	text := "| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n" +
		"| `guardA` / `guardB` | guard | f | p | inv-x |\n" +
		"| plainName | action | g | q | - |\n"
	names := NamedUnitNames(text)
	for _, want := range []string{"guardA", "guardB", "plainName"} {
		if !names[want] {
			t.Errorf("missing %s", want)
		}
	}
}

// --- event completeness ---

func TestRestingStateMissingEventIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"park":{"target":"Parked"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Parked":{"on":{"unpark":{"target":"Draft"}}}}}`)
	errs := errsOf(t, m, "w")
	if !contains(errs, "Parked neither handles nor explicitly ignores event 'publish'") {
		t.Errorf("missing park-publish err: %v", errs)
	}
	if !contains(errs, "Draft neither handles nor explicitly ignores event 'unpark'") {
		t.Errorf("missing draft-unpark err: %v", errs)
	}
}

func TestIgnoresNotationDischargesCompleteness(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"park":{"target":"Parked"}},
		         "_ignores":{"unpark":"not parked; nothing to do"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"Parked":{"on":{"unpark":{"target":"Draft"}},"_ignores":{"publish":"a parked widget cannot be published; unpark first","park":"already parked; idempotent no-op"}}}}}`)
	if contains(errsOf(t, m, "w"), "neither handles") {
		t.Fail()
	}
}

func TestIgnoresRequiresReasonStrings(t *testing.T) {
	m := minimalMachine()
	ig, _ := ir.LoadMachineJSONStr("x", `{"publish":""}`)
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("_ignores", ig)
	if !contains(errsOf(t, m, "w"), "_ignores must map event names to reason strings") {
		t.Fail()
	}
}

func TestHandlingAndIgnoringSameEventIsError(t *testing.T) {
	m := minimalMachine()
	ig, _ := ir.LoadMachineJSONStr("x", `{"publish":"never mind"}`)
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("_ignores", ig)
	if !contains(errsOf(t, m, "w"), "both handles and ignores event 'publish'") {
		t.Fail()
	}
}

func TestTransientStatesAreExemptFromCompleteness(t *testing.T) {
	// persisting has invoke: must not be required to handle publish
	if contains(errsOf(t, minimalMachine(), "w"), "persisting neither handles") {
		t.Fail()
	}
}

func TestEmptyTransitionTableIsDriftNotAbsence(t *testing.T) {
	// Regression: a present-but-row-less table must reconcile (and report every
	// machine transition as drift), never read as "no table present".
	headerOnly := "## Transition matrix\n\n" +
		"| # | source | event / after / always | guard | target | actions |\n" +
		"|---|---|---|---|---|---|\n"
	_, drift, n := ReconcileMatrix(minimalMachine(), headerOnly, "w")
	if len(drift) == 0 {
		t.Fatal("header-only transition table reconciled clean; deleting all rows must be drift")
	}
	if !contains(drift, "machine transition has no matrix row") {
		t.Fatalf("expected machine-side drift, got %v", drift)
	}
	if n != 5 {
		t.Errorf("machine rows reconciled n=%d, want 5 (all drifted, none matched)", n)
	}
}

func TestAllRowsFilteredTableIsDriftNotAbsence(t *testing.T) {
	filtered := "## Transition matrix\n\n" +
		"| # | source | event / after / always | guard | target | actions |\n" +
		"|---|---|---|---|---|---|\n" +
		"| 1 | Published | (final) | - | - | - |\n"
	_, drift, _ := ReconcileMatrix(minimalMachine(), filtered, "w")
	if len(drift) == 0 {
		t.Fatal("documentation-only table reconciled clean against a machine with transitions")
	}
}

func TestMalformedSeparatorTransitionTableIsErrorNotAbsence(t *testing.T) {
	// Regression: a separator row without pipes (e.g. bare dashes under the
	// header) split the block, so a table that LOOKS like a transition table
	// parsed as no table at all and rows contradicting the machine passed.
	broken := "## Transition matrix\n\n" +
		"| # | source | event / after / always | guard | target | actions |\n" +
		"---\n" +
		"| 1 | Draft | publish | guardCanPublish | persisting | setPending |\n" +
		"| 2 | Draft | frob | - | Nowhere | - |\n"
	errs, drift, n := ReconcileMatrix(minimalMachine(), broken, "w")
	if !contains(errs, "no transition table parsed") {
		t.Fatalf("malformed separator row read as absence: errs=%v drift=%v n=%d", errs, drift, n)
	}
}

func TestProseRowKeywordsAreNotATransitionHeader(t *testing.T) {
	// Regression (07-03 re-review): the header detection matched "source",
	// "target", and "actions" as substrings of the whole line, so an ordinary
	// failure-catalog row containing "resource"/"retarget"/"compensating
	// actions" hard-errored a matrix that has no transition table at all.
	prose := "## Failure catalog\n\n" +
		"| failure | detection | recovery | compensation | bound |\n" +
		"|---|---|---|---|---|\n" +
		"| upstream source feed outage | health probe | retarget to backup feed | replay compensating actions | retry <= 3 |\n"
	errs, drift, n := ReconcileMatrix(minimalMachine(), prose, "w")
	if contains(errs, "no transition table parsed") {
		t.Fatalf("prose data row misread as a transition-table header: errs=%v drift=%v n=%d", errs, drift, n)
	}
}

func TestUnknownTransitionKeyIsError(t *testing.T) {
	// {"tagret": "B"} used to silently become an internal self-transition
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("on",
		strObjL(`{"publish":[{"tagret":"Published","actions":"a"},{"actions":"b"}],"pub2":{"target":"Published"}}`))
	if !contains(errsOf(t, m, "w"), "unsupported key 'tagret' in transition") {
		t.Fail()
	}
}

func TestCompoundInitialMissingChildIsError(t *testing.T) {
	m := minimalMachine()
	st := m.AsObject().Get2("states").AsObject()
	st.Get2("Draft").AsObject().Get2("on").AsObject().Set("wrap", strObjL(`{"target":"Wrapper"}`))
	st.Set("Wrapper", strObjL(`{"initial":"Ghost","states":{"Inner":{"type":"final"}},"onDone":{"target":"Published"}}`))
	if !contains(errsOf(t, m, "w"), "initial 'Ghost' is not one of its children") {
		t.Fail()
	}
}

func TestWildcardAndPlatformEventNamesAreErrors(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Get2("on").AsObject().
		Set("*", strObjL(`{"target":"Published"}`))
	if !contains(errsOf(t, m, "w"), "outside the supported subset") {
		t.Fail()
	}
	m2 := minimalMachine()
	m2.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Get2("on").AsObject().
		Set("done.invoke.save", strObjL(`{"target":"Published"}`))
	if !contains(errsOf(t, m2, "w"), "outside the supported subset") {
		t.Fail()
	}
}

func TestUnguardedAlwaysCycleIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"a","states":{
	  "a":{"always":{"target":"b"}},
	  "b":{"always":{"target":"a"}}}}`)
	if !contains(errsOf(t, m, "w"), "unguarded always cycle") {
		t.Fail()
	}
}

func TestDuplicateGuardBranchIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("on",
		strObjL(`{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"target":"Published","guard":"g"},{"actions":"b"}]}`))
	if !contains(errsOf(t, m, "w"), "two branches with the same guard") {
		t.Fail()
	}
}

func TestEmptyGuardStringIsError(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Get2("states").AsObject().Get2("Draft").AsObject().Set("on",
		strObjL(`{"publish":[{"target":"persisting","guard":"","actions":"a"},{"actions":"b"}]}`))
	if !contains(errsOf(t, m, "w"), "empty guard string") {
		t.Fail()
	}
}

func strObjL(s string) *ir.Value { v, _ := ir.LoadMachineJSONStr("x", s); return v }

func mustMachine(t *testing.T, src string) *ir.Value {
	t.Helper()
	m, err := ir.LoadMachineJSONStr("w", src)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return m
}

// --- IR-F01: empty branch lists ---

func TestEmptyEventBranchListIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m1","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":"Done"},"PING":[]}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "on:PING has no branches") {
		t.Fatalf("empty on-branch list passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F20: empty always list ---

func TestEmptyAlwaysListIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m20","initial":"Sneaky","states":{
		"Sneaky":{"always":[],"on":{"GO":{"target":"Other"}}},
		"Other":{"on":{"GO":{"target":"Done"},"PING":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "always has no branches") {
		t.Fatalf("empty always list passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F02: after: {} does not satisfy the invoke-timeout requirement ---

func TestEmptyAfterDoesNotSatisfyInvokeTimeout(t *testing.T) {
	m := mustMachine(t, `{"id":"m2","initial":"Working","states":{
		"Working":{"invoke":{"src":"doWork","onDone":{"target":"Done"},"onError":{"target":"Failed"}},"after":{}},
		"Done":{"type":"final"},
		"Failed":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "empty after") {
		t.Fatalf("after: {} vacuously satisfied the timeout requirement: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F03: empty on:{} / invoke:[] is not an escape for guarded always ---

func TestEmptyOnIsNotAnEscapeForGuardedAlways(t *testing.T) {
	m := mustMachine(t, `{"id":"m3","initial":"Router","states":{
		"Router":{"on":{},"always":[
			{"target":"A","guard":"isA"},
			{"target":"B","guard":"isB"}]},
		"A":{"type":"final"},
		"B":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "fully guarded always-list") {
		t.Fatalf("empty on:{} counted as an escape: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F04 / IR-F19: onDone placement ---

func TestOnDoneOnAtomicStateIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m4","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":"Done"}},"onDone":{"target":"Done"}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "onDone but no child states") {
		t.Fatalf("onDone on an atomic state passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestOnDoneWithoutFinalDescendantIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m19","initial":"Grp","states":{
		"Grp":{"initial":"A","onDone":{"target":"End"},"states":{
			"A":{"on":{"GO":{"target":"A"}}}}},
		"End":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "onDone but no final descendant") {
		t.Fatalf("onDone without a final descendant passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F05: #id.path resolution ---

func TestQualifiedTargetBadPathIsDangling(t *testing.T) {
	// #m5.Wrong.Leaf used to silently resolve via the last-segment fallback.
	m := mustMachine(t, `{"id":"m5","initial":"A","states":{
		"A":{"on":{"GO":{"target":"#m5.Wrong.Leaf"},"BACK":{"target":"A"}}},
		"Grp":{"initial":"Leaf","states":{"Leaf":{"on":{"BACK":{"target":"#m5.A"}}}}}}}`)
	if !contains(errsOf(t, m, "w"), "dangling target '#m5.Wrong.Leaf'") {
		t.Fatalf("bogus qualified path passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestFullyQualifiedTargetResolves(t *testing.T) {
	// #m5b.Grp.Leaf used to report ambiguous because of the simple-name fallback.
	m := mustMachine(t, `{"id":"m5b","initial":"A","states":{
		"A":{"on":{"GO":{"target":"#m5b.Grp.Leaf"},"BACK":{"target":"A"}}},
		"Grp":{"initial":"Leaf","states":{"Leaf":{"on":{"BACK":{"target":"#m5b.A"}}}}},
		"Grp2":{"initial":"Leaf","states":{"Leaf":{"on":{"BACK":{"target":"#m5b.A"}}}}}}}`)
	errs := errsOf(t, m, "w")
	for _, e := range errs {
		if strings.Contains(e, "ambiguous") || strings.Contains(e, "dangling target '#m5b.Grp.Leaf'") {
			t.Fatalf("valid fully-qualified target rejected: %v", errs)
		}
	}
}

func TestCustomStateIdTargetResolves(t *testing.T) {
	// state-level id: keys were accepted but never registered.
	m := mustMachine(t, `{"id":"m5c","initial":"A","states":{
		"A":{"on":{"GO":{"target":"#checkoutFlow"}}},
		"B":{"id":"checkoutFlow","on":{"GO":{"target":"A"}}}}}`)
	if contains(errsOf(t, m, "w"), "dangling target '#checkoutFlow'") {
		t.Fatalf("registered custom id did not resolve: %v", errsOf(t, m, "w"))
	}
}

func TestDuplicateStateIdIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"A","states":{
		"A":{"id":"dup","on":{"GO":{"target":"B"}}},
		"B":{"id":"dup","on":{"GO":{"target":"A"}}}}}`)
	if !contains(errsOf(t, m, "w"), "duplicate state id 'dup'") {
		t.Fatalf("duplicate state id passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F06: bare targets are sibling-scoped; .child is out of contract ---

func TestBareTargetIsSiblingScopedNotGlobal(t *testing.T) {
	// grp.x targets "b": the sibling grp.b must win over the same-named root
	// state, which then becomes unreachable (the old resolver picked the root).
	m := mustMachine(t, `{"id":"m6","initial":"grp","states":{
		"grp":{"initial":"x","states":{
			"x":{"on":{"GO":{"target":"b"}}},
			"b":{"type":"final"}}},
		"b":{"on":{"HUP":{"target":"b"}}}}}`)
	errs := errsOf(t, m, "w")
	if contains(errs, "dangling target 'b'") {
		t.Fatalf("sibling target did not resolve: %v", errs)
	}
	if !contains(errs, "unreachable state b") {
		t.Fatalf("root-level b should be unreachable once the sibling wins: %v", errs)
	}
}

func TestDotRelativeTargetIsRejected(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"A","states":{
		"A":{"initial":"C","states":{"C":{"type":"final"}},"on":{"GO":{"target":".C"}}},
		"B":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "relative target '.C'") {
		t.Fatalf("relative .child target passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestBareDottedTargetIsRejected(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"A","states":{
		"A":{"on":{"GO":{"target":"Grp.Leaf"}}},
		"Grp":{"initial":"Leaf","states":{"Leaf":{"type":"final"}}}}}`)
	if !contains(errsOf(t, m, "w"), "dangling target 'Grp.Leaf'") {
		t.Fatalf("bare dotted target passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F08 / IR-F24: invoke shape ---

func TestInvokeWithoutSrcIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m8","initial":"W","states":{
		"W":{"invoke":{"onDone":{"target":"Done"},"onError":{"target":"Failed"}},"after":{"T":{"target":"Failed"}}},
		"Done":{"type":"final"},
		"Failed":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "has no src") {
		t.Fatalf("srcless invoke passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestJunkInvokeEntryIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m24","initial":"W","states":{
		"W":{"invoke":[42,{"src":"realActor","onDone":{"target":"Done"},"onError":{"target":"Done"}}],"after":{"T":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	errs := errsOf(t, m, "w")
	if !contains(errs, "invoke entries must be objects") {
		t.Fatalf("junk invoke element produced no proper error: %v", errs)
	}
	if contains(errs, "invoke None") {
		t.Fatalf("junk invoke element still yields the 'invoke None' message: %v", errs)
	}
}

// --- IR-F18: completeness over resting leaf states ---

func TestNestedRestingLeafCompletenessIsEnforced(t *testing.T) {
	m := mustMachine(t, `{"id":"m18","initial":"Phase1","states":{
		"Phase1":{"initial":"WaitA","onDone":{"target":"Phase2"},"states":{
			"WaitA":{"on":{"GO":{"target":"DoneA"}}},
			"DoneA":{"type":"final"}}},
		"Phase2":{"initial":"WaitB","onDone":{"target":"End"},"states":{
			"WaitB":{"on":{"STOP":{"target":"DoneB"}}},
			"DoneB":{"type":"final"}}},
		"End":{"type":"final"}}}`)
	errs := errsOf(t, m, "w")
	if !contains(errs, "Phase1.WaitA neither handles nor explicitly ignores event 'STOP'") {
		t.Fatalf("nested resting leaf escaped completeness: %v", errs)
	}
	if !contains(errs, "Phase2.WaitB neither handles nor explicitly ignores event 'GO'") {
		t.Fatalf("nested resting leaf escaped completeness: %v", errs)
	}
}

func TestAncestorHandlerCreditsLeafCompleteness(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"P","states":{
		"P":{"initial":"Wait","on":{"STOP":{"target":"End"}},"states":{
			"Wait":{"on":{"GO":{"target":"Fin"}}},
			"Fin":{"type":"final"}}},
		"End":{"type":"final"}}}`)
	errs := errsOf(t, m, "w")
	if contains(errs, "Wait neither handles nor explicitly ignores event") {
		t.Fatalf("ancestor handler not credited: %v", errs)
	}
}

// --- IR-F22: state and event name pattern ---

func TestStateAndEventNamePatternIsEnforced(t *testing.T) {
	m := mustMachine(t, `{"id":"m22","initial":"S","states":{
		"S":{"on":{"A|on:B":{"target":"End"}}},
		"S|on:A":{"on":{"B":{"target":"End"}}},
		"End":{"type":"final"}}}`)
	errs := errsOf(t, m, "w")
	if !contains(errs, "state name 'S|on:A'") {
		t.Fatalf("separator-injecting state name passed lint: %v", errs)
	}
	if !contains(errs, "event name 'A|on:B'") {
		t.Fatalf("separator-injecting event name passed lint: %v", errs)
	}
}

func TestNonASCIIStateNameIsRejected(t *testing.T) {
	m := mustMachine(t, `{"id":"m11","initial":"Éveil","states":{
		"Éveil":{"on":{"GO":{"target":"Autre"}}},
		"Autre":{"on":{"GO":{"target":"Done"},"STOP":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "state name 'Éveil'") {
		t.Fatalf("non-ASCII state name passed lint: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F23: after keys are named delays declared in _delays ---

func TestNumericAfterKeyIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m23","initial":"Waiting","states":{
		"Waiting":{"invoke":{"src":"fetchThing","onDone":{"target":"Done"},"onError":{"target":"Done"}},
		           "after":{"5000":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "after key '5000' is a raw millisecond value") {
		t.Fatalf("raw-ms after key passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestUndeclaredAfterDelayIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"W","states":{
		"W":{"invoke":{"src":"s","onDone":{"target":"Done"},"onError":{"target":"Done"}},
		     "after":{"opTimeout":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "after delay 'opTimeout' is not declared in _delays") {
		t.Fatalf("undeclared delay passed lint: %v", errsOf(t, m, "w"))
	}
}

func TestDeclaredDelaysAreClean(t *testing.T) {
	for _, e := range errsOf(t, minimalMachine(), "w") {
		if strings.Contains(e, "_delays") || strings.Contains(e, "after delay") {
			t.Errorf("declared delay wrongly flagged: %s", e)
		}
	}
}

// --- IR-F25: absorbing non-final states ---

func TestAbsorbingStateIsError(t *testing.T) {
	m := mustMachine(t, `{"id":"m25","initial":"Start","states":{
		"Start":{"on":{"GO":{"target":"Trap"},"FIN":{"target":"Done"}},"_ignores":{"PING":"not meaningful before Trap"}},
		"Trap":{"on":{"PING":{"actions":["logPing"]}},"_ignores":{"GO":"already trapped","FIN":"cannot finish from Trap"}},
		"Done":{"type":"final"}}}`)
	if !contains(errsOf(t, m, "w"), "absorbing non-final leaf state Trap") {
		t.Fatalf("absorbing state passed the dead-end check: %v", errsOf(t, m, "w"))
	}
}

// --- IR-F12b: _oracle_tag validation ---

func TestOracleTagIsValidatedRootKey(t *testing.T) {
	m := minimalMachine()
	m.AsObject().Set("_oracle_tag", ir.StringValue("DEALAGG"))
	errs := errsOf(t, m, "w")
	if contains(errs, "unsupported root key '_oracle_tag'") {
		t.Fatalf("_oracle_tag rejected as unknown: %v", errs)
	}
	if contains(errs, "_oracle_tag") {
		t.Fatalf("valid _oracle_tag flagged: %v", errs)
	}
	bad := minimalMachine()
	bad.AsObject().Set("_oracle_tag", ir.StringValue("deal!"))
	if !contains(errsOf(t, bad, "w"), "_oracle_tag") {
		t.Fatalf("invalid _oracle_tag passed lint")
	}
}

// --- IR-F09: only marker rows are excluded from reconciliation ---

func TestAnnotatedRowsReconcileUnlessMarkerRow(t *testing.T) {
	m := mustMachine(t, `{"id":"m9","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	matrix := "## Transition matrix\n\n" +
		"| source | event | guard | target | actions |\n" +
		"|---|---|---|---|---|\n" +
		"| Idle | GO | - | Done | - |\n" +
		"| Idle | (any event) | - | Failed | logUnexpected |\n" +
		"| Idle | HALT | - | Done (final) | - |\n"
	errs, drift, _ := ReconcileMatrix(m, matrix, "w")
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if !contains(drift, "Idle --HALT") {
		t.Fatalf("row with (final) in the target cell was silently dropped: %v", drift)
	}
	if contains(drift, "(any event)") {
		t.Fatalf("documentation-only (any event) marker row reconciled: %v", drift)
	}
}

// --- IR-F16: every transition table reconciles, not just the first ---

func TestAllTransitionTablesAreReconciled(t *testing.T) {
	m := mustMachine(t, `{"id":"m16","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	matrix := "## Main transitions\n\n" +
		"| source | event | guard | target | actions |\n" +
		"|---|---|---|---|---|\n" +
		"| Idle | GO | - | Done | - |\n\n" +
		"## Supplementary transitions\n\n" +
		"| source | event | guard | target | actions |\n" +
		"|---|---|---|---|---|\n" +
		"| Idle | ABORT | - | Cancelled | rollbackEverything |\n"
	_, drift, _ := ReconcileMatrix(m, matrix, "w")
	if !contains(drift, "Idle --ABORT") {
		t.Fatalf("second transition table was not reconciled: %v", drift)
	}
}

// --- IR-F26: prose tables with substring headers are not transition tables ---

func TestProseTableWithSubstringHeadersIsNotSelected(t *testing.T) {
	m := mustMachine(t, `{"id":"m26","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":"Done"}}},
		"Done":{"type":"final"}}}`)
	matrix := "## Capacity planning\n\n" +
		"| resource | event log | guard rails | retarget | actions needed |\n" +
		"|---|---|---|---|---|\n" +
		"| db-pool | audit | rate-limit | replica | scale up |\n\n" +
		"## Transition matrix\n\n" +
		"| source | event | guard | target | actions |\n" +
		"|---|---|---|---|---|\n" +
		"| Idle | GO | - | Done | - |\n"
	errs, drift, n := ReconcileMatrix(m, matrix, "w")
	if len(errs) != 0 || len(drift) != 0 {
		t.Fatalf("prose table shadowed the real one: errs=%v drift=%v", errs, drift)
	}
	if n != 1 {
		t.Errorf("n=%d want 1", n)
	}
}

// --- IR-F15: unreadable matrix file is a hard error ---

func TestUnreadableMatrixFileIsHardError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not apply")
	}
	d := t.TempDir()
	mp := filepath.Join(d, "W.machine.json")
	os.WriteFile(mp, []byte(`{"id":"w","initial":"A","states":{"A":{"on":{"GO":{"target":"B"}}},"B":{"type":"final"}}}`), 0644)
	mx := filepath.Join(d, "W.matrix.md")
	os.WriteFile(mx, []byte("| source | event | guard | target | actions |\n|---|---|---|---|---|\n| A | WRONG | - | Nowhere | - |\n"), 0644)
	os.Chmod(mx, 0000)
	defer os.Chmod(mx, 0644)
	_, errs, _, drift := Lint(mp)
	if len(drift) != 0 {
		t.Fatalf("unreadable matrix reconciled: %v", drift)
	}
	if !contains(errs, "cannot read matrix") {
		t.Fatalf("unreadable matrix was a silent pass: errs=%v", errs)
	}
}

// --- IR-F28: named-unit fallback accepts only a single IDENT cell ---

func TestNamedUnitFallbackAcceptsOnlySingleIdent(t *testing.T) {
	text := "| name | kind | signature |\n|---|---|---|\n" +
		"| the guard checks retries and limits | guard | f |\n" +
		"| plainName | action | g |\n"
	names := NamedUnitNames(text)
	if !names["plainName"] {
		t.Errorf("single-IDENT cell not collected")
	}
	for _, junk := range []string{"the", "guard", "checks", "retries", "and", "limits"} {
		if names[junk] {
			t.Errorf("prose word %q over-collected", junk)
		}
	}
}

// --- IR-F29: matrix reconciliation keys on full paths ---

func TestSameNamedNestedStatesDoNotCrossMatch(t *testing.T) {
	m := mustMachine(t, `{"id":"m","initial":"A","states":{
		"A":{"initial":"Dup","on":{"NEXT":{"target":"B"}},"states":{
			"Dup":{"on":{"GO":{"target":"Fin"}}},
			"Fin":{"type":"final"}}},
		"B":{"initial":"Dup","states":{
			"Dup":{"on":{"HALT":{"target":"Fin"}}},
			"Fin":{"type":"final"}}}}}`)
	ambiguous := "| source | event | guard | target | actions |\n|---|---|---|---|---|\n" +
		"| Dup | GO | - | Fin | - |\n"
	errs, _, _ := ReconcileMatrix(m, ambiguous, "w")
	if !contains(errs, "ambiguous state name 'Dup'") {
		t.Fatalf("ambiguous simple name cross-matched silently: %v", errs)
	}
	qualified := "| source | event | guard | target | actions |\n|---|---|---|---|---|\n" +
		"| A | NEXT | - | B | - |\n" +
		"| A.Dup | GO | - | A.Fin | - |\n" +
		"| B.Dup | HALT | - | B.Fin | - |\n"
	errs, drift, _ := ReconcileMatrix(m, qualified, "w")
	if len(errs) != 0 || len(drift) != 0 {
		t.Fatalf("qualified paths did not reconcile: errs=%v drift=%v", errs, drift)
	}
}
