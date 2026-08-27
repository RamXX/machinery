package gates

import (
	"path/filepath"
	"strings"
	"testing"
)

// srcDoc is the source artifact every embed fixture copies from.
const srcDoc = `# Source

## Events

| event | producer | consumer | delivery |
|---|---|---|---|
| request | orders | payments | at-least-once |
| markPaid | payments | orders | at-least-once |
| settle | payments | ledger | at-most-once |
`

// embedFixture writes a source document plus an embedding document carrying
// the given marker and table, and runs Ge over the design.
func embedFixture(t *testing.T, marker, table string) *Gate {
	t.Helper()
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "SOURCE.md"), srcDoc)
	body := "# Shard\n\n" + marker + "\n\n" + table + "\n"
	mustWrite(t, filepath.Join(design, "SHARD.md"), body)
	return CheckEmbeds(design)
}

const goodMarker = `<!-- machinery:embed from="SOURCE.md" table="event,producer,consumer,delivery" claims="subset,complete" -->`

const allRows = `| event | producer | consumer | delivery |
|---|---|---|---|
| request | orders | payments | at-least-once |
| markPaid | payments | orders | at-least-once |
| settle | payments | ledger | at-most-once |`

func TestEmbedFidelityHappyPath(t *testing.T) {
	g := embedFixture(t, goodMarker, allRows)
	if len(g.Errs) != 0 {
		t.Fatalf("a faithful embed must pass: %v", g.Errs)
	}
	if g.Counts["embed tables verified"] != 1 || g.Counts["rows compared"] != 3 {
		t.Errorf("counts = %+v, want 1 table and 3 rows compared", g.Counts)
	}
	if g.Counts["subset claims"] != 1 || g.Counts["complete claims"] != 1 {
		t.Errorf("claims not counted: %+v", g.Counts)
	}
}

// The defect this gate exists for: a copied row edited on one side only.
func TestEmbedSubsetCatchesAnEditedRow(t *testing.T) {
	rows := strings.Replace(allRows, "| markPaid | payments | orders | at-least-once |",
		"| markPaid | payments | orders | exactly-once |", 1)
	g := embedFixture(t, goodMarker, rows)
	if !hasErr(g, "row 'markPaid' is not a byte-identical copy") {
		t.Fatalf("an edited copy must fail subset: %v", g.Errs)
	}
}

// The other half: a source row that never made it into the copy.
func TestEmbedCompleteCatchesAMissingRow(t *testing.T) {
	rows := strings.Replace(allRows, "\n| settle | payments | ledger | at-most-once |", "", 1)
	g := embedFixture(t, goodMarker, rows)
	if !hasErr(g, "source row 'settle' is selected but absent here") {
		t.Fatalf("a dropped source row must fail complete: %v", g.Errs)
	}
}

// The two claims are independent: subset alone tolerates a partial copy.
func TestEmbedSubsetAloneToleratesAPartialCopy(t *testing.T) {
	marker := strings.Replace(goodMarker, `claims="subset,complete"`, `claims="subset"`, 1)
	rows := strings.Replace(allRows, "\n| settle | payments | ledger | at-most-once |", "", 1)
	g := embedFixture(t, marker, rows)
	if len(g.Errs) != 0 {
		t.Fatalf("subset alone must tolerate a narrower copy: %v", g.Errs)
	}
	if g.Counts["complete claims"] != 0 {
		t.Errorf("complete must not be counted when it is not claimed: %+v", g.Counts)
	}
}

// complete alone says nothing about rows the shard adds.
func TestEmbedCompleteAloneToleratesAnExtraRow(t *testing.T) {
	marker := strings.Replace(goodMarker, `claims="subset,complete"`, `claims="complete"`, 1)
	g := embedFixture(t, marker, allRows+"\n| local | shard | shard | n/a |")
	if len(g.Errs) != 0 {
		t.Fatalf("complete alone must ignore extra rows: %v", g.Errs)
	}
}

