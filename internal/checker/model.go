package checker

import (
	"fmt"
	"os"
	"strings"

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
	Name        string
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
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: model must be a regular, non-symlink file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseModel(path, data)
}

func parseModel(path string, data []byte) (*Model, error) {
	root, err := ir.LoadYAML(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	obj := root.AsObject()
	if obj == nil {
		return nil, fmt.Errorf("%s: not a yaml mapping", path)
	}

	enums := map[string][]string{}
	if ev := obj.Get2("enums"); ev != nil {
		if ev.Kind != ir.KindObject {
			return nil, fmt.Errorf("%s: enums must be a mapping", path)
		}
		eo := ev.AsObject()
		for _, name := range eo.Keys() {
			var vals []string
			veo := eo.Get2(name)
			if veo == nil || veo.Kind != ir.KindObject {
				return nil, fmt.Errorf("%s: enum %q must be a mapping", path, name)
			}
			values := veo.AsObject().Get2("values")
			if values == nil || values.Kind != ir.KindArray {
				return nil, fmt.Errorf("%s: enum %q values must be an array", path, name)
			}
			seenValues := map[string]bool{}
			for i, vv := range values.AsArray() {
				vo := vv.AsObject()
				if vo == nil {
					return nil, fmt.Errorf("%s: enum %q values[%d] must be a mapping", path, name, i)
				}
				valueName := vo.GetString("name")
				if strings.TrimSpace(valueName) == "" {
					return nil, fmt.Errorf("%s: enum %q values[%d].name must be non-empty", path, name, i)
				}
				if seenValues[valueName] {
					return nil, fmt.Errorf("%s: enum %q repeats value %q", path, name, valueName)
				}
				seenValues[valueName] = true
				vals = append(vals, valueName)
			}
			enums[name] = vals
		}
	}

	m := &Model{}
	seenInvariants := map[string]bool{}
	if top := obj.Get2("invariants"); top != nil {
		if top.Kind != ir.KindArray {
			return nil, fmt.Errorf("%s: invariants must be an array", path)
		}
		for i, iv := range top.AsArray() {
			io := iv.AsObject()
			if io == nil {
				return nil, fmt.Errorf("%s: invariants[%d] must be a mapping", path, i)
			}
			id, statement := io.GetString("id"), io.GetString("statement")
			if strings.TrimSpace(id) == "" || strings.TrimSpace(statement) == "" {
				return nil, fmt.Errorf("%s: invariants[%d] requires non-empty id and statement", path, i)
			}
			if seenInvariants[id] {
				return nil, fmt.Errorf("%s: invariant id %q is declared more than once", path, id)
			}
			seenInvariants[id] = true
			m.Invariants = append(m.Invariants, Invariant{ID: id, OwnerKind: "top", Statement: statement})
		}
	}

	ents := obj.GetObject("entities")
	if ents == nil || ents.Len() == 0 {
		return nil, fmt.Errorf("%s: declares no entities", path)
	}
	for _, ename := range ents.Keys() {
		if strings.TrimSpace(ename) == "" {
			return nil, fmt.Errorf("%s: entity name must be non-empty", path)
		}
		ev := ents.Get2(ename)
		if ev == nil || ev.Kind != ir.KindObject {
			return nil, fmt.Errorf("%s: entity %q must be a mapping", path, ename)
		}
		eo := ev.AsObject()
		e := Entity{Name: ename}
		attributes := eo.Get2("attributes")
		if attributes != nil && attributes.Kind != ir.KindArray {
			return nil, fmt.Errorf("%s: entity %q attributes must be an array", path, ename)
		}
		seenAttrs := map[string]bool{}
		for i, av := range arr(attributes) {
			ao := av.AsObject()
			if ao == nil {
				return nil, fmt.Errorf("%s: entity %q attributes[%d] must be a mapping", path, ename, i)
			}
			name, typ := ao.GetString("name"), ao.GetString("type")
			if strings.TrimSpace(name) == "" || strings.TrimSpace(typ) == "" {
				return nil, fmt.Errorf("%s: entity %q attributes[%d] requires non-empty name and type", path, ename, i)
			}
			if seenAttrs[name] {
				return nil, fmt.Errorf("%s: entity %q repeats attribute %q", path, ename, name)
			}
			seenAttrs[name] = true
			e.Attributes = append(e.Attributes, Attr{Name: name, Type: typ})
			if isLifecycleAttr(name) {
				if vals, ok := enums[typ]; ok {
					e.StatusEnum = vals
				}
			}
		}
		relationships := eo.Get2("relationships")
		if relationships != nil && relationships.Kind != ir.KindArray {
			return nil, fmt.Errorf("%s: entity %q relationships must be an array", path, ename)
		}
		for i, rv := range arr(relationships) {
			ro := rv.AsObject()
			if ro == nil {
				return nil, fmt.Errorf("%s: entity %q relationships[%d] must be a mapping", path, ename, i)
			}
			to := ro.GetString("entity")
			cardinality := ro.GetString("cardinality")
			if strings.TrimSpace(to) == "" {
				return nil, fmt.Errorf("%s: entity %q relationships[%d].entity must be non-empty", path, ename, i)
			}
			switch cardinality {
			case "1:1", "1:n", "n:1", "n:m":
			default:
				return nil, fmt.Errorf("%s: entity %q relationships[%d] has unsupported cardinality %q", path, ename, i, cardinality)
			}
			name := ro.GetString("role")
			if name == "" {
				name = ro.GetString("name")
			}
			m.Relationships = append(m.Relationships, Relationship{From: ename, To: to, Cardinality: cardinality, Name: name})
		}
		invariants := eo.Get2("invariants")
		if invariants != nil && invariants.Kind != ir.KindArray {
			return nil, fmt.Errorf("%s: entity %q invariants must be an array", path, ename)
		}
		for i, iv := range arr(invariants) {
			io := iv.AsObject()
			if io == nil {
				return nil, fmt.Errorf("%s: entity %q invariants[%d] must be a mapping", path, ename, i)
			}
			id, statement := io.GetString("id"), io.GetString("statement")
			if strings.TrimSpace(id) == "" || strings.TrimSpace(statement) == "" {
				return nil, fmt.Errorf("%s: entity %q invariants[%d] requires non-empty id and statement", path, ename, i)
			}
			if seenInvariants[id] {
				return nil, fmt.Errorf("%s: invariant id %q is declared more than once", path, id)
			}
			seenInvariants[id] = true
			m.Invariants = append(m.Invariants, Invariant{ID: id, OwnerKind: "entity", Owner: ename, Statement: statement})
		}
		m.Entities = append(m.Entities, e)
	}
	entityNames := map[string]bool{}
	for _, entity := range m.Entities {
		entityNames[entity.Name] = true
	}
	for _, relationship := range m.Relationships {
		if !entityNames[relationship.To] {
			return nil, fmt.Errorf("%s: relationship %s -> %s names unknown target entity", path, relationship.From, relationship.To)
		}
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
