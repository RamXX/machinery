// Package report prints stage blocks and computes exit codes (docs/formats.md section 3).
package report

import (
	"fmt"
	"io"
	"sort"
)

// Severity of a finding. Error and Drift block; Warn does not.
type Severity int

const (
	Error Severity = iota
	Drift
	Warn
)

// Finding is one line of a stage block.
type Finding struct {
	Sev Severity
	Msg string
}

// Count is one entry of the checked line.
type Count struct {
	N    int
	Noun string
}

// Outcome is how a stage ended.
type Outcome int

const (
	Passed Outcome = iota
	Failed
	NotApplicable
	NotDemonstrated
)

// Stage is the result of one check stage.
type Stage struct {
	Name     string
	Checked  []Count
	Findings []Finding
	NA       string // set: not applicable, names the absent feature
	ND       string // set: not demonstrated, names the reason
}

// New returns an empty stage result.
func New(name string) *Stage { return &Stage{Name: name} }

// Count appends an entry to the checked line.
func (s *Stage) Count(n int, noun string) *Stage {
	s.Checked = append(s.Checked, Count{n, noun})
	return s
}

// Errorf adds an ERROR finding.
func (s *Stage) Errorf(format string, a ...any) {
	s.Findings = append(s.Findings, Finding{Error, fmt.Sprintf(format, a...)})
}

// Driftf adds a DRIFT finding.
func (s *Stage) Driftf(format string, a ...any) {
	s.Findings = append(s.Findings, Finding{Drift, fmt.Sprintf(format, a...)})
}

// Warnf adds a warn finding.
func (s *Stage) Warnf(format string, a ...any) {
	s.Findings = append(s.Findings, Finding{Warn, fmt.Sprintf(format, a...)})
}

// NotApplicable marks the stage as not applicable and returns it.
func (s *Stage) NotApplicable(absent string) *Stage {
	s.NA = absent
	return s
}

// NotDemonstrated marks the stage as not demonstrated and returns it.
func (s *Stage) NotDemonstrated(reason string) *Stage {
	if s.ND == "" {
		s.ND = reason
	}
	return s
}

// Blocking counts ERROR plus DRIFT findings.
func (s *Stage) Blocking() int {
	n := 0
	for _, f := range s.Findings {
		if f.Sev != Warn {
			n++
		}
	}
	return n
}

// Count3 returns the number of findings per severity.
func (s *Stage) Count3() (errs, drift, warns int) {
	for _, f := range s.Findings {
		switch f.Sev {
		case Error:
			errs++
		case Drift:
			drift++
		default:
			warns++
		}
	}
	return
}

// Outcome computes how the stage ended.
func (s *Stage) Outcome() Outcome {
	switch {
	case s.Blocking() > 0:
		return Failed
	case s.NA != "":
		return NotApplicable
	case s.ND != "":
		return NotDemonstrated
	case len(s.Checked) == 0:
		return NotDemonstrated
	}
	return Passed
}

// OK reports whether the stage does not block.
func (s *Stage) OK() bool {
	o := s.Outcome()
	return o == Passed || o == NotApplicable
}

func label(sev Severity) string {
	switch sev {
	case Error:
		return "ERROR"
	case Drift:
		return "DRIFT"
	}
	return "warn "
}

// Print writes the stage block.
func Print(w io.Writer, s *Stage) {
	fmt.Fprintf(w, "== %s ==\n", s.Name)
	out := s.Outcome()
	if out == NotApplicable {
		fmt.Fprintf(w, "not applicable: %s\n", s.NA)
		return
	}
	if len(s.Checked) > 0 {
		fmt.Fprint(w, "checked: ")
		for i, c := range s.Checked {
			if i > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprintf(w, "%d %s", c.N, c.Noun)
		}
		fmt.Fprintln(w)
	}
	fs := append([]Finding(nil), s.Findings...)
	sort.SliceStable(fs, func(i, j int) bool {
		if fs[i].Sev != fs[j].Sev {
			return fs[i].Sev < fs[j].Sev
		}
		return fs[i].Msg < fs[j].Msg
	})
	for _, f := range fs {
		fmt.Fprintf(w, "  %s  %s\n", label(f.Sev), f.Msg)
	}
	switch out {
	case Passed:
		fmt.Fprintln(w, "passed")
	case Failed:
		fmt.Fprintf(w, "failed: %d finding(s)\n", s.Blocking())
	case NotDemonstrated:
		reason := s.ND
		if reason == "" {
			reason = "nothing to check"
		}
		fmt.Fprintf(w, "not demonstrated: %s\n", reason)
	}
}

// Summary writes the last line of a command.
func Summary(w io.Writer, command string, stages []*Stage) {
	var p, f, d, n int
	for _, s := range stages {
		switch s.Outcome() {
		case Passed:
			p++
		case Failed:
			f++
		case NotDemonstrated:
			d++
		case NotApplicable:
			n++
		}
	}
	fmt.Fprintf(w, "machinery %s: %d passed, %d failed, %d not demonstrated, %d not applicable\n", command, p, f, d, n)
}

// ExitCode implements section 3.3.
func ExitCode(stages []*Stage) int {
	for _, s := range stages {
		if !s.OK() {
			return 1
		}
	}
	return 0
}

// PrintAll prints every stage, the summary line, and returns the exit code.
func PrintAll(w io.Writer, command string, stages []*Stage) int {
	for _, s := range stages {
		Print(w, s)
	}
	Summary(w, command, stages)
	return ExitCode(stages)
}
