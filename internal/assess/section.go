package assess

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/gitx"
	"github.com/RamXX/machinery/internal/report"
)

// sectionDef is one section of section 18.3, in order.
type sectionDef struct {
	name string
	run  func(a *assessment, s *section)
}

var sectionDefs = []sectionDef{
	{"inventory", (*assessment).inventory},
	{"design", (*assessment).design},
	{"generation", (*assessment).generation},
	{"trace", stageSection("trace")},
	{"tests", stageSection("tests")},
	{"arch", stageSection("arch")},
	{"formal", (*assessment).formal},
	{"history", (*assessment).history},
	{"decisions", (*assessment).decisions},
	{"backlog", (*assessment).backlog},
}

// section is the result of one section: either a list of stages, or a
// stage-like result of its own (own), with lines and facts.
type section struct {
	name   string
	own    *report.Stage
	stages []*report.Stage
	lines  []string
	facts  map[string]any
	text   string // the saved output of a section of its own, before its block
	rules  []rule
}

type rule struct {
	sev   report.Severity
	key   string
	count int
}

// run runs one section and saves its output. A crash fails the section;
// the others still run.
func (a *assessment) run(def sectionDef) *section {
	s := &section{name: def.name, own: report.New(def.name), facts: map[string]any{}}
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.own.Errorf("crashed: %v", r)
			}
		}()
		def.run(a, s)
	}()
	a.scrub(s)
	s.rules = a.rulesOf(s)
	a.save(s)
	return s
}

func (a *assessment) save(s *section) {
	dir := filepath.Join(a.out, "stages")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.own.Errorf("cannot save the output: %v", err)
		return
	}
	saved := map[string]bool{}
	for _, st := range s.stages {
		var b strings.Builder
		report.Print(&b, st)
		os.WriteFile(filepath.Join(dir, st.Name+".txt"), []byte(b.String()), 0o644)
		saved[st.Name] = true
	}
	if len(s.stages) > 0 && (saved[s.name] || len(s.own.Findings) == 0) {
		return
	}
	var b strings.Builder
	b.WriteString(s.text)
	for _, l := range s.lines {
		b.WriteString(l + "\n")
	}
	report.Print(&b, s.own)
	os.WriteFile(filepath.Join(dir, s.name+".txt"), []byte(b.String()), 0o644)
}

// scrub removes the absolute paths of the snapshot, the out directory and
// the project from what a section reports, so the report names files by
// relative path only (tool error lines often carry absolute ones).
func (a *assessment) scrub(s *section) {
	var pairs []string
	for _, d := range []struct{ path, name string }{{a.snap, "snapshot"}, {a.out, "out"}, {a.root, a.name}} {
		for _, p := range uniqStrings([]string{d.path, gitx.RealPath(d.path)}) {
			if p != "" {
				pairs = append(pairs, p+string(filepath.Separator), "", p, d.name)
			}
		}
	}
	r := strings.NewReplacer(pairs...)
	for i := range s.own.Findings {
		s.own.Findings[i].Msg = r.Replace(s.own.Findings[i].Msg)
	}
	for _, st := range s.stages {
		for i := range st.Findings {
			st.Findings[i].Msg = r.Replace(st.Findings[i].Msg)
		}
	}
	for i := range s.lines {
		s.lines[i] = r.Replace(s.lines[i])
	}
	s.text = r.Replace(s.text)
}

// output is the saved file of a section, relative to the out directory.
func (s *section) output() string {
	if len(s.stages) == 1 {
		return "stages/" + s.stages[0].Name + ".txt"
	}
	if len(s.stages) > 1 {
		return ""
	}
	return "stages/" + s.name + ".txt"
}

// outcome decides the result of a section (section 18.3).
func (s *section) outcome() (report.Outcome, string) {
	if len(s.stages) == 0 {
		o := s.own.Outcome()
		switch o {
		case report.NotApplicable:
			return o, s.own.NA
		case report.NotDemonstrated:
			if s.own.ND == "" {
				return o, "nothing to check"
			}
			return o, s.own.ND
		}
		return o, ""
	}
	if s.own.Blocking() > 0 {
		return report.Failed, ""
	}
	var nd, na []string
	passed := false
	for _, st := range s.stages {
		switch st.Outcome() {
		case report.Failed:
			return report.Failed, ""
		case report.NotDemonstrated:
			r := st.ND
			if r == "" {
				r = "nothing to check"
			}
			nd = append(nd, st.Name+": "+r)
		case report.Passed:
			passed = true
		case report.NotApplicable:
			na = append(na, st.NA)
		}
	}
	switch {
	case len(nd) > 0:
		if len(s.stages) == 1 {
			return report.NotDemonstrated, s.stages[0].ND
		}
		return report.NotDemonstrated, strings.Join(nd, "; ")
	case passed:
		return report.Passed, ""
	}
	return report.NotApplicable, strings.Join(uniqStrings(na), "; ")
}

// findings counts the blocking findings of a section.
func (s *section) findings() int {
	n := s.own.Blocking()
	for _, st := range s.stages {
		n += st.Blocking()
	}
	return n
}

// checked is the checked line of a section: the stage's own for a section
// of one stage, the number of stages for a longer list.
func (s *section) checked() []report.Count {
	switch {
	case len(s.stages) == 1:
		return s.stages[0].Checked
	case len(s.stages) > 1:
		return []report.Count{{N: len(s.stages), Noun: "stages"}}
	}
	return s.own.Checked
}

