// Package checkers connects external deterministic checkers to machinery
// (docs/formats.md section 22): the declared facts, the projection a checker
// reads, the evidence it writes, the machine-local registry that holds its
// command, and the pure judgment of the evidence.
package checkers

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/contract"
	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gen"
)

// ProjectionFile is the name of the projection inside the gen directory.
const ProjectionFile = "projection.json"

func init() {
	gen.Extra = append(gen.Extra, func(d *design.Design) (gen.File, bool, error) {
		if !Configured(d) {
			return gen.File{}, false, nil
		}
		proj, _ := Build(d)
		data, err := proj.Bytes()
		if err != nil {
			return gen.File{}, false, err
		}
		return gen.File{Path: ProjectionPath(d), Data: data}, true, nil
	})
}

// Configured reports whether the project has checkers or a facts file.
func Configured(d *design.Design) bool {
	return len(d.P.Cfg.Checkers) > 0 || d.P.Cfg.Design.Facts != ""
}

// ProjectionPath is the project-relative path of projection.json.
func ProjectionPath(d *design.Design) string { return gen.Dir(d) + "/" + ProjectionFile }

type Attribute struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Relationship struct {
	Entity      string `json:"entity"`
	Cardinality string `json:"cardinality"`
	Ownership   string `json:"ownership"`
	Role        string `json:"role"`
}

type Action struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Actor     string   `json:"actor"`
	Preserves []string `json:"preserves"`
}

type Entity struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Attributes    []Attribute    `json:"attributes"`
	Relationships []Relationship `json:"relationships"`
	Actions       []Action       `json:"actions"`
}

type Enum struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type Invariant struct {
	ID        string `json:"id"`
	Entity    string `json:"entity"`
	Statement string `json:"statement"`
}

type Scenario struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Actors     []string `json:"actors"`
	Invariants []string `json:"invariants"`
}

type State struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Resting bool   `json:"resting"`
}

type Transition struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Kind    string   `json:"kind"`
	Event   string   `json:"event"`
	Guard   string   `json:"guard"`
	Target  string   `json:"target"`
	Effects []string `json:"effects"`
}

type Ignore struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Event string `json:"event"`
}

type Refusal struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	Handler string `json:"handler"`
}

type Clause struct {
	ID         string `json:"id"`
	Transition string `json:"transition"`
	Guard      string `json:"guard"`
	Clause     string `json:"clause"`
}

type Invoke struct {
	State string `json:"state"`
	Src   string `json:"src"`
}

type Machine struct {
	ID          string       `json:"id"`
	Tag         string       `json:"tag"`
	Role        string       `json:"role"`
	LifecycleOf string       `json:"lifecycle_of"`
	States      []State      `json:"states"`
	Transitions []Transition `json:"transitions"`
	Ignores     []Ignore     `json:"ignores"`
	Refusals    []Refusal    `json:"refusals"`
	Clauses     []Clause     `json:"clauses"`
	Invokes     []Invoke     `json:"invokes"`
}

type Operation struct {
	Contract string   `json:"contract"`
	ID       string   `json:"id"`
	Method   string   `json:"method"`
	Path     string   `json:"path"`
	Actions  []string `json:"actions"`
}

type FailureMode struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"`
	Required string `json:"required"`
}

type Dependency struct {
	Name    string `json:"name"`
	Posture string `json:"posture"`
}

// Projection is the whole design as one document (section 22.3).
type Projection struct {
	Version      int           `json:"version"`
	Entities     []Entity      `json:"entities"`
	Enums        []Enum        `json:"enums"`
	Invariants   []Invariant   `json:"invariants"`
	Scenarios    []Scenario    `json:"scenarios"`
	Machines     []Machine     `json:"machines"`
	Operations   []Operation   `json:"operations"`
	FailureModes []FailureMode `json:"failure_modes"`
	Dependencies []Dependency  `json:"dependencies"`
	Facts        []Fact        `json:"facts"`
}

// Bytes renders projection.json.
func (pr *Projection) Bytes() ([]byte, error) { return gen.JSON(pr) }

