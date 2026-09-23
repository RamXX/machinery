package datalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ParseFacts parses one fact file in Soufflé's default input format: one
// tuple per line, columns separated by tabs. A final newline is optional, a
// trailing carriage return on a line is dropped, and an empty line is a
// one-column tuple holding the empty symbol. Every line must have the same
// number of columns. name labels errors.
func ParseFacts(data, name string) ([][]string, error) {
	if data == "" {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSuffix(data, "\n"), "\n")
	rows := make([][]string, 0, len(lines))
	for n, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if strings.ContainsRune(line, '\r') {
			return nil, fmt.Errorf("%s:%d: a carriage return inside a line is not supported", name, n+1)
		}
		row := strings.Split(line, "\t")
		if len(rows) > 0 && len(row) != len(rows[0]) {
			return nil, fmt.Errorf("%s:%d: %d columns, earlier lines have %d", name, n+1, len(row), len(rows[0]))
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// ReadFacts reads every <relation>.facts file directly inside dir.
func ReadFacts(dir string) (Inputs, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	in := Inputs{}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".facts") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, fn := range names {
		path := filepath.Join(dir, fn)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		rows, err := ParseFacts(string(data), path)
		if err != nil {
			return nil, err
		}
		in[strings.TrimSuffix(fn, ".facts")] = rows
	}
	return in, nil
}
