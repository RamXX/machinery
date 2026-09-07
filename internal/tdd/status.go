// Cheap read-only freshness status (docs/test-assurance-contract.md
// sections 1, 4 and 7): the explicit replay-not-performed report over the
// held InputView, the loaded declarations and the explicit external store.
// Status publishes independent store/source/control/judgment dimensions,
// distinguishes missing/unregistered-draft/registered-not-executed/
// recorded-only/stale/historical evidence, and NEVER constitutes execution
// evidence: no hash-only receipt becomes proof tests ran. It performs no
// store writes and launches no processes.
package tdd

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// runSummary is the subset of a run/v1 record status reads (recorded-only
// inspection; full validation belongs to the execution/evidence stories).
type runSummary struct {
	design    string
	milestone string
	result    string
	phase     string
	judgment  string
}

// Status reports the persisted declaration/execution state as recorded-only
// evidence with independent store/source/control/judgment dimensions.
func Status(ctx context.Context, req StatusRequest) (StatusReport, error) {
	ctx, cancel := nonExecContext(ctx)
	defer cancel()
	if err := checkCtx(ctx); err != nil {
		return StatusReport{}, err
	}
	if err := validateHeldInputView(req.Inputs); err != nil {
		return StatusReport{}, err
	}
	if err := req.Inputs.Revalidate(); err != nil {
		return StatusReport{}, fmt.Errorf("STALE_INPUT: the held InputView failed revalidation: %w", err)
	}
	if err := ValidateStorePlacement(req.Store, req.Inputs.SourceRoot, req.Inputs.ControlRoot); err != nil {
		return StatusReport{}, err
	}
	v, err := openStore(ctx, req.Store, req.Plan.ProjectID)
	if err != nil {
		return StatusReport{}, err
	}
	if err := v.close(); err != nil {
		return StatusReport{}, err
	}
	rep := StatusReport{
		DesignConsistency:   "checked",
		SourceTestBinding:   "missing",
		TestExecution:       "not-performed",
		NegativeSensitivity: "missing",
		Judgment:            "missing",
		FormalExecution:     "not-performed",
		RuntimeResiduals:    "unverified",
		Custody:             "cleaned",
		Provenance:          "unauthenticated-host",
		StoreProjectID:      v.id.ProjectID,
		StoreHeadDigest:     v.headDigest,
	}
	rep.Diagnostics = append(rep.Diagnostics,
		Diagnostic{Code: "REPLAY_REQUIRED", Subject: "status",
			Message: "status is recorded-only freshness evidence; replay not performed this invocation; no hash-only receipt proves tests ran"},
		Diagnostic{Code: "REPLAY_REQUIRED", Subject: "negative-sensitivity",
			Message: "negative sensitivity is demonstrated only by replay; status never demonstrates it"},
	)
	// design consistency: the in-memory finalized reconciliation
	if err := Validate(req.Plan, req.Manifests, req.Inventory); err != nil {
		rep.DesignConsistency = "failed"
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
			Code: "MISSING_CONTRACT", Subject: "validate",
			Message: fmt.Sprintf("declaration reconciliation failed: %v", err),
		})
	}
	// current inventories: source (role-classified), control, judgment
	var manifestForWalk *Manifest
	if len(req.Manifests) > 0 {
		manifestForWalk = &req.Manifests[0]
	}
	engine, err := newInspectEngine(ctx, req, manifestForWalk)
	if err != nil {
		return StatusReport{}, err
	}
	rep.SourceDigest = engine.sourceDigest
	rep.ControlDigest = engine.controlDigest
	rep.JudgmentDigest = engine.judgmentDigest
	// registration state: exact authored bytes against the head entries
	planDigest := digestOfBytes(req.Plan.Raw())
	planEntryFound := false
	planCurrent := false
	for _, p := range v.head.Plans {
		if p.Design == req.Plan.Design {
			planEntryFound = true
			planCurrent = p.PlanDigest == planDigest
		}
	}
	if planEntryFound && !planCurrent {
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
			Code: "STALE_INPUT", Subject: "plan",
			Message: "the plan bytes are a stale authored revision relative to the registered head entry",
		})
	}
	for i := range req.Manifests {
		m := &req.Manifests[i]
		found, registered := false, false
		for _, hm := range v.head.Milestones {
			if hm.Design != req.Plan.Design || hm.Milestone != m.ID {
				continue
			}
			found = true
			registered = hm.ManifestDi == digestOfBytes(m.Raw())
		}
		switch {
		case found && registered:
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
				Code: "MISSING_CONTRACT", Subject: m.ID,
				Message: fmt.Sprintf("milestone %s is registered-not-executed; registration is authored state, never execution authority", m.ID),
			})
		case found && !registered:
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
				Code: "STALE_INPUT", Subject: m.ID,
				Message: fmt.Sprintf("milestone %s local bytes are a stale authored revision relative to the registered head entry", m.ID),
			})
		default:
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
				Code: "MISSING_CONTRACT", Subject: m.ID,
				Message: fmt.Sprintf("milestone %s is an unregistered-draft; capture does not register and no draft receives an executable pass", m.ID),
			})
		}
	}
	// recorded-only execution history per (design, milestone)
	runs, err := readRunSummaries(req.Store)
	if err != nil {
		return StatusReport{}, err
	}
	latest := map[string]runSummary{}
	for _, r := range runs {
		latest[r.design+"\x00"+r.milestone] = r // deterministic sorted order
	}
	anyRun, latestResult := false, ""
	historical := false
	for i := range req.Manifests {
		m := &req.Manifests[i]
		if r, ok := latest[req.Plan.Design+"\x00"+m.ID]; ok {
			anyRun = true
			historical = true
			latestResult = r.result
			rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
				Code: "REPLAY_REQUIRED", Subject: m.ID,
				Message: fmt.Sprintf("milestone %s has historical recorded-only %s/%s evidence; replay not performed this invocation", m.ID, r.phase, r.result),
			})
		}
	}
	switch {
	case anyRun && (latestResult == "fail" || latestResult == "error"):
		rep.TestExecution = "failed"
	case anyRun:
		rep.TestExecution = "recorded-only"
	}
	// source/test binding against the retained baseline bundle
	binding, err := bindingState(req, engine)
	if err != nil {
		return StatusReport{}, err
	}
	rep.SourceTestBinding = binding
	if binding == "stale" {
		rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
			Code: "STALE_INPUT", Subject: "binding",
			Message: "frozen, dependency or design inputs drifted from the retained baseline; current evidence is stale",
		})
	}
	// judgment freshness relative to recorded history
	if rep.JudgmentDigest != "" {
		rep.Judgment = "bound"
		if historical {
			boundCurrent := false
			for _, r := range latest {
				if r.judgment == rep.JudgmentDigest {
					boundCurrent = true
				}
			}
			if !boundCurrent {
				rep.Judgment = "stale"
				rep.Diagnostics = append(rep.Diagnostics, Diagnostic{
					Code: "STALE_INPUT", Subject: "judgment",
					Message: "run history binds an older judgment control; refresh the judgment for the current implementation",
				})
			}
		}
	}
	return rep, nil
}

