package adapters

// Frozen RED unit suite for MAC-imtz (python-unittest/v1). Witness grammar,
// harness report vocabulary, reconciliation automaton and environment
// subjects over synthetic closed inputs; the native execution proofs live
// in python_integration_test.go. On the RED stub every behavior subject
// fails on the pending implementation; the byte-pin, fixture-freeze and
// lookup controls pass. pySampleReport is a REAL harness stream captured
// from a native CPython 3.14.7 execution of the frozen embedded bytes.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
)

func pyUnitSuite() *tdd.Suite {
	return &tdd.Suite{
		ID:      "python-unit",
		Adapter: AdapterPythonUnittest,
		Runtime: tdd.RuntimeRef{Profile: "python-unittest/v1", Version: "3.14.7", Platform: "darwin/arm64", Closure: "sha256:" + strings.Repeat("a", 64)},
		Root:    ".",
		Files:   []string{"unit_test.py"},
		Tests: []tdd.Test{
			{
				ID: "pass", Role: "positive", Source: "unit_test.py",
				Native:     tdd.NativeID{Module: "unit_test", Class: "UnitTests", Method: "test_pass"},
				Assertions: []tdd.Assertion{{ID: "unit/alpha", Source: "unit_test.py", Line: 6, Helper: "machinery-check/v1"}},
			},
			{
				ID: "fail", Role: "negative", Source: "unit_test.py",
				Native:     tdd.NativeID{Module: "unit_test", Class: "UnitTests", Method: "test_fail"},
				Assertions: []tdd.Assertion{{ID: "unit/beta", Source: "unit_test.py", Line: 9, Helper: "machinery-check/v1"}},
			},
		},
	}
}

// TestPythonLookupResolvesClosedIdentity is a passing control: the registry
// resolves exactly the closed identity registered through the documented
// seam.
func TestPythonLookupResolvesClosedIdentity(t *testing.T) {
	adapter, err := Lookup(AdapterPythonUnittest)
	if err != nil || adapter.ID() != AdapterPythonUnittest {
		t.Fatalf("closed identity did not resolve: %v %+v", err, adapter)
	}
	for _, id := range []string{"python-pytest/v1", "python-doctest/v1", "python-unittest/v2"} {
		if _, err := Lookup(id); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_ADAPTER") {
			t.Fatalf("foreign adapter identity %q must fail closed: %v", id, err)
		}
	}
}

// TestPythonHelperBytesAreBytePinned is a passing control: the embedded
// helper bytes equal the frozen pin and carry the closed transport contract.
func TestPythonHelperBytesAreBytePinned(t *testing.T) {
	body := PythonHelperSource()
	if digest(body) != PythonHelperPinnedSHA256 {
		t.Fatalf("embedded helper bytes are not the frozen pin: sha256:%s", digest(body))
	}
	for _, required := range []string{
		"machinery.tdd.witness/v1",
		"def check(case, assertion_id, condition):",
		`raise TypeError("machinery-check/v1 condition must be strictly boolean")`,
		"raise AssertionError(",
	} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("helper bytes lack the frozen transport fragment %q", required)
		}
	}
	for _, forbidden := range []string{"except ", "except:", "finally:"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("helper must never catch surrounding failures; found %q", forbidden)
		}
	}
}

// TestPythonBootstrapBytesAreBytePinned is a passing control: the embedded
// bootstrap/harness bytes equal the frozen pin and carry the closed harness
// mechanics (isolated interpreter preparation, stdlib identity anchors, the
// report stream with its terminal sentinel, and deterministic async residue
// detection).
func TestPythonBootstrapBytesAreBytePinned(t *testing.T) {
	body := PythonBootstrapSource()
	if digest(body) != PythonBootstrapPinnedSHA256 {
		t.Fatalf("embedded bootstrap bytes are not the frozen pin: sha256:%s", digest(body))
	}
	for _, required := range []string{
		"machinery.tdd.harness.python/v1",
		"machinery.tdd.suite.python/v1",
		"bootstrap-end",
		"sys.dont_write_bytecode = True",
		"warnings.simplefilter(\"error\", RuntimeWarning)",
		"sys.unraisablehook = recording_hook",
		"class HarnessResult(unittest.TestResult):",
		"def addSubTest(",
		"def addExpectedFailure(",
		"def addUnexpectedSuccess(",
		"failureException is not AssertionError",
		"load_tests",
	} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("bootstrap bytes lack the frozen harness fragment %q", required)
		}
	}
}

