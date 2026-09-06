package gates

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These source fixtures test discovery. Only TestOracleCoverageParserRuntimeControl
// separately executes a fixture; CheckOracleCoverage itself must never claim that.
func covActiveParser(pkg, path string) string {
	return fmt.Sprintf(`package %s
import ("os"; "strings"; "testing")
func TestOracle(t *testing.T) {
 data, err := os.ReadFile(%q)
 if err != nil { t.Fatal(err) }
 rows := 0
 for _, line := range strings.Split(string(data), "\n") {
  cells := strings.Split(strings.Trim(line, "|"), "|")
  if len(cells) < 2 || !strings.HasPrefix(strings.TrimSpace(cells[0]), "T-") && !strings.HasPrefix(strings.TrimSpace(cells[0]), "A-") { continue }
  rows++
  if strings.TrimSpace(cells[1]) == "" { t.Fatal("oracle row has no stable id") }
  for i := range cells { cells[i] = strings.TrimSpace(cells[i]) }
  switch {
  case len(cells) >= 7:
   got := "invalid transition"
   if cells[2] == "A" && cells[3] == "on:go" { got = "B" }
   if cells[2] == "B" && cells[3] == "on:stop" { got = "A" }
   if got != cells[5] { t.Fatalf("oracle transition mismatch: got %%s want %%s", got, cells[5]) }
  case len(cells) == 5:
   got := "deny"
   if cells[2] == "admin any" { got = "allow" }
   if got != cells[3] { t.Fatalf("oracle policy mismatch: got %%s want %%s", got, cells[3]) }
  default:
   if cells[2] != "A" { t.Fatal("oracle boundary fixture source mismatch") }
  }
  t.Log("checked oracle row")
 }
 if rows == 0 { t.Fatal("no oracle rows checked") }
}
`, pkg, path)
}

func covLiteralTest(ids string) string {
	return "package fixture\nimport \"testing\"\nfunc TestRows(t *testing.T) { for _, id := range []string{" + ids + "} { if id == \"\" { t.Fatal(id) } } }\n"
}

