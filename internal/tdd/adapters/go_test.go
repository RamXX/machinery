package adapters

// Frozen RED unit suite for MAC-wi2u (go-testing/v1). Parser, transport,
// selection, environment and reconciliation subjects over synthetic closed
// inputs; the native execution proofs live in go_integration_test.go. On the
// RED stub every subject fails on the absent behavior; the byte-pin,
// fixture-freeze and lookup controls pass.

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
)

const unitModule = "unit.test/p"

func unitSuite() *tdd.Suite {
	return &tdd.Suite{
		ID:      "go-unit",
		Adapter: AdapterGoTesting,
		Runtime: tdd.RuntimeRef{Profile: "go", Version: "1.27.1", Platform: "darwin/arm64", Closure: "sha256:" + strings.Repeat("a", 64)},
		Root:    ".",
		Files:   []string{"a_test.go"},
		Tests: []tdd.Test{
			{
				ID: "pass", Role: "positive", Source: "a_test.go",
				Native:     tdd.NativeID{Package: unitModule, Test: "TestPass"},
				Assertions: []tdd.Assertion{{ID: "unit/alpha", Source: "a_test.go", Line: 10, Helper: "machinery-check/v1"}},
			},
			{
				ID: "fail", Role: "negative", Source: "a_test.go",
				Native:     tdd.NativeID{Package: unitModule, Test: "TestFail"},
				Assertions: []tdd.Assertion{{ID: "unit/beta", Source: "a_test.go", Line: 15, Helper: "machinery-check/v1"}},
			},
		},
	}
}

func digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func eventLine(action, pkg, test, output string) string {
	payload := map[string]any{"Action": action}
	if pkg != "" {
		payload["Package"] = pkg
	}
	if test != "" {
		payload["Test"] = test
	}
	if output != "" {
		payload["Output"] = output
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return string(body)
}

func witnessOutput(test, id, site string, line int64, value bool) string {
	return fmt.Sprintf("machinery-check/v1 witness id=%s value=%t test=%s site=%s:%d\n", id, value, test, site, line)
}

// TestLookupExposesOnlyClosedGoAdapter: the closed adapter resolves and every
// other identity fails UNSUPPORTED_ADAPTER.
func TestLookupExposesOnlyClosedGoAdapter(t *testing.T) {
	adapter, err := Lookup(AdapterGoTesting)
	if err != nil || adapter.ID() != AdapterGoTesting {
		t.Fatalf("closed go adapter must resolve: %v %+v", err, adapter)
	}
	for _, id := range []string{"rust-cargo/v1", "go-testing/v2", "", "python-unittest/v1"} {
		if _, err := Lookup(id); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_ADAPTER") {
			t.Fatalf("lookup %q must fail UNSUPPORTED_ADAPTER, got %v", id, err)
		}
	}
}

// TestHelperTransportBytesAreBytePinned is a passing control: the embedded
// helper bytes equal the frozen pin and carry the closed transport contract.
func TestHelperTransportBytesAreBytePinned(t *testing.T) {
	body := HelperSource()
	if digest(body) != GoHelperPinnedSHA256 {
		t.Fatalf("embedded helper bytes are not the frozen pin: sha256:%s", digest(body))
	}
	for _, required := range []string{
		"machinery-check/v1 witness id=%s value=%t test=%s site=%s:%d",
		"func Check(t *testing.T, id string, condition bool)",
		"t.Errorf(\"machinery-check/v1 assertion %s evaluated false at %s:%d\"",
	} {
		if !strings.Contains(string(body), required) {
			t.Fatalf("helper bytes lack the frozen transport fragment %q", required)
		}
	}
	if strings.Contains(string(body), "recover()") {
		t.Fatal("helper must never recover surrounding failures")
	}
}

