package gates

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOracleCoverageSkippedBindings(t *testing.T) {
	tests := []struct{ name, path, source, config string }{
		{"module skip", "test/rows_test.exs", `defmodule RowsTest do
 use ExUnit.Case
 @moduletag skip: "image unavailable"
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, ""},
		{"module pending", "test/rows_test.exs", `defmodule RowsTest do
 @moduletag :pending
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, ""},
		{"test skip", "test/rows_test.exs", `defmodule RowsTest do
 @tag :skip
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, ""},
		{"excluded atom", "test/rows_test.exs", `defmodule RowsTest do
 @moduletag :reader_image
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, `ExUnit.start(exclude: [:reader_image])`},
		{"excluded value", "test/rows_test.exs", `defmodule RowsTest do
 @moduletag reader_image: true
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, `ExUnit.configure(exclude: [reader_image: true])`},
		{"literal gate", "test/rows_test.exs", `defmodule RowsTest do
 @moduletag skip: not false
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`, ""},
		{"go skip", "rows_test.go", `package rows
import "testing"
func TestRows(check *testing.T) { check.Skip("unavailable"); _ = []string{"THIN-aaa111", "THIN-bbb222"} }`, ""},
		{"pytest decorator", "test_rows.py", `import pytest
@pytest.mark.skip(reason="unavailable")
def test_rows():
 assert ["THIN-aaa111", "THIN-bbb222"]
`, ""},
		{"pytest module", "test_rows.py", `import pytest
pytestmark = pytest.mark.skip(reason="unavailable")
def test_rows():
 assert ["THIN-aaa111", "THIN-bbb222"]
`, ""},
		{"node skip", "rows.test.js", `test.skip("rows", () => { expect(["THIN-aaa111", "THIN-bbb222"]).toBeTruthy(); });`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			design, impl := writeCovFixture(t, map[string]string{"machines/Thing.oracle.md": covOracleMD, "impl/" + tt.path: tt.source, "impl/test/test_helper.exs": tt.config})
			g := CheckOracleCoverage(design, impl)
			errs := strings.Join(g.Errs, "\n")
			for _, id := range []string{"THIN-aaa111", "THIN-bbb222"} {
				if !strings.Contains(errs, "UNRUN "+id) || !strings.Contains(errs, tt.path) {
					t.Errorf("skipped binding must be UNRUN with path: %s", errs)
				}
			}
			if g.Counts["ids covered by literal"] != 0 {
				t.Errorf("skipped suite credited ids: %v", g.Counts)
			}
			bindings, _, _ := OracleBindings(design, impl, map[string][]string{"Thing.oracle.md": {"THIN-aaa111", "THIN-bbb222"}})
			if len(bindings) != 0 {
				t.Errorf("skipped suite exported bindings: %v", bindings)
			}
		})
	}
}

func TestOracleCoverageSkipScope(t *testing.T) {
	design, impl := writeCovFixture(t, map[string]string{
		"machines/Thing.oracle.md": covOracleMD,
		"impl/test/rows_test.exs": `defmodule SkippedTest do
 @moduletag skip: true
 test "skipped" do
  assert "THIN-aaa111"
 end
end
defmodule ActiveTest do
 @tag skip: false
 test "active" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`,
	})
	if g := CheckOracleCoverage(design, impl); len(g.Errs) != 0 {
		t.Fatalf("active binding lost credit: %v", g.Errs)
	}
}

func writeRunRecord(t *testing.T, design, impl, suite string, sources []string) {
	t.Helper()
	hash := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(impl, rel))
		if err != nil {
			t.Fatal(err)
		}
		return fmt.Sprintf("%x", sha256.Sum256(b))
	}
	named := map[string]string{}
	for _, source := range sources {
		named[source] = hash(source)
	}
	rows := []map[string]any{{"type": "out-of-band", "suite": suite, "sha256": hash(suite), "sources": named, "result": "passed"}}
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	writeSuiteFile(t, filepath.Join(design, "assurance", "test-runs.json"), string(b))
}

func TestOracleCoverageRecordedRun(t *testing.T) {
	for _, mutation := range []string{"none", "suite", "source", "missing source", "invalid record"} {
		t.Run(mutation, func(t *testing.T) {
			suite := "test/rows_test.exs"
			design, impl := writeCovFixture(t, map[string]string{
				"machines/Thing.oracle.md": covOracleMD,
				"impl/" + suite: `defmodule RowsTest do
 @moduletag skip: true
 test "rows" do
  assert ["THIN-aaa111", "THIN-bbb222"] != []
 end
end`,
				"impl/lib/reader.ex": "defmodule Reader do\nend\n",
			})
			writeRunRecord(t, design, impl, suite, []string{"lib/reader.ex"})
			switch mutation {
			case "suite":
				writeSuiteFile(t, filepath.Join(impl, suite), "# changed\n"+mustRead(t, filepath.Join(impl, suite)))
			case "source":
				writeSuiteFile(t, filepath.Join(impl, "lib/reader.ex"), "# changed\n")
			case "missing source":
				if err := os.Remove(filepath.Join(impl, "lib/reader.ex")); err != nil {
					t.Fatal(err)
				}
			case "invalid record":
				writeSuiteFile(t, filepath.Join(design, "assurance/test-runs.json"), `[{"type":"out-of-band","suite":"test/rows_test.exs","result":"failed"}]`)
			}
			g := CheckOracleCoverage(design, impl)
			if mutation == "none" {
				if len(g.Errs) != 0 || g.Counts["ids covered by literal"] != 2 {
					t.Fatalf("matching record did not credit suite: %v %v", g.Errs, g.Counts)
				}
				bindings, _, errs := OracleBindings(design, impl, map[string][]string{"Thing.oracle.md": {"THIN-aaa111"}})
				if len(bindings) != 1 || len(errs) != 0 {
					t.Fatalf("recorded bindings: %v %v", bindings, errs)
				}
			} else if errs := strings.Join(g.Errs, "\n"); !strings.Contains(errs, "test-runs.json") || !strings.Contains(errs, "UNRUN") {
				t.Fatalf("stale/invalid record must block and withhold credit: %s", errs)
			}
		})
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
