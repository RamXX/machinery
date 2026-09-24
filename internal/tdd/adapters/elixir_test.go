package adapters

// RED contract for MAC-8yai (unit subjects): the closed elixir-exunit/v1
// adapter surfaces. The frozen asset pins are a passing control; every
// parse/reconcile/validation subject fails on the RED stub with the
// pending-implementation error and defines the closed behavior, including
// the empirically pinned ExUnit shapes (passed=nil state, AssertionError
// causality, describe-prefixed native names, effective-option reporting).

import (
	"os"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
)

func exSHA(b []byte) string { return tsDigestHex(b) }

func exConformanceFixtureBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("assets/elixir/conformance/conformance_test.exs")
	if err != nil {
		t.Fatal(err)
	}
	if got := exSHA(b); got != ElixirConformanceFixtureSHA256 {
		t.Fatalf("frozen conformance fixture bytes changed: sha256 %s", got)
	}
	return b
}

// TestElixirAssetInventoryMatchesFrozenPins is a passing control: every
// embedded asset byte set is present and equals its frozen pin, the
// embedded transport is the frozen machinery-check/v1 helper, and the
// embedded reporter emits the terminal sentinel.
func TestElixirAssetInventoryMatchesFrozenPins(t *testing.T) {
	files := map[string][]byte{
		"mix.exs":                elixirMixSource,
		"machinery/json.exs":     elixirJSONSource,
		"machinery/reporter.exs": elixirReporterSource,
		"machinery/check.exs":    elixirCheckSource,
		"test/test_helper.exs":   elixirBootstrapSource,
	}
	want := ElixirAssets()
	if len(want) != len(files) {
		t.Fatalf("frozen asset inventory has %d entries, embedded set has %d", len(want), len(files))
	}
	for name, body := range files {
		pin, ok := want[name]
		if !ok {
			t.Fatalf("embedded asset %q is absent from the frozen inventory", name)
		}
		if got := exSHA(body); got != pin {
			t.Fatalf("embedded asset %s sha256 %s does not match its frozen pin %s", name, got, pin)
		}
	}
	if got := exSHA(ElixirHelperSource()); got != ElixirCheckPinnedSHA256 {
		t.Fatalf("helper copy sha256 %s does not match the frozen pin", got)
	}
	if !strings.Contains(string(elixirCheckSource), "machinery-check/v1") {
		t.Fatal("helper transport does not carry the machinery-check/v1 identity")
	}
	if !strings.Contains(string(elixirReporterSource), ElixirReporterSentinel) {
		t.Fatal("embedded reporter does not emit the terminal sentinel")
	}
	if !strings.Contains(string(elixirBootstrapSource), "Machinery.Assurance.Reporter") {
		t.Fatal("embedded bootstrap does not install the closed formatter")
	}
}

// TestElixirReporterSurvivesItsOwnSuiteFinished freezes the deterministic
// teardown shape of the embedded formatter (MAC-k3rb). ExUnit.Runner calls
// ExUnit.EventManager.stop/1 immediately after the suite_finished cast, and
// that function calls GenServer.stop/3 on every child the formatter
// DynamicSupervisor still lists. A formatter that terminates itself inside
// that cast races the supervisor's exit-signal handling: when the child is
// still listed but already dead, the teardown exits
// {:noproc, {GenServer, :stop, [pid, :normal, :infinity]}} and kills the
// runner task after a fully executed suite. The reporter therefore closes
// its stream in that cast and stays alive, so ExUnit stops it exactly once.
func TestElixirReporterSurvivesItsOwnSuiteFinished(t *testing.T) {
	source := string(elixirReporterSource)
	clause := strings.Index(source, "def handle_cast({:suite_finished, _times_us}, state) do")
	if clause < 0 {
		t.Fatal("embedded reporter no longer declares the suite_finished formatter clause")
	}
	next := strings.Index(source[clause:], "\n  def ")
	if next < 0 {
		t.Fatal("embedded reporter suite_finished clause is unterminated")
	}
	body := source[clause : clause+next]
	if strings.Contains(body, "{:stop,") {
		t.Fatal("embedded reporter stops itself on suite_finished; that races ExUnit.EventManager.stop/1 into a noproc teardown")
	}
	if !strings.Contains(body, "{:noreply, close(state)}") {
		t.Fatalf("embedded reporter suite_finished clause must close the stream and stay alive: %s", body)
	}
	if !strings.Contains(source, "def terminate(_reason, state), do: close(state)") {
		t.Fatal("embedded reporter must still close its stream from terminate/2")
	}
}