func TestOracleCoverageRejectsNonExecutableEvidence(t *testing.T) {
	cases := []struct{ name, file, source string }{
		{"quoted_go_comment", "rows_test.go", "package fixture\n// TODO parse \"Thing.oracle.md\" using \"|\"\n"},
		{"go_block_comment", "rows_test.go", "package fixture\n/* parse \"Thing.oracle.md\" with \"|\" */\n"},
		{"unused_go_constants", "rows_test.go", "package fixture\nconst oracle = \"Thing.oracle.md\"\nconst delimiter = \"|\"\n"},
		{"uncalled_go_parser", "rows_test.go", strings.Replace(covActiveParser("fixture", "Thing.oracle.md"), "func TestOracle", "func parseOracle", 1)},
		{"go_lowercase_test_helper", "rows_test.go", "package fixture\nfunc Testhelper() { _ = []string{\"THIN-aaa111\", \"THIN-bbb222\"} }\n"},
		{"go_invalid_test_signature", "rows_test.go", "package fixture\nfunc TestRows() { _ = []string{\"THIN-aaa111\", \"THIN-bbb222\"} }\n"},
		{"disabled_go_literal", "rows_test.go", "//go:build ignore\n\n" + covLiteralTest(`"THIN-aaa111", "THIN-bbb222"`)},
		{"disabled_go_parser", "rows_test.go", "//go:build ignore\n\n" + covActiveParser("fixture", "Thing.oracle.md")},
		{"legacy_disabled_go_literal", "rows_test.go", "// +build ignore\n\n" + covLiteralTest(`"THIN-aaa111", "THIN-bbb222"`)},
		{"elixir_hash_ids", "rows_test.exs", "# THIN-aaa111 THIN-bbb222\n"},
		{"elixir_hash_parser", "rows_test.exs", "# parse \"Thing.oracle.md\" with \"|\"\n"},
		{"python_module_docstring", "test_rows.py", "\"\"\"THIN-aaa111 THIN-bbb222\"\"\"\n"},
		{"python_test_docstring", "test_rows.py", "def test_rows():\n    '''THIN-aaa111 THIN-bbb222'''\n    assert True\n"},
		{"python_parser_docstring", "test_rows.py", "\"\"\"parse 'Thing.oracle.md' with '|'\"\"\"\n"},
		{"js_unused_declarations", "rows.test.js", "const oracle = 'Thing.oracle.md'; const delimiter = '|';\n"},
		{"python_unused_declarations", "test_rows.py", "oracle = 'Thing.oracle.md'\ndelimiter = '|'\n"},
		{"ruby_unused_declarations", "rows_spec.rb", "ORACLE = 'Thing.oracle.md'\nDELIMITER = '|'\n"},
		{"elixir_unused_declarations", "rows_test.exs", "defmodule Rows do\n @oracle \"Thing.oracle.md\"\n @delimiter \"|\"\nend\n"},
		{"go_unrelated_split", "rows_test.go", "package fixture\nimport (\"strings\"; \"testing\")\nfunc TestRows(t *testing.T) { t.Log(\"Thing.oracle.md\"); if len(strings.Split(\"a|b\", \"|\")) != 2 { t.Fatal(\"split\") } }\n"},
		{"go_read_without_row_checks", "rows_test.go", "package fixture\nimport (\"os\"; \"strings\"; \"testing\")\nfunc TestRows(t *testing.T) { data, err := os.ReadFile(\"Thing.oracle.md\"); if err != nil { t.Fatal(err) }; t.Log(strings.Split(string(data), \"|\")) }\n"},
		{"elixir_moduledoc_ids", "rows_test.exs", "defmodule RowsTest do\n  @moduledoc \"\"\"\n  THIN-aaa111 THIN-bbb222\n  \"\"\"\nend\n"},
		{"elixir_doc_ids", "rows_test.exs", "defmodule RowsTest do\n  @doc \"\"\"\n  THIN-aaa111 THIN-bbb222\n  \"\"\"\n  def helper, do: :ok\nend\n"},
		{"elixir_moduledoc_parser", "rows_test.exs", "defmodule RowsTest do\n  @moduledoc \"\"\"\n  parse \"Thing.oracle.md\" using \"|\"\n  \"\"\"\nend\n"},
		{"elixir_doc_parser", "rows_test.exs", "defmodule RowsTest do\n  @doc \"\"\"\n  parse \"Thing.oracle.md\" using \"|\"\n  \"\"\"\n  def helper, do: :ok\nend\n"},
		{"ruby_block_comment_ids", "rows_spec.rb", "=begin\nTHIN-aaa111 THIN-bbb222\n=end\n"},
		{"ruby_block_comment_parser", "rows_spec.rb", "=begin\nparse \"Thing.oracle.md\" using \"|\"\n=end\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/" + tc.file: tc.source})
			g := CheckOracleCoverage(design, impl)
			if len(g.Errs) == 0 || g.Counts["ids covered by literal"] != 0 || g.Counts["machines covered by conformance parse"] != 0 {
				t.Fatalf("non-executable evidence established coverage: errs=%v counts=%v", g.Errs, g.Counts)
			}
			if !strings.Contains(strings.Join(g.Errs, " "), "THIN-aaa111") && !strings.Contains(strings.Join(g.Errs, " "), "no test files") {
				t.Fatalf("missing coverage diagnosis: %v", g.Errs)
			}
		})
	}
}

func TestOracleCoverageMixedActiveAndDisabled(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/active_test.go":      covLiteralTest(`"THIN-aaa111"`),
		"impl/disabled_test.go":    "//go:build ignore\n\n" + covLiteralTest(`"THIN-bbb222"`),
	})
	g := CheckOracleCoverage(design, impl)
	if g.Counts["ids covered by literal"] != 1 || g.Counts["machines covered by conformance parse"] != 0 || !strings.Contains(strings.Join(g.Errs, " "), "1 of 2 stable ids appear in no test file (THIN-bbb222)") {
		t.Fatalf("disabled row contaminated active coverage: errs=%v counts=%v", g.Errs, g.Counts)
	}
}

