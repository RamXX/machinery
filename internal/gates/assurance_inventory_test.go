// Frozen RED/GREEN suite for gates.AssuranceInventory: qualified obligation
// derivation from a held design snapshot (AC2) and real integration of valid
// and deliberately corrupted design+manifest trees through
// AssuranceInventory and tdd.Validate (AC7).
package gates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

func sha256Hex(t *testing.T, b []byte) string {
	t.Helper()
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

const (
	invFixtureProjectID = "6f281c6e-0d2a-4cd4-9a1f-93a51d1e77aa"
	invFixtureSrc       = "subject v1\nanchor-line\n"
)

var invFixtureFiles = map[string]string{
	"BUILD.md":                    "# Build\n\n## Build plan\n\n**M1 - Alpha behavior.**\nStatus: open\nDoD: alpha-s1 covered; inv-owned preserved.\n",
	"domain.modelith.yaml":        "kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n        preserves: [inv-owned]\n    invariants:\n      - id: inv-owned\ninvariants:\n  - id: inv-global\n",
	"machines/Alpha.machine.json": "{}",
	"machines/Alpha.oracle.md":    "# Alpha oracle\n\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| alpha-t1 | alpha-s1 | gate-x | refuses malformed input |\n",
	"machines/Alpha.matrix.md":    "# Alpha matrix\n\n| unit | kind | detail |\n| --- | --- | --- |\n| gate-x | guard | CLAUSES{clause-a} |\n",
	"src.txt":                     invFixtureSrc,
	"tests/alpha_test.go":         "package tests\n",
}

// writeInventoryDesign materializes the fixture design tree and returns its
// root; extra files are appended after the base set.
func writeInventoryDesign(t *testing.T, extra map[string]string) string {
	t.Helper()
	design := t.TempDir()
	files := map[string]string{}
	for k, v := range invFixtureFiles {
		files[k] = v
	}
	for k, v := range extra {
		files[k] = v
	}
	for name, content := range files {
		path := filepath.Join(design, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir fixture: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
	}
	if err := os.Chmod(design, 0o755); err != nil {
		t.Fatalf("chmod design: %v", err)
	}
	return design
}

// bindInventoryControls renders the plan and milestone documents with
// contract-bound digests, exactly as the corpus harness does, and returns
// the written tree root.
func bindInventoryControls(t *testing.T, design string) string {
	t.Helper()
	payload, err := tdd.DesignPayloadDigest(design)
	if err != nil {
		t.Fatalf("setup: payload digest: %v", err)
	}
	planReview, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionPlan(inventoryBasePlan(t, "placeholder")), payload)
	if err != nil {
		t.Fatalf("setup: plan review digest: %v", err)
	}
	planBytes := inventoryRenderPlan(t, inventoryBasePlan(t, planReview))
	manifestReview, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionManifest(inventoryBaseManifest(t, "placeholder")), payload)
	if err != nil {
		t.Fatalf("setup: manifest review digest: %v", err)
	}
	manifestBytes := inventoryRenderManifest(t, inventoryBaseManifest(t, manifestReview))
	if err := os.MkdirAll(filepath.Join(design, "assurance", "milestones"), 0o755); err != nil {
		t.Fatalf("mkdir controls: %v", err)
	}
	if err := os.WriteFile(filepath.Join(design, "assurance", "plan.json"), planBytes, 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	if err := os.WriteFile(filepath.Join(design, "assurance", "milestones", "M1.json"), manifestBytes, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	return design
}

func inventoryBasePlan(t *testing.T, reviewDigest string) tdd.Plan {
	t.Helper()
	sum := sha256Hex(t, []byte(invFixtureSrc))
	p := tdd.Plan{
		Schema:     protocol.SchemaPlan,
		Design:     ".",
		Milestones: []tdd.PlanMilestone{{ID: "M1", Manifest: "assurance/milestones/M1.json"}},
		ProjectID:  invFixtureProjectID,
	}
	p.RuntimeObligations = []tdd.RuntimeObligation{
		inventoryRO(t, "rt-concurrency", protocol.CategoryConcurrency, protocol.DispositionTest, sum, reviewDigest),
		inventoryRO(t, "rt-delivery", protocol.CategoryDeliveryReplay, protocol.DispositionTest, sum, reviewDigest),
		inventoryRO(t, "rt-migration", protocol.CategoryMigration, protocol.DispositionTest, sum, reviewDigest),
		inventoryRO(t, "rt-restore", protocol.CategoryRestore, protocol.DispositionTest, sum, reviewDigest),
		inventoryRO(t, "rt-load", protocol.CategoryLoad, protocol.DispositionNotApplicable, sum, reviewDigest),
		inventoryRO(t, "rt-security", protocol.CategorySecurityBoundary, protocol.DispositionTest, sum, reviewDigest),
		inventoryRO(t, "rt-obs", protocol.CategoryObservability, protocol.DispositionTest, sum, reviewDigest),
	}
	return p
}

func inventoryRO(t *testing.T, id, category, disposition, srcDigest, reviewDigest string) tdd.RuntimeObligation {
	t.Helper()
	tests := []tdd.TestRef{}
	if disposition == protocol.DispositionTest {
		tests = []tdd.TestRef{{Design: ".", Milestone: "M1", Suite: "s1", Test: "t2"}}
	}
	reason := ""
	if disposition == protocol.DispositionNotApplicable {
		reason = "single-user tool; no load envelope to demonstrate"
	}
	return tdd.RuntimeObligation{
		ID: id, Category: category, Owner: "platform",
		SourceRefs:  []tdd.SourceRef{{Path: "src.txt", Anchor: "anchor-line", Digest: srcDigest}},
		Disposition: disposition, Reason: reason,
		Review: tdd.Review{Reviewer: "rev-one", Rationale: "discovery reviewed", SubjectDigest: reviewDigest},
		Tests:  tests,
	}
}

func inventoryRenderPlan(t *testing.T, p tdd.Plan) []byte {
	t.Helper()
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
		tests := []any{}
		for _, tf := range ro.Tests {
			tests = append(tests, map[string]any{"design": tf.Design, "milestone": tf.Milestone, "suite": tf.Suite, "test": tf.Test})
		}
		ros = append(ros, map[string]any{
			"id": ro.ID, "category": ro.Category, "owner": ro.Owner,
			"source_refs": srs, "disposition": ro.Disposition, "reason": ro.Reason,
			"review": map[string]any{"reviewer": ro.Review.Reviewer, "rationale": ro.Review.Rationale, "subject_digest": ro.Review.SubjectDigest},
			"tests":  tests,
		})
	}
	b, err := jsonMarshal(map[string]any{
		"schema": p.Schema, "design": p.Design, "milestones": ms,
		"project_id": p.ProjectID, "runtime_obligations": ros,
	})
	if err != nil {
		t.Fatalf("render plan: %v", err)
	}
	return b
}

func inventoryBaseManifest(t *testing.T, reviewDigest string) tdd.Manifest {
	t.Helper()
	rev := tdd.Review{Reviewer: "rev-one", Rationale: "variant reviewed", SubjectDigest: reviewDigest}
	ref := func(test string) tdd.TestRef {
		return tdd.TestRef{Design: ".", Milestone: "M1", Suite: "s1", Test: test}
	}
	exp := func(test, outcome string, assertions ...string) tdd.Expectation {
		return tdd.Expectation{Test: ref(test), Outcome: outcome, Assertions: assertions}
	}
	newAssert := func(id string, line int64) tdd.Assertion {
		return tdd.Assertion{ID: id, Source: "tests/alpha_test.go", Line: line, Helper: protocol.AssertionHelperV1}
	}
	return tdd.Manifest{
		Schema:              protocol.SchemaMilestone,
		ID:                  "M1",
		Revision:            1,
		Repository:          ".",
		ImplementationRoots: []string{"."},
		FrozenRoots:         []string{"tests"},
		SubjectEntries:      []tdd.SubjectEntry{{Path: "src.txt", Kind: "file"}},
		Suites: []tdd.Suite{{
			ID:      "s1",
			Adapter: protocol.AdapterGoTesting,
			Runtime: tdd.RuntimeRef{Profile: "go", Version: "go1.27.1", Platform: "darwin/arm64", Closure: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
			Root:    ".", Files: []string{"tests/alpha_test.go"},
			Tests: []tdd.Test{
				{ID: "t1", Native: tdd.NativeID{Package: "github.com/example/tests", Test: "TestAlphaRefusal"}, Source: "tests/alpha_test.go", Role: protocol.RoleNegative, Assertions: []tdd.Assertion{newAssert("asr-1", 10)}},
				{ID: "t2", Native: tdd.NativeID{Package: "github.com/example/tests", Test: "TestAlphaPublish"}, Source: "tests/alpha_test.go", Role: protocol.RolePositive, Assertions: []tdd.Assertion{newAssert("asr-2", 20)}},
			},
			Environment:     []tdd.EnvironmentVar{{Name: "LC_ALL", Value: "C"}},
			DependencyRoots: []string{},
		}},
		Obligations: []tdd.Obligation{
			{Key: tdd.ObligationKey{Design: ".", Kind: protocol.KindOracleRow, Owner: "machines/Alpha.oracle.md", ID: "alpha-s1"}, Positive: []tdd.TestRef{ref("t2")}, Negative: []tdd.TestRef{ref("t1")}},
			{Key: tdd.ObligationKey{Design: ".", Kind: protocol.KindGuardClause, Owner: "Alpha", ID: "gate-x:clause-a"}, Negative: []tdd.TestRef{ref("t1")}},
			{Key: tdd.ObligationKey{Design: ".", Kind: protocol.KindInvariant, Owner: "Widget", ID: "inv-owned"}, Positive: []tdd.TestRef{ref("t2")}},
			{Key: tdd.ObligationKey{Design: ".", Kind: protocol.KindInvariant, Owner: "model", ID: "inv-global"}, Positive: []tdd.TestRef{ref("t2")}},
			{Key: tdd.ObligationKey{Design: ".", Kind: protocol.KindRuntime, Owner: "platform", ID: "rt-load"}, Positive: []tdd.TestRef{ref("t2")}},
		},
		Baseline: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		Variants: []tdd.Variant{
			{ID: "safe-1", Kind: protocol.VariantSafeControl, Source: "sha256:3333333333333333333333333333333333333333333333333333333333333333", Pair: "pair-1", TargetTests: []tdd.TestRef{ref("t1")}, Expected: []tdd.Expectation{exp("t1", protocol.OutcomePass, "asr-1"), exp("t2", protocol.OutcomeAssertionFail, "asr-2")}, Review: rev},
			{ID: "unsafe-1", Kind: protocol.VariantUnsafeChallenge, Source: "sha256:4444444444444444444444444444444444444444444444444444444444444444", Pair: "pair-1", TargetTests: []tdd.TestRef{ref("t1")}, Expected: []tdd.Expectation{exp("t1", protocol.OutcomeAssertionFail, "asr-1"), exp("t2", protocol.OutcomeAssertionFail, "asr-2")}, Review: rev},
			{ID: "safe-2", Kind: protocol.VariantSafeControl, Source: "sha256:5555555555555555555555555555555555555555555555555555555555555555", Pair: "", TargetTests: []tdd.TestRef{}, Expected: []tdd.Expectation{exp("t1", protocol.OutcomeAssertionFail, "asr-1"), exp("t2", protocol.OutcomePass, "asr-2")}, Review: rev},
		},
		RedExpectations: []tdd.Expectation{exp("t1", protocol.OutcomeAssertionFail, "asr-1"), exp("t2", protocol.OutcomeAssertionFail, "asr-2")},
		Checks:          []tdd.Check{{ID: "chk-1", Kind: protocol.CheckKindFormat, Profile: "go-format/v1", Inputs: []string{"tests/alpha_test.go"}}},
		RedControls: []tdd.RedControl{
			{Test: ref("t1"), Assertion: "asr-1", SafeVariant: "safe-1"},
			{Test: ref("t2"), Assertion: "asr-2", SafeVariant: "safe-2"},
		},
		Limits: tdd.Limits{
			WallMS: protocol.LimitWallDefaultMS, CleanupMS: protocol.LimitCleanupDefaultMS,
			StdoutBytes: protocol.LimitStdoutDefault, StderrBytes: protocol.LimitStderrDefault,
			EventBytes: protocol.LimitEventBytesDefault, EventCount: protocol.LimitEventCountDefault,
			Jobs: protocol.LimitJobsDefault, BundleBytes: protocol.LimitBundleDefault,
			Entries: protocol.LimitEntriesDefault, Depth: protocol.LimitDepthDefault,
		},
		Review: tdd.Review{Reviewer: "rev-one", Rationale: "milestone reviewed", SubjectDigest: reviewDigest},
	}
}

func inventoryRenderManifest(t *testing.T, m tdd.Manifest) []byte {
	t.Helper()
	strs := func(xs []string) []any {
		out := []any{}
		for _, x := range xs {
			out = append(out, x)
		}
		return out
	}
	refs := func(rs []tdd.TestRef) []any {
		out := []any{}
		for _, r := range rs {
			out = append(out, map[string]any{"design": r.Design, "milestone": r.Milestone, "suite": r.Suite, "test": r.Test})
		}
		return out
	}
	exps := func(es []tdd.Expectation) []any {
		out := []any{}
		for _, e := range es {
			out = append(out, map[string]any{"test": map[string]any{"design": e.Test.Design, "milestone": e.Test.Milestone, "suite": e.Test.Suite, "test": e.Test.Test}, "outcome": e.Outcome, "assertions": strs(e.Assertions)})
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
			tests = append(tests, map[string]any{"id": tst.ID, "native": map[string]any{"package": tst.Native.Package, "test": tst.Native.Test}, "source": tst.Source, "role": tst.Role, "assertions": as})
		}
		env := []any{}
		for _, e := range s.Environment {
			env = append(env, map[string]any{"name": e.Name, "value": e.Value})
		}
		suites = append(suites, map[string]any{
			"id": s.ID, "adapter": s.Adapter,
			"runtime": map[string]any{"profile": s.Runtime.Profile, "version": s.Runtime.Version, "platform": s.Runtime.Platform, "closure": s.Runtime.Closure},
			"root":    s.Root, "files": strs(s.Files), "tests": tests, "environment": env,
			"dependency_roots": strs(s.DependencyRoots),
		})
	}
	obls := []any{}
	for _, o := range m.Obligations {
		obls = append(obls, map[string]any{
			"key":      map[string]any{"design": o.Key.Design, "kind": o.Key.Kind, "owner": o.Key.Owner, "id": o.Key.ID},
			"positive": refs(o.Positive), "negative": refs(o.Negative),
		})
	}
	variants := []any{}
	for _, v := range m.Variants {
		variants = append(variants, map[string]any{
			"id": v.ID, "kind": v.Kind, "source": v.Source, "pair": v.Pair,
			"target_tests": refs(v.TargetTests), "expected": exps(v.Expected),
			"review": map[string]any{"reviewer": v.Review.Reviewer, "rationale": v.Review.Rationale, "subject_digest": v.Review.SubjectDigest},
		})
	}
	checks := []any{}
	for _, c := range m.Checks {
		checks = append(checks, map[string]any{"id": c.ID, "kind": c.Kind, "profile": c.Profile, "inputs": strs(c.Inputs)})
	}
	controls := []any{}
	for _, rc := range m.RedControls {
		controls = append(controls, map[string]any{"test": map[string]any{"design": rc.Test.Design, "milestone": rc.Test.Milestone, "suite": rc.Test.Suite, "test": rc.Test.Test}, "assertion": rc.Assertion, "safe_variant": rc.SafeVariant})
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
	b, err := jsonMarshal(map[string]any{
		"schema": m.Schema, "id": m.ID, "revision": m.Revision, "predecessor": pred,
		"repository":           m.Repository,
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
		"review": map[string]any{"reviewer": m.Review.Reviewer, "rationale": m.Review.Rationale, "subject_digest": m.Review.SubjectDigest},
	})
	if err != nil {
		t.Fatalf("render manifest: %v", err)
	}
	return b
}

func hasObligation(inv tdd.Inventory, design, kind, owner, id string) bool {
	for _, o := range inv.Obligations {
		if o.Key.Design == design && o.Key.Kind == kind && o.Key.Owner == owner && o.Key.ID == id {
			return true
		}
	}
	return false
}

// AC2: qualified {design,kind,owner,id} keys derived from the authoritative
// design sources; deterministic digest.
func TestAssuranceInventoryDerivesQualifiedObligations(t *testing.T) {
	design := bindInventoryControls(t, writeInventoryDesign(t, nil))
	inv, err := AssuranceInventory(design)
	if err != nil {
		t.Fatalf("AssuranceInventory: %v", err)
	}
	for _, key := range [][4]string{
		{".", protocol.KindOracleRow, "machines/Alpha.oracle.md", "alpha-s1"},
		{".", protocol.KindGuardClause, "Alpha", "gate-x:clause-a"},
		{".", protocol.KindInvariant, "Widget", "inv-owned"},
		{".", protocol.KindInvariant, "model", "inv-global"},
		{".", protocol.KindRuntime, "platform", "rt-concurrency"},
		{".", protocol.KindRuntime, "platform", "rt-delivery"},
		{".", protocol.KindRuntime, "platform", "rt-migration"},
		{".", protocol.KindRuntime, "platform", "rt-restore"},
		{".", protocol.KindRuntime, "platform", "rt-load"},
		{".", protocol.KindRuntime, "platform", "rt-security"},
		{".", protocol.KindRuntime, "platform", "rt-obs"},
	} {
		if !hasObligation(inv, key[0], key[1], key[2], key[3]) {
			t.Fatalf("inventory misses obligation %v", key)
		}
	}
	if len(inv.Milestones) != 1 || inv.Milestones[0] != (tdd.MilestoneKey{Design: ".", Milestone: "M1"}) {
		t.Fatalf("inventory milestones wrong: %+v", inv.Milestones)
	}
	if !strings.HasPrefix(inv.Digest, "sha256:") || len(inv.Digest) != 71 {
		t.Fatalf("inventory digest not a digest: %q", inv.Digest)
	}
	again, err := AssuranceInventory(design)
	if err != nil || again.Digest != inv.Digest {
		t.Fatalf("inventory digest must be deterministic: %v %s vs %s", err, again.Digest, inv.Digest)
	}
}

// AC2: the same stable-id text under another owner never pools.
func TestAssuranceInventoryOwnerPoolingIsDistinct(t *testing.T) {
	extra := map[string]string{
		"machines/Beta.machine.json": "{}",
		"machines/Beta.oracle.md":    "# Beta oracle\n\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| beta-t1 | alpha-s1 | gate-x | same text id, other owner |\n",
	}
	design := bindInventoryControls(t, writeInventoryDesign(t, extra))
	inv, err := AssuranceInventory(design)
	if err != nil {
		t.Fatalf("AssuranceInventory: %v", err)
	}
	if !hasObligation(inv, ".", protocol.KindOracleRow, "machines/Alpha.oracle.md", "alpha-s1") ||
		!hasObligation(inv, ".", protocol.KindOracleRow, "machines/Beta.oracle.md", "alpha-s1") {
		t.Fatal("same text id under two owners must yield two qualified obligations")
	}
}

// AC7/AC2: corrupted trees fail closed through AssuranceInventory.
func TestAssuranceInventoryFailsClosedOnCorruption(t *testing.T) {
	cases := []struct {
		name  string
		mutfn func(t *testing.T, design string)
	}{
		{"corrupt-plan-json", func(t *testing.T, design string) {
			bindInventoryControls(t, design)
			if err := os.WriteFile(filepath.Join(design, "assurance", "plan.json"), []byte("{not json"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"machine-without-oracle", func(t *testing.T, design string) {
			bindInventoryControls(t, design)
			if err := os.WriteFile(filepath.Join(design, "machines", "Gamma.machine.json"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"control-namespace-pollution", func(t *testing.T, design string) {
			bindInventoryControls(t, design)
			if err := os.WriteFile(filepath.Join(design, "assurance", "notes.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			design := writeInventoryDesign(t, nil)
			c.mutfn(t, design)
			if _, err := AssuranceInventory(design); err == nil {
				t.Fatalf("%s must fail closed", c.name)
			}
		})
	}
}

// AC2/AC7: deleting the milestone cannot reduce requirements — the valid
// flow passes, the milestone-deleted flow fails validation.
func TestAssuranceInventoryMilestoneDeletionTightens(t *testing.T) {
	design := bindInventoryControls(t, writeInventoryDesign(t, nil))
	inv, err := AssuranceInventory(design)
	if err != nil {
		t.Fatalf("AssuranceInventory: %v", err)
	}
	plan, err := tdd.LoadPlan(design)
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	var manifests []tdd.Manifest
	for _, pm := range plan.Milestones {
		m, err := tdd.LoadManifest(filepath.Join(design, filepath.FromSlash(pm.Manifest)))
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		manifests = append(manifests, m)
	}
	if err := tdd.Validate(plan, manifests, inv); err != nil {
		t.Fatalf("valid tree must validate: %v", err)
	}
	noBuild := strings.Replace(invFixtureFiles["BUILD.md"], "**M1 - Alpha behavior.**\nStatus: open\nDoD: alpha-s1 covered; inv-owned preserved.\n", "", 1)
	if noBuild == invFixtureFiles["BUILD.md"] {
		t.Fatal("fixture edit did not apply")
	}
	if err := os.WriteFile(filepath.Join(design, "BUILD.md"), []byte(noBuild), 0o644); err != nil {
		t.Fatal(err)
	}
	inv2, err := AssuranceInventory(design)
	if err != nil {
		t.Fatalf("AssuranceInventory after milestone deletion: %v", err)
	}
	if len(inv2.Milestones) != 0 {
		t.Fatalf("deleted milestone must leave the inventory: %+v", inv2.Milestones)
	}
	err = tdd.Validate(plan, manifests, inv2)
	if err == nil || !strings.Contains(err.Error(), "current") {
		t.Fatalf("milestone deletion must block validation, got %v", err)
	}
}

// AC7: real integration of valid and deliberately corrupted trees through
// AssuranceInventory and Validate.
func TestAssuranceValidateRealIntegration(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		design := bindInventoryControls(t, writeInventoryDesign(t, nil))
		inv, err := AssuranceInventory(design)
		if err != nil {
			t.Fatalf("AssuranceInventory: %v", err)
		}
		plan, err := tdd.LoadPlan(design)
		if err != nil {
			t.Fatalf("LoadPlan: %v", err)
		}
		m, err := tdd.LoadManifest(filepath.Join(design, "assurance", "milestones", "M1.json"))
		if err != nil {
			t.Fatalf("LoadManifest: %v", err)
		}
		if err := tdd.Validate(plan, []tdd.Manifest{m}, inv); err != nil {
			t.Fatalf("Validate: %v", err)
		}
	})
	t.Run("subject-overlaps-frozen", func(t *testing.T) {
		design := writeInventoryDesign(t, nil)
		payload, err := tdd.DesignPayloadDigest(design)
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		planReview, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionPlan(inventoryBasePlan(t, "p")), payload)
		if err != nil {
			t.Fatal(err)
		}
		m := inventoryBaseManifest(t, "placeholder")
		mReview, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionManifest(m), payload)
		if err != nil {
			t.Fatal(err)
		}
		m.SubjectEntries = []tdd.SubjectEntry{{Path: "tests", Kind: "directory"}}
		if err := os.MkdirAll(filepath.Join(design, "assurance", "milestones"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(design, "assurance", "plan.json"), inventoryRenderPlan(t, inventoryBasePlan(t, planReview)), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(design, "assurance", "milestones", "M1.json"), inventoryRenderManifest(t, m), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err = tdd.LoadManifest(filepath.Join(design, "assurance", "milestones", "M1.json"))
		if err == nil || !strings.Contains(err.Error(), "subject") {
			t.Fatalf("subject overlapping frozen entries must fail, got %v", err)
		}
		_ = mReview
	})
}
