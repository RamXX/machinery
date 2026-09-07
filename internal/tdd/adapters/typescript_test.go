package adapters

// RED contract for MAC-avfp (unit subjects): the closed node-test-typescript
// /v1 adapter surfaces. The frozen asset pins and lookup subjects are passing
// controls; every parse/reconcile/validation subject fails on the RED stub
// with the pending-implementation error and defines the closed behavior.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
)

func tsSHA(t *testing.T, b []byte) string {
	t.Helper()
	return tsDigestHex(b)
}

func tsConformanceFixtureBytes(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("assets/typescript/conformance/conformance.ts")
	if err != nil {
		t.Fatal(err)
	}
	if got := tsSHA(t, b); got != TypeScriptConformanceFixtureSHA256 {
		t.Fatalf("frozen conformance fixture bytes changed: sha256 %s", got)
	}
	return b
}

// TestTypeScriptAssetInventoryMatchesFrozenPins is a passing control: every
// embedded asset byte set is present and equals its frozen pin, and the
// embedded helper bytes are the frozen machinery-check/v1 transport.
func TestTypeScriptAssetInventoryMatchesFrozenPins(t *testing.T) {
	files := map[string][]byte{
		"machinery-check.ts": tsHelperSource,
		"node-ambient.d.ts":  tsAmbientSource,
		"tsconfig.json":      tsConfigSource,
		"package.json":       tsPackageSource,
		"reporter.mjs":       tsReporterSource,
	}
	want := TypeScriptAssets()
	if len(want) != len(files) {
		t.Fatalf("frozen asset inventory has %d entries, embedded set has %d", len(want), len(files))
	}
	for name, body := range files {
		pin, ok := want[name]
		if !ok {
			t.Fatalf("embedded asset %q is absent from the frozen inventory", name)
		}
		if got := tsSHA(t, body); got != pin {
			t.Fatalf("embedded asset %s sha256 %s does not match its frozen pin %s", name, got, pin)
		}
	}
	if got := tsSHA(t, TypeScriptHelperSource()); got != TypeScriptHelperPinnedSHA256 {
		t.Fatalf("helper copy sha256 %s does not match the frozen pin", got)
	}
	if !strings.Contains(string(tsHelperSource), "machinery.tdd.witness/v1") {
		t.Fatal("helper transport does not carry the witness schema identity")
	}
	if !strings.Contains(string(tsReporterSource), TypeScriptReporterSentinel) {
		t.Fatal("embedded reporter does not emit the terminal sentinel")
	}
}

// TestTypeScriptLookupResolvesClosedIdentity is a passing control: the
// registry resolves exactly the closed adapter identity.
func TestTypeScriptLookupResolvesClosedIdentity(t *testing.T) {
	adapter, err := Lookup(AdapterNodeTestTS)
	if err != nil || adapter.ID() != AdapterNodeTestTS {
		t.Fatalf("closed identity did not resolve: %v %+v", err, adapter)
	}
	if _, err := Lookup("jest/v1"); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_ADAPTER") {
		t.Fatalf("foreign adapter identity must fail closed: %v", err)
	}
}

// TestParseTypeScriptWitness freezes the witness grammar: the closed schema,
// strictly boolean conditions, the evaluated phase, the compiled call site
// and the recorded native failure class.
func TestParseTypeScriptWitness(t *testing.T) {
	good := `{"schema":"machinery.tdd.witness/v1","assertion":"unit/alpha","phase":"evaluated","site":"/prep/out/conformance.js:4:5","condition":true,"thrown":null}`
	w, ok := parseTypeScriptWitness(good)
	if !ok || w.assertion != "unit/alpha" || !w.condition || w.thrown != "" || w.site != "/prep/out/conformance.js:4:5" {
		t.Fatalf("witness did not parse: %+v ok=%v", w, ok)
	}
	fail := `{"schema":"machinery.tdd.witness/v1","assertion":"unit/alpha","phase":"evaluated","site":"/prep/out/conformance.js:4:5","condition":false,"thrown":"AssertionError"}`
	w, ok = parseTypeScriptWitness(fail)
	if !ok || w.condition || w.thrown != "AssertionError" {
		t.Fatalf("failure witness did not parse: %+v ok=%v", w, ok)
	}
	for _, bad := range []string{
		`not json`,
		`{"schema":"other/v1","assertion":"a","phase":"evaluated","site":"x:1:1","condition":true,"thrown":null}`,
		`{"schema":"machinery.tdd.witness/v1","assertion":"a","phase":"entered","site":"x:1:1","condition":true,"thrown":null}`,
		`{"schema":"machinery.tdd.witness/v1","assertion":"a","phase":"evaluated","site":"","condition":true,"thrown":null}`,
	} {
		if _, ok := parseTypeScriptWitness(bad); ok {
			t.Fatalf("invalid witness accepted: %s", bad)
		}
	}
}

