// Package project finds, loads and validates machinery.yaml, with
// design/adoption.yaml merged into it (docs/formats.md sections 2 and 2.1).
package project

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ConfigName is the project file.
const ConfigName = "machinery.yaml"

// Config mirrors machinery.yaml. Field order is the order config prints.
type Config struct {
	Version      int               `yaml:"version"`
	Scope        []string          `yaml:"scope,omitempty"`
	Design       Design            `yaml:"design"`
	Import       *Import           `yaml:"import,omitempty"`
	Contracts    []Contract        `yaml:"contracts,omitempty"`
	Checkers     []Checker         `yaml:"checkers,omitempty"`
	Dependencies []Dependency      `yaml:"dependencies,omitempty"`
	Systems      []System          `yaml:"systems,omitempty"`
	Trace        *Trace            `yaml:"trace,omitempty"`
	Tests        *Tests            `yaml:"tests,omitempty"`
	Constraints  *Constraints      `yaml:"constraints,omitempty"`
	Receipts     Receipts          `yaml:"receipts"`
	Evidence     *Evidence         `yaml:"evidence,omitempty"`
	Commands     Commands          `yaml:"commands,omitempty"`
	Backlog      *Backlog          `yaml:"backlog,omitempty"`
	Plan         *PlanConfig       `yaml:"plan,omitempty"`
	Hooks        *Hooks            `yaml:"hooks,omitempty"`
	Review       *Review           `yaml:"review,omitempty"`
	Delegate     *Delegate         `yaml:"delegate,omitempty"`
	Routing      map[string]string `yaml:"routing,omitempty"`
	Vital        *Vital            `yaml:"vital,omitempty"`
	Tools        Tools             `yaml:"tools"`
	Waivers      []Waiver          `yaml:"waivers,omitempty"`
	Assess       *Assess           `yaml:"assess,omitempty"`
	// ExcusedFile holds the entries of design/excused.yaml (section 2.2),
	// merged after those of evidence.adapter.excused
	ExcusedFile []AdapterExcused `yaml:"-"`
}

// Assess is read only by machinery assess (section 18).
type Assess struct {
	Decisions *Decisions `yaml:"decisions,omitempty"`
}

// Decisions names a project's own decision records (section 18.7).
type Decisions struct {
	Files  []string `yaml:"files"`
	Record string   `yaml:"record"`
}

// RecordPattern compiles the expression that matches the heading line of one record.
func (d *Decisions) RecordPattern() (*regexp.Regexp, error) { return regexp.Compile(d.Record) }

type Design struct {
	Models       []string `yaml:"models,omitempty"`
	Machines     string   `yaml:"machines,omitempty"`
	FailureModes string   `yaml:"failure_modes,omitempty"`
	Risk         string   `yaml:"risk,omitempty"`
	Formal       string   `yaml:"formal,omitempty"`
	Facts        string   `yaml:"facts,omitempty"`
	Owed         string   `yaml:"owed,omitempty"`
	Status       string   `yaml:"status,omitempty"`
	Gen          string   `yaml:"gen,omitempty"`
}

// Import names design sources kept in the project's own formats (section 2).
type Import struct {
	Machines     *ImportMachines     `yaml:"machines,omitempty"`
	FailureModes *ImportFailureModes `yaml:"failure_modes,omitempty"`
	Planned      []Planned           `yaml:"planned,omitempty"`
}

// Planned reads a project's own "planned" marks (section 2): where the mark
// is looked for and the expression whose group story names the story.
type Planned struct {
	In      string `yaml:"in"`
	Pattern string `yaml:"pattern"`
}

// Where a planned mark is looked for.
const (
	PlannedInStatement  = "model-statement"
	PlannedInScenario   = "scenario-description"
	PlannedInFailureRow = "failure-row"
)

// Regexp compiles the pattern of a planned mark.
func (pl Planned) Regexp() (*regexp.Regexp, error) { return regexp.Compile(pl.Pattern) }

// ImportedPlanned returns the import.planned entries.
func (c *Config) ImportedPlanned() []Planned {
	if c.Import == nil {
		return nil
	}
	return c.Import.Planned
}

// PlanConfig tunes machinery plan (section 15.2).
type PlanConfig struct {
	WireMaxFiles int `yaml:"wire_max_files,omitempty"`
}

// DefaultWireMaxFiles is the number of test files up to which wire-evidence
// stories are one per file (section 15.2).
const DefaultWireMaxFiles = 20

// WireMaxFiles returns plan.wire_max_files, or its default.
func (c *Config) WireMaxFiles() int {
	if c.Plan == nil || c.Plan.WireMaxFiles == 0 {
		return DefaultWireMaxFiles
	}
	return c.Plan.WireMaxFiles
}

// ImportFailureModes reads a failure-mode catalog kept as Markdown tables
// (section 5.4.1).
type ImportFailureModes struct {
	File    string    `yaml:"file"`
	ID      string    `yaml:"id"`
	Columns FMColumns `yaml:"columns"`
	Links   string    `yaml:"links,omitempty"`
}

// FMColumns lists the header names of each column a failure-mode row needs.
type FMColumns struct {
	Failure  []string `yaml:"failure"`
	Behavior []string `yaml:"behavior"`
	// Required names the column that holds the required value, for a
	// catalog that has one; optional.
	Required []string `yaml:"required,omitempty"`
}

// IDPattern compiles the id expression so that it matches a cell in full.
func (f *ImportFailureModes) IDPattern() (*regexp.Regexp, error) {
	return regexp.Compile("^(?:" + f.ID + ")$")
}

// ImportedFailureModes returns the import.failure_modes block, or nil.
func (c *Config) ImportedFailureModes() *ImportFailureModes {
	if c.Import == nil {
		return nil
	}
	return c.Import.FailureModes
}

// HasFailureModes reports whether the project has failure modes, native or imported.
func (c *Config) HasFailureModes() bool {
	return c.Design.FailureModes != "" || c.ImportedFailureModes() != nil
}

// ImportMachines reads machines written in a JSON dialect (section 5.2.1).
type ImportMachines struct {
	Files    []string            `yaml:"files"`
	States   string              `yaml:"states,omitempty"`
	Rename   Rename              `yaml:"rename,omitempty"`
	Terminal *Terminal           `yaml:"terminal,omitempty"`
	Register *Register           `yaml:"register,omitempty"`
	Bind     map[string]Bind     `yaml:"bind,omitempty"`
	Annotate map[string]Annotate `yaml:"annotate,omitempty"`
}

