package main

// RED contract for MAC-bz1y: the four-language native assurance conformance
// catalog of the required integration lane. These black-box subjects drive
// the real in-process lane runner against fixture roots seeded with the
// frozen catalog fixtures; none replace a native runtime. The failing
// subjects define the closed catalog behavior; the two controls pin the
// frozen fixture bytes and the preserved v1 semantics for foreign modules.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type assuranceAdapterReceipt struct {
	ID     string `json:"id"`
	Suites int    `json:"suites"`
	Status string `json:"status"`
}

type assuranceLaneReport struct {
	Version   int            `json:"version"`
	Status    string         `json:"status"`
	Suites    []suiteReceipt `json:"suites"`
	Assurance *struct {
		Schema   string                    `json:"schema"`
		Status   string                    `json:"status"`
		Platform string                    `json:"platform"`
		Adapters []assuranceAdapterReceipt `json:"adapters"`
		Runtimes []runtimeReceipt          `json:"runtimes"`
		Suites   []suiteReceipt            `json:"suites"`
	} `json:"assurance"`
}

var assuranceAdapterIDs = []string{"go-testing/v1", "node-test-typescript/v1", "python-unittest/v1", "elixir-exunit/v1"}

func assuranceInvoke(t *testing.T, root string) (int, string, assuranceLaneReport) {
	t.Helper()
	work := t.TempDir()
	reportPath := filepath.Join(work, "report.json")
	args := []string{"--root", root, "--lane", "required", "--report", reportPath, "--work-dir", filepath.Join(work, "owned"), "--cache-dir", filepath.Join(work, "cache")}
	var out, errout bytes.Buffer
	status := run(args, &out, &errout)
	var got assuranceLaneReport
	if b, err := os.ReadFile(reportPath); err == nil {
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("malformed report: %v", err)
		}
	}
	return status, out.String() + errout.String(), got
}

func assuranceRequireFailure(t *testing.T, root, diagnostic string) {
	t.Helper()
	rc, out, report := assuranceInvoke(t, root)
	if rc == 0 || report.Status == "passed" || !strings.Contains(strings.ToLower(out), strings.ToLower(diagnostic)) {
		t.Fatalf("require rejection diagnostic %q, status=%d report=%+v output=%s", diagnostic, rc, report, out)
	}
}

// assuranceSeed copies the frozen catalog fixtures from the repository into a
// fixture lane root: the closed schema, the runtime pins, every assurance
// fragment and the complete probe fixture tree.
func assuranceSeed(t *testing.T, root string) {
	t.Helper()
	source := filepath.Join(laneRepo(t), "testdata", "integration-lanes")
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	sawFragment := false
	for _, entry := range entries {
		name := entry.Name()
		assured := name == "assurance.schema.json" || name == "assurance-runtime-pins.json" ||
			(strings.HasPrefix(name, "assurance-") && strings.HasSuffix(name, ".json"))
		if !assured {
			continue
		}
		sawFragment = sawFragment || strings.HasPrefix(name, "assurance-")
		b, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		laneWrite(t, root, "testdata/integration-lanes/"+name, string(b))
	}
	if !sawFragment {
		t.Fatal("repository does not carry a frozen assurance fragment")
	}
	err = filepath.WalkDir(filepath.Join(source, "assurance-probes"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		laneWrite(t, root, "testdata/integration-lanes/"+filepath.ToSlash(rel), string(b))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// MAC-wi2u/MAC-avfp compatible extension (approved alongside their
	// REDs): downstream adapter fragments declare native-conformance sources
	// inside the adapters' exclusively owned asset directories OUTSIDE
	// testdata/integration-lanes. Seed every declared fragment source from
	// the repository so the closed catalog validates identically in fixture
	// roots; frozen v1 and probe seeding semantics are unchanged and the
	// copy is idempotent for sources already seeded above.
	for _, name := range entries {
		if !strings.HasPrefix(name.Name(), "assurance-") || !strings.HasSuffix(name.Name(), ".json") {
			continue
		}
		var fragment struct {
			Suites []struct {
				Sources []string `json:"source_files"`
			} `json:"suites"`
		}
		b, err := os.ReadFile(filepath.Join(source, name.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &fragment); err != nil {
			t.Fatal(err)
		}
		for _, suite := range fragment.Suites {
			for _, rel := range suite.Sources {
				if strings.HasPrefix(rel, "testdata/integration-lanes/") {
					continue // already seeded above with the frozen catalog
				}
				body, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Dir(source)), filepath.FromSlash(rel)))
				if err != nil {
					t.Fatal(err)
				}
				laneWrite(t, root, rel, string(body))
			}
		}
	}
}

