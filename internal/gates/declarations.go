// Package gates implements deterministic design checks. This file owns the
// declaration grammar family of the consistency layer, Stage 1
// (docs/consistency-layer-proposal.md, section 3.1).
//
// Machinery has one declaration idiom: an upper-case word immediately
// followed by a braced, comma-separated closed list, placed in a table cell.
// CLAUSES{}, READS{}, VALUES{} and ORACLESET{} are parsed by their own
// owners (clausedecl.go, payloadreads.go, values.go, the Gb packet reader).
// This file adds five members and parses them into typed values, so the
// projection can turn them into facts without re-reading the design:
//
//	WRITES{Order.status, Order.total}   the closed set of stored facts a unit writes
//	WRITES{}                            a read-only unit
//	USES{Order.total, line_item_count}  the closed set of facts a unit reads or names
//	PRODUCES{Order.markPaid}            the Modelith actions a matrix row's cascade or
//	                                    consumer arm performs; each owes an admission
//	CARRIES{column:Order.status, outbox:order.confirmed}
//	                                    what carries a unit's effect; kinds are
//	                                    column, outbox, sink, signal, action
//	SUPERSEDES{type:LegacyOrder}        on an Architecture Contract row: this row
//	                                    replaces a stable type id (kind type only)
//
// What is enforced here is the grammar, at parse time, and the closure of the
// group-name vocabulary (the Gy-rules gate decides on the projected facts): an upper-case
// word followed by `{` in a matrix table cell that names no known group is an
// ERROR, exactly as an unknown `_` annotation is in the machine lint.
//
// Decisions the grammar settles, each pinned by a test:
//
//   - A group opens and closes inside ONE table cell. A cell that opens a
//     group and never closes it is an error for that row; the scan never
//     continues into the next cell or the next row, so a brace closed on the
//     following table row cannot swallow that row.
//   - A group in an inline code span is still a group. "Prose never declares"
//     means a sentence cannot create a fact by implication; a literal
//     `WRITES{...}` is not an implication, it is the declaration spelled in
//     full, and backticks are how cells format code. Treating a backticked
//     group as prose would give the same text two meanings by typography.
//     Fenced code blocks are different: a fence is an example, not a row, and
//     fenced lines are never scanned.
//   - Only the ASCII braces `{` and `}` open and close a group. A lookalike
//     (FULLWIDTH LEFT CURLY BRACKET U+FF5B, MEDIUM LEFT CURLY BRACKET
//     ORNAMENT U+2774, and the like) is prose: it neither declares nor trips
//     the unknown-group rule, because the rule exists to catch a misspelled
//     group NAME, and a lookalike brace is not a spelling of any group. Inside
//     a real (ASCII-braced) CARRIES or SUPERSEDES group a lookalike colon is
//     just a member without the `:` separator, and is an error like any
//     other malformed member.
//   - Line endings are normalized (CRLF to LF) before anything is read.
package gates

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// Declaration group names. The five below are parsed here; the rest are
// owned elsewhere and only recognized, so the unknown-group rule accepts them.
const (
	GroupWrites     = "WRITES"
	GroupUses       = "USES"
	GroupProduces   = "PRODUCES"
	GroupCarries    = "CARRIES"
	GroupSupersedes = "SUPERSEDES"
)

// knownDeclarationGroups is the closed group-name vocabulary. `payload {`
// is lower-case and never reaches the upper-case opening scan.
var knownDeclarationGroups = map[string]bool{
	"CLAUSES": true, "RETIRED": true, "READS": true, "VALUES": true, "ORACLESET": true,
	GroupWrites: true, GroupUses: true, GroupProduces: true, GroupCarries: true, GroupSupersedes: true,
}

// parsedHere reports whether a group is parsed by this file (the rest are
// recognized only, and parsed by their own owners).
func parsedHere(group string) bool {
	switch group {
	case GroupWrites, GroupUses, GroupProduces, GroupCarries, GroupSupersedes:
		return true
	}
	return false
}