// Register names a register of observed transitions and the keys of its
// entries (section 5.2.1).
type Register struct {
	File     string `yaml:"file"`
	Machine  string `yaml:"machine"`
	Observed string `yaml:"observed"`
}

// RegisterFile returns the register file of import.machines, or "".
func (im *ImportMachines) RegisterFile() string {
	if im == nil || im.Register == nil {
		return ""
	}
	return im.Register.File
}

// Terminal names the root key whose reason says that a machine is never
// left in its absorbing states (section 5.2.1).
type Terminal struct {
	FromRoot string `yaml:"from_root"`
}

// TerminalKey returns the root key of import.machines.terminal, or "".
func (im *ImportMachines) TerminalKey() string {
	if im.Terminal == nil {
		return ""
	}
	return im.Terminal.FromRoot
}

// Bind is what an imported machine binds to in the model (section 5.2.1):
// the entity of a lifecycle machine, or the ids an operational machine serves.
type Bind struct {
	LifecycleOf string   `yaml:"lifecycle_of,omitempty"`
	Aspect      string   `yaml:"aspect,omitempty"`
	Serves      []string `yaml:"serves,omitempty"`
}

// Annotate says about an imported machine what its dialect has no key for
// (section 5.2.1): the events that arrive from outside, the events that
// report a failure, the states that record one, and the events that are a
// model action under another name. The three lists hold exact names and
// patterns in which "*" stands for any run of characters.
type Annotate struct {
	ExternalEvents []string          `yaml:"external_events,omitempty"`
	FailureEvents  []string          `yaml:"failure_events,omitempty"`
	FailureStates  []string          `yaml:"failure_states,omitempty"`
	Aliases        map[string]string `yaml:"aliases,omitempty"`
	// RefusalEvents maps an event by which the machine writes that it
	// refused a caller's command to the action it refuses.
	// Each carries the reason the event is a refusal and no failure.
	RefusalEvents map[string]Refusal `yaml:"refusal_events,omitempty"`
	// NotFailures dismisses the name hint of an event or a state, by its
	// exact name, with the reason it is no failure.
	NotFailures map[string]string `yaml:"not_failures,omitempty"`
}

// Refusal is one entry of refusal_events: the action the event refuses and
// the reason it is a refusal and no failure.
type Refusal struct {
	Action string `yaml:"action"`
	Reason string `yaml:"reason"`
}

// UnmarshalYAML reads {action, reason} and says what an entry is when it is
// something else, such as a bare action name.
func (r *Refusal) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		got := n.Value
		if n.Kind != yaml.ScalarNode {
			got = "a list"
		}
		return fmt.Errorf("line %d: a refusal_events entry must be {action: <action>, reason: \"<why the event is a refusal and no failure>\"}, got %s", n.Line, got)
	}
	type plain Refusal
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	*r = Refusal(p)
	return nil
}

// OneLine reports text that prints on one line: no line break and no other
// control character. What the tool prints from a config or an annotation
// (an action, a reason) is held to it, so that no declaration can put text
// of its author's choice at the start of an output line.
func OneLine(s string) bool {
	for _, c := range s {
		if unicode.IsControl(c) {
			return false
		}
	}
	return true
}

