// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// elixir-exunit/v1 adapter (MAC-8yai): it embeds the byte-pinned
// machinery-check/v1 assertion transport, the embedded Machinery ExUnit
// reporter, the bounded JSON codec, the frozen harness Mix project
// definition and the frozen bootstrap from the owned assets directory,
// prepares captured Elixir suites (verified bundle materialization, typed
// assertion call-site validation with comment-derived evidence rejected,
// fresh private MIX_BUILD_PATH, offline closed environment, frozen mix
// argv with explicit seed and max_cases) and executes them through actual
// Mix compilation under warnings-as-errors and native ExUnit execution
// under processscope custody, normalizing native events to
// machinery.tdd.event/v1.
package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
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

	// ElixirNativeTestPrefix is the native name-atom prefix every ExUnit
	// test function carries; declared names never include it.
	ElixirNativeTestPrefix = "test "

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

	// ElixirNativeTimeoutMS is the closed effective native per-test
	// timeout the reporter must report (the frozen argv never overrides
	// it; per-test tag timeouts stay inside this ceiling).
	ElixirNativeTimeoutMS = int64(60000)

	// elixirEventsFile is the private native event channel name.
	elixirEventsFile = "events.jsonl"

	elixirMaxStreamLines = 100000
	elixirMaxLineLen     = 1 << 20
)

var (
	elixirAssertionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	elixirTestFilePattern    = regexp.MustCompile(`^test/[A-Za-z0-9_]+_test\.exs$`)
	elixirEnvNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	elixirClosedEnvKeys      = map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "LANG": true, "LC_ALL": true,
		"TZ": true, "NO_COLOR": true, "MIX_ENV": true, "MIX_HOME": true,
		"MIX_BUILD_PATH": true, "MACHINERY_ASSURANCE_EVENTS": true,
	}
	// elixirStreamKeys is the closed per-event payload key set.
	elixirStreamKeys = map[string]map[string]bool{
		"suite_started": {
			"seed": true, "max_cases": true, "trace": true, "timeout": true,
			"max_failures": true, "include": true, "exclude": true,
			"formatters": true, "dry_run": true, "repeat_until_failure": true,
		},
		"module_started":  {"module": true},
		"module_finished": {"module": true, "state": true},
		"test_started":    {"module": true, "describe": true, "test": true, "file": true, "line": true, "async": true},
		"test_finished": {
			"module": true, "describe": true, "test": true, "file": true, "line": true,
			"state": true, "class": true, "message": true,
		},
		"witness": {
			"module": true, "describe": true, "test": true, "id": true, "value": true,
			"file": true, "line": true, "verdict": true,
		},
		"suite_finished":       {"tests": true, "failures": true, "skipped": true, "invalid": true, "excluded": true},
		"max_failures_reached": {},
		ElixirReporterSentinel: {},
	}
	// elixirTestStates is the closed native outcome vocabulary.
	elixirTestStates = map[string]bool{
		"passed": true, "failed": true, "skipped": true, "excluded": true, "invalid": true, "unknown": true,
	}
)

// ElixirAdapter is the closed elixir-exunit/v1 Adapter.
type ElixirAdapter struct{}

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*ElixirAdapter)(nil)

// Elixir returns the elixir-exunit/v1 adapter.
func Elixir() *ElixirAdapter { return &ElixirAdapter{} }

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

// ID implements tdd.Adapter.
func (a *ElixirAdapter) ID() string { return AdapterElixirExunit }