// TestElixirLookupResolvesClosedIdentity freezes the registry seam: the
// closed adapter identity resolves through the shared Lookup and foreign
// identities fail closed.
func TestElixirLookupResolvesClosedIdentity(t *testing.T) {
	adapter, err := Lookup(AdapterElixirExunit)
	if err != nil || adapter.ID() != AdapterElixirExunit {
		t.Fatalf("closed identity did not resolve: %v %+v", err, adapter)
	}
	if _, err := Lookup("exunit-interop/v1"); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_ADAPTER") {
		t.Fatalf("foreign adapter identity must fail closed: %v", err)
	}
}

// exRealStreamLines is a frozen sample of REAL embedded-reporter output
// (Elixir 1.20.4 / OTP 29.0.6 under the frozen harness argv): the effective
// suite_started options, module lifecycle, the describe-prefixed native
// names, helper witnesses with the ok verdict, passed tests with the
// empirically pinned null native state, the summary and the terminal
// sentinel.
const exRealStreamLines = `{"t":"suite_started","d":{"seed":7,"max_cases":2,"trace":false,"timeout":60000,"max_failures":"infinity","include":[],"exclude":[],"formatters":["Elixir.Machinery.Assurance.Reporter"],"dry_run":false,"repeat_until_failure":0}}
{"t":"module_started","d":{"module":"Elixir.Machinery.ConformanceTest"}}
{"t":"test_started","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance multiple assertions","file":"test/conformance_test.exs","line":17,"async":true}}
{"t":"witness","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance multiple assertions","id":"conformance/multi-a","value":true,"file":"test/conformance_test.exs","line":18,"verdict":"ok"}}
{"t":"witness","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance multiple assertions","id":"conformance/multi-b","value":true,"file":"test/conformance_test.exs","line":19,"verdict":"ok"}}
{"t":"test_finished","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance multiple assertions","file":"test/conformance_test.exs","line":17,"state":"passed","class":"","message":""}}
{"t":"test_started","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"conformance parent identity","test":"test conformance parent identity conformance nested identity","file":"test/conformance_test.exs","line":12,"async":true}}
{"t":"witness","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"conformance parent identity","test":"test conformance parent identity conformance nested identity","id":"conformance/nested","value":true,"file":"test/conformance_test.exs","line":13,"verdict":"ok"}}
{"t":"test_finished","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"conformance parent identity","test":"test conformance parent identity conformance nested identity","file":"test/conformance_test.exs","line":12,"state":"passed","class":"","message":""}}
{"t":"test_started","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance witness executes native assertion","file":"test/conformance_test.exs","line":7,"async":true}}
{"t":"witness","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance witness executes native assertion","id":"conformance/witness","value":true,"file":"test/conformance_test.exs","line":8,"verdict":"ok"}}
{"t":"test_finished","d":{"module":"Elixir.Machinery.ConformanceTest","describe":"","test":"test conformance witness executes native assertion","file":"test/conformance_test.exs","line":7,"state":"passed","class":"","message":""}}
{"t":"module_finished","d":{"module":"Elixir.Machinery.ConformanceTest","state":"finished"}}
{"t":"suite_finished","d":{"tests":3,"failures":0,"skipped":0,"invalid":0,"excluded":0}}
{"t":"machinery:reporter:end","d":{}}
`

