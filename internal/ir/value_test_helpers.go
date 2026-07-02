package ir

import (
	"bytes"
	"encoding/json"
)

// LoadMachineJSONStr parses a JSON string (for tests) preserving key order.
func LoadMachineJSONStr(label, src string) (*Value, error) {
	dec := json.NewDecoder(bytes.NewReader([]byte(src)))
	dec.UseNumber()
	v, err := orderedDecode(dec)
	if err != nil {
		return nil, err
	}
	_ = label
	return v, nil
}