func assuranceFixture(t *testing.T) string {
	t.Helper()
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) {}`)
	assuranceSeed(t, root)
	return root
}

func assuranceFragment(t *testing.T, root string) map[string]any {
	t.Helper()
	return assuranceDecode(t, root, "testdata/integration-lanes/assurance-probes.json")
}

func assuranceDecode(t *testing.T, root, rel string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(b, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func assuranceEncode(t *testing.T, root, rel string, value any) {
	t.Helper()
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	laneWrite(t, root, rel, string(b)+"\n")
}

func assuranceTamper(t *testing.T, root, rel, old, replacement string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), old) {
		t.Fatalf("frozen probe source %s lacks tamper anchor %q", rel, old)
	}
	laneWrite(t, root, rel, strings.Replace(string(b), old, replacement, 1))
}

// TestAssuranceRequiredLaneCarriesClosedCatalog is a passing control: the
// repository carries the frozen catalog bytes with the exact first-release
// adapter versions, the git gate runtime pin and both native platforms.
func TestAssuranceRequiredLaneCarriesClosedCatalog(t *testing.T) {
	dir := filepath.Join(laneRepo(t), "testdata", "integration-lanes")
	var pins struct {
		Version   int      `json:"version"`
		Platforms []string `json:"platforms"`
		Adapters  map[string]struct {
			Runtime    string `json:"runtime"`
			Version    string `json:"version"`
			Typescript string `json:"typescript"`
			OTP        string `json:"otp"`
			Erts       string `json:"erts"`
			Mix        string `json:"mix"`
		} `json:"adapters"`
		Git struct {
			Version string `json:"version"`
		} `json:"git"`
	}
	b, err := os.ReadFile(filepath.Join(dir, "assurance-runtime-pins.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &pins); err != nil {
		t.Fatal(err)
	}
	if pins.Version != 1 || len(pins.Platforms) != 2 || pins.Platforms[0] != "darwin/arm64" || pins.Platforms[1] != "linux/amd64" {
		t.Fatalf("platforms are not the closed native pair: %+v", pins)
	}
	want := map[string][5]string{
		"go-testing/v1":           {"go", "1.27.1", "", "", ""},
		"node-test-typescript/v1": {"node", "26.8.1", "7.0.2", "", ""},
		"python-unittest/v1":      {"python", "3.14.7", "", "", ""},
		"elixir-exunit/v1":        {"elixir", "1.20.4", "", "17.0.6", "1.20.4"},
	}
	if len(pins.Adapters) != len(want) {
		t.Fatalf("adapter pin set is not the closed four-language union: %+v", pins.Adapters)
	}
	for id, values := range want {
		got, ok := pins.Adapters[id]
		if !ok || got.Runtime != values[0] || got.Version != values[1] || got.Typescript != values[2] || got.Erts != values[3] || got.Mix != values[4] {
			t.Fatalf("adapter pin %s is not the exact first-release catalog: %+v", id, got)
		}
	}
	if pins.Adapters["elixir-exunit/v1"].OTP != "29" || pins.Git.Version != "2.55.0" {
		t.Fatalf("otp/git gate pins are wrong: %+v", pins)
	}
	schema, err := os.ReadFile(filepath.Join(dir, "assurance.schema.json"))
	if err != nil || !bytes.Contains(schema, []byte("machinery.assurance.lane/v1")) {
		t.Fatalf("closed assurance schema missing: %v", err)
	}
	var fragment struct {
		Schema   string `json:"schema"`
		Fragment string `json:"fragment"`
		Owner    string `json:"owner"`
		Suites   []struct {
			ID          string   `json:"id"`
			Adapter     string   `json:"adapter"`
			Tests       []string `json:"tests"`
			SourceFiles []string `json:"source_files"`
		} `json:"suites"`
	}
	b, err = os.ReadFile(filepath.Join(dir, "assurance-probes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &fragment); err != nil {
		t.Fatal(err)
	}
	if fragment.Schema != "machinery.assurance.lane/v1" || fragment.Fragment != "assurance-probes" || fragment.Owner != "MAC-bz1y" {
		t.Fatalf("catalog fragment identity is wrong: %+v", fragment)
	}
	adapters := map[string]bool{}
	for _, suite := range fragment.Suites {
		adapters[suite.Adapter] = true
		for _, source := range suite.SourceFiles {
			if _, err := os.Stat(filepath.Join(laneRepo(t), filepath.FromSlash(source))); err != nil {
				t.Fatalf("probe source %s missing: %v", source, err)
			}
		}
	}
	for _, id := range assuranceAdapterIDs {
		if !adapters[id] {
			t.Fatalf("catalog fragment does not cover adapter %s", id)
		}
	}
}

// TestAssuranceCatalogPreservesV1LaneForForeignModules is a passing control:
// a foreign-module root without assurance files keeps exact v1 semantics and
// no assurance section in its report.
func TestAssuranceCatalogPreservesV1LaneForForeignModules(t *testing.T) {
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) {}`)
	rc, out, report := assuranceInvoke(t, root)
	if rc != 0 || report.Status != "passed" || len(report.Suites) != 1 || report.Suites[0].ID != "fixture" {
		t.Fatalf("v1 lane semantics changed for foreign modules: %d %s %+v", rc, out, report)
	}
	if report.Assurance != nil {
		t.Fatalf("foreign lane without assurance files must not report assurance: %+v", report.Assurance)
	}
}

