package gates

// G5-pack: the recursive-decomposition gate. Two modes, selected by what the
// design contains:
//
//	parent (design/decomposition.yaml): the committed packs must byte-match a
//	fresh generation, and every pinned child must have been built against the
//	CURRENT pack (two-way hash pinning: stale packs are the recursion analog
//	of stale oracles).
//
//	child (design/pack/pack.yaml): the copied pack verifies its content hash;
//	the packmap pins that same hash and reconciles against both machines; the
//	committed refinement artifacts byte-match a fresh generation (TLC checks
//	them via verify-formal); the owned entities keep the pack's public shape;
//	the delegated invariants trace; and every boundary event is handled or
//	produced.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
)

// CheckPack implements G5-pack. Applicable when the design is a decomposed
// parent, a decomposed child, or both (a mid-level design in a deeper tree).
func CheckPack(design string) *Gate {
	g := NewGate("G5-pack  contract packs")
	g.startOrder()
	isParent := pack.HasDecomposition(design)
	isChild := pack.HasPack(design)
	if !isParent && !isChild {
		g.Errs = append(g.Errs, "nothing to check: no decomposition.yaml (parent) and no pack/ (child); G5 does not apply to this design")
		return g
	}
	if isParent {
		checkParentPacks(design, g)
	}
	if isChild {
		checkChildPack(design, g)
	}
	return g
}

func checkParentPacks(design string, g *Gate) {
	fresh, err := pack.GeneratePacks(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return
	}
	d, _ := pack.LoadDecomposition(design)
	var ids []string
	for id := range fresh {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		g.Count("subsystems")
		dir := filepath.Join(design, "packs", id+".pack")
		stale := false
		var names []string
		for name := range fresh[id] {
			names = append(names, name)
		}
		sort.Strings(names) // deterministic finding order across runs
		for _, name := range names {
			want := fresh[id][name]
			got, rerr := os.ReadFile(filepath.Join(dir, name))
			if rerr != nil {
				g.Errs = append(g.Errs, "pack "+id+": missing committed file "+name+"; run machinery pack generate")
				stale = true
				continue
			}
			if string(got) != want {
				g.Drift = append(g.Drift, "pack "+id+": committed "+name+" is stale (differs from a fresh generation); rerun machinery pack generate and re-issue the pack to the child")
				stale = true
			}
		}
		// extra committed files are stale artifacts a fresh generation never
		// produces; the child would copy and trust them
		if entries, rerr := os.ReadDir(dir); rerr == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				if _, ok := fresh[id][e.Name()]; !ok {
					g.Errs = append(g.Errs, "pack "+id+": committed file "+e.Name()+" is not part of a fresh generation; remove it and rerun machinery pack generate")
					stale = true
				}
			}
		}
		if !stale {
			g.Count("packs fresh")
		}
	}
	// two-way pinning: children built against the current pack
	for _, s := range d.Subsystems {
		if s.ChildDesign == "" {
			// non-blocking: the parent-first workflow generates packs before
			// any child design exists, but an unpinned subsystem is unchecked
			g.Count("children unpinned")
			g.Warns = append(g.Warns, "subsystem "+s.ID+" declares no child_design; its pack pin is unchecked (set child_design in decomposition.yaml once the child design exists)")
			continue
		}
		childDir := filepath.Join(design, s.ChildDesign)
		pm, err := pack.LoadPackMap(childDir)
		if err != nil {
			g.Errs = append(g.Errs, "subsystem "+s.ID+": child design "+s.ChildDesign+" has no readable packmap.yaml: "+err.Error())
			continue
		}
		currentHash := pack.ContentHash(fresh[s.ID])
		if pm.PackHash != currentHash {
			g.Errs = append(g.Errs, fmt.Sprintf("subsystem %s: the child at %s was built against pack %.12s but the current pack is %.12s; regenerate the pack, re-copy it to the child, and re-run the child's gates", s.ID, s.ChildDesign, pm.PackHash, currentHash))
		} else {
			g.Count("children pinned")
		}
	}
	g.RequireNonzero("subsystems", "no subsystems parsed from decomposition.yaml")
}

func checkChildPack(design string, g *Gate) {
	manifest, err := pack.LoadPackManifest(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return
	}
	files, err := pack.PackFilesOnDisk(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
		return
	}
	g.Count("pack files", len(files))
	if got := pack.ContentHash(files); got != manifest.GetString("content_hash") {
		g.Errs = append(g.Errs, "the copied pack fails its own content hash: it was edited by hand or partially copied; the pack is the parent's frozen artifact, re-copy it")
		return
	}
	g.Count("pack hash verified")

	// packmap present, pinned, reconciled; refinement artifacts fresh
	freshRef, err := pack.GenerateRefinement(design)
	if err != nil {
		g.Errs = append(g.Errs, err.Error())
	} else {
		var names []string
		for n := range freshRef {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			got, rerr := os.ReadFile(filepath.Join(design, "formal", n))
			if rerr != nil {
				g.Errs = append(g.Errs, "no committed refinement artifact formal/"+n+"; run machinery pack refine and commit it (verify-formal TLC-checks it)")
			} else if string(got) != freshRef[n] {
				g.Drift = append(g.Drift, "committed formal/"+n+" is stale (differs from a fresh generation); rerun machinery pack refine")
			} else {
				g.Count("refinement artifacts fresh")
			}
		}
	}

	// owned entities keep the pack's public shape (lifecycle enum values exact)
	sliceV, err := ir.LoadYAML([]byte(files["domain.modelith.yaml"]))
	if err == nil && sliceV.AsObject() != nil {
		checkOwnedShape(design, sliceV, g)
	} else {
		g.Errs = append(g.Errs, "pack domain.modelith.yaml is not parseable yaml")
	}

	// delegated invariants must trace in the child (matrices or BUILD tables)
	checkDelegatedInvariants(design, manifest, g)

	// boundary events: consumed handled somewhere, produced emitted somewhere
	checkBoundaryEvents(design, files["events.md"], g)
}

