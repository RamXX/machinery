package formal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/project"
)

// Skeleton renders the generated skeleton of a spec from the design.
func Skeleton(d *design.Design, s *Spec) ([]byte, error) {
	if s.Tool == "alloy" {
		for _, m := range d.Models {
			if m.Path == s.Model {
				return AlloySkeleton(m), nil
			}
		}
		return nil, fmt.Errorf("%s: model %s is not a configured model", s.Path, s.Model)
	}
	for _, m := range d.Machines {
		if m.Base == s.Machine {
			return TLASkeleton(m), nil
		}
	}
	return nil, fmt.Errorf("%s: no machine file %s.machine.json", s.Path, s.Machine)
}

func timeoutOf(r *Run) time.Duration {
	if r.Timeout > 0 {
		return time.Duration(r.Timeout) * time.Second
	}
	return 300 * time.Second
}

func expectOf(r *Run) string {
	switch {
	case r.Expect == "finding" && r.Assumption != "":
		return "refuted"
	case r.Expect == "finding":
		return "reproduced"
	}
	return "undetermined"
}

// Check runs one spec for real and writes its result and trace files.
func Check(p *project.Project, d *design.Design, s *Spec, workers int) (*Result, error) {
	res, _, err := run(p, d, s, workers, true)
	return res, err
}

// Recheck runs one spec for real, writes nothing, and returns the committed
// files that differ from the fresh run. Committed results are data anyone can
// edit; a fresh run is what makes them trustworthy.
//
// A fresh run that did not complete has no verdict to compare: it returns a
// ToolError naming the first run or mutation that did not complete.
func Recheck(p *project.Project, d *design.Design, s *Spec, workers int) (differ []string, err error) {
	res, differ, err := run(p, d, s, workers, false)
	if err != nil {
		return nil, err
	}
	if inc := res.Incomplete(); len(inc) > 0 {
		return nil, &ToolError{inc[0]}
	}
	return differ, nil
}

