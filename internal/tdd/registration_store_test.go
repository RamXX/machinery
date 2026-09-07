// Frozen RED/GREEN suite for the MAC-62s6 authored-revision registration
// surface: pure lineage validation and deterministic successor construction,
// registration graph reconciliation, and the real durable store transaction
// (read reservation release, writer acquisition, exact expected-head
// compare, archiving, revalidation, compare-and-advance, idempotence and
// HEAD_CONFLICT) over real stores and real captured bundles. No mocks.
package tdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ---- registration fixture: base design + real store + real bundles ----

const (
	regSubjectV2 = "subject v2 safe-control\nanchor-line\n"
	regSubjectV3 = "subject v3 unsafe-challenge\nanchor-line\n"
	regSubjectV4 = "subject v4 red-control\nanchor-line\n"
)

// regFixture is a fully bound registration project: the frozen base design
// with finalized controls referencing REAL captured bundles in a real store.
type regFixture struct {
	design     string
	ctl        string
	store      string
	head0      string
	head0Bytes []byte
	plan       Plan
	m1         Manifest
}

// regCaptureOne captures the current subject state under the base manifest
// shape and returns the retained bundle ref.
func regCaptureOne(t *testing.T, f *regFixture, name string) string {
	t.Helper()
	ref, err := Capture(context.Background(), CaptureRequest{
		Inputs:   (&testInputView{src: f.design, ctl: f.ctl}).view(),
		Manifest: baseCaptureManifest(), Name: name, Store: f.store, Limits: captureLimits(),
	})
	if err != nil {
		t.Fatalf("capture %s: %v", name, err)
	}
	return ref.Ref()
}