// SHA256 is the hex digest evidence binds to.
func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// IDs is the set of ids a checker may name: entities, actions, invariants,
// scenarios, transition cases and failure modes.
func (pr *Projection) IDs() map[string]bool {
	ids := map[string]bool{}
	for _, e := range pr.Entities {
		ids[e.ID] = true
		for _, a := range e.Actions {
			ids[a.ID] = true
		}
	}
	for _, i := range pr.Invariants {
		ids[i.ID] = true
	}
	for _, s := range pr.Scenarios {
		ids[s.ID] = true
	}
	for _, m := range pr.Machines {
		for _, t := range m.Transitions {
			ids[t.ID] = true
		}
		for _, ig := range m.Ignores {
			ids[ig.ID] = true
		}
		for _, rf := range m.Refusals {
			ids[rf.ID] = true
		}
		for _, cl := range m.Clauses {
			ids[cl.ID] = true
		}
	}
	for _, f := range pr.FailureModes {
		ids[f.ID] = true
	}
	return ids
}

// SortedIDs lists IDs in byte order.
func (pr *Projection) SortedIDs() []string {
	var res []string
	for id := range pr.IDs() {
		res = append(res, id)
	}
	sort.Strings(res)
	return res
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func prefixed(prefix string, names []string) []string {
	res := []string{}
	for _, n := range names {
		res = append(res, prefix+n)
	}
	sort.Strings(res)
	return res
}

// Build derives the projection from the loaded design. The second result lists
// the problems of the facts file; when there are any, the projection carries no
// facts and the stage reports the problems.
func Build(d *design.Design) (*Projection, []string) {
	pr := &Projection{Version: 1, Entities: []Entity{}, Enums: []Enum{}, Invariants: []Invariant{}, Scenarios: []Scenario{},
		Machines: []Machine{}, Operations: []Operation{}, FailureModes: []FailureMode{}, Dependencies: []Dependency{}, Facts: []Fact{}}
	for _, m := range d.Models {
		for _, en := range m.Enums {
			pr.Enums = append(pr.Enums, Enum{Name: en.Name, Values: nonNil(en.Values)})
		}
		for _, inv := range m.Invariants {
			pr.Invariants = append(pr.Invariants, Invariant{ID: "inv:" + inv.ID, Statement: design.Plain(inv.Statement)})
		}
		for _, e := range m.Entities {
			ent := Entity{ID: "entity:" + e.Name, Name: e.Name, Attributes: []Attribute{}, Relationships: []Relationship{}, Actions: []Action{}}
			for _, a := range e.Attributes {
				ent.Attributes = append(ent.Attributes, Attribute{Name: a.Name, Type: a.Type})
			}
			for _, r := range e.Relationships {
				ent.Relationships = append(ent.Relationships, Relationship{Entity: r.Entity, Cardinality: r.Cardinality, Ownership: r.Ownership, Role: r.Role})
			}
			for _, a := range e.Actions {
				ent.Actions = append(ent.Actions, Action{ID: design.ActID(e.Name, a.Name), Name: a.Name, Actor: a.Actor, Preserves: prefixed("inv:", a.Preserves)})
			}
			for _, inv := range e.Invariants {
				pr.Invariants = append(pr.Invariants, Invariant{ID: "inv:" + inv.ID, Entity: e.Name, Statement: design.Plain(inv.Statement)})
			}
			sort.Slice(ent.Attributes, func(i, j int) bool { return ent.Attributes[i].Name < ent.Attributes[j].Name })
			sort.Slice(ent.Relationships, func(i, j int) bool {
				a, b := ent.Relationships[i], ent.Relationships[j]
				if a.Entity != b.Entity {
					return a.Entity < b.Entity
				}
				return a.Role < b.Role
			})
			sort.Slice(ent.Actions, func(i, j int) bool { return ent.Actions[i].ID < ent.Actions[j].ID })
			pr.Entities = append(pr.Entities, ent)
		}
		for _, s := range m.Scenarios {
			actors := append([]string{}, s.Actors...)
			sort.Strings(actors)
			pr.Scenarios = append(pr.Scenarios, Scenario{ID: "sc:" + design.Slug(s.Name), Name: s.Name, Actors: actors, Invariants: prefixed("inv:", s.Invariants)})
		}
	}
	for _, m := range d.Machines {
		role := "lifecycle"
		if m.Role == "operational" {
			role = "operational"
		}
		mm := Machine{ID: m.ID, Tag: m.Tag, Role: role, LifecycleOf: d.LifecycleEntity(m),
			States: []State{}, Transitions: []Transition{}, Ignores: []Ignore{}, Refusals: []Refusal{}, Clauses: []Clause{}, Invokes: []Invoke{}}
		for _, s := range m.Table.States {
			mm.States = append(mm.States, State{Path: s.Path, Kind: s.Kind, Resting: s.Resting})
			for _, src := range s.Invoke {
				mm.Invokes = append(mm.Invokes, Invoke{State: s.Path, Src: src})
			}
		}
		for _, t := range m.Table.Transitions {
			mm.Transitions = append(mm.Transitions, Transition{ID: t.ID, Source: t.Source, Kind: t.Kind, Event: t.Event, Guard: t.Guard, Target: t.Target, Effects: nonNil(t.Effects)})
		}
		for _, ig := range m.Table.Ignores {
			mm.Ignores = append(mm.Ignores, Ignore{ID: ig.ID, State: ig.State, Event: ig.Event})
		}
		for _, rf := range m.Table.Refusals {
			mm.Refusals = append(mm.Refusals, Refusal{ID: rf.ID, State: rf.State, Handler: rf.Handler})
		}
		for _, cl := range m.Table.Clauses {
			mm.Clauses = append(mm.Clauses, Clause{ID: cl.ID, Transition: cl.Transition, Guard: cl.Guard, Clause: cl.Clause})
		}
		sort.Slice(mm.Refusals, func(i, j int) bool { return mm.Refusals[i].ID < mm.Refusals[j].ID })
		sort.Slice(mm.Clauses, func(i, j int) bool { return mm.Clauses[i].ID < mm.Clauses[j].ID })
		sort.Slice(mm.States, func(i, j int) bool { return mm.States[i].Path < mm.States[j].Path })
		sort.Slice(mm.Transitions, func(i, j int) bool { return mm.Transitions[i].ID < mm.Transitions[j].ID })
		sort.Slice(mm.Ignores, func(i, j int) bool { return mm.Ignores[i].ID < mm.Ignores[j].ID })
		sort.Slice(mm.Invokes, func(i, j int) bool {
			if mm.Invokes[i].State != mm.Invokes[j].State {
				return mm.Invokes[i].State < mm.Invokes[j].State
			}
			return mm.Invokes[i].Src < mm.Invokes[j].Src
		})
		pr.Machines = append(pr.Machines, mm)
	}
	ops, _ := contract.Load(d.P)
	for _, op := range ops {
		acts := []string{}
		for _, a := range op.Actions {
			if !strings.HasPrefix(a, "act:") {
				a = "act:" + a
			}
			acts = append(acts, a)
		}
		sort.Strings(acts)
		pr.Operations = append(pr.Operations, Operation{Contract: op.Contract, ID: op.ID, Method: op.Method, Path: op.Path, Actions: acts})
	}
	for _, fm := range d.FailureModes {
		pr.FailureModes = append(pr.FailureModes, FailureMode{ID: "fm:" + fm.ID, Subject: fm.Subject, Required: fm.Required})
	}
	for _, dep := range d.P.Cfg.Dependencies {
		pr.Dependencies = append(pr.Dependencies, Dependency{Name: dep.Name, Posture: dep.Posture})
	}
	sort.Slice(pr.Entities, func(i, j int) bool { return pr.Entities[i].Name < pr.Entities[j].Name })
	sort.Slice(pr.Enums, func(i, j int) bool { return pr.Enums[i].Name < pr.Enums[j].Name })
	sort.Slice(pr.Invariants, func(i, j int) bool { return pr.Invariants[i].ID < pr.Invariants[j].ID })
	sort.Slice(pr.Scenarios, func(i, j int) bool { return pr.Scenarios[i].ID < pr.Scenarios[j].ID })
	sort.Slice(pr.Machines, func(i, j int) bool { return pr.Machines[i].ID < pr.Machines[j].ID })
	sort.Slice(pr.Operations, func(i, j int) bool {
		a, b := pr.Operations[i], pr.Operations[j]
		if a.Contract != b.Contract {
			return a.Contract < b.Contract
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		return a.Path < b.Path
	})
	sort.Slice(pr.FailureModes, func(i, j int) bool { return pr.FailureModes[i].ID < pr.FailureModes[j].ID })
	sort.Slice(pr.Dependencies, func(i, j int) bool { return pr.Dependencies[i].Name < pr.Dependencies[j].Name })
	facts, problems := LoadFacts(d.P, pr.IDs())
	if len(problems) == 0 && facts != nil {
		pr.Facts = facts
	}
	return pr, problems
}
