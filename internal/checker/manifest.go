package checker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"

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
		ID             string `yaml:"id"`
		Description    string `yaml:"description"`
		RuntimeClosure string `yaml:"runtime_closure"`
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

// ValidateManifestSet applies constraints that can only be decided across the
// whole checker inventory. The folded comparisons model case-insensitive
// APFS/NTFS behavior even when machinery is currently running on Linux.
func ValidateManifestSet(manifests []*Manifest) error {
	ids := map[string]string{}
	targets := map[string]string{}
	for _, manifest := range manifests {
		id := manifest.Checker.ID
		foldedID := strings.ToLower(id)
		if prior, exists := ids[foldedID]; exists {
			return fmt.Errorf("checker ids %q and %q collide on a case-insensitive filesystem", prior, id)
		}
		ids[foldedID] = id
		for _, output := range []struct{ field, rel string }{
			{"projection_out", manifest.Evidence.ProjectionOut},
			{"evidence_in", manifest.Evidence.EvidenceIn},
		} {
			portable := strings.ToLower(filepath.ToSlash(filepath.Clean(output.rel)))
			label := id + "." + output.field + " (" + output.rel + ")"
			if prior, exists := targets[portable]; exists {
				return fmt.Errorf("checker output paths %s and %s alias on a case-insensitive filesystem", prior, label)
			}
			targets[portable] = label
		}
	}
	return nil
}

// LoadManifest parses and validates a checker manifest. Absence, malformed YAML,
// or a missing required field is an error: an unusable manifest must fail loudly,
// never degrade to a silent skip.
func LoadManifest(path string) (*Manifest, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: checker manifest must be a regular, non-symlink file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m, err := parseManifest(path, data)
	if err != nil {
		return nil, err
	}
	design := filepath.Dir(filepath.Dir(path))
	for _, output := range []struct{ field, rel string }{
		{"evidence.projection_out", m.Evidence.ProjectionOut},
		{"evidence.evidence_in", m.Evidence.EvidenceIn},
	} {
		if _, err := ConfinedPath(design, output.rel); err != nil {
			return nil, fmt.Errorf("%s: %s: %w", path, output.field, err)
		}
	}
	return m, nil
}

func parseManifest(path string, data []byte) (*Manifest, error) {
	var doc yaml.Node
	nodeDecoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := nodeDecoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: empty YAML document", path)
	}
	if err := rejectInvalidYAMLMappingKeys(doc.Content[0], "$manifest"); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var extra yaml.Node
	if err := nodeDecoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("%s: multiple YAML documents are not allowed", path)
		}
		return nil, fmt.Errorf("%s: trailing YAML data: %w", path, err)
	}
	var m Manifest
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&m); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	m.Path = path
	if m.Checker.ID == "" {
		return nil, fmt.Errorf("%s: checker.id is required", path)
	}
	if err := validatePortableComponent(m.Checker.ID); err != nil {
		return nil, fmt.Errorf("%s: checker.id %q is not portable: %w", path, m.Checker.ID, err)
	}
	if !validSHA256(m.Checker.RuntimeClosure) {
		return nil, fmt.Errorf("%s: checker.runtime_closure must be a lowercase sha256:<64 hex> OCI closure digest", path)
	}
	if m.EmitsOracle {
		return nil, fmt.Errorf("%s: emits_oracle=true is reserved until the manifest declares an owned oracle path and Gk can verify it", path)
	}
	if len(m.Projection.Include) == 0 {
		return nil, fmt.Errorf("%s: projection.include must name at least one layer", path)
	}
	seen := map[string]bool{}
	for _, layer := range m.Projection.Include {
		if !knownLayers[layer] {
			return nil, fmt.Errorf("%s: projection.include names unknown layer %q", path, layer)
		}
		if seen[layer] {
			return nil, fmt.Errorf("%s: projection.include names layer %q more than once", path, layer)
		}
		seen[layer] = true
	}
	seenRequires := map[string]bool{}
	for _, requirement := range m.Projection.Requires {
		if strings.TrimSpace(requirement) == "" {
			return nil, fmt.Errorf("%s: projection.requires contains an empty requirement", path)
		}
		if seenRequires[requirement] {
			return nil, fmt.Errorf("%s: projection.requires names %q more than once", path, requirement)
		}
		seenRequires[requirement] = true
	}
	seenClaims := map[string]bool{}
	for _, claim := range m.Coverage.Claim {
		if strings.TrimSpace(claim) == "" {
			return nil, fmt.Errorf("%s: coverage.claim contains an empty pattern", path)
		}
		if seenClaims[claim] {
			return nil, fmt.Errorf("%s: coverage.claim names %q more than once", path, claim)
		}
		if _, err := pathpkg.Match(claim, "probe"); err != nil {
			return nil, fmt.Errorf("%s: coverage.claim pattern %q is invalid: %w", path, claim, err)
		}
		seenClaims[claim] = true
	}
	seenResiduals := map[string]bool{}
	for i, residual := range m.Coverage.Residuals {
		if strings.TrimSpace(residual.ID) == "" || strings.TrimSpace(residual.Reason) == "" {
			return nil, fmt.Errorf("%s: coverage.residuals[%d] requires non-empty id and reason", path, i)
		}
		if seenResiduals[residual.ID] {
			return nil, fmt.Errorf("%s: coverage.residuals names %q more than once", path, residual.ID)
		}
		seenResiduals[residual.ID] = true
	}
	if m.Evidence.ProjectionOut == "" || m.Evidence.EvidenceIn == "" {
		return nil, fmt.Errorf("%s: evidence.projection_out and evidence.evidence_in are required", path)
	}
	if filepath.Clean(m.Evidence.ProjectionOut) == filepath.Clean(m.Evidence.EvidenceIn) {
		return nil, fmt.Errorf("%s: evidence.projection_out and evidence.evidence_in must be different files", path)
	}
	ownerPrefix := "checkers/" + m.Checker.ID + "/"
	for _, rel := range []string{m.Evidence.ProjectionOut, m.Evidence.EvidenceIn} {
		if !strings.HasPrefix(filepath.ToSlash(filepath.Clean(rel)), ownerPrefix) {
			return nil, fmt.Errorf("%s: checker %q output %q must be owned under %s", path, m.Checker.ID, rel, ownerPrefix)
		}
	}
	return &m, nil
}

