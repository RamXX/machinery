// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// elixir-exunit/v1 adapter (MAC-8yai): it embeds the byte-pinned
// machinery-check/v1 assertion transport, the embedded Machinery ExUnit
// reporter, the bounded JSON codec, the frozen harness Mix project
// definition and the frozen bootstrap from the owned assets directory,
// prepares captured Elixir suites (verified bundle materialization, typed
// assertion call-site validation with comment-derived evidence rejected,
// fresh private MIX_BUILD_PATH, offline closed environment, frozen mix
// argv) and executes them through actual Mix compilation and ExUnit
// execution under processscope custody, normalizing native events to
// machinery.tdd.event/v1.
package adapters

import (
	"context"
	_ "embed"
	"errors"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

//go:embed assets/elixir/mix.exs
var elixirMixSource []byte

//go:embed assets/elixir/machinery/json.exs
var elixirJSONSource []byte

//go:embed assets/elixir/machinery/reporter.exs
var elixirReporterSource []byte

//go:embed assets/elixir/machinery/check.exs
var elixirCheckSource []byte

//go:embed assets/elixir/test/test_helper.exs
var elixirBootstrapSource []byte

// Closed constants of the elixir-exunit/v1 adapter.
const (
	// AdapterElixirExunit is the closed adapter identity.
	AdapterElixirExunit = protocol.AdapterElixirExunit

	// ElixirEventSchema is the normalized event identity of every
	// emitted event.
	ElixirEventSchema = "machinery.tdd.event/v1"

	// ElixirReporterSentinel is the terminal embedded-reporter marker;
	// its absence proves a truncated stream.
	ElixirReporterSentinel = "machinery:reporter:end"

	// ElixirReporterModule is the sole permitted effective formatter.
	ElixirReporterModule = "Elixir.Machinery.Assurance.Reporter"

	// The pinned exact sha256 of every embedded asset; each pin is
	// verified at materialization and re-verified after every native run.
	ElixirMixPinnedSHA256       = "b7683871d74b1f1824bd6ada217306f5c923fd713620a40c617e1f41f3ba8af0"
	ElixirJSONPinnedSHA256      = "49005258cac1f89e365c06da4748c4325dd6012eb28c4e5a8d3dbf77ab51d75e"
	ElixirReporterPinnedSHA256  = "6562724bc4f47891ba42a2f4efe4446939e8ad29fcb722882097cc8f514155cd"
	ElixirCheckPinnedSHA256     = "a4a7890e05477bc4389a878f2eef7c1218897d1665ce34c52249418811eea061"
	ElixirBootstrapPinnedSHA256 = "440a650f545d84838491ba617d30d587aff46e621eb4b8c111a824af5ab7f064"

	// ElixirConformanceFixtureSHA256 pins the frozen native conformance
	// fixture bytes executed by the required contributor lane fragment
	// testdata/integration-lanes/assurance-elixir.json.
	ElixirConformanceFixtureSHA256 = "7f1b0e6a6e48fee9d9c93b80d4bca7cbf05f371798c2c60dbe7bdc66a693abfe"

	// ElixirSuiteSeed and ElixirSuiteMaxCases are the explicit effective
	// ExUnit options of every frozen invocation; reconciliation rejects
	// any drift the embedded reporter reports.
	ElixirSuiteSeed     = int64(1)
	ElixirSuiteMaxCases = int64(2)
)

// errElixirAdapterPending is the RED placeholder for the pending GREEN
// implementation.
var errElixirAdapterPending = errors.New("UNSUPPORTED_FEATURE: the elixir-exunit/v1 native adapter is not implemented yet")

// ElixirAdapter is the closed elixir-exunit/v1 Adapter.
type ElixirAdapter struct{ pending struct{} }

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*ElixirAdapter)(nil)

// Elixir returns the elixir-exunit/v1 adapter.
func Elixir() *ElixirAdapter { return &ElixirAdapter{} }

// ID implements tdd.Adapter.
func (a *ElixirAdapter) ID() string { return AdapterElixirExunit }

// ElixirAssets returns the frozen asset inventory: embedded asset name to
// its pinned exact sha256.
func ElixirAssets() map[string]string {
	return map[string]string{
		"mix.exs":                ElixirMixPinnedSHA256,
		"machinery/json.exs":     ElixirJSONPinnedSHA256,
		"machinery/reporter.exs": ElixirReporterPinnedSHA256,
		"machinery/check.exs":    ElixirCheckPinnedSHA256,
		"test/test_helper.exs":   ElixirBootstrapPinnedSHA256,
	}
}