// elixirPrepared is the adapter-owned opaque prepared state: the frozen
// suite inventory, the prepared harness layout with its pinned digests,
// the exact closed argv/environment and the pinned runtime members. Only
// this adapter's Run accepts it.
type elixirPrepared struct {
	suite       *tdd.Suite
	inputs      tdd.InputView
	scope       processscope.Scope
	scratch     string
	harnessDir  string
	eventsPath  string
	mixBin      string
	closure     string
	elixirBin   string
	erlangBin   string
	argv        []string
	env         []string
	timeoutMS   int64
	limits      processscope.Limits
	fileDigests map[string]string
	ran         bool
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned Elixir 1.20.4 / OTP 29.0.6 (ERTS 17.0.6) runtime
// closure under the live scope, materializes and re-verifies the captured
// source bundle plus every embedded pinned asset into a fresh private Mix
// harness, validates each registered assertion as a typed helper call at
// its exact frozen line (comment-derived call sites are rejected), and
// freezes the exact closed argv and environment.
func (a *ElixirAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	if req.Suite.Adapter != AdapterElixirExunit {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_ADAPTER: suite %s declares adapter %q, not %s", req.Suite.ID, req.Suite.Adapter, AdapterElixirExunit)
	}
	if req.Scope == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a live processscope scope", req.Suite.ID)
	}
	if req.Runtime == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the pinned Elixir %s runtime handle", req.Suite.ID, runtimeclosure.ElixirIdentityVersion)
	}
	handle, ok := req.Runtime.(*runtimeclosure.Elixir)
	if !ok {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the exact approved Elixir runtime closure handle, got %T", req.Suite.ID, req.Runtime)
	}
	// Reject absent/mismatched/mutated runtime before any suite work.
	if err := handle.Validate(ctx, req.Scope); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s runtime validation failed: %w", req.Suite.ID, err)
	}
	identity := handle.Identity()
	if req.Suite.Runtime != identity {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s declares runtime %+v, but the pinned closure is %+v", req.Suite.ID, req.Suite.Runtime, identity)
	}
	if req.Suite.Root != protocol.RepositoryRoot {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: elixir-exunit/v1 suites are harness-root suites; suite %s declares root %q", req.Suite.ID, req.Suite.Root)
	}
	if len(req.Suite.DependencyRoots) != 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: elixir-exunit/v1 executes the frozen deps-free harness project offline; suite %s declares dependency roots %v", req.Suite.ID, req.Suite.DependencyRoots)
	}
	if len(req.Suite.Files) == 0 || len(req.Suite.Tests) == 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s declares no files or no tests", req.Suite.ID)
	}
	if !filepath.IsAbs(req.Scratch) {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires an absolute scratch root", req.Suite.ID)
	}
	if req.Inputs.Revalidate == nil || req.Inputs.Release == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires held input view callbacks", req.Suite.ID)
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: the held input view of suite %s failed revalidation: %w", req.Suite.ID, err)
	}
	if _, err := processscope.NormalizeLimits(processscope.Limits(req.Limits)); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s limits: %w", req.Suite.ID, err)
	}
	limits, err := processscope.NormalizeLimits(processscope.Limits(req.Limits))
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s limits: %w", req.Suite.ID, err)
	}
	// Verified bundle materialization: only a capture-produced BundleRef
	// carries a materialized store object root.
	objectRoot := req.Source.Materialized()
	if objectRoot == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s source bundle was not produced by capture", req.Suite.ID)
	}
	storeRoot := filepath.Dir(filepath.Dir(objectRoot))
	projectID, err := elixirStoreProjectID(storeRoot)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s store identity: %w", req.Suite.ID, err)
	}
	bundleDir := filepath.Join(req.Scratch, "bundle")
	harnessDir := filepath.Join(req.Scratch, "harness")
	runRoot := filepath.Join(req.Scratch, "run")
	for _, dir := range []string{filepath.Join(runRoot, "home"), filepath.Join(runRoot, "tmp"), harnessDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: preparing private %s root: %w", dir, err)
		}
	}
	if err := tdd.MaterializeBundle(ctx, storeRoot, projectID, req.Source.Ref(), bundleDir); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: materializing the captured bundle of suite %s: %w", req.Suite.ID, err)
	}
	// Frozen declared sources from the captured bundle.
	sources := map[string][]byte{}
	for _, file := range req.Suite.Files {
		body, err := os.ReadFile(filepath.Join(bundleDir, filepath.FromSlash(file)))
		if err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("MISSING_TEST: suite file %s is absent from the captured bundle: %w", file, err)
		}
		sources[file] = body
	}
	if err := validateElixirSuite(&req.Suite, sources); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s frozen sources: %w", req.Suite.ID, err)
	}
	// Byte-pinned harness materialization with round-trip checks.
	digests := map[string]string{}
	materialize := func(rel string, body []byte) error {
		target := filepath.Join(harnessDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, body, 0o644); err != nil {
			return err
		}
		readback, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(readback, body) {
			return fmt.Errorf("the materialized asset %s does not reproduce its frozen bytes", rel)
		}
		sum := fmt.Sprintf("%x", sha256.Sum256(body))
		digests[rel] = sum
		return nil
	}
	assets := map[string][]byte{
		"mix.exs":                elixirMixSource,
		"machinery/json.exs":     elixirJSONSource,
		"machinery/reporter.exs": elixirReporterSource,
		"machinery/check.exs":    elixirCheckSource,
		"test/test_helper.exs":   elixirBootstrapSource,
	}
	pins := ElixirAssets()
	for rel, body := range assets {
		if sum := fmt.Sprintf("%x", sha256.Sum256(body)); sum != pins[rel] {
			return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: embedded asset %s does not match its frozen pin sha256:%s", rel, sum)
		}
		if err := materialize(rel, body); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: materializing the frozen harness: %w", err)
		}
	}
	for _, file := range req.Suite.Files {
		if err := materialize(file, sources[file]); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: materializing the frozen test source %s: %w", file, err)
		}
	}
	eventsPath := filepath.Join(runRoot, elixirEventsFile)
	env, err := buildElixirRunEnv(runRoot, handle.ElixirBinDir(), handle.ErlangBinDir(), eventsPath, req.Suite.Environment)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s environment: %w", req.Suite.ID, err)
	}
	timeoutMS := limits.WallMS
	if timeoutMS < 1000 {
		timeoutMS = 1000
	}
	argv := append([]string{
		"test",
		"--seed", fmt.Sprintf("%d", ElixirSuiteSeed),
		"--max-cases", fmt.Sprintf("%d", ElixirSuiteMaxCases),
		"--warnings-as-errors",
	}, req.Suite.Files...)
	return tdd.NewPreparedSuite(&elixirPrepared{
		suite: &req.Suite, inputs: req.Inputs, scope: req.Scope, scratch: req.Scratch,
		harnessDir: harnessDir, eventsPath: eventsPath,
		mixBin: handle.MixPath(), closure: identity.Closure,
		elixirBin: handle.ElixirPath(), erlangBin: handle.ErlangPath(),
		argv: argv, env: env, timeoutMS: timeoutMS, limits: limits, fileDigests: digests,
	}), nil
}

