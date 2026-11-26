package design

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// FailureMode is one entry of failure-modes.yaml (section 5.4).
type FailureMode struct {
	ID       string `yaml:"id"`
	Kind     string `yaml:"kind,omitempty"`
	Subject  string `yaml:"subject"`
	Failure  string `yaml:"failure"`
	Required string `yaml:"required"`
	Behavior string `yaml:"behavior"`

	// Source is the file the failure mode comes from: the failure-mode file
	// or the imported catalog (section 5.4.1).
	Source string `yaml:"-"`
	// Imported is true for a row of the catalog; Linked when the links file
	// gives it a subject and a required value.
	Imported bool `yaml:"-"`
	Linked   bool `yaml:"-"`
}

// Unlinked reports an imported row without a link: it names nothing, so it
// takes part in no subject rule (section 5.4.1).
func (fm FailureMode) Unlinked() bool { return fm.Imported && !fm.Linked }

type failureFile struct {
	Version      int           `yaml:"version"`
	FailureModes []FailureMode `yaml:"failure_modes"`
}

var (
	kebabRe   = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
	subjectRe = regexp.MustCompile(`^(invoke:[^.\s]+\.[^\s]+|dependency:[^\s]+|act:[A-Za-z0-9_]+\.[A-Za-z0-9_]+|entity:[A-Za-z0-9_]+|state:[^.\s]+\.[^\s]+|event:[^.\s]+\.[^\s]+)$`)
)

func strictYAML(data []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		msg := strings.TrimPrefix(err.Error(), "yaml: unmarshal errors:\n  ")
		msg = strings.TrimPrefix(msg, "yaml: ")
		return fmt.Errorf("%s", strings.ReplaceAll(msg, "\n  ", "; "))
	}
	return nil
}

// LoadFailureModes parses and validates the failure-mode file. configured is
// false when the project has none.
func LoadFailureModes(p *project.Project) (fms []FailureMode, problems []string, configured bool) {
	rel := p.Cfg.Design.FailureModes
	if rel == "" {
		return nil, nil, false
	}
	data, err := p.Read(rel)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read: %s", rel, shortErr(err))}, true
	}
	var f failureFile
	if err := strictYAML(data, &f); err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", rel, err)}, true
	}
	if f.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", rel))
	}
	seen := map[string]bool{}
	for i := range f.FailureModes {
		fm := &f.FailureModes[i]
		where := fmt.Sprintf("%s: failure mode %d", rel, i+1)
		if fm.ID != "" {
			where = fmt.Sprintf("%s: %s", rel, fm.ID)
		}
		fm.Source = rel
		switch {
		case !kebabRe.MatchString(fm.ID):
			problems = append(problems, where+": id must match ^[a-z][a-z0-9-]+$")
		case seen[fm.ID]:
			problems = append(problems, where+": duplicate id")
		}
		seen[fm.ID] = true
		problems = append(problems, checkValues(where, fm)...)
		if strings.TrimSpace(fm.Failure) == "" {
			problems = append(problems, where+": failure is required")
		}
		if strings.TrimSpace(fm.Behavior) == "" {
			problems = append(problems, where+": behavior is required")
		}
	}
	return f.FailureModes, problems, true
}

// checkValues holds kind, subject and required to section 5.4, defaulting
// kind to fault. where prefixes every problem.
func checkValues(where string, fm *FailureMode) (problems []string) {
	if fm.Kind == "" {
		fm.Kind = "fault"
	}
	if fm.Kind != "fault" && fm.Kind != "abuse" {
		problems = append(problems, fmt.Sprintf("%s: kind must be fault or abuse, got %q", where, fm.Kind))
	}
	if !subjectRe.MatchString(fm.Subject) {
		problems = append(problems, fmt.Sprintf("%s: subject %q is not invoke:<machine id>.<src>, dependency:<name>, act:<Entity>.<action>, entity:<Name>, state:<machine id>.<State> or event:<machine id>.<event>", where, fm.Subject))
	}
	switch fm.Required {
	case "fail-closed", "retry-bounded", "surface":
	case "refuse":
		if fm.Kind != "abuse" {
			problems = append(problems, where+": required: refuse is only valid with kind: abuse")
		}
	default:
		problems = append(problems, fmt.Sprintf("%s: required must be fail-closed, retry-bounded, surface or refuse, got %q", where, fm.Required))
	}
	if fm.Kind == "abuse" && !strings.HasPrefix(fm.Subject, "act:") && !strings.HasPrefix(fm.Subject, "entity:") {
		problems = append(problems, where+": kind: abuse needs an act or entity subject")
	}
	return problems
}

// Tier names in rising order.
const (
	Routine       = "routine"
	Consequential = "consequential"
	Critical      = "critical"
	Vital         = "vital"
)