// AnnotateKeys returns the keys of import.machines.annotate, sorted.
func (im *ImportMachines) AnnotateKeys() []string {
	keys := make([]string, 0, len(im.Annotate))
	for k := range im.Annotate {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// BindKeys returns the keys of import.machines.bind, sorted.
func (im *ImportMachines) BindKeys() []string {
	keys := make([]string, 0, len(im.Bind))
	for k := range im.Bind {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Rename maps the project's keys to machinery keys, per level of a machine.
type Rename struct {
	Root       map[string]string `yaml:"root,omitempty"`
	State      map[string]string `yaml:"state,omitempty"`
	Transition map[string]string `yaml:"transition,omitempty"`
}

// The states values of import.machines.
const (
	StatesKeep       = "keep"
	StatesCapitalize = "capitalize"
	StatesPascal     = "pascal"
)

// ImportedMachines returns the import.machines block, or nil.
func (c *Config) ImportedMachines() *ImportMachines {
	if c.Import == nil {
		return nil
	}
	return c.Import.Machines
}

// HasMachines reports whether the project has machines, native or imported.
func (c *Config) HasMachines() bool { return c.Design.Machines != "" || c.ImportedMachines() != nil }

type Contract struct {
	Kind string `yaml:"kind"`
	Path string `yaml:"path"`
}

type Checker struct {
	ID       string   `yaml:"id"`
	Covers   []string `yaml:"covers"`
	Evidence string   `yaml:"evidence,omitempty"`
}

type Dependency struct {
	Name    string `yaml:"name"`
	Posture string `yaml:"posture"`
}

// System names the system as a whole, or a part of it that is not an entity,
// a dependency or a machine, for failure-mode subjects (section 5.4).
type System struct {
	Name  string `yaml:"name"`
	About string `yaml:"about,omitempty"`
}

type Trace struct {
	Code    []string `yaml:"code,omitempty"`
	Tests   []string `yaml:"tests,omitempty"`
	Markers *Markers `yaml:"markers,omitempty"`
}

// Markers are a project's own trace markers, literal prefixes (section 8.1).
type Markers struct {
	Serves    []string `yaml:"serves,omitempty"`
	Verifies  []string `yaml:"verifies,omitempty"`
	Scenarios []string `yaml:"scenarios,omitempty"`
	// ScenariosExact claim only when the text is a scenario name (section 8.1).
	ScenariosExact []string `yaml:"scenarios_exact,omitempty"`
}

type Tests struct {
	E2E      []string `yaml:"e2e,omitempty"`
	Files    []string `yaml:"files,omitempty"`
	Doubles  []string `yaml:"doubles,omitempty"`
	Baseline string   `yaml:"baseline,omitempty"`
}

type Layer struct {
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths"`
}

type Size struct {
	Paths        []string `yaml:"paths,omitempty"`
	MaxFileLines int      `yaml:"max_file_lines,omitempty"`
	MaxFuncLines int      `yaml:"max_func_lines,omitempty"`
}

type Constraints struct {
	Layers   []Layer `yaml:"layers,omitempty"`
	Size     *Size   `yaml:"size,omitempty"`
	Cores    *Cores  `yaml:"cores,omitempty"`
	Baseline string  `yaml:"baseline,omitempty"`
}

// Cores is the shape of the core of every vital area (section 9).
type Cores struct {
	DenyImports  []string `yaml:"deny_imports,omitempty"`
	MaxFileLines int      `yaml:"max_file_lines,omitempty"`
	MaxFuncLines int      `yaml:"max_func_lines,omitempty"`
}

// Vital holds the word lists of machinery vital candidates (section 23.3). A
// list that is set replaces its default, even when empty.
type Vital struct {
	Names   *[]string `yaml:"names,omitempty"`
	Failure *[]string `yaml:"failure,omitempty"`
	Effects *[]string `yaml:"effects,omitempty"`
}

type Receipts struct {
	Dir string `yaml:"dir,omitempty"`
}

// Evidence holds the policy for evidence machinery writes itself, and the
// paths a run stays valid across (section 10.1).
type Evidence struct {
	// Exempt are globs of paths whose change leaves a run of an earlier
	// commit valid; they never match a path that can change a verdict
	Exempt  []string `yaml:"exempt,omitempty"`
	Adapter *Adapter `yaml:"adapter,omitempty"`
}

// Adapter is the policy for adapter evidence (section 10.3). A list that is
// set replaces its default, even when empty.
type Adapter struct {
	Allow          *[]string   `yaml:"allow,omitempty"`
	Tiers          *[]string   `yaml:"tiers,omitempty"`
	MinGranularity Granularity `yaml:"min_granularity,omitempty"`
	// Claims is all (the default) or any: whether every claimant of an id
	// must pass, or one (section 10.3)
	Claims  string           `yaml:"claims,omitempty"`
	Sources []AdapterSource  `yaml:"sources,omitempty"`
	Excused []AdapterExcused `yaml:"excused,omitempty"`
}

// Values of evidence.adapter.claims.
const (
	ClaimsAll = "all"
	ClaimsAny = "any"
)

// AdapterExcused leaves claimants that cannot run where the evidence is made
// out of the rule of every claimant (section 10.3): the test functions of the
// files its tests globs match, only those its functions patterns match when
// it has any, with the reason.
type AdapterExcused struct {
	Tests     []string `yaml:"tests"`
	Functions []string `yaml:"functions,omitempty"`
	Reason    string   `yaml:"reason"`
	// N is the number of the entry in the file it is written in, from 1,
	// and From that file when it is not machinery.yaml; Config.AdapterExcused
	// sets both
	N    int    `yaml:"-"`
	From string `yaml:"-"`
}

// Label is how a message names the entry: "entry <n>" for one of
// machinery.yaml, "<file>: entry <n>" for one of another file.
func (e AdapterExcused) Label() string {
	if e.From != "" {
		return fmt.Sprintf("%s: entry %d", e.From, e.N)
	}
	return fmt.Sprintf("entry %d", e.N)
}

// Key is how the ratchet names the entry: by its key in machinery.yaml, or
// by its file.
func (e AdapterExcused) Key() string {
	if e.From != "" {
		return e.Label()
	}
	return "evidence.adapter.excused: " + e.Label()
}

// Name is how a message names what the entry covers: its tests globs, and
// its functions patterns when it has any.
func (e AdapterExcused) Name() string {
	s := "tests " + strings.Join(e.Tests, ", ")
	if len(e.Functions) > 0 {
		s += "; functions " + strings.Join(e.Functions, ", ")
	}
	return s
}

// MatchFunction reports whether a test function name matches one of the
// functions patterns; true for every name when there is none.
func (e AdapterExcused) MatchFunction(name string) bool {
	if len(e.Functions) == 0 {
		return true
	}
	for _, f := range e.Functions {
		if ok, _ := path.Match(f, name); ok {
			return true
		}
	}
	return false
}

// AdapterSource maps test packages built from copies to the test files
// under trace.tests they are built from (section 10.3). With Dir, Package is
// an import path pattern of go and each package it matches is built from the
// test files of its own directory.
type AdapterSource struct {
	Package string   `yaml:"package"`
	Files   []string `yaml:"files,omitempty"`
	Dir     bool     `yaml:"dir,omitempty"`
}

// Matches reports whether the entry is for the package: by its glob, or with
// Dir by its import path pattern.
func (s AdapterSource) Matches(pkg string) bool {
	if s.Dir {
		return MatchImportPattern(s.Package, pkg)
	}
	return Match([]string{s.Package}, pkg)
}

// Wildcard reports a package glob that may match several packages: one with
// *, ?, [ or {. An import path has none of them.
func (s AdapterSource) Wildcard() bool { return strings.ContainsAny(s.Package, "*?[{") }

// Maps returns what the entry maps its packages to, as the ratchet compares
// it: the file globs, or "dir: true".
func (s AdapterSource) Maps() []string {
	if s.Dir {
		return []string{"dir: true"}
	}
	return s.Files
}

// Name is how a message names the entry's package.
func (s AdapterSource) Name() string {
	if s.Dir {
		return s.Package + ", dir: true"
	}
	return s.Package
}

// MatchImportPattern matches an import path against a pattern as go does:
// "..." matches any string, slashes included, and "x/..." matches x too.
func MatchImportPattern(pattern, path string) bool {
	re := regexp.QuoteMeta(pattern)
	re = strings.ReplaceAll(re, `\.\.\.`, `.*`)
	if strings.HasSuffix(re, `/.*`) {
		re = strings.TrimSuffix(re, `/.*`) + `(/.*)?`
	}
	ok, _ := regexp.MatchString("^"+re+"$", path)
	return ok
}

// Granularities of an adapter receipt, the weaker first.
const (
	GranularityFile     = "file"
	GranularityFunction = "function"
)

// Granularity is evidence.adapter.min_granularity: one value, or a mapping
// from default, a kind or a tier to a value.
type Granularity struct {
	Default string
	By      map[string]string
}

// UnmarshalYAML reads the scalar and the mapping form.
func (g *Granularity) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		g.Default = n.Value
		return nil
	}
	by := map[string]string{}
	if err := n.Decode(&by); err != nil {
		return err
	}
	g.Default = by["default"]
	delete(by, "default")
	g.By = by
	return nil
}

// MarshalYAML prints the form that was written.
func (g Granularity) MarshalYAML() (any, error) {
	if len(g.By) == 0 {
		return g.Default, nil
	}
	out := map[string]string{}
	for k, v := range g.By {
		out[k] = v
	}
	if g.Default != "" {
		out["default"] = g.Default
	}
	return out, nil
}

// IsZero reports an unset value, which config does not print.
func (g Granularity) IsZero() bool { return g.Default == "" && len(g.By) == 0 }

// AdapterKinds are the kinds an adapter pass can prove, and the default of
// evidence.adapter.allow; AdapterTiers is the default of evidence.adapter.tiers.
var (
	AdapterKinds = []string{"inv", "sc", "fm", "ck"}
	AdapterTiers = []string{"routine", "consequential"}
	tierNames    = []string{"routine", "consequential", "critical", "vital"}
)

// AdapterPolicy returns the effective policy for adapter evidence: the
// kinds and tiers an adapter pass may prove.
func (c *Config) AdapterPolicy() (kinds, tiers []string, min Granularity) {
	kinds, tiers = AdapterKinds, AdapterTiers
	if c.Evidence == nil || c.Evidence.Adapter == nil {
		return kinds, tiers, min
	}
	a := c.Evidence.Adapter
	if a.Allow != nil {
		kinds = *a.Allow
	}
	if a.Tiers != nil {
		tiers = *a.Tiers
	}
	return kinds, tiers, a.MinGranularity
}

// AdapterClaimsAll reports whether every claimant of an id must pass for
// adapter evidence to prove it: evidence.adapter.claims is all, or unset.
func (c *Config) AdapterClaimsAll() bool {
	return c.Evidence == nil || c.Evidence.Adapter == nil || c.Evidence.Adapter.Claims != ClaimsAny
}

// AdapterExcused returns the entries of evidence.adapter.excused, each
// numbered in the file it is written in.
func (c *Config) AdapterExcused() []AdapterExcused {
	var res []AdapterExcused
	if c.Evidence != nil && c.Evidence.Adapter != nil {
		for i, e := range c.Evidence.Adapter.Excused {
			e.N, e.From = i+1, ""
			res = append(res, e)
		}
	}
	for i, e := range c.ExcusedFile {
		e.N, e.From = i+1, ExcusedName
		res = append(res, e)
	}
	return res
}

// EvidenceExempt returns the globs of evidence.exempt.
func (c *Config) EvidenceExempt() []string {
	if c.Evidence == nil {
		return nil
	}
	return c.Evidence.Exempt
}

// AdapterSources returns evidence.adapter.sources.
func (c *Config) AdapterSources() []AdapterSource {
	if c.Evidence == nil || c.Evidence.Adapter == nil {
		return nil
	}
	return c.Evidence.Adapter.Sources
}

// validateEvidence holds the evidence block to section 2.
func (c *Config) validateEvidence() error {
	if c.Evidence == nil {
		return nil
	}
	for _, g := range c.Evidence.Exempt {
		if err := CheckPath("evidence.exempt", g); err != nil {
			return err
		}
		if strings.TrimSpace(g) == "" || !ValidGlob(g) {
			return Errf("evidence.exempt: %q is not a valid glob", g)
		}
	}
	if c.Evidence.Adapter == nil {
		return nil
	}
	a := c.Evidence.Adapter
	in := func(list []string, s string) bool {
		for _, v := range list {
			if v == s {
				return true
			}
		}
		return false
	}
	if a.Allow != nil {
		for _, k := range *a.Allow {
			if !in(AdapterKinds, k) {
				return Errf("evidence.adapter.allow: %s: the adapter proves only inv, sc, fm and ck ids", k)
			}
		}
	}
	if a.Tiers != nil {
		for _, t := range *a.Tiers {
			if !in(tierNames, t) {
				return Errf("evidence.adapter.tiers: %q is not a tier: routine, consequential, critical or vital", t)
			}
		}
	}
	g := a.MinGranularity
	if g.Default != "" && g.Default != GranularityFunction && g.Default != GranularityFile {
		key := "evidence.adapter.min_granularity"
		if g.By != nil {
			key += ".default"
		}
		return Errf("%s must be function or file, got %q", key, g.Default)
	}
	keys := make([]string, 0, len(g.By))
	for k := range g.By {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !in(AdapterKinds, k) && !in(tierNames, k) {
			return Errf("evidence.adapter.min_granularity: %q is not default, a kind (inv, sc, fm, ck) or a tier", k)
		}
		if v := g.By[k]; v != GranularityFunction && v != GranularityFile {
			return Errf("evidence.adapter.min_granularity.%s must be function or file, got %q", k, v)
		}
	}
	if a.Claims != "" && a.Claims != ClaimsAll && a.Claims != ClaimsAny {
		return Errf("evidence.adapter.claims must be all or any, got %q", a.Claims)
	}
	if err := validateExcused(a.Excused, "evidence.adapter.excused", func(int) string { return "evidence.adapter.excused.tests" }); err != nil {
		return err
	}
	for i, s := range a.Sources {
		switch {
		case strings.TrimSpace(s.Package) == "":
			return Errf("evidence.adapter.sources: entry %d: package is required", i+1)
		case s.Dir && len(s.Files) > 0:
			return Errf("evidence.adapter.sources: entry %d: files and dir: true do not combine; write one entry for each", i+1)
		case s.Dir && strings.ContainsAny(s.Package, "*?[]{}\\"):
			return Errf("evidence.adapter.sources: entry %d: package %q with dir: true is an import path pattern: ... is its only wildcard", i+1, s.Package)
		case s.Dir:
			continue
		case !ValidGlob(s.Package):
			return Errf("evidence.adapter.sources: entry %d: package %q is not a valid glob", i+1, s.Package)
		case len(s.Files) == 0:
			return Errf("evidence.adapter.sources: entry %d: files is required, or dir: true", i+1)
		}
		for _, f := range s.Files {
			if err := CheckPath("evidence.adapter.sources.files", f); err != nil {
				return err
			}
			if strings.TrimSpace(f) == "" || !ValidGlob(f) {
				return Errf("evidence.adapter.sources: entry %d: files %q is not a valid glob", i+1, f)
			}
		}
	}
	return nil
}

// validateExcused holds entries of evidence.adapter.excused to section 2: in
// messages they are "<where>: entry <n>", and pathKey names the key of the
// tests of entry n in the messages of a path that leaves the project.
func validateExcused(entries []AdapterExcused, where string, pathKey func(n int) string) error {
	for i, e := range entries {
		if len(e.Tests) == 0 {
			return Errf("%s: entry %d: tests is required: globs of files under trace.tests", where, i+1)
		}
		for _, g := range e.Tests {
			if err := CheckPath(pathKey(i+1), g); err != nil {
				return err
			}
			if strings.TrimSpace(g) == "" || !ValidGlob(g) {
				return Errf("%s: entry %d: tests %q is not a valid glob", where, i+1, g)
			}
		}
		for _, f := range e.Functions {
			if _, err := path.Match(f, ""); err != nil || strings.TrimSpace(f) == "" {
				return Errf("%s: entry %d: functions %q is not a valid pattern", where, i+1, f)
			}
		}
		switch {
		case strings.TrimSpace(e.Reason) == "":
			return Errf("%s: entry %d: reason is required: say why these tests cannot run here", where, i+1)
		case !OneLine(e.Reason):
			return Errf("%s: entry %d: reason %q has a line break or a control character; a reason is one line", where, i+1, e.Reason)
		}
	}
	return nil
}

type Commands struct {
	E2E []string `yaml:"e2e,omitempty"`
}

type Backlog struct {
	Tool string `yaml:"tool"`
	Base string `yaml:"base,omitempty"`
}

// Hooks sets what machinery hook does at each point (section 21.1): block,
// warn or off.
type Hooks struct {
	Commit string `yaml:"commit,omitempty"`
	Push   string `yaml:"push,omitempty"`
	Stop   string `yaml:"stop,omitempty"`
}

// The hook modes and their defaults (section 21.1).
const (
	HookBlock = "block"
	HookWarn  = "warn"
	HookOff   = "off"
)

var hookDefaults = Hooks{Commit: HookBlock, Push: HookBlock, Stop: HookWarn}

// HookMode returns the mode of a hook point: "commit", "push" or "stop".
func (c *Config) HookMode(point string) string {
	h := hookDefaults
	if c.Hooks != nil {
		h = *c.Hooks
	}
	switch point {
	case "commit":
		return h.Commit
	case "push":
		return h.Push
	}
	return h.Stop
}

// Review says which of the checker's reviews gate dispatch (section 15.10):
// required or off.
type Review struct {
	Backlog string `yaml:"backlog,omitempty"`
	Design  string `yaml:"design,omitempty"`
}

// The review settings (section 2).
const (
	ReviewRequired = "required"
	ReviewOff      = "off"
)

// ReviewOn reports whether a review of the kind ("backlog" or "design") is
// required before dispatch.
func (c *Config) ReviewOn(kind string) bool {
	v := ""
	if c.Review != nil {
		v = c.Review.Backlog
		if kind == "design" {
			v = c.Review.Design
		}
	}
	if v == "" {
		return c.Backlog != nil
	}
	return v == ReviewRequired
}

// Delegate names the owner's policy for answers on the owner's behalf
// (section 15.9). Policy may lie outside the project.
type Delegate struct {
	Policy string `yaml:"policy"`
}

type Tools struct {
	Modelith string `yaml:"modelith,omitempty"`
	Java     string `yaml:"java,omitempty"`
	Node     string `yaml:"node,omitempty"`
	BD       string `yaml:"bd,omitempty"`
	Dir      string `yaml:"dir,omitempty"`
	XState   bool   `yaml:"xstate,omitempty"`
}

type Waiver struct {
	Rule   string `yaml:"rule"`
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
}

// Project is a loaded project.
type Project struct {
	Root string // absolute
	Cfg  Config
}

// ConfigError is a usage or config error: exit 2.
type ConfigError struct{ Msg string }

func (e *ConfigError) Error() string { return e.Msg }

// Errf builds a ConfigError.
func Errf(format string, a ...any) error { return &ConfigError{fmt.Sprintf(format, a...)} }

// Find walks up from dir to the directory holding machinery.yaml.
func Find(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, ConfigName)); err == nil {
			return d, nil
		}
		if d == filepath.Dir(d) {
			return "", Errf("no %s found in %s or any parent directory", ConfigName, abs)
		}
	}
}

