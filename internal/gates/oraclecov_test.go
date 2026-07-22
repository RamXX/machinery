package gates

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

const covOracleMD = `# Generated transition oracle: thing

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-THIN-01 | THIN-aaa111 | A | on:go | - | B | - |
| T-THIN-02 | THIN-bbb222 | B | on:stop | - | A | - |
`

const covPolicyOracleMD = `# Generated authorization oracle: policy

## Decisions

| test id | stable id | rule | expectation | invariants |
|---|---|---|---|---|
| A-POL-01 | POL-111aaa | admin any | allow | rbac-admin |
| A-POL-02 | POL-222bbb | rep other | deny | rbac-scope |
`

// writeCovFixture builds a design and an impl tree; keys prefixed impl/ land
// under the impl dir.
func writeCovFixture(t *testing.T, files map[string]string) (design, impl string) {
	t.Helper()
	design, impl = t.TempDir(), t.TempDir()
	for name, content := range files {
		if rel, ok := strings.CutPrefix(name, "impl/"); ok {
			writeSuiteFile(t, filepath.Join(impl, rel), content)
			continue
		}
		writeSuiteFile(t, filepath.Join(design, name), content)
	}
	return design, impl
}

func TestCheckOracleCoverageClean(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.machine.json": `{"id": "thing", "initial": "A", "states": {"A": {}, "B": {}}}`,
		"machines/Thing.oracle.md":    covOracleMD,
		"impl/thing_test.go": "package thing\n\n// keyed on the oracle stable ids\n" +
			"// T-THIN-01_THIN-aaa111 and THIN-bbb222 are exercised here\n",
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Gt not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	want := map[string]int{"machines": 1, "oracle rows": 2, "ids covered by literal": 2, "test files scanned": 1}
	for count, n := range want {
		if g.Counts[count] != n {
			t.Errorf("Gt counted %s=%d, want %d: %+v", count, g.Counts[count], n, g.Counts)
		}
	}
}

func TestCheckOracleCoverageMissingIDs(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		// THIN-bbb222 appears only hyphen-glued: X-THIN-bbb222 is a
		// different id, not a citation
		"impl/thing_test.go": "package thing\n\n// THIN-aaa111 and X-THIN-bbb222\n",
	})
	g := CheckOracleCoverage(design, impl)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "Thing.oracle.md: 1 of 2 stable ids appear in no test file (THIN-bbb222)") {
		t.Fatalf("want the missing id named per machine, got %v", g.Errs)
	}
	if g.Counts["ids covered by literal"] != 1 {
		t.Errorf("ids covered by literal = %d, want 1: %+v", g.Counts["ids covered by literal"], g.Counts)
	}
}

// Only test files count: ids buried in production source prove nothing about
// the test suite.
func TestCheckOracleCoverageIgnoresProductionSources(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/thing.go":            "package thing\n\n// THIN-bbb222\n",
		"impl/thing_test.go":       "package thing\n\n// THIN-aaa111\n",
	})
	g := CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "1 of 2 stable ids appear in no test file (THIN-bbb222)") {
		t.Fatalf("ids in non-test files must not count: %v", g.Errs)
	}
	if g.Counts["test files scanned"] != 1 {
		t.Errorf("test files scanned = %d, want 1: %+v", g.Counts["test files scanned"], g.Counts)
	}
}

// An impl with ZERO test files fails with one loud corpus-level error, not
// per-machine missing-id errors whose remedy is impossible without tests, and
// the zero stays visible in the checked line.
func TestCheckOracleCoverageNoTestFilesFailsLoudly(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/thing.go":            "package thing\n\n// THIN-aaa111 THIN-bbb222\n",
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 1 || !strings.Contains(g.Errs[0], "no test files under "+impl) {
		t.Fatalf("want the single no-test-files error, got %v", g.Errs)
	}
	if !strings.Contains(g.Errs[0], "#[cfg(test)]") || !strings.Contains(g.Errs[0], "*_test.go") {
		t.Errorf("the error must hint at the supported test-file shapes: %v", g.Errs)
	}
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "0 test files scanned") {
		t.Errorf("checked line must show the zero corpus explicitly: %v", g.checkedExtra)
	}
}

