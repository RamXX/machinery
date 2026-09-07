//go:build unix

package adapters_test

// Frozen RED native conformance suite for MAC-wi2u (go-testing/v1). Every
// subject drives the REAL production chain — captured source bundle, pinned
// Go 1.27.1 runtime closure validated under a live processscope scope, the
// byte-pinned helper transport materialized by the adapter, real `go test
// -json -count=1 -shuffle=off` executions under custody, normalized
// machinery.tdd.event/v1 accounting — plus the required contributor-lane
// fragment. On the RED stub every subject fails on the absent adapter; no
// mock runner, no skip-if-missing, no fabricated output.

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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/adapters"
)

const conformanceModule = "machinery.test/conformance"
const negativeModule = "neg.test"
const laneProjectUUID = "5f6d8b1e-9a2f-4c73-8a11-2f0a2c9d7e42"

// laneFragmentAssuranceGo is the EXACT frozen byte content of the owned
// required contributor fragment testdata/integration-lanes/assurance-go.json
// (GREEN ships this file byte-identically).
const laneFragmentAssuranceGo = `{
  "schema": "machinery.assurance.lane/v1",
  "fragment": "assurance-go",
  "owner": "MAC-wi2u",
  "suites": [
    {
      "id": "assurance-go-conformance",
      "lane": "assurance",
      "adapter": "go-testing/v1",
      "kind": "native-conformance",
      "package": "./conformance",
      "source_files": [
        "internal/tdd/adapters/assets/go/conformance/go.mod",
        "internal/tdd/adapters/assets/go/conformance/conformance_test.go"
      ],
      "tests": [
        "TestConformanceWitnessPass",
        "TestConformanceSubtestIdentity",
        "TestConformanceSubtestIdentity/first",
        "TestConformanceSubtestIdentity/second",
        "TestConformanceMultipleAssertions"
      ],
      "runtimes": [
        "go"
      ],
      "timeout": "5m",
      "stdout_limit": 1048576,
      "stderr_limit": 1048576
    }
  ]
}
`

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	io, ok, err := processscope.InheritedInternalIO(ctx)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "machinery: invalid internal channel claim:", err)
		os.Exit(2)
	}
	if ok {
		handled, code := processscope.ServeInternal(os.Args[1:], io)
		if !handled {
			fmt.Fprintln(os.Stderr, "machinery: internal protocol failure")
			os.Exit(2)
		}
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func openAdapterScope(t *testing.T) processscope.Scope {
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

func closeAdapterScope(t *testing.T, scope processscope.Scope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	report, err := scope.Close(ctx)
	if err != nil || report.Status != processscope.StatusCleaned {
		t.Fatalf("custody close did not verify: %v %+v", err, report)
	}
}

func assetBytes(t *testing.T, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot(t), "internal", "tdd", "adapters", "assets", "go", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// conformanceFiles returns the frozen conformance fixture bytes, optionally
// with an anchored mutation applied to the test source.
func conformanceFiles(t *testing.T, mutate func(source string) string) map[string][]byte {
	t.Helper()
	source := string(assetBytes(t, "conformance/conformance_test.go"))
	if mutate != nil {
		source = mutate(source)
	}
	return map[string][]byte{
		"go.mod":              assetBytes(t, "conformance/go.mod"),
		"conformance_test.go": []byte(source),
	}
}

func writeModule(t *testing.T, dir string, files map[string][]byte) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertionLine(t *testing.T, source, id string) int64 {
	t.Helper()
	anchor := fmt.Sprintf("%q", id)
	for i, line := range strings.Split(source, "\n") {
		if strings.Contains(line, anchor) && strings.Contains(line, "machinerycheck.Check(t,") {
			return int64(i + 1)
		}
	}
	t.Fatalf("assertion id %s has no helper call site in source", id)
	return 0
}

func leaf(t *testing.T, source, path, id string) tdd.Test {
	t.Helper()
	test := tdd.Test{ID: path, Source: "conformance_test.go", Native: tdd.NativeID{Package: conformanceModule, Test: path}}
	if id != "" {
		test.Assertions = []tdd.Assertion{{ID: id, Source: "conformance_test.go", Line: assertionLine(t, source, id), Helper: "machinery-check/v1"}}
	}
	return test
}

func conformanceSuite(t *testing.T, files map[string][]byte) tdd.Suite {
	t.Helper()
	source := string(files["conformance_test.go"])
	return tdd.Suite{
		ID:      "go-conformance",
		Adapter: "go-testing/v1",
		Runtime: tdd.RuntimeRef{}, // filled from the live handle by runGoSuite
		Root:    ".",
		Files:   []string{"go.mod", "conformance_test.go"},
		Tests: []tdd.Test{
			leaf(t, source, "TestConformanceWitnessPass", "conformance/witness-pass"),
			leaf(t, source, "TestConformanceSubtestIdentity", ""),
			leaf(t, source, "TestConformanceSubtestIdentity/first", "conformance/subtest-first"),
			leaf(t, source, "TestConformanceSubtestIdentity/second", "conformance/subtest-second"),
			multiAssertionLeaf(t, source),
		},
	}
}

// multiAssertionLeaf returns the multiple-assertions test with both sites.
func multiAssertionLeaf(t *testing.T, source string) tdd.Test {
	t.Helper()
	return tdd.Test{
		ID:     "TestConformanceMultipleAssertions",
		Source: "conformance_test.go",
		Native: tdd.NativeID{Package: conformanceModule, Test: "TestConformanceMultipleAssertions"},
		Assertions: []tdd.Assertion{
			{ID: "conformance/multi-a", Source: "conformance_test.go", Line: assertionLine(t, source, "conformance/multi-a"), Helper: "machinery-check/v1"},
			{ID: "conformance/multi-b", Source: "conformance_test.go", Line: assertionLine(t, source, "conformance/multi-b"), Helper: "machinery-check/v1"},
		},
	}
}

// runGoSuite drives the complete production chain over a fixture module and
// returns the execution, the collected normalized events and any error.
func runGoSuite(t *testing.T, files map[string][]byte, suite tdd.Suite, mutateScratch func(moduleDir string)) (tdd.Execution, []tdd.Event, error) {
	return runGoSuiteChecked(t, files, suite, mutateScratch, nil)
}

// runGoSuiteChecked additionally mutates the suite declaration after the
// live runtime identity is bound, so subjects can prove closed rejections
// of mismatched declared runtimes.
func runGoSuiteChecked(t *testing.T, files map[string][]byte, suite tdd.Suite, mutateScratch func(moduleDir string), mutateSuite func(*tdd.Suite)) (tdd.Execution, []tdd.Event, error) {
	t.Helper()
	scope := openAdapterScope(t)
	defer closeAdapterScope(t, scope)
	base := t.TempDir()
	sourceDir := filepath.Join(base, "src")
	writeModule(t, sourceDir, files)
	controlDir := filepath.Join(base, "control")
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(controlDir, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	handle, err := runtimeclosure.OpenGo(context.Background(), runtimeclosure.GoRequest{})
	if err != nil {
		return tdd.Execution{}, nil, fmt.Errorf("open pinned go closure: %w", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Validate(context.Background(), scope); err != nil {
		return tdd.Execution{}, nil, fmt.Errorf("validate pinned go closure: %w", err)
	}
	suite.Runtime = handle.Identity()
	if mutateSuite != nil {
		mutateSuite(&suite)
	}
	suiteRoots := map[string][]byte{}
	for name, body := range files {
		suiteRoots[name] = body
	}
	released := false
	inputs := tdd.InputView{
		SourceRoot:  sourceDir,
		DesignPath:  ".",
		ControlRoot: controlDir,
		Revalidate: func() error {
			for name, want := range suiteRoots {
				got, err := os.ReadFile(filepath.Join(sourceDir, filepath.FromSlash(name)))
				if err != nil || fmt.Sprintf("%x", sha256.Sum256(got)) != fmt.Sprintf("%x", sha256.Sum256(want)) {
					return fmt.Errorf("fixture input %s changed", name)
				}
			}
			return nil
		},
		Release: func() error {
			if released {
				return errors.New("double release")
			}
			released = true
			return nil
		},
	}
	ctx := context.Background()
	storeDir := filepath.Join(base, "store")
	if _, err := tdd.InitStore(ctx, storeDir, laneProjectUUID); err != nil {
		return tdd.Execution{}, nil, fmt.Errorf("init store: %w", err)
	}
	manifest := tdd.Manifest{
		Schema: tdd.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, Suites: []tdd.Suite{suite},
	}
	bundle, err := tdd.Capture(ctx, tdd.CaptureRequest{Inputs: inputs, Manifest: manifest, Name: "conformance", Store: storeDir, Limits: tdd.Limits{WallMS: 300000}})
	if err != nil {
		return tdd.Execution{}, nil, fmt.Errorf("capture bundle: %w", err)
	}
	adapter, err := adapters.Lookup("go-testing/v1")
	if err != nil {
		return tdd.Execution{}, nil, err
	}
	request := tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle,
		Scratch: filepath.Join(base, "run"), Runtime: handle, Scope: scope,
		Limits: tdd.Limits{WallMS: 300000, CleanupMS: 30000},
	}
	prepared, err := adapter.Prepare(ctx, request)
	if err != nil {
		return tdd.Execution{}, nil, fmt.Errorf("prepare: %w", err)
	}
	if mutateScratch != nil {
		runDir := request.Scratch
		mutateScratch(runDir)
	}
	var events []tdd.Event
	execution, runErr := adapter.Run(ctx, prepared, func(e tdd.Event) error {
		events = append(events, e)
		return nil
	})
	return execution, events, runErr
}

func requireEvent(t *testing.T, events []tdd.Event, kind, test, assertion, outcome string) {
	t.Helper()
	for _, e := range events {
		if e.Kind != kind || e.Assertion != assertion || e.Outcome != outcome {
			continue
		}
		if test == "" && e.Native == nil {
			return
		}
		if e.Native != nil && e.Native.Test == test {
			return
		}
	}
	t.Fatalf("normalized stream lacks %s/%s/%s/%s event: %+v", kind, test, assertion, outcome, events)
}

// TestGoAdapterExecutesFrozenConformanceSuiteNatively is the positive
// conformance proof: real capture, real pinned runtime, real native go test
// execution, complete normalized lifecycle, every registered assertion
// witnessed at its frozen call site, custody verified.
func TestGoAdapterExecutesFrozenConformanceSuiteNatively(t *testing.T) {
	files := conformanceFiles(t, nil)
	suite := conformanceSuite(t, files)
	execution, events, err := runGoSuite(t, files, suite, nil)
	if err != nil {
		t.Fatalf("native conformance execution failed: %v", err)
	}
	if execution.Outcome != "pass" {
		t.Fatalf("outcome is %q, want pass (events follow)\n%+v", execution.Outcome, events)
	}
	if execution.ExitCode == nil || *execution.ExitCode != 0 {
		t.Fatalf("native exit status is %+v", execution.ExitCode)
	}
	if execution.Custody.Status != "cleaned" {
		t.Fatalf("custody did not verify: %+v", execution.Custody)
	}
	requireEvent(t, events, "suite-start", "", "", "")
	requireEvent(t, events, "discovered", "TestConformanceWitnessPass", "", "")
	requireEvent(t, events, "test-start", "TestConformanceSubtestIdentity/first", "", "")
	requireEvent(t, events, "assertion", "TestConformanceWitnessPass", "conformance/witness-pass", "pass")
	requireEvent(t, events, "assertion", "TestConformanceSubtestIdentity/first", "conformance/subtest-first", "pass")
	requireEvent(t, events, "assertion", "TestConformanceMultipleAssertions", "conformance/multi-b", "pass")
	requireEvent(t, events, "test-end", "TestConformanceSubtestIdentity/second", "", "pass")
	requireEvent(t, events, "suite-end", "", "", "pass")
	for _, e := range events {
		if e.Schema != "machinery.tdd.event/v1" || e.Suite != "go-conformance" {
			t.Fatalf("event carries wrong schema/suite identity: %+v", e)
		}
		if e.Design != "" || e.Milestone != "" {
			t.Fatalf("native stream reassigned owner identity: %+v", e)
		}
	}
	if len(execution.Events) != len(events) {
		t.Fatalf("execution event inventory diverges from the sink stream")
	}
	if !strings.HasPrefix(execution.StdoutDigest, "sha256:") || len(execution.StdoutDigest) != 71 {
		t.Fatalf("raw stdout digest missing: %q", execution.StdoutDigest)
	}
}

// TestGoAdapterProvesNativeAssertionFailureRED is the unsafe challenge of
// the same frozen suite: exactly one registered assertion evaluates false,
// the failure is reconciled to its exact call site, every OTHER registered
// assertion still executes and passes.
func TestGoAdapterProvesNativeAssertionFailureRED(t *testing.T) {
	files := conformanceFiles(t, func(source string) string {
		return strings.Replace(source, "answer == 42", "answer == 43", 1)
	})
	suite := conformanceSuite(t, files)
	source := string(files["conformance_test.go"])
	execution, events, err := runGoSuite(t, files, suite, nil)
	if err != nil {
		t.Fatalf("expected a reconciled assertion failure, got error: %v", err)
	}
	if execution.Outcome != "assertion-fail" {
		t.Fatalf("outcome is %q, want assertion-fail", execution.Outcome)
	}
	if execution.ExitCode == nil || *execution.ExitCode == 0 {
		t.Fatalf("native exit must be nonzero on failure: %+v", execution.ExitCode)
	}
	requireEvent(t, events, "assertion", "TestConformanceWitnessPass", "conformance/witness-pass", "assertion-fail")
	requireEvent(t, events, "test-end", "TestConformanceWitnessPass", "", "assertion-fail")
	requireEvent(t, events, "test-end", "TestConformanceMultipleAssertions", "", "pass")
	requireEvent(t, events, "test-end", "TestConformanceSubtestIdentity/first", "", "pass")
	for _, e := range events {
		if e.Kind == "assertion" && e.Assertion == "conformance/witness-pass" {
			if e.Source != "conformance_test.go" || e.Line != assertionLine(t, source, "conformance/witness-pass") {
				t.Fatalf("failing assertion not bound to its frozen site: %+v", e)
			}
		}
	}
}

// negativeModuleFiles is the frozen adversarial fixture module whose cases
// are selected one suite at a time.
func negativeModuleFiles(t *testing.T) map[string][]byte {
	t.Helper()
	source := `package negatives

import (
	"os"
	"testing"

	"neg.test/machinerycheck"
)

func TestDirectFatal(t *testing.T) {
	t.Fatal("direct failure is not assertion evidence")
}

func setup(t *testing.T) {
	t.Fatal("setup failure before any assertion")
}

func TestSetupFatal(t *testing.T) {
	setup(t)
	machinerycheck.Check(t, "neg/setup-after", true)
}

func TestEarlyExit(t *testing.T) {
	os.Exit(3)
}

func TestExtraFailure(t *testing.T) {
	machinerycheck.Check(t, "neg/extra", false)
	t.Errorf("extra direct failure")
}

func TestSkipChild(t *testing.T) {
	t.Run("kid", func(t *testing.T) {
		t.Skip("skipped child cannot become success")
	})
}

func TestDuplicateNames(t *testing.T) {
	t.Run("twin", func(t *testing.T) {
		machinerycheck.Check(t, "neg/twin-a", true)
	})
	t.Run("twin", func(t *testing.T) {
		machinerycheck.Check(t, "neg/twin-b", true)
	})
}
`
	return map[string][]byte{
		"go.mod":           []byte("module neg.test\n\ngo 1.27\n"),
		"negative_test.go": []byte(source),
	}
}

func negativeSuite(t *testing.T, paths ...string) tdd.Suite {
	t.Helper()
	files := negativeModuleFiles(t)
	source := string(files["negative_test.go"])
	suite := tdd.Suite{
		ID: "go-negative", Adapter: "go-testing/v1", Root: ".",
		Files: []string{"go.mod", "negative_test.go"},
	}
	assertionsOf := map[string]string{
		"TestSetupFatal":          "neg/setup-after",
		"TestExtraFailure":        "neg/extra",
		"TestDuplicateNames/twin": "neg/twin-a",
	}
	for _, path := range paths {
		test := tdd.Test{ID: path, Source: "negative_test.go", Native: tdd.NativeID{Package: negativeModule, Test: path}}
		if id, ok := assertionsOf[path]; ok && strings.Contains(source, fmt.Sprintf("%q", id)) {
			test.Assertions = []tdd.Assertion{{ID: id, Source: "negative_test.go", Line: assertionLine(t, source, id), Helper: "machinery-check/v1"}}
		}
		suite.Tests = append(suite.Tests, test)
	}
	return suite
}

// TestGoAdapterRejectsNonAssertionFailureEvidence proves direct t.Fatal in a
// test body is never eligible assertion-based RED.
func TestGoAdapterRejectsDirectFatalAsAssertionEvidence(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestDirectFatal"), nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("direct t.Fatal must fail UNEXPECTED_FAILURE, got %v", err)
	}
}

// TestGoAdapterRejectsSetupFatalBeforeAssertion proves a setup failure
// before the registered assertion is an error, not assertion RED.
func TestGoAdapterRejectsSetupFatalBeforeAssertion(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestSetupFatal"), nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("setup t.Fatal must fail UNEXPECTED_FAILURE, got %v", err)
	}
}

// TestGoAdapterRejectsEarlyProcessExit proves an abrupt test-binary exit
// cannot masquerade as a complete execution.
func TestGoAdapterRejectsEarlyProcessExit(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestEarlyExit"), nil)
	if err == nil {
		t.Fatal("early os.Exit must fail closed")
	}
}

// TestGoAdapterRejectsExtraDirectFailure proves a helper-backed failure plus
// an extra direct t.Errorf is rejected: failure accounting is exact.
func TestGoAdapterRejectsExtraDirectFailure(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestExtraFailure"), nil)
	if err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("extra direct failure must fail UNEXPECTED_FAILURE, got %v", err)
	}
}

// TestGoAdapterRejectsSkippedSubtest proves a required skip can never become
// success.
func TestGoAdapterRejectsSkippedSubtest(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestSkipChild", "TestSkipChild/kid"), nil)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "skip") {
		t.Fatalf("skipped subtest must fail on skip, got %v", err)
	}
}

// TestGoAdapterDetectsDuplicateSubtestNames proves duplicate child names
// surface as unexpected native identities, never silent coverage.
func TestGoAdapterDetectsDuplicateSubtestNames(t *testing.T) {
	_, _, err := runGoSuite(t, negativeModuleFiles(t), negativeSuite(t, "TestDuplicateNames", "TestDuplicateNames/twin"), nil)
	if err == nil || !strings.Contains(err.Error(), "DUPLICATE_TEST") {
		t.Fatalf("duplicate subtest identity must fail DUPLICATE_TEST, got %v", err)
	}
}

// TestGoAdapterDetectsLateSourceMutation proves materialized sources are
// re-verified after preparation: a scratch mutation fails the run.
func TestGoAdapterDetectsLateSourceMutation(t *testing.T) {
	files := conformanceFiles(t, nil)
	suite := conformanceSuite(t, files)
	_, _, err := runGoSuite(t, files, suite, func(runDir string) {
		path := filepath.Join(runDir, "module", "conformance_test.go")
		if err := os.WriteFile(path, []byte("package conformance\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("late source mutation must fail STALE_INPUT, got %v", err)
	}
}

// TestGoAdapterRejectsAssertionSiteMismatch proves Prepare rejects a
// declared call site that is not the typed helper call at that exact line.
func TestGoAdapterRejectsAssertionSiteMismatch(t *testing.T) {
	files := conformanceFiles(t, nil)
	suite := conformanceSuite(t, files)
	suite.Tests[0].Assertions[0].Line++
	_, _, err := runGoSuite(t, files, suite, nil)
	if err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("site mismatch must fail ASSERTION_MISMATCH at Prepare, got %v", err)
	}
}

// TestGoAdapterRejectsLookalikeHelper proves a frozen suite importing a
// different helper package is rejected: only the byte-pinned embedded
// transport may witness assertions.
func TestGoAdapterRejectsLookalikeHelper(t *testing.T) {
	files := conformanceFiles(t, func(source string) string {
		return strings.Replace(source, `"machinery.test/conformance/machinerycheck"`, `"machinery.test/conformance/lookalike"`, 1)
	})
	// materialize a lookalike helper so the module still compiles standalone
	files["lookalike/machinerycheck.go"] = []byte("package machinerycheck\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"runtime\"\n\t\"testing\"\n)\n\nfunc Check(t *testing.T, id string, condition bool) {\n\t_, file, line, _ := runtime.Caller(1)\n\tfmt.Fprintf(os.Stdout, \"machinery-check/v1 witness id=%s value=%t test=%s site=%s:%d\\n\", id, condition, t.Name(), file, line)\n\tif !condition {\n\t\tt.Errorf(\"machinery-check/v1 assertion %s evaluated false at %s:%d\", id, file, line)\n\t}\n}\n")
	suite := conformanceSuite(t, files)
	_, _, err := runGoSuite(t, files, suite, nil)
	if err == nil || (!strings.Contains(err.Error(), "ASSERTION_MISMATCH") && !strings.Contains(err.Error(), "UNSUPPORTED")) {
		t.Fatalf("lookalike helper transport must be rejected, got %v", err)
	}
}

// TestGoAdapterRejectsRuntimeIdentityMismatch proves a suite declaring a
// different runtime version fails closed before any work.
func TestGoAdapterRejectsRuntimeIdentityMismatch(t *testing.T) {
	files := conformanceFiles(t, nil)
	suite := conformanceSuite(t, files)
	_, _, err := runGoSuiteChecked(t, files, suite, nil, func(s *tdd.Suite) {
		s.Runtime = tdd.RuntimeRef{Profile: "go", Version: "1.26.0", Platform: "darwin/arm64", Closure: "sha256:" + strings.Repeat("a", 64)}
	})
	if err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("runtime identity mismatch must fail UNSUPPORTED_VERSION, got %v", err)
	}
}

// TestGoAdapterRejectsMissingRuntime proves an absent pinned runtime fails
// before any suite work.
func TestGoAdapterRejectsMissingRuntime(t *testing.T) {
	if _, err := runtimeclosure.OpenGo(context.Background(), runtimeclosure.GoRequest{RuntimeRoot: filepath.Join(t.TempDir(), "absent")}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("absent runtime must fail UNSUPPORTED_VERSION, got %v", err)
	}
}

//
// Required contributor lane fragment (MAC-bz1y closed catalog + this
// story's assurance-go.json).
//

var (
	laneBinaryOnce sync.Once
	laneBinaryPath string
	laneBinaryErr  error
)

func laneBinary(t *testing.T) string {
	t.Helper()
	laneBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "machinery-lane-binary-")
		if err != nil {
			laneBinaryErr = err
			return
		}
		bin := filepath.Join(dir, "integration-lane")
		goExe, err := exec.LookPath("go")
		if err != nil {
			laneBinaryErr = err
			return
		}
		scope := openAdapterScope(t)
		defer closeAdapterScope(t, scope)
		attached, err := scope.Attach(processscope.Command{
			Executable: goExe,
			Args:       []string{"build", "-o", bin, "./scripts/integration-lane"},
			Dir:        repoRoot(t),
			Env:        os.Environ(),
			RuntimeDigest: "sha256:" + func() string {
				exe, err := exec.LookPath("go")
				if err != nil {
					laneBinaryErr = err
					return ""
				}
				body, err := os.ReadFile(exe)
				if err != nil {
					laneBinaryErr = err
					return ""
				}
				sum := sha256.Sum256(body)
				return fmt.Sprintf("%x", sum)
			}(),
		})
		if laneBinaryErr != nil {
			return
		}
		if err != nil {
			laneBinaryErr = err
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if _, err := scope.Run(ctx, attached, processscope.Streams{}); err != nil {
			laneBinaryErr = fmt.Errorf("build lane: %w", err)
			return
		}
		if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
			laneBinaryErr = fmt.Errorf("lane binary missing after build")
			return
		}
		laneBinaryPath = bin
	})
	if laneBinaryErr != nil {
		t.Fatalf("lane binary: %v", laneBinaryErr)
	}
	return laneBinaryPath
}