func TestAssuranceCatalogExecutesFourLanguageProbesNatively(t *testing.T) {
	root := assuranceFixture(t)
	rc, out, report := assuranceInvoke(t, root)
	if rc != 0 || report.Status != "passed" {
		t.Fatalf("four-language assurance lane must pass natively: status=%d output=%s", rc, out)
	}
	if report.Assurance == nil || report.Assurance.Status != "passed" || report.Assurance.Schema != "machinery.assurance.lane/v1" {
		t.Fatalf("assurance section missing from passed report: %+v", report.Assurance)
	}
	if report.Assurance.Platform != "darwin/arm64" && report.Assurance.Platform != "linux/amd64" {
		t.Fatalf("assurance platform not a pinned native platform: %q", report.Assurance.Platform)
	}
	if len(report.Assurance.Adapters) != 4 {
		t.Fatalf("assurance adapters are not the closed four-language union: %+v", report.Assurance.Adapters)
	}
	for _, adapter := range report.Assurance.Adapters {
		if adapter.Status != "passed" || adapter.Suites != 1 {
			t.Fatalf("adapter %s not accounted exactly: %+v", adapter.ID, adapter)
		}
	}
	// MAC-wi2u/MAC-avfp RED supersession (justified, approved with each
	// RED): the exact count 4 pinned the probe-only catalog state. The
	// closed catalog contract (assurance.CONTRACT.md) declares the
	// downstream adapter stories (MAC-wi2u Go, MAC-avfp TypeScript,
	// MAC-imtz Python, MAC-8yai Elixir) each add a NEW named
	// native-conformance fragment, so the union count is >= 4 with all four
	// frozen probe suites still present and exactly accounted below. The
	// four probe receipts and the per-adapter probe accounting stay
	// byte-exact.
	if len(report.Suites) != 1 || len(report.Assurance.Suites) < 4 {
		t.Fatalf("merged union must keep the v1 suite and add four assurance suites: %+v", report)
	}
	receipts := map[string]suiteReceipt{}
	for _, suite := range report.Assurance.Suites {
		receipts[suite.ID] = suite
	}
	markers := map[string]string{
		"assurance-probe-elixir": "Result: 1 passed",
		"assurance-probe-go":     "TestAssuranceRuntimeProbe",
		"assurance-probe-node":   "ok 1 - assurance runtime probe executes a native TypeScript closure",
		"assurance-probe-python": "probe_test.AssuranceRuntimeProbe.test_assurance_runtime_probe",
	}
	for id, marker := range markers {
		suite, ok := receipts[id]
		if !ok {
			t.Fatalf("assurance suite %s missing: %+v", id, report.Assurance.Suites)
		}
		if suite.Selected != 1 || suite.Started != 1 || suite.Passed != 1 || suite.Failed != 0 || suite.Skipped != 0 {
			t.Fatalf("suite %s not accounted exactly: %+v", id, suite)
		}
		if len(suite.Tests) != 1 || suite.Tests[0].Status != "passed" || suite.Events == "" || len(suite.EventsSHA) != 64 {
			t.Fatalf("suite %s lacks retained native evidence: %+v", id, suite)
		}
		events, err := os.ReadFile(suite.Events)
		if err != nil || !strings.Contains(string(events), marker) {
			t.Fatalf("suite %s events are not its native stream (%v): %s", id, err, events)
		}
		if fmt.Sprintf("%x", sha256.Sum256(events)) != suite.EventsSHA {
			t.Fatalf("suite %s report hash does not bind retained events", id)
		}
	}
	identities := map[string]string{}
	for _, runtime := range report.Assurance.Runtimes {
		identities[runtime.ID] = runtime.Identity
	}
	for id, want := range map[string]string{
		"go":     "go1.27.1",
		"node":   "v26.8.1",
		"tsc":    "Version 7.0.2",
		"python": "Python 3.14.7",
		"elixir": "Elixir 1.20.4",
		"mix":    "Mix 1.20.4",
	} {
		if !strings.Contains(identities[id], want) {
			t.Fatalf("runtime %s identity does not match the exact pin (%q vs %q)", id, identities[id], want)
		}
	}
	if !strings.Contains(identities["elixir"], "erts-17.0.6") {
		t.Fatalf("elixir identity does not bind the pinned ERTS closure: %q", identities["elixir"])
	}
}