// The wholesale (conformance-parse) citation is earned, not mentioned: the
// oracle file name must sit inside a string literal, in a file that shows
// parse evidence. A bare comment naming every oracle once wholesale-covered
// 272 rows with zero assertions (GATE-4).
func TestGtWholesaleNeedsStringLiteralAndParseEvidence(t *testing.T) {
	t.Run("comment mention covers nothing", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/Thing.oracle.md": covOracleMD,
			"impl/x_test.go": "package x\n\n// TODO: see Thing.oracle.md for the table we should cover\n" +
				"func nothing() {}\n",
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "2 of 2 stable ids appear in no test file") {
			t.Fatalf("a comment mention must not wholesale-cover the oracle: %v", g.Errs)
		}
	})
	t.Run("quoted mention without parse evidence covers nothing", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/Thing.oracle.md": covOracleMD,
			"impl/x_test.go": "package x\n\nconst oraclePath = " +
				"\"../../design/machines/Thing.oracle.md\"\n",
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "2 of 2 stable ids appear in no test file") {
			t.Fatalf("a path constant nothing parses must not wholesale-cover the oracle: %v", g.Errs)
		}
	})
	t.Run("parse evidence must live in the citing file", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/Thing.oracle.md": covOracleMD,
			"impl/a_test.go": "package x\n\nconst oraclePath = " +
				"\"../../design/machines/Thing.oracle.md\"\n",
			"impl/b_test.go": "package x\n" + parseEvidence,
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "2 of 2 stable ids appear in no test file") {
			t.Fatalf("evidence in another file must not launder the mention: %v", g.Errs)
		}
	})
}

// The file-name mention needs word boundaries on BOTH sides: [A-Za-z0-9_.-]
// neighbors disqualify, so purchase-order.oracle.md never wholesale-covers
// order.oracle.md (NG-2) and order.oracle.md.bak covers nothing (NG-3).
func TestGtCitationBoundaries(t *testing.T) {
	orderOracle := "| test id | stable id | source |\n|---|---|---|\n| T-ORD-01 | ORD-aaa111 | A |\n"
	purchaseOracle := "| test id | stable id | source |\n|---|---|---|\n| T-PUR-01 | PUR-ccc333 | A |\n"
	t.Run("hyphenated sibling does not cover", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/order.oracle.md":          orderOracle,
			"machines/purchase-order.oracle.md": purchaseOracle,
			"impl/purchase_test.go": "package p\n\nconst oraclePath = " +
				"\"../../design/machines/purchase-order.oracle.md\"\n" + parseEvidence,
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "order.oracle.md: 1 of 1 stable ids appear in no test file (ORD-aaa111)") {
			t.Fatalf("order must stay uncovered by the purchase-order mention: %v", g.Errs)
		}
		if g.Counts["machines covered by conformance parse"] != 1 {
			t.Errorf("machines covered by conformance parse = %d, want 1 (purchase-order): %+v",
				g.Counts["machines covered by conformance parse"], g.Counts)
		}
	})
	t.Run("a suffixed artifact is not the oracle", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/order.oracle.md": orderOracle,
			"impl/x_test.go": "package p\n\nconst notes = \"see order.oracle.md.bak and order.oracle.mdx for history\"\n" +
				parseEvidence,
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "order.oracle.md: 1 of 1 stable ids appear in no test file (ORD-aaa111)") {
			t.Fatalf(".bak/.mdx mentions must not cover the oracle: %v", g.Errs)
		}
	})
}

