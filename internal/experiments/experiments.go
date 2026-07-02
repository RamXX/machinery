// Package experiments contains the shared, language-neutral mutation
// experiments ported from the pytest suite (tests/test_machine_lint.py,
// test_machinery_check.py, test_formal_gens.py). They encode the review's
// vacuity/drift findings as data; the Go table runner applies each mutation
// to a fixture design and asserts the expected finding + exit.
package experiments

// Experiment is one adversarial mutation: apply it to a clean design and
// assert the tool catches it (expected finding substring + nonzero exit).
type Experiment struct {
	Name     string // human label
	Tool     string // "lint" | "check" | "tla" | "refine" | "compose"
	Mutation string // describes what to mutate (for documentation)
	// Expect is applied to the synthesized fixture: the substring the gate
	// finding or generator error must contain, and whether it must be nonzero.
	ExpectSubstr string
	ExpectExit   bool // true if the tool must exit nonzero
}

// MachineLintExperiments are the lint IR mutations (machine_lint review findings).
var MachineLintExperiments = []Experiment{
	{Name: "unknown-root-key", Tool: "lint", Mutation: "add unsupported root key",
		ExpectSubstr: "unsupported root key", ExpectExit: true},
	{Name: "parallel-state", Tool: "lint", Mutation: "state type parallel",
		ExpectSubstr: "parallel", ExpectExit: true},
	{Name: "dangling-target", Tool: "lint", Mutation: "target NoSuchState",
		ExpectSubstr: "dangling target", ExpectExit: true},
	{Name: "state-level-ondone", Tool: "lint", Mutation: "compound onDone -> ghost",
		ExpectSubstr: "dangling target 'NoSuchState'", ExpectExit: true},
	{Name: "dead-end-leaf", Tool: "lint", Mutation: "Parked with no transitions",
		ExpectSubstr: "dead-end non-final leaf state Parked", ExpectExit: true},
	{Name: "invoke-no-onerror", Tool: "lint", Mutation: "drop onError",
		ExpectSubstr: "has no onError", ExpectExit: true},
	{Name: "invoke-no-after", Tool: "lint", Mutation: "drop after",
		ExpectSubstr: "no after/timeout", ExpectExit: true},
	{Name: "final-with-transitions", Tool: "lint", Mutation: "final declares on",
		ExpectSubstr: "final state Published declares transitions", ExpectExit: true},
	{Name: "compound-no-initial", Tool: "lint", Mutation: "Wrapper without initial",
		ExpectSubstr: "compound state Wrapper has no initial", ExpectExit: true},
	{Name: "shadowed-branch", Tool: "lint", Mutation: "unguarded branch not last",
		ExpectSubstr: "unreachable", ExpectExit: true},
	{Name: "guarded-always-no-escape", Tool: "lint", Mutation: "fully guarded always",
		ExpectSubstr: "fully guarded always-list", ExpectExit: true},
	{Name: "ambiguous-target", Tool: "lint", Mutation: "two states named Dup",
		ExpectSubstr: "ambiguous target", ExpectExit: true},
	{Name: "bad-initial", Tool: "lint", Mutation: "initial Nowhere",
		ExpectSubstr: "initial 'Nowhere'", ExpectExit: true},
	{Name: "resting-missing-event", Tool: "lint", Mutation: "Parked ignores publish",
		ExpectSubstr: "neither handles nor explicitly ignores event 'publish'", ExpectExit: true},
	{Name: "both-handles-ignores", Tool: "lint", Mutation: "publish in on and _ignores",
		ExpectSubstr: "both handles and ignores event 'publish'", ExpectExit: true},
}

