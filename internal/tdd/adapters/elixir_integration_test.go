package adapters

// RED contract for MAC-8yai (native subjects): the closed elixir-exunit/v1
// adapter executes REAL Elixir suites — actual Mix compilation with
// warnings-as-errors, native ExUnit execution, embedded Machinery reporter,
// processscope custody, normalized machinery.tdd.event/v1 accounting —
// through the REAL production chain (InitStore -> Capture -> OpenElixir +
// Validate -> Prepare -> Run -> process-free Close). Compile failures,
// skips, setup assertions, early VM halts, unwitnessed failures, stale
// harness bytes, site mismatches, runtime mismatch and never-settling
// tests are rejected. The required contributor lane fragment executes
// alongside the frozen catalog.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
)

// laneFragmentAssuranceElixir is the exact frozen fragment GREEN ships at
// testdata/integration-lanes/assurance-elixir.json.
const laneFragmentAssuranceElixir = `{
  "schema": "machinery.assurance.lane/v1",
  "fragment": "assurance-elixir",
  "owner": "MAC-8yai",
  "suites": [
    {
      "id": "assurance-elixir-conformance",
      "lane": "assurance",
      "adapter": "elixir-exunit/v1",
      "kind": "native-conformance",
      "package": ".",
      "source_files": [
        "internal/tdd/adapters/assets/elixir/conformance/conformance_test.exs"
      ],
      "tests": [
        "conformance witness executes native assertion",
        "conformance parent identity conformance nested identity",
        "conformance multiple assertions"
      ],
      "runtimes": ["elixir"],
      "timeout": "5m",
      "stdout_limit": 1048576,
      "stderr_limit": 1048576
    }
  ]
}
`

// The package TestMain (processscope internal-IO dispatcher) is defined once
// in go_integration_test.go and serves every adapter suite.

func exRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func exOpenAdapterScope(t *testing.T) processscope.Scope {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	scope, err := processscope.Open(ctx, processscope.Options{
		HelperExecutable: exe,
		HelperDigest:     fmt.Sprintf("%x", sum),
		ScratchRoot:      t.TempDir(),
		Limits:           processscope.Limits{Jobs: 4, WallMS: 1200000, CleanupMS: 30000},
	})
	if err != nil {
		t.Fatalf("open custody scope: %v", err)
	}
	return scope
}

func exCloseAdapterScope(t *testing.T, scope processscope.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	report, err := scope.Close(ctx)
	if err != nil || report.Status != processscope.StatusCleaned {
		t.Fatalf("custody close did not verify: %v %+v", err, report)
	}
}