func TestAssuranceCatalogIsMandatoryInMachineryLane(t *testing.T) {
	root, _ := laneFixture(t, `func TestPilot(t *testing.T) {}`)
	laneWrite(t, root, "go.mod", "module github.com/RamXX/machinery\n\ngo 1.27.0\n")
	assuranceRequireFailure(t, root, "assurance catalog absent")
}

func TestAssuranceCatalogRejectsIncompleteLanguageUnion(t *testing.T) {
	// MAC-8yai justified supersession (disclosed with its GREEN): the
	// dropped-probe subject moves from elixir-exunit/v1 to
	// python-unittest/v1. The frozen probe fragment now coexists with the
	// MAC-8yai native-conformance fragment, so dropping only the elixir
	// probe suite no longer makes the adapter absent from the union; the
	// catalog still fails closed on it (the orphaned probe-source check),
	// but with a different diagnostic. python-unittest/v1 owns no
	// conformance inventory yet, so dropping its probe suite keeps failing
	// exactly at the closed-union boundary with the original diagnostic.
	// The test's intent - removing one adapter's suites must fail the
	// closed union - is unchanged.
	//
	// MAC-ui8a union reconciliation (disclosed with the MAC-8yai merge):
	// in the integrated four-adapter union every closed adapter owns a
	// native-conformance fragment alongside its frozen probe suite, so
	// removing one adapter's suites now means removing them from every
	// fragment (its probe suite and its conformance fragment). Dropping
	// only the probe suite would leave the adapter present through its
	// conformance fragment and fail later on the orphaned probe-source
	// check with a different diagnostic. python-unittest/v1 keeps the
	// subject role and the catalog still fails exactly at the closed-union
	// boundary with the original diagnostic.
	root := assuranceFixture(t)
	fragment := assuranceFragment(t, root)
	var kept []any
	for _, suite := range fragment["suites"].([]any) {
		if suite.(map[string]any)["adapter"] != "python-unittest/v1" {
			kept = append(kept, suite)
		}
	}
	fragment["suites"] = kept
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-probes.json", fragment)
	if err := os.Remove(filepath.Join(root, "testdata", "integration-lanes", "assurance-python.json")); err != nil {
		t.Fatal(err)
	}
	assuranceRequireFailure(t, root, "assurance union incomplete")
}

