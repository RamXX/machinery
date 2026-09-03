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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SchemaVersion is the contract version this package produces and accepts.
const SchemaVersion = "1.0"

// checkerStructuredFileMaxBytes bounds every model, manifest, registry,
// projection, and evidence document before a parser can allocate from
// attacker-controlled cardinalities. Opaque checker traces have their own
// caller-supplied bound.
const checkerStructuredFileMaxBytes int64 = 16 << 20

const (
	checkerDirectoryBatch = 256
	checkerMaxEntries     = 65_536
)

func readCheckerStructuredFile(path, kind string) (data []byte, retErr error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: %s must be a regular, non-symlink file", path, kind)
	}
	if before.Size() < 0 || before.Size() > checkerStructuredFileMaxBytes {
		return nil, fmt.Errorf("%s: %s exceeds %d-byte limit", path, kind, checkerStructuredFileMaxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("%s: %s changed before it was opened", path, kind)
	}
	data, err = io.ReadAll(io.LimitReader(file, opened.Size()+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) != opened.Size() {
		return nil, fmt.Errorf("%s: %s changed beyond its exact %d-byte snapshot", path, kind, opened.Size())
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	pathAfter, pathErr := os.Lstat(path)
	if pathErr != nil {
		return nil, pathErr
	}
	if pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.Mode().IsRegular() ||
		!os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) ||
		opened.Size() != after.Size() || opened.Size() != pathAfter.Size() ||
		opened.Mode() != after.Mode() || opened.Mode() != pathAfter.Mode() ||
		!opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		projectionControlChangeID(opened) != projectionControlChangeID(after) || projectionControlChangeID(opened) != projectionControlChangeID(pathAfter) {
		return nil, fmt.Errorf("%s: %s changed while it was read", path, kind)
	}
	return data, nil
}

func readDirectoryBounded(path string, maxEntries int) ([]os.DirEntry, error) {
	if maxEntries <= 0 {
		return nil, fmt.Errorf("directory entry limit must be positive")
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("directory %s must be a real directory", path)
	}
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	opened, err := dir.Stat()
	if err != nil || !opened.IsDir() || !os.SameFile(before, opened) || before.Mode() != opened.Mode() {
		return nil, errors.Join(err, fmt.Errorf("directory %s changed while opening", path), dir.Close())
	}
	entries := make([]os.DirEntry, 0, min(checkerDirectoryBatch, maxEntries))
	var readErr error
	for {
		batch, err := dir.ReadDir(checkerDirectoryBatch)
		if len(batch) > maxEntries-len(entries) {
			readErr = fmt.Errorf("directory %s exceeds %d-entry limit", path, maxEntries)
			break
		}
		entries = append(entries, batch...)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			readErr = err
			break
		}
	}
	after, statErr := dir.Stat()
	pathAfter, pathErr := os.Lstat(path)
	closeErr := dir.Close()
	if err := errors.Join(readErr, statErr, pathErr, closeErr); err != nil {
		return nil, err
	}
	if pathAfter.Mode()&os.ModeSymlink != 0 || !pathAfter.IsDir() ||
		!os.SameFile(opened, after) || !os.SameFile(opened, pathAfter) || opened.Mode() != after.Mode() || opened.Mode() != pathAfter.Mode() ||
		!opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(pathAfter.ModTime()) ||
		projectionControlChangeID(opened) != projectionControlChangeID(after) || projectionControlChangeID(opened) != projectionControlChangeID(pathAfter) {
		return nil, fmt.Errorf("directory %s changed while being inventoried", path)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

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
	b, err := readCheckerStructuredFile(modelPath, "model")
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
	entries, err := readDirectoryBounded(dir, checkerMaxEntries)
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
	entries, err := readDirectoryBounded(design, checkerMaxEntries)
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