func regWriteSubject(t *testing.T, design, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(design, "src.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRegFixture materializes the bound project. Subject ends at v1; controls
// are written last with real bundle refs and contract-bound reviews.
func newRegFixture(t *testing.T) *regFixture {
	t.Helper()
	f := &regFixture{design: writeBaseDesign(t), store: filepath.Join(t.TempDir(), "store")}
	init, err := InitStore(context.Background(), f.store, fixtureProjectID)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	f.head0, f.head0Bytes = init.HeadDigest, nil
	f.ctl = t.TempDir()
	if err := os.MkdirAll(filepath.Join(f.design, "assurance", "milestones"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(f.ctl, "milestones"), 0o755); err != nil {
		t.Fatal(err)
	}
	// provisional controls so the control materialization is populated
	planBytes, manifestBytes := bindBaseDocuments(t, f.design)
	regWriteControls(t, f, planBytes, manifestBytes)
	// capture the retained source states by really changing the subject
	regWriteSubject(t, f.design, regSubjectV2)
	safe := regCaptureOne(t, f, "safe-1")
	regWriteSubject(t, f.design, regSubjectV3)
	unsafe := regCaptureOne(t, f, "unsafe-1")
	regWriteSubject(t, f.design, regSubjectV4)
	safe2 := regCaptureOne(t, f, "safe-2")
	regWriteSubject(t, f.design, fixtureSrc)
	base := regCaptureOne(t, f, "baseline")
	// finalized bound controls with the real refs
	payload, err := DesignPayloadDigest(f.design)
	if err != nil {
		t.Fatalf("payload digest: %v", err)
	}
	plan := basePlan(fixtureSrcDigest(t), "placeholder")
	planReview, err := ReviewSubjectDigest(ReviewProjectionPlan(plan), payload)
	if err != nil {
		t.Fatal(err)
	}
	planBytes = renderPlan(basePlan(fixtureSrcDigest(t), planReview))
	m := baseManifest("placeholder")
	m.Baseline = base
	m.Variants[0].Source = safe
	m.Variants[1].Source = unsafe
	m.Variants[2].Source = safe2
	manifestReview, err := ReviewSubjectDigest(ReviewProjectionManifest(m), payload)
	if err != nil {
		t.Fatal(err)
	}
	m.Review = Review{Reviewer: "rev-one", Rationale: "milestone reviewed", SubjectDigest: manifestReview}
	for i := range m.Variants {
		m.Variants[i].Review = m.Review
	}
	manifestBytes = renderManifest(m)
	regWriteControls(t, f, planBytes, manifestBytes)
	if f.plan, err = LoadPlan(f.design); err != nil {
		t.Fatalf("load plan: %v", err)
	}
	if f.m1, err = LoadManifest(filepath.Join(f.design, "assurance", "milestones", "M1.json")); err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if raw, err := ReadArchivedHead(context.Background(), f.store, fixtureProjectID, f.head0); err != nil {
		t.Fatalf("read gen-0 head: %v", err)
	} else {
		f.head0Bytes = raw
	}
	return f
}

// regWriteControls writes the exact same control bytes into the design
// tree and the separate control materialization (plan.json and milestones/
// live at the control root itself).
func regWriteControls(t *testing.T, f *regFixture, planBytes, manifestBytes []byte) {
	t.Helper()
	for rel, data := range map[string][]byte{
		"plan.json":          planBytes,
		"milestones/M1.json": manifestBytes,
	} {
		if err := os.WriteFile(filepath.Join(f.design, "assurance", filepath.FromSlash(rel)), data, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(f.ctl, filepath.FromSlash(rel)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// regBindManifest re-renders and rebinds one milestone document on disk and
// reloads it (revision advance / plan edit helpers).
func regBindManifest(t *testing.T, f *regFixture, m Manifest) Manifest {
	t.Helper()
	payload, err := DesignPayloadDigest(f.design)
	if err != nil {
		t.Fatal(err)
	}
	dig, err := ReviewSubjectDigest(ReviewProjectionManifest(m), payload)
	if err != nil {
		t.Fatal(err)
	}
	m.Review = Review{Reviewer: "rev-one", Rationale: "milestone reviewed", SubjectDigest: dig}
	for i := range m.Variants {
		m.Variants[i].Review = m.Review
	}
	regWriteControls(t, f, mustRead(t, filepath.Join(f.design, "assurance", "plan.json")), renderManifest(m))
	loaded, err := LoadManifest(filepath.Join(f.design, "assurance", "milestones", "M1.json"))
	if err != nil {
		t.Fatalf("reload bound manifest: %v", err)
	}
	return loaded
}

func regInputs(f *regFixture) RegistrationInputs {
	return RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{f.m1}}
}

func regHeadArchiveCount(t *testing.T, store string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(store, "ledger", "heads"))
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// ---- successor construction and lineage ----

// AC1/AC3: first registration derives revision 1/null with the exact
// controls and a canonical, deterministic successor head.
func TestRegistrationSuccessorFirstRevision(t *testing.T) {
	f := newRegFixture(t)
	succ, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if err != nil {
		t.Fatalf("first successor: %v", err)
	}
	if succ.PreviousDigest != f.head0 || succ.Generation != 1 {
		t.Fatalf("successor lineage = (%s, gen %d), want (%s, gen 1)", succ.PreviousDigest, succ.Generation, f.head0)
	}
	if succ.HeadDigest != digestOfBytes(succ.HeadBytes) {
		t.Fatal("successor digest does not name its own bytes")
	}
	eh, err := DecodeExpectedHead(succ.HeadBytes)
	if err != nil {
		t.Fatalf("successor does not decode: %v", err)
	}
	if len(eh.Plans) != 1 || eh.Plans[0].Design != "." || eh.Plans[0].PlanDigest != digestOfBytes(f.plan.Raw()) {
		t.Fatalf("successor plans = %+v", eh.Plans)
	}
	if len(eh.Milestones) != 1 || eh.Milestones[0].Key.Milestone != "M1" || eh.Milestones[0].Revision != 1 || eh.Milestones[0].Predecessor != "" || eh.Milestones[0].ManifestDigest != digestOfBytes(f.m1.Raw()) {
		t.Fatalf("successor milestones = %+v", eh.Milestones)
	}
	if len(succ.Controls) != 2 || len(succ.TargetKeys) != 1 || succ.TargetKeys[0].Milestone != "M1" {
		t.Fatalf("successor controls/targets malformed: %d controls, %+v", len(succ.Controls), succ.TargetKeys)
	}
	// determinism: identical inputs, identical bytes
	again, _ := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if string(again.HeadBytes) != string(succ.HeadBytes) {
		t.Fatal("successor construction is not deterministic")
	}
}

// AC1: successor revision rules — revision 1/null only for absent keys,
// prior revision+1 with the exact same-key predecessor otherwise.
func TestRegistrationSuccessorLineageRules(t *testing.T) {
	f := newRegFixture(t)
	bad := f.m1
	bad.Revision = 2
	bad = regBindManifest(t, f, bad)
	if _, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{bad}}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("first revision 2 accepted: err = %v", err)
	}
	withPred := f.m1
	withPred.Predecessor = "sha256:" + strings.Repeat("ab", 32)
	withPred = regBindManifest(t, f, withPred)
	if _, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{withPred}}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("first revision with predecessor accepted: err = %v", err)
	}
	// a committed revision 1, then a legal revision 2 and illegal skips
	succ, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.store, "ledger", "heads", strings.TrimPrefix(succ.HeadDigest, "sha256:")+".json"), succ.HeadBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	next := f.m1
	next.Revision = 3
	next = regBindManifest(t, f, next)
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{next}}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("revision skip 1->3 accepted: err = %v", err)
	}
	wrongPred := f.m1
	wrongPred.Revision = 2
	wrongPred.Predecessor = "sha256:" + strings.Repeat("cd", 32)
	wrongPred = regBindManifest(t, f, wrongPred)
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{wrongPred}}); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("cross-manifest predecessor accepted: err = %v", err)
	}
	ok := f.m1
	ok.Revision = 2
	ok.Predecessor = digestOfBytes(f.m1.Raw())
	ok = regBindManifest(t, f, ok)
	got, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{ok}})
	if err != nil {
		t.Fatalf("legal revision 2 rejected: %v", err)
	}
	eh, _ := DecodeExpectedHead(got.HeadBytes)
	if eh.Generation != 2 || eh.Milestones[0].Revision != 2 || eh.Milestones[0].Predecessor != digestOfBytes(f.m1.Raw()) {
		t.Fatalf("revision-2 head = gen %d, %+v", eh.Generation, eh.Milestones[0])
	}
}