// Run implements tdd.Adapter: it executes the frozen argv of the prepared
// harness as one guarded custody job through actual Mix compilation and
// native ExUnit execution, normalizes the embedded reporter stream to the
// closed machinery.tdd.event/v1 sequence, reconciles the complete
// module/test lifecycle against the frozen inventory with the registered
// assertion witnesses and effective ExUnit options, enforces exit
// concordance and re-verifies the prepared sources (late mutation).
func (a *ElixirAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	state, ok := prepared.PreparedState().(*elixirPrepared)
	if !ok || state == nil {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: run requires a prepared suite produced by this adapter's Prepare")
	}
	if state.ran {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: the prepared suite state is single-use")
	}
	state.ran = true
	execution := tdd.Execution{Suite: state.suite.ID, Custody: tdd.CustodyReport{Status: processscope.StatusCleaned}}
	if err := ctx.Err(); err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: run canceled before launch: %w", err)
	}
	if err := elixirVerifyUnchanged(state); err != nil {
		return execution, err
	}
	if err := state.inputs.Revalidate(); err != nil {
		return execution, fmt.Errorf("STALE_INPUT: held inputs changed before execution: %w", err)
	}
	runChild, err := state.scope.Child(ctx)
	if err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: open the native execution child scope: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if report, closeErr := runChild.Close(closeCtx); closeErr != nil || report.Status != processscope.StatusCleaned {
			for _, job := range report.Jobs {
				if !job.Registered || !job.Terminated || !job.Reaped {
					execution.Custody.Status = processscope.StatusCleanupFailed
					return
				}
			}
		}
	}()
	attached, err := runChild.Attach(processscope.Command{
		Executable:    state.mixBin,
		Args:          state.argv,
		Dir:           state.harnessDir,
		Env:           state.env,
		RuntimeDigest: "sha256:" + state.closure,
		DeadlineMS:    state.timeoutMS,
	})
	if err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: attach the native mix test process: %w", err)
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(state.timeoutMS)*time.Millisecond)
	defer cancel()
	var stdout, stderr bytes.Buffer
	result, runErr := runChild.Run(runCtx, attached, processscope.Streams{
		Stdout: &stdout, Stderr: &stderr,
		StdoutLimit: state.limits.StdoutBytes, StderrLimit: state.limits.StderrBytes,
	})
	execution.StdoutDigest = fmt.Sprintf("%x", sha256.Sum256(stdout.Bytes()))
	execution.StderrDigest = fmt.Sprintf("%x", sha256.Sum256(stderr.Bytes()))
	if result.Completed {
		code := int64(result.ExitCode)
		execution.ExitCode = &code
	}
	if runErr != nil {
		var scopeErr *processscope.Error
		if errors.As(runErr, &scopeErr) {
			switch scopeErr.Code {
			case processscope.CodeTimeout:
				return execution, fmt.Errorf("TIMEOUT: the native mix test process exceeded its wall deadline: %w", runErr)
			case processscope.CodeOutputLimit:
				return execution, fmt.Errorf("OUTPUT_LIMIT: the native mix test process exceeded its stream budget: %w", runErr)
			case processscope.CodeCanceled:
				return execution, fmt.Errorf("CUSTODY_ERROR: the native mix test process was cancelled: %w", runErr)
			}
		}
		execution.Custody.Status = processscope.StatusCleanupFailed
		return execution, fmt.Errorf("CUSTODY_ERROR: the native mix test process failed under custody: %w (stdout: %s)", runErr, boundedString(stdout.String(), 2048))
	}
	// Late source mutation is checked before reconciling the evidence
	// stream so a mutated harness cannot hide behind stream diagnostics.
	if err := elixirVerifyUnchanged(state); err != nil {
		execution.Outcome = "error"
		return execution, err
	}
	raw, readErr := os.ReadFile(state.eventsPath)
	if readErr != nil || int64(len(raw)) > state.limits.EventBytes {
		if execution.ExitCode != nil && *execution.ExitCode != 0 {
			return execution, fmt.Errorf("BUILD_ERROR: the native mix invocation failed without a native event stream (exit=%d stdout: %s stderr: %s)", *execution.ExitCode, boundedString(stdout.String(), 2048), boundedString(stderr.String(), 2048))
		}
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: the native event stream is absent without a native failure status (read: %w)", readErr)
	}
	streamEvents, parseErr := parseElixirReporterStream(raw)
	var rec elixirReconciliation
	var reconcileErr error
	if parseErr == nil {
		rec, reconcileErr = reconcileElixirStream(streamEvents, state.suite, elixirExpectedConfig())
	} else {
		reconcileErr = parseErr
	}
	if reconcileErr != nil {
		execution.Events = rec.Events
		execution.Outcome = "error"
		a.emitElixir(sink, rec.Events)
		if result.ExitCode == 1 {
			// A mix-level abort (compile failure, warnings-as-errors abort
			// or an empty selection) may leave the autorun reporter's
			// truncated stream behind; it is a precondition failure, never
			// RED evidence.
			return execution, fmt.Errorf("BUILD_ERROR: suite %s aborted at the mix level (exit=%d stdout: %s stderr: %s; stream: %w)", state.suite.ID, result.ExitCode, boundedString(stdout.String(), 2048), boundedString(stderr.String(), 2048), reconcileErr)
		}
		return execution, fmt.Errorf("suite %s native stream: %w", state.suite.ID, reconcileErr)
	}
	if !result.Completed || result.Signal != "" {
		execution.Outcome = "error"
		a.emitElixir(sink, rec.Events)
		return execution, fmt.Errorf("CUSTODY_ERROR: the native mix test invocation did not complete cleanly (completed=%v signal=%q exit=%d)", result.Completed, result.Signal, result.ExitCode)
	}
	switch {
	case rec.Outcome == "pass" && result.ExitCode == 0:
	case rec.Outcome == "assertion-fail" && result.ExitCode == 2:
	case result.ExitCode == 1:
		execution.Outcome = "error"
		a.emitElixir(sink, rec.Events)
		return execution, fmt.Errorf("BUILD_ERROR: suite %s aborted at the mix level after execution (exit=1 stdout: %s stderr: %s)", state.suite.ID, boundedString(stdout.String(), 2048), boundedString(stderr.String(), 2048))
	default:
		execution.Outcome = "error"
		a.emitElixir(sink, rec.Events)
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: suite %s exit status %d does not concord with reconciled outcome %q", state.suite.ID, result.ExitCode, rec.Outcome)
	}
	if int64(len(rec.Events)) > state.limits.EventCount {
		execution.Outcome = "error"
		return execution, fmt.Errorf("OUTPUT_LIMIT: normalized event count %d exceeds the budget %d", len(rec.Events), state.limits.EventCount)
	}
	execution.Events = rec.Events
	execution.Outcome = rec.Outcome
	a.emitElixir(sink, rec.Events)
	return execution, nil
}

func (a *ElixirAdapter) emitElixir(sink tdd.EventSink, events []tdd.Event) {
	if sink == nil {
		return
	}
	for _, e := range events {
		if err := sink(e); err != nil {
			return
		}
	}
}

// elixirVerifyUnchanged re-hashes the prepared harness sources and assets.
func elixirVerifyUnchanged(state *elixirPrepared) error {
	for name, want := range state.fileDigests {
		body, err := os.ReadFile(filepath.Join(state.harnessDir, filepath.FromSlash(name)))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(body)) != want {
			return fmt.Errorf("STALE_INPUT: prepared source %s changed after preparation (want sha256:%s)", name, want)
		}
	}
	return nil
}

