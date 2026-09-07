// Authored-revision registration surface of docs/test-assurance-contract.md
// section 4: pure lineage validation and deterministic next-head
// construction plus the durable store registration transaction. tdd owns
// the derivation; internal/assuranceflow owns commit authority through
// CommitRegistration. The transaction runs under the fixed 600000 ms
// non-execution owner deadline with one 10000 ms cleanup grace, launches no
// subprocesses, never retries automatically and NEVER advances the head for
// any execution/status operation — only Register commits authored revisions.
package tdd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// Registration state literals of the authored state machine. Registration
// proves authored state only; it is never execution evidence.
const (
	StateRegisteredNotExecuted        = "registered-not-executed"
	StateAlreadyRegisteredNotExecuted = "already-registered-not-executed"
)

// RegisteredManifest binds one milestone entry of an expected head with the
// exact bytes that entry names (archived control bytes for cross-design
// resolution, current loaded bytes for untargeted same-design entries).
type RegisteredManifest struct {
	Key      MilestoneKey
	Revision int64
	Digest   string
	Manifest Manifest
}

// HeadPlanEntry is one plan entry of a decoded head.
type HeadPlanEntry struct {
	Design     string
	PlanDigest string
}

// HeadMilestoneEntry is one milestone entry of a decoded head.
type HeadMilestoneEntry struct {
	Key            MilestoneKey
	Revision       int64
	ManifestDigest string
	Predecessor    string // "" encodes null
}

// ExpectedHead is the decoded closed view of one archived head document.
type ExpectedHead struct {
	ProjectID  string
	Generation int64
	Digest     string
	Previous   string
	Plans      []HeadPlanEntry
	Milestones []HeadMilestoneEntry
}

// DecodeArchivedManifest decodes exact archived control bytes for one head
// milestone entry. Review binding was verified when that revision was
// registered; archived bytes are immutable, so the projection is not
// re-bound against a payload this checkout may not hold. The repository-
// root design "." resolves through its own checkout (LoadManifest), never
// through archived decoding without a payload.
func DecodeArchivedManifest(design, milestone string, raw []byte) (Manifest, error) {
	if !validMilestoneID(milestone) {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived milestone %q is not canonical", milestone)
	}
	if design == protocol.RepositoryRoot {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: the repository-root design resolves through its own checkout, not archived decoding without a payload")
	}
	if err := validateRootPath(design); err != nil {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived design root: %w", err)
	}
	msPath := filepath.Join(protocol.ControlDirName, protocol.MilestonesDirName, milestone+".json")
	_, doc, err := readControlJSONFromBytes(msPath, raw)
	if err != nil {
		return Manifest{}, err
	}
	m, err := decodeManifest(doc, msPath, design, "", "", raw)
	if err != nil {
		return Manifest{}, err
	}
	if m.ID != milestone {
		return Manifest{}, fmt.Errorf("INVALID_SCHEMA: archived manifest for milestone %q declares milestone %q", milestone, m.ID)
	}
	return m, nil
}

// DecodeExpectedHead decodes closed archived head bytes with full chain-rule
// validation; the bytes must be the canonical encoding.
func DecodeExpectedHead(raw []byte) (ExpectedHead, error) {
	h, err := decodeHeadClosed(raw)
	if err != nil {
		return ExpectedHead{}, err
	}
	eh := ExpectedHead{ProjectID: h.ProjectID, Generation: h.Generation, Digest: digestOfBytes(raw), Previous: h.Previous}
	for _, p := range h.Plans {
		eh.Plans = append(eh.Plans, HeadPlanEntry{Design: p.Design, PlanDigest: p.PlanDigest})
	}
	for _, m := range h.Milestones {
		eh.Milestones = append(eh.Milestones, HeadMilestoneEntry{
			Key:      MilestoneKey{Design: m.Design, Milestone: m.Milestone},
			Revision: m.Revision, ManifestDigest: m.ManifestDi, Predecessor: m.PredecessorDi,
		})
	}
	return eh, nil
}

