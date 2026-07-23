package checker

import (
	"fmt"
	"os"

	"github.com/RamXX/machinery/internal/ir"
)

// Model is the lightweight view of a Modelith domain the projection needs. It
// is read through internal/ir, the same order-preserving parser the relational
// generators use, so no second YAML schema drifts from Modelith's.
type Model struct {
	Entities      []Entity
	Invariants    []Invariant
	Relationships []Relationship
}

// Entity is a domain entity with its lifecycle enum values (when it has one)
// and its attributes.
type Entity struct {
	Name       string
	StatusEnum []string
	Attributes []Attr
}

// Attr is a single attribute (Modelith carries name and type; cardinality and
// uniqueness live in invariants, not on the attribute, so they are absent here).
type Attr struct {
	Name string
	Type string
}

// Invariant is a domain invariant with its owner. OwnerKind is "entity" (with
// Owner naming it) or "top".
type Invariant struct {
	ID        string
	OwnerKind string
	Owner     string
	Statement string
}

// Relationship is a directed edge From an entity To another, with cardinality.
type Relationship struct {
	From        string
	To          string
	Cardinality string
}

func arr(v *ir.Value) []*ir.Value {
	if v == nil || v.Kind != ir.KindArray {
		return nil
	}
	return v.AsArray()
}

func isLifecycleAttr(name string) bool {
	switch name {
	case "status", "stage", "state":
		return true
	}
	return false
}

// LoadModel parses a Modelith domain model into a Model. An unreadable file, a
// non-mapping root, or a model with no entities is an error: the projection has
// no meaning without a domain.
func LoadModel(path string) (*Model, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	root, err := ir.LoadYAML(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	obj := root.AsObject()
	if obj == nil {
		return nil, fmt.Errorf("%s: not a yaml mapping", path)
	}

	enums := map[string][]string{}
	if ev := obj.Get2("enums"); ev != nil && ev.Kind == ir.KindObject {
		eo := ev.AsObject()
		for _, name := range eo.Keys() {
			var vals []string
			if veo := eo.Get2(name); veo != nil && veo.Kind == ir.KindObject {
				for _, vv := range arr(veo.AsObject().Get2("values")) {
					if vo := vv.AsObject(); vo != nil {
						vals = append(vals, vo.GetString("name"))
					}
				}
			}
			enums[name] = vals
		}
	}

	m := &Model{}
	for _, iv := range arr(obj.Get2("invariants")) {
		if io := iv.AsObject(); io != nil {
			if id := io.GetString("id"); id != "" {
				m.Invariants = append(m.Invariants, Invariant{ID: id, OwnerKind: "top", Statement: io.GetString("statement")})
			}
		}
	}

	ents := obj.GetObject("entities")
	if ents == nil || ents.Len() == 0 {
		return nil, fmt.Errorf("%s: declares no entities", path)
	}
	for _, ename := range ents.Keys() {
		ev := ents.Get2(ename)
		if ev == nil || ev.Kind != ir.KindObject {
			continue
		}
		eo := ev.AsObject()
		e := Entity{Name: ename}
		for _, av := range arr(eo.Get2("attributes")) {
			ao := av.AsObject()
			if ao == nil {
				continue
			}
			name, typ := ao.GetString("name"), ao.GetString("type")
			e.Attributes = append(e.Attributes, Attr{Name: name, Type: typ})
			if isLifecycleAttr(name) {
				if vals, ok := enums[typ]; ok {
					e.StatusEnum = vals
				}
			}
		}
		for _, rv := range arr(eo.Get2("relationships")) {
			ro := rv.AsObject()
			if ro == nil {
				continue
			}
			to := ro.GetString("entity")
			if to == "" {
				continue
			}
			m.Relationships = append(m.Relationships, Relationship{From: ename, To: to, Cardinality: ro.GetString("cardinality")})
		}
		for _, iv := range arr(eo.Get2("invariants")) {
			if io := iv.AsObject(); io != nil {
				if id := io.GetString("id"); id != "" {
					m.Invariants = append(m.Invariants, Invariant{ID: id, OwnerKind: "entity", Owner: ename, Statement: io.GetString("statement")})
				}
			}
		}
		m.Entities = append(m.Entities, e)
	}
	return m, nil
}

// InvariantIDs returns the set of every invariant id in the model (top and
// entity), for reconciling coverage claims and residuals.
func (m *Model) InvariantIDs() map[string]bool {
	ids := make(map[string]bool, len(m.Invariants))
	for _, iv := range m.Invariants {
		ids[iv.ID] = true
	}
	return ids
}
