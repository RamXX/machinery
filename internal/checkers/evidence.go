package checkers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	"github.com/RamXX/machinery/v2/internal/project"
)

// Finding is one finding of a checker (section 22.4).
type Finding struct {
	Severity string `json:"severity"`
	Subject  string `json:"subject"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

// ObligationDecl is an obligation a checker declares (section 22.5).
type ObligationDecl struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Subject string `json:"subject"`
}

// Evidence is the document a checker writes.
type Evidence struct {
	Version     int              `json:"version"`
	Checker     string           `json:"checker"`
	InputSHA256 string           `json:"input_sha256"`
	Verdict     string           `json:"verdict"`
	Covers      []string         `json:"covers"`
	Findings    []Finding        `json:"findings"`
	Obligations []ObligationDecl `json:"obligations"`
	Details     json.RawMessage  `json:"details"`
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

var requiredKeys = []string{"version", "checker", "input_sha256", "verdict", "covers", "findings"}
var knownKeys = map[string]bool{"version": true, "checker": true, "input_sha256": true, "verdict": true,
	"covers": true, "findings": true, "obligations": true, "details": true}

// Parse validates an evidence document for the checker id against the
// projection (its digest and its ids). stale is true when the document is well
// formed but bound to another projection; then the ids are not checked, since
// the design they came from is gone. problems is non-empty for invalid evidence.
func Parse(data []byte, id, sha string, ids map[string]bool) (ev *Evidence, stale bool, problems []string) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false, []string{"not a JSON object: " + err.Error()}
	}
	var keys []string
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !knownKeys[k] {
			problems = append(problems, fmt.Sprintf("unknown key %q", k))
		}
	}
	for _, k := range requiredKeys {
		if _, ok := raw[k]; !ok {
			problems = append(problems, fmt.Sprintf("missing key %q", k))
		}
	}
	if len(problems) > 0 {
		return nil, false, problems
	}
	var e Evidence
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return nil, false, []string{err.Error()}
	}
	add := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }
	if e.Version != 1 {
		add("version must be 1")
	}
	if e.Checker != id {
		add("checker is %q, want %q", e.Checker, id)
	}
	if !shaRe.MatchString(e.InputSHA256) {
		add("input_sha256 must be a lowercase sha256 hex digest")
	}
	errs := 0
	for i, f := range e.Findings {
		switch f.Severity {
		case "error":
			errs++
		case "warn":
		default:
			add("finding %d: severity must be error or warn, got %q", i+1, f.Severity)
		}
		if f.Subject == "" {
			add("finding %d: subject is required", i+1)
		}
		if f.Message == "" {
			add("finding %d: message is required", i+1)
		}
	}
	switch e.Verdict {
	case "pass":
		if errs > 0 {
			add("verdict is pass but a finding has severity error")
		}
	case "fail":
		if errs == 0 {
			add("verdict is fail but no finding has severity error")
		}
	default:
		add("verdict must be pass or fail, got %q", e.Verdict)
	}
	seen := map[string]bool{}
	for i, o := range e.Obligations {
		switch {
		case !nameRe.MatchString(o.Name):
			add("obligation %d: name %q must match ^[a-z][a-z0-9-]+$", i+1, o.Name)
		case seen[o.Name]:
			add("obligation %d: duplicate name %q", i+1, o.Name)
		}
		seen[o.Name] = true
		if o.Title == "" {
			add("obligation %d: title is required", i+1)
		}
		if o.Subject == "" {
			add("obligation %d: subject is required", i+1)
		}
	}
	if len(problems) > 0 {
		return nil, false, problems
	}
	if e.InputSHA256 != sha {
		return &e, true, nil
	}
	known := func(s string) bool { return s == System || ids[s] }
	for i, f := range e.Findings {
		if !known(f.Subject) {
			add("finding %d: subject %q is not an id of the projection", i+1, f.Subject)
		}
	}
	for i, o := range e.Obligations {
		if !known(o.Subject) {
			add("obligation %d: subject %q is not an id of the projection", i+1, o.Subject)
		}
	}
	for _, c := range e.Covers {
		if !ids[c] {
			add("covers: %q is not an id of the projection", c)
		}
	}
	if len(problems) > 0 {
		return nil, false, problems
	}
	return &e, false, nil
}

// Status values of the evidence of one checker.
const (
	Missing = "missing"
	Invalid = "invalid"
	Stale   = "stale"
	Pass    = "pass"
	Fail    = "fail"
)

// Assessment is what is known about the committed evidence of one checker.
type Assessment struct {
	Checker  project.Checker
	Status   string
	Evidence *Evidence // set unless the status is missing or invalid
	Problems []string  // set when the status is invalid
}

// Assess reads and validates the committed evidence of a checker.
func Assess(p *project.Project, ck project.Checker, sha string, ids map[string]bool) Assessment {
	a := Assessment{Checker: ck}
	if !p.Exists(ck.Evidence) {
		a.Status = Missing
		return a
	}
	data, err := p.Read(ck.Evidence)
	if err != nil {
		a.Status, a.Problems = Invalid, []string{"cannot read: " + lastPart(err)}
		return a
	}
	ev, stale, problems := Parse(data, ck.ID, sha, ids)
	switch {
	case len(problems) > 0:
		a.Status, a.Problems = Invalid, problems
	case stale:
		a.Status, a.Evidence = Stale, ev
	default:
		a.Status, a.Evidence = ev.Verdict, ev
	}
	return a
}
