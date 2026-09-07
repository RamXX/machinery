// Package assuranceflow is the privileged closed orchestration layer of
// docs/test-assurance-contract.md section 7: it imports gates, tdd and
// processscope (never the reverse) and owns same-invocation preconditions,
// original snapshots, finalizers, external-store publication and sealed
// results. This file implements Register — the explicit authored-revision
// transaction of section 4. Register performs no subprocesses; it commits
// authored data only and never claims execution, replay or Gv acceptance.
package assuranceflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// RegisterRequest is the closed registration request: explicit paths, the
// caller-supplied expected head digest, and the selected milestone targets
// (empty selects every milestone declared by the design's plan; an empty
// resolved target set is an error).
type RegisterRequest struct {
	Design         string
	Implementation string
	Store          string
	ExpectedHead   string
	Milestones     []string
}

// Registration is the private-constructed authored-data result. It exposes
// exactly the identity/head/selection fields and the registered state; it
// carries NO replay, custody-success or test-pass accessor and cannot
// satisfy Verification.
type Registration struct {
	projectID    string
	storeID      string
	previousHead string
	headDigest   string
	generation   int64
	keys         []tdd.MilestoneKey
	state        string
}

// ProjectID returns the registered project identity.
func (r Registration) ProjectID() string { return r.projectID }

// StoreID returns the external store identity.
func (r Registration) StoreID() string { return r.storeID }

// PreviousHead returns the exact head digest this registration advanced.
func (r Registration) PreviousHead() string { return r.previousHead }

// HeadDigest returns the committed successor head digest.
func (r Registration) HeadDigest() string { return r.headDigest }

// Generation returns the committed head generation.
func (r Registration) Generation() int64 { return r.generation }

// MilestoneKeys returns the sorted selected milestone keys.
func (r Registration) MilestoneKeys() []tdd.MilestoneKey {
	out := make([]tdd.MilestoneKey, len(r.keys))
	copy(out, r.keys)
	return out
}

// State returns registered-not-executed or already-registered-not-executed.
func (r Registration) State() string { return r.state }

var (
	digestRe    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	milestoneRe = regexp.MustCompile(`^M(0|[1-9][0-9]*)$`)
)

// validDesignRoot accepts exactly "." or a clean relative slash path.
func validDesignRoot(d string) bool {
	if d == "." {
		return true
	}
	if d == "" || strings.HasPrefix(d, "/") || strings.Contains(d, "\\") || strings.Contains(d, "..") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(d))
	return clean == d && !strings.HasPrefix(clean, "/")
}

// controlSnapshot holds the original design view of one registration: the
// exact control bytes, the design payload digest and the rooted
// root-generation identity captured before the store transaction starts.
type controlSnapshot struct {
	designDir     string
	payloadDigest string
	rootIdentity  os.FileInfo
	controls      map[string][]byte // slash-relative path -> exact bytes
}

