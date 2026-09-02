package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/ir"
)

// refreshDesign writes a two-document design: a source and one embedding
// document whose marker and table are given verbatim.
func refreshDesign(t *testing.T, source, embedding string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "ARCHITECTURE.md"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, "SHARD.md"), []byte(embedding), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func shardText(t *testing.T, design string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(design, "SHARD.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const keyedSource = "# root\n\n" +
	"| name | note |\n|---|---|\n" +
	"| `alpha` | the alpha note, current |\n" +
	"| `beta` | the beta note, current |\n"

const eventSource = "# root\n\n" +
	"| event | producer | consumer | payload |\n|---|---|---|---|\n" +
	"| `job.done` | ops | core (sse lane) | id |\n" +
	"| `job.done` | ops | core (durable lane) | id + hash |\n" +
	"| `job.done` | ops | risk (durable lane) | id |\n"

func TestRefreshEmbeds(t *testing.T) {
	cases := []struct {
		name        string
		source      string
		embedding   string
		wantIn      []string
		wantNotIn   []string
		recopied    int
		appended    int
		orphans     int
		kept        int
		wantProblem string
	}{
		{
			name:      "a drifted row is re-copied from its source",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | the alpha note, STALE |\n",
			wantIn:    []string{"| `alpha` | the alpha note, current |"},
			wantNotIn: []string{"STALE"},
			recopied:  1,
		},
		{
			name:      "an already-current row is left byte-identical",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | the alpha note, current |\n",
			wantIn:    []string{"| `alpha` | the alpha note, current |"},
			recopied:  0,
		},
		{
			name:      "a localized row is never touched",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` (shard-local: this shard restates it) | wholly local |\n",
			wantIn:    []string{"| `alpha` (shard-local: this shard restates it) | wholly local |"},
			kept:      1,
			recopied:  0,
		},
		{
			name:      "a row with no source row is kept and reported",
			source:    keyedSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `gamma` | renamed away |\n",
			wantIn:    []string{"| `gamma` | renamed away |"},
			orphans:   1,
		},
		{
			name:      "complete plus where appends the missing selected rows",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset,complete\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | id |\n",
			wantIn:    []string{"| `job.done` | ops | core (durable lane) | id + hash |"},
			wantNotIn: []string{"risk (durable lane)"},
			appended:  1,
		},
		{
			name:      "subset alone never appends",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | id |\n",
			wantNotIn: []string{"durable lane"},
			appended:  0,
		},
		{
			name:      "event lanes sharing an event and a consumer stay distinct",
			source:    eventSource,
			embedding: "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" claims=\"subset\" -->\n| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (durable lane) | STALE |\n| `job.done` | ops | core (sse lane) | id |\n",
			wantIn:    []string{"| `job.done` | ops | core (durable lane) | id + hash |", "| `job.done` | ops | core (sse lane) | id |"},
			wantNotIn: []string{"STALE"},
			recopied:  1,
		},
		{
			name:        "a header that is not the source's is refused, not rewritten",
			source:      keyedSource,
			embedding:   "<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note | extra |\n|---|---|---|\n| `alpha` | x | y |\n",
			wantIn:      []string{"| `alpha` | x | y |"},
			wantProblem: "does not carry the source's columns",
		},
		{
			name:        "a missing source is refused, not rewritten",
			source:      keyedSource,
			embedding:   "<!-- machinery:embed from=\"NOPE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | x |\n",
			wantIn:      []string{"| `alpha` | x |"},
			wantProblem: "does not exist or is empty",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := refreshDesign(t, tc.source, tc.embedding)
			reports, changed, err := RefreshEmbeds(d, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(reports) != 1 {
				t.Fatalf("reports = %d, want 1", len(reports))
			}
			r := reports[0]
			if tc.wantProblem != "" {
				if !strings.Contains(r.Problem, tc.wantProblem) {
					t.Fatalf("problem = %q, want it to name %q", r.Problem, tc.wantProblem)
				}
				if len(changed) != 0 {
					t.Fatalf("a refused marker changed files: %v", changed)
				}
			} else if r.Problem != "" {
				t.Fatalf("unexpected problem: %s", r.Problem)
			}
			if r.Recopied != tc.recopied || r.Appended != tc.appended || r.Kept != tc.kept || len(r.Orphans) != tc.orphans {
				t.Fatalf("report = (recopied %d, appended %d, kept %d, orphans %d), want (%d, %d, %d, %d)",
					r.Recopied, r.Appended, r.Kept, len(r.Orphans), tc.recopied, tc.appended, tc.kept, tc.orphans)
			}
			got := shardText(t, d)
			for _, want := range tc.wantIn {
				if !strings.Contains(got, want) {
					t.Fatalf("result does not carry %q:\n%s", want, got)
				}
			}
			for _, no := range tc.wantNotIn {
				if strings.Contains(got, no) {
					t.Fatalf("result still carries %q:\n%s", no, got)
				}
			}
		})
	}
}

func TestRefreshIsIdempotent(t *testing.T) {
	d := refreshDesign(t, eventSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"event,producer,consumer,payload\" where=\"consumer=core\" claims=\"subset,complete\" -->\n"+
			"| event | producer | consumer | payload |\n|---|---|---|---|\n| `job.done` | ops | core (sse lane) | STALE |\n")
	if _, changed, err := RefreshEmbeds(d, false); err != nil || len(changed) != 1 {
		t.Fatalf("first run: changed %v, err %v", changed, err)
	}
	first := shardText(t, d)
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("second run changed %v; the refresh is not idempotent", changed)
	}
	if reports[0].Recopied != 0 || reports[0].Appended != 0 {
		t.Fatalf("second run reported work: %+v", reports[0])
	}
	if got := shardText(t, d); got != first {
		t.Fatalf("second run rewrote the document:\n%s\n---\n%s", first, got)
	}
}

func TestRefreshDryRunWritesNothing(t *testing.T) {
	d := refreshDesign(t, keyedSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n| name | note |\n|---|---|\n| `alpha` | STALE |\n")
	before := shardText(t, d)
	reports, changed, err := RefreshEmbeds(d, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 1 {
		t.Fatalf("dry run reported %d changed files, want 1", len(changed))
	}
	if reports[0].Recopied != 1 {
		t.Fatalf("dry run reported %d re-copied rows, want 1", reports[0].Recopied)
	}
	if got := shardText(t, d); got != before {
		t.Fatalf("dry run wrote the file")
	}
}

func TestRefreshLeavesProseAndOtherTablesAlone(t *testing.T) {
	embedding := "# shard\n\nSome prose.\n\n| unrelated | table |\n|---|---|\n| a | b |\n\n" +
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n\n" +
		"| name | note |\n|---|---|\n| `alpha` | STALE |\n\nTrailing prose.\n"
	d := refreshDesign(t, keyedSource, embedding)
	if _, _, err := RefreshEmbeds(d, false); err != nil {
		t.Fatal(err)
	}
	got := shardText(t, d)
	for _, want := range []string{"Some prose.", "| unrelated | table |", "| a | b |", "Trailing prose.", "| `alpha` | the alpha note, current |"} {
		if !strings.Contains(got, want) {
			t.Fatalf("result lost %q:\n%s", want, got)
		}
	}
}

func TestEmbedRowKey(t *testing.T) {
	cases := []struct {
		name  string
		cells []string
		cols  []string
		want  string
	}{
		{"backticked first cell", []string{"`alpha`", "x"}, []string{"name", "note"}, "alpha"},
		{"bare first cell", []string{"alpha thing", "x"}, []string{"name", "note"}, "alpha"},
		{"event table keys on three cells", []string{"`e`", "ops", "core (sse)", "p"}, []string{"event", "producer", "consumer", "payload"},
			"e\x00ops\x00core\x00core (sse)"},
		{"empty row", nil, []string{"name"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := embedRowKey(tc.cells, tc.cols); got != tc.want {
				t.Fatalf("embedRowKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLeadToken(t *testing.T) {
	cases := []struct{ in, want string }{
		{"`alpha`", "alpha"},
		{"  `alpha.beta` (note)", "alpha.beta"},
		{"plain-token rest", "plain-token"},
		{"(only a parenthetical)", "(only a parenthetical)"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := leadToken(tc.in); got != tc.want {
			t.Fatalf("leadToken(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSeparatorRow(t *testing.T) {
	for _, in := range []string{"|---|---|", "| --- | :---: |"} {
		if !isSeparatorRow(in) {
			t.Fatalf("%q is a separator row", in)
		}
	}
	for _, in := range []string{"| a | b |", "not a table"} {
		if isSeparatorRow(in) {
			t.Fatalf("%q is not a separator row", in)
		}
	}
}

func TestFilterRowIdxMatchesFilterRows(t *testing.T) {
	tbls := ir.ParseMdTables(eventSource)
	if len(tbls) != 1 {
		t.Fatalf("fixture parsed to %d tables", len(tbls))
	}
	tbl := tbls[0]
	idx, err := filterRowIdx(tbl, "consumer=core")
	if err != "" {
		t.Fatal(err)
	}
	rows, err := filterRows(tbl, "consumer=core")
	if err != "" {
		t.Fatal(err)
	}
	if len(idx) != len(rows) {
		t.Fatalf("index count %d != row count %d", len(idx), len(rows))
	}
	for i := range idx {
		if tbl.Rows[idx[i]][0] != rows[i][0] {
			t.Fatalf("index %d does not address row %d", idx[i], i)
		}
	}
}

const collidingSource = "# root\n\n" +
	"| failure | detection | recovery |\n|---|---|---|\n" +
	"| duplicate `request` redelivery | dedupe by orderId | drop |\n" +
	"| duplicate `capture` redelivery | dedupe by id | drop |\n"

func TestRefreshRefusesAmbiguousKeys(t *testing.T) {
	// Both source rows lead with the bare token "duplicate", so the key does
	// not separate them. A drifted copy of one must NOT be rewritten with the
	// other's text; it is reported instead.
	d := refreshDesign(t, collidingSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"failure,detection,recovery\" claims=\"subset\" -->\n"+
			"| failure | detection | recovery |\n|---|---|---|\n| duplicate `capture` redelivery | dedupe by id | STALE |\n")
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 {
		t.Fatalf("an ambiguous key rewrote the document: %v", changed)
	}
	if len(reports[0].Ambiguous) != 1 {
		t.Fatalf("ambiguity not reported: %+v", reports[0])
	}
	if got := shardText(t, d); !strings.Contains(got, "STALE") {
		t.Fatalf("the row was rewritten from a colliding source row:\n%s", got)
	}
}

func TestRefreshLeavesADuplicatedCopyAlone(t *testing.T) {
	// A copy that carries the same source row twice is a de-duplication
	// judgment, not a re-copy: the second occurrence is already a byte copy
	// and must not be rewritten with the next unclaimed source row.
	d := refreshDesign(t, keyedSource,
		"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n"+
			"| name | note |\n|---|---|\n| `alpha` | the alpha note, current |\n| `alpha` | the alpha note, current |\n")
	reports, changed, err := RefreshEmbeds(d, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) != 0 || reports[0].Recopied != 0 {
		t.Fatalf("the duplicated row was rewritten: changed %v, report %+v", changed, reports[0])
	}
	if got := shardText(t, d); strings.Count(got, "the alpha note, current") != 2 {
		t.Fatalf("the duplicate did not survive untouched:\n%s", got)
	}
}

func TestRefreshRefusesAnUnresolvableSelector(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		wantSub string
	}{
		{
			name:    "no source table has the columns",
			source:  "# root\n\n| other | columns |\n|---|---|\n| a | b |\n",
			wantSub: "resolves to 0 source tables",
		},
		{
			name:    "two source tables have them",
			source:  keyedSource + "\nand again\n\n| name | note |\n|---|---|\n| `alpha` | a second table |\n",
			wantSub: "resolves to 2 source tables",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := refreshDesign(t, tc.source,
				"<!-- machinery:embed from=\"ARCHITECTURE.md\" table=\"name,note\" claims=\"subset\" -->\n"+
					"| name | note |\n|---|---|\n| `alpha` | STALE |\n")
			reports, changed, err := RefreshEmbeds(d, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(changed) != 0 {
				t.Fatalf("an unresolvable selector rewrote the document: %v", changed)
			}
			if !strings.Contains(reports[0].Problem, tc.wantSub) {
				t.Fatalf("problem = %q, want it to name %q", reports[0].Problem, tc.wantSub)
			}
		})
	}
}

func TestSortReportsIsDeterministic(t *testing.T) {
	rs := []RefreshReport{
		{File: "b.md", From: "y.md"},
		{File: "a.md", From: "z.md"},
		{File: "a.md", From: "a.md"},
	}
	SortReports(rs)
	want := []string{"a.md:a.md", "a.md:z.md", "b.md:y.md"}
	for i, r := range rs {
		if got := r.File + ":" + r.From; got != want[i] {
			t.Fatalf("report %d = %q, want %q", i, got, want[i])
		}
	}
}