// TestParseElixirReporterStreamAcceptsRealVocabulary freezes the closed
// parser over the frozen real sample: every event decodes with its exact
// identity (module atoms de-Elixirified, native names de-"test"-prefixed),
// the effective options decode with their closed shapes, and the terminal
// sentinel is present exactly once at the end.
func TestParseElixirReporterStreamAcceptsRealVocabulary(t *testing.T) {
	events, err := parseElixirReporterStream([]byte(exRealStreamLines))
	if err != nil {
		t.Fatalf("real reporter vocabulary rejected: %v", err)
	}
	if len(events) != 15 {
		t.Fatalf("expected 15 decoded events, got %d", len(events))
	}
	if events[0].Type != "suite_started" || events[0].Config == nil {
		t.Fatalf("suite_started did not decode: %+v", events[0])
	}
	cfg := events[0].Config
	if cfg.Seed != 7 || cfg.MaxCases != 2 || cfg.Trace || cfg.Timeout != 60000 ||
		cfg.MaxFailures != "infinity" || cfg.DryRun || cfg.RepeatUntilFailure != 0 ||
		len(cfg.Include) != 0 || len(cfg.Exclude) != 0 || len(cfg.Formatters) != 1 || cfg.Formatters[0] != ElixirReporterModule {
		t.Fatalf("effective options did not decode: %+v", cfg)
	}
	if events[1].Type != "module_started" || events[1].Module != "Machinery.ConformanceTest" {
		t.Fatalf("module_started did not decode: %+v", events[1])
	}
	started := events[2]
	if started.Type != "test_started" || started.Test == nil || started.Test.Module != "Machinery.ConformanceTest" ||
		started.Test.Name != "conformance multiple assertions" || started.Test.File != "test/conformance_test.exs" ||
		started.Test.Line != 17 || !started.Test.Async {
		t.Fatalf("test_started did not decode: %+v", started.Test)
	}
	nested := events[6]
	if nested.Test == nil || nested.Test.Describe != "conformance parent identity" ||
		nested.Test.Name != "conformance parent identity conformance nested identity" {
		t.Fatalf("describe-prefixed native name did not decode: %+v", nested.Test)
	}
	witness := events[3]
	if witness.Type != "witness" || witness.Witness == nil || witness.Witness.ID != "conformance/multi-a" ||
		!witness.Witness.Value || witness.Witness.Verdict != "ok" || witness.Witness.Line != 18 ||
		witness.Witness.Name != "conformance multiple assertions" {
		t.Fatalf("witness did not decode: %+v", witness.Witness)
	}
	finished := events[5]
	if finished.State != "passed" || finished.Class != "" || finished.Test == nil {
		t.Fatalf("passed test_finished (null native state) did not decode: %+v", finished)
	}
	if events[12].Type != "module_finished" || events[12].ModuleDone != "finished" {
		t.Fatalf("module_finished did not decode: %+v", events[12])
	}
	if events[13].Type != "suite_finished" {
		t.Fatalf("suite_finished missing: %+v", events[13])
	}
	if events[14].Type != ElixirReporterSentinel {
		t.Fatalf("terminal sentinel missing: %+v", events[14])
	}
}

// TestParseElixirReporterStreamRejectsForgery freezes the closed-parser
// negative surface: unknown event identities, unknown payload keys,
// malformed lines, missing, duplicated or non-terminal sentinels and
// trailing junk all fail closed.
func TestParseElixirReporterStreamRejectsForgery(t *testing.T) {
	base := strings.TrimSuffix(exRealStreamLines, "\n")
	cases := []struct {
		name  string
		lines []string
	}{
		{"unknown-identity", []string{`{"t":"exunit:machinery","d":{"module":"M"}}`}},
		{"malformed-json", []string{`{"t":"suite_started","d":`}},
		{"unknown-key", []string{`{"t":"module_started","d":{"module":"M","extra":1}}`}},
		{"missing-sentinel", strings.Split(strings.ReplaceAll(base, `{"t":"machinery:reporter:end","d":{}}`, ""), "\n")},
		{"duplicated-sentinel", append(strings.Split(base, "\n"), `{"t":"machinery:reporter:end","d":{}}`)},
		{"junk-after-sentinel", append(strings.Split(base, "\n"), `forged native line`)},
		{"empty-stream", nil},
		{"bad-config-shape", []string{`{"t":"suite_started","d":{"seed":"one","max_cases":2,"trace":false,"timeout":60000,"max_failures":"infinity","include":[],"exclude":[],"formatters":["Elixir.Machinery.Assurance.Reporter"],"dry_run":false,"repeat_until_failure":0}}`, `{"t":"machinery:reporter:end","d":{}}`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(strings.Join(tc.lines, "\n"))
			if _, err := parseElixirReporterStream(raw); err == nil {
				t.Fatal("forged reporter stream accepted")
			}
		})
	}
}

// exUnitSuite is the frozen unit inventory: one module, one plain case with
// a single registered assertion.
func exUnitSuite() *tdd.Suite {
	return &tdd.Suite{
		ID: "unit-ex", Adapter: AdapterElixirExunit, Root: ".", Files: []string{"test/unit_test.exs"},
		Tests: []tdd.Test{
			{ID: "alpha", Source: "test/unit_test.exs", Native: tdd.NativeID{Module: "Unit.CaseTest", Name: "alpha case", File: "test/unit_test.exs", Line: 5},
				Assertions: []tdd.Assertion{{ID: "unit/alpha", Source: "test/unit_test.exs", Line: 6, Helper: tdd.AssertionHelperV1}}},
		},
	}
}

func exEvent(kind string, mutate func(*elixirEvent)) elixirEvent {
	e := elixirEvent{Type: kind}
	if mutate != nil {
		mutate(&e)
	}
	return e
}

func exSummaryEvent(tests, failures, skipped, invalid, excluded int64) elixirEvent {
	return exEvent("suite_finished", func(e *elixirEvent) {
		e.Summary = &elixirSummary{Tests: tests, Failures: failures, Skipped: skipped, Invalid: invalid, Excluded: excluded}
	})
}