// newControlSnapshot captures the original view: the full design payload
// digest and every authored control byte under design/assurance/.
func newControlSnapshot(designDir string) (*controlSnapshot, error) {
	fi, err := os.Lstat(designDir)
	if err != nil || !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("MISSING_CONTRACT: design root %s is not a real directory", designDir)
	}
	payload, err := tdd.DesignPayloadDigest(designDir)
	if err != nil {
		return nil, err
	}
	s := &controlSnapshot{designDir: designDir, payloadDigest: payload, rootIdentity: fi, controls: map[string][]byte{}}
	add := func(rel string) error {
		raw, err := os.ReadFile(filepath.Join(designDir, protocol.ControlDirName, filepath.FromSlash(rel)))
		if err != nil {
			return err
		}
		if int64(len(raw)) > protocol.ControlInputMaxBytes {
			return fmt.Errorf("OUTPUT_LIMIT: control %s exceeds the control-input cap", rel)
		}
		s.controls[rel] = raw
		return nil
	}
	if err := add(protocol.PlanFileName); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Join(designDir, protocol.ControlDirName, protocol.MilestonesDirName))
	if err != nil {
		return nil, fmt.Errorf("MISSING_CONTRACT: the milestones control namespace is unavailable: %w", err)
	}
	for _, e := range entries {
		if !e.Type().IsRegular() || !strings.HasSuffix(e.Name(), ".json") {
			return nil, fmt.Errorf("INVALID_SCHEMA: control namespace milestones/ holds unknown entry %q", e.Name())
		}
		if err := add(protocol.MilestonesDirName + "/" + e.Name()); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// revalidate re-reads the original view and compares exact source/control
// and root-generation identities (the release/reacquire/revalidate cycle of
// the registration transaction's final input check).
func (s *controlSnapshot) revalidate() error {
	fi, err := os.Lstat(s.designDir)
	if err != nil || !os.SameFile(fi, s.rootIdentity) {
		return fmt.Errorf("STALE_INPUT: the design root identity changed during registration (root-generation swap)")
	}
	payload, err := tdd.DesignPayloadDigest(s.designDir)
	if err != nil {
		return fmt.Errorf("STALE_INPUT: the design payload failed revalidation: %w", err)
	}
	if payload != s.payloadDigest {
		return fmt.Errorf("STALE_INPUT: the design payload digest changed during registration (%s != %s)", payload, s.payloadDigest)
	}
	for rel, want := range s.controls {
		got, err := os.ReadFile(filepath.Join(s.designDir, protocol.ControlDirName, filepath.FromSlash(rel)))
		if err != nil || string(got) != string(want) {
			return fmt.Errorf("STALE_INPUT: control %s changed during registration", rel)
		}
	}
	return nil
}

// Register commits one explicit reviewed authored revision against the
// expected external head and returns only registered-not-executed or
// already-registered-not-executed. The whole operation is bounded by the
// fixed 600000 ms non-execution owner deadline plus one 10000 ms cleanup
// grace, including locks, publication and output. Errors before the durable
// commit consume no revision; a post-commit close/output failure returns no
// successful Registration although the head may have advanced, and an exact
// retry confirms the already-registered state.
func Register(ctx context.Context, req RegisterRequest, output io.Writer) (Registration, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(protocol.LimitWallDefaultMS)*time.Millisecond)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return Registration{}, fmt.Errorf("TIMEOUT: registration cancelled before entry; no result is claimed")
	}
	if err := validateRegisterRequest(req); err != nil {
		return Registration{}, err
	}
	implFi, err := os.Lstat(req.Implementation)
	if err != nil || !implFi.IsDir() || implFi.Mode()&os.ModeSymlink != 0 {
		return Registration{}, fmt.Errorf("MISSING_CONTRACT: implementation root %s is not a real directory", req.Implementation)
	}
	impl, err := filepath.Abs(req.Implementation)
	if err != nil {
		return Registration{}, fmt.Errorf("INVALID_SCHEMA: implementation root: %w", err)
	}
	designDir := filepath.Join(impl, filepath.FromSlash(req.Design))
	if err := tdd.ValidateStorePlacement(req.Store, impl, designDir); err != nil {
		return Registration{}, err
	}
	// original immutable source/control snapshot and current plan
	snap, err := newControlSnapshot(designDir)
	if err != nil {
		return Registration{}, err
	}
	plan, err := tdd.LoadPlan(designDir)
	if err != nil {
		return Registration{}, err
	}
	// snapshot and loaded bytes must be the same original view
	if string(snap.controls[protocol.PlanFileName]) != string(plan.Raw()) {
		return Registration{}, fmt.Errorf("STALE_INPUT: the plan changed between the original snapshot and loading")
	}
	selection, err := resolveSelection(req.Milestones, plan)
	if err != nil {
		return Registration{}, err
	}
	expectedRaw, err := tdd.ReadArchivedHead(ctx, req.Store, plan.ProjectID, req.ExpectedHead)
	if err != nil {
		return Registration{}, err
	}
	expected, err := tdd.DecodeExpectedHead(expectedRaw)
	if err != nil {
		return Registration{}, err
	}
	targets := make([]tdd.Manifest, 0, len(selection))
	manifestPath := func(id string) string {
		return filepath.Join(designDir, protocol.ControlDirName, protocol.MilestonesDirName, id+".json")
	}
	for _, id := range selection {
		m, err := tdd.LoadManifest(manifestPath(id))
		if err != nil {
			return Registration{}, err
		}
		if string(snap.controls[protocol.MilestonesDirName+"/"+id+".json"]) != string(m.Raw()) {
			return Registration{}, fmt.Errorf("STALE_INPUT: manifest %s changed between the original snapshot and loading", id)
		}
		targets = append(targets, m)
	}
	// untargeted registered milestones of this design: current bytes loaded
	var untargeted []tdd.RegisteredManifest
	registeredKeys := map[tdd.MilestoneKey]tdd.HeadMilestoneEntry{}
	for _, e := range expected.Milestones {
		registeredKeys[e.Key] = e
		if e.Key.Design != plan.Design {
			continue
		}
		if containsMilestone(selection, e.Key.Milestone) {
			continue
		}
		m, err := tdd.LoadManifest(manifestPath(e.Key.Milestone))
		if err != nil {
			return Registration{}, err
		}
		untargeted = append(untargeted, tdd.RegisteredManifest{Key: e.Key, Revision: e.Revision, Digest: e.ManifestDigest, Manifest: m})
	}
	// authoritative inventory and the registration graph
	inventory, err := gates.AssuranceInventory(designDir)
	if err != nil {
		return Registration{}, err
	}
	external, err := resolveExternalRegistered(ctx, req.Store, plan, targets, expected, registeredKeys)
	if err != nil {
		return Registration{}, err
	}
	if err := tdd.ValidateRegistrationGraph(plan, targets, append(append([]tdd.RegisteredManifest{}, untargeted...), external...), inventory); err != nil {
		return Registration{}, err
	}
	successor, err := tdd.BuildRegistrationSuccessor(req.ExpectedHead, expectedRaw, tdd.RegistrationInputs{
		ProjectID: plan.ProjectID, Plan: plan, PlanBytes: plan.Raw(),
		Targets: targets, Untargeted: untargeted,
	})
	if err != nil {
		return Registration{}, err
	}
	res, err := tdd.CommitRegistration(ctx, tdd.StoreRegistrationRequest{
		Store: req.Store, ProjectID: plan.ProjectID, ExpectedHead: req.ExpectedHead,
		Successor: successor, Revalidate: snap.revalidate,
	})
	if err != nil {
		return Registration{}, err
	}
	state := tdd.StateRegisteredNotExecuted
	if res.Idempotent {
		state = tdd.StateAlreadyRegisteredNotExecuted
	}
	reg := Registration{
		projectID: res.ProjectID, storeID: res.StoreID, previousHead: res.PreviousHead,
		headDigest: res.HeadDigest, generation: res.Generation, keys: successor.TargetKeys, state: state,
	}
	if err := writeRegistrationOutput(output, req, reg); err != nil {
		return Registration{}, fmt.Errorf("CUSTODY_ERROR: registration output failed after the durable commit (the head may already be advanced; an exact retry confirms registration): %w", err)
	}
	return reg, nil
}

