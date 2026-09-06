package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// goldenBin builds this worktree's real CLI; runBinWithEnv starts that binary.
// All fixture inputs and command homes are private temporary directories.
func TestOracleCoverageCLIRealTrees(t *testing.T) {
	const oracle = "| test id | stable id | source | trigger | guard | target | actions |\n|---|---|---|---|---|---|---|\n| T-THIN-01 | THIN-aaa111 | A | on:go | - | B | - |\n| T-THIN-02 | THIN-bbb222 | B | on:stop | - | A | - |\n"
	const literal = "package fixture\nimport \"testing\"\nfunc TestRows(t *testing.T) { for _, id := range []string{\"THIN-aaa111\", \"THIN-bbb222\"} { if id == \"\" { t.Fatal(id) } } }\n"
	const parser = `package fixture
import ("os"; "strings"; "testing")
func TestOracle(t *testing.T) {
 data, err := os.ReadFile("../design/machines/Thing.oracle.md")
 if err != nil { t.Fatal(err) }
 rows := 0
 for _, line := range strings.Split(string(data), "\n") {
  cells := strings.Split(line, "|")
  if len(cells) < 8 || !strings.HasPrefix(strings.TrimSpace(cells[1]), "T-") { continue }
  rows++
  got := "invalid transition"
  if strings.TrimSpace(cells[3]) == "A" && strings.TrimSpace(cells[4]) == "on:go" { got = "B" }
  if strings.TrimSpace(cells[3]) == "B" && strings.TrimSpace(cells[4]) == "on:stop" { got = "A" }
  if got != strings.TrimSpace(cells[6]) { t.Fatal("oracle transition mismatch") }
 }
 if rows == 0 { t.Fatal("no oracle rows checked") }
}
`
	cases := []struct {
		name     string
		files    map[string]string
		wantCode int
		want     string
	}{
		{"literal_control", map[string]string{"rows_test.go": literal}, 0, "2 ids covered by literal"},
		{"parser_control", map[string]string{"rows_test.go": parser}, 0, "1 machines covered by conformance parse"},
		{"quoted_comment", map[string]string{"rows_test.go": "package fixture\n// TODO parse \"Thing.oracle.md\" using \"|\"\n"}, 1, "2 of 2 stable ids"},
		{"unused_constants", map[string]string{"rows_test.go": "package fixture\nconst oracle = \"Thing.oracle.md\"\nconst delimiter = \"|\"\n"}, 1, "2 of 2 stable ids"},
		{"disabled_go", map[string]string{"rows_test.go": "//go:build ignore\n\n" + literal}, 1, ""},
		{"elixir_comment", map[string]string{"rows_test.exs": "# THIN-aaa111 THIN-bbb222\n"}, 1, "2 of 2 stable ids"},
		{"zero_tests", map[string]string{"rows.go": "package fixture\n"}, 1, "no test files"},
		{"mixed_disabled", map[string]string{"active_test.go": strings.Replace(literal, ", \"THIN-bbb222\"", "", 1), "disabled_test.go": "//go:build ignore\n\n" + literal}, 1, "1 of 2 stable ids"},
		{"malformed_reference", map[string]string{"rows_test.go": strings.Replace(parser, "Thing.oracle.md", "Thing.oracle.md.bak", 1)}, 1, "2 of 2 stable ids"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
			writeText(t, filepath.Join(design, "machines", "Thing.machine.json"), `{"id":"thing","initial":"A","states":{"A":{},"B":{}}}`)
			writeText(t, filepath.Join(design, "machines", "Thing.oracle.md"), oracle)
			for name, source := range tc.files {
				writeText(t, filepath.Join(impl, name), source)
			}
			out, stderr, code := runBinWithEnv(t, []string{"HOME=" + t.TempDir(), "MACHINERY_CONFIG_DIR=" + privateTestConfigDir(t)}, "check", design, "--impl", impl, "--gate", "gt")
			if !strings.Contains(out, "Gt-tests") || stderr != "" {
				t.Fatalf("CLI did not reach real Gt: code=%d out=%s stderr=%s", code, out, stderr)
			}
			if code != tc.wantCode || !strings.Contains(out, tc.want) {
				t.Fatalf("wrong coverage decision: exit=%d want=%d output=%s", code, tc.wantCode, out)
			}
		})
	}
}

func TestOracleCoverageCLILabelsDiscoveryOnly(t *testing.T) {
	root := t.TempDir()
	design, impl := filepath.Join(root, "design"), filepath.Join(root, "impl")
	writeText(t, filepath.Join(design, "machines", "Thing.machine.json"), `{"id":"thing","initial":"A","states":{"A":{}}}`)
	writeText(t, filepath.Join(design, "machines", "Thing.oracle.md"), "| test id | stable id |\n|---|---|\n| T-THIN-01 | THIN-aaa111 |\n")
	writeText(t, filepath.Join(impl, "rows_test.go"), "package fixture\nimport \"testing\"\nfunc TestRows(t *testing.T) { t.Fatal(\"THIN-aaa111: this assertion was not run by the scanner\") }\n")
	out, stderr, code := runBinWithEnv(t, []string{"HOME=" + t.TempDir(), "MACHINERY_CONFIG_DIR=" + privateTestConfigDir(t)}, "check", design, "--impl", impl, "--gate", "gt")
	if code != 0 || stderr != "" {
		t.Fatalf("literal discovery control failed: code=%d out=%s stderr=%s", code, out, stderr)
	}
	text := strings.ToLower(out)
	if !strings.Contains(text, "discovery") || !(strings.Contains(text, "not executed") || strings.Contains(text, "does not execute") || strings.Contains(text, "not execution")) {
		t.Fatalf("CLI conflates discovery with assertion execution: %s", out)
	}
}