// TestPythonConformanceFixtureBytesAreFrozen is a passing control: the
// frozen conformance fixture bytes carry their pin and every registered
// identity, including the IsolatedAsyncioTestCase member.
func TestPythonConformanceFixtureBytesAreFrozen(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("assets", "python", "conformance", "conformance_test.py"))
	if err != nil {
		t.Fatal(err)
	}
	if digest(fixture) != PythonConformanceFixtureSHA256 {
		t.Fatalf("conformance fixture bytes drifted: sha256:%s", digest(fixture))
	}
	source := string(fixture)
	for _, id := range []string{"conformance/async-witness", "conformance/class-identity", "conformance/multi-a", "conformance/multi-b", "conformance/witness-pass"} {
		if !strings.Contains(source, `machinery_check.check(self, "`+id+`"`) {
			t.Fatalf("frozen fixture lacks the registered assertion call site %s", id)
		}
	}
	if !strings.Contains(source, "unittest.IsolatedAsyncioTestCase") {
		t.Fatal("frozen fixture must prove the stdlib async test surface")
	}
}

// TestParsePythonWitness freezes the witness grammar: the closed schema, the
// strictly boolean condition, the exact test identity, the file:line site
// and the recorded native failure class.
func TestParsePythonWitness(t *testing.T) {
	good := `{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "unit_test.UnitTests.test_pass", "thrown": null}`
	w, ok := parsePythonWitness(good)
	if !ok || w.assertion != "unit/alpha" || !w.condition || w.thrown != "" || w.file != "unit_test.py" || w.line != 6 || w.test != "unit_test.UnitTests.test_pass" {
		t.Fatalf("witness did not parse: %+v ok=%v", w, ok)
	}
	fail := `{"assertion": "unit/beta", "condition": false, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:9", "test": "unit_test.UnitTests.test_fail", "thrown": "AssertionError"}`
	w, ok = parsePythonWitness(fail)
	if !ok || w.condition || w.thrown != "AssertionError" || w.line != 9 {
		t.Fatalf("failing witness did not parse: %+v ok=%v", w, ok)
	}
	for _, bad := range []string{
		`{"assertion": "unit/alpha", "condition": 1, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "t", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.other/v1", "site": "unit_test.py:6", "test": "t", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "other", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "t", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:x", "test": "t", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:0", "test": "t", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "", "thrown": null}`,
		`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "t"}`,
		`not json`,
	} {
		if w, ok := parsePythonWitness(bad); ok {
			t.Fatalf("malformed witness must not parse: %q -> %+v", bad, w)
		}
	}
}

// pySampleReport is a frozen REAL harness stream captured from a native
// CPython 3.14.7 execution of the frozen embedded helper and bootstrap
// during RED (note the stdlib loader's alphabetical execution order).
const pySampleReport = `{"kind": "suite-start", "schema": "machinery.tdd.harness.python/v1", "tests": ["unit_test.UnitTests.test_fail", "unit_test.UnitTests.test_pass"]}
{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}
{"assertion": "unit/beta", "condition": false, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:9", "test": "unit_test.UnitTests.test_fail", "thrown": "AssertionError"}
{"exception": "AssertionError", "kind": "test-end", "outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}
{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}
{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "unit_test.UnitTests.test_pass", "thrown": null}
{"kind": "test-end", "outcome": "pass", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}
{"errors": 0, "expected_failures": 0, "failures": 1, "kind": "suite-end", "schema": "machinery.tdd.harness.python/v1", "skipped": 0, "success": false, "tests_run": 2, "unexpected_successes": 0}
{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}
`