func exAssetBytes(t *testing.T, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(exRepoRoot(t), "internal", "tdd", "adapters", "assets", "elixir", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// exChainResult is one completed production-chain suite execution.
type exChainResult struct {
	execution tdd.Execution
	events    []tdd.Event
}

// exRunCaptured captures frozen fixture files into a real content-addressed
// store bundle and executes the declared suite through the closed
// production chain on a fresh custody root (the lane's own per-suite
// pattern): pinned runtime closure validated under the live scope, adapter
// Prepare over the captured bundle, adapter Run with the recording sink.
func exRunCaptured(t *testing.T, ctx context.Context, files map[string][]byte, suite tdd.Suite, limits tdd.Limits, mutatePrepared func(scratch string)) (exChainResult, error) {
	t.Helper()
	scope := exOpenAdapterScope(t)
	defer exCloseAdapterScope(t, scope)
	var result exChainResult
	src := t.TempDir()
	control := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(src, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return result, err
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			return result, err
		}
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o600); err != nil {
		return result, err
	}
	released := 0
	inputs := tdd.InputView{
		SourceRoot: src, DesignPath: ".", ControlRoot: control,
		Revalidate: func() error { return nil },
		Release:    func() error { released++; return nil },
	}
	handle, err := runtimeclosure.OpenElixir(ctx, runtimeclosure.ElixirRequest{})
	if err != nil {
		return result, fmt.Errorf("pinned Elixir closure: %w", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Validate(ctx, scope); err != nil {
		return result, fmt.Errorf("pinned Elixir closure validation: %w", err)
	}
	suite.Runtime = handle.Identity()
	store := filepath.Join(t.TempDir(), "store")
	if _, err := tdd.InitStore(ctx, store, "00000000-0000-4000-8000-0000000000ea"); err != nil {
		return result, fmt.Errorf("capture store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{
		Inputs: inputs, Manifest: manifest, Name: "ex-conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		return result, fmt.Errorf("capture: %w", err)
	}
	if bundle.Materialized() == "" {
		return result, errors.New("capture produced no verified materialization")
	}
	adapter := Elixir()
	scratch := filepath.Join(t.TempDir(), "run")
	if limits.WallMS == 0 {
		limits = tdd.Limits{WallMS: 240000, CleanupMS: 10000}
	}
	prepared, err := adapter.Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle, Scratch: scratch,
		Runtime: handle, Scope: scope, Limits: limits,
	})
	if err != nil {
		return result, fmt.Errorf("prepare: %w", err)
	}
	if mutatePrepared != nil {
		mutatePrepared(scratch)
	}
	execution, err := adapter.Run(ctx, prepared, func(e tdd.Event) error {
		result.events = append(result.events, e)
		return nil
	})
	closeErr := handle.Close()
	result.execution = execution
	return result, errors.Join(err, closeErr)
}

// exConformanceSuite declares the frozen fixture's closed inventory: three
// executed native identities (one through a describe parent) and the four
// registered machinery-check/v1 call sites.
func exConformanceSuite(t *testing.T, id string) tdd.Suite {
	t.Helper()
	return exConformanceSuiteOf(t, id, string(exAssetBytes(t, "conformance/conformance_test.exs")))
}

func exConformanceSuiteOf(t *testing.T, id, fixture string) tdd.Suite {
	t.Helper()
	lineOf := func(anchor string) int64 {
		for i, line := range strings.Split(fixture, "\n") {
			if strings.Contains(line, anchor) {
				return int64(i + 1)
			}
		}
		t.Fatalf("frozen fixture lacks the %q site", anchor)
		return 0
	}
	const file = "test/conformance_test.exs"
	suite := tdd.Suite{ID: id, Adapter: AdapterElixirExunit, Root: ".", Files: []string{file}}
	leaf := func(testID, name, declAnchor string, calls ...string) tdd.Test {
		test := tdd.Test{ID: testID, Source: file, Native: tdd.NativeID{Module: "Machinery.ConformanceTest", Name: name, File: file, Line: lineOf(declAnchor)}}
		for _, call := range calls {
			match := exCheckAnchor.FindStringSubmatch(call)
			if match == nil {
				t.Fatalf("anchor %q does not name a typed helper call", call)
			}
			test.Assertions = append(test.Assertions, tdd.Assertion{
				ID: match[1], Source: file, Line: lineOf(call), Helper: tdd.AssertionHelperV1,
			})
		}
		return test
	}
	suite.Tests = []tdd.Test{
		leaf("witness", "conformance witness executes native assertion",
			`test "conformance witness executes native assertion"`,
			`Machinery.Check.check(ctx, "conformance/witness"`),
		leaf("nested", "conformance parent identity conformance nested identity",
			`test "conformance nested identity"`,
			`Machinery.Check.check(ctx, "conformance/nested"`),
		leaf("multi", "conformance multiple assertions",
			`test "conformance multiple assertions"`,
			`Machinery.Check.check(ctx, "conformance/multi-a"`,
			`Machinery.Check.check(ctx, "conformance/multi-b"`),
	}
	return suite
}

// exCheckAnchor matches the frozen typed helper call form of the fixture.
var exCheckAnchor = regexp.MustCompile(`Machinery\.Check\.check\(ctx, "([^"]+)"`)

// exFrozenFiles is the captured source root of the frozen fixture suite.
func exFrozenFiles(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{"test/conformance_test.exs": exAssetBytes(t, "conformance/conformance_test.exs")}
}

// TestElixirAdapterExecutesRealMixSuiteNatively is the positive native
// proof: the frozen fixture compiles through actual Mix under
// warnings-as-errors and executes through native ExUnit with the embedded
// reporter under custody; the normalized event stream accounts every
// identity (including the describe'd nested identity) and every registered
// assertion, and the effective ExUnit options are the frozen ones.
func TestElixirAdapterExecutesRealMixSuiteNatively(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	result, err := exRunCaptured(t, ctx, exFrozenFiles(t), exConformanceSuite(t, "native-positive"), tdd.Limits{}, nil)
	if err != nil {
		t.Fatalf("real native suite execution failed: %v", err)
	}
	if result.execution.Outcome != "pass" || result.execution.ExitCode == nil || *result.execution.ExitCode != 0 {
		t.Fatalf("execution outcome %q exit %+v is not a native pass", result.execution.Outcome, result.execution.ExitCode)
	}
	if result.execution.Custody.Status != "cleaned" {
		t.Fatalf("custody did not verify: %+v", result.execution.Custody)
	}
	if len(result.execution.StdoutDigest) != 64 || len(result.execution.StderrDigest) != 64 {
		t.Fatalf("raw stream digests missing: %+v", result.execution)
	}
	assertions := map[string]string{}
	for _, e := range result.events {
		if e.Kind == "assertion" {
			assertions[e.Assertion] = e.Outcome
		}
	}
	for _, id := range []string{"conformance/witness", "conformance/nested", "conformance/multi-a", "conformance/multi-b"} {
		if assertions[id] != "pass" {
			t.Fatalf("registered assertion %s not witnessed as pass: %+v", id, assertions)
		}
	}
	kinds := map[string]int{}
	nested := false
	for _, e := range result.events {
		if e.Schema != ElixirEventSchema || e.Suite != "native-positive" {
			t.Fatalf("normalized event identity wrong: %+v", e)
		}
		kinds[e.Kind]++
		if e.Native != nil && e.Native.Name == "conformance parent identity conformance nested identity" {
			nested = true
		}
	}
	if !nested {
		t.Fatal("describe'd nested identity never normalized")
	}
	for _, want := range []string{"suite-start", "discovered", "test-start", "assertion", "test-end", "suite-end"} {
		if kinds[want] == 0 {
			t.Fatalf("normalized stream lacks %s events: %v", want, kinds)
		}
	}
}

// TestElixirAdapterProvesNativeAssertionFailureRED freezes the same-test
// safe/unsafe calibration: the identical suite passes on the safe control
// and fails AT THE EXACT REGISTERED ASSERTION on the unsafe challenge, with
// the helper witness and the native ExUnit.AssertionError cause.
func TestElixirAdapterProvesNativeAssertionFailureRED(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	frozen := exAssetBytes(t, "conformance/conformance_test.exs")
	files := map[string][]byte{"test/conformance_test.exs": frozen}
	safeResult, err := exRunCaptured(t, ctx, files, exConformanceSuite(t, "calibration-safe"), tdd.Limits{}, nil)
	if err != nil || safeResult.execution.Outcome != "pass" {
		t.Fatalf("safe control must pass natively: err=%v outcome=%q", err, safeResult.execution.Outcome)
	}
	unsafe := strings.Replace(string(frozen), "6 * 7 == 42", "6 * 7 == 43", 1)
	files = map[string][]byte{"test/conformance_test.exs": []byte(unsafe)}
	unsafeResult, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "calibration-unsafe", unsafe), tdd.Limits{}, nil)
	if err != nil {
		t.Fatalf("unsafe challenge must be reconciled, not errored: %v", err)
	}
	if unsafeResult.execution.Outcome != "assertion-fail" {
		t.Fatalf("unsafe challenge outcome %q is not the accounted assertion-fail", unsafeResult.execution.Outcome)
	}
	if unsafeResult.execution.ExitCode == nil || *unsafeResult.execution.ExitCode != 2 {
		t.Fatalf("unsafe challenge exit %+v is not the native failure status", unsafeResult.execution.ExitCode)
	}
	challengeFailed := false
	for _, e := range unsafeResult.events {
		if e.Kind == "assertion" && e.Assertion == "conformance/witness" && e.Outcome == "assertion-fail" {
			challengeFailed = true
		}
	}
	if !challengeFailed {
		t.Fatal("challenge did not fail at the registered assertion")
	}
	failEvent := false
	for _, e := range unsafeResult.events {
		if e.Kind == "test-end" && e.Native != nil && e.Native.Name == "conformance witness executes native assertion" && e.Outcome == "assertion-fail" {
			failEvent = true
		}
	}
	if !failEvent {
		t.Fatal("assertion-fail was not normalized onto the failing native identity")
	}
}

