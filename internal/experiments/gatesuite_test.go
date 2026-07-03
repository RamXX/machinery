package experiments

// The filesystem-fixture runner for MachineryCheckExperiments: a faithful port
// of legacy test_machinery_check.py + fixtures.py. Every gate must fail loudly
// on absence and on each review mutation, and pass on the coherent synthetic
// design. These are permanent regressions; do not weaken an assertion to make
// a code change pass.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ramirosalas/machinery/internal/gates"
	"github.com/ramirosalas/machinery/internal/ir"
	"github.com/ramirosalas/machinery/internal/oracle"
)

const fixtureMachineJSON = `{"id":"widget","initial":"Draft","context":{"widgetId":null},"states":{
 "Draft":{"on":{"publish":[
   {"target":"persisting","guard":"guardCanPublish","actions":"setPending"},
   {"actions":"recordDenied"}]}},
 "Published":{"type":"final"},
 "persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},
                         "onError":{"target":"Draft","actions":"recordError"}},
               "after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}
`

const fixtureModelith = `kind: modelith
version: 1
title: Widget
enums:
  WidgetStatus:
    values:
      - name: Draft
      - name: Published
entities:
  Widget:
    attributes:
      - name: status
        type: WidgetStatus
    actions:
      - name: publish
    invariants:
      - id: widget-owned
invariants: []
scenarios: []
`

const fixtureWorkspaceDSL = `workspace "Widget" "Test system." {
  model {
    user = person "User" "Uses it."
    sys = softwareSystem "Widget System" "The system." {
      app      = component "App" "Application logic." "Go"
      storelib = component "Store" "Persistence." "Go"
      db       = container "DB" "The database." "SQLite" "Database"
    }
    user -> app "Uses"
    app -> storelib "Persists"
    storelib -> db "SQL"
  }
}
`

const fixtureArchitecture = `# Architecture: Widget

## 1. Shape

One binary.

## 2. Deployment example (not the contract)

` + "```yaml" + `
replicas: 3
` + "```" + `

## 4. Architecture Contract

` + "```yaml" + `
contract_version: 2
boundaries:
  - id: widget.app
    kind: component
    element: app
    code: [ "internal/app/**" ]
  - id: widget.store
    kind: component
    element: storelib
    code: [ "internal/store/**" ]
    exposes: [ "internal/store/store.go" ]
externals:
  - id: external.db
    element: db
    imports: [ "example.com/dbdriver" ]
ignore: [ "internal/scaffold/**" ]
dependency_rules:
  allow:
    - widget.app -> widget.store
    - widget.store -> external.db
  deny:
    - "widget.app -> external.db"
` + "```" + `

## 6. Dependency mitigation posture

| dependency | failure modes | mitigation | residual | bound |
|---|---|---|---|---|
| ` + "`db`" + ` | unavailable, corrupt | retry, backup | surface after retries | retry <= 3 |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| ` + "`Widget`" + ` | in-process | db row | single writer |
`

const fixtureMatrix = "# Widget machine - contract and oracle\n\n" +
	"## (a) Named-unit contract table\n\n" +
	"| name | kind | signature | pre / post | maps to |\n" +
	"|---|---|---|---|---|\n" +
	"| `saveWidget` | actor | `(input) -> row \\| err` | atomic persist | `db` |\n" +
	"| `guardCanPublish` | guard | `(ctx,evt) -> bool` | actor may publish | inv `widget-owned` |\n" +
	"| `setPending` | action | `(ctx) -> ctx` | stash pending | - |\n" +
	"| `recordDenied` | action | `(ctx) -> ctx` | rejection reason | surfaces `widget-owned` |\n" +
	"| `commit` | action | `(ctx) -> ctx` | commit status | - |\n" +
	"| `recordError` / `recordTimeout` | action | `(ctx) -> ctx` | classify error | - |\n\n" +
	"## (c) Transition matrix\n\n" +
	"| # | source | event / after / always | guard | target | actions |\n" +
	"|---|---|---|---|---|---|\n" +
	"| 1 | Draft | publish | guardCanPublish | persisting | setPending |\n" +
	"| 2 | Draft | publish | !guardCanPublish | Draft (internal) | recordDenied |\n" +
	"| 3 | persisting | invoke onDone | - | Published | commit |\n" +
	"| 4 | persisting | invoke onError | - | Draft | recordError |\n" +
	"| 5 | persisting | after persistTimeout | - | Draft | recordTimeout |\n"

