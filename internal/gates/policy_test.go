package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/alloy"
	"github.com/RamXX/machinery/internal/version"
)

func TestCheckPolicyCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	g := CheckPolicy(design)
	if len(g.Errs) != 0 {
		t.Errorf("Gp errors: %v", g.Errs)
	}
	if len(g.Drift) != 0 {
		t.Errorf("Gp drift: %v", g.Drift)
	}
	if g.Counts["policy rules"] == 0 || g.Counts["solver commands generated"] == 0 {
		t.Errorf("Gp counted nothing: %+v", g.Counts)
	}
}

// copyDesign clones the go-crm design tree into a temp dir so mutations
// never touch the example.
func copyDesign(t *testing.T) string {
	t.Helper()
	src := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	dst := t.TempDir()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestCheckPolicyMissingAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	if err := os.Remove(filepath.Join(design, "formal", alloy.AnnotationName)); err != nil {
		t.Fatal(err)
	}
	g := CheckPolicy(design)
	if len(g.Errs) == 0 || !strings.Contains(g.Errs[0], "never authored") {
		t.Errorf("want explicit-request error, got %v", g.Errs)
	}
}

func TestCheckPolicyStaleModelIsDrift(t *testing.T) {
	design := copyDesign(t)
	als := filepath.Join(design, "formal", alloy.OutputName)
	if err := os.WriteFile(als, []byte("// hand-edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckPolicy(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "stale") {
		t.Errorf("want DRIFT for stale model, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckPolicyMissingModelIsDrift(t *testing.T) {
	design := copyDesign(t)
	if err := os.Remove(filepath.Join(design, "formal", alloy.OutputName)); err != nil {
		t.Fatal(err)
	}
	g := CheckPolicy(design)
	if len(g.Drift) == 0 || !strings.Contains(g.Drift[0], "never generated") {
		t.Errorf("want DRIFT for missing model, got drift=%v errs=%v", g.Drift, g.Errs)
	}
}

func TestCheckPolicyBrokenAnnotationErrors(t *testing.T) {
	design := copyDesign(t)
	ann := filepath.Join(design, "formal", alloy.AnnotationName)
	raw, err := os.ReadFile(ann)
	if err != nil {
		t.Fatal(err)
	}
	// drift attack: the annotation references an invariant the domain model
	// no longer declares
	broken := strings.Replace(string(raw), "invariant: rbac-crud-verbs", "invariant: rbac-crud-verbz", 1)
	if broken == string(raw) {
		t.Fatal("fixture assumption broken: rbac-crud-verbs not found")
	}
	if err := os.WriteFile(ann, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckPolicy(design)
	if len(g.Errs) == 0 {
		t.Errorf("drifted annotation must fail generation, got errs=%v", g.Errs)
	}
}

func TestSelectIncludesGp(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gp"] {
		t.Error("default selection must include gp")
	}
	if _, err := Select(design, "gp,g3", ""); err != nil {
		t.Errorf("explicit gp rejected: %v", err)
	}
	if _, err := Select(design, "gq", ""); err == nil {
		t.Error("unknown gate must error")
	}
}

func TestRunSelectedSkipsGpWithoutAnnotation(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "fulfillment", "design")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range RunSelected(design, "", sel) {
		if strings.Contains(g.Title, "Gp-policy") {
			t.Error("Gp must not run on a design without the annotation")
		}
	}
}

// P-F10: a committed Policy.als stamped by another machinery version, content
// otherwise fresh, is not drift; the skew surfaces through VersionSkewNote.
func TestCheckPolicyVersionOnlySkewIsNotDrift(t *testing.T) {
	design := copyDesign(t)
	path := filepath.Join(design, "formal", alloy.OutputName)
	// normalize to the CURRENT generation first (the committed example may be
	// pre-stamp), then rewrite the stamp to a different version
	dm := filepath.Join(design, "domain.modelith.yaml")
	als, _, _, err := alloy.GenerateAll(dm, filepath.Join(design, "formal", alloy.AnnotationName))
	if err != nil {
		t.Fatal(err)
	}
	stamped := strings.Replace(als, version.AlloyStamp(), "// machinery-version: v0.0.9", 1)
	if stamped == als {
		t.Fatal("fresh generation carries no stamp to rewrite")
	}
	if err := os.WriteFile(path, []byte(stamped), 0o644); err != nil {
		t.Fatal(err)
	}
	g := CheckPolicy(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("version-only skew reported as drift: errs=%v drift=%v", g.Errs, g.Drift)
	}
	note := VersionSkewNote([]*Gate{g})
	if !strings.Contains(note, "v0.0.9") {
		t.Errorf("skew note = %q, want v0.0.9 named", note)
	}
}