// AC3: plan change must advance every registered milestone of the design;
// untargeted registered mismatch and registered deletion block.
func TestRegistrationSuccessorPlanChangeRules(t *testing.T) {
	f := newRegFixture(t)
	succ, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.store, "ledger", "heads", strings.TrimPrefix(succ.HeadDigest, "sha256:")+".json"), succ.HeadBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	changed := f.plan
	changed.RuntimeObligations[0].Reason = "plan edited; registration must advance M1"
	payload, err := DesignPayloadDigest(f.design)
	if err != nil {
		t.Fatal(err)
	}
	dig, err := ReviewSubjectDigest(ReviewProjectionPlan(changed), payload)
	if err != nil {
		t.Fatal(err)
	}
	for i := range changed.RuntimeObligations {
		changed.RuntimeObligations[i].Review = Review{Reviewer: "rev-one", Rationale: "discovery reviewed", SubjectDigest: dig}
	}
	newPlanBytes := renderPlan(changed)
	// untargeted M1 under a changed plan is rejected
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: changed, PlanBytes: newPlanBytes, Targets: nil, Untargeted: []RegisteredManifest{{Key: MilestoneKey{Design: ".", Milestone: "M1"}, Revision: 1, Digest: digestOfBytes(f.m1.Raw()), Manifest: f.m1}}}); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("plan change without advancing registered M1 accepted: err = %v", err)
	}
	// advancing M1 under the changed plan is allowed
	adv := f.m1
	adv.Revision = 2
	adv.Predecessor = digestOfBytes(f.m1.Raw())
	adv = regBindManifest(t, f, adv)
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: changed, PlanBytes: newPlanBytes, Targets: []Manifest{adv}}); err != nil {
		t.Errorf("plan change with M1 advance rejected: %v", err)
	}
	// unchanged plan, untargeted registered control bytes drifted
	unt := []RegisteredManifest{{Key: MilestoneKey{Design: ".", Milestone: "M1"}, Revision: 1, Digest: digestOfBytes(f.m1.Raw()), Manifest: adv}}
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: nil, Untargeted: unt}); err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Errorf("untargeted registered mismatch accepted: err = %v", err)
	}
	// registered milestone removed from the plan is not implicit retirement
	shrunk := f.plan
	shrunk.Milestones = nil
	shrunkBytes := renderPlan(shrunk)
	if _, err := BuildRegistrationSuccessor(succ.HeadDigest, succ.HeadBytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: shrunk, PlanBytes: shrunkBytes, Targets: nil, Untargeted: unt}); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("registered milestone deletion accepted: err = %v", err)
	}
	// expected bytes must hash to the expected digest
	if _, err := BuildRegistrationSuccessor("sha256:"+strings.Repeat("0", 64), f.head0Bytes, regInputs(f)); err == nil || !strings.Contains(err.Error(), "CONTROL_ROLLBACK") {
		t.Errorf("expected-digest mismatch accepted: err = %v", err)
	}
	// project identity must match
	if _, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, RegistrationInputs{ProjectID: "11111111-2222-4333-8444-555555555555", Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{f.m1}}); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("project mismatch accepted: err = %v", err)
	}
}

