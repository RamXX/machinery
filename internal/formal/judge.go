package formal

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/report"
)

// Finding is a formal-stage finding, tagged with the spec or area it is about.
type Finding struct {
	Sev  report.Severity
	Spec string // "" for an area-level finding
	Area string // set for coverage findings
	Msg  string
}

// Judgement is the pure verdict of stage formal (section 16.9).
type Judgement struct {
	Applicable  bool
	Specs       []*Spec
	Findings    []Finding
	ND          string
	Runs        int
	Reach       int
	Mutations   int
	Assumptions int
	Unproven    []UnprovenAssumption
	Proven      map[string]bool // fv id -> proven
	Results     map[string]*Result
}

// UnprovenAssumption is an assumption nobody has shown to hold.
type UnprovenAssumption struct {
	Spec string
	ID   string
	Text string
}

// Judge reads manifests, results, the baseline, the risk file and git
// history. It runs no tool.
func Judge(d *design.Design) (*Judgement, error) {
	p := d.P
	j := &Judgement{Proven: map[string]bool{}, Results: map[string]*Result{}}
	specs, err := Load(p)
	if err != nil {
		return nil, err
	}
	j.Specs = specs
	var critical []design.Area
	areaByName := map[string]design.Area{}
	for _, a := range d.Areas {
		areaByName[a.Name] = a
		if a.Tier == design.Critical {
			critical = append(critical, a)
		}
	}
	if len(specs) == 0 && len(critical) == 0 {
		return j, nil
	}
	j.Applicable = true
	owed := d.OwedSet()
	base, berr := LoadBaseline(p)
	if berr != nil {
		j.Findings = append(j.Findings, Finding{Sev: report.Error, Msg: berr.Error()})
		base = &Baseline{Specs: map[string]*SpecBaseline{}}
	}
	add := func(sev report.Severity, spec, format string, a ...any) {
		j.Findings = append(j.Findings, Finding{Sev: sev, Spec: spec, Msg: fmt.Sprintf(format, a...)})
	}
	noResult := map[string]bool{}
	live := map[string]bool{}
	for _, s := range specs {
		live[s.Spec] = true
		for _, prob := range s.Problems {
			add(report.Error, s.Spec, "%s", prob)
		}
		if len(s.Problems) > 0 {
			continue
		}
		for _, an := range s.Areas {
			if _, ok := areaByName[an]; !ok {
				add(report.Error, s.Spec, "%s: unknown area %s", s.Spec, an)
			}
		}
		// skeleton drift and property file
		want, serr := Skeleton(d, s)
		if serr != nil {
			add(report.Error, s.Spec, "%v", serr)
			continue
		}
		skeleton, rerr := p.Read(s.SkeletonPath())
		switch {
		case rerr != nil:
			add(report.Drift, s.Spec, "%s: missing; run machinery formal gen", s.SkeletonPath())
		case !bytes.Equal(skeleton, want):
			add(report.Drift, s.Spec, "%s: stale; run machinery formal gen", s.SkeletonPath())
		}
		props, perr := p.Read(s.PropsPath())
		if perr != nil {
			add(report.Error, s.Spec, "%s: property file is missing", s.PropsPath())
			continue
		}
		if s.Tool == "alloy" && HasCommands(props) {
			add(report.Error, s.Spec, "%s: commands belong in the manifest runs, not in the property file", s.PropsPath())
		}
		cur, missing := s.Current(p)
		for _, name := range missing {
			add(report.Error, s.Spec, "%s: property %s is not defined in %s", s.Spec, name, s.PropsPath())
		}
		// results
		res, ok := LoadResult(p, s)
		j.Runs += len(s.Runs)
		j.Mutations += len(s.Mutations)
		j.Assumptions += len(s.Assumptions)
		for _, r := range s.Runs {
			j.Reach += len(r.Reach)
		}
		if !ok {
			noResult[s.Spec] = true
			if j.ND == "" {
				j.ND = fmt.Sprintf("no result for %s; run machinery formal check", s.Spec)
			}
		} else {
			j.Results[s.Spec] = res
			if res.InputSHA256 != s.InputHash(skeleton, props) {
				add(report.Drift, s.Spec, "%s: stale; run machinery formal check", s.ResultPath())
			}
			judgeRuns(s, res, add)
			judgeMutations(d, s, res, add)
		}
		// baseline
		if rec, ok := base.Specs[s.Spec]; !ok {
			add(report.Drift, s.Spec, "baseline: %s: spec not recorded; run machinery formal baseline", s.Spec)
		} else {
			adds, weak := Diff(rec, cur)
			for _, a := range adds {
				add(report.Drift, s.Spec, "baseline: %s: %s not recorded; run machinery formal baseline", s.Spec, a)
			}
			for _, w := range weak {
				add(report.Error, s.Spec, "weakened: %s: %s; accept with machinery formal baseline --accept --reason", s.Spec, w)
			}
		}
		// sync
		if story, pending := s.PendingBug(); pending {
			add(report.Error, s.Spec, "sync: %s: possible bug under investigation (%s)", s.Spec, story)
		} else if commits, problem := Drift(p, s); problem != "" {
			add(report.Error, s.Spec, "sync: %s: %s", s.Spec, problem)
		} else if len(commits) > 0 {
			add(report.Drift, s.Spec, "sync: %s: %d commit(s) touched its sources since %s; classify with machinery formal sync %s",
				s.Spec, len(commits), gitx.Short(s.Sources.ReadAt), s.Spec)
		}
		// assumptions
		for _, a := range s.Assumptions {
			unproven := a.Status != "proven"
			if a.Status == "proven" {
				if !d.Known(a.Evidence) || d.IsAct(a.Evidence) {
					add(report.Error, s.Spec, "%s: assumption %s: evidence %s is not an obligation", s.Spec, a.ID, a.Evidence)
					unproven = true
				} else if _, isOwed := owed[a.Evidence]; isOwed {
					unproven = true
				}
			}
			if unproven {
				j.Unproven = append(j.Unproven, UnprovenAssumption{s.Spec, a.ID, a.Text})
				add(report.Warn, s.Spec, "%s: assumption %s unproven: %s", s.Spec, a.ID, a.Text)
			}
		}
	}
	for name := range base.Specs {
		if !live[name] {
			add(report.Error, name, "weakened: %s: spec removed; accept with machinery formal baseline --accept --reason", name)
		}
	}
	// coverage (16.1)
	covered := map[string]bool{}
	for _, a := range critical {
		covered[a.Name] = true
		for _, q := range a.Questions {
			tool := design.Questions[q]
			found := false
			for _, s := range specs {
				if s.Tool == tool && contains(s.Areas, a.Name) && contains(s.Questions, q) {
					found = true
				}
			}
			if !found {
				covered[a.Name] = false
				j.Findings = append(j.Findings, Finding{Sev: report.Error, Area: a.Name,
					Msg: fmt.Sprintf("fv:%s: no %s spec answers %s", a.Name, tool, q)})
			}
		}
	}
	// owed demotion: a spec all of whose areas are owed does not block
	specOwed := map[string]bool{}
	for _, s := range specs {
		all := len(s.Areas) > 0
		for _, an := range s.Areas {
			if _, ok := owed["fv:"+an]; !ok {
				all = false
			}
		}
		specOwed[s.Spec] = all
	}
	blocking := map[string]bool{} // spec -> has ERROR or DRIFT
	for i := range j.Findings {
		f := &j.Findings[i]
		if f.Sev == report.Warn {
			continue
		}
		demote := false
		if f.Area != "" {
			_, demote = owed["fv:"+f.Area]
		} else if f.Spec != "" {
			demote = specOwed[f.Spec]
			blocking[f.Spec] = true
		}
		if demote {
			f.Sev = report.Warn
			f.Msg = "owed: " + f.Msg
		}
	}
	if j.ND != "" {
		allOwed := true
		for name := range noResult {
			if !specOwed[name] {
				allOwed = false
			}
		}
		if allOwed {
			j.ND = ""
		}
	}
	for _, a := range critical {
		ok := covered[a.Name]
		for _, s := range specs {
			if contains(s.Areas, a.Name) && (blocking[s.Spec] || noResult[s.Spec] || len(s.Problems) > 0) {
				ok = false
			}
		}
		j.Proven["fv:"+a.Name] = ok
	}
	sort.SliceStable(j.Findings, func(a, b int) bool { return j.Findings[a].Msg < j.Findings[b].Msg })
	return j, nil
}