func run(p *project.Project, d *design.Design, s *Spec, workers int, write bool) (_ *Result, differ []string, _ error) {
	put := func(rel string, v any) error {
		if write {
			return writeJSON(p, rel, v)
		}
		want, err := gen.JSON(v)
		if err != nil {
			return err
		}
		if got, rerr := p.Read(rel); rerr != nil || !bytes.Equal(got, want) {
			differ = append(differ, rel)
		}
		return nil
	}
	if len(s.Problems) > 0 {
		return nil, nil, fmt.Errorf("%s", s.Problems[0])
	}
	want, err := Skeleton(d, s)
	if err != nil {
		return nil, nil, err
	}
	skeleton, rerr := p.Read(s.SkeletonPath())
	if rerr != nil || !bytes.Equal(skeleton, want) {
		return nil, nil, fmt.Errorf("%s: skeleton is stale or missing: run machinery formal gen first", s.SkeletonPath())
	}
	props, perr := p.Read(s.PropsPath())
	if perr != nil {
		return nil, nil, fmt.Errorf("%s: property file is missing", s.PropsPath())
	}
	tool := ToolFor(s.Tool)
	if err := Ready(p, tool); err != nil {
		return nil, nil, err
	}
	tmp, err := os.MkdirTemp("", "machinery-formal-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(tmp)
	res := &Result{Version: 1, Spec: s.Spec, Tool: s.Tool, ToolVersion: tool.Label(),
		InputSHA256: s.InputHash(skeleton, props), Runs: []RunResult{}, Mutations: []MutationResult{}}
	machineID := ""
	if s.Tool == "tla" {
		for _, m := range d.Machines {
			if m.Base == s.Machine {
				machineID = m.ID
			}
		}
	}
	files := map[string][]byte{s.SkeletonName(): skeleton, s.Properties + s.Ext(): props}
	stage := func(name string, mutated map[string][]byte) (string, error) {
		dir := filepath.Join(tmp, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
		for fn, data := range files {
			if m, ok := mutated[fn]; ok {
				data = m
			}
			if err := os.WriteFile(filepath.Join(dir, fn), data, 0o644); err != nil {
				return "", err
			}
		}
		return dir, nil
	}
	var chk checker
	if s.Tool == "alloy" {
		chk = &alloyChecker{s: s, r: &alloyRunner{p.Cfg.Tools.Java, tool.Path(p.ToolsDir())}}
	} else {
		chk = &tlaChecker{s: s, r: &tlcRunner{p.Cfg.Tools.Java, tool.Path(p.ToolsDir()), workers}}
	}
	for i := range s.Runs {
		run := &s.Runs[i]
		dir, err := stage("run-"+run.Name, nil)
		if err != nil {
			return nil, nil, err
		}
		rr, trace := chk.run(dir, run)
		tracePath := s.TracePath(run.Name)
		switch {
		case rr.Outcome == "incomplete":
			// no verdict: the trace file stays as it was (section 16.9)
		case trace != nil:
			trace.Version, trace.ID, trace.Spec, trace.Run, trace.Tool = 1, "cx:"+s.Spec+":"+run.Name, s.Spec, run.Name, s.Tool
			trace.Violates, trace.Expect, trace.Machine = rr.Violated, expectOf(run), machineID
			if trace.Steps == nil {
				trace.Steps = []Step{}
			}
			if trace.Instance == nil {
				trace.Instance = map[string]map[string][]string{}
			}
			if err := put(tracePath, trace); err != nil {
				return nil, nil, err
			}
			rr.Trace = strings.TrimPrefix(tracePath, s.Dir+"/")
		case p.Exists(tracePath):
			if write {
				os.Remove(p.Abs(tracePath))
			} else {
				differ = append(differ, tracePath)
			}
		}
		if rr.Reach == nil {
			rr.Reach = map[string]bool{}
		}
		res.Runs = append(res.Runs, rr)
	}
	for _, m := range s.Mutations {
		fn := s.MutationFile(m)
		// inapplicable is decided here, before any tool runs, and only here
		outcome, reason := "inapplicable", ""
		if src, ok := files[fn]; ok && bytes.Count(src, []byte(m.From)) == 1 {
			mutated := bytes.Replace(src, []byte(m.From), []byte(m.To), 1)
			dir, err := stage("mut-"+m.ID, map[string][]byte{fn: mutated})
			if err != nil {
				return nil, nil, err
			}
			if run := s.RunByName(m.Run); run != nil {
				outcome, reason = chk.mutation(dir, run, m.Detects)
			}
		}
		res.Mutations = append(res.Mutations, MutationResult{ID: m.ID, Outcome: outcome, Reason: reason})
	}
	if err := put(s.ResultPath(), res); err != nil {
		return nil, nil, err
	}
	return res, differ, nil
}

// checker runs one tool. A run whose tool run did not complete is
// "incomplete" with a reason, and so is a mutation (section 16.9).
type checker interface {
	run(dir string, r *Run) (RunResult, *Trace)
	mutation(dir string, r *Run, detects string) (outcome, reason string)
}

// incomplete is the result of a run that gave no verdict.
func incomplete(r *Run, reason string) (RunResult, *Trace) {
	return RunResult{Name: r.Name, Outcome: "incomplete", Reason: reason, Reach: map[string]bool{}}, nil
}

type tlaChecker struct {
	s *Spec
	r *tlcRunner
}

func (c *tlaChecker) run(dir string, r *Run) (RunResult, *Trace) {
	rr := RunResult{Name: r.Name, Reach: map[string]bool{}}
	out := c.r.run(dir, c.s.Properties, tlcConfig(r, r.Invariants, r.Properties), true, timeoutOf(r))
	if out.Outcome == "incomplete" {
		return incomplete(r, out.Reason)
	}
	rr.Outcome, rr.Violated = out.Outcome, out.Violated
	var trace *Trace
	if out.Outcome == "violated" {
		if out.Kind == "property" {
			rr.Violated = strings.Join(r.Properties, ",")
			if len(r.Properties) > 1 {
				for _, prop := range r.Properties {
					one := c.r.run(dir, c.s.Properties, tlcConfig(r, nil, []string{prop}), false, timeoutOf(r))
					if one.Outcome == "incomplete" {
						return incomplete(r, "property "+prop+": "+one.Reason)
					}
					if one.Outcome == "violated" {
						rr.Violated = prop
						out = one
						break
					}
				}
			}
		}
		// With several workers TLC may report a different, equally short
		// counterexample each time. The trace file is generated data that must
		// not change between runs, so the trace comes from one worker.
		if c.r.workers > 1 {
			single := *c.r
			single.workers = 1
			var det tlcOutcome
			switch out.Kind {
			case "property":
				det = single.run(dir, c.s.Properties, tlcConfig(r, nil, []string{rr.Violated}), false, timeoutOf(r))
			case "deadlock":
				det = single.run(dir, c.s.Properties, tlcConfig(r, r.Invariants, r.Properties), true, timeoutOf(r))
			default:
				det = single.run(dir, c.s.Properties, tlcConfig(r, []string{rr.Violated}, nil), false, timeoutOf(r))
			}
			if det.Outcome == "incomplete" {
				return incomplete(r, "trace: "+det.Reason)
			}
			if det.Outcome == "violated" {
				out = det
			}
		}
		trace = &Trace{Steps: out.steps(), Loop: out.loopStep()}
	}
	for _, name := range r.Reach {
		mod := "MCReach"
		text := fmt.Sprintf("---- MODULE %s ----\nEXTENDS %s\nMCNotReached == ~(%s)\n====\n", mod, c.s.Properties, name)
		if err := os.WriteFile(filepath.Join(dir, mod+".tla"), []byte(text), 0o644); err != nil {
			return incomplete(r, "reach "+name+": "+relativeTo(dir, err.Error()))
		}
		one := c.r.run(dir, mod, tlcConfig(r, []string{"MCNotReached"}, nil), false, timeoutOf(r))
		if one.Outcome == "incomplete" {
			return incomplete(r, "reach "+name+": "+one.Reason)
		}
		rr.Reach[name] = one.Outcome == "violated" && one.Violated == "MCNotReached"
	}
	return rr, trace
}

func (c *tlaChecker) mutation(dir string, r *Run, detects string) (string, string) {
	var out tlcOutcome
	if contains(r.Properties, detects) {
		out = c.r.run(dir, c.s.Properties, tlcConfig(r, nil, []string{detects}), false, timeoutOf(r))
	} else {
		out = c.r.run(dir, c.s.Properties, tlcConfig(r, []string{detects}, nil), false, timeoutOf(r))
	}
	switch out.Outcome {
	case "violated":
		return "detected", ""
	case "passed":
		return "missed", ""
	}
	return "incomplete", out.Reason
}

type alloyChecker struct {
	s *Spec
	r *alloyRunner
}

func (c *alloyChecker) withCommands(dir string, commands []string) error {
	fn := filepath.Join(dir, c.s.Properties+".als")
	props, err := os.ReadFile(fn)
	if err != nil {
		return err
	}
	text := string(props) + "\n" + strings.Join(commands, "\n") + "\n"
	return os.WriteFile(fn, []byte(text), 0o644)
}

func (c *alloyChecker) run(dir string, r *Run) (RunResult, *Trace) {
	rr := RunResult{Name: r.Name, Reach: map[string]bool{}}
	scope := alloyScope(r)
	var cmds []string
	for _, a := range r.Invariants {
		cmds = append(cmds, fmt.Sprintf("check %s for %d", a, scope))
	}
	for _, pd := range r.Reach {
		cmds = append(cmds, fmt.Sprintf("run %s for %d", pd, scope))
	}
	if err := c.withCommands(dir, cmds); err != nil {
		return incomplete(r, relativeTo(dir, err.Error()))
	}
	found, known, reason := c.r.exec(dir, c.s.Properties+".als", timeoutOf(r))
	if reason != "" {
		return incomplete(r, reason)
	}
	// a receipt without one of the commands is output that cannot be read
	for _, a := range r.Invariants {
		if !known[a] {
			return incomplete(r, "Alloy did not check assertion "+a)
		}
	}
	for _, pd := range r.Reach {
		if !known[pd] {
			return incomplete(r, "Alloy did not run predicate "+pd)
		}
	}
	rr.Outcome = "passed"
	var trace *Trace
	for _, a := range r.Invariants {
		if inst, ok := found[a]; ok && rr.Outcome == "passed" {
			rr.Outcome, rr.Violated = "violated", a
			trace = &Trace{Instance: inst}
		}
	}
	for _, pd := range r.Reach {
		_, ok := found[pd]
		rr.Reach[pd] = ok
	}
	return rr, trace
}

func (c *alloyChecker) mutation(dir string, r *Run, detects string) (string, string) {
	if err := c.withCommands(dir, []string{fmt.Sprintf("check %s for %d", detects, alloyScope(r))}); err != nil {
		return "incomplete", relativeTo(dir, err.Error())
	}
	found, known, reason := c.r.exec(dir, c.s.Properties+".als", timeoutOf(r))
	switch {
	case reason != "":
		return "incomplete", reason
	case !known[detects]:
		return "incomplete", "Alloy did not check assertion " + detects
	}
	if _, ok := found[detects]; ok {
		return "detected", ""
	}
	return "missed", ""
}

// HasCommands reports whether an Alloy property file carries its own commands.
func HasCommands(props []byte) bool { return alloyCommandRe.Match(props) }
