package ir

// LoadMachineJSONStr parses a JSON string (for tests) preserving key order.
func LoadMachineJSONStr(label, src string) (*Value, error) {
	return LoadMachineJSONBytes(label, []byte(src))
}