type adder func(sev report.Severity, spec, format string, a ...any)

func judgeRuns(s *Spec, res *Result, add adder) {
	for _, r := range s.Runs {
		rr := res.run(r.Name)
		if rr == nil {
			continue // the stale-result finding covers it
		}
		cx := "cx:" + s.Spec + ":" + r.Name
		switch {
		case rr.Outcome == "error":
			add(report.Error, s.Spec, "%s run %s: the model checker failed: %s", s.Spec, r.Name, rr.Violated)
		case r.Expect == "pass" && rr.Outcome == "violated":
			add(report.Error, s.Spec, "%s run %s: %s violated; replay %s against the real system", s.Spec, r.Name, rr.Violated, cx)
		case r.Expect == "finding" && rr.Outcome == "passed":
			add(report.Error, s.Spec, "%s run %s: expected a finding and the run passed; the bug is gone or the model changed: set expect: pass", s.Spec, r.Name)
		case r.Expect == "finding":
			why := "story " + r.Story
			if r.Assumption != "" {
				why = "assumption " + r.Assumption
			}
			add(report.Warn, s.Spec, "finding: %s run %s: %s violated (%s)", s.Spec, r.Name, rr.Violated, why)
		}
		if r.Expect == "pass" {
			for _, name := range r.Reach {
				if !rr.Reach[name] {
					add(report.Error, s.Spec, "%s run %s: reach %s not reached: the pass is vacuous", s.Spec, r.Name, name)
				}
			}
		}
	}
}