func TestOracleCoverageLiteralLanguageControls(t *testing.T) {
	js := "test('oracle rows', () => { for (const id of ['THIN-aaa111', 'THIN-bbb222']) { expect(id.length).toBeGreaterThan(0); } });\n"
	cases := map[string]string{
		"rows_test.go": covLiteralTest(`"THIN-aaa111", "THIN-bbb222"`),
		"test_rows.py": "def test_rows():\n    for row in ['THIN-aaa111', 'THIN-bbb222']:\n        assert row\n",
		"rows.test.ts": js, "rows.test.tsx": js, "rows.test.js": js, "rows.test.jsx": js, "rows.test.mjs": js, "rows.test.cjs": js,
		"rows_spec.rb":  "describe 'oracle rows' do\n it('has ids') { ['THIN-aaa111', 'THIN-bbb222'].each { |id| expect(id).not_to be_empty } }\nend\n",
		"rows_test.exs": "defmodule RowsTest do\n use ExUnit.Case\n test \"oracle rows\" do\n for id <- [\"THIN-aaa111\", \"THIN-bbb222\"], do: assert(String.length(id) > 0)\n end\nend\n",
		"tests/rows.rs": "#[test]\nfn rows() { for id in [\"THIN-aaa111\", \"THIN-bbb222\"] { assert!(!id.is_empty()); } }\n",
	}
	for file, source := range cases {
		t.Run(file, func(t *testing.T) {
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/" + file: source})
			g := CheckOracleCoverage(design, impl)
			if len(g.Errs) != 0 || g.Counts["ids covered by literal"] != 2 || g.Counts["machines covered by conformance parse"] != 0 {
				t.Fatalf("active literal discovery lost: errs=%v counts=%v", g.Errs, g.Counts)
			}
		})
	}
}

func TestOracleCoverageActiveParserLanguageControls(t *testing.T) {
	js := "const fs = require('node:fs');\ntest('oracle rows', () => { let rows = 0; for (const line of fs.readFileSync('Thing.oracle.md', 'utf8').split('\\n')) { const cells = line.split('|').map(s => s.trim()); if (!cells[1]?.startsWith('T-')) continue; rows++; const got = cells[3] === 'A' && cells[4] === 'on:go' ? 'B' : cells[3] === 'B' && cells[4] === 'on:stop' ? 'A' : 'invalid'; expect(got).toBe(cells[6]); } expect(rows).toBeGreaterThan(0); });\n"
	cases := map[string]string{
		"rows_test.go": covActiveParser("fixture", "Thing.oracle.md"),
		"test_rows.py": "def test_rows():\n    rows = 0\n    with open('Thing.oracle.md') as oracle:\n        for line in oracle:\n            cells = [cell.strip() for cell in line.split('|')]\n            if len(cells) > 2 and cells[1].startswith('T-'):\n                rows += 1\n                got = {('A', 'on:go'): 'B', ('B', 'on:stop'): 'A'}.get((cells[3], cells[4]))\n                assert got == cells[6]\n    assert rows > 0\n",
		"rows.test.js": js, "rows.test.ts": js,
		"rows_spec.rb":  "describe 'oracle' do\n it 'checks rows' do\n rows = 0\n File.readlines('Thing.oracle.md').each do |line|\n cells = line.split('|').map(&:strip)\n next unless cells[1]&.start_with?('T-')\n rows += 1\n got = {['A', 'on:go'] => 'B', ['B', 'on:stop'] => 'A'}[[cells[3], cells[4]]]\n expect(got).to eq(cells[6])\n end\n expect(rows).to be > 0\n end\nend\n",
		"rows_test.exs": "defmodule RowsTest do\n use ExUnit.Case\n test \"oracle rows\" do\n rows = File.read!(\"Thing.oracle.md\") |> String.split(\"\\n\") |> Enum.filter(&String.starts_with?(&1, \"| T-\"))\n assert length(rows) > 0\n for row <- rows do\n cells = String.split(row, \"|\") |> Enum.map(&String.trim/1)\n got = Map.get(%{{\"A\", \"on:go\"} => \"B\", {\"B\", \"on:stop\"} => \"A\"}, {Enum.at(cells, 3), Enum.at(cells, 4)})\n assert got == Enum.at(cells, 6)\n end\n end\nend\n",
		"tests/rows.rs": "#[test]\nfn oracle_rows() { let data = std::fs::read_to_string(\"Thing.oracle.md\").unwrap(); let mut rows = 0; for line in data.lines() { let cells: Vec<_> = line.split('|').map(str::trim).collect(); if cells.len() > 2 && cells[1].starts_with(\"T-\") { rows += 1; let got = match (cells[3], cells[4]) { (\"A\", \"on:go\") => \"B\", (\"B\", \"on:stop\") => \"A\", _ => \"invalid\" }; assert_eq!(got, cells[6]); } } assert!(rows > 0); }\n",
	}
	for file, source := range cases {
		t.Run(file, func(t *testing.T) {
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/" + file: source})
			g := CheckOracleCoverage(design, impl)
			if len(g.Errs) != 0 || g.Counts["machines covered by conformance parse"] != 1 || g.Counts["ids covered by literal"] != 0 {
				t.Fatalf("active parser discovery lost: errs=%v counts=%v", g.Errs, g.Counts)
			}
		})
	}
}

