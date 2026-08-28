package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func proseDesign(t *testing.T, body string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.oracle.md"), []byte("| T-DEAL-01 | DEAL-abc123 | a | b | c | d | - |\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "BUILD.md"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

const twoRowTable = "| a | b |\n|---|---|\n| 1 | 2 |\n| 3 | 4 |\n"

func TestProseCountMismatchWarns(t *testing.T) {
	d := proseDesign(t, "**Consumed by ops (3 rows, counted off this table).**\n\n"+twoRowTable)
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "claims 3 rows") && strings.Contains(w, "has 2 data rows") {
			found = true
		}
	}
	if !found {
		t.Fatalf("mismatch not flagged: %v", g.Warns)
	}
}

func TestProseCountMatchSilent(t *testing.T) {
	d := proseDesign(t, "**Consumed by ops (2 rows, counted off this table).**\n\n"+twoRowTable)
	g := CheckIDCitations(d)
	if len(g.Warns) != 0 {
		t.Fatalf("correct claim flagged: %v", g.Warns)
	}
	if g.Counts["row-count claims beside tables"] != 1 {
		t.Fatalf("claim not counted: %v", g.Counts)
	}
}

func TestProseCountHistoricalExempt(t *testing.T) {
	d := proseDesign(t, "This header read (5 rows, counted off this table), then 6, before landing here.\n\n"+twoRowTable)
	g := CheckIDCitations(d)
	if len(g.Warns) != 0 {
		t.Fatalf("historical note flagged: %v", g.Warns)
	}
}

func TestProseCountUnboundClaimIgnored(t *testing.T) {
	// A count about something else above a table must not bind to it.
	d := proseDesign(t, "the seven runscope invariants and 174 rows total live elsewhere\n\n"+twoRowTable)
	g := CheckIDCitations(d)
	if len(g.Warns) != 0 {
		t.Fatalf("unbound claim flagged: %v", g.Warns)
	}
}
