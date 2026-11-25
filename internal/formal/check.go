package formal

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RamXX/machinery/internal/design"
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
	if len(s.Problems) > 0 {
		return nil, fmt.Errorf("%s", s.Problems[0])
	}
	want, err := Skeleton(d, s)
	if err != nil {
		return nil, err
	}
	skeleton, rerr := p.Read(s.SkeletonPath())
	if rerr != nil || !bytes.Equal(skeleton, want) {
		return nil, fmt.Errorf("%s: skeleton is stale or missing: run machinery formal gen first", s.SkeletonPath())
	}
	props, perr := p.Read(s.PropsPath())
	if perr != nil {
		return nil, fmt.Errorf("%s: property file is missing", s.PropsPath())
	}
	tool := ToolFor(s.Tool)
	if err := Ready(p, tool); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "machinery-formal-")
	if err != nil {
		return nil, err
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
			return nil, err
		}
		rr, trace := chk.run(dir, run)
		tracePath := s.TracePath(run.Name)
		if trace != nil {
			trace.Version, trace.ID, trace.Spec, trace.Run, trace.Tool = 1, "cx:"+s.Spec+":"+run.Name, s.Spec, run.Name, s.Tool
			trace.Violates, trace.Expect, trace.Machine = rr.Violated, expectOf(run), machineID
			if trace.Steps == nil {
				trace.Steps = []Step{}
			}
			if trace.Instance == nil {
				trace.Instance = map[string]map[string][]string{}
			}
			if err := writeJSON(p, tracePath, trace); err != nil {
				return nil, err
			}
			rr.Trace = strings.TrimPrefix(tracePath, s.Dir+"/")
		} else if p.Exists(tracePath) {
			os.Remove(p.Abs(tracePath))
		}
		if rr.Reach == nil {
			rr.Reach = map[string]bool{}
		}
		res.Runs = append(res.Runs, rr)
	}
	for _, m := range s.Mutations {
		fn := s.MutationFile(m)
		outcome := "inapplicable"
		if src, ok := files[fn]; ok && bytes.Count(src, []byte(m.From)) == 1 {
			mutated := bytes.Replace(src, []byte(m.From), []byte(m.To), 1)
			dir, err := stage("mut-"+m.ID, map[string][]byte{fn: mutated})
			if err != nil {
				return nil, err
			}
			if run := s.RunByName(m.Run); run != nil {
				outcome = chk.mutation(dir, run, m.Detects)
			}
		}
		res.Mutations = append(res.Mutations, MutationResult{ID: m.ID, Outcome: outcome})
	}
	if err := writeJSON(p, s.ResultPath(), res); err != nil {
		return nil, err
	}
	return res, nil
}

// checker runs one tool.
type checker interface {
	run(dir string, r *Run) (RunResult, *Trace)
	mutation(dir string, r *Run, detects string) string
}

type tlaChecker struct {
	s *Spec
	r *tlcRunner
}

func (c *tlaChecker) run(dir string, r *Run) (RunResult, *Trace) {
	rr := RunResult{Name: r.Name, Reach: map[string]bool{}}
	out := c.r.run(dir, c.s.Properties, tlcConfig(r, r.Invariants, r.Properties), true, timeoutOf(r))
	rr.Outcome, rr.Violated = out.Outcome, out.Violated
	var trace *Trace
	if out.Outcome == "violated" {
		if out.Kind == "property" {
			rr.Violated = strings.Join(r.Properties, ",")
			if len(r.Properties) > 1 {
				for _, prop := range r.Properties {
					one := c.r.run(dir, c.s.Properties, tlcConfig(r, nil, []string{prop}), false, timeoutOf(r))
					if one.Outcome == "violated" {
						rr.Violated = prop
						out = one
						break
					}
				}
			}
		}
		trace = &Trace{Steps: out.steps(), Loop: out.loopStep()}
	}
	for _, name := range r.Reach {
		mod := "MCReach"
		text := fmt.Sprintf("---- MODULE %s ----\nEXTENDS %s\nMCNotReached == ~(%s)\n====\n", mod, c.s.Properties, name)
		if err := os.WriteFile(filepath.Join(dir, mod+".tla"), []byte(text), 0o644); err != nil {
			rr.Reach[name] = false
			continue
		}
		one := c.r.run(dir, mod, tlcConfig(r, []string{"MCNotReached"}, nil), false, timeoutOf(r))
		rr.Reach[name] = one.Outcome == "violated" && one.Violated == "MCNotReached"
	}
	return rr, trace
}

func (c *tlaChecker) mutation(dir string, r *Run, detects string) string {
	var out tlcOutcome
	if contains(r.Properties, detects) {
		out = c.r.run(dir, c.s.Properties, tlcConfig(r, nil, []string{detects}), false, timeoutOf(r))
	} else {
		out = c.r.run(dir, c.s.Properties, tlcConfig(r, []string{detects}, nil), false, timeoutOf(r))
	}
	switch out.Outcome {
	case "violated":
		return "detected"
	case "passed":
		return "missed"
	}
	return "inapplicable"
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
		rr.Outcome, rr.Violated = "error", err.Error()
		return rr, nil
	}
	found, known, errLine := c.r.exec(dir, c.s.Properties+".als", timeoutOf(r))
	if errLine != "" {
		rr.Outcome, rr.Violated = "error", errLine
		return rr, nil
	}
	rr.Outcome = "passed"
	var trace *Trace
	for _, a := range r.Invariants {
		if !known[a] {
			rr.Outcome, rr.Violated = "error", "assertion "+a+" was not checked"
			return rr, nil
		}
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

func (c *alloyChecker) mutation(dir string, r *Run, detects string) string {
	if err := c.withCommands(dir, []string{fmt.Sprintf("check %s for %d", detects, alloyScope(r))}); err != nil {
		return "inapplicable"
	}
	found, known, errLine := c.r.exec(dir, c.s.Properties+".als", timeoutOf(r))
	if errLine != "" || !known[detects] {
		return "inapplicable"
	}
	if _, ok := found[detects]; ok {
		return "detected"
	}
	return "missed"
}

// HasCommands reports whether an Alloy property file carries its own commands.
func HasCommands(props []byte) bool { return alloyCommandRe.Match(props) }
