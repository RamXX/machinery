// Package adapters owns the closed first-release native adapters of
// docs/test-assurance-contract.md section 6. This file is the
// python-unittest/v1 adapter (MAC-imtz): it embeds the byte-pinned
// machinery-check/v1 assertion helper and the byte-pinned bootstrap/harness
// from the owned assets directory, prepares captured Python suites
// (verified bundle materialization, embedded-harness static AST validation
// of every registered assertion call site, native TestLoader discovery with
// full TestCase.id() reconciliation, exact frozen argv under CPython's
// isolated mode with a closed environment and no bytecode) and executes
// them under processscope custody, normalizing the harness JSONL report to
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
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

//go:embed assets/python/machinery_check.py
var pythonHelperSource []byte

//go:embed assets/python/bootstrap.py
var pythonBootstrapSource []byte

// Closed constants of the python-unittest/v1 adapter.
const (
	// AdapterPythonUnittest is the closed adapter identity.
	AdapterPythonUnittest = protocol.AdapterPythonUnittest

	// PythonEventSchema is the normalized event identity of every emitted
	// event.
	PythonEventSchema = "machinery.tdd.event/v1"

	// PythonWitnessSchema is the closed helper witness identity.
	PythonWitnessSchema = "machinery.tdd.witness/v1"

	// PythonHarnessSchema is the closed embedded-harness record identity.
	PythonHarnessSchema = "machinery.tdd.harness.python/v1"

	// PythonHelperPinnedSHA256 and PythonBootstrapPinnedSHA256 pin the
	// exact embedded asset bytes; each pin is verified at materialization
	// and re-verified after every native run.
	PythonHelperPinnedSHA256    = "a6dbeaecd11d7207bf9aa00ef9804dc8173e5378eb2ca9c38ee6a1e9924feb81"
	PythonBootstrapPinnedSHA256 = "90ef6c4c77c7b71b6879ef4152bf96e7016baa8ac506018b93fe621ce3a6e8f5"

	// PythonConformanceFixtureSHA256 pins the frozen native conformance
	// fixture bytes executed by the required contributor lane fragment
	// testdata/integration-lanes/assurance-python.json.
	PythonConformanceFixtureSHA256 = "53151c61000022e78a34d1874ab3581bb0a37821d05eb0fa7ca96dad564b5038"

	// PythonHelperFile and PythonBootstrapFile are the materialized asset
	// names inside every prepared suite root.
	PythonHelperFile    = "machinery_check.py"
	PythonBootstrapFile = "_machinery_bootstrap.py"

	pythonRequiredVersion = "3.14.7"
	pythonProbeTimeoutMS  = int64(120000)
	pythonReportMaxBytes  = int64(16 << 20)
	pythonMaxReportLines  = 100000
)

var (
	pythonAssertionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
	pythonEnvNamePattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pythonModulePattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pythonClassPattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pythonMethodPattern      = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	pythonClosedEnvKeys      = map[string]bool{
		"PATH": true, "HOME": true, "TMPDIR": true, "TZ": true, "LANG": true,
		"LC_ALL": true, "LC_CTYPE": true, "NO_COLOR": true,
		"PYTHONHASHSEED": true, "PYTHONDONTWRITEBYTECODE": true, "PYTHONNOUSERSITE": true,
	}
	pythonReportOutcomes = map[string]bool{
		"pass": true, "assertion-fail": true, "error": true, "skipped": true,
		"expected-failure": true, "unexpected-success": true,
	}
	// pythonReportKeys is the closed per-kind key set of the harness
	// report vocabulary; every other key is a forged stream.
	pythonReportKeys = map[string]map[string]bool{
		"suite-start":         {"kind": true, "schema": true, "tests": true},
		"test-start":          {"kind": true, "schema": true, "test": true},
		"test-end":            {"kind": true, "schema": true, "test": true, "outcome": true, "exception": true},
		"unsupported-feature": {"kind": true, "schema": true, "test": true, "feature": true},
		"suite-error":         {"kind": true, "schema": true, "code": true, "message": true},
		"suite-end": {
			"kind": true, "schema": true, "tests_run": true, "failures": true, "errors": true,
			"skipped": true, "expected_failures": true, "unexpected_successes": true, "success": true,
		},
		"bootstrap-end": {"kind": true, "schema": true},
		"witness":       {"kind": true, "schema": true, "assertion": true, "test": true, "site": true, "condition": true, "thrown": true},
	}
)

// PythonAdapter is the closed python-unittest/v1 Adapter.
type PythonAdapter struct{}

// Compile-time contract check of the closed Adapter boundary.
var _ tdd.Adapter = (*PythonAdapter)(nil)

// Python returns the python-unittest/v1 adapter.
func Python() *PythonAdapter { return &PythonAdapter{} }

// ID implements tdd.Adapter.
func (a *PythonAdapter) ID() string { return AdapterPythonUnittest }

// PythonHelperSource returns a copy of the embedded byte-pinned assertion
// helper transport.
func PythonHelperSource() []byte { return append([]byte(nil), pythonHelperSource...) }

// PythonBootstrapSource returns a copy of the embedded byte-pinned
// bootstrap/harness.
func PythonBootstrapSource() []byte { return append([]byte(nil), pythonBootstrapSource...) }

// PythonAssets returns the frozen asset inventory: embedded asset name to
// its pinned exact sha256.
func PythonAssets() map[string]string {
	return map[string]string{
		PythonHelperFile:    PythonHelperPinnedSHA256,
		PythonBootstrapFile: PythonBootstrapPinnedSHA256,
	}
}

// pythonPrepared is the adapter-owned opaque prepared state: the frozen
// suite inventory, the prepared scratch layout identities, the exact closed
// argv/environment and the pinned runtime members. Only this adapter's Run
// accepts it.
type pythonPrepared struct {
	suite       *tdd.Suite
	inputs      tdd.InputView
	scope       processscope.Scope
	scratch     string
	suiteDir    string
	inventory   string
	report      string
	pythonBin   string
	closure     string
	argv        []string
	env         []string
	limits      processscope.Limits
	fileDigests map[string]string
	ran         bool
}

