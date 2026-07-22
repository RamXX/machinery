package experiments

// End-to-end regressions for the MAC-r466 gate-review findings, adapted from
// the reviewer's reproduction fixtures (agent-gates f1/f2/f4/f9 and
// agent-newgates f1..f8). Each scenario once passed silently; every one must
// now fail loudly or behave per the recorded decision. Finding-level unit
// tests live next to the gates (internal/gates, internal/hook); these hold
// the multi-artifact shapes together.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/gates"
)

func writeReviewFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gateErrs(gs []*gates.Gate, title string) []string {
	for _, g := range gs {
		if strings.Contains(g.Title, title) {
			return g.Errs
		}
	}
	return nil
}

func hasGate(gs []*gates.Gate, title string) bool {
	for _, g := range gs {
		if strings.Contains(g.Title, title) {
			return true
		}
	}
	return false
}

// reviewParentContract is the machine-less decomposed parent's contract with
// a DENIED edge the impl violates (agent-gates/f1 shape, minimized).
const reviewParentContract = `# Architecture

## Architecture Contract

` + "```yaml" + `
contract_version: 2
boundaries:
  - id: orders
    code: ["orders/**"]
  - id: payments
    code: ["payments/**"]
dependency_rules:
  allow: []
  deny:
    - "orders -> payments"
` + "```" + `
`

// GATE-1 (agent-gates/f1): a machine-less decomposed parent with an explicit
// --impl must run G4; v0.3.x narrowed it away and exited 0 over
// contract-DENIED edges. Gt stays skipped, named in the note.
func TestReviewNarrowedParentWithImplRunsG4(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	writeReviewFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeReviewFile(t, filepath.Join(design, "ARCHITECTURE.md"), reviewParentContract)
	writeReviewFile(t, filepath.Join(impl, "go.mod"), "module example.com/shop\n")
	writeReviewFile(t, filepath.Join(impl, "orders", "orders.go"),
		"package orders\n\nimport \"example.com/shop/payments\"\n\nvar _ = payments.Charge\n")
	writeReviewFile(t, filepath.Join(impl, "payments", "payments.go"), "package payments\n\nfunc Charge() {}\n")

	sel, err := gates.Select(design, "", impl)
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["g4"] || sel.Run["gt"] {
		t.Fatalf("narrowed parent with impl must keep g4 and skip gt: %v", sel.Run)
	}
	if !strings.Contains(sel.Note, "gt skipped: no machines") {
		t.Fatalf("the note must name the skipped gt explicitly: %q", sel.Note)
	}
	out := gates.RunSelected(design, impl, sel)
	if !hasGate(out, "G4-import") {
		t.Fatal("G4 did not run on the narrowed parent although --impl was supplied")
	}
	joined := strings.Join(gateErrs(out, "G4-import"), "\n")
	if !strings.Contains(joined, "orders -> payments is denied by the contract") {
		t.Fatalf("the contract-DENIED edge must fail loudly: %q", joined)
	}
	if hasGate(out, "Gt-tests") {
		t.Fatal("Gt is machine-dependent and stays skipped on the narrowed parent")
	}
}

// GATE-2 (agent-gates/f2): a design path carrying glob metacharacters must
// not defeat machine detection and silently narrow g3/gx away.
func TestReviewMetacharPathDoesNotNarrow(t *testing.T) {
	design := filepath.Join(t.TempDir(), "des[1]")
	writeReviewFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	writeReviewFile(t, filepath.Join(design, "machines", "Order.machine.json"), "{}\n")
	sel, err := gates.Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Note != "" || !sel.Run["g3"] || !sel.Run["gx"] {
		t.Fatalf("metachar path must not trigger the machine-less narrowing: note=%q run=%v", sel.Note, sel.Run)
	}
}

