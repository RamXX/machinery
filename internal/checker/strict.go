package checker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// decodeStrictJSON enforces the closed JSON contracts used at the checker
// boundary. encoding/json otherwise accepts duplicate object keys (last value
// wins), unknown struct fields, and a second trailing document.
func decodeStrictJSON(data []byte, dst any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON documents are not allowed")
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := walkJSONValue(dec, "$", nil); err != nil {
		return err
	}
	if tok, err := dec.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON documents are not allowed (second document starts with %v)", tok)
		}
		return fmt.Errorf("trailing JSON data: %w", err)
	}
	return nil
}

func walkJSONValue(dec *json.Decoder, location string, first json.Token) error {
	tok := first
	var err error
	if tok == nil {
		tok, err = dec.Token()
		if err != nil {
			return err
		}
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%s: object key is not a string", location)
			}
			if seen[key] {
				return fmt.Errorf("%s: duplicate JSON key %q", location, key)
			}
			seen[key] = true
			if err := walkJSONValue(dec, location+"."+key, nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return fmt.Errorf("%s: malformed JSON object", location)
		}
	case '[':
		for i := 0; dec.More(); i++ {
			if err := walkJSONValue(dec, fmt.Sprintf("%s[%d]", location, i), nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return fmt.Errorf("%s: malformed JSON array", location)
		}
	default:
		return fmt.Errorf("%s: unexpected JSON delimiter %q", location, delim)
	}
	return nil
}

// rejectInvalidYAMLMappingKeys rejects ambiguous mappings before decoding a
// manifest into Go structs. YAML otherwise permits duplicate and non-scalar
// keys whose resolution varies by consumer.
func rejectInvalidYAMLMappingKeys(node *yaml.Node, location string) error {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.AliasNode {
		return rejectInvalidYAMLMappingKeys(node.Alias, location)
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return fmt.Errorf("%s: mapping key at line %d must be a string scalar", location, key.Line)
			}
			if seen[key.Value] {
				return fmt.Errorf("%s: duplicate YAML key %q at line %d", location, key.Value, key.Line)
			}
			seen[key.Value] = true
			if err := rejectInvalidYAMLMappingKeys(node.Content[i+1], location+"."+key.Value); err != nil {
				return err
			}
		}
		return nil
	}
	for i, child := range node.Content {
		if err := rejectInvalidYAMLMappingKeys(child, fmt.Sprintf("%s[%d]", location, i)); err != nil {
			return err
		}
	}
	return nil
}