func TestAssuranceCatalogRejectsDuplicateNativeIdentity(t *testing.T) {
	root := assuranceFixture(t)
	fragment := assuranceFragment(t, root)
	var goSuite map[string]any
	for _, suite := range fragment["suites"].([]any) {
		if suite.(map[string]any)["adapter"] == "go-testing/v1" {
			goSuite = suite.(map[string]any)
		}
	}
	goSuite["id"] = "assurance-probe-go-duplicate"
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-duplicate.json", map[string]any{
		"schema": "machinery.assurance.lane/v1", "fragment": "assurance-duplicate", "owner": "MAC-bz1y",
		"suites": []any{goSuite},
	})
	assuranceRequireFailure(t, root, "duplicate assurance")
}

func TestAssuranceCatalogRejectsUnregisteredProbeCase(t *testing.T) {
	root := assuranceFixture(t)
	fragment := assuranceFragment(t, root)
	for _, suite := range fragment["suites"].([]any) {
		if suite.(map[string]any)["adapter"] == "go-testing/v1" {
			suite.(map[string]any)["tests"] = []string{"TestAssuranceRuntimeProbe", "TestUnknownExtra"}
		}
	}
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-probes.json", fragment)
	assuranceRequireFailure(t, root, "probe inventory")
}

func TestAssuranceCatalogRejectsEmptyCaseInventory(t *testing.T) {
	root := assuranceFixture(t)
	fragment := assuranceFragment(t, root)
	for _, suite := range fragment["suites"].([]any) {
		if suite.(map[string]any)["adapter"] == "go-testing/v1" {
			suite.(map[string]any)["tests"] = []string{}
		}
	}
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-probes.json", fragment)
	assuranceRequireFailure(t, root, "case inventory")
}

func TestAssuranceCatalogRejectsPinTampering(t *testing.T) {
	root := assuranceFixture(t)
	pins := assuranceDecode(t, root, "testdata/integration-lanes/assurance-runtime-pins.json")
	pins["adapters"].(map[string]any)["elixir-exunit/v1"].(map[string]any)["version"] = "1.19.0"
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-runtime-pins.json", pins)
	assuranceRequireFailure(t, root, "assurance pin")
}

func TestAssuranceCatalogRejectsRuntimeVersionMismatch(t *testing.T) {
	root := assuranceFixture(t)
	shim := t.TempDir()
	path := filepath.Join(shim, "node")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho v25.9.9\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	assuranceRequireFailure(t, root, "unsupported node runtime")
}