// TierRank orders tiers.
func TierRank(t string) int {
	switch t {
	case Vital:
		return 3
	case Critical:
		return 2
	case Consequential:
		return 1
	}
	return 0
}

// Modeled reports whether a tier asks for a formal model: critical and vital
// (sections 16 and 23).
func Modeled(tier string) bool { return TierRank(tier) >= TierRank(Critical) }

// Area is one entry of risk.yaml (section 23). Property, Core and Assumes
// belong to a vital area (section 23.1).
type Area struct {
	Name      string       `yaml:"name"`
	Tier      string       `yaml:"tier"`
	Why       string       `yaml:"why"`
	IDs       []string     `yaml:"ids"`
	Questions []string     `yaml:"questions,omitempty"`
	Property  []Property   `yaml:"property,omitempty"`
	Core      *Core        `yaml:"core,omitempty"`
	Assumes   []Assumption `yaml:"assumes,omitempty"`
}

// Lowering records a deliberate lowering of a tier, or with Area and What a
// weakening of what a vital area states (section 23).
type Lowering struct {
	IDs    []string `yaml:"ids,omitempty"`
	From   string   `yaml:"from,omitempty"`
	Area   string   `yaml:"area,omitempty"`
	What   string   `yaml:"what,omitempty"`
	Reason string   `yaml:"reason"`
}

type riskFile struct {
	Version  int         `yaml:"version"`
	Areas    []Area      `yaml:"areas"`
	Lowered  []Lowering  `yaml:"lowered,omitempty"`
	NotVital []Dismissal `yaml:"not_vital,omitempty"`
}

// ParseNotVital returns the dismissed candidates of risk file bytes (nil when
// the file does not parse; ParseRisk reports that).
func ParseNotVital(data []byte) []Dismissal {
	var f riskFile
	if strictYAML(data, &f) != nil {
		return nil
	}
	return f.NotVital
}

// ParseLowered returns the lowered entries of risk file bytes (nil when the
// file does not parse; ParseRisk reports that).
func ParseLowered(data []byte) []Lowering {
	var f riskFile
	if strictYAML(data, &f) != nil {
		return nil
	}
	return f.Lowered
}

// Questions maps each formal question to its tool (section 16.1).
var Questions = map[string]string{"ordering": "tla", "concurrency": "tla", "recovery": "tla", "structure": "alloy"}

// LoadRisk parses the risk file and checks the shape rules. Id resolution is
// the risk stage's job. configured is false when the project has none.
func LoadRisk(p *project.Project) (areas []Area, problems []string, configured bool) {
	rel := p.Cfg.Design.Risk
	if rel == "" {
		return nil, nil, false
	}
	data, err := p.Read(rel)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read: %s", rel, shortErr(err))}, true
	}
	return ParseRisk(rel, data)
}

// ParseRisk parses risk file bytes.
func ParseRisk(rel string, data []byte) (areas []Area, problems []string, configured bool) {
	var f riskFile
	if err := strictYAML(data, &f); err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", rel, err)}, true
	}
	if f.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", rel))
	}
	seen := map[string]bool{}
	for i, a := range f.Areas {
		where := fmt.Sprintf("%s: area %d", rel, i+1)
		if a.Name != "" {
			where = fmt.Sprintf("%s: %s", rel, a.Name)
		}
		switch {
		case !kebabRe.MatchString(a.Name):
			problems = append(problems, where+": name must match ^[a-z][a-z0-9-]+$")
		case seen[a.Name]:
			problems = append(problems, where+": duplicate name")
		}
		seen[a.Name] = true
		if a.Tier != Consequential && a.Tier != Critical && a.Tier != Vital {
			problems = append(problems, fmt.Sprintf("%s: tier must be consequential, critical or vital, got %q (an id in no area is routine)", where, a.Tier))
		}
		if strings.TrimSpace(a.Why) == "" {
			problems = append(problems, where+": why is required")
		}
		if len(a.IDs) == 0 {
			problems = append(problems, where+": ids needs at least one id pattern")
		}
		for _, q := range a.Questions {
			if Questions[q] == "" {
				problems = append(problems, fmt.Sprintf("%s: unknown question %q (ordering, concurrency, recovery, structure)", where, q))
			}
		}
		if Modeled(a.Tier) && len(a.Questions) == 0 {
			problems = append(problems, fmt.Sprintf("%s: a %s area lists at least one question", where, a.Tier))
		}
		if !Modeled(a.Tier) && len(a.Questions) > 0 {
			problems = append(problems, where+": only a critical or vital area lists questions")
		}
		problems = append(problems, vitalShape(where, a)...)
	}
	for i, l := range f.Lowered {
		where := fmt.Sprintf("%s: lowered entry %d", rel, i+1)
		vitalForm := l.Area != "" || l.What != ""
		switch {
		case vitalForm && (len(l.IDs) > 0 || l.From != ""):
			problems = append(problems, where+": an entry has ids and from, or area and what, not both")
		case vitalForm:
			if !kebabRe.MatchString(l.Area) {
				problems = append(problems, where+": area must match ^[a-z][a-z0-9-]+$")
			}
			if strings.TrimSpace(l.What) == "" {
				problems = append(problems, where+": what is required")
			}
		default:
			if len(l.IDs) == 0 {
				problems = append(problems, where+": ids needs at least one id pattern")
			}
			if l.From != Consequential && l.From != Critical && l.From != Vital {
				problems = append(problems, where+": from must be consequential, critical or vital")
			}
		}
		if strings.TrimSpace(l.Reason) == "" {
			problems = append(problems, where+": reason is required")
		}
	}
	problems = append(problems, notVitalShape(rel, f.NotVital)...)
	return f.Areas, problems, true
}

