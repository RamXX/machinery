package lint

import (
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

// A neutral session machine: heartbeat must refresh the context in every state
// that is not terminal. Paused declares heartbeat ignored, which contradicts
// the declared invariant: the exact conflict the rule exists to refuse.
func sessionMachine(t *testing.T, invariants string) *ir.Value {
	t.Helper()
	m, err := ir.LoadMachineJSONStr("s", `{"id":"session","initial":"Active",
		"context":{"lastSeen":null},`+invariants+`"states":{
		"Active":{"on":{"heartbeat":{"actions":"recordSeen"},"pause":{"target":"Paused"},
		                "close":{"target":"Closed"}},
		          "_ignores":{"resume":"already active"}},
		"Paused":{"on":{"resume":{"target":"Active"},"close":{"target":"Closed"}},
		          "_ignores":{"heartbeat":"paused sessions batch their liveness on resume",
		                      "pause":"already paused"}},
		"Closed":{"type":"final"}}}`)
	if err != nil {
		t.Fatalf("machine: %v", err)
	}
	return m
}

const conflictingInvariant = `"_invariants":[
	{"id":"liveness-always-recorded",
	 "statement":"a heartbeat refreshes lastSeen in every non-terminal state",
	 "requires":[{"event":"heartbeat","in":"all","except":["Closed"],"effect":"context"}]}],`

func TestInvariantIgnoreConflictIsAnError(t *testing.T) {
	errs := errsOf(t, sessionMachine(t, conflictingInvariant), "s")
	if !contains(errs, "liveness-always-recorded") || !contains(errs, "declares it ignored") {
		t.Fatalf("expected the ignore-consistency error, got: %v", errs)
	}
	if !contains(errs, "Paused") {
		t.Fatalf("the error must name the conflicting state, got: %v", errs)
	}
}

func TestNarrowedScopeClearsTheConflict(t *testing.T) {
	inv := `"_invariants":[
		{"id":"liveness-always-recorded",
		 "statement":"a heartbeat refreshes lastSeen while active",
		 "requires":[{"event":"heartbeat","in":["Active"],"effect":"context"}]}],`
	if errs := errsOf(t, sessionMachine(t, inv), "s"); len(errs) != 0 {
		t.Fatalf("narrowed invariant must lint clean, got: %v", errs)
	}
}

func TestDeclaringTheHandlerClearsTheConflict(t *testing.T) {
	m, err := ir.LoadMachineJSONStr("s", `{"id":"session","initial":"Active",
		"context":{"lastSeen":null},`+conflictingInvariant+`"states":{
		"Active":{"on":{"heartbeat":{"actions":"recordSeen"},"pause":{"target":"Paused"},
		                "close":{"target":"Closed"}},
		          "_ignores":{"resume":"already active"}},
		"Paused":{"on":{"resume":{"target":"Active"},"heartbeat":{"actions":"recordSeen"},
		                "close":{"target":"Closed"}},
		          "_ignores":{"pause":"already paused"}},
		"Closed":{"type":"final"}}}`)
	if err != nil {
		t.Fatalf("machine: %v", err)
	}
	if errs := errsOf(t, m, "s"); len(errs) != 0 {
		t.Fatalf("declared handler must lint clean, got: %v", errs)
	}
}

func TestInvariantWithoutRequiresIsAValidProseLaw(t *testing.T) {
	inv := `"_invariants":[{"id":"sessions-are-cheap","statement":"a session never allocates"}],`
	if errs := errsOf(t, sessionMachine(t, inv), "s"); len(errs) != 0 {
		t.Fatalf("prose-only invariant must lint clean, got: %v", errs)
	}
}

func TestMachineWithoutInvariantsLintsAsBefore(t *testing.T) {
	if errs := errsOf(t, sessionMachine(t, ""), "s"); len(errs) != 0 {
		t.Fatalf("no _invariants block must change nothing, got: %v", errs)
	}
}

func TestInvariantBlockShapeIsValidated(t *testing.T) {
	cases := []struct{ inv, want string }{
		{`"_invariants":{"not":"an array"},`, "must be an array"},
		{`"_invariants":[{"id":"BadCase","statement":"x"}],`, "kebab-case"},
		{`"_invariants":[{"id":"a-law","statement":"  "}],`, "statement must be"},
		{`"_invariants":[{"id":"a-law","statement":"x","bogus":1}],`, "unsupported key"},
		{`"_invariants":[{"id":"a-law","statement":"x"},{"id":"a-law","statement":"y"}],`, "declared twice"},
		{`"_invariants":[{"id":"a-law","statement":"x",
			"requires":[{"event":"ghost","in":"all","effect":"context"}]}],`, "not an event"},
		{`"_invariants":[{"id":"a-law","statement":"x",
			"requires":[{"event":"heartbeat","in":"all","effect":"logging"}]}],`, "effect must be"},
		{`"_invariants":[{"id":"a-law","statement":"x",
			"requires":[{"event":"heartbeat","in":["Ghost"],"effect":"context"}]}],`, "not a state"},
		{`"_invariants":[{"id":"a-law","statement":"x",
			"requires":[{"event":"heartbeat","in":["Active"],"except":["Closed"],"effect":"context"}]}],`, "only valid with"},
		{`"_invariants":[{"id":"a-law","statement":"x",
			"requires":[{"event":"heartbeat","in":[],"effect":"context"}]}],`, "empty list"},
	}
	for _, c := range cases {
		errs := errsOf(t, sessionMachine(t, c.inv), "s")
		if !contains(errs, c.want) {
			t.Errorf("invariants %s: expected error containing %q, got: %v", c.inv, c.want, errs)
		}
	}
}

func TestUnknownGuardInWhenIsAWarningNotAnError(t *testing.T) {
	inv := `"_invariants":[{"id":"a-law","statement":"x",
		"requires":[{"event":"heartbeat","when":"guardFresh","in":["Active"],"effect":"context"}]}],`
	m := sessionMachine(t, inv)
	errs, warns, _, _ := LintMachine(m, "s")
	if len(errs) != 0 {
		t.Fatalf("unknown guard must not error, got: %v", errs)
	}
	found := false
	for _, w := range warns {
		if strings.Contains(w, "guardFresh") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning naming the undeclared guard, got: %v", warns)
	}
}

func TestAncestorIgnoreConflictsForTheWholeSubtree(t *testing.T) {
	// The ignore lives on the parent; the invariant covers the child. Ancestor
	// credit applies to ignores exactly as it does to handlers.
	m, err := ir.LoadMachineJSONStr("s", `{"id":"session","initial":"Work",
		"context":{"lastSeen":null},
		"_invariants":[{"id":"liveness-always-recorded","statement":"x",
			"requires":[{"event":"heartbeat","in":["Work.Deep"],"effect":"context"}]}],
		"states":{
		"Work":{"initial":"Deep","_ignores":{"heartbeat":"batched at the parent"},
			"states":{"Deep":{"on":{"finish":{"actions":"recordDone"}}}}}}}`)
	if err != nil {
		t.Fatalf("machine: %v", err)
	}
	errs := errsOf(t, m, "s")
	if !contains(errs, "Work.Deep") || !contains(errs, "declares it ignored") {
		t.Fatalf("expected the subtree conflict, got: %v", errs)
	}
}
