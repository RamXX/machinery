package gates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/version"
)

const gateModel = `kind: DomainModel
version: v1
title: T
entities:
  DataSubject:
    attributes:
      - {name: email, type: string}
    relationships:
      - {entity: Export, cardinality: 1:n}
    invariants:
      - {id: priv-consent, statement: "Consent required."}
      - {id: priv-retention, statement: "Retention bounded."}
  Export:
    attributes:
      - {name: name, type: string}
`

type ckOpts struct {
	verdict   string
	coverage  []checker.CoverageRow
	residuals string // YAML lines under coverage, or ""
	findings  []checker.Finding
}

func passCoverage() []checker.CoverageRow {
	return []checker.CoverageRow{
		{Element: "inv:priv-consent", Verdict: "pass"},
		{Element: "inv:priv-retention", Verdict: "pass"},
	}
}

// setupChecker writes a complete, by-default-passing checker design and returns
// the design dir plus the projection and evidence paths for per-case mutation.
func setupChecker(t *testing.T, o ckOpts) (design, projPath, evPath, modelPath string) {
	t.Helper()
	design = t.TempDir()
	modelPath = filepath.Join(design, "d.modelith.yaml")
	must(t, os.WriteFile(modelPath, []byte(gateModel), 0o644))

	must(t, os.MkdirAll(filepath.Join(design, "checkers", "test"), 0o755))
	man := "checker: {id: test}\n" +
		"projection: {include: [model, invariants, relationships]}\n" +
		"coverage:\n  claim: [\"priv-*\"]\n" + o.residuals +
		"evidence:\n  projection_out: checkers/test/projection.json\n  evidence_in: checkers/test/evidence.json\n"
	must(t, os.WriteFile(filepath.Join(design, "checkers", "test.checker.yaml"), []byte(man), 0o644))

	m, err := checker.LoadManifest(filepath.Join(design, "checkers", "test.checker.yaml"))
	must(t, err)
	model, err := checker.LoadModel(modelPath)
	must(t, err)
	did, err := checker.DesignID(modelPath)
	must(t, err)
	proj, err := checker.Generate(model, m, did, version.Version)
	must(t, err)
	rendered, err := proj.Render()
	must(t, err)
	projPath = filepath.Join(design, "checkers", "test", "projection.json")
	must(t, os.WriteFile(projPath, rendered, 0o644))
	hash, err := proj.InputHash()
	must(t, err)

	ev := checker.Evidence{EvidenceSchema: checker.SchemaVersion, InputHash: hash, Verdict: o.verdict, Coverage: o.coverage, Findings: o.findings}
	ev.Checker.ID = "test"
	ev.Checker.Version = "t"
	evPath = filepath.Join(design, "checkers", "test", "evidence.json")
	writeJSONFile(t, evPath, ev)
	return
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func writeJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	must(t, err)
	must(t, os.WriteFile(path, b, 0o644))
}

func onlyGate(t *testing.T, design string) *Gate {
	t.Helper()
	gs := CheckExternalCheckers(design)
	if len(gs) != 1 {
		t.Fatalf("expected 1 gate, got %d", len(gs))
	}
	return gs[0]
}

func blocking(g *Gate) int { return len(g.Errs) + len(g.Drift) }

func TestGkPass(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	g := onlyGate(t, design)
	if b := blocking(g); b != 0 {
		t.Fatalf("clean design has %d blocking findings: errs=%v drift=%v", b, g.Errs, g.Drift)
	}
}

func TestGkMissingEvidenceIsError(t *testing.T) {
	design, _, evPath, _ := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	must(t, os.Remove(evPath))
	g := onlyGate(t, design)
	if len(g.Errs) == 0 {
		t.Fatal("absent evidence must be an ERROR")
	}
}

func TestGkStaleBindingIsDrift(t *testing.T) {
	design, _, evPath, _ := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	var ev checker.Evidence
	raw, _ := os.ReadFile(evPath)
	must(t, json.Unmarshal(raw, &ev))
	ev.InputHash = "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000"
	writeJSONFile(t, evPath, ev)
	g := onlyGate(t, design)
	if len(g.Drift) == 0 {
		t.Fatalf("a verdict over a different design must be DRIFT: %+v", g)
	}
}

func TestGkCoverageGapIsError(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{
		verdict:  "pass",
		coverage: []checker.CoverageRow{{Element: "inv:priv-consent", Verdict: "pass"}},
	})
	g := onlyGate(t, design)
	if len(g.Errs) == 0 {
		t.Fatal("a claimed invariant with no coverage and no residual must be an ERROR")
	}
}

func TestGkResidualWaivesCoverage(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{
		verdict:   "pass",
		coverage:  []checker.CoverageRow{{Element: "inv:priv-consent", Verdict: "pass"}},
		residuals: "  residuals:\n    - {id: priv-retention, reason: \"operational control\"}\n",
	})
	g := onlyGate(t, design)
	if b := blocking(g); b != 0 {
		t.Fatalf("a declared residual should waive coverage, got %d blocking: %v %v", b, g.Errs, g.Drift)
	}
}

func TestGkUnknownResidualIsError(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{
		verdict:   "pass",
		coverage:  passCoverage(),
		residuals: "  residuals:\n    - {id: nonexistent-inv, reason: \"x\"}\n",
	})
	g := onlyGate(t, design)
	if len(g.Errs) == 0 {
		t.Fatal("a residual naming a nonexistent invariant must be an ERROR")
	}
}

func TestGkFailVerdictIsError(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{
		verdict:  "fail",
		coverage: passCoverage(),
		findings: []checker.Finding{{Severity: "blocking", Code: "leak", Message: "email reaches export unredacted"}},
	})
	g := onlyGate(t, design)
	if len(g.Errs) == 0 {
		t.Fatal("a fail verdict must be an ERROR")
	}
}

func TestGkStaleProjectionIsDrift(t *testing.T) {
	design, _, _, modelPath := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	// Grow the model after the projection was committed: the committed projection
	// is now stale relative to a fresh generation.
	grown := gateModel + "  AuditLog:\n    attributes:\n      - {name: line, type: string}\n"
	must(t, os.WriteFile(modelPath, []byte(grown), 0o644))
	g := onlyGate(t, design)
	if len(g.Drift) == 0 {
		t.Fatalf("a stale committed projection must be DRIFT: %+v", g)
	}
}

func TestGkCheckerIDMismatchIsError(t *testing.T) {
	design, _, evPath, _ := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	var ev checker.Evidence
	raw, _ := os.ReadFile(evPath)
	must(t, json.Unmarshal(raw, &ev))
	ev.Checker.ID = "someone-else"
	writeJSONFile(t, evPath, ev)
	g := onlyGate(t, design)
	if len(g.Errs) == 0 {
		t.Fatal("evidence produced for a different checker id must be an ERROR")
	}
}

func TestHasCheckers(t *testing.T) {
	design, _, _, _ := setupChecker(t, ckOpts{verdict: "pass", coverage: passCoverage()})
	if !HasCheckers(design) {
		t.Fatal("HasCheckers should detect the committed manifest")
	}
	if HasCheckers(t.TempDir()) {
		t.Fatal("HasCheckers should be false with no checkers/ dir")
	}
}
