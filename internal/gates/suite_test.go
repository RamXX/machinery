package gates

import (
	"os"
	"path/filepath"
	"strings"
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
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Run["g3"] || sel.Run["gx"] || sel.Run["g4"] || sel.Run["gt"] {
		t.Errorf("machine-less decomposed parent still selects g3/gx/g4/gt: %v", sel.Run)
	}
	if sel.Run["gb"] {
		t.Errorf("gb must not survive the narrowing without a BUILD.md: %v", sel.Run)
	}
	if !sel.Run["g2"] || !sel.Run["g5"] {
		t.Errorf("machine-less decomposed parent must run g2,g5: %v", sel.Run)
	}
	if sel.Note == "" {
		t.Error("narrowing must announce itself with an informational note")
	}
}

// Gb is artifact-activated: the machine-less-parent narrowing keeps it when
// the parent carries a BUILD.md (the manifest is still its artifact), and
// the note lists it between g2 and g5, matching the canonical run order.
func TestSelectNarrowingKeepsGbWhenBuildDocExists(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "BUILD.md"), "# B\n\nMode: manifest\n")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gb"] {
		t.Errorf("narrowing dropped gb although BUILD.md exists: %v", sel.Run)
	}
	if sel.Run["gt"] {
		t.Errorf("gt is never part of the narrowed parent list: %v", sel.Run)
	}
	want := "note: decomposed parent with no machines/; running g2,gb,g5 (G3/Gx run on the child designs; gt skipped: no machines)"
	if sel.Note != want {
		t.Errorf("note = %q\nwant %q", sel.Note, want)
	}
}

// G4 is NOT machine-dependent: contract + code suffice, so an explicit
// --impl keeps it on a narrowed machine-less parent. Gt IS machine-dependent
// and stays skipped, but the note must name it explicitly (GATE-1).
func TestSelectNarrowedParentKeepsG4WithImpl(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	sel, err := Select(design, "", "impl")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["g4"] {
		t.Errorf("--impl on a narrowed parent must keep g4: %v", sel.Run)
	}
	if sel.Run["gt"] || sel.Run["g3"] || sel.Run["gx"] {
		t.Errorf("gt/g3/gx stay machine-dependent: %v", sel.Run)
	}
	want := "note: decomposed parent with no machines/; running g2,g4,g5 (G3/Gx run on the child designs; gt skipped: no machines)"
	if sel.Note != want {
		t.Errorf("note = %q\nwant %q", sel.Note, want)
	}
}

// An empty gate token ("g2,") once produced `unknown gate(s): ` with an
// empty name; it must be its own clear error.
func TestSelectRejectsEmptyGateToken(t *testing.T) {
	design := t.TempDir()
	for _, list := range []string{"g2,", ",g2", "g2,,g3"} {
		if _, err := Select(design, list, ""); err == nil || !strings.Contains(err.Error(), "empty gate name") {
			t.Errorf("Select(%q) error = %v, want an empty-gate-name error", list, err)
		}
	}
}

// Machine detection must survive glob metacharacters in the design path: a
// path like "des[1]" once made filepath.Glob report zero machines and
// silently narrowed g3/gx away (GATE-2).
func TestSelectMetacharDesignPathKeepsFullSuite(t *testing.T) {
	design := filepath.Join(t.TempDir(), "des[1]")
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "machines", "Order.machine.json"), "{}\n")
	if !HasMachines(design) {
		t.Fatal("HasMachines must see machines behind a metachar path")
	}
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["g3"] || !sel.Run["gx"] {
		t.Errorf("a metachar design path must not defeat machine detection: %v", sel.Run)
	}
	if sel.Note != "" {
		t.Errorf("no narrowing note expected with machines present: %q", sel.Note)
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
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn", "g2", "g5"} {
		if !sel.Run[g] {
			t.Errorf("narrowing dropped artifact-activated gate %s: %v", g, sel.Run)
		}
	}
	// g3/gx are machine-dependent and narrow away; g4 is absent here only
	// because no --impl was supplied (with one it runs, see
	// TestSelectNarrowedParentKeepsG4WithImpl)
	for _, g := range []string{"g3", "gx", "g4", "gt"} {
		if sel.Run[g] {
			t.Errorf("without --impl the narrowed parent must not select %s: %v", g, sel.Run)
		}
	}
	want := "note: decomposed parent with no machines/; running gm,gs,gp,gi,gn,g2,g5 (G3/Gx run on the child designs; gt skipped: no machines)"
	if sel.Note != want {
		t.Errorf("note = %q\nwant %q (the note must list what actually runs)", sel.Note, want)
	}
}

// The default suite carries the full vocabulary, and the validator accepts
// exactly that vocabulary: gb and gt joined in the same release.
func TestSelectDefaultListAndVocabulary(t *testing.T) {
	design := t.TempDir()
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []string{"gm", "gs", "gp", "gi", "gn", "g2", "g3", "gx", "gb", "g4", "gt", "g5"} {
		if !sel.Run[g] {
			t.Errorf("default list omits %s: %v", g, sel.Run)
		}
	}
	if _, err := Select(design, "gb,gt", ""); err != nil {
		t.Errorf("gb,gt must be a valid explicit list: %v", err)
	}
	if _, err := Select(design, "gz", ""); err == nil {
		t.Error("an unknown gate must be rejected")
	}
}

// Gb activates on the BUILD.md artifact (like gp/gi/gn on theirs); Gt is
// impl-facing and runs only when an impl dir is given (like g4).
func TestRunSelectedActivatesGbAndGt(t *testing.T) {
	design := t.TempDir()
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	titles := func(impl string) string {
		var out []string
		for _, g := range RunSelected(design, impl, sel) {
			out = append(out, g.Title)
		}
		return strings.Join(out, "\n")
	}
	if got := titles(""); strings.Contains(got, "Gb-plan") || strings.Contains(got, "Gt-tests") {
		t.Errorf("no BUILD.md and no impl: neither Gb nor Gt may run, got\n%s", got)
	}
	writeSuiteFile(t, filepath.Join(design, "BUILD.md"), "# B\n\nMode: full\n")
	if got := titles(""); !strings.Contains(got, "Gb-plan") {
		t.Errorf("BUILD.md exists: Gb must auto-activate, got\n%s", got)
	}
	impl := t.TempDir()
	got := titles(impl)
	if !strings.Contains(got, "Gt-tests") {
		t.Errorf("with an impl dir Gt must run, got\n%s", got)
	}
	// canonical order: Gb between Gx and G4, Gt between G4 and G5
	if gb, g4 := strings.Index(got, "Gb-plan"), strings.Index(got, "G4-import"); g4 >= 0 && gb > g4 {
		t.Errorf("Gb must run before G4:\n%s", got)
	}
	if g4, gt := strings.Index(got, "G4-import"), strings.Index(got, "Gt-tests"); g4 >= 0 && gt < g4 {
		t.Errorf("Gt must run after G4:\n%s", got)
	}
}

func TestSelectKeepsFullDefaultOnceMachinesExist(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeSuiteFile(t, filepath.Join(design, "machines", "Order.machine.json"), "{}\n")
	sel, err := Select(design, "", "")
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
