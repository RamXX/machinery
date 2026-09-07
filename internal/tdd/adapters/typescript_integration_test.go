package adapters

// RED contract for MAC-avfp (native subjects): the closed
// node-test-typescript/v1 adapter executes REAL TypeScript suites — pinned
// native compiler build step, Node's actual node:test runner, embedded
// Machinery reporter, processscope custody, normalized machinery.tdd.event/v1
// accounting — through the REAL production chain (InitStore -> Capture ->
// OpenTypeScript+Validate -> Prepare -> Run -> process-free Close). Compile
// failures, skips, early exits, unwitnessed failures, stale outputs, site
// mismatches, runtime mismatch and never-settling tests are rejected. The
// required contributor lane fragment executes alongside the frozen catalog.

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

// laneFragmentAssuranceTypeScript is the exact frozen fragment GREEN ships at
// testdata/integration-lanes/assurance-typescript.json.
const laneFragmentAssuranceTypeScript = `{
  "schema": "machinery.assurance.lane/v1",
  "fragment": "assurance-typescript",
  "owner": "MAC-avfp",
  "suites": [
    {
      "id": "assurance-node-conformance",
      "lane": "assurance",
      "adapter": "node-test-typescript/v1",
      "kind": "native-conformance",
      "package": ".",
      "source_files": [
        "internal/tdd/adapters/assets/typescript/package.json",
        "internal/tdd/adapters/assets/typescript/node-ambient.d.ts",
        "internal/tdd/adapters/assets/typescript/machinery-check.ts",
        "internal/tdd/adapters/assets/typescript/conformance/conformance.ts"
      ],
      "tests": [
        "conformance witness executes native assertion",
        "conformance parent identity",
        "conformance parent identity > conformance nested identity",
        "conformance multiple assertions"
      ],
      "runtimes": ["node", "tsc"],
      "timeout": "5m",
      "stdout_limit": 1048576,
      "stderr_limit": 1048576
    }
  ]
}
`

// The package TestMain (processscope internal-IO dispatcher) is defined once
// in go_integration_test.go and serves both adapter suites.

func tsRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func tsOpenAdapterScope(t *testing.T) processscope.Scope {
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

func tsCloseAdapterScope(t *testing.T, scope processscope.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	report, err := scope.Close(ctx)
	if err != nil || report.Status != processscope.StatusCleaned {
		t.Fatalf("custody close did not verify: %v %+v", err, report)
	}
}

func tsAssetBytes(t *testing.T, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(tsRepoRoot(t), "internal", "tdd", "adapters", "assets", "typescript", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// tsChainResult is one completed production-chain suite execution.
type tsChainResult struct {
	execution tdd.Execution
	events    []tdd.Event
}

// tsRunCaptured captures frozen fixture files into a real content-addressed
// store bundle and executes the declared suite through the closed production
// chain on a fresh custody root (the lane's own per-suite pattern): pinned
// runtime closure validated under the live scope, adapter Prepare over the
// captured bundle, adapter Run with the recording sink.
func tsRunCaptured(t *testing.T, ctx context.Context, files map[string][]byte, suite tdd.Suite, mutatePrepared func(scratch string)) (tsChainResult, error) {
	t.Helper()
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	var result tsChainResult
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
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o644); err != nil {
		return result, err
	}
	released := 0
	inputs := tdd.InputView{
		SourceRoot: src, DesignPath: ".", ControlRoot: control,
		Revalidate: func() error { return nil },
		Release:    func() error { released++; return nil },
	}
	handle, err := runtimeclosure.OpenTypeScript(ctx, runtimeclosure.TypeScriptRequest{})
	if err != nil {
		return result, fmt.Errorf("pinned TypeScript closure: %w", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Validate(ctx, scope); err != nil {
		return result, fmt.Errorf("pinned TypeScript closure validation: %w", err)
	}
	suite.Runtime = handle.Identity()
	store := filepath.Join(t.TempDir(), "store")
	if _, err := tdd.InitStore(ctx, store, "00000000-0000-4000-8000-0000000000af"); err != nil {
		return result, fmt.Errorf("capture store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{
		Inputs: inputs, Manifest: manifest, Name: "ts-conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		return result, fmt.Errorf("capture: %w", err)
	}
	if bundle.Materialized() == "" {
		return result, errors.New("capture produced no verified materialization")
	}
	adapter := TypeScript()
	scratch := filepath.Join(t.TempDir(), "run")
	prepared, err := adapter.Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle, Scratch: scratch,
		Runtime: handle, Scope: scope,
		Limits: tdd.Limits{WallMS: 240000, CleanupMS: 10000},
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

// tsConformanceSuite declares the frozen fixture's closed inventory: three
// top-level cases, the nested child identity and the four registered
// machinery-check/v1 call sites.
func tsConformanceSuite(t *testing.T, id string) tdd.Suite {
	fixture := string(tsAssetBytes(t, "conformance/conformance.ts"))
	callLine := func(anchor string) int64 {
		for i, line := range strings.Split(fixture, "\n") {
			if strings.Contains(line, anchor) {
				return int64(i + 1)
			}
		}
		t.Fatalf("frozen fixture lacks the %q call site", anchor)
		return 0
	}
	suite := tdd.Suite{ID: id, Adapter: AdapterNodeTestTS, Root: ".", Files: []string{"conformance.ts"}}
	leaf := func(testID, name string, path []string, line int64, anchors ...string) tdd.Test {
		test := tdd.Test{ID: testID, Source: "conformance.ts", Native: tdd.NativeID{Source: "conformance.ts", Path: path, Line: line, Column: 1}}
		for _, anchor := range anchors {
			match := regexp.MustCompile(`check\([a-zA-Z]+,\s*"([^"]+)"`).FindStringSubmatch(anchor)
			if match == nil {
				t.Fatalf("anchor %q does not name a typed helper call", anchor)
			}
			test.Assertions = append(test.Assertions, tdd.Assertion{
				ID: match[1], Source: "conformance.ts",
				Line: callLine(anchor), Helper: tdd.AssertionHelperV1,
			})
		}
		return test
	}
	suite.Tests = []tdd.Test{
		leaf("witness", "conformance witness executes native assertion", []string{"conformance witness executes native assertion"}, 4,
			`check(t, "conformance/witness"`),
		{ID: "parent", Source: "conformance.ts", Native: tdd.NativeID{Source: "conformance.ts", Path: []string{"conformance parent identity"}, Line: 8, Column: 1}},
		leaf("nested", "conformance nested identity", []string{"conformance parent identity", "conformance nested identity"}, 9,
			`check(ct, "conformance/nested"`),
		leaf("multi", "conformance multiple assertions", []string{"conformance multiple assertions"}, 14,
			`check(t, "conformance/multi-a"`, `check(t, "conformance/multi-b"`),
	}
	return suite
}

func tsFrozenFiles(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{"conformance.ts": tsAssetBytes(t, "conformance/conformance.ts")}
}

// TestTypeScriptAdapterExecutesRealCompiledSuiteNatively is the positive
// native proof: the frozen fixture compiles with the pinned native compiler
// and executes through Node's actual node:test runner under custody with the
// embedded reporter; the normalized event stream accounts every identity and
// every registered assertion.
func TestTypeScriptAdapterExecutesRealCompiledSuiteNatively(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	result, err := tsRunCaptured(t, ctx, tsFrozenFiles(t), tsConformanceSuite(t, "native-positive"), nil)
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
		if e.Schema != TypeScriptEventSchema || e.Suite != "native-positive" {
			t.Fatalf("normalized event identity wrong: %+v", e)
		}
		kinds[e.Kind]++
		if e.Native != nil && len(e.Native.Path) == 2 && e.Native.Path[0] == "conformance parent identity" {
			nested = true
		}
	}
	if !nested {
		t.Fatal("nested child identity never normalized")
	}
	for _, want := range []string{"suite-start", "discovered", "test-start", "assertion", "test-end", "suite-end"} {
		if kinds[want] == 0 {
			t.Fatalf("normalized stream lacks %s events: %v", want, kinds)
		}
	}
}

// TestTypeScriptAdapterProvesNativeAssertionFailureRED freezes the
// same-test safe/unsafe calibration: the identical suite passes on the safe
// control and fails AT THE EXACT REGISTERED ASSERTION on the unsafe
// challenge, with the helper witness and the native AssertionError cause.
func TestTypeScriptAdapterProvesNativeAssertionFailureRED(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	frozen := tsAssetBytes(t, "conformance/conformance.ts")
	files := map[string][]byte{"conformance.ts": frozen}
	safeResult, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "calibration-safe"), nil)
	if err != nil || safeResult.execution.Outcome != "pass" {
		t.Fatalf("safe control must pass natively: err=%v outcome=%q", err, safeResult.execution.Outcome)
	}
	unsafe := strings.Replace(string(frozen), "6 * 7 === 42", "6 * 7 === 43", 1)
	files = map[string][]byte{"conformance.ts": []byte(unsafe)}
	unsafeResult, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "calibration-unsafe"), nil)
	if err != nil {
		t.Fatalf("unsafe challenge must be reconciled, not errored: %v", err)
	}
	if unsafeResult.execution.Outcome != "fail" {
		t.Fatalf("unsafe challenge outcome %q is not the accounted fail", unsafeResult.execution.Outcome)
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
		if e.Kind == "test-end" && e.Native != nil && e.Native.Path[0] == "conformance witness executes native assertion" && e.Outcome == "assertion-fail" {
			failEvent = true
		}
	}
	if !failEvent {
		t.Fatal("assertion-fail was not normalized onto the failing native identity")
	}
}

// TestTypeScriptAdapterRejectsCompileFailureAsBuildError freezes the
// precondition boundary: a TypeScript compile failure is BUILD_ERROR, never
// RED evidence.
func TestTypeScriptAdapterRejectsCompileFailureAsBuildError(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	broken := strings.Replace(string(tsAssetBytes(t, "conformance/conformance.ts")), "const answer", "const answer: number", 1)
	files := map[string][]byte{"conformance.ts": []byte(broken + "\nconst brokenCompileGate: number = \"not a number\";\n")}
	_, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "compile-fail"), nil)
	if err == nil || !strings.Contains(err.Error(), "BUILD_ERROR") {
		t.Fatalf("compile failure must be BUILD_ERROR: %v", err)
	}
}