// TestPythonReportStreamVocabulary freezes the closed harness JSONL
// vocabulary: the frozen REAL sample parses; unknown schemas, kinds and
// keys, malformed lines, missing/duplicated/non-terminal sentinels and
// trailing junk all fail closed.
func TestPythonReportStreamVocabulary(t *testing.T) {
	records, err := parsePythonReport([]byte(pySampleReport))
	if err != nil {
		t.Fatalf("frozen real harness stream must parse: %v", err)
	}
	if len(records) != 9 {
		t.Fatalf("frozen stream carries %d records, want 9", len(records))
	}
	if records[0].Kind != "suite-start" || len(records[0].Tests) != 2 || records[8].Kind != "bootstrap-end" {
		t.Fatalf("frozen stream shape wrong: %+v", records[0])
	}
	suiteEnd := records[7]
	if suiteEnd.Kind != "suite-end" || suiteEnd.TestsRun != 2 || suiteEnd.Failures != 1 || suiteEnd.Success {
		t.Fatalf("suite-end accounting wrong: %+v", suiteEnd)
	}
	bad := map[string]string{
		"unknown kind":       strings.Replace(pySampleReport, `{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, `{"kind": "test-mid", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, 1),
		"unknown schema":     strings.Replace(pySampleReport, "machinery.tdd.harness.python/v1", "machinery.tdd.harness.python/v2", 1),
		"unknown key":        strings.Replace(pySampleReport, `{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}`, `{"extra": 1, "kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}`, 1),
		"missing sentinel":   strings.TrimSuffix(pySampleReport, `{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}`+"\n"),
		"doubled sentinel":   pySampleReport + `{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}` + "\n",
		"midstream sentinel": strings.Replace(pySampleReport, `{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, `{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, 1),
		"malformed line":     pySampleReport + "{not json}\n",
		"empty stream":       "\n",
	}
	for name, stream := range bad {
		if _, err := parsePythonReport([]byte(stream)); err == nil {
			t.Fatalf("%s must fail closed", name)
		}
	}
}

// TestPythonReconciliationPassSequence freezes the legal automaton over the
// frozen REAL stream: both declared identities start/terminate exactly once,
// the failing test fails at its registered assertion with the native
// AssertionError cause, and the normalized machinery.tdd.event/v1 sequence is
// complete.
func TestPythonReconciliationPassSequence(t *testing.T) {
	records, err := parsePythonReport([]byte(pySampleReport))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := reconcilePythonReport(records, pyUnitSuite())
	if err != nil {
		t.Fatalf("legal stream must reconcile: %v", err)
	}
	if rec.Outcome != "assertion-fail" {
		t.Fatalf("reconciled outcome %q is not the accounted assertion-fail", rec.Outcome)
	}
	outcomes := map[string]string{}
	for _, e := range rec.Events {
		if e.Schema != PythonEventSchema || e.Suite != "python-unit" {
			t.Fatalf("normalized event identity wrong: %+v", e)
		}
		if e.Kind == "assertion" {
			outcomes[e.Assertion] = e.Outcome
		}
	}
	if outcomes["unit/alpha"] != "pass" || outcomes["unit/beta"] != "assertion-fail" {
		t.Fatalf("assertion outcomes wrong: %v", outcomes)
	}
	kinds := map[string]int{}
	for _, e := range rec.Events {
		kinds[e.Kind]++
	}
	for _, want := range []string{"suite-start", "discovered", "test-start", "assertion", "test-end", "suite-end"} {
		if kinds[want] == 0 {
			t.Fatalf("normalized stream lacks %s events: %v", want, kinds)
		}
	}
	if len(rec.Assertions) != 2 || len(rec.Started) != 2 || len(rec.Completed) != 2 {
		t.Fatalf("reconciled inventory wrong: %+v", rec)
	}
}

// TestPythonReconciliationViolationMatrix freezes the negative automaton:
// every violation of the closed lifecycle, witness binding or accounting is
// rejected with its machine diagnostic.
func TestPythonReconciliationViolationMatrix(t *testing.T) {
	replace := func(old, new string) string { return strings.Replace(pySampleReport, old, new, 1) }
	sentinel := `{"kind": "bootstrap-end", "schema": "machinery.tdd.harness.python/v1"}` + "\n"
	cases := []struct {
		name   string
		stream string
		code   string
	}{
		{"skip terminal", replace(`{"exception": "AssertionError", "kind": "test-end", "outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}`, `{"kind": "test-end", "outcome": "skipped", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}`), "UNSUPPORTED_FEATURE"},
		{"expected failure", replace(`"outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"`, `"outcome": "expected-failure", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"`), "UNSUPPORTED_FEATURE"},
		{"unexpected success", replace(`"outcome": "pass", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"`, `"outcome": "unexpected-success", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"`), "UNEXPECTED_FAILURE"},
		{"error terminal", replace(`{"exception": "AssertionError", "kind": "test-end", "outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}`, `{"exception": "ValueError", "kind": "test-end", "outcome": "error", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"}`), "UNEXPECTED_FAILURE"},
		{"foreign exception class", replace(`"exception": "AssertionError", "kind": "test-end", "outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"`, `"exception": "RuntimeError", "kind": "test-end", "outcome": "assertion-fail", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_fail"`), "UNEXPECTED_FAILURE"},
		{"unwitnessed failure", strings.ReplaceAll(pySampleReport, `{"assertion": "unit/beta", "condition": false, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:9", "test": "unit_test.UnitTests.test_fail", "thrown": "AssertionError"}`+"\n", ""), "UNEXPECTED_FAILURE"},
		{"false witness on passing test", replace(`"assertion": "unit/alpha", "condition": true`, `"assertion": "unit/alpha", "condition": false`), "ASSERTION_MISMATCH"},
		{"witness site drift", replace(`"site": "unit_test.py:9"`, `"site": "unit_test.py:8"`), "ASSERTION_MISMATCH"},
		{"unknown witness assertion", replace(`"assertion": "unit/beta"`, `"assertion": "unit/gamma"`), "ASSERTION_MISMATCH"},
		{"duplicate witness", strings.Replace(pySampleReport, sentinel, `{"assertion": "unit/beta", "condition": false, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:9", "test": "unit_test.UnitTests.test_fail", "thrown": "AssertionError"}`+"\n"+sentinel, 1), "ASSERTION_MISMATCH"},
		{"witness outside lifecycle", strings.Replace(pySampleReport, sentinel, `{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "unit_test.UnitTests.test_pass", "thrown": null}`+"\n"+sentinel, 1), "INVALID_SCHEMA"},
		{"subtest record", replace(`{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, `{"feature": "subTest", "kind": "unsupported-feature", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`), "UNSUPPORTED_FEATURE"},
		{"harness suite error", replace(`{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`, `{"code": "DUPLICATE_TERMINAL", "kind": "suite-error", "message": "boom", "schema": "machinery.tdd.harness.python/v1"}`), "DUPLICATE_TERMINAL"},
		{"summary count drift", replace(`"failures": 1, "kind": "suite-end"`, `"failures": 0, "kind": "suite-end"`), "INCOMPLETE_EVENTS"},
		{"missing declared identity", strings.ReplaceAll(pySampleReport, `{"kind": "test-start", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`+"\n"+`{"assertion": "unit/alpha", "condition": true, "kind": "witness", "schema": "machinery.tdd.witness/v1", "site": "unit_test.py:6", "test": "unit_test.UnitTests.test_pass", "thrown": null}`+"\n"+`{"kind": "test-end", "outcome": "pass", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`+"\n", ""), "MISSING_TEST"},
		{"undeclared identity", replace(`"tests": ["unit_test.UnitTests.test_fail", "unit_test.UnitTests.test_pass"]`, `"tests": ["unit_test.UnitTests.test_fail", "unit_test.UnitTests.test_pass", "unit_test.UnitTests.test_extra"]`), "DUPLICATE_TEST"},
		{"duplicate terminal", strings.Replace(pySampleReport, sentinel, `{"kind": "test-end", "outcome": "pass", "schema": "machinery.tdd.harness.python/v1", "test": "unit_test.UnitTests.test_pass"}`+"\n"+sentinel, 1), "INVALID_SCHEMA"},
	}
	for _, c := range cases {
		records, err := parsePythonReport([]byte(c.stream))
		if err != nil {
			t.Fatalf("%s: stream must parse before reconciliation: %v", c.name, err)
		}
		if _, err := reconcilePythonReport(records, pyUnitSuite()); err == nil || !strings.Contains(err.Error(), c.code) {
			t.Fatalf("%s: want %s, got %v", c.name, c.code, err)
		}
	}
}

// TestBuildPythonEnv freezes the closed environment: private roots, UTC,
// deterministic hash seed, bytecode disabled, no PYTHON*/MACHINERY*
// inheritance and no closed-key or duplicate overrides.
func TestBuildPythonEnv(t *testing.T) {
	env, err := buildPythonEnv("/scratch", []tdd.EnvironmentVar{{Name: "SUITE_FLAG", Value: "1"}})
	if err != nil {
		t.Fatalf("closed environment must build: %v", err)
	}
	joined := strings.Join(env, "\n")
	for _, required := range []string{"HOME=/scratch/home", "TMPDIR=/scratch/tmp", "TZ=UTC", "PYTHONHASHSEED=0", "PYTHONDONTWRITEBYTECODE=1", "SUITE_FLAG=1"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("closed environment lacks %s: %v", required, env)
		}
	}
	for _, forbidden := range []string{"PYTHONPATH=", "PYTHONHOME=", "PYTHONSTARTUP=", "MACHINERY_"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("closed environment must never inherit %s", forbidden)
		}
	}
	for _, closed := range []tdd.EnvironmentVar{
		{Name: "PYTHONPATH", Value: "/evil"}, {Name: "PYTHONHASHSEED", Value: "1"},
		{Name: "MACHINERY_PROCESSSCOPE_CAP", Value: "x"}, {Name: "PATH", Value: "/evil"},
		{Name: "HOME", Value: "/evil"},
	} {
		if _, err := buildPythonEnv("/scratch", []tdd.EnvironmentVar{closed}); err == nil {
			t.Fatalf("declared environment may not override the closed key %s", closed.Name)
		}
	}
	if _, err := buildPythonEnv("/scratch", []tdd.EnvironmentVar{{Name: "A", Value: "1"}, {Name: "A", Value: "2"}}); err == nil {
		t.Fatal("duplicate declared environment names must fail")
	}
}

// TestPythonNativeIdentityRendering freezes the exact TestCase.id() spelling
// of python native identities.
func TestPythonNativeIdentityRendering(t *testing.T) {
	if got := pythonNativeID(tdd.NativeID{Module: "conformance_test", Class: "ConformanceWitness", Method: "test_witness_pass"}); got != "conformance_test.ConformanceWitness.test_witness_pass" {
		t.Fatalf("native identity rendering %q is not the exact TestCase.id() spelling", got)
	}
}