// laneSeedRoot builds a foreign-module fixture lane root carrying the frozen
// v1 pilot lane, the complete assurance catalog and this story's fragment.
func laneSeedRoot(t *testing.T, includeGoFragment bool, tamperConformance func(source string) string) string {
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
	repo := repoRoot(t)
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
	// frozen probe fixture tree
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
	// this story's frozen conformance fixture sources at their repo paths
	conformance := assetBytes(t, "conformance/conformance_test.go")
	if tamperConformance != nil {
		conformance = []byte(tamperConformance(string(conformance)))
	}
	write("internal/tdd/adapters/assets/go/conformance/go.mod", assetBytes(t, "conformance/go.mod"))
	write("internal/tdd/adapters/assets/go/conformance/conformance_test.go", conformance)
	if includeGoFragment {
		write("testdata/integration-lanes/assurance-go.json", []byte(laneFragmentAssuranceGo))
	}
	return root
}

type laneReport struct {
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

func laneRun(t *testing.T, root string, envOverrides ...string) (bool, string, laneReport) {
	t.Helper()
	bin := laneBinary(t)
	work := t.TempDir()
	reportPath := filepath.Join(work, "report.json")
	scope := openAdapterScope(t)
	defer closeAdapterScope(t, scope)
	body, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	env := os.Environ()
	overrides := map[string]string{}
	for _, item := range envOverrides {
		key, value, _ := strings.Cut(item, "=")
		overrides[key] = value
	}
	filtered := make([]string, 0, len(env)+len(overrides))
	for _, item := range env {
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
	var report laneReport
	if raw, readErr := os.ReadFile(reportPath); readErr == nil {
		if json.Unmarshal(raw, &report) != nil {
			t.Fatalf("malformed lane report")
		}
	}
	return rc == 0 && report.Status == "passed", stdout.String() + "\n" + stderr.String(), report
}

// TestContributorLaneExecutesGoConformanceFragment proves the required lane
// executes this story's fragment as a real native adapter conformance suite
// alongside the frozen probes and the v1 pilot lane.
func TestContributorLaneExecutesGoConformanceFragment(t *testing.T) {
	root := laneSeedRoot(t, true, nil)
	ok, out, report := laneRun(t, root)
	if !ok {
		t.Fatalf("required lane with the go conformance fragment must pass: %+v out=%q", report, out)
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
		if report.Assurance.Suites[i].ID == "assurance-go-conformance" {
			receipt = &report.Assurance.Suites[i]
		}
	}
	if receipt == nil {
		t.Fatalf("go conformance suite receipt missing: %+v", report.Assurance.Suites)
	}
	if receipt.Selected != 5 || receipt.Started != 5 || receipt.Passed != 5 || receipt.Failed != 0 || receipt.Skipped != 0 {
		t.Fatalf("conformance receipt not accounted exactly: %+v", receipt)
	}
	if receipt.Events == "" || len(receipt.EventsSHA) != 64 {
		t.Fatalf("conformance events not retained: %+v", receipt)
	}
	events, err := os.ReadFile(receipt.Events)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(events)) != receipt.EventsSHA {
		t.Fatalf("retained conformance events do not match the report hash: %v", err)
	}
	if !strings.Contains(string(events), "TestConformanceWitnessPass") {
		t.Fatalf("retained conformance events are not the native normalized stream: %s", events)
	}
}

// TestContributorLaneRejectsConformanceOmission proves omitting this story's
// required fragment fails the closed union instead of silently shrinking it.
func TestContributorLaneRejectsConformanceOmission(t *testing.T) {
	root := laneSeedRoot(t, false, nil)
	ok, out, _ := laneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "conformance") {
		t.Fatalf("omitting the go conformance fragment must fail the union, ok=%v out=%q", ok, out)
	}
}

// TestContributorLaneRejectsNativeSkipInConformanceFixture proves a native
// skip inside the conformance fixture fails the lane on the real stream.
func TestContributorLaneRejectsNativeSkipInConformanceFixture(t *testing.T) {
	root := laneSeedRoot(t, true, func(source string) string {
		return strings.Replace(source, "answer := 6 * 7", "t.Skip(\"required conformance must not skip\")\n\tanswer := 6 * 7", 1)
	})
	ok, out, _ := laneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "skip") {
		t.Fatalf("native skip in the conformance fixture must fail the lane, ok=%v out=%q", ok, out)
	}
}

// TestContributorLaneRejectsRuntimeAbsence proves a stale go runtime fails
// the lane before any suite runs.
func TestContributorLaneRejectsRuntimeAbsence(t *testing.T) {
	root := laneSeedRoot(t, true, nil)
	shim := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then printf 'go version go1.26.9 %s/%s\\n' \"$(uname -s | tr A-Z a-z)\" \"$(uname -m)\"; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(shim, "go"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, out, _ := laneRun(t, root, "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if ok || !strings.Contains(strings.ToLower(out), "unsupported go runtime") {
		t.Fatalf("stale go runtime must fail provisioning, ok=%v out=%q", ok, out)
	}
}
