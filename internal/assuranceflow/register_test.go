// Frozen RED/GREEN suite for the MAC-62s6 assuranceflow.Register API: the
// explicit authored-revision transaction over real design trees, real
// external stores and real captured bundles. Covers fresh registration,
// multi-milestone and child-design graphs, conflicting writers, exact retry,
// intervening heads, source ABA, review mismatch, draft/registered
// mismatches, deletion, pre/post-commit failures and failing output, under
// the fixed non-execution deadline. No mocks anywhere on the store path.
package assuranceflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

const (
	flowProjectID = "8a1f6c2e-5b3d-4c84-9d0a-2f6b7c9d1e01"
	flowSubject   = "flow subject v1\nflow-anchor\n"
	flowGoSource  = "package tests\n"
	flowChildSub  = "child subject v1\nchild-anchor\n"
)

func flowSHA(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

type flowProject struct {
	impl      string
	store     string
	head0     string
	baseline  string
	safeRef   string
	unsafeRef string
}

func flowWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newFlowProject materializes the repo skeleton (parent design "app" with
// M1/M2, child design "child" with M1), a real store and real captured
// bundles; the subject ends at v1 so payload digests bind cleanly.
func newFlowProject(t *testing.T) *flowProject {
	t.Helper()
	p := &flowProject{impl: t.TempDir(), store: filepath.Join(t.TempDir(), "store")}
	flowWrite(t, filepath.Join(p.impl, "app", "BUILD.md"), "# Build\n\n## Build plan\n\n**M1 - Alpha.**\nStatus: open\nDoD: alpha covered.\n\n**M2 - Beta.**\nStatus: open\nDoD: beta covered.\n")
	flowWrite(t, filepath.Join(p.impl, "app", "machines", "A.machine.json"), "{}")
	flowWrite(t, filepath.Join(p.impl, "app", "machines", "A.oracle.md"), "# A oracle\n\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| a-t1 | alpha-s1 | g1 | refuses malformed input |\n| a-t2 | alpha-s2 | g2 | publishes widget |\n")
	flowWrite(t, filepath.Join(p.impl, "app", "src.txt"), flowSubject)
	flowWrite(t, filepath.Join(p.impl, "app", "tests", "alpha_test.go"), flowGoSource)
	flowWrite(t, filepath.Join(p.impl, "child", "BUILD.md"), "# Build\n\n## Build plan\n\n**M1 - Gamma.**\nStatus: open\nDoD: gamma covered.\n")
	flowWrite(t, filepath.Join(p.impl, "child", "machines", "C.machine.json"), "{}")
	flowWrite(t, filepath.Join(p.impl, "child", "machines", "C.oracle.md"), "# C oracle\n\n| test id | stable id | guard | behavior |\n| --- | --- | --- | --- |\n| c-t1 | gamma-s1 | g1 | isolates child scope |\n")
	flowWrite(t, filepath.Join(p.impl, "child", "src.txt"), flowChildSub)
	flowWrite(t, filepath.Join(p.impl, "child", "tests", "gamma_test.go"), flowGoSource)
	init, err := tdd.InitStore(context.Background(), p.store, flowProjectID)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	p.head0 = init.HeadDigest
	p.baseline = p.capture(t, flowSubject, "baseline")
	p.safeRef = p.capture(t, "flow subject v2 safe\nflow-anchor\n", "safe-1")
	p.unsafeRef = p.capture(t, "flow subject v3 unsafe\nflow-anchor\n", "unsafe-1")
	flowWrite(t, filepath.Join(p.impl, "app", "src.txt"), flowSubject)
	return p
}

// capture retains one real app source state under a minimal valid manifest
// shape and returns the bundle ref.
func (p *flowProject) capture(t *testing.T, subject, name string) string {
	t.Helper()
	flowWrite(t, filepath.Join(p.impl, "app", "src.txt"), subject)
	ctl := filepath.Join(t.TempDir(), "ctl")
	flowWrite(t, filepath.Join(ctl, protocol.PlanFileName), "{}\n")
	flowWrite(t, filepath.Join(ctl, protocol.MilestonesDirName, "M1.json"), "{}\n")
	ref, err := tdd.Capture(context.Background(), tdd.CaptureRequest{
		Inputs: tdd.InputView{
			SourceRoot:          p.impl,
			DesignPath:          "app",
			ImplementationPaths: []string{"app"},
			ControlRoot:         ctl,
			Revalidate:          func() error { return nil },
			Release:             func() error { return nil },
		},
		Manifest: p.captureShape(), Name: name, Store: p.store, Limits: tdd.Limits{},
	})
	if err != nil {
		t.Fatalf("capture %s: %v", name, err)
	}
	return ref.Ref()
}

func (p *flowProject) captureShape() tdd.Manifest {
	return tdd.Manifest{
		Schema: protocol.SchemaMilestone, ID: "M1", Revision: 1, Repository: ".",
		ImplementationRoots: []string{"."}, FrozenRoots: []string{"app/tests"},
		SubjectEntries: []tdd.SubjectEntry{{Path: "app/src.txt", Kind: "file"}},
		Suites: []tdd.Suite{{
			ID: "s1", Adapter: protocol.AdapterGoTesting,
			Runtime: tdd.RuntimeRef{Profile: "go", Version: "go1.27.1", Platform: "darwin/arm64", Closure: "sha256:" + strings.Repeat("c", 64)},
			Root:    "app", Files: []string{"app/tests/alpha_test.go"},
			Tests: []tdd.Test{{
				ID: "t1", Native: tdd.NativeID{Package: "github.com/example/tests", Test: "TestAlphaRefusal"},
				Source: "app/tests/alpha_test.go", Role: tdd.RoleNegative,
				Assertions: []tdd.Assertion{{ID: "asr-1", Source: "app/tests/alpha_test.go", Line: 10, Helper: protocol.AssertionHelperV1}},
			}},
		}},
	}
}

// ---- plan authoring (struct-first; the loader is the code under test) ----

func flowPlanStruct(design string, milestones []string) tdd.Plan {
	srcDigest := flowSHA([]byte(flowSubject))
	subject, anchor := "src.txt", "flow-anchor"
	if design == "child" {
		srcDigest = flowSHA([]byte(flowChildSub))
		subject, anchor = "src.txt", "child-anchor"
	}
	ros := make([]tdd.RuntimeObligation, 0, len(protocol.RuntimeCategories))
	for _, cat := range protocol.RuntimeCategories {
		ros = append(ros, tdd.RuntimeObligation{
			ID: "rt-" + cat, Category: cat, Owner: "platform",
			SourceRefs:  []tdd.SourceRef{{Path: subject, Anchor: anchor, Digest: srcDigest}},
			Disposition: tdd.DispositionNotApplicable, Reason: "out of scope for this revision",
		})
	}
	ms := make([]tdd.PlanMilestone, 0, len(milestones))
	for _, m := range milestones {
		ms = append(ms, tdd.PlanMilestone{ID: m, Manifest: design + "/assurance/milestones/" + m + ".json"})
	}
	return tdd.Plan{Schema: protocol.SchemaPlan, Design: design, Milestones: ms, ProjectID: flowProjectID, RuntimeObligations: ros}
}

// bindPlan renders and binds the plan (optionally edited after struct
// authoring) over the real design payload and writes it.
func bindPlan(t *testing.T, impl, design string, milestones []string, edit func(*tdd.Plan)) {
	t.Helper()
	designDir := filepath.Join(impl, design)
	plan := flowPlanStruct(design, milestones)
	if edit != nil {
		edit(&plan)
	}
	payload, err := tdd.DesignPayloadDigest(designDir)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionPlan(plan), payload)
	if err != nil {
		t.Fatal(err)
	}
	review := tdd.Review{Reviewer: "rev-flow", Rationale: "reviewed flow control", SubjectDigest: digest}
	for i := range plan.RuntimeObligations {
		plan.RuntimeObligations[i].Review = review
	}
	flowWrite(t, filepath.Join(designDir, protocol.ControlDirName, protocol.PlanFileName), renderFlowPlan(plan))
}

func renderFlowPlan(p tdd.Plan) string {
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
			"id": ro.ID, "category": ro.Category, "owner": ro.Owner, "source_refs": srs,
			"disposition": ro.Disposition, "reason": ro.Reason,
			"review": map[string]any{"reviewer": ro.Review.Reviewer, "rationale": ro.Review.Rationale, "subject_digest": ro.Review.SubjectDigest},
			"tests":  []any{},
		})
	}
	b, err := json.Marshal(map[string]any{"schema": p.Schema, "design": p.Design, "milestones": ms, "project_id": p.ProjectID, "runtime_obligations": ros})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ---- milestone authoring (doc maps bound through DecodeArchivedManifest) ----