func exConfigEvent(mutate func(*elixirEffectiveConfig)) elixirEvent {
	return exEvent("suite_started", func(e *elixirEvent) {
		cfg := elixirExpectedConfig()
		if mutate != nil {
			mutate(&cfg)
		}
		e.Config = &cfg
	})
}

func exTestIdentity() *elixirTestIdentity {
	return &elixirTestIdentity{Module: "Unit.CaseTest", Describe: "", Name: "alpha case", File: "test/unit_test.exs", Line: 5, Async: true}
}

func exWitnessEvent(value bool, mutate func(*elixirWitness)) elixirEvent {
	return exEvent("witness", func(e *elixirEvent) {
		w := &elixirWitness{Module: "Unit.CaseTest", Describe: "", Name: "alpha case", ID: "unit/alpha",
			File: "test/unit_test.exs", Line: 6, Value: value, Verdict: "ok"}
		if mutate != nil {
			mutate(w)
		}
		e.Witness = w
	})
}

// exLegalPassStream builds the legal pass automaton in native order:
// suite_started, module lifecycle, one complete test with its witness, the
// exact summary and the terminal sentinel.
func exLegalPassStream() []elixirEvent {
	return []elixirEvent{
		exConfigEvent(nil),
		exEvent("module_started", func(e *elixirEvent) { e.Module = "Unit.CaseTest" }),
		exEvent("test_started", func(e *elixirEvent) { e.Test = exTestIdentity() }),
		exWitnessEvent(true, nil),
		exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "passed" }),
		exEvent("module_finished", func(e *elixirEvent) { e.Module = "Unit.CaseTest"; e.ModuleDone = "finished" }),
		exSummaryEvent(1, 0, 0, 0, 0),
		exEvent(ElixirReporterSentinel, nil),
	}
}

// TestReconcileElixirStreamLegalPass freezes the normalized event automaton:
// the legal pass sequence produces the exact closed machinery.tdd.event/v1
// order with owner-assigned identity and a passing assertion outcome bound
// to the registered site.
func TestReconcileElixirStreamLegalPass(t *testing.T) {
	rec, err := reconcileElixirStream(exLegalPassStream(), exUnitSuite(), elixirExpectedConfig())
	if err != nil {
		t.Fatalf("legal pass stream rejected: %v", err)
	}
	if rec.Outcome != "pass" {
		t.Fatalf("outcome %q is not pass", rec.Outcome)
	}
	kinds := make([]string, 0, len(rec.Events))
	for _, e := range rec.Events {
		kinds = append(kinds, e.Kind)
		if e.Schema != ElixirEventSchema || e.Suite != "unit-ex" || e.Sequence <= 0 {
			t.Fatalf("normalized event carries a wrong identity: %+v", e)
		}
		if e.Design != "" || e.Milestone != "" {
			t.Fatalf("adapter invented owner identity it does not own: %+v", e)
		}
	}
	want := []string{"suite-start", "discovered", "test-start", "assertion", "test-end", "suite-end"}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("normalized kinds %v, want %v", kinds, want)
	}
	if len(rec.Assertions) != 1 || rec.Assertions[0].ID != "unit/alpha" || rec.Assertions[0].Outcome != "pass" {
		t.Fatalf("assertion not accounted: %+v", rec.Assertions)
	}
	if len(rec.Started) != 1 || len(rec.Completed) != 1 || len(rec.Discovered) != 1 {
		t.Fatalf("inventory not accounted: %+v", rec)
	}
	if rec.Started[0].Module != "Unit.CaseTest" || rec.Started[0].Name != "alpha case" || rec.Started[0].File != "test/unit_test.exs" || rec.Started[0].Line != 5 {
		t.Fatalf("native identity lost: %+v", rec.Started)
	}
}