// TestTypeScriptAdapterRejectsSkippedCase freezes the skip boundary on the
// real native stream.
func TestTypeScriptAdapterRejectsSkippedCase(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	skipped := strings.Replace(string(tsAssetBytes(t, "conformance/conformance.ts")),
		`check(t, "conformance/multi-b", 4 * 5 === 20);`,
		`check(t, "conformance/multi-b", 4 * 5 === 20);
  t.skip("required conformance case must not skip");`, 1)
	files := map[string][]byte{"conformance.ts": []byte(skipped)}
	_, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "skip-case"), nil)
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_FEATURE") {
		t.Fatalf("native skip must be rejected as UNSUPPORTED_FEATURE: %v", err)
	}
}

// TestTypeScriptAdapterRejectsEarlyProcessExit freezes the truncated-stream
// boundary: process.exit(0) before a complete inventory cannot become
// success even though the runner exits zero.
func TestTypeScriptAdapterRejectsEarlyProcessExit(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	exiting := strings.Replace(string(tsAssetBytes(t, "conformance/conformance.ts")),
		`check(t, "conformance/witness", 6 * 7 === 42);`,
		`check(t, "conformance/witness", 6 * 7 === 42); process.exit(0);`, 1)
	exiting = strings.Replace(exiting,
		`import { check } from "./machinery-check.js";`,
		`import { check } from "./machinery-check.js"; declare const process: { exit(code?: number): void };`, 1)
	files := map[string][]byte{"conformance.ts": []byte(exiting)}
	_, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "early-exit"), nil)
	if err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
		t.Fatalf("early process exit must surface the unexecuted declared identity: %v", err)
	}
}