// flowManifestDoc renders one milestone control with null-or-real bundle
// refs; reviews carry a placeholder valid digest until bindManifest binds
// them over the real design payload.
func flowManifestDoc(design, id string, revision int64, predecessor string, finalized bool) map[string]any {
	subject := design + "/src.txt"
	testFile := design + "/tests/alpha_test.go"
	owner, stable := "machines/A.oracle.md", "alpha-s1"
	if design == "child" {
		testFile = design + "/tests/gamma_test.go"
		owner, stable = "machines/C.oracle.md", "gamma-s1"
	}
	tests := []any{map[string]any{
		"id": "t1", "native": map[string]any{"package": "github.com/example/tests", "test": "TestRefusal"},
		"source": testFile, "role": "negative",
		"assertions": []any{map[string]any{"id": "asr-1", "source": testFile, "line": 10, "helper": protocol.AssertionHelperV1}},
	}}
	if id == "M2" {
		tests = append(tests, map[string]any{
			"id": "t2", "native": map[string]any{"package": "github.com/example/tests", "test": "TestPublish"},
			"source": testFile, "role": "positive",
			"assertions": []any{map[string]any{"id": "asr-2", "source": testFile, "line": 20, "helper": protocol.AssertionHelperV1}},
		})
	}
	ref := func(test string) map[string]any {
		return map[string]any{"design": design, "milestone": id, "suite": "s1", "test": test}
	}
	exp := func(test, outcome string, assertions ...string) map[string]any {
		return map[string]any{"test": ref(test), "outcome": outcome, "assertions": assertions}
	}
	obls := []any{map[string]any{
		"key":      map[string]any{"design": design, "kind": "oracle-row", "owner": owner, "id": stable},
		"positive": []any{}, "negative": []any{ref("t1")},
	}}
	exps := []any{exp("t1", "assertion-fail", "asr-1")}
	variants := []any{}
	controls := []any{}
	if finalized {
		variants = []any{
			map[string]any{"id": "safe-1", "kind": "safe-control", "source": nil, "pair": "pair-1",
				"target_tests": []any{ref("t1")}, "expected": []any{exp("t1", "pass", "asr-1")}},
			map[string]any{"id": "unsafe-1", "kind": "unsafe-challenge", "source": nil, "pair": "pair-1",
				"target_tests": []any{ref("t1")}, "expected": []any{exp("t1", "assertion-fail", "asr-1")}},
		}
		controls = []any{map[string]any{"test": ref("t1"), "assertion": "asr-1", "safe_variant": "safe-1"}}
	}
	if id == "M2" {
		obls = append(obls, map[string]any{
			"key":      map[string]any{"design": design, "kind": "oracle-row", "owner": owner, "id": "alpha-s2"},
			"positive": []any{ref("t2")}, "negative": []any{},
		})
		exps = append(exps, exp("t2", "assertion-fail", "asr-2"))
		if finalized {
			variants = []any{
				map[string]any{"id": "safe-1", "kind": "safe-control", "source": nil, "pair": "pair-1",
					"target_tests": []any{ref("t1")}, "expected": []any{exp("t1", "pass", "asr-1"), exp("t2", "assertion-fail", "asr-2")}},
				map[string]any{"id": "unsafe-1", "kind": "unsafe-challenge", "source": nil, "pair": "pair-1",
					"target_tests": []any{ref("t1")}, "expected": []any{exp("t1", "assertion-fail", "asr-1"), exp("t2", "assertion-fail", "asr-2")}},
				map[string]any{"id": "safe-2", "kind": "safe-control", "source": nil, "pair": "",
					"target_tests": []any{}, "expected": []any{exp("t1", "assertion-fail", "asr-1"), exp("t2", "pass", "asr-2")}},
			}
			controls = append(controls, map[string]any{"test": ref("t2"), "assertion": "asr-2", "safe_variant": "safe-2"})
		}
	}
	var pred any
	if predecessor != "" {
		pred = predecessor
	}
	var baseline any
	placeholder := map[string]any{"reviewer": "rev-flow", "rationale": "placeholder", "subject_digest": "sha256:" + strings.Repeat("0", 64)}
	for _, v := range variants {
		v.(map[string]any)["review"] = placeholder
	}
	return map[string]any{
		"schema": protocol.SchemaMilestone, "id": id, "revision": revision, "predecessor": pred,
		"repository": ".", "implementation_roots": []string{"."}, "frozen_roots": []string{design + "/tests"},
		"subject_entries": []any{map[string]any{"path": subject, "kind": "file"}},
		"suites": []any{map[string]any{
			"id": "s1", "adapter": protocol.AdapterGoTesting,
			"runtime": map[string]any{"profile": "go", "version": "go1.27.1", "platform": "darwin/arm64", "closure": "sha256:" + strings.Repeat("c", 64)},
			"root":    design, "files": []any{testFile}, "tests": tests, "environment": []any{}, "dependency_roots": []any{},
		}},
		"obligations": obls, "baseline": baseline, "variants": variants, "red_expectations": exps,
		"checks": []any{}, "red_controls": controls,
		"limits": map[string]any{
			"wall_ms": protocol.LimitWallDefaultMS, "cleanup_ms": protocol.LimitCleanupDefaultMS,
			"stdout_bytes": protocol.LimitStdoutDefault, "stderr_bytes": protocol.LimitStderrDefault,
			"event_bytes": protocol.LimitEventBytesDefault, "event_count": protocol.LimitEventCountDefault,
			"jobs": protocol.LimitJobsDefault, "bundle_bytes": protocol.LimitBundleDefault,
			"entries": protocol.LimitEntriesDefault, "depth": protocol.LimitDepthDefault,
		},
		"review": placeholder,
	}
}