// TestReconcileElixirStreamAssertionFailureBindsRegisteredSite freezes the
// RED evidence path: a native failure is an assertion-fail only when the
// helper witness records a false strictly-boolean condition with the ok
// verdict and the native class is exactly ExUnit.AssertionError at the
// registered frozen site.
func TestReconcileElixirStreamAssertionFailureBindsRegisteredSite(t *testing.T) {
	stream := []elixirEvent{
		exConfigEvent(nil),
		exEvent("module_started", func(e *elixirEvent) { e.Module = "Unit.CaseTest" }),
		exEvent("test_started", func(e *elixirEvent) { e.Test = exTestIdentity() }),
		exWitnessEvent(false, nil),
		exEvent("test_finished", func(e *elixirEvent) {
			e.Test = exTestIdentity()
			e.State = "failed"
			e.Class = "ExUnit.AssertionError"
			e.Message = "machinery-check/v1 assertion unit/alpha evaluated false at test/unit_test.exs:6"
		}),
		exEvent("module_finished", func(e *elixirEvent) { e.Module = "Unit.CaseTest"; e.ModuleDone = "finished" }),
		exSummaryEvent(1, 1, 0, 0, 0),
		exEvent(ElixirReporterSentinel, nil),
	}
	rec, err := reconcileElixirStream(stream, exUnitSuite(), elixirExpectedConfig())
	if err != nil {
		t.Fatalf("assertion-fail stream rejected: %v", err)
	}
	if rec.Outcome != "assertion-fail" || len(rec.Failed) != 1 {
		t.Fatalf("outcome %q failed=%d is not the accounted assertion fail", rec.Outcome, len(rec.Failed))
	}
	if len(rec.Assertions) != 1 || rec.Assertions[0].Outcome != "assertion-fail" || rec.Assertions[0].ID != "unit/alpha" {
		t.Fatalf("failing assertion not bound: %+v", rec.Assertions)
	}
	for _, e := range rec.Events {
		if e.Kind == "test-end" && e.Outcome != "assertion-fail" {
			t.Fatalf("failed test end outcome %q is not assertion-fail", e.Outcome)
		}
	}
}