func TestAssuranceCatalogRejectsSchemaDrift(t *testing.T) {
	root := assuranceFixture(t)
	b, err := os.ReadFile(filepath.Join(root, "testdata/integration-lanes/assurance.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	laneWrite(t, root, "testdata/integration-lanes/assurance.schema.json", string(b)+" \n")
	assuranceRequireFailure(t, root, "assurance schema")
}

func TestAssuranceCatalogRejectsUnknownAdapter(t *testing.T) {
	root := assuranceFixture(t)
	fragment := assuranceFragment(t, root)
	for _, suite := range fragment["suites"].([]any) {
		if suite.(map[string]any)["adapter"] == "go-testing/v1" {
			suite.(map[string]any)["adapter"] = "rust-cargo/v1"
		}
	}
	assuranceEncode(t, root, "testdata/integration-lanes/assurance-probes.json", fragment)
	assuranceRequireFailure(t, root, "assurance adapter")
}

func TestAssuranceCatalogRejectsOrphanProbeSource(t *testing.T) {
	root := assuranceFixture(t)
	laneWrite(t, root, "testdata/integration-lanes/assurance-probes/go/orphan_test.go", "package probe\n\nimport \"testing\"\n\nfunc TestOrphan(t *testing.T) { t.Fatal(\"must not vanish\") }\n")
	assuranceRequireFailure(t, root, "unregistered assurance source")
}

// Each tamper edits the frozen probe bytes inside the fixture root so the
// native language runtime itself reports the skip or the failure; the lane
// must fail closed on the real native stream, never on a stubbed verdict.
func TestAssuranceCatalogNativeSkipCannotBecomeSuccess(t *testing.T) {
	for _, tc := range []struct{ language, rel, old, new string }{
		{"go", "testdata/integration-lanes/assurance-probes/go/assurance_probe_test.go",
			"	if 6*7 != 42 {", "	t.Skip(\"required assurance probe must not skip\")\n	if 6*7 != 42 {"},
		{"node", "testdata/integration-lanes/assurance-probes/node/test.assurance-probe.ts",
			"assert.equal(answer, 42);", "t.skip(\"required assurance probe must not skip\");"},
		{"python", "testdata/integration-lanes/assurance-probes/python/probe_test.py",
			"self.assertEqual(6 * 7, 42)", "self.skipTest(\"required assurance probe must not skip\")"},
		{"elixir", "testdata/integration-lanes/assurance-probes/elixir/test/probe_test.exs",
			"  test \"assurance runtime probe executes a native ExUnit closure\" do", "  @tag :skip\n  test \"assurance runtime probe executes a native ExUnit closure\" do"},
	} {
		t.Run(tc.language, func(t *testing.T) {
			root := assuranceFixture(t)
			assuranceTamper(t, root, tc.rel, tc.old, tc.new)
			assuranceRequireFailure(t, root, "skip")
		})
	}
}

func TestAssuranceCatalogNativeFailuresCannotBecomeSuccess(t *testing.T) {
	for _, tc := range []struct{ language, rel, old, new string }{
		{"go", "testdata/integration-lanes/assurance-probes/go/assurance_probe_test.go",
			"if 6*7 != 42 {", "if 6*7 != 43 {"},
		{"node", "testdata/integration-lanes/assurance-probes/node/test.assurance-probe.ts",
			"assert.equal(answer, 42);", "assert.equal(answer, 43);"},
		{"python", "testdata/integration-lanes/assurance-probes/python/probe_test.py",
			"self.assertEqual(6 * 7, 42)", "self.assertEqual(6 * 7, 43)"},
		{"elixir", "testdata/integration-lanes/assurance-probes/elixir/test/probe_test.exs",
			"assert 6 * 7 == 42", "assert 6 * 7 == 43"},
	} {
		t.Run(tc.language, func(t *testing.T) {
			root := assuranceFixture(t)
			assuranceTamper(t, root, tc.rel, tc.old, tc.new)
			assuranceRequireFailure(t, root, "fail")
		})
	}
}
