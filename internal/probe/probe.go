// Package probe mutates the generated tables and checks that the tests notice
// (docs/formats.md section 19).
package probe

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/evidence"
	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/machine"
	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// Options selects what to probe and how to run the tests.
type Options struct {
	IDs      []string
	Prefixes []string
	Max      int // 0: no limit
	Argv     []string
	Stack    string
	Stdout   io.Writer
	Stderr   io.Writer
}

// Mutation is one change to one transition of one table.
type Mutation struct {
	Case    string
	Kind    string // retarget | drop-effect
	Change  string // what was changed, for the report
	Machine *machine.Machine
	apply   func(t *machine.Transition)
}

// Mutations lists, in id order, the mutations of the proven transition cases
// that the selectors pick.
func Mutations(d *design.Design, ev *evidence.Evaluation, opts Options) []Mutation {
	var res []Mutation
	for _, o := range d.Obligations {
		if o.Kind != "tc" || ev.State(o.ID) != evidence.Proven || !selected(o.ID, opts) {
			continue
		}
		m := d.MachineOfCase(o.ID)
		if m == nil {
			continue
		}
		var tr *machine.Transition
		for i := range m.Table.Transitions {
			if m.Table.Transitions[i].ID == o.ID {
				tr = &m.Table.Transitions[i]
			}
		}
		if tr == nil {
			continue
		}
		if tr.Target != "" {
			for _, s := range m.Table.States {
				if s.Path != tr.Target && s.Path != tr.Source {
					old, to := tr.Target, s.Path
					res = append(res, Mutation{Case: o.ID, Kind: "retarget", Machine: m,
						Change: fmt.Sprintf("target %s -> %s", old, to),
						apply:  func(t *machine.Transition) { t.Target = to }})
					break
				}
			}
		}
		if len(tr.Actions) > 0 {
			name := tr.Actions[0]
			res = append(res, Mutation{Case: o.ID, Kind: "drop-effect", Machine: m,
				Change: "dropped " + name,
				apply: func(t *machine.Transition) {
					t.Actions = without(t.Actions, name, true)
					t.Effects = without(t.Effects, name, false)
				}})
		}
	}
	if opts.Max > 0 && len(res) > opts.Max {
		res = res[:opts.Max]
	}
	return res
}

func selected(id string, opts Options) bool {
	if len(opts.IDs) == 0 && len(opts.Prefixes) == 0 {
		return true
	}
	for _, s := range opts.IDs {
		if s == id {
			return true
		}
	}
	for _, p := range opts.Prefixes {
		if strings.HasPrefix(id, strings.TrimSuffix(p, "*")) {
			return true
		}
	}
	return false
}

// without removes name from list: only its first occurrence when first is true.
func without(list []string, name string, first bool) []string {
	res := []string{}
	done := false
	for _, n := range list {
		if n == name && !(first && done) {
			done = true
			continue
		}
		res = append(res, n)
	}
	return res
}

// Stage runs every selected mutation and reports stage probe.
func Stage(p *project.Project, d *design.Design, opts Options) *report.Stage {
	s := report.New("probe")
	ev, err := evidence.Evaluate(p, d, false)
	if err != nil {
		s.Errorf("%v", err)
		return s
	}
	if ev.Dirty {
		return s.NotDemonstrated("working tree has uncommitted changes: the probe needs the evidence of a clean tree")
	}
	if !ev.HasEvidence {
		return s.NotDemonstrated(ev.NoEvidence())
	}
	muts := Mutations(d, ev, opts)
	if len(muts) == 0 {
		return s.NotDemonstrated("nothing to check")
	}
	killed, survived := 0, 0
	for _, m := range muts {
		res, err := runOne(p, d, m, opts)
		switch {
		case err != nil:
			s.Errorf("probe run failed: %s %s: %v", m.Case, m.Kind, err)
		case res == "none":
			s.Errorf("probe run produced no receipts: %s %s", m.Case, m.Kind)
		case res == "killed":
			killed++
		default:
			survived++
			s.Errorf("survived: %s %s (%s)", m.Case, m.Kind, m.Change)
		}
	}
	s.Count(len(muts), "mutations").Count(killed, "killed").Count(survived, "survived")
	return s
}

// runOne runs the tests against one mutated copy of the tables and returns
// killed, survived or none (no receipt at all).
func runOne(p *project.Project, d *design.Design, m Mutation, opts Options) (string, error) {
	tmp, err := os.MkdirTemp("", "machinery-probe-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)
	tables := filepath.Join(tmp, "tables")
	if err := copyTables(p, gen.Dir(d)+"/tables", tables); err != nil {
		return "", err
	}
	if err := mutate(filepath.Join(tables, m.Machine.ID+".table.json"), m); err != nil {
		return "", err
	}
	file := filepath.Join(tmp, "receipts.jsonl")
	_, _, err = evidence.Run(p, evidence.RunOptions{
		Stack:        opts.Stack,
		Env:          []string{"MACHINERY_TABLES_DIR=" + tables, "MACHINERY_PROBE_CASE=" + m.Case},
		ReceiptsFile: file,
		Stdout:       opts.Stdout,
		Stderr:       opts.Stderr,
	}, opts.Argv)
	if err != nil {
		return "", err
	}
	j, n, err := evidence.JudgeFile(d, file)
	if err != nil {
		return "", err
	}
	switch {
	case n == 0:
		return "none", nil
	case j.States[m.Case] == evidence.Proven:
		return "survived", nil
	}
	return "killed", nil
}

func copyTables(p *project.Project, rel, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(p.Abs(rel))
	if err != nil {
		return fmt.Errorf("%s: %v; run machinery gen", rel, err)
	}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.Abs(rel), e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func mutate(path string, m Mutation) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("table %s.table.json is missing; run machinery gen", m.Machine.ID)
	}
	var t machine.Table
	if err := json.Unmarshal(data, &t); err != nil {
		return fmt.Errorf("table %s.table.json: %v", m.Machine.ID, err)
	}
	found := false
	for i := range t.Transitions {
		if t.Transitions[i].ID == m.Case {
			m.apply(&t.Transitions[i])
			found = true
		}
	}
	if !found {
		return fmt.Errorf("table %s.table.json has no transition %s; run machinery gen", m.Machine.ID, m.Case)
	}
	out, err := gen.JSON(&t)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