const fixtureBuild = `# BUILD

Mode: full (self-contained).

## Traceability

| invariant | enforced by |
|---|---|
| widget-owned | guardCanPublish |

## State migration

No persisted instances yet.

## Toolchain and versions

Go 1.26; stdlib testing.
`

const fixtureGoMod = "module widget\n\ngo 1.22\n"

const fixtureAppGo = "package app\n\nimport (\n\t\"fmt\"\n\n\t\"widget/internal/store\"\n)\n\nfunc Run() { fmt.Println(store.Open()) }\n"

const fixtureStoreGo = "package store\n\nimport \"example.com/dbdriver\"\n\nfunc Open() string { return dbdriver.Name }\n"

// writeFixtureDesign mirrors fixtures.write_design.
func writeFixtureDesign(t *testing.T, root string) string {
	t.Helper()
	d := filepath.Join(root, "design")
	mustMkdir(t, filepath.Join(d, "machines"))
	mustWrite(t, filepath.Join(d, "widget.modelith.yaml"), fixtureModelith)
	mustWrite(t, filepath.Join(d, "workspace.dsl"), fixtureWorkspaceDSL)
	mustWrite(t, filepath.Join(d, "ARCHITECTURE.md"), fixtureArchitecture)
	mustWrite(t, filepath.Join(d, "BUILD.md"), fixtureBuild)
	mustWrite(t, filepath.Join(d, "machines", "Widget.machine.json"), fixtureMachineJSON)
	mustWrite(t, filepath.Join(d, "machines", "Widget.matrix.md"), fixtureMatrix)
	regenOracle(t, d, "Widget")
	return d
}

// writeFixtureImpl mirrors fixtures.write_impl.
func writeFixtureImpl(t *testing.T, root string) string {
	t.Helper()
	impl := filepath.Join(root, "impl")
	mustMkdir(t, filepath.Join(impl, "internal", "app"))
	mustMkdir(t, filepath.Join(impl, "internal", "store"))
	mustWrite(t, filepath.Join(impl, "go.mod"), fixtureGoMod)
	mustWrite(t, filepath.Join(impl, "internal", "app", "app.go"), fixtureAppGo)
	mustWrite(t, filepath.Join(impl, "internal", "store", "store.go"), fixtureStoreGo)
	return impl
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// editFile mirrors _edit: fail on fixture drift, then replace.
func editFile(t *testing.T, path, old, new string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("fixture drift: %q not in %s", old, path)
	}
	mustWrite(t, path, strings.Replace(string(data), old, new, 1))
}

// regenOracle re-renders the committed oracle from the machine on disk.
func regenOracle(t *testing.T, design, name string) {
	t.Helper()
	mp := filepath.Join(design, "machines", name+".machine.json")
	m, err := ir.LoadMachineJSON(mp)
	if err != nil {
		t.Fatal(err)
	}
	// G3 renders with the machine path as sourceName; match it exactly.
	mustWrite(t, filepath.Join(design, "machines", name+".oracle.md"), oracle.Render(m, mp))
}

func fixture(t *testing.T) (design, impl string) {
	t.Helper()
	root := t.TempDir()
	return writeFixtureDesign(t, root), writeFixtureImpl(t, root)
}

// ------------------------------ baseline ----------------------------------

func TestSyntheticDesignPassesAllGates(t *testing.T) {
	design, impl := fixture(t)
	for _, g := range []*gates.Gate{
		gates.CheckC4(design), gates.CheckMachines(design),
		gates.CheckTraceability(design), gates.CheckImports(design, impl),
	} {
		if len(g.Errs) != 0 || len(g.Drift) != 0 {
			t.Errorf("%s: errs=%v drift=%v", g.Title, g.Errs, g.Drift)
		}
	}
}

