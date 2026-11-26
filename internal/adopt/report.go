package adopt

import (
	"fmt"
	"io"
	"strings"

	"github.com/RamXX/machinery/internal/report"
)

func list(vs []string, none string) string {
	if len(vs) == 0 {
		return none
	}
	return strings.Join(vs, ", ")
}

// next is the suggested steps, one line each (section 17.2).
func next(res *Result, o Options) []string {
	var n []string
	if !o.Write {
		n = append(n, "machinery adopt --write  writes the files above")
	}
	n = append(n, "machinery check  holds new code in scope to the rules")
	switch {
	case len(res.Blocked) > 0:
		n = append(n, "fix the findings below: owed entries are written once model, machines and failure-modes pass")
	case res.Models == 0 && res.Machines == 0:
		n = append(n, "model the riskiest stateful resource under design/, then run machinery adopt --write again")
	case res.Backlog.Found:
		n = append(n, "machinery plan --apply  adds stories for the owed ids beside the existing ones")
	}
	n = append(n, "machinery adopt --paths <glob> --write  widens the scope when the team is ready")
	return n
}

// FactLines are the lines of the report that describe the project: scope,
// languages, code and test files, design sources.
func (r *Result) FactLines() []string {
	return []string{
		"scope: " + strings.Join(r.Scope, ", "),
		"languages: " + list(r.Languages, "none found"),
		fmt.Sprintf("code files: %d in scope", r.Counts.Code),
		fmt.Sprintf("test files: %d under e2e roots, %d elsewhere (unit tests), %d with test doubles",
			r.Counts.E2E, r.Counts.Unit, r.Counts.Doubles),
		fmt.Sprintf("design: %d model(s), %d machine(s), %d contract(s)", r.Models, r.Machines, r.Contracts),
	}
}

// Print writes the dry-run report, or after --write the files written.
func (r *Result) Print(w io.Writer) {
	if r.Write {
		fmt.Fprintf(w, "adopt: wrote %d file(s)\n", len(r.Written))
		for _, f := range r.Written {
			fmt.Fprintf(w, "  %s\n", f)
		}
	} else {
		fmt.Fprintf(w, "adopt: %s at %s\n", r.Name, r.Sha)
		for _, l := range r.FactLines() {
			fmt.Fprintln(w, l)
		}
		fmt.Fprintln(w, r.Backlog.Line())
		fmt.Fprintln(w, "would write:")
		for _, c := range r.Would {
			fmt.Fprintf(w, "  %s  %s\n", c.Path, c.What)
		}
		if len(r.Would) == 0 {
			fmt.Fprintln(w, "  nothing")
		}
		fmt.Fprintln(w, "next:")
		for _, n := range r.Next {
			fmt.Fprintf(w, "  %s\n", n)
		}
	}
	if lines := r.BlockedLines(); len(lines) > 0 {
		for _, l := range lines {
			fmt.Fprintln(w, l)
		}
	}
}

// BlockedLines is what the report says when generation is refused (section
// 17.2): exactly the findings that block, and that no waiver covers them.
func (r *Result) BlockedLines() []string {
	if len(r.Blocked) == 0 {
		return nil
	}
	var body []string
	errs, n := 0, 0
	for _, s := range r.Blocked {
		listed := false
		for _, f := range s.Sorted() {
			if f.Sev == report.Error {
				body = append(body, fmt.Sprintf("  %s: %s", s.Name, f.Msg))
				errs++
				n++
				listed = true
			}
		}
		if !listed && s.Outcome() == report.NotDemonstrated {
			reason := s.ND
			if reason == "" {
				reason = "nothing to check"
			}
			body = append(body, fmt.Sprintf("  %s: not demonstrated: %s", s.Name, reason))
			n++
		}
	}
	lines := append([]string{fmt.Sprintf("blocked: machinery gen refuses while %d finding(s) stand; no owed entry is written until they are fixed", n)}, body...)
	if errs > 0 {
		lines = append(lines, "fix in the project first: no waiver covers a finding of stage model, machines or failure-modes; only cross-reference rules are waived (section 6.7)")
	}
	return lines
}
