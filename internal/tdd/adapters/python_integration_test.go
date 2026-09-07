package adapters

// RED contract for MAC-imtz (native subjects): the closed python-unittest/v1
// adapter executes REAL Python suites — pinned CPython 3.14.7 in isolated
// mode, the embedded byte-pinned helper and bootstrap/harness, real stdlib
// unittest lifecycle with full TestCase.id() reconciliation, processscope
// custody, normalized machinery.tdd.event/v1 accounting — through the REAL
// production chain (InitStore -> Capture -> OpenPython+Validate -> Prepare
// -> Run -> process-free Close). Import/syntax failures, skips, xfails,
// setUp assertions, custom failureException, subTest, load_tests, monkey
// patching, early exits, async residue, stale bytecode, unwitnessed
// failures, site/runtime mismatches and never-settling tasks are rejected.
// The required contributor lane fragment executes alongside the frozen
// catalog. The package TestMain (processscope internal-IO dispatcher) is
// defined once in go_integration_test.go.

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/processscope"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	"github.com/RamXX/machinery/internal/tdd"
)

// laneFragmentAssurancePython is the exact frozen fragment GREEN ships at
// testdata/integration-lanes/assurance-python.json.
const laneFragmentAssurancePython = `{
  "schema": "machinery.assurance.lane/v1",
  "fragment": "assurance-python",
  "owner": "MAC-imtz",
  "suites": [
    {
      "id": "assurance-python-conformance",
      "lane": "assurance",
      "adapter": "python-unittest/v1",
      "kind": "native-conformance",
      "package": ".",
      "source_files": [
        "internal/tdd/adapters/assets/python/conformance/conformance_test.py"
      ],
      "tests": [
        "conformance_test.ConformanceAsync.test_async_witness",
        "conformance_test.ConformanceIdentity.test_class_identity",
        "conformance_test.ConformanceMultiple.test_multiple",
        "conformance_test.ConformanceWitness.test_witness_pass"
      ],
      "runtimes": ["python"],
      "timeout": "5m",
      "stdout_limit": 1048576,
      "stderr_limit": 1048576
    }
  ]
}
`