func validateRegisterRequest(req RegisterRequest) error {
	if !validDesignRoot(req.Design) {
		return fmt.Errorf("INVALID_SCHEMA: design root %q must be \".\" or a clean relative slash path", req.Design)
	}
	if req.Implementation == "" {
		return fmt.Errorf("INVALID_SCHEMA: the explicit implementation path is required")
	}
	if req.Store == "" {
		return fmt.Errorf("INVALID_SCHEMA: the explicit store path is required; there is no implicit fallback")
	}
	if !digestRe.MatchString(req.ExpectedHead) {
		return fmt.Errorf("INVALID_SCHEMA: expected head %q is not a sha256 digest; there is no wildcard or empty-string comparison", req.ExpectedHead)
	}
	seen := map[string]bool{}
	for _, m := range req.Milestones {
		if !milestoneRe.MatchString(m) {
			return fmt.Errorf("INVALID_SCHEMA: milestone target %q is not canonical (M plus an unsigned decimal)", m)
		}
		if seen[m] {
			return fmt.Errorf("INVALID_SCHEMA: milestone target %q is repeated; repeated flags form a unique sorted set in the CLI owner", m)
		}
		seen[m] = true
	}
	return nil
}

func resolveSelection(requested []string, plan tdd.Plan) ([]string, error) {
	declared := map[string]bool{}
	for _, pm := range plan.Milestones {
		declared[pm.ID] = true
	}
	selection := requested
	if len(selection) == 0 {
		selection = selection[:0]
		for _, pm := range plan.Milestones {
			selection = append(selection, pm.ID)
		}
		sort.Strings(selection)
	}
	if len(selection) == 0 {
		return nil, fmt.Errorf("MISSING_CONTRACT: the resolved registration target set is empty; an empty selection is invalid")
	}
	for _, id := range selection {
		if !declared[id] {
			return nil, fmt.Errorf("MISSING_CONTRACT: milestone %s is not declared by the plan of design %s", id, plan.Design)
		}
	}
	sort.Strings(selection)
	return selection, nil
}

