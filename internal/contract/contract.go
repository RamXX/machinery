// Package contract reads the operations of the configured API contracts
// (docs/formats.md sections 5.3 and 6.5). It does not validate OpenAPI or
// proto: it extracts what the cross-reference stage needs.
package contract

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Operation is one operation of a contract: an OpenAPI operation or a proto rpc.
type Operation struct {
	Contract string   // project-relative path of the contract file
	ID       string   // operationId, or the rpc name; "" when missing
	Method   string   // HTTP method in upper case, or "RPC"
	Path     string   // URL path, or "<Service>/<Rpc>"
	Actions  []string // bound actions, "<Entity>.<action>"
}

// Subject is how a finding and a waiver name the operation: "POST /orders".
func (o Operation) Subject() string { return o.Method + " " + o.Path }

// Load reads every configured contract. Problems are files that cannot be
// read or understood; they belong to the stage that reports them.
func Load(p *project.Project) (ops []Operation, problems []string) {
	for _, c := range p.Cfg.Contracts {
		data, err := p.Read(c.Path)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: cannot read: %s", c.Path, lastPart(err)))
			continue
		}
		var got []Operation
		var probs []string
		switch c.Kind {
		case "openapi":
			got, probs = parseOpenAPI(c.Path, data)
		case "proto":
			got, probs = parseProto(c.Path, data)
		}
		ops = append(ops, got...)
		problems = append(problems, probs...)
	}
	return ops, problems
}

func lastPart(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, ": "); i >= 0 {
		return s[i+2:]
	}
	return s
}

var httpMethods = []string{"get", "put", "post", "delete", "options", "head", "patch", "trace"}

func parseOpenAPI(rel string, data []byte) ([]Operation, []string) {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, []string{fmt.Sprintf("%s: not valid YAML or JSON: %s", rel, strings.TrimPrefix(err.Error(), "yaml: "))}
	}
	if v, _ := doc["openapi"].(string); !strings.HasPrefix(v, "3.") {
		return nil, []string{fmt.Sprintf("%s: not an OpenAPI 3.x document (the openapi key must be a 3.x version string)", rel)}
	}
	paths, _ := doc["paths"].(map[string]any)
	var keys []string
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var ops []Operation
	var problems []string
	for _, path := range keys {
		item, _ := paths[path].(map[string]any)
		for _, m := range httpMethods {
			raw, ok := item[m].(map[string]any)
			if !ok {
				continue
			}
			op := Operation{Contract: rel, Method: strings.ToUpper(m), Path: path}
			op.ID, _ = raw["operationId"].(string)
			acts, err := actionList(raw["x-machinery-action"])
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: %s: %v", rel, op.Subject(), err))
			}
			op.Actions = acts
			ops = append(ops, op)
		}
	}
	return ops, problems
}

// actionList accepts a string or a list of strings.
func actionList(v any) ([]string, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		return []string{x}, nil
	case []any:
		var res []string
		for _, e := range x {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("x-machinery-action must be a string or a list of strings")
			}
			res = append(res, s)
		}
		return res, nil
	}
	return nil, fmt.Errorf("x-machinery-action must be a string or a list of strings")
}