// Prepare implements tdd.Adapter: it validates the closed request, opens and
// validates the pinned CPython 3.14.7 runtime closure under the live scope,
// materializes and re-verifies the captured source bundle plus the embedded
// byte-pinned helper and bootstrap, validates every registered assertion as
// a typed helper call at its exact frozen line through the embedded harness
// under the pinned closure, proves the native TestLoader discovery of the
// complete TestCase.id() inventory and freezes the exact argv and closed
// environment.
func (a *PythonAdapter) Prepare(ctx context.Context, req tdd.SuiteRequest) (tdd.PreparedSuite, error) {
	if req.Suite.Adapter != AdapterPythonUnittest {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_ADAPTER: suite %s declares adapter %q, not %s", req.Suite.ID, req.Suite.Adapter, AdapterPythonUnittest)
	}
	if req.Scope == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a live processscope scope", req.Suite.ID)
	}
	if req.Runtime == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the pinned CPython %s runtime handle", req.Suite.ID, pythonRequiredVersion)
	}
	handle, ok := req.Runtime.(*runtimeclosure.Python)
	if !ok {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s requires the exact approved CPython runtime closure handle, got %T", req.Suite.ID, req.Runtime)
	}
	// Reject absent/mismatched/mutated runtime before any suite work.
	if err := handle.Validate(ctx, req.Scope); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s runtime validation failed: %w", req.Suite.ID, err)
	}
	identity := handle.Identity()
	if req.Suite.Runtime != identity {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_VERSION: suite %s declares runtime %+v, but the pinned closure is %+v", req.Suite.ID, req.Suite.Runtime, identity)
	}
	if req.Inputs.Revalidate == nil || req.Inputs.Release == nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires held input view callbacks", req.Suite.ID)
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s held inputs failed revalidation: %w", req.Suite.ID, err)
	}
	if req.Suite.Root != protocol.RepositoryRoot {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: python-unittest/v1 suites are flat root suites; suite %s declares root %q", req.Suite.ID, req.Suite.Root)
	}
	if len(req.Suite.DependencyRoots) != 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("UNSUPPORTED_FEATURE: python-unittest/v1 resolves the frozen local dependency closure from the captured bundle; suite %s declares dependency roots %v", req.Suite.ID, req.Suite.DependencyRoots)
	}
	if len(req.Suite.Files) == 0 || len(req.Suite.Tests) == 0 {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s declares no files or no tests", req.Suite.ID)
	}
	if req.Scratch == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s requires a private scratch root", req.Suite.ID)
	}
	if err := validatePythonSuiteShape(&req.Suite); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s: %w", req.Suite.ID, err)
	}
	limits, err := processscope.NormalizeLimits(processscope.Limits(req.Limits))
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s limits: %w", req.Suite.ID, err)
	}
	for _, dir := range []string{"home", "tmp"} {
		if err := os.MkdirAll(filepath.Join(req.Scratch, dir), 0o700); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: preparing private %s root: %w", dir, err)
		}
	}
	// Verified bundle materialization: only a capture-produced BundleRef
	// carries a materialized store object root.
	objectRoot := req.Source.Materialized()
	if objectRoot == "" {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s source bundle was not produced by capture", req.Suite.ID)
	}
	storeRoot := filepath.Dir(filepath.Dir(objectRoot))
	projectID, err := goStoreProjectID(storeRoot)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: suite %s store identity: %w", req.Suite.ID, err)
	}
	suiteDir := filepath.Join(req.Scratch, "suite")
	if err := tdd.MaterializeBundle(ctx, storeRoot, projectID, req.Source.Ref(), suiteDir); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: materializing the captured bundle of suite %s: %w", req.Suite.ID, err)
	}
	// Byte-pinned asset materialization with round-trip checks.
	for name, pin := range PythonAssets() {
		var body []byte
		switch name {
		case PythonHelperFile:
			body = pythonHelperSource
		default:
			body = pythonBootstrapSource
		}
		if sum := fmt.Sprintf("%x", sha256.Sum256(body)); sum != pin {
			return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: embedded asset %s does not match its frozen pin sha256:%s", name, sum)
		}
		target := filepath.Join(suiteDir, name)
		if err := os.WriteFile(target, body, 0o644); err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: materializing the embedded asset %s: %w", name, err)
		}
		readback, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(readback, body) {
			return tdd.PreparedSuite{}, fmt.Errorf("CUSTODY_ERROR: the materialized asset %s does not round-trip its pinned bytes: %w", name, err)
		}
	}
	// No stale bytecode may exist in the prepared suite tree: CPython
	// executes planted unchecked-hash pyc even beside fresh sources.
	if err := rejectPythonBytecode(suiteDir); err != nil {
		return tdd.PreparedSuite{}, err
	}
	inventoryPath := filepath.Join(req.Scratch, "inventory.json")
	if err := writePythonInventory(inventoryPath, suiteDir, &req.Suite); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s inventory: %w", req.Suite.ID, err)
	}
	probeEnv, err := buildPythonEnv(req.Scratch, req.Suite.Environment)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s environment: %w", req.Suite.ID, err)
	}
	bootstrap := filepath.Join(suiteDir, PythonBootstrapFile)
	// Static validation probe: every registered assertion call site, the
	// closed suite grammar and the unsupported-feature surface, under the
	// pinned closure in isolated mode.
	validateOut, err := pythonRunProbe(ctx, req.Scope, handle, suiteDir, bootstrap, []string{"validate", inventoryPath}, probeEnv, limits)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s frozen sources: %w", req.Suite.ID, err)
	}
	if err := pythonProbeVerdict(validateOut, "validate"); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s frozen sources: %w", req.Suite.ID, err)
	}
	// Native discovery probe: the real TestLoader enumerates the complete
	// TestCase.id() inventory which must reconcile exactly with the
	// declared set before any execution.
	discoverOut, err := pythonRunProbe(ctx, req.Scope, handle, suiteDir, bootstrap, []string{"discover", inventoryPath}, probeEnv, limits)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s native discovery: %w", req.Suite.ID, err)
	}
	if err := pythonReconcileDiscovery(discoverOut, &req.Suite); err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("suite %s native discovery: %w", req.Suite.ID, err)
	}
	digests := map[string]string{}
	for _, name := range append(append([]string{}, req.Suite.Files...), PythonHelperFile, PythonBootstrapFile) {
		sum, err := goHashFile(filepath.Join(suiteDir, filepath.FromSlash(name)))
		if err != nil {
			return tdd.PreparedSuite{}, fmt.Errorf("STALE_INPUT: hashing prepared source %s: %w", name, err)
		}
		digests[name] = sum
	}
	env, err := buildPythonEnv(req.Scratch, req.Suite.Environment)
	if err != nil {
		return tdd.PreparedSuite{}, fmt.Errorf("INVALID_SCHEMA: suite %s environment: %w", req.Suite.ID, err)
	}
	suiteCopy := req.Suite
	return tdd.NewPreparedSuite(&pythonPrepared{
		suite: &suiteCopy, inputs: req.Inputs, scope: req.Scope, scratch: req.Scratch,
		suiteDir: suiteDir, inventory: inventoryPath, report: filepath.Join(req.Scratch, "report.jsonl"),
		pythonBin: handle.Binary(), closure: identity.Closure,
		// -B, not just PYTHONDONTWRITEBYTECODE in env: -I is isolated mode,
		// which makes CPython ignore every PYTHON* environment variable, so
		// the env setting below cannot reach this interpreter. Without -B it
		// writes __pycache__ into its own stdlib, and runtimeclosure
		// fingerprints that tree into the pinned closure digest.
		argv: []string{"-I", "-B", bootstrap, "run", inventoryPath, filepath.Join(req.Scratch, "report.jsonl")},
		env:  env, limits: limits, fileDigests: digests,
	}), nil
}

