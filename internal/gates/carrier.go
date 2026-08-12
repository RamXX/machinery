// Gc-carrier: invariant carrier reconciliation. Declaring an invariant is
// nearly free; carrying it costs something, and nothing else forces the
// trade at declaration time. This gate does: every declared invariant must
// have a named carrier: an action that lists it in preserves, a relational
// layer that compiles it, or an explicit waiver with a reason stating where
// enforcement lands. It needs only the domain model, so it runs from
// Phase 1; Gx-trace re-checks enforcement against the machines and the
// BUILD tables once those exist, but by then the model has been green under
// this reconciliation the whole way up.
//
// Waivers live in two places, both already validated elsewhere or here:
// a relational layer's residuals: section (id + reason, held by the layer
// generator), and the design-level annex formal/waivers.yaml (id + reason,
// held here) for invariants no relational layer touches. The annex is
// reconciled in both directions: a waiver naming an undeclared invariant is
// as much a finding as an invariant nothing carries.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/ir"
)

// WaiverAnnexName is the design-level waiver annex file under formal/.
const WaiverAnnexName = "waivers.yaml"

// waiverTopKeys and waiverEntryKeys pin the annex schema; an unknown key is
// a typo that would otherwise silently waive nothing.
var (
	waiverTopKeys   = map[string]bool{"waivers": true, "_comment": true}
	waiverEntryKeys = map[string]bool{"invariant": true, "reason": true, "_comment": true}
)

