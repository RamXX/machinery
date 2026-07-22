package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture lays down a minimal design + Go impl: boundaries alpha and
// beta, one cross-boundary import alpha -> beta, and (optionally) a second
// offender file, an unmapped legacy file, and an orphan module-internal
// import. rules is the YAML under dependency_rules (already indented two
// spaces per level, or empty).
type fixtureOpts struct {
	rules        string
	secondFile   bool // alpha/b.go also imports beta
	legacyFile   bool // legacy/old.go outside every boundary
	orphanImport bool // alpha/a.go imports an unmapped module-internal package
}

func writeFixture(t *testing.T, o fixtureOpts) (design, impl string) {
	t.Helper()
	root := t.TempDir()
	design = filepath.Join(root, "design")
	impl = filepath.Join(root, "impl")
	rules := o.rules
	if rules == "" {
		rules = "  allow: []\n  deny: []"
	}
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\n" +
		"dependency_rules:\n" + rules + "\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "go.mod"), "module example.com/m\n")
	aImports := "\t\"example.com/m/beta\"\n"
	if o.orphanImport {
		aImports += "\t\"example.com/m/internal/oldpkg\"\n"
	}
	mustWrite(t, filepath.Join(impl, "alpha", "a.go"), "package alpha\n\nimport (\n"+aImports+")\n")
	mustWrite(t, filepath.Join(impl, "beta", "b.go"), "package beta\n")
	if o.secondFile {
		mustWrite(t, filepath.Join(impl, "alpha", "b.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	}
	if o.legacyFile {
		mustWrite(t, filepath.Join(impl, "legacy", "old.go"), "package old\n")
	}
	return design, impl
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasErr(g *Gate, needle string) bool {
	for _, e := range g.Errs {
		if strings.Contains(e, needle) {
			return true
		}
	}
	return false
}

func hasNote(g *Gate, needle string) bool {
	for _, n := range g.Notes {
		if strings.Contains(n, needle) {
			return true
		}
	}
	return false
}

// --- G2: baseline rule validation ---

func TestG2AllowPlusBaselineConflicts(t *testing.T) {
	design, _ := writeFixture(t, fixtureOpts{rules: "  allow: [\"alpha -> beta\"]\n  baseline: [\"alpha -> beta\"]"})
	g := CheckC4(design)
	if !hasErr(g, "both allowed and baselined") {
		t.Fatalf("allow+baseline must be a G2 contradiction, got %v", g.Errs)
	}
}

func TestG2DenyPlusBaselineIsLegitimate(t *testing.T) {
	design, _ := writeFixture(t, fixtureOpts{rules: "  deny: [\"alpha -> beta\"]\n  baseline: [\"alpha -> beta\"]"})
	g := CheckC4(design)
	if hasErr(g, "baselined") {
		t.Fatalf("deny+baseline records intent plus tolerated debt and must pass, got %v", g.Errs)
	}
	if g.Counts["baseline rules"] != 1 {
		t.Fatalf("baseline rules not counted: %v", g.Counts)
	}
}

func TestG2BaselineReferencesDeclaredBoundaries(t *testing.T) {
	design, _ := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> ghost\"]"})
	g := CheckC4(design)
	if !hasErr(g, "undeclared boundary") {
		t.Fatalf("a baseline rule naming an unknown boundary must fail G2, got %v", g.Errs)
	}
}

// A wildcard baseline rule would amnesty the whole edge space: baseline is an
// enumerated-edges ratchet, so G2 hard-errors on it (GATE-7).
func TestG2WildcardBaselineRuleIsAnError(t *testing.T) {
	design, _ := writeFixture(t, fixtureOpts{rules: "  baseline: [\"* -> *\"]"})
	g := CheckC4(design)
	if !hasErr(g, "wildcard") {
		t.Fatalf("a wildcard baseline rule must be a hard G2 ERROR, got %v", g.Errs)
	}
}

// G4 never honors a wildcard baseline rule: in a --gate g4 run (where G2's
// finding is absent) the edges it would have amnestied stay undeclared.
func TestG4WildcardBaselineDoesNotAmnesty(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"* -> *\"]"})
	g := CheckImports(design, impl)
	if !hasErr(g, "undeclared cross-boundary edge alpha -> beta") {
		t.Fatalf("the wildcard-baselined edge must stay undeclared, got %v", g.Errs)
	}
	if g.Counts["baselined edges"] != 0 {
		t.Errorf("no edge may count as baselined under a wildcard rule: %+v", g.Counts)
	}
}

