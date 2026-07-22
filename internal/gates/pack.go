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
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/pack"
	"github.com/RamXX/machinery/internal/version"
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
	// stale pack directories: a packs/<id>.pack with no current subsystem
	// (after a rename or removal) is a dead contract a child keeps trusting;
	// the freshness loop above only walks CURRENT subsystems, so the orphan
	// must be sought explicitly
	if entries, rerr := os.ReadDir(filepath.Join(design, "packs")); rerr == nil {
		current := map[string]bool{}
		for _, id := range ids {
			current[id+".pack"] = true
		}
		for _, e := range entries {
			if e.IsDir() && !current[e.Name()] {
				g.Errs = append(g.Errs, "packs/"+e.Name()+" corresponds to no current subsystem (stale after a rename or removal); its child keeps validating against a dead contract, remove the directory and re-issue the current pack")
			}
		}
	}
	// per-pack boundary-event visibility: a zero must be seen in every gate
	// run, not discovered after empty packs ship (the H2 dogfood finding).
	// Strict generation makes an unwaived zero unreachable here, but the
	// counts stay printed so the evidence is in the output, not implied.
	var evCounts []string
	for _, id := range ids {
		evCounts = append(evCounts, fmt.Sprintf("%s %d", id, pack.CountBoundaryEvents(fresh[id]["events.md"])))
	}
	if len(evCounts) > 0 {
		g.CheckedExtra("boundary events: " + strings.Join(evCounts, ", "))
	}
	// the amendment counter, visible in every run
	g.CheckedExtra(fmt.Sprintf("pack revision %d", d.Revision))
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

	// pack_revision is the amendment-visibility counter (parent rev N -> N+1
	// must be seen at the child); verify it and keep it in the output
	if rev, ok := packRevisionOf(manifest); ok {
		g.CheckedExtra("pack revision " + rev)
	} else {
		g.Errs = append(g.Errs, "the pack manifest declares no pack_revision (or it is not a positive integer); the pack predates the amendment counter or was edited, regenerate it at the parent and re-copy")
	}

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
			} else if g.recordStamp(string(got)); version.Strip(string(got)) != version.Strip(freshRef[n]) {
				// stamp lines stripped from both sides (P-F10): a version-only
				// skew is a note, never DRIFT
				g.Drift = append(g.Drift, "committed formal/"+n+" is stale (differs from a fresh generation); rerun machinery pack refine")
			} else {
				g.Count("refinement artifacts fresh")
			}
		}
		// two-way diff: a *PackRefinement.* artifact a fresh generation does
		// not produce is a stale binding from a previous packmap; the one-way
		// diff above never looked, so a rebind left the old "proof" committed
		// and green while nothing checked it anymore
		if entries, rerr := os.ReadDir(filepath.Join(design, "formal")); rerr == nil {
			for _, e := range entries {
				n := e.Name()
				if e.IsDir() || (!strings.HasSuffix(n, "PackRefinement.tla") && !strings.HasSuffix(n, "PackRefinement.cfg")) {
					continue
				}
				if _, ok := freshRef[n]; !ok {
					g.Errs = append(g.Errs, "committed formal/"+n+" is a refinement artifact a fresh generation does not produce (a stale binding from a previous packmap); remove it")
				}
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
		childEnt := childEntities.Get2(en).AsObject()
		// frozen attributes must exist in the child by name (the child may
		// add attributes, never drop these: boundary event payloads name them)
		childAttrs := map[string]bool{}
		for _, a := range childEnt.Get2("attributes").AsArray() {
			if n := a.AsObject().GetString("name"); n != "" {
				childAttrs[n] = true
			}
		}
		for _, a := range entities.Get2(en).AsObject().Get2("attributes").AsArray() {
			ao := a.AsObject()
			name := ao.GetString("name")
			if name != "" {
				if childAttrs[name] {
					g.Count("owned attributes present")
				} else {
					g.Errs = append(g.Errs, "owned entity "+en+" attribute "+name+" is missing from the child domain model; the pack's frozen attributes can be extended, never dropped (boundary event payloads name them)")
				}
			}
			// lifecycle enum values must match exactly
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
		// frozen entity invariants: same id present, statement kept. The
		// child may EXTEND the statement with enforcement detail (append),
		// never rewrite or weaken it.
		childInvs := map[string]string{}
		for _, iv := range childEnt.Get2("invariants").AsArray() {
			if io := iv.AsObject(); io != nil {
				childInvs[io.GetString("id")] = io.GetString("definition")
			}
		}
		for _, iv := range entities.Get2(en).AsObject().Get2("invariants").AsArray() {
			io := iv.AsObject()
			if io == nil {
				continue
			}
			iid := io.GetString("id")
			if iid == "" {
				continue
			}
			got, ok := childInvs[iid]
			switch {
			case !ok:
				g.Errs = append(g.Errs, "owned entity "+en+" invariant "+iid+" is missing from the child domain model; the pack's entity invariants are frozen")
			case !statementKept(io.GetString("definition"), got):
				g.Errs = append(g.Errs, "owned entity "+en+" invariant "+iid+" drifted from the pack: pack says "+ir.Repr(io.GetString("definition"))+" but the child says "+ir.Repr(got)+"; the frozen statement may gain appended enforcement detail, never be rewritten")
			default:
				g.Count("owned invariants intact")
			}
		}
	}
}

// statementKept reports whether the child's invariant definition preserves
// the pack's, comparing whitespace-normalized text. Equality passes; so does
// the child APPENDING detail after the pack's full statement (the shipped
// examples add "Structural: ..." enforcement notes); anything else is drift.
func statementKept(pack, child string) bool {
	p := strings.Join(strings.Fields(pack), " ")
	c := strings.Join(strings.Fields(child), " ")
	return c == p || strings.HasPrefix(c, p+" ")
}

// packRevisionOf reads the manifest's pack_revision as a positive integer,
// returning its decimal text and whether it is valid.
func packRevisionOf(manifest *ir.Object) (string, bool) {
	rev := manifest.Get2("pack_revision")
	if rev == nil || rev.Kind != ir.KindNumber {
		return "", false
	}
	n, err := strconv.Atoi(string(rev.AsNumber()))
	if err != nil || n < 1 {
		return "", false
	}
	return string(rev.AsNumber()), true
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
	for i, v := range manifest.Get2("delegated_invariants").AsArray() {
		if v != nil && v.Kind == ir.KindString {
			ids = append(ids, v.AsString())
			continue
		}
		// a non-string entry (an unquoted colon-bearing id reparsed as a
		// mapping) is an obligation the child would silently shed; nothing is
		// ever dropped from the delegation list
		g.Errs = append(g.Errs, fmt.Sprintf("pack manifest delegated_invariants entry %d is not a plain string (%s); a colon-bearing id must be quoted when the pack is generated, regenerate it at the parent", i+1, ir.Repr(v)))
	}
	if len(ids) == 0 {
		g.Count("delegated invariants", 0)
		return
	}
	var cells []string
	for _, f := range sortedGlobExt(filepath.Join(design, "machines"), ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readFileOrErr(f, g)) {
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(design, "BUILD.md")); err == nil {
		for _, tbl := range ir.ParseMdTables(readFileOrErr(filepath.Join(design, "BUILD.md"), g)) {
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
			// every action position counts as emission evidence: entry and
			// exit actions and invoke actors emit too, not just transitions
			for a := range ir.ActionsOf(s.Node, nil, s.Path) {
				actionCells = append(actionCells, a)
			}
			for _, inv := range ir.InvokesOf(s.Node) {
				if io := inv.AsObject(); io != nil {
					if src := io.GetString("src"); src != "" {
						actionCells = append(actionCells, src)
					}
				}
			}
		}
	}
	for _, f := range sortedGlobExt(filepath.Join(design, "machines"), ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readFileOrErr(f, g)) {
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
				g.Errs = append(g.Errs, "boundary event "+ir.Repr(e.name)+" (produced) appears in no machine action (entry, exit, transition, invoke src) and no matrix cell; a neighbor relies on this subsystem emitting it. If the emitting action is named differently, name the event whole-token in that action's matrix row")
			}
		}
	}
}
