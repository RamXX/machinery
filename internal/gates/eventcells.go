// Event-contract cell quality (G2). The event table was counted and nothing
// more: `machinery check` reported "8 event contracts" over rows whose
// delivery, ordering, or dedupe cells could be blank and whose producer and
// consumer could name components no model declares. The mitigation table has
// had exactly that name resolution since the beginning (a backticked token
// that is neither a workspace.dsl element nor a declared external is an
// ERROR); the event table, which governs the coupling G4-import cannot see,
// had none.
//
// Two obligations, both ERROR:
//
//   - the columns the format contract names exist, and no row leaves one of
//     them empty. An EMPTY cell is an unanswered question; "none", "n/a", or
//     "-" are answers and pass (the format documents no placeholder token, so
//     nothing here rejects one).
//   - every producer and consumer name resolves to a declared workspace.dsl
//     element or a declared external, the same resolution a mitigation row
//     gets.
//
// Where this and `machinery pack generate` disagree, the pack is the stricter
// one: it additionally requires an event column and rejects a cell naming
// anything but a subsystem component or a boundary element (externals do not
// qualify there). Nothing G2 accepts can therefore make pack generation pass
// silently; the pack still fails loudly on its own contract, which is why the
// event column is required there and not here.

package gates

import (
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// eventTableCols are the columns the event-contract format contract names,
// with the substring each is located by (colContaining's forgiving rule, so
// "payload (Modelith attributes)" and "dedupe key" are found).
var eventTableCols = []struct{ name, needle string }{
	{"producer", "producer"},
	{"consumer", "consumer"},
	{"payload", "payload"},
	{"delivery", "delivery"},
	{"ordering", "ordering"},
	{"dedupe", "dedupe"},
}

// eventContractTable is one header-matching markdown table with its column map.
type eventContractTable struct {
	Header []string
	Rows   [][]string
	Cols   map[string]int
}

// eventRow is one event-contract row: its cells, the column map of the table
// it came from, and the cumulative row number findings address it by.
type eventRow struct {
	Cells []string
	Cols  map[string]int
	Row   int
}

// Cell returns the raw text of a named column ("" when the column is absent).
func (r eventRow) Cell(col string) string { return cellAt(r.Cells, r.Cols[col]) }

// Clean returns a named column cleaned the way every cell-reading gate cleans
// one: backticks and parenthesized annotations stripped.
func (r eventRow) Clean(col string) string { return ir.CleanCell(r.Cell(col)) }

// Where renders the address a finding uses: the cumulative row number and,
// when the table has an event column, the event it names. Identical in shape
// to pack generation's "event-contract row 3 (event 'markPaid')".
func (r eventRow) Where() string {
	where := "event-contract row " + strconv.Itoa(r.Row)
	if ev := r.Clean("event"); ev != "" {
		where += " (event " + ir.Repr(ev) + ")"
	}
	return where
}

// eventContractTables returns EVERY markdown table whose header names
// producer, consumer, and delivery: the one header rule G2's presence check,
// the enumeration-source check, and pack generation already share. A design
// legitimately splits its event contract across sections, and a first-match
// locator would silently excuse every later table (PACK-1).
func eventContractTables(text string) []eventContractTable {
	var out []eventContractTable
	for _, tbl := range ir.ParseMdTables(text) {
		hl := strings.ToLower(strings.Join(tbl.Header, " "))
		if !strings.Contains(hl, "producer") || !strings.Contains(hl, "consumer") || !strings.Contains(hl, "delivery") {
			continue
		}
		cols := map[string]int{"event": colContaining(tbl.Header, "event")}
		for _, c := range eventTableCols {
			cols[c.name] = colContaining(tbl.Header, c.needle)
		}
		out = append(out, eventContractTable{Header: tbl.Header, Rows: tbl.Rows, Cols: cols})
	}
	return out
}

// eventContractRows flattens every event-contract table into rows numbered
// cumulatively, so a finding addresses the same row the pack's findings do
// when a design later decomposes.
func eventContractRows(text string) []eventRow {
	var out []eventRow
	n := 0
	for _, tbl := range eventContractTables(text) {
		for _, r := range tbl.Rows {
			if len(r) == 0 {
				continue
			}
			n++
			out = append(out, eventRow{Cells: r, Cols: tbl.Cols, Row: n})
		}
	}
	return out
}

// checkEventCells holds every event-contract row to the format contract's
// columns and resolves its producer and consumer names against the declared
// model. externalIDs maps each declared external id to its bound element name,
// the same universe the mitigation-coverage check resolves against.
func checkEventCells(g *Gate, text string, els map[string]dslEl, externalIDs map[string]string) {
	tables := eventContractTables(text)
	if len(tables) == 0 {
		return
	}
	declared := map[string]bool{}
	for name := range els {
		declared[name] = true
	}
	for id, el := range externalIDs {
		declared[id] = true
		if el != "" {
			declared[el] = true
		}
	}
	var known []string
	for n := range declared {
		known = append(known, n)
	}
	sort.Strings(known)
	for _, tbl := range tables {
		for _, c := range eventTableCols {
			if tbl.Cols[c.name] < 0 {
				g.Errs = append(g.Errs, "an event-contract table has no "+c.name+" column; the contract's columns are producer, consumer, payload, delivery, ordering, and dedupe, and an absent column is a question the table never asked")
			}
		}
	}
	for _, r := range eventContractRows(text) {
		where := r.Where()
		for _, c := range eventTableCols {
			if r.Cols[c.name] < 0 {
				continue // the missing column is reported once per table above
			}
			if strings.TrimSpace(r.Cell(c.name)) == "" {
				g.Errs = append(g.Errs, where+": empty "+c.name+" cell; an unanswered column is not a contract (write the answer, \"none\" or \"n/a\" included when that is the answer)")
				continue
			}
			g.Count("event-contract cells answered")
		}
		// delivery-vs-dedupe consistency: "at-least-once" is a promise that
		// duplicates WILL arrive, so a bare no-answer dedupe cell contradicts
		// it. A reasoned cell ("none (idempotent consumer: upsert by id)")
		// passes: the parenthesized reason is the answer. This demotes the
		// mechanical slice of the g3.event-redelivery attestation; the
		// ADEQUACY of a stated dedupe story stays judgment.
		if r.Cols["delivery"] >= 0 && r.Cols["dedupe"] >= 0 {
			del := strings.ToLower(ir.CleanCell(r.Cell("delivery")))
			ded := strings.TrimSpace(r.Cell("dedupe"))
			if strings.Contains(del, "at-least-once") || strings.Contains(del, "at least once") {
				switch strings.ToLower(ded) {
				case "none", "n/a", "na", "-":
					g.Errs = append(g.Errs, where+": delivery is at-least-once but dedupe is a bare "+ir.Repr(ded)+"; duplicates will arrive, so name the dedupe key or state why none is safe ('none (idempotent consumer: <reason>)')")
				case "":
					// the empty-cell finding above already names it
				default:
					g.Count("at-least-once rows with a dedupe answer")
				}
			}
		}
		for _, col := range []string{"producer", "consumer"} {
			clean := r.Clean(col)
			if clean == "" {
				continue // the empty-cell finding above already names it
			}
			if declared[clean] {
				g.Count("event-contract participants resolved")
				continue
			}
			g.Errs = append(g.Errs, where+": "+col+" "+ir.Repr(r.Cell(col))+" is neither a workspace.dsl element nor a declared external (declared: "+strings.Join(known, ", ")+"); one component per cell, annotations only in parentheses")
		}
	}
}

// cellAt returns row cell i, or "" when the column is absent or the row short.
func cellAt(r []string, i int) string {
	if i >= 0 && i < len(r) {
		return r[i]
	}
	return ""
}
