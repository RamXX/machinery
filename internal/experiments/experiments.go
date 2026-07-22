// Package experiments contains the shared, language-neutral mutation
// experiments from the adversarial design reviews. They encode every
// vacuity/drift finding as data; the runners in this package's tests apply
// each mutation to a fixture design and assert the tool catches it
// (experiments_test.go for the lint table, gatesuite_test.go for the full
// gate suite on a synthesized design+impl fixture). Review findings convert
// 1:1 into entries here; do not remove or weaken an entry to make a change
// pass.
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
	{Name: "ambiguous-target", Tool: "lint", Mutation: "two nested states named Dup; a bare target that the removed simple-name fallback would have matched ambiguously must fail sibling-scoped resolution instead of guessing",
		ExpectSubstr: "no sibling of A is named 'Dup'", ExpectExit: true},
	{Name: "bad-initial", Tool: "lint", Mutation: "initial Nowhere",
		ExpectSubstr: "initial 'Nowhere'", ExpectExit: true},
	{Name: "resting-missing-event", Tool: "lint", Mutation: "Parked ignores publish",
		ExpectSubstr: "neither handles nor explicitly ignores event 'publish'", ExpectExit: true},
	{Name: "both-handles-ignores", Tool: "lint", Mutation: "publish in on and _ignores",
		ExpectSubstr: "both handles and ignores event 'publish'", ExpectExit: true},
	{Name: "kebab-case-unit-name", Tool: "lint", Mutation: "rename a guard to kebab-case",
		ExpectSubstr: "is not a valid identifier", ExpectExit: true},
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

// PackExperiments are the recursive-decomposition (G5-pack) failure modes,
// run against the checkout-split fixture in packsuite_test.go.
var PackExperiments = []Experiment{
	{Name: "edited-pack-fails-hash", Tool: "check", Mutation: "hand-edit a pack file",
		ExpectSubstr: "fails its own content hash", ExpectExit: true},
	{Name: "stale-child-pin", Tool: "check", Mutation: "change a contract machine at the parent",
		ExpectSubstr: "was built against pack", ExpectExit: true},
	{Name: "dropped-consumed-event", Tool: "check", Mutation: "child stops handling a boundary event",
		ExpectSubstr: "is handled or ignored by no machine", ExpectExit: true},
	{Name: "dropped-produced-event", Tool: "check", Mutation: "child stops emitting a boundary event",
		ExpectSubstr: "appears in no machine action", ExpectExit: true},
	{Name: "frozen-enum-drift", Tool: "check", Mutation: "child renames a public enum value",
		ExpectSubstr: "drifted from the pack", ExpectExit: true},
	{Name: "delegated-invariant-untraced", Tool: "check", Mutation: "child drops the delegated invariant",
		ExpectSubstr: "delegated invariant", ExpectExit: true},
	{Name: "partial-packmap", Tool: "pack", Mutation: "drop a state from the map",
		ExpectSubstr: "no mapping entry", ExpectExit: true},
	{Name: "stale-refinement-artifact", Tool: "check", Mutation: "hand-edit the committed refinement module",
		ExpectSubstr: "stale", ExpectExit: true},
	{Name: "double-ownership", Tool: "pack", Mutation: "two subsystems own one entity",
		ExpectSubstr: "ownership must be exactly-once", ExpectExit: true},
	{Name: "unowned-entity", Tool: "pack", Mutation: "an entity with no owner",
		ExpectSubstr: "owned by no subsystem", ExpectExit: true},
	{Name: "contract-outside-subset", Tool: "pack", Mutation: "contract machine uses after:",
		ExpectSubstr: "restricted to plain on-transitions", ExpectExit: true},
	// 2026-07-11 H2 dogfood review: lossy event extraction. Cells the exact
	// full-cell match could not resolve were silently dropped, shipping packs
	// with near-empty events.md files that still claimed boundary
	// completeness; G5 (freshness-only) could not see it.
	{Name: "lossy-event-cell", Tool: "pack", Mutation: "a consumer cell names two components",
		ExpectSubstr: "names more than one component", ExpectExit: true},
	{Name: "unknown-event-participant", Tool: "pack", Mutation: "a producer cell names an undeclared component",
		ExpectSubstr: "is not a known component", ExpectExit: true},
	{Name: "zero-boundary-events", Tool: "pack", Mutation: "no table row names a subsystem",
		ExpectSubstr: "extracts zero boundary events", ExpectExit: true},
	{Name: "lossy-table-fails-g5", Tool: "check", Mutation: "G5 regenerates in memory over a lossy table",
		ExpectSubstr: "names more than one component", ExpectExit: true},
}

