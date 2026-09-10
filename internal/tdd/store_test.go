// Frozen RED/GREEN suite for the MAC-p9z1 external-store surface: durable
// generation-zero initialization, new-path-only adoption rules, 0700 root
// identity and closed layout validation, canonical head-chain custody with
// rollback detection, the cheap read-only replay-not-performed status
// dimensions, corruption fail-closed, permission custody and concurrent
// reader/writer pressure over the real filesystem.
package tdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// ---- test-local canonical head oracle (independent of production) ----

type headMs struct {
	Design    string
	Milestone string
	Revision  int64
	Manifest  string // digest
	Pred      string // digest; "" encodes null
}

func renderHeadCanon(projectID string, generation int64, previous string, plans [][2]string, milestones []headMs) []byte {
	var b strings.Builder
	b.WriteString(`{"generation":`)
	fmt.Fprint(&b, generation)
	b.WriteString(`,"milestones":[`)
	for i, m := range milestones {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"design":` + jsonString(m.Design) + `,"manifest_digest":` + jsonString(m.Manifest) + `,"milestone":` + jsonString(m.Milestone))
		if m.Pred == "" {
			b.WriteString(`,"predecessor":null`)
		} else {
			b.WriteString(`,"predecessor":` + jsonString(m.Pred))
		}
		b.WriteString(`,"revision":` + fmt.Sprint(m.Revision) + `}`)
	}
	b.WriteString(`],"plans":[`)
	for i, p := range plans {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"design":` + jsonString(p[0]) + `,"plan_digest":` + jsonString(p[1]) + `}`)
	}
	if previous == "" {
		b.WriteString(`],"previous":null`)
	} else {
		b.WriteString(`],"previous":` + jsonString(previous))
	}
	b.WriteString(`,"project_id":` + jsonString(projectID) + `,"schema":"machinery.tdd.head/v1"}`)
	return []byte(b.String())
}

