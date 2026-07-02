package ir

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// LoadYAML parses YAML preserving mapping key order, returning *Value (the same
// ordered representation JSON uses). This mirrors Python's dict insertion-order
// semantics, which the generators depend on (e.g. compose_gen iterates the
// aggregates map in source order to build the TLA+ variable list).
func LoadYAML(data []byte) (*Value, error) {
	var n yaml.Node
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, err
	}
	if n.Kind == 0 || n.Content == nil {
		return NullValue(), nil
	}
	return yamlNodeToValue(n.Content[0])
}

func yamlNodeToValue(n *yaml.Node) (*Value, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return NullValue(), nil
		}
		return yamlNodeToValue(n.Content[0])
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return StringValue(n.Value), nil
		case "!!int":
			return NumberValue(json.Number(n.Value)), nil
		case "!!float":
			return NumberValue(json.Number(n.Value)), nil
		case "!!bool":
			b := n.Value == "true" || n.Value == "True" || n.Value == "TRUE"
			return BoolValue(b), nil
		case "!!null":
			return NullValue(), nil
		default:
			// unknown tag: treat as string to be safe
			return StringValue(n.Value), nil
		}
	case yaml.SequenceNode:
		arr := make([]*Value, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := yamlNodeToValue(c)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return ArrayValue(arr), nil
	case yaml.MappingNode:
		o := NewObject()
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			v, err := yamlNodeToValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			o.Set(k, v)
		}
		return ObjectValue(o), nil
	}
	return nil, fmt.Errorf("ir: unsupported yaml node kind %d", n.Kind)
}

// (the jsonNumber helper was removed: YAML scalars map directly onto json.Number.)