// TestConformanceFixtureBytesAreFrozen is a passing control: the frozen
// conformance fixture bytes carry their pins and every registered identity.
func TestConformanceFixtureBytesAreFrozen(t *testing.T) {
	testBytes, err := os.ReadFile(filepath.Join("assets", "go", "conformance", "conformance_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	modBytes, err := os.ReadFile(filepath.Join("assets", "go", "conformance", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if digest(testBytes) != GoConformanceTestSHA256 || digest(modBytes) != GoConformanceGoModSHA256 {
		t.Fatalf("conformance fixture bytes drifted: test=%s mod=%s", digest(testBytes), digest(modBytes))
	}
	source := string(testBytes)
	for _, id := range []string{"conformance/witness-pass", "conformance/subtest-first", "conformance/subtest-second", "conformance/multi-a", "conformance/multi-b"} {
		if !strings.Contains(source, fmt.Sprintf("%q", id)) {
			t.Fatalf("conformance fixture lacks registered assertion id %s", id)
		}
	}
	if !strings.Contains(string(modBytes), "module machinery.test/conformance") {
		t.Fatal("conformance module identity drifted")
	}
	if !strings.Contains(source, `"machinery.test/conformance/machinerycheck"`) {
		t.Fatal("conformance fixture does not import the materialized helper package")
	}
}

// TestParseWitnessLine pins the closed witness line grammar.
func TestParseWitnessLine(t *testing.T) {
	line := "machinery-check/v1 witness id=unit/alpha value=true test=TestPass site=a_test.go:10"
	w, ok := parseWitnessLine(line)
	if !ok || w.id != "unit/alpha" || !w.value || w.test != "TestPass" || w.site != "a_test.go" || w.line != 10 {
		t.Fatalf("witness not parsed: %+v ok=%v", w, ok)
	}
	if w, ok := parseWitnessLine(strings.Replace(line, "value=true", "value=false", 1)); !ok || w.value {
		t.Fatalf("false witness not parsed: %+v ok=%v", w, ok)
	}
	for _, broken := range []string{
		"",
		"machinery-check/v1 witness",
		"machinery-check/v1 witness id=x value=true site=a.go:1",
		"machinery-check/v1 witness id=x value=true test=T site=a.go",
		"machinery-check/v1 witness id=x value=maybe test=T site=a.go:3",
		"machinery-check/v1 witness id=x value=true test=T site=a.go:0",
		"ordinary log line",
		"machinery-check/v1 witness id=x value=true test=T site=a.go:3 extra=1",
	} {
		if _, ok := parseWitnessLine(broken); ok {
			t.Fatalf("broken witness line accepted: %q", broken)
		}
	}
}

// TestParseGoTestStreamClosedVocabulary pins the closed test2json decode.
func TestParseGoTestStreamClosedVocabulary(t *testing.T) {
	raw := strings.Join([]string{
		`{"Time":"2026-09-07T03:28:16.038147-07:00","Action":"start","Package":` + jsonQuote(unitModule) + `}`,
		eventLine("run", unitModule, "TestPass", ""),
		eventLine("output", unitModule, "TestPass", witnessOutput("TestPass", "unit/alpha", "a_test.go", 10, true)),
		eventLine("pass", unitModule, "TestPass", ""),
		eventLine("pass", unitModule, "", ""),
	}, "\n")
	events, err := parseGoTestStream([]byte(raw))
	if err != nil {
		t.Fatalf("legal stream rejected: %v", err)
	}
	if len(events) != 5 || events[1].Action != "run" || events[1].Test != "TestPass" {
		t.Fatalf("stream not decoded: %+v", events)
	}
	if !strings.Contains(events[2].Output, "machinery-check/v1 witness") {
		t.Fatalf("output not retained: %+v", events[2])
	}
	for name, line := range map[string]string{
		"unknown key":     `{"Action":"run","Package":"p","Test":"T","Surprise":1}`,
		"unknown action":  `{"Action":"teleport","Package":"p"}`,
		"not an object":   `["run"]`,
		"garbage":         `{`,
		"missing action":  `{"Package":"p"}`,
		"non-string test": `{"Action":"run","Test":7}`,
	} {
		if _, err := parseGoTestStream([]byte(line + "\n")); err == nil {
			t.Fatalf("%s must be rejected by the closed decoder: %s", name, line)
		}
	}
	if events, err := parseGoTestStream(nil); err != nil || len(events) != 0 {
		t.Fatalf("empty stream must decode to no events: %v %v", events, err)
	}
}

func jsonQuote(s string) string {
	body, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(body)
}

// TestBuildGoRunPatternAnchorsDeclaredIdentities pins the anchored native
// selection: parent deduplication and regex metacharacter quoting.
func TestBuildGoRunPatternAnchorsDeclaredIdentities(t *testing.T) {
	pattern, err := buildGoRunPattern([]tdd.Test{
		{Native: tdd.NativeID{Test: "TestAlpha"}},
		{Native: tdd.NativeID{Test: "TestBeta/first"}},
		{Native: tdd.NativeID{Test: "TestBeta/second"}},
		{Native: tdd.NativeID{Test: "TestMeta(x)?*"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `^(TestAlpha|TestBeta|TestMeta\(x\)\?\*)$`
	if pattern != want {
		t.Fatalf("run pattern is %q, want %q", pattern, want)
	}
	if _, err := buildGoRunPattern([]tdd.Test{{Native: tdd.NativeID{Test: ""}}}); err == nil {
		t.Fatal("empty native test identity must fail")
	}
	if _, err := buildGoRunPattern(nil); err == nil {
		t.Fatal("empty selection must fail")
	}
}

// TestBuildGoEnvIsClosedAndMinimal pins the closed native environment.
func TestBuildGoEnvIsClosedAndMinimal(t *testing.T) {
	env, err := buildGoEnv("/scratch", "/goroot", []tdd.EnvironmentVar{{Name: "SUITE_FLAG", Value: "declared"}})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(env, "\n")
	for _, required := range []string{
		"PATH=/goroot/bin:/usr/bin:/bin", "HOME=/scratch/home", "TMPDIR=/scratch/tmp",
		"GOFLAGS=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "CGO_ENABLED=0",
		"GOCACHE=/scratch/gocache", "GOMODCACHE=/scratch/gomodcache", "GOPATH=/scratch/gopath",
		"SUITE_FLAG=declared",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("closed env lacks %s:\n%s", required, joined)
		}
	}
	for _, forbidden := range []string{"MACHINERY_SENTINEL", "GOFLAGS=-mod=", "GOTOOLCHAIN=go1.99.0", "GOPROXY=https://"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("closed env retained hostile setting %s:\n%s", forbidden, joined)
		}
	}
	for _, reserved := range []string{"GOFLAGS", "GOPROXY", "GOTOOLCHAIN", "PATH", "GOCACHE", "GOMODCACHE", "GOPATH", "HOME", "TMPDIR", "GOWORK", "CGO_ENABLED"} {
		if _, err := buildGoEnv("/scratch", "/goroot", []tdd.EnvironmentVar{{Name: reserved, Value: "hostile"}}); err == nil {
			t.Fatalf("declared environment may not override the closed key %s", reserved)
		}
	}
	if _, err := buildGoEnv("/scratch", "/goroot", []tdd.EnvironmentVar{{Name: "1BAD", Value: "x"}}); err == nil {
		t.Fatal("invalid declared environment name must fail")
	}
}

// TestValidateGoSuiteSourcesMatrix pins the frozen-source structural
// validation over synthetic module trees.
func TestValidateGoSuiteSourcesMatrix(t *testing.T) {
	const validSource = `package p

import (
	"testing"

	"unit.test/p/machinerycheck"
)

func TestPass(t *testing.T) {
	machinerycheck.Check(t, "unit/alpha", 6*7 == 42)
}

func TestFail(t *testing.T) {
	machinerycheck.Check(t, "unit/beta", 6*7 == 43)
}
`
	suiteFor := func(t *testing.T, mutations ...func(*tdd.Suite)) *tdd.Suite {
		s := unitSuite()
		for _, m := range mutations {
			m(s)
		}
		return s
	}
	lineOf := func(source, anchor string) int64 {
		for i, line := range strings.Split(source, "\n") {
			if strings.Contains(line, anchor) {
				return int64(i + 1)
			}
		}
		t.Fatalf("anchor %q not in source", anchor)
		return 0
	}
	writeModule := func(t *testing.T, source string) string {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, GoHelperDir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, GoHelperDir, "machinerycheck.go"), HelperSource(), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "a_test.go"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	baseSuite := func(t *testing.T, source string) *tdd.Suite {
		s := suiteFor(t)
		s.Tests[0].Assertions[0].Line = lineOf(source, `"unit/alpha"`)
		s.Tests[1].Assertions[0].Line = lineOf(source, `"unit/beta"`)
		return s
	}

	t.Run("valid sources pass", func(t *testing.T) {
		dir := writeModule(t, validSource)
		if err := validateGoSuiteSources(dir, unitModule, baseSuite(t, validSource)); err != nil {
			t.Fatalf("valid frozen sources rejected: %v", err)
		}
	})
	reject := func(t *testing.T, name, source string, mutate func(*tdd.Suite), want string) {
		t.Helper()
		dir := writeModule(t, source)
		suite := baseSuite(t, source)
		if mutate != nil {
			mutate(suite)
		}
		err := validateGoSuiteSources(dir, unitModule, suite)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s must fail with %q, got %v", name, want, err)
		}
	}
	t.Run("wrong line rejected", func(t *testing.T) {
		reject(t, "line", validSource, func(s *tdd.Suite) { s.Tests[0].Assertions[0].Line++ }, "ASSERTION_MISMATCH")
	})
	t.Run("wrong id rejected", func(t *testing.T) {
		reject(t, "id", validSource, func(s *tdd.Suite) { s.Tests[0].Assertions[0].ID = "unit/other" }, "ASSERTION_MISMATCH")
	})
	t.Run("wrong file rejected", func(t *testing.T) {
		reject(t, "file", validSource, func(s *tdd.Suite) { s.Tests[0].Assertions[0].Source = "other_test.go" }, "MISSING_TEST")
	})
	t.Run("helper import missing rejected", func(t *testing.T) {
		source := strings.Replace(validSource, `"unit.test/p/machinerycheck"`, `"elsewhere/machinerycheck"`, 1)
		dir := writeModule(t, source)
		if err := validateGoSuiteSources(dir, unitModule, baseSuite(t, validSource)); err == nil {
			t.Fatal("lookalike helper import must be rejected")
		}
	})
	t.Run("non-helper call site rejected", func(t *testing.T) {
		source := strings.Replace(validSource, `machinerycheck.Check(t, "unit/alpha", 6*7 == 42)`, `localCheck(t, "unit/alpha", 6*7 == 42)`, 1)
		suite := baseSuite(t, validSource)
		dir := writeModule(t, source)
		err := validateGoSuiteSources(dir, unitModule, suite)
		if err == nil || !strings.Contains(err.Error(), "ASSERTION_MISMATCH") {
			t.Fatalf("non-typed call site must fail ASSERTION_MISMATCH, got %v", err)
		}
	})
	t.Run("non-literal id rejected", func(t *testing.T) {
		source := strings.Replace(validSource, `"unit/alpha", 6*7 == 42`, `assertionID, 6*7 == 42`, 1)
		dir := writeModule(t, source)
		if err := validateGoSuiteSources(dir, unitModule, baseSuite(t, validSource)); err == nil {
			t.Fatal("non-literal assertion id must be rejected")
		}
	})
	t.Run("custom TestMain rejected", func(t *testing.T) {
		reject(t, "testmain", validSource+"func TestMain(m *testing.M) { os.Exit(m.Run()) }\n", nil, "UNSUPPORTED_FEATURE")
	})
	t.Run("benchmark entry rejected", func(t *testing.T) {
		reject(t, "bench", validSource+"func BenchmarkX(b *testing.B) { b.ResetTimer() }\n", nil, "UNSUPPORTED_FEATURE")
	})
	t.Run("fuzz entry rejected", func(t *testing.T) {
		reject(t, "fuzz", validSource+"func FuzzY(f *testing.F) { f.Fuzz(func(t *testing.T, s string) {}) }\n", nil, "UNSUPPORTED_FEATURE")
	})
	t.Run("dynamic subtest name rejected", func(t *testing.T) {
		source := validSource + "func TestDyn(t *testing.T) {\n\tt.Run(fmt.Sprintf(\"x-%d\", 1), func(t *testing.T) {})\n}\n"
		suite := suiteFor(t)
		suite.Tests = append(suite.Tests, tdd.Test{
			ID: "dyn", Source: "a_test.go", Native: tdd.NativeID{Package: unitModule, Test: "TestDyn/x-1"},
		})
		dir := writeModule(t, source)
		if err := validateGoSuiteSources(dir, unitModule, suite); err == nil {
			t.Fatal("generated subtest identity must be rejected")
		}
	})
	t.Run("undeclared fixed subtest rejected", func(t *testing.T) {
		source := validSource + "func TestSub(t *testing.T) {\n\tt.Run(\"kid\", func(t *testing.T) {\n\t\tmachinerycheck.Check(t, \"unit/kid\", true)\n\t})\n\tt.Run(\"kid2\", func(t *testing.T) {\n\t\tmachinerycheck.Check(t, \"unit/kid2\", true)\n\t})\n}\n"
		suite := baseSuite(t, source)
		suite.Tests = append(suite.Tests, tdd.Test{
			ID: "kid", Source: "a_test.go", Native: tdd.NativeID{Package: unitModule, Test: "TestSub/kid"},
			Assertions: []tdd.Assertion{{ID: "unit/kid", Source: "a_test.go", Line: lineOf(source, `"unit/kid"`), Helper: "machinery-check/v1"}},
		})
		dir := writeModule(t, source)
		err := validateGoSuiteSources(dir, unitModule, suite)
		if err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
			t.Fatalf("undeclared fixed subtest must fail MISSING_TEST, got %v", err)
		}
	})
	t.Run("declared subtest without parent coverage rejected", func(t *testing.T) {
		source := validSource + "func TestSub(t *testing.T) {\n\tt.Run(\"kid\", func(t *testing.T) {\n\t\tmachinerycheck.Check(t, \"unit/kid\", true)\n\t})\n}\n"
		suite := baseSuite(t, source)
		suite.Tests = append(suite.Tests, tdd.Test{
			ID: "kid", Source: "a_test.go", Native: tdd.NativeID{Package: unitModule, Test: "TestSub/kid"},
			Assertions: []tdd.Assertion{{ID: "unit/kid", Source: "a_test.go", Line: lineOf(source, `"unit/kid"`), Helper: "machinery-check/v1"}},
		})
		dir := writeModule(t, source)
		err := validateGoSuiteSources(dir, unitModule, suite)
		if err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
			t.Fatalf("child without declared parent must fail MISSING_TEST, got %v", err)
		}
	})
}