// TestTypeScriptAdapterRejectsUnregisteredAssertionFailure freezes the
// causality boundary: a direct native assertion outside the byte-pinned
// helper is UNEXPECTED_FAILURE, never assertion RED.
func TestTypeScriptAdapterRejectsUnregisteredAssertionFailure(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	direct := strings.Replace(string(tsAssetBytes(t, "conformance/conformance.ts")),
		`check(t, "conformance/multi-b", 4 * 5 === 20);`,
		`check(t, "conformance/multi-b", 4 * 5 === 20);
  throw new Error("direct failure outside the transport");`, 1)
	files := map[string][]byte{"conformance.ts": []byte(direct)}
	_, err := tsRunCaptured(t, ctx, files, tsConformanceSuite(t, "direct-failure"), nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("direct failure must be UNEXPECTED_FAILURE: %v", err)
	}
}

// TestTypeScriptAdapterRejectsStalePreparedOutput freezes the late-mutation
// boundary: tampering the prepared scratch between Prepare and Run fails
// closed instead of executing stale bytes.
func TestTypeScriptAdapterRejectsStalePreparedOutput(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	_, err := tsRunCaptured(t, ctx, tsFrozenFiles(t), tsConformanceSuite(t, "stale"), func(scratch string) {
		candidates, _ := filepath.Glob(filepath.Join(scratch, "out", "conformance.js"))
		if len(candidates) == 1 {
			_ = os.WriteFile(candidates[0], []byte("// stale swap\n"), 0o644)
		}
	})
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("stale compiled output must be rejected: %v", err)
	}
}

// TestTypeScriptAdapterRejectsAssertionSiteMismatch freezes the static
// source binding: a registered assertion whose frozen line is not the exact
// typed helper call site is rejected at Prepare time.
func TestTypeScriptAdapterRejectsAssertionSiteMismatch(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	suite := tsConformanceSuite(t, "site-mismatch")
	suite.Tests[0].Assertions[0].Line = 3
	_, err := tsRunCaptured(t, ctx, tsFrozenFiles(t), suite, nil)
	if err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("registered call-site mismatch must be rejected: %v", err)
	}
}

// TestTypeScriptAdapterRejectsRuntimeIdentityMismatch freezes the runtime
// binding: a suite declaring a different runtime closure is rejected before
// any work.
func TestTypeScriptAdapterRejectsRuntimeIdentityMismatch(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	suite := tsConformanceSuite(t, "runtime-mismatch")
	handle, err := runtimeclosure.OpenTypeScript(ctx, runtimeclosure.TypeScriptRequest{})
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	identity.Version = "25.0.0/7.0.2"
	suite.Runtime = identity
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "conformance.ts"), tsAssetBytes(t, "conformance/conformance.ts"), 0o644); err != nil {
		t.Fatal(err)
	}
	inputs := tdd.InputView{
		SourceRoot: src, DesignPath: ".", ControlRoot: t.TempDir(),
		Revalidate: func() error { return nil }, Release: func() error { return nil },
	}
	_, err = TypeScript().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: tdd.BundleRef{}, Scratch: filepath.Join(t.TempDir(), "run"),
		Runtime: handle, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	})
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("runtime identity mismatch must be rejected before work: %v", err)
	}
	if _, err := TypeScript().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: tdd.BundleRef{}, Scratch: filepath.Join(t.TempDir(), "run2"),
		Runtime: nil, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("missing runtime handle must be rejected: %v", err)
	}
}

// TestTypeScriptAdapterRejectsForeignPreparedState freezes the opaque
// prepared boundary: a foreign or zero prepared carrier carries no
// authority.
func TestTypeScriptAdapterRejectsForeignPreparedState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := TypeScript().Run(ctx, tdd.PreparedSuite{}, func(tdd.Event) error { return nil }); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Fatalf("foreign prepared state must be rejected: %v", err)
	}
}