// TestReconcileElixirStreamRejectsViolations freezes the reconciliation
// negative surface: every violation fails closed with its closed diagnostic
// identity and never converts into success.
func TestReconcileElixirStreamRejectsViolations(t *testing.T) {
	drop := func(kind string) func([]elixirEvent) []elixirEvent {
		return func(s []elixirEvent) []elixirEvent {
			out := s[:0:0]
			for _, e := range s {
				if e.Type != kind {
					out = append(out, e)
				}
			}
			return out
		}
	}
	replace := func(kind string, with elixirEvent) func([]elixirEvent) []elixirEvent {
		return func(s []elixirEvent) []elixirEvent {
			out := s[:0:0]
			for _, e := range s {
				if e.Type == kind {
					out = append(out, with)
				} else {
					out = append(out, e)
				}
			}
			return out
		}
	}
	// Streams stay legal at the framing level: extra violation events are
	// inserted before the terminal sentinel.
	insert := func(s []elixirEvent, extra elixirEvent) []elixirEvent {
		out := append([]elixirEvent{}, s[:len(s)-1]...)
		return append(append(out, extra), s[len(s)-1])
	}
	failedFinished := exEvent("test_finished", func(e *elixirEvent) {
		e.Test = exTestIdentity()
		e.State = "failed"
		e.Class = "ExUnit.AssertionError"
	})
	skippedFinished := exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "skipped" })
	excludedFinished := exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "excluded" })
	invalidFinished := exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "invalid" })
	raiseFinished := exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "failed"; e.Class = "RuntimeError" })
	timeoutFinished := exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "failed"; e.Class = "ExUnit.TimeoutError" })
	wrongSummary := exSummaryEvent(1, 1, 0, 0, 0)
	lyingSummary := exSummaryEvent(1, 0, 0, 0, 0)
	unknownNative := exEvent("test_started", func(e *elixirEvent) {
		e.Test = &elixirTestIdentity{Module: "Unit.CaseTest", Name: "unregistered case", File: "test/unit_test.exs", Line: 40}
	})
	lateWitness := exWitnessEvent(true, nil)
	wrongSite := exWitnessEvent(true, func(w *elixirWitness) { w.Line = 9 })
	wrongFile := exWitnessEvent(true, func(w *elixirWitness) { w.File = "test/other_test.exs" })
	pidMismatch := exWitnessEvent(true, func(w *elixirWitness) { w.Verdict = "pid-mismatch" })
	malformedWitness := exWitnessEvent(true, func(w *elixirWitness) { w.Verdict = "malformed" })
	unknownAssertion := exWitnessEvent(true, func(w *elixirWitness) { w.ID = "unit/ghost" })
	wrongDescribe := exWitnessEvent(true, func(w *elixirWitness) { w.Describe = "other describe" })
	invalidModuleDone := exEvent("module_finished", func(e *elixirEvent) { e.Module = "Unit.CaseTest"; e.ModuleDone = "invalid" })
	maxFailures := exEvent("max_failures_reached", nil)
	cases := []struct {
		name string
		mut  func([]elixirEvent) []elixirEvent
		want string
	}{
		{"missing-suite-started", drop("suite_started"), "INCOMPLETE_EVENTS"},
		{"missing-module-start", drop("module_started"), "INCOMPLETE_EVENTS"},
		{"missing-test-start", drop("test_started"), "ASSERTION_MISMATCH"},
		{"missing-test-finish", drop("test_finished"), "INCOMPLETE_EVENTS"},
		{"missing-summary", drop("suite_finished"), "INCOMPLETE_EVENTS"},
		{"missing-sentinel", drop(ElixirReporterSentinel), "INCOMPLETE_EVENTS"},
		{"unknown-native-identity", func(s []elixirEvent) []elixirEvent { return insert(s, unknownNative) }, "MISSING_TEST"},
		{"skipped-state", replace("test_finished", skippedFinished), "UNSUPPORTED_FEATURE"},
		{"excluded-state", replace("test_finished", excludedFinished), "UNSUPPORTED_FEATURE"},
		{"invalid-state", replace("test_finished", invalidFinished), "UNEXPECTED_FAILURE"},
		{"plain-raise", replace("test_finished", raiseFinished), "UNEXPECTED_FAILURE"},
		{"timeout-is-error", replace("test_finished", timeoutFinished), "UNEXPECTED_FAILURE"},
		{"unwitnessed-assertion-failure", func(s []elixirEvent) []elixirEvent {
			return replace("test_finished", failedFinished)(drop("witness")(s))
		}, "UNEXPECTED_FAILURE"},
		{"summary-disagrees", replace("suite_finished", wrongSummary), "INCOMPLETE_EVENTS"},
		{"summary-lies", func(s []elixirEvent) []elixirEvent {
			return replace("suite_finished", lyingSummary)(replace("test_finished", failedFinished)(replace("witness", exWitnessEvent(false, nil))(s)))
		}, "INCOMPLETE_EVENTS"},
		{"late-witness", func(s []elixirEvent) []elixirEvent {
			idx := -1
			for i, e := range s {
				if e.Type == "test_finished" {
					idx = i
				}
			}
			out := append([]elixirEvent{}, s...)
			return append(append(out[:idx+1:idx+1], lateWitness), out[idx+1:]...)
		}, "ASSERTION_MISMATCH"},
		{"witness-site-mismatch", replace("witness", wrongSite), "ASSERTION_MISMATCH"},
		{"witness-file-mismatch", replace("witness", wrongFile), "ASSERTION_MISMATCH"},
		{"witness-pid-mismatch", replace("witness", pidMismatch), "ASSERTION_MISMATCH"},
		{"witness-malformed", replace("witness", malformedWitness), "ASSERTION_MISMATCH"},
		{"witness-describe-mismatch", replace("witness", wrongDescribe), "ASSERTION_MISMATCH"},
		{"unknown-witness-assertion", func(s []elixirEvent) []elixirEvent { return insert(s, unknownAssertion) }, "ASSERTION_MISMATCH"},
		{"invalid-module-state", replace("module_finished", invalidModuleDone), "UNEXPECTED_FAILURE"},
		{"max-failures-truncation", func(s []elixirEvent) []elixirEvent { return insert(s, maxFailures) }, "INCOMPLETE_EVENTS"},
		{"config-seed-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.Seed = 99 })), "INVALID_SCHEMA"},
		{"config-max-cases-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.MaxCases = 8 })), "INVALID_SCHEMA"},
		{"config-trace-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.Trace = true })), "INVALID_SCHEMA"},
		{"config-exclude-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.Exclude = []string{"wip"} })), "UNSUPPORTED_FEATURE"},
		{"config-include-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.Include = []string{"wip"} })), "UNSUPPORTED_FEATURE"},
		{"config-formatter-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.Formatters = []string{"ExUnit.CLIFormatter"} })), "UNSUPPORTED_FEATURE"},
		{"config-max-failures-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.MaxFailures = "1" })), "UNSUPPORTED_FEATURE"},
		{"config-dry-run-drift", replace("suite_started", exConfigEvent(func(c *elixirEffectiveConfig) { c.DryRun = true })), "UNSUPPORTED_FEATURE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := reconcileElixirStream(tc.mut(exLegalPassStream()), exUnitSuite(), elixirExpectedConfig())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("violation %s must fail with %s: err=%v rec=%+v", tc.name, tc.want, err, rec)
			}
		})
	}
}

// TestReconcileElixirStreamRejectsDuplicateExecution freezes the
// exactly-once rule: a second complete lifecycle for the same declared
// identity (the shuffled concurrent identity collision) never certifies.
func TestReconcileElixirStreamRejectsDuplicateExecution(t *testing.T) {
	base := exLegalPassStream()
	duplicate := append([]elixirEvent{}, base[:len(base)-1]...)
	duplicate = append(duplicate,
		exEvent("test_started", func(e *elixirEvent) { e.Test = exTestIdentity() }),
		exWitnessEvent(true, nil),
		exEvent("test_finished", func(e *elixirEvent) { e.Test = exTestIdentity(); e.State = "passed" }),
		base[len(base)-1],
	)
	if _, err := reconcileElixirStream(duplicate, exUnitSuite(), elixirExpectedConfig()); err == nil || !strings.Contains(err.Error(), "DUPLICATE_TEST") {
		t.Fatalf("duplicate native execution must fail closed: %v", err)
	}
}

// TestReconcileElixirStreamPreservesOwnerIdentity freezes the unforgeable
// ownership rule: the normalized stream always carries the owner-assigned
// suite identity and never invents design/milestone owner fields.
func TestReconcileElixirStreamPreservesOwnerIdentity(t *testing.T) {
	rec, err := reconcileElixirStream(exLegalPassStream(), exUnitSuite(), elixirExpectedConfig())
	if err != nil {
		t.Fatalf("legal stream rejected: %v", err)
	}
	for _, e := range rec.Events {
		if e.Suite != "unit-ex" || e.Design != "" || e.Milestone != "" || e.Sequence > int64(len(rec.Events)) {
			t.Fatalf("native payload reassigned owner identity: %+v", e)
		}
	}
}

// TestValidateElixirSuite freezes the static source binding: elixir-native
// identities bound to declared test files, the frozen ExUnit.Case grammar,
// typed check call sites at the exact frozen lines, comment-derived
// evidence rejected, and duplicate rejection.
func TestValidateElixirSuite(t *testing.T) {
	frozen := map[string][]byte{"test/conformance_test.exs": exConformanceFixtureBytes(t)}
	suite := &tdd.Suite{
		ID: "unit-ex", Adapter: AdapterElixirExunit, Root: ".", Files: []string{"test/conformance_test.exs"},
		Tests: []tdd.Test{{
			ID: "witness", Source: "test/conformance_test.exs",
			Native:     tdd.NativeID{Module: "Machinery.ConformanceTest", Name: "conformance witness executes native assertion", File: "test/conformance_test.exs", Line: 7},
			Assertions: []tdd.Assertion{{ID: "conformance/witness", Source: "test/conformance_test.exs", Line: 8, Helper: tdd.AssertionHelperV1}},
		}},
	}
	if err := validateElixirSuite(suite, frozen); err != nil {
		t.Fatalf("valid frozen suite rejected: %v", err)
	}
	badHelper := *suite
	badHelper.Tests = []tdd.Test{suite.Tests[0]}
	badHelper.Tests[0].Assertions = []tdd.Assertion{{ID: "a", Source: "test/conformance_test.exs", Line: 8, Helper: "exunit-interop/v1"}}
	if err := validateElixirSuite(&badHelper, frozen); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_FEATURE") {
		t.Fatalf("foreign helper accepted: %v", err)
	}
	badLine := *suite
	badLine.Tests = []tdd.Test{suite.Tests[0]}
	badLine.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/witness", Source: "test/conformance_test.exs", Line: 10, Helper: tdd.AssertionHelperV1}}
	if err := validateElixirSuite(&badLine, frozen); err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("assertion line without the typed call accepted: %v", err)
	}
	commented := *suite
	commented.Tests = []tdd.Test{suite.Tests[0]}
	commented.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/ghost", Source: "test/conformance_test.exs", Line: 5, Helper: tdd.AssertionHelperV1}}
	if err := validateElixirSuite(&commented, frozen); err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("non-call line accepted: %v", err)
	}
	foreignSource := *suite
	foreignSource.Tests = []tdd.Test{suite.Tests[0]}
	foreignSource.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/witness", Source: "test/other_test.exs", Line: 8, Helper: tdd.AssertionHelperV1}}
	if err := validateElixirSuite(&foreignSource, frozen); err == nil {
		t.Fatal("assertion bound to a foreign source accepted")
	}
	foreignNative := *suite
	foreignNative.Tests = []tdd.Test{suite.Tests[0]}
	foreignNative.Tests[0].Native = tdd.NativeID{Package: "pkg", Test: "TestX"}
	if err := validateElixirSuite(&foreignNative, frozen); err == nil {
		t.Fatal("non-elixir native identity accepted")
	}
	testPrefixed := *suite
	testPrefixed.Tests = []tdd.Test{suite.Tests[0]}
	testPrefixed.Tests[0].Native = tdd.NativeID{Module: "Machinery.ConformanceTest", Name: "test conformance witness executes native assertion", File: "test/conformance_test.exs", Line: 7}
	if err := validateElixirSuite(&testPrefixed, frozen); err == nil {
		t.Fatal("native name carrying the test-prefix accepted")
	}
	unknownFile := *suite
	unknownFile.Tests = []tdd.Test{suite.Tests[0]}
	unknownFile.Tests[0].Native.File = "test/ghost_test.exs"
	if err := validateElixirSuite(&unknownFile, frozen); err == nil {
		t.Fatal("identity bound to an undeclared file accepted")
	}
	dup := *suite
	dup.Tests = append(append([]tdd.Test{}, suite.Tests...), suite.Tests[0])
	if err := validateElixirSuite(&dup, frozen); err == nil || !strings.Contains(err.Error(), "DUPLICATE_TEST") {
		t.Fatalf("duplicate native identity accepted: %v", err)
	}
	badGrammar := *suite
	badGrammar.Tests = []tdd.Test{suite.Tests[0]}
	badGrammar.Files = []string{"lib/unit.ex"}
	badGrammar.Tests[0].Source = "lib/unit.ex"
	badGrammar.Tests[0].Native.File = "lib/unit.ex"
	badGrammar.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/witness", Source: "lib/unit.ex", Line: 8, Helper: tdd.AssertionHelperV1}}
	if err := validateElixirSuite(&badGrammar, frozen); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_FEATURE") {
		t.Fatalf("file outside the frozen test/ grammar accepted: %v", err)
	}
}