func TestGatesReportWhatTheyChecked(t *testing.T) {
	design, impl := fixture(t)
	g2 := gates.CheckC4(design)
	if g2.Counts["boundaries"] != 2 {
		t.Errorf("g2 boundaries=%d", g2.Counts["boundaries"])
	}
	if g2.Counts["dsl elements"] < 3 {
		t.Errorf("g2 dsl elements=%d", g2.Counts["dsl elements"])
	}
	if g2.Counts["dependencies with mitigation rows"] < 1 {
		t.Errorf("g2 mitigation rows=%d", g2.Counts["dependencies with mitigation rows"])
	}
	g3 := gates.CheckMachines(design)
	if g3.Counts["machines"] != 1 || g3.Counts["transitions"] != 5 || g3.Counts["oracles fresh"] != 1 {
		t.Errorf("g3 counts=%v", g3.Counts)
	}
	gx := gates.CheckTraceability(design)
	if gx.Counts["lifecycle machines traced"] != 1 || gx.Counts["invariants enforced"] != 1 {
		t.Errorf("gx counts=%v", gx.Counts)
	}
	g4 := gates.CheckImports(design, impl)
	if g4.Counts["go files checked"] != 2 || g4.Counts["edges verified"] < 2 {
		t.Errorf("g4 counts=%v", g4.Counts)
	}
}

// ----------------------- experiment A: empty design ------------------------

func TestEmptyDesignFailsEveryGate(t *testing.T) {
	d := filepath.Join(t.TempDir(), "design")
	mustMkdir(t, d)
	mustWrite(t, filepath.Join(d, "ARCHITECTURE.md"),
		"## 4. Architecture Contract\n\n```yaml\ncontract_version: 1\n```\n")
	if len(gates.CheckC4(d).Errs) == 0 {
		t.Error("empty design passed G2")
	}
	if len(gates.CheckMachines(d).Errs) == 0 {
		t.Error("empty design passed G3")
	}
	if len(gates.CheckTraceability(d).Errs) == 0 {
		t.Error("empty design passed Gx")
	}
}

func TestMissingImplDirIsError(t *testing.T) {
	design, _ := fixture(t)
	if len(gates.CheckImports(design, "/nonexistent/impl").Errs) == 0 {
		t.Error("missing impl dir passed G4")
	}
}

func TestImplWithNoSourceIsError(t *testing.T) {
	design, _ := fixture(t)
	empty := filepath.Join(t.TempDir(), "emptyimpl")
	mustMkdir(t, empty)
	g := gates.CheckImports(design, empty)
	if !containsAnyOf(g.Errs, "checked nothing", "no imports", "no source files", "nothing checked") {
		t.Errorf("empty impl errs=%v", g.Errs)
	}
}

// --------------------- experiment B: mitigation table ----------------------

func TestDeletedMitigationTableIsError(t *testing.T) {
	design, _ := fixture(t)
	arch := filepath.Join(design, "ARCHITECTURE.md")
	data, _ := os.ReadFile(arch)
	text := string(data)
	start := strings.Index(text, "## 6.")
	end := strings.Index(text, "## 7.")
	mustWrite(t, arch, text[:start]+text[end:])
	g := gates.CheckC4(design)
	if !containsAnyOf(g.Errs, "no mitigation", "mitigation row") {
		t.Errorf("errs=%v", g.Errs)
	}
}

func TestUnknownDependencyInMitigationTableIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "| `db` |", "| `dbz` |")
	if !containsAny(gates.CheckC4(design).Errs, "`dbz`") {
		t.Error("unknown mitigation dependency passed G2")
	}
}

// --------------------- experiment H: contract locator ----------------------

func TestContractFoundDespiteEarlierYamlBlock(t *testing.T) {
	design, _ := fixture(t)
	if gates.CheckC4(design).Counts["boundaries"] != 2 {
		t.Error("decoy yaml block won over the contract")
	}
}

func TestDuplicateBoundaryIDIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"  - id: widget.store", "  - id: widget.app")
	if !containsAny(gates.CheckC4(design).Errs, "duplicate boundary id") {
		t.Error("duplicate boundary id passed G2")
	}
}

func TestEdgeBothAllowedAndDeniedIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		`- "widget.app -> external.db"`, `- "widget.app -> widget.store"`)
	if !containsAny(gates.CheckC4(design).Errs, "both allowed and denied") {
		t.Error("allowed+denied edge passed G2")
	}
}

func TestRuleReferencingUndeclaredBoundaryIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"- widget.app -> widget.store", "- widget.app -> widget.ghost")
	if !containsAny(gates.CheckC4(design).Errs, "undeclared boundary 'widget.ghost'") {
		t.Error("undeclared boundary passed G2")
	}
}

func TestMissingWorkspaceDSLIsError(t *testing.T) {
	design, _ := fixture(t)
	if err := os.Remove(filepath.Join(design, "workspace.dsl")); err != nil {
		t.Fatal(err)
	}
	if !containsAny(gates.CheckC4(design).Errs, "workspace.dsl") {
		t.Error("missing workspace.dsl passed G2")
	}
}

func TestBoundaryWithoutDSLElementIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "workspace.dsl"),
		`storelib = component "Store"`, `storelibX = component "Store"`)
	if !containsAny(gates.CheckC4(design).Errs, "maps to no workspace.dsl element") {
		t.Error("unbound boundary passed G2")
	}
}

// ------------------- experiments D/E: machines and oracle -------------------

func TestStaleOracleIsDrift(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.machine.json"),
		`"actions":"setPending"`, `"actions":"setPendingRenamed"`)
	if !containsAny(gates.CheckMachines(design).Drift, "stale") {
		t.Error("stale committed oracle passed G3")
	}
}

func TestMissingOracleIsError(t *testing.T) {
	design, _ := fixture(t)
	if err := os.Remove(filepath.Join(design, "machines", "Widget.oracle.md")); err != nil {
		t.Fatal(err)
	}
	if !containsAny(gates.CheckMachines(design).Errs, "no committed oracle") {
		t.Error("missing oracle passed G3")
	}
}

func TestRetargetedTransitionIsMatrixDrift(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.machine.json"),
		`"onDone":{"target":"Published","actions":"commit"}`,
		`"onDone":{"target":"Draft","actions":"commit"}`)
	regenOracle(t, design, "Widget")
	if !containsAny(gates.CheckMachines(design).Drift, "no matrix row") {
		t.Error("retargeted transition passed G3 reconciliation")
	}
}

func TestUnitWithoutNamedUnitRowIsDrift(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"| `guardCanPublish` | guard |", "| `guardCanPublishX` | guard |")
	if !containsAny(gates.CheckMachines(design).Drift,
		"guard 'guardCanPublish' has no named-unit contract row") {
		t.Error("missing named-unit row passed G3")
	}
}

// ----------------------- experiment G + Gx hardening -----------------------

func TestUnenforcedInvariantIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"inv `widget-owned`", "inv `widget-possessed`")
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"surfaces `widget-owned`", "surfaces it")
	editFile(t, filepath.Join(design, "BUILD.md"), "| widget-owned |", "| nothing |")
	g := gates.CheckTraceability(design)
	found := false
	for _, e := range g.Errs {
		if strings.Contains(e, "widget-owned") && strings.Contains(e, "enforced by nothing") {
			found = true
		}
	}
	if !found {
		t.Errorf("unenforced invariant passed Gx: %v", g.Errs)
	}
}

func TestInvariantMatchIsWholeToken(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"inv `widget-owned`", "inv `widget-owned-by-nobody`")
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"surfaces `widget-owned`", "surfaces nothing")
	editFile(t, filepath.Join(design, "BUILD.md"),
		"| widget-owned |", "| widget-owned-by-nobody |")
	if !containsAny(gates.CheckTraceability(design).Errs, "'widget-owned' is referenced by no") {
		t.Error("substring invariant match passed Gx")
	}
}

