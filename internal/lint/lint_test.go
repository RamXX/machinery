package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
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

func TestAmbiguousSimpleTargetIsError(t *testing.T) {
	m, _ := ir.LoadMachineJSONStr("w", `{"id":"widget","initial":"Draft","states":{
		"Draft":{"on":{"publish":[{"target":"persisting","guard":"g","actions":"a"},{"actions":"b"}],"a":{"target":"A"},"b":{"target":"B"}},
		         "publish2":{"target":"Published"}},
		"Published":{"type":"final"},
		"persisting":{"invoke":{"src":"s","onDone":{"target":"Published"},"onError":{"target":"Draft"}},"after":{"t":{"target":"Draft"}}},
		"A":{"initial":"Dup","states":{"Dup":{"type":"final"}},"on":{"x":{"target":"Dup"}}},
		"B":{"initial":"Dup","states":{"Dup":{"type":"final"}}}}}`)
	if !contains(errsOf(t, m, "w"), "ambiguous target") {
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