// TestValidateElixirSuiteRejectsCommentedCallSite freezes the prior
// oracle-coverage defect guard: a commented-out helper call is never a
// registered assertion site; assertion evidence is execution-derived.
func TestValidateElixirSuiteRejectsCommentedCallSite(t *testing.T) {
	source := append([]byte{}, exConformanceFixtureBytes(t)...)
	source = append(source, []byte(`
  # Machinery.Check.check(ctx, "conformance/commented", true)
`)...)
	frozen := map[string][]byte{"test/conformance_test.exs": source}
	suite := &tdd.Suite{
		ID: "unit-ex", Adapter: AdapterElixirExunit, Root: ".", Files: []string{"test/conformance_test.exs"},
		Tests: []tdd.Test{{
			ID: "witness", Source: "test/conformance_test.exs",
			Native:     tdd.NativeID{Module: "Machinery.ConformanceTest", Name: "conformance witness executes native assertion", File: "test/conformance_test.exs", Line: 7},
			Assertions: []tdd.Assertion{{ID: "conformance/commented", Source: "test/conformance_test.exs", Line: 22, Helper: tdd.AssertionHelperV1}},
		}},
	}
	if err := validateElixirSuite(suite, frozen); err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
		t.Fatalf("commented-out call site accepted: %v", err)
	}
}

