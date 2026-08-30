package gates

import (
	"os"
	"path/filepath"
	"testing"
)

const adjudicationOracle = `# Oracle: Deal

| test id | stable id | given | event | next | actions | notes |
|---|---|---|---|---|---|---|
| T-DEAL-01 | DEAL-abc123 | Open | win | Won | save | - |
| T-DEAL-02 | DEAL-def456 | Open | lose | Lost | save | - |
`

func adjudicationFixture(t *testing.T, evidence string) string {
	t.Helper()
	design := t.TempDir()
	for _, dir := range []string{"machines", AdjudicationDirName} {
		if err := os.MkdirAll(filepath.Join(design, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(t, filepath.Join(design, "machines", "Deal.oracle.md"), adjudicationOracle)
	if evidence != "" {
		mustWrite(t, filepath.Join(design, AdjudicationDirName, "Deal.yaml"), evidence)
	}
	return design
}

const adjudicationClean = `adjudication_version: 1
machine: Deal
rows:
  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-08-29, note: legacy always saves before the stage flips; model fixed}
  - {id: DEAL-def456, verdict: model-is-truth, date: 2026-08-29, note: the code silently swallows the loss reason, defect: BUG-41}
`

func TestAdjudicationClean(t *testing.T) {
	g := CheckAdjudications(adjudicationFixture(t, adjudicationClean))
	if len(g.Errs) != 0 {
		t.Fatalf("clean evidence must pass: %v", g.Errs)
	}
	if g.Counts["adjudicated rows"] != 2 || g.Counts["code-is-truth verdicts"] != 1 || g.Counts["model-is-truth verdicts"] != 1 {
		t.Fatalf("counts wrong: %+v", g.Counts)
	}
	if !AdjudicationActive(adjudicationFixture(t, adjudicationClean)) {
		t.Fatal("AdjudicationActive must be true")
	}
}

func TestAdjudicationMutations(t *testing.T) {
	cases := []struct {
		name, evidence, want string
	}{
		{"unknown root key", adjudicationClean + "bogus: true\n", `unsupported key "bogus"`},
		{"bad version", "adjudication_version: 2\nmachine: Deal\nrows: []\n", "adjudication_version must be the integer 1"},
		{"machine mismatch", "adjudication_version: 1\nmachine: Task\nrows:\n  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-08-29, note: n}\n", "does not match the file name"},
		{"empty rows", "adjudication_version: 1\nmachine: Deal\nrows: []\n", "rows must be a non-empty list"},
		{"dangling id", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-999999, verdict: code-is-truth, date: 2026-08-29, note: n}\n", "resolves to no stable id"},
		{"bad verdict", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-abc123, verdict: maybe, date: 2026-08-29, note: n}\n", "not code-is-truth or model-is-truth"},
		{"bad date", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-13-01, note: n}\n", "not a real YYYY-MM-DD date"},
		{"missing note", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-08-29}\n", "note is required"},
		{"model-is-truth needs defect", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-abc123, verdict: model-is-truth, date: 2026-08-29, note: n}\n", "requires defect"},
		{"duplicate id", "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-08-29, note: a}\n  - {id: DEAL-abc123, verdict: code-is-truth, date: 2026-08-29, note: b}\n", "adjudicates DEAL-abc123 twice"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := CheckAdjudications(adjudicationFixture(t, tc.evidence))
			if !hasErr(g, tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestAdjudicationRemovedIDAllowance(t *testing.T) {
	design := adjudicationFixture(t, "adjudication_version: 1\nmachine: Deal\nrows:\n  - {id: DEAL-999999, verdict: code-is-truth, date: 2026-08-29, note: adjudicated before the redesign}\n")
	mustWrite(t, filepath.Join(design, "removed-ids.txt"), "DEAL-999999\n")
	g := CheckAdjudications(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a removed-id verdict is history, not a dangling reference: %v", g.Errs)
	}
}

func TestAdjudicationEmptyDirFails(t *testing.T) {
	design := adjudicationFixture(t, "")
	g := CheckAdjudications(design)
	if !hasErr(g, "holds no *.yaml") {
		t.Fatalf("an empty evidence directory must fail: %v", g.Errs)
	}
}

func TestAdjudicationMissingOracleFails(t *testing.T) {
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, AdjudicationDirName), 0755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, AdjudicationDirName, "Deal.yaml"), adjudicationClean)
	g := CheckAdjudications(design)
	if !hasErr(g, "no committed oracle") {
		t.Fatalf("evidence without an oracle must fail: %v", g.Errs)
	}
}
