package stages

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/arch"
	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/formal"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
	"github.com/RamXX/machinery/internal/testpolicy"
)

func init() { Bind("ratchet", Ratchet) }

// baseInventory is what obligations.json at the base holds.
type baseInventory struct {
	Obligations []struct {
		ID string `json:"id"`
	} `json:"obligations"`
	Served []struct {
		ID string `json:"id"`
	} `json:"served"`
}

// Ratchet is stage ratchet (section 11.3): nothing established at the base
// ref may quietly stop being established.
func Ratchet(ctx *Context) *report.Stage {
	s := report.New("ratchet")
	if ctx.Base == "" {
		return s.NotDemonstrated("no base ref; run machinery check --base <ref>")
	}
	p, ref := ctx.P, ctx.Base
	raw, ok := gitx.Show(p.Root, ref, project.ConfigName)
	if !ok {
		// the project did not exist at the base: nothing was established there
		return s.NotApplicable("no " + project.ConfigName + " at " + ref + ": nothing was established there")
	}
	bcfg, err := project.Parse(raw)
	if err != nil {
		s.Errorf("cannot read %s at %s: %v", project.ConfigName, ref, err)
		return s
	}
	bp := &project.Project{Root: p.Root, Cfg: *bcfg}
	d := ctx.Design()

	var inv baseInventory
	if data, ok := gitx.Show(p.Root, ref, strings.TrimSuffix(bcfg.Design.Gen, "/")+"/obligations.json"); ok {
		if err := json.Unmarshal(data, &inv); err != nil {
			s.Errorf("obligations.json at %s: %v", ref, err)
		}
	}
	baseOwed := map[string]bool{}
	if data, ok := gitx.Show(p.Root, ref, bcfg.Design.Owed); ok {
		entries, _ := design.ParseOwed(bcfg.Design.Owed, data)
		for _, e := range entries {
			baseOwed[e.ID] = true
		}
	}

	// proven-to-owed
	nowOwed := d.OwedSet()
	for _, o := range inv.Obligations {
		if !baseOwed[o.ID] {
			if _, owed := nowOwed[o.ID]; owed {
				s.Errorf("ratchet: %s was proven at %s and is owed now", o.ID, ref)
			}
		}
	}

	// scope-narrowed
	nowScope := set(p.Cfg.Scope)
	for _, g := range bcfg.Scope {
		if !nowScope[g] {
			s.Errorf("ratchet: scope narrowed: %s", g)
		}
	}
	baseScope := set(bcfg.Scope)
	widened := false
	for _, g := range p.Cfg.Scope {
		if !baseScope[g] {
			widened = true
		}
	}

	// tier-lowered
	tiered := ratchetTiers(s, d, bp, ref, inv)

	// baseline-grew
	entries := ratchetBaselines(s, p, bp, ref, widened)

	// spec-weakened
	entries += ratchetSpecs(s, p, bp, ref)

	s.Count(len(inv.Obligations), "obligations").Count(len(bcfg.Scope), "scope globs").
		Count(tiered, "tiered ids").Count(entries, "baseline entries")
	return s
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// ratchetTiers reports lowered tiers and returns how many ids had a tier above
// routine at the base.
func ratchetTiers(s *report.Stage, d *design.Design, bp *project.Project, ref string, inv baseInventory) int {
	if bp.Cfg.Design.Risk == "" {
		return 0
	}
	data, ok := gitx.Show(bp.Root, ref, bp.Cfg.Design.Risk)
	if !ok {
		return 0
	}
	baseAreas, _, _ := design.ParseRisk(bp.Cfg.Design.Risk, data)
	var ids []string
	for _, o := range inv.Obligations {
		ids = append(ids, o.ID)
	}
	for _, a := range inv.Served {
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)
	nowByName := map[string]design.Area{}
	for _, a := range d.Areas {
		nowByName[a.Name] = a
	}
	tiered := 0
	for _, id := range ids {
		old, in := design.TierOf(baseAreas, id)
		if old == design.Routine {
			continue
		}
		tiered++
		if !d.Known(id) {
			continue
		}
		cur, _ := d.Tier(id)
		if design.TierRank(cur) >= design.TierRank(old) {
			continue
		}
		reason := ""
		for _, a := range in {
			if n, ok := nowByName[a.Name]; ok && strings.TrimSpace(n.Why) != strings.TrimSpace(a.Why) {
				reason = n.Why
				break
			}
		}
		if reason != "" {
			s.Warnf("ratchet: tier lowered: %s %s -> %s (%s)", id, old, cur, strings.TrimSpace(reason))
		} else {
			s.Errorf("ratchet: tier lowered: %s %s -> %s", id, old, cur)
		}
	}
	return tiered
}

// ratchetBaselines compares the arch and tests baselines with the base's and
// returns the number of entries looked at.
func ratchetBaselines(s *report.Stage, p, bp *project.Project, ref string, widened bool) int {
	n := 0
	for _, pair := range [][2]string{
		{arch.BaselinePath(bp), arch.BaselinePath(p)},
		{testpolicy.BaselinePath(bp), testpolicy.BaselinePath(p)},
	} {
		baseRel, nowRel := pair[0], pair[1]
		had := map[baseline.Entry]bool{}
		if baseRel != "" {
			if data, ok := gitx.Show(p.Root, ref, baseRel); ok {
				es, _ := baseline.Parse(baseRel, data)
				for _, e := range es {
					had[e] = true
				}
			}
		}
		now, _ := baseline.Load(p, nowRel)
		for _, e := range now {
			n++
			if !had[e] && !widened {
				s.Errorf("ratchet: baseline grew: %s %s", e.Rule, e.Subject)
			}
		}
	}
	return n
}

// ratchetSpecs warns about formal acceptances the base did not have and
// returns the number of acceptances looked at.
func ratchetSpecs(s *report.Stage, p, bp *project.Project, ref string) int {
	var raw []byte
	var rerr error = errors.New("missing")
	if data, ok := gitx.Show(p.Root, ref, formal.BaselinePath(bp)); ok {
		raw, rerr = data, nil
	}
	base, err := formal.ParseBaseline(raw, rerr)
	if err != nil {
		s.Errorf("formal baseline at %s: %v", ref, err)
		return 0
	}
	now, err := formal.LoadBaseline(p)
	if err != nil {
		s.Errorf("%v", err)
		return 0
	}
	had := map[formal.Acceptance]bool{}
	for _, sb := range base.Specs {
		for _, a := range sb.Accepted {
			had[a] = true
		}
	}
	for _, a := range base.Removed {
		had[a] = true
	}
	n := 0
	names := make([]string, 0, len(now.Specs))
	for name := range now.Specs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		for _, a := range now.Specs[name].Accepted {
			n++
			if !had[a] {
				s.Warnf("ratchet: weakened since %s: %s: %s (%s)", ref, name, a.Change, a.Reason)
			}
		}
	}
	for _, a := range now.Removed {
		n++
		if had[a] {
			continue
		}
		spec, change := removedSpec(a.Change)
		s.Warnf("ratchet: weakened since %s: %s: %s (%s)", ref, spec, change, a.Reason)
	}
	return n
}

// removedSpec splits "spec <name> removed" into the spec name and "spec removed".
func removedSpec(change string) (spec, what string) {
	if rest, ok := strings.CutPrefix(change, "spec "); ok {
		if name, ok := strings.CutSuffix(rest, " removed"); ok {
			return name, "spec removed"
		}
	}
	return "-", change
}