// Stable ids in PRODUCTION .rs text (a transition-table constant) prove
// nothing about the tests: only the #[cfg(test)] module span of a production
// .rs file feeds Gt's corpus (NG-7).
func TestGtRustProductionTextIsNotTestCorpus(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/src/machine.rs": "pub const T1: &str = \"THIN-aaa111\";\npub const T2: &str = \"THIN-bbb222\";\n\n" +
			"pub fn transition() {}\n\n#[cfg(test)]\nmod tests {\n    #[test]\n    fn unrelated() { assert!(true); }\n}\n",
	})
	g := CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "2 of 2 stable ids appear in no test file") {
		t.Fatalf("production .rs text must not cover oracle ids: %v", g.Errs)
	}
	// the same ids INSIDE the cfg(test) module do cover
	design2, impl2 := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/src/machine.rs": "pub fn transition() {}\n\n" +
			"#[cfg(test)]\nmod tests {\n    // exercises THIN-aaa111 and THIN-bbb222\n}\n",
	})
	if g2 := CheckOracleCoverage(design2, impl2); len(g2.Errs) != 0 {
		t.Fatalf("ids in the cfg(test) span must cover: %v", g2.Errs)
	}
}

// Rust test shapes: any *.rs under a tests/ directory and any .rs file with a
// #[cfg(test)] module both feed the corpus.
func TestCheckOracleCoverageRustTestShapes(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/tests/foo.rs":        "// exercises T-THIN-01_THIN-aaa111\n",
		"impl/src/lib.rs":          "pub fn f() {}\n\n#[cfg(test)]\nmod tests {\n    // THIN-bbb222\n}\n",
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("Rust test files must satisfy Gt: %v", g.Errs)
	}
	if g.Counts["test files scanned"] != 2 {
		t.Errorf("test files scanned = %d, want 2: %+v", g.Counts["test files scanned"], g.Counts)
	}
}

// parseEvidence is the conformance-parse fingerprint every real oracle test
// carries: a string literal containing the markdown table-row delimiter (the
// go-crm oracle tests split rows on "|").
const parseEvidence = "\nfunc cells(line string) []string { return strings.Split(strings.Trim(line, \"|\"), \"|\") }\n"

func TestCheckOracleCoverageConformanceParse(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/oracle_test.go": "package thing\n\nconst oraclePath = " +
			"\"../../design/machines/Thing.oracle.md\"\n" + parseEvidence,
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("a conformance-parsed machine is covered wholesale: %v", g.Errs)
	}
	if g.Counts["machines covered by conformance parse"] != 1 {
		t.Errorf("machines covered by conformance parse = %d, want 1: %+v",
			g.Counts["machines covered by conformance parse"], g.Counts)
	}
}

// The conformance-parse mention is a file-name token, not a substring: a
// test naming PurchaseOrder.oracle.md never covers a machine named Order.
func TestCheckOracleCoverageConformanceParseIsNotSubstring(t *testing.T) {
	orderOracle := "| test id | stable id | source |\n|---|---|---|\n| T-ORD-01 | ORD-aaa111 | A |\n"
	purchaseOracle := "| test id | stable id | source |\n|---|---|---|\n| T-PUR-01 | PUR-ccc333 | A |\n"
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Order.oracle.md":         orderOracle,
		"machines/PurchaseOrder.oracle.md": purchaseOracle,
		"impl/purchase_test.go": "package p\n\nconst oraclePath = " +
			"\"../../design/machines/PurchaseOrder.oracle.md\"\n" + parseEvidence,
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 1 || !strings.Contains(g.Errs[0], "Order.oracle.md: 1 of 1 stable ids appear in no test file (ORD-aaa111)") {
		t.Fatalf("Order must stay uncovered by the PurchaseOrder mention: %v", g.Errs)
	}
	if g.Counts["machines covered by conformance parse"] != 1 {
		t.Errorf("machines covered by conformance parse = %d, want 1 (PurchaseOrder): %+v",
			g.Counts["machines covered by conformance parse"], g.Counts)
	}
}

func TestCheckOracleCoverageMachinesWithoutOracles(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.machine.json": `{"id": "thing", "initial": "A", "states": {"A": {}}}`,
		"impl/thing_test.go":          "package thing\n",
	})
	g := CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "no committed *.oracle.md") {
		t.Fatalf("machines without committed oracles leave Gt nothing to cover: %v", g.Errs)
	}
}

