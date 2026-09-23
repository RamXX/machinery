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

func TestLedgerSelfReviewTableCellTrailingPipe(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "| 3 | done | `self-review: reality=clean depth=clean scope=clean coverage=clean consistency=clean` |\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("table-cell line errored: %v", g.Errs)
	}
	if g.Counts["self-review lines"] != 1 {
		t.Fatalf("line not counted: %v", g.Counts)
	}
}

func TestLedgerSelfReviewNestedParenReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean depth=accepted(deferred (see M6); revisit after split (Oct)) scope=clean coverage=fixed(re-derived (twice)) consistency=clean\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("nested-paren reason errored: %v", g.Errs)
	}
}

func TestLedgerSelfReviewNestedParenReasonInTableCell(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "| 5 | `self-review: reality=clean depth=clean scope=clean coverage=clean consistency=accepted(narrowed (admin only); disclosure stands)` |\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("nested-paren reason in table cell errored: %v", g.Errs)
	}
}

func TestLedgerSelfReviewBadKeyInTableCell(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "| 2 | `self-review: realty=clean depth=clean scope=clean coverage=clean consistency=clean` |\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "does not parse") {
		t.Fatalf("bad key in table cell not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewCleanWithNestedParenReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean(fine (really)) depth=clean scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "clean carries a reason") {
		t.Fatalf("clean with nested-paren reason not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewMissingKeyInTableCell(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "| 4 | `self-review: reality=clean depth=clean scope=clean coverage=clean` |\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "missing consistency") {
		t.Fatalf("missing key in table cell not flagged: %v", g.Errs)
	}
}

func TestLedgerSelfReviewUnbalancedParenReason(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"STATE.md": "self-review: reality=clean depth=fixed(open (never closed scope=clean coverage=clean consistency=clean\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "does not parse") {
		t.Fatalf("unbalanced paren not flagged: %v", g.Errs)
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

// The modelith render is generated by a renderer that emits em dashes, and the
// strip is a mechanical post-processing step. A surviving em dash there is a
// skipped step, so Gl errors instead of warning.
func TestLedgerModelithRenderEmDashErrors(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"domain.modelith.md": "clean line\na rendered line with an em dash — right here\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "domain.modelith.md:2", "em dash") {
		t.Fatalf("em dash in the modelith render must error: errs=%v warns=%v", g.Errs, g.Warns)
	}
	if !hasFinding(g.Errs, "perl -CSD -i -pe") {
		t.Fatalf("the error must name the mechanical fix: %v", g.Errs)
	}
	if hasFinding(g.Warns, "em dash") {
		t.Fatalf("the render em dash must not also warn: %v", g.Warns)
	}
}

// The promotion is keyed on the render suffix, not on the directory: a legacy
// or per-context render is the same generated artifact.
func TestLedgerNestedModelithRenderEmDashErrors(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"legacy/domain.modelith.md": "a legacy render with an em dash — here\n",
	})
	g := CheckLedger(d)
	if !hasFinding(g.Errs, "legacy/domain.modelith.md:1", "em dash") {
		t.Fatalf("nested render em dash must error: errs=%v warns=%v", g.Errs, g.Warns)
	}
}