func checkOwnedShape(design string, slice *ir.Value, g *Gate) {
	childDM := loadModelith(design, g)
	if childDM == nil {
		return
	}
	childEntities := childDM.AsObject().GetObject("entities")
	childEnums := childDM.AsObject().GetObject("enums")
	so := slice.AsObject()
	sliceEnums := so.GetObject("enums")
	entities := so.GetObject("entities")
	for _, en := range entities.Keys() {
		if !childEntities.Has(en) {
			g.Errs = append(g.Errs, "owned entity "+en+" is missing from the child domain model; the pack's public shape is frozen")
			continue
		}
		g.Count("owned entities present")
		// lifecycle enum values must match exactly
		for _, a := range entities.Get2(en).AsObject().Get2("attributes").AsArray() {
			ao := a.AsObject()
			t := ao.GetString("type")
			if !sliceEnums.Has(t) {
				continue
			}
			want := enumValues(sliceEnums.Get2(t))
			got := enumValues(childEnums.Get2(t))
			if strings.Join(want, "|") != strings.Join(got, "|") {
				g.Errs = append(g.Errs, "enum "+t+" drifted from the pack: pack ["+strings.Join(want, ", ")+"] vs child ["+strings.Join(got, ", ")+"]; the public lifecycle shape is frozen (internal states live in the machine, not the enum)")
			} else {
				g.Count("owned enums shape-checked")
			}
		}
	}
}

func enumValues(v *ir.Value) []string {
	var out []string
	if v == nil {
		return out
	}
	for _, e := range v.AsObject().Get2("values").AsArray() {
		out = append(out, e.AsObject().GetString("name"))
	}
	return out
}

func checkDelegatedInvariants(design string, manifest *ir.Object, g *Gate) {
	var ids []string
	for _, v := range manifest.Get2("delegated_invariants").AsArray() {
		if v != nil && v.Kind == ir.KindString {
			ids = append(ids, v.AsString())
		}
	}
	if len(ids) == 0 {
		g.Count("delegated invariants", 0)
		return
	}
	var cells []string
	for _, f := range sortedGlobExt(filepath.Join(design, "machines"), ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readOrEmpty(f)) {
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(design, "BUILD.md")); err == nil {
		for _, tbl := range ir.ParseMdTables(readOrEmpty(filepath.Join(design, "BUILD.md"))) {
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
			}
		}
	}
	corpus := strings.Join(cells, "\n")
	sort.Strings(ids)
	for _, iid := range ids {
		if tokenIn(iid, corpus) {
			g.Count("delegated invariants traced")
		} else {
			g.Errs = append(g.Errs, "delegated invariant "+ir.Repr(iid)+" is referenced by no matrix or BUILD.md table in the child; the parent delegated its enforcement here")
		}
	}
}

func checkBoundaryEvents(design, eventsMD string, g *Gate) {
	type ev struct{ name, dir string }
	var evs []ev
	for _, tbl := range ir.ParseMdTables(eventsMD) {
		ei := ir.FindCol(tbl.Header, "event")
		di := ir.FindCol(tbl.Header, "direction")
		if ei < 0 || di < 0 {
			continue
		}
		for _, r := range tbl.Rows {
			if ei < len(r) && di < len(r) {
				evs = append(evs, ev{ir.CleanCell(r[ei]), ir.CleanCell(r[di])})
			}
		}
	}
	if len(evs) == 0 {
		g.Count("boundary events", 0)
		return
	}
	// gather what the child's machines handle and fire
	handled := map[string]bool{}
	var actionCells []string
	for _, mf := range sortedGlobExt(filepath.Join(design, "machines"), ".machine.json") {
		m, err := ir.LoadMachineJSON(mf)
		if err != nil {
			continue // G3 reports the parse error
		}
		for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			so := s.Node.AsObject()
			if so == nil {
				continue
			}
			for _, k := range so.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range so.GetObject("_ignores").Keys() {
				handled[k] = true
			}
			for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
				actionCells = append(actionCells, tr.Actions...)
			}
		}
	}
	for _, f := range sortedGlobExt(filepath.Join(design, "machines"), ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readOrEmpty(f)) {
			for _, r := range tbl.Rows {
				actionCells = append(actionCells, r...)
			}
		}
	}
	fired := strings.Join(actionCells, "\n")
	for _, e := range evs {
		switch e.dir {
		case "consumes":
			if handled[e.name] {
				g.Count("consumed events handled")
			} else {
				g.Errs = append(g.Errs, "boundary event "+ir.Repr(e.name)+" (consumed) is handled or ignored by no machine; a neighbor relies on this subsystem reacting to it")
			}
		case "produces":
			// emission only: an action or matrix row must fire the event.
			// handled[] proves consumption, not emission; accepting it let a
			// child stop emitting by merely declaring an on: handler
			if tokenIn(e.name, fired) {
				g.Count("produced events emitted")
			} else {
				g.Errs = append(g.Errs, "boundary event "+ir.Repr(e.name)+" (produced) appears in no machine action or matrix row; a neighbor relies on this subsystem emitting it")
			}
		}
	}
}