// ---- registration graph validation ----

// AC1: finalized targets of current milestones, authoritative obligation
// keys, and reference resolution over targets plus exactly registered
// manifests (children register first; drafts cannot discharge requirements).
func TestRegistrationGraphValidation(t *testing.T) {
	f := newRegFixture(t)
	inv := baseInventory("")
	if err := ValidateRegistrationGraph(f.plan, []Manifest{f.m1}, nil, inv); err != nil {
		t.Fatalf("valid graph rejected: %v", err)
	}
	draft := f.m1
	draft.Baseline = ""
	if err := ValidateRegistrationGraph(f.plan, []Manifest{draft}, nil, inv); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("draft target accepted: err = %v", err)
	}
	invNoM1 := baseInventory("no-milestones")
	if err := ValidateRegistrationGraph(f.plan, []Manifest{f.m1}, nil, invNoM1); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("non-current milestone accepted: err = %v", err)
	}
	invented := f.m1
	invented.Obligations = append([]Obligation{{
		Key:      ObligationKey{Design: ".", Kind: KindOracleRow, Owner: "machines/Ghost.oracle.md", ID: "ghost-s1"},
		Positive: []TestRef{ref("t2")},
	}}, f.m1.Obligations...)
	if err := ValidateRegistrationGraph(f.plan, []Manifest{invented}, nil, inv); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("invented obligation key accepted: err = %v", err)
	}
	// a cross-design reference resolves only through an exactly registered
	// manifest; an unregistered child design blocks registration
	childRef := TestRef{Design: "child", Milestone: "M1", Suite: "s1", Test: "c1"}
	parent := f.m1
	parent.Obligations = append([]Obligation{{Key: ObligationKey{Design: ".", Kind: KindOracleRow, Owner: "machines/Alpha.oracle.md", ID: "alpha-s1"}, Positive: []TestRef{childRef}}}, f.m1.Obligations...)
	if err := ValidateRegistrationGraph(f.plan, []Manifest{parent}, nil, inv); err == nil || !strings.Contains(err.Error(), "MISSING_TEST") {
		t.Errorf("unregistered child reference accepted: err = %v", err)
	}
	child := baseManifest("sha256:" + strings.Repeat("0", 64))
	child.ID = "M1"
	childRaw := renderManifest(child)
	childLoaded, err := DecodeArchivedManifest("child", "M1", childRaw)
	if err != nil {
		t.Fatalf("decode archived child manifest: %v", err)
	}
	reg := []RegisteredManifest{{Key: MilestoneKey{Design: "child", Milestone: "M1"}, Revision: 1, Digest: digestOfBytes(childRaw), Manifest: childLoaded}}
	if err := ValidateRegistrationGraph(f.plan, []Manifest{parent}, reg, inv); err != nil {
		t.Errorf("registered child reference rejected: %v", err)
	}
}

// AC2 (decode surfaces): archived decoding binds the milestone identity and
// rejects non-canonical head bytes.
func TestRegistrationArchivedDecoding(t *testing.T) {
	f := newRegFixture(t)
	if _, err := DecodeArchivedManifest(".", "M2", f.m1.Raw()); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("archived manifest milestone mismatch accepted: err = %v", err)
	}
	if _, err := DecodeExpectedHead([]byte(`{"schema":"machinery.tdd.head/v1"}`)); err == nil {
		t.Error("non-canonical head bytes decoded")
	}
	eh, err := DecodeExpectedHead(f.head0Bytes)
	if err != nil || eh.Generation != 0 || eh.Digest != f.head0 || len(eh.Plans) != 0 || len(eh.Milestones) != 0 || eh.Previous != "" {
		t.Fatalf("gen-0 decode = %+v, %v", eh, err)
	}
}

// ---- durable store transaction ----