// Load finds and loads the project. explicit is the --project flag ("" when unset).
func Load(explicit string) (*Project, error) {
	var root string
	if explicit != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(filepath.Join(abs, ConfigName)); err != nil {
			return nil, Errf("no %s in %s", ConfigName, explicit)
		}
		root = abs
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		r, err := Find(wd)
		if err != nil {
			return nil, err
		}
		root = r
	}
	if r, err := filepath.EvalSymlinks(root); err == nil {
		root = r
	}
	data, err := os.ReadFile(filepath.Join(root, ConfigName))
	if err != nil {
		return nil, Errf("%v", err)
	}
	adoption, err := ReadAdoption(root)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseWith(data, adoption)
	if err != nil {
		return nil, err
	}
	if cfg.ExcusedFile, err = ReadExcused(root); err != nil {
		return nil, err
	}
	p := &Project{Root: root, Cfg: *cfg}
	if err := p.checkExempt(); err != nil {
		return nil, err
	}
	return p, nil
}

// Parse decodes and validates a machinery.yaml alone and applies defaults.
func Parse(data []byte) (*Config, error) { return ParseWith(data, nil) }

func cleanYAMLError(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "yaml: unmarshal errors:\n  ")
	msg = strings.TrimPrefix(msg, "yaml: ")
	return strings.ReplaceAll(msg, "\n  ", "; ")
}

