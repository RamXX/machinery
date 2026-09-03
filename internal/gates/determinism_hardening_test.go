package gates

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

func TestContractSchemaRejectsSilentObligationDrops(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\n"+
		"contract_version: 2\nboundaries:\n  - id: app\n    code: [\"app/**\"]\nexternls: []\n"+
		"dependency_rules:\n  allow: app -> app\n```\n")
	g := CheckC4(design)
	for _, want := range []string{"unsupported key 'externls'", "dependency_rules.allow must be a list"} {
		if !hasErr(g, want) {
			t.Errorf("missing %q in %v", want, g.Errs)
		}
	}
}

func TestContractSchemaRejectsNonStringScalarsAndAllListShapes(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\n"+
		"contract_version: 2\n_comment: [not, text]\nboundaries:\n"+
		"  - id: 7\n    kind: [component]\n    element: {name: app}\n    code: [\"app/**\"]\n    provides: service\n    consumes: [ok, 9]\n    _comment: false\n"+
		"externals:\n  - id: true\n    element: 42\n    imports: [\"example.com/x\"]\n"+
		"dependency_rules:\n  allow: []\n  assert:\n    - no_path: [app, external]\n      _comment: 1\n```")
	g := CheckC4(design)
	got := strings.Join(g.Errs, "\n")
	for _, want := range []string{
		"root._comment must be a non-empty string",
		"boundaries[0].id must be a non-empty string",
		"boundaries[0].kind must be a non-empty string",
		"boundaries[0].element must be a non-empty string",
		"boundaries[0].provides must be a list of strings",
		"boundaries[0].consumes[1] must be a non-empty string",
		"boundaries[0]._comment must be a non-empty string",
		"externals[0].id must be a non-empty string",
		"externals[0].element must be a non-empty string",
		"dependency_rules.assert[0].no_path must be a non-empty string",
		"dependency_rules.assert[0]._comment must be a non-empty string",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing scalar/list type error %q in %v", want, g.Errs)
		}
	}
}

func copyGateTree(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		from, to := filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyGateTree(t, from, to)
			continue
		}
		raw, err := os.ReadFile(from)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(to, raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func parentPackReadFixture(t *testing.T) string {
	t.Helper()
	design := filepath.Join(t.TempDir(), "design")
	copyGateTree(t, filepath.Join("..", "..", "examples", "checkout-split", "parent", "design"), design)
	return design
}

func TestParentPackReadRejectsSymlinkAndNonDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture is POSIX-specific")
	}
	t.Run("symlink", func(t *testing.T) {
		design := parentPackReadFixture(t)
		target := filepath.Join(design, "packs", "orders.pack")
		external := filepath.Join(t.TempDir(), "orders.pack")
		copyGateTree(t, target, external)
		if err := os.RemoveAll(target); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(external, target); err != nil {
			t.Fatal(err)
		}
		g := CheckPack(design)
		if !hasErr(g, "orders.pack must be a real directory inside packs/") {
			t.Fatalf("symlinked pack directory was followed: %v", g.Errs)
		}
	})
	t.Run("non-directory", func(t *testing.T) {
		design := parentPackReadFixture(t)
		target := filepath.Join(design, "packs", "payments.pack")
		if err := os.RemoveAll(target); err != nil {
			t.Fatal(err)
		}
		mustWrite(t, target, "not a directory\n")
		g := CheckPack(design)
		if !hasErr(g, "payments.pack must be a real directory inside packs/") {
			t.Fatalf("non-directory pack was accepted: %v", g.Errs)
		}
	})
}

func TestParentPackReadPropagatesDirectoryEnumerationFailure(t *testing.T) {
	design := parentPackReadFixture(t)
	old := readPackDir
	readPackDir = func(path string) ([]os.DirEntry, error) {
		if filepath.Base(path) == "orders.pack" {
			return nil, errors.New("injected ReadDir failure")
		}
		return os.ReadDir(path)
	}
	t.Cleanup(func() { readPackDir = old })
	g := CheckPack(design)
	if !hasErr(g, "pack orders: cannot enumerate committed directory: injected ReadDir failure") {
		t.Fatalf("ReadDir failure was silently ignored: %v", g.Errs)
	}
}

func TestCheckMachinesRejectsOrphanGeneratedArtifacts(t *testing.T) {
	design := t.TempDir()
	writeSuiteFile(t, filepath.Join(design, "machines", "Ghost.oracle.md"), covOracleMD)
	writeSuiteFile(t, filepath.Join(design, "machines", "Other.matrix.md"), "# stale matrix\n")
	g := CheckMachines(design)
	got := strings.Join(g.Errs, "\n")
	for _, want := range []string{
		"Ghost.oracle.md: orphan oracle has no corresponding Ghost.machine.json",
		"Other.matrix.md: orphan matrix has no corresponding Other.machine.json",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing exact-basename reconciliation error %q in %v", want, g.Errs)
		}
	}
}

