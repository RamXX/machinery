// AssuranceInventory: the authoritative obligation inventory of
// docs/test-assurance-contract.md section 3, derived from a held immutable
// design snapshot. The inventory includes every root BUILD milestone, every
// committed oracle row in scope (machines/*.oracle.md plus the relational
// formal oracles when present), machine-owned guard-clause IDs from
// CLAUSES declarations, invariant IDs from the modelith model, and the
// declared runtime obligations of the design's assurance plan. Existing
// generators remain authoritative for their own stable IDs; keys are the
// qualified tuple {design, kind, owner, id} so obligations never pool
// merely because a short id or name matches. No process is launched.
package gates

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/tdd"
	"github.com/RamXX/machinery/internal/tdd/protocol"
)

// readPlanDeclaration reads the design's plan.json leniently: the inventory
// needs the declared design identity and runtime obligations even when a
// review has gone stale (that staleness is LoadPlan/Validate's error, never
// a reason to shrink the authoritative requirements). Hard corruption and
// an unparsable declaration still fail closed.
func readPlanDeclaration(design string) (designID string, runtime []tdd.InventoryObligation, err error) {
	planPath := filepath.Join(design, protocol.ControlDirName, protocol.PlanFileName)
	data, rerr := readDesignFile(design, planPath)
	if rerr != nil {
		if errors.Is(rerr, fs.ErrNotExist) {
			// no declaration yet: the only remaining canonical identity
			// spelling is the repository root itself
			return protocol.RepositoryRoot, nil, nil
		}
		return "", nil, fmt.Errorf("cannot read assurance declaration: %w", rerr)
	}
	// the exact control namespace is enforced even for lenient inventory
	// reads: pollution of assurance/ is corruption, not a default
	if nerr := tdd.ValidateControlNamespace(design); nerr != nil {
		return "", nil, nerr
	}
	v, perr := ir.LoadMachineJSONBytes(planPath, data)
	if perr != nil {
		return "", nil, fmt.Errorf("INVALID_SCHEMA: %s: %w", planPath, perr)
	}
	root := v.AsObject()
	if root == nil {
		return "", nil, fmt.Errorf("INVALID_SCHEMA: %s is not a JSON object", planPath)
	}
	dv := root.Get2("design")
	if dv == nil || dv.Kind != ir.KindString || dv.AsString() == "" {
		return "", nil, fmt.Errorf("INVALID_SCHEMA: %s carries no string design root", planPath)
	}
	designID = dv.AsString()
	roVals := root.Get2("runtime_obligations")
	if roVals == nil || roVals.IsNull() {
		return designID, nil, nil
	}
	if roVals.Kind != ir.KindArray {
		return "", nil, fmt.Errorf("INVALID_SCHEMA: %s runtime_obligations must be an array", planPath)
	}
	for i, item := range roVals.AsArray() {
		io := item.AsObject()
		if io == nil {
			return "", nil, fmt.Errorf("INVALID_SCHEMA: %s runtime_obligations[%d] is not an object", planPath, i)
		}
		id, owner := io.GetString("id"), io.GetString("owner")
		if id == "" || owner == "" {
			return "", nil, fmt.Errorf("INVALID_SCHEMA: %s runtime_obligations[%d] needs a nonempty id and owner", planPath, i)
		}
		runtime = append(runtime, tdd.InventoryObligation{
			Key: tdd.ObligationKey{Design: designID, Kind: protocol.KindRuntime, Owner: owner, ID: id},
		})
	}
	return designID, runtime, nil
}

