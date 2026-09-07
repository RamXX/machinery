// Package tdd owns the closed executable-assurance schemas, obligation
// reconciliation and declaration validation of docs/test-assurance-contract.md.
// It MUST NOT import internal/gates; gates derives authoritative obligations
// and compares through this package. The section-7 request/response records
// below carry already-held snapshots, manifests and scopes; this story
// implements the declaration producer (load + validate) only — capture,
// execution and evidence issuance are downstream consumers.
package tdd

import (
	"context"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// Closed enum constants re-exported from protocol as the approved API.
const (
	SchemaPlan      = protocol.SchemaPlan
	SchemaMilestone = protocol.SchemaMilestone

	KindOracleRow   = protocol.KindOracleRow
	KindGuardClause = protocol.KindGuardClause
	KindInvariant   = protocol.KindInvariant
	KindRuntime     = protocol.KindRuntime

	DispositionTest          = protocol.DispositionTest
	DispositionNotApplicable = protocol.DispositionNotApplicable
	DispositionUnverified    = protocol.DispositionUnverified

	CategoryConcurrency      = protocol.CategoryConcurrency
	CategoryDeliveryReplay   = protocol.CategoryDeliveryReplay
	CategoryMigration        = protocol.CategoryMigration
	CategoryRestore          = protocol.CategoryRestore
	CategoryLoad             = protocol.CategoryLoad
	CategorySecurityBoundary = protocol.CategorySecurityBoundary
	CategoryObservability    = protocol.CategoryObservability

	RolePositive   = protocol.RolePositive
	RoleNegative   = protocol.RoleNegative
	RoleControl    = protocol.RoleControl
	RoleRegression = protocol.RoleRegression

	VariantSafeControl     = protocol.VariantSafeControl
	VariantUnsafeChallenge = protocol.VariantUnsafeChallenge
	OutcomePass            = protocol.OutcomePass
	OutcomeAssertionFail   = protocol.OutcomeAssertionFail

	CheckKindDesign       = protocol.CheckKindDesign
	CheckKindArchitecture = protocol.CheckKindArchitecture
	CheckKindTypecheck    = protocol.CheckKindTypecheck
	CheckKindFormat       = protocol.CheckKindFormat
	CheckKindLint         = protocol.CheckKindLint

	AssertionHelperV1 = protocol.AssertionHelperV1
)

// Review is the freshness-bound judgment record. subject_digest binds the
// entire review projection C of the control and the authoritative design
// payload; it is not an authenticated signature.
type Review struct {
	Reviewer      string
	Rationale     string
	SubjectDigest string
}

// TestRef is the full four-part qualified test identity: design root,
// milestone, suite, test. M1/unit/T and M2/unit/T are DISTINCT tests.
type TestRef struct {
	Design    string
	Milestone string
	Suite     string
	Test      string
}

// SourceRef binds a runtime obligation to exact reviewed source bytes.
type SourceRef struct {
	Path   string
	Anchor string
	Digest string
}

// RuntimeObligation is one declared runtime/NFR requirement row.
type RuntimeObligation struct {
	ID          string
	Category    string
	Owner       string
	SourceRefs  []SourceRef
	Disposition string
	Reason      string
	Review      Review
	Tests       []TestRef
}

// PlanMilestone declares one current BUILD milestone and its manifest path.
type PlanMilestone struct {
	ID       string
	Manifest string
}

// Plan is the closed design/assurance/plan.json record. The unexported
// load-derived fields carry the exact bytes, the bound payload digest and
// the positional design directory; they are not authored fields.
type Plan struct {
	Schema             string
	Design             string
	Milestones         []PlanMilestone
	ProjectID          string
	RuntimeObligations []RuntimeObligation

	raw           []byte
	payloadDigest string
	designDir     string
}

// Raw returns the exact authored plan bytes the reviews were verified
// against; nil for a hand-built record.
func (p Plan) Raw() []byte { return p.raw }

// Assertion is one registered native assertion call site.
type Assertion struct {
	ID     string
	Source string
	Line   int64
	Helper string
}

// NativeID is the closed native-identity union; the selected adapter
// discriminates it and no arbitrary keys are accepted. Exactly the fields
// of the matching adapter family may be set.
type NativeID struct {
	Package string   // go
	Test    string   // go
	Source  string   // node
	Path    []string // node
	Line    int64    // node, elixir
	Column  int64    // node
	Module  string   // python, elixir
	Class   string   // python
	Method  string   // python
	Name    string   // elixir
	File    string   // elixir
}

// Test is one declared leaf test with its bound assertions. Every leaf has
// at least one assertion; aggregate parents are inventory nodes, not
// substitute leaf coverage.
type Test struct {
	ID         string
	Native     NativeID
	Source     string
	Role       string
	Assertions []Assertion
}

// EnvironmentVar is one explicitly declared native environment input.
type EnvironmentVar struct {
	Name  string
	Value string
}

// RuntimeRef names the pinned runtime closure the suite executes under.
type RuntimeRef struct {
	Profile  string
	Version  string
	Platform string
	Closure  string
}

// Suite is one adapter-executed test suite.
type Suite struct {
	ID              string
	Adapter         string
	Runtime         RuntimeRef
	Root            string
	Files           []string
	Tests           []Test
	Environment     []EnvironmentVar
	DependencyRoots []string
}

// ObligationKey is the qualified obligation identity. design and owner are
// canonical relative paths or declared entity IDs appropriate to the kind;
// obligations never pool merely because a short id or name matches.
type ObligationKey struct {
	Design string
	Kind   string
	Owner  string
	ID     string
}

// Obligation is one manifest-declared obligation mapping.
type Obligation struct {
	Key      ObligationKey
	Positive []TestRef
	Negative []TestRef
}

// Expectation is one expected per-test outcome of a baseline or variant run.
type Expectation struct {
	Test       TestRef
	Outcome    string
	Assertions []string
}

// Variant is one retained safe-control or unsafe-challenge source state.
type Variant struct {
	ID          string
	Kind        string
	Source      string // Ref; "" encodes the schema's null (draft)
	Pair        string
	TargetTests []TestRef
	Expected    []Expectation
	Review      Review
}

// Check is one declared non-test precondition.
type Check struct {
	ID      string
	Kind    string
	Profile string
	Inputs  []string
}

// RedControl maps one full (TestRef, failing assertion ID) exactly once to
// the safe-control variant where the same test and assertion pass under the
// identical frozen closure.
type RedControl struct {
	Test        TestRef
	Assertion   string
	SafeVariant string
}

// Limits is the closed per-milestone budget record: ONE cumulative wall
// deadline and one cleanup bound, never per-command resets.
type Limits struct {
	WallMS      int64
	CleanupMS   int64
	StdoutBytes int64
	StderrBytes int64
	EventBytes  int64
	EventCount  int64
	Jobs        int64
	BundleBytes int64
	Entries     int64
	Depth       int64
}

// SubjectEntry is one mutable subject path of the milestone.
type SubjectEntry struct {
	Path string
	Kind string
}

// Manifest is the closed design/assurance/milestones/<id>.json record. The
// unexported load-derived fields carry the exact bytes, the bound payload
// digest, and the positional design identity/dir/path.
type Manifest struct {
	Schema              string
	ID                  string
	Revision            int64
	Predecessor         string // Digest; "" encodes null
	Repository          string
	ImplementationRoots []string
	FrozenRoots         []string
	SubjectEntries      []SubjectEntry
	Suites              []Suite
	Obligations         []Obligation
	Baseline            string // Ref; "" encodes null (draft)
	Variants            []Variant
	RedExpectations     []Expectation
	Checks              []Check
	RedControls         []RedControl
	Limits              Limits
	Review              Review

	raw           []byte
	payloadDigest string
	designID      string
	designDir     string
	relPath       string
}

// Raw returns the exact authored manifest bytes; nil for a hand-built record.
func (m Manifest) Raw() []byte { return m.raw }

// MilestoneKey identifies one current milestone of one design.
type MilestoneKey struct {
	Design    string
	Milestone string
}

// InventoryObligation is one authoritative inventory row.
type InventoryObligation struct {
	Key ObligationKey
}

// Inventory is the authoritative obligation inventory derived from a held
// immutable design snapshot.
type Inventory struct {
	Obligations []InventoryObligation
	Milestones  []MilestoneKey
	Digest      string
}

// Diagnostic is one machine diagnostic of a run or check result.
type Diagnostic struct {
	Code    string
	Subject string
	Message string
}

// AssertionOutcome is one observed assertion execution result.
type AssertionOutcome struct {
	Test    TestRef
	ID      string
	Outcome string
}

// ExecutionEntry is one normalized suite execution of a run record.
type ExecutionEntry struct {
	Source     string
	Suite      string
	Stream     string
	Discovered []NativeID
	Started    []NativeID
	Completed  []NativeID
	Assertions []AssertionOutcome
	Outcome    string
	ExitCode   *int64
}

// CustodyReport is the independent custody result of a run.
type CustodyReport struct {
	Status     string
	Transcript string
}

// RunRecord is the closed append-only evidence revision record. This story
// only declares the schema; evidence issuance is downstream.
type RunRecord struct {
	Schema                 string
	RunID                  string
	Design                 string
	Milestone              string
	ManifestDigest         string
	JudgmentControlDigest  string
	ControlInventoryDigest string
	InventoryDigest        string
	FrozenDigest           string
	SubjectDigest          string
	RuntimeDigest          string
	Phase                  string
	Result                 string
	Executions             []ExecutionEntry
	Custody                CustodyReport
	Provenance             string
	Diagnostics            []Diagnostic
}

// Event is one normalized adapter event of the closed event contract.
type Event struct {
	Schema    string
	Sequence  int64
	Design    string
	Milestone string
	Suite     string
	Kind      string
	Native    *NativeID
	Assertion string
	Outcome   string
	Source    string
	Line      int64
}

// InputView is one held immutable input materialization. Constructors reject
// a missing validator/release callback or an unverified source; Release is
// the owner-controlled idempotent FINAL release.
type InputView struct {
	SourceRoot          string   // immutable, topology-preserving source materialization
	DesignPath          string   // relative to SourceRoot; payload excludes the reserved control plane
	ImplementationPaths []string // relative to SourceRoot
	ControlRoot         string   // separate immutable control materialization
	Revalidate          func() error
	Release             func() error
}

// BundleRef is a validated content-addressed Ref plus its verified immutable
// materialization handle; it cannot be made valid by deserializing a path.
type BundleRef struct {
	ref   string
	root  string
	valid bool
}

// Ref returns the content-addressed bundle id.
func (b BundleRef) Ref() string { return b.ref }

// Materialized returns the verified immutable materialization root (empty
// when the handle was not produced by capture).
func (b BundleRef) Materialized() string { return b.root }

// RuntimeHandle holds a pinned runtime closure's identity and its
// Validate/Close custody operations.
type RuntimeHandle interface {
	Identity() RuntimeRef
	Validate(context.Context, processscope.Scope) error
	Close() error
}

// PendingChecks is the closed owner-bound handle to produced check results;
// Final errors until every owning view has completed release. It exposes
// exactly Provisional and Final.
type PendingChecks interface {
	Provisional() ([]CheckResult, error)
	Final() ([]CheckResult, error)
}

// CheckResult is one precondition check outcome. No pending pass is public.
type CheckResult struct {
	InvocationID  string
	StateID       string
	Profile       string
	SourceDigest  string
	ControlDigest string
	Status        string // pass | fail | error
	Diagnostics   []Diagnostic
}

// CheckRequest is the closed production request bridging tdd execution to
// the gates-side check registry inside assuranceflow.
type CheckRequest struct {
	InvocationID  string
	StateID       string
	Inputs        InputView
	Checks        []Check
	SourceDigest  string
	ControlDigest string
	Scope         processscope.Scope
	Runtimes      []RuntimeHandle
}

// CheckExecutor runs declared non-test preconditions over the same held
// snapshots. Implemented only by the registry inside internal/assuranceflow;
// never a project plugin, JSON receipt or user callback.
type CheckExecutor interface {
	Run(context.Context, CheckRequest) (PendingChecks, error)
}

// CaptureRequest captures one validated content-addressed bundle.
type CaptureRequest struct {
	Inputs   InputView
	Manifest Manifest
	Name     string
	Store    string
	Limits   Limits
}

// StatusRequest reports persisted declaration/execution state.
type StatusRequest struct {
	Inputs    InputView
	Plan      Plan
	Manifests []Manifest
	Inventory Inventory
	Store     string
}

// ExecuteRequest executes one milestone phase (red | green).
type ExecuteRequest struct {
	Inputs    InputView
	Plan      Plan
	Manifest  Manifest
	Inventory Inventory
	Phase     string // closed enum red | green
	Store     string
	Runtimes  []RuntimeHandle
	Scope     processscope.Scope
	Checks    CheckExecutor
}

// VerifyRequest independently replays one milestone or the whole design
// (Milestone "" = all; otherwise one exact manifest ID).
type VerifyRequest struct {
	Inputs    InputView
	Plan      Plan
	Manifests []Manifest
	Inventory Inventory
	Milestone string
	Store     string
	Runtimes  []RuntimeHandle
	Scope     processscope.Scope
	Checks    CheckExecutor
}

// SuiteRequest is one adapter Prepare/Run unit over a captured source bundle.
type SuiteRequest struct {
	Inputs  InputView
	Suite   Suite
	Source  BundleRef
	Scratch string
	Runtime RuntimeHandle
	Scope   processscope.Scope
	Limits  Limits
}

// EventSink receives normalized adapter events.
type EventSink func(Event) error

// PreparedSuite is opaque adapter-owned state produced ONLY by successful
// Prepare (exact argv, build outputs and native inventory); it is not a
// deserializable receipt. MAC-wi2u completes the placeholder with an
// unexported state carrier: only an adapter's own concrete state type is
// accepted by its Run, so a foreign or deserialized value carries no
// authority (additive compatible completion of the MAC-6h0s contract; no
// existing field or behavior changed).
type PreparedSuite struct {
	opaque struct{}
	state  any
}

// NewPreparedSuite binds adapter-owned prepared state into the opaque
// carrier. It grants no authority by itself: the producing adapter's Run
// validates the concrete state type it emitted.
func NewPreparedSuite(state any) PreparedSuite { return PreparedSuite{state: state} }

// PreparedState exposes the bound state to the owning adapter only; other
// callers receive nil.
func (p PreparedSuite) PreparedState() any { return p.state }

// Execution contains the normalized event inventory, raw-stream digests,
// target exit status and independent custody result of one suite run.
type Execution struct {
	Suite        string
	Events       []Event
	StdoutDigest string
	StderrDigest string
	Outcome      string
	ExitCode     *int64
	Custody      CustodyReport
}

// Adapter is one closed native adapter. The four first-release adapters are
// declared in protocol.SupportedAdapters.
type Adapter interface {
	ID() string
	Prepare(context.Context, SuiteRequest) (PreparedSuite, error)
	Run(context.Context, PreparedSuite, EventSink) (Execution, error)
}

// StatusReport carries the section-1 dimension values plus store/source
// binding digests and diagnostics. It is never execution evidence.
type StatusReport struct {
	DesignConsistency   string
	SourceTestBinding   string
	TestExecution       string
	NegativeSensitivity string
	Judgment            string
	FormalExecution     string
	RuntimeResiduals    string
	Custody             string
	Provenance          string
	StoreProjectID      string
	StoreHeadDigest     string
	SourceDigest        string
	ControlDigest       string
	JudgmentDigest      string
	Diagnostics         []Diagnostic
}

// Candidate holds provisional outcomes and pending precondition handles of
// one execution; it never carries final success.
type Candidate struct {
	Phase      string
	Milestone  MilestoneKey
	Executions []Execution
	Checks     PendingChecks
}