func TestCheckMachinesRejectsOutsideSymlinksBeforeReading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink fixture is POSIX-specific")
	}
	design, outside := t.TempDir(), t.TempDir()
	machine := `{"id":"fixture","initial":"A","states":{"A":{"on":{"go":"B"}},"B":{"final":true}}}`
	mustWrite(t, filepath.Join(design, "machines", "Good.machine.json"), machine)
	mustWrite(t, filepath.Join(outside, "Outside.machine.json"), machine)
	mustWrite(t, filepath.Join(outside, "Outside.oracle.md"), covOracleMD)
	mustWrite(t, filepath.Join(outside, "Outside.matrix.md"), "# plausible outside matrix\n")
	links := map[string]string{
		"Ghost.machine.json": filepath.Join(outside, "Outside.machine.json"),
		"Good.oracle.md":     filepath.Join(outside, "Outside.oracle.md"),
		"Good.matrix.md":     filepath.Join(outside, "Outside.matrix.md"),
		"Ghost.oracle.md":    filepath.Join(outside, "Outside.oracle.md"),
		"Ghost.matrix.md":    filepath.Join(outside, "Outside.matrix.md"),
	}
	for name, target := range links {
		if err := os.Symlink(target, filepath.Join(design, "machines", name)); err != nil {
			t.Fatal(err)
		}
	}
	g := CheckMachines(design)
	got := strings.Join(g.Errs, "\n")
	for _, want := range []string{
		"Ghost.machine.json: matching machine source must be a regular file",
		"Good.oracle.md: matching committed oracle must be a regular file",
		"Good.matrix.md: matching committed matrix must be a regular file",
		"Ghost.oracle.md: matching committed oracle must be a regular file",
		"Ghost.matrix.md: matching committed matrix must be a regular file",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing symlink confinement error %q in %v", want, g.Errs)
		}
	}
	if g.Counts["machines"] != 1 || g.Counts["oracles fresh"] != 0 || g.Counts["matrix rows reconciled"] != 0 {
		t.Fatalf("outside sentinel contributed verified inventory: %+v", g.Counts)
	}
}

