// Frozen RED/GREEN suite for the MAC-6h0s declaration producer: closed
// plan/milestone decoding, review projection, typed digests and finalized
// Validate reconciliation, driven by the closed corpus in
// testdata/assurance/schema-cases.json plus independent stdlib-only oracles
// for the exact projection/digest encodings.
package tdd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// ---- frozen base design tree ----

const (
	fixtureProjectID = "6f281c6e-0d2a-4cd4-9a1f-93a51d1e77aa"
	fixtureSrc       = "subject v1\nanchor-line\n"
	fixtureGoSource  = "package tests\n"
	fixtureBaseline  = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	fixtureSafe1     = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	fixtureUnsafe1   = "sha256:4444444444444444444444444444444444444444444444444444444444444444"
	fixtureSafe2     = "sha256:5555555555555555555555555555555555555555555555555555555555555555"
	fixtureClosure   = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

var fixtureDesignFiles = map[string]string{
	"BUILD.md": "# Build\n\n## Build plan\n\n**M1 - Alpha behavior.**\nStatus: open\nDoD: alpha-s1 covered; inv-owned preserved.\n",
	"domain.modelith.yaml": "kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n        preserves: [inv-owned]\n    invariants:\n      - id: inv-owned\ninvariants:\n  - id: inv-global\n",
	"machines/Alpha.machine.json": "{}",
	"machines/Alpha.oracle.md":    "# Alpha oracle\n\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| alpha-t1 | alpha-s1 | gate-x | refuses malformed input |\n",
	"machines/Alpha.matrix.md":    "# Alpha matrix\n\n| unit | kind | detail |\n| --- | --- | --- |\n| gate-x | guard | CLAUSES{clause-a} |\n",
	"src.txt":           fixtureSrc,
	"tests/alpha_test.go": fixtureGoSource,
}

// writeBaseDesign materializes the frozen design tree with fixed portable
// modes so payload digests are deterministic.
func writeBaseDesign(t *testing.T) string {
	t.Helper()
	design := t.TempDir()
	for name, content := range fixtureDesignFiles {
		path := filepath.Join(design, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir fixture: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	if err := os.Chmod(design, 0o755); err != nil {
		t.Fatalf("chmod design root: %v", err)
	}
	return design
}

func fixtureSrcDigest(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(fixtureSrc))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ---- frozen typed base documents ----

func ref(test string) TestRef {
	return TestRef{Design: ".", Milestone: "M1", Suite: "s1", Test: test}
}

func basePlan(srcDigest, reviewDigest string) Plan {
	tRef := func(id, category, disposition string) RuntimeObligation {
		tests := []TestRef{}
		if disposition == DispositionTest {
			tests = []TestRef{ref("t2")}
		}
		reason := ""
		if disposition == DispositionNotApplicable {
			reason = "single-user tool; no load envelope to demonstrate"
		}
		return RuntimeObligation{
			ID: id, Category: category, Owner: "platform",
			SourceRefs:  []SourceRef{{Path: "src.txt", Anchor: "anchor-line", Digest: srcDigest}},
			Disposition: disposition, Reason: reason,
			Review: Review{Reviewer: "rev-one", Rationale: "discovery reviewed", SubjectDigest: reviewDigest},
			Tests: tests,
		}
	}
	return Plan{
		Schema:     protocol.SchemaPlan,
		Design:     ".",
		Milestones: []PlanMilestone{{ID: "M1", Manifest: "assurance/milestones/M1.json"}},
		ProjectID:  fixtureProjectID,
		RuntimeObligations: []RuntimeObligation{
			tRef("rt-concurrency", CategoryConcurrency, DispositionTest),
			tRef("rt-delivery", CategoryDeliveryReplay, DispositionTest),
			tRef("rt-migration", CategoryMigration, DispositionTest),
			tRef("rt-restore", CategoryRestore, DispositionTest),
			tRef("rt-load", CategoryLoad, DispositionNotApplicable),
			tRef("rt-security", CategorySecurityBoundary, DispositionTest),
			tRef("rt-obs", CategoryObservability, DispositionTest),
		},
	}
}

func baseManifest(reviewDigest string) Manifest {
	newAssert := func(id string, line int64) Assertion {
		return Assertion{ID: id, Source: "tests/alpha_test.go", Line: line, Helper: protocol.AssertionHelperV1}
	}
	exp := func(test, outcome string, assertions ...string) Expectation {
		return Expectation{Test: ref(test), Outcome: outcome, Assertions: assertions}
	}
	rev := Review{Reviewer: "rev-one", Rationale: "variant reviewed", SubjectDigest: reviewDigest}
	return Manifest{
		Schema:              protocol.SchemaMilestone,
		ID:                  "M1",
		Revision:            1,
		Repository:          ".",
		ImplementationRoots: []string{"."},
		FrozenRoots:         []string{"tests"},
		SubjectEntries:      []SubjectEntry{{Path: "src.txt", Kind: "file"}},
		Suites: []Suite{{
			ID:      "s1",
			Adapter: protocol.AdapterGoTesting,
			Runtime: RuntimeRef{Profile: "go", Version: "go1.27.1", Platform: "darwin/arm64", Closure: fixtureClosure},
			Root:    ".", Files: []string{"tests/alpha_test.go"},
			Tests: []Test{
				{ID: "t1", Native: NativeID{Package: "github.com/example/tests", Test: "TestAlphaRefusal"}, Source: "tests/alpha_test.go", Role: RoleNegative, Assertions: []Assertion{newAssert("asr-1", 10)}},
				{ID: "t2", Native: NativeID{Package: "github.com/example/tests", Test: "TestAlphaPublish"}, Source: "tests/alpha_test.go", Role: RolePositive, Assertions: []Assertion{newAssert("asr-2", 20)}},
			},
			Environment:     []EnvironmentVar{{Name: "LC_ALL", Value: "C"}},
			DependencyRoots: []string{},
		}},
		Obligations: []Obligation{
			{Key: ObligationKey{Design: ".", Kind: KindOracleRow, Owner: "machines/Alpha.oracle.md", ID: "alpha-s1"}, Positive: []TestRef{ref("t2")}, Negative: []TestRef{ref("t1")}},
			{Key: ObligationKey{Design: ".", Kind: KindGuardClause, Owner: "Alpha", ID: "gate-x:clause-a"}, Negative: []TestRef{ref("t1")}},
			{Key: ObligationKey{Design: ".", Kind: KindInvariant, Owner: "Widget", ID: "inv-owned"}, Positive: []TestRef{ref("t2")}},
			{Key: ObligationKey{Design: ".", Kind: KindInvariant, Owner: "model", ID: "inv-global"}, Positive: []TestRef{ref("t2")}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-load"}, Positive: []TestRef{ref("t2")}},
		},
		Baseline: fixtureBaseline,
		Variants: []Variant{
			{ID: "safe-1", Kind: VariantSafeControl, Source: fixtureSafe1, Pair: "pair-1", TargetTests: []TestRef{ref("t1")}, Expected: []Expectation{exp("t1", OutcomePass, "asr-1"), exp("t2", OutcomeAssertionFail, "asr-2")}, Review: rev},
			{ID: "unsafe-1", Kind: VariantUnsafeChallenge, Source: fixtureUnsafe1, Pair: "pair-1", TargetTests: []TestRef{ref("t1")}, Expected: []Expectation{exp("t1", OutcomeAssertionFail, "asr-1"), exp("t2", OutcomeAssertionFail, "asr-2")}, Review: rev},
			{ID: "safe-2", Kind: VariantSafeControl, Source: fixtureSafe2, Pair: "", TargetTests: []TestRef{}, Expected: []Expectation{exp("t1", OutcomeAssertionFail, "asr-1"), exp("t2", OutcomePass, "asr-2")}, Review: rev},
		},
		RedExpectations: []Expectation{exp("t1", OutcomeAssertionFail, "asr-1"), exp("t2", OutcomeAssertionFail, "asr-2")},
		Checks:          []Check{{ID: "chk-1", Kind: CheckKindFormat, Profile: "go-format/v1", Inputs: []string{"tests/alpha_test.go"}}},
		RedControls: []RedControl{
			{Test: ref("t1"), Assertion: "asr-1", SafeVariant: "safe-1"},
			{Test: ref("t2"), Assertion: "asr-2", SafeVariant: "safe-2"},
		},
		Limits: Limits{
			WallMS: protocol.LimitWallDefaultMS, CleanupMS: protocol.LimitCleanupDefaultMS,
			StdoutBytes: protocol.LimitStdoutDefault, StderrBytes: protocol.LimitStderrDefault,
			EventBytes: protocol.LimitEventBytesDefault, EventCount: protocol.LimitEventCountDefault,
			Jobs: protocol.LimitJobsDefault, BundleBytes: protocol.LimitBundleDefault,
			Entries: protocol.LimitEntriesDefault, Depth: protocol.LimitDepthDefault,
		},
		Review: Review{Reviewer: "rev-one", Rationale: "milestone reviewed", SubjectDigest: reviewDigest},
	}
}

// ---- typed-record JSON renderers (authoring side; decode is the code under test) ----

func renderReview(r Review) map[string]any {
	return map[string]any{"reviewer": r.Reviewer, "rationale": r.Rationale, "subject_digest": r.SubjectDigest}
}

func renderTestRef(r TestRef) map[string]any {
	return map[string]any{"design": r.Design, "milestone": r.Milestone, "suite": r.Suite, "test": r.Test}
}

func renderRefs(rs []TestRef) []any {
	out := []any{}
	for _, r := range rs {
		out = append(out, renderTestRef(r))
	}
	return out
}

func renderPlan(p Plan) []byte {
	ms := []any{}
	for _, m := range p.Milestones {
		ms = append(ms, map[string]any{"id": m.ID, "manifest": m.Manifest})
	}
	ros := []any{}
	for _, ro := range p.RuntimeObligations {
		srs := []any{}
		for _, s := range ro.SourceRefs {
			srs = append(srs, map[string]any{"path": s.Path, "anchor": s.Anchor, "digest": s.Digest})
		}
		ros = append(ros, map[string]any{
			"id": ro.ID, "category": ro.Category, "owner": ro.Owner,
			"source_refs": srs, "disposition": ro.Disposition, "reason": ro.Reason,
			"review": renderReview(ro.Review), "tests": renderRefs(ro.Tests),
		})
	}
	doc := map[string]any{
		"schema": p.Schema, "design": p.Design, "milestones": ms,
		"project_id": p.ProjectID, "runtime_obligations": ros,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return b
}

func renderManifest(m Manifest) []byte {
	strs := func(xs []string) []any {
		out := []any{}
		for _, x := range xs {
			out = append(out, x)
		}
		return out
	}
	suites := []any{}
	for _, s := range m.Suites {
		tests := []any{}
		for _, tst := range s.Tests {
			as := []any{}
			for _, a := range tst.Assertions {
				as = append(as, map[string]any{"id": a.ID, "source": a.Source, "line": a.Line, "helper": a.Helper})
			}
			native := map[string]any{"package": tst.Native.Package, "test": tst.Native.Test}
			tests = append(tests, map[string]any{"id": tst.ID, "native": native, "source": tst.Source, "role": tst.Role, "assertions": as})
		}
		env := []any{}
		for _, e := range s.Environment {
			env = append(env, map[string]any{"name": e.Name, "value": e.Value})
		}
		suites = append(suites, map[string]any{
			"id": s.ID, "adapter": s.Adapter,
			"runtime": map[string]any{"profile": s.Runtime.Profile, "version": s.Runtime.Version, "platform": s.Runtime.Platform, "closure": s.Runtime.Closure},
			"root": s.Root, "files": strs(s.Files), "tests": tests, "environment": env,
			"dependency_roots": strs(s.DependencyRoots),
		})
	}
	obls := []any{}
	for _, o := range m.Obligations {
		obls = append(obls, map[string]any{
			"key":       map[string]any{"design": o.Key.Design, "kind": o.Key.Kind, "owner": o.Key.Owner, "id": o.Key.ID},
			"positive":  renderRefs(o.Positive),
			"negative":  renderRefs(o.Negative),
		})
	}
	exps := func(es []Expectation) []any {
		out := []any{}
		for _, e := range es {
			out = append(out, map[string]any{"test": renderTestRef(e.Test), "outcome": e.Outcome, "assertions": strs(e.Assertions)})
		}
		return out
	}
	variants := []any{}
	for _, v := range m.Variants {
		variants = append(variants, map[string]any{
			"id": v.ID, "kind": v.Kind, "source": v.Source, "pair": v.Pair,
			"target_tests": renderRefs(v.TargetTests), "expected": exps(v.Expected),
			"review": renderReview(v.Review),
		})
	}
	checks := []any{}
	for _, c := range m.Checks {
		checks = append(checks, map[string]any{"id": c.ID, "kind": c.Kind, "profile": c.Profile, "inputs": strs(c.Inputs)})
	}
	controls := []any{}
	for _, rc := range m.RedControls {
		controls = append(controls, map[string]any{"test": renderTestRef(rc.Test), "assertion": rc.Assertion, "safe_variant": rc.SafeVariant})
	}
	subjects := []any{}
	for _, se := range m.SubjectEntries {
		subjects = append(subjects, map[string]any{"path": se.Path, "kind": se.Kind})
	}
	var pred, baseline any
	if m.Predecessor != "" {
		pred = m.Predecessor
	}
	if m.Baseline != "" {
		baseline = m.Baseline
	}
	doc := map[string]any{
		"schema": m.Schema, "id": m.ID, "revision": m.Revision, "predecessor": pred,
		"repository": m.Repository,
		"implementation_roots": strs(m.ImplementationRoots), "frozen_roots": strs(m.FrozenRoots),
		"subject_entries": subjects, "suites": suites, "obligations": obls,
		"baseline": baseline, "variants": variants, "red_expectations": exps(m.RedExpectations),
		"checks": checks, "red_controls": controls,
		"limits": map[string]any{
			"wall_ms": m.Limits.WallMS, "cleanup_ms": m.Limits.CleanupMS,
			"stdout_bytes": m.Limits.StdoutBytes, "stderr_bytes": m.Limits.StderrBytes,
			"event_bytes": m.Limits.EventBytes, "event_count": m.Limits.EventCount,
			"jobs": m.Limits.Jobs, "bundle_bytes": m.Limits.BundleBytes,
			"entries": m.Limits.Entries, "depth": m.Limits.Depth,
		},
		"review": renderReview(m.Review),
	}
	b, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return b
}

// bindBaseDocuments writes the base tree's control documents with
// contract-bound digests computed over the actual written tree.
func bindBaseDocuments(t *testing.T, design string) (planBytes, manifestBytes []byte) {
	t.Helper()
	payload, err := DesignPayloadDigest(design)
	if err != nil {
		t.Fatalf("setup: design payload digest: %v", err)
	}
	planReview, err := ReviewSubjectDigest(ReviewProjectionPlan(basePlan(fixtureSrcDigest(t), "placeholder")), payload)
	if err != nil {
		t.Fatalf("setup: plan review digest: %v", err)
	}
	planBytes = renderPlan(basePlan(fixtureSrcDigest(t), planReview))
	manifestReview, err := ReviewSubjectDigest(ReviewProjectionManifest(baseManifest("placeholder")), payload)
	if err != nil {
		t.Fatalf("setup: manifest review digest: %v", err)
	}
	manifestBytes = renderManifest(baseManifest(manifestReview))
	return planBytes, manifestBytes
}

// ---- corpus mutation engine ----

type corpusOp struct {
	Op      string          `json:"op"`
	Path    string          `json:"path"`
	Key     string          `json:"key"`
	Doc     string          `json:"doc"`
	Value   json.RawMessage `json:"value"`
	Raw     string          `json:"raw"`
	Content string          `json:"content"`
	Append  string          `json:"append"`
}

type corpusCase struct {
	Name           string     `json:"name"`
	Target         string     `json:"target"`
	Ops            []corpusOp `json:"ops"`
	Valid          bool       `json:"valid"`
	ValidLoadDraft bool       `json:"valid_load_draft"`
	Expect         []string   `json:"expect"`
	Inventory      string     `json:"inventory"`
	LoadAll        bool       `json:"load_all"`
}

type corpusFile struct {
	Schema string       `json:"schema"`
	Cases  []corpusCase `json:"cases"`
}

// splitJSONPath parses "a.b[3].c" into navigation steps.
func splitJSONPath(path string) []string {
	if path == "" {
		return nil
	}
	var parts []string
	for _, seg := range strings.Split(path, ".") {
		for strings.Contains(seg, "]") {
			i := strings.Index(seg, "[")
			if i > 0 {
				parts = append(parts, seg[:i])
			}
			j := strings.Index(seg, "]")
			parts = append(parts, seg[i+1:j])
			seg = seg[j+1:]
		}
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	return parts
}

// containerAt walks to the parent container of the final segment.
func containerAt(root any, path string) (any, string, error) {
	parts := splitJSONPath(path)
	if len(parts) == 0 {
		return root, "", nil
	}
	cur := root
	for _, p := range parts[:len(parts)-1] {
		switch node := cur.(type) {
		case map[string]any:
			next, ok := node[p]
			if !ok {
				return nil, "", fmt.Errorf("missing path segment %q", p)
			}
			cur = next
		case []any:
			var idx int
			if _, err := fmt.Sscanf(p, "%d", &idx); err != nil || idx < 0 || idx >= len(node) {
				return nil, "", fmt.Errorf("bad array index %q", p)
			}
			cur = node[idx]
		default:
			return nil, "", fmt.Errorf("cannot descend into %T at %q", cur, p)
		}
	}
	return cur, parts[len(parts)-1], nil
}

func setPath(root any, path string, val any) error {
	holder, key, err := containerAt(root, path)
	if err != nil {
		return err
	}
	switch node := holder.(type) {
	case map[string]any:
		node[key] = val
	case []any:
		var idx int
		if _, err := fmt.Sscanf(key, "%d", &idx); err != nil || idx < 0 || idx >= len(node) {
			return fmt.Errorf("bad array index %q", key)
		}
		node[idx] = val
	default:
		return fmt.Errorf("cannot set inside %T", holder)
	}
	return nil
}

func applyCorpusOps(t *testing.T, doc []byte, ops []corpusOp) []byte {
	t.Helper()
	if len(ops) == 0 {
		return doc
	}
	// A raw op replaces the whole document; tree ops re-marshal; text ops
	// append raw bytes last so trailing-data corruption stays byte-exact.
	var tree any
	if err := json.Unmarshal(doc, &tree); err != nil {
		t.Fatalf("setup: base document does not parse: %v", err)
	}
	var textAppends []string
	for _, op := range ops {
		switch op.Op {
		case "raw":
			return []byte(op.Raw)
		case "text":
			textAppends = append(textAppends, op.Append)
		case "set":
			var val any
			if err := json.Unmarshal(op.Value, &val); err != nil {
				t.Fatalf("setup: op value: %v", err)
			}
			if err := setPath(tree, op.Path, val); err != nil {
				t.Fatalf("setup: set %s: %v", op.Path, err)
			}
		case "setraw":
			if err := setPath(tree, op.Path, json.RawMessage(op.Raw)); err != nil {
				t.Fatalf("setup: setraw %s: %v", op.Path, err)
			}
		case "remove":
			holder, key, err := containerAt(tree, op.Path)
			if err != nil {
				t.Fatalf("setup: remove %s: %v", op.Path, err)
			}
			m, ok := holder.(map[string]any)
			if !ok {
				t.Fatalf("setup: remove %s: not an object", op.Path)
			}
			delete(m, key)
		case "insert":
			var val any
			if err := json.Unmarshal(op.Value, &val); err != nil {
				t.Fatalf("setup: insert value: %v", err)
			}
			holder, _, err := containerAt(tree, op.Path)
			if err != nil {
				t.Fatalf("setup: insert %s: %v", op.Path, err)
			}
			m, ok := holder.(map[string]any)
			if !ok {
				t.Fatalf("setup: insert %s: not an object", op.Path)
			}
			m[op.Key] = val
		case "append":
			var val any
			if err := json.Unmarshal(op.Value, &val); err != nil {
				t.Fatalf("setup: append value: %v", err)
			}
			holder, _, err := containerAt(tree, op.Path)
			if err != nil {
				t.Fatalf("setup: append %s: %v", op.Path, err)
			}
			arr, ok := holder.([]any)
			if !ok {
				t.Fatalf("setup: append %s: not an array", op.Path)
			}
			_ = setPath(tree, op.Path, append(append([]any{}, arr...), val))
		case "filename", "file", "delfile":
			// handled by the driver
		default:
			t.Fatalf("setup: unknown op %q", op.Op)
		}
	}
	out, err := json.Marshal(tree)
	if err != nil {
		t.Fatalf("setup: remarshal: %v", err)
	}
	for _, app := range textAppends {
		out = append(out, app...)
	}
	return out
}

// ---- corpus driver ----

func baseInventory(kind string) Inventory {
	inv := Inventory{
		Obligations: []InventoryObligation{
			{Key: ObligationKey{Design: ".", Kind: KindOracleRow, Owner: "machines/Alpha.oracle.md", ID: "alpha-s1"}},
			{Key: ObligationKey{Design: ".", Kind: KindGuardClause, Owner: "Alpha", ID: "gate-x:clause-a"}},
			{Key: ObligationKey{Design: ".", Kind: KindInvariant, Owner: "Widget", ID: "inv-owned"}},
			{Key: ObligationKey{Design: ".", Kind: KindInvariant, Owner: "model", ID: "inv-global"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-concurrency"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-delivery"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-migration"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-restore"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-load"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-security"}},
			{Key: ObligationKey{Design: ".", Kind: KindRuntime, Owner: "platform", ID: "rt-obs"}},
		},
		Milestones: []MilestoneKey{{Design: ".", Milestone: "M1"}},
	}
	if kind == "no-milestones" {
		inv.Milestones = nil
	}
	return inv
}

func TestSchemaCases(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "assurance", "schema-cases.json"))
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var corpus corpusFile
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parse corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("empty corpus")
	}
	seen := map[string]bool{}
	for _, c := range corpus.Cases {
		if seen[c.Name] {
			t.Fatalf("duplicate corpus case name %q", c.Name)
		}
		seen[c.Name] = true
		t.Run(c.Name, func(t *testing.T) {
			design := writeBaseDesign(t)
			planBytes, manifestBytes := bindBaseDocuments(t, design)
			// doc selection per op (validate cases can mutate either document)
			planDoc, milestoneDoc := planBytes, manifestBytes
			filename := "M1.json"
			var fileAdds [][2]string
			for _, op := range c.Ops {
				switch op.Op {
				case "filename":
					filename = op.Value2()
				case "file":
					fileAdds = append(fileAdds, [2]string{op.Path, op.Content})
				}
			}
			var planOps, msOps []corpusOp
			for _, op := range c.Ops {
				if op.Op == "filename" || op.Op == "file" {
					continue
				}
				if c.Target == "validate" && op.Doc == "plan" {
					planOps = append(planOps, op)
				} else if c.Target == "plan" {
					planOps = append(planOps, op)
				} else {
					msOps = append(msOps, op)
				}
			}
			planDoc = applyCorpusOps(t, planDoc, planOps)
			milestoneDoc = applyCorpusOps(t, milestoneDoc, msOps)
			for _, name := range []string{protocol.ControlDirName + "/" + protocol.MilestonesDirName, protocol.ControlDirName} {
				if err := os.MkdirAll(filepath.Join(design, filepath.FromSlash(name)), 0o755); err != nil {
					t.Fatalf("mkdir control: %v", err)
				}
			}
			if err := os.WriteFile(filepath.Join(design, "assurance", "plan.json"), planDoc, 0o644); err != nil {
				t.Fatalf("write plan: %v", err)
			}
			msPath := filepath.Join(design, "assurance", "milestones", filename)
			if err := os.WriteFile(msPath, milestoneDoc, 0o644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}
			for _, fa := range fileAdds {
				if err := os.WriteFile(filepath.Join(design, filepath.FromSlash(fa[0])), []byte(fa[1]), 0o644); err != nil {
					t.Fatalf("write added file: %v", err)
				}
			}
			switch c.Target {
			case "plan":
				_, err := LoadPlan(design)
				assertCase(t, c, err)
			case "milestone":
				_, err := LoadManifest(msPath)
				if c.ValidLoadDraft {
					if err != nil {
						t.Fatalf("draft must load: %v", err)
					}
					return
				}
				assertCase(t, c, err)
			case "validate":
				plan, err := LoadPlan(design)
				if err != nil {
					assertCase(t, c, err)
					return
				}
				var manifests []Manifest
				if c.LoadAll {
					entries, rerr := os.ReadDir(filepath.Join(design, "assurance", "milestones"))
					if rerr != nil {
						t.Fatalf("read milestones: %v", rerr)
					}
					for _, e := range entries {
						m, lmErr := LoadManifest(filepath.Join(design, "assurance", "milestones", e.Name()))
						if lmErr != nil {
							assertCase(t, c, lmErr)
							return
						}
						manifests = append(manifests, m)
					}
				} else {
					for _, pm := range plan.Milestones {
						m, lmErr := LoadManifest(filepath.Join(design, filepath.FromSlash(pm.Manifest)))
						if lmErr != nil {
							assertCase(t, c, lmErr)
							return
						}
						manifests = append(manifests, m)
					}
				}
				assertCase(t, c, Validate(plan, manifests, baseInventory(c.Inventory)))
			default:
				t.Fatalf("unknown target %q", c.Target)
			}
		})
	}
}

// Value2 returns the filename-op operand carried in the value field.
func (op corpusOp) Value2() string {
	var s string
	if err := json.Unmarshal(op.Value, &s); err != nil {
		return ""
	}
	return s
}

func assertCase(t *testing.T, c corpusCase, err error) {
	t.Helper()
	if c.Valid {
		if err != nil {
			t.Fatalf("expected valid, got error: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected error containing %v, got nil", c.Expect)
	}
	for _, needle := range c.Expect {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("error %q does not contain %q", err.Error(), needle)
		}
	}
}

// ---- AC5: exact review projection with a hand-written canonical literal ----

func TestReviewProjectionExactness(t *testing.T) {
	digest := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	p := Plan{
		Schema: protocol.SchemaPlan, Design: ".", ProjectID: fixtureProjectID,
		Milestones: []PlanMilestone{{ID: "M1", Manifest: "assurance/milestones/M1.json"}},
		RuntimeObligations: []RuntimeObligation{{
			ID: "rt-concurrency", Category: CategoryConcurrency, Owner: "platform",
			SourceRefs:  []SourceRef{{Path: "src.txt", Anchor: "anchor-line", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}},
			Disposition: DispositionTest,
			Review:      Review{Reviewer: "rev-one", Rationale: "discovery reviewed", SubjectDigest: digest},
			Tests:       []TestRef{ref("t2")},
		}},
	}
	want := `{"design":".","milestones":[{"id":"M1","manifest":"assurance/milestones/M1.json"}],"project_id":"6f281c6e-0d2a-4cd4-9a1f-93a51d1e77aa","runtime_obligations":[{"category":"concurrency","disposition":"test","id":"rt-concurrency","owner":"platform","reason":"","review":null,"source_refs":[{"anchor":"anchor-line","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","path":"src.txt"}],"tests":[{"design":".","milestone":"M1","suite":"s1","test":"t2"}]}],"schema":"machinery.tdd.plan/v1"}`
	if got := string(ReviewProjectionPlan(p)); got != want {
		t.Fatalf("plan projection mismatch:\n got: %s\nwant: %s", got, want)
	}
	// Ignoring is positional ONLY: mutating any review field never changes C...
	p.RuntimeObligations[0].Review.Rationale = "changed"
	p.RuntimeObligations[0].Review.Reviewer = "other"
	if string(ReviewProjectionPlan(p)) != want {
		t.Fatal("review-field edits must not change the projection")
	}
	// ...while any non-review field does.
	p.RuntimeObligations[0].Reason = "review"
	if string(ReviewProjectionPlan(p)) == want {
		t.Fatal("a non-review edit must change the projection")
	}
	// Manifest side: only top-level review and variants[i].review are nulled.
	m := baseManifest("placeholder")
	c1 := string(ReviewProjectionManifest(m))
	m.Review.Rationale = "changed"
	m.Variants[0].Review.Reviewer = "changed"
	if c2 := string(ReviewProjectionManifest(m)); c2 != c1 {
		t.Fatal("manifest review edits must not change the projection")
	}
	m.Variants[0].Expected[0].Assertions = []string{"asr-9"}
	if string(ReviewProjectionManifest(m)) == c1 {
		t.Fatal("variant content edits must change the projection")
	}
	if !strings.Contains(c1, `"review":null`) {
		t.Fatal("projection must null the review positions")
	}
	if !strings.Contains(c1, `"value":"C"`) {
		t.Fatal("non-review data named like review content must survive verbatim")
	}
}

// TestReviewSubjectDigestEncoding pins the domain||length||C||payload layout
// with an independent stdlib computation.
func TestReviewSubjectDigestEncoding(t *testing.T) {
	projection := []byte(`{"a":1,"b":"review"}`)
	payloadBytes := make([]byte, 32)
	for i := range payloadBytes {
		payloadBytes[i] = 0x11
	}
	h := sha256.New()
	h.Write([]byte("machinery.tdd.review/v1"))
	var lenBuf [8]byte
	binary.BigEndian.PutUint64(lenBuf[:], uint64(len(projection)))
	h.Write(lenBuf[:])
	h.Write(projection)
	h.Write(payloadBytes)
	want := "sha256:" + hex.EncodeToString(h.Sum(nil))
	got, err := ReviewSubjectDigest(projection, "sha256:"+hex.EncodeToString(payloadBytes))
	if err != nil {
		t.Fatalf("subject digest: %v", err)
	}
	if got != want {
		t.Fatalf("subject digest mismatch: got %s want %s", got, want)
	}
}

// TestDesignPayloadDigestContract pins the typed tree encoding with an
// independent encoder and proves the control namespace and .git exclusions.
func TestDesignPayloadDigestContract(t *testing.T) {
	encode := func(entries []struct {
		path string
		dir  bool
		perm uint32
		size uint64
		dig  []byte
	}) string {
		h := sha256.New()
		h.Write([]byte("machinery.tdd.tree/v1"))
		for _, e := range entries {
			if e.dir {
				h.Write([]byte{0})
			} else {
				h.Write([]byte{1})
			}
			var b8 [8]byte
			binary.BigEndian.PutUint64(b8[:], uint64(len(e.path)))
			h.Write(b8[:])
			h.Write([]byte(e.path))
			var b4 [4]byte
			binary.BigEndian.PutUint32(b4[:], e.perm)
			h.Write(b4[:])
			binary.BigEndian.PutUint64(b8[:], e.size)
			h.Write(b8[:])
			h.Write(e.dig)
			role := "design"
			binary.BigEndian.PutUint64(b8[:], uint64(len(role)))
			h.Write(b8[:])
			h.Write([]byte(role))
		}
		return "sha256:" + hex.EncodeToString(h.Sum(nil))
	}
	fileDig := func(content string) []byte {
		sum := sha256.Sum256([]byte(content))
		return sum[:]
	}
	design := t.TempDir()
	if err := os.Chmod(design, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"BUILD.md":            "x",
		"src.txt":             "y",
		"sub/nested.txt":      "z",
		"assurance/plan.json": "{}",
		".git/config":         "g",
	} {
		path := filepath.Join(design, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	expected := encode([]struct {
		path string
		dir  bool
		perm uint32
		size uint64
		dig  []byte
	}{
		{".", true, 0o755, 0, make([]byte, 32)},
		{".git", true, 0o755, 0, make([]byte, 32)},
		{".git/config", false, 0o644, 1, fileDig("g")},
		{"BUILD.md", false, 0o644, 1, fileDig("x")},
		{"src.txt", false, 0o644, 1, fileDig("y")},
		{"sub", true, 0o755, 0, make([]byte, 32)},
		{"sub/nested.txt", false, 0o644, 1, fileDig("z")},
	})
	got, err := DesignPayloadDigest(design)
	if err != nil {
		t.Fatalf("payload digest: %v", err)
	}
	if got != expected {
		t.Fatalf("payload digest mismatch:\n got %s\nwant %s", got, expected)
	}
	// control-namespace content never changes the payload digest
	if err := os.WriteFile(filepath.Join(design, "assurance", "extra.json"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := DesignPayloadDigest(design)
	if err != nil || again != got {
		t.Fatalf("control-namespace edit changed the payload digest: %v %s vs %s", err, again, got)
	}
	// a real source edit does
	if err := os.WriteFile(filepath.Join(design, "src.txt"), []byte("y2"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := DesignPayloadDigest(design)
	if err != nil || changed == got {
		t.Fatalf("source edit must change the payload digest: %v", err)
	}
	// symlinks and special entries fail closed
	if err := os.Symlink(filepath.Join(design, "BUILD.md"), filepath.Join(design, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := DesignPayloadDigest(design); err == nil {
		t.Fatal("symlink in the design payload must fail")
	}
}

// TestValidateCrossDesignResolutionAndCycle proves qualified cross-design
// test references resolve through provided manifests and design-reference
// cycles fail (acyclic parent/child refs).
func TestValidateCrossDesignResolutionAndCycle(t *testing.T) {
	childRef := TestRef{Design: "child", Milestone: "M1", Suite: "s1", Test: "t2"}
	rewrite := func(m Manifest, design string) Manifest {
		m.designID = design
		fix := func(rs []TestRef) []TestRef {
			out := append([]TestRef{}, rs...)
			for i := range out {
				out[i].Design = design
			}
			return out
		}
		for i := range m.Obligations {
			m.Obligations[i].Key.Design = design
			m.Obligations[i].Positive = fix(m.Obligations[i].Positive)
			m.Obligations[i].Negative = fix(m.Obligations[i].Negative)
		}
		for i := range m.RedExpectations {
			m.RedExpectations[i].Test.Design = design
		}
		for i := range m.Variants {
			m.Variants[i].TargetTests = fix(m.Variants[i].TargetTests)
			for j := range m.Variants[i].Expected {
				m.Variants[i].Expected[j].Test.Design = design
			}
		}
		for i := range m.RedControls {
			m.RedControls[i].Test.Design = design
		}
		return m
	}
	parent := rewrite(baseManifest("sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), ".")
	parent.relPath = "assurance/milestones/M1.json"
	child := rewrite(parent, "child")
	child.relPath = "child/assurance/milestones/M1.json"
	inv := baseInventory("")
	plan := basePlan(fixtureSrcDigest(t), "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	// A parent obligation may cite the child's test through a provided child manifest.
	parentCross := parent
	parentCross.Obligations = append([]Obligation{{Key: ObligationKey{Design: ".", Kind: KindInvariant, Owner: "model", ID: "inv-global"}, Positive: []TestRef{childRef}}}, parent.Obligations...)
	if err := Validate(plan, []Manifest{parentCross, child}, inv); err != nil {
		t.Fatalf("cross-design resolution must hold: %v", err)
	}
	// Without the child manifest the reference dangles.
	if err := Validate(plan, []Manifest{parentCross}, inv); err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
		t.Fatalf("dangling cross-design reference must fail with MISSING_TEST, got %v", err)
	}
	// A child citation back into the parent closes a design cycle.
	childCycle := child
	childCycle.Obligations = append([]Obligation{{Key: ObligationKey{Design: "child", Kind: KindInvariant, Owner: "model", ID: "inv-cite"}, Positive: []TestRef{{Design: ".", Milestone: "M1", Suite: "s1", Test: "t1"}}}}, child.Obligations...)
	err := Validate(plan, []Manifest{parentCross, childCycle}, inv)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("design reference cycle must fail, got %v", err)
	}
}