// The promotion is scoped to em dashes in the render. Emojis there, and em
// dashes in every hand-written file, stay at the warn tier.
func TestLedgerHouseStyleWarnTierUnchanged(t *testing.T) {
	d := ledgerDesign(t, map[string]string{
		"domain.modelith.md":   "a rendered line with \U0001F600 an emoji\n",
		"ARCHITECTURE.md":      "a hand-written line with an em dash — here\n",
		"notes-modelith.md":    "not a render, just a similar name — em dash here\n",
		"STATE.md":             "- Phase 1: `self-review: reality=clean depth=clean scope=clean coverage=clean consistency=clean`\n",
		"machines/Deal.md":     "a hand-written machine note — em dash here\n",
		"domain.modelith.yaml": "description: the source model — em dash here\n",
	})
	g := CheckLedger(d)
	if len(g.Errs) != 0 {
		t.Fatalf("only the render em dash is an error: %v", g.Errs)
	}
	for _, want := range []string{"domain.modelith.md:1", "ARCHITECTURE.md:1", "notes-modelith.md:1", "machines/Deal.md:1", "domain.modelith.yaml:1"} {
		if !hasFinding(g.Warns, want) {
			t.Fatalf("missing warn for %s: %v", want, g.Warns)
		}
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

func TestLedgerNearMissMarkerWarns(t *testing.T) {
	// a marker-shaped comment that matches no known marker grammar arms
	// nothing, silently: the tier it meant to arm never runs
	d := ledgerDesign(t, map[string]string{"ARCHITECTURE.md": "# A\n" +
		"<!-- machinery: embed from=\"m.md\" -->\n" + // near miss: space after the colon
		"<!-- machinery:read-complete -->\n" + // near miss: misspelled marker
		"<!-- machinery:reads-complete -->\n" + // valid
		"<!-- machinery:embed from=\"m.md\" table=\"a\" claims=\"subset\" -->\n" + // valid
		"<!-- plain prose comment -->\n"})
	g := CheckLedger(d)
	var hits int
	for _, w := range g.Warns {
		if strings.Contains(w, "matches no known machinery marker grammar") {
			hits++
		}
	}
	if hits != 2 {
		t.Fatalf("expected exactly the two near-miss markers to warn, got %d: %v", hits, g.Warns)
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

// The same table in two hand-written documents keeps the marker advice: an
// author can mark a copy they made.
func TestLedgerDuplicateTableHandWrittenAdvice(t *testing.T) {
	table := "| a | b |\n|---|---|\n| 1 | 2 |\n"
	d := ledgerDesign(t, map[string]string{
		"ARCHITECTURE.md": "# A\n\n" + table,
		"BUILD.md":        "# B\n\n" + table,
	})
	g := CheckLedger(d)
	if !hasFinding(g.Warns, "the same table appears at", "no machinery:embed marker on the copy") {
		t.Fatalf("hand-written duplicate lost its warning: %v", g.Warns)
	}
	if hasFinding(g.Warns, "modelith YAML source") {
		t.Fatalf("hand-written duplicate must not get the generated-render advice: %v", g.Warns)
	}
}

// Both copies inside a generated *.modelith.md render: telling the author to
// mark it would be telling them to hand-edit a generated file. The advice
// points at the YAML source instead, and the tier stays warn.
func TestLedgerDuplicateTableInGeneratedRenderPointsAtSource(t *testing.T) {
	table := "| attribute | type |\n|---|---|\n| id | uuid |\n"
	d := ledgerDesign(t, map[string]string{
		"domain.modelith.md": "# Domain\n\n## Widget\n\n" + table + "\n## Gadget\n\n" + table,
	})
	g := CheckLedger(d)
	if !hasFinding(g.Warns, "two entities rendering byte-identical tables", "modelith YAML source", "then re-render") {
		t.Fatalf("generated-render duplicate did not get the source advice: %v", g.Warns)
	}
	if hasFinding(g.Warns, "no machinery:embed marker on the copy") {
		t.Fatalf("generated-render duplicate must not be told to mark it: %v", g.Warns)
	}
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("the finding must stay a warning: errs=%v drift=%v", g.Errs, g.Drift)
	}
}

// One copy in a generated render and one hand-written is still a copy someone
// made: the marker advice applies.
func TestLedgerDuplicateTableMixedKeepsMarkerAdvice(t *testing.T) {
	table := "| attribute | type |\n|---|---|\n| id | uuid |\n"
	d := ledgerDesign(t, map[string]string{
		"domain.modelith.md": "# Domain\n\n## Widget\n\n" + table,
		"BUILD.md":           "# B\n\n" + table,
	})
	g := CheckLedger(d)
	if !hasFinding(g.Warns, "no machinery:embed marker on the copy") {
		t.Fatalf("mixed duplicate lost the marker advice: %v", g.Warns)
	}
}

// undeclaredFactDesign writes a design with one machine (Order, states
// Placed, saving, Paid) and the given matrix body.
func undeclaredFactDesign(t *testing.T, matrix string) string {
	t.Helper()
	design := t.TempDir()
	if err := os.MkdirAll(filepath.Join(design, "machines"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(design, "machines", "Order.machine.json"), wiringMachine)
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), matrix)
	return design
}

func undeclaredFactWarns(g *Gate) []string {
	var out []string
	for _, w := range g.Warns {
		if strings.Contains(w, "undeclared fact reference") {
			out = append(out, w)
		}
	}
	return out
}

func TestLedgerUndeclaredFactReference(t *testing.T) {
	const header = "| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n"
	row := func(name, contract, mapsTo string) string {
		return "| `" + name + "` | action | `(ctx) -> ctx` | " + contract + " | " + mapsTo + " |\n"
	}
	t.Run("positive: a bare backticked fact in a contract cell", func(t *testing.T) {
		g := CheckLedger(undeclaredFactDesign(t, "# Order\n\n"+header+row("persistOrder", "stores `Order.total` and `line_item_count`", "-")))
		got := undeclaredFactWarns(g)
		want := []string{
			"machines/Order.matrix.md:5: row 'persistOrder': undeclared fact reference `Order.total`; declare it in USES{} or WRITES{} or drop the backticks",
			"machines/Order.matrix.md:5: row 'persistOrder': undeclared fact reference `line_item_count`; declare it in USES{} or WRITES{} or drop the backticks",
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("got %q", got)
		}
		if len(g.Errs) != 0 {
			t.Fatalf("the tier is a warning, never an error: %v", g.Errs)
		}
	})
	t.Run("positive: a payload reads cell", func(t *testing.T) {
		g := CheckLedger(undeclaredFactDesign(t, "# Order\n\n| event | reacting unit | payload reads |\n|---|---|---|\n| `paid` | `markPaid` | reads `payment_ref` |\n"))
		if got := undeclaredFactWarns(g); len(got) != 1 || !strings.Contains(got[0], "row 'paid': undeclared fact reference `payment_ref`") {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("positive: a declaration on another row does not declare this one", func(t *testing.T) {
		g := CheckLedger(undeclaredFactDesign(t, "# Order\n\n"+header+row("persistOrder", "WRITES{Order.total}", "-")+row("guardTotal", "true iff `Order.total` > 0", "-")))
		if got := undeclaredFactWarns(g); len(got) != 1 || !strings.Contains(got[0], "row 'guardTotal'") {
			t.Fatalf("got %q", got)
		}
	})
	negatives := []struct{ name, rows string }{
		{"declared in USES on the same row", row("guardTotal", "true iff `Order.total` > 0. USES{Order.total}", "-")},
		{"declared in WRITES on the same row", row("persistOrder", "stores `Order.total`. WRITES{Order.total}", "-")},
		{"inside a group", row("persistOrder", "USES{`Order.total`} CLAUSES{`a_b`} VALUES{`x_y`} READS{`c_d`}", "-")},
		{"inside a payload group", row("emitPaid", "payload {`Order.id`, `order_ref`}", "-")},
		{"inside and named by a derived form", row("scoreOrder", "computes `risk_score`; derived: risk_score (from `line_total` at read time, never stored)", "-")},
		{"the row's own unit name", row("emit_order", "the `emit_order` step", "-")},
		{"a machine state", row("persistOrder", "entered from `Order.saving`", "-")},
		{"not fact-shaped", row("persistOrder", "`Order`, `ctx.retries`, `totalCents`, `order-paid-final`, `ORDER_ID`, `Order.Paid`, `a.b.c`", "-")},
		{"outside the contract, clause and payload cells", row("persistOrder", "persists", "`Order.total`, `line_item_count`")},
		{"fenced example", "```\n| `persistOrder` | action | `(ctx) -> ctx` | `Order.total` | - |\n```\n"},
	}
	for _, tc := range negatives {
		t.Run("negative: "+tc.name, func(t *testing.T) {
			g := CheckLedger(undeclaredFactDesign(t, "# Order\n\n"+header+tc.rows))
			if got := undeclaredFactWarns(g); len(got) != 0 {
				t.Fatalf("false positive: %q", got)
			}
		})
	}
}

// An unnamed VALUES{...} group binds by its unit's name, exactly. A unit name
// equal to an enum name only when case is ignored binds nothing, so Gl warns
// and tells the author to name the group; an exact unit name, a named group,
// and a unit name matching no enum in any case stay silent.
func TestValuesEnumCaseMismatchWarns(t *testing.T) {
	model := "kind: DomainModel\nversion: v1\nenums:\n  OrderState:\n    values: [{name: Open}, {name: Closed}]\n" +
		"entities:\n  Order:\n    attributes: [{name: state, type: OrderState}]\n"
	matrix := "| name | kind | contract (pre / post) |\n|---|---|---|\n" +
		"| `orderState` | guard | VALUES{Open, Closed} |\n" +
		"| `OrderState` | guard | VALUES{Open, Closed} |\n" +
		"| `orderstate` | guard | VALUES OrderState{Open, Closed} |\n" +
		"| `reason` | guard | VALUES{late, early} |\n"
	d := ledgerDesign(t, map[string]string{"domain.modelith.yaml": model, "machines/Order.matrix.md": matrix})
	g := CheckLedger(d)
	var hits []string
	for _, w := range g.Warns {
		if strings.Contains(w, "VALUES") {
			hits = append(hits, w)
		}
	}
	want := "machines/Order.matrix.md:3: row 'orderState': unnamed VALUES{...} takes the unit name 'orderState', which differs from enum 'OrderState' only in case and so binds no enum; name the group (VALUES OrderState{...}) to bind it"
	if len(hits) != 1 || hits[0] != want {
		t.Fatalf("want exactly the orderState warning, got %v", hits)
	}
	if len(g.Errs) != 0 {
		t.Fatalf("the tier is a warning, never an error: %v", g.Errs)
	}
}