// bindManifest writes one bound milestone control. extra mutates the doc
// after refs are set (used for cross-design obligation variants).
func bindManifest(t *testing.T, impl, design, id string, revision int64, predecessor string, finalized bool, refs []string, extra func(map[string]any)) {
	t.Helper()
	designDir := filepath.Join(impl, design)
	doc := flowManifestDoc(design, id, revision, predecessor, finalized)
	if finalized {
		doc["baseline"] = refs[0]
		if vars, ok := doc["variants"].([]any); ok {
			for i := range vars {
				v := vars[i].(map[string]any)
				if i < len(refs)-1 {
					v["source"] = refs[i+1]
				}
			}
		}
	}
	if extra != nil {
		extra(doc)
	}
	rendered := mustJSON(t, doc)
	loaded, err := tdd.DecodeArchivedManifest(design, id, []byte(rendered))
	if err != nil {
		t.Fatalf("authoring decode %s/%s: %v", design, id, err)
	}
	payload, err := tdd.DesignPayloadDigest(designDir)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := tdd.ReviewSubjectDigest(tdd.ReviewProjectionManifest(loaded), payload)
	if err != nil {
		t.Fatal(err)
	}
	review := map[string]any{"reviewer": "rev-flow", "rationale": "reviewed flow control", "subject_digest": digest}
	doc["review"] = review
	for _, v := range doc["variants"].([]any) {
		v.(map[string]any)["review"] = review
	}
	flowWrite(t, filepath.Join(designDir, protocol.ControlDirName, protocol.MilestonesDirName, id+".json"), mustJSON(t, doc))
}

