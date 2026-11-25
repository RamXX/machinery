package design

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/machine"
	"github.com/RamXX/machinery/internal/project"
)

// Obligation is an entry of obligations.json (section 7.1).
type Obligation struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Source  string   `json:"source"`
	Title   string   `json:"title"`
	Effects []string `json:"-"`
	// HasEffects is true for tc ids: effects is then always written, even when empty.
	HasEffects bool `json:"-"`
}

// Served is an act id.
type Served struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

// TraceRef is what the inventory needs from a counterexample trace file.
type TraceRef struct {
	Path     string
	ID       string `json:"id"`
	Spec     string `json:"spec"`
	Run      string `json:"run"`
	Violates string `json:"violates"`
	Expect   string `json:"expect"`
}

// Design is everything loaded from a project's design sources.
type Design struct {
	P *project.Project

	Models        []*Model
	ModelProblems []string

	Machines        []*machine.Machine
	MachineProblems []string

	FailureModes []FailureMode
	FMProblems   []string
	HasFM        bool

	Areas        []Area
	RiskProblems []string
	HasRisk      bool

	Owed         []OwedEntry
	OwedProblems []string

	Traces []TraceRef

	Obligations []Obligation // sorted by id
	Served      []Served     // sorted by id
	Duplicates  []string     // ids declared more than once

	byID map[string]*Obligation
	acts map[string]bool
}

// Load reads every design source. It never fails: problems are kept per
// source for the stage that owns them.
func Load(p *project.Project) *Design {
	d := &Design{P: p}
	d.Models, d.ModelProblems = LoadModels(p)
	ms, probs, err := machine.LoadDir(p)
	if err != nil {
		probs = append(probs, err.Error())
	}
	d.Machines, d.MachineProblems = ms, probs
	d.FailureModes, d.FMProblems, d.HasFM = LoadFailureModes(p)
	d.Areas, d.RiskProblems, d.HasRisk = LoadRisk(p)
	d.Owed, d.OwedProblems = LoadOwed(p)
	d.Traces = loadTraces(p)
	d.inventory()
	return d
}

func loadTraces(p *project.Project) []TraceRef {
	files, _ := p.Glob([]string{strings.TrimSuffix(p.Cfg.Design.Formal, "/") + "/traces/*.trace.json"})
	var res []TraceRef
	for _, f := range files {
		data, err := p.Read(f)
		if err != nil {
			continue
		}
		var t TraceRef
		if json.Unmarshal(data, &t) != nil || t.ID == "" {
			continue
		}
		t.Path = f
		res = append(res, t)
	}
	return res
}

type checkerEvidence struct {
	Checker     string `json:"checker"`
	Obligations []struct {
		Name  string `json:"name"`
		Title string `json:"title"`
	} `json:"obligations"`
}

func (d *Design) inventory() {
	seen := map[string]bool{}
	add := func(o Obligation) {
		if seen[o.ID] {
			d.Duplicates = append(d.Duplicates, o.ID)
			return
		}
		seen[o.ID] = true
		d.Obligations = append(d.Obligations, o)
	}
	d.acts = map[string]bool{}
	for _, m := range d.Models {
		for _, inv := range m.Invariants {
			add(Obligation{ID: "inv:" + inv.ID, Kind: "inv", Source: m.Path, Title: Plain(inv.Statement)})
		}
		for _, e := range m.Entities {
			for _, inv := range e.Invariants {
				add(Obligation{ID: "inv:" + inv.ID, Kind: "inv", Source: m.Path, Title: Plain(inv.Statement)})
			}
			for _, a := range e.Actions {
				id := ActID(e.Name, a.Name)
				if d.acts[id] {
					d.Duplicates = append(d.Duplicates, id)
					continue
				}
				d.acts[id] = true
				d.Served = append(d.Served, Served{ID: id, Source: m.Path})
			}
		}
		for _, s := range m.Scenarios {
			add(Obligation{ID: "sc:" + Slug(s.Name), Kind: "sc", Source: m.Path, Title: s.Name})
		}
	}
	for _, m := range d.Machines {
		for _, c := range m.Table.Cases() {
			add(Obligation{ID: c.ID, Kind: "tc", Source: m.Path, Title: c.Title, Effects: c.Effects, HasEffects: true})
		}
	}
	for _, fm := range d.FailureModes {
		add(Obligation{ID: "fm:" + fm.ID, Kind: "fm", Source: d.P.Cfg.Design.FailureModes, Title: fm.Failure})
	}
	for _, a := range d.Areas {
		if a.Tier == Critical {
			add(Obligation{ID: "fv:" + a.Name, Kind: "fv", Source: d.P.Cfg.Design.Risk, Title: fmt.Sprintf("Formal model of %s: %s", a.Name, a.Why)})
		}
	}
	for _, t := range d.Traces {
		add(Obligation{ID: t.ID, Kind: "cx", Source: t.Path, Title: fmt.Sprintf("Replay %s (%s run %s)", t.Violates, t.Spec, t.Run)})
	}
	for _, ck := range d.P.Cfg.Checkers {
		data, err := d.P.Read(ck.Evidence)
		if err != nil {
			continue
		}
		var ev checkerEvidence
		if json.Unmarshal(data, &ev) != nil {
			continue
		}
		for _, o := range ev.Obligations {
			add(Obligation{ID: "ck:" + ck.ID + ":" + o.Name, Kind: "ck", Source: ck.Evidence, Title: o.Title})
		}
	}
	sort.Slice(d.Obligations, func(i, j int) bool { return d.Obligations[i].ID < d.Obligations[j].ID })
	sort.Slice(d.Served, func(i, j int) bool { return d.Served[i].ID < d.Served[j].ID })
	sort.Strings(d.Duplicates)
	d.byID = map[string]*Obligation{}
	for i := range d.Obligations {
		d.byID[d.Obligations[i].ID] = &d.Obligations[i]
	}
}

