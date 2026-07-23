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
func ManifestPaths(design string) []string {
	dir := filepath.Join(design, "checkers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".checker.yaml") {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// HasCheckers reports whether the design opted into the layer (any manifest).
func HasCheckers(design string) bool { return len(ManifestPaths(design)) > 0 }

// ModelPath returns the design's *.modelith.yaml source, or "" if there is none.
// The directory is listed (not globbed) so a design path with glob metacharacters
// cannot defeat detection.
func ModelPath(design string) string {
	entries, err := os.ReadDir(design)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".modelith.yaml") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return filepath.Join(design, names[0])
}

func setOf(xs []string) map[string]bool {
	m := make(map[string]bool, len(xs))
	for _, x := range xs {
		m[x] = true
	}
	return m
}