func pyAssetBytes(t *testing.T, rel string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(tsRepoRoot(t), "internal", "tdd", "adapters", "assets", "python", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// pyChainResult is one completed production-chain suite execution.
type pyChainResult struct {
	execution tdd.Execution
	events    []tdd.Event
}

// pyRunCaptured captures frozen fixture files into a real content-addressed
// store bundle and executes the declared suite through the closed production
// chain on a fresh custody root (the lane's own per-suite pattern): pinned
// CPython runtime closure validated under the live scope, adapter Prepare
// over the captured bundle, adapter Run with the recording sink.
func pyRunCaptured(t *testing.T, ctx context.Context, files map[string][]byte, suite tdd.Suite, mutatePrepared func(scratch string)) (pyChainResult, error) {
	t.Helper()
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	var result pyChainResult
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
	handle, err := runtimeclosure.OpenPython(ctx, runtimeclosure.PythonRequest{})
	if err != nil {
		return result, fmt.Errorf("pinned CPython closure: %w", err)
	}
	defer func() { _ = handle.Close() }()
	if err := handle.Validate(ctx, scope); err != nil {
		return result, fmt.Errorf("pinned CPython closure validation: %w", err)
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
		Inputs: inputs, Manifest: manifest, Name: "py-conformance", Store: store,
		Limits: tdd.Limits{WallMS: 300000},
	})
	if err != nil {
		return result, fmt.Errorf("capture: %w", err)
	}
	if bundle.Materialized() == "" {
		return result, errors.New("capture produced no verified materialization")
	}
	adapter := Python()
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

// pyCallLine locates the exact line of an assertion's typed helper call in
// frozen fixture bytes.
func pyCallLine(t *testing.T, source []byte, id string) int64 {
	t.Helper()
	anchor := `machinery_check.check(self, "` + id + `"`
	for i, line := range strings.Split(string(source), "\n") {
		if strings.Contains(line, anchor) {
			return int64(i + 1)
		}
	}
	t.Fatalf("frozen fixture lacks the %q call site", id)
	return 0
}

// pyConformanceSuite declares the frozen fixture's closed inventory: four
// native TestCase.id() identities (including the IsolatedAsyncioTestCase
// member) and the five registered machinery-check/v1 call sites.
func pyConformanceSuite(t *testing.T, id string) tdd.Suite {
	t.Helper()
	fixture := pyAssetBytes(t, "conformance/conformance_test.py")
	suite := tdd.Suite{ID: id, Adapter: AdapterPythonUnittest, Root: ".", Files: []string{"conformance_test.py"}}
	leaf := func(testID, class, method string, ids ...string) tdd.Test {
		test := tdd.Test{ID: testID, Source: "conformance_test.py", Native: tdd.NativeID{Module: "conformance_test", Class: class, Method: method}}
		for _, assertionID := range ids {
			test.Assertions = append(test.Assertions, tdd.Assertion{
				ID: assertionID, Source: "conformance_test.py",
				Line: pyCallLine(t, fixture, assertionID), Helper: tdd.AssertionHelperV1,
			})
		}
		return test
	}
	suite.Tests = []tdd.Test{
		leaf("async", "ConformanceAsync", "test_async_witness", "conformance/async-witness"),
		leaf("identity", "ConformanceIdentity", "test_class_identity", "conformance/class-identity"),
		leaf("multiple", "ConformanceMultiple", "test_multiple", "conformance/multi-a", "conformance/multi-b"),
		leaf("witness", "ConformanceWitness", "test_witness_pass", "conformance/witness-pass"),
	}
	return suite
}

// TestPythonAdapterExecutesRealUnittestSuiteNatively is the positive native
// proof: the frozen fixture executes through the real stdlib unittest
// lifecycle under CPython 3.14.7 isolated mode with the embedded harness;
// the normalized event stream accounts every identity and every registered
// assertion, including the async member.
func TestPythonAdapterExecutesRealUnittestSuiteNatively(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	result, err := pyRunCaptured(t, ctx, map[string][]byte{"conformance_test.py": pyAssetBytes(t, "conformance/conformance_test.py")}, pyConformanceSuite(t, "native-positive"), nil)
	if err != nil {
		t.Fatalf("real native suite execution failed: %v", err)
	}
	if result.execution.Outcome != "pass" || result.execution.ExitCode == nil || *result.execution.ExitCode != 0 {
		t.Fatalf("execution outcome %q exit %+v is not a native pass", result.execution.Outcome, result.execution.ExitCode)
	}
	if result.execution.Custody.Status != processscope.StatusCleaned {
		t.Fatalf("custody did not verify: %+v", result.execution.Custody)
	}
	if len(result.execution.StdoutDigest) != 64 || len(result.execution.StderrDigest) != 64 {
		t.Fatalf("raw stream digests missing: %+v", result.execution)
	}
	assertions := map[string]string{}
	started, ended := map[string]int{}, map[string]int{}
	for _, e := range result.events {
		if e.Schema != PythonEventSchema || e.Suite != "native-positive" {
			t.Fatalf("normalized event identity wrong: %+v", e)
		}
		switch e.Kind {
		case "assertion":
			assertions[e.Assertion] = e.Outcome
		case "test-start":
			started[pythonNativeID(*e.Native)]++
		case "test-end":
			ended[pythonNativeID(*e.Native)]++
		}
	}
	for _, id := range []string{"conformance/async-witness", "conformance/class-identity", "conformance/multi-a", "conformance/multi-b", "conformance/witness-pass"} {
		if assertions[id] != "pass" {
			t.Fatalf("registered assertion %s not witnessed as pass: %v", id, assertions)
		}
	}
	for _, identity := range []string{
		"conformance_test.ConformanceAsync.test_async_witness",
		"conformance_test.ConformanceIdentity.test_class_identity",
		"conformance_test.ConformanceMultiple.test_multiple",
		"conformance_test.ConformanceWitness.test_witness_pass",
	} {
		if started[identity] != 1 || ended[identity] != 1 {
			t.Fatalf("native identity %s not accounted exactly: started=%v ended=%v", identity, started[identity], ended[identity])
		}
	}
	kinds := map[string]int{}
	for _, e := range result.events {
		kinds[e.Kind]++
	}
	for _, want := range []string{"suite-start", "discovered", "test-start", "assertion", "test-end", "suite-end"} {
		if kinds[want] == 0 {
			t.Fatalf("normalized stream lacks %s events: %v", want, kinds)
		}
	}
}

// TestPythonAdapterProvesNativeAssertionFailureRED freezes the same-test
// safe/unsafe calibration: the identical suite passes on the safe control
// and fails AT THE EXACT REGISTERED ASSERTION on the unsafe challenge, with
// the helper witness and the native AssertionError cause.
func TestPythonAdapterProvesNativeAssertionFailureRED(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	frozen := pyAssetBytes(t, "conformance/conformance_test.py")
	files := map[string][]byte{"conformance_test.py": frozen}
	safeResult, err := pyRunCaptured(t, ctx, files, pyConformanceSuite(t, "calibration-safe"), nil)
	if err != nil || safeResult.execution.Outcome != "pass" {
		t.Fatalf("safe control must pass natively: err=%v outcome=%q", err, safeResult.execution.Outcome)
	}
	unsafe := strings.Replace(string(frozen), "6 * 7 == 42", "6 * 7 == 43", 1)
	files = map[string][]byte{"conformance_test.py": []byte(unsafe)}
	unsafeResult, err := pyRunCaptured(t, ctx, files, pyConformanceSuite(t, "calibration-unsafe"), nil)
	if err != nil {
		t.Fatalf("unsafe challenge must be reconciled, not errored: %v", err)
	}
	if unsafeResult.execution.Outcome != "assertion-fail" {
		t.Fatalf("unsafe challenge outcome %q is not the accounted assertion-fail", unsafeResult.execution.Outcome)
	}
	if unsafeResult.execution.ExitCode == nil || *unsafeResult.execution.ExitCode != 1 {
		t.Fatalf("unsafe challenge exit %+v does not concord with the accounted failure", unsafeResult.execution.ExitCode)
	}
	challengeFailed, failEvent := false, false
	for _, e := range unsafeResult.events {
		if e.Kind == "assertion" && e.Assertion == "conformance/witness-pass" && e.Outcome == "assertion-fail" {
			challengeFailed = true
		}
		if e.Kind == "test-end" && e.Native != nil && e.Native.Method == "test_witness_pass" && e.Outcome == "assertion-fail" {
			failEvent = true
		}
	}
	if !challengeFailed || !failEvent {
		t.Fatalf("challenge did not fail at the registered assertion: witness=%v terminal=%v", challengeFailed, failEvent)
	}
}

// TestPythonAdapterRejectsAdversarialSuites freezes the negative calibration
// on real native executions: every adversarial shape is rejected with its
// machine diagnostic instead of becoming success or RED evidence.
func TestPythonAdapterRejectsAdversarialSuites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	frozen := string(pyAssetBytes(t, "conformance/conformance_test.py"))
	replace := func(old, new string) string { return strings.Replace(frozen, old, new, 1) }
	cases := []struct {
		name   string
		source string
		code   string
	}{
		{"syntax error", strings.Replace(frozen, "import unittest", "import unittest\n(oops", 1), "BUILD_ERROR"},
		{"module level raise", frozen + "\nraise ValueError(\"boom\")\n", "BUILD_ERROR"},
		{"skip decorator", replace("    def test_witness_pass(self):", "    @unittest.skip(\"no\")\n    def test_witness_pass(self):"), "UNSUPPORTED_FEATURE"},
		{"expected failure decorator", replace("    def test_witness_pass(self):", "    @unittest.expectedFailure\n    def test_witness_pass(self):"), "UNSUPPORTED_FEATURE"},
		{"custom failure exception", replace("class ConformanceWitness(unittest.TestCase):", "class ConformanceWitness(unittest.TestCase):\n    failureException = ValueError"), "UNSUPPORTED_FEATURE"},
		{"subtest", replace("        machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42)", "        machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42)\n        with self.subTest(i=1):\n            pass"), "UNSUPPORTED_FEATURE"},
		{"load_tests omission", frozen + "\n\ndef load_tests(loader, tests, pattern):\n    return tests\n", "UNSUPPORTED_FEATURE"},
		{"early interpreter exit", replace("        machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42)", "        import sys; machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42); sys.exit(0)"), "UNSUPPORTED_FEATURE"},
		{"unittest monkeypatch", replace("        machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42)", "        unittest.TestCase.run = lambda self, result: None; machinery_check.check(self, \"conformance/witness-pass\", 6 * 7 == 42)"), "UNSUPPORTED_FEATURE"},
		{"helper call in setup", frozen + "\n\nclass SetupOnly(unittest.TestCase):\n    def setUp(self):\n        machinery_check.check(self, \"setup/only\", True)\n", "UNSUPPORTED_FEATURE"},
	}
	for _, c := range cases {
		files := map[string][]byte{"conformance_test.py": []byte(c.source)}
		_, err := pyRunCaptured(t, ctx, files, pyConformanceSuite(t, "adversarial"), nil)
		if err == nil || !strings.Contains(err.Error(), c.code) {
			t.Fatalf("%s: want %s, got %v", c.name, c.code, err)
		}
	}
}

// TestPythonAdapterRejectsRuntimeFailures freezes the runtime-shape
// calibration on real native executions: setUp assertions, unwitnessed
// direct failures, cancelled and unawaited async work and duplicated
// terminals are rejected even though the process ran to completion.
func TestPythonAdapterRejectsRuntimeFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	type fixture struct {
		name, source, class, method, id string
		code                            string
	}
	fixtures := []fixture{
		{
			name: "setUp assertion", class: "S", method: "test_s", id: "shape/s", code: "UNEXPECTED_FAILURE",
			source: "import unittest\nimport machinery_check\n\nclass S(unittest.TestCase):\n    def setUp(self):\n        self.assertEqual(1, 2)\n    def test_s(self):\n        machinery_check.check(self, \"shape/s\", True)\n",
		},
		{
			name: "unwitnessed direct failure", class: "D", method: "test_d", id: "shape/d", code: "UNEXPECTED_FAILURE",
			source: "import unittest\nimport machinery_check\n\nclass D(unittest.TestCase):\n    def test_d(self):\n        machinery_check.check(self, \"shape/d\", True)\n        self.assertEqual(1, 2)\n",
		},
		{
			name: "cancelled async work", class: "C", method: "test_c", id: "shape/c", code: "UNEXPECTED_FAILURE",
			source: "import asyncio\nimport unittest\nimport machinery_check\n\nclass C(unittest.IsolatedAsyncioTestCase):\n    async def test_c(self):\n        machinery_check.check(self, \"shape/c\", True)\n        raise asyncio.CancelledError()\n",
		},
		{
			name: "unawaited coroutine", class: "U", method: "test_u", id: "shape/u", code: "UNEXPECTED_FAILURE",
			source: "import unittest\nimport machinery_check\n\nclass U(unittest.IsolatedAsyncioTestCase):\n    async def test_u(self):\n        async def inner():\n            return 1\n        inner()\n        machinery_check.check(self, \"shape/u\", True)\n",
		},
		{
			name: "duplicate terminal", class: "T", method: "test_t", id: "shape/t", code: "DUPLICATE_TERMINAL",
			source: "import unittest\nimport machinery_check\n\nclass T(unittest.TestCase):\n    def test_t(self):\n        machinery_check.check(self, \"shape/t\", 1 + 1 == 3)\n    def tearDown(self):\n        raise RuntimeError(\"teardown boom\")\n",
		},
	}
	for _, f := range fixtures {
		files := map[string][]byte{"shapes_test.py": []byte(f.source)}
		suite := tdd.Suite{ID: "shape", Adapter: AdapterPythonUnittest, Root: ".", Files: []string{"shapes_test.py"}}
		test := tdd.Test{ID: f.name, Source: "shapes_test.py", Native: tdd.NativeID{Module: "shapes_test", Class: f.class, Method: f.method}}
		test.Assertions = append(test.Assertions, tdd.Assertion{ID: f.id, Source: "shapes_test.py", Line: pyCallLine(t, []byte(f.source), f.id), Helper: tdd.AssertionHelperV1})
		suite.Tests = []tdd.Test{test}
		_, err := pyRunCaptured(t, ctx, files, suite, nil)
		if err == nil || !strings.Contains(err.Error(), f.code) {
			t.Fatalf("%s: want %s, got %v", f.name, f.code, err)
		}
	}
}

