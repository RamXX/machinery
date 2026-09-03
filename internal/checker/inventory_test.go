package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInventoryRejectsOrphanCheckerOutputsAndDirectories(t *testing.T) {
	design := t.TempDir()
	checkers := filepath.Join(design, "checkers")
	if err := os.MkdirAll(filepath.Join(checkers, "owned"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(checkers, "owned.checker.yaml")
	body := "checker: {id: owned, runtime_closure: " + checkerTestRuntime + "}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/owned/projection.json, evidence_in: checkers/owned/evidence.json}\n"
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(checkers, "owned", "projection.json"),
		filepath.Join(checkers, "owned", "evidence.json"),
	} {
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if problems := InventoryProblems(design, []*Manifest{manifest}); len(problems) != 0 {
		t.Fatalf("owned inventory rejected: %v", problems)
	}
	orphan := filepath.Join(checkers, "old")
	if err := os.Mkdir(orphan, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "projection.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	problems := InventoryProblems(design, []*Manifest{manifest})
	if len(problems) == 0 || !strings.Contains(strings.Join(problems, "\n"), "orphan checker directory") {
		t.Fatalf("orphan inventory was accepted: %v", problems)
	}
}

func TestValidateManifestSetRejectsPortableOutputAliases(t *testing.T) {
	manifest := func(id, projection, evidence string) *Manifest {
		m := &Manifest{}
		m.Checker.ID = id
		m.Evidence.ProjectionOut = projection
		m.Evidence.EvidenceIn = evidence
		return m
	}
	for name, manifests := range map[string][]*Manifest{
		"checker id": {
			manifest("Privacy", "a/projection.json", "a/evidence.json"),
			manifest("privacy", "b/projection.json", "b/evidence.json"),
		},
		"cross-kind output": {
			manifest("a", "Shared/Output.json", "a/evidence.json"),
			manifest("b", "b/projection.json", "shared/output.JSON"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateManifestSet(manifests); err == nil {
				t.Fatal("portable alias was accepted")
			}
		})
	}
}
