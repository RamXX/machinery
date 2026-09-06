package gates

import (
	"strings"
	"testing"
)

func TestOracleParserEvidenceStaysInsideTest(t *testing.T) {
	js := `let rows = 0; for (const line of fs.readFileSync('Thing.oracle.md', 'utf8').split('\n')) { const cells = line.split('|').map(s => s.trim()); if (!cells[1]?.startsWith('T-')) continue; rows++; const got = cells[3] === 'A' && cells[4] === 'on:go' ? 'B' : cells[3] === 'B' && cells[4] === 'on:stop' ? 'A' : 'invalid'; expect(got).toBe(cells[6]); } expect(rows).toBeGreaterThan(0);`
	python := "    rows = 0\n    with open('Thing.oracle.md') as oracle:\n        for line in oracle:\n            cells = [cell.strip() for cell in line.split('|')]\n            if len(cells) > 2 and cells[1].startswith('T-'):\n                rows += 1\n                got = {('A', 'on:go'): 'B', ('B', 'on:stop'): 'A'}.get((cells[3], cells[4]))\n                assert got == cells[6]\n    assert rows > 0\n"
	ruby := "rows = 0\nFile.readlines('Thing.oracle.md').each do |line|\ncells = line.split('|').map(&:strip)\nnext unless cells[1]&.start_with?('T-')\nrows += 1\ngot = {['A', 'on:go'] => 'B', ['B', 'on:stop'] => 'A'}[[cells[3], cells[4]]]\nexpect(got).to eq(cells[6])\nend\nexpect(rows).to be > 0\n"
	elixir := "rows = File.read!(\"Thing.oracle.md\") |> String.split(\"\\n\") |> Enum.filter(&String.starts_with?(&1, \"| T-\"))\nassert length(rows) > 0\nfor row <- rows do\ncells = String.split(row, \"|\") |> Enum.map(&String.trim/1)\ngot = Map.get(%{{\"A\", \"on:go\"} => \"B\", {\"B\", \"on:stop\"} => \"A\"}, {Enum.at(cells, 3), Enum.at(cells, 4)})\nassert got == Enum.at(cells, 6)\nend\n"
	rust := `let data = std::fs::read_to_string("Thing.oracle.md").unwrap(); let mut rows = 0; for line in data.lines() { let cells: Vec<_> = line.split('|').map(str::trim).collect(); if cells.len() > 2 && cells[1].starts_with("T-") { rows += 1; let got = match (cells[3], cells[4]) { ("A", "on:go") => "B", ("B", "on:stop") => "A", _ => "invalid" }; assert_eq!(got, cells[6]); } } assert!(rows > 0);`
	cases := []struct{ file, prelude, active, helper, body, end, helperEnd string }{
		{"rows.test.js", "const fs = require('node:fs');\n", "test('rows', () => {", "function helper() {", js, "});\n", "}\n"},
		{"rows.test.ts", "import * as fs from 'node:fs';\n", "test('rows', () => {", "function helper() {", js, "});\n", "}\n"},
		{"test_rows.py", "", "def test_rows():\n", "def helper():\n", python, "\n", "\n"},
		{"rows_spec.rb", "", "it 'rows' do\n", "def helper\n", ruby, "end\n", "end\n"},
		{"rows_test.exs", "use ExUnit.Case\n", "test \"rows\" do\n", "def helper do\n", elixir, "end\n", "end\n"},
		{"tests/rows.rs", "", "#[test]\nfn rows() {", "fn helper() {", rust, "}\n", "}\n"},
	}
	for _, tc := range cases {
		for _, mode := range []string{"active", "helper_after_test", "source_after_test"} {
			t.Run(tc.file+"/"+mode, func(t *testing.T) {
				source := tc.prelude + tc.active + tc.body + tc.end
				if mode != "active" {
					empty := ""
					if strings.HasSuffix(tc.file, ".py") {
						empty = "    pass\n"
					}
					source = tc.prelude + tc.active + empty + tc.end
					if mode == "helper_after_test" {
						source += tc.helper + tc.body + tc.helperEnd
					} else {
						body := tc.body
						if strings.HasSuffix(tc.file, ".py") {
							body = strings.TrimPrefix(strings.ReplaceAll(body, "\n    ", "\n"), "    ")
						}
						source += body
					}
				}
				design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/" + tc.file: source})
				g := CheckOracleCoverage(design, impl)
				if mode == "active" {
					if len(g.Errs) != 0 || g.Counts["machines covered by conformance parse"] != 1 {
						t.Fatalf("genuine parser lost: %+v", g)
					}
				} else if len(g.Errs) == 0 || g.Counts["machines covered by conformance parse"] != 0 {
					t.Fatalf("source outside test credited: %+v", g)
				}
			})
		}
	}
}