// TestPythonAdapterRejectsStaleBytecode freezes the stale-pyc oracle proven
// empirically during RED: CPython executes a planted unchecked-hash pyc even
// when fresh sources are present, so any __pycache__ in the prepared suite
// tree — before or after preparation — is STALE_INPUT, never evidence.
func TestPythonAdapterRejectsStaleBytecode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	plant := func(scratch string) {
		suiteDir := filepath.Join(scratch, "suite")
		cache := filepath.Join(suiteDir, "__pycache__")
		if err := os.MkdirAll(cache, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cache, "conformance_test.cpython-314.pyc"), []byte("stale bytecode"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := pyRunCaptured(t, ctx, map[string][]byte{"conformance_test.py": pyAssetBytes(t, "conformance/conformance_test.py")}, pyConformanceSuite(t, "stale-pyc-prepared"), plant)
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("planted bytecode after preparation must be rejected: %v", err)
	}
	captured := map[string][]byte{
		"conformance_test.py":                          pyAssetBytes(t, "conformance/conformance_test.py"),
		"__pycache__/conformance_test.cpython-314.pyc": []byte("stale bytecode"),
	}
	_, err = pyRunCaptured(t, ctx, captured, pyConformanceSuite(t, "stale-pyc-captured"), nil)
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("stale bytecode captured with the suite must be rejected at preparation: %v", err)
	}
}