func mustJSON(t *testing.T, doc map[string]any) string {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func (p *flowProject) refs() []string {
	return []string{p.baseline, p.safeRef, p.unsafeRef, p.safeRef}
}

// bindFinalizedProject binds app (M1+M2) and child (M1) finalized controls.
func (p *flowProject) bindFinalizedProject(t *testing.T) {
	t.Helper()
	bindPlan(t, p.impl, "app", []string{"M1", "M2"}, nil)
	bindPlan(t, p.impl, "child", []string{"M1"}, nil)
	bindManifest(t, p.impl, "app", "M1", 1, "", true, p.refs(), nil)
	bindManifest(t, p.impl, "app", "M2", 1, "", true, p.refs(), nil)
	bindManifest(t, p.impl, "child", "M1", 1, "", true, p.refs(), nil)
}

func (p *flowProject) register(ctx context.Context, design string, milestones []string, expected string, output io.Writer) (Registration, error) {
	return Register(ctx, RegisterRequest{
		Design: design, Implementation: p.impl, Store: p.store, ExpectedHead: expected, Milestones: milestones,
	}, output)
}

type flowHeadDoc struct {
	Generation int64 `json:"generation"`
	Previous   any   `json:"previous"`
	Plans      []struct {
		Design     string `json:"design"`
		PlanDigest string `json:"plan_digest"`
	} `json:"plans"`
	Milestones []struct {
		Design         string `json:"design"`
		Milestone      string `json:"milestone"`
		Revision       int64  `json:"revision"`
		ManifestDigest string `json:"manifest_digest"`
		Predecessor    any    `json:"predecessor"`
	} `json:"milestones"`
}

func flowHead(t *testing.T, store string) flowHeadDoc {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(store, "ledger", "head.json"))
	if err != nil {
		t.Fatal(err)
	}
	var h flowHeadDoc
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatal(err)
	}
	return h
}