// TestTypeScriptAdapterRejectsNeverSettlingTestAsTimeout freezes the wall
// boundary on a real hung native process: the scoped deadline kills the job
// and the run fails TIMEOUT, never success.
func TestTypeScriptAdapterRejectsNeverSettlingTestAsTimeout(t *testing.T) {
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	hanging := "import { test } from \"node:test\";\n" +
		"test(\"conformance witness executes native assertion\", (): Promise<void> => new Promise(() => {}));\n"
	suite := tdd.Suite{ID: "hang", Adapter: AdapterNodeTestTS, Root: ".", Files: []string{"conformance.ts"},
		Tests: []tdd.Test{
			{ID: "a", Source: "conformance.ts", Native: tdd.NativeID{Source: "conformance.ts", Path: []string{"conformance witness executes native assertion"}, Line: 2, Column: 1}},
		}}
	src := t.TempDir()
	control := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "conformance.ts"), []byte(hanging), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	inputs := tdd.InputView{SourceRoot: src, DesignPath: ".", ControlRoot: control,
		Revalidate: func() error { return nil }, Release: func() error { return nil }}
	handle, err := runtimeclosure.OpenTypeScript(ctx, runtimeclosure.TypeScriptRequest{})
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Validate(ctx, scope); err != nil {
		t.Fatalf("closure validation: %v", err)
	}
	suite.Runtime = handle.Identity()
	store := filepath.Join(t.TempDir(), "store")
	if _, err := tdd.InitStore(ctx, store, "00000000-0000-4000-8000-0000000000af"); err != nil {
		t.Fatal(err)
	}
	manifest := tdd.Manifest{Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite}}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{Inputs: inputs, Manifest: manifest, Name: "hang", Store: store, Limits: tdd.Limits{WallMS: 120000}})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	prepared, err := TypeScript().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle, Scratch: filepath.Join(t.TempDir(), "run"),
		Runtime: handle, Scope: scope, Limits: tdd.Limits{WallMS: 20000, CleanupMS: 10000},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	_, err = TypeScript().Run(ctx, prepared, func(tdd.Event) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Fatalf("never-settling native test must die on the wall deadline as TIMEOUT: %v", err)
	}
}

//
// Required contributor lane fragment (MAC-bz1y closed catalog + this
// story's assurance-typescript.json).
//

var (
	tsLaneBinaryOnce sync.Once
	tsLaneBinaryPath string
	tsLaneBinaryErr  error
)

func tsLaneBinary(t *testing.T) string {
	t.Helper()
	tsLaneBinaryOnce.Do(func() {
		buildDir, err := os.MkdirTemp("", "machinery-lane-binary-")
		if err != nil {
			tsLaneBinaryErr = err
			return
		}
		bin := filepath.Join(buildDir, "integration-lane")
		scope := tsOpenAdapterScope(t)
		defer tsCloseAdapterScope(t, scope)
		goExe, err := exec.LookPath("go")
		if err != nil {
			tsLaneBinaryErr = err
			return
		}
		body, err := os.ReadFile(goExe)
		if err != nil {
			tsLaneBinaryErr = err
			return
		}
		sum := sha256.Sum256(body)
		attached, err := scope.Attach(processscope.Command{
			Executable:    goExe,
			Args:          []string{"build", "-o", bin, "./scripts/integration-lane"},
			Dir:           tsRepoRoot(t),
			Env:           os.Environ(),
			RuntimeDigest: "sha256:" + fmt.Sprintf("%x", sum),
		})
		if err != nil {
			tsLaneBinaryErr = err
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := scope.Run(ctx, attached, processscope.Streams{}); err != nil {
			tsLaneBinaryErr = fmt.Errorf("build lane: %w", err)
			return
		}
		if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
			tsLaneBinaryErr = fmt.Errorf("lane binary missing after build")
			return
		}
		tsLaneBinaryPath = bin
	})
	if tsLaneBinaryErr != nil {
		t.Fatalf("lane binary: %v", tsLaneBinaryErr)
	}
	return tsLaneBinaryPath
}

