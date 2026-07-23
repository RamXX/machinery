package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectAll(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := "checker: {id: test}\n" +
		"projection: {include: [model, invariants, relationships]}\n" +
		"evidence: {projection_out: checkers/test/projection.json, evidence_in: checkers/test/evidence.json}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := ProjectAll(design, "v0")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].CheckerID != "test" || results[0].Path != "checkers/test/projection.json" {
		t.Fatalf("unexpected results: %+v", results)
	}

	// ProjectAll created the nested output directory and wrote a valid projection
	b, err := os.ReadFile(filepath.Join(design, "checkers", "test", "projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProjection(b)
	if err != nil {
		t.Fatal(err)
	}
	h, err := p.InputHash()
	if err != nil {
		t.Fatal(err)
	}
	// the generated block mirrors the binding hash for adapters
	var raw struct {
		Generated map[string]string `json:"generated"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Generated["input_hash"] != h {
		t.Fatalf("committed projection mirror %q != fresh hash %q", raw.Generated["input_hash"], h)
	}
}

func TestProjectAllNoModelErrors(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	// a manifest but no *.modelith.yaml to project from
	man := "checker: {id: test}\nprojection: {include: [model]}\nevidence: {projection_out: p, evidence_in: e}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v0"); err == nil {
		t.Fatal("expected an error projecting with no domain model")
	}
}

func TestProjectAllPropagatesUnsupportedLayer(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(sampleModel), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(design, "checkers"), 0o755); err != nil {
		t.Fatal(err)
	}
	man := "checker: {id: test}\nprojection: {include: [model, machines]}\nevidence: {projection_out: p, evidence_in: e}\n"
	if err := os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ProjectAll(design, "v0"); err == nil {
		t.Fatal("expected an error from an unsupported include layer")
	}
}
