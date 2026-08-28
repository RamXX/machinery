package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func clauseDesign(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	oracle := "| T-DEAL-01 | DEAL-abc123 | a | b | c | d | - |\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.oracle.md"), []byte(oracle), 0644); err != nil {
		t.Fatal(err)
	}
	matrix := "| name | kind | sig | contract | maps | test | fixture |\n|---|---|---|---|---|---|---|\n" +
		"| `guardCloseEvidenced` | guard | s | true iff an artifact ops holds exists CLAUSES{resolved-task, applied-record} RETIRED{sop-coverage} | inv `x-y` | unit | f |\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(matrix), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestClauseRetiredSurvivorWarns(t *testing.T) {
	d := clauseDesign(t)
	body := "Positive coverage for guardCloseEvidenced needs resolved-task, applied-record, and sop-coverage each independently.\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "RETIRED clause sop-coverage") {
			found = true
		}
	}
	if !found {
		t.Fatalf("retired survivor not flagged: %v", g.Warns)
	}
}

func TestClausePartialEnumerationWarns(t *testing.T) {
	d := clauseDesign(t)
	body := "the falsifying case: guardCloseEvidenced with only resolved-task present\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "applied-record missing") {
			found = true
		}
	}
	if !found {
		t.Fatalf("partial enumeration not flagged: %v", g.Warns)
	}
}

func TestClauseFullOrZeroEnumerationSilent(t *testing.T) {
	d := clauseDesign(t)
	body := "guardCloseEvidenced reads resolved-task and applied-record together.\n" +
		"guardCloseEvidenced refuses a bare close with no evidence.\n"
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "guardCloseEvidenced") {
			t.Fatalf("full or zero enumeration flagged: %v", g.Warns)
		}
	}
}

func TestClauseLedgersExempt(t *testing.T) {
	d := clauseDesign(t)
	body := "the old guardCloseEvidenced read resolved-task alone here\n"
	if err := os.WriteFile(filepath.Join(d, "STATE.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "STATE.md") {
			t.Fatalf("ledger judged for clause drift: %v", g.Warns)
		}
	}
}