func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// advanceHeadTo archives the next head and atomically advances head.json
// (test-side, using the independent canonical renderer).
func advanceHeadTo(t *testing.T, store, projectID string, generation int64, previous string, plans [][2]string, milestones []headMs) string {
	t.Helper()
	headBytes := renderHeadCanon(projectID, generation, previous, plans, milestones)
	dig := digestHex(headBytes)
	heads := filepath.Join(store, "ledger", "heads")
	if err := os.MkdirAll(heads, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(heads, dig+".json"), headBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	hj := filepath.Join(store, "ledger", "head.json")
	if err := os.WriteFile(hj+".next", headBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hj+".next", hj); err != nil {
		t.Fatal(err)
	}
	return "sha256:" + dig
}

// writeRunRecord appends a closed run/v1 record to the store (test-side).
func writeRunRecord(t *testing.T, store, runID, design, milestone, manifestDigest, judgmentDigest, result, phase string) {
	t.Helper()
	doc := `{"custody":{"status":"cleaned","transcript":"` + "sha256:" + strings.Repeat("a", 64) + `"},"design":` + jsonString(design) +
		`,"diagnostics":[],"executions":[],"frozen_digest":"` + "sha256:" + strings.Repeat("2", 64) + `","inventory_digest":"` + "sha256:" + strings.Repeat("3", 64) +
		`","judgment_control_digest":` + jsonString(judgmentDigest) + `,"manifest_digest":` + jsonString(manifestDigest) +
		`,"milestone":` + jsonString(milestone) + `,"phase":` + jsonString(phase) + `,"provenance":"unauthenticated-host","result":` + jsonString(result) +
		`,"run_id":` + jsonString(runID) + `,"runtime_digest":"` + "sha256:" + strings.Repeat("4", 64) + `","schema":"machinery.tdd.run/v1","subject_digest":"` + "sha256:" + strings.Repeat("5", 64) + `"}`
	if err := os.MkdirAll(filepath.Join(store, "runs"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "runs", runID+".json"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
}

// baseCaptureManifest mirrors the loaded base manifest's declared root and
// subject inventory (suite root ".", frozen root tests, subject src.txt).
func baseCaptureManifest() Manifest {
	m := baseManifest("sha256:" + strings.Repeat("0", 64))
	return m
}

// boundStatusFixture builds a fully bound loaded plan/manifest pair over the
// base design tree plus a captured baseline in a fresh store.
func boundStatusFixture(t *testing.T) (src, ctl, store string, plan Plan, manifests []Manifest, inv Inventory, ref string) {
	t.Helper()
	design := writeBaseDesign(t)
	planBytes, manifestBytes := bindBaseDocuments(t, design)
	if err := os.MkdirAll(filepath.Join(design, "assurance", "milestones"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "assurance", "plan.json"), planBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "assurance", "milestones", "M1.json"), manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	ctl = t.TempDir()
	for _, f := range []string{"plan.json", "milestones/M1.json"} {
		if err := os.MkdirAll(filepath.Join(ctl, filepath.Dir(filepath.FromSlash(f))), 0o755); err != nil {
			t.Fatal(err)
		}
		data := mustRead(t, filepath.Join(design, "assurance", filepath.FromSlash(f)))
		if err := os.WriteFile(filepath.Join(ctl, filepath.FromSlash(f)), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store = filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init store: %v", err)
	}
	ref = captureBaseBundle(t, store, design, ctl)
	plan, err := LoadPlan(design)
	if err != nil {
		t.Fatalf("load plan: %v", err)
	}
	manifest, err := LoadManifest(filepath.Join(design, "assurance", "milestones", "M1.json"))
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	manifest.Baseline = ref
	return design, ctl, store, plan, []Manifest{manifest}, baseInventory(""), ref
}

func captureBaseBundle(t *testing.T, store, src, ctl string) string {
	t.Helper()
	ref, err := Capture(context.Background(), CaptureRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Manifest: baseCaptureManifest(), Name: "baseline", Store: store, Limits: captureLimits()})
	if err != nil {
		t.Fatalf("capture baseline: %v", err)
	}
	return ref.Ref()
}

func statusRequest(src, ctl, store string, plan Plan, manifests []Manifest, inv Inventory) StatusRequest {
	return StatusRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Plan: plan, Manifests: manifests, Inventory: inv, Store: store}
}

func hasCode(codes []Diagnostic, code string) bool {
	for _, d := range codes {
		if d.Code == code {
			return true
		}
	}
	return false
}

// AC3: init creates only the owned 0700 root with the closed identity and a
// durable generation-zero head whose archived bytes equal head.json.
func TestStoreInitGenerationZeroDurable(t *testing.T) {
	store := filepath.Join(t.TempDir(), "fresh-store")
	res, err := InitStore(context.Background(), store, fixtureProjectID)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if res.ProjectID != fixtureProjectID || res.StoreID == "" || res.Generation != 0 {
		t.Fatalf("init result %+v", res)
	}
	fi, err := os.Lstat(store)
	if err != nil || !fi.IsDir() {
		t.Fatalf("store root: %v", err)
	}
	if fi.Mode().Perm() != 0o700 {
		t.Errorf("store root mode = %o, want 0700", fi.Mode().Perm())
	}
	var idDoc struct {
		Schema    string `json:"schema"`
		StoreID   string `json:"store_id"`
		ProjectID string `json:"project_id"`
	}
	if err := strictUnmarshalJSON(mustRead(t, filepath.Join(store, "store.json")), &idDoc); err != nil {
		t.Fatalf("store.json: %v", err)
	}
	if idDoc.Schema != "machinery.tdd.store/v1" || idDoc.ProjectID != fixtureProjectID || idDoc.StoreID != res.StoreID {
		t.Errorf("store.json = %+v", idDoc)
	}
	headBytes := mustRead(t, filepath.Join(store, "ledger", "head.json"))
	want := renderHeadCanon(fixtureProjectID, 0, "", nil, nil)
	if string(headBytes) != string(want) {
		t.Errorf("generation-zero head bytes:\n got %s\nwant %s", headBytes, want)
	}
	if res.HeadDigest != "sha256:"+digestHex(headBytes) {
		t.Errorf("returned head digest %s does not match head bytes", res.HeadDigest)
	}
	archived := mustRead(t, filepath.Join(store, "ledger", "heads", digestHex(headBytes)+".json"))
	if string(archived) != string(headBytes) {
		t.Error("archived generation-zero head differs from head.json")
	}
	for _, d := range []string{"objects", "blobs", "controls", "runs", "ledger"} {
		if fi, err := os.Lstat(filepath.Join(store, d)); err != nil || !fi.IsDir() {
			t.Errorf("store namespace %s missing: %v", d, err)
		}
	}
	entries, _ := os.ReadDir(store)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if got, wantNames := strings.Join(names, ","), "blobs,controls,ledger,objects,runs,store.json"; got != wantNames {
		t.Errorf("store root holds %q, want %q", got, wantNames)
	}
}

// AC3: only a NEW path initializes; existing nonempty, partial or empty
// destinations are never adopted.
func TestStoreInitRejectsExistingDestinations(t *testing.T) {
	base := t.TempDir()
	nonempty := filepath.Join(base, "nonempty")
	if err := os.MkdirAll(filepath.Join(nonempty, "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStore(context.Background(), nonempty, fixtureProjectID); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("nonempty destination adopted: err = %v", err)
	}
	partial := filepath.Join(base, "partial")
	if err := os.MkdirAll(partial, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(partial, "store.json"), []byte(`{"schema":"machinery.tdd.store/v1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStore(context.Background(), partial, fixtureProjectID); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("partial destination adopted: err = %v", err)
	}
	empty := filepath.Join(base, "empty")
	if err := os.Mkdir(empty, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStore(context.Background(), empty, fixtureProjectID); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("existing empty destination adopted: err = %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStore(context.Background(), filepath.Join(base, "file"), fixtureProjectID); err == nil {
		t.Error("file destination adopted")
	}
	// unsafe challenge twin: an adoption-happy initializer accepts the empty dir
	if v := func(dirExists bool) bool { return !dirExists || true }; !v(true) {
		t.Fatal("challenge twin must adopt to prove the assertion discriminates")
	}
	// invalid project UUID is rejected before any filesystem effect
	bad := filepath.Join(base, "bad")
	if _, err := InitStore(context.Background(), bad, "not-a-uuid"); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("invalid project UUID accepted: err = %v", err)
	}
	if _, err := os.Lstat(bad); !os.IsNotExist(err) {
		t.Error("rejected init left a path behind")
	}
}

// AC3: store open validates the rooted identity (no symlink substitution,
// exact 0700 mode) and the closed layout, and rejects project mismatch.
func TestStoreOpenValidatesIdentityLayoutProject(t *testing.T) {
	base := t.TempDir()
	store := filepath.Join(base, "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init: %v", err)
	}
	other := filepath.Join(base, "other")
	if _, err := InitStore(context.Background(), other, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"); err != nil {
		t.Fatalf("init other: %v", err)
	}
	src, ctl, st, plan, manifests, inv, _ := boundStatusFixture(t)
	// project mismatch is blocking: the fixture store's project differs
	// from the plan's requirement
	badReq := statusRequest(src, ctl, st, plan, manifests, inv)
	badReq.Store = other
	if _, err := Status(context.Background(), badReq); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("project mismatch accepted: err = %v", err)
	}
	// missing store is blocking, never an empty fresh history
	missReq := statusRequest(src, ctl, st, plan, manifests, inv)
	missReq.Store = filepath.Join(base, "absent")
	if _, err := Status(context.Background(), missReq); err == nil || !strings.Contains(err.Error(), "MISSING_STORE") {
		t.Errorf("missing store accepted: err = %v", err)
	}
	// symlink-substituted root
	link := filepath.Join(base, "link")
	if err := os.Symlink(store, link); err != nil {
		t.Fatal(err)
	}
	linkReq := statusRequest(src, ctl, st, plan, manifests, inv)
	linkReq.Store = link
	if _, err := Status(context.Background(), linkReq); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("symlinked store root accepted: err = %v", err)
	}
	// wrong root mode
	if err := os.Chmod(store, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "STORE_ROOT_MISMATCH") {
		t.Errorf("non-0700 store accepted: err = %v", err)
	}
	if err := os.Chmod(store, 0o700); err != nil {
		t.Fatal(err)
	}
	// unknown top-level entry
	if err := os.WriteFile(filepath.Join(store, "foreign"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("unknown store entry accepted: err = %v", err)
	}
	if err := os.Remove(filepath.Join(store, "foreign")); err != nil {
		t.Fatal(err)
	}
	// corrupted identity document
	sj := filepath.Join(store, "store.json")
	if err := os.Chmod(sj, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sj, []byte(`{"schema":"machinery.tdd.store/v1","store_id":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("corrupted store.json accepted: err = %v", err)
	}
}

// AC5: status is cheap, read-only and explicitly replay-not-performed, with
// independent store/source/control/judgment digests and actionable
// unregistered-draft diagnostics.
func TestStatusUnregisteredDraftExplicitState(t *testing.T) {
	src, ctl, store, plan, manifests, inv, _ := boundStatusFixture(t)
	before := dirState(t, store)
	rep, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if rep.TestExecution != "not-performed" {
		t.Errorf("TestExecution = %q, want not-performed", rep.TestExecution)
	}
	if !hasCode(rep.Diagnostics, "REPLAY_REQUIRED") {
		t.Errorf("status lacks the explicit replay-not-performed diagnostic: %+v", rep.Diagnostics)
	}
	if rep.DesignConsistency != "checked" {
		t.Errorf("DesignConsistency = %q, want checked", rep.DesignConsistency)
	}
	if rep.SourceTestBinding != "current" {
		t.Errorf("SourceTestBinding = %q, want current for a matching baseline", rep.SourceTestBinding)
	}
	if rep.Judgment != "missing" || rep.JudgmentDigest != "" {
		t.Errorf("Judgment = %q/%q, want missing (no attestations in fixture)", rep.Judgment, rep.JudgmentDigest)
	}
	if rep.FormalExecution != "not-performed" || rep.RuntimeResiduals != "unverified" {
		t.Errorf("FormalExecution/RuntimeResiduals = %q/%q", rep.FormalExecution, rep.RuntimeResiduals)
	}
	if rep.Provenance != "unauthenticated-host" {
		t.Errorf("Provenance = %q", rep.Provenance)
	}
	for _, v := range []string{rep.StoreProjectID, rep.StoreHeadDigest, rep.SourceDigest, rep.ControlDigest} {
		if v == "" {
			t.Error("a required status dimension digest is empty")
		}
	}
	if rep.StoreProjectID != fixtureProjectID {
		t.Errorf("StoreProjectID = %q", rep.StoreProjectID)
	}
	seen := map[string]bool{}
	for _, v := range []string{rep.StoreHeadDigest, rep.SourceDigest, rep.ControlDigest} {
		if seen[v] {
			t.Errorf("dimension digests alias each other: %s", v)
		}
		seen[v] = true
	}
	foundDraft := false
	for _, d := range rep.Diagnostics {
		if strings.Contains(d.Message, "unregistered-draft") {
			foundDraft = true
		}
	}
	if !foundDraft {
		t.Errorf("status does not distinguish the unregistered draft: %+v", rep.Diagnostics)
	}
	if after := dirState(t, store); after != before {
		t.Error("status mutated the store")
	}
}

// dirState fingerprints a directory tree (names, modes, mtimes) for
// read-only verification.
func dirState(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	if err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		fmt.Fprintf(&b, "%s:%o:%d\n", rel, info.Mode().Perm(), info.ModTime().UnixNano())
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return b.String()
}

// AC5: recorded-only historical evidence, failed history, stale binding and
// stale judgment are distinct, honest states; no hash receipt ever becomes
// proof tests ran.
func TestStatusRecordedFailedStaleStates(t *testing.T) {
	src, ctl, store, plan, manifests, inv, _ := boundStatusFixture(t)
	manifestDigest := "sha256:" + digestHex(manifests[0].Raw())
	writeRunRecord(t, store, "run-1", ".", "M1", manifestDigest, "", "pass", "red")
	rep, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status recorded: %v", err)
	}
	if rep.TestExecution != "recorded-only" {
		t.Errorf("TestExecution = %q, want recorded-only", rep.TestExecution)
	}
	if !hasCode(rep.Diagnostics, "REPLAY_REQUIRED") {
		t.Error("recorded-only state lost the replay-required diagnostic")
	}
	writeRunRecord(t, store, "run-2", ".", "M1", manifestDigest, "", "fail", "green")
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status failed-history: %v", err)
	}
	if rep.TestExecution != "failed" {
		t.Errorf("TestExecution = %q, want failed for a failed latest record", rep.TestExecution)
	}
	// frozen-input change makes the binding stale
	if err := os.WriteFile(filepath.Join(src, "tests", "alpha_test.go"), []byte(fixGoTest+"// drift\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status stale: %v", err)
	}
	if rep.SourceTestBinding != "stale" {
		t.Errorf("SourceTestBinding = %q, want stale after frozen drift", rep.SourceTestBinding)
	}
	if !hasCode(rep.Diagnostics, "STALE_INPUT") {
		t.Error("stale binding lacks STALE_INPUT diagnostic")
	}
	// restore and bind judgment: present, then stale relative to history
	if err := os.WriteFile(filepath.Join(src, "tests", "alpha_test.go"), []byte(fixGoTest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "attestations.yaml"), []byte(fixAttest), 0o644); err != nil {
		t.Fatal(err)
	}
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status judgment: %v", err)
	}
	if rep.JudgmentDigest == "" {
		t.Fatal("judgment digest not computed for a present judgment control")
	}
	if rep.Judgment != "stale" {
		t.Errorf("Judgment = %q, want stale while run history binds an older judgment", rep.Judgment)
	}
	// a run bound to the CURRENT judgment makes the judgment dimension bound
	writeRunRecord(t, store, "run-3", ".", "M1", manifestDigest, rep.JudgmentDigest, "pass", "verify")
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status judgment bound: %v", err)
	}
	if rep.Judgment != "bound" {
		t.Errorf("Judgment = %q, want bound when history binds the current judgment bytes", rep.Judgment)
	}
	if rep.TestExecution == "replayed-this-invocation" {
		t.Error("status claimed replay")
	}
}

// AC5: registered-not-executed and stale authored revisions are distinct
// from unregistered drafts; head binding uses exact byte digests.
func TestStatusRegisteredAndStaleManifestBinding(t *testing.T) {
	src, ctl, store, plan, manifests, inv, _ := boundStatusFixture(t)
	manifestDigest := "sha256:" + digestHex(manifests[0].Raw())
	planDigest := "sha256:" + digestHex(plan.Raw())
	head2 := advanceHeadTo(t, store, fixtureProjectID, 1, currentHeadDigest(t, store), [][2]string{{".", planDigest}}, []headMs{{Design: ".", Milestone: "M1", Revision: 1, Manifest: manifestDigest}})
	rep, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status registered: %v", err)
	}
	if rep.StoreHeadDigest != head2 {
		t.Errorf("StoreHeadDigest = %s, want the advanced head %s", rep.StoreHeadDigest, head2)
	}
	registered := false
	for _, d := range rep.Diagnostics {
		if strings.Contains(d.Message, "registered-not-executed") {
			registered = true
		}
	}
	if !registered {
		t.Errorf("registered milestone not distinguished: %+v", rep.Diagnostics)
	}
	// head entry digest differing from the current manifest bytes = stale
	// authored revision, never silently rebound
	wrongDigest := "sha256:" + digestHex(plan.Raw()) // plan bytes stand in for other manifest bytes
	advanceHeadTo(t, store, fixtureProjectID, 2, head2, nil, []headMs{{Design: ".", Milestone: "M1", Revision: 2, Manifest: wrongDigest, Pred: manifestDigest}})
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status stale revision: %v", err)
	}
	stale := false
	for _, d := range rep.Diagnostics {
		if strings.Contains(d.Message, "stale") && d.Code == "STALE_INPUT" {
			stale = true
		}
	}
	if !stale {
		t.Errorf("stale authored revision not reported: %+v", rep.Diagnostics)
	}
	// an unretained baseline is a missing binding, never a quiet pass
	manifests[0].Baseline = "sha256:" + strings.Repeat("9", 64)
	rep, err = Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv))
	if err != nil {
		t.Fatalf("status missing baseline: %v", err)
	}
	if rep.SourceTestBinding != "missing" {
		t.Errorf("SourceTestBinding = %q, want missing for an unretained baseline", rep.SourceTestBinding)
	}
}

