package checkers

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Fact is one declared fact (docs/formats.md section 22.2).
type Fact struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Value     any    `json:"value"`
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)

// System is the subject of a fact or a finding that is about the whole system.
const System = "system"

// LoadFacts reads the facts file. ids is the set of ids of the projection; a
// subject must be one of them or the word system. When the file is invalid the
// problems are returned and no fact is.
func LoadFacts(p *project.Project, ids map[string]bool) (facts []Fact, problems []string) {
	rel := p.Cfg.Design.Facts
	if rel == "" {
		return nil, nil
	}
	data, err := p.Read(rel)
	if err != nil {
		return nil, []string{"cannot read: " + lastPart(err)}
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []string{"not valid YAML: " + strings.TrimPrefix(err.Error(), "yaml: ")}
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, []string{"must be a mapping with version and facts"}
	}
	var list *yaml.Node
	version := false
	for _, kv := range pairs(doc.Content[0]) {
		switch kv[0].Value {
		case "version":
			version = true
			if kv[1].Value != "1" {
				problems = append(problems, "version must be 1")
			}
		case "facts":
			list = kv[1]
		default:
			problems = append(problems, fmt.Sprintf("unknown key %q", kv[0].Value))
		}
	}
	if !version {
		problems = append(problems, "version is required")
	}
	if list == nil {
		problems = append(problems, "facts is required")
	} else if list.Kind != yaml.SequenceNode {
		problems = append(problems, "facts must be a list")
	}
	if list != nil && list.Kind == yaml.SequenceNode {
		first := map[string]int{}
		for i, n := range list.Content {
			f, probs := parseFact(i+1, n, ids)
			problems = append(problems, probs...)
			if len(probs) > 0 {
				continue
			}
			key := f.Subject + " " + f.Predicate
			if j, dup := first[key]; dup {
				problems = append(problems, fmt.Sprintf("fact %d: duplicate of fact %d (subject %s, predicate %s)", i+1, j, f.Subject, f.Predicate))
				continue
			}
			first[key] = i + 1
			facts = append(facts, f)
		}
	}
	if len(problems) > 0 {
		return nil, problems
	}
	sort.SliceStable(facts, func(i, j int) bool {
		if facts[i].Subject != facts[j].Subject {
			return facts[i].Subject < facts[j].Subject
		}
		return facts[i].Predicate < facts[j].Predicate
	})
	return facts, nil
}

func pairs(n *yaml.Node) [][2]*yaml.Node {
	var res [][2]*yaml.Node
	for i := 0; i+1 < len(n.Content); i += 2 {
		res = append(res, [2]*yaml.Node{n.Content[i], n.Content[i+1]})
	}
	return res
}

func parseFact(no int, n *yaml.Node, ids map[string]bool) (Fact, []string) {
	var f Fact
	var probs []string
	bad := func(format string, a ...any) {
		probs = append(probs, fmt.Sprintf("fact %d: ", no)+fmt.Sprintf(format, a...))
	}
	if n.Kind != yaml.MappingNode {
		bad("must be a mapping with subject, predicate and value")
		return f, probs
	}
	var haveSubject, havePredicate, haveValue bool
	for _, kv := range pairs(n) {
		switch kv[0].Value {
		case "subject":
			haveSubject = true
			f.Subject = kv[1].Value
		case "predicate":
			havePredicate = true
			f.Predicate = kv[1].Value
		case "value":
			haveValue = true
			f.Value = scalar(kv[1])
		default:
			bad("unknown key %q", kv[0].Value)
		}
	}
	switch {
	case !haveSubject || f.Subject == "":
		bad("subject is required")
	case f.Subject != System && !ids[f.Subject]:
		bad("unknown subject %q; use an id of the projection or %s", f.Subject, System)
	}
	switch {
	case !havePredicate || f.Predicate == "":
		bad("predicate is required")
	case !nameRe.MatchString(f.Predicate):
		bad("predicate %q must match ^[a-z][a-z0-9-]+$", f.Predicate)
	}
	if !haveValue || f.Value == nil {
		bad("value must be a string, a number or a boolean")
	}
	return f, probs
}

// scalar decodes a YAML scalar to a string, a number or a boolean; anything
// else (a list, a mapping, null) is nil.
func scalar(n *yaml.Node) any {
	if n.Kind != yaml.ScalarNode {
		return nil
	}
	var v any
	if n.Decode(&v) != nil {
		return nil
	}
	switch x := v.(type) {
	case string, bool, int, int64, uint64, float64:
		return x
	}
	return nil
}

func lastPart(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}