func TestCheckMachinesRejectsMatchingDirectories(t *testing.T) {
	design := t.TempDir()
	for _, name := range []string{"Dir.machine.json", "Dir.oracle.md", "Dir.matrix.md"} {
		if err := os.MkdirAll(filepath.Join(design, "machines", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	g := CheckMachines(design)
	got := strings.Join(g.Errs, "\n")
	for _, want := range []string{
		"Dir.machine.json: matching machine source must be a regular file",
		"Dir.oracle.md: matching committed oracle must be a regular file",
		"Dir.matrix.md: matching committed matrix must be a regular file",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing non-regular confinement error %q in %v", want, g.Errs)
		}
	}
}

func TestPackFrozenInvariantUsesCanonicalStatementWithLegacyFallback(t *testing.T) {
	load := func(t *testing.T, yaml string) *ir.Object {
		t.Helper()
		v, err := ir.LoadYAML([]byte(yaml))
		if err != nil {
			t.Fatal(err)
		}
		return v.AsObject()
	}
	canonical := load(t, "statement: canonical truth\ndefinition: contradictory legacy text\n")
	if got := invariantStatement(canonical); got != "canonical truth" {
		t.Fatalf("canonical statement did not win: %q", got)
	}
	legacy := load(t, "definition: legacy truth\n")
	if got := invariantStatement(legacy); got != "legacy truth" {
		t.Fatalf("legacy definition compatibility lost: %q", got)
	}
	if statementKept(invariantStatement(canonical), "contradictory legacy text") {
		t.Fatal("G5 compared the legacy definition instead of canonical statement")
	}
	if !statementKept(invariantStatement(canonical), "canonical truth Structural: enforced by storage") {
		t.Fatal("G5 rejected an allowed appended enforcement detail")
	}
}

func TestContractRejectsDuplicateParticipantsAndBindings(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\n"+
		"contract_version: 2\nboundaries:\n  - {id: app, element: app, code: [\"a/**\"]}\n"+
		"externals:\n  - {id: app, element: app, imports: [\"example.com/x\"]}\n  - {id: ext, element: app, imports: [\"example.com/y\"]}\n```\n")
	g := CheckC4(design)
	if !hasErr(g, "declared as both a boundary and an external") || !hasErr(g, "bound by multiple contract participants") {
		t.Fatalf("participant closure was not enforced: %v", g.Errs)
	}
}

func TestLanguageScannersCoverPreviouslyInvisibleImports(t *testing.T) {
	if got := rustImports("pub(crate) use crate::{repo::save, auth::check};\n"); !reflect.DeepEqual(got, []string{"src/repo/save", "src/auth/check"}) {
		t.Fatalf("grouped visible Rust use = %v", got)
	}
	if got := pyImports("importlib.import_module(\"pkg.secret\")\n__import__('other.mod')\n", "pkg/x.py"); !reflect.DeepEqual(got, []string{"pkg/secret", "other/mod"}) {
		t.Fatalf("literal Python dynamic imports = %v", got)
	}
	if got := tsImports("const r = /\"/; import x from \"forbidden\";\n", "src/x.ts"); !reflect.DeepEqual(got, []string{"forbidden"}) {
		t.Fatalf("TS regex literal hid import: %v", got)
	}
}

func TestRustCfgTestLiteralBraceCannotHideProductionImport(t *testing.T) {
	prod, tests := rustSplitTests("#[cfg(test)]\nmod tests { const X: &str = \"{\"; }\nuse crate::forbidden::Thing;\n")
	if len(tests) != 1 || !strings.Contains(prod, "use crate::forbidden::Thing") {
		t.Fatalf("bad split: prod=%q tests=%q", prod, tests)
	}
}

func TestBoundaryMatchRejectsEqualSpecificity(t *testing.T) {
	if got, amb := boundaryMatch("internal/x.go", [][2]string{{"internal/**", "a"}, {"internal/**", "b"}}); got != "" || !reflect.DeepEqual(amb, []string{"a", "b"}) {
		t.Fatalf("got owner=%q ambiguous=%v", got, amb)
	}
}

func TestManifestDeclaredExternalCannotDisappear(t *testing.T) {
	design, impl := t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n  - {id: app, code: [\"src/**\"]}\ndependency_rules: {allow: []}\n```\n")
	mustWrite(t, filepath.Join(impl, "package.json"), `{"name":"app","dependencies":{"left-pad":"1.0.0"}}`)
	mustWrite(t, filepath.Join(impl, "src", "index.ts"), `import pad from "left-pad";`)
	if g := CheckImports(design, impl); !hasErr(g, "manifest-declared dependency left-pad") {
		t.Fatalf("undeclared manifest dependency disappeared: %v", g.Errs)
	}
}

func TestOracleIDsInCommentsDoNotCover(t *testing.T) {
	if got := stripTestComments("package x\n// THIN-aaa111\nconst id = \"THIN-bbb222\"\n", ".go"); idTokenIn("THIN-aaa111", got) || !idTokenIn("THIN-bbb222", got) {
		t.Fatalf("comment stripping result %q", got)
	}
	if got := executableTestText("package x\nconst id = \"THIN-aaa111\"\nfunc TestOther(t *testing.T) {}\n", "x_test.go"); idTokenIn("THIN-aaa111", got) {
		t.Fatalf("dead package-level id counted as executable test coverage: %q", got)
	}
}

func TestTraceabilityRejectsDuplicateLifecycleClaimants(t *testing.T) {
	design := t.TempDir()
	model := "kind: DomainModel\nversion: v1\nenums:\n  Status:\n    values: [{name: Open}, {name: Done}]\nentities:\n  Order:\n    attributes: [{name: status, type: Status}]\n    actions: [{name: finish}]\n"
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	machine := `{"id":"%s","_lifecycle_of":"Order","initial":"Open","states":{"Open":{"on":{"finish":"Done"}},"Done":{"type":"final"}}}`
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), strings.Replace(machine, "%s", "order", 1))
	mustWrite(t, filepath.Join(design, "machines", "OrderCopy.machine.json"), strings.Replace(machine, "%s", "order-copy", 1))
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"), "machine: order\npattern: control-flow-only\nreason: lifecycle data has no refinement beyond the checked control-flow enum\n")
	mustWrite(t, filepath.Join(design, "formal", "OrderCopy.semantics.yaml"), "machine: order-copy\npattern: control-flow-only\nreason: duplicate fixture\n")
	if g := CheckTraceability(design); !hasErr(g, "has 2 lifecycle machines") {
		t.Fatalf("duplicate lifecycle ownership passed: %v", g.Errs)
	}
}