var (
	checkerIDRe  = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
	systemNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	// WaivableRules are the cross-reference rules a waiver may name (section 6.7).
	WaivableRules = map[string]bool{
		"machine-binding": true, "state-enum": true, "event-action": true, "action-event": true,
		"operation-id": true, "operation-action": true, "action-operation": true,
		"subject-resolves": true, "invoke-failure-mode": true, "dependency-failure-mode": true,
		"tiered-action-abuse": true, "failure-state-mode": true, "failure-event-mode": true,
		"entity-machine": true,
	}
)

// CheckPath validates a project-relative path: forward slashes, inside the root.
func CheckPath(key, p string) error {
	if p == "" {
		return nil
	}
	if strings.Contains(p, "\\") {
		return Errf("%s: %q must use forward slashes", key, p)
	}
	if path.IsAbs(p) || filepath.IsAbs(p) {
		return Errf("%s: %q must be relative to the project root", key, p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return Errf("%s: %q must stay inside the project root", key, p)
		}
	}
	return nil
}

func (c *Config) validate() error {
	if c.Version != 2 {
		return Errf("%s: version must be 2", ConfigName)
	}
	check := func(key string, ps ...string) error {
		for _, p := range ps {
			if err := CheckPath(key, p); err != nil {
				return err
			}
		}
		return nil
	}
	d := c.Design
	steps := []error{
		check("scope", c.Scope...),
		check("design.models", d.Models...),
		check("design", d.Machines, d.FailureModes, d.Risk, d.Formal, d.Facts, d.Owed, d.Status, d.Gen),
		check("receipts.dir", c.Receipts.Dir),
	}
	for _, ct := range c.Contracts {
		if ct.Kind != "openapi" && ct.Kind != "proto" {
			return Errf("contracts: kind must be openapi or proto, got %q", ct.Kind)
		}
		if ct.Path == "" {
			return Errf("contracts: path is required")
		}
		steps = append(steps, check("contracts.path", ct.Path))
	}
	seen := map[string]bool{}
	for _, ck := range c.Checkers {
		if !checkerIDRe.MatchString(ck.ID) {
			return Errf("checkers: id %q must match ^[a-z][a-z0-9-]+$", ck.ID)
		}
		if seen[ck.ID] {
			return Errf("checkers: duplicate id %q", ck.ID)
		}
		seen[ck.ID] = true
		if len(ck.Covers) == 0 {
			return Errf("checkers: %s: covers is required", ck.ID)
		}
		steps = append(steps, check("checkers.evidence", ck.Evidence))
	}
	for _, dep := range c.Dependencies {
		if dep.Name == "" {
			return Errf("dependencies: name is required")
		}
		if dep.Posture != "real" && dep.Posture != "sandbox" {
			return Errf("dependencies: %s: posture must be real or sandbox, got %q", dep.Name, dep.Posture)
		}
	}
	systems := map[string]bool{}
	for _, sy := range c.Systems {
		switch {
		case sy.Name == "":
			return Errf("systems: name is required")
		case !systemNameRe.MatchString(sy.Name):
			return Errf("systems: name %q must match ^[a-z][a-z0-9-]*$", sy.Name)
		case systems[sy.Name]:
			return Errf("systems: duplicate name %q", sy.Name)
		}
		systems[sy.Name] = true
	}
	if c.Trace != nil {
		steps = append(steps, check("trace.code", c.Trace.Code...), check("trace.tests", c.Trace.Tests...))
		if m := c.Trace.Markers; m != nil {
			for _, kind := range []struct {
				name    string
				markers []string
			}{{"serves", m.Serves}, {"verifies", m.Verifies}, {"scenarios", m.Scenarios}, {"scenarios_exact", m.ScenariosExact}} {
				for _, mk := range kind.markers {
					if strings.TrimSpace(mk) == "" {
						return Errf("trace.markers.%s: a marker must not be empty", kind.name)
					}
				}
			}
			for _, mk := range m.ScenariosExact {
				for _, other := range m.Scenarios {
					if mk == other {
						return Errf("trace.markers.scenarios_exact: %q is also in trace.markers.scenarios; a marker has one mode", mk)
					}
				}
			}
		}
	}
	if c.Tests != nil {
		steps = append(steps, check("tests.e2e", c.Tests.E2E...), check("tests.baseline", c.Tests.Baseline))
	}
	if c.Constraints != nil {
		names := map[string]bool{}
		for _, l := range c.Constraints.Layers {
			if l.Name == "" || len(l.Paths) == 0 {
				return Errf("constraints.layers: each layer needs a name and paths")
			}
			if names[l.Name] {
				return Errf("constraints.layers: duplicate layer %q", l.Name)
			}
			names[l.Name] = true
			steps = append(steps, check("constraints.layers.paths", l.Paths...))
		}
		if s := c.Constraints.Size; s != nil {
			if s.MaxFileLines < 0 || s.MaxFuncLines < 0 {
				return Errf("constraints.size: limits must not be negative")
			}
			steps = append(steps, check("constraints.size.paths", s.Paths...))
		}
		if cr := c.Constraints.Cores; cr != nil {
			if cr.MaxFileLines <= 0 {
				return Errf("constraints.cores.max_file_lines is required and must be positive")
			}
			if cr.MaxFuncLines <= 0 {
				return Errf("constraints.cores.max_func_lines is required and must be positive")
			}
			for _, d := range cr.DenyImports {
				if strings.TrimSpace(d) == "" {
					return Errf("constraints.cores.deny_imports: an entry must not be empty")
				}
			}
		}
		steps = append(steps, check("constraints.baseline", c.Constraints.Baseline))
	}
	if err := c.validateImport(); err != nil {
		return err
	}
	if err := c.validateAssess(); err != nil {
		return err
	}
	if err := c.validateEvidence(); err != nil {
		return err
	}
	if c.Plan != nil && c.Plan.WireMaxFiles < 0 {
		return Errf("plan.wire_max_files must not be negative")
	}
	if c.Backlog != nil && c.Backlog.Tool != "bd" {
		return Errf("backlog.tool: only bd is supported, got %q", c.Backlog.Tool)
	}
	if h := c.Hooks; h != nil {
		for _, pt := range []struct{ key, v string }{{"commit", h.Commit}, {"push", h.Push}, {"stop", h.Stop}} {
			switch pt.v {
			case "", HookBlock, HookWarn, HookOff:
			default:
				return Errf("hooks.%s must be block, warn or off, got %q", pt.key, pt.v)
			}
		}
	}
	if r := c.Review; r != nil {
		for _, pt := range []struct{ key, v string }{{"backlog", r.Backlog}, {"design", r.Design}} {
			switch pt.v {
			case "", ReviewRequired, ReviewOff:
			default:
				return Errf("review.%s must be required or off, got %q", pt.key, pt.v)
			}
		}
	}
	if c.Delegate != nil && strings.TrimSpace(c.Delegate.Policy) == "" {
		return Errf("delegate.policy is required")
	}
	for k := range c.Routing {
		if k != "routine" && k != "consequential" && k != "critical" && k != "vital" {
			return Errf("routing: unknown tier %q", k)
		}
	}
	if v := c.Vital; v != nil {
		for _, l := range []struct {
			name  string
			words *[]string
		}{{"names", v.Names}, {"failure", v.Failure}, {"effects", v.Effects}} {
			if l.words == nil {
				continue
			}
			for _, w := range *l.words {
				if strings.TrimSpace(w) == "" {
					return Errf("vital.%s: a word must not be empty", l.name)
				}
			}
		}
	}
	for _, w := range c.Waivers {
		if !WaivableRules[w.Rule] {
			return Errf("waivers: %q is not a rule that can be waived", w.Rule)
		}
		if w.ID == "" || strings.TrimSpace(w.Reason) == "" {
			return Errf("waivers: rule %s needs an id and a reason", w.Rule)
		}
	}
	return errors.Join(steps...)
}