func passStream() []string {
	return []string{
		eventLine("start", unitModule, "", ""),
		eventLine("run", unitModule, "TestPass", ""),
		eventLine("output", unitModule, "TestPass", witnessOutput("TestPass", "unit/alpha", "a_test.go", 10, true)),
		eventLine("pass", unitModule, "TestPass", ""),
		eventLine("output", unitModule, "", "PASS\n"),
		eventLine("pass", unitModule, "", ""),
	}
}

func failureStream() []string {
	return []string{
		eventLine("start", unitModule, "", ""),
		eventLine("run", unitModule, "TestFail", ""),
		eventLine("output", unitModule, "TestFail", witnessOutput("TestFail", "unit/beta", "a_test.go", 15, false)),
		eventLine("output", unitModule, "TestFail", "    machinerycheck.go:39: machinery-check/v1 assertion unit/beta evaluated false at a_test.go:15\n"),
		eventLine("fail", unitModule, "TestFail", ""),
		eventLine("fail", unitModule, "", ""),
	}
}

func mustParse(t *testing.T, lines []string) []goJSONEvent {
	t.Helper()
	events, err := parseGoTestStream([]byte(strings.Join(lines, "\n")))
	if err != nil {
		t.Fatalf("stream must parse: %v", err)
	}
	return events
}

