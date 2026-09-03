package checker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/dirscan"
)

func TestInventoryEnforcesAggregateCeilingBeforeWalkingEntries(t *testing.T) {
	design := t.TempDir()
	checkers := filepath.Join(design, "checkers")
	if err := os.Mkdir(checkers, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(checkers, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	problems := inventoryProblems(design, nil, dirscan.WalkLimits{MaxEntries: 2, MaxDepth: 4})
	if len(problems) != 1 || !strings.Contains(problems[0], "2-entry limit") {
		t.Fatalf("high-entry checker inventory was accepted: %v", problems)
	}
}

func TestInventoryEnforcesDepthCeilingBeforeWalkingEntries(t *testing.T) {
	design := t.TempDir()
	checkers := filepath.Join(design, "checkers")
	if err := os.MkdirAll(filepath.Join(checkers, "owned", "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Path: filepath.Join(checkers, "owned.checker.yaml")}
	manifest.Checker.ID = "owned"
	manifest.Evidence.ProjectionOut = "checkers/owned/projection.json"
	manifest.Evidence.EvidenceIn = "checkers/owned/evidence.json"
	problems := inventoryProblems(design, []*Manifest{manifest}, dirscan.WalkLimits{MaxEntries: 10, MaxDepth: 1})
	if len(problems) != 1 || !strings.Contains(problems[0], "depth limit") {
		t.Fatalf("deep checker inventory was accepted: %v", problems)
	}
}

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

func TestInventoryFailsClosedOnOversizedUnownedJSON(t *testing.T) {
	design := t.TempDir()
	checkers := filepath.Join(design, "checkers")
	owned := filepath.Join(checkers, "owned")
	if err := os.MkdirAll(owned, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Path: filepath.Join(checkers, "owned.checker.yaml")}
	manifest.Checker.ID = "owned"
	manifest.Evidence.ProjectionOut = "checkers/owned/projection.json"
	manifest.Evidence.EvidenceIn = "checkers/owned/evidence.json"
	orphan := filepath.Join(owned, "opaque.json")
	file, err := os.OpenFile(orphan, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(checkerStructuredFileMaxBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	problems := InventoryProblems(design, []*Manifest{manifest})
	if len(problems) != 1 || !strings.Contains(problems[0], "byte limit") {
		t.Fatalf("oversized orphan checker JSON did not fail closed: %v", problems)
	}
}