// validateAssess holds the assess block to section 2.
func (c *Config) validateAssess() error {
	if c.Assess == nil || c.Assess.Decisions == nil {
		return nil
	}
	d := c.Assess.Decisions
	if len(d.Files) == 0 {
		return Errf("assess.decisions.files is required")
	}
	for _, g := range d.Files {
		if err := CheckPath("assess.decisions.files", g); err != nil {
			return err
		}
		if !ValidGlob(g) {
			return Errf("assess.decisions.files: %q is not a valid glob", g)
		}
	}
	if strings.TrimSpace(d.Record) == "" {
		return Errf("assess.decisions.record is required")
	}
	if _, err := d.RecordPattern(); err != nil {
		return Errf("assess.decisions.record is not a regular expression: %v", err)
	}
	return nil
}

// validateImport holds the import block to section 2.
func (c *Config) validateImport() error {
	if err := c.validateImportedFailureModes(); err != nil {
		return err
	}
	for n, pl := range c.ImportedPlanned() {
		at := fmt.Sprintf("import.planned: entry %d", n+1)
		switch pl.In {
		case PlannedInStatement, PlannedInScenario:
		case PlannedInFailureRow:
			if c.ImportedFailureModes() == nil {
				return Errf("%s: in: failure-row needs import.failure_modes; it reads the rows of that catalog", at)
			}
		default:
			return Errf("%s: in must be model-statement, scenario-description or failure-row, got %q", at, pl.In)
		}
		if strings.TrimSpace(pl.Pattern) == "" {
			return Errf("%s: pattern is required", at)
		}
		re, err := pl.Regexp()
		if err != nil {
			return Errf("%s: pattern is not a regular expression: %v", at, err)
		}
		if re.SubexpIndex("story") < 0 {
			return Errf("%s: pattern %q needs a named group story, as in (?P<story>[a-z0-9-]+)", at, pl.Pattern)
		}
	}
	im := c.ImportedMachines()
	if im == nil {
		return nil
	}
	if c.Design.Machines != "" {
		return Errf("import.machines: design.machines is also set; read the machines from one of them")
	}
	if len(im.Files) == 0 {
		return Errf("import.machines.files is required")
	}
	for _, g := range im.Files {
		if err := CheckPath("import.machines.files", g); err != nil {
			return err
		}
		if !ValidGlob(g) {
			return Errf("import.machines.files: %q is not a valid glob", g)
		}
	}
	switch im.States {
	case "", StatesKeep, StatesCapitalize, StatesPascal:
	default:
		return Errf("import.machines.states must be capitalize, pascal or keep, got %q", im.States)
	}
	for _, lv := range []struct {
		name string
		m    map[string]string
	}{{"root", im.Rename.Root}, {"state", im.Rename.State}, {"transition", im.Rename.Transition}} {
		for k, v := range lv.m {
			if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
				return Errf("import.machines.rename.%s: a key and its new name must not be empty", lv.name)
			}
		}
	}
	if rg := im.Register; rg != nil {
		for _, f := range []struct{ name, v string }{{"file", rg.File}, {"machine", rg.Machine}, {"observed", rg.Observed}} {
			if strings.TrimSpace(f.v) == "" {
				return Errf("import.machines.register.%s is required", f.name)
			}
		}
		if err := CheckPath("import.machines.register.file", rg.File); err != nil {
			return err
		}
	}
	if t := im.Terminal; t != nil {
		if strings.TrimSpace(t.FromRoot) == "" {
			return Errf("import.machines.terminal.from_root is required")
		}
		if _, renamed := im.Rename.Root[t.FromRoot]; renamed {
			return Errf("import.machines.terminal.from_root: %s is also renamed by import.machines.rename.root; keep one", t.FromRoot)
		}
	}
	for _, k := range im.BindKeys() {
		b := im.Bind[k]
		if strings.TrimSpace(k) == "" || !ValidGlob(k) {
			return Errf("import.machines.bind: %q is not a valid glob", k)
		}
		if err := CheckPath("import.machines.bind", k); err != nil {
			return err
		}
		lifecycle := strings.TrimSpace(b.LifecycleOf) != ""
		switch {
		case lifecycle && len(b.Serves) > 0:
			return Errf("import.machines.bind.%s: set lifecycle_of or serves, not both", k)
		case !lifecycle && len(b.Serves) == 0:
			return Errf("import.machines.bind.%s: set lifecycle_of or serves", k)
		case b.Aspect != "" && !lifecycle:
			return Errf("import.machines.bind.%s: aspect needs lifecycle_of; an aspect is part of an entity's lifecycle", k)
		}
		for _, id := range b.Serves {
			if strings.TrimSpace(id) == "" {
				return Errf("import.machines.bind.%s.serves: an id must not be empty", k)
			}
		}
	}
	for _, k := range im.AnnotateKeys() {
		a := im.Annotate[k]
		if strings.TrimSpace(k) == "" || !ValidGlob(k) {
			return Errf("import.machines.annotate: %q is not a valid glob", k)
		}
		if err := CheckPath("import.machines.annotate", k); err != nil {
			return err
		}
		if len(a.ExternalEvents)+len(a.FailureEvents)+len(a.FailureStates)+len(a.Aliases)+len(a.RefusalEvents)+len(a.NotFailures) == 0 {
			return Errf("import.machines.annotate.%s: set external_events, failure_events, failure_states, aliases, refusal_events or not_failures", k)
		}
		for _, l := range []struct {
			name     string
			patterns []string
		}{{"external_events", a.ExternalEvents}, {"failure_events", a.FailureEvents}, {"failure_states", a.FailureStates}} {
			for _, pat := range l.patterns {
				if strings.TrimSpace(pat) == "" {
					return Errf("import.machines.annotate.%s.%s: a pattern must not be empty", k, l.name)
				}
				if strings.Trim(pat, "*") == "" {
					return Errf("import.machines.annotate.%s.%s: pattern %q has no literal part; a pattern names the events or states it is for", k, l.name, pat)
				}
			}
		}
		for ev, act := range a.Aliases {
			if strings.TrimSpace(ev) == "" || strings.TrimSpace(act) == "" {
				return Errf("import.machines.annotate.%s.aliases: an event and its action must not be empty", k)
			}
		}
		refused := make([]string, 0, len(a.RefusalEvents))
		for ev := range a.RefusalEvents {
			refused = append(refused, ev)
		}
		sort.Strings(refused)
		for _, ev := range refused {
			rf := a.RefusalEvents[ev]
			if strings.TrimSpace(ev) == "" || strings.TrimSpace(rf.Action) == "" {
				return Errf("import.machines.annotate.%s.refusal_events: an event and its action must not be empty", k)
			}
			if strings.TrimSpace(rf.Reason) == "" {
				return Errf("import.machines.annotate.%s.refusal_events.%s: reason is required: say why the event is a refusal and no failure", k, ev)
			}
		}
		for name, reason := range a.NotFailures {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(reason) == "" {
				return Errf("import.machines.annotate.%s.not_failures: a name and its reason must not be empty", k)
			}
		}
		// a name, an action and a reason are printed: each is one line
		texts := map[string][]string{}
		for ev, act := range a.Aliases {
			texts["aliases"] = append(texts["aliases"], ev, act)
		}
		for ev, rf := range a.RefusalEvents {
			texts["refusal_events"] = append(texts["refusal_events"], ev, rf.Action, rf.Reason)
		}
		for name, reason := range a.NotFailures {
			texts["not_failures"] = append(texts["not_failures"], name, reason)
		}
		for _, list := range []string{"aliases", "refusal_events", "not_failures"} {
			sort.Strings(texts[list])
			for _, text := range texts[list] {
				if !OneLine(text) {
					return Errf("import.machines.annotate.%s.%s: %q has a line break or a control character; a name, an action and a reason are one line", k, list, text)
				}
			}
		}
	}
	return nil
}