// TestReconcileLegalPassStream pins the exact normalized event sequence of a
// passing native suite with one registered assertion witness.
func TestReconcileLegalPassStream(t *testing.T) {
	suite := unitSuite()
	suite.Tests = suite.Tests[:1]
	rec, err := reconcileGoStream(mustParse(t, passStream()), suite)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Outcome != "pass" {
		t.Fatalf("outcome is %q", rec.Outcome)
	}
	want := []struct {
		kind      string
		test      string
		assertion string
		outcome   string
	}{
		{"suite-start", "", "", ""},
		{"discovered", "TestPass", "", ""},
		{"test-start", "TestPass", "", ""},
		{"assertion", "TestPass", "unit/alpha", "pass"},
		{"test-end", "TestPass", "", "pass"},
		{"suite-end", "", "", "pass"},
	}
	if len(rec.Events) != len(want) {
		t.Fatalf("normalized stream is %+v", rec.Events)
	}
	for i, expect := range want {
		got := rec.Events[i]
		if got.Schema != GoEventSchema || got.Sequence != int64(i+1) || got.Suite != "go-unit" || got.Kind != expect.kind {
			t.Fatalf("event %d is %+v, want kind=%s", i, got, expect.kind)
		}
		if expect.test != "" && (got.Native == nil || got.Native.Test != expect.test || got.Native.Package != unitModule) {
			t.Fatalf("event %d carries wrong native identity: %+v", i, got)
		}
		if expect.test == "" && got.Native != nil {
			t.Fatalf("event %d must carry no native identity: %+v", i, got)
		}
		if got.Assertion != expect.assertion || outcomeOf(got) != expect.outcome {
			t.Fatalf("event %d is %+v, want assertion=%s outcome=%s", i, got, expect.assertion, expect.outcome)
		}
	}
	if len(rec.Discovered) != 1 || len(rec.Started) != 1 || len(rec.Completed) != 1 || len(rec.Assertions) != 1 || rec.Assertions[0].Outcome != "pass" {
		t.Fatalf("reconciliation inventory is %+v", rec)
	}
}

