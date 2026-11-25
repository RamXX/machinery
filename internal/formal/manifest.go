// Package formal implements formal models and the checks that keep them sound
// (docs/formats.md section 16).
package formal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/design"
	"github.com/RamXX/machinery/internal/project"
)

// Suffix of a spec manifest.
const Suffix = ".spec.yaml"

// Manifest is <spec>.spec.yaml (section 16.2).
type Manifest struct {
	Version     int          `yaml:"version" json:"version"`
	Spec        string       `yaml:"spec" json:"spec"`
	Tool        string       `yaml:"tool" json:"tool"`
	Areas       []string     `yaml:"areas" json:"areas"`
	Questions   []string     `yaml:"questions" json:"questions"`
	Machine     string       `yaml:"machine,omitempty" json:"machine,omitempty"`
	Model       string       `yaml:"model,omitempty" json:"model,omitempty"`
	Properties  string       `yaml:"properties" json:"properties"`
	Sources     Sources      `yaml:"sources" json:"-"`
	Assumptions []Assumption `yaml:"assumptions,omitempty" json:"assumptions,omitempty"`
	Runs        []Run        `yaml:"runs" json:"runs"`
	Mutations   []Mutation   `yaml:"mutations,omitempty" json:"mutations,omitempty"`
}

// Sources is the sync block (section 16.7).
type Sources struct {
	ReadAt string     `yaml:"read_at"`
	Paths  []string   `yaml:"paths"`
	Log    []SyncNote `yaml:"log,omitempty"`
}

// SyncNote is one classification of source drift.
type SyncNote struct {
	At       string `yaml:"at"`
	From     string `yaml:"from"`
	Class    string `yaml:"class"`
	Evidence string `yaml:"evidence,omitempty"`
	Story    string `yaml:"story,omitempty"`
}

// Assumption is a numbered host assumption (section 16.3).
type Assumption struct {
	ID       string `yaml:"id" json:"id"`
	Text     string `yaml:"text" json:"text"`
	Status   string `yaml:"status" json:"status"`
	Evidence string `yaml:"evidence,omitempty" json:"evidence,omitempty"`
}