// AC2/AC3: fresh registration accepts the exact empty generation-zero head
// with no prior execution receipt, archives canonical controls/head, and
// advances exactly one generation; the store stays quiescent (exportable).
func TestCommitRegistrationFresh(t *testing.T) {
	f := newRegFixture(t)
	succ, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if err != nil {
		t.Fatal(err)
	}
	res, err := CommitRegistration(context.Background(), StoreRegistrationRequest{
		Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: succ,
		Revalidate: func() error { return nil },
	})
	if err != nil {
		t.Fatalf("fresh commit: %v", err)
	}
	if res.Idempotent || res.PreviousHead != f.head0 || res.HeadDigest != succ.HeadDigest || res.Generation != 1 {
		t.Fatalf("fresh result = %+v", res)
	}
	v, err := openStore(context.Background(), f.store, fixtureProjectID)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer v.close()
	if v.headDigest != succ.HeadDigest || v.head.Generation != 1 || len(v.head.Milestones) != 1 {
		t.Fatalf("committed head = %s gen %d", v.headDigest, v.head.Generation)
	}
	for _, c := range succ.Controls {
		raw, rerr := os.ReadFile(filepath.Join(f.store, "controls", strings.TrimPrefix(c.Digest, "sha256:")+".json"))
		if rerr != nil || digestOfBytes(raw) != c.Digest {
			t.Fatalf("control %s not archived byte-identical", c.Digest)
		}
	}
	if _, err := os.Stat(filepath.Join(f.store, "staging")); !os.IsNotExist(err) {
		t.Fatal("staging survived the transaction")
	}
	if _, err := ExportStore(context.Background(), f.store, filepath.Join(t.TempDir(), "exp.tddexp")); err != nil {
		t.Fatalf("store not quiescent after commit: %v", err)
	}
	if runs, _ := os.ReadDir(filepath.Join(f.store, "runs")); len(runs) != 0 {
		t.Fatal("registration produced execution records")
	}
}

// AC4/AC5: exact retry is idempotent (already-registered, no new
// generation); a different desired successor against the same expected head
// is HEAD_CONFLICT and never rebases.
func TestCommitRegistrationIdempotentAndConflict(t *testing.T) {
	f := newRegFixture(t)
	succ, _ := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	req := StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: succ, Revalidate: func() error { return nil }}
	if _, err := CommitRegistration(context.Background(), req); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	again, err := CommitRegistration(context.Background(), req)
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if !again.Idempotent || again.Generation != 1 || again.HeadDigest != succ.HeadDigest {
		t.Fatalf("retry result = %+v", again)
	}
	// a divergent successor from the same expected head loses the CAS
	other := f.m1
	other.Checks = []Check{{ID: "chk-2", Kind: CheckKindFormat, Profile: "go-format/v1", Inputs: []string{"tests/alpha_test.go"}}}
	other = regBindManifest(t, f, other)
	otherSucc, err := BuildRegistrationSuccessor(f.head0, f.head0Bytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{other}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CommitRegistration(context.Background(), StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: otherSucc, Revalidate: func() error { return nil }}); err == nil || !strings.Contains(err.Error(), "HEAD_CONFLICT") {
		t.Fatalf("CAS loser did not receive HEAD_CONFLICT: err = %v", err)
	}
	v, err := openStore(context.Background(), f.store, fixtureProjectID)
	if err != nil {
		t.Fatal(err)
	}
	defer v.close()
	if v.headDigest != succ.HeadDigest || v.head.Generation != 1 {
		t.Fatal("conflict disturbed the committed head")
	}
}