func flowDigest(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return flowSHA(raw)
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("output failed") }

// ---- the suite ----

// AC1/AC2/AC3: fresh registration of all milestones accepts the exact empty
// generation-zero head with no execution receipt, returns sorted keys,
// state registered-not-executed, and advances the head exactly once.
func TestRegisterFreshAllMilestones(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	var out strings.Builder
	reg, err := p.register(context.Background(), "app", nil, p.head0, &out)
	if err != nil {
		t.Fatalf("fresh register: %v", err)
	}
	if reg.State() != tdd.StateRegisteredNotExecuted {
		t.Fatalf("state = %q", reg.State())
	}
	if reg.ProjectID() != flowProjectID || reg.StoreID() == "" || reg.PreviousHead() != p.head0 || reg.Generation() != 1 {
		t.Fatalf("registration = %+v", reg)
	}
	keys := reg.MilestoneKeys()
	if len(keys) != 2 || keys[0].Design != "app" || keys[0].Milestone != "M1" || keys[1].Milestone != "M2" {
		t.Fatalf("milestone keys = %+v", keys)
	}
	h := flowHead(t, p.store)
	if h.Generation != 1 || h.Previous != p.head0 || len(h.Plans) != 1 || h.Plans[0].Design != "app" {
		t.Fatalf("head = %+v", h)
	}
	if len(h.Milestones) != 2 || h.Milestones[0].Revision != 1 || h.Milestones[0].Predecessor != nil || h.Milestones[1].Revision != 1 {
		t.Fatalf("head milestones = %+v", h.Milestones)
	}
	if runs, _ := os.ReadDir(filepath.Join(p.store, "runs")); len(runs) != 0 {
		t.Fatal("registration fabricated execution records")
	}
	if !strings.Contains(out.String(), tdd.StateRegisteredNotExecuted) || !strings.Contains(out.String(), reg.HeadDigest()) {
		t.Fatalf("output = %q", out.String())
	}
	if reg.HeadDigest() != flowDigest(t, filepath.Join(p.store, "ledger", "head.json")) {
		t.Fatal("returned head digest does not name the committed head bytes")
	}
}

// AC1: explicit selection, duplicate/unknown targets and draft handling.
func TestRegisterSelectionValidation(t *testing.T) {
	p := newFlowProject(t)
	bindPlan(t, p.impl, "app", []string{"M1", "M2"}, nil)
	bindManifest(t, p.impl, "app", "M1", 1, "", true, p.refs(), nil)
	bindManifest(t, p.impl, "app", "M2", 1, "", false, nil, nil) // untargeted draft
	reg, err := p.register(context.Background(), "app", []string{"M1"}, p.head0, io.Discard)
	if err != nil {
		t.Fatalf("selected M1 with untargeted draft M2: %v", err)
	}
	if keys := reg.MilestoneKeys(); len(keys) != 1 || keys[0].Milestone != "M1" {
		t.Fatalf("keys = %+v", keys)
	}
	if _, err := p.register(context.Background(), "app", []string{"M1", "M1"}, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("duplicate milestone targets accepted: err = %v", err)
	}
	if _, err := p.register(context.Background(), "app", []string{"M7"}, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("undeclared milestone accepted: err = %v", err)
	}
	if _, err := p.register(context.Background(), "app", []string{"M2"}, reg.HeadDigest(), io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("draft target accepted as finalized: err = %v", err)
	}
}

