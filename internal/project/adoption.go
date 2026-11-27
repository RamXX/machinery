package project

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AdoptionName is the file machinery adopt owns: what adoption added to the
// configuration of a project whose machinery.yaml it never edits (section 2.1).
const AdoptionName = "design/adoption.yaml"

// Adoption mirrors design/adoption.yaml: the keys adopt adds, each meaning
// what it means in machinery.yaml. Field order is the order adopt writes.
type Adoption struct {
	Version     int                 `yaml:"version"`
	Scope       []string            `yaml:"scope,omitempty"`
	Design      AdoptionDesign      `yaml:"design,omitempty"`
	Contracts   []Contract          `yaml:"contracts,omitempty"`
	Trace       AdoptionTrace       `yaml:"trace,omitempty"`
	Tests       AdoptionTests       `yaml:"tests,omitempty"`
	Constraints AdoptionConstraints `yaml:"constraints,omitempty"`
	Backlog     AdoptionBacklog     `yaml:"backlog,omitempty"`
	Waivers     []Waiver            `yaml:"waivers,omitempty"`
}

type AdoptionDesign struct {
	Models   []string `yaml:"models,omitempty"`
	Machines string   `yaml:"machines,omitempty"`
}

type AdoptionTrace struct {
	Code  []string `yaml:"code,omitempty"`
	Tests []string `yaml:"tests,omitempty"`
}

type AdoptionTests struct {
	E2E      []string `yaml:"e2e,omitempty"`
	Baseline string   `yaml:"baseline,omitempty"`
}

type AdoptionConstraints struct {
	Baseline string `yaml:"baseline,omitempty"`
}

type AdoptionBacklog struct {
	Tool string `yaml:"tool,omitempty"`
}

// ReadAdoption reads design/adoption.yaml of the project at root. It returns
// nil when there is none, and never nil for a file that exists.
func ReadAdoption(root string) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(AdoptionName)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, Errf("%s: %v", AdoptionName, err)
	}
	return append([]byte{}, data...), nil
}

// ParseAdoption decodes design/adoption.yaml: one document, version 1, only
// the keys of Adoption.
func ParseAdoption(data []byte) (*Adoption, error) {
	var a Adoption
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&a); err != nil && !errors.Is(err, io.EOF) {
		return nil, Errf("%s: %s", AdoptionName, cleanYAMLError(err))
	}
	if a.Version != 1 {
		return nil, Errf("%s: version must be 1", AdoptionName)
	}
	var more yaml.Node
	if err := dec.Decode(&more); !errors.Is(err, io.EOF) {
		return nil, Errf("%s: the file holds more than one YAML document", AdoptionName)
	}
	return &a, nil
}

// Add merges b into a as the configuration merges the file (section 2.1): a
// list gets the items of b after its own, a scalar is the one that is set,
// and one set in both with two values is an error naming the two files.
func (a *Adoption) Add(b *Adoption, aName, bName string) error {
	a.Scope = append(a.Scope, b.Scope...)
	a.Design.Models = append(a.Design.Models, b.Design.Models...)
	a.Contracts = append(a.Contracts, b.Contracts...)
	a.Trace.Code = append(a.Trace.Code, b.Trace.Code...)
	a.Trace.Tests = append(a.Trace.Tests, b.Trace.Tests...)
	a.Tests.E2E = append(a.Tests.E2E, b.Tests.E2E...)
	a.Waivers = append(a.Waivers, b.Waivers...)
	for _, s := range []struct {
		key      string
		dst, src *string
	}{
		{"design.machines", &a.Design.Machines, &b.Design.Machines},
		{"tests.baseline", &a.Tests.Baseline, &b.Tests.Baseline},
		{"constraints.baseline", &a.Constraints.Baseline, &b.Constraints.Baseline},
		{"backlog.tool", &a.Backlog.Tool, &b.Backlog.Tool},
	} {
		switch {
		case *s.src == "" || *s.dst == *s.src:
		case *s.dst == "":
			*s.dst = *s.src
		default:
			return Errf("%s: %s has %q and %s has %q: keep one", s.key, aName, *s.dst, bName, *s.src)
		}
	}
	return nil
}

// part returns what of the configuration design/adoption.yaml can hold.
func (c *Config) part() *Adoption {
	a := &Adoption{Scope: c.Scope, Contracts: c.Contracts, Waivers: c.Waivers}
	a.Design = AdoptionDesign{Models: c.Design.Models, Machines: c.Design.Machines}
	if c.Trace != nil {
		a.Trace = AdoptionTrace{Code: c.Trace.Code, Tests: c.Trace.Tests}
	}
	if c.Tests != nil {
		a.Tests = AdoptionTests{E2E: c.Tests.E2E, Baseline: c.Tests.Baseline}
	}
	if c.Constraints != nil {
		a.Constraints.Baseline = c.Constraints.Baseline
	}
	if c.Backlog != nil {
		a.Backlog.Tool = c.Backlog.Tool
	}
	return a
}

// merge puts design/adoption.yaml into the configuration read from
// machinery.yaml (section 2.1). A block only the file has becomes a block of
// the configuration.
func (c *Config) merge(b *Adoption) error {
	a := c.part()
	if err := a.Add(b, ConfigName, AdoptionName); err != nil {
		return err
	}
	c.Scope, c.Contracts, c.Waivers = a.Scope, a.Contracts, a.Waivers
	c.Design.Models, c.Design.Machines = a.Design.Models, a.Design.Machines
	if len(a.Trace.Code)+len(a.Trace.Tests) > 0 {
		if c.Trace == nil {
			c.Trace = &Trace{}
		}
		c.Trace.Code, c.Trace.Tests = a.Trace.Code, a.Trace.Tests
	}
	if len(a.Tests.E2E) > 0 || a.Tests.Baseline != "" {
		if c.Tests == nil {
			c.Tests = &Tests{}
		}
		c.Tests.E2E, c.Tests.Baseline = a.Tests.E2E, a.Tests.Baseline
	}
	if a.Constraints.Baseline != "" {
		if c.Constraints == nil {
			c.Constraints = &Constraints{}
		}
		c.Constraints.Baseline = a.Constraints.Baseline
	}
	if a.Backlog.Tool != "" {
		if c.Backlog == nil {
			c.Backlog = &Backlog{}
		}
		c.Backlog.Tool = a.Backlog.Tool
	}
	return nil
}

// Raw decodes machinery.yaml and merges design/adoption.yaml into it
// (adoption nil: there is none). Nothing is validated and no default is
// applied: the result is the configuration as the two files wrote it.
func Raw(data, adoption []byte) (*Config, error) {
	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, Errf("%s: %s", ConfigName, cleanYAMLError(err))
	}
	if adoption == nil {
		return &cfg, nil
	}
	a, err := ParseAdoption(adoption)
	if err != nil {
		return nil, err
	}
	if err := cfg.merge(a); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ParseWith decodes machinery.yaml with design/adoption.yaml merged into it
// (nil: there is none), validates the result and applies defaults.
func ParseWith(data, adoption []byte) (*Config, error) {
	cfg, err := Raw(data, adoption)
	if err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	cfg.applyDefaults()
	return cfg, nil
}

// Present returns the bytes of a file that exists, never nil, and nil for
// one that does not: the form Raw and ParseWith take design/adoption.yaml in.
func Present(data []byte, ok bool) []byte {
	if !ok {
		return nil
	}
	return append([]byte{}, data...)
}
