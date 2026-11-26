package assess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/gen"
	"github.com/RamXX/machinery/internal/report"
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

type jsonReport struct {
	Config   string        `json:"config"`
	Dirty    jsonDirty     `json:"dirty"`
	Head     string        `json:"head"`
	Project  string        `json:"project"`
	Sections []jsonSection `json:"sections"`
	Summary  jsonSummary   `json:"summary"`
	Version  int           `json:"version"`
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
		Project: a.name, Sections: []jsonSection{}, Version: 1}
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
	return b.String()
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