// The ratchet snapshot date is read back, not just recorded: G4 notes the
// snapshot age so tolerated debt stays visible (GATE-8; non-blocking).
func TestG4RatchetAgeNote(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]"})
	if err := WriteRatchet(design, &Ratchet{Date: "2026-01-15", Edges: map[string][]string{"alpha -> beta": {"alpha/a.go"}}}); err != nil {
		t.Fatal(err)
	}
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("G4 must stay green at snapshot: %v", g.Errs)
	}
	if !hasNote(g, "ratchet snapshot 2026-01-15") || !hasNote(g, "old") {
		t.Fatalf("want a ratchet-age note naming the snapshot date, got %v", g.Notes)
	}
}

// --- G4: Rust production scanning (NG-1) ---

// A production .rs file with an inline #[cfg(test)] module is NOT a test
// file: G4 scans its production portion, so cross-boundary imports and the
// unmapped-file error stay visible.
func TestG4RustInlineTestModuleStillScanned(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"src/app/**\"]\n  - id: core\n    code: [\"src/core/**\"]\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "src", "app", "main.rs"),
		"use crate::core::engine;\n\npub fn run() {}\n\n#[cfg(test)]\nmod tests {\n    #[test]\n    fn t() {}\n}\n")
	mustWrite(t, filepath.Join(impl, "src", "core", "lib.rs"), "pub mod engine {}\n")
	g := CheckImports(design, impl)
	if !hasErr(g, "undeclared cross-boundary edge app -> core") {
		t.Fatalf("the production import in a cfg(test)-bearing file must be judged: %v", g.Errs)
	}
	if g.Counts["rust files checked"] != 2 {
		t.Errorf("rust files checked = %d, want 2: %+v", g.Counts["rust files checked"], g.Counts)
	}
	// an import that lives ONLY inside the cfg(test) module is not a
	// production edge
	mustWrite(t, filepath.Join(impl, "src", "app", "main.rs"),
		"pub fn run() {}\n\n#[cfg(test)]\nmod tests {\n    use crate::core::engine;\n    #[test]\n    fn t() {}\n}\n")
	if g2 := CheckImports(design, impl); hasErr(g2, "undeclared cross-boundary edge app -> core") {
		t.Fatalf("a test-only import must not create a production edge: %v", g2.Errs)
	}
}

// f1-g4rust: a .rs file outside every boundary used to vanish wholesale as a
// "test file" because it contained #[cfg(test)], suppressing the
// unmapped-file error.
func TestG4RustUnmappedFileWithInlineTestsIsAnError(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	mustWrite(t, filepath.Join(impl, "rogue.rs"),
		"use std::collections::HashMap;\n\npub fn production_logic() {}\n\n#[cfg(test)]\nmod tests {\n    #[test]\n    fn t() {}\n}\n")
	g := CheckImports(design, impl)
	if !hasErr(g, "source file rogue.rs maps to no contract boundary") {
		t.Fatalf("the unmapped production .rs file must be an ERROR: %v", g.Errs)
	}
}

// --- G4: baseline toleration and the ratchet ---

func TestG4BaselineWithoutRatchetFails(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]"})
	g := CheckImports(design, impl)
	if !hasErr(g, "no "+RatchetFile) {
		t.Fatalf("baseline rules without a snapshot must fail on absence, got %v", g.Errs)
	}
}

func TestG4RatchetGreenAtSnapshot(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]"})
	if err := WriteRatchet(design, &Ratchet{Date: "2026-07", Edges: map[string][]string{"alpha -> beta": {"alpha/a.go"}}}); err != nil {
		t.Fatal(err)
	}
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("edge at its snapshot must pass, got %v", g.Errs)
	}
	if g.Counts["baselined edges"] != 1 || g.Counts["ratcheted edges"] != 1 {
		t.Fatalf("baselined/ratcheted counts wrong: %v", g.Counts)
	}
}

func TestG4RatchetGrowthFails(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]", secondFile: true})
	if err := WriteRatchet(design, &Ratchet{Date: "2026-07", Edges: map[string][]string{"alpha -> beta": {"alpha/a.go"}}}); err != nil {
		t.Fatal(err)
	}
	g := CheckImports(design, impl)
	if !hasErr(g, "grew by 1 new offender file") || !hasErr(g, "alpha/b.go") {
		t.Fatalf("a new offender file on a baselined edge must fail and be named, got %v", g.Errs)
	}
}