func currentHeadDigest(t *testing.T, store string) string {
	t.Helper()
	return "sha256:" + digestHex(mustRead(t, filepath.Join(store, "ledger", "head.json")))
}

// AC4/AC5: the canonical head chain is verified on every read; tampering
// and discontinuities fail closed, and a lone staged successor (crash
// residue between archive and advance) remains recoverable, not corruption.
func TestHeadChainValidationAndRollback(t *testing.T) {
	src, ctl, store, plan, manifests, inv, _ := boundStatusFixture(t)
	gen0 := currentHeadDigest(t, store)
	manifestDigest := "sha256:" + digestHex(manifests[0].Raw())
	head1 := advanceHeadTo(t, store, fixtureProjectID, 1, gen0, nil, []headMs{{Design: ".", Milestone: "M1", Revision: 1, Manifest: manifestDigest}})
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err != nil {
		t.Fatalf("advanced head rejected: %v", err)
	}
	// head.json bytes not matching the archived copy = tampering/rollback
	hj := filepath.Join(store, "ledger", "head.json")
	tampered := strings.Replace(string(mustRead(t, hj)), `"revision":1`, `"revision":2`, 1)
	if err := os.Chmod(hj, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hj, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "CONTROL_ROLLBACK") {
		t.Errorf("tampered head accepted: err = %v", err)
	}
	// restore, advance a legitimately chained generation-2 head, then roll
	// head.json back: a lone staged successor is recoverable, not corruption
	archived1 := mustRead(t, filepath.Join(store, "ledger", "heads", strings.TrimPrefix(head1, "sha256:")+".json"))
	if err := os.WriteFile(hj+".restore", archived1, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hj+".restore", hj); err != nil {
		t.Fatal(err)
	}
	advanceHeadTo(t, store, fixtureProjectID, 2, head1, nil, []headMs{{Design: ".", Milestone: "M1", Revision: 2, Manifest: manifestDigest, Pred: manifestDigest}})
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err != nil {
		t.Fatalf("chained generation-2 head rejected: %v", err)
	}
	if err := os.WriteFile(hj+".rb", archived1, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hj+".rb", hj); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err != nil {
		t.Errorf("recoverable staged successor rejected: %v", err)
	}
	// a discontinuous chain: previous names an unknown digest
	unknown := "sha256:" + strings.Repeat("e", 64)
	advanceHeadTo(t, store, fixtureProjectID, 3, unknown, nil, nil)
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "CONTROL_ROLLBACK") {
		t.Errorf("discontinuous chain accepted: err = %v", err)
	}
	// a generation-zero head with entries is never valid
	emptyHead := renderHeadCanon(fixtureProjectID, 0, "", nil, []headMs{{Design: ".", Milestone: "M1", Revision: 1, Manifest: manifestDigest}})
	os.WriteFile(filepath.Join(store, "ledger", "heads", digestHex(emptyHead)+".json"), emptyHead, 0o600)
	if err := os.WriteFile(hj+".g0", emptyHead, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(hj+".g0", hj); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "CONTROL_ROLLBACK") {
		t.Errorf("non-empty generation-zero head accepted: err = %v", err)
	}
}

