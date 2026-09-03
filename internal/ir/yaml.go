package ir

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// LoadYAML parses YAML preserving mapping key order, returning *Value (the same
// ordered representation JSON uses). This mirrors Python's dict insertion-order
// semantics, which the generators depend on (e.g. compose_gen iterates the
// aggregates map in source order to build the TLA+ variable list).
func LoadYAML(data []byte) (*Value, error) {
	var n yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&n); err != nil {
		return nil, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("ir: decode trailing YAML document: %w", err)
		}
		return nil, fmt.Errorf("ir: multiple YAML documents are not allowed")
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
		case "!!timestamp":
			// Dates in evidence and ledgers are contract strings, not host-local
			// time values. Preserve the canonical source spelling while still
			// rejecting non-core custom tags below.
			return StringValue(n.Value), nil
		default:
			return nil, fmt.Errorf("ir: unsupported YAML scalar tag %q at line %d", n.Tag, n.Line)
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
		keyLines := map[string]int{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			kn := n.Content[i]
			if kn.Kind != yaml.ScalarNode {
				return nil, fmt.Errorf("ir: non-scalar YAML mapping key at line %d", kn.Line)
			}
			k := kn.Value
			if first, exists := keyLines[k]; exists {
				return nil, fmt.Errorf("ir: duplicate YAML key %q at line %d (first defined at line %d)", k, kn.Line, first)
			}
			keyLines[k] = kn.Line
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
