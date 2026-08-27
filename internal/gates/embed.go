// Ge-embed: declared-embed fidelity.
//
// Machinery's own sharding doctrine tells authors to COPY rows between
// documents so every shard stands alone: a shard restates the root's matrix
// rows, a child restates the parent's event rows. It is the one sanctioned
// duplication class with no generator behind it, and until now it was held
// only by prose ("byte-identical; a diff is a defect") that nothing verified.
// A marker turns that promise into a claim a tool can check.
//
// Adoption is opt-in per table: an unmarked table carries no obligation, so
// existing designs stay green until an author marks one. What is NOT optional
// is a marker that cannot be resolved: an unreadable source, a selector
// matching no table or several, an unknown attribute. Those are loud errors,
// never a silent skip, because a marker nobody can resolve reads exactly like
// a promise nobody checks.

package gates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// embedMarker is the directive an author places on the line(s) before an
// embedded table. HTML comment syntax so it renders invisibly.
var embedMarker = regexp.MustCompile(`<!--\s*machinery:embed\s+(.*?)-->`)

// embedAttr splits key="value" pairs inside a marker.
var embedAttr = regexp.MustCompile(`([a-z_]+)\s*=\s*"([^"]*)"`)

// shardLocalRe is the localization escape, with its reason captured. In the
// FIRST cell it exempts the whole row (a shard addition or a wholly rewritten
// row); in any other cell it exempts that cell alone and the rest of the row
// must still match. Distinct from every other waiver token in the suite:
// '(no machine:)', '(not placed:)', and '(no contract:)' answer other
// questions entirely.
var shardLocalRe = regexp.MustCompile(`\(shard-local:\s*([^)]*)\)`)

// EmbedActive reports whether any markdown file under design carries an embed
// marker. Ge is artifact-activated on that, like Gk on a checker manifest.
func EmbedActive(design string) bool {
	for _, f := range markdownFiles(design) {
		if embedMarker.MatchString(readOrEmpty(f)) {
			return true
		}
	}
	return false
}