func TestG4RatchetShrinkAndStaleEdgesNote(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]"})
	if err := WriteRatchet(design, &Ratchet{Date: "2026-07", Edges: map[string][]string{
		"alpha -> beta":  {"alpha/a.go", "alpha/gone.go"},
		"gamma -> delta": {"gamma/x.go"},
	}}); err != nil {
		t.Fatal(err)
	}
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("shrinkage is good news, not an error: %v", g.Errs)
	}
	if !hasNote(g, "can tighten") || !hasNote(g, "gamma -> delta") {
		t.Fatalf("shrunk and resolved edges must nudge a re-snapshot, got %v", g.Notes)
	}
}

func TestG4DenyPlusBaselineTolerates(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  deny: [\"alpha -> beta\"]\n  baseline: [\"alpha -> beta\"]"})
	if err := WriteRatchet(design, &Ratchet{Date: "2026-07", Edges: map[string][]string{"alpha -> beta": {"alpha/a.go"}}}); err != nil {
		t.Fatal(err)
	}
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("baseline must shadow deny while the debt burns down, got %v", g.Errs)
	}
}

func TestG4UnbaselinedEdgeStillUndeclared(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{})
	g := CheckImports(design, impl)
	if !hasErr(g, "undeclared cross-boundary edge alpha -> beta") {
		t.Fatalf("no baseline, no allow: the edge must stay an error, got %v", g.Errs)
	}
}

// --- the generator ---

func TestBuildBaselineProposesAndSnapshots(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{secondFile: true, legacyFile: true, orphanImport: true})
	rep, err := BuildBaseline(design, impl, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Proposed) != 1 || rep.Proposed[0].Edge != "alpha -> beta" || rep.Proposed[0].More != 1 {
		t.Fatalf("proposed rules wrong: %+v", rep.Proposed)
	}
	files := rep.Ratchet.Edges["alpha -> beta"]
	if len(files) != 2 || files[0] != "alpha/a.go" || files[1] != "alpha/b.go" {
		t.Fatalf("snapshot files wrong: %v", files)
	}
	if len(rep.IgnoreGlobs) != 1 || rep.IgnoreGlobs[0] != "legacy/**" {
		t.Fatalf("ignore suggestions wrong: %v", rep.IgnoreGlobs)
	}
	if len(rep.Orphans) != 1 || rep.Orphans[0].Ref != "example.com/m/internal/oldpkg" {
		t.Fatalf("orphan refs wrong: %+v", rep.Orphans)
	}
}

func TestBuildBaselineDoesNotProposeAllowedEdges(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  allow: [\"alpha -> beta\"]"})
	rep, err := BuildBaseline(design, impl, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Proposed) != 0 || len(rep.Ratchet.Edges) != 0 {
		t.Fatalf("an intended (allowed) edge is not debt: %+v", rep)
	}
}

func TestBaselineClosesTheLoop(t *testing.T) {
	// the flagship Stage-1 flow: scan, paste the proposed rules, commit the
	// snapshot, and the gate goes green; then a new offender file trips it
	design, impl := writeFixture(t, fixtureOpts{legacyFile: true})
	rep, err := BuildBaseline(design, impl, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	rules := "  baseline:\n"
	for _, p := range rep.Proposed {
		rules += "    - \"" + p.Edge + "\"\n"
	}
	rules = strings.TrimRight(rules, "\n")
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\n" +
		"ignore:\n  - \"legacy/**\"\ndependency_rules:\n" + rules + "\n```\n"
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	if err := WriteRatchet(design, rep.Ratchet); err != nil {
		t.Fatal(err)
	}
	if g := CheckImports(design, impl); len(g.Errs) != 0 {
		t.Fatalf("after baseline the gate must be green, got %v", g.Errs)
	}
	mustWrite(t, filepath.Join(impl, "alpha", "new.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	if g := CheckImports(design, impl); !hasErr(g, "grew by 1 new offender file") {
		t.Fatalf("a new offender after baselining must trip the ratchet, got %v", g.Errs)
	}
}

func TestRatchetRoundTrip(t *testing.T) {
	design := t.TempDir()
	in := &Ratchet{Date: "2026-07", Edges: map[string][]string{"a -> b": {"z.go", "a.go"}}}
	if err := WriteRatchet(design, in); err != nil {
		t.Fatal(err)
	}
	out, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	if out.Date != "2026-07" || len(out.Edges["a -> b"]) != 2 || out.Edges["a -> b"][0] != "a.go" {
		t.Fatalf("round trip lost data or order: %+v", out)
	}
	if r, err := LoadRatchet(t.TempDir()); r != nil || err != nil {
		t.Fatalf("missing ratchet is (nil, nil), got %v %v", r, err)
	}
	mustWrite(t, filepath.Join(design, RatchetFile), "{not json")
	if _, err := LoadRatchet(design); err == nil {
		t.Fatal("a corrupt ratchet must fail loudly")
	}
}