func TestOrphanMapsToReferenceIsDrift(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.matrix.md"),
		"inv `widget-owned`", "inv `widget-owned` and `stale-invariant-ref`")
	if !containsAny(gates.CheckTraceability(design).Drift, "stale-invariant-ref") {
		t.Error("orphan maps-to reference passed Gx")
	}
}

func TestMachineStateNotInEnumIsError(t *testing.T) {
	design, _ := fixture(t)
	mp := filepath.Join(design, "machines", "Widget.machine.json")
	editFile(t, mp, `"Published":{"type":"final"},`,
		`"Published":{"type":"final"},
 "Archived":{"type":"final"},`)
	editFile(t, mp, `"publish":[`, `"archive":{"target":"Archived"},"publish":[`)
	regenOracle(t, design, "Widget")
	if !containsAny(gates.CheckTraceability(design).Errs,
		"'Archived' is not a value of enum WidgetStatus") {
		t.Error("state outside enum passed Gx (the SagaStatus/FailedDirty drift class)")
	}
}

func TestEnumValueWithoutStateIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "widget.modelith.yaml"),
		"      - name: Published", "      - name: Published\n      - name: Retired")
	if !containsAny(gates.CheckTraceability(design).Errs, "'Retired' has no machine state") {
		t.Error("enum value without state passed Gx")
	}
}

func TestMachineEventNotAnActionIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.machine.json"),
		`"publish":[`, `"mysteryEvent":{"target":"Published"},"publish":[`)
	regenOracle(t, design, "Widget")
	if !containsAny(gates.CheckTraceability(design).Errs,
		"'mysteryEvent' is not a Modelith action") {
		t.Error("unknown event passed Gx")
	}
}

func TestUnmappedMachineIsError(t *testing.T) {
	design, _ := fixture(t)
	src := filepath.Join(design, "machines", "Widget.machine.json")
	dst := filepath.Join(design, "machines", "Gadget.machine.json")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, dst, string(data))
	regenOracle(t, design, "Gadget")
	if !containsAny(gates.CheckTraceability(design).Errs,
		"Gadget.machine.json: maps to no Modelith entity") {
		t.Error("unmapped machine passed Gx (the hardcoded-lifecycle hole)")
	}
}

func TestOperationalRoleIsAccepted(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.machine.json"),
		`{"id":"widget",`, `{"id":"widget","_role":"operational",`)
	regenOracle(t, design, "Widget")
	g := gates.CheckTraceability(design)
	if containsAny(g.Errs, "maps to no Modelith entity") {
		t.Errorf("operational role rejected: %v", g.Errs)
	}
	if !containsAny(g.Errs, "has lifecycle enum WidgetStatus but no machine") {
		t.Errorf("entity without lifecycle machine passed Gx: %v", g.Errs)
	}
}

func TestPlacementRowWithoutMachineIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"| `Widget` | in-process |", "| `Widget` | in-process |\n| `Gizmo` | actor |")
	if !containsAny(gates.CheckTraceability(design).Errs, "`Gizmo` has no machine") {
		t.Error("placement row without machine passed Gx")
	}
}

func TestPlacementWaiverIsAccepted(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"| `Widget` | in-process | db row | single writer |",
		"| `Widget` | in-process | db row | single writer |\n"+
			"| `Gizmo` | pure function (no machine: stateless transform) | - | - |")
	if containsAny(gates.CheckTraceability(design).Errs, "Gizmo") {
		t.Error("placement waiver rejected")
	}
}

// ----------------------- experiment F1: import bypass -----------------------

func TestSingleFormImportViolationIsCaught(t *testing.T) {
	design, impl := fixture(t)
	mustWrite(t, filepath.Join(impl, "internal", "app", "sneaky.go"),
		"package app\n\nimport \"example.com/dbdriver\"\n\nvar _ = dbdriver.Name\n")
	if !containsAny(gates.CheckImports(design, impl).Errs, "widget.app -> external.db is denied") {
		t.Error("single-form import bypassed G4")
	}
}