// GATE-4 (agent-gates/f4): one comment line naming every oracle once
// wholesale-covered hundreds of rows with zero assertions.
func TestReviewCommentMentionCoversNoOracle(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	oracle := "# oracle\n\n| test id | stable id | source |\n|---|---|---|\n| T-A-01 | AAA-111111 | X |\n"
	oracle2 := "# oracle\n\n| test id | stable id | source |\n|---|---|---|\n| T-B-01 | BBB-222222 | X |\n"
	writeReviewFile(t, filepath.Join(design, "machines", "Deal.oracle.md"), oracle)
	writeReviewFile(t, filepath.Join(design, "machines", "Task.oracle.md"), oracle2)
	writeReviewFile(t, filepath.Join(impl, "x", "x_test.go"),
		"package x\n\n// TODO: see Deal.oracle.md Task.oracle.md for the tables we should eventually cover\nfunc nothing() {}\n")
	g := gates.CheckOracleCoverage(design, impl)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "Deal.oracle.md: 1 of 1 stable ids") ||
		!strings.Contains(joined, "Task.oracle.md: 1 of 1 stable ids") {
		t.Fatalf("comment mentions must cover nothing; every oracle stays uncovered: %v", g.Errs)
	}
	if g.Counts["machines covered by conformance parse"] != 0 {
		t.Fatalf("no conformance parse may be credited: %+v", g.Counts)
	}
}

// GATE-7 (agent-gates/f9): baseline: ["* -> *"] amnestied the entire edge
// space. G2 now hard-errors on the rule, and G4 refuses to match it, so the
// denied edge stays loud in both front doors.
func TestReviewWildcardBaselineFailsBothGates(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	contract := strings.Replace(reviewParentContract,
		"dependency_rules:\n  allow: []\n",
		"dependency_rules:\n  baseline: [\"* -> *\"]\n  allow: []\n", 1)
	writeReviewFile(t, filepath.Join(design, "ARCHITECTURE.md"), contract)
	writeReviewFile(t, filepath.Join(impl, "go.mod"), "module example.com/shop\n")
	writeReviewFile(t, filepath.Join(impl, "orders", "orders.go"),
		"package orders\n\nimport \"example.com/shop/payments\"\n\nvar _ = payments.Charge\n")
	writeReviewFile(t, filepath.Join(impl, "payments", "payments.go"), "package payments\n\nfunc Charge() {}\n")

	g2 := gates.CheckC4(design)
	if !strings.Contains(strings.Join(g2.Errs, "\n"), "wildcard") {
		t.Fatalf("G2 must hard-error on the wildcard baseline rule: %v", g2.Errs)
	}
	g4 := gates.CheckImports(design, impl)
	if !strings.Contains(strings.Join(g4.Errs, "\n"), "orders -> payments is denied by the contract") {
		t.Fatalf("the denied edge must stay loud under a wildcard baseline: %v", g4.Errs)
	}
}

// NG-1 + NG-7 (agent-newgates/f1, f2, f4): production .rs files with inline
// #[cfg(test)] modules are split, never skipped: G4 sees production imports
// and unmapped files; Gt sees only the cfg(test) span.
func TestReviewRustCfgTestSplit(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	writeReviewFile(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n"+
		"```yaml\ncontract_version: 2\nboundaries:\n  - id: app\n    code: [\"app/**\"]\n```\n")
	writeReviewFile(t, filepath.Join(design, "machines", "Thing.oracle.md"),
		"# o\n\n| test id | stable id | source |\n|---|---|---|\n| T-THIN-01 | THIN-aaa111 | A |\n")
	// production ids + inline test module, outside every boundary (f1+f4)
	writeReviewFile(t, filepath.Join(impl, "rogue.rs"),
		"pub const T1: &str = \"THIN-aaa111\";\n\npub fn production_logic() {}\n\n"+
			"#[cfg(test)]\nmod tests {\n    #[test]\n    fn t() { assert!(true); }\n}\n")

	g4 := gates.CheckImports(design, impl)
	if !strings.Contains(strings.Join(g4.Errs, "\n"), "source file rogue.rs maps to no contract boundary") {
		t.Fatalf("the unmapped production .rs file must be an ERROR: %v", g4.Errs)
	}
	gt := gates.CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(gt.Errs, "\n"), "1 of 1 stable ids appear in no test file") {
		t.Fatalf("production .rs text must not feed Gt's corpus: %v", gt.Errs)
	}
}