// MachineryCheckExperiments are the gate-suite review findings.
var MachineryCheckExperiments = []Experiment{
	{Name: "empty-design", Tool: "check", Mutation: "nearly-empty design",
		ExpectSubstr: "does not exist", ExpectExit: true},
	{Name: "deleted-mitigation-table", Tool: "check", Mutation: "drop mitigation table",
		ExpectSubstr: "no mitigation", ExpectExit: true},
	{Name: "stale-oracle", Tool: "check", Mutation: "edit a committed oracle",
		ExpectSubstr: "stale", ExpectExit: true},
	{Name: "missing-oracle", Tool: "check", Mutation: "delete committed oracle",
		ExpectSubstr: "no committed oracle", ExpectExit: true},
	{Name: "retargeted-transition-drift", Tool: "check", Mutation: "retarget onDone",
		ExpectSubstr: "no matrix row", ExpectExit: true},
	{Name: "unit-without-namedunit-row", Tool: "check", Mutation: "rename a guard",
		ExpectSubstr: "has no named-unit contract row", ExpectExit: true},
	{Name: "unenforced-invariant", Tool: "check", Mutation: "drop invariant reference",
		ExpectSubstr: "enforced by nothing", ExpectExit: true},
	{Name: "invariant-not-whole-token", Tool: "check", Mutation: "widget-owned-by-nobody",
		ExpectSubstr: "is referenced by no", ExpectExit: true},
	{Name: "machine-state-not-in-enum", Tool: "check", Mutation: "Archived not in enum",
		ExpectSubstr: "'Archived' is not a value of enum", ExpectExit: true},
	{Name: "enum-value-without-state", Tool: "check", Mutation: "Retired enum value",
		ExpectSubstr: "'Retired' has no machine state", ExpectExit: true},
	{Name: "machine-event-not-action", Tool: "check", Mutation: "mysteryEvent",
		ExpectSubstr: "'mysteryEvent' is not a Modelith action", ExpectExit: true},
	{Name: "unmapped-machine", Tool: "check", Mutation: "Gadget with no entity",
		ExpectSubstr: "maps to no Modelith entity", ExpectExit: true},
	{Name: "placement-row-no-machine", Tool: "check", Mutation: "Gizmo placement",
		ExpectSubstr: "`Gizmo` has no machine", ExpectExit: true},
	{Name: "single-form-import-bypass", Tool: "check", Mutation: "import dbdriver",
		ExpectSubstr: "widget.app -> external.db is denied", ExpectExit: true},
	{Name: "undeclared-cross-boundary", Tool: "check", Mutation: "store imports app",
		ExpectSubstr: "undeclared cross-boundary edge widget.store -> widget.app", ExpectExit: true},
	{Name: "import-unexposed-internals", Tool: "check", Mutation: "import store/inner",
		ExpectSubstr: "not in the exposes list of widget.store", ExpectExit: true},
	{Name: "source-outside-contract", Tool: "check", Mutation: "rogue package",
		ExpectSubstr: "maps to no contract boundary", ExpectExit: true},
}

// RefineExperiments are the data-refinement reconciliation failures.
var RefineExperiments = []Experiment{
	{Name: "lifecycle-stage-drift", Tool: "refine", Mutation: "drop a stage",
		ExpectSubstr: "domain states disagree", ExpectExit: true},
	{Name: "lifecycle-machine-edit", Tool: "refine", Mutation: "Won advances",
		ExpectSubstr: "must reject 'advanceStage'", ExpectExit: true},
	{Name: "lifecycle-stale-rollback", Tool: "refine", Mutation: "drop rollback route",
		ExpectSubstr: "rollback routing", ExpectExit: true},
	{Name: "lifecycle-missing-event-name", Tool: "refine", Mutation: "drop advance_event",
		ExpectSubstr: "advance_event", ExpectExit: true},
	{Name: "saga-step-order-drift", Tool: "refine", Mutation: "reorder saga steps",
		ExpectSubstr: "onDone", ExpectExit: true},
	{Name: "saga-failure-route-drift", Tool: "refine", Mutation: "Paying fails clean",
		ExpectSubstr: "failure paths", ExpectExit: true},
	{Name: "saga-missing-undo", Tool: "refine", Mutation: "drop Paying undo",
		ExpectSubstr: "compensating", ExpectExit: true},
	{Name: "terminal-phase-order-drift", Tool: "refine", Mutation: "swap phase order",
		ExpectSubstr: "onDone", ExpectExit: true},
	{Name: "terminal-unserved-phase-retry", Tool: "refine", Mutation: "Optimizing -> collectRetry",
		ExpectSubstr: "failure paths", ExpectExit: true},
}

// ComposeExperiments are the composition validation failures.
var ComposeExperiments = []Experiment{
	{Name: "compose-step-order-drift", Tool: "compose", Mutation: "swap sequence",
		ExpectSubstr: "forward chain", ExpectExit: true},
	{Name: "compose-coordinator-edit", Tool: "compose", Mutation: "reroute failure",
		ExpectSubstr: "failure paths", ExpectExit: true},
	{Name: "compose-missing-undo", Tool: "compose", Mutation: "drop undo",
		ExpectSubstr: "undo", ExpectExit: true},
}

// All returns every experiment across all tools.
func All() []Experiment {
	return concat(MachineLintExperiments, MachineryCheckExperiments, RefineExperiments, ComposeExperiments)
}

func concat(parts ...[]Experiment) []Experiment {
	var out []Experiment
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
