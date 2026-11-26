// Package project finds, loads and validates machinery.yaml (docs/formats.md section 2).
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
	Trace        *Trace            `yaml:"trace,omitempty"`
	Tests        *Tests            `yaml:"tests,omitempty"`
	Constraints  *Constraints      `yaml:"constraints,omitempty"`
	Receipts     Receipts          `yaml:"receipts"`
	Commands     Commands          `yaml:"commands,omitempty"`
	Backlog      *Backlog          `yaml:"backlog,omitempty"`
	Routing      map[string]string `yaml:"routing,omitempty"`
	Tools        Tools             `yaml:"tools"`
	Waivers      []Waiver          `yaml:"waivers,omitempty"`
	Assess       *Assess           `yaml:"assess,omitempty"`
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
	Gen          string   `yaml:"gen,omitempty"`
}

// Import names design sources kept in the project's own formats (section 2).
type Import struct {
	Machines     *ImportMachines     `yaml:"machines,omitempty"`
	FailureModes *ImportFailureModes `yaml:"failure_modes,omitempty"`
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
	Files  []string        `yaml:"files"`
	States string          `yaml:"states,omitempty"`
	Rename Rename          `yaml:"rename,omitempty"`
	Bind   map[string]Bind `yaml:"bind,omitempty"`
}

// Bind is what an imported machine binds to in the model (section 5.2.1):
// the entity of a lifecycle machine, or the ids an operational machine serves.
type Bind struct {
	LifecycleOf string   `yaml:"lifecycle_of,omitempty"`
	Serves      []string `yaml:"serves,omitempty"`
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
	Baseline string  `yaml:"baseline,omitempty"`
}

type Receipts struct {
	Dir string `yaml:"dir,omitempty"`
}

type Commands struct {
	E2E []string `yaml:"e2e,omitempty"`
}

type Backlog struct {
	Tool string `yaml:"tool"`
	Base string `yaml:"base,omitempty"`
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
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return &Project{Root: root, Cfg: *cfg}, nil
}

// Parse decodes and validates a config document and applies defaults.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, Errf("%s: %s", ConfigName, cleanYAMLError(err))
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func cleanYAMLError(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "yaml: unmarshal errors:\n  ")
	msg = strings.TrimPrefix(msg, "yaml: ")
	return strings.ReplaceAll(msg, "\n  ", "; ")
}

var (
	checkerIDRe = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
	// WaivableRules are the cross-reference rules a waiver may name (section 6.7).
	WaivableRules = map[string]bool{
		"machine-binding": true, "state-enum": true, "event-action": true, "action-event": true,
		"operation-id": true, "operation-action": true, "action-operation": true,
		"subject-resolves": true, "invoke-failure-mode": true, "dependency-failure-mode": true,
		"tiered-action-abuse": true, "failure-state-mode": true, "failure-event-mode": true,
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
		check("design", d.Machines, d.FailureModes, d.Risk, d.Formal, d.Facts, d.Owed, d.Gen),
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
	if c.Trace != nil {
		steps = append(steps, check("trace.code", c.Trace.Code...), check("trace.tests", c.Trace.Tests...))
		if m := c.Trace.Markers; m != nil {
			for _, kind := range []struct {
				name    string
				markers []string
			}{{"serves", m.Serves}, {"verifies", m.Verifies}, {"scenarios", m.Scenarios}} {
				for _, mk := range kind.markers {
					if strings.TrimSpace(mk) == "" {
						return Errf("trace.markers.%s: a marker must not be empty", kind.name)
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
		steps = append(steps, check("constraints.baseline", c.Constraints.Baseline))
	}
	if err := c.validateImport(); err != nil {
		return err
	}
	if err := c.validateAssess(); err != nil {
		return err
	}
	if c.Backlog != nil && c.Backlog.Tool != "bd" {
		return Errf("backlog.tool: only bd is supported, got %q", c.Backlog.Tool)
	}
	for k := range c.Routing {
		if k != "routine" && k != "consequential" && k != "critical" {
			return Errf("routing: unknown tier %q", k)
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
	if im.States != "" && im.States != StatesKeep && im.States != StatesCapitalize {
		return Errf("import.machines.states must be capitalize or keep, got %q", im.States)
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
		}
		for _, id := range b.Serves {
			if strings.TrimSpace(id) == "" {
				return Errf("import.machines.bind.%s.serves: an id must not be empty", k)
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
		name  string
		names []string
	}{{"failure", f.Columns.Failure}, {"behavior", f.Columns.Behavior}} {
		if len(col.names) == 0 {
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
	"routine":       "small model",
	"consequential": "standard model; fresh verifier",
	"critical":      "strongest model for the formal model; standard model for code; fresh verifier",
}

func (c *Config) applyDefaults() {
	if len(c.Scope) == 0 {
		c.Scope = []string{"**"}
	}
	if c.Design.Owed == "" {
		c.Design.Owed = "design/owed.yaml"
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