// TestPythonAdapterRejectsStalePreparedOutput freezes the late-mutation
// boundary: tampering the materialized helper bytes between Prepare and Run
// fails closed instead of executing stale bytes.
func TestPythonAdapterRejectsStalePreparedOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	_, err := pyRunCaptured(t, ctx, map[string][]byte{"conformance_test.py": pyAssetBytes(t, "conformance/conformance_test.py")}, pyConformanceSuite(t, "stale"), func(scratch string) {
		helper := filepath.Join(scratch, "suite", PythonHelperFile)
		if err := os.WriteFile(helper, []byte("# stale swap\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	})
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("stale prepared bytes must be rejected: %v", err)
	}
}

// TestPythonAdapterRejectsAssertionSiteMismatch freezes the static source
// binding: a registered assertion whose frozen line is not the exact typed
// helper call site is rejected at Prepare time.
func TestPythonAdapterRejectsAssertionSiteMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	suite := pyConformanceSuite(t, "site-mismatch")
	suite.Tests[3].Assertions[0].Line = 3
	_, err := pyRunCaptured(t, ctx, map[string][]byte{"conformance_test.py": pyAssetBytes(t, "conformance/conformance_test.py")}, suite, nil)
	if err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("registered call-site mismatch must be rejected: %v", err)
	}
}

