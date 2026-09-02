package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// clauseDeclDesign writes a one-machine design whose named-unit table holds
// exactly the rows given (already pipe-delimited, without the header).
func clauseDeclDesign(t *testing.T, rows string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.MkdirAll(filepath.Join(d, "machines"), 0755); err != nil {
		t.Fatal(err)
	}
	oracle := "| T-DEAL-01 | DEAL-abc123 | a | b | c | d | - |\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.oracle.md"), []byte(oracle), 0644); err != nil {
		t.Fatal(err)
	}
	matrix := "| name | kind | signature | pre / post | maps to |\n|---|---|---|---|---|\n" + rows
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(matrix), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func completenessWarns(g *Gate) []string {
	var out []string
	for _, w := range g.Warns {
		if strings.Contains(w, "compound contract") || strings.Contains(w, "waives its clause declaration") {
			out = append(out, w)
		}
	}
	return out
}

func TestClauseCompleteness(t *testing.T) {
	cases := []struct {
		name    string
		row     string
		warns   bool
		wantSub string
	}{
		{
			name:    "conjunction with no declaration warns",
			row:     "| `guardCanAdvance` | guard | `(ctx,evt) -> bool` | true iff the stage moves forward AND the actor may write | inv `a-b` |\n",
			warns:   true,
			wantSub: "'and'",
		},
		{
			name:    "disjunction with no declaration warns",
			row:     "| `guardEitherLane` | guard | `(ctx,evt) -> bool` | true iff the lane is sse or durable | inv `a-b` |\n",
			warns:   true,
			wantSub: "'or'",
		},
		{
			name:    "any quantifier warns",
			row:     "| `guardAnyStale` | guard | `(ctx) -> bool` | true iff any pinned input version is stale | inv `a-b` |\n",
			warns:   true,
			wantSub: "'any'",
		},
		{
			name:    "either warns",
			row:     "| `guardEither` | guard | `(ctx) -> bool` | true iff either arm is armed | inv `a-b` |\n",
			warns:   true,
			wantSub: "'either'",
		},
		{
			name:    "all of warns",
			row:     "| `guardAllOf` | guard | `(ctx) -> bool` | true iff all of the parts resolved | inv `a-b` |\n",
			warns:   true,
			wantSub: "'all of'",
		},
		{
			name:    "one of warns",
			row:     "| `guardOneOf` | guard | `(ctx) -> bool` | true iff the member parses as one of the declared types | inv `a-b` |\n",
			warns:   true,
			wantSub: "'one of'",
		},
		{
			name:  "single-clause contract is silent",
			row:   "| `guardStageIsWon` | guard | `(ctx) -> bool` | true iff `ctx.pendingStage` equals that stage | inv `a-b` |\n",
			warns: false,
		},
		{
			name:  "declared vocabulary discharges the obligation",
			row:   "| `guardCanAdvance` | guard | `(ctx,evt) -> bool` | true iff the stage moves forward AND the actor may write CLAUSES{stage-forward, actor-writes} | inv `a-b` |\n",
			warns: false,
		},
		{
			name:  "single-clause waiver with a reason discharges it",
			row:   "| `guardWindowElapsed` | guard | `(ctx,evt) -> bool` | true iff `now` is at or after `validUntil` (single-clause: the 'or' is the comparison idiom, not a disjunction) | inv `a-b` |\n",
			warns: false,
		},
		{
			name:    "single-clause waiver with no reason warns",
			row:     "| `guardWindowElapsed` | guard | `(ctx,evt) -> bool` | true iff `now` is at or after `validUntil` (single-clause:) | inv `a-b` |\n",
			warns:   true,
			wantSub: "names no reason",
		},
		{
			name:  "a non-guard row is never read",
			row:   "| `saveDeal` | actor | `(input) -> row` | post: the row is written and the tx commits | C4 `a -> b` |\n",
			warns: false,
		},
		{
			name:  "trailing rationale prose is not the contract",
			row:   "| `guardConsumedAtBudget` | guard | `(ctx) -> bool` | true iff `consumed >= budgetLimit`. Unknown readings are disclosed alongside and the known total is compared | inv `a-b` |\n",
			warns: false,
		},
		{
			name:  "hyphenated wording is not a whole-token match",
			row:   "| `guardAndersonRule` | guard | `(ctx) -> bool` | true iff the at-or-after-anderson bound holds | inv `a-b` |\n",
			warns: false,
		},
		{
			name:    "the contract sentence is the one carrying iff",
			row:     "| `guardProjectionDirty` | guard | `(ctx,evt) -> bool` | ADDED 2026-09-01 by judgment. On `onError`: true iff the projection is dirty or lagging its watermark | inv `a-b` |\n",
			warns:   true,
			wantSub: "'or'",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := CheckIDCitations(clauseDeclDesign(t, tc.row))
			got := completenessWarns(g)
			if tc.warns && len(got) == 0 {
				t.Fatalf("expected a completeness warning, got none (all warns: %v)", g.Warns)
			}
			if !tc.warns && len(got) != 0 {
				t.Fatalf("expected no completeness warning, got %v", got)
			}
			if tc.warns && !strings.Contains(got[0], tc.wantSub) {
				t.Fatalf("warning %q does not name %q", got[0], tc.wantSub)
			}
			if tc.warns && !strings.Contains(got[0], "machines/Deal.matrix.md:3") {
				t.Fatalf("warning %q does not name the file and line", got[0])
			}
		})
	}
}