// TestElixirAdapterRejectsCompileFailureAsBuildError freezes the
// precondition boundary: a Mix compile failure is BUILD_ERROR, never RED
// evidence.
func TestElixirAdapterRejectsCompileFailureAsBuildError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	broken := string(exAssetBytes(t, "conformance/conformance_test.exs")) + "\ndefmodule Broken do\n  this is not elixir(\nend\n"
	files := map[string][]byte{"test/conformance_test.exs": []byte(broken)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuite(t, "compile-fail"), tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "BUILD_ERROR") {
		t.Fatalf("compile failure must be BUILD_ERROR: %v", err)
	}
}

// TestElixirAdapterRejectsSkippedCase freezes the skip boundary on the real
// native stream: a required case skipped by tag can never certify.
func TestElixirAdapterRejectsSkippedCase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	skipped := strings.Replace(string(exAssetBytes(t, "conformance/conformance_test.exs")),
		`  test "conformance multiple assertions", ctx do`,
		`  @tag :skip
  test "conformance multiple assertions", ctx do`, 1)
	files := map[string][]byte{"test/conformance_test.exs": []byte(skipped)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "skip-case", skipped), tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_FEATURE") {
		t.Fatalf("native skip must be rejected as UNSUPPORTED_FEATURE: %v", err)
	}
}

