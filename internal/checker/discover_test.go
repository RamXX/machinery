package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManifestPathsAndHasCheckers(t *testing.T) {
	design := t.TempDir()
	if HasCheckers(design) {
		t.Fatal("no checkers/ dir should mean no checkers")
	}
	if got := ManifestPaths(design); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}

	dir := filepath.Join(design, "checkers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// two manifests plus a non-manifest file that must be ignored
	for _, n := range []string{"b.checker.yaml", "a.checker.yaml", "notes.md"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := ManifestPaths(design)
	if len(got) != 2 {
		t.Fatalf("expected 2 manifests, got %v", got)
	}
	// sorted order
	if !strings.HasSuffix(got[0], "a.checker.yaml") || !strings.HasSuffix(got[1], "b.checker.yaml") {
		t.Fatalf("manifests not sorted: %v", got)
	}
	if !HasCheckers(design) {
		t.Fatal("HasCheckers should be true now")
	}
}

func TestDesignID(t *testing.T) {
	p := writeTemp(t, "d.modelith.yaml", sampleModel)
	id, err := DesignID(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, "sha256:") || len(id) != len("sha256:")+64 {
		t.Fatalf("unexpected design id: %q", id)
	}
	// same bytes hash the same
	id2, _ := DesignID(p)
	if id != id2 {
		t.Fatal("DesignID not stable")
	}
	if _, err := DesignID(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error on missing file")
	}
}

func TestLoadEvidenceOK(t *testing.T) {
	body := `{"evidence_schema":"1.0","checker":{"id":"c","version":"1"},"input_hash":"sha256:abc","verdict":"pass","coverage":[{"element":"inv:x","verdict":"pass"}]}`
	ev, err := LoadEvidence(writeTemp(t, "e.json", body))
	if err != nil {
		t.Fatal(err)
	}
	if ev.Verdict != "pass" || ev.Checker.ID != "c" || len(ev.Coverage) != 1 {
		t.Fatalf("evidence parsed wrong: %+v", ev)
	}
	if _, err := LoadEvidence(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error on missing evidence")
	}
}