func (c *Config) validateImportedFailureModes() error {
	f := c.ImportedFailureModes()
	if f == nil {
		return nil
	}
	if f.File == "" {
		return Errf("import.failure_modes.file is required")
	}
	if strings.TrimSpace(f.ID) == "" {
		return Errf("import.failure_modes.id is required")
	}
	if _, err := regexp.Compile(f.ID); err != nil {
		return Errf("import.failure_modes.id is not a regular expression: %v", err)
	}
	for _, col := range []struct {
		name     string
		names    []string
		optional bool
	}{{"failure", f.Columns.Failure, false}, {"behavior", f.Columns.Behavior, false}, {"required", f.Columns.Required, true}} {
		if len(col.names) == 0 && !col.optional {
			return Errf("import.failure_modes.columns.%s is required", col.name)
		}
		for _, n := range col.names {
			if strings.TrimSpace(n) == "" {
				return Errf("import.failure_modes.columns.%s: a header name must not be empty", col.name)
			}
		}
	}
	if err := CheckPath("import.failure_modes.file", f.File); err != nil {
		return err
	}
	return CheckPath("import.failure_modes.links", f.Links)
}

// Default routing texts (section 2).
var DefaultRouting = map[string]string{
	"routine":       "standard model",
	"consequential": "standard model; fresh verifier of at least the builder's class",
	"critical":      "strongest model for the formal model; standard model for code; fresh verifier of at least the builder's class",
	"vital":         "strongest model for the formal model and the core; standard model elsewhere; fresh verifier of at least the builder's class",
}

