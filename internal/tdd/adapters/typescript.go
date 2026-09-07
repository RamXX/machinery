// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// node-test-typescript/v1 adapter (MAC-avfp): it embeds the byte-pinned
// machinery-check/v1 assertion transport, the ambient node type closure, the
// frozen compiler configuration, the frozen module identity and the embedded
// Machinery reporter from the owned assets directory, prepares captured
// TypeScript suites (verified bundle materialization, typed assertion
// call-site validation, separate fresh pinned-compiler build step with
// noEmitOnError, non-incremental clean output, source maps with embedded
// sources whose sourcesContent must equal the frozen bundle bytes) and
// executes only that invocation's compiled outputs through Node's actual
// node:test runner under processscope custody, normalizing native events to
// machinery.tdd.event/v1.
package adapters

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// tsDigestHex is the shared exact-byte digest helper of the asset pins.
func tsDigestHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

//go:embed assets/typescript/machinery-check.ts
var tsHelperSource []byte

//go:embed assets/typescript/node-ambient.d.ts
var tsAmbientSource []byte

//go:embed assets/typescript/tsconfig.json
var tsConfigSource []byte

//go:embed assets/typescript/package.json
var tsPackageSource []byte

//go:embed assets/typescript/reporter.mjs
var tsReporterSource []byte

// Closed constants of the node-test-typescript/v1 adapter.
const (
	// AdapterNodeTestTS is the closed adapter identity.
	AdapterNodeTestTS = protocol.AdapterNodeTestTS

	// TypeScriptEventSchema is the normalized event identity of every
	// emitted event.
	TypeScriptEventSchema = "machinery.tdd.event/v1"

	// TypeScriptWitnessSchema is the closed helper witness identity.
	TypeScriptWitnessSchema = "machinery.tdd.witness/v1"

	// TypeScriptReporterSentinel is the terminal embedded-reporter marker;
	// its absence proves a truncated stream.
	TypeScriptReporterSentinel = "machinery:reporter:end"

	// The pinned exact sha256 of every embedded asset; each pin is verified
	// at materialization and re-verified after every native run.
	TypeScriptHelperPinnedSHA256   = "f44691b8d2c80af24ee1e990ad86dfb8ef4fadc2b520281e334a2106d2d8319f"
	TypeScriptAmbientPinnedSHA256  = "5a76344c274a65e3b1ef32e4990c8390b0e5d246d03d2d1e74970bcad61b5f28"
	TypeScriptConfigPinnedSHA256   = "bd796144d36a5da159b072f4bd731d1d15f3fb94b25aed46d71e432782598257"
	TypeScriptPackagePinnedSHA256  = "256acf4930104c357b78e8d9ea02eb35e5a72948fb0a015bb3534470838912ba"
	TypeScriptReporterPinnedSHA256 = "04fef043d0307ff729886146db95c1188c766eac79a95816dea0e106ccd03d5f"

	// TypeScriptConformanceFixtureSHA256 pins the frozen native conformance
	// fixture bytes executed by the required contributor lane fragment
	// testdata/integration-lanes/assurance-typescript.json.
	TypeScriptConformanceFixtureSHA256 = "5fcd808b1f62d4adcb9ab0d0e246a8c59a64df2caa3dfe13f9f408fddf8e60bc"

	// TypeScriptCompileTimeoutMS bounds the separate pinned build step.
	TypeScriptCompileTimeoutMS = int64(300000)
)

// errTypeScriptAdapterPending is the RED placeholder for the pending GREEN
// implementation.
var errTypeScriptAdapterPending = errors.New("UNSUPPORTED_FEATURE: the node-test-typescript/v1 native adapter is not implemented yet")

// TypeScriptAdapter is the closed node-test-typescript/v1 Adapter.
type TypeScriptAdapter struct{ pending struct{} }

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*TypeScriptAdapter)(nil)

// TypeScript returns the node-test-typescript/v1 adapter.
func TypeScript() *TypeScriptAdapter { return &TypeScriptAdapter{} }

// Lookup resolves one closed adapter identity.
func Lookup(id string) (tdd.Adapter, error) {
	if id != AdapterNodeTestTS {
		return nil, fmt.Errorf("UNSUPPORTED_ADAPTER: %q is not a closed first-release native adapter", id)
	}
	return TypeScript(), nil
}

// ID implements tdd.Adapter.
func (a *TypeScriptAdapter) ID() string { return AdapterNodeTestTS }

// TypeScriptHelperSource returns a copy of the embedded byte-pinned helper
// transport.
func TypeScriptHelperSource() []byte { return append([]byte(nil), tsHelperSource...) }