// bindingState compares the current role-classified inventories against the
// retained baseline bundle: frozen/dependency/design inputs must match
// exactly; subject drift is legitimate implementation change. A baseline
// bundle that is absent is a missing binding; a bundle whose blobs vanished
// is hard corruption.
func bindingState(req StatusRequest, engine *inspectEngine) (string, error) {
	if len(req.Manifests) == 0 {
		return "missing", nil
	}
	m := &req.Manifests[0]
	if m.Baseline == "" {
		return "missing", nil
	}
	raw, _, err := readControlJSON(filepath.Join(req.Store, storeObjects, strings.TrimPrefix(m.Baseline, "sha256:"), "bundle.json"))
	if err != nil {
		return "missing", nil
	}
	entries, treeDigest, err := decodeBundleClosed(raw)
	if err != nil || treeDigest != m.Baseline {
		return "missing", nil
	}
	for _, e := range entries {
		if e.Kind != "file" {
			continue
		}
		if _, err := os.Lstat(filepath.Join(req.Store, storeBlobs, strings.TrimPrefix(e.Digest, "sha256:"))); err != nil {
			return "", fmt.Errorf("INVALID_SCHEMA: baseline blob %s is missing from the store; missing store content is blocking", e.Digest)
		}
	}
	current := map[string][]bundleEntryData{}
	for _, e := range engine.sourceEntries {
		current[e.Role] = append(current[e.Role], e)
	}
	retained := map[string][]bundleEntryData{}
	for _, e := range entries {
		retained[e.Role] = append(retained[e.Role], e)
	}
	for _, role := range []string{RoleFrozen, RoleDependency, RoleDesign} {
		if treeDigestOfEntries(current[role]) != treeDigestOfEntries(retained[role]) {
			return "stale", nil
		}
	}
	return "current", nil
}