// Run implements tdd.Adapter: it executes the prepared suite as one guarded
// custody job through the pinned CPython interpreter in isolated mode,
// decodes the closed harness JSONL report vocabulary, reconciles the
// complete lifecycle against the frozen inventory with witness-bound
// assertion outcomes, enforces exit concordance and re-verifies the
// prepared sources against late mutation.
func (a *PythonAdapter) Run(ctx context.Context, prepared tdd.PreparedSuite, sink tdd.EventSink) (tdd.Execution, error) {
	state, ok := prepared.PreparedState().(*pythonPrepared)
	if !ok || state == nil {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: Run requires this adapter's own prepared state; a foreign carrier carries no authority")
	}
	if state.ran {
		return tdd.Execution{}, fmt.Errorf("INVALID_SCHEMA: the prepared suite state is single-use")
	}
	state.ran = true
	execution := tdd.Execution{Suite: state.suite.ID}
	if err := ctx.Err(); err != nil {
		return execution, fmt.Errorf("CUSTODY_ERROR: run canceled before launch: %w", err)
	}
	if err := state.verifyUnchanged(); err != nil {
		return execution, err
	}
	if err := state.inputs.Revalidate(); err != nil {
		return execution, fmt.Errorf("STALE_INPUT: held inputs changed before execution: %w", err)
	}
	if err := rejectPythonBytecode(state.suiteDir); err != nil {
		execution.Outcome = "error"
		return execution, err
	}
	var stdout, stderr bytes.Buffer
	result, err := goGuardedChild(ctx, state.scope, processscope.Command{
		Executable:    state.pythonBin,
		Args:          state.argv,
		Dir:           state.suiteDir,
		Env:           state.env,
		RuntimeDigest: state.closure,
		DeadlineMS:    state.limits.WallMS,
	}, &stdout, &stderr, state.limits.StdoutBytes, state.limits.StderrBytes)
	if err != nil {
		var scopeErr *processscope.Error
		if errors.As(err, &scopeErr) {
			switch scopeErr.Code {
			case processscope.CodeTimeout:
				execution.Outcome = "error"
				return execution, fmt.Errorf("TIMEOUT: the native unittest process exceeded its wall deadline: %w", err)
			case processscope.CodeOutputLimit:
				execution.Outcome = "error"
				return execution, fmt.Errorf("OUTPUT_LIMIT: the native unittest process exceeded its stream budget: %w", err)
			}
		}
		execution.Outcome = "error"
		return execution, fmt.Errorf("CUSTODY_ERROR: native unittest invocation failed under custody: %w (stderr: %s)", err, boundedString(stderr.String(), 2048))
	}
	exit := int64(result.ExitCode)
	execution.ExitCode = &exit
	execution.StdoutDigest = fmt.Sprintf("%x", sha256.Sum256(stdout.Bytes()))
	execution.StderrDigest = fmt.Sprintf("%x", sha256.Sum256(stderr.Bytes()))
	// Late source mutation is checked before reconciling the evidence
	// stream so a mutated suite cannot hide behind stream diagnostics.
	if err := state.verifyUnchanged(); err != nil {
		execution.Outcome = "error"
		return execution, err
	}
	if err := rejectPythonBytecode(state.suiteDir); err != nil {
		execution.Outcome = "error"
		return execution, err
	}
	raw, err := os.ReadFile(state.report)
	if err != nil {
		execution.Outcome = "error"
		return execution, fmt.Errorf("INCOMPLETE_EVENTS: the harness report is absent: %w", err)
	}
	if int64(len(raw)) > pythonReportMaxBytes {
		execution.Outcome = "error"
		return execution, fmt.Errorf("OUTPUT_LIMIT: the harness report exceeds its bound")
	}
	records, err := parsePythonReport(raw)
	var rec pythonReconciliation
	var reconcileErr error
	if err == nil {
		rec, reconcileErr = reconcilePythonReport(records, state.suite)
	} else {
		reconcileErr = err
	}
	if reconcileErr != nil {
		execution.Events = rec.Events
		execution.Outcome = "error"
		a.emitPython(sink, rec.Events)
		return execution, fmt.Errorf("suite %s native stream: %w", state.suite.ID, reconcileErr)
	}
	if !result.Completed || result.Signal != "" {
		execution.Outcome = "error"
		a.emitPython(sink, rec.Events)
		return execution, fmt.Errorf("CUSTODY_ERROR: native unittest invocation did not complete cleanly (completed=%v signal=%q exit=%d)", result.Completed, result.Signal, result.ExitCode)
	}
	switch {
	case rec.Outcome == "pass" && result.ExitCode == 0:
	case rec.Outcome == protocol.OutcomeAssertionFail && result.ExitCode == 1:
	default:
		execution.Outcome = "error"
		a.emitPython(sink, rec.Events)
		return execution, fmt.Errorf("UNEXPECTED_FAILURE: suite %s exit status %d does not concord with reconciled outcome %q", state.suite.ID, result.ExitCode, rec.Outcome)
	}
	if int64(len(rec.Events)) > state.limits.EventCount {
		execution.Outcome = "error"
		return execution, fmt.Errorf("OUTPUT_LIMIT: normalized event count exceeds the budget")
	}
	execution.Events = rec.Events
	execution.Outcome = rec.Outcome
	execution.Custody = tdd.CustodyReport{Status: processscope.StatusCleaned}
	a.emitPython(sink, rec.Events)
	return execution, nil
}