// Once any oracle exists, a machine missing its own committed oracle is an
// ERROR: it would otherwise be invisible to the coverage scan.
func TestCheckOracleCoverageMachineMissingItsOracle(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Order.machine.json": `{"id": "order", "initial": "A", "states": {"A": {}}}`,
		"machines/Order.oracle.md":    covOracleMD,
		"machines/Task.machine.json":  `{"id": "task", "initial": "A", "states": {"A": {}}}`,
		"impl/order_test.go":          "package order\n\n// THIN-aaa111 and THIN-bbb222\n",
	})
	g := CheckOracleCoverage(design, impl)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "Task.machine.json: no committed oracle (Task.oracle.md)") ||
		!strings.Contains(joined, "machinery oracle") {
		t.Fatalf("the oracle-less machine must be named with the remedy: %v", g.Errs)
	}
	if strings.Contains(joined, "Order.machine.json") {
		t.Errorf("the machine with a committed oracle must not be flagged: %v", g.Errs)
	}
}

func TestCheckOracleCoverageNoMachines(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"impl/thing_test.go": "package thing\n",
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("a machine-less design has no transition-test obligation: %v", g.Errs)
	}
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "0 machines") {
		t.Errorf("checked line must show 0 machines explicitly: %v", g.checkedExtra)
	}
}

func TestCheckOracleCoverageFormalOracles(t *testing.T) {
	t.Run("covered by file-name literal", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"formal/Policy.oracle.md": covPolicyOracleMD,
			"impl/authz_test.go": "package authz\n\nconst oraclePath = " +
				"\"../../design/formal/Policy.oracle.md\"\n" + parseEvidence,
		})
		g := CheckOracleCoverage(design, impl)
		if len(g.Errs) != 0 {
			t.Fatalf("Gt not clean: %v", g.Errs)
		}
		if g.Counts["formal oracles covered"] != 1 {
			t.Errorf("formal oracles covered = %d, want 1: %+v", g.Counts["formal oracles covered"], g.Counts)
		}
	})
	t.Run("uncovered", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"formal/Policy.oracle.md": covPolicyOracleMD,
			"impl/authz_test.go":      "package authz\n",
		})
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "formal/Policy.oracle.md: 2 of 2 stable ids") {
			t.Fatalf("uncovered formal oracle must fail: %v", g.Errs)
		}
		if g.Counts["formal oracles covered"] != 0 {
			t.Errorf("formal oracles covered = %d, want 0: %+v", g.Counts["formal oracles covered"], g.Counts)
		}
	})
}

// The offender list caps at 10 ids with an "and N more" tail, mirroring how
// G4 caps its ratchet offender lists.
func TestCheckOracleCoverageCapsOffenderList(t *testing.T) {
	var rows strings.Builder
	rows.WriteString("## Transitions\n\n| test id | stable id | source |\n|---|---|---|\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&rows, "| T-BIG-%02d | BIG-%06d | A |\n", i, i)
	}
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Big.oracle.md": rows.String(),
		"impl/big_test.go":       "package big\n",
	})
	g := CheckOracleCoverage(design, impl)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "and 2 more") {
		t.Fatalf("want a capped list with 'and 2 more', got %v", g.Errs)
	}
	if strings.Contains(joined, "BIG-000011") {
		t.Fatalf("ids beyond the cap must be elided: %v", g.Errs)
	}
}

// Gt honors the contract's ignore globs the same way G4 does: an ignored
// test file is invisible to the coverage scan.
func TestCheckOracleCoverageHonorsContractIgnore(t *testing.T) {
	arch := "# Architecture\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\nignore:\n  - \"vendor/**\"\n```\n"
	design, impl := writeCovFixture(t, map[string]string{
		"ARCHITECTURE.md":           arch,
		"machines/Thing.oracle.md":  covOracleMD,
		"impl/app/other_test.go":    "package app\n",
		"impl/vendor/thing_test.go": "package thing\n\n// THIN-aaa111 THIN-bbb222\n",
	})
	g := CheckOracleCoverage(design, impl)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "2 of 2 stable ids appear in no test file") {
		t.Fatalf("an ignored test file must not count as coverage: %v", g.Errs)
	}
	if g.Counts["test files scanned"] != 1 {
		t.Errorf("test files scanned = %d, want 1 (the vendored file is invisible): %+v", g.Counts["test files scanned"], g.Counts)
	}
}

