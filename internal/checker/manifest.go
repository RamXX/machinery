package checker

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Residual is a claimed element the checker cannot decide, waived with a reason.
// Coverage is a hard rule: a claimed element is covered by evidence or a residual,
// never silently dropped.
type Residual struct {
	ID     string `yaml:"id"`
	Reason string `yaml:"reason"`
}

// Manifest is the tool-neutral contract a design commits under
// design/checkers/<id>.checker.yaml. It names no binary: the registry resolves
// that outside the design (see docs/external-checkers.md).
type Manifest struct {
	Checker struct {
		ID          string `yaml:"id"`
		Description string `yaml:"description"`
	} `yaml:"checker"`
	Projection struct {
		Include  []string `yaml:"include"`
		Requires []string `yaml:"requires"`
	} `yaml:"projection"`
	Coverage struct {
		Claim     []string   `yaml:"claim"`
		Residuals []Residual `yaml:"residuals"`
	} `yaml:"coverage"`
	// Config is opaque to machinery: it is passed through to the checker's adapter
	// (for example, which attributes are sensitive). Keeping it here lets the
	// projection stay generic while the checker supplies its own domain knowledge.
	Config   map[string]any `yaml:"config"`
	Evidence struct {
		ProjectionOut string `yaml:"projection_out"`
		EvidenceIn    string `yaml:"evidence_in"`
	} `yaml:"evidence"`
	EmitsOracle bool `yaml:"emits_oracle"`

	// Path is the manifest file path (set by LoadManifest, not from YAML).
	Path string `yaml:"-"`
}

// LoadManifest parses and validates a checker manifest. Absence, malformed YAML,
// or a missing required field is an error: an unusable manifest must fail loudly,
// never degrade to a silent skip.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	m.Path = path
	if m.Checker.ID == "" {
		return nil, fmt.Errorf("%s: checker.id is required", path)
	}
	if len(m.Projection.Include) == 0 {
		return nil, fmt.Errorf("%s: projection.include must name at least one layer", path)
	}
	if m.Evidence.ProjectionOut == "" || m.Evidence.EvidenceIn == "" {
		return nil, fmt.Errorf("%s: evidence.projection_out and evidence.evidence_in are required", path)
	}
	return &m, nil
}