// producedAction is the Entity.action shape of a PRODUCES member: exactly one
// dot, both halves identifiers, the shape of a Modelith action id.
var producedAction = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*\.[A-Za-z][A-Za-z0-9_]*$`)

// carrierKinds is the closed CARRIES kind vocabulary; supersessionKinds the
// closed SUPERSEDES one (Stage 1 admits type replacement only).
var (
	carrierKinds      = map[string]bool{"column": true, "outbox": true, "sink": true, "signal": true, "action": true}
	supersessionKinds = map[string]bool{"type": true}
)

// groupOpening finds an all-capitals word of two or more characters directly
// followed by an ASCII `{`. The leading class keeps the match whole-word
// under RE2's lack of lookbehind: `RankedConstituent{` (CamelCase) and
// `xWRITES{` never match; `err{` is lower-case and never matches. A
// hyphenated or underscored name (`OWNED-BY{`) is one word, so a private
// group is reported whole.
var groupOpening = regexp.MustCompile(`(?:^|[^A-Za-z0-9_-])([A-Z][A-Z0-9_-]*[A-Z0-9])\{`)

// Declaration is one parsed WRITES, USES, PRODUCES, CARRIES, or SUPERSEDES group.
type Declaration struct {
	File    string            // design-relative, slash-separated path
	Line    int               // 1-based line of the table row
	Row     string            // the row's unit name (named-unit tables) or first cell
	Column  string            // header label of the cell carrying the group
	Group   string            // GroupWrites, GroupUses, GroupProduces, GroupCarries, GroupSupersedes
	Members []string          // members in source order; "kind:target" for CARRIES/SUPERSEDES
	Pairs   []DeclarationPair // CARRIES and SUPERSEDES members, split
}

// DeclarationPair is one kind:target member.
type DeclarationPair struct {
	Kind, Target string
}

// DeclarationError is one parse-time finding, addressed by file, line and row.
type DeclarationError struct {
	File    string
	Line    int
	Row     string
	Message string
}

// String renders the finding in the `file:line: message` shape every matrix
// finding in Gx uses (the VALUES findings among them).
func (e DeclarationError) String() string {
	where := e.File + ":" + strconv.Itoa(e.Line) + ": "
	if e.Row != "" {
		where += "row " + ir.Repr(e.Row) + ": "
	}
	return where + e.Message
}

// groupSpan is one upper-case group opening found in a cell.
type groupSpan struct {
	name   string
	start  int // index of the group name
	open   int // index of `{`
	end    int // index just past `}`, or -1 when the group never closes
	body   string
	nested bool // another `{` opens before this group closes
}

// scanGroups returns every upper-case group opening in one cell, in order.
// A closed group is skipped over whole, so a group spelled inside another
// group's body is not a second group; an unclosed one is reported and the
// scan resumes just past its brace.
func scanGroups(cell string) []groupSpan {
	var out []groupSpan
	for pos := 0; pos < len(cell); {
		loc := groupOpening.FindStringSubmatchIndex(cell[pos:])
		if loc == nil {
			break
		}
		span := groupSpan{name: cell[pos+loc[2] : pos+loc[3]], start: pos + loc[2], open: pos + loc[1] - 1, end: -1}
		rest := cell[span.open+1:]
		closeAt := strings.IndexByte(rest, '}')
		if closeAt >= 0 {
			span.body = rest[:closeAt]
			span.end = span.open + 1 + closeAt + 1
			span.nested = strings.IndexByte(span.body, '{') >= 0
			pos = span.end
		} else {
			pos = span.open + 1
		}
		out = append(out, span)
	}
	return out
}

// parseDottedMembers parses a WRITES or USES body: comma-separated members,
// each the dotted identifier grammar payload fields use (an optional pair of
// surrounding backticks is tolerated, as in payload {}). allowEmpty admits
// the whole-group empty form.
func parseDottedMembers(group, body string, allowEmpty bool) ([]string, string) {
	if strings.TrimSpace(body) == "" {
		if allowEmpty {
			return []string{}, ""
		}
		verb := "uses"
		if group == GroupProduces {
			verb = "produces"
		}
		return nil, group + "{} is empty; a unit that " + verb + " nothing declares no " + group + " group"
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range strings.Split(body, ",") {
		member := unquoteMember(raw)
		if member == "" {
			return nil, group + " declaration contains an empty member"
		}
		if !payloadField.MatchString(member) {
			return nil, group + " member " + ir.Repr(member) + " is not a dotted identifier (Entity.attr or snake_case)"
		}
		if seen[member] {
			return nil, group + " declaration has duplicate member " + ir.Repr(member)
		}
		seen[member] = true
		out = append(out, member)
	}
	return out, ""
}

// parseKindPairs parses a CARRIES or SUPERSEDES body: comma-separated
// kind:target members over a closed kind vocabulary.
func parseKindPairs(group, body string, kinds map[string]bool) ([]string, []DeclarationPair, string) {
	if strings.TrimSpace(body) == "" {
		return nil, nil, group + "{} is empty; write " + group + "{kind:target, ...} or drop the group"
	}
	var names []string
	for k := range kinds {
		names = append(names, k)
	}
	sort.Strings(names)
	seen := map[string]bool{}
	var members []string
	var pairs []DeclarationPair
	for _, raw := range strings.Split(body, ",") {
		member := unquoteMember(raw)
		if member == "" {
			return nil, nil, group + " declaration contains an empty member"
		}
		kind, target, ok := strings.Cut(member, ":")
		if !ok {
			return nil, nil, group + " member " + ir.Repr(member) + " has no kind; write kind:target with kind one of " + strings.Join(names, ", ")
		}
		kind, target = strings.TrimSpace(kind), strings.TrimSpace(target)
		if !kinds[kind] {
			return nil, nil, group + " member " + ir.Repr(member) + " has unknown kind " + ir.Repr(kind) + "; kinds are " + strings.Join(names, ", ")
		}
		if !payloadField.MatchString(target) {
			return nil, nil, group + " member " + ir.Repr(member) + " target " + ir.Repr(target) + " is not a dotted identifier"
		}
		key := kind + ":" + target
		if seen[key] {
			return nil, nil, group + " declaration has duplicate member " + ir.Repr(key)
		}
		seen[key] = true
		members = append(members, key)
		pairs = append(pairs, DeclarationPair{Kind: kind, Target: target})
	}
	return members, pairs, ""
}

func unquoteMember(raw string) string {
	member := strings.TrimSpace(raw)
	if len(member) >= 2 && member[0] == '`' && member[len(member)-1] == '`' {
		member = strings.TrimSpace(member[1 : len(member)-1])
	}
	return member
}

// parseGroup parses one closed group span of a parsed-here kind.
func parseGroup(span groupSpan) (Declaration, string) {
	d := Declaration{Group: span.name}
	var why string
	switch span.name {
	case GroupWrites:
		d.Members, why = parseDottedMembers(GroupWrites, span.body, true)
	case GroupUses:
		d.Members, why = parseDottedMembers(GroupUses, span.body, false)
	case GroupProduces:
		d.Members, why = parseDottedMembers(GroupProduces, span.body, false)
		for _, m := range d.Members {
			if why == "" && !producedAction.MatchString(m) {
				why = "PRODUCES member " + ir.Repr(m) + " is not one Entity.action identifier"
			}
		}
	case GroupCarries:
		d.Members, d.Pairs, why = parseKindPairs(GroupCarries, span.body, carrierKinds)
	case GroupSupersedes:
		d.Members, d.Pairs, why = parseKindPairs(GroupSupersedes, span.body, supersessionKinds)
	}
	return d, why
}

// tableRow is one data row of a markdown table, with its line number and the
// header it sits under.
type tableRow struct {
	line   int
	header []string
	cells  []string
}

// walkTableRows yields the data rows of every markdown table in text,
// skipping fenced code. Header rows and separator rows are not yielded.
func walkTableRows(text string) []tableRow {
	var out []tableRow
	var header []string
	inTable := false
	for i, line := range strings.Split(maskFences(text), "\n") {
		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "|") {
			inTable, header = false, nil
			continue
		}
		cells := ir.SplitRowCells(line)
		for j := range cells {
			cells[j] = strings.TrimSpace(cells[j])
		}
		if !inTable {
			inTable, header = true, cells
			continue
		}
		if isMarkdownSeparator(cells) {
			continue
		}
		out = append(out, tableRow{line: i + 1, header: header, cells: cells})
	}
	return out
}

// rowIdentity returns the name a finding addresses a row by: the name cell of
// a named-unit table, else the first cell, backticks stripped.
func rowIdentity(r tableRow) string {
	col := 0
	if ni, _, _, ok := namedUnitCols(r.header); ok {
		col = ni
	}
	return strings.TrimSpace(strings.ReplaceAll(cellAt(r.cells, col), "`", ""))
}

func normalizeNewlines(body []byte) string {
	return strings.ReplaceAll(string(body), "\r\n", "\n")
}

// ParseMatrixDeclarations walks one matrix file and returns every WRITES,
// USES, PRODUCES, and CARRIES declaration in it, row by row, plus every parse-time
// finding: a malformed, empty (where the empty form is not legal),
// duplicated, unterminated or nested group, more than one group of a name on
// one row, a SUPERSEDES group (which belongs on an Architecture Contract
// row), and an upper-case group name outside the known vocabulary. Every cell
// of every table row is scanned; fenced code is not. file is the path the
// findings and declarations carry.
func ParseMatrixDeclarations(file string, body []byte) ([]Declaration, []DeclarationError) {
	var decls []Declaration
	var errs []DeclarationError
	for _, r := range walkTableRows(normalizeNewlines(body)) {
		row := rowIdentity(r)
		fail := func(msg string) {
			errs = append(errs, DeclarationError{File: file, Line: r.line, Row: row, Message: msg})
		}
		perGroup := map[string]int{}
		for ci, cell := range r.cells {
			for _, span := range scanGroups(cell) {
				switch {
				case !knownDeclarationGroups[span.name]:
					fail("unknown declaration group " + span.name + "{...}; known groups are CARRIES, CLAUSES, ORACLESET, PRODUCES, READS, USES, VALUES, WRITES (SUPERSEDES on Architecture Contract rows) and payload {...}")
					continue
				case !parsedHere(span.name):
					continue // CLAUSES, READS, VALUES, ORACLESET: owned by their own parsers
				case span.end < 0:
					fail(span.name + "{ is never closed in its cell; a group opens and closes inside one table cell and cannot span cells or rows")
					continue
				case span.nested:
					fail(span.name + "{...} contains another '{' before it closes; groups do not nest")
					continue
				case span.name == GroupSupersedes:
					fail("SUPERSEDES{...} belongs on an Architecture Contract row, not in a matrix")
					continue
				}
				perGroup[span.name]++
				if perGroup[span.name] > 1 {
					fail("row carries more than one " + span.name + " group; merge them into one closed set")
					continue
				}
				d, why := parseGroup(span)
				if why != "" {
					fail(why)
					continue
				}
				d.File, d.Line, d.Row = file, r.line, row
				if ci < len(r.header) {
					d.Column = r.header[ci]
				}
				decls = append(decls, d)
			}
		}
	}
	return decls, errs
}

// contractBoundaryID reads the `- id: X` opener of a contract YAML list item.
var contractBoundaryID = regexp.MustCompile(`^\s*-\s*id:\s*["']?([^"'\s#]+)`)

// ParseContractDeclarations walks an ARCHITECTURE.md and returns every
// SUPERSEDES declaration on an Architecture Contract row: a markdown table
// row (row identity is its first cell), or a line of the contract YAML fence
// (row identity is the enclosing `- id:` item, else "contract"). Other group
// names are not read here; the unknown-group rule is a matrix rule.
func ParseContractDeclarations(file string, body []byte) ([]Declaration, []DeclarationError) {
	text := normalizeNewlines(body)
	var decls []Declaration
	var errs []DeclarationError
	visit := func(line int, row, column string, cells []string) {
		count := 0
		for _, cell := range cells {
			for _, span := range scanGroups(cell) {
				if span.name != GroupSupersedes {
					continue
				}
				fail := func(msg string) {
					errs = append(errs, DeclarationError{File: file, Line: line, Row: row, Message: msg})
				}
				switch {
				case span.end < 0:
					fail("SUPERSEDES{ is never closed in its cell; a group opens and closes inside one table cell and cannot span cells or rows")
					continue
				case span.nested:
					fail("SUPERSEDES{...} contains another '{' before it closes; groups do not nest")
					continue
				}
				count++
				if count > 1 {
					fail("row carries more than one SUPERSEDES group; merge them into one closed set")
					continue
				}
				d, why := parseGroup(span)
				if why != "" {
					fail(why)
					continue
				}
				d.File, d.Line, d.Row, d.Column = file, line, row, column
				decls = append(decls, d)
			}
		}
	}
	for _, r := range walkTableRows(text) {
		visit(r.line, rowIdentity(r), "", r.cells)
	}
	if fence, ok := ir.ContractFence(text); ok && strings.Contains(fence, GroupSupersedes+"{") {
		if at := strings.Index(text, fence); at >= 0 {
			first := strings.Count(text[:at], "\n") + 1
			row := "contract"
			for i, line := range strings.Split(fence, "\n") {
				if m := contractBoundaryID.FindStringSubmatch(line); m != nil {
					row = m[1]
				}
				visit(first+i, row, "contract", []string{line})
			}
		}
	}
	sort.SliceStable(decls, func(i, j int) bool { return decls[i].Line < decls[j].Line })
	sort.SliceStable(errs, func(i, j int) bool { return errs[i].Line < errs[j].Line })
	return decls, errs
}

// checkDeclarations surfaces the parse-time findings of every matrix and of
// ARCHITECTURE.md in Gx-trace, beside the VALUES findings, in the same
// `file:line: message` shape. It returns the parsed declarations for callers
// that want them; Gx itself only counts them.
func checkDeclarations(g *Gate, design, archText string) []Declaration {
	var all []Declaration
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, err := readDesignFile(design, path)
		if err != nil {
			continue // the VALUES and payload readers already report an unreadable matrix
		}
		decls, errs := ParseMatrixDeclarations(filepath.Base(path), body)
		for _, e := range errs {
			g.Errs = append(g.Errs, e.String())
		}
		all = append(all, decls...)
	}
	if archText != "" {
		decls, errs := ParseContractDeclarations("ARCHITECTURE.md", []byte(archText))
		for _, e := range errs {
			g.Errs = append(g.Errs, e.String())
		}
		all = append(all, decls...)
	}
	g.Count("declaration groups parsed", len(all))
	return all
}