// TestElixirAdapterRejectsEarlyVMHalt freezes the truncated-stream boundary:
// :erlang.halt(0) before a complete inventory cannot become success even
// though the VM exits zero.
func TestElixirAdapterRejectsEarlyVMHalt(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	exiting := strings.Replace(string(exAssetBytes(t, "conformance/conformance_test.exs")),
		`    Machinery.Check.check(ctx, "conformance/witness", 6 * 7 == 42)`,
		`    Machinery.Check.check(ctx, "conformance/witness", 6 * 7 == 42)
    :erlang.halt(0)`, 1)
	files := map[string][]byte{"test/conformance_test.exs": []byte(exiting)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "early-halt", exiting), tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "INCOMPLETE_EVENTS") {
		t.Fatalf("early VM halt must surface the truncated stream: %v", err)
	}
}

// TestElixirAdapterRejectsUnregisteredAssertionFailure freezes the
// causality boundary: a plain native assert outside the byte-pinned helper
// is UNEXPECTED_FAILURE, never assertion RED.
func TestElixirAdapterRejectsUnregisteredAssertionFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	direct := strings.Replace(string(exAssetBytes(t, "conformance/conformance_test.exs")),
		`    Machinery.Check.check(ctx, "conformance/multi-b", 4 * 5 == 20)`,
		`    Machinery.Check.check(ctx, "conformance/multi-b", 4 * 5 == 20)
    assert 1 == 2`, 1)
	files := map[string][]byte{"test/conformance_test.exs": []byte(direct)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "direct-failure", direct), tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("direct native assert must be UNEXPECTED_FAILURE: %v", err)
	}
}

// TestElixirAdapterRejectsOrdinaryRaiseAndSetupAssertion freezes the error
// classes: an ordinary raise and an assertion inside setup are
// UNEXPECTED_FAILURE, never assertion RED.
func TestElixirAdapterRejectsOrdinaryRaiseAndSetupAssertion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	raising := strings.Replace(string(exAssetBytes(t, "conformance/conformance_test.exs")),
		`    Machinery.Check.check(ctx, "conformance/multi-b", 4 * 5 == 20)`,
		`    Machinery.Check.check(ctx, "conformance/multi-b", 4 * 5 == 20)
    raise "ordinary failure outside the transport"`, 1)
	files := map[string][]byte{"test/conformance_test.exs": []byte(raising)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "ordinary-raise", raising), tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("ordinary raise must be UNEXPECTED_FAILURE: %v", err)
	}
	setupSource := string(exAssetBytes(t, "conformance/conformance_test.exs")) + `
defmodule SetupAssertSuite do
  use ExUnit.Case, async: false
  require Machinery.Check

  setup do
    assert 1 == 2
  end

  test "setup assertion case", ctx do
    Machinery.Check.check(ctx, "setup/case", true)
  end
end
`
	files = map[string][]byte{"test/conformance_test.exs": []byte(setupSource)}
	suite := exConformanceSuiteOf(t, "setup-assert", setupSource)
	suiteDecl := int64(0)
	suiteCall := int64(0)
	for i, line := range strings.Split(setupSource, "\n") {
		if strings.Contains(line, `test "setup assertion case"`) && suiteDecl == 0 {
			suiteDecl = int64(i + 1)
		}
		if strings.Contains(line, `Machinery.Check.check(ctx, "setup/case"`) {
			suiteCall = int64(i + 1)
		}
	}
	suite.Tests = append(suite.Tests, tdd.Test{
		ID: "setup", Source: "test/conformance_test.exs",
		Native:     tdd.NativeID{Module: "SetupAssertSuite", Name: "setup assertion case", File: "test/conformance_test.exs", Line: suiteDecl},
		Assertions: []tdd.Assertion{{ID: "setup/case", Source: "test/conformance_test.exs", Line: suiteCall, Helper: tdd.AssertionHelperV1}},
	})
	_, err = exRunCaptured(t, ctx, files, suite, tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("setup assertion must be UNEXPECTED_FAILURE: %v", err)
	}
}