// *.test.mjs is a test file in an extension langExts never maps for import
// parsing; the walk must still collect it or the pattern is dead code.
func TestCheckOracleCoverageScansMjsTestFiles(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/thing.test.mjs":      "// THIN-aaa111 and THIN-bbb222\n",
	})
	g := CheckOracleCoverage(design, impl)
	if len(g.Errs) != 0 {
		t.Fatalf("a .test.mjs suite must satisfy Gt: %v", g.Errs)
	}
	if g.Counts["test files scanned"] != 1 {
		t.Errorf("test files scanned = %d, want 1: %+v", g.Counts["test files scanned"], g.Counts)
	}
}

// The shared classifier (one classifier, two gates: G4 skips what Gt scans)
// recognizes every documented test-file shape by path.
func TestIsTestFileShapes(t *testing.T) {
	cases := []struct {
		rel  string
		want bool
	}{
		{"a_test.go", true},
		{"pkg/deep/a_test.go", true},
		{"test_a.py", true},
		{"a_test.py", true},
		{"a.test.ts", true},
		{"a.test.tsx", true},
		{"a.test.js", true},
		{"a.test.jsx", true},
		{"a.test.mjs", true},
		{"a.test.cjs", true},
		{"a_test.exs", true},
		{"a_spec.rb", true},
		// a .rs file is test-only iff under tests/ or benches/: production
		// files with inline #[cfg(test)] modules are split, never skipped
		{"a_test.rs", false},
		{"tests/foo.rs", true},
		{"crate/tests/deep/foo.rs", true},
		{"benches/bench.rs", true},
		{"src/lib.rs", false},
		{"tests/helper.go", false}, // the tests/ dir rule is Rust-only
		{"a.go", false},
		{"a.mjs", false},
	}
	for _, tc := range cases {
		if got := isTestFile(tc.rel); got != tc.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tc.rel, got, tc.want)
		}
	}
}

// rustSplitTests carves a production .rs file into the production portion (what
// G4 scans) and the #[cfg(test)] item spans (what Gt scans); the two never mix.
func TestRustSplitTests(t *testing.T) {
	src := "use std::fmt;\n\npub fn f() {}\n\n#[cfg(test)]\nmod tests {\n    use super::*;\n    #[test]\n    fn t() { assert!(true); }\n}\n\npub fn g() {}\n"
	prod, tests := rustSplitTests(src)
	if strings.Contains(prod, "assert!(true)") || strings.Contains(prod, "use super::*") {
		t.Errorf("production portion still carries test text:\n%s", prod)
	}
	if !strings.Contains(prod, "use std::fmt;") || !strings.Contains(prod, "pub fn g() {}") {
		t.Errorf("production portion lost production text:\n%s", prod)
	}
	if len(tests) != 1 || !strings.Contains(tests[0], "assert!(true)") {
		t.Errorf("test spans wrong: %q", tests)
	}
	if strings.Count(prod, "\n") != strings.Count(src, "\n") {
		t.Error("masking must preserve the line structure")
	}
	// a braceless cfg(test) item ends at its semicolon
	prod2, tests2 := rustSplitTests("#[cfg(test)]\nuse crate::mocks::fake;\nuse crate::real;\n")
	if strings.Contains(prod2, "mocks") || !strings.Contains(prod2, "crate::real") {
		t.Errorf("braceless span wrong: prod=%q", prod2)
	}
	if len(tests2) != 1 || !strings.Contains(tests2[0], "mocks") {
		t.Errorf("braceless test span wrong: %q", tests2)
	}
	// no cfg(test): everything is production
	if prod3, tests3 := rustSplitTests("pub fn f() {}\n"); prod3 != "pub fn f() {}\n" || tests3 != nil {
		t.Errorf("split without cfg(test) must be identity: %q %q", prod3, tests3)
	}
}
