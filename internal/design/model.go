// Package design reads the hand-written design sources and derives the id
// inventory (docs/formats.md sections 4, 5 and 7.1).
package design

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// Model is what machinery extracts from one Modelith file. Validation and
// rendering belong to the modelith executable.
type Model struct {
	Path       string
	Title      string
	Enums      []Enum
	Entities   []Entity
	Invariants []Invariant // top-level
	Scenarios  []Scenario
}

type Enum struct {
	Name   string
	Values []string
}

type Entity struct {
	Name          string
	Definition    string
	SubtypeOf     string
	Derived       bool
	Attributes    []Attribute
	Relationships []Relationship
	Actions       []Action
	Invariants    []Invariant
}

type Attribute struct {
	Name string
	Type string
}

type Relationship struct {
	Entity      string
	Cardinality string
	Role        string
	Ownership   string // owned | referenced
	Symmetric   bool
}

type Action struct {
	Name        string
	Actor       string
	Preserves   []string
	Description string
}

type Invariant struct {
	ID        string
	Statement string
	Line      int // of the statement, 1-based
}

type Scenario struct {
	Name       string
	Actors     []string
	Invariants []string
	Steps      []string
}

func mapPairs(n *yaml.Node) [][2]*yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	var res [][2]*yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		res = append(res, [2]*yaml.Node{n.Content[i], n.Content[i+1]})
	}
	return res
}

func child(n *yaml.Node, key string) *yaml.Node {
	for _, kv := range mapPairs(n) {
		if kv[0].Value == key {
			return kv[1]
		}
	}
	return nil
}

func str(n *yaml.Node, key string) string {
	if c := child(n, key); c != nil && c.Kind == yaml.ScalarNode {
		return c.Value
	}
	return ""
}

func strList(n *yaml.Node, key string) []string {
	c := child(n, key)
	if c == nil || c.Kind != yaml.SequenceNode {
		return nil
	}
	var res []string
	for _, e := range c.Content {
		if e.Kind == yaml.ScalarNode {
			res = append(res, e.Value)
		}
	}
	return res
}

func seq(n *yaml.Node, key string) []*yaml.Node {
	c := child(n, key)
	if c == nil || c.Kind != yaml.SequenceNode {
		return nil
	}
	return c.Content
}

func invariants(n *yaml.Node) []Invariant {
	var res []Invariant
	for _, e := range seq(n, "invariants") {
		inv := Invariant{ID: str(e, "id"), Statement: str(e, "statement"), Line: e.Line}
		if c := child(e, "statement"); c != nil {
			inv.Line = c.Line
		}
		res = append(res, inv)
	}
	return res
}

// ParseModel extracts ids and structure from a Modelith document.
func ParseModel(rel string, data []byte) (*Model, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("%s: %v", rel, err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: not a YAML mapping", rel)
	}
	root := doc.Content[0]
	m := &Model{Path: rel, Title: str(root, "title")}
	for _, kv := range mapPairs(child(root, "enums")) {
		e := Enum{Name: kv[0].Value}
		for _, v := range seq(kv[1], "values") {
			if v.Kind == yaml.ScalarNode {
				e.Values = append(e.Values, v.Value)
			} else {
				e.Values = append(e.Values, str(v, "name"))
			}
		}
		m.Enums = append(m.Enums, e)
	}
	for _, kv := range mapPairs(child(root, "entities")) {
		en := Entity{Name: kv[0].Value, Definition: str(kv[1], "definition"), SubtypeOf: str(kv[1], "subtypeOf"), Derived: str(kv[1], "derived") == "true"}
		for _, a := range seq(kv[1], "attributes") {
			en.Attributes = append(en.Attributes, Attribute{Name: str(a, "name"), Type: str(a, "type")})
		}
		for _, r := range seq(kv[1], "relationships") {
			own := str(r, "ownership")
			if own == "" {
				own = "referenced"
			}
			en.Relationships = append(en.Relationships, Relationship{Entity: str(r, "entity"), Cardinality: str(r, "cardinality"),
				Role: str(r, "role"), Ownership: own, Symmetric: str(r, "symmetric") == "true"})
		}
		for _, a := range seq(kv[1], "actions") {
			if a.Kind == yaml.ScalarNode {
				en.Actions = append(en.Actions, Action{Name: a.Value})
			} else {
				en.Actions = append(en.Actions, Action{Name: str(a, "name"), Actor: str(a, "actor"), Preserves: strList(a, "preserves"), Description: str(a, "description")})
			}
		}
		en.Invariants = invariants(kv[1])
		m.Entities = append(m.Entities, en)
	}
	m.Invariants = invariants(root)
	for _, s := range seq(root, "scenarios") {
		m.Scenarios = append(m.Scenarios, Scenario{Name: str(s, "name"), Actors: strList(s, "actors"), Invariants: strList(s, "invariants_touched"), Steps: strList(s, "steps")})
	}
	return m, nil
}

// LoadModels parses every configured model. Unreadable files are problems.
func LoadModels(p *project.Project) (models []*Model, problems []string) {
	for _, rel := range p.Cfg.Design.Models {
		data, err := p.Read(rel)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: cannot read: %v", rel, shortErr(err)))
			continue
		}
		m, err := ParseModel(rel, data)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		models = append(models, m)
	}
	return
}

func shortErr(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug is the scenario slug of section 4.
func Slug(name string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(name), "-"), "-")
}

// Entity returns the named entity of any model.
func FindEntity(models []*Model, name string) (*Entity, *Model) {
	for _, m := range models {
		for i := range m.Entities {
			if m.Entities[i].Name == name {
				return &m.Entities[i], m
			}
		}
	}
	return nil, nil
}

// ActID renders an action id.
func ActID(entity, action string) string { return "act:" + entity + "." + action }
