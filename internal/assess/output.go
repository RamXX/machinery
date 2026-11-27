package assess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/v2/internal/gen"
	"github.com/RamXX/machinery/v2/internal/report"
)

// The report of section 18.9. Struct fields are declared in key order, so
// that every object of report.json has its keys sorted.

type jsonCount struct {
	N    int    `json:"n"`
	Noun string `json:"noun"`
}

type jsonRule struct {
	Count    int    `json:"count"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
}

type jsonStage struct {
	Checked  []jsonCount `json:"checked"`
	Findings int         `json:"findings"`
	Name     string      `json:"name"`
	Output   string      `json:"output"`
	Reason   string      `json:"reason"`
	Result   string      `json:"result"`
}

type jsonSection struct {
	Checked  []jsonCount    `json:"checked"`
	Facts    map[string]any `json:"facts"`
	Findings int            `json:"findings"`
	Lines    []string       `json:"lines"`
	Name     string         `json:"name"`
	Output   string         `json:"output"`
	Reason   string         `json:"reason"`
	Result   string         `json:"result"`
	Rules    []jsonRule     `json:"rules"`
	Stages   []jsonStage    `json:"stages"`
}

type jsonDirty struct {
	Changed   int `json:"changed"`
	Untracked int `json:"untracked"`
}

type jsonSummary struct {
	Failed          int `json:"failed"`
	NotApplicable   int `json:"not_applicable"`
	NotDemonstrated int `json:"not_demonstrated"`
	Passed          int `json:"passed"`
	Sections        int `json:"sections"`
}

// jsonMachine is the findings that name one machine.
type jsonMachine struct {
	Drift    int        `json:"drift"`
	Errors   int        `json:"errors"`
	Machine  string     `json:"machine"`
	Rules    []jsonRule `json:"rules"`
	Source   string     `json:"source"`
	Warnings int        `json:"warnings"`
}

// jsonEntity is the findings that name one entity.
type jsonEntity struct {
	Drift    int        `json:"drift"`
	Entity   string     `json:"entity"`
	Errors   int        `json:"errors"`
	Rules    []jsonRule `json:"rules"`
	Warnings int        `json:"warnings"`
}

type jsonReport struct {
	Config   string        `json:"config"`
	Dirty    jsonDirty     `json:"dirty"`
	Entities []jsonEntity  `json:"entities"`
	Head     string        `json:"head"`
	Machines []jsonMachine `json:"machines"`
	Outside  []string      `json:"outside_links"`
	Project  string        `json:"project"`
	Sections []jsonSection `json:"sections"`
	Summary  jsonSummary   `json:"summary"`
	Version  int           `json:"version"`
}

func jsonRules(rs []rule) []jsonRule {
	res := []jsonRule{}
	for _, ru := range rs {
		res = append(res, jsonRule{ru.count, ru.key, sevWord(ru.sev)})
	}
	return res
}

func counts(cs []report.Count) []jsonCount {
	res := []jsonCount{}
	for _, c := range cs {
		res = append(res, jsonCount{c.N, c.Noun})
	}
	return res
}

func stageResult(st *report.Stage) (string, string) {
	switch o := st.Outcome(); o {
	case report.NotApplicable:
		return outcomeWord(o), st.NA
	case report.NotDemonstrated:
		if st.ND == "" {
			return outcomeWord(o), "nothing to check"
		}
		return outcomeWord(o), st.ND
	default:
		return outcomeWord(o), ""
	}
}

func (a *assessment) build() jsonReport {
	r := jsonReport{Config: a.cfgLabel, Dirty: jsonDirty{a.changed, a.untrack}, Head: a.head,
		Outside: append([]string{}, a.outside...), Project: a.name, Sections: []jsonSection{}, Version: 1,
		Machines: []jsonMachine{}, Entities: []jsonEntity{}}
	machines, entities := a.groups()
	for _, g := range machines {
		r.Machines = append(r.Machines, jsonMachine{g.drift, g.errors, g.name, jsonRules(g.rules()), g.source, g.warns})
	}
	for _, g := range entities {
		r.Entities = append(r.Entities, jsonEntity{g.drift, g.name, g.errors, jsonRules(g.rules()), g.warns})
	}
	for _, s := range a.sections {
		o, reason := s.outcome()
		js := jsonSection{Checked: counts(s.checked()), Facts: s.facts, Findings: s.findings(), Lines: s.lines,
			Name: s.name, Output: s.output(), Reason: reason, Result: outcomeWord(o), Rules: []jsonRule{}, Stages: []jsonStage{}}
		if js.Lines == nil {
			js.Lines = []string{}
		}
		if js.Facts == nil {
			js.Facts = map[string]any{}
		}
		for _, ru := range s.rules {
			js.Rules = append(js.Rules, jsonRule{ru.count, ru.key, sevWord(ru.sev)})
		}
		for _, st := range s.stages {
			res, why := stageResult(st)
			js.Stages = append(js.Stages, jsonStage{counts(st.Checked), st.Blocking(), st.Name, "stages/" + st.Name + ".txt", why, res})
		}
		switch o {
		case report.Passed:
			r.Summary.Passed++
		case report.Failed:
			r.Summary.Failed++
		case report.NotDemonstrated:
			r.Summary.NotDemonstrated++
		default:
			r.Summary.NotApplicable++
		}
		r.Sections = append(r.Sections, js)
	}
	r.Summary.Sections = len(r.Sections)
	return r
}

// markdown renders the same content for people.
func (a *assessment) markdown(r jsonReport) string {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	w("# machinery assess: %s at %s", r.Project, r.Head)
	w("")
	w("- config: %s", r.Config)
	if line := a.dirtyLine(); line != "" {
		w("- %s", line)
	} else {
		w("- dirty: none; the work tree matches HEAD")
	}
	if line := a.outsideLine(); line != "" {
		w("- %s", line)
	} else {
		w("- outside links: none")
	}
	w("- %s", a.totalLine())
	w("")
	w("| Section | Result | Findings |")
	w("|---|---|---|")
	for i, s := range a.sections {
		w("| %s | %s | %d |", s.name, s.outcomeLine(), r.Sections[i].Findings)
	}
	for i, s := range a.sections {
		js := r.Sections[i]
		w("")
		w("## %s", s.name)
		w("")
		w("result: %s", s.outcomeLine())
		if len(js.Checked) > 0 {
			w("checked: %s", countsText(s.checked()))
		}
		if js.Output != "" {
			w("output: %s", js.Output)
		}
		if len(js.Lines) > 0 {
			w("")
			w("```")
			for _, l := range js.Lines {
				w("%s", l)
			}
			w("```")
		}
		if len(js.Stages) > 1 {
			w("")
			w("| Stage | Result | Checked | Findings | Output |")
			w("|---|---|---|---|---|")
			for j, st := range js.Stages {
				res := st.Result
				if st.Reason != "" {
					res += ": " + st.Reason
				}
				w("| %s | %s | %s | %d | %s |", st.Name, res, countsText(s.stages[j].Checked), st.Findings, st.Output)
			}
		}
		if len(js.Rules) > 0 {
			w("")
			w("Findings per rule:")
			w("")
			for _, ru := range js.Rules {
				w("- %d %s: %s", ru.Count, ru.Severity, ru.Rule)
			}
		}
	}
	group := func(title, none string, heads []string, rules [][]jsonRule) {
		w("")
		w("## %s", title)
		w("")
		if len(heads) == 0 {
			w("%s", none)
			return
		}
		for i, h := range heads {
			w("- %s", h)
			for _, ru := range rules[i] {
				w("  - %d %s: %s", ru.Count, ru.Severity, ru.Rule)
			}
		}
	}
	var heads []string
	var rules [][]jsonRule
	for _, m := range r.Machines {
		heads = append(heads, fmt.Sprintf("%s (%s): %s", m.Machine, m.Source, tally(m.Errors, m.Drift, m.Warnings)))
		rules = append(rules, m.Rules)
	}
	group("Findings per machine", "No finding names a machine.", heads, rules)
	heads, rules = nil, nil
	for _, e := range r.Entities {
		heads = append(heads, fmt.Sprintf("%s: %s", e.Entity, tally(e.Errors, e.Drift, e.Warnings)))
		rules = append(rules, e.Rules)
	}
	group("Findings per entity", "No finding names an entity.", heads, rules)
	return b.String()
}

// tally is the counts of a group, by severity.
func tally(errors, drift, warns int) string {
	return fmt.Sprintf("%d error(s), %d drift, %d warn(s)", errors, drift, warns)
}

func (a *assessment) writeReports() error {
	r := a.build()
	data, err := gen.JSON(r)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(a.out, "report.json"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.out, "report.md"), []byte(a.markdown(r)), 0o644)
}