func (c *Config) applyDefaults() {
	if len(c.Scope) == 0 {
		c.Scope = []string{"**"}
	}
	if c.Design.Owed == "" {
		c.Design.Owed = "design/owed.yaml"
	}
	if c.Design.Status == "" {
		c.Design.Status = "design/status.yaml"
	}
	if c.Design.Gen == "" {
		c.Design.Gen = "design/gen"
	}
	if c.Design.Formal == "" {
		c.Design.Formal = "design/formal"
	}
	if c.Receipts.Dir == "" {
		c.Receipts.Dir = ".machinery/receipts"
	}
	if c.Trace != nil && len(c.Trace.Code) == 0 {
		c.Trace.Code = append([]string(nil), c.Scope...)
	}
	if c.Tests != nil && len(c.Tests.E2E) == 0 && c.Trace != nil {
		c.Tests.E2E = append([]string(nil), c.Trace.Tests...)
	}
	if c.Constraints != nil && c.Constraints.Size != nil && len(c.Constraints.Size.Paths) == 0 && c.Trace != nil {
		c.Constraints.Size.Paths = append([]string(nil), c.Trace.Code...)
	}
	if c.Backlog != nil && c.Backlog.Base == "" {
		c.Backlog.Base = "main"
	}
	if h := c.Hooks; h != nil {
		for _, f := range []struct {
			v   *string
			def string
		}{{&h.Commit, hookDefaults.Commit}, {&h.Push, hookDefaults.Push}, {&h.Stop, hookDefaults.Stop}} {
			if *f.v == "" {
				*f.v = f.def
			}
		}
	}
	if c.Review != nil || c.Backlog != nil {
		r := Review{}
		if c.Review != nil {
			r = *c.Review
		}
		def := ReviewOff
		if c.Backlog != nil {
			def = ReviewRequired
		}
		for _, v := range []*string{&r.Backlog, &r.Design} {
			if *v == "" {
				*v = def
			}
		}
		c.Review = &r
	}
	if c.Routing == nil {
		c.Routing = map[string]string{}
	}
	for k, v := range DefaultRouting {
		if c.Routing[k] == "" {
			c.Routing[k] = v
		}
	}
	t := &c.Tools
	if t.Modelith == "" {
		t.Modelith = "modelith"
	}
	if t.Java == "" {
		t.Java = "java"
	}
	if t.Node == "" {
		t.Node = "node"
	}
	if t.BD == "" {
		t.BD = "bd"
	}
	for i := range c.Checkers {
		if c.Checkers[i].Evidence == "" {
			c.Checkers[i].Evidence = "design/checkers/" + c.Checkers[i].ID + ".evidence.json"
		}
	}
}

// Abs returns the absolute path of a project-relative path.
func (p *Project) Abs(rel string) string { return filepath.Join(p.Root, filepath.FromSlash(rel)) }

// Rel returns the project-relative, slash-separated form of an absolute path.
func (p *Project) Rel(abs string) string {
	r, err := filepath.Rel(p.Root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(r)
}

// ToolsDir resolves the tools directory (section 2).
func (p *Project) ToolsDir() string { return ToolsDir(p.Cfg.Tools.Dir, p.Root) }

// ToolsDir resolves the tools directory from the config value, the environment
// and the project root, in that order.
func ToolsDir(configured, root string) string {
	if configured != "" {
		if filepath.IsAbs(configured) {
			return configured
		}
		return filepath.Join(root, configured)
	}
	if env := os.Getenv("MACHINERY_TOOLS"); env != "" {
		return env
	}
	return filepath.Join(root, ".machinery", "tools")
}

// TestRoots returns the e2e roots (tests.e2e, else trace.tests).
func (p *Project) TestRoots() []string {
	if p.Cfg.Tests != nil && len(p.Cfg.Tests.E2E) > 0 {
		return p.Cfg.Tests.E2E
	}
	if p.Cfg.Trace != nil {
		return p.Cfg.Trace.Tests
	}
	return nil
}

// Marshal renders a config as YAML.
func Marshal(c *Config) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	enc.Close()
	return buf.Bytes(), nil
}