func outcomeWord(o report.Outcome) string {
	switch o {
	case report.Passed:
		return "passed"
	case report.Failed:
		return "failed"
	case report.NotApplicable:
		return "not applicable"
	}
	return "not demonstrated"
}

// outcomeLine is the last line a stage block would have (section 3.2).
func (s *section) outcomeLine() string {
	o, reason := s.outcome()
	switch o {
	case report.Passed:
		return "passed"
	case report.Failed:
		return fmt.Sprintf("failed: %d finding(s)", s.findings())
	}
	return outcomeWord(o) + ": " + reason
}

func countsText(cs []report.Count) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = fmt.Sprintf("%d %s", c.N, c.Noun)
	}
	return strings.Join(parts, ", ")
}

func (s *section) summaryLine() string {
	line := s.name + ": " + s.outcomeLine()
	if o, _ := s.outcome(); o != report.NotApplicable && len(s.checked()) > 0 {
		line += "; checked: " + countsText(s.checked())
	}
	return line
}

func (a *assessment) totalLine() string {
	var p, f, d, n int
	for _, s := range a.sections {
		switch o, _ := s.outcome(); o {
		case report.Passed:
			p++
		case report.Failed:
			f++
		case report.NotDemonstrated:
			d++
		default:
			n++
		}
	}
	return fmt.Sprintf("assess: %d sections, %d passed, %d failed, %d not demonstrated, %d not applicable", len(a.sections), p, f, d, n)
}

// The rule of a finding (section 18.4).
var (
	ruleWordRe = regexp.MustCompile(`^([a-z][a-z0-9]*(?:-[a-z0-9]+)*): `)
	plainRe    = regexp.MustCompile(`^_?[a-z][a-z0-9_-]*$`)
)

func ruleOf(stage, msg string, names map[string]bool) string {
	m := msg
	for {
		i := strings.Index(m, ": ")
		if i <= 0 {
			break
		}
		head := m[:i]
		if strings.Contains(head, " ") || (!strings.ContainsAny(head, "/.") && !names[head]) {
			break
		}
		m = m[i+2:]
	}
	if w := ruleWordRe.FindStringSubmatch(m); w != nil && !names[w[1]] {
		return stage + ": " + w[1]
	}
	if i := strings.Index(m, "; "); i >= 0 {
		m = m[:i]
	}
	if i := strings.Index(m, " ("); i >= 0 {
		m = m[:i]
	}
	var out []string
	for _, w := range strings.Fields(m) {
		core := strings.TrimRight(w, ",.:")
		tail := w[len(core):]
		if !plainRe.MatchString(core) || names[core] {
			core = "*"
		}
		if core == "*" && len(out) > 0 && strings.HasPrefix(out[len(out)-1], "*") && !strings.HasSuffix(out[len(out)-1], ":") {
			out[len(out)-1] = "*" + tail
			continue
		}
		out = append(out, core+tail)
	}
	return stage + ": " + strings.Join(out, " ")
}

// designNames are the names a design declares; in a rule they become *.
func designNames(d *design.Design) map[string]bool {
	n := map[string]bool{}
	for _, m := range d.Machines {
		n[m.ID], n[m.Base] = true, true
		t := m.Table
		for _, st := range t.States {
			n[st.Path] = true
			for _, seg := range strings.Split(st.Path, ".") {
				n[seg] = true
			}
		}
		for _, tr := range t.Transitions {
			n[tr.Event], n[tr.Guard] = true, true
		}
		for _, ig := range t.Ignores {
			n[ig.Event] = true
		}
		for _, lists := range [][]string{t.Units.Guards, t.Units.Actions, t.Units.Actors} {
			for _, u := range lists {
				n[u] = true
			}
		}
	}
	for _, md := range d.Models {
		for _, e := range md.Enums {
			n[e.Name] = true
			for _, v := range e.Values {
				n[v] = true
			}
		}
		for _, inv := range md.Invariants {
			n[inv.ID] = true
		}
		for _, e := range md.Entities {
			n[e.Name] = true
			for _, ac := range e.Actions {
				n[ac.Name] = true
			}
			for _, inv := range e.Invariants {
				n[inv.ID] = true
			}
		}
	}
	for _, fm := range d.FailureModes {
		n[fm.ID] = true
	}
	delete(n, "")
	return n
}

func (a *assessment) rulesOf(s *section) []rule {
	names := designNames(a.ctx.Design())
	type k struct {
		sev report.Severity
		key string
	}
	counts := map[k]int{}
	add := func(stage string, fs []report.Finding) {
		for _, f := range fs {
			counts[k{f.Sev, ruleOf(stage, f.Msg, names)}]++
		}
	}
	add(s.name, s.own.Findings)
	for _, st := range s.stages {
		add(st.Name, st.Findings)
	}
	var res []rule
	for key, n := range counts {
		res = append(res, rule{key.sev, key.key, n})
	}
	sort.Slice(res, func(i, j int) bool {
		if res[i].count != res[j].count {
			return res[i].count > res[j].count
		}
		if res[i].sev != res[j].sev {
			return res[i].sev < res[j].sev
		}
		return res[i].key < res[j].key
	})
	return res
}

func sevWord(s report.Severity) string {
	switch s {
	case report.Error:
		return "error"
	case report.Drift:
		return "drift"
	}
	return "warn"
}

func uniqStrings(items []string) []string {
	seen := map[string]bool{}
	var res []string
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			res = append(res, s)
		}
	}
	return res
}