// tsLaneSeedRoot builds a foreign-module fixture lane root carrying the frozen
// v1 pilot lane, the complete assurance catalog and this story's fragment.
func tsLaneSeedRoot(t *testing.T, includeFragment bool, tamperConformance func(source string) string) string {
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
	repo := tsRepoRoot(t)
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
	conformance := tsAssetBytes(t, "conformance/conformance.ts")
	if tamperConformance != nil {
		conformance = []byte(tamperConformance(string(conformance)))
	}
	write("internal/tdd/adapters/assets/typescript/package.json", tsAssetBytes(t, "package.json"))
	write("internal/tdd/adapters/assets/typescript/node-ambient.d.ts", tsAssetBytes(t, "node-ambient.d.ts"))
	write("internal/tdd/adapters/assets/typescript/machinery-check.ts", tsAssetBytes(t, "machinery-check.ts"))
	write("internal/tdd/adapters/assets/typescript/conformance/conformance.ts", conformance)
	if includeFragment {
		write("testdata/integration-lanes/assurance-typescript.json", []byte(laneFragmentAssuranceTypeScript))
	}
	return root
}

type tsLaneReport struct {
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

func tsLaneRun(t *testing.T, root string, envOverrides ...string) (bool, string, tsLaneReport) {
	t.Helper()
	bin := tsLaneBinary(t)
	work := t.TempDir()
	reportPath := filepath.Join(work, "report.json")
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
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
	var report tsLaneReport
	if raw, readErr := os.ReadFile(reportPath); readErr == nil {
		if json.Unmarshal(raw, &report) != nil {
			t.Fatalf("malformed lane report")
		}
	}
	return rc == 0 && report.Status == "passed", stdout.String() + "\n" + stderr.String(), report
}

// TestContributorLaneExecutesTypeScriptConformanceFragment proves the
// required lane executes this story's fragment as a real native adapter
// conformance suite alongside the frozen probes and the v1 pilot lane.
func TestContributorLaneExecutesTypeScriptConformanceFragment(t *testing.T) {
	root := tsLaneSeedRoot(t, true, nil)
	ok, out, report := tsLaneRun(t, root)
	if !ok {
		t.Fatalf("required lane with the typescript conformance fragment must pass: %+v output=%s", report, out)
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
		if report.Assurance.Suites[i].ID == "assurance-node-conformance" {
			receipt = &report.Assurance.Suites[i]
		}
	}
	if receipt == nil {
		t.Fatalf("typescript conformance suite receipt missing: %+v", report.Assurance.Suites)
	}
	if receipt.Selected != 4 || receipt.Started != 4 || receipt.Passed != 4 || receipt.Failed != 0 || receipt.Skipped != 0 {
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

// TestTypeScriptContributorLaneRejectsConformanceOmission proves omitting this story's
// required fragment fails the closed union instead of silently shrinking it.
func TestTypeScriptContributorLaneRejectsConformanceOmission(t *testing.T) {
	root := tsLaneSeedRoot(t, false, nil)
	ok, out, _ := tsLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "conformance") {
		t.Fatalf("omitting the typescript conformance fragment must fail the union, ok=%v out=%q", ok, out)
	}
}

// TestTypeScriptContributorLaneRejectsNativeSkipInConformanceFixture proves a native
// skip inside the conformance fixture fails the lane on the real stream.
func TestTypeScriptContributorLaneRejectsNativeSkipInConformanceFixture(t *testing.T) {
	root := tsLaneSeedRoot(t, true, func(source string) string {
		return strings.Replace(source, `check(t, "conformance/multi-b", 4 * 5 === 20);`, `check(t, "conformance/multi-b", 4 * 5 === 20);
  t.skip("required conformance must not skip");`, 1)
	})
	ok, out, _ := tsLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "skip") {
		t.Fatalf("native skip in the conformance fixture must fail the lane, ok=%v out=%q", ok, out)
	}
}

// TestTypeScriptContributorLaneRejectsRuntimeAbsence proves a stale node runtime fails
// the lane before any suite runs.
func TestTypeScriptContributorLaneRejectsRuntimeAbsence(t *testing.T) {
	root := tsLaneSeedRoot(t, true, nil)
	shim := t.TempDir()
	if err := os.WriteFile(filepath.Join(shim, "node"), []byte("#!/bin/sh\necho v25.9.9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, out, _ := tsLaneRun(t, root, "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if ok || !strings.Contains(strings.ToLower(out), "unsupported node runtime") {
		t.Fatalf("stale node runtime must fail provisioning, ok=%v out=%q", ok, out)
	}
}
