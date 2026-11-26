package dialect

import (
	"fmt"
	"regexp"
	"strings"
)

// Row is one failure-mode row of a Markdown catalog (section 5.4.1).
type Row struct {
	Cell     string // the first cell, trimmed, as written
	ID       string // the machinery id: the cell lower-cased
	Line     int    // 1-based
	Failure  string
	Behavior string
	Cells    []string // every cell of the row, trimmed
}

// Columns names, for each column a failure-mode row needs, the header
// names that may stand for it, in the order they are tried.
type Columns struct {
	Failure  []string
	Behavior []string
}

var (
	delimCell = regexp.MustCompile(`^:?-+:?$`)
	fmIDRe    = regexp.MustCompile(`^[a-z][a-z0-9-]+$`)
)

// Catalog reads the failure-mode rows of a Markdown document: every table
// row whose first cell matches id, with its columns found by the header of
// its own table. Problems name the file and line; a duplicate row is not
// returned.
func Catalog(rel string, data []byte, id *regexp.Regexp, cols Columns) (rows []Row, problems []string) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	first := map[string]int{}
	fence := ""
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if f := fenceOf(trimmed); f != "" {
			switch {
			case fence == "":
				fence = f
			case strings.HasPrefix(trimmed, fence):
				fence = ""
			}
			continue
		}
		if fence != "" || !isRow(lines[i]) {
			continue
		}
		if i+1 >= len(lines) || !isDelimiter(lines[i+1]) {
			// rows outside every table: a blank line ended their table, or
			// it never had a header. One missing header is one finding for
			// the whole run of lines, not one per row.
			var orphans []Row
			j := i
			for ; j < len(lines) && isRow(lines[j]) && (j+1 >= len(lines) || !isDelimiter(lines[j+1])); j++ {
				if cells := splitRow(lines[j]); len(cells) > 0 && id.MatchString(cells[0]) {
					orphans = append(orphans, Row{Cell: cells[0], Line: j + 1})
				}
			}
			const why = "(a table needs a header row and a delimiter row above its rows, and a blank line ends it)"
			switch n := len(orphans); {
			case n == 1:
				problems = append(problems, fmt.Sprintf("%s:%d: %s: the row is in no table %s", rel, orphans[0].Line, orphans[0].Cell, why))
			case n > 1:
				last := orphans[n-1]
				problems = append(problems, fmt.Sprintf("%s:%d: %s: the row and the %d after it, to %s on line %d, are in no table %s", rel, orphans[0].Line, orphans[0].Cell, n-1, last.Cell, last.Line, why))
			}
			i = j - 1
			continue
		}
		header := splitRow(lines[i])
		failure, behavior := column(header, cols.Failure), column(header, cols.Behavior)
		j := i + 2
		for ; j < len(lines) && isRow(lines[j]); j++ {
			cells := splitRow(lines[j])
			if len(cells) == 0 || !id.MatchString(cells[0]) {
				continue
			}
			r := Row{Cell: cells[0], ID: strings.ToLower(cells[0]), Line: j + 1,
				Failure: cellAt(cells, failure), Behavior: cellAt(cells, behavior), Cells: cells}
			where := fmt.Sprintf("%s:%d: %s", rel, r.Line, r.Cell)
			if !fmIDRe.MatchString(r.ID) {
				problems = append(problems, fmt.Sprintf("%s: the id lower-cased, %s, must match ^[a-z][a-z0-9-]+$", where, r.ID))
			}
			if at, dup := first[r.ID]; dup {
				problems = append(problems, fmt.Sprintf("%s: duplicate id, first on line %d", where, at))
				continue
			}
			first[r.ID] = r.Line
			for _, c := range []struct {
				name  string
				index int
				names []string
				value string
			}{{"failure", failure, cols.Failure, r.Failure}, {"behavior", behavior, cols.Behavior, r.Behavior}} {
				switch {
				case c.index < 0:
					problems = append(problems, fmt.Sprintf("%s: the table header has no %s column (%s)", where, c.name, strings.Join(c.names, ", ")))
				case c.value == "":
					problems = append(problems, fmt.Sprintf("%s: the %s cell is empty", where, c.name))
				}
			}
			rows = append(rows, r)
		}
		i = j - 1
	}
	return rows, problems
}

// fenceOf returns the fence a line opens or closes, or "".
func fenceOf(trimmed string) string {
	for _, f := range []string{"```", "~~~"} {
		if strings.HasPrefix(trimmed, f) {
			return f
		}
	}
	return ""
}

func isRow(line string) bool {
	return strings.TrimSpace(line) != "" && strings.Contains(line, "|")
}

// isDelimiter reports a table delimiter row: cells of dashes with an
// optional colon at either end.
func isDelimiter(line string) bool {
	if !strings.Contains(line, "-") {
		return false
	}
	cells := splitRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, c := range cells {
		if !delimCell.MatchString(c) {
			return false
		}
	}
	return true
}

// splitRow returns the trimmed cells of a table row. Leading and trailing
// pipes are optional, and \| is a pipe inside a cell.
func splitRow(line string) []string {
	s := strings.TrimSpace(line)
	s = strings.TrimPrefix(s, "|")
	if strings.HasSuffix(s, "|") && !strings.HasSuffix(s, "\\|") {
		s = s[:len(s)-1]
	}
	var cells []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '\\' && i+1 < len(s) && s[i+1] == '|':
			b.WriteByte('|')
			i++
		case s[i] == '|':
			cells = append(cells, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(s[i])
		}
	}
	return append(cells, strings.TrimSpace(b.String()))
}

// column returns the index of the first name that is a header cell, or -1.
func column(header, names []string) int {
	for _, n := range names {
		for i, h := range header {
			if strings.EqualFold(h, strings.TrimSpace(n)) {
				return i
			}
		}
	}
	return -1
}

func cellAt(cells []string, i int) string {
	if i < 0 || i >= len(cells) {
		return ""
	}
	return cells[i]
}
