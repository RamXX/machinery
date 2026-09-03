package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func idciteDesign(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(d, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("machines/Deal.oracle.md", "| T-DEAL-01 | DEAL-abc123 | Lead | on:advance | canAdvance | Won | - |\n| T-DEAL-02 | DEAL-def456 | Lead | on:drop | - | Lost | - |\n")
	return d
}

func writeDesignFile(t *testing.T, design, rel, body string) {
	t.Helper()
	p := filepath.Join(design, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestIDCiteResolvesAndDangles(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "BUILD.md", "covered by `DEAL-abc123` and by DEAL-999999 too\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 1 || !strings.Contains(g.Errs[0], "DEAL-999999") || !strings.Contains(g.Errs[0], "BUILD.md:1") {
		t.Fatalf("errs: %v", g.Errs)
	}
}

func TestIDCiteSuffixConvention(t *testing.T) {
	// N3: a suffixed id with a resolving base is neither dangling nor a
	// citation of the base; a dangling base errors on the suffixed form.
	d := idciteDesign(t)
	writeDesignFile(t, d, "BUILD.md", "clause tests DEAL-abc123a and DEAL-abc123a..c hold; DEAL-777777b dangles\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 1 || !strings.Contains(g.Errs[0], "DEAL-777777b") {
		t.Fatalf("errs: %v", g.Errs)
	}
	if g.Counts["suffixed-form citations"] != 3 {
		t.Fatalf("suffixed count wrong: %v", g.Counts)
	}
	if g.Counts["citations resolved"] != 0 {
		t.Fatalf("a suffixed form must not credit the base as resolved: %v", g.Counts)
	}
}

func TestIDCiteRemovedOracleTagStillErrors(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "BUILD.md", "sha OTHER-abc123 is not a citation\n")
	g := CheckIDCitations(d)
	if !hasErr(g, "OTHER-abc123 resolves to no committed oracle row") {
		t.Fatalf("a stable-id-shaped citation from a removed oracle family must remain dangling: %v", g.Errs)
	}
}

func TestIDCiteRemovedAllowance(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "removed-ids.txt", "# removed 2026-08-28 with the retry re-cut\nDEAL-999999\n")
	writeDesignFile(t, d, "BUILD.md", "historically DEAL-999999 held this\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 0 {
		t.Fatalf("allowed removed id flagged: %v", g.Errs)
	}
	if g.Counts["allowed removed-id citations"] != 1 {
		t.Fatalf("allowance not counted: %v", g.Counts)
	}
}

func TestIDCiteLedgersUnjudged(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "DECISIONS.md", "the old DEAL-999999 row was replaced; T-DEAL-04 renumbered\n")
	writeDesignFile(t, d, "STATE.md", "| 5.1 | DEAL-888888 moved |\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 0 || len(g.Warns) != 0 {
		t.Fatalf("ledger judged: errs %v warns %v", g.Errs, g.Warns)
	}
	if g.Counts["ledger citations (historical, unjudged)"] != 2 {
		t.Fatalf("ledger citations not counted: %v", g.Counts)
	}
}

func TestIDCitePositionalWarning(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "BUILD.md", "see T-DEAL-01 for the row\n")
	g := CheckIDCitations(d)
	if len(g.Warns) != 1 || !strings.Contains(g.Warns[0], "T-DEAL-01") {
		t.Fatalf("warns: %v", g.Warns)
	}
	if len(g.Errs) != 0 {
		t.Fatalf("positional citation must warn, not error: %v", g.Errs)
	}
}

func TestIDCiteGeneratedFilesSkipped(t *testing.T) {
	d := idciteDesign(t)
	writeDesignFile(t, d, "formal/Deal.tla", "DEAL-999999\n")
	writeDesignFile(t, d, "packs/p.md", "DEAL-999999\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 0 {
		t.Fatalf("generated artifacts scanned: %v", g.Errs)
	}
}

func TestIDCiteNoOraclesWarns(t *testing.T) {
	d := t.TempDir()
	writeDesignFile(t, d, "BUILD.md", "DEAL-abc123\n")
	g := CheckIDCitations(d)
	if len(g.Errs) != 0 || len(g.Warns) != 1 {
		t.Fatalf("no-oracle design: errs %v warns %v", g.Errs, g.Warns)
	}
}