// TestPythonAdapterRejectsRuntimeIdentityMismatch freezes the runtime
// binding: a suite declaring a different runtime closure is rejected before
// any work, and a missing handle never starts.
func TestPythonAdapterRejectsRuntimeIdentityMismatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	suite := pyConformanceSuite(t, "runtime-mismatch")
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	handle, err := runtimeclosure.OpenPython(ctx, runtimeclosure.PythonRequest{})
	if err != nil {
		t.Fatalf("closure: %v", err)
	}
	defer func() { _ = handle.Close() }()
	identity := handle.Identity()
	identity.Version = "3.13.0"
	suite.Runtime = identity
	src := t.TempDir()
	inputs := tdd.InputView{
		SourceRoot: src, DesignPath: ".", ControlRoot: t.TempDir(),
		Revalidate: func() error { return nil }, Release: func() error { return nil },
	}
	if _, err := Python().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: tdd.BundleRef{}, Scratch: filepath.Join(t.TempDir(), "run"),
		Runtime: handle, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("runtime identity mismatch must be rejected before work: %v", err)
	}
	if _, err := Python().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: pyConformanceSuite(t, "runtime-missing"), Source: tdd.BundleRef{},
		Scratch: filepath.Join(t.TempDir(), "run2"), Runtime: nil, Scope: scope, Limits: tdd.Limits{WallMS: 60000},
	}); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_VERSION") {
		t.Fatalf("missing runtime handle must be rejected: %v", err)
	}
}

