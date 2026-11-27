package stages

import (
	"encoding/json"
	"errors"
	"fmt"
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

	// proven-to-owed, obligation-removed, demoted, retired
	nowOwed := d.OwedSet()
	for _, a := range inv.Served {
		ratchetStatus(s, d, a.ID, baseOwed[a.ID])
	}
	for _, o := range inv.Obligations {
		if ratchetStatus(s, d, o.ID, baseOwed[o.ID]) {
			continue
		}
		if !d.Known(o.ID) {
			s.Warnf("ratchet: obligation removed since %s: %s", ref, o.ID)
			continue
		}
		if !baseOwed[o.ID] {
			if _, owed := nowOwed[o.ID]; owed {
				s.Errorf("ratchet: %s was proven at %s and is owed now", o.ID, ref)
			}
		}
	}

	// stage-disabled: a stage that applied at the base still applies
	for _, st := range stageKeys(bcfg) {
		if st.on && !stageOn(&p.Cfg, st.stage) {
			s.Errorf("ratchet: stage disabled: %s applied at %s and %s is gone now", st.stage, ref, st.key)
		}
	}

	// review-off: a review gate the base required is reported when it is off now
	for _, kind := range []string{"backlog", "design"} {
		if bcfg.ReviewOn(kind) && !p.Cfg.ReviewOn(kind) {
			s.Warnf("ratchet: review switched off: review.%s was required at %s and is off now", kind, ref)
		}
	}

	ratchetAdapter(s, bcfg, &p.Cfg)

	// scope-narrowed
	nowScope := set(p.Cfg.Scope)
	for _, g := range bcfg.Scope {
		if !nowScope[g] {
			s.Errorf("ratchet: scope narrowed: %s", g)
		}
	}
	baseScope := set(bcfg.Scope)
	var widened []string // scope globs the base did not have
	for _, g := range p.Cfg.Scope {
		if !baseScope[g] {
			widened = append(widened, g)
		}
	}

	// tier-lowered and vital-weakened
	base := loadBaseRisk(bp, ref)
	var lowered []design.Lowering
	if rel := p.Cfg.Design.Risk; rel != "" {
		if now, err := p.Read(rel); err == nil {
			lowered = design.ParseLowered(now)
		}
	}
	tiered := ratchetTiers(s, d, base, lowered, inv)
	vital := ratchetVital(s, d, base, lowered, inv)

	// baseline-grew
	entries := ratchetBaselines(s, p, bp, ref, widened)

	// spec-weakened
	entries += ratchetSpecs(s, p, bp, ref)

	s.Count(len(inv.Obligations), "obligations").Count(len(bcfg.Scope), "scope globs").
		Count(tiered, "tiered ids").Count(entries, "baseline entries")
	if vital > 0 {
		s.Count(vital, "vital areas")
	}
	return s
}

// ratchetStatus reports an id of the base that has a status now (section
// 4.1): demoted to proposed is an error, retiring what was proven is
// reported. It returns true when the id has a status.
func ratchetStatus(s *report.Stage, d *design.Design, id string, wasOwed bool) bool {
	switch d.Status(id) {
	case design.Proposed:
		s.Errorf("ratchet: demoted to proposed: %s; proposed is for candidates, not a way out of an obligation", id)
	case design.Retired:
		if !wasOwed {
			s.Warnf("ratchet: retired: %s (%s)", id, strings.TrimSpace(d.StatusEntryOf(id).Reason))
		}
	default:
		return false
	}
	return true
}

type stageKey struct {
	stage, key string
	on         bool
}

// stageKeys lists, for the stages whose applicability is a config key, whether
// the key is present in a config.
func stageKeys(c *project.Config) []stageKey {
	return []stageKey{
		{"model", "design.models", len(c.Design.Models) > 0},
		{"machines", machinesKey(c), c.HasMachines()},
		{"failure-modes", failureModesKey(c), c.HasFailureModes()},
		{"risk", "design.risk", c.Design.Risk != ""},
		{"xref-contract", "contracts", len(c.Contracts) > 0},
		{"xstate", "tools.xstate", c.Tools.XState},
		{"checkers", "checkers", len(c.Checkers) > 0 || c.Design.Facts != ""},
		{"arch", "constraints", c.Constraints != nil},
		{"tests", "tests or trace.tests", c.Tests != nil || (c.Trace != nil && len(c.Trace.Tests) > 0)},
		{"trace", "trace", c.Trace != nil},
	}
}

// machinesKey names the key that makes stage machines apply.
func machinesKey(c *project.Config) string {
	if c.ImportedMachines() != nil {
		return "import.machines"
	}
	return "design.machines"
}

// failureModesKey names the key that makes stage failure-modes apply.
func failureModesKey(c *project.Config) string {
	if c.Design.FailureModes == "" && c.ImportedFailureModes() != nil {
		return "import.failure_modes"
	}
	return "design.failure_modes"
}

func stageOn(c *project.Config, stage string) bool {
	for _, st := range stageKeys(c) {
		if st.stage == stage {
			return st.on
		}
	}
	return false
}

// subjectFile returns the file a baseline subject names ("file", "file:
// marker", "file -> target", "file:Func").
func subjectFile(subject string) string {
	if i := strings.Index(subject, " -> "); i >= 0 {
		subject = subject[:i]
	}
	if i := strings.Index(subject, ":"); i >= 0 {
		subject = subject[:i]
	}
	return strings.TrimSpace(subject)
}