// TestElixirAdapterRejectsStalePreparedHarness freezes the late-mutation
// boundary: tampering the prepared harness bytes between Prepare and Run
// fails closed instead of executing stale transport code.
func TestElixirAdapterRejectsStalePreparedHarness(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	_, err := exRunCaptured(t, ctx, exFrozenFiles(t), exConformanceSuite(t, "stale"), tdd.Limits{}, func(scratch string) {
		victim := filepath.Join(scratch, "harness", "machinery", "check.exs")
		if body, err := os.ReadFile(victim); err == nil {
			_ = os.WriteFile(victim, append(body, []byte("\n# stale swap\n")...), 0o644)
		}
	})
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("stale prepared harness must be rejected: %v", err)
	}
}

// TestElixirAdapterRejectsAssertionSiteMismatch freezes the static source
// binding: a registered assertion whose frozen line is not the exact typed
// helper call site is rejected at Prepare time.
func TestElixirAdapterRejectsAssertionSiteMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	suite := exConformanceSuite(t, "site-mismatch")
	suite.Tests[0].Assertions[0].Line = 3
	_, err := exRunCaptured(t, ctx, exFrozenFiles(t), suite, tdd.Limits{}, nil)
	if err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("registered call-site mismatch must be rejected: %v", err)
	}
}

// TestElixirAdapterRejectsRuntimeIdentityMismatch freezes the runtime
// binding: a suite declaring a different runtime closure is rejected before
// any work, and a missing handle is rejected the same way.
func TestElixirAdapterRejectsRuntimeIdentityMismatch(t *testing.T) {
	scope := exOpenAdapterScope(t)
	defer exCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	suite := exConformanceSuite(t, "runtime-mismatch")
	handle, err := runtimeclosure.OpenElixir(ctx, runtimeclosure.ElixirRequest{})
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	identity.Version = "1.19.0/29.0.6/17.0.6"
	suite.Runtime = identity
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "test"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "test", "conformance_test.exs"), exAssetBytes(t, "conformance/conformance_test.exs"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputs := tdd.InputView{
		SourceRoot: src, DesignPath: ".", ControlRoot: t.TempDir(),
		Revalidate: func() error { return nil }, Release: func() error { return nil },
	}
	_, err = Elixir().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: tdd.BundleRef{}, Scratch: filepath.Join(t.TempDir(), "run"),
		Runtime: handle, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	})
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("runtime identity mismatch must be rejected before work: %v", err)
	}
	if _, err := Elixir().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: tdd.BundleRef{}, Scratch: filepath.Join(t.TempDir(), "run2"),
		Runtime: nil, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("missing runtime handle must be rejected: %v", err)
	}
}

// TestElixirAdapterRejectsForeignPreparedState freezes the opaque prepared
// boundary: a foreign or zero prepared carrier carries no authority.
func TestElixirAdapterRejectsForeignPreparedState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := Elixir().Run(ctx, tdd.PreparedSuite{}, func(tdd.Event) error { return nil }); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Fatalf("foreign prepared state must be rejected: %v", err)
	}
}