func TestParenFormImportViolationIsCaught(t *testing.T) {
	design, impl := fixture(t)
	mustWrite(t, filepath.Join(impl, "internal", "app", "sneaky.go"),
		"package app\n\nimport (\n\t\"example.com/dbdriver\"\n)\n\nvar _ = dbdriver.Name\n")
	if !containsAny(gates.CheckImports(design, impl).Errs, "widget.app -> external.db is denied") {
		t.Error("paren-form import bypassed G4")
	}
}

func TestUndeclaredCrossBoundaryEdgeIsError(t *testing.T) {
	design, impl := fixture(t)
	mustWrite(t, filepath.Join(impl, "internal", "store", "back.go"),
		"package store\n\nimport \"widget/internal/app\"\n\nvar _ = app.Run\n")
	if !containsAny(gates.CheckImports(design, impl).Errs,
		"undeclared cross-boundary edge widget.store -> widget.app") {
		t.Error("undeclared cross-boundary edge passed G4")
	}
}

func TestImportOfUnexposedInternalsIsError(t *testing.T) {
	design, impl := fixture(t)
	mustMkdir(t, filepath.Join(impl, "internal", "store", "inner"))
	mustWrite(t, filepath.Join(impl, "internal", "store", "inner", "inner.go"),
		"package inner\n\nvar Name = \"x\"\n")
	mustWrite(t, filepath.Join(impl, "internal", "app", "deep.go"),
		"package app\n\nimport \"widget/internal/store/inner\"\n\nvar _ = inner.Name\n")
	if !containsAny(gates.CheckImports(design, impl).Errs, "not in the exposes list of widget.store") {
		t.Error("unexposed internal import passed G4")
	}
}

func TestSourceOutsideContractIsError(t *testing.T) {
	design, impl := fixture(t)
	mustMkdir(t, filepath.Join(impl, "internal", "rogue"))
	mustWrite(t, filepath.Join(impl, "internal", "rogue", "r.go"), "package rogue\n")
	if !containsAny(gates.CheckImports(design, impl).Errs, "maps to no contract boundary") {
		t.Error("rogue package passed G4")
	}
}

func TestContractIgnoreGlobsAreRespected(t *testing.T) {
	design, impl := fixture(t)
	mustMkdir(t, filepath.Join(impl, "internal", "scaffold"))
	mustWrite(t, filepath.Join(impl, "internal", "scaffold", "s.go"), "package scaffold\n")
	g := gates.CheckImports(design, impl)
	if containsAny(g.Errs, "scaffold") {
		t.Errorf("ignore glob not respected: %v", g.Errs)
	}
	if g.Counts["files ignored by contract"] != 1 {
		t.Errorf("ignored count=%d", g.Counts["files ignored by contract"])
	}
}

func TestPythonImportsAreChecked(t *testing.T) {
	design, _ := fixture(t)
	impl := filepath.Join(t.TempDir(), "pyimpl")
	mustMkdir(t, filepath.Join(impl, "internal", "app"))
	mustMkdir(t, filepath.Join(impl, "internal", "store"))
	mustWrite(t, filepath.Join(impl, "internal", "app", "app.py"),
		"import example.com  # not internal\nfrom internal.store import store\n")
	mustWrite(t, filepath.Join(impl, "internal", "store", "store.py"), "X = 1\n")
	g := gates.CheckImports(design, impl)
	if containsAny(g.Errs, "denied") {
		t.Errorf("allowed python edge denied: %v", g.Errs)
	}
	if g.Counts["python files checked"] != 2 {
		t.Errorf("python files checked=%d", g.Counts["python files checked"])
	}
}

func containsAnyOf(findings []string, subs ...string) bool {
	for _, s := range subs {
		if containsAny(findings, s) {
			return true
		}
	}
	return false
}

// ------------------- 2026-07-02 review: port regressions -------------------