// AC1: child designs register first; a parent reference resolves only
// through the child's registered manifest, never a future draft.
func TestRegisterChildrenFirst(t *testing.T) {
	p := newFlowProject(t)
	bindPlan(t, p.impl, "app", []string{"M1", "M2"}, nil)
	bindPlan(t, p.impl, "child", []string{"M1"}, nil)
	childRef := func(doc map[string]any) {
		// rebind M2's alpha-s2 obligation onto the child's registered test
		for _, o := range doc["obligations"].([]any) {
			key := o.(map[string]any)["key"].(map[string]any)
			if key["id"] == "alpha-s2" {
				o.(map[string]any)["positive"] = []any{map[string]any{"design": "child", "milestone": "M1", "suite": "s1", "test": "t1"}}
			}
		}
	}
	bindManifest(t, p.impl, "app", "M1", 1, "", true, p.refs(), nil)
	bindManifest(t, p.impl, "app", "M2", 1, "", true, p.refs(), childRef)
	bindManifest(t, p.impl, "child", "M1", 1, "", true, p.refs(), nil)
	if _, err := p.register(context.Background(), "app", nil, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
		t.Fatalf("parent with unregistered child accepted: err = %v", err)
	}
	if h := flowHead(t, p.store); h.Generation != 0 {
		t.Fatal("failed parent registration advanced the head")
	}
	child, err := p.register(context.Background(), "child", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatalf("child register: %v", err)
	}
	appReg, err := p.register(context.Background(), "app", nil, child.HeadDigest(), io.Discard)
	if err != nil {
		t.Fatalf("parent register after child: %v", err)
	}
	h := flowHead(t, p.store)
	if h.Generation != 2 || len(h.Milestones) != 3 || h.Milestones[0].Design != "app" || h.Milestones[2].Design != "child" {
		t.Fatalf("head = %+v", h)
	}
	if appReg.PreviousHead() != child.HeadDigest() {
		t.Fatal("parent did not advance from the child's head")
	}
}

// AC4/AC6: concurrent registrations with one expected head — exactly one
// commit; the divergent loser receives HEAD_CONFLICT.
func TestRegisterConflictingWriters(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	type outcome struct {
		reg Registration
		err error
	}
	out := make(chan outcome, 2)
	var wg sync.WaitGroup
	for _, design := range []string{"app", "child"} {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			var targets []string
			if d == "app" {
				targets = []string{"M1"} // diverges from the child design's successor
			}
			r, err := p.register(context.Background(), d, targets, p.head0, io.Discard)
			out <- outcome{r, err}
		}(design)
	}
	wg.Wait()
	close(out)
	var ok, conflict int
	for o := range out {
		switch {
		case o.err == nil:
			ok++
		case strings.Contains(o.err.Error(), "HEAD_CONFLICT"):
			conflict++
		default:
			t.Fatalf("unexpected error: %v", o.err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("outcomes = %d ok, %d conflict", ok, conflict)
	}
	if h := flowHead(t, p.store); h.Generation != 1 || len(h.Milestones) != 1 {
		t.Fatalf("head after conflict = %+v", h)
	}
	if _, err := os.Stat(filepath.Join(p.store, "staging")); !os.IsNotExist(err) {
		t.Fatal("staging survived conflicting writers")
	}
}

// AC4/AC5: exact retry is already-registered; an intervening head forces
// HEAD_CONFLICT without rebase; registering from the new head works.
func TestRegisterRetryAndInterveningHead(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	first, err := p.register(context.Background(), "app", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := p.register(context.Background(), "app", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatalf("exact retry: %v", err)
	}
	if retry.State() != tdd.StateAlreadyRegisteredNotExecuted || retry.HeadDigest() != first.HeadDigest() || retry.Generation() != first.Generation() {
		t.Fatalf("retry = %+v", retry)
	}
	if h := flowHead(t, p.store); h.Generation != 1 {
		t.Fatal("idempotent retry advanced the head")
	}
	if _, err := p.register(context.Background(), "child", nil, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "HEAD_CONFLICT") {
		t.Fatalf("stale expected head accepted: err = %v", err)
	}
	child, err := p.register(context.Background(), "child", nil, first.HeadDigest(), io.Discard)
	if err != nil {
		t.Fatalf("rebased child register: %v", err)
	}
	if child.Generation() != 2 {
		t.Fatalf("child generation = %d", child.Generation())
	}
}

// AC4: the flow-held control snapshot detects a real source ABA — the
// design root is wholesale replaced between snapshot and revalidation while
// exact bytes are restored; control byte drift is detected the same way.
func TestRegisterSourceABA(t *testing.T) {
	p := newFlowProject(t)
	bindPlan(t, p.impl, "app", []string{"M1"}, nil)
	bindManifest(t, p.impl, "app", "M1", 1, "", true, p.refs(), nil)
	app := filepath.Join(p.impl, "app")
	snap, err := newControlSnapshot(app)
	if err != nil {
		t.Fatal(err)
	}
	// A -> B -> A with a genuinely NEW root generation: the original root
	// moves away and an exact-byte copy takes its place, so content digests
	// alone cannot detect the swap
	swapped := filepath.Join(p.impl, "app.aba")
	if err := os.Rename(app, swapped); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(app, 0o755); err != nil {
		t.Fatal(err)
	}
	filepath.WalkDir(swapped, func(path string, d os.DirEntry, err error) error {
		if err != nil || path == swapped {
			return err
		}
		rel, _ := filepath.Rel(swapped, path)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(app, rel), 0o755)
		}
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		return os.WriteFile(filepath.Join(app, rel), raw, 0o644)
	})
	if err := snap.revalidate(); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("root-generation ABA undetected: err = %v", err)
	}
	os.RemoveAll(swapped)
	snap2, err := newControlSnapshot(app)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(app, protocol.ControlDirName, protocol.PlanFileName)
	flowWrite(t, planPath, strings.Replace(mustReadFile(t, planPath), "rev-flow", "rev-tamper", 1))
	if err := snap2.revalidate(); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("control byte drift undetected: err = %v", err)
	}
}