func TestOracleCoverageParserRuntimeControl(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD})
	path := filepath.Join(design, "machines", "Thing.oracle.md")
	writeSuiteFile(t, filepath.Join(impl, "rows_test.go"), covActiveParser("fixture", path))
	if g := CheckOracleCoverage(design, impl); len(g.Errs) != 0 || g.Counts["machines covered by conformance parse"] != 1 {
		t.Fatalf("runtime-backed parser was not discovered: %+v", g)
	}
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprintf("corrupt_%t", corrupt), func(t *testing.T) {
			if corrupt {
				writeSuiteFile(t, path, strings.Replace(covOracleMD, "| on:go | - | B |", "| on:go | - | A |", 1))
			}
			ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "-v", "-timeout=30s", "rows_test.go")
			cmd.Dir = impl
			cmd.Env = append(os.Environ(), "GOWORK=off")
			out, err := cmd.CombinedOutput()
			t.Logf("native fixture exit=%v\n%s", err, out)
			if ctx.Err() != nil {
				t.Fatalf("fixture process timed out: %s", out)
			}
			if !corrupt && (err != nil || bytes.Count(out, []byte("checked oracle row")) != 2) {
				t.Fatalf("parser control did not check both rows: %v\n%s", err, out)
			}
			if corrupt && (err == nil || !bytes.Contains(out, []byte("oracle transition mismatch: got B want A"))) {
				t.Fatalf("parser control failed to reach its row assertion: %v\n%s", err, out)
			}
		})
	}
}

func TestOracleCoverageMalformedReferencesStayUncovered(t *testing.T) {
	for _, name := range []string{"Thing.oracle.md.bak", "NotThing.oracle.md", "purchase-Thing.oracle.md"} {
		t.Run(name, func(t *testing.T) {
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/rows_test.go": covActiveParser("fixture", name)})
			g := CheckOracleCoverage(design, impl)
			if len(g.Errs) != 1 || !strings.Contains(g.Errs[0], "2 of 2 stable ids") || g.Counts["machines covered by conformance parse"] != 0 {
				t.Fatalf("malformed reference established coverage: errs=%v counts=%v", g.Errs, g.Counts)
			}
		})
	}
}

func TestOracleCoverageOutputLabelsDiscoveryOnly(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/rows_test.go": covLiteralTest(`"THIN-aaa111", "THIN-bbb222"`)})
	var output bytes.Buffer
	if code := CheckOracleCoverage(design, impl).Emit(&output); code != 0 {
		t.Fatalf("positive discovery failed: %s", &output)
	}
	text := strings.ToLower(output.String())
	if !strings.Contains(text, "discovery") || !(strings.Contains(text, "not executed") || strings.Contains(text, "does not execute") || strings.Contains(text, "not execution")) {
		t.Fatalf("static gate output does not distinguish discovery from execution: %s", &output)
	}
}