func judgeMutations(d *design.Design, s *Spec, res *Result, add adder) {
	caught := map[string]bool{}
	for _, m := range s.Mutations {
		caught[m.Detects] = true
		if m.Origin != "" && !gitx.Exists(d.P.Root, m.Origin) {
			add(report.Error, s.Spec, "%s mutation %s: origin %s is not a commit", s.Spec, m.ID, m.Origin)
		}
		mr := res.mutation(m.ID)
		if mr == nil {
			continue
		}
		switch mr.Outcome {
		case "missed":
			add(report.Error, s.Spec, "%s mutation %s missed: %s does not catch %q", s.Spec, m.ID, m.Detects, m.Bug)
		case "inapplicable":
			add(report.Error, s.Spec, "%s mutation %s does not apply", s.Spec, m.ID)
		}
	}
	seen := map[string]bool{}
	for _, r := range s.Runs {
		if r.Expect != "pass" {
			continue
		}
		for _, name := range append(append([]string{}, r.Invariants...), r.Properties...) {
			if name == "TypeOK" || caught[name] || seen[name] {
				continue
			}
			seen[name] = true
			add(report.Error, s.Spec, "%s: %s has no mutation it must catch", s.Spec, name)
		}
	}
}

// Stage renders a judgement as stage formal.
func (j *Judgement) Stage() *report.Stage {
	s := report.New("formal")
	if !j.Applicable {
		return s.NotApplicable("no formal model and no critical area")
	}
	s.Count(len(j.Specs), "specs").Count(j.Runs, "runs").Count(j.Reach, "reach predicates").Count(j.Mutations, "mutations").Count(j.Assumptions, "assumptions")
	for _, f := range j.Findings {
		s.Findings = append(s.Findings, report.Finding{Sev: f.Sev, Msg: f.Msg})
	}
	if j.ND != "" {
		s.NotDemonstrated(j.ND)
	}
	return s
}

// ProvenAreas returns the fv ids the formal stage proves (the hook the
// evidence loop uses).
func ProvenAreas(d *design.Design) map[string]bool {
	j, err := Judge(d)
	if err != nil {
		return nil
	}
	return j.Proven
}