// AC6: a mutated review binding blocks registration (review mismatch) and
// consumes no revision.
func TestRegisterReviewMismatch(t *testing.T) {
	p := newFlowProject(t)
	bindPlan(t, p.impl, "app", []string{"M1"}, nil)
	bindManifest(t, p.impl, "app", "M1", 1, "", true, p.refs(), nil)
	planPath := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.PlanFileName)
	flowWrite(t, planPath, tamperFirstDigest(t, mustReadFile(t, planPath)))
	if _, err := p.register(context.Background(), "app", nil, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("review mismatch accepted: err = %v", err)
	}
	if h := flowHead(t, p.store); h.Generation != 0 {
		t.Fatal("failed registration consumed a revision")
	}
}

// tamperFirstDigest flips the final hex digit of the first subject_digest
// in a rendered control: the value stays grammatically valid but no longer
// binds the projection (rationale text alone is not digest-bound).
func tamperFirstDigest(t *testing.T, doc string) string {
	t.Helper()
	marker := `"subject_digest":"sha256:`
	i := strings.Index(doc, marker)
	if i < 0 {
		t.Fatal("no subject digest found to tamper")
	}
	j := i + len(marker) + 64 - 1
	if j >= len(doc) {
		t.Fatal("short digest")
	}
	last := string(doc[j])
	if last == "0" {
		last = "1"
	} else {
		last = "0"
	}
	return doc[:j] + last + doc[j+1:]
}

