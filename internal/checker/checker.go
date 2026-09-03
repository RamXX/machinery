// Package checker implements the pluggable external-checker layer: the
// deterministic half machinery runs in `machinery check --gate gk`. It projects
// a design into the canonical projection contract, reconciles a checker manifest
// against the model, and verifies that committed evidence binds to the current
// design. It never runs an external engine; that is the verify-checkers phase.
//
// The two committed contracts are schemas/projection.schema.json (machinery ->
// checker) and schemas/evidence.schema.json (checker -> machinery); this package
// is their Go realization. See docs/external-checkers.md for the builder guide.
package checker

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaVersion is the contract version this package produces and accepts.
const SchemaVersion = "1.0"

// encodeJSON renders v deterministically: struct field order is fixed, map keys
// are sorted by encoding/json, and HTML escaping is off so a stable_id like
// "rel:A->B" stays readable. The trailing newline the encoder adds is trimmed;
// callers re-add one for on-disk files.
func encodeJSON(v any, indent bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func sha256Prefixed(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// DesignID hashes the source artifacts a projection derives from. v1 projects
// only the domain model, so it hashes that file. It is provenance carried into
// the projection, not the binding hash (that is Projection.InputHash).
func DesignID(modelPath string) (string, error) {
	b, err := os.ReadFile(modelPath)
	if err != nil {
		return "", err
	}
	return sha256Prefixed(b), nil
}

// ManifestPaths returns the sorted *.checker.yaml files under design/checkers.
// The directory is listed rather than globbed so a design path containing glob
// metacharacters cannot defeat detection (the GATE-2 lesson).
func ManifestPaths(design string) ([]string, error) {
	dir := filepath.Join(design, "checkers")
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("inspect checker directory %s: %w", dir, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("checker directory %s must not be a symlink", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("checker path %s is not a directory", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read checker directory %s: %w", dir, err)
	}
	var out []string
	portableNames := map[string]string{}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".checker.yaml") {
			continue
		}
		if err := validatePortableComponent(e.Name()); err != nil {
			return nil, fmt.Errorf("checker manifest name %q is not portable: %w", e.Name(), err)
		}
		path := filepath.Join(dir, e.Name())
		entryInfo, lerr := os.Lstat(path)
		if lerr != nil {
			return nil, fmt.Errorf("inspect checker manifest %s: %w", path, lerr)
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 || !entryInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("checker manifest %s must be a regular, non-symlink file", path)
		}
		folded := strings.ToLower(e.Name())
		if prior, exists := portableNames[folded]; exists {
			return nil, fmt.Errorf("checker manifests %q and %q collide on a case-insensitive filesystem", prior, e.Name())
		}
		portableNames[folded] = e.Name()
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
}

// HasCheckers reports whether the design opted into the layer (any manifest).
func HasCheckers(design string) (bool, error) {
	dir := filepath.Join(design, "checkers")
	_, statErr := os.Lstat(dir)
	if os.IsNotExist(statErr) {
		return false, nil
	}
	paths, err := ManifestPaths(design)
	return len(paths) > 0 || statErr == nil, err
}

// ModelPath returns the design's *.modelith.yaml source, or "" if there is none.
// The directory is listed (not globbed) so a design path with glob metacharacters
// cannot defeat detection.
func ModelPath(design string) string {
	paths := ModelPaths(design)
	if len(paths) != 1 {
		return ""
	}
	return paths[0]
}

// ModelPaths returns every root Modelith model in deterministic name order.
// Callers that project a design require exactly one; silently choosing the
// first lets an arbitrary filesystem name decide which domain is checked.
func ModelPaths(design string) []string {
	entries, err := os.ReadDir(design)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".modelith.yaml") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, filepath.Join(design, name))
	}
	return out
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
