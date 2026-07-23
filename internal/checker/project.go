package checker

import (
	"fmt"
	"os"
	"path/filepath"
)

// ProjectResult records one manifest's generated projection.
type ProjectResult struct {
	CheckerID string
	Path      string // projection file written, relative to the design
}

// ProjectAll generates and writes the committed projection for every checker
// manifest in the design. It is the write side of the gk contract: the gate
// reads exactly what this produced, so `machinery project` then `git commit` is
// the loop a builder runs before the checker's adapter consumes the projection.
func ProjectAll(design, machineryVersion string) ([]ProjectResult, error) {
	modelPath := ModelPath(design)
	if modelPath == "" {
		return nil, fmt.Errorf("no *.modelith.yaml in %s; nothing to project", design)
	}
	model, err := LoadModel(modelPath)
	if err != nil {
		return nil, err
	}
	designID, err := DesignID(modelPath)
	if err != nil {
		return nil, err
	}
	var out []ProjectResult
	for _, mp := range ManifestPaths(design) {
		man, err := LoadManifest(mp)
		if err != nil {
			return nil, err
		}
		proj, err := Generate(model, man, designID, machineryVersion)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", man.Checker.ID, err)
		}
		rendered, err := proj.Render()
		if err != nil {
			return nil, err
		}
		dest := filepath.Join(design, man.Evidence.ProjectionOut)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dest, rendered, 0o644); err != nil {
			return nil, err
		}
		out = append(out, ProjectResult{CheckerID: man.Checker.ID, Path: man.Evidence.ProjectionOut})
	}
	return out, nil
}
