package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/alloy"
)

func TestCheckIntegrityCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	g := CheckIntegrity(design)
	if len(g.Errs) != 0 {
		t.Errorf("Gi errors: %v", g.Errs)
	}
	if len(g.Drift) != 0 {
		t.Errorf("Gi drift: %v", g.Drift)
	}
	if g.Counts["unique keys"] == 0 || g.Counts["solver commands generated"] == 0 {
		t.Errorf("Gi counted nothing: %+v", g.Counts)
	}
}

func TestCheckIntegrityCleanOnFulfillment(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "fulfillment", "design")
	g := CheckIntegrity(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Errorf("Gi not clean on fulfillment: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if g.Counts["relationships"] == 0 {
		t.Errorf("Gi did not count the modeled relationships: %+v", g.Counts)
	}
}

func TestCheckIntegrityMissingAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	if err := os.Remove(filepath.Join(design, "formal", alloy.IntegrityAnnotationName)); err != nil {
		t.Fatal(err)
	}
	g := CheckIntegrity(design)
	if len(g.Errs) == 0 || !strings.Contains(g.Errs[0], "never authored") {
		t.Errorf("want explicit-request error, got %v", g.Errs)
	}
}

func TestCheckIntegrityStaleModelIsDrift(t *testing.T) {
	design := copyDesign(t)
	als := filepath.Join(design, "formal", alloy.IntegrityOutputName)
	if err := os.WriteFile(als, []byte("// hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckIntegrity(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "stale") {
		t.Errorf("want DRIFT for stale model, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckIntegrityMissingModelIsDrift(t *testing.T) {
	design := copyDesign(t)
	if err := os.Remove(filepath.Join(design, "formal", alloy.IntegrityOutputName)); err != nil {
		t.Fatal(err)
	}
	g := CheckIntegrity(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "never generated") {
		t.Errorf("want DRIFT for missing model, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckIntegrityBrokenAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	ann := filepath.Join(design, "formal", alloy.IntegrityAnnotationName)
	raw, err := os.ReadFile(ann)
	if err != nil {
		t.Fatal(err)
	}
	// drift attack: the annotation binds an invariant the domain model no
	// longer declares
	broken := strings.Replace(string(raw), "invariant: username-unique", "invariant: username-uniqz", 1)
	if broken == string(raw) {
		t.Fatal("fixture assumption broken: username-unique not found")
	}
	if err := os.WriteFile(ann, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckIntegrity(design)
	if len(g.Errs) == 0 {
		t.Errorf("drifted annotation must fail generation, got errs=%v", g.Errs)
	}
}

func TestSelectIncludesGi(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gi"] {
		t.Error("default selection must include gi")
	}
	if _, err := Select(design, "gi,g3", ""); err != nil {
		t.Errorf("explicit gi rejected: %v", err)
	}
}

func TestRunSelectedSkipsGiWithoutAnnotation(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "portfolio-engine", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range RunSelected(design, "", sel) {
		if strings.Contains(g.Title, "Gi-integrity") {
			t.Error("Gi must not run on a design without the annotation")
		}
	}
}