func containsMilestone(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// resolveExternalRegistered loads the archived manifests of every head
// milestone entry referenced by the selected controls but not part of this
// design's targets: other finalized milestones of this design that are
// already registered, and child designs that registered under their own
// design command first.
func resolveExternalRegistered(ctx context.Context, store string, plan tdd.Plan, targets []tdd.Manifest, expected tdd.ExpectedHead, registered map[tdd.MilestoneKey]tdd.HeadMilestoneEntry) ([]tdd.RegisteredManifest, error) {
	covered := map[tdd.MilestoneKey]bool{}
	for _, m := range targets {
		covered[tdd.MilestoneKey{Design: plan.Design, Milestone: m.ID}] = true
	}
	needed := map[tdd.MilestoneKey]bool{}
	consider := func(r tdd.TestRef) {
		k := tdd.MilestoneKey{Design: r.Design, Milestone: r.Milestone}
		if !covered[k] {
			needed[k] = true
		}
	}
	for _, ro := range plan.RuntimeObligations {
		for _, r := range ro.Tests {
			consider(r)
		}
	}
	for _, m := range targets {
		for _, o := range m.Obligations {
			for _, r := range o.Positive {
				consider(r)
			}
			for _, r := range o.Negative {
				consider(r)
			}
		}
		for _, e := range m.RedExpectations {
			consider(e.Test)
		}
		for _, v := range m.Variants {
			for _, r := range v.TargetTests {
				consider(r)
			}
			for _, e := range v.Expected {
				consider(e.Test)
			}
		}
		for _, rc := range m.RedControls {
			consider(rc.Test)
		}
	}
	var out []tdd.RegisteredManifest
	for k := range needed {
		entry, ok := registered[k]
		if !ok {
			continue // unresolved: ValidateRegistrationGraph reports MISSING_TEST
		}
		raw, err := tdd.ReadArchivedControl(ctx, store, plan.ProjectID, entry.ManifestDigest)
		if err != nil {
			return nil, err
		}
		m, err := tdd.DecodeArchivedManifest(k.Design, k.Milestone, raw)
		if err != nil {
			return nil, err
		}
		out = append(out, tdd.RegisteredManifest{Key: k, Revision: entry.Revision, Digest: entry.ManifestDigest, Manifest: m})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key.Design != out[j].Key.Design {
			return out[i].Key.Design < out[j].Key.Design
		}
		return out[i].Key.Milestone < out[j].Key.Milestone
	})
	return out, nil
}

// writeRegistrationOutput renders the bounded scope-labeled result line.
func writeRegistrationOutput(output io.Writer, req RegisterRequest, reg Registration) error {
	if output == nil {
		return nil
	}
	keys := make([]string, 0, len(reg.keys))
	for _, k := range reg.keys {
		keys = append(keys, k.Design+"/"+k.Milestone)
	}
	doc, err := json.Marshal(map[string]any{
		"schema": "machinery.tdd.register/v1", "design": req.Design,
		"state": reg.state, "project_id": reg.projectID, "store_id": reg.storeID,
		"previous_head": reg.previousHead, "head_digest": reg.headDigest,
		"generation": reg.generation, "milestones": keys,
		"provenance": "unauthenticated-host",
		"note":       "registered authored revisions only; no execution, replay or judgment is claimed",
	})
	if err != nil {
		return err
	}
	if _, err := output.Write(append(doc, '\n')); err != nil {
		return err
	}
	return nil
}