// TestElixirAdapterRejectsNeverSettlingTestAsTimeout freezes the wall
// boundary on a real hung BEAM: the scoped deadline kills the job and the
// run fails TIMEOUT, never success.
func TestElixirAdapterRejectsNeverSettlingTestAsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	hanging := strings.Replace(string(exAssetBytes(t, "conformance/conformance_test.exs")),
		`    Machinery.Check.check(ctx, "conformance/witness", 6 * 7 == 42)`,
		`    Machinery.Check.check(ctx, "conformance/witness", 6 * 7 == 42)
    Process.sleep(:infinity)`, 1)
	files := map[string][]byte{"test/conformance_test.exs": []byte(hanging)}
	_, err := exRunCaptured(t, ctx, files, exConformanceSuiteOf(t, "hang", hanging), tdd.Limits{WallMS: 20000, CleanupMS: 10000}, nil)
	if err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Fatalf("never-settling native test must die on the wall deadline as TIMEOUT: %v", err)
	}
}

//
// Required contributor lane fragment (MAC-bz1y closed catalog + this
// story's assurance-elixir.json).
//

var (
	exLaneBinaryOnce sync.Once
	exLaneBinaryPath string
	exLaneBinaryErr  error
)

func exLaneBinary(t *testing.T) string {
	t.Helper()
	exLaneBinaryOnce.Do(func() {
		buildDir, err := os.MkdirTemp("", "machinery-lane-binary-")
		if err != nil {
			exLaneBinaryErr = err
			return
		}
		bin := filepath.Join(buildDir, "integration-lane")
		scope := exOpenAdapterScope(t)
		defer exCloseAdapterScope(t, scope)
		goExe, err := exec.LookPath("go")
		if err != nil {
			exLaneBinaryErr = err
			return
		}
		body, err := os.ReadFile(goExe)
		if err != nil {
			exLaneBinaryErr = err
			return
		}
		sum := sha256.Sum256(body)
		attached, err := scope.Attach(processscope.Command{
			Executable:    goExe,
			Args:          []string{"build", "-o", bin, "./scripts/integration-lane"},
			Dir:           exRepoRoot(t),
			Env:           os.Environ(),
			RuntimeDigest: "sha256:" + fmt.Sprintf("%x", sum),
		})
		if err != nil {
			exLaneBinaryErr = err
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := scope.Run(ctx, attached, processscope.Streams{}); err != nil {
			exLaneBinaryErr = fmt.Errorf("build lane: %w", err)
			return
		}
		if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
			exLaneBinaryErr = fmt.Errorf("lane binary missing after build")
			return
		}
		exLaneBinaryPath = bin
	})
	if exLaneBinaryErr != nil {
		t.Fatalf("lane binary: %v", exLaneBinaryErr)
	}
	return exLaneBinaryPath
}