// AssuranceInventory derives the authoritative obligation inventory over a
// held immutable design snapshot (gates.AcquireSnapshot's DesignPath). It
// fails closed on unreadable or inconsistent sources, mirroring the
// existing gates' rules: every machine needs its committed oracle once any
// oracle exists (and machines without any oracle at all block), CLAUSES
// declarations must be well formed, and a present modelith model must
// declare invariants.
func AssuranceInventory(design string) (tdd.Inventory, error) {
	g := NewGate("assurance-inventory")
	designID, runtimeObls, err := readPlanDeclaration(design)
	if err != nil {
		return tdd.Inventory{}, err
	}
	var obligations []tdd.InventoryObligation

	// current root BUILD milestones
	var milestones []tdd.MilestoneKey
	if HasBuildDoc(design) {
		text := readDesignFileOrErr(design, filepath.Join(design, "BUILD.md"), g)
		if len(g.Errs) > 0 {
			return tdd.Inventory{}, errors.New(strings.Join(g.Errs, "; "))
		}
		for _, m := range mustPlanMilestones(text) {
			if !m.numOK {
				return tdd.Inventory{}, fmt.Errorf("INVALID_SCHEMA: BUILD.md milestone %q has a non-numeric id", m.numRaw)
			}
			milestones = append(milestones, tdd.MilestoneKey{Design: designID, Milestone: "M" + strconv.Itoa(m.num)})
		}
	}

	// committed oracle rows: machines/*.oracle.md stable ids owned by their
	// canonical design-relative path
	machineFiles, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.machine.json", "machine source")
	machineStems := map[string]bool{}
	for _, path := range machineFiles {
		machineStems[strings.TrimSuffix(filepath.Base(path), ".machine.json")] = true
	}
	oraclePaths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.oracle.md", "committed oracle")
	oracleStems := map[string]bool{}
	for _, path := range oraclePaths {
		oracleStems[strings.TrimSuffix(filepath.Base(path), ".oracle.md")] = true
	}
	if len(oraclePaths) == 0 && len(machineFiles) > 0 {
		return tdd.Inventory{}, fmt.Errorf("MISSING_CONTRACT: %d machine(s) under %s but no committed *.oracle.md; the authoritative inventory cannot shrink by deleting oracles", len(machineFiles), filepath.Join(design, "machines"))
	}
	for _, path := range oraclePaths {
		stem := strings.TrimSuffix(filepath.Base(path), ".oracle.md")
		if !machineStems[stem] {
			return tdd.Inventory{}, fmt.Errorf("MISSING_CONTRACT: orphan oracle %s has no corresponding %s.machine.json; it cannot be dropped from the authoritative inventory silently", filepath.Base(path), stem)
		}
	}
	// once any oracle exists, every machine needs its own committed oracle:
	// deleting an oracle must not shrink the authoritative inventory
	for _, path := range machineFiles {
		stem := strings.TrimSuffix(filepath.Base(path), ".machine.json")
		if !oracleStems[stem] {
			return tdd.Inventory{}, fmt.Errorf("MISSING_CONTRACT: machine %s has no committed oracle (%s.oracle.md); the authoritative inventory cannot shrink by deleting an oracle", filepath.Base(path), stem)
		}
	}
	for _, path := range oraclePaths {
		rel := filepath.ToSlash(strings.TrimPrefix(strings.TrimPrefix(path, design), "/"))
		_, stableIDs := oracleTableIDs(readDesignFileOrErr(design, path, g))
		for _, id := range stableIDs {
			obligations = append(obligations, tdd.InventoryObligation{
				Key: tdd.ObligationKey{Design: designID, Kind: protocol.KindOracleRow, Owner: rel, ID: id},
			})
		}
	}
	if len(g.Errs) > 0 {
		return tdd.Inventory{}, errors.New(strings.Join(g.Errs, "; "))
	}
	// relational formal oracles when opted in (Gp/Gn own their health)
	for _, name := range formalOracleNames {
		path := filepath.Join(design, "formal", name)
		if body, err := readDesignFile(design, path); err != nil || len(body) == 0 {
			continue
		}
		_, stableIDs := oracleTableIDs(readDesignFileOrErr(design, path, g))
		for _, id := range stableIDs {
			obligations = append(obligations, tdd.InventoryObligation{
				Key: tdd.ObligationKey{Design: designID, Kind: protocol.KindOracleRow, Owner: "formal/" + name, ID: id},
			})
		}
	}

	// machine-owned guard-clause obligations from CLAUSES declarations:
	// owner is the declaring machine, id is guard ":" clause
	decls := collectClauseDecls(g, design)
	if len(g.Errs) > 0 {
		return tdd.Inventory{}, errors.New(strings.Join(g.Errs, "; "))
	}
	for _, d := range decls {
		for _, clause := range d.active {
			obligations = append(obligations, tdd.InventoryObligation{
				Key: tdd.ObligationKey{Design: designID, Kind: protocol.KindGuardClause, Owner: d.owner, ID: d.guard + ":" + clause},
			})
		}
	}

	// invariant obligations from the modelith model: owner is the declaring
	// entity ("model" for model-level invariants), one global id space. A
	// present model must declare at least one invariant (the Gc rule); an
	// absent model contributes none.
	invOwners, invErr := modelithInvariantOwners(design)
	if invErr != nil {
		return tdd.Inventory{}, invErr
	}
	for _, owner := range sortedStringKeys(invOwners) {
		for _, id := range invOwners[owner] {
			obligations = append(obligations, tdd.InventoryObligation{
				Key: tdd.ObligationKey{Design: designID, Kind: protocol.KindInvariant, Owner: owner, ID: id},
			})
		}
	}

	// declared runtime obligations from the plan declaration
	obligations = append(obligations, runtimeObls...)

	sort.Slice(obligations, func(i, j int) bool {
		a, b := obligations[i].Key, obligations[j].Key
		if a.Design != b.Design {
			return a.Design < b.Design
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Owner != b.Owner {
			return a.Owner < b.Owner
		}
		return a.ID < b.ID
	})
	sort.Slice(milestones, func(i, j int) bool {
		if milestones[i].Design != milestones[j].Design {
			return milestones[i].Design < milestones[j].Design
		}
		return milestones[i].Milestone < milestones[j].Milestone
	})
	h := sha256.New()
	h.Write([]byte(protocol.InventoryDomain))
	writeField := func(s string) {
		fmt.Fprintf(h, "%d:%s", len(s), s)
	}
	for _, o := range obligations {
		writeField(o.Key.Design)
		writeField(o.Key.Kind)
		writeField(o.Key.Owner)
		writeField(o.Key.ID)
	}
	for _, m := range milestones {
		writeField(m.Design)
		writeField(m.Milestone)
	}
	return tdd.Inventory{
		Obligations: obligations,
		Milestones:  milestones,
		Digest:      fmt.Sprintf("sha256:%x", h.Sum(nil)),
	}, nil
}

func sortedStringKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// mustPlanMilestones discards the has-plan flag: the inventory lists every
// milestone block the Build plan section declares, and a waived section
// (N/A) declares none.
func mustPlanMilestones(text string) []planMilestone {
	ms, _ := planMilestonesOf(text)
	return ms
}

// modelithInvariantOwners reads the model's invariants with their declaring
// entity owners, using the same loadModelith authority as Gc. An absent
// model contributes none (Gc owns reporting that for real designs); a
// present but unreadable or empty model fails closed, as does a present
// model declaring no invariants at all.
func modelithInvariantOwners(design string) (map[string][]string, error) {
	paths, invalid, err := regularGlob(design, "*.modelith.yaml")
	if err != nil {
		return nil, fmt.Errorf("cannot enumerate modelith models: %w", err)
	}
	if len(invalid) > 0 {
		return nil, fmt.Errorf("%s: modelith source must be a regular file inside the design; symlinks and special entries are rejected", filepath.Base(invalid[0]))
	}
	if len(paths) == 0 {
		return nil, nil
	}
	if len(paths) > 1 {
		return nil, fmt.Errorf("multiple modelith models in %s", design)
	}
	g := NewGate("assurance-inventory-modelith")
	dm := loadModelith(design, g)
	if dm == nil || len(g.Errs) > 0 {
		return nil, errors.New(strings.Join(g.Errs, "; "))
	}
	dmo := dm.AsObject()
	if dmo == nil {
		return nil, fmt.Errorf("modelith model in %s is not a mapping", design)
	}
	out := map[string][]string{}
	record := func(owner string, iv *ir.Value) {
		if io := iv.AsObject(); io != nil {
			if id := io.GetString("id"); id != "" {
				out[owner] = append(out[owner], id)
			}
		}
	}
	for _, i := range objSlice(dmo.Get2("invariants")) {
		record("model", i)
	}
	if entities := dmo.GetObject("entities"); entities != nil {
		for _, ename := range entities.Keys() {
			e := entities.Get2(ename).AsObject()
			if e == nil {
				continue
			}
			for _, i := range objSlice(e.Get2("invariants")) {
				record(ename, i)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("MISSING_CONTRACT: the modelith model in %s declares no invariants; the authoritative inventory cannot be empty for a declared model", design)
	}
	return out, nil
}