// NG-4 + NG-5 + NG-6 + NG-8 (agent-newgates/f5, f6, f7, f8): the Gb plan
// checks are fence-correct and literal about waivers and numbering.
func TestReviewBuildPlanShapes(t *testing.T) {
	check := func(t *testing.T, build, wantErr string) {
		t.Helper()
		design := t.TempDir()
		writeReviewFile(t, filepath.Join(design, "BUILD.md"), build)
		writeReviewFile(t, filepath.Join(design, "machines", "Thing.oracle.md"),
			"# o\n\n| test id | stable id | source |\n|---|---|---|\n| T-CMD-01 | CMD-abc123 | A |\n")
		g := gates.CheckBuildPlan(design)
		if !strings.Contains(strings.Join(g.Errs, "\n"), wantErr) {
			t.Fatalf("want error containing %q, got %v", wantErr, g.Errs)
		}
	}
	t.Run("f5 fenced Mode does not override", func(t *testing.T) {
		check(t, "# B\n\nExample:\n\n```text\nMode: manifest (shards under BUILD/)\n```\n\nMode: full (single BUILD.md)\n\n"+
			"## 9. Build plan\n\n**M0 - Data layer.** No definition of done here and not a walking skeleton.\n",
			"states no definition of done")
	})
	t.Run("f6 prose N/A is not a waiver", func(t *testing.T) {
		check(t, "# B\n\nMode: full\n\n## Build plan\n\nN/A rows in the oracle table are excluded from milestone scoping.\n",
			"not in the waiver form")
	})
	t.Run("f7 M1 and M01 collide", func(t *testing.T) {
		check(t, "# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n"+
			"**M1 - First slice.** DoD: rows green.\n\n**M01 - Also the first slice.** DoD: rows green.\n",
			"milestone M1 is declared 2 times")
	})
	t.Run("f8 nested fence does not leak a DoD", func(t *testing.T) {
		check(t, "# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n"+
			"**M1 - Breadth slice.** This milestone has NO real DoD. Example snippet:\n\n"+
			"````markdown\n```text\nDoD: example text inside a documentation fence, not a commitment.\n```\n````\n\n"+
			"More prose, still no definition of done for M1.\n",
			"milestone M1 (Breadth slice) states no definition of done")
	})
}

// GATE-5 (agent-gates/f5): the Gx placement waiver needs a reason in the
// component or machine-placement cell; an empty '(no machine:)' waives
// nothing.
func TestReviewPlacementWaiverEmptyReason(t *testing.T) {
	design := t.TempDir()
	writeReviewFile(t, filepath.Join(design, "domain.modelith.yaml"),
		"kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n    invariants:\n      - id: widget-owned\n")
	writeReviewFile(t, filepath.Join(design, "machines", "Ops.machine.json"),
		`{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`)
	writeReviewFile(t, filepath.Join(design, "ARCHITECTURE.md"),
		"# A\n\n## Placement\n\n| component | machine placement | persistence | concurrency |\n|---|---|---|---|\n"+
			"| `GhostComponent` | wherever (no machine:) | none | n/a |\n")
	g := gates.CheckTraceability(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "`GhostComponent` has no machine and no '(no machine: <reason>)' waiver") {
		t.Fatalf("an empty waiver reason must fail loudly: %v", g.Errs)
	}
}

// NG-9 (agent-newgates/f10): an unreadable committed artifact is a hard
// error naming the file, never silently an empty file.
func TestReviewUnreadableOracleIsHardError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not deny reads")
	}
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	writeReviewFile(t, filepath.Join(design, "machines", "Thing.oracle.md"),
		"# o\n\n| test id | stable id | source |\n|---|---|---|\n| T-THIN-01 | THIN-aaa111 | A |\n")
	writeReviewFile(t, filepath.Join(impl, "thing_test.go"), "package thing\n\n// THIN-aaa111\n")
	path := filepath.Join(design, "machines", "Thing.oracle.md")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	g := gates.CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "Thing.oracle.md is unreadable") {
		t.Fatalf("the unreadable oracle must be named in a hard error: %v", g.Errs)
	}
}
