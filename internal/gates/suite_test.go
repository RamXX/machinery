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