// AC3: untargeted registered control mismatch blocks; registered milestone
// deletion (file or plan entry) blocks.
func TestRegisterDraftAndDeletionMismatches(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	first, err := p.register(context.Background(), "app", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m1Path := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M1.json")
	// edit the registered M2 bytes without selecting them; advancing only
	// M1 in the same design is blocked by the untargeted mismatch
	m1Old := flowDigest(t, m1Path)
	bindManifest(t, p.impl, "app", "M2", 1, "", true, []string{p.baseline, p.safeRef, p.unsafeRef, p.unsafeRef}, nil)
	bindManifest(t, p.impl, "app", "M1", 2, m1Old, true, p.refs(), nil)
	if _, err := p.register(context.Background(), "app", []string{"M1"}, first.HeadDigest(), io.Discard); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("untargeted registered mismatch accepted: err = %v", err)
	}
	// restore M2, then delete the registered manifest file
	bindManifest(t, p.impl, "app", "M2", 1, "", true, p.refs(), nil)
	if err := os.Remove(filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M2.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := p.register(context.Background(), "app", []string{"M1"}, first.HeadDigest(), io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Fatalf("registered manifest deletion accepted: err = %v", err)
	}
	// restore the file, then remove the milestone from the plan
	bindManifest(t, p.impl, "app", "M2", 1, "", true, p.refs(), nil)
	bindPlan(t, p.impl, "app", []string{"M1"}, nil)
	if _, err := p.register(context.Background(), "app", []string{"M1"}, first.HeadDigest(), io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Fatalf("registered plan-entry deletion accepted: err = %v", err)
	}
	if h := flowHead(t, p.store); h.Generation != 1 {
		t.Fatal("mismatch handling advanced the head")
	}
}

// AC3: changing previously registered plan bytes requires advancing every
// registered milestone of that design in the same transaction.
func TestRegisterPlanChangeAdvancesAll(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	first, err := p.register(context.Background(), "app", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m1Path := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M1.json")
	m2Path := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M2.json")
	m1Old, m2Old := flowDigest(t, m1Path), flowDigest(t, m2Path)
	// plan edit with only M1 advanced
	bindPlan(t, p.impl, "app", []string{"M1", "M2"}, func(pl *tdd.Plan) {
		pl.RuntimeObligations[0].Reason = "plan edited; every registered milestone must advance"
	})
	bindManifest(t, p.impl, "app", "M1", 2, m1Old, true, p.refs(), nil)
	if _, err := p.register(context.Background(), "app", []string{"M1"}, first.HeadDigest(), io.Discard); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Fatalf("plan change without advancing M2 accepted: err = %v", err)
	}
	// advance both registered milestones in one transaction
	bindManifest(t, p.impl, "app", "M2", 2, m2Old, true, p.refs(), nil)
	reg, err := p.register(context.Background(), "app", nil, first.HeadDigest(), io.Discard)
	if err != nil {
		t.Fatalf("plan change advancing all: %v", err)
	}
	h := flowHead(t, p.store)
	if reg.Generation() != 2 || h.Generation != 2 || len(h.Milestones) != 2 || h.Milestones[0].Revision != 2 || h.Milestones[0].Predecessor != m1Old || h.Milestones[1].Predecessor != m2Old {
		t.Fatalf("head after plan-change advance = %+v", h.Milestones)
	}
}

// AC5/AC6: failing output after the durable commit returns no successful
// Registration although the head advanced; the exact retry confirms the
// already-registered state without erasing the committed history.
func TestRegisterFailingOutputAfterCommit(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	reg1, err := p.register(context.Background(), "app", nil, p.head0, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	// a revision-2 registration whose output fails after publication
	m1Path := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M1.json")
	m2Path := filepath.Join(p.impl, "app", protocol.ControlDirName, protocol.MilestonesDirName, "M2.json")
	bindManifest(t, p.impl, "app", "M1", 2, flowDigest(t, m1Path), true, p.refs(), nil)
	bindManifest(t, p.impl, "app", "M2", 2, flowDigest(t, m2Path), true, p.refs(), nil)
	var failOut failWriter
	if _, err := p.register(context.Background(), "app", nil, reg1.HeadDigest(), &failOut); err == nil || !strings.Contains(err.Error(), "CUSTODY_ERROR") {
		t.Fatalf("failing output returned success: err = %v", err)
	}
	if h := flowHead(t, p.store); h.Generation != 2 {
		t.Fatalf("postcommit failure did not advance the head: %+v", h)
	}
	// exact retry with working output confirms registration
	reg2, err := p.register(context.Background(), "app", nil, reg1.HeadDigest(), io.Discard)
	if err != nil {
		t.Fatalf("confirming retry: %v", err)
	}
	if reg2.State() != tdd.StateAlreadyRegisteredNotExecuted || reg2.Generation() != 2 {
		t.Fatalf("retry = %+v", reg2)
	}
}

// AC6: request validation, store placement and cancellation are fail-closed.
func TestRegisterRequestFailClosed(t *testing.T) {
	p := newFlowProject(t)
	p.bindFinalizedProject(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.register(ctx, "app", nil, p.head0, io.Discard); err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("cancelled register accepted: err = %v", err)
	}
	if _, err := p.register(context.Background(), "app", nil, "not-a-digest", io.Discard); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("invalid expected head accepted: err = %v", err)
	}
	bad := RegisterRequest{Design: "../escape", Implementation: p.impl, Store: p.store, ExpectedHead: p.head0}
	if _, err := Register(context.Background(), bad, io.Discard); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("escaping design root accepted: err = %v", err)
	}
	inside := RegisterRequest{Design: "app", Implementation: p.impl, Store: filepath.Join(p.impl, "store"), ExpectedHead: p.head0}
	if _, err := Register(context.Background(), inside, io.Discard); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("store inside governed roots accepted: err = %v", err)
	}
	if _, err := p.register(context.Background(), "app", nil, "sha256:"+strings.Repeat("3", 64), io.Discard); err == nil || !strings.Contains(err.Error(), "HISTORY_UNAVAILABLE") {
		t.Errorf("unknown expected head accepted: err = %v", err)
	}
}