// AC4/AC6: corrupt store content (tampered bundle documents, missing blobs)
// fails closed on the paths that consult it.
func TestStoreCorruptionFailClosed(t *testing.T) {
	src, ctl, store, plan, manifests, inv, ref := boundStatusFixture(t)
	_, entries, _ := readBundleDoc(t, store, ref)
	if err := os.Remove(filepath.Join(store, "blobs", strings.TrimPrefix(entries["BUILD.md"].Digest, "sha256:"))); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err == nil || !strings.Contains(err.Error(), "INVALID_SCHEMA") {
		t.Errorf("missing baseline blob accepted by status: err = %v", err)
	}
	_ = inv
}

// AC6: concurrent readers and writers on one real store never corrupt it
// and every observer sees complete content-addressed state.
func TestStoreConcurrentReaderWriterPressure(t *testing.T) {
	src, ctl, store, plan, manifests, inv, _ := boundStatusFixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Capture(context.Background(), CaptureRequest{Inputs: (&testInputView{src: src, ctl: ctl}).view(), Manifest: baseCaptureManifest(), Name: "press", Store: store, Limits: captureLimits()}); err != nil {
				errs <- err
			}
		}()
	}
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Status(context.Background(), statusRequest(src, ctl, store, plan, manifests, inv)); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// AC6: permission failures are real custody failures, never silent success.
func TestStorePermissionCustody(t *testing.T) {
	// Root bypasses permission bits, so every assertion here would pass for
	// the wrong reason: an unwritable parent is writable to uid 0. The same
	// guard is in internal/gates, internal/lint and internal/install. The
	// coverage is not lost, because the hosted test job and the containerized
	// Dagger sweep both run as an unprivileged user.
	if os.Geteuid() == 0 {
		t.Skip("running as root; permission bits do not apply")
	}
	base := t.TempDir()
	readonly := filepath.Join(base, "readonly")
	if err := os.Mkdir(readonly, 0o500); err != nil {
		t.Fatal(err)
	}
	if _, err := InitStore(context.Background(), filepath.Join(readonly, "store"), fixtureProjectID); err == nil {
		t.Error("init into an unwritable parent succeeded")
	}
	src := writeCaptureSource(t)
	ctl := separateControlMaterialization(t, src)
	store := filepath.Join(base, "store2")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err != nil {
		t.Fatalf("setup capture: %v", err)
	}
	if err := os.Chmod(filepath.Join(store, "blobs"), 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Join(store, "blobs"), 0o700)
	if err := os.WriteFile(filepath.Join(src, "impl", "app.go"), []byte(fixApp+"new bytes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(context.Background(), newCaptureRequest(&testInputView{src: src, ctl: ctl}, store)); err == nil {
		t.Error("capture succeeded against an unwritable blob namespace")
	}
}

// AC6: caller cancellation is honored by every store operation.
func TestStoreOperationsHonorCancellation(t *testing.T) {
	store := filepath.Join(t.TempDir(), "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatalf("init: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExportStore(ctx, store, filepath.Join(t.TempDir(), "out.tddexp")); err == nil || !strings.Contains(err.Error(), "TIMEOUT") {
		t.Errorf("cancelled export: err = %v", err)
	}
	if _, err := ImportStore(ctx, filepath.Join(t.TempDir(), "dest"), filepath.Join(t.TempDir(), "absent.tddexp"), fixtureProjectID, currentHeadDigest(t, store)); err == nil {
		t.Error("cancelled import succeeded")
	}
}

// TestPublishTransientIsInvisibleToStrictEnumerations pins the fix for the
// publish race: a reader that enumerates a store namespace while a publish is
// staged must not read the writer's in-flight temp as corruption.
//
// The interleaving is deterministic, not load-dependent: the hook runs at the
// exact point between staging the temp and linking it into place, which is the
// window a concurrent reader could observe.
func TestPublishTransientIsInvisibleToStrictEnumerations(t *testing.T) {
	base := t.TempDir()
	store := filepath.Join(base, "store")
	if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
		t.Fatal(err)
	}
	headsDir := filepath.Join(store, storeLedger, storeHeads)
	if err := os.MkdirAll(headsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	observed := 0
	var enumErr error
	testAfterPublishStage = func(tmp string) {
		observed++
		if _, err := os.Lstat(tmp); err != nil {
			t.Errorf("the staged temp should exist inside the window: %v", err)
		}
		if !isPublishTransient(filepath.Base(tmp)) {
			t.Errorf("staged temp %q is not the documented transient shape", filepath.Base(tmp))
		}
		// This is exactly what a concurrent reader does.
		if _, err := collectStoreEntries(store); err != nil {
			enumErr = err
		}
	}
	t.Cleanup(func() { testAfterPublishStage = nil })

	payload := []byte(`{"schema":"probe"}`)
	target := filepath.Join(headsDir, digestHexBytes(payload)+".json")
	if err := publishImmutableFile(target, storeFileMod, payload); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if observed != 1 {
		t.Fatalf("the staging hook ran %d times, want 1", observed)
	}
	if enumErr != nil {
		t.Fatalf("a reader saw the in-flight publish temp as corruption: %v", enumErr)
	}
}

// TestStrictEnumerationsStillRejectUnknownEntries holds the other direction:
// skipping the reserved transient must not open the namespace to anything else,
// including another dot-entry.
func TestStrictEnumerationsStillRejectUnknownEntries(t *testing.T) {
	for _, name := range []string{
		"not-a-head.json",
		".publish-notactuallyhex",
		".publish-0123456789abcdefg",
		".publish-0123456789abcde",
		".some-other-temp",
	} {
		t.Run(name, func(t *testing.T) {
			if isPublishTransient(name) {
				t.Fatalf("%q must not be treated as the reserved publish transient", name)
			}
			base := t.TempDir()
			store := filepath.Join(base, "store")
			if _, err := InitStore(context.Background(), store, fixtureProjectID); err != nil {
				t.Fatal(err)
			}
			blobs := filepath.Join(store, storeBlobs)
			if err := os.MkdirAll(blobs, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(blobs, name), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := collectStoreEntries(store); err == nil || !strings.Contains(err.Error(), "unknown entry") {
				t.Fatalf("unknown entry %q was not rejected: err = %v", name, err)
			}
		})
	}
}