// MatchID reports whether an id matches a pattern (a full id or a prefix ending in *).
func MatchID(pattern, id string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(id, strings.TrimSuffix(pattern, "*"))
	}
	return pattern == id
}

// MatchAny reports whether an id matches any pattern.
func MatchAny(patterns []string, id string) bool {
	for _, pt := range patterns {
		if MatchID(pt, id) {
			return true
		}
	}
	return false
}

// TierOf returns the highest tier among the areas an id belongs to, and those areas.
func TierOf(areas []Area, id string) (tier string, in []Area) {
	tier = Routine
	for _, a := range areas {
		if MatchAny(a.IDs, id) {
			in = append(in, a)
			if TierRank(a.Tier) > TierRank(tier) {
				tier = a.Tier
			}
		}
	}
	return
}

// OwedEntry is one entry of owed.yaml (section 11.1).
type OwedEntry struct {
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
	Story  string `yaml:"story,omitempty"`
}

type owedFile struct {
	Version int         `yaml:"version"`
	Owed    []OwedEntry `yaml:"owed"`
}

// ParseOwed parses owed file bytes and checks shape rules (not id existence).
func ParseOwed(rel string, data []byte) (entries []OwedEntry, problems []string) {
	var f owedFile
	if err := strictYAML(data, &f); err != nil {
		return nil, []string{fmt.Sprintf("%s: %v", rel, err)}
	}
	if f.Version != 1 {
		problems = append(problems, fmt.Sprintf("%s: version must be 1", rel))
	}
	seen := map[string]bool{}
	for i, e := range f.Owed {
		switch {
		case e.ID == "":
			problems = append(problems, fmt.Sprintf("%s: entry %d: id is required", rel, i+1))
		case seen[e.ID]:
			problems = append(problems, fmt.Sprintf("%s: duplicate id %s", rel, e.ID))
		}
		seen[e.ID] = true
		if strings.TrimSpace(e.Reason) == "" {
			problems = append(problems, fmt.Sprintf("%s: %s: reason is required", rel, e.ID))
		}
	}
	return f.Owed, problems
}

// LoadOwed reads the owed file. A missing file is an empty list.
func LoadOwed(p *project.Project) (entries []OwedEntry, problems []string) {
	rel := p.Cfg.Design.Owed
	if !p.Exists(rel) {
		return nil, nil
	}
	data, err := p.Read(rel)
	if err != nil {
		return nil, []string{fmt.Sprintf("%s: cannot read: %s", rel, shortErr(err))}
	}
	return ParseOwed(rel, data)
}

// MarshalOwed renders the owed file with entries sorted by id.
func MarshalOwed(entries []OwedEntry) []byte {
	es := append([]OwedEntry(nil), entries...)
	sort.Slice(es, func(i, j int) bool { return es[i].ID < es[j].ID })
	var b strings.Builder
	b.WriteString("version: 1\nowed:")
	if len(es) == 0 {
		b.WriteString(" []\n")
		return []byte(b.String())
	}
	b.WriteString("\n")
	for _, e := range es {
		fmt.Fprintf(&b, "  - id: %s\n    reason: %s\n", e.ID, yamlQuote(e.Reason))
		if e.Story != "" {
			fmt.Fprintf(&b, "    story: %s\n", e.Story)
		}
	}
	return []byte(b.String())
}

func yamlQuote(s string) string {
	return "\"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", " ").Replace(s) + "\""
}

// WriteOwed writes the owed file.
func WriteOwed(p *project.Project, entries []OwedEntry) error {
	return p.WriteAtomic(p.Cfg.Design.Owed, MarshalOwed(entries))
}
