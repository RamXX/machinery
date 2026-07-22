package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/oracle"
	"github.com/RamXX/machinery/internal/version"
)

// repoRoot resolves to the repo root (the test runs from internal/gates).
func repoRoot() string { return "../.." }

func TestCheckC4CleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckC4(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G2 errors: %v", ex, g.Errs)
			}
			if g.Counts["boundaries"] == 0 {
				t.Errorf("%s: no boundaries parsed", ex)
			}
		})
	}
}

func TestCheckMachinesCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckMachines(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: G3 errors: %v", ex, g.Errs)
			}
			if len(g.Drift) != 0 {
				t.Errorf("%s: G3 drift: %v", ex, g.Drift)
			}
			if g.Counts["machines"] == 0 {
				t.Errorf("%s: no machines parsed", ex)
			}
		})
	}
}

func TestCheckTraceabilityCleanOnExamples(t *testing.T) {
	for _, ex := range []string{"go-crm", "fulfillment", "portfolio-engine"} {
		t.Run(ex, func(t *testing.T) {
			design := filepath.Join(repoRoot(), "examples", ex, "design")
			g := CheckTraceability(design)
			if len(g.Errs) != 0 {
				t.Errorf("%s: Gx errors: %v", ex, g.Errs)
			}
		})
	}
}

func TestCheckImportsCleanOnGoCRM(t *testing.T) {
	design := filepath.Join(repoRoot(), "examples", "go-crm", "design")
	impl := filepath.Join(repoRoot(), "examples", "go-crm", "impl")
	g := CheckImports(design, impl)
	if len(g.Errs) != 0 {
		t.Errorf("G4 errors: %v", g.Errs)
	}
	if g.Counts["go files checked"] == 0 {
		t.Error("no go files checked")
	}
}

// G2 concatenates EVERY event-contract table (header with producer, consumer,
// delivery): the first-match locator silently ignored later tables (PACK-1).
func TestG2EventContractTablesConcatenate(t *testing.T) {
	design := t.TempDir()
	dsl := "workspace \"W\" \"sys\" {\n  model {\n    sys = softwareSystem \"S\" \"sys\" {\n" +
		"      app = component \"App\" \"logic\" \"Go\"\n" +
		"      bus = container \"Bus\" \"events\" \"NATS\" \"Queue\"\n    }\n  }\n}\n"
	arch := "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n" +
		"  - id: app\n    code: [\"app/**\"]\n```\n\n" +
		"## Events (orders)\n\n| producer | consumer | payload | delivery |\n|---|---|---|---|\n" +
		"| app | bus | OrderPlaced | at-least-once |\n| app | bus | OrderPaid | at-least-once |\n\n" +
		"## Events (payments)\n\n| producer | consumer | payload | delivery |\n|---|---|---|---|\n" +
		"| bus | app | PaymentSettled | at-least-once |\n"
	mustWrite(t, filepath.Join(design, "workspace.dsl"), dsl)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), arch)
	g := CheckC4(design)
	if g.Counts["event contracts"] != 3 {
		t.Fatalf("event contracts = %d, want 3 (rows of BOTH tables): %+v", g.Counts["event contracts"], g.Counts)
	}
	if hasErr(g, "no event-contract table") {
		t.Fatalf("the tables exist and must be found: %v", g.Errs)
	}
}