func TestContractFenceNotAMappingIsErrorNotPanic(t *testing.T) {
	design, _ := fixture(t)
	arch := filepath.Join(design, "ARCHITECTURE.md")
	data, _ := os.ReadFile(arch)
	text := string(data)
	start := strings.Index(text, "```yaml\ncontract_version")
	end := strings.Index(text[start:], "\n```") + start
	mustWrite(t, arch, text[:start]+"```yaml\n- a\n- b"+text[end:])
	g := gates.CheckC4(design)
	if !containsAnyOf(g.Errs, "not a mapping", "no Architecture Contract", "no contract_version") {
		t.Errorf("non-mapping contract fence: errs=%v", g.Errs)
	}
}

func TestEmptyModelithIsErrorNotPanic(t *testing.T) {
	design, _ := fixture(t)
	mustWrite(t, filepath.Join(design, "widget.modelith.yaml"), "")
	if !containsAnyOf(gates.CheckTraceability(design).Errs, "not a yaml mapping", "no entities") {
		t.Error("empty modelith passed Gx")
	}
}

func TestModelithWithoutEntitiesIsError(t *testing.T) {
	design, _ := fixture(t)
	mustWrite(t, filepath.Join(design, "widget.modelith.yaml"),
		"kind: modelith\nversion: 1\ntitle: Widget\n")
	if !containsAny(gates.CheckTraceability(design).Errs, "no entities") {
		t.Error("entity-less modelith passed Gx")
	}
}

func TestG4OnlyRunWithUnparseableRuleDoesNotPanic(t *testing.T) {
	design, impl := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"- widget.app -> widget.store", "- not-an-edge-rule")
	// must not panic; the rule itself is G2's finding
	g := gates.CheckImports(design, impl)
	_ = g
	if !containsAny(gates.CheckC4(design).Errs, "unparseable") {
		t.Error("G2 must report the unparseable rule")
	}
}

func TestG4FollowsDirectorySymlinks(t *testing.T) {
	design, impl := fixture(t)
	// move the store boundary code outside the impl tree, reachable only via symlink
	outside := filepath.Join(t.TempDir(), "outside-store")
	mustMkdir(t, outside)
	mustWrite(t, filepath.Join(outside, "store.go"), fixtureStoreGo)
	// sneak a denied import into the symlinked dir
	mustWrite(t, filepath.Join(outside, "sneaky.go"),
		"package store\n\nimport \"widget/internal/app\"\n\nvar _ = app.Run\n")
	storeDir := filepath.Join(impl, "internal", "store")
	if err := os.RemoveAll(storeDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, storeDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	g := gates.CheckImports(design, impl)
	if !containsAny(g.Errs, "undeclared cross-boundary edge widget.store -> widget.app") {
		t.Errorf("code behind a directory symlink is invisible to G4: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestDebacktickedPlacementRowIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"| `Widget` | in-process |", "| Widget | in-process |")
	if !containsAny(gates.CheckTraceability(design).Errs, "names no component in backticks") {
		t.Error("de-backticked placement row passed Gx")
	}
}

func TestNullOnIsLintErrorNotCrash(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "machines", "Widget.machine.json"),
		`"on":{"publish":[`, `"on2":null,"on":{"publish":[`)
	// on2 is an unsupported key AND null containers elsewhere must not crash;
	// exercise the direct null case too
	mp := filepath.Join(design, "machines", "Widget.machine.json")
	data, _ := os.ReadFile(mp)
	mustWrite(t, mp, strings.Replace(string(data), `"on2":null,"on":{"publish":[`, `"on":null,"unused":{"publish":[`, 1))
	g := gates.CheckMachines(design) // must not panic
	if len(g.Errs) == 0 {
		t.Errorf("machine with \"on\": null passed G3: %v", g.Errs)
	}
}

func TestNonObjectMachineIsErrorNotPanic(t *testing.T) {
	design, _ := fixture(t)
	mustWrite(t, filepath.Join(design, "machines", "Gadget.machine.json"), `"hello"`)
	mustWrite(t, filepath.Join(design, "machines", "Gadget.oracle.md"), "stale")
	g := gates.CheckMachines(design) // must not panic
	if !containsAny(g.Errs, "not an object") {
		t.Errorf("non-object machine: errs=%v", g.Errs)
	}
	gx := gates.CheckTraceability(design) // must not panic either
	_ = gx
}

func TestTrailingGarbageMachineJSONIsRejected(t *testing.T) {
	design, _ := fixture(t)
	mp := filepath.Join(design, "machines", "Widget.machine.json")
	data, _ := os.ReadFile(mp)
	mustWrite(t, mp, string(data)+`{"junk": true}`)
	if !containsAny(gates.CheckMachines(design).Errs, "invalid JSON") {
		t.Error("trailing garbage after the machine object was accepted")
	}
}

// ---------------- template-conformance checks (2026-07-02) ----------------

func TestMissingModeDeclarationIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "BUILD.md"), "Mode: full (self-contained).\n\n", "")
	if !containsAny(gates.CheckTraceability(design).Errs, "declares no mode") {
		t.Error("BUILD.md without a mode declaration passed Gx")
	}
}

func TestMissingToolchainSectionIsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "BUILD.md"), "## Toolchain and versions", "## Nothing here")
	if !containsAny(gates.CheckTraceability(design).Errs, "no Toolchain heading") {
		t.Error("BUILD.md without toolchain pins passed Gx")
	}
}

func TestMissingStateMigrationSectionIsErrorWhenPersisted(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "BUILD.md"), "## State migration", "## Something else")
	if !containsAny(gates.CheckTraceability(design).Errs, "no State migration heading") {
		t.Error("persisted placement without a state-migration section passed Gx")
	}
	// but an all-in-memory design does not need the section
	d2, _ := fixture(t)
	editFile(t, filepath.Join(d2, "ARCHITECTURE.md"),
		"| `Widget` | in-process | db row | single writer |",
		"| `Widget` | in-process | in-memory | single writer |")
	editFile(t, filepath.Join(d2, "BUILD.md"), "## State migration", "## Something else")
	if containsAny(gates.CheckTraceability(d2).Errs, "no State migration heading") {
		t.Error("in-memory-only design required a state-migration section")
	}
}

func TestContractVersion1IsError(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"), "contract_version: 2", "contract_version: 1")
	if !containsAny(gates.CheckC4(design).Errs, "contract_version 1 is not supported") {
		t.Error("v1 contract passed G2 (value was not checked, only presence)")
	}
}

func TestQueueCoupledDesignRequiresEventContractTable(t *testing.T) {
	design, _ := fixture(t)
	editFile(t, filepath.Join(design, "workspace.dsl"),
		`db       = container "DB" "The database." "SQLite" "Database"`,
		`db       = container "DB" "The database." "SQLite" "Database"
      bus      = container "Bus" "The broker." "NATS" "Queue"`)
	// bus now needs a mitigation row too; add one so only the event-contract error remains
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"| `db` | unavailable, corrupt | retry, backup | surface after retries | retry <= 3 |",
		"| `db` | unavailable, corrupt | retry, backup | surface after retries | retry <= 3 |\n"+
			"| `bus` | down, redelivery | outbox, dedupe | at-least-once | ack window |")
	g := gates.CheckC4(design)
	if !containsAny(g.Errs, "no event-contract table") {
		t.Errorf("queue-coupled design without an event-contract table passed G2: %v", g.Errs)
	}
	// adding the table satisfies it
	editFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"## 6. Dependency mitigation posture",
		"## 5b. Event contracts\n\n"+
			"| event | producer | consumer | payload | delivery | ordering | dedupe |\n"+
			"|---|---|---|---|---|---|---|\n"+
			"| widget.published | app | store | Widget.status | at-least-once | none | widgetId |\n\n"+
			"## 6. Dependency mitigation posture")
	g2 := gates.CheckC4(design)
	if containsAny(g2.Errs, "no event-contract table") {
		t.Errorf("event-contract table not recognized: %v", g2.Errs)
	}
}