// markdownFiles lists every *.md under design, sorted, so findings come out in
// a deterministic order whatever the filesystem's walk order is.
func markdownFiles(design string) []string {
	var out []string
	_ = filepath.WalkDir(design, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// one unreadable subtree must not hide the markers in its
			// siblings; the gates that own those files report the read error
			return nil //nolint:nilerr // keep walking; readFileOrErr reports unreadable files
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			out = append(out, path)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// embedDirective is one parsed marker plus the table it marks.
type embedDirective struct {
	file    string // the embedding document
	from    string // source artifact, relative to the embedding file
	table   string // comma-separated column names identifying the source table
	where   string // optional "<col>[|<col>]=<token>" row filter
	claims  string // comma-separated: subset, complete
	tblText string // the marked table's raw markdown
}

// scanEmbeds reads one document and pairs each marker with the table that
// follows it. The marked table is the next markdown table to START after the
// marker line: a marker with no table after it is an error, never a no-op.
func scanEmbeds(path string, g *Gate) []embedDirective {
	text := readFileOrErr(path, g)
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var out []embedDirective
	for i := 0; i < len(lines); i++ {
		m := embedMarker.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		d := embedDirective{file: path}
		for _, a := range embedAttr.FindAllStringSubmatch(m[1], -1) {
			switch a[1] {
			case "from":
				d.from = a[2]
			case "table":
				d.table = a[2]
			case "where":
				d.where = a[2]
			case "claims":
				d.claims = a[2]
			default:
				g.Errs = append(g.Errs, relPathOf(path)+": embed marker has unknown attribute "+ir.Repr(a[1])+" (supported: from, table, where, claims)")
			}
		}
		// the next table block after the marker
		var tbl []string
		for j := i + 1; j < len(lines); j++ {
			trimmed := strings.TrimLeft(lines[j], " \t")
			if strings.HasPrefix(trimmed, "|") {
				tbl = append(tbl, lines[j])
				continue
			}
			if len(tbl) > 0 {
				break
			}
			if strings.TrimSpace(trimmed) == "" || strings.HasPrefix(trimmed, "<!--") {
				continue // blank lines and further comments may sit between
			}
			break // prose intervened: the marker marks nothing
		}
		d.tblText = strings.Join(tbl, "\n")
		out = append(out, d)
	}
	return out
}

// relPathOf shortens a path for findings: the last two segments are enough to
// name a design file and stay stable across checkouts.
func relPathOf(path string) string {
	dir, file := filepath.Split(filepath.Clean(path))
	parent := filepath.Base(filepath.Clean(dir))
	if parent == "." || parent == string(filepath.Separator) {
		return file
	}
	return parent + "/" + file
}

// parseOneTable parses raw markdown that should hold exactly one table.
func parseOneTable(raw string) (ir.MdTable, bool) {
	tbls := ir.ParseMdTables(raw)
	if len(tbls) != 1 {
		return ir.MdTable{}, false
	}
	return tbls[0], true
}

// headerHasAll reports whether every named column appears in the header.
func headerHasAll(header []string, cols []string) bool {
	for _, c := range cols {
		if colContaining(header, strings.ToLower(strings.TrimSpace(c))) < 0 {
			return false
		}
	}
	return true
}

// cellLocalized reports whether a cell carries the localization escape, and
// whether its reason is non-empty.
func cellLocalized(cell string) (localized, reasoned bool) {
	m := shardLocalRe.FindStringSubmatch(cell)
	if m == nil {
		return false, false
	}
	return true, strings.TrimSpace(m[1]) != ""
}

// rowMatches reports whether an embed row equals a source row on every cell
// the embed does not exempt. Exempt cells are the localized ones.
func rowMatches(embed, src []string, exempt map[int]bool) bool {
	if len(embed) != len(src) {
		return false
	}
	for i := range embed {
		if exempt[i] {
			continue
		}
		if embed[i] != src[i] {
			return false
		}
	}
	return true
}

// CheckEmbeds implements Ge-embed.
func CheckEmbeds(design string) *Gate {
	g := NewGate("Ge-embed  declared-embed fidelity")
	g.startOrder()
	files := markdownFiles(design)
	found := false
	for _, f := range files {
		for _, d := range scanEmbeds(f, g) {
			found = true
			g.Count("embed markers")
			checkOneEmbed(g, d)
		}
	}
	if !found {
		g.Errs = append(g.Errs, "no embed markers found under "+design+
			"; Ge checks declared copies, so forcing it on a design that declares none checks nothing"+
			` (a marker reads: <!-- machinery:embed from="<path>" table="<col>,<col>" claims="subset,complete" -->)`)
	}
	return g
}

// checkOneEmbed verifies one marker: it resolves, its source table is unique,
// and the two claims hold row by row.
func checkOneEmbed(g *Gate, d embedDirective) {
	// findings name the marker compactly (file, source, selector): the raw
	// marker text is precise but too long to read at the head of every line
	where := relPathOf(d.file) + ": embed of " + d.from + " [" + d.table + "]"
	// 1. the marker itself
	missing := []string{}
	if d.from == "" {
		missing = append(missing, "from")
	}
	if d.table == "" {
		missing = append(missing, "table")
	}
	if d.claims == "" {
		missing = append(missing, "claims")
	}
	if len(missing) > 0 {
		g.Errs = append(g.Errs, where+" is missing required attribute(s): "+strings.Join(missing, ", "))
		return
	}
	var subset, complete bool
	for _, c := range strings.Split(d.claims, ",") {
		switch strings.TrimSpace(c) {
		case "subset":
			subset = true
		case "complete":
			complete = true
		case "":
		default:
			g.Errs = append(g.Errs, where+": unknown claim "+ir.Repr(strings.TrimSpace(c))+" (supported: subset, complete)")
			return
		}
	}
	if !subset && !complete {
		g.Errs = append(g.Errs, where+": claims names neither subset nor complete; a marker that claims nothing verifies nothing")
		return
	}
	// 2. the marked table
	embedTbl, ok := parseOneTable(d.tblText)
	if !ok {
		g.Errs = append(g.Errs, where+" marks no table; the marker sits immediately before the table it describes")
		return
	}
	// 3. the source artifact
	srcPath := filepath.Join(filepath.Dir(d.file), d.from)
	srcText := readOrEmpty(srcPath)
	if srcText == "" {
		g.Errs = append(g.Errs, where+": source "+ir.Repr(d.from)+" does not exist or is empty (resolved to "+srcPath+")")
		return
	}
	cols := strings.Split(d.table, ",")
	var matched []ir.MdTable
	for _, t := range ir.ParseMdTables(srcText) {
		if headerHasAll(t.Header, cols) {
			matched = append(matched, t)
		}
	}
	if len(matched) == 0 {
		g.Errs = append(g.Errs, where+": no table in "+ir.Repr(d.from)+" has all the columns "+ir.Repr(d.table))
		return
	}
	if len(matched) > 1 {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: %d tables in %s have the columns %s; name a column that distinguishes them (the selector must resolve to exactly one table)",
			where, len(matched), ir.Repr(d.from), ir.Repr(d.table)))
		return
	}
	srcTbl := matched[0]
	// 4. the two tables must carry the same columns, or row equality is
	// meaningless: an embed is a copy, not a projection
	if len(srcTbl.Header) != len(embedTbl.Header) {
		g.Errs = append(g.Errs, fmt.Sprintf("%s: the embedded table has %d columns, the source table %d; an embed carries the source's columns unchanged",
			where, len(embedTbl.Header), len(srcTbl.Header)))
		return
	}
	for i := range srcTbl.Header {
		if srcTbl.Header[i] != embedTbl.Header[i] {
			g.Errs = append(g.Errs, fmt.Sprintf("%s: column %d is %s here and %s in the source; an embed carries the source's columns unchanged",
				where, i+1, ir.Repr(embedTbl.Header[i]), ir.Repr(srcTbl.Header[i])))
			return
		}
	}
	// 5. the row filter
	srcRows := srcTbl.Rows
	if d.where != "" {
		filtered, err := filterRows(srcTbl, d.where)
		if err != "" {
			g.Errs = append(g.Errs, where+": "+err)
			return
		}
		srcRows = filtered
	}
	g.Count("embed tables verified")

	// 6. the claims. Exemptions are computed ONCE per embed row: they are a
	// property of the row, and recomputing them per comparison would both
	// multiply the counts and repeat every malformed-localization finding.
	type embedRow struct {
		cells    []string
		exempt   map[int]bool
		rowLocal bool
	}
	rows := make([]embedRow, 0, len(embedTbl.Rows))
	for _, cells := range embedTbl.Rows {
		er := embedRow{cells: cells, exempt: map[int]bool{}}
		for i, cell := range cells {
			loc, reasoned := cellLocalized(cell)
			if !loc {
				continue
			}
			if !reasoned {
				g.Errs = append(g.Errs, where+": localization in row "+ir.Repr(firstCell(cells))+" names no reason; write '(shard-local: <reason>)'")
			}
			if i == 0 {
				er.rowLocal = true
				g.Count("rows localized")
				continue
			}
			er.exempt[i] = true
			g.Count("cells localized")
		}
		rows = append(rows, er)
	}
	if subset {
		g.Count("subset claims")
		for _, er := range rows {
			if er.rowLocal {
				continue
			}
			g.Count("rows compared")
			matchedRow := false
			for _, src := range srcRows {
				if rowMatches(er.cells, src, er.exempt) {
					matchedRow = true
					break
				}
			}
			if !matchedRow {
				g.Errs = append(g.Errs, where+": row "+ir.Repr(firstCell(er.cells))+" is not a byte-identical copy of any source row it selects; re-copy it, or mark what differs with '(shard-local: <reason>)'")
			}
		}
	}
	if complete {
		g.Count("complete claims")
		for _, src := range srcRows {
			matchedRow := false
			for _, er := range rows {
				if er.rowLocal {
					continue
				}
				if rowMatches(er.cells, src, er.exempt) {
					matchedRow = true
					break
				}
			}
			if !matchedRow {
				g.Errs = append(g.Errs, where+": source row "+ir.Repr(firstCell(src))+" is selected but absent here; copy it in, narrow the where= filter, or drop the complete claim")
			}
		}
	}
}

// firstCell names a row in a finding (its key column).
func firstCell(row []string) string {
	if len(row) == 0 {
		return ""
	}
	return row[0]
}

// filterRows applies a "<col>[|<col>]=<token>" filter to the source rows: keep
// the rows whose named column(s) contain the token as a whole token, the same
// matching rule the rest of the suite uses for ids. Returns a message on a
// malformed filter or an unknown column.
func filterRows(t ir.MdTable, where string) ([][]string, string) {
	parts := strings.SplitN(where, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return nil, "where=" + ir.Repr(where) + " is not of the form '<column>=<token>' (several columns: '<col>|<col>=<token>')"
	}
	token := strings.TrimSpace(parts[1])
	var idx []int
	for _, name := range strings.Split(parts[0], "|") {
		i := colContaining(t.Header, strings.ToLower(strings.TrimSpace(name)))
		if i < 0 {
			return nil, "where= names column " + ir.Repr(strings.TrimSpace(name)) + ", which the source table does not have"
		}
		idx = append(idx, i)
	}
	var out [][]string
	for _, r := range t.Rows {
		for _, i := range idx {
			if i < len(r) && tokenIn(token, r[i]) {
				out = append(out, r)
				break
			}
		}
	}
	return out, ""
}