// readRunSummaries reads runs/<run id>.json in sorted order, validating the
// fields status reports.
func readRunSummaries(store string) ([]runSummary, error) {
	dir := filepath.Join(store, storeRuns)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("INVALID_SCHEMA: cannot enumerate runs/: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !runNameRe.MatchString(e.Name()) {
			return nil, fmt.Errorf("INVALID_SCHEMA: runs/ holds unknown entry %q", e.Name())
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var out []runSummary
	for _, name := range names {
		raw, _, err := readControlJSON(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		_, doc, err := readControlJSONFromBytes(name, raw)
		if err != nil {
			return nil, err
		}
		root, err := asObject(doc, name)
		if err != nil {
			return nil, err
		}
		schema, err := reqStr(root, "schema", name+".schema")
		if err != nil || schema != "machinery.tdd.run/v1" {
			return nil, fmt.Errorf("INVALID_SCHEMA: %s is not a machinery.tdd.run/v1 record", name)
		}
		var r runSummary
		r.design, _ = reqStr(root, "design", name+".design")
		r.milestone, _ = reqStr(root, "milestone", name+".milestone")
		r.result, _ = reqStr(root, "result", name+".result")
		r.phase, _ = reqStr(root, "phase", name+".phase")
		r.judgment, _ = reqStr(root, "judgment_control_digest", name+".judgment_control_digest")
		switch r.result {
		case "pass", "fail", "error":
		default:
			return nil, fmt.Errorf("INVALID_SCHEMA: %s.result %q is not a closed result", name, r.result)
		}
		out = append(out, r)
	}
	return out, nil
}

// inspectEngine recomputes the current inventories without writing.
type inspectEngine struct {
	sourceDigest   string
	controlDigest  string
	judgmentDigest string
	sourceEntries  []bundleEntryData
}

// newInspectEngine walks the held view read-only with the same classifier
// and exclusions as capture.
func newInspectEngine(ctx context.Context, req StatusRequest, m *Manifest) (*inspectEngine, error) {
	creq := CaptureRequest{
		Inputs: req.Inputs,
		Limits: Limits{
			Entries:     protocol.LimitEntriesDefault,
			Depth:       protocol.LimitDepthDefault,
			BundleBytes: protocol.LimitBundleDefault,
		},
	}
	if m != nil {
		creq.Manifest = *m
		if err := validateCaptureRequest(creq); err != nil {
			return nil, err
		}
	}
	engine, err := newCaptureEngine(ctx, creq, nil)
	if err != nil {
		return nil, err
	}
	defer engine.root.Close()
	if err := engine.walk(); err != nil {
		return nil, err
	}
	ctl, err := engine.captureControls(false)
	if err != nil {
		return nil, err
	}
	jud, err := engine.captureJudgment(false)
	if err != nil {
		return nil, err
	}
	return &inspectEngine{
		sourceDigest:   treeDigestOfEntries(engine.entries),
		controlDigest:  ctl,
		judgmentDigest: jud,
		sourceEntries:  engine.entries,
	}, nil
}

var _ = path.Join