func set(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// baseRisk is the risk file at the base: its areas and its dismissals.
type baseRisk struct {
	areas    []design.Area
	notVital []design.Dismissal
}

// loadBaseRisk reads the risk file of the base; empty when it had none.
func loadBaseRisk(bp *project.Project, ref string) baseRisk {
	if bp.Cfg.Design.Risk == "" {
		return baseRisk{}
	}
	data, ok := gitx.Show(bp.Root, ref, bp.Cfg.Design.Risk)
	if !ok {
		return baseRisk{}
	}
	areas, _, _ := design.ParseRisk(bp.Cfg.Design.Risk, data)
	return baseRisk{areas: areas, notVital: design.ParseNotVital(data)}
}

// baseIDs lists the obligations and actions of the base, sorted.
func baseIDs(inv baseInventory) []string {
	var ids []string
	for _, o := range inv.Obligations {
		ids = append(ids, o.ID)
	}
	for _, a := range inv.Served {
		ids = append(ids, a.ID)
	}
	sort.Strings(ids)
	return ids
}

// ratchetTiers reports lowered tiers and returns how many ids had a tier above
// routine at the base.
func ratchetTiers(s *report.Stage, d *design.Design, base baseRisk, lowered []design.Lowering, inv baseInventory) int {
	if len(base.areas) == 0 {
		return 0
	}
	tiered := 0
	for _, id := range baseIDs(inv) {
		old, in := design.TierOf(base.areas, id)
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
		for _, l := range lowered {
			if l.From == old && design.MatchAny(l.IDs, id) {
				reason = l.Reason
				break
			}
		}
		_ = in
		if reason != "" {
			s.Warnf("ratchet: tier lowered: %s %s -> %s (%s)", id, old, cur, strings.TrimSpace(reason))
		} else {
			s.Errorf("ratchet: tier lowered: %s %s -> %s; record it under lowered: in the risk file with a reason", id, old, cur)
		}
	}
	return tiered
}

// ratchetBaselines compares the arch and tests baselines with the base's and
// returns the number of entries looked at.
func ratchetBaselines(s *report.Stage, p, bp *project.Project, ref string, widened []string) int {
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
			if had[e] {
				continue
			}
			// a new entry is allowed only for a file that is newly in scope
			file := subjectFile(e.Subject)
			if project.Match(widened, file) && !project.Match(bp.Cfg.Scope, file) {
				continue
			}
			s.Errorf("ratchet: baseline grew: %s %s", e.Rule, e.Subject)
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

// ratchetAdapter is rule adapter-widened (section 11.3): the policy
// evidence.adapter lets adapter evidence prove more than at the base. The
// policies are compared by what they admit: for each tier and kind, the least
// granularity an adapter pass needs, or never. Ids of a critical or vital area
// that become provable are an error; every other widening is named.
func ratchetAdapter(s *report.Stage, base, now *project.Config) {
	bNeed, nNeed := admits(base), admits(now)
	changes := [][2]string{{never, project.GranularityFunction}, {never, project.GranularityFile}, {project.GranularityFunction, project.GranularityFile}}
	for _, tier := range []string{design.Routine, design.Consequential, design.Critical, design.Vital} {
		for _, c := range changes {
			var kinds []string
			for _, k := range project.AdapterKinds {
				if bNeed[[2]string{tier, k}] == c[0] && nNeed[[2]string{tier, k}] == c[1] {
					kinds = append(kinds, k)
				}
			}
			if len(kinds) == 0 {
				continue
			}
			what := fmt.Sprintf("ratchet: adapter widened: %s %s ids: %s -> %s", tier, strings.Join(kinds, ", "), c[0], c[1])
			if c[0] == never && (tier == design.Critical || tier == design.Vital) {
				s.Errorf("%s; an id of a critical or vital area needs a receipt its test emits after its own assertions", what)
			} else {
				s.Warnf("%s", what)
			}
		}
	}
	// sources, as pairs of a package glob and a files glob
	had := map[[2]string]bool{}
	for _, e := range base.AdapterSources() {
		for _, f := range e.Maps() {
			had[[2]string{e.Package, f}] = true
		}
	}
	var pkgs []string
	added := map[string][]string{}
	for _, e := range now.AdapterSources() {
		for _, f := range e.Maps() {
			key := [2]string{e.Package, f}
			if had[key] {
				continue
			}
			had[key] = true
			if added[e.Package] == nil {
				pkgs = append(pkgs, e.Package)
			}
			added[e.Package] = append(added[e.Package], f)
		}
	}
	for _, p := range pkgs {
		s.Warnf("ratchet: adapter widened: evidence.adapter.sources: %s <- %s added", p, strings.Join(added[p], ", "))
	}
}

const never = "never"

// admits returns, for each tier and kind an adapter pass may prove, the
// least granularity it needs: function, file or never (section 10.3).
func admits(c *project.Config) map[[2]string]string {
	kinds, tiers, min := c.AdapterPolicy()
	in := func(list []string, s string) bool {
		for _, v := range list {
			if v == s {
				return true
			}
		}
		return false
	}
	res := map[[2]string]string{}
	for _, tier := range []string{design.Routine, design.Consequential, design.Critical, design.Vital} {
		for _, k := range project.AdapterKinds {
			need := never
			if in(kinds, k) && in(tiers, tier) {
				need = project.GranularityFile
				if min.Default == project.GranularityFunction || min.By[k] == project.GranularityFunction || min.By[tier] == project.GranularityFunction {
					need = project.GranularityFunction
				}
			}
			res[[2]string{tier, k}] = need
		}
	}
	return res
}