func TestControlFlowOnlySemanticsRequiresReasonAndBinding(t *testing.T) {
	design := t.TempDir()
	model := "kind: DomainModel\nversion: v1\nenums:\n  Status:\n    values: [{name: Open}, {name: Done}]\nentities:\n  Order:\n    attributes: [{name: status, type: Status}]\n    actions: [{name: finish}]\n"
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), `{"id":"order","initial":"Open","states":{"Open":{"on":{"finish":"Done"}},"Done":{"type":"final"}}}`)
	mustWrite(t, filepath.Join(design, "formal", "Order.semantics.yaml"), "machine: wrong\npattern: control-flow-only\nreason: \"\"\n")
	g := CheckTraceability(design)
	if !hasErr(g, "machine must equal") || !hasErr(g, "requires a non-empty specific reason") {
		t.Fatalf("control-flow-only schema passed: %v", g.Errs)
	}
}

func TestMissingLifecycleSemanticsDiagnosticsAreByteStable(t *testing.T) {
	design := t.TempDir()
	model := "kind: DomainModel\nversion: v1\nenums:\n  Status:\n    values: [{name: Open}, {name: Done}]\nentities:\n  Alpha:\n    attributes: [{name: status, type: Status}]\n    actions: [{name: finish}]\n  Zeta:\n    attributes: [{name: status, type: Status}]\n    actions: [{name: finish}]\n"
	mustWrite(t, filepath.Join(design, "domain.modelith.yaml"), model)
	machine := `{"id":"%s","initial":"Open","states":{"Open":{"on":{"finish":"Done"}},"Done":{"type":"final"}}}`
	mustWrite(t, filepath.Join(design, "machines", "Alpha.machine.json"), strings.Replace(machine, "%s", "alpha", 1))
	mustWrite(t, filepath.Join(design, "machines", "Zeta.machine.json"), strings.Replace(machine, "%s", "zeta", 1))
	want := "Alpha.machine.json: no formal/Alpha.semantics.yaml"
	var first string
	for i := 0; i < 50; i++ {
		got := strings.Join(CheckTraceability(design).Errs, "\n")
		if i == 0 {
			first = got
		} else if got != first {
			t.Fatalf("Gx output changed across identical runs:\nfirst:\n%s\nrun %d:\n%s", first, i, got)
		}
		if alpha, zeta := strings.Index(got, want), strings.Index(got, "Zeta.machine.json: no formal/Zeta.semantics.yaml"); alpha < 0 || zeta < 0 || alpha > zeta {
			t.Fatalf("missing lifecycle semantics are not byte-sorted: %s", got)
		}
	}
}

func TestAttestationRejectsSymlinkReferent(t *testing.T) {
	design := t.TempDir()
	outside := filepath.Join(t.TempDir(), "ARCHITECTURE.md")
	mustWrite(t, outside, "outside\n")
	if err := os.Symlink(outside, filepath.Join(design, "ARCHITECTURE.md")); err != nil {
		t.Fatal(err)
	}
	hash, err := ContentHash(outside)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, AttestationsFileName), "attestation_version: 1\nattestations:\n  - claim: g2.nfr-content\n    attestor: reviewer\n    date: 2026-09-02\n    covers:\n      - {path: ARCHITECTURE.md, hash: "+hash+"}\n")
	if g := CheckAttestations(design); !hasErr(g, "reached through a symlink") {
		t.Fatalf("symlink referent passed: %v", g.Errs)
	}
}

func TestAttestationClaimMustCoverItsSubject(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "architecture\n")
	mustWrite(t, filepath.Join(design, "DECISIONS.md"), "decision\n")
	hash, err := ContentHash(filepath.Join(design, "DECISIONS.md"))
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, AttestationsFileName), "attestation_version: 1\nattestations:\n  - claim: g2.nfr-content\n    attestor: reviewer\n    date: 2026-09-02\n    covers:\n      - {path: DECISIONS.md, hash: "+hash+"}\n")
	if g := CheckAttestations(design); !hasErr(g, "does not cover required subject 'ARCHITECTURE.md'") {
		t.Fatalf("unrelated current file discharged claim: %v", g.Errs)
	}
}

func TestEventSourceRequiresPayloadOrValidEmbed(t *testing.T) {
	for _, prefix := range []string{"Source:\n\n", "<!-- machinery:embed typo -->\n\n"} {
		g := NewGate("x")
		checkEventTableSources(g, prefix+"| producer | consumer | delivery |\n|---|---|---|\n| a | b | once |\n")
		if len(g.Errs) == 0 {
			t.Fatalf("invalid source evidence passed: %q", prefix)
		}
	}
}
