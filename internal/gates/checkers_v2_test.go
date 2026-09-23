package gates

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/version"
)

// copyTree copies a directory tree of regular files.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The v1 compatibility proof: the committed pii-flow projection (a v1-only
// manifest) is regenerated through the v2-aware writer, with the fact reader
// supplied, and must come out byte for byte as committed.
func TestPIIFlowV1ProjectionIsByteIdenticalThroughTheV2Writer(t *testing.T) {
	src := filepath.Join("..", "..", "examples", "pii-flow", "design")
	committedPath := filepath.Join(src, "checkers", "pii-flow", "projection.json")
	committed, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := checker.ParseProjection(committed)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ProjectionSchema != checker.SchemaVersion || parsed.Layers != nil {
		t.Fatalf("the committed pii-flow projection must stay 1.0 with no layers: %s", parsed.ProjectionSchema)
	}
	design := filepath.Join(t.TempDir(), "design")
	copyTree(t, src, design)
	results, err := checker.ProjectAllWithFacts(design, parsed.MachineryVersion, LoadDesignFacts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("one projection expected, got %+v", results)
	}
	after, err := os.ReadFile(filepath.Join(design, "checkers", "pii-flow", "projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(committed, after) {
		t.Fatalf("v1 projection bytes changed:\nbefore:\n%s\nafter:\n%s", committed, after)
	}
}

const v2GateMachine = `{
  "id": "subject",
  "initial": "Active",
  "states": {
    "Active": {"on": {"export": {"target": "Exported"}}},
    "Exported": {"type": "final"}
  }
}
`

// setupV2Checker writes a design whose manifest includes the machines layer,
// projects it through the v2 writer, and binds evidence to its input hash.
func setupV2Checker(t *testing.T) (design string) {
	t.Helper()
	design = t.TempDir()
	must(t, os.WriteFile(filepath.Join(design, "d.modelith.yaml"), []byte(gateModel), 0o644))
	must(t, os.MkdirAll(filepath.Join(design, "machines"), 0o755))
	must(t, os.WriteFile(filepath.Join(design, "machines", "DataSubject.machine.json"), []byte(v2GateMachine), 0o644))
	must(t, os.MkdirAll(filepath.Join(design, "checkers", "test"), 0o755))
	man := "checker: {id: test, runtime_closure: " + gateRuntimeClosure + "}\n" +
		"projection: {include: [model, invariants, machines]}\n" +
		"coverage:\n  claim: [\"priv-*\"]\n" +
		"evidence:\n  projection_out: checkers/test/projection.json\n  evidence_in: checkers/test/evidence.json\n"
	must(t, os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644))
	if _, err := checker.ProjectAllWithFacts(design, version.Version, LoadDesignFacts); err != nil {
		t.Fatal(err)
	}
	rendered, err := os.ReadFile(filepath.Join(design, "checkers", "test", "projection.json"))
	must(t, err)
	proj, err := checker.ParseProjection(rendered)
	must(t, err)
	if proj.ProjectionSchema != checker.ProjectionSchemaV2 {
		t.Fatalf("a manifest including machines must get the 2.0 projection, got %s", proj.ProjectionSchema)
	}
	if got := len(proj.Layers.Rows("machines", "state")); got != 2 {
		t.Fatalf("the machines layer must carry both states, got %d", got)
	}
	hash, err := proj.InputHash()
	must(t, err)
	ev := checker.Evidence{EvidenceSchema: checker.SchemaVersion, InputHash: hash, RuntimeClosure: gateRuntimeClosure, Verdict: "pass", Coverage: passCoverage()}
	ev.Checker.ID = "test"
	ev.Checker.Version = "t"
	writeJSONFile(t, filepath.Join(design, "checkers", "test", "evidence.json"), ev)
	return design
}

func TestGkBindsAV2Projection(t *testing.T) {
	design := setupV2Checker(t)
	g := onlyGate(t, design)
	if b := blocking(g); b != 0 {
		t.Fatalf("a fresh v2 projection with bound evidence must pass: errs=%v drift=%v", g.Errs, g.Drift)
	}
}

func TestGkV2ProjectionGoesStaleWhenAMachineChanges(t *testing.T) {
	design := setupV2Checker(t)
	changed := strings.Replace(v2GateMachine, `"Exported": {"type": "final"}`, `"Exported": {"type": "final"}, "Erased": {"type": "final"}`, 1)
	must(t, os.WriteFile(filepath.Join(design, "machines", "DataSubject.machine.json"), []byte(changed), 0o644))
	g := onlyGate(t, design)
	joined := strings.Join(g.Drift, "\n")
	if !strings.Contains(joined, "is stale") || !strings.Contains(joined, "input_hash does not match") {
		t.Fatalf("a machine edit must stale both the v2 projection and the evidence binding: drift=%v errs=%v", g.Drift, g.Errs)
	}
}