// ElixirHelperSource returns a copy of the embedded byte-pinned assertion
// transport.
func ElixirHelperSource() []byte { return append([]byte(nil), elixirCheckSource...) }

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6) runtime
// closure under the live scope, materializes and re-verifies the captured
// source bundle plus every embedded pinned asset into a fresh private Mix
// harness, validates each registered assertion as a typed helper call at its
// exact frozen line (comment-derived call sites are rejected), and freezes
// the exact closed argv and environment.
func (a *ElixirAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	return tdd.PreparedSuite{}, errElixirAdapterPending
}

// Run implements tdd.Adapter: it executes the frozen argv of the prepared
// harness as one guarded custody job through actual Mix compilation and
// ExUnit execution, normalizes the embedded reporter stream to the closed
// machinery.tdd.event/v1 sequence, reconciles the complete module/test
// lifecycle against the frozen inventory with the registered assertion
// witnesses and effective ExUnit options, enforces exit concordance and
// re-verifies the prepared sources (late mutation).
func (a *ElixirAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	return tdd.Execution{}, errElixirAdapterPending
}

// elixirEffectiveConfig is the closed effective-option record the embedded
// reporter reports from the runtime's own suite_started cast.
type elixirEffectiveConfig struct {
	Seed               int64
	MaxCases           int64
	Trace              bool
	Timeout            int64
	MaxFailures        string
	Include            []string
	Exclude            []string
	Formatters         []string
	DryRun             bool
	RepeatUntilFailure int64
}

// elixirTestIdentity is one native module/describe/name/file/line identity.
type elixirTestIdentity struct {
	Module   string
	Describe string
	Name     string
	File     string
	Line     int64
	Async    bool
}

// elixirWitness is one parsed machinery-check/v1 witness line.
type elixirWitness struct {
	Module   string
	Describe string
	Name     string
	ID       string
	File     string
	Line     int64
	Value    bool
	Verdict  string
}

// elixirSummary is the final native accounting the reporter accumulated
// from the complete stream.
type elixirSummary struct {
	Tests    int64
	Failures int64
	Skipped  int64
	Invalid  int64
	Excluded int64
}

// elixirEvent is one decoded embedded-reporter event of the closed
// vocabulary.
type elixirEvent struct {
	Type       string
	Config     *elixirEffectiveConfig
	Module     string
	ModuleDone string
	Test       *elixirTestIdentity
	State      string
	Class      string
	Message    string
	Witness    *elixirWitness
	Summary    *elixirSummary
}

// elixirReconciliation is the reconciled normalized result of one native
// reporter stream.
type elixirReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
	Failed     []tdd.NativeID
}

// parseElixirReporterStream decodes the closed embedded-reporter JSONL
// vocabulary; unknown event identities, malformed lines, unknown keys and
// missing, duplicated or non-terminal sentinels fail closed.
func parseElixirReporterStream(raw []byte) ([]elixirEvent, error) {
	return nil, errElixirAdapterPending
}

// reconcileElixirStream reconciles the native lifecycle against the
// complete frozen inventory: exactly-once module/test lifecycle per
// declared identity, witness-bound assertion outcomes at the exact
// registered frozen call sites with pid-verdict enforcement, effective
// ExUnit option enforcement, skip/exclude/invalid rejection, AssertionError
// causality for every expected failure and final summary concordance.
func reconcileElixirStream(events []elixirEvent, suite *tdd.Suite, expect elixirEffectiveConfig) (elixirReconciliation, error) {
	return elixirReconciliation{}, errElixirAdapterPending
}

// validateElixirSuite validates the closed suite shape: elixir-native
// identities bound to declared test files, the frozen ExUnit.Case grammar,
// and typed helper call sites at the exact frozen lines with
// comment-derived evidence rejected.
func validateElixirSuite(suite *tdd.Suite, sources map[string][]byte) error {
	return errElixirAdapterPending
}

// buildElixirRunEnv builds the closed native environment: the pinned
// Elixir/OTP binaries on PATH, private HOME/TMPDIR/MIX_HOME/MIX_BUILD_PATH,
// the private events channel, UTC, no color, declared suite variables only;
// loader injection, custody transport variables and Mix configuration
// overrides are never inherited.
func buildElixirRunEnv(scratch, elixirBinDir, erlangBinDir, eventsPath string, extra []tdd.EnvironmentVar) ([]string, error) {
	return nil, errElixirAdapterPending
}

// elixirExpectedConfig returns the frozen effective-option record of this
// adapter's closed invocation.
func elixirExpectedConfig() elixirEffectiveConfig {
	return elixirEffectiveConfig{
		Seed: ElixirSuiteSeed, MaxCases: ElixirSuiteMaxCases, Trace: false,
		Timeout: 60000, MaxFailures: "infinity", Include: []string{}, Exclude: []string{},
		Formatters: []string{ElixirReporterModule}, DryRun: false, RepeatUntilFailure: 0,
	}
}
