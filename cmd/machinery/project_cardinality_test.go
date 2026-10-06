package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
)

func TestProjectManyToManyAndCheckGk(t *testing.T) {
	design := filepath.Join(t.TempDir(), "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	copyDirInto(t, filepath.Join(repoRootDir(t), "examples", "pii-flow", "design"), design)
	modelPath := checker.ModelPaths(design)[0]
	body, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	model := strings.Replace(string(body), "cardinality: 1:n", "cardinality: n:n", 1)
	if model == string(body) {
		t.Fatal("fixture has no one-to-many relationship to replace")
	}
	if err := os.WriteFile(modelPath, []byte(model), 0o644); err != nil {
		t.Fatal(err)
	}
	out, stderr, code := runBin(t, "project", design)
	if code != 0 {
		t.Fatalf("project exited %d: %s%s", code, out, stderr)
	}
	projPath := filepath.Join(design, "checkers", "pii-flow", "projection.json")
	body, err = os.ReadFile(projPath)
	if err != nil {
		t.Fatal(err)
	}
	proj, err := checker.ParseProjection(body)
	if err != nil {
		t.Fatal(err)
	}
	if proj.ProjectionSchema != "1.0" {
		t.Fatalf("projection schema = %q", proj.ProjectionSchema)
	}
	found := false
	for _, rel := range proj.Model.Relationships {
		if rel.Cardinality == "n:m" {
			found = true
		}
	}
	if !found {
		t.Fatal("projection has no normalized many-to-many relationship")
	}
	_, _, code = runBin(t, "check", design, "--gate", "gk")
	if code == 0 {
		t.Fatal("Gk accepted evidence bound to the old model")
	}
	// Rebind fixture evidence to exercise Gk, without claiming an engine run.
	evPath := filepath.Join(design, "checkers", "pii-flow", "evidence.json")
	body, err = os.ReadFile(evPath)
	if err != nil {
		t.Fatal(err)
	}
	var evidence checker.Evidence
	if err := json.Unmarshal(body, &evidence); err != nil {
		t.Fatal(err)
	}
	evidence.InputHash, err = proj.InputHash()
	if err != nil {
		t.Fatal(err)
	}
	writeJSON(t, evPath, evidence)
	out, stderr, code = runBin(t, "check", design, "--gate", "gk")
	if code != 0 {
		t.Fatalf("check --gate gk exited %d: %s%s", code, out, stderr)
	}
}