// AC4/AC6: concurrent registrations with one expected head — identical
// payloads collapse to one commit plus one already-registered; divergent
// payloads produce exactly one winner and one HEAD_CONFLICT loser.
func TestCommitRegistrationConcurrentWriters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		diverge bool
		want    string // "idempotent" | "conflict"
	}{
		{"identical payload", false, "idempotent"},
		{"divergent payload", true, "conflict"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRegFixture(t)
			succA, _ := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
			succB := succA
			if tc.diverge {
				other := f.m1
				other.Checks = []Check{{ID: "chk-2", Kind: CheckKindFormat, Profile: "go-format/v1", Inputs: []string{"tests/alpha_test.go"}}}
				other = regBindManifest(t, f, other)
				succB, _ = BuildRegistrationSuccessor(f.head0, f.head0Bytes, RegistrationInputs{ProjectID: fixtureProjectID, Plan: f.plan, PlanBytes: f.plan.Raw(), Targets: []Manifest{other}})
			}
			results := make(chan StoreRegistrationResult, 2)
			errs := make(chan error, 2)
			var wg sync.WaitGroup
			for _, s := range []RegistrationSuccessor{succA, succB} {
				wg.Add(1)
				go func(s RegistrationSuccessor) {
					defer wg.Done()
					r, err := CommitRegistration(context.Background(), StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: s, Revalidate: func() error { return nil }})
					results <- r
					errs <- err
				}(s)
			}
			wg.Wait()
			close(results)
			close(errs)
			var wins, idem, conflicts int
			for err := range errs {
				switch {
				case err == nil:
					wins++
				case strings.Contains(err.Error(), "HEAD_CONFLICT"):
					conflicts++
				default:
					t.Fatalf("unexpected error: %v", err)
				}
			}
			for r := range results {
				if r.Idempotent {
					idem++
				}
			}
			if wins != 1 {
				t.Fatalf("commits = %d, want exactly 1", wins)
			}
			if tc.want == "idempotent" && idem != 1 {
				t.Fatalf("idempotent confirmations = %d, want 1", idem)
			}
			if tc.want == "conflict" && conflicts != 1 {
				t.Fatalf("HEAD_CONFLICT results = %d, want 1", conflicts)
			}
			if _, err := os.Stat(filepath.Join(f.store, "staging")); !os.IsNotExist(err) {
				t.Fatal("staging survived concurrent transactions")
			}
		})
	}
}

// AC4/AC5: a failed final revalidation (real mid-transaction source change)
// aborts without advancing the head and leaves no staged successor archive;
// precommit errors consume no revision.
func TestCommitRegistrationRevalidateAborts(t *testing.T) {
	f := newRegFixture(t)
	succ, _ := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	before := regHeadArchiveCount(t, f.store)
	snap := f.design
	_, err := CommitRegistration(context.Background(), StoreRegistrationRequest{
		Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: succ,
		Revalidate: func() error {
			regWriteSubject(t, snap, regSubjectV2) // real source mutation at the transaction point
			return fmt.Errorf("STALE_INPUT: design payload changed during registration")
		},
	})
	if err == nil || !strings.Contains(err.Error(), "STALE_INPUT") {
		t.Fatalf("mutating revalidation accepted: err = %v", err)
	}
	if regHeadArchiveCount(t, f.store) != before {
		t.Fatal("aborted transaction staged a successor head archive")
	}
	v, verr := openStore(context.Background(), f.store, fixtureProjectID)
	if verr != nil {
		t.Fatal(verr)
	}
	defer v.close()
	if v.headDigest != f.head0 || v.head.Generation != 0 {
		t.Fatal("aborted transaction advanced the head")
	}
}

// AC6: missing retained bundles block; a missing expected-head archive is
// HISTORY_UNAVAILABLE; cancellation is honored with no state change.
func TestCommitRegistrationFailureModes(t *testing.T) {
	f := newRegFixture(t)
	succ, _ := BuildRegistrationSuccessor(f.head0, f.head0Bytes, regInputs(f))
	if _, err := CommitRegistration(context.Background(), StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: "sha256:" + strings.Repeat("9", 64), Successor: succ, Revalidate: func() error { return nil }}); err == nil || !strings.Contains(err.Error(), "HISTORY_UNAVAILABLE") {
		t.Errorf("missing expected-head archive accepted: err = %v", err)
	}
	missing := succ
	missing.BundleRefs = []string{"sha256:" + strings.Repeat("7", 64)}
	if _, err := CommitRegistration(context.Background(), StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: missing, Revalidate: func() error { return nil }}); err == nil || !strings.Contains(err.Error(), "MISSING_CONTRACT") {
		t.Errorf("missing retained bundle accepted: err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CommitRegistration(ctx, StoreRegistrationRequest{Store: f.store, ProjectID: fixtureProjectID, ExpectedHead: f.head0, Successor: succ, Revalidate: func() error { return nil }}); err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("cancelled commit accepted: err = %v", err)
	}
	v, err := openStore(context.Background(), f.store, fixtureProjectID)
	if err != nil {
		t.Fatal(err)
	}
	defer v.close()
	if v.headDigest != f.head0 {
		t.Fatal("failed commits advanced the head")
	}
	if _, err := os.Stat(filepath.Join(f.store, "staging")); !os.IsNotExist(err) {
		t.Fatal("failed commits left transaction staging behind")
	}
}
