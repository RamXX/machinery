package gates

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSuiteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A decomposed parent with no machines narrows the default suite to g2,g5
// with a note. Machine-less means no *.machine.json: an EMPTY machines/
// directory once defeated the narrowing and failed the parent on G3/Gx,
// contradicting the documented behavior (the H2 dogfood finding).
func TestSelectNarrowsDecomposedParentWithEmptyMachinesDir(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	sel, err := Select(design, "")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Run["g3"] || sel.Run["gx"] || sel.Run["g4"] {
		t.Errorf("machine-less decomposed parent still selects g3/gx/g4: %v", sel.Run)
	}
	if !sel.Run["g2"] || !sel.Run["g5"] {
		t.Errorf("machine-less decomposed parent must run g2,g5: %v", sel.Run)
	}
	if sel.Note == "" {
		t.Error("narrowing must announce itself with an informational note")
	}
}

// v0.3.0 regression (caught on H2): the machine-less narrowing dropped the
// artifact-activated relational gates. Narrowing may only skip the
// machine-dependent gates (g3, gx, g4); gm/gs/gp/gi/gn keep their
// auto-activation whenever their source artifacts exist.
func TestSelectNarrowingKeepsArtifactActivatedGates(t *testing.T) {
	design := t.TempDir() // no machines/ directory at all
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "migration.yaml"), "contract_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "legacy", "surface.yaml"), "surface_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "formal", "policy.relational.yaml"), "subjects: {}\n")
	writeSuiteFile(t, filepath.Join(design, "formal", "integrity.relational.yaml"), "entities: []\n")
	writeSuiteFile(t, filepath.Join(design, "formal", "isolation.relational.yaml"), "tenant: {}\n")
	sel, err := Select(design, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn", "g2", "g5"} {
		if !sel.Run[g] {
			t.Errorf("narrowing dropped artifact-activated gate %s: %v", g, sel.Run)
		}
	}
	for _, g := range []string{"g3", "gx", "g4"} {
		if sel.Run[g] {
			t.Errorf("narrowing must skip machine-dependent gate %s: %v", g, sel.Run)
		}
	}
	want := "note: decomposed parent with no machines/; running gm,gs,gp,gi,gn,g2,g5 (G3/Gx/G4 run on the child designs)"
	if sel.Note != want {
		t.Errorf("note = %q\nwant %q (the note must list what actually runs)", sel.Note, want)
	}
}

func TestSelectKeepsFullDefaultOnceMachinesExist(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "machines", "Order.machine.json"), "{}\n")
	sel, err := Select(design, "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["g3"] || !sel.Run["gx"] {
		t.Errorf("decomposed parent WITH machines must keep g3,gx: %v", sel.Run)
	}
	if sel.Note != "" {
		t.Errorf("no narrowing note expected with machines present: %q", sel.Note)
	}
}