// ConfinedPath resolves a manifest path below design without following a
// symlink in any existing component. The leaf may be absent (projection_out
// normally is before `machinery project`), but every existing parent is
// inspected with Lstat. This closes both lexical traversal and symlink escape.
func ConfinedPath(design, rel string) (string, error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return "", err
	}
	defer root.close()
	clean, err := root.confinedRel(rel)
	if err != nil {
		return "", err
	}
	return root.display(clean), nil
}

// ReadConfinedFile performs validation and the read through one open design
// root. Unlike ConfinedPath followed by os.ReadFile, an intermediate directory
// swap cannot redirect this operation outside the design.
func ReadConfinedFile(design, rel string) (data []byte, retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	defer closeRoot(&retErr, root)
	clean, err := root.confinedRel(rel)
	if err != nil {
		return nil, err
	}
	return root.readRegular(clean, "checker artifact", false)
}

// ReadConfinedFileBounded reads one strict regular file through a rooted
// capability and refuses content larger than maxBytes. It is used for opaque
// checker artifacts whose format machinery does not parse but must bind
// exactly without allowing unbounded memory consumption.
func ReadConfinedFileBounded(design, rel string, maxBytes int64) (data []byte, retErr error) {
	root, err := openDesignRoot(design)
	if err != nil {
		return nil, err
	}
	defer closeRoot(&retErr, root)
	clean, err := root.confinedRel(rel)
	if err != nil {
		return nil, err
	}
	return root.readRegularBounded(clean, "checker artifact", false, maxBytes)
}

func validatePortableRelativePath(rel string) error {
	if rel == "" || filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return fmt.Errorf("must be a non-empty design-relative path")
	}
	if strings.Contains(rel, `\`) {
		return fmt.Errorf("must use '/' separators, not backslashes")
	}
	if len(rel) >= 2 && ((rel[0] >= 'A' && rel[0] <= 'Z') || (rel[0] >= 'a' && rel[0] <= 'z')) && rel[1] == ':' {
		return fmt.Errorf("windows drive-absolute paths are not allowed")
	}
	for _, component := range strings.Split(rel, "/") {
		if component == "." || component == ".." {
			return fmt.Errorf("path component %q is not allowed", component)
		}
		if err := validatePortableComponent(component); err != nil {
			return fmt.Errorf("path component %q is not portable: %w", component, err)
		}
	}
	return nil
}

func validatePortableComponent(component string) error {
	if component == "" {
		return fmt.Errorf("must be non-empty")
	}
	if component[len(component)-1] == '.' || component[len(component)-1] == ' ' {
		return fmt.Errorf("must not end in a dot or space")
	}
	for i, c := range component {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '_', c == '-':
			continue
		default:
			return fmt.Errorf("character %q at offset %d is outside [A-Za-z0-9._-]", c, i)
		}
	}
	stem := component
	if dot := strings.IndexByte(stem, '.'); dot >= 0 {
		stem = stem[:dot]
	}
	upper := strings.ToUpper(stem)
	reserved := map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true, "CLOCK$": true}
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
		reserved[upper] = true
	}
	if reserved[upper] {
		return fmt.Errorf("%q is a reserved Windows device basename", stem)
	}
	return nil
}

// SemanticHash binds every committed manifest field that can change what the
// checker receives, decides, covers, or emits. Set-like lists are normalized
// so harmless YAML reordering does not move the binding; config is encoded by
// encoding/json, whose map-key ordering is deterministic.
func (m *Manifest) SemanticHash() (string, error) {
	include := append([]string(nil), m.Projection.Include...)
	requires := append([]string(nil), m.Projection.Requires...)
	claim := append([]string(nil), m.Coverage.Claim...)
	residuals := append([]Residual(nil), m.Coverage.Residuals...)
	sort.Strings(include)
	sort.Strings(requires)
	sort.Strings(claim)
	sort.Slice(residuals, func(i, j int) bool {
		if residuals[i].ID != residuals[j].ID {
			return residuals[i].ID < residuals[j].ID
		}
		return residuals[i].Reason < residuals[j].Reason
	})
	binding := struct {
		Checker struct {
			ID, Description string
		}
		Include, Requires []string
		Claim             []string
		Residuals         []Residual
		Config            map[string]any
		ProjectionOut     string
		EvidenceIn        string
		EmitsOracle       bool
	}{}
	binding.Checker.ID = m.Checker.ID
	binding.Checker.Description = m.Checker.Description
	binding.Include, binding.Requires = include, requires
	binding.Claim, binding.Residuals = claim, residuals
	binding.Config = m.Config
	binding.ProjectionOut = filepath.ToSlash(filepath.Clean(m.Evidence.ProjectionOut))
	binding.EvidenceIn = filepath.ToSlash(filepath.Clean(m.Evidence.EvidenceIn))
	binding.EmitsOracle = m.EmitsOracle
	b, err := json.Marshal(binding)
	if err != nil {
		return "", fmt.Errorf("manifest semantic input is not canonical JSON: %w", err)
	}
	return sha256Prefixed(b), nil
}