// exLaneSeedRoot builds a foreign-module fixture lane root carrying the
// frozen v1 pilot lane, the complete assurance catalog and this story's
// fragment. WithSeedSiblings controls whether the frozen go/node
// conformance fragments are seeded alongside; the omission subject seeds
// none so the closed union fails exactly on the missing conformance lane
// (the frozen foreign-root contract of the catalog).
func exLaneSeedRoot(t *testing.T, withSiblings, includeFragment bool, tamperConformance func(source string) string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel string, body []byte) {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", []byte("module lane.example/fixture\n\ngo 1.27.0\n"))
	write("sample/native.go", []byte("package sample\n"))
	write("sample/pilot_integration_test.go", []byte("//go:build machinery_integration\n\npackage sample\n\nimport \"testing\"\n\nfunc TestPilot(t *testing.T) {}\n"))
	repo := exRepoRoot(t)
	laneDir := filepath.Join(repo, "testdata", "integration-lanes")
	for _, name := range []string{"schema.json", "runtime-pins.json", "assurance.schema.json", "assurance-runtime-pins.json", "assurance-probes.json"} {
		body, err := os.ReadFile(filepath.Join(laneDir, name))
		if err != nil {
			t.Fatal(err)
		}
		write("testdata/integration-lanes/"+name, body)
	}
	pilot := map[string]any{
		"version": 1,
		"suites": []any{map[string]any{
			"id": "fixture", "lane": "required", "adapter": "go-json", "package": "./sample",
			"source_files": []string{"sample/pilot_integration_test.go"},
			"tests":        []string{"TestPilot"},
			"runtimes":     []string{"go"},
			"timeout":      "30s",
			"stdout_limit": 1 << 20,
			"stderr_limit": 1 << 20,
		}},
	}
	pilotBody, err := json.MarshalIndent(pilot, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	write("testdata/integration-lanes/pilot.json", append(pilotBody, '\n'))
	probeRoot := filepath.Join(laneDir, "assurance-probes")
	err = filepath.WalkDir(probeRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(laneDir, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		write("testdata/integration-lanes/"+filepath.ToSlash(rel), body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Seed the frozen sibling fragments and their conformance sources from
	// the repository so the closed catalog validates identically in
	// fixture roots; the omission subject seeds none of them so the union
	// fails exactly on the missing native-conformance lane.
	var fragments []string
	if withSiblings {
		fragments = []string{"assurance-go.json", "assurance-typescript.json"}
	}
	for _, name := range fragments {
		body, err := os.ReadFile(filepath.Join(laneDir, name))
		if err != nil {
			t.Fatal(err)
		}
		write("testdata/integration-lanes/"+name, body)
		var fragment struct {
			Suites []struct {
				Sources []string `json:"source_files"`
			} `json:"suites"`
		}
		if json.Unmarshal(body, &fragment) != nil {
			t.Fatal("malformed frozen fragment")
		}
		for _, suite := range fragment.Suites {
			for _, rel := range suite.Sources {
				body, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(rel)))
				if err != nil {
					t.Fatal(err)
				}
				write(rel, body)
			}
		}
	}
	conformance := exAssetBytes(t, "conformance/conformance_test.exs")
	if tamperConformance != nil {
		conformance = []byte(tamperConformance(string(conformance)))
	}
	write("internal/tdd/adapters/assets/elixir/conformance/conformance_test.exs", conformance)
	if includeFragment {
		write("testdata/integration-lanes/assurance-elixir.json", []byte(laneFragmentAssuranceElixir))
	}
	return root
}

type exLaneReport struct {
	Version   int    `json:"version"`
	Status    string `json:"status"`
	Assurance *struct {
		Status string `json:"status"`
		Suites []struct {
			ID        string `json:"id"`
			Selected  int    `json:"selected"`
			Started   int    `json:"started"`
			Passed    int    `json:"passed"`
			Failed    int    `json:"failed"`
			Skipped   int    `json:"skipped"`
			Events    string `json:"events_file"`
			EventsSHA string `json:"events_sha256"`
		} `json:"suites"`
	} `json:"assurance"`
}

func exLaneRun(t *testing.T, root string, envOverrides ...string) (bool, string, exLaneReport) {
	t.Helper()
	bin := exLaneBinary(t)
	work := t.TempDir()
	reportPath := filepath.Join(work, "report.json")
	scope := exOpenAdapterScope(t)
	defer exCloseAdapterScope(t, scope)
	body, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	overrides := map[string]string{}
	for _, item := range envOverrides {
		key, value, _ := strings.Cut(item, "=")
		overrides[key] = value
	}
	filtered := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, hit := overrides[key]; !hit {
			filtered = append(filtered, item)
		}
	}
	for key, value := range overrides {
		filtered = append(filtered, key+"="+value)
	}
	attached, err := scope.Attach(processscope.Command{
		Executable: bin,
		Args: []string{
			"--root", root, "--lane", "required", "--report", reportPath,
			"--work-dir", filepath.Join(work, "owned"), "--cache-dir", filepath.Join(work, "cache"),
		},
		Dir:           root,
		Env:           filtered,
		RuntimeDigest: "sha256:" + fmt.Sprintf("%x", sum),
	})
	if err != nil {
		t.Fatalf("attach lane: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var stdout, stderr bytes.Buffer
	result, err := scope.Run(ctx, attached, processscope.Streams{
		Stdout: &stdout, Stderr: &stderr, StdoutLimit: 1 << 20, StderrLimit: 1 << 20,
	})
	if err != nil {
		t.Fatalf("run lane: %v", err)
	}
	rc := 1
	if result.Completed && result.Signal == "" {
		rc = result.ExitCode
	}
	var report exLaneReport
	if raw, readErr := os.ReadFile(reportPath); readErr == nil {
		if json.Unmarshal(raw, &report) != nil {
			t.Fatalf("malformed lane report")
		}
	}
	return rc == 0 && report.Status == "passed", stdout.String() + "\n" + stderr.String(), report
}

// TestContributorLaneExecutesElixirConformanceFragment proves the required
// lane executes this story's fragment as a real native adapter conformance
// suite alongside the frozen probes, the frozen go/node conformance
// fragments and the v1 pilot lane.
func TestContributorLaneExecutesElixirConformanceFragment(t *testing.T) {
	root := exLaneSeedRoot(t, true, true, nil)
	ok, out, report := exLaneRun(t, root)
	if !ok {
		t.Fatalf("required lane with the elixir conformance fragment must pass: %+v output=%s", report, out)
	}
	if report.Assurance == nil || report.Assurance.Status != "passed" {
		t.Fatalf("assurance section missing or failed: %+v", report.Assurance)
	}
	var receipt *struct {
		ID        string `json:"id"`
		Selected  int    `json:"selected"`
		Started   int    `json:"started"`
		Passed    int    `json:"passed"`
		Failed    int    `json:"failed"`
		Skipped   int    `json:"skipped"`
		Events    string `json:"events_file"`
		EventsSHA string `json:"events_sha256"`
	}
	for i := range report.Assurance.Suites {
		if report.Assurance.Suites[i].ID == "assurance-elixir-conformance" {
			receipt = &report.Assurance.Suites[i]
		}
	}
	if receipt == nil {
		t.Fatalf("elixir conformance suite receipt missing: %+v", report.Assurance.Suites)
	}
	if receipt.Selected != 3 || receipt.Started != 3 || receipt.Passed != 3 || receipt.Failed != 0 || receipt.Skipped != 0 {
		t.Fatalf("conformance receipt not accounted exactly: %+v", receipt)
	}
	if receipt.Events == "" || len(receipt.EventsSHA) != 64 {
		t.Fatalf("conformance events not retained: %+v", receipt)
	}
	events, err := os.ReadFile(receipt.Events)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(events)) != receipt.EventsSHA {
		t.Fatalf("retained conformance events do not match the report hash: %v", err)
	}
	if !strings.Contains(string(events), "conformance witness executes native assertion") {
		t.Fatalf("retained conformance events are not the native normalized stream: %s", events)
	}
}

// TestElixirContributorLaneRejectsConformanceOmission proves omitting this
// story's required fragment fails the closed union instead of silently
// shrinking it.
func TestElixirContributorLaneRejectsConformanceOmission(t *testing.T) {
	// Seeding no conformance fragments at all: omitting this story's
	// fragment leaves the closed catalog without its required
	// native-conformance lane (the frozen foreign-root union contract).
	root := exLaneSeedRoot(t, false, false, nil)
	ok, out, _ := exLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "conformance") {
		t.Fatalf("omitting the elixir conformance fragment must fail the union, ok=%v out=%q", ok, out)
	}
}

// TestElixirContributorLaneRejectsNativeSkipInConformanceFixture proves a
// native skip inside the conformance fixture fails the lane on the real
// stream.
func TestElixirContributorLaneRejectsNativeSkipInConformanceFixture(t *testing.T) {
	root := exLaneSeedRoot(t, true, true, func(source string) string {
		return strings.Replace(source, `  test "conformance multiple assertions", ctx do`,
			`  @tag :skip
  test "conformance multiple assertions", ctx do`, 1)
	})
	ok, out, _ := exLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "skip") {
		t.Fatalf("native skip in the conformance fixture must fail the lane, ok=%v out=%q", ok, out)
	}
}

// TestElixirContributorLaneRejectsRuntimeAbsence proves a stale elixir
// runtime fails the lane before any suite runs.
func TestElixirContributorLaneRejectsRuntimeAbsence(t *testing.T) {
	root := exLaneSeedRoot(t, true, true, nil)
	shim := t.TempDir()
	if err := os.WriteFile(filepath.Join(shim, "elixir"), []byte("#!/bin/sh\necho \"Elixir 1.19.0\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, out, _ := exLaneRun(t, root, "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if ok || !strings.Contains(strings.ToLower(out), "unsupported elixir runtime") {
		t.Fatalf("stale elixir runtime must fail provisioning, ok=%v out=%q", ok, out)
	}
}
