// Package testpolicy enforces the end-to-end-only rules on test files
// (docs/formats.md section 9.4).
package testpolicy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/baseline"
	"github.com/RamXX/machinery/internal/project"
)

// Violation is one finding of the tests stage.
type Violation = baseline.Violation

// Counts is what the stage looked at.
type Counts struct {
	TestFiles int // test files in scope or under an e2e root
	E2EFiles  int // files under the e2e roots
	Markers   int // test-double markers looked for
}

// DefaultFiles are the test-file patterns used when tests.files is empty.
var DefaultFiles = []string{
	"**/*_test.go",
	"**/*.{test,spec}.{js,jsx,ts,tsx,mjs,cjs}",
	"**/test_*.py",
	"**/*_test.py",
	"**/*_test.exs",
	"**/*_test.rs",
}

// DefaultMarkers are the plain substrings that mark a test double.
var DefaultMarkers = []string{
	"gomock", "testify/mock", "sqlmock", "httpmock", "unittest.mock", "MagicMock",
	"jest.mock(", "vi.mock(", "sinon", "nock(", "Mox.", "mockall",
}

// Policy returns the test-file patterns and markers in force.
func Policy(p *project.Project) (files, markers []string) {
	files, markers = DefaultFiles, DefaultMarkers
	if t := p.Cfg.Tests; t != nil {
		if len(t.Files) > 0 {
			files = t.Files
		}
		if len(t.Doubles) > 0 {
			markers = t.Doubles
		}
	}
	return files, markers
}

// Applies reports whether the project configures the policy: a tests block or
// trace.tests.
func Applies(p *project.Project) bool {
	return p.Cfg.Tests != nil || (p.Cfg.Trace != nil && len(p.Cfg.Trace.Tests) > 0)
}

// BaselinePath is the configured baseline file, or "".
func BaselinePath(p *project.Project) string {
	if p.Cfg.Tests == nil {
		return ""
	}
	return p.Cfg.Tests.Baseline
}

// Violations applies both rules. The result is sorted by rule, then subject.
func Violations(p *project.Project) ([]Violation, Counts, error) {
	patterns, markers := Policy(p)
	roots := p.TestRoots()
	all, err := p.Files()
	if err != nil {
		return nil, Counts{}, err
	}
	c := Counts{Markers: len(markers)}
	var vs []Violation
	for _, f := range all {
		underRoot := project.Match(roots, f)
		if underRoot {
			c.E2EFiles++
		}
		if !project.Match(patterns, f) || !(underRoot || p.InScope(f)) {
			continue
		}
		c.TestFiles++
		if !underRoot {
			vs = append(vs, Violation{Rule: "unit-test", Subject: f, Message: "unit-test: " + f})
			continue
		}
		data, err := p.Read(f)
		if err != nil {
			return nil, c, err
		}
		for _, m := range markers {
			if strings.Contains(string(data), m) {
				subject := fmt.Sprintf("%s: %s", f, m)
				vs = append(vs, Violation{Rule: "test-double", Subject: subject, Message: "test-double: " + subject})
			}
		}
	}
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].Rule != vs[j].Rule {
			return vs[i].Rule < vs[j].Rule
		}
		return vs[i].Subject < vs[j].Subject
	})
	return vs, c, nil
}

// WriteBaseline records the violations in the configured baseline file.
func WriteBaseline(p *project.Project, vs []Violation) error {
	rel := BaselinePath(p)
	if rel == "" {
		return project.Errf("tests.baseline is not set: --write-baseline needs a baseline file in machinery.yaml")
	}
	return baseline.Write(p, rel, vs)
}