// TestPythonAdapterRejectsForeignPreparedState freezes the opaque prepared
// boundary: a foreign or zero prepared carrier carries no authority.
func TestPythonAdapterRejectsForeignPreparedState(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := Python().Run(ctx, tdd.PreparedSuite{}, func(tdd.Event) error { return nil }); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Fatalf("foreign prepared state must be rejected: %v", err)
	}
}

// TestPythonAdapterRejectsNeverSettlingAsyncTaskAsTimeout freezes the wall
// boundary on a real hung native process: an uncancellable background task
// keeps the runner's teardown alive until the scoped deadline kills the job
// and the run fails TIMEOUT, never success.
func TestPythonAdapterRejectsNeverSettlingAsyncTaskAsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	source := "import asyncio\nimport unittest\nimport machinery_check\n\nclass H(unittest.IsolatedAsyncioTestCase):\n    async def test_h(self):\n        async def linger():\n            while True:\n                try:\n                    await asyncio.sleep(3600)\n                except asyncio.CancelledError:\n                    continue\n        asyncio.create_task(linger())\n        machinery_check.check(self, \"shape/h\", True)\n"
	suite := tdd.Suite{ID: "hang", Adapter: AdapterPythonUnittest, Root: ".", Files: []string{"shapes_test.py"},
		Tests: []tdd.Test{{
			ID: "hang", Source: "shapes_test.py", Native: tdd.NativeID{Module: "shapes_test", Class: "H", Method: "test_h"},
			Assertions: []tdd.Assertion{{ID: "shape/h", Source: "shapes_test.py", Line: pyCallLine(t, []byte(source), "shape/h"), Helper: tdd.AssertionHelperV1}},
		}}}
	scope := tsOpenAdapterScope(t)
	defer tsCloseAdapterScope(t, scope)
	src := t.TempDir()
	control := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "shapes_test.py"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(control, "plan.json"), []byte(`{"schema":"machinery.tdd.plan/v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := tdd.InputView{SourceRoot: src, DesignPath: ".", ControlRoot: control,
		Revalidate: func() error { return nil }, Release: func() error { return nil }}
	handle, err := runtimeclosure.OpenPython(ctx, runtimeclosure.PythonRequest{})
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
	prepared, err := Python().Prepare(ctx, tdd.SuiteRequest{
		Inputs: inputs, Suite: suite, Source: bundle, Scratch: filepath.Join(t.TempDir(), "run"),
		Runtime: handle, Scope: scope, Limits: tdd.Limits{WallMS: 20000, CleanupMS: 10000},
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	_, err = Python().Run(ctx, prepared, func(tdd.Event) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Fatalf("never-settling async task must die on the wall deadline as TIMEOUT: %v", err)
	}
}

//
// Required contributor lane fragment (MAC-bz1y closed catalog + this
// story's assurance-python.json).
//

// pyLaneSeedRoot builds a foreign-module fixture lane root carrying the
// frozen v1 pilot lane, the complete assurance catalog and this story's
// fragment with the frozen python conformance assets.
func pyLaneSeedRoot(t *testing.T, includeFragment bool, tamperConformance func(source string) string) string {
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
	laneDir := filepath.Join(tsRepoRoot(t), "testdata", "integration-lanes")
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
	write("testdata/integration-lanes/pilot.json", pilotBody)
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
	conformance := pyAssetBytes(t, "conformance/conformance_test.py")
	if tamperConformance != nil {
		conformance = []byte(tamperConformance(string(conformance)))
	}
	write("internal/tdd/adapters/assets/python/machinery_check.py", pyAssetBytes(t, "machinery_check.py"))
	write("internal/tdd/adapters/assets/python/bootstrap.py", pyAssetBytes(t, "bootstrap.py"))
	write("internal/tdd/adapters/assets/python/conformance/conformance_test.py", conformance)
	if includeFragment {
		write("testdata/integration-lanes/assurance-python.json", []byte(laneFragmentAssurancePython))
	}
	return root
}

// TestContributorLaneExecutesPythonConformanceFragment proves the required
// lane executes this story's fragment as a real native adapter conformance
// suite alongside the frozen probes and the v1 pilot lane.
func TestContributorLaneExecutesPythonConformanceFragment(t *testing.T) {
	root := pyLaneSeedRoot(t, true, nil)
	ok, out, report := tsLaneRun(t, root)
	if !ok {
		t.Fatalf("required lane with the python conformance fragment must pass: %+v output=%s", report, out)
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
		if report.Assurance.Suites[i].ID == "assurance-python-conformance" {
			receipt = &report.Assurance.Suites[i]
		}
	}
	if receipt == nil {
		t.Fatalf("python conformance suite receipt missing: %+v", report.Assurance.Suites)
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
	if !strings.Contains(string(events), `"Module":"conformance_test"`) || !strings.Contains(string(events), `"Method":"test_async_witness"`) {
		t.Fatalf("retained conformance events are not the native normalized stream: %s", events)
	}
}

// TestPythonContributorLaneRejectsConformanceOmission proves omitting this
// story's required fragment fails the closed union instead of silently
// shrinking it.
func TestPythonContributorLaneRejectsConformanceOmission(t *testing.T) {
	root := pyLaneSeedRoot(t, false, nil)
	ok, out, _ := tsLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "conformance") {
		t.Fatalf("omitting the python conformance fragment must fail the union, ok=%v out=%q", ok, out)
	}
}

// TestPythonContributorLaneRejectsNativeSkipInConformanceFixture proves a
// native skip inside the conformance fixture fails the lane on the real
// stream.
func TestPythonContributorLaneRejectsNativeSkipInConformanceFixture(t *testing.T) {
	root := pyLaneSeedRoot(t, true, func(source string) string {
		return strings.Replace(source, "    def test_witness_pass(self):", "    @unittest.skip(\"no\")\n    def test_witness_pass(self):", 1)
	})
	ok, out, _ := tsLaneRun(t, root)
	if ok || !strings.Contains(strings.ToLower(out), "skip") {
		t.Fatalf("native skip in the conformance fixture must fail the lane, ok=%v out=%q", ok, out)
	}
}

// TestPythonContributorLaneRejectsRuntimeAbsence proves a stale python
// runtime fails the lane before any suite runs.
func TestPythonContributorLaneRejectsRuntimeAbsence(t *testing.T) {
	root := pyLaneSeedRoot(t, true, nil)
	shim := t.TempDir()
	if err := os.WriteFile(filepath.Join(shim, "python3"), []byte("#!/bin/sh\necho Python 3.13.0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, out, _ := tsLaneRun(t, root, "PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	if ok || !strings.Contains(strings.ToLower(out), "unsupported python runtime") {
		t.Fatalf("stale python runtime must fail provisioning, ok=%v out=%q", ok, out)
	}
}