func TestEmbedWhereFilter(t *testing.T) {
	t.Run("narrows the source set", func(t *testing.T) {
		marker := strings.Replace(goodMarker, `claims=`, `where="producer|consumer=orders" claims=`, 1)
		rows := strings.Replace(allRows, "\n| settle | payments | ledger | at-most-once |", "", 1)
		g := embedFixture(t, marker, rows)
		if len(g.Errs) != 0 {
			t.Fatalf("the filter must exclude the ledger row: %v", g.Errs)
		}
	})
	t.Run("an unknown column is an error", func(t *testing.T) {
		marker := strings.Replace(goodMarker, `claims=`, `where="nosuch=orders" claims=`, 1)
		g := embedFixture(t, marker, allRows)
		if !hasErr(g, "where= names column 'nosuch'") {
			t.Fatalf("an unknown filter column must fail: %v", g.Errs)
		}
	})
	t.Run("a malformed filter is an error", func(t *testing.T) {
		marker := strings.Replace(goodMarker, `claims=`, `where="producer" claims=`, 1)
		g := embedFixture(t, marker, allRows)
		if !hasErr(g, "is not of the form '<column>=<token>'") {
			t.Fatalf("a malformed filter must fail: %v", g.Errs)
		}
	})
	t.Run("matching is whole-token", func(t *testing.T) {
		// 'order' must not match the producer 'orders'
		marker := strings.Replace(goodMarker, `claims="subset,complete"`, `where="producer=order" claims="complete"`, 1)
		g := embedFixture(t, marker, allRows)
		if len(g.Errs) != 0 {
			t.Fatalf("a partial token selects no rows, so complete is vacuous here: %v", g.Errs)
		}
	})
}

func TestEmbedLocalization(t *testing.T) {
	t.Run("a first-cell marker exempts the whole row", func(t *testing.T) {
		rows := allRows + "\n| shardOnly (shard-local: this shard adds a local retry event) | x | y | z |"
		g := embedFixture(t, goodMarker, rows)
		if len(g.Errs) != 0 {
			t.Fatalf("a row-localized addition must be exempt: %v", g.Errs)
		}
		if g.Counts["rows localized"] != 1 {
			t.Errorf("rows localized = %d, want 1: %+v", g.Counts["rows localized"], g.Counts)
		}
	})
	t.Run("a later-cell marker exempts only that cell", func(t *testing.T) {
		rows := strings.Replace(allRows, "| markPaid | payments | orders | at-least-once |",
			"| markPaid | payments | orders | (shard-local: this shard consumes it exactly once) |", 1)
		g := embedFixture(t, goodMarker, rows)
		if len(g.Errs) != 0 {
			t.Fatalf("a cell-localized row must still match on its other cells: %v", g.Errs)
		}
		if g.Counts["cells localized"] != 1 {
			t.Errorf("cells localized = %d, want 1: %+v", g.Counts["cells localized"], g.Counts)
		}
	})
	t.Run("a cell exemption does not excuse the rest of the row", func(t *testing.T) {
		rows := strings.Replace(allRows, "| markPaid | payments | orders | at-least-once |",
			"| markPaid | LEDGER | orders | (shard-local: consumed once) |", 1)
		g := embedFixture(t, goodMarker, rows)
		if !hasErr(g, "row 'markPaid' is not a byte-identical copy") {
			t.Fatalf("only the named cell is exempt: %v", g.Errs)
		}
	})
	t.Run("a localization with no reason is an error", func(t *testing.T) {
		rows := allRows + "\n| shardOnly (shard-local:) | x | y | z |"
		g := embedFixture(t, goodMarker, rows)
		if !hasErr(g, "names no reason") {
			t.Fatalf("an empty localization reason must fail: %v", g.Errs)
		}
	})
}