func (a *PythonAdapter) emitPython(sink tdd.EventSink, events []tdd.Event) {
	if sink == nil {
		return
	}
	for _, e := range events {
		if err := sink(e); err != nil {
			return
		}
	}
}

// verifyUnchanged re-hashes the prepared sources and assets after the run.
func (p *pythonPrepared) verifyUnchanged() error {
	for name, want := range p.fileDigests {
		got, err := goHashFile(filepath.Join(p.suiteDir, filepath.FromSlash(name)))
		if err != nil || got != want {
			return fmt.Errorf("STALE_INPUT: prepared source %s changed after preparation (want sha256:%s, got %s: %w)", name, want, got, err)
		}
	}
	return nil
}

// pythonRunProbe executes one embedded-bootstrap subcommand as a guarded
// child custody job under the pinned interpreter and returns its stdout.
func pythonRunProbe(ctx context.Context, scope processscope.Scope, handle *runtimeclosure.Python, dir, bootstrap string, args []string, env []string, limits processscope.Limits) ([]byte, error) {
	deadline := limits.WallMS
	if deadline <= 0 || deadline > pythonProbeTimeoutMS {
		deadline = pythonProbeTimeoutMS
	}
	var stdout, stderr bytes.Buffer
	result, err := goGuardedChild(ctx, scope, processscope.Command{
		Executable: handle.Binary(),
		// -B for the same reason as the suite argv: isolated mode ignores
		// PYTHONDONTWRITEBYTECODE from the environment.
		Args:          append([]string{"-I", "-B", bootstrap}, args...),
		Dir:           dir,
		Env:           env,
		RuntimeDigest: handle.Identity().Closure,
		DeadlineMS:    deadline,
	}, &stdout, &stderr, 1<<20, 1<<20)
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("custody: %w (stderr: %s)", err, boundedString(stderr.String(), 2048))
	}
	if !result.Completed || result.ExitCode != 0 {
		return stdout.Bytes(), fmt.Errorf("exit=%d signal=%q stdout: %s stderr: %s", result.ExitCode, result.Signal, boundedString(stdout.String(), 2048), boundedString(stderr.String(), 2048))
	}
	return stdout.Bytes(), nil
}

// pythonProbeVerdict decodes the closed single-object verdict of the
// validate/discover subcommands.
func pythonProbeVerdict(raw []byte, kind string) error {
	var doc struct {
		Schema  string `json:"schema"`
		Kind    string `json:"kind"`
		Status  string `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
		File    string `json:"file"`
		Line    int64  `json:"line"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &doc); err != nil {
		return fmt.Errorf("INVALID_SCHEMA: the %s verdict is not a closed JSON object: %w", kind, err)
	}
	if doc.Schema != PythonHarnessSchema || doc.Kind != kind {
		return fmt.Errorf("INVALID_SCHEMA: the verdict is not the closed %s record", kind)
	}
	if doc.Status == "ok" {
		return nil
	}
	if doc.Status != "error" || doc.Code == "" {
		return fmt.Errorf("INVALID_SCHEMA: the %s verdict carries an unknown status %q", kind, doc.Status)
	}
	return fmt.Errorf("%s: %s (%s:%d)", doc.Code, doc.Message, doc.File, doc.Line)
}

