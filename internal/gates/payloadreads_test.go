package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readsDesign(t *testing.T, matrixLine, archRow string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.oracle.md"), []byte("| T-DEAL-01 | DEAL-abc123 | a | b | c | d | - |\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(matrixLine+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "ARCHITECTURE.md"), []byte(archRow+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

const archSettledRow = "| `order.settled` | billing | ledger | Order id, settlementClass, declineCause | at-least-once | FIFO | order id |"

func TestPayloadReadsCarriedSilent(t *testing.T) {
	d := readsDesign(t, "| `order.settled` (consumed) | routes on it READS{settlementClass, declineCause} |", archSettledRow)
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "order.settled") {
			t.Fatalf("carried reads flagged: %v", g.Warns)
		}
	}
	if g.Counts["payload READS declarations"] != 1 {
		t.Fatalf("declaration not counted: %v", g.Counts)
	}
}

func TestPayloadReadsMissingFieldWarns(t *testing.T) {
	d := readsDesign(t, "| `order.settled` (consumed) | routes on it READS{settlementClass, failureCause} |", archSettledRow)
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "reads failureCause") && strings.Contains(w, "payload-sufficiency") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing field not flagged: %v", g.Warns)
	}
}

// An armed design's event reads belong to Gx-trace, at ERROR strength (see
// readscomplete.go): the opt-in warn tier stands down for the events the armed
// contract names, so one defect never earns two findings.
func TestPayloadReadsStandsDownOnAnArmedDesign(t *testing.T) {
	d := readsDesign(t, "| `order.settled` (consumed) | routes on it READS{settlementClass, failureCause} |",
		"<!-- machinery:reads-complete -->\n\n| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n"+archSettledRow)
	g := CheckIDCitations(d)
	for _, w := range g.Warns {
		if strings.Contains(w, "order.settled") {
			t.Fatalf("the armed tier owns this event; Gd must stand down: %v", g.Warns)
		}
	}
}

// Standing down is per event, not wholesale: a declaration naming an event the
// armed contract never lists is still nobody else's, so the warn tier keeps it.
func TestPayloadReadsKeepsEventsTheArmedContractDoesNotName(t *testing.T) {
	d := readsDesign(t, "| `ghost.event` (consumed) | READS{x} |",
		"<!-- machinery:reads-complete -->\n\n| event | producer | consumer | payload | delivery | ordering | dedupe |\n|---|---|---|---|---|---|---|\n"+archSettledRow)
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "ghost.event") && strings.Contains(w, "no event-table row") {
			found = true
		}
	}
	if !found {
		t.Fatalf("an unlisted event keeps its opt-in warn: %v", g.Warns)
	}
}

func TestPayloadReadsNoEventRowWarns(t *testing.T) {
	d := readsDesign(t, "| `ghost.event` (consumed) | READS{x} |", archSettledRow)
	g := CheckIDCitations(d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "ghost.event") && strings.Contains(w, "no event-table row") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing event row not flagged: %v", g.Warns)
	}
}