// elixirStoreProjectID reads the closed store identity of a capture store.
func elixirStoreProjectID(storeRoot string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(storeRoot, "store.json"))
	if err != nil || len(raw) > 1<<20 {
		return "", fmt.Errorf("cannot read the store identity: %w", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	if len(doc) != 3 || doc["schema"] != "machinery.tdd.store/v1" {
		return "", fmt.Errorf("store.json is not the closed store identity")
	}
	id, _ := doc["project_id"].(string)
	if id == "" {
		return "", fmt.Errorf("store.json lacks the project identity")
	}
	return id, nil
}

// buildElixirRunEnv builds the closed native environment: the pinned
// Elixir/OTP binaries on PATH, private HOME/TMPDIR/MIX_HOME/MIX_BUILD_PATH
// under the run root, the private events channel, UTC, no color, declared
// suite variables only; Mix configuration overrides and custody transport
// variables are never inherited.
func buildElixirRunEnv(runRoot, elixirBinDir, erlangBinDir, eventsPath string, extra []tdd.EnvironmentVar) ([]string, error) {
	declared := map[string]string{}
	for _, item := range extra {
		if !elixirEnvNamePattern.MatchString(item.Name) {
			return nil, fmt.Errorf("declared environment name %q is not a closed variable name", item.Name)
		}
		if len(item.Value) > 65536 {
			return nil, fmt.Errorf("declared environment value of %s exceeds the bounded size", item.Name)
		}
		if elixirClosedEnvKeys[item.Name] || strings.HasPrefix(item.Name, "MIX_") || strings.HasPrefix(item.Name, "MACHINERY_") {
			continue // the closed native environment is never overridable
		}
		if _, duplicate := declared[item.Name]; duplicate {
			return nil, fmt.Errorf("declared environment name %s is duplicated", item.Name)
		}
		declared[item.Name] = item.Value
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	env := []string{
		"PATH=" + elixirBinDir + string(os.PathListSeparator) + erlangBinDir + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
		"HOME=" + filepath.Join(runRoot, "home"),
		"TMPDIR=" + filepath.Join(runRoot, "tmp"),
		"TZ=UTC",
		"LC_ALL=C.UTF-8",
		"NO_COLOR=1",
		"MIX_ENV=test",
		"MIX_HOME=" + filepath.Join(runRoot, "home", ".mix"),
		"MIX_BUILD_PATH=" + filepath.Join(runRoot, "build"),
		"MACHINERY_ASSURANCE_EVENTS=" + eventsPath,
	}
	for _, name := range names {
		env = append(env, name+"="+declared[name])
	}
	return env, nil
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

// elixirRawEvent mirrors the reporter line shape {"t":..., "d":...}.
type elixirRawEvent struct {
	Type string          `json:"t"`
	D    json.RawMessage `json:"d"`
}

// parseElixirReporterStream decodes the closed embedded-reporter JSONL
// vocabulary; unknown event identities, unknown payload keys, malformed
// lines and missing, duplicated or non-terminal sentinels fail closed.
// Module atoms are de-Elixirified and native test names de-"test"-prefixed
// so identities compare as declared.
func parseElixirReporterStream(raw []byte) ([]elixirEvent, error) {
	text := strings.TrimSuffix(string(raw), "\n")
	if text == "" {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the reporter stream is empty")
	}
	lines := strings.Split(text, "\n")
	if int64(len(lines)) > elixirMaxStreamLines {
		return nil, fmt.Errorf("OUTPUT_LIMIT: the reporter stream exceeds %d lines", elixirMaxStreamLines)
	}
	events := make([]elixirEvent, 0, len(lines))
	sentinel := -1
	for i, line := range lines {
		if len(line) > elixirMaxLineLen {
			return nil, fmt.Errorf("OUTPUT_LIMIT: a reporter line exceeds the bound")
		}
		var rawEvent elixirRawEvent
		if err := json.Unmarshal([]byte(line), &rawEvent); err != nil || rawEvent.Type == "" {
			return nil, fmt.Errorf("INCOMPLETE_EVENTS: malformed reporter line %d", i+1)
		}
		keys, known := elixirStreamKeys[rawEvent.Type]
		if !known {
			return nil, fmt.Errorf("INCOMPLETE_EVENTS: unknown native event identity %q", rawEvent.Type)
		}
		if rawEvent.Type == ElixirReporterSentinel {
			if sentinel != -1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: duplicated terminal sentinel")
			}
			if i != len(lines)-1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is not terminal")
			}
			sentinel = i
			events = append(events, elixirEvent{Type: rawEvent.Type})
			continue
		}
		event, err := decodeElixirEvent(rawEvent.Type, rawEvent.D, keys)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if sentinel == -1 {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is missing; the stream is truncated")
	}
	return events, nil
}

func decodeElixirEvent(kind string, d json.RawMessage, keys map[string]bool) (elixirEvent, error) {
	event := elixirEvent{Type: kind}
	if len(d) == 0 {
		if len(keys) != 0 {
			return event, fmt.Errorf("INCOMPLETE_EVENTS: %s payload is absent", kind)
		}
		return event, nil
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(d, &doc); err != nil {
		return event, fmt.Errorf("INCOMPLETE_EVENTS: malformed %s payload: %w", kind, err)
	}
	for key := range doc {
		if !keys[key] {
			return event, fmt.Errorf("INVALID_SCHEMA: %s payload carries unknown key %q", kind, key)
		}
	}
	if len(doc) != len(keys) {
		return event, fmt.Errorf("INVALID_SCHEMA: %s payload is missing closed keys (%d of %d)", kind, len(doc), len(keys))
	}
	str := func(name string) (string, error) {
		var value string
		if err := json.Unmarshal(doc[name], &value); err != nil {
			return "", fmt.Errorf("INVALID_SCHEMA: %s payload key %q is not a string", kind, name)
		}
		return value, nil
	}
	integer := func(name string) (int64, error) {
		var value int64
		if err := json.Unmarshal(doc[name], &value); err != nil {
			return 0, fmt.Errorf("INVALID_SCHEMA: %s payload key %q is not an integer", kind, name)
		}
		return value, nil
	}
	boolean := func(name string) (bool, error) {
		var value bool
		if err := json.Unmarshal(doc[name], &value); err != nil {
			return false, fmt.Errorf("INVALID_SCHEMA: %s payload key %q is not a boolean", kind, name)
		}
		return value, nil
	}
	stringsOf := func(name string) ([]string, error) {
		var values []string
		if err := json.Unmarshal(doc[name], &values); err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: %s payload key %q is not a string array", kind, name)
		}
		return values, nil
	}
	var err error
	switch kind {
	case "suite_started":
		config := &elixirEffectiveConfig{}
		if config.Seed, err = integer("seed"); err != nil {
			return event, err
		}
		if config.MaxCases, err = integer("max_cases"); err != nil {
			return event, err
		}
		if config.Trace, err = boolean("trace"); err != nil {
			return event, err
		}
		if config.Timeout, err = integer("timeout"); err != nil {
			return event, err
		}
		if config.MaxFailures, err = str("max_failures"); err != nil {
			return event, err
		}
		if config.Include, err = stringsOf("include"); err != nil {
			return event, err
		}
		if config.Exclude, err = stringsOf("exclude"); err != nil {
			return event, err
		}
		if config.Formatters, err = stringsOf("formatters"); err != nil {
			return event, err
		}
		if config.DryRun, err = boolean("dry_run"); err != nil {
			return event, err
		}
		if config.RepeatUntilFailure, err = integer("repeat_until_failure"); err != nil {
			return event, err
		}
		event.Config = config
	case "module_started":
		if event.Module, err = str("module"); err != nil {
			return event, err
		}
		event.Module = strings.TrimPrefix(event.Module, "Elixir.")
	case "module_finished":
		if event.Module, err = str("module"); err != nil {
			return event, err
		}
		event.Module = strings.TrimPrefix(event.Module, "Elixir.")
		if event.ModuleDone, err = str("state"); err != nil {
			return event, err
		}
	case "test_started", "test_finished":
		identity := &elixirTestIdentity{}
		if identity.Module, err = str("module"); err != nil {
			return event, err
		}
		if identity.Describe, err = str("describe"); err != nil {
			return event, err
		}
		if identity.Name, err = str("test"); err != nil {
			return event, err
		}
		if identity.File, err = str("file"); err != nil {
			return event, err
		}
		if identity.Line, err = integer("line"); err != nil {
			return event, err
		}
		identity.Module = strings.TrimPrefix(identity.Module, "Elixir.")
		identity.Name = strings.TrimPrefix(identity.Name, ElixirNativeTestPrefix)
		event.Test = identity
		if kind == "test_started" {
			if identity.Async, err = boolean("async"); err != nil {
				return event, err
			}
		}
		if kind == "test_finished" {
			if event.State, err = str("state"); err != nil {
				return event, err
			}
			if event.Class, err = str("class"); err != nil {
				return event, err
			}
			if event.Message, err = str("message"); err != nil {
				return event, err
			}
			if !elixirTestStates[event.State] {
				return event, fmt.Errorf("INVALID_SCHEMA: unknown native test state %q", event.State)
			}
		}
	case "witness":
		witness := &elixirWitness{}
		if witness.Module, err = str("module"); err != nil {
			return event, err
		}
		if witness.Describe, err = str("describe"); err != nil {
			return event, err
		}
		if witness.Name, err = str("test"); err != nil {
			return event, err
		}
		if witness.ID, err = str("id"); err != nil {
			return event, err
		}
		if witness.File, err = str("file"); err != nil {
			return event, err
		}
		if witness.Line, err = integer("line"); err != nil {
			return event, err
		}
		if witness.Value, err = boolean("value"); err != nil {
			return event, err
		}
		if witness.Verdict, err = str("verdict"); err != nil {
			return event, err
		}
		witness.Module = strings.TrimPrefix(witness.Module, "Elixir.")
		witness.Name = strings.TrimPrefix(witness.Name, ElixirNativeTestPrefix)
		event.Witness = witness
	case "suite_finished":
		summary := &elixirSummary{}
		if summary.Tests, err = integer("tests"); err != nil {
			return event, err
		}
		if summary.Failures, err = integer("failures"); err != nil {
			return event, err
		}
		if summary.Skipped, err = integer("skipped"); err != nil {
			return event, err
		}
		if summary.Invalid, err = integer("invalid"); err != nil {
			return event, err
		}
		if summary.Excluded, err = integer("excluded"); err != nil {
			return event, err
		}
		event.Summary = summary
	}
	return event, nil
}

// elixirTestState is the reconciliation state of one declared identity.
type elixirTestState struct {
	test      *tdd.Test
	started   bool
	finished  bool
	describe  string
	file      string
	line      int64
	async     bool
	state     string
	class     string
	order     int64
	witnessed map[string]bool
	sawFalse  bool
	witnesses []*elixirWitness
}

// elixirModuleState is the reconciliation state of one declared module.
type elixirModuleState struct {
	started  bool
	finished bool
	terminal string
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

func elixirIdentityKey(module, name string) string { return module + "\x00" + name }

// reconcileElixirStream reconciles the native lifecycle against the
// complete frozen inventory: exactly-once module/test lifecycle per
// declared identity, witness-bound assertion outcomes at the exact
// registered frozen call sites with pid-verdict enforcement, effective
// ExUnit option enforcement, skip/exclude/invalid rejection,
// AssertionError causality for every expected failure and final summary
// concordance.
func reconcileElixirStream(events []elixirEvent, suite *tdd.Suite, expect elixirEffectiveConfig) (elixirReconciliation, error) {
	rec := elixirReconciliation{}
	if len(suite.Tests) == 0 {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: empty declared selection cannot yield a successful execution")
	}
	states := map[string]*elixirTestState{}
	modules := map[string]*elixirModuleState{}
	assertionOwner := map[string]string{}
	assertionOf := map[string]*tdd.Assertion{}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		if test.Native.Module == "" || test.Native.Name == "" {
			return rec, fmt.Errorf("INVALID_SCHEMA: test %s lacks its elixir native identity", test.ID)
		}
		key := elixirIdentityKey(test.Native.Module, test.Native.Name)
		if _, duplicate := states[key]; duplicate {
			return rec, fmt.Errorf("DUPLICATE_TEST: native identity %s/%s is declared more than once", test.Native.Module, test.Native.Name)
		}
		states[key] = &elixirTestState{test: test, witnessed: map[string]bool{}}
		if modules[test.Native.Module] == nil {
			modules[test.Native.Module] = &elixirModuleState{}
		}
		for j := range test.Assertions {
			assertion := &test.Assertions[j]
			if _, duplicate := assertionOwner[assertion.ID]; duplicate {
				return rec, fmt.Errorf("INVALID_SCHEMA: assertion id %s is registered more than once in suite %s", assertion.ID, suite.ID)
			}
			assertionOwner[assertion.ID] = key
			assertionOf[assertion.ID] = assertion
		}
	}
	if len(events) == 0 || events[len(events)-1].Type != ElixirReporterSentinel {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the terminal reporter sentinel is absent")
	}
	var config *elixirEffectiveConfig
	var summary *elixirSummary
	var failures, skipped, excluded, invalid int64
	startCounter := int64(0)
	startOrder := []string{}
	for _, event := range events {
		switch event.Type {
		case ElixirReporterSentinel:
			continue
		case "suite_started":
			if config != nil {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: duplicated suite lifecycle start")
			}
			config = event.Config
		case "module_started":
			module, declared := modules[event.Module]
			if !declared {
				return rec, fmt.Errorf("MISSING_TEST: native module %s is outside the frozen inventory of this suite", event.Module)
			}
			if module.started {
				return rec, fmt.Errorf("DUPLICATE_TEST: native module %s started more than once", event.Module)
			}
			module.started = true
		case "module_finished":
			module, declared := modules[event.Module]
			if !declared {
				return rec, fmt.Errorf("MISSING_TEST: native module %s is outside the frozen inventory of this suite", event.Module)
			}
			if !module.started || module.finished {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: native module %s lifecycle is not complete", event.Module)
			}
			module.finished = true
			module.terminal = event.ModuleDone
			if event.ModuleDone == "invalid" {
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: native module %s reported an invalid state (setup_all failure)", event.Module)
			}
			if event.ModuleDone != "finished" {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: native module %s reported unknown terminal %q", event.Module, event.ModuleDone)
			}
		case "test_started":
			key := elixirIdentityKey(event.Test.Module, event.Test.Name)
			state, declared := states[key]
			if !declared {
				return rec, fmt.Errorf("MISSING_TEST: native identity %s/%s is outside the frozen inventory of this suite", event.Test.Module, event.Test.Name)
			}
			if state.started || state.finished {
				return rec, fmt.Errorf("DUPLICATE_TEST: native test %s/%s started more than once", event.Test.Module, event.Test.Name)
			}
			state.started = true
			startCounter++
			state.order = startCounter
			state.describe = event.Test.Describe
			state.file = event.Test.File
			state.line = event.Test.Line
			state.async = event.Test.Async
			startOrder = append(startOrder, key)
		case "test_finished":
			key := elixirIdentityKey(event.Test.Module, event.Test.Name)
			state, declared := states[key]
			if !declared {
				return rec, fmt.Errorf("MISSING_TEST: native identity %s/%s is outside the frozen inventory of this suite", event.Test.Module, event.Test.Name)
			}
			if !state.started || state.finished {
				return rec, fmt.Errorf("DUPLICATE_TEST: native terminal of %s/%s is not in a complete lifecycle", event.Test.Module, event.Test.Name)
			}
			if event.Test.Describe != state.describe {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: native describe of %s/%s changed across its lifecycle (%q vs %q)", event.Test.Module, event.Test.Name, event.Test.Describe, state.describe)
			}
			if event.Test.File != state.test.Native.File || event.Test.Line != state.test.Native.Line {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: native identity %s/%s reports %s:%d, want the declared site %s:%d", event.Test.Module, event.Test.Name, event.Test.File, event.Test.Line, state.test.Native.File, state.test.Native.Line)
			}
			state.finished = true
			state.state = event.State
			state.class = event.Class
			switch event.State {
			case "passed":
			case "failed":
				failures++
			case "skipped":
				return rec, fmt.Errorf("UNSUPPORTED_FEATURE: declared test %s/%s was skipped by the native runner; required skips can never become success", event.Test.Module, event.Test.Name)
			case "excluded":
				return rec, fmt.Errorf("UNSUPPORTED_FEATURE: declared test %s/%s was excluded by a native filter; silent selection can never become success", event.Test.Module, event.Test.Name)
			case "invalid":
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: declared test %s/%s reported an invalid native state", event.Test.Module, event.Test.Name)
			default:
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared test %s/%s reported unknown state %q", event.Test.Module, event.Test.Name, event.State)
			}
		case "witness":
			witness := event.Witness
			key := elixirIdentityKey(witness.Module, witness.Name)
			state, declared := states[key]
			if !declared {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness carries native identity %s/%s outside the registered inventory", witness.Module, witness.Name)
			}
			if !state.started || state.finished {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s arrived outside its active native lifecycle", witness.ID)
			}
			if witness.Describe != state.describe {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s claims describe %q but its test reports %q", witness.ID, witness.Describe, state.describe)
			}
			ownerKey, known := assertionOwner[witness.ID]
			if !known {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness carries assertion %q outside the registered inventory", witness.ID)
			}
			if ownerKey != key {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s claims test %s/%s but is registered for another identity", witness.ID, witness.Module, witness.Name)
			}
			if witness.Verdict != "ok" {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s carries verdict %q; only the framework-owned pid binding is trusted", witness.ID, witness.Verdict)
			}
			assertion := assertionOf[witness.ID]
			if witness.File != assertion.Source || witness.Line != assertion.Line {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s reports site %s:%d, want the registered call site %s:%d", witness.ID, witness.File, witness.Line, assertion.Source, assertion.Line)
			}
			if state.witnessed[witness.ID] {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: assertion %s was witnessed more than once", witness.ID)
			}
			state.witnessed[witness.ID] = true
			if !witness.Value {
				state.sawFalse = true
			}
			state.witnesses = append(state.witnesses, witness)
		case "suite_finished":
			if summary != nil {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: duplicated final native summary")
			}
			summary = event.Summary
		case "max_failures_reached":
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: the native runner truncated the suite at its failure ceiling")
		default:
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: unknown native event identity %q reached reconciliation", event.Type)
		}
	}
	if config == nil {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the effective native options were never reported")
	}
	if summary == nil {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the final native summary is missing")
	}
	// Effective-option enforcement: the frozen argv fixes every closed
	// option; any drift the reporter reports is a protocol violation.
	switch {
	case config.Seed != expect.Seed:
		return rec, fmt.Errorf("INVALID_SCHEMA: effective native seed %d is not the frozen %d", config.Seed, expect.Seed)
	case config.MaxCases != expect.MaxCases:
		return rec, fmt.Errorf("INVALID_SCHEMA: effective native max_cases %d is not the frozen %d", config.MaxCases, expect.MaxCases)
	case config.Trace != expect.Trace:
		return rec, fmt.Errorf("INVALID_SCHEMA: effective native trace %v is not the frozen %v", config.Trace, expect.Trace)
	case config.Timeout != expect.Timeout:
		return rec, fmt.Errorf("INVALID_SCHEMA: effective native timeout %d is not the frozen %d", config.Timeout, expect.Timeout)
	}
	if len(config.Include) != 0 || len(config.Exclude) != 0 {
		return rec, fmt.Errorf("UNSUPPORTED_FEATURE: effective native filters include=%v exclude=%v; silent selection can never become success", config.Include, config.Exclude)
	}
	if len(config.Formatters) != 1 || config.Formatters[0] != ElixirReporterModule {
		return rec, fmt.Errorf("UNSUPPORTED_FEATURE: effective native formatters %v are not exactly the embedded reporter", config.Formatters)
	}
	if config.MaxFailures != expect.MaxFailures {
		return rec, fmt.Errorf("UNSUPPORTED_FEATURE: effective native max_failures %q is not the frozen %q", config.MaxFailures, expect.MaxFailures)
	}
	if config.DryRun {
		return rec, fmt.Errorf("UNSUPPORTED_FEATURE: effective native dry_run mode can never certify execution")
	}
	if config.RepeatUntilFailure != expect.RepeatUntilFailure {
		return rec, fmt.Errorf("UNSUPPORTED_FEATURE: effective native repeat_until_failure %d is not the frozen %d", config.RepeatUntilFailure, expect.RepeatUntilFailure)
	}
	// Declared-inventory accounting with assertion causality.
	for i := range suite.Tests {
		test := &suite.Tests[i]
		state := states[elixirIdentityKey(test.Native.Module, test.Native.Name)]
		if !state.started {
			return rec, fmt.Errorf("MISSING_TEST: declared test %s/%s never started in the native runner", test.Native.Module, test.Native.Name)
		}
		if !state.finished {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared test %s/%s never reported a terminal state", test.Native.Module, test.Native.Name)
		}
		if state.state == "passed" && state.sawFalse {
			return rec, fmt.Errorf("ASSERTION_MISMATCH: witness reported a false condition but native test %s/%s passed", test.Native.Module, test.Native.Name)
		}
		if state.state == "failed" {
			if state.class != "ExUnit.AssertionError" {
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s/%s failed with class %q, not the helper-backed ExUnit.AssertionError", test.Native.Module, test.Native.Name, state.class)
			}
			if !state.sawFalse {
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s/%s has no helper-backed false witness (assertions=%d)", test.Native.Module, test.Native.Name, len(test.Assertions))
			}
		}
		for _, assertion := range test.Assertions {
			if !state.witnessed[assertion.ID] {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: registered assertion %s never executed in test %s/%s", assertion.ID, test.Native.Module, test.Native.Name)
			}
		}
	}
	for module, state := range modules {
		if !state.started {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared module %s never started", module)
		}
		if !state.finished {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared module %s never reported its terminal state", module)
		}
	}
	// Summary concordance: the reporter's own accounting must equal the
	// reconciled events exactly.
	if summary.Tests != int64(len(suite.Tests)) || summary.Failures != failures ||
		summary.Skipped != skipped || summary.Invalid != invalid || summary.Excluded != excluded {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: native summary %+v disagrees with the reconciled accounting (tests=%d failures=%d skipped=%d invalid=%d excluded=%d)", summary, len(suite.Tests), failures, skipped, invalid, excluded)
	}
	// Normalized emission: suite-start, discovered inventory, then per
	// test in native start order with witnesses in execution order.
	failedAny := failures > 0
	rec.Outcome = "pass"
	if failedAny {
		rec.Outcome = "assertion-fail"
	}
	sequence := int64(0)
	emit := func(kind string, test *tdd.Test, assertion string, outcome string, source string, line int64) {
		sequence++
		event := tdd.Event{
			Schema: ElixirEventSchema, Sequence: sequence, Suite: suite.ID,
			Kind: kind, Assertion: assertion, Outcome: outcome, Source: source, Line: line,
		}
		if test != nil {
			native := test.Native
			event.Native = &native
		}
		rec.Events = append(rec.Events, event)
	}
	emit("suite-start", nil, "", "", "", 0)
	for i := range suite.Tests {
		test := &suite.Tests[i]
		rec.Discovered = append(rec.Discovered, test.Native)
		emit("discovered", test, "", "", "", 0)
	}
	sort.Slice(startOrder, func(i, j int) bool { return states[startOrder[i]].order < states[startOrder[j]].order })
	for _, key := range startOrder {
		state := states[key]
		test := state.test
		emit("test-start", test, "", "", "", 0)
		rec.Started = append(rec.Started, test.Native)
		for _, witness := range state.witnesses {
			outcome := "pass"
			if !witness.Value {
				outcome = "assertion-fail"
			}
			assertion := assertionOf[witness.ID]
			emit("assertion", test, witness.ID, outcome, assertion.Source, assertion.Line)
			rec.Assertions = append(rec.Assertions, tdd.AssertionOutcome{
				Test: tdd.TestRef{Suite: suite.ID, Test: test.ID}, ID: witness.ID, Outcome: outcome,
			})
		}
		outcome := "pass"
		if state.state == "failed" {
			outcome = "assertion-fail"
			rec.Failed = append(rec.Failed, test.Native)
		}
		emit("test-end", test, "", outcome, "", 0)
		rec.Completed = append(rec.Completed, test.Native)
	}
	emit("suite-end", nil, "", rec.Outcome, "", 0)
	return rec, nil
}

// validateElixirSuite validates the closed suite shape: elixir-native
// identities bound to declared test files under the frozen test/ grammar,
// the frozen ExUnit.Case grammar, and typed helper call sites at the exact
// frozen lines with comment-derived evidence rejected.
func validateElixirSuite(suite *tdd.Suite, sources map[string][]byte) error {
	if len(suite.Files) == 0 || len(suite.Tests) == 0 {
		return fmt.Errorf("INVALID_SCHEMA: suite %s declares no files or no tests", suite.ID)
	}
	seenFiles := map[string]bool{}
	for _, file := range suite.Files {
		if !elixirTestFilePattern.MatchString(file) || file == "test/test_helper.exs" {
			return fmt.Errorf("UNSUPPORTED_FEATURE: suite file %q is outside the frozen test/<name>_test.exs grammar of elixir-exunit/v1", file)
		}
		if seenFiles[file] {
			return fmt.Errorf("INVALID_SCHEMA: suite declares file %s more than once", file)
		}
		seenFiles[file] = true
	}
	for _, test := range suite.Tests {
		if test.Native.Package != "" || test.Native.Test != "" || test.Native.Source != "" ||
			len(test.Native.Path) != 0 || test.Native.Column != 0 || test.Native.Class != "" || test.Native.Method != "" {
			return fmt.Errorf("UNSUPPORTED_FEATURE: test %s carries a non-elixir native identity", test.ID)
		}
		if test.Native.Module == "" || test.Native.Name == "" || test.Native.File == "" || test.Native.Line <= 0 {
			return fmt.Errorf("INVALID_SCHEMA: test %s lacks its elixir module/name/file/line identity", test.ID)
		}
		if strings.HasPrefix(test.Native.Name, ElixirNativeTestPrefix) {
			return fmt.Errorf("INVALID_SCHEMA: test %s native name %q carries the native test- prefix; declared names never include it", test.ID, test.Native.Name)
		}
		if _, ok := sources[test.Native.File]; !ok {
			return fmt.Errorf("INVALID_SCHEMA: test %s binds to undeclared source %s", test.ID, test.Native.File)
		}
		if test.Source != test.Native.File {
			return fmt.Errorf("INVALID_SCHEMA: test %s source %s does not match its native file %s", test.ID, test.Source, test.Native.File)
		}
	}
	assertionIDs := map[string]bool{}
	for _, test := range suite.Tests {
		lines := strings.Split(string(sources[test.Source]), "\n")
		hasCase := false
		hasRequire := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "use ExUnit.Case" || strings.HasPrefix(trimmed, "use ExUnit.Case,") {
				hasCase = true
			}
			if trimmed == "require Machinery.Check" {
				hasRequire = true
			}
		}
		if !hasCase {
			return fmt.Errorf("UNSUPPORTED_FEATURE: source %s of test %s does not use ExUnit.Case; only plain ExUnit.Case suites are strict identities", test.Source, test.ID)
		}
		if !hasRequire && len(test.Assertions) > 0 {
			return fmt.Errorf("ASSERTION_MISMATCH: source %s does not require the byte-pinned helper transport Machinery.Check", test.Source)
		}
		for _, assertion := range test.Assertions {
			if assertion.Helper != protocol.AssertionHelperV1 {
				return fmt.Errorf("UNSUPPORTED_FEATURE: assertion %s uses helper %q; only %s is closed", assertion.ID, assertion.Helper, protocol.AssertionHelperV1)
			}
			if assertion.Source != test.Source {
				return fmt.Errorf("INVALID_SCHEMA: assertion %s binds to source %s outside its test's source %s", assertion.ID, assertion.Source, test.Source)
			}
			if !elixirAssertionIDPattern.MatchString(assertion.ID) {
				return fmt.Errorf("INVALID_SCHEMA: assertion identity %q is not a closed ID", assertion.ID)
			}
			if assertionIDs[assertion.ID] {
				return fmt.Errorf("DUPLICATE_TEST: assertion %s is registered twice", assertion.ID)
			}
			assertionIDs[assertion.ID] = true
			if assertion.Line <= 0 || assertion.Line > int64(len(lines)) {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion %s line %d is outside its frozen source", assertion.ID, assertion.Line)
			}
			// The typed call site: the trimmed line must BEGIN with the
			// qualified helper call and carry the exact id literal on the
			// same line. A commented-out call can never satisfy this, so
			// assertion evidence stays execution-derived.
			line := strings.TrimSpace(lines[assertion.Line-1])
			if !strings.HasPrefix(line, "Machinery.Check.check(") || !strings.Contains(line, `"`+assertion.ID+`"`) {
				return fmt.Errorf("ASSERTION_MISMATCH: %s:%d is not the typed Machinery.Check.check call site of assertion %s (line %q)", assertion.Source, assertion.Line, assertion.ID, truncateElixir(line, 96))
			}
		}
	}
	return nil
}

func truncateElixir(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// elixirExpectedConfig returns the frozen effective-option record of this
// adapter's closed invocation.
func elixirExpectedConfig() elixirEffectiveConfig {
	return elixirEffectiveConfig{
		Seed: ElixirSuiteSeed, MaxCases: ElixirSuiteMaxCases, Trace: false,
		Timeout: ElixirNativeTimeoutMS, MaxFailures: "infinity", Include: []string{}, Exclude: []string{},
		Formatters: []string{ElixirReporterModule}, DryRun: false, RepeatUntilFailure: 0,
	}
}