// Run is one configuration of the model (section 16.4).
type Run struct {
	Name       string         `yaml:"name" json:"name"`
	Constants  map[string]any `yaml:"constants,omitempty" json:"constants,omitempty"`
	Spec       string         `yaml:"spec,omitempty" json:"spec,omitempty"`
	Invariants []string       `yaml:"invariants,omitempty" json:"invariants,omitempty"`
	Properties []string       `yaml:"properties,omitempty" json:"properties,omitempty"`
	Reach      []string       `yaml:"reach,omitempty" json:"reach,omitempty"`
	Expect     string         `yaml:"expect" json:"expect"`
	Assumption string         `yaml:"assumption,omitempty" json:"assumption,omitempty"`
	Story      string         `yaml:"story,omitempty" json:"story,omitempty"`
	Timeout    int            `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}

// Mutation is a plausible bug the spec must catch (section 16.5).
type Mutation struct {
	ID      string `yaml:"id" json:"id"`
	Bug     string `yaml:"bug" json:"bug"`
	File    string `yaml:"file,omitempty" json:"file,omitempty"`
	From    string `yaml:"from" json:"from"`
	To      string `yaml:"to" json:"to"`
	Detects string `yaml:"detects" json:"detects"`
	Run     string `yaml:"run" json:"run"`
	Origin  string `yaml:"origin,omitempty" json:"origin,omitempty"`
}

// Spec is a loaded manifest with its resolved file names.
type Spec struct {
	Manifest
	Path     string   // project-relative manifest path
	Dir      string   // formal directory, no trailing slash
	Problems []string // manifest errors
}

var (
	specNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
	assumeRe   = regexp.MustCompile(`^A[0-9]+$`)
	mutRe      = regexp.MustCompile(`^M[0-9]+$`)
)

// Dir returns the formal directory without a trailing slash.
func Dir(p *project.Project) string { return strings.TrimSuffix(p.Cfg.Design.Formal, "/") }

// Ext is the file extension of the spec's tool.
func (s *Spec) Ext() string {
	if s.Tool == "alloy" {
		return ".als"
	}
	return ".tla"
}

// SkeletonName is the base name of the generated skeleton file.
func (s *Spec) SkeletonName() string {
	if s.Tool == "alloy" {
		base := path.Base(s.Model)
		for _, suf := range []string{".modelith.yaml", ".modelith.yml", ".yaml", ".yml"} {
			if strings.HasSuffix(base, suf) {
				base = strings.TrimSuffix(base, suf)
				break
			}
		}
		return base + ".als"
	}
	return s.Machine + ".tla"
}

// SkeletonPath is the project-relative path of the generated skeleton.
func (s *Spec) SkeletonPath() string { return s.Dir + "/" + s.SkeletonName() }

// PropsPath is the project-relative path of the property file.
func (s *Spec) PropsPath() string { return s.Dir + "/" + s.Properties + s.Ext() }

// ResultPath is the project-relative path of the result file.
func (s *Spec) ResultPath() string { return s.Dir + "/" + s.Spec + ".result.json" }

// TracePath is the project-relative path of a run's trace file.
func (s *Spec) TracePath(run string) string {
	return s.Dir + "/traces/" + s.Spec + "." + run + ".trace.json"
}

// OwnFiles lists the files that belong to the spec (never sources).
func (s *Spec) OwnFiles() []string {
	return []string{s.Path, s.SkeletonPath(), s.PropsPath(), s.ResultPath()}
}

// RunByName returns the named run.
func (s *Spec) RunByName(name string) *Run {
	for i := range s.Runs {
		if s.Runs[i].Name == name {
			return &s.Runs[i]
		}
	}
	return nil
}

// Load reads every manifest of the formal directory, sorted by path.
func Load(p *project.Project) ([]*Spec, error) {
	files, err := p.Glob([]string{Dir(p) + "/*" + Suffix})
	if err != nil {
		return nil, err
	}
	var specs []*Spec
	for _, f := range files {
		specs = append(specs, loadOne(p, f))
	}
	return specs, nil
}

func loadOne(p *project.Project, rel string) *Spec {
	s := &Spec{Path: rel, Dir: Dir(p)}
	stem := strings.TrimSuffix(path.Base(rel), Suffix)
	data, err := p.Read(rel)
	if err != nil {
		s.Spec = stem
		s.Problems = append(s.Problems, fmt.Sprintf("%s: cannot read", rel))
		return s
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s.Manifest); err != nil {
		msg := strings.TrimPrefix(err.Error(), "yaml: unmarshal errors:\n  ")
		s.Spec = stem
		s.Problems = append(s.Problems, fmt.Sprintf("%s: %s", rel, strings.ReplaceAll(strings.TrimPrefix(msg, "yaml: "), "\n  ", "; ")))
		return s
	}
	s.validate(stem)
	return s
}

func (s *Spec) bad(format string, a ...any) {
	s.Problems = append(s.Problems, s.Path+": "+fmt.Sprintf(format, a...))
}

func (s *Spec) validate(stem string) {
	if s.Version != 1 {
		s.bad("version must be 1")
	}
	if !specNameRe.MatchString(s.Spec) || s.Spec != stem {
		s.bad("spec must match ^[a-z][a-z0-9-]+$ and equal the file name stem %q", stem)
		s.Spec = stem
	}
	switch s.Tool {
	case "tla":
		if s.Machine == "" {
			s.bad("tool tla needs machine")
		}
	case "alloy":
		if s.Model == "" {
			s.bad("tool alloy needs model")
		}
	default:
		s.bad("tool must be tla or alloy, got %q", s.Tool)
	}
	if len(s.Areas) == 0 {
		s.bad("areas needs at least one risk area")
	}
	if len(s.Questions) == 0 {
		s.bad("questions needs at least one question")
	}
	for _, q := range s.Questions {
		tool, ok := design.Questions[q]
		switch {
		case !ok:
			s.bad("unknown question %q", q)
		case tool != s.Tool:
			s.bad("question %s is answered with %s, not %s", q, tool, s.Tool)
		}
	}
	if s.Properties == "" {
		s.bad("properties is required")
	}
	if len(s.Sources.Paths) == 0 {
		s.bad("sources.paths needs at least one path")
	}
	if s.Sources.ReadAt == "" {
		s.bad("sources.read_at is required")
	}
	aids := map[string]bool{}
	for _, a := range s.Assumptions {
		if !assumeRe.MatchString(a.ID) || aids[a.ID] {
			s.bad("assumption id %q must match ^A[0-9]+$ and be unique", a.ID)
		}
		aids[a.ID] = true
		if strings.TrimSpace(a.Text) == "" {
			s.bad("assumption %s: text is required", a.ID)
		}
		switch a.Status {
		case "unproven":
		case "proven":
			if a.Evidence == "" {
				s.bad("assumption %s: status proven needs evidence", a.ID)
			}
		default:
			s.bad("assumption %s: status must be proven or unproven", a.ID)
		}
	}
	if len(s.Runs) == 0 {
		s.bad("runs needs at least one run")
	}
	rnames := map[string]bool{}
	for _, r := range s.Runs {
		if !specNameRe.MatchString(r.Name) || rnames[r.Name] {
			s.bad("run name %q must match ^[a-z][a-z0-9-]+$ and be unique", r.Name)
		}
		rnames[r.Name] = true
		switch r.Expect {
		case "pass":
			if len(r.Reach) == 0 {
				s.bad("run %s: a run with expect: pass lists at least one reach predicate", r.Name)
			}
			if r.Assumption != "" || r.Story != "" {
				s.bad("run %s: assumption and story belong to expect: finding", r.Name)
			}
		case "finding":
			if (r.Assumption == "") == (r.Story == "") {
				s.bad("run %s: expect: finding needs exactly one of assumption and story", r.Name)
			}
			if r.Assumption != "" && !aids[r.Assumption] {
				s.bad("run %s: unknown assumption %s", r.Name, r.Assumption)
			}
		default:
			s.bad("run %s: expect must be pass or finding", r.Name)
		}
		if len(r.Invariants)+len(r.Properties) == 0 {
			s.bad("run %s: needs at least one invariant or property", r.Name)
		}
		if s.Tool == "alloy" && len(r.Properties) > 0 {
			s.bad("run %s: alloy runs use invariants (assertions) and reach, not properties", r.Name)
		}
	}
	mids := map[string]bool{}
	for _, m := range s.Mutations {
		if !mutRe.MatchString(m.ID) || mids[m.ID] {
			s.bad("mutation id %q must match ^M[0-9]+$ and be unique", m.ID)
		}
		mids[m.ID] = true
		if strings.TrimSpace(m.Bug) == "" || m.From == "" {
			s.bad("mutation %s: bug and from are required", m.ID)
		}
		r := s.RunByName(m.Run)
		switch {
		case r == nil:
			s.bad("mutation %s: unknown run %q", m.ID, m.Run)
		case r.Expect != "pass":
			s.bad("mutation %s: run %s must be a run with expect: pass", m.ID, m.Run)
		case !contains(r.Invariants, m.Detects) && !contains(r.Properties, m.Detects):
			s.bad("mutation %s: %s is not an invariant or property of run %s", m.ID, m.Detects, m.Run)
		}
		if m.File != "" && m.File != s.SkeletonName() && m.File != s.Properties+s.Ext() {
			s.bad("mutation %s: file must be %s or %s", m.ID, s.SkeletonName(), s.Properties+s.Ext())
		}
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// MutationFile returns the file a mutation applies to.
func (s *Spec) MutationFile(m Mutation) string {
	if m.File != "" {
		return m.File
	}
	return s.SkeletonName()
}

// InputHash binds a result to the spec bytes: the manifest without its
// sources block, the skeleton and the property file.
func (s *Spec) InputHash(skeleton, props []byte) string {
	mj, _ := json.Marshal(s.Manifest)
	h := sha256.New()
	h.Write(mj)
	h.Write([]byte{'\n'})
	h.Write(skeleton)
	h.Write([]byte{'\n'})
	h.Write(props)
	return hex.EncodeToString(h.Sum(nil))
}

// PropertyNames lists every invariant, property and reach name used by the runs, sorted.
func (s *Spec) PropertyNames() []string {
	set := map[string]bool{}
	for _, r := range s.Runs {
		for _, n := range r.Invariants {
			set[n] = true
		}
		for _, n := range r.Properties {
			set[n] = true
		}
		for _, n := range r.Reach {
			set[n] = true
		}
	}
	var res []string
	for n := range set {
		res = append(res, n)
	}
	sort.Strings(res)
	return res
}