// loadWaiverAnnex reads formal/waivers.yaml and returns the waived ids in
// declaration order. An absent annex is fine (empty result); a malformed
// one is an ERROR because a waiver that does not parse waives nothing.
func loadWaiverAnnex(design string, g *Gate) []string {
	path := filepath.Join(design, "formal", WaiverAnnexName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	v, err := ir.LoadYAML(data)
	if err != nil {
		g.Errs = append(g.Errs, "formal/"+WaiverAnnexName+": invalid YAML: "+err.Error())
		return nil
	}
	root := v.AsObject()
	if root == nil {
		g.Errs = append(g.Errs, "formal/"+WaiverAnnexName+": not a yaml mapping (empty file?)")
		return nil
	}
	for _, k := range root.Keys() {
		if !waiverTopKeys[k] {
			g.Errs = append(g.Errs, fmt.Sprintf("formal/%s: unknown key %s (expected waivers:)", WaiverAnnexName, ir.Repr(k)))
		}
	}
	var out []string
	seen := map[string]bool{}
	for i, wv := range objSlice(root.Get2("waivers")) {
		wo := wv.AsObject()
		if wo == nil {
			g.Errs = append(g.Errs, fmt.Sprintf("formal/%s: waivers[%d] is not a mapping", WaiverAnnexName, i))
			continue
		}
		for _, k := range wo.Keys() {
			if !waiverEntryKeys[k] {
				g.Errs = append(g.Errs, fmt.Sprintf("formal/%s: waivers[%d] has unknown key %s", WaiverAnnexName, i, ir.Repr(k)))
			}
		}
		id, reason := wo.GetString("invariant"), wo.GetString("reason")
		if id == "" || reason == "" {
			g.Errs = append(g.Errs, fmt.Sprintf("formal/%s: waivers[%d] needs both an invariant id and a reason; an unexplained waiver is a hole", WaiverAnnexName, i))
			continue
		}
		if seen[id] {
			g.Errs = append(g.Errs, fmt.Sprintf("formal/%s: invariant %s is waived twice", WaiverAnnexName, ir.Repr(id)))
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// checkerClaims returns the invariant ids carried by an external checker's
// coverage.claim globs and the ids its coverage residuals waive, restricted
// to the declared set. Gk owns validating the manifests, the globs, and the
// evidence behind the claims; a malformed manifest contributes nothing here.
func checkerClaims(design string, declared map[string]bool) (claimed, waived map[string]bool) {
	claimed, waived = map[string]bool{}, map[string]bool{}
	for _, mp := range checker.ManifestPaths(design) {
		man, err := checker.LoadManifest(mp)
		if err != nil {
			continue
		}
		residual := map[string]bool{}
		for _, r := range man.Coverage.Residuals {
			if declared[r.ID] {
				residual[r.ID] = true
				waived[r.ID] = true
			}
		}
		for id := range declared {
			if residual[id] {
				continue
			}
			for _, pat := range man.Coverage.Claim {
				if ok, _ := filepath.Match(pat, id); ok {
					claimed[id] = true
					break
				}
			}
		}
	}
	return claimed, waived
}

// annexWaiverIDs reads the annex leniently for cross-gate crediting
// (Gx-trace consumes it the way it consumes alloy.ResidualIDs); Gc-carrier
// owns validation and error reporting. Absent or malformed yields nil.
func annexWaiverIDs(design string) map[string]bool {
	data, err := os.ReadFile(filepath.Join(design, "formal", WaiverAnnexName))
	if err != nil {
		return nil
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil
	}
	ids := map[string]bool{}
	for _, wv := range objSlice(v.AsObject().Get2("waivers")) {
		if wo := wv.AsObject(); wo != nil && wo.GetString("invariant") != "" && wo.GetString("reason") != "" {
			ids[wo.GetString("invariant")] = true
		}
	}
	return ids
}

// CheckCarriers implements Gc-carrier.
func CheckCarriers(design string) *Gate {
	g := NewGate("Gc-carrier  invariant carrier reconciliation")
	g.startOrder()
	dm := loadModelith(design, g)
	if dm == nil {
		return g
	}
	dmo := dm.AsObject()

	// declared invariants: top-level plus per-entity, one global id space
	declared := map[string]bool{}
	for _, i := range objSlice(dmo.Get2("invariants")) {
		if id := i.AsObject().GetString("id"); id != "" {
			declared[id] = true
		}
	}
	entities := dmo.GetObject("entities")
	for _, ename := range entities.Keys() {
		e := entities.Get2(ename).AsObject()
		for _, i := range objSlice(e.Get2("invariants")) {
			if id := i.AsObject().GetString("id"); id != "" {
				declared[id] = true
			}
		}
	}
	g.Count("invariants declared", len(declared))

	// preserves references: the model's own carrier claims, bound both ways
	// (an unknown id in preserves is a claim that binds to nothing)
	preserved := map[string]bool{}
	for _, ename := range entities.Keys() {
		e := entities.Get2(ename).AsObject()
		for _, a := range objSlice(e.Get2("actions")) {
			ao := a.AsObject()
			if ao == nil {
				continue
			}
			aname := ao.GetString("name")
			for _, pv := range listStrings(ao.Get2("preserves")) {
				g.Count("preserves references", 1)
				if !declared[pv] {
					g.Errs = append(g.Errs, fmt.Sprintf("%s.%s preserves unknown invariant %s (typo or a renamed id; a carrier claim must bind to a declared invariant)", ename, aname, ir.Repr(pv)))
					continue
				}
				preserved[pv] = true
			}
		}
	}

	// relational layers: carried ids credit the layer, residual ids credit
	// the waiver idiom (the layer generators validate both)
	formal := filepath.Join(design, "formal")
	carried := map[string]bool{}
	waived := map[string]bool{}
	var absent []string
	for _, lay := range []struct {
		name string
		ann  string
		ids  func(string) map[string]bool
	}{
		{"policy", alloy.AnnotationName, alloy.CarriedIDs},
		{"integrity", alloy.IntegrityAnnotationName, alloy.CarriedIntegrityIDs},
		{"isolation", alloy.IsolationAnnotationName, alloy.CarriedIsolationIDs},
	} {
		path := filepath.Join(formal, lay.ann)
		if _, err := os.Stat(path); err != nil {
			absent = append(absent, lay.name)
			continue
		}
		for id := range lay.ids(path) {
			carried[id] = true
		}
		for id := range alloy.ResidualIDs(path) {
			waived[id] = true
		}
	}
	if len(absent) > 0 {
		g.Notes = append(g.Notes, "relational layers not opted in: "+strings.Join(absent, ", "))
	}

	// external checkers carry what they claim: a manifest's coverage.claim
	// globs name the invariants its evidence must cover (Gk reconciles the
	// evidence; here the registered claim is the carrier), and its coverage
	// residuals are waivers with a reason, validated by Gk.
	checkerCarried, checkerWaived := checkerClaims(design, declared)
	for id := range checkerWaived {
		waived[id] = true
	}

	// machines, once authored, carry invariants too: a matrix row's maps-to
	// cell is reconciled against the machine's real guards and actions by
	// G3, so it is a checked carrier, not prose. Before Phase 3 there are
	// no matrices and this corpus is empty, which costs nothing; that is
	// the point of running from Phase 1.
	var unitCells []string
	for _, f := range globExt(filepath.Join(design, "machines"), ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readOrEmpty(f)) {
			mi := ir.FindCol(tbl.Header, "maps to")
			if mi < 0 {
				continue
			}
			for _, r := range tbl.Rows {
				if mi < len(r) {
					unitCells = append(unitCells, r[mi])
				}
			}
		}
	}
	unitCorpus := strings.Join(unitCells, "\n")

	// annex waivers, reconciled against the declared set
	for _, id := range loadWaiverAnnex(design, g) {
		if !declared[id] {
			g.Errs = append(g.Errs, fmt.Sprintf("formal/%s waives unknown invariant %s (stale entry or a typo)", WaiverAnnexName, ir.Repr(id)))
			continue
		}
		if preserved[id] || carried[id] || checkerCarried[id] || tokenIn(id, unitCorpus) {
			g.Warns = append(g.Warns, fmt.Sprintf("formal/%s waives invariant %s, but it already has a carrier; remove the stale waiver", WaiverAnnexName, ir.Repr(id)))
			continue
		}
		waived[id] = true
	}

	// the reconciliation: every declared invariant lands in exactly one
	// bucket, and the empty bucket is the finding
	var ids []string
	for id := range declared {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		switch {
		case preserved[id]:
			g.Count("carried by preserves")
		case carried[id]:
			g.Count("carried by a relational layer")
		case tokenIn(id, unitCorpus):
			g.Count("carried by a machine unit")
		case checkerCarried[id]:
			g.Count("carried by an external checker")
		case waived[id]:
			g.Count("waived with reason")
		default:
			g.Errs = append(g.Errs, "invariant "+ir.Repr(id)+" has no carrier: no action preserves it, no relational layer compiles it, no machine unit maps to it, no external checker claims it, and no waiver states where enforcement lands (name it in an action's preserves, carry it in a relational layer, or waive it in formal/"+WaiverAnnexName+")")
		}
	}
	g.RequireNonzero("invariants declared", "the domain model declares no invariants")
	return g
}

// listStrings returns the string items of a scalar-or-list value.
func listStrings(v *ir.Value) []string {
	if v == nil {
		return nil
	}
	items := []*ir.Value{v}
	if v.Kind == ir.KindArray {
		items = v.AsArray()
	}
	var out []string
	for _, it := range items {
		if it != nil && it.Kind == ir.KindString {
			out = append(out, it.AsString())
		}
	}
	return out
}
