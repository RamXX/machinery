// Gk-checkers: the pluggable external-checker gate. Deterministic half only (no
// external engine): for each design/checkers/<id>.checker.yaml, the manifest
// reconciles against the domain model, the committed projection is fresh, and the
// committed evidence binds to the current design and reports a pass with complete
// coverage. Running the engine is verify-checkers' job, exactly as verify-formal
// is for the relational layers. See docs/external-checkers.md.

package gates

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/version"
)

// HasCheckers reports whether the design opted into the external-checker layer.
// A discovery error returns true so the suite runs Gk and surfaces the error;
// filesystem faults must never be interpreted as an absent optional layer.
func HasCheckers(design string) bool {
	has, err := checker.HasCheckers(design)
	return has || err != nil
}

// CheckExternalCheckers runs Gk for every committed manifest, one gate each, in
// manifest-path order so the output is stable.
func CheckExternalCheckers(design string) []*Gate {
	paths, err := checker.ManifestPaths(design)
	if err != nil || len(paths) == 0 {
		g := NewGate("Gk external checker discovery")
		g.startOrder()
		if err != nil {
			g.Errs = append(g.Errs, err.Error())
		} else {
			g.Errs = append(g.Errs, "no checkers/*.checker.yaml in "+design+"; explicitly requested Gk has no checker contract to verify")
		}
		return []*Gate{g}
	}
	var out []*Gate
	var manifests []*checker.Manifest
	for _, mp := range paths {
		out = append(out, checkOneChecker(design, mp))
		if manifest, loadErr := checker.LoadManifest(mp); loadErr == nil {
			manifests = append(manifests, manifest)
		}
	}
	if err := checker.ValidateManifestSet(manifests); err != nil {
		out[0].Errs = append(out[0].Errs, err.Error())
	}
	out[0].Errs = append(out[0].Errs, checker.InventoryProblems(design, manifests)...)
	return out
}

func checkOneChecker(design, manifestPath string) *Gate {
	base := filepath.Base(manifestPath)
	man, err := checker.LoadManifest(manifestPath)
	title := "Gk external checker " + base
	if err == nil {
		title = "Gk-" + man.Checker.ID + " external checker"
	}
	g := NewGate(title)
	g.startOrder()
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}

	// The model is the projection source. Its absence is a hard failure: a
	// checker over no domain is meaningless.
	modelPaths := checker.ModelPaths(design)
	if len(modelPaths) != 1 {
		g.Errs = append(g.Errs, "expected exactly one *.modelith.yaml in "+design+"; found "+fmt.Sprint(len(modelPaths)))
		return g
	}
	modelPath := modelPaths[0]
	model, err := checker.LoadModel(modelPath)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}

	// A claim over invariants the projection never carries is an incoherent
	// manifest, caught before anything downstream trusts the coverage.
	includesInvariants := false
	for _, l := range man.Projection.Include {
		if l == "invariants" {
			includesInvariants = true
		}
	}
	if len(man.Coverage.Claim) > 0 && !includesInvariants {
		g.Errs = append(g.Errs, "coverage.claim is set but projection.include does not include \"invariants\"; the claim can never be reconciled")
		return g
	}

	designID, err := checker.DesignID(modelPath)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	fresh, err := checker.Generate(model, man, designID, version.Version)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	freshHash, err := fresh.InputHash()
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	g.Count("layers projected", len(fresh.Include))

	// Reconcile the claim and residuals against the model.
	invIDs := model.InvariantIDs()
	claimed := claimedInvariants(model, man.Coverage.Claim)
	residual := map[string]bool{}
	for _, r := range man.Coverage.Residuals {
		if !invIDs[r.ID] {
			g.Errs = append(g.Errs, "residual names unknown invariant "+quoteID(r.ID))
			continue
		}
		if strings.TrimSpace(r.Reason) == "" {
			g.Errs = append(g.Errs, "residual "+quoteID(r.ID)+" has no reason; a waiver without a reason is not a waiver")
		}
		if !claimed[r.ID] {
			g.Errs = append(g.Errs, "residual "+quoteID(r.ID)+" is not matched by coverage.claim; a waiver cannot exist outside the checker's obligation set")
		}
		residual[r.ID] = true
	}
	if len(man.Coverage.Claim) > 0 && len(claimed) == 0 {
		g.Errs = append(g.Errs, "coverage.claim matched no invariants in the model; an empty obligation set is not coverage")
	}
	if len(claimed) > 0 {
		g.Count("invariants claimed", len(claimed))
	}
	if len(residual) > 0 {
		g.Count("residuals (waived with reason)", len(residual))
	}

	// Freshness of the committed projection. An existing-but-empty file must
	// take the parse path, not fall between the branches: a zero-byte
	// committed projection once produced no finding at all (neither branch
	// matched), silently dropping the freshness obligation.
	projectionBytes, projectionErr := checker.ReadConfinedFile(design, man.Evidence.ProjectionOut)
	if projectionErr == nil {
		committed, perr := checker.ParseProjection(projectionBytes)
		switch {
		case perr != nil:
			g.Errs = append(g.Errs, man.Evidence.ProjectionOut+" is not valid projection JSON: "+perr.Error())
		default:
			if cv := committed.MachineryVersion; cv != "" && cv != version.Version {
				if g.stampVersions == nil {
					g.stampVersions = map[string]bool{}
				}
				g.stampVersions[cv] = true
			}
			eq, cerr := checker.ContentEqual(committed, fresh)
			switch {
			case cerr != nil:
				g.Errs = append(g.Errs, cerr.Error())
			case !eq:
				g.Drift = append(g.Drift, man.Evidence.ProjectionOut+" is stale: it does not match a fresh projection of the domain model. Run 'machinery project' and commit the output.")
			default:
				g.Count("committed projection fresh")
			}
		}
	} else if errors.Is(projectionErr, os.ErrNotExist) {
		g.Drift = append(g.Drift, man.Evidence.ProjectionOut+" is not committed; the checker input was never generated. Run 'machinery project' and commit it.")
	} else {
		g.Errs = append(g.Errs, man.Evidence.ProjectionOut+" is unreadable: "+projectionErr.Error())
	}

	// Evidence: absence is failure.
	ev, err := checker.LoadEvidenceConfined(design, man.Evidence.EvidenceIn)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			g.Errs = append(g.Errs, man.Evidence.EvidenceIn+" is not committed; the checker was requested but never ran. An empty check is a failure, not a pass.")
			return g
		}
		g.Errs = append(g.Errs, err.Error())
		return g
	}
	if ev.Checker.ID != man.Checker.ID {
		g.Errs = append(g.Errs, "evidence checker id "+quoteID(ev.Checker.ID)+" does not match manifest "+quoteID(man.Checker.ID)+"; this evidence was produced for a different checker")
	}
	if ev.RuntimeClosure != man.Checker.RuntimeClosure {
		g.Errs = append(g.Errs, "evidence runtime_closure does not match checker.runtime_closure in the manifest; the verdict was produced by a different runtime closure")
	} else {
		g.Count("evidence bound to runtime closure")
	}
	// Binding: does the verdict cover the current design?
	if ev.InputHash != freshHash {
		g.Drift = append(g.Drift, "evidence input_hash does not match the current design projection; the verdict was computed over a different design. Re-run the checker and commit fresh evidence.")
	} else {
		g.Count("evidence bound to design")
	}

	surfaceFindings(g, ev)
	reconcileCoverage(g, fresh, claimed, residual, ev)
	g.Count("elements covered", len(ev.Coverage))
	return g
}

