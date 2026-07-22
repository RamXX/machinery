package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/alloy"
)

func TestCheckIsolationCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	g := CheckIsolation(design)
	if len(g.Errs) != 0 {
		t.Errorf("Gn errors: %v", g.Errs)
	}
	if len(g.Drift) != 0 {
		t.Errorf("Gn drift: %v", g.Drift)
	}
	if g.Counts["references"] == 0 || g.Counts["tenant oracle rows generated"] == 0 {
		t.Errorf("Gn counted nothing: %+v", g.Counts)
	}
}

func TestCheckIsolationMissingAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	if err := os.Remove(filepath.Join(design, "formal", alloy.IsolationAnnotationName)); err != nil {
		t.Fatal(err)
	}
	g := CheckIsolation(design)
	if len(g.Errs) == 0 || !strings.Contains(g.Errs[0], "never authored") {
		t.Errorf("want explicit-request error, got %v", g.Errs)
	}
}

func TestCheckIsolationStaleModelIsDrift(t *testing.T) {
	design := copyDesign(t)
	als := filepath.Join(design, "formal", alloy.IsolationOutputName)
	if err := os.WriteFile(als, []byte("// hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckIsolation(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "stale") {
		t.Errorf("want DRIFT for stale model, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckIsolationStaleOracleIsDrift(t *testing.T) {
	design := copyDesign(t)
	oracle := filepath.Join(design, "formal", alloy.IsolationOracleName)
	if err := os.WriteFile(oracle, []byte("# hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckIsolation(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "stale") {
		t.Errorf("want DRIFT for stale oracle, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckIsolationBrokenAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	ann := filepath.Join(design, "formal", alloy.IsolationAnnotationName)
	raw, err := os.ReadFile(ann)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(raw), "invariant: task-deal-same-tenant", "invariant: task-deal-same-tenanz", 1)
	if broken == string(raw) {
		t.Fatal("fixture assumption broken: task-deal-same-tenant not found")
	}
	if err := os.WriteFile(ann, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckIsolation(design)
	if len(g.Errs) == 0 {
		t.Errorf("drifted annotation must fail generation, got errs=%v", g.Errs)
	}
}

func TestSelectIncludesGn(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gn"] {
		t.Error("default selection must include gn")
	}
	if _, err := Select(design, "gn,g3", ""); err != nil {
		t.Errorf("explicit gn rejected: %v", err)
	}
}

func TestRunSelectedSkipsGnWithoutAnnotation(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "fulfillment", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range RunSelected(design, "", sel) {
		if strings.Contains(g.Title, "Gn-isolation") {
			t.Error("Gn must not run on a design without the annotation")
		}
	}
}