func outcomeOf(e tdd.Event) string {
	if e.Kind == "assertion" || e.Kind == "test-end" || e.Kind == "suite-end" {
		return e.Outcome
	}
	return ""
}

// TestReconcileAssertionFailure pins the RED signal: a false registered
// assertion is a native failure reconciled to the exact call site, not an
// aggregate-only result.
func TestReconcileAssertionFailure(t *testing.T) {
	suite := unitSuite()
	suite.Tests = suite.Tests[1:]
	rec, err := reconcileGoStream(mustParse(t, failureStream()), suite)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Outcome != "assertion-fail" {
		t.Fatalf("outcome is %q, want assertion-fail", rec.Outcome)
	}
	var failEnd, assertion bool
	for _, e := range rec.Events {
		if e.Kind == "assertion" && e.Assertion == "unit/beta" && e.Outcome == "assertion-fail" {
			assertion = true
			if e.Source != "a_test.go" || e.Line != 15 {
				t.Fatalf("failing assertion not bound to its frozen site: %+v", e)
			}
		}
		if e.Kind == "test-end" && e.Native != nil && e.Native.Test == "TestFail" && e.Outcome == "assertion-fail" {
			failEnd = true
		}
	}
	if !assertion || !failEnd {
		t.Fatalf("failure not reconciled assertion-specifically: %+v", rec.Events)
	}
	if rec.Assertions[0].Outcome != "assertion-fail" {
		t.Fatalf("assertion inventory is %+v", rec.Assertions)
	}
}