// TestBuildElixirRunEnv freezes the closed environment: the pinned runtime
// binaries first on PATH, private HOME/TMPDIR/MIX roots, the private events
// channel, fixed locale/timezone/no-color, declared suite variables only,
// and Mix/custody transport overrides never inherited.
func TestBuildElixirRunEnv(t *testing.T) {
	env, err := buildElixirRunEnv("/prep/run", "/opt/ex/bin", "/opt/erl/bin", "/prep/run/events.jsonl", []tdd.EnvironmentVar{
		{Name: "SUITE_FLAG", Value: "1"}, {Name: "MIX_ENV", Value: "prod"}, {Name: "MACHINERY_ASSURANCE_EVENTS", Value: "/evil"},
	})
	if err != nil {
		t.Fatalf("closed environment rejected: %v", err)
	}
	joined := "\x00" + strings.Join(env, "\x00") + "\x00"
	for _, want := range []string{
		"HOME=/prep/run/home", "TMPDIR=/prep/run/tmp", "TZ=UTC", "NO_COLOR=1",
		"PATH=/opt/ex/bin:/opt/erl/bin:/usr/bin:/bin", "MIX_ENV=test",
		"MIX_HOME=/prep/run/home/.mix", "MIX_BUILD_PATH=/prep/run/build",
		"MACHINERY_ASSURANCE_EVENTS=/prep/run/events.jsonl", "SUITE_FLAG=1",
	} {
		if !strings.Contains(joined, "\x00"+want+"\x00") {
			t.Fatalf("closed environment lacks %q: %v", want, env)
		}
	}
	if strings.Count(joined, "MIX_ENV=") != 1 || strings.Count(joined, "MACHINERY_ASSURANCE_EVENTS=") != 1 {
		t.Fatalf("declared environment overrode a closed key: %v", env)
	}
	for _, name := range []string{"MIX_DEPS_PATH", "MACHINERY_PROCESSSCOPE", "LANG"} {
		if strings.Contains(joined, name+"=") {
			t.Fatalf("injected variable %s inherited: %v", name, env)
		}
	}
}