// claimedInvariants returns the invariant ids matched by any claim glob.
func claimedInvariants(m *checker.Model, claim []string) map[string]bool {
	out := map[string]bool{}
	for _, iv := range m.Invariants {
		for _, glob := range claim {
			if ok, _ := path.Match(glob, iv.ID); ok {
				out[iv.ID] = true
				break
			}
		}
	}
	return out
}

// surfaceFindings renders the checker's findings and the verdict. A blocking
// finding, or a fail verdict, is an ERROR; advisory is a warn; info is a note.
func surfaceFindings(g *Gate, ev *checker.Evidence) {
	blocking := 0
	for _, f := range ev.Findings {
		line := f.Message
		if f.Code != "" {
			line = f.Code + ": " + line
		}
		if f.Element != "" {
			line += " [" + f.Element + "]"
		}
		switch f.Severity {
		case "blocking":
			blocking++
			g.Errs = append(g.Errs, line)
		case "advisory":
			g.Warns = append(g.Warns, line)
		default:
			g.Notes = append(g.Notes, line)
		}
	}
	for _, row := range ev.Coverage {
		if row.Verdict == "fail" {
			g.Errs = append(g.Errs, "checker coverage verdict: fail ["+row.Element+"]")
		}
	}
	switch ev.Verdict {
	case "fail":
		g.Errs = append(g.Errs, "checker verdict: fail")
	case "pass":
		if blocking > 0 {
			g.Errs = append(g.Errs, "checker reports a pass verdict but carries blocking findings; the verdict and the findings disagree")
		}
	}
}

// reconcileCoverage enforces the hard rule: every claimed invariant is covered by
// evidence or a declared residual. It also rejects evidence that decides an
// element the projection does not carry.
func reconcileCoverage(g *Gate, p *checker.Projection, claimed, residual map[string]bool, ev *checker.Evidence) {
	covered := map[string]bool{}
	for _, row := range ev.Coverage {
		covered[row.Element] = true
	}
	var missing []string
	for id := range claimed {
		if residual[id] || covered["inv:"+id] {
			continue
		}
		missing = append(missing, id)
	}
	sort.Strings(missing)
	for _, id := range missing {
		g.Errs = append(g.Errs, "claimed invariant "+quoteID(id)+" is neither covered by evidence nor a declared residual; coverage is a hard rule")
	}

	known := projectionElementIDs(p)
	var unknown []string
	for _, row := range ev.Coverage {
		if !known[row.Element] {
			unknown = append(unknown, row.Element)
		}
	}
	for _, e := range uniqueSorted(unknown) {
		g.Errs = append(g.Errs, "evidence covers "+quoteID(e)+", which is not an element in the projection")
	}
}

// projectionElementIDs collects every stable_id the projection carries, so the
// gate can tell an evidence row about a real element from one about a phantom.
func projectionElementIDs(p *checker.Projection) map[string]bool {
	ids := map[string]bool{}
	if p.Model != nil {
		for _, e := range p.Model.Entities {
			ids[e.StableID] = true
			for _, a := range e.Attributes {
				ids[a.StableID] = true
			}
		}
		for _, iv := range p.Model.Invariants {
			ids[iv.StableID] = true
		}
		for _, r := range p.Model.Relationships {
			ids[r.StableID] = true
		}
	}
	return ids
}

func quoteID(s string) string { return "'" + s + "'" }

func uniqueSorted(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