func TestClauseCompletenessCounts(t *testing.T) {
	rows := "| `guardA` | guard | s | true iff a and b | inv `a-b` |\n" +
		"| `guardB` | guard | s | true iff c CLAUSES{c} | inv `a-b` |\n" +
		"| `guardC` | guard | s | true iff d or e (single-clause: one comparison) | inv `a-b` |\n" +
		"| `actA` | action | s | post: x | - |\n"
	g := CheckIDCitations(clauseDeclDesign(t, rows))
	if got := g.Counts["guard rows read for clause completeness"]; got != 3 {
		t.Fatalf("guard rows read = %d, want 3", got)
	}
	if got := g.Counts["guard rows with a clause declaration"]; got != 1 {
		t.Fatalf("declared rows = %d, want 1", got)
	}
	if got := g.Counts["single-clause waivers"]; got != 1 {
		t.Fatalf("waivers = %d, want 1", got)
	}
}

func TestGuardContractStatement(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"first sentence when no iff", "post: x holds. Then y.", "post: x holds."},
		{"the iff sentence, not the first", "ADDED today. true iff a or b. More prose.", " true iff a or b."},
		{"a period inside backticks does not split", "true iff `v1.2` is pinned", "true iff `v1.2` is pinned"},
		{"empty cell", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := guardContractStatement(tc.in); got != tc.want {
				t.Fatalf("guardContractStatement(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNamedUnitCols(t *testing.T) {
	cases := []struct {
		name             string
		header           []string
		ok               bool
		wantN, wantK, wP int
	}{
		{"canonical header", []string{"name", "kind", "signature", "pre / post", "maps to"}, true, 0, 1, 3},
		{"contract-wrapped pre/post", []string{"name", "kind", "signature", "contract (pre / post)", "maps to", "test type"}, true, 0, 1, 3},
		{"a data row is not a header", []string{"`guardName`", "guard", "s", "true iff two kinds diverge", "inv `a-b`"}, false, -1, -1, -1},
		{"an unrelated table", []string{"failure", "detection", "transition"}, false, -1, -1, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n, k, p, ok := namedUnitCols(tc.header)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if n != tc.wantN || k != tc.wantK || p != tc.wP {
				t.Fatalf("cols = (%d,%d,%d), want (%d,%d,%d)", n, k, p, tc.wantN, tc.wantK, tc.wP)
			}
		})
	}
}

func TestClipStatement(t *testing.T) {
	long := strings.Repeat("x", 200)
	if got := clipStatement(long); len([]rune(got)) != 93 {
		t.Fatalf("clip length = %d, want 93", len([]rune(got)))
	}
	if got := clipStatement("  a   b\nc  "); got != "a b c" {
		t.Fatalf("clip whitespace = %q", got)
	}
}

func TestSplitTableRow(t *testing.T) {
	got := splitTableRow("| a | b | c |")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("splitTableRow = %#v", got)
	}
	if got := splitTableRow("| a |"); len(got) != 1 || got[0] != "a" {
		t.Fatalf("single cell = %#v", got)
	}
}