func TestOracleHelperDiscoveryRequiresConnectedRows(t *testing.T) {
	const loader = `package fixture
import ("bufio"; "os"; "path/filepath"; "strings"; "testing")
const oraclePath = "Thing.oracle.md"
type row struct { source, trigger, target string }
func loadRows(t *testing.T) []row {
 f, err := os.Open(filepath.FromSlash(oraclePath)); if err != nil { t.Fatal(err) }; defer f.Close()
 var rows []row
 sc := bufio.NewScanner(f)
 for sc.Scan() {
  line := sc.Text(); if !strings.HasPrefix(line, "| T-") { continue }
  cells := strings.Split(strings.Trim(line, "|"), "|")
  if len(cells) < 7 { t.Fatal("malformed row") }
  rows = append(rows, row{source: strings.TrimSpace(cells[2]), trigger: strings.TrimSpace(cells[3]), target: strings.TrimSpace(cells[5])})
 }
 if sc.Err() != nil || len(rows) == 0 { t.Fatal("no readable oracle rows") }
 return rows
}
`
	const check = `t.Run("row", func(t *testing.T) { got := "invalid"; if row.source == "A" && row.trigger == "on:go" { got = "B" }; if row.source == "B" && row.trigger == "on:stop" { got = "A" }; if got != row.target { t.Errorf("transition mismatch") } })`
	const direct = `for _, row := range loadRows(t) { ` + check + ` }`
	const assigned = `rows := loadRows(t); for _, row := range rows { ` + check + ` }`
	cases := []struct {
		name, helper, body string
		covered            bool
	}{
		{"direct_range", loader, direct, true},
		{"assigned_range", loader, assigned, true},
		{"uncalled_loader", loader, `t.Log("nothing loaded")`, false},
		{"unused_rows", loader, `rows := loadRows(t); _ = rows; if false { t.Fatal("unrelated") }`, false},
		{"constant_assertion", loader, `for _, row := range loadRows(t) { _ = row; if false { t.Fatal("unrelated") } }`, false},
		{"wrong_oracle", strings.Replace(loader, `"Thing.oracle.md"`, `"Other.oracle.md"`, 1), direct, false},
		{"constant_target_field", strings.Replace(loader, "target: strings.TrimSpace(cells[5])", `target: "B"`, 1), direct, false},
		{"uncalled_assertion_closure", loader, `for _, row := range loadRows(t) { check := func() { if row.target != "B" { t.Fatal("unreached") } }; _ = check }`, false},
		{"recursive_loader", strings.Replace(loader, "return rows", "return loadRows(t)", 1), direct, false},
		{"ambiguous_return", strings.Replace(loader, "return rows", "if len(rows) > 3 { return nil }; return rows", 1), direct, false},
		{"shadowed_path", strings.Replace(loader, "f, err :=", `oraclePath := "Other.oracle.md"; _ = oraclePath; f, err :=`, 1), direct, false},
		{"reassigned_path", strings.Replace(strings.Replace(loader, "const oraclePath", "var oraclePath", 1), "f, err :=", `oraclePath = "Other.oracle.md"; f, err :=`, 1), direct, false},
		{"mutable_local_path", strings.Replace(strings.Replace(loader, "f, err :=", `p := "Thing.oracle.md"; f, err :=`, 1), "FromSlash(oraclePath)", "FromSlash(p)", 1), direct, false},
		{"call_budget", loader, `rows := loadRows(t); ` + strings.Repeat(`rows = loadRows(t); `, 128) + `for _, row := range rows { ` + check + ` }`, false},
		{"unrelated_receiver", loader + "type logger struct{}\nfunc (logger) Errorf(string, ...any) {}\n", `other := logger{}; for _, row := range loadRows(t) { if row.target != "B" { other.Errorf("decoy") } }`, false},
	}
	// Exhaustion cannot convert a recursive/prolonged call graph into credit.
	deep := loader
	for i := 0; i < 9; i++ {
		deep += "func step" + string(rune('A'+i)) + "(t *testing.T) []row { return step" + string(rune('B'+i)) + "(t) }\n"
	}
	deep += "func stepJ(t *testing.T) []row { return loadRows(t) }\n"
	cases = append(cases, struct {
		name, helper, body string
		covered            bool
	}{"depth_bound", deep, strings.Replace(direct, "loadRows(t)", "stepA(t)", 1), false})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.helper + "\nfunc TestRows(t *testing.T) { " + tc.body + " }\n"
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/rows_test.go": source})
			g := CheckOracleCoverage(design, impl)
			if tc.covered {
				if len(g.Errs) != 0 || g.Counts["machines covered by conformance parse"] != 1 {
					t.Fatalf("connected helper not discovered: %+v", g)
				}
			} else if len(g.Errs) == 0 || g.Counts["machines covered by conformance parse"] != 0 || g.Counts["ids covered by literal"] != 0 {
				t.Fatalf("disconnected helper credited: %+v", g)
			}
		})
	}
}