// PackReviewExperiments are the 2026-07-21 adversarial-review findings on the
// recursive-decomposition mechanism (task MAC-sadm), run against a fixture
// regenerated and re-pinned with the CURRENT generator in
// pack_review_experiments_test.go. Entries with ExpectExit false guard a
// positive property the review found silently violated (rows merged, ids
// round-tripped) rather than an error message.
var PackReviewExperiments = []Experiment{
	{Name: "second-event-table-dropped", Tool: "pack", Mutation: "append a second event-contract table; every table's rows must merge into the packs",
		ExpectSubstr: "| refundRequested | consumes | orders |", ExpectExit: false},
	{Name: "duplicate-component-direction-flip", Tool: "pack", Mutation: "two subsystems claim one component (direction flips in the duplicating pack)",
		ExpectSubstr: "claimed by both", ExpectExit: true},
	{Name: "undelegated-invariant", Tool: "pack", Mutation: "a top-level invariant delegated to no subsystem and not retained",
		ExpectSubstr: "delegated to no subsystem", ExpectExit: true},
	{Name: "child-drops-owned-attribute", Tool: "check", Mutation: "child deletes a frozen pack attribute",
		ExpectSubstr: "is missing from the child domain model", ExpectExit: true},
	{Name: "child-weakens-entity-invariant", Tool: "check", Mutation: "child rewrites a frozen entity invariant's statement",
		ExpectSubstr: "drifted from the pack", ExpectExit: true},
	{Name: "stale-pack-dir-after-rename", Tool: "check", Mutation: "packs/<id>.pack survives a subsystem rename",
		ExpectSubstr: "no current subsystem", ExpectExit: true},
	{Name: "orphaned-child-green", Tool: "check", Mutation: "the renamed subsystem's child keeps validating its dead pack; only the parent can flag the orphan",
		ExpectSubstr: "packs/payments.pack", ExpectExit: true},
	{Name: "hash-laundering-window", Tool: "check", Mutation: "child edits its pack and recomputes the content hash; the parent pin catches it (unpinned children are the documented limit)",
		ExpectSubstr: "was built against pack", ExpectExit: true},
	{Name: "packmap-shim-machine", Tool: "pack", Mutation: "packmap binds a machine with no stake in the contract",
		ExpectSubstr: "neither the lifecycle machine", ExpectExit: true},
	{Name: "stale-packrefinement-extras", Tool: "check", Mutation: "a *PackRefinement.* artifact a fresh generation does not produce stays committed",
		ExpectSubstr: "a refinement artifact a fresh generation does not produce", ExpectExit: true},
	{Name: "pack-subdirectory-smuggling", Tool: "check", Mutation: "a subdirectory smuggled into the frozen child pack/ escapes the content hash",
		ExpectSubstr: "contains a directory", ExpectExit: true},
	{Name: "waiver-count-injection", Tool: "pack", Mutation: "a newline in a boundary_events waiver reason forges the events.md count line",
		ExpectSubstr: "single line", ExpectExit: true},
	{Name: "colon-id-roundtrip", Tool: "check", Mutation: "a delegated invariant id containing ': ' must survive the manifest round-trip; a non-string entry is an error, never a silent drop",
		ExpectSubstr: "not a plain string", ExpectExit: true},
	{Name: "child_design-nonexistent-dir", Tool: "check", Mutation: "child_design points at a directory that does not exist",
		ExpectSubstr: "no readable packmap.yaml", ExpectExit: true},
}

// All returns every experiment across all tools.
func All() []Experiment {
	return concat(MachineLintExperiments, MachineryCheckExperiments, RefineExperiments, ComposeExperiments, PackExperiments, PackReviewExperiments)
}

func concat(parts ...[]Experiment) []Experiment {
	var out []Experiment
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