// ValidateRegistrationGraph validates the registration transaction shape:
// selected targets are finalized manifests of current inventory milestones,
// obligation keys are authoritative (never invented), and every referenced
// test resolves to a selected target or an exactly registered manifest.
// Strict completeness obligations belong to complete/strict, not register.
func ValidateRegistrationGraph(plan Plan, targets []Manifest, registered []RegisteredManifest, inv Inventory) error {
	var errs []string
	add := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }
	if len(targets) == 0 {
		return fmt.Errorf("MISSING_CONTRACT: the resolved registration target set is empty; an empty selection is invalid")
	}
	current := map[string]bool{}
	for _, mk := range inv.Milestones {
		if mk.Design == plan.Design {
			current[mk.Milestone] = true
		}
	}
	declared := map[string]bool{}
	for _, pm := range plan.Milestones {
		declared[pm.ID] = true
	}
	invKeys := map[string]bool{}
	for _, o := range inv.Obligations {
		invKeys[o.Key.Design+"\x00"+o.Key.Kind+"\x00"+o.Key.Owner+"\x00"+o.Key.ID] = true
	}
	tests := map[testKey]bool{}
	declare := func(m *Manifest) {
		for _, s := range m.Suites {
			for _, tst := range s.Tests {
				tests[testKey{m.designID, m.ID, s.ID, tst.ID}] = true
			}
		}
	}
	seen := map[string]bool{}
	for mi := range targets {
		m := &targets[mi]
		if m.designID != plan.Design {
			add("INVALID_SCHEMA: target manifest %s belongs to design %s, not the registered design %s", m.ID, m.designID, plan.Design)
			continue
		}
		if seen[m.ID] {
			add("INVALID_SCHEMA: milestone %s is selected more than once", m.ID)
			continue
		}
		seen[m.ID] = true
		if !declared[m.ID] {
			add("MISSING_CONTRACT: milestone %s is not declared by the current plan", m.ID)
		}
		if !current[m.ID] {
			add("MISSING_CONTRACT: milestone %s is not current in the authoritative inventory; deleting or renaming a BUILD milestone cannot reduce requirements", m.ID)
		}
		if m.Baseline == "" {
			add("MISSING_CONTRACT: milestone %s is an unregistered draft (null baseline); only finalized revisions register", m.ID)
		}
		for vi, v := range m.Variants {
			if v.Source == "" {
				add("MISSING_CONTRACT: milestone %s variant %s has no retained source state; only finalized revisions register", m.ID, v.ID)
			}
			_ = vi
		}
		if len(m.Suites) == 0 {
			add("MISSING_CONTRACT: milestone %s declares no suites", m.ID)
		}
		for _, o := range m.Obligations {
			k := o.Key.Design + "\x00" + o.Key.Kind + "\x00" + o.Key.Owner + "\x00" + o.Key.ID
			if !invKeys[k] {
				add("MISSING_CONTRACT: obligation {design=%s kind=%s owner=%s id=%s} is not in the authoritative inventory (invented, stale or pooled key)", o.Key.Design, o.Key.Kind, o.Key.Owner, o.Key.ID)
			}
		}
		declare(m)
	}
	for ri := range registered {
		declare(&registered[ri].Manifest)
	}
	resolve := func(r TestRef, where string) {
		if !tests[refKey(r)] {
			add("MISSING_TEST: %s references test %s which is neither a selected finalized target nor an exactly current registered manifest; a future draft cannot discharge a registered requirement", where, refKey(r))
		}
	}
	for _, ro := range plan.RuntimeObligations {
		for _, r := range ro.Tests {
			resolve(r, "plan runtime obligation "+ro.ID)
		}
	}
	for mi := range targets {
		m := &targets[mi]
		for _, o := range m.Obligations {
			for _, r := range o.Positive {
				resolve(r, "manifest "+m.ID+" obligation positive")
			}
			for _, r := range o.Negative {
				resolve(r, "manifest "+m.ID+" obligation negative")
			}
		}
		for _, e := range m.RedExpectations {
			resolve(e.Test, "manifest "+m.ID+" red_expectations")
		}
		for _, v := range m.Variants {
			for _, r := range v.TargetTests {
				resolve(r, "variant "+v.ID+" target_tests")
			}
			for _, e := range v.Expected {
				resolve(e.Test, "variant "+v.ID+" expected")
			}
		}
		for _, rc := range m.RedControls {
			resolve(rc.Test, "manifest "+m.ID+" red_controls")
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// RegistrationInputs carries the deterministic successor inputs.
type RegistrationInputs struct {
	ProjectID  string
	Plan       Plan
	PlanBytes  []byte
	Targets    []Manifest // selected finalized milestones of Plan.Design
	Untargeted []RegisteredManifest
}

// RegistrationControl is one exact control byte set to archive.
type RegistrationControl struct {
	Digest string
	Bytes  []byte
}

// RegistrationSuccessor is the deterministic desired successor of one
// registration transaction against its expected head.
type RegistrationSuccessor struct {
	HeadBytes      []byte
	HeadDigest     string
	PreviousDigest string
	Generation     int64
	Controls       []RegistrationControl
	BundleRefs     []string
	TargetKeys     []MilestoneKey
}

// BuildRegistrationSuccessor validates lineage and derives the canonical
// next head plus the exact controls to archive. Lineage rules: revision
// 1/null for keys absent from the expected head, prior revision+1 with the
// exact same-key predecessor otherwise; a plan byte change advances every
// registered milestone of the design in the same transaction; registered
// milestone deletion is rejected; untargeted registered controls must match
// the registered bytes exactly. Pure: no I/O.
func BuildRegistrationSuccessor(expectedDigest string, expectedRaw []byte, in RegistrationInputs) (RegistrationSuccessor, error) {
	if !validDigest(expectedDigest) {
		return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: expected head %q is not a digest; there is no wildcard comparison", expectedDigest)
	}
	eh, err := decodeHeadClosed(expectedRaw)
	if err != nil {
		return RegistrationSuccessor{}, err
	}
	if digestOfBytes(expectedRaw) != expectedDigest {
		return RegistrationSuccessor{}, fmt.Errorf("CONTROL_ROLLBACK: the expected-head archive bytes do not hash to %s", expectedDigest)
	}
	if eh.ProjectID != in.ProjectID {
		return RegistrationSuccessor{}, fmt.Errorf("STORE_ROOT_MISMATCH: expected head project %s does not match the registered project %s", eh.ProjectID, in.ProjectID)
	}
	if in.Plan.ProjectID != in.ProjectID {
		return RegistrationSuccessor{}, fmt.Errorf("STORE_ROOT_MISMATCH: plan project %s does not match the registered project %s", in.Plan.ProjectID, in.ProjectID)
	}
	if len(in.PlanBytes) == 0 {
		return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: the exact current plan bytes are required to derive a successor")
	}
	planDigest := digestOfBytes(in.PlanBytes)
	planPrev := ""
	for _, p := range eh.Plans {
		if p.Design == in.Plan.Design {
			planPrev = p.PlanDigest
		}
	}
	planChanged := planPrev != "" && planPrev != planDigest
	declared := map[string]bool{}
	for _, pm := range in.Plan.Milestones {
		declared[pm.ID] = true
	}
	sorted := make([]Manifest, len(in.Targets))
	copy(sorted, in.Targets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	targets := map[string]*Manifest{}
	for i := range sorted {
		m := &sorted[i]
		if m.designID != in.Plan.Design {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: target manifest %s belongs to design %s, not %s", m.ID, m.designID, in.Plan.Design)
		}
		if _, dup := targets[m.ID]; dup {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: milestone %s is selected more than once", m.ID)
		}
		if !declared[m.ID] {
			return RegistrationSuccessor{}, fmt.Errorf("MISSING_CONTRACT: milestone %s is not declared by the current plan", m.ID)
		}
		targets[m.ID] = m
	}
	registered := map[string]headMilestoneEntry{}
	for _, m := range eh.Milestones {
		if m.Design == in.Plan.Design {
			registered[m.Milestone] = m
		}
	}
	for id, m := range targets {
		prior, ok := registered[id]
		if !ok {
			if m.Revision != 1 || m.Predecessor != "" {
				return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: milestone %s is absent from the expected head; its first revision must be 1 with a null predecessor (got revision %d)", id, m.Revision)
			}
			continue
		}
		if m.Revision != prior.Revision+1 {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: milestone %s revision %d does not advance registered revision %d by exactly one", id, m.Revision, prior.Revision)
		}
		if m.Predecessor != prior.ManifestDi {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: milestone %s predecessor %s is not the exact registered manifest digest %s of the same key", id, m.Predecessor, prior.ManifestDi)
		}
	}
	untargeted := map[string]RegisteredManifest{}
	for _, u := range in.Untargeted {
		if u.Key.Design != in.Plan.Design {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: untargeted entry %s/%s belongs to design %s, not %s", u.Key.Design, u.Key.Milestone, u.Key.Design, in.Plan.Design)
		}
		if _, dup := untargeted[u.Key.Milestone]; dup {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: untargeted entry for milestone %s supplied more than once", u.Key.Milestone)
		}
		untargeted[u.Key.Milestone] = u
	}
	for id, prior := range registered {
		if targets[id] != nil {
			continue
		}
		if !declared[id] {
			return RegistrationSuccessor{}, fmt.Errorf("MISSING_CONTRACT: registered milestone %s is no longer declared by the current plan; removal is not implicit retirement in first-release registration", id)
		}
		u, ok := untargeted[id]
		if !ok {
			return RegistrationSuccessor{}, fmt.Errorf("MISSING_CONTRACT: registered milestone %s is not selected and its current control bytes were not supplied for the exact-match check", id)
		}
		if u.Digest != prior.ManifestDi {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: untargeted entry for milestone %s names digest %s, not the registered head digest %s", id, u.Digest, prior.ManifestDi)
		}
		if digestOfBytes(u.Manifest.Raw()) != prior.ManifestDi {
			return RegistrationSuccessor{}, fmt.Errorf("STALE_INPUT: untargeted registered milestone %s changed on disk (current bytes %s, registered %s); the current authored target may differ from the registered head only in this registration transition", id, digestOfBytes(u.Manifest.Raw()), prior.ManifestDi)
		}
		if planChanged {
			return RegistrationSuccessor{}, fmt.Errorf("MISSING_CONTRACT: the exact plan bytes changed but registered milestone %s is not advanced in this transaction; a plan change advances every registered milestone of the design", id)
		}
	}
	for id := range untargeted {
		if _, isReg := registered[id]; !isReg {
			return RegistrationSuccessor{}, fmt.Errorf("INVALID_SCHEMA: untargeted entry for milestone %s has no registered head entry", id)
		}
	}
	if len(targets) == 0 {
		return RegistrationSuccessor{}, fmt.Errorf("MISSING_CONTRACT: the resolved registration target set is empty; an empty selection is invalid")
	}
	next := headDoc{
		ProjectID: eh.ProjectID, Generation: eh.Generation + 1, Previous: expectedDigest,
	}
	for _, p := range eh.Plans {
		if p.Design == in.Plan.Design {
			continue
		}
		next.Plans = append(next.Plans, p)
	}
	next.Plans = append(next.Plans, headPlanEntry{Design: in.Plan.Design, PlanDigest: planDigest})
	sort.Slice(next.Plans, func(i, j int) bool { return next.Plans[i].Design < next.Plans[j].Design })
	for _, m := range eh.Milestones {
		if m.Design == in.Plan.Design {
			continue
		}
		next.Milestones = append(next.Milestones, m)
	}
	keys := make([]MilestoneKey, 0, len(sorted))
	for i := range sorted {
		m := &sorted[i]
		next.Milestones = append(next.Milestones, headMilestoneEntry{
			Design: in.Plan.Design, Milestone: m.ID, Revision: m.Revision,
			ManifestDi: digestOfBytes(m.Raw()), PredecessorDi: m.Predecessor,
		})
		keys = append(keys, MilestoneKey{Design: in.Plan.Design, Milestone: m.ID})
	}
	sort.Slice(next.Milestones, func(i, j int) bool {
		if next.Milestones[i].Design != next.Milestones[j].Design {
			return next.Milestones[i].Design < next.Milestones[j].Design
		}
		return next.Milestones[i].Milestone < next.Milestones[j].Milestone
	})
	headBytes := encodeHeadCanonical(next)
	controls := []RegistrationControl{{Digest: planDigest, Bytes: in.PlanBytes}}
	seenCtl := map[string]bool{planDigest: true}
	refs := map[string]bool{}
	for i := range sorted {
		m := &sorted[i]
		d := digestOfBytes(m.Raw())
		if !seenCtl[d] {
			seenCtl[d] = true
			controls = append(controls, RegistrationControl{Digest: d, Bytes: m.Raw()})
		}
		if m.Baseline != "" {
			refs[m.Baseline] = true
		}
		for _, v := range m.Variants {
			if v.Source != "" {
				refs[v.Source] = true
			}
		}
	}
	bundleRefs := make([]string, 0, len(refs))
	for r := range refs {
		bundleRefs = append(bundleRefs, r)
	}
	sort.Strings(bundleRefs)
	return RegistrationSuccessor{
		HeadBytes: headBytes, HeadDigest: digestOfBytes(headBytes),
		PreviousDigest: expectedDigest, Generation: eh.Generation + 1,
		Controls: controls, BundleRefs: bundleRefs, TargetKeys: keys,
	}, nil
}

// ReadArchivedHead opens the store read-only under the plan's project
// identity and returns the exact archived bytes of the expected head.
// A missing or non-matching archive is HISTORY_UNAVAILABLE.
func ReadArchivedHead(ctx context.Context, store, projectID, expectedHead string) ([]byte, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if !validDigest(expectedHead) {
		return nil, fmt.Errorf("INVALID_SCHEMA: expected head %q is not a digest; there is no wildcard comparison", expectedHead)
	}
	v, err := openStore(ctx, store, projectID)
	if err != nil {
		return nil, err
	}
	defer v.close()
	raw, rerr := os.ReadFile(filepath.Join(store, storeLedger, storeHeads, strings.TrimPrefix(expectedHead, "sha256:")+".json"))
	if rerr != nil || digestOfBytes(raw) != expectedHead {
		return nil, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s is not available byte-exact in store %s", expectedHead, store)
	}
	if _, derr := decodeHeadClosed(raw); derr != nil {
		return nil, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s does not decode: %w", expectedHead, derr)
	}
	return raw, nil
}

// ReadArchivedControl returns the exact archived control bytes for one
// digest, verified byte-identical against the address. A committed head
// entry whose control bytes are missing is a broken chain, not fresh state.
func ReadArchivedControl(ctx context.Context, store, projectID, digest string) ([]byte, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return nil, err
	}
	if !validDigest(digest) {
		return nil, fmt.Errorf("INVALID_SCHEMA: control digest %q is not a digest", digest)
	}
	v, err := openStore(ctx, store, projectID)
	if err != nil {
		return nil, err
	}
	defer v.close()
	raw, rerr := os.ReadFile(filepath.Join(store, "controls", strings.TrimPrefix(digest, "sha256:")+".json"))
	if rerr != nil || digestOfBytes(raw) != digest {
		return nil, fmt.Errorf("CONTROL_ROLLBACK: archived control %s is not available byte-exact in store %s; a committed head cannot name missing controls", digest, store)
	}
	return raw, nil
}

// StoreRegistrationRequest is one durable registration transaction.
type StoreRegistrationRequest struct {
	Store        string
	ProjectID    string
	ExpectedHead string
	Successor    RegistrationSuccessor
	Revalidate   func() error // final input revalidation before publication
}

// StoreRegistrationResult reports the committed or idempotent outcome.
// Idempotent marks the exact already-registered exception.
type StoreRegistrationResult struct {
	StoreID      string
	ProjectID    string
	PreviousHead string
	HeadDigest   string
	Generation   int64
	Idempotent   bool
}

// writerLease bounds how long one writer token may be considered live: the
// fixed non-execution owner deadline plus the cleanup grace plus margin.
const writerLease = 660 * time.Second

// storeWriter is the external-store writer: an atomic mkdir token inside
// the store's transaction staging namespace. A crashed writer leaves an
// expired token; exactly one waiter takes an expired token over via atomic
// rename. Residual: a takeover racing a fresh acquisition within that
// expired window can transiently double-hold; the compare-and-advance plus
// post-rename verification converts that into a detected CUSTODY_ERROR,
// never a silent double commit or head regression.
type storeWriter struct {
	store string
	held  bool
}

func acquireStoreWriter(ctx context.Context, store string) (*storeWriter, error) {
	staging := filepath.Join(store, storeStaging)
	token := filepath.Join(staging, "writer.lock")
	for {
		if err := checkCtx(ctx); err != nil {
			return nil, err
		}
		if err := os.MkdirAll(staging, storeRootMod); err != nil {
			return nil, fmt.Errorf("CUSTODY_ERROR: opening transaction staging: %w", err)
		}
		if err := os.Mkdir(token, storeRootMod); err == nil {
			deadline := time.Now().Add(writerLease).UnixMilli()
			if werr := os.WriteFile(filepath.Join(token, "owner.json"), []byte(fmt.Sprintf(`{"deadline_ms":%d,"pid":%d}`, deadline, os.Getpid())), storeFileMod); werr != nil {
				os.RemoveAll(token)
				return nil, fmt.Errorf("CUSTODY_ERROR: recording writer ownership: %w", werr)
			}
			return &storeWriter{store: store, held: true}, nil
		} else if !os.IsExist(err) {
			return nil, fmt.Errorf("CUSTODY_ERROR: acquiring store writer: %w", err)
		}
		if tokenExpired(token) {
			victim := filepath.Join(staging, fmt.Sprintf("expired-%d", time.Now().UnixNano()))
			if rerr := os.Rename(token, victim); rerr == nil {
				cctx, ccancel := cleanupContext()
				_ = cctx
				os.RemoveAll(victim)
				ccancel()
				continue
			}
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("TIMEOUT: another writer holds the store transaction lock; no result is claimed")
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// tokenExpired reports whether the holder's lease is past the bounded
// non-execution deadline; an unreadable token is judged by its age alone.
func tokenExpired(token string) bool {
	deadline := int64(0)
	if raw, err := os.ReadFile(filepath.Join(token, "owner.json")); err == nil {
		if _, err := fmt.Sscanf(string(raw), `{"deadline_ms":%d`, &deadline); err == nil && deadline > 0 {
			return time.Now().UnixMilli() > deadline
		}
	}
	fi, err := os.Stat(token)
	return err == nil && time.Since(fi.ModTime()) > writerLease
}

// release drops the writer and removes the transaction staging namespace so
// the store returns to a quiescent, exportable state.
func (w *storeWriter) release() error {
	if w == nil || !w.held {
		return nil
	}
	w.held = false
	cctx, ccancel := cleanupContext()
	defer ccancel()
	_ = cctx
	if err := os.RemoveAll(filepath.Join(w.store, storeStaging)); err != nil {
		return fmt.Errorf("CUSTODY_ERROR: releasing the store writer (the head may already be advanced; an exact retry confirms registration): %w", err)
	}
	return nil
}

// CommitRegistration performs the section-4 registration transaction:
// release the read reservation, acquire the external-store writer, compare
// the exact expected head (the idempotent exception is checked here),
// archive and stage the derived successor, revalidate the original inputs,
// then durably compare-and-advance head.json exactly one generation. The
// CAS loser receives HEAD_CONFLICT; nothing rebases or retries.
func CommitRegistration(ctx context.Context, req StoreRegistrationRequest) (StoreRegistrationResult, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return StoreRegistrationResult{}, err
	}
	if req.Store == "" {
		return StoreRegistrationResult{}, fmt.Errorf("INVALID_SCHEMA: the explicit store path is required; there is no implicit fallback")
	}
	if !validDigest(req.ExpectedHead) {
		return StoreRegistrationResult{}, fmt.Errorf("INVALID_SCHEMA: expected head %q is not a digest; there is no wildcard comparison", req.ExpectedHead)
	}
	if len(req.Successor.HeadBytes) == 0 || req.Successor.HeadDigest != digestOfBytes(req.Successor.HeadBytes) {
		return StoreRegistrationResult{}, fmt.Errorf("INVALID_SCHEMA: the derived successor head bytes do not hash to their digest")
	}
	// read reservation: full store validation, then release before writer
	v, err := openStore(ctx, req.Store, req.ProjectID)
	if err != nil {
		return StoreRegistrationResult{}, err
	}
	storeID := v.id.StoreID
	expectedRaw, rerr := os.ReadFile(filepath.Join(req.Store, storeLedger, storeHeads, strings.TrimPrefix(req.ExpectedHead, "sha256:")+".json"))
	if rerr != nil || digestOfBytes(expectedRaw) != req.ExpectedHead {
		v.close()
		return StoreRegistrationResult{}, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s is not available byte-exact in store %s", req.ExpectedHead, req.Store)
	}
	if _, derr := decodeHeadClosed(expectedRaw); derr != nil {
		v.close()
		return StoreRegistrationResult{}, fmt.Errorf("HISTORY_UNAVAILABLE: the archived expected head %s does not decode: %w", req.ExpectedHead, derr)
	}
	if err := v.close(); err != nil {
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: releasing the store read reservation: %w", err)
	}
	w, err := acquireStoreWriter(ctx, req.Store)
	if err != nil {
		return StoreRegistrationResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			w.release()
		}
	}()
	// exact expected-head comparison with the idempotent exception
	currentRaw, cerr := os.ReadFile(filepath.Join(req.Store, storeLedger, "head.json"))
	if cerr != nil {
		return StoreRegistrationResult{}, fmt.Errorf("INVALID_SCHEMA: reading the current head: %w", cerr)
	}
	currentDigest := digestOfBytes(currentRaw)
	ch, derr := decodeHeadClosed(currentRaw)
	if derr != nil {
		return StoreRegistrationResult{}, derr
	}
	switch currentDigest {
	case req.ExpectedHead:
		// normal advance below
	case req.Successor.HeadDigest:
		for _, c := range req.Successor.Controls {
			if err := verifyArchivedControl(req.Store, c); err != nil {
				return StoreRegistrationResult{}, err
			}
		}
		if err := verifyRetainedBundles(req.Store, req.Successor.BundleRefs); err != nil {
			return StoreRegistrationResult{}, err
		}
		if rerr := w.release(); rerr != nil {
			return StoreRegistrationResult{}, rerr
		}
		committed = true
		return StoreRegistrationResult{
			StoreID: storeID, ProjectID: ch.ProjectID, PreviousHead: ch.Previous,
			HeadDigest: currentDigest, Generation: ch.Generation, Idempotent: true,
		}, nil
	default:
		return StoreRegistrationResult{}, fmt.Errorf("HEAD_CONFLICT: the current head %s is neither the expected %s nor the exact desired successor %s; rebase the authored registration against the new head explicitly — no automatic merge or last-writer-wins", currentDigest, req.ExpectedHead, req.Successor.HeadDigest)
	}
	if err := verifyRetainedBundles(req.Store, req.Successor.BundleRefs); err != nil {
		return StoreRegistrationResult{}, err
	}
	// archive and stage all selected finalized controls and the next head
	for _, c := range req.Successor.Controls {
		if aerr := publishImmutableFile(filepath.Join(req.Store, "controls", strings.TrimPrefix(c.Digest, "sha256:")+".json"), storeFileMod, c.Bytes); aerr != nil {
			return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: archiving control %s: %w", c.Digest, aerr)
		}
	}
	staged := filepath.Join(req.Store, storeStaging, "head.next")
	if serr := durableWriteBytes(staged, storeFileMod, req.Successor.HeadBytes); serr != nil {
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: staging the derived next head: %w", serr)
	}
	// finalize the original snapshot: reacquire, compare, complete release
	if req.Revalidate != nil {
		if verr := req.Revalidate(); verr != nil {
			os.Remove(staged)
			return StoreRegistrationResult{}, verr
		}
	}
	if err := checkCtx(ctx); err != nil {
		os.Remove(staged)
		return StoreRegistrationResult{}, err
	}
	// durable archive before the advance; a crash between the two leaves a
	// tolerated recoverable successor, never a head naming missing bytes
	if aerr := publishImmutableFile(filepath.Join(req.Store, storeLedger, storeHeads, strings.TrimPrefix(req.Successor.HeadDigest, "sha256:")+".json"), storeFileMod, req.Successor.HeadBytes); aerr != nil {
		os.Remove(staged)
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: archiving the next head: %w", aerr)
	}
	// final compare-and-advance while holding the writer
	againRaw, aerr := os.ReadFile(filepath.Join(req.Store, storeLedger, "head.json"))
	if aerr != nil {
		os.Remove(staged)
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: re-reading the current head: %w", aerr)
	}
	if digestOfBytes(againRaw) != currentDigest {
		os.Remove(staged)
		os.Remove(filepath.Join(req.Store, storeLedger, storeHeads, strings.TrimPrefix(req.Successor.HeadDigest, "sha256:")+".json"))
		return StoreRegistrationResult{}, fmt.Errorf("HEAD_CONFLICT: the head advanced while the writer was held (%s != %s); no rebase is performed", digestOfBytes(againRaw), currentDigest)
	}
	if rerr := os.Rename(staged, filepath.Join(req.Store, storeLedger, "head.json")); rerr != nil {
		os.Remove(staged)
		os.Remove(filepath.Join(req.Store, storeLedger, storeHeads, strings.TrimPrefix(req.Successor.HeadDigest, "sha256:")+".json"))
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: compare-and-advance of head.json: %w", rerr)
	}
	if err := fsyncDir(filepath.Join(req.Store, storeLedger)); err != nil {
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: fsync of the advanced head (the head may already be advanced; an exact retry confirms registration): %w", err)
	}
	finalRaw, ferr := os.ReadFile(filepath.Join(req.Store, storeLedger, "head.json"))
	if ferr != nil || digestOfBytes(finalRaw) != req.Successor.HeadDigest {
		return StoreRegistrationResult{}, fmt.Errorf("CUSTODY_ERROR: post-advance verification failed; the committed head is not the derived successor (another writer may have advanced concurrently)")
	}
	if err := checkCtx(ctx); err != nil {
		return StoreRegistrationResult{}, fmt.Errorf("%w; the head may already be advanced and an exact retry confirms registration", err)
	}
	if rerr := w.release(); rerr != nil {
		return StoreRegistrationResult{}, rerr
	}
	committed = true
	return StoreRegistrationResult{
		StoreID: storeID, ProjectID: ch.ProjectID, PreviousHead: currentDigest,
		HeadDigest: req.Successor.HeadDigest, Generation: ch.Generation + 1,
	}, nil
}

// verifyArchivedControl fails unless the control is present byte-exact.
func verifyArchivedControl(store string, c RegistrationControl) error {
	raw, err := os.ReadFile(filepath.Join(store, "controls", strings.TrimPrefix(c.Digest, "sha256:")+".json"))
	if err != nil || digestOfBytes(raw) != c.Digest || string(raw) != string(c.Bytes) {
		return fmt.Errorf("CUSTODY_ERROR: archived control %s is not byte-identical to the derived control; the idempotent exception requires all archived controls identical", c.Digest)
	}
	return nil
}

// verifyRetainedBundles proves every referenced bundle exists in objects/.
func verifyRetainedBundles(store string, refs []string) error {
	for _, r := range refs {
		if fi, err := os.Lstat(filepath.Join(store, storeObjects, strings.TrimPrefix(r, "sha256:"), "bundle.json")); err != nil || fi.Mode().Perm() != storeFileMod {
			return fmt.Errorf("MISSING_CONTRACT: retained bundle %s is not present in the store objects namespace; registration requires the selected finalized references to be retained", r)
		}
	}
	return nil
}
