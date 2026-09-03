package checker

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InventoryProblems performs the reverse side of checker ownership: every
// committed projection/evidence artifact must be owned by a manifest, and a
// top-level checker implementation directory must correspond to a checker id.
// Unrecognized source files inside an owned checker directory are intentionally
// allowed (adapters and native rule files are checker-defined).
func InventoryProblems(design string, manifests []*Manifest) []string {
	root, err := filepath.Abs(filepath.Join(design, "checkers"))
	if err != nil {
		return []string{err.Error()}
	}
	expectedFiles := map[string]bool{}
	expectedDirs := map[string]bool{}
	generatedRoots := map[string]bool{}
	var problems []string
	for _, manifest := range manifests {
		manifestPath, absErr := filepath.Abs(manifest.Path)
		if absErr != nil {
			problems = append(problems, absErr.Error())
			continue
		}
		expectedFiles[portablePath(manifestPath)] = true
		expectedDirs[portablePath(filepath.Join(root, manifest.Checker.ID))] = true
		for _, rel := range []string{manifest.Evidence.ProjectionOut, manifest.Evidence.EvidenceIn} {
			resolved, resolveErr := ConfinedPath(design, rel)
			if resolveErr != nil {
				problems = append(problems, fmt.Sprintf("checker %s output %q is unsafe: %v", manifest.Checker.ID, rel, resolveErr))
				continue
			}
			expectedFiles[portablePath(resolved)] = true
			for dir := filepath.Dir(resolved); pathWithin(root, dir); dir = filepath.Dir(dir) {
				expectedDirs[portablePath(dir)] = true
				if portablePath(dir) == portablePath(root) {
					break
				}
			}
		}
		evidencePath, resolveErr := ConfinedPath(design, manifest.Evidence.EvidenceIn)
		if resolveErr == nil {
			generatedRoots[portablePath(filepath.Join(filepath.Dir(evidencePath), "generated"))] = true
			if evidence, loadErr := LoadEvidence(evidencePath); loadErr == nil && evidence.TraceRef != "" {
				traceRel := filepath.Join(filepath.Dir(manifest.Evidence.EvidenceIn), evidence.TraceRef)
				trace, traceErr := ConfinedPath(design, traceRel)
				if traceErr != nil {
					problems = append(problems, fmt.Sprintf("checker %s trace_ref %q is unsafe: %v", manifest.Checker.ID, evidence.TraceRef, traceErr))
				} else {
					expectedFiles[portablePath(trace)] = true
				}
			}
		}
	}

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			problems = append(problems, "checker inventory contains symlink "+relativeInventoryPath(root, path))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if filepath.Dir(path) == root && !expectedDirs[portablePath(path)] {
				problems = append(problems, "orphan checker directory "+relativeInventoryPath(root, path)+" has no manifest checker.id owner")
				return filepath.SkipDir
			}
			return nil
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return infoErr
		}
		if !info.Mode().IsRegular() {
			problems = append(problems, "checker inventory entry "+relativeInventoryPath(root, path)+" is not a regular file")
			return nil
		}
		if expectedFiles[portablePath(path)] {
			return nil
		}
		for generatedRoot := range generatedRoots {
			candidate := portablePath(path)
			if candidate == generatedRoot || strings.HasPrefix(candidate, generatedRoot+"/") {
				problems = append(problems, "orphan checker-generated artifact "+relativeInventoryPath(root, path)+" is not declared by current evidence.trace_ref")
				return nil
			}
		}
		if isCheckerOutputArtifact(path) {
			problems = append(problems, "orphan checker output "+relativeInventoryPath(root, path)+" is not declared by any manifest")
		}
		return nil
	})
	if walkErr != nil {
		problems = append(problems, "walk checker inventory: "+walkErr.Error())
	}
	sort.Strings(problems)
	return problems
}

func isCheckerOutputArtifact(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if base == "projection.json" || base == "evidence.json" {
		return true
	}
	if filepath.Ext(base) != ".json" {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return false
	}
	return object["projection_schema"] != nil || object["evidence_schema"] != nil
}

func portablePath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func relativeInventoryPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(filepath.Join("checkers", rel))
}