// tsRealStreamLines is a frozen sample of REAL node:test embedded-reporter
// output (Node 26.8.1, paths normalized to a fixed prepared root): the
// enqueue/dequeue vocabulary with entry-file scoping, a passed complete, the
// helper witness diagnostic, both native summaries and the terminal
// sentinel.
const tsRealStreamLines = `{"t":"test:enqueue","d":{"nesting":0,"name":"alpha case","type":"test","testId":1,"parentId":0,"line":5,"column":3,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:dequeue","d":{"nesting":0,"name":"alpha case","type":"test","testId":1,"parentId":0,"line":5,"column":3,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:complete","d":{"name":"alpha case","nesting":0,"testNumber":1,"testId":1,"parentId":0,"details":{"duration_ms":0.5,"type":"test","passed":true},"line":5,"column":3,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:start","d":{"nesting":0,"name":"alpha case","type":"test","testId":1,"parentId":0,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:pass","d":{"name":"alpha case","nesting":0,"testNumber":1,"testId":1,"parentId":0,"details":{"duration_ms":0.5,"type":"test"},"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:diagnostic","d":{"nesting":0,"message":"{\"schema\":\"machinery.tdd.witness/v1\",\"assertion\":\"unit/alpha\",\"phase\":\"evaluated\",\"site\":\"/prep/out/conformance.js:4:5\",\"condition\":true,\"thrown\":null}","level":"info","line":5,"column":3,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:summary","d":{"success":true,"counts":{"tests":1,"failed":0,"passed":1,"cancelled":0,"skipped":0,"todo":0,"topLevel":1,"suites":0},"duration_ms":4.2,"file":"/prep/out/conformance.js","entryFile":"/prep/out/conformance.js"}}
{"t":"test:summary","d":{"success":true,"counts":{"tests":1,"failed":0,"passed":1,"cancelled":0,"skipped":0,"todo":0,"topLevel":1,"suites":0},"duration_ms":46.2,"file":null}}
{"t":"machinery:reporter:end"}
`

// TestParseTypeScriptReporterStreamAcceptsRealVocabulary freezes the closed
// parser over the frozen real sample: every event decodes with its identity,
// entry-file scoping and line/column, and the terminal sentinel is present
// exactly once.
func TestParseTypeScriptReporterStreamAcceptsRealVocabulary(t *testing.T) {
	events, err := parseTypeScriptReporterStream([]byte(tsRealStreamLines))
	if err != nil {
		t.Fatalf("real reporter vocabulary rejected: %v", err)
	}
	if len(events) != 9 {
		t.Fatalf("expected 9 decoded events, got %d", len(events))
	}
	if events[0].Type != "test:enqueue" || events[0].Name != "alpha case" || events[0].EntryFile != "/prep/out/conformance.js" || events[0].Line != 5 || events[0].Column != 3 {
		t.Fatalf("enqueue did not decode: %+v", events[0])
	}
	if events[2].Type != "test:complete" || events[2].Counts != nil {
		t.Fatalf("complete did not decode: %+v", events[2])
	}
	if events[5].Type != "test:diagnostic" || !strings.Contains(events[5].Message, "machinery.tdd.witness/v1") {
		t.Fatalf("witness diagnostic did not decode: %+v", events[5])
	}
	if events[7].Type != "test:summary" || events[7].Counts == nil || !events[7].Counts.Success || events[7].Counts.Passed != 1 || events[7].EntryFile != "" {
		t.Fatalf("final summary did not decode: %+v", events[7].Counts)
	}
	if events[8].Type != TypeScriptReporterSentinel {
		t.Fatalf("terminal sentinel missing: %+v", events[8])
	}
}

