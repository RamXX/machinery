package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ledgerDesign(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for name, body := range files {
		path := filepath.Join(d, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func hasFinding(list []string, substrs ...string) bool {
	for _, f := range list {
		all := true
		for _, s := range substrs {
			if !strings.Contains(f, s) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

func TestLedgerSelfReviewCleanLine(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "- Phase 1: `self-review: reality=clean depth=fixed scope=clean coverage=accepted(no edge scenarios yet) consistency=clean`\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("clean line errored: %v", g.Errs)
	}
	if g.Counts["self-review lines"] != 1 {
		t.Fatalf("line not counted: %v", g.Counts)
	}
}

func TestLedgerSelfReviewFixedWithReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=fixed(stale reference replaced) depth=clean scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("fixed(<reason>) errored: %v", g.Errs)
	}
}

func TestLedgerSelfReviewMissingKey(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean depth=clean scope=clean coverage=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "missing consistency") {
		t.Fatalf("missing key not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewBadVerdict(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clan depth=clean scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "does not parse") {
		t.Fatalf("bad verdict not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewAcceptedNeedsReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean depth=clean scope=accepted() coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "accepted names no reason") {
		t.Fatalf("empty accepted reason not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewCleanWithReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean(but) depth=clean scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "clean carries a reason") {
		t.Fatalf("clean(<reason>) not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewDuplicateKey(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean reality=clean depth=clean scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "states reality twice") {
		t.Fatalf("duplicate key not flagged: %v", g.Errs)
	}
}

func TestLedgerDecisionDates(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"DECISIONS.md": "# DECISIONS\n\n- 2026-08-29 user: chose Go.\n- 2026-13-40 user: impossible date.\n",
	})
	g := CheckLedger(d)
	if g.Counts["dated decision entries"] != 2 {
		t.Fatalf("dated entries not counted: %v", g.Counts)
	}
	if !hasFinding(g.Errs, "2026-13-40", "not a real calendar date") {
		t.Fatalf("bad date not flagged: %v", g.Errs)
	}
}

func TestLedgerAuthorProposedNote(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"DECISIONS.md": "# DECISIONS\n\n## Author-proposed, unconfirmed\n\n- guard x assumes single-tenant\n- enum value Archived\n\n## Later\n\n- 2026-08-29 user: fine.\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Notes, "2 author-proposed") {
		t.Fatalf("unconfirmed items not noted: %v", g.Notes)
	}
	if len(g.Errs) != 0 {
		t.Fatalf("notes must not error: %v", g.Errs)
	}
}

func TestLedgerHouseStyleEmDashAndEmoji(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"ARCHITECTURE.md":    "clean line\na line with an em dash — right here\n",
		"domain.modelith.md": "a rendered line with \U0001F600 an emoji\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Warns, "ARCHITECTURE.md:2", "em dash") {
		t.Fatalf("em dash not warned: %v", g.Warns)
	}
	if !hasFinding(g.Warns, "domain.modelith.md:1", "emoji") {
		t.Fatalf("emoji not warned: %v", g.Warns)
	}
	if len(g.Errs) != 0 {
		t.Fatalf("style findings must be warnings: %v", g.Errs)
	}
}

func TestLedgerHouseStyleSkipsGenerated(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"machines/Deal.oracle.md": "generated — with an em dash\n",
		"formal/Policy.als":       "generated — model\n",
	})
	g := CheckLedger(d)
	if len(g.Warns) != 0 {
		t.Fatalf("generated artifacts must be skipped: %v", g.Warns)
	}
}

func TestLedgerUnicodeSymbolsAreFine(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"ARCHITECTURE.md": "check ✓, arrow →, en dash –: all plain Unicode\n",
	})
	g := CheckLedger(d)
	if len(g.Warns) != 0 {
		t.Fatalf("plain Unicode must not warn: %v", g.Warns)
	}
}

func TestLedgerNoLedgersStillScansStyle(t *testing.T) {
	d := ledgerDesign(t, map[string]string{"ARCHITECTURE.md": "clean\n"})
	g := CheckLedger(d)
	if len(g.Errs) != 0 || len(g.Warns) != 0 {
		t.Fatalf("empty design must be quiet: errs=%v warns=%v", g.Errs, g.Warns)
	}
	if g.Counts["files style-scanned"] != 1 {
		t.Fatalf("style scan did not run: %v", g.Counts)
	}
	if LedgerActive(d) {
		t.Fatal("LedgerActive must be false with no ledgers")
	}
}