func TestEmbedMarkerGrammar(t *testing.T) {
	cases := []struct {
		name, marker, want string
	}{
		{"missing from", `<!-- machinery:embed table="event,producer" claims="subset" -->`,
			"missing required attribute(s): from"},
		{"missing table", `<!-- machinery:embed from="SOURCE.md" claims="subset" -->`,
			"missing required attribute(s): table"},
		{"missing claims", `<!-- machinery:embed from="SOURCE.md" table="event,producer" -->`,
			"missing required attribute(s): claims"},
		{"unknown attribute", `<!-- machinery:embed from="SOURCE.md" table="event,producer" claims="subset" rows="all" -->`,
			"unknown attribute 'rows'"},
		{"unknown claim", `<!-- machinery:embed from="SOURCE.md" table="event,producer" claims="identical" -->`,
			"unknown claim 'identical'"},
		{"empty claims list", `<!-- machinery:embed from="SOURCE.md" table="event,producer" claims="," -->`,
			"claims names neither subset nor complete"},
		{"missing source", `<!-- machinery:embed from="NOPE.md" table="event,producer" claims="subset" -->`,
			"does not exist or is empty"},
		{"selector matches nothing", `<!-- machinery:embed from="SOURCE.md" table="event,ghost" claims="subset" -->`,
			"has all the columns"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := embedFixture(t, tc.marker, allRows)
			if !hasErr(g, tc.want) {
				t.Fatalf("want an error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

// A selector that matches two tables resolves to neither: the gate refuses to
// guess which one the author meant.
func TestEmbedAmbiguousSelectorIsError(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "SOURCE.md"), srcDoc+"\n## More events\n\n"+
		"| event | producer | consumer | delivery |\n|---|---|---|---|\n| other | a | b | at-least-once |\n")
	mustWrite(t, filepath.Join(design, "SHARD.md"), "# Shard\n\n"+goodMarker+"\n\n"+allRows+"\n")
	g := CheckEmbeds(design)
	if !hasErr(g, "2 tables in 'SOURCE.md' have the columns") {
		t.Fatalf("an ambiguous selector must fail: %v", g.Errs)
	}
}

// An embed carries the source's columns unchanged: comparing rows across
// different column sets would compare nothing.
func TestEmbedHeaderMustMatch(t *testing.T) {
	t.Run("different column count", func(t *testing.T) {
		g := embedFixture(t, goodMarker,
			"| event | producer | consumer |\n|---|---|---|\n| request | orders | payments |")
		if !hasErr(g, "an embed carries the source's columns unchanged") {
			t.Fatalf("a narrower embed table must fail: %v", g.Errs)
		}
	})
	t.Run("renamed column", func(t *testing.T) {
		g := embedFixture(t, goodMarker,
			strings.Replace(allRows, "| event | producer | consumer | delivery |",
				"| event | producer | consumer | guarantee |", 1))
		if !hasErr(g, "an embed carries the source's columns unchanged") {
			t.Fatalf("a renamed column must fail: %v", g.Errs)
		}
	})
}

// A marker that marks nothing is a promise again, so it fails loudly.
func TestEmbedMarkerWithNoTableIsError(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "SOURCE.md"), srcDoc)
	mustWrite(t, filepath.Join(design, "SHARD.md"),
		"# Shard\n\n"+goodMarker+"\n\nProse, and then no table at all.\n")
	g := CheckEmbeds(design)
	if !hasErr(g, "marks no table") {
		t.Fatalf("a marker with no table must fail: %v", g.Errs)
	}
}

// Adoption is opt-in: a design with no markers does not activate the gate,
// and forcing it there checks nothing, which is an error like every other
// empty check in the suite.
func TestEmbedActivationIsOptIn(t *testing.T) {
	design := t.TempDir()
	mustWrite(t, filepath.Join(design, "PLAIN.md"), "# Plain\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	if EmbedActive(design) {
		t.Fatal("a design with no marker must not activate Ge")
	}
	g := CheckEmbeds(design)
	if !hasErr(g, "no embed markers found") {
		t.Fatalf("forcing Ge with no markers must fail: %v", g.Errs)
	}
	mustWrite(t, filepath.Join(design, "SOURCE.md"), srcDoc)
	mustWrite(t, filepath.Join(design, "SHARD.md"), "# S\n\n"+goodMarker+"\n\n"+allRows+"\n")
	if !EmbedActive(design) {
		t.Fatal("a design with a marker must activate Ge")
	}
}

// The source may live outside the design tree (a child embedding its parent's
// table is the motivating case).
func TestEmbedSourceMayBeOutsideTheDesign(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "parent", "ARCHITECTURE.md"), srcDoc)
	design := filepath.Join(root, "child", "design")
	marker := strings.Replace(goodMarker, `from="SOURCE.md"`, `from="../../parent/ARCHITECTURE.md"`, 1)
	mustWrite(t, filepath.Join(design, "ARCHITECTURE.md"), "# Child\n\n"+marker+"\n\n"+allRows+"\n")
	g := CheckEmbeds(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a relative source outside the design must resolve: %v", g.Errs)
	}
}