// TestParseTypeScriptReporterStreamRejectsForgery freezes the closed-parser
// negative surface: unknown event identities, malformed lines, missing,
// duplicated or non-terminal sentinels and trailing junk all fail closed.
func TestParseTypeScriptReporterStreamRejectsForgery(t *testing.T) {
	base := strings.TrimSuffix(tsRealStreamLines, "\n")
	cases := []struct {
		name  string
		lines []string
	}{
		{"unknown-identity", []string{`{"t":"test:machinery","d":{"name":"x"}}`}},
		{"malformed-json", []string{`{"t":"test:enqueue","d":`}},
		{"missing-sentinel", strings.Split(strings.ReplaceAll(base, `{"t":"machinery:reporter:end"}`, ""), "\n")},
		{"duplicated-sentinel", append(strings.Split(base, "\n"), `{"t":"machinery:reporter:end"}`)},
		{"junk-after-sentinel", append(strings.Split(base, "\n"), `forged native line`)},
		{"empty-stream", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(strings.Join(tc.lines, "\n"))
			if _, err := parseTypeScriptReporterStream(raw); err == nil {
				t.Fatal("forged reporter stream accepted")
			}
		})
	}
}

// tsOracleMapDocument rebuilds the frozen compiler-produced map document of
// the frozen conformance fixture (real tsc 7.0.2 output; mappings captured
// from the pinned compile) with the fixture bytes as embedded sources.
func tsOracleMapDocument(t *testing.T, content string) []byte {
	t.Helper()
	doc := map[string]any{
		"version":        3,
		"file":           "conformance.js",
		"sourceRoot":     "",
		"sources":        []string{"../conformance.ts"},
		"names":          []string{},
		"mappings":       "AAAA,OAAO,EAAE,IAAI,EAAE,MAAM,WAAW,CAAC;AACjC,OAAO,EAAE,KAAK,EAAE,MAAM,sBAAsB,CAAC;AAE7C,IAAI,CAAC,+CAA+C,EAAE,CAAC,CAAC,EAAE,EAAE;IAC1D,KAAK,CAAC,CAAC,EAAE,qBAAqB,EAAE,CAAC,GAAG,CAAC,KAAK,EAAE,CAAC,CAAC;AAChD,CAAC,CAAC,CAAC;AAEH,IAAI,CAAC,6BAA6B,EAAE,KAAK,EAAE,CAAC,CAAC,EAAE,EAAE;IAC9C,MAAM,CAAC,CAAC,IAAI,CAAC,6BAA6B,EAAE,CAAC,CAAC,EAAE,EAAE,EAAE;QACjD,KAAK,CAAC,EAAE,EAAE,oBAAoB,EAAE,CAAC,GAAG,CAAC,KAAK,EAAE,CAAC,CAAC;IAChD,CAAC,CAAC,CAAC;AACL,CAAC,CAAC,CAAC;AAEH,IAAI,CAAC,iCAAiC,EAAE,CAAC,CAAC,EAAE,EAAE;IAC5C,KAAK,CAAC,CAAC,EAAE,qBAAqB,EAAE,CAAC,GAAG,CAAC,KAAK,EAAE,CAAC,CAAC;IAC9C,KAAK,CAAC,CAAC,EAAE,qBAAqB,EAAE,CAAC,GAAG,CAAC,KAAK,EAAE,CAAC,CAAC;AAChD,CAAC,CAAC,CAAC",
		"sourcesContent": []string{content},
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestTypeScriptSourceMapOracle freezes the verified source-map binding: the
// compiler-produced map loads only against its own frozen source bytes and
// maps the four frozen witness sites back to the exact registered
// TypeScript call lines.
func TestTypeScriptSourceMapOracle(t *testing.T) {
	frozen := string(tsConformanceFixtureBytes(t))
	m, err := loadTypeScriptSourceMap(tsOracleMapDocument(t, frozen), "../conformance.ts", []byte(frozen))
	if err != nil {
		t.Fatalf("verified map rejected: %v", err)
	}
	for _, tc := range []struct{ genLine, genCol, wantLine, wantCol int64 }{
		{4, 5, 5, 3},
		{8, 9, 10, 5},
		{12, 5, 15, 3},
		{13, 5, 16, 3},
	} {
		line, col, ok := m.SourcePosition(tc.genLine, tc.genCol)
		if !ok || line != tc.wantLine || col != tc.wantCol {
			t.Fatalf("map position %d:%d resolved to %d:%d ok=%v, want %d:%d", tc.genLine, tc.genCol, line, col, ok, tc.wantLine, tc.wantCol)
		}
	}
}

// TestTypeScriptSourceMapRejectsMismatchedSourcesContent freezes the copied
// JSON / stale map rejection: a map whose embedded sources differ from the
// frozen bundle bytes never certifies anything.
func TestTypeScriptSourceMapRejectsMismatchedSourcesContent(t *testing.T) {
	frozen := string(tsConformanceFixtureBytes(t))
	tampered := strings.Replace(frozen, "6 * 7 === 42", "6 * 7 === 43", 1)
	if _, err := loadTypeScriptSourceMap(tsOracleMapDocument(t, tampered), "../conformance.ts", []byte(frozen)); err == nil {
		t.Fatal("map with mismatched embedded sources accepted")
	}
	if _, err := loadTypeScriptSourceMap(tsOracleMapDocument(t, frozen), "../other.ts", []byte(frozen)); err == nil {
		t.Fatal("map for a different source identity accepted")
	}
}

func tsUnitSuite() *tdd.Suite {
	return &tdd.Suite{
		ID: "unit-ts", Adapter: AdapterNodeTestTS, Root: ".", Files: []string{"conformance.ts"},
		Tests: []tdd.Test{
			{ID: "alpha", Source: "conformance.ts", Native: tdd.NativeID{Source: "conformance.ts", Path: []string{"alpha case"}, Line: 5, Column: 3},
				Assertions: []tdd.Assertion{{ID: "unit/alpha", Source: "conformance.ts", Line: 5, Helper: tdd.AssertionHelperV1}}},
		},
	}
}

func tsUnitEntries() tsEntryIndex { return tsEntryIndex{"/prep/out/conformance.js": "conformance.ts"} }

func tsUnitMaps() map[string]tsSourceMap {
	return map[string]tsSourceMap{"conformance.ts": {
		Generation: 3, Source: "conformance.ts",
		segments: []tsMapSegment{
			{GenLine: 1, GenCol: 0, SrcLine: 1, SrcCol: 1},
			{GenLine: 3, GenCol: 0, SrcLine: 4, SrcCol: 1},
			{GenLine: 4, GenCol: 2, SrcLine: 5, SrcCol: 3},
			{GenLine: 7, GenCol: 0, SrcLine: 9, SrcCol: 1},
			{GenLine: 8, GenCol: 8, SrcLine: 10, SrcCol: 5},
		},
	}}
}

func tsEvent(kind string, mutate func(*tsNativeEvent)) tsNativeEvent {
	e := tsNativeEvent{Type: kind, File: "/prep/out/conformance.js", EntryFile: "/prep/out/conformance.js"}
	if mutate != nil {
		mutate(&e)
	}
	return e
}

func tsWitnessLine(condition bool, site, thrown string) tsNativeEvent {
	msg := `{"schema":"machinery.tdd.witness/v1","assertion":"unit/alpha","phase":"evaluated","site":"` + site + `","condition":` + jsonBool(condition) + `,"thrown":` + thrown + `}`
	return tsEvent("test:diagnostic", func(e *tsNativeEvent) { e.Message = msg; e.Name = "alpha case" })
}

func jsonBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func tsLegalPassStream() []tsNativeEvent {
	return []tsNativeEvent{
		tsEvent("test:enqueue", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1; e.Line, e.Column = 5, 3 }),
		tsEvent("test:dequeue", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:complete", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:start", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:pass", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsWitnessLine(true, "/prep/out/conformance.js:4:5", "null"),
		tsEvent("test:summary", func(e *tsNativeEvent) { e.EntryFile, e.File = "", ""; e.Counts = &tsSummaryCounts{Tests: 1, Passed: 1, Success: true} }),
		tsEvent(TypeScriptReporterSentinel, func(e *tsNativeEvent) { e.File, e.EntryFile = "", "" }),
	}
}

// TestReconcileTypeScriptStreamLegalPass freezes the normalized event
// automaton: the legal pass sequence produces the exact closed
// machinery.tdd.event/v1 order with owner-assigned identity and a passing
// assertion outcome bound to the registered site.
func TestReconcileTypeScriptStreamLegalPass(t *testing.T) {
	rec, err := reconcileTypeScriptStream(tsLegalPassStream(), tsUnitSuite(), tsUnitEntries(), tsUnitMaps())
	if err != nil {
		t.Fatalf("legal pass stream rejected: %v", err)
	}
	if rec.Outcome != "pass" {
		t.Fatalf("outcome %q is not pass", rec.Outcome)
	}
	kinds := make([]string, 0, len(rec.Events))
	for _, e := range rec.Events {
		kinds = append(kinds, e.Kind)
		if e.Schema != TypeScriptEventSchema || e.Suite != "unit-ts" || e.Sequence <= 0 {
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
	if rec.Started[0].Path[0] != "alpha case" {
		t.Fatalf("native identity lost: %+v", rec.Started)
	}
}

// TestReconcileTypeScriptStreamAssertionFailureBindsRegisteredSite freezes
// the RED evidence path: a native failure is an assertion-fail only when the
// helper witness records a false strictly-boolean condition, a native
// AssertionError cause, and a call site the verified source map resolves to
// the exact registered frozen line.
func TestReconcileTypeScriptStreamAssertionFailureBindsRegisteredSite(t *testing.T) {
	stream := []tsNativeEvent{
		tsEvent("test:enqueue", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:dequeue", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:complete", func(e *tsNativeEvent) { e.Name = "alpha case"; e.Error = &tsNativeError{Name: "Error", Message: "machinery-check/v1 assertion unit/alpha failed", Code: "ERR_TEST_FAILURE"} }),
		tsEvent("test:start", func(e *tsNativeEvent) { e.Name = "alpha case"; e.TestID = 1 }),
		tsEvent("test:fail", func(e *tsNativeEvent) { e.Name = "alpha case"; e.Error = &tsNativeError{Name: "Error", Message: "machinery-check/v1 assertion unit/alpha failed", Code: "ERR_TEST_FAILURE"} }),
		tsWitnessLine(false, "/prep/out/conformance.js:4:5", `"AssertionError"`),
		tsEvent("test:summary", func(e *tsNativeEvent) { e.EntryFile, e.File = "", ""; e.Counts = &tsSummaryCounts{Tests: 1, Passed: 0, Failed: 1, Success: false} }),
		tsEvent(TypeScriptReporterSentinel, func(e *tsNativeEvent) { e.File, e.EntryFile = "", "" }),
	}
	rec, err := reconcileTypeScriptStream(stream, tsUnitSuite(), tsUnitEntries(), tsUnitMaps())
	if err != nil {
		t.Fatalf("assertion-fail stream rejected: %v", err)
	}
	if rec.Outcome != "fail" || len(rec.Failed) != 1 {
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

// TestReconcileTypeScriptStreamRejectsViolations freezes the reconciliation
// negative surface: every violation fails closed with its closed diagnostic
// identity and never converts into success.
func TestReconcileTypeScriptStreamRejectsViolations(t *testing.T) {
	drop := func(kind string) func([]tsNativeEvent) []tsNativeEvent {
		return func(s []tsNativeEvent) []tsNativeEvent {
			out := s[:0:0]
			for _, e := range s {
				if e.Type != kind {
					out = append(out, e)
				}
			}
			return out
		}
	}
	replace := func(kind string, with tsNativeEvent) func([]tsNativeEvent) []tsNativeEvent {
		return func(s []tsNativeEvent) []tsNativeEvent {
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
	badSummary := tsEvent("test:summary", func(e *tsNativeEvent) { e.EntryFile, e.File = "", ""; e.Counts = &tsSummaryCounts{Tests: 1, Passed: 1, Skipped: 1, Success: true} })
	unknownTest := tsEvent("test:enqueue", func(e *tsNativeEvent) { e.Name = "unregistered case"; e.TestID = 9 })
	skipFlag := tsEvent("test:complete", func(e *tsNativeEvent) { e.Name = "alpha case"; e.Skip = true })
	failNoWitness := tsEvent("test:complete", func(e *tsNativeEvent) {
		e.Name = "alpha case"
		e.Error = &tsNativeError{Name: "Error", Message: "direct assertion outside the helper", Code: "ERR_TEST_FAILURE"}
	})
	wrongSite := tsWitnessLine(false, "/prep/out/conformance.js:3:1", `"AssertionError"`)
	wrongClass := tsWitnessLine(false, "/prep/out/conformance.js:4:5", `"TypeError"`)
	unknownAssertion := tsEvent("test:diagnostic", func(e *tsNativeEvent) {
		e.Message = `{"schema":"machinery.tdd.witness/v1","assertion":"unit/ghost","phase":"evaluated","site":"/prep/out/conformance.js:4:5","condition":true,"thrown":null}`
	})
	cancelled := tsEvent("test:summary", func(e *tsNativeEvent) { e.EntryFile, e.File = "", ""; e.Counts = &tsSummaryCounts{Tests: 1, Passed: 1, Cancelled: 1, Success: true} })
	cases := []struct {
		name string
		mut  func([]tsNativeEvent) []tsNativeEvent
		want string
	}{
		{"missing-declared", drop("test:enqueue"), "MISSING_TEST"},
		{"missing-complete", drop("test:complete"), "INCOMPLETE_EVENTS"},
		{"missing-sentinel", drop(TypeScriptReporterSentinel), "INCOMPLETE_EVENTS"},
		{"missing-summary", drop("test:summary"), "INCOMPLETE_EVENTS"},
		{"unknown-native-identity", func(s []tsNativeEvent) []tsNativeEvent { return append(s, unknownTest) }, "MISSING_TEST"},
		{"skipped-case", replace("test:complete", skipFlag), "UNSUPPORTED_FEATURE"},
		{"summary-skip-count", replace("test:summary", badSummary), "UNSUPPORTED_FEATURE"},
		{"summary-cancel-count", replace("test:summary", cancelled), "INCOMPLETE_EVENTS"},
		{"unwitnessed-failure", replace("test:complete", failNoWitness), "UNEXPECTED_FAILURE"},
		{"witness-site-mismatch", func(s []tsNativeEvent) []tsNativeEvent { return replace("test:diagnostic", wrongSite)(drop("test:pass")(s)) }, "ASSERTION_MISMATCH"},
		{"witness-wrong-class", func(s []tsNativeEvent) []tsNativeEvent { return replace("test:diagnostic", wrongClass)(drop("test:pass")(s)) }, "UNEXPECTED_FAILURE"},
		{"unknown-witness-assertion", func(s []tsNativeEvent) []tsNativeEvent { return append(s, unknownAssertion) }, "ASSERTION_MISMATCH"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := reconcileTypeScriptStream(tc.mut(tsLegalPassStream()), tsUnitSuite(), tsUnitEntries(), tsUnitMaps())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("violation %s must fail with %s: err=%v rec=%+v", tc.name, tc.want, err, rec)
			}
		})
	}
}

// TestReconcileTypeScriptStreamPreservesOwnerIdentity freezes the
// unforgeable ownership rule: a native diagnostic attempting to smuggle
// owner identity fields never reassigns them, and unknown diagnostic
// payloads stay bounded diagnostics.
func TestReconcileTypeScriptStreamPreservesOwnerIdentity(t *testing.T) {
	stream := append(tsLegalPassStream(), tsEvent("test:diagnostic", func(e *tsNativeEvent) {
		e.Message = `{"schema":"machinery.tdd.event/v1","sequence":999,"design":"evil","milestone":"M9","suite":"evil-suite","kind":"suite-end"}`
	}))
	rec, err := reconcileTypeScriptStream(stream, tsUnitSuite(), tsUnitEntries(), tsUnitMaps())
	if err != nil {
		t.Fatalf("forged diagnostic broke reconciliation: %v", err)
	}
	for _, e := range rec.Events {
		if e.Suite != "unit-ts" || e.Design != "" || e.Milestone != "" || e.Sequence > int64(len(rec.Events)) {
			t.Fatalf("native payload reassigned owner identity: %+v", e)
		}
	}
}

// TestValidateTypeScriptSuite freezes the static source binding: node-native
// identities bound to declared files, the frozen helper identity, typed
// check call sites at the exact frozen lines, and duplicate rejection.
func TestValidateTypeScriptSuite(t *testing.T) {
	frozen := map[string][]byte{"conformance.ts": tsConformanceFixtureBytes(t)}
	if err := validateTypeScriptSuite(tsUnitSuite(), frozen); err == nil {
		t.Fatal("RED stub must not validate suites yet")
	}
	suite := &tdd.Suite{
		ID: "unit-ts", Adapter: AdapterNodeTestTS, Root: ".", Files: []string{"conformance.ts"},
		Tests: []tdd.Test{{
			ID: "witness", Source: "conformance.ts",
			Native:     tdd.NativeID{Source: "conformance.ts", Path: []string{"conformance witness executes native assertion"}, Line: 4, Column: 1},
			Assertions: []tdd.Assertion{{ID: "conformance/witness", Source: "conformance.ts", Line: 5, Helper: tdd.AssertionHelperV1}},
		}},
	}
	if err := validateTypeScriptSuite(suite, frozen); err != nil {
		t.Fatalf("valid frozen suite rejected: %v", err)
	}
	badHelper := *suite
	badHelper.Tests = []tdd.Test{suite.Tests[0]}
	badHelper.Tests[0].Assertions = []tdd.Assertion{{ID: "a", Source: "conformance.ts", Line: 5, Helper: "chai/v1"}}
	if err := validateTypeScriptSuite(&badHelper, frozen); err == nil || !strings.Contains(err.Error(), "UNSUPPORTED_FEATURE") {
		t.Fatalf("foreign helper accepted: %v", err)
	}
	badLine := *suite
	badLine.Tests = []tdd.Test{suite.Tests[0]}
	badLine.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/witness", Source: "conformance.ts", Line: 7, Helper: tdd.AssertionHelperV1}}
	if err := validateTypeScriptSuite(&badLine, frozen); err == nil {
		t.Fatal("assertion line without a typed check call accepted")
	}
	badSource := *suite
	badSource.Tests = []tdd.Test{suite.Tests[0]}
	badSource.Tests[0].Assertions = []tdd.Assertion{{ID: "conformance/witness", Source: "other.ts", Line: 5, Helper: tdd.AssertionHelperV1}}
	if err := validateTypeScriptSuite(&badSource, frozen); err == nil {
		t.Fatal("assertion bound to a foreign source accepted")
	}
	foreignNative := *suite
	foreignNative.Tests = []tdd.Test{suite.Tests[0]}
	foreignNative.Tests[0].Native = tdd.NativeID{Package: "pkg", Test: "TestX"}
	if err := validateTypeScriptSuite(&foreignNative, frozen); err == nil {
		t.Fatal("non-node native identity accepted")
	}
	unknownFile := *suite
	unknownFile.Tests = []tdd.Test{suite.Tests[0]}
	unknownFile.Tests[0].Native.Source = "ghost.ts"
	if err := validateTypeScriptSuite(&unknownFile, frozen); err == nil {
		t.Fatal("identity bound to an undeclared file accepted")
	}
	dup := *suite
	dup.Tests = append(append([]tdd.Test{}, suite.Tests...), suite.Tests[0])
	if err := validateTypeScriptSuite(&dup, frozen); err == nil {
		t.Fatal("duplicate native identity accepted")
	}
}

// TestBuildTypeScriptRunEnv freezes the closed environment: private roots,
// fixed locale/timezone, minimal PATH, declared suite variables only, and
// loader/test-context/custody injection never inherited.
func TestBuildTypeScriptRunEnv(t *testing.T) {
	env, err := buildTypeScriptRunEnv("/prep", "/opt/node/bin", []tdd.EnvironmentVar{
		{Name: "SUITE_FLAG", Value: "1"}, {Name: "NODE_OPTIONS", Value: "--import=evil"},
	})
	if err != nil {
		t.Fatalf("closed environment rejected: %v", err)
	}
	joined := "\x00" + strings.Join(env, "\x00") + "\x00"
	for _, want := range []string{"HOME=/prep/home", "TMPDIR=/prep/tmp", "TZ=UTC", "NO_COLOR=1", "PATH=/opt/node/bin:/usr/bin:/bin", "SUITE_FLAG=1"} {
		if !strings.Contains(joined, "\x00"+want+"\x00") {
			t.Fatalf("closed environment lacks %q: %v", want, env)
		}
	}
	if strings.Contains(joined, "NODE_OPTIONS") || strings.Contains(joined, "NODE_TEST_CONTEXT") || strings.Contains(joined, "MACHINERY_PROCESSSCOPE") {
		t.Fatalf("loader/test-context/custody injection inherited: %v", env)
	}
}