// TestReconcileCoversParentAndChildLifecycle pins subtest accounting: the
// parent runs and completes, the declared leaf child carries the witness and
// parent-only pass is never leaf coverage.
func TestReconcileCoversParentAndChildLifecycle(t *testing.T) {
	suite := unitSuite()
	suite.Tests = []tdd.Test{
		{ID: "parent", Source: "a_test.go", Native: tdd.NativeID{Package: unitModule, Test: "TestSub"}},
		{
			ID: "kid", Source: "a_test.go", Native: tdd.NativeID{Package: unitModule, Test: "TestSub/kid"},
			Assertions: []tdd.Assertion{{ID: "unit/kid", Source: "a_test.go", Line: 4, Helper: "machinery-check/v1"}},
		},
	}
	stream := []string{
		eventLine("start", unitModule, "", ""),
		eventLine("run", unitModule, "TestSub", ""),
		eventLine("run", unitModule, "TestSub/kid", ""),
		eventLine("output", unitModule, "TestSub/kid", witnessOutput("TestSub/kid", "unit/kid", "a_test.go", 4, true)),
		eventLine("pass", unitModule, "TestSub/kid", ""),
		eventLine("pass", unitModule, "TestSub", ""),
		eventLine("pass", unitModule, "", ""),
	}
	rec, err := reconcileGoStream(mustParse(t, stream), suite)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Outcome != "pass" || len(rec.Completed) != 2 {
		t.Fatalf("parent+child not both accounted: %+v", rec)
	}
	// Parent-only pass (child never ran) must be INCOMPLETE_EVENTS.
	truncated := append([]string{}, stream...)
	truncated = removeEvent(truncated, `"Test":"TestSub/kid"`)
	if _, err := reconcileGoStream(mustParse(t, truncated), suite); err == nil || !strings.Contains(err.Error(), "INCOMPLETE_EVENTS") {
		t.Fatalf("missing child execution must fail INCOMPLETE_EVENTS, got %v", err)
	}
}

