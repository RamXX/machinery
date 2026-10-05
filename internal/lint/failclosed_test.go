package lint

import (
	"strings"
	"testing"
)

// Fail closed by contract: a state tagged `unchecked` or `omitted` (an
// applicable check has not run, or a needed input was absent) must never
// reach a state tagged `accepting` (caller-facing success), by any kind of
// transition. Each fixture below violates the rule through one edge kind.

const failClosedDelays = `"_delays":{"checkTimeout":"5000 ms - checker deadline"},`

func failClosedErrs(t *testing.T, src string) []string {
	t.Helper()
	return errsOf(t, mustMachine(t, src), "Gate.machine.json")
}

func onlyFailClosed(errs []string) []string {
	var out []string
	for _, e := range errs {
		if strings.Contains(e, "fail-closed") {
			out = append(out, e)
		}
	}
	return out
}

func TestFailClosedUncheckedToAcceptingIsErrorForEveryEdgeKind(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"on", `{"id":"gate","initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"},"fail":{"target":"Failed"}}},
			"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"Passed"},"skip":{"target":"Failed"},"fail":{"target":"Failed"}}},
			"Passed":{"type":"final","tags":"accepting"},
			"Failed":{"type":"final"}}}`,
			"on:report transition from unchecked state Skipped enters accepting state Passed"},
		{"always", `{"id":"gate","initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["omitted"],"always":{"target":"Passed"}},
			"Passed":{"type":"final","tags":["accepting"]}}}`,
			"always transition from omitted state Skipped enters accepting state Passed"},
		{"after", `{"id":"gate",` + failClosedDelays + `"initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"after":{"checkTimeout":{"target":"Passed"}}},
			"Passed":{"type":"final","tags":["accepting"]}}}`,
			"after:checkTimeout transition from unchecked state Skipped enters accepting state Passed"},
		{"invoke onDone", `{"id":"gate",` + failClosedDelays + `"initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"invoke":{"src":"probe","onDone":{"target":"Passed"},"onError":{"target":"Failed"}},
				"after":{"checkTimeout":{"target":"Failed"}}},
			"Passed":{"type":"final","tags":["accepting"]},
			"Failed":{"type":"final"}}}`,
			"invoke probe.onDone transition from unchecked state Skipped enters accepting state Passed"},
		{"invoke onError", `{"id":"gate",` + failClosedDelays + `"initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"invoke":{"src":"probe","onDone":{"target":"Failed"},"onError":{"target":"Passed"}},
				"after":{"checkTimeout":{"target":"Failed"}}},
			"Passed":{"type":"final","tags":["accepting"]},
			"Failed":{"type":"final"}}}`,
			"invoke probe.onError transition from unchecked state Skipped enters accepting state Passed"},
		{"compound onDone", `{"id":"gate","initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"initial":"Waiting","states":{
				"Waiting":{"on":{"settle":{"target":"Settled"}}},
				"Settled":{"type":"final"}},
				"onDone":{"target":"Passed"}},
			"Passed":{"type":"final","tags":["accepting"]}}}`,
			"onDone transition from unchecked state Skipped enters accepting state Passed"},
		{"descendant of unchecked", `{"id":"gate","initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"initial":"Waiting","states":{
				"Waiting":{"on":{"report":{"target":"#gate.Passed"}}}}},
			"Passed":{"type":"final","tags":["accepting"]}}}`,
			"on:report transition from Skipped.Waiting (inside unchecked state Skipped) enters accepting state Passed"},
		{"accepting reached through a compound initial", `{"id":"gate","initial":"Checking","states":{
			"Checking":{"on":{"skip":{"target":"Skipped"}}},
			"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"Done"}}},
			"Done":{"initial":"Passed","states":{"Passed":{"type":"final","tags":["accepting"]}}}}}`,
			"on:report transition from unchecked state Skipped enters accepting state Done.Passed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs := failClosedErrs(t, c.src)
			if !contains(errs, c.want) {
				t.Fatalf("want an error containing %q, got: %v", c.want, errs)
			}
		})
	}
}

func TestFailClosedAcceptingInitialIsError(t *testing.T) {
	errs := failClosedErrs(t, `{"id":"gate","initial":"Passed","states":{
		"Passed":{"tags":["accepting"],"on":{"recheck":{"target":"Failed"}}},
		"Failed":{"type":"final"}}}`)
	if !contains(errs, "fail-closed: accepting state Passed is entered at machine start") {
		t.Fatalf("an accepting initial state must fail: %v", errs)
	}
	nested := failClosedErrs(t, `{"id":"gate","initial":"Run","states":{
		"Run":{"initial":"Passed","states":{
			"Passed":{"tags":["accepting"],"on":{"recheck":{"target":"#gate.Failed"}}}}},
		"Failed":{"type":"final"}}}`)
	if !contains(nested, "fail-closed: accepting state Run.Passed is entered at machine start") {
		t.Fatalf("an accepting state on the initial child chain must fail: %v", nested)
	}
}

func TestFailClosedAcceptingAndUncheckedOnOneStateIsError(t *testing.T) {
	errs := failClosedErrs(t, `{"id":"gate","initial":"Checking","states":{
		"Checking":{"on":{"done":{"target":"Passed"}}},
		"Passed":{"type":"final","tags":["accepting","omitted"]}}}`)
	if !contains(errs, "fail-closed: state Passed is tagged both accepting and omitted") {
		t.Fatalf("a state tagged accepting and omitted must fail: %v", errs)
	}
}

func TestFailClosedMalformedTagsAreError(t *testing.T) {
	errs := failClosedErrs(t, `{"id":"gate","initial":"Checking","states":{
		"Checking":{"tags":{"accepting":true},"on":{"done":{"target":"Passed"}}},
		"Passed":{"type":"final"}}}`)
	if !contains(errs, "state Checking tags must be a string or an array of non-empty strings") {
		t.Fatalf("an unreadable tags value must fail loudly: %v", errs)
	}
}

func TestFailClosedCleanMachinesPass(t *testing.T) {
	// The unchecked path ends in a failing outcome, and success is reachable
	// only from the checked path: clean.
	errs := failClosedErrs(t, `{"id":"gate","initial":"Checking","states":{
		"Checking":{"on":{"pass":{"target":"Passed"},"skip":{"target":"Skipped"}}},
		"Skipped":{"tags":["unchecked"],"on":{"report":{"target":"NotChecked"},"pass":{"target":"NotChecked"},"skip":{"target":"NotChecked"}}},
		"Passed":{"type":"final","tags":["accepting"]},
		"NotChecked":{"type":"final","tags":["omitted"]}}}`)
	if got := onlyFailClosed(errs); len(got) != 0 {
		t.Fatalf("a fail-closed machine must lint clean of fail-closed findings: %v", got)
	}
	// a machine with no tags at all is unaffected
	if got := onlyFailClosed(errsOf(t, minimalMachine(), "w")); len(got) != 0 {
		t.Fatalf("an untagged machine must be unaffected: %v", got)
	}
	// unrelated tags are inert
	errs = failClosedErrs(t, `{"id":"gate","initial":"Checking","states":{
		"Checking":{"tags":["busy"],"on":{"done":{"target":"Passed"}}},
		"Passed":{"type":"final","tags":"terminal"}}}`)
	if got := onlyFailClosed(errs); len(got) != 0 || contains(errs, "tags must be") {
		t.Fatalf("unrelated tags must be inert: %v", errs)
	}
}