// Plain strips the backticks Modelith text uses around entity names.
func Plain(s string) string { return strings.ReplaceAll(s, "`", "") }

// Obligation returns the obligation with the id, or nil.
func (d *Design) Obligation(id string) *Obligation { return d.byID[id] }

// IsAct reports whether id is a model action.
func (d *Design) IsAct(id string) bool { return d.acts[id] }

// Known reports whether id is an obligation or an action.
func (d *Design) Known(id string) bool { return d.byID[id] != nil || d.acts[id] }

// AllIDs lists every act and obligation id, sorted.
func (d *Design) AllIDs() []string {
	var ids []string
	for _, s := range d.Served {
		ids = append(ids, s.ID)
	}
	for _, o := range d.Obligations {
		ids = append(ids, o.ID)
	}
	sort.Strings(ids)
	return ids
}

// Tier returns the tier of an id and the areas that give it.
func (d *Design) Tier(id string) (string, []Area) { return TierOf(d.Areas, id) }

// OwedSet returns the owed ids.
func (d *Design) OwedSet() map[string]OwedEntry {
	res := map[string]OwedEntry{}
	for _, e := range d.Owed {
		res[e.ID] = e
	}
	return res
}

// MachineByID returns the machine with the given machine id.
func (d *Design) MachineByID(id string) *machine.Machine {
	for _, m := range d.Machines {
		if m.ID == id {
			return m
		}
	}
	return nil
}

// LifecycleEntity returns the entity a machine is the lifecycle of, or "".
func (d *Design) LifecycleEntity(m *machine.Machine) string {
	if m.LifecycleOf != "" {
		return m.LifecycleOf
	}
	if m.Role == "operational" {
		return ""
	}
	if e, _ := FindEntity(d.Models, m.Base); e != nil {
		return e.Name
	}
	return ""
}

// MachineOfCase returns the machine that owns a tc id.
func (d *Design) MachineOfCase(id string) *machine.Machine {
	for _, m := range d.Machines {
		for _, c := range m.Table.Cases() {
			if c.ID == id {
				return m
			}
		}
	}
	return nil
}

// ContractFiles lists the design sources that make up the contract hash
// (section 15.4), sorted.
func (d *Design) ContractFiles() []string {
	c := d.P.Cfg
	set := map[string]bool{}
	for _, m := range c.Design.Models {
		set[m] = true
	}
	for _, m := range d.Machines {
		set[m.Path] = true
	}
	for _, f := range []string{c.Design.FailureModes, c.Design.Risk, c.Design.Facts} {
		if f != "" && d.P.Exists(f) {
			set[f] = true
		}
	}
	for _, ct := range c.Contracts {
		set[ct.Path] = true
	}
	var res []string
	for f := range set {
		res = append(res, f)
	}
	sort.Strings(res)
	return res
}