func removeEvent(lines []string, marker string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if !strings.Contains(line, marker) {
			out = append(out, line)
		}
	}
	return out
}

// TestReconcileRejectsStreamForgeries is the enumerated adversarial stream
// matrix: every forgery class fails closed with its closed diagnostic.
func TestReconcileRejectsStreamForgeries(t *testing.T) {
	singlePass := func() *tdd.Suite {
		s := unitSuite()
		s.Tests = s.Tests[:1]
		return s
	}
	base := passStream()
	replace := func(from, to string) []string {
		out := append([]string{}, base...)
		for i, line := range out {
			if strings.Contains(line, from) {
				out[i] = strings.Replace(line, from, to, 1)
			}
		}
		return out
	}
	for _, tc := range []struct {
		name   string
		stream []string
		want   string
	}{
		{"empty stream", nil, "INCOMPLETE_EVENTS"},
		{"no package start", base[1:], "INCOMPLETE_EVENTS"},
		{"no package terminal", base[:len(base)-1], "INCOMPLETE_EVENTS"},
		{"unexpected native identity", append(append([]string{}, base[:3]...), eventLine("run", unitModule, "TestUnexpected", ""), base[3], base[4], base[5]), "DUPLICATE_TEST"},
		{"duplicate terminal", append(append([]string{}, base...), eventLine("pass", unitModule, "TestPass", "")), "DUPLICATE_TEST"},
		{"terminal without run", []string{eventLine("start", unitModule, "", ""), eventLine("pass", unitModule, "TestPass", ""), eventLine("pass", unitModule, "", "")}, "DUPLICATE_TEST"},
		{"skipped declared test", replace(`"Action":"pass","Package":`+jsonQuote(unitModule)+`,"Test":"TestPass"`, `"Action":"skip","Package":`+jsonQuote(unitModule)+`,"Test":"TestPass"`), "skipped"},
		{"cached result", replace("PASS\\n", "ok \\tunit.test/p\\t0.1s (cached)\\n"), "cached"},
		{"build failed package", []string{`{"ImportPath":` + jsonQuote(unitModule) + `,"Action":"build-fail"}`, eventLine("start", unitModule, "", ""), eventLine("output", unitModule, "", "FAIL\\tunit.test/p [build failed]\\n"), eventLine("fail", unitModule, "", "")}, "BUILD_ERROR"},
		{"duplicate witness", append(append([]string{}, base[:3]...), base[2], base[3], base[4], base[5]), "ASSERTION_MISMATCH"},
		{"unknown witness id", replace("id=unit/alpha", "id=unit/ghost"), "ASSERTION_MISMATCH"},
		{"witness test mismatch", replace("test=TestPass", "test=TestFail"), "ASSERTION_MISMATCH"},
		{"witness site mismatch", replace("site=a_test.go:10", "site=a_test.go:11"), "ASSERTION_MISMATCH"},
		{"pass with false witness", replace("value=true", "value=false"), "ASSERTION_MISMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := reconcileGoStream(mustParse(t, tc.stream), singlePass()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("must fail with %q, got %v", tc.want, err)
			}
		})
	}
	// A native failure with no witnessed registered assertion is never
	// assertion-specific evidence.
	noWitness := []string{
		eventLine("start", unitModule, "", ""),
		eventLine("run", unitModule, "TestPass", ""),
		eventLine("output", unitModule, "TestPass", "    a_test.go:11: direct failure\\n"),
		eventLine("fail", unitModule, "TestPass", ""),
		eventLine("fail", unitModule, "", ""),
	}
	if _, err := reconcileGoStream(mustParse(t, noWitness), singlePass()); err == nil || !strings.Contains(err.Error(), "UNEXPECTED_FAILURE") {
		t.Fatalf("unwitnessed failure must fail UNEXPECTED_FAILURE, got %v", err)
	}
	// Pause without a matching continue never completes a test.
	paused := []string{
		eventLine("start", unitModule, "", ""),
		eventLine("run", unitModule, "TestPass", ""),
		eventLine("pause", unitModule, "TestPass", ""),
		eventLine("pass", unitModule, "TestPass", ""),
		eventLine("pass", unitModule, "", ""),
	}
	if _, err := reconcileGoStream(mustParse(t, paused), singlePass()); err == nil {
		t.Fatal("pause without continue must fail closed")
	}
}

// TestReconcileRejectsEmptyAndMissingSelection pins empty and incomplete
// inventory accounting.
func TestReconcileRejectsEmptyAndMissingSelection(t *testing.T) {
	full := unitSuite()
	passOnly := mustParse(t, passStream())
	broken := unitSuite()
	broken.Tests = full.Tests
	// The failing declared test never executes in the passing stream.
	if _, err := reconcileGoStream(passOnly, broken); err == nil || !strings.Contains(err.Error(), "INCOMPLETE_EVENTS") {
		t.Fatalf("declared-but-never-run test must fail INCOMPLETE_EVENTS, got %v", err)
	}
	if _, err := reconcileGoStream(passOnly, &tdd.Suite{ID: "x", Tests: nil}); err == nil {
		t.Fatal("empty declared selection must fail")
	}
}
