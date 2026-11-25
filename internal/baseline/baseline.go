// Package baseline reads and writes the baseline files of the arch and tests
// stages (docs/formats.md sections 9 and 9.4) and applies them to violations.
package baseline

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
	"github.com/RamXX/machinery/internal/report"
)

// Violation is one finding of a static stage. Rule and Subject identify it in
// a baseline file; Message is what the stage prints.
type Violation struct {
	Rule    string
	Subject string
	Message string
}

// Entry is one line of a baseline file.
type Entry struct {
	Rule    string `yaml:"rule"`
	Subject string `yaml:"subject"`
}

type file struct {
	Version  int     `yaml:"version"`
	Baseline []Entry `yaml:"baseline"`
}

// Load reads a baseline file. A missing file, or an empty path, is an empty
// baseline. Problems are messages for the stage to report as ERROR.
func Load(p *project.Project, rel string) (entries []Entry, problems []string) {
	if rel == "" || !p.Exists(rel) {
		return nil, nil
	}
	data, err := p.Read(rel)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read: %v", rel, err)}
	}
	var f file
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil {
		msg := strings.TrimPrefix(err.Error(), "yaml: unmarshal errors:\n  ")
		msg = strings.TrimPrefix(msg, "yaml: ")
		return nil, []string{fmt.Sprintf("%s: %s", rel, strings.ReplaceAll(msg, "\n  ", "; "))}
	}
	if f.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", rel))
	}
	seen := map[Entry]bool{}
	for i, e := range f.Baseline {
		switch {
		case e.Rule == "" || e.Subject == "":
			problems = append(problems, fmt.Sprintf("%s: entry %d: rule and subject are required", rel, i+1))
		case seen[e]:
			problems = append(problems, fmt.Sprintf("%s: duplicate entry: %s %s", rel, e.Rule, e.Subject))
		}
		seen[e] = true
	}
	return f.Baseline, problems
}

// Write replaces the baseline file with the given violations, sorted.
func Write(p *project.Project, rel string, vs []Violation) error {
	es := make([]Entry, 0, len(vs))
	seen := map[Entry]bool{}
	for _, v := range vs {
		e := Entry{v.Rule, v.Subject}
		if !seen[e] {
			seen[e] = true
			es = append(es, e)
		}
	}
	sort.Slice(es, func(i, j int) bool {
		if es[i].Rule != es[j].Rule {
			return es[i].Rule < es[j].Rule
		}
		return es[i].Subject < es[j].Subject
	})
	var b strings.Builder
	b.WriteString("version: 1\nbaseline:")
	if len(es) == 0 {
		b.WriteString(" []\n")
	} else {
		b.WriteString("\n")
		for _, e := range es {
			fmt.Fprintf(&b, "  - {rule: %s, subject: %s}\n", e.Rule, strconv.Quote(e.Subject))
		}
	}
	return p.WriteAtomic(rel, []byte(b.String()))
}

// Apply adds the findings of a stage: a violation in the baseline is a warn,
// any other is an ERROR, and a baseline entry no violation matches is a stale
// ERROR.
func Apply(s *report.Stage, entries []Entry, vs []Violation) {
	listed := map[Entry]bool{}
	for _, e := range entries {
		listed[e] = true
	}
	hit := map[Entry]bool{}
	for _, v := range vs {
		e := Entry{v.Rule, v.Subject}
		if listed[e] {
			hit[e] = true
			s.Warnf("baselined: %s", v.Message)
			continue
		}
		s.Errorf("%s", v.Message)
	}
	for _, e := range entries {
		if !hit[e] {
			s.Errorf("stale baseline entry: %s %s", e.Rule, e.Subject)
		}
	}
}