// TypeScriptAssets returns the frozen asset inventory: embedded asset name
// to its pinned exact sha256.
func TypeScriptAssets() map[string]string {
	return map[string]string{
		"machinery-check.ts": TypeScriptHelperPinnedSHA256,
		"node-ambient.d.ts":  TypeScriptAmbientPinnedSHA256,
		"tsconfig.json":      TypeScriptConfigPinnedSHA256,
		"package.json":       TypeScriptPackagePinnedSHA256,
		"reporter.mjs":       TypeScriptReporterPinnedSHA256,
	}
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned Node 26.8.1 / TypeScript 7.0.2 runtime closure under
// the live scope, materializes and re-verifies the captured source bundle
// plus every embedded pinned asset, validates each registered assertion as a
// typed helper call at its exact frozen line, compiles the suite fresh with
// the pinned native compiler (noEmitOnError, non-incremental clean output,
// source maps with embedded sources verified against the frozen bytes) and
// freezes the exact closed argv and environment.
func (a *TypeScriptAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	return tdd.PreparedSuite{}, errTypeScriptAdapterPending
}

// Run implements tdd.Adapter: it executes the prepared compiled outputs
// through Node's actual node:test runner with the embedded Machinery
// reporter under processscope custody, normalizes the native stream to the
// closed machinery.tdd.event/v1 sequence, reconciles the complete
// dequeue/complete lifecycle against the frozen inventory with
// witness-bound assertion outcomes, enforces exit concordance and re-verifies
// prepared sources against late mutation.
func (a *TypeScriptAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	return tdd.Execution{}, errTypeScriptAdapterPending
}

// tsWitness is one parsed machinery-check/v1 witness diagnostic.
type tsWitness struct {
	assertion string
	site      string
	condition bool
	thrown    string
}

// parseTypeScriptWitness parses one helper witness diagnostic message.
func parseTypeScriptWitness(message string) (tsWitness, bool) {
	return tsWitness{}, false
}

// tsNativeError is the shaped native failure cause of one test event.
type tsNativeError struct {
	Name    string
	Message string
	Code    string
}

// tsSummaryCounts is the shaped final native summary accounting.
type tsSummaryCounts struct {
	Tests     int64
	Failed    int64
	Passed    int64
	Cancelled int64
	Skipped   int64
	Todo      int64
	Success   bool
}

// tsNativeEvent is one decoded embedded-reporter event of the closed
// node:test reporter vocabulary.
type tsNativeEvent struct {
	Type      string
	Nesting   int64
	Name      string
	File      string
	EntryFile string
	Line      int64
	Column    int64
	TestID    int64
	ParentID  int64
	Skip      bool
	Todo      bool
	Message   string
	Error     *tsNativeError
	Counts    *tsSummaryCounts
}

// parseTypeScriptReporterStream decodes the closed embedded-reporter JSONL
// vocabulary; unknown event identities, malformed lines and missing or
// duplicated terminal sentinels fail closed.
func parseTypeScriptReporterStream(raw []byte) ([]tsNativeEvent, error) {
	return nil, errTypeScriptAdapterPending
}

// tsSourceMap is one verified compiler-produced source map whose embedded
// sourcesContent equals the frozen bundle bytes.
type tsSourceMap struct {
	Generation int64
	Source     string
	segments   []tsMapSegment
}

// tsMapSegment is one decoded mapping segment: generated column to original
// line/column in the single frozen source.
type tsMapSegment struct {
	GenLine  int64
	GenCol   int64
	SrcLine  int64
	SrcCol   int64
}

// loadTypeScriptSourceMap decodes and verifies one compiler-produced .js.map:
// the named source's embedded content must equal the frozen source bytes.
func loadTypeScriptSourceMap(mapBytes []byte, sourceName string, frozen []byte) (tsSourceMap, error) {
	return tsSourceMap{}, errTypeScriptAdapterPending
}

// SourcePosition maps one generated position back to the frozen original
// source position.
func (m tsSourceMap) SourcePosition(genLine, genCol int64) (srcLine, srcCol int64, ok bool) {
	return 0, 0, false
}

// tsEntryIndex maps native entry-file identities to suite-relative sources.
type tsEntryIndex map[string]string

// tsReconciliation is the reconciled normalized result of one native stream.
type tsReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
	Failed     []tdd.NativeID
}

// reconcileTypeScriptStream reconciles native lifecycle events against the
// complete frozen inventory: enqueue/dequeue/complete per declared identity,
// skip/todo/cancelled rejection, witness-bound assertion outcomes at the
// exact registered call sites through the verified source maps, and final
// summary concordance.
func reconcileTypeScriptStream(events []tsNativeEvent, suite *tdd.Suite, entries tsEntryIndex, maps map[string]tsSourceMap) (tsReconciliation, error) {
	return tsReconciliation{}, errTypeScriptAdapterPending
}

// buildTypeScriptRunEnv builds the closed native environment: private
// HOME/TMPDIR, UTC, no color, minimal PATH, declared suite variables only;
// loader injection, native test-context and custody transport variables are
// never inherited.
func buildTypeScriptRunEnv(scratch, nodeDir string, extra []tdd.EnvironmentVar) ([]string, error) {
	return nil, errTypeScriptAdapterPending
}

// validateTypeScriptSuite validates the closed suite shape: node-native
// identities bound to declared files, typed helper call sites at the exact
// frozen lines, and the frozen assertion helper identity.
func validateTypeScriptSuite(suite *tdd.Suite, sources map[string][]byte) error {
	return errTypeScriptAdapterPending
}