// pythonReconcileDiscovery reconciles the native TestLoader enumeration
// against the declared inventory: exact set equality in both directions.
func pythonReconcileDiscovery(raw []byte, suite *tdd.Suite) error {
	var doc struct {
		Schema  string   `json:"schema"`
		Kind    string   `json:"kind"`
		Status  string   `json:"status"`
		Code    string   `json:"code"`
		Message string   `json:"message"`
		Tests   []string `json:"tests"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(raw), &doc); err != nil {
		return fmt.Errorf("INVALID_SCHEMA: the discovery verdict is not a closed JSON object: %w", err)
	}
	if doc.Schema != PythonHarnessSchema || doc.Kind != "discover" {
		return fmt.Errorf("INVALID_SCHEMA: the verdict is not the closed discovery record")
	}
	if doc.Status != "ok" {
		if doc.Status == "error" && doc.Code != "" {
			return fmt.Errorf("%s: %s", doc.Code, doc.Message)
		}
		return fmt.Errorf("INVALID_SCHEMA: the discovery verdict carries an unknown status %q", doc.Status)
	}
	declared := map[string]bool{}
	for i := range suite.Tests {
		declared[pythonNativeID(suite.Tests[i].Native)] = true
	}
	discovered := map[string]bool{}
	for _, id := range doc.Tests {
		if discovered[id] {
			return fmt.Errorf("DUPLICATE_TEST: the native loader enumerated %s more than once", id)
		}
		discovered[id] = true
		if !declared[id] {
			return fmt.Errorf("DUPLICATE_TEST: native identity %q is outside the frozen inventory of this suite", id)
		}
	}
	for id := range declared {
		if !discovered[id] {
			return fmt.Errorf("MISSING_TEST: declared native identity %s was never discovered by the native loader", id)
		}
	}
	return nil
}

// validatePythonSuiteShape validates the closed suite shape: python-native
// identities bound to declared flat .py files, the closed assertion
// registration grammar and duplicate rejection.
func validatePythonSuiteShape(suite *tdd.Suite) error {
	files := map[string]bool{}
	for _, file := range suite.Files {
		if filepath.ToSlash(file) != filepath.Base(file) || !strings.HasSuffix(file, ".py") {
			return fmt.Errorf("UNSUPPORTED_FEATURE: suite declares non-flat or non-Python file %q", file)
		}
		files[file] = true
	}
	seenTests := map[string]bool{}
	seenAssertions := map[string]bool{}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		native := test.Native
		if !pythonModulePattern.MatchString(native.Module) || !pythonClassPattern.MatchString(native.Class) || !pythonMethodPattern.MatchString(native.Method) {
			return fmt.Errorf("INVALID_SCHEMA: test %s declares an invalid python native identity %s.%s.%s", test.ID, native.Module, native.Class, native.Method)
		}
		if native.Package != "" || native.Test != "" || native.Source != "" || len(native.Path) != 0 || native.Line != 0 || native.Column != 0 || native.Name != "" || native.File != "" {
			return fmt.Errorf("UNSUPPORTED_FEATURE: test %s carries a non-python native identity", test.ID)
		}
		file := native.Module + ".py"
		if !files[file] {
			return fmt.Errorf("INVALID_SCHEMA: test %s binds to undeclared source %s", test.ID, file)
		}
		if test.Source != file {
			return fmt.Errorf("INVALID_SCHEMA: test %s declares source %s, but its native identity binds to %s", test.ID, test.Source, file)
		}
		identity := pythonNativeID(native)
		if seenTests[identity] {
			return fmt.Errorf("DUPLICATE_TEST: native identity %s is declared more than once", identity)
		}
		seenTests[identity] = true
		for j := range test.Assertions {
			assertion := &test.Assertions[j]
			if assertion.Helper != tdd.AssertionHelperV1 {
				return fmt.Errorf("UNSUPPORTED_FEATURE: assertion %s uses helper %q; only %s is closed", assertion.ID, assertion.Helper, tdd.AssertionHelperV1)
			}
			if !pythonAssertionIDPattern.MatchString(assertion.ID) {
				return fmt.Errorf("INVALID_SCHEMA: assertion identity %q is not a closed ID", assertion.ID)
			}
			if assertion.Source != test.Source {
				return fmt.Errorf("INVALID_SCHEMA: assertion %s binds to source %s outside its test's source %s", assertion.ID, assertion.Source, test.Source)
			}
			if assertion.Line < 1 {
				return fmt.Errorf("ASSERTION_MISMATCH: assertion %s line %d is not positive", assertion.ID, assertion.Line)
			}
			if seenAssertions[assertion.ID] {
				return fmt.Errorf("DUPLICATE_TEST: assertion %s is registered more than once", assertion.ID)
			}
			seenAssertions[assertion.ID] = true
		}
	}
	return nil
}

// writePythonInventory writes the closed bootstrap inventory document.
func writePythonInventory(path, suiteDir string, suite *tdd.Suite) error {
	type inventoryAssertion struct {
		ID   string `json:"id"`
		Line int64  `json:"line"`
	}
	type inventoryTest struct {
		Module     string               `json:"module"`
		Class      string               `json:"class"`
		Method     string               `json:"method"`
		File       string               `json:"file"`
		Assertions []inventoryAssertion `json:"assertions"`
	}
	ordered := append([]tdd.Test{}, suite.Tests...)
	sort.Slice(ordered, func(i, j int) bool {
		a := pythonNativeID(ordered[i].Native)
		b := pythonNativeID(ordered[j].Native)
		return a < b
	})
	doc := struct {
		Schema string          `json:"schema"`
		Root   string          `json:"root"`
		Tests  []inventoryTest `json:"tests"`
	}{Schema: "machinery.tdd.suite.python/v1", Root: suiteDir}
	for _, test := range ordered {
		entry := inventoryTest{Module: test.Native.Module, Class: test.Native.Class, Method: test.Native.Method, File: test.Source}
		for _, assertion := range test.Assertions {
			entry.Assertions = append(entry.Assertions, inventoryAssertion{ID: assertion.ID, Line: assertion.Line})
		}
		doc.Tests = append(doc.Tests, entry)
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

// rejectPythonBytecode fails closed on any __pycache__ directory or .pyc
// file inside the prepared suite tree: CPython executes planted
// unchecked-hash bytecode even beside fresh sources, so stale bytecode can
// never be accepted as evidence.
func rejectPythonBytecode(suiteDir string) error {
	var found []string
	err := filepath.WalkDir(suiteDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() && name == "__pycache__" {
			found = append(found, path)
		}
		if !entry.IsDir() && strings.HasSuffix(name, ".pyc") {
			found = append(found, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("STALE_INPUT: walking the prepared suite for stale bytecode: %w", err)
	}
	if len(found) != 0 {
		return fmt.Errorf("STALE_INPUT: stale bytecode artifacts in the prepared suite tree: %v", found)
	}
	return nil
}

// buildPythonEnv builds the closed native environment: private HOME/TMPDIR,
// UTC, deterministic hash seed, bytecode and user-site disabled, minimal
// PATH and the explicitly declared suite variables; PYTHON* and MACHINERY*
// injection is never inherited.
func buildPythonEnv(scratch string, extra []tdd.EnvironmentVar) ([]string, error) {
	declared := map[string]string{}
	for _, item := range extra {
		if !pythonEnvNamePattern.MatchString(item.Name) {
			return nil, fmt.Errorf("INVALID_SCHEMA: declared environment name %q is not a closed variable name", item.Name)
		}
		if len(item.Value) > 65536 {
			return nil, fmt.Errorf("INVALID_SCHEMA: declared environment value of %s exceeds the bounded size", item.Name)
		}
		if pythonClosedEnvKeys[item.Name] || strings.HasPrefix(item.Name, "PYTHON") || strings.HasPrefix(item.Name, "MACHINERY_") {
			return nil, fmt.Errorf("INVALID_SCHEMA: declared environment may not override the closed key %s", item.Name)
		}
		if _, duplicate := declared[item.Name]; duplicate {
			return nil, fmt.Errorf("INVALID_SCHEMA: declared environment name %s is duplicated", item.Name)
		}
		declared[item.Name] = item.Value
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	env := []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(scratch, "home"),
		"TMPDIR=" + filepath.Join(scratch, "tmp"),
		"TZ=UTC",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"NO_COLOR=1",
		"PYTHONHASHSEED=0",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONNOUSERSITE=1",
	}
	for _, name := range names {
		env = append(env, name+"="+declared[name])
	}
	return env, nil
}

// pythonNativeID renders the exact TestCase.id() spelling of a declared
// native identity.
func pythonNativeID(n tdd.NativeID) string {
	return n.Module + "." + n.Class + "." + n.Method
}

// pythonWitness is one decoded machinery-check/v1 witness record.
type pythonWitness struct {
	assertion string
	test      string
	file      string
	line      int64
	condition bool
	thrown    string
}

// parsePythonWitness decodes one closed witness record line.
func parsePythonWitness(line string) (pythonWitness, bool) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &doc); err != nil {
		return pythonWitness{}, false
	}
	if len(doc) != 7 {
		return pythonWitness{}, false
	}
	var schema, kind, assertion, test, site, thrown string
	var condition bool
	for key, raw := range doc {
		switch key {
		case "schema":
			if json.Unmarshal(raw, &schema) != nil {
				return pythonWitness{}, false
			}
		case "kind":
			if json.Unmarshal(raw, &kind) != nil {
				return pythonWitness{}, false
			}
		case "assertion":
			if json.Unmarshal(raw, &assertion) != nil {
				return pythonWitness{}, false
			}
		case "test":
			if json.Unmarshal(raw, &test) != nil {
				return pythonWitness{}, false
			}
		case "site":
			if json.Unmarshal(raw, &site) != nil {
				return pythonWitness{}, false
			}
		case "condition":
			if json.Unmarshal(raw, &condition) != nil {
				return pythonWitness{}, false
			}
		case "thrown":
			if string(raw) == "null" {
				continue
			}
			if json.Unmarshal(raw, &thrown) != nil {
				return pythonWitness{}, false
			}
		default:
			return pythonWitness{}, false
		}
	}
	if schema != PythonWitnessSchema || kind != "witness" || assertion == "" || test == "" {
		return pythonWitness{}, false
	}
	if !pythonAssertionIDPattern.MatchString(assertion) {
		return pythonWitness{}, false
	}
	file, lineText, found := strings.Cut(site, ":")
	if !found || file == "" || strings.ContainsAny(file, `/\`) {
		return pythonWitness{}, false
	}
	lineNumber, err := strconvParseInt(lineText)
	if err != nil || lineNumber < 1 {
		return pythonWitness{}, false
	}
	if thrown != "" && thrown != "AssertionError" {
		return pythonWitness{}, false
	}
	return pythonWitness{assertion: assertion, test: test, file: file, line: lineNumber, condition: condition, thrown: thrown}, true
}

func strconvParseInt(text string) (int64, error) {
	return strconv.ParseInt(text, 10, 64)
}

// pythonReportRecord is one decoded harness JSONL record.
type pythonReportRecord struct {
	Schema              string
	Kind                string
	Tests               []string
	Test                string
	Outcome             string
	Exception           string
	Feature             string
	Code                string
	Message             string
	TestsRun            int64
	Failures            int64
	Errors              int64
	Skipped             int64
	ExpectedFailures    int64
	UnexpectedSuccesses int64
	Success             bool
	witness             *pythonWitness
}

// parsePythonReport decodes the closed harness JSONL stream; unknown
// schemas, kinds and keys, malformed lines and missing, duplicated or
// non-terminal sentinels fail closed.
func parsePythonReport(raw []byte) ([]pythonReportRecord, error) {
	text := strings.TrimSuffix(string(raw), "\n")
	if text == "" {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the harness report is empty")
	}
	lines := strings.Split(text, "\n")
	if int64(len(lines)) > pythonMaxReportLines {
		return nil, fmt.Errorf("OUTPUT_LIMIT: the harness report exceeds %d lines", pythonMaxReportLines)
	}
	records := make([]pythonReportRecord, 0, len(lines))
	sentinel := -1
	for i, line := range lines {
		if line == "" {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d is empty", i+1)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &doc); err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d is not a JSON object: %w", i+1, err)
		}
		var schema, kind string
		if err := json.Unmarshal(doc["schema"], &schema); err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries a non-string schema", i+1)
		}
		if err := json.Unmarshal(doc["kind"], &kind); err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries a non-string kind", i+1)
		}
		if schema != PythonHarnessSchema && schema != PythonWitnessSchema {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries unknown schema %q", i+1, schema)
		}
		if kind == "bootstrap-end" {
			if len(doc) != 2 {
				return nil, fmt.Errorf("INVALID_SCHEMA: the terminal sentinel carries unknown keys")
			}
			if sentinel != -1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: duplicated terminal sentinel")
			}
			if i != len(lines)-1 {
				return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is not terminal")
			}
			sentinel = i
			records = append(records, pythonReportRecord{Schema: schema, Kind: kind})
			continue
		}
		if sentinel != -1 {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report carries records after the terminal sentinel")
		}
		closedKeys, known := pythonReportKeys[kind]
		if !known || (schema == PythonWitnessSchema) != (kind == "witness") {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries unknown kind %q for schema %q", i+1, kind, schema)
		}
		delete(doc, "schema")
		delete(doc, "kind")
		for key := range doc {
			if !closedKeys[key] {
				return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries unknown key %q for kind %q", i+1, key, kind)
			}
		}
		record := pythonReportRecord{Schema: schema, Kind: kind}
		var err error
		switch kind {
		case "witness":
			witness, ok := parsePythonWitness(line)
			if !ok {
				return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d is not a closed witness record", i+1)
			}
			record.witness = &witness
			record.Test = witness.test
		case "suite-start":
			err = json.Unmarshal(doc["tests"], &record.Tests)
		case "test-start", "unsupported-feature":
			record.Test = stringField(doc, "test")
			record.Feature = stringField(doc, "feature")
		case "test-end":
			record.Test = stringField(doc, "test")
			record.Outcome = stringField(doc, "outcome")
			if raw, present := doc["exception"]; present {
				_ = json.Unmarshal(raw, &record.Exception)
			}
			if !pythonReportOutcomes[record.Outcome] {
				return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries unknown outcome %q", i+1, record.Outcome)
			}
		case "suite-error":
			record.Code = stringField(doc, "code")
			record.Message = stringField(doc, "message")
			if record.Code == "" {
				return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries an empty suite-error code", i+1)
			}
		case "suite-end":
			record.TestsRun = intField(doc, "tests_run")
			record.Failures = intField(doc, "failures")
			record.Errors = intField(doc, "errors")
			record.Skipped = intField(doc, "skipped")
			record.ExpectedFailures = intField(doc, "expected_failures")
			record.UnexpectedSuccesses = intField(doc, "unexpected_successes")
			if raw, present := doc["success"]; present {
				if err := json.Unmarshal(raw, &record.Success); err != nil {
					return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d carries a non-boolean success", i+1)
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("INVALID_SCHEMA: harness report line %d is malformed for kind %q: %w", i+1, kind, err)
		}
		records = append(records, record)
	}
	if sentinel == -1 {
		return nil, fmt.Errorf("INCOMPLETE_EVENTS: the terminal sentinel is missing; the report is truncated")
	}
	return records, nil
}

// stringField and intField are tiny closed decoders of the report
// vocabulary.
func stringField(doc map[string]json.RawMessage, key string) string {
	raw, present := doc[key]
	if !present {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func intField(doc map[string]json.RawMessage, key string) int64 {
	raw, present := doc[key]
	if !present {
		return -1
	}
	var value int64
	if json.Unmarshal(raw, &value) != nil {
		return -1
	}
	return value
}

// pythonReconciliation is the reconciled normalized result of one harness
// report.
type pythonReconciliation struct {
	Events     []tdd.Event
	Outcome    string
	Discovered []tdd.NativeID
	Started    []tdd.NativeID
	Completed  []tdd.NativeID
	Assertions []tdd.AssertionOutcome
}

type pythonTestState struct {
	test         *tdd.Test
	started      bool
	terminal     string // "" | pass | assertion-fail
	witnessed    map[string]bool
	witnessFalse bool
}

// reconcilePythonReport reconciles the harness lifecycle against the
// complete frozen inventory with the registered assertion witnesses and
// emits the exact normalized machinery.tdd.event/v1 sequence.
func reconcilePythonReport(records []pythonReportRecord, suite *tdd.Suite) (pythonReconciliation, error) {
	rec := pythonReconciliation{}
	if len(suite.Tests) == 0 {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: empty declared selection cannot yield a successful execution")
	}
	states := map[string]*pythonTestState{}
	assertionOwner := map[string]*tdd.Assertion{}
	assertionTest := map[string]*tdd.Test{}
	assertionFalse := map[string]bool{}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		identity := pythonNativeID(test.Native)
		if _, duplicate := states[identity]; duplicate {
			return rec, fmt.Errorf("INVALID_SCHEMA: native identity %s is declared more than once", identity)
		}
		states[identity] = &pythonTestState{test: test, witnessed: map[string]bool{}}
		for j := range test.Assertions {
			assertion := &test.Assertions[j]
			if _, duplicate := assertionOwner[assertion.ID]; duplicate {
				return rec, fmt.Errorf("INVALID_SCHEMA: assertion %s is registered more than once", assertion.ID)
			}
			assertionOwner[assertion.ID] = assertion
			assertionTest[assertion.ID] = test
		}
	}
	sequence := int64(0)
	emit := func(kind string, test *tdd.Test, assertionID, outcome, source string, line int64) {
		sequence++
		event := tdd.Event{
			Schema: PythonEventSchema, Sequence: sequence, Suite: suite.ID,
			Kind: kind, Assertion: assertionID, Outcome: outcome, Source: source, Line: line,
		}
		if test != nil {
			native := test.Native
			event.Native = &native
		}
		rec.Events = append(rec.Events, event)
	}
	var order []string
	seenSuiteStart := false
	seenSuiteEnd := false
	active := ""
	sawFailure := false
	for _, record := range records {
		switch record.Kind {
		case "bootstrap-end":
			continue
		case "suite-start":
			if seenSuiteStart {
				return rec, fmt.Errorf("INVALID_SCHEMA: duplicated suite-start record")
			}
			seenSuiteStart = true
			for _, identity := range record.Tests {
				if _, declared := states[identity]; !declared {
					return rec, fmt.Errorf("DUPLICATE_TEST: native identity %q is outside the frozen inventory of this suite", identity)
				}
				order = append(order, identity)
			}
			if len(order) != len(states) {
				for identity := range states {
					missing := true
					for _, listed := range order {
						if listed == identity {
							missing = false
						}
					}
					if missing {
						return rec, fmt.Errorf("MISSING_TEST: declared identity %s never entered the native suite", identity)
					}
				}
			}
			emit("suite-start", nil, "", "", "", 0)
			for _, identity := range order {
				test := states[identity].test
				native := test.Native
				rec.Discovered = append(rec.Discovered, native)
				emit("discovered", test, "", "", "", 0)
			}
		case "test-start":
			if !seenSuiteStart || seenSuiteEnd {
				return rec, fmt.Errorf("INVALID_SCHEMA: test-start of %q is outside the suite lifecycle", record.Test)
			}
			state, declared := states[record.Test]
			if !declared {
				return rec, fmt.Errorf("DUPLICATE_TEST: native identity %q is outside the frozen inventory", record.Test)
			}
			if state.started || state.terminal != "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: native test %s started more than once", record.Test)
			}
			if active != "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: native test %s started while %s is still active", record.Test, active)
			}
			state.started = true
			active = record.Test
			rec.Started = append(rec.Started, state.test.Native)
			emit("test-start", state.test, "", "", "", 0)
		case "witness":
			if active == "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: witness for %s arrived outside an active test lifecycle", record.witness.assertion)
			}
			state := states[active]
			witness := *record.witness
			assertion, known := assertionOwner[witness.assertion]
			if !known {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness carries assertion %q outside the registered inventory", witness.assertion)
			}
			owner := assertionTest[witness.assertion]
			if pythonNativeID(owner.Native) != active || witness.test != active {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s claims test %q but arrived inside %q", witness.assertion, witness.test, active)
			}
			if filepath.ToSlash(filepath.Base(assertion.Source)) != witness.file || assertion.Line != witness.line {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: witness of %s reports site %s:%d, want the registered call site %s:%d", witness.assertion, witness.file, witness.line, filepath.Base(assertion.Source), assertion.Line)
			}
			if state.witnessed[witness.assertion] {
				return rec, fmt.Errorf("ASSERTION_MISMATCH: assertion %s was witnessed more than once", witness.assertion)
			}
			state.witnessed[witness.assertion] = true
			if !witness.condition {
				state.witnessFalse = true
				assertionFalse[witness.assertion] = true
			}
			outcome := protocol.OutcomePass
			if !witness.condition {
				outcome = protocol.OutcomeAssertionFail
			}
			emit("assertion", state.test, witness.assertion, outcome, assertion.Source, assertion.Line)
		case "test-end":
			if active == "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: terminal of %q arrived without an active test lifecycle", record.Test)
			}
			if record.Test != active {
				return rec, fmt.Errorf("INVALID_SCHEMA: terminal of %q does not match the active identity %q", record.Test, active)
			}
			state := states[active]
			if state.terminal != "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: native test %s reported more than one terminal", record.Test)
			}
			switch record.Outcome {
			case "pass":
				if state.witnessFalse {
					return rec, fmt.Errorf("ASSERTION_MISMATCH: witness reported a false condition but native test %s passed", record.Test)
				}
				state.terminal = "pass"
				emit("test-end", state.test, "", protocol.OutcomePass, "", 0)
			case "assertion-fail":
				if record.Exception != "AssertionError" {
					return rec, fmt.Errorf("UNEXPECTED_FAILURE: test %s failed with %q, want the native AssertionError of the registered helper", record.Test, record.Exception)
				}
				if !state.witnessFalse {
					return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s has no witnessed registered assertion failure", record.Test)
				}
				state.terminal = "fail"
				sawFailure = true
				emit("test-end", state.test, "", protocol.OutcomeAssertionFail, "", 0)
			case "skipped":
				return rec, fmt.Errorf("UNSUPPORTED_FEATURE: declared test %s was skipped; required skips can never become success", record.Test)
			case "expected-failure":
				return rec, fmt.Errorf("UNSUPPORTED_FEATURE: declared test %s reported an expected failure; xfail shortcuts never qualify", record.Test)
			case "unexpected-success":
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: declared test %s reported an unexpected success under an expected-failure marker", record.Test)
			default:
				return rec, fmt.Errorf("UNEXPECTED_FAILURE: declared test %s errored with %q", record.Test, record.Exception)
			}
			rec.Completed = append(rec.Completed, state.test.Native)
			active = ""
		case "unsupported-feature":
			return rec, fmt.Errorf("UNSUPPORTED_FEATURE: the native lifecycle reported the unsupported feature %q inside %s", record.Feature, record.Test)
		case "suite-error":
			return rec, fmt.Errorf("%s: harness reported: %s", record.Code, record.Message)
		case "suite-end":
			if !seenSuiteStart || seenSuiteEnd || active != "" {
				return rec, fmt.Errorf("INVALID_SCHEMA: suite-end is not in a complete suite lifecycle")
			}
			seenSuiteEnd = true
			failures := 0
			for i := range suite.Tests {
				if states[pythonNativeID(suite.Tests[i].Native)].terminal == "fail" {
					failures++
				}
			}
			if record.TestsRun != int64(len(states)) || record.Failures != int64(failures) || record.Errors != 0 || record.Skipped != 0 || record.ExpectedFailures != 0 || record.UnexpectedSuccesses != 0 {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: suite-end accounting disagrees with the reconciled lifecycle (run=%d failures=%d errors=%d skipped=%d xfail=%d uxsuccess=%d)", record.TestsRun, record.Failures, record.Errors, record.Skipped, record.ExpectedFailures, record.UnexpectedSuccesses)
			}
			if record.Success != (failures == 0) {
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: the native summary success flag disagrees with the reconciled failures")
			}
		default:
			return rec, fmt.Errorf("INVALID_SCHEMA: harness report carries unknown kind %q", record.Kind)
		}
	}
	if !seenSuiteStart {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the harness report carries no suite lifecycle")
	}
	if !seenSuiteEnd {
		return rec, fmt.Errorf("INCOMPLETE_EVENTS: the harness report carries no complete suite accounting")
	}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		state := states[pythonNativeID(test.Native)]
		if !state.started || state.terminal == "" {
			return rec, fmt.Errorf("INCOMPLETE_EVENTS: declared test %s never executed to a terminal state (started=%v terminal=%q)", pythonNativeID(test.Native), state.started, state.terminal)
		}
		for _, assertion := range test.Assertions {
			if !state.witnessed[assertion.ID] {
				if state.terminal == "fail" {
					return rec, fmt.Errorf("UNEXPECTED_FAILURE: failing test %s provides no witnessed registered assertion for %s", pythonNativeID(test.Native), assertion.ID)
				}
				return rec, fmt.Errorf("INCOMPLETE_EVENTS: registered assertion %s never executed in test %s", assertion.ID, pythonNativeID(test.Native))
			}
		}
	}
	for i := range suite.Tests {
		test := &suite.Tests[i]
		for _, assertion := range test.Assertions {
			outcome := protocol.OutcomePass
			if assertionFalse[assertion.ID] {
				outcome = protocol.OutcomeAssertionFail
			}
			rec.Assertions = append(rec.Assertions, tdd.AssertionOutcome{
				Test: tdd.TestRef{Suite: suite.ID, Test: test.ID}, ID: assertion.ID, Outcome: outcome,
			})
		}
	}
	if sawFailure {
		rec.Outcome = protocol.OutcomeAssertionFail
	} else {
		rec.Outcome = protocol.OutcomePass
	}
	emit("suite-end", nil, "", rec.Outcome, "", 0)
	return rec, nil
}