// The Gx placement waiver must sit in the component or machine-placement
// cell and carry a non-empty reason: '(no machine:)' with an empty reason, or
// the token buried elsewhere in the row, waives nothing (GATE-5).
func TestGxPlacementWaiverColumnAndReason(t *testing.T) {
	writeGxFixture := func(t *testing.T, placementRow string) *Gate {
		t.Helper()
		design := t.TempDir()
		mustWrite(t, filepath.Join(design, "domain.modelith.yaml"),
			"kind: modelith\nversion: 1\nentities:\n  Widget:\n    actions:\n      - name: publish\n    invariants:\n      - id: widget-owned\n")
		mustWrite(t, filepath.Join(design, "machines", "Ops.machine.json"),
			`{"id":"ops","_role":"operational","initial":"A","states":{"A":{}}}`)
		mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"),
			"# A\n\n## Placement\n\n| component | machine placement | persistence | concurrency |\n|---|---|---|---|\n"+
				placementRow+"\n")
		return CheckTraceability(design)
	}
	t.Run("empty reason is not a waiver", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | wherever (no machine:) | none | n/a |")
		if !hasErr(g, "has no machine and no '(no machine: <reason>)' waiver") {
			t.Fatalf("an empty waiver reason must not waive: %v", g.Errs)
		}
	})
	t.Run("token outside the machine columns is not a waiver", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | somewhere | none | racy (no machine: honest note in the wrong column) |")
		if !hasErr(g, "has no machine and no '(no machine: <reason>)' waiver") {
			t.Fatalf("the token buried in another cell must not waive: %v", g.Errs)
		}
	})
	t.Run("placement-cell waiver with a reason waives", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` | none (no machine: pure transform) | none | n/a |")
		if hasErr(g, "has no machine and no") {
			t.Fatalf("a reasoned placement-cell waiver must waive: %v", g.Errs)
		}
		if g.Counts["placement rows waived"] != 1 {
			t.Errorf("placement rows waived = %d, want 1: %+v", g.Counts["placement rows waived"], g.Counts)
		}
	})
	t.Run("component-cell waiver with a reason waives", func(t *testing.T) {
		g := writeGxFixture(t, "| `Ghost` (no machine: reference data, upserted) | none | none | n/a |")
		if hasErr(g, "has no machine and no") {
			t.Fatalf("a reasoned component-cell waiver must waive (the portfolio-engine shape): %v", g.Errs)
		}
		if g.Counts["placement rows waived"] != 1 {
			t.Errorf("placement rows waived = %d, want 1: %+v", g.Counts["placement rows waived"], g.Counts)
		}
	})
}

func TestTokenInWholeToken(t *testing.T) {
	if !tokenIn("inv-1", "foo inv-1 bar") {
		t.Error("inv-1 should match standalone")
	}
	if tokenIn("inv-1", "foo inv-12 bar") {
		t.Error("inv-1 must not match inside inv-12")
	}
}

func TestGateEmitFormatting(t *testing.T) {
	g := NewGate("Test")
	g.Count("widgets")
	g.Count("widgets")
	g.Count("gadgets", 3)
	if g.Counts["widgets"] != 2 || g.Counts["gadgets"] != 3 {
		t.Errorf("counts wrong: %+v", g.Counts)
	}
	// order must be insertion order
	if g.countOrder[0] != "widgets" || g.countOrder[1] != "gadgets" {
		t.Errorf("count order wrong: %v", g.countOrder)
	}
}

// --- P-F10: generator version stamps and the freshness diff ---

const stampWidgetMachine = `{"id":"widget","initial":"Draft",
  "_delays":{"persistTimeout":"3000 ms - test bound"},
  "states":{
  "Draft":{"on":{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}]}},
  "Published":{"type":"final"},
  "persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`

// writeStampFixture builds a design with one machine and a committed oracle
// whose text the caller can transform first (identity when nil).
func writeStampFixture(t *testing.T, mutate func(string) string) string {
	t.Helper()
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mp := filepath.Join(mdir, "Widget.machine.json")
	if err := os.WriteFile(mp, []byte(stampWidgetMachine), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := ir.LoadMachineJSON(mp)
	if err != nil {
		t.Fatal(err)
	}
	text := oracle.Render(m, mp)
	if mutate != nil {
		text = mutate(text)
	}
	if err := os.WriteFile(filepath.Join(mdir, "Widget.oracle.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return design
}

// A committed oracle stamped by a DIFFERENT machinery version, content
// otherwise identical, is fresh: version-only skew is never DRIFT.
func TestOracleVersionOnlySkewIsNotDrift(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("version-only skew reported as drift: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if g.Counts["oracles fresh"] != 1 {
		t.Errorf("oracle not counted fresh: %v", g.Counts)
	}
	note := VersionSkewNote([]*Gate{g})
	if !strings.Contains(note, "v0.0.1") || !strings.Contains(note, version.Version) {
		t.Errorf("skew note = %q, want both versions named", note)
	}
	if !strings.HasPrefix(note, "note: artifacts generated by machinery ") || !strings.HasSuffix(note, "; regenerate on upgrade") {
		t.Errorf("skew note format drifted: %q", note)
	}
}

// Content drift is still DRIFT, stamp or no stamp.
func TestOracleContentDriftStillDrift(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, "| Draft |", "| Drafted |", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) == 0 {
		t.Fatal("content drift not reported")
	}
}

// A committed oracle with NO stamp (pre-stamp artifact) is fresh when the
// content matches, and produces no skew note.
func TestOracleMissingStampIsFreshAndSilent(t *testing.T) {
	design := writeStampFixture(t, func(text string) string {
		return strings.Replace(text, version.MarkdownStamp()+"\n", "", 1)
	})
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("pre-stamp oracle reported stale: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if note := VersionSkewNote([]*Gate{g}); note != "" {
		t.Errorf("missing stamp must not be skew: %q", note)
	}
}

// A committed oracle stamped with the RUNNING version produces no note.
func TestOracleCurrentStampIsSilent(t *testing.T) {
	design := writeStampFixture(t, nil)
	g := CheckMachines(design)
	if len(g.Drift) != 0 || len(g.Errs) != 0 {
		t.Fatalf("identical oracle reported stale: errs=%v drift=%v", g.Errs, g.Drift)
	}
	if note := VersionSkewNote([]*Gate{g}); note != "" {
		t.Errorf("current stamp must not be skew: %q", note)
	}
}
