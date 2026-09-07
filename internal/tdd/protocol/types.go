// Package protocol holds the closed v1 executable-assurance constants and
// scalar grammar rules from docs/test-assurance-contract.md section 4:
// schema identities, closed enums, exact defaults/caps and the portable
// path/digest/ID grammars. It depends only on the standard library so any
// layer can share one vocabulary authority without import cycles; the typed
// records live in internal/tdd, which re-exports these constants as the
// approved public API surface of section 7.
package protocol

// Schema identity strings. A control document carrying any other schema
// value is not this contract (UNSUPPORTED_VERSION, never best-effort).
const (
	SchemaPlan      = "machinery.tdd.plan/v1"
	SchemaMilestone = "machinery.tdd.milestone/v1"
)

// Control-plane layout: the only allowed contents of the design's reserved
// assurance/ namespace are the authored plan.json and milestones/.
const (
	ControlDirName    = "assurance"
	PlanFileName      = "plan.json"
	MilestonesDirName = "milestones"
	AssertionHelperV1 = "machinery-check/v1"
	RepositoryRoot    = "."
)

// Obligation kinds of the qualified obligation key {design, kind, owner, id}.
const (
	KindOracleRow   = "oracle-row"
	KindGuardClause = "guard-clause"
	KindInvariant   = "invariant"
	KindRuntime     = "runtime"
)

// Runtime/NFR disposition values. "unverified" always blocks strict
// completion; "test" requires mapped tests; "not-applicable" requires a
// nonempty rationale and a bound review.
const (
	DispositionTest          = "test"
	DispositionNotApplicable = "not-applicable"
	DispositionUnverified    = "unverified"
)

// The seven approved runtime/NFR categories. Every one needs an explicit
// disposition; a generic runtime row cannot hide a missing category.
const (
	CategoryConcurrency      = "concurrency"
	CategoryDeliveryReplay   = "delivery-replay"
	CategoryMigration        = "migration"
	CategoryRestore          = "restore"
	CategoryLoad             = "load"
	CategorySecurityBoundary = "security-boundary"
	CategoryObservability    = "observability"
)

// RuntimeCategories is the closed category set in canonical order.
var RuntimeCategories = []string{
	CategoryConcurrency, CategoryDeliveryReplay, CategoryMigration,
	CategoryRestore, CategoryLoad, CategorySecurityBoundary, CategoryObservability,
}

// Test roles within a suite.
const (
	RolePositive   = "positive"
	RoleNegative   = "negative"
	RoleControl    = "control"
	RoleRegression = "regression"
)

// Variant kinds and expectation outcomes.
const (
	VariantSafeControl     = "safe-control"
	VariantUnsafeChallenge = "unsafe-challenge"
	OutcomePass            = "pass"
	OutcomeAssertionFail   = "assertion-fail"
)

// Check kinds (the closed set the check profiles map onto).
const (
	CheckKindDesign       = "design"
	CheckKindArchitecture = "architecture"
	CheckKindTypecheck    = "typecheck"
	CheckKindFormat       = "format"
	CheckKindLint         = "lint"
)

// First-release native adapters. A supported language is not every framework
// in that language; anything outside this set is UNSUPPORTED_ADAPTER.
const (
	AdapterGoTesting      = "go-testing/v1"
	AdapterNodeTestTS     = "node-test-typescript/v1"
	AdapterPythonUnittest = "python-unittest/v1"
	AdapterElixirExunit   = "elixir-exunit/v1"
)

// SupportedAdapters is the closed adapter catalog of the first release.
var SupportedAdapters = map[string]bool{
	AdapterGoTesting: true, AdapterNodeTestTS: true,
	AdapterPythonUnittest: true, AdapterElixirExunit: true,
}

// SupportedPlatforms are the initial native execution platforms; anything
// else fails UNSUPPORTED_PLATFORM rather than falling back to receipts.
var SupportedPlatforms = map[string]bool{
	"darwin/arm64": true, "linux/amd64": true,
}

// CheckProfiles is the closed profile catalog: profile -> required check
// kind. Unknown profiles block; a shell command is not a precondition.
var CheckProfiles = map[string]string{
	"machinery-design/v1":       CheckKindDesign,
	"machinery-architecture/v1": CheckKindArchitecture,
	"go-format/v1":              CheckKindFormat,
	"go-vet/v1":                 CheckKindLint,
	"typescript-typecheck/v1":   CheckKindTypecheck,
	"python-compile/v1":         CheckKindTypecheck,
	"elixir-format/v1":          CheckKindFormat,
	"elixir-compile/v1":         CheckKindTypecheck,
}

// Exact defaults and absolute caps from the contract. Every JSON control
// input is also capped at ControlInputMaxBytes on read/parse; exceeding a
// bound fails, truncation can never support success.
const (
	LimitWallDefaultMS     int64 = 600000
	LimitWallMaxMS         int64 = 3600000
	LimitCleanupDefaultMS  int64 = 10000
	LimitCleanupMaxMS      int64 = 30000
	LimitStdoutDefault     int64 = 4194304
	LimitStderrDefault     int64 = 4194304
	LimitEventBytesDefault int64 = 16777216
	LimitEventCountDefault int64 = 100000
	LimitJobsDefault       int64 = 1
	LimitBundleDefault     int64 = 536870912
	LimitEntriesDefault    int64 = 100000
	LimitDepthDefault      int64 = 64
	LimitStreamMaxBytes    int64 = 67108864 // 64 MiB per output/event stream
	LimitEventMaxCount     int64 = 1000000
	LimitJobsMax           int64 = 4
	LimitBundleMaxBytes    int64 = 4294967296
	LimitEntriesMax        int64 = 1000000
	LimitDepthMax          int64 = 128
	ControlInputMaxBytes   int64 = 16 << 20
)

// Event kinds and outcomes of the normalized event contract, plus the
// execution-phase and provenance literals.
const (
	EventSuiteStart   = "suite-start"
	EventDiscovered   = "discovered"
	EventTestStart    = "test-start"
	EventAssertion    = "assertion"
	EventTestEnd      = "test-end"
	EventSuiteEnd     = "suite-end"
	EventDiagnostic   = "diagnostic"
	EventError        = "error"

	OutcomeError       = "error"
	OutcomeSkipped     = "skipped"
	OutcomeUnsupported = "unsupported"

	PhaseRed    = "red"
	PhaseGreen  = "green"
	PhaseVerify = "verify"

	ProvenanceUnauthenticatedHost = "unauthenticated-host"
)

// Tree and review digest domains of section 4.
const (
	TreeDigestDomain  = "machinery.tdd.tree/v1"
	ReviewDomain      = "machinery.tdd.review/v1"
	InventoryDomain   = "machinery.tdd.inventory/v1"
	DesignPayloadRole = "design"
)
