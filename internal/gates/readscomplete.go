// Consumer-READS completeness (Gx-trace, opt-in by a design's own marker).
// The stage-one READS mechanism (payloadreads.go) reconciles a DECLARED read
// against the payload rows that carry the event. It is opt-in per declaration,
// which is exactly the hole a real defect fell through: an event-contract row
// claimed its consumer drafts a new entity from the event, the payload carried
// nothing the target entity's creation invariant needs, and every gate passed.
// The row's cells were answered (G2), the reaction was declared (Gx), the
// invariant had an enforcement row (Gx). Nothing forced the consumer side to
// say what it READS, so the insufficiency of the payload for the declared
// reaction was invisible: an undeclared consumer carried no obligation, and the
// defective row was simply never armed.
//
// This is the completeness tier over the same mechanism and the same syntax:
// once a design arms it, a consumer row with no READS declaration is an ERROR
// naming the row and the event, and a declared field the row's payload cell
// does not carry is an ERROR too, rather than the opt-in warn.
//
// ARMING. A `<!-- machinery:reads-complete -->` marker anywhere in
// ARCHITECTURE.md. Three reasons for that mechanism over the alternatives:
//
//   - it is the repo's existing opt-in idiom for a claim about a hand-written
//     table. Ge-embed activates on exactly such a marker ("a marker turns that
//     promise into a claim a tool can check"), and G2 already reads the lines
//     above an event table for a `Source:` note or an embed marker, so authors
//     and gates both know the neighborhood. A CLI flag would be a rule someone
//     has to remember; a marker travels with the design it governs.
//   - it is a property of the design's event CONTRACT, not of one table. Every
//     markdown table whose header names producer, consumer, and delivery is an
//     event-contract table, and the contract is legitimately split across
//     several (PACK-1). A design cannot honestly claim completeness for half
//     its contract, so the marker arms every row the document carries.
//   - an unarmed design is untouched: no marker, no obligation, no counts, the
//     stage-one warn tier exactly as it was.
//
// The marker's placement is by convention directly above the event-contract
// section, where a reader meets it before the rows; nothing here depends on
// that, because a table-position rule would silently disarm a design that
// reorganized its sections.
//
// SEMANTICS, per event-contract row, for every event its event cell names
// (the same reading the wiring check gives the cell):
//
//   - `(no reads: <reason>)` in the consumer cell waives the row. A reason is
//     mandatory, as with every house waiver; an empty one is an unanswered
//     question. This is the honest answer for a pure signal: a consumer that
//     reads nothing off the payload and refetches by id.
//   - otherwise a matrix row naming the event declares READS{field, ...}
//     and names this exact participant in its consumer column. A legacy row
//     without that column resolves only if the event has one distinct consumer.
//   - declarations for an event/consumer pair agree on one exact field set,
//     including across machines; neither unions nor sibling declarations count.
//   - every declared field appears whole-token in THAT ROW's payload cell, or
//     the row is an ERROR. The stage-one check searches every event-table row
//     for the event across the design; here the row under judgment carries its
//     own payload, so the reconciliation is against the cell that owes it.
//
// A `(no machine: <reason>)` waiver does NOT discharge the reads obligation.
// It answers a different question (nothing reacts to this event AS A MACHINE
// EVENT), and the motivating defect's consumer was exactly such a row: the
// reaction ran through an invoke actor, and the payload was short anyway. One
// generic waiver token would let an answer to one question waive another.
//
// HOST: Gx-trace, for the reason the wiring check states in eventwiring.go.
// This joins an ARCHITECTURE.md table to the committed machines' matrices, and
// every check that does that already lives here. Unlike the wiring check it
// runs on a packed design too: G5 reconciles boundary-event DIRECTION from the
// generated events.md and has no notion of READS, so nothing double-reports.
// What does need standing down is the stage-one warn tier, and it does: Gd
// skips the events an armed contract names (see checkPayloadReads).

package gates

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// readsCompleteMarker arms the completeness tier. HTML comment syntax, so it
// renders invisibly, exactly like the embed marker.
var readsCompleteMarker = regexp.MustCompile(`<!--\s*machinery:reads-complete\s*-->`)

// noReadsWaiverRe waives one row's reads obligation, with its reason captured.
// Distinct from every other waiver token in the suite: '(no machine:)',
// '(not placed:)', '(no contract:)', and '(no closure:)' answer other
// questions entirely.
var noReadsWaiverRe = regexp.MustCompile(`\(no reads:\s*([^)]*)\)`)

// readsCompleteArmed reports whether an ARCHITECTURE.md arms the tier.
func readsCompleteArmed(archText string) bool {
	return readsCompleteMarker.MatchString(archText)
}

// armedReadsEvents returns every event an armed design's event contract names,
// and nil when the design is unarmed. Gd reads it to stand its opt-in warn
// tier down for those events.
func armedReadsEvents(design string) map[string]bool {
	archText := readDesignOrEmpty(design, filepath.Join(design, "ARCHITECTURE.md"))
	if !readsCompleteArmed(archText) {
		return nil
	}
	out := map[string]bool{}
	for _, r := range eventContractRows(archText) {
		for _, ev := range eventNamesOf(r.Cell("event")) {
			out[ev] = true
		}
	}
	return out
}

type readsEdge struct {
	event, consumer string
}

type consumerReadsLine struct {
	text, events, consumer, where string
	ownerColumns                  int
}

type consumerReadSet struct {
	fields []string // sorted: declarations agree as sets, not ordered lists
	where  string
}

// collectConsumerReads preserves table-local ownership and physical locations.
// Non-table declarations keep the legacy unique-consumer rule. Matrix filenames
// locate diagnostics only: a machine name is not an architectural participant.
func collectConsumerReads(g *Gate, design string) []consumerReadsLine {
	var out []consumerReadsLine
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "payload matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable payload matrix: "+err.Error())
			continue
		}
		lines := strings.Split(string(body), "\n")
		add := func(line string, lineNo int, cells, header []string) {
			// A declaration is a GROUP, READS{...}, which is what v0.6.11
			// collected. READS is also an ordinary English verb, and a matrix
			// uses it as one: a residual cell says the machine "only READS
			// those rows" off an event another layer produces. Collecting the
			// bare word makes that sentence a declaration that can only be
			// reported as an incomplete one. A row that does carry a group is
			// still held to exactly one complete declaration.
			if !strings.Contains(line, "READS{") {
				return
			}
			d := consumerReadsLine{text: line, events: line, where: filepath.Base(path) + ":" + strconv.Itoa(lineNo+1)}
			var eventCells []string
			for i, h := range header {
				if ir.FindCol([]string{h}, "consumer") >= 0 {
					d.ownerColumns++
					d.consumer = ir.CleanCell(cellAt(cells, i))
				} else {
					eventCells = append(eventCells, cellAt(cells, i))
				}
			}
			if len(header) > 0 {
				d.events = strings.Join(eventCells, " | ")
			}
			// A field or consumer spelling is not an event reference.
			d.events = readsDecl.ReplaceAllString(d.events, "")
			out = append(out, d)
		}
		for i := 0; i < len(lines); {
			if !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				add(lines[i], i, nil, nil)
				i++
				continue
			}
			end := i + 1
			for end < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[end]), "|") {
				end++
			}
			tables := ir.ParseMdTables(strings.Join(lines[i:end], "\n"))
			if len(tables) == 0 {
				add(lines[i], i, nil, nil)
			}
			for _, tbl := range tables {
				start := end - len(tbl.Rows)
				for j, cells := range tbl.Rows {
					add(lines[start+j], start+j, cells, tbl.Header)
				}
			}
			i = end
		}
	}
	return out
}

var readsWord = regexp.MustCompile(`\bREADS\b`)

// parseConsumerReadSet rejects empty and duplicate members instead of silently
// discarding them. One row declares exactly one complete set.
//
// A MEMBER is any trimmed, non-empty text, which is the grammar the tier has
// always read. READS members name domain fields in the design's ubiquitous
// language ('occurrence time', 'unsupported-element accounting'), not code
// symbols; an identifier-only rule would narrow what a matrix can say about
// the payload it reads, and the reconciliation against the payload cell is
// whole-token containment, which needs no such rule.
func parseConsumerReadSet(text string) ([]string, string) {
	matches := readsDecl.FindAllStringSubmatch(text, -1)
	if len(matches) != 1 || len(readsWord.FindAllStringIndex(text, -1)) != 1 {
		return nil, "expected exactly one complete READS{field, ...} declaration per row"
	}
	end := readsDecl.FindStringIndex(text)[1]
	if end < len(text) && (text[end] == '}' || text[end] == '{' || isTokenChar(text[end])) {
		return nil, "malformed READS declaration suffix; write READS{field, ...}"
	}
	var fields []string
	seen := map[string]bool{}
	for _, member := range strings.Split(matches[0][1], ",") {
		field := strings.TrimSpace(member)
		if len(field) >= 2 && field[0] == '`' && field[len(field)-1] == '`' {
			field = strings.TrimSpace(field[1 : len(field)-1])
		}
		if field == "" {
			return nil, "empty READS member " + ir.Repr(member) + "; name each field explicitly, and drop the stray separator"
		}
		if seen[field] {
			return nil, "duplicate READS field " + ir.Repr(field)
		}
		seen[field] = true
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields, ""
}

// bindConsumerReads resolves ownership before comparing declarations. A legacy
// declaration remains ambiguous even if explicit declarations cover its siblings.
func bindConsumerReads(g *Gate, design string, rows []eventRow) map[readsEdge]consumerReadSet {
	owners := map[string]map[string]bool{}
	var events []string
	for _, r := range rows {
		for _, ev := range eventNamesOf(r.Cell("event")) {
			if owners[ev] == nil {
				owners[ev] = map[string]bool{}
				events = append(events, ev)
			}
			owners[ev][r.Clean("consumer")] = true
		}
	}
	bound := map[readsEdge]consumerReadSet{}
	for _, d := range collectConsumerReads(g, design) {
		for _, ev := range events {
			if !tokenIn(ev, d.events) {
				continue
			}
			consumer := d.consumer
			switch {
			case d.ownerColumns > 1:
				g.Errs = append(g.Errs, d.where+": READS for event "+ir.Repr(ev)+" has duplicate consumer columns; keep one explicit consumer column")
				continue
			case d.ownerColumns == 1 && (consumer == "" || !owners[ev][consumer]):
				g.Errs = append(g.Errs, d.where+": READS for event "+ir.Repr(ev)+" names consumer "+ir.Repr(consumer)+", not an exact event-contract consumer; name one participant in the consumer column")
				continue
			case d.ownerColumns == 0:
				if len(owners[ev]) != 1 || owners[ev][""] {
					g.Errs = append(g.Errs, d.where+": legacy READS for event "+ir.Repr(ev)+" has ambiguous consumer ownership; add an explicit consumer column matching each event-contract participant")
					continue
				}
				for owner := range owners[ev] {
					consumer = owner // exactly one distinct owner, even for repeated edges
				}
			}
			fields, problem := parseConsumerReadSet(d.text)
			if problem != "" {
				g.Errs = append(g.Errs, d.where+": event "+ir.Repr(ev)+", consumer "+ir.Repr(consumer)+": "+problem)
				continue
			}
			edge := readsEdge{event: ev, consumer: consumer}
			if previous, exists := bound[edge]; exists {
				if strings.Join(previous.fields, ",") != strings.Join(fields, ",") {
					g.Errs = append(g.Errs, d.where+": conflicting READS for event "+ir.Repr(ev)+", consumer "+ir.Repr(consumer)+"; exact field set differs from "+previous.where+" (all rows and machines for this edge must agree)")
				}
				continue
			}
			bound[edge] = consumerReadSet{fields: fields, where: d.where}
		}
	}
	return bound
}

// checkReadsComplete holds every event-contract row of an armed design to a
// consumer READS declaration, and every declared field to the row's payload.
func checkReadsComplete(g *Gate, design, archText string) {
	if !readsCompleteArmed(archText) {
		return // unarmed: the stage-one warn tier is the whole rule
	}
	tables := eventContractTables(archText)
	if len(tables) == 0 {
		g.Errs = append(g.Errs, "ARCHITECTURE.md arms the consumer-READS completeness tier (machinery:reads-complete) but carries no event-contract table; the claim has no subject")
		return
	}
	for _, tbl := range tables {
		if tbl.Cols["event"] < 0 {
			g.Errs = append(g.Errs, "an armed event-contract table has no event column; the completeness tier keys READS declarations by event name, so this table can never satisfy it (name the events, or drop the machinery:reads-complete marker)")
		}
	}
	rows := eventContractRows(archText)
	decls := bindConsumerReads(g, design, rows)
	for _, r := range rows {
		if strings.Contains(r.Cell("consumer"), "(no reads:") {
			waivers := noReadsWaiverRe.FindAllStringSubmatch(r.Cell("consumer"), -1)
			if len(waivers) != 1 || strings.Count(r.Cell("consumer"), "(no reads:") != 1 {
				g.Errs = append(g.Errs, r.Where()+": malformed or duplicate READS waiver; write exactly one '(no reads: <reason>)' in this consumer cell")
				continue
			}
			m := waivers[0]
			if strings.TrimSpace(m[1]) == "" {
				g.Errs = append(g.Errs, r.Where()+": the consumer cell's READS waiver names no reason; write '(no reads: <reason>)'")
			} else {
				g.Count("event-contract consumer reads waived")
			}
			continue
		}
		if r.Cols["event"] < 0 {
			continue // the table's missing column is reported once above
		}
		payload := r.Cell("payload")
		for _, ev := range eventNamesOf(r.Cell("event")) {
			consumer := r.Clean("consumer")
			d, declared := decls[readsEdge{event: ev, consumer: consumer}]
			if !declared {
				g.Count("event-contract consumer reads missing")
				g.Errs = append(g.Errs, r.Where()+": no consumer READS declaration for event "+ir.Repr(ev)+", consumer "+ir.Repr(consumer)+"; state READS{field, ...} beside the event name and name this participant in the matrix consumer column, or waive this consumer cell with '(no reads: <reason>)'")
				continue
			}
			g.Count("event-contract consumer reads declared")
			for _, f := range d.fields {
				if tokenIn(f, payload) {
					g.Count("declared read fields carried")
					continue
				}
				g.Errs = append(g.Errs, d.where+": reads "+f+" from event "+ir.Repr(ev)+", consumer "+ir.Repr(consumer)+", but "+r.Where()+" carries no such field in its payload cell as a whole token; the payload-sufficiency drift (widen the payload, or fix the declaration)")
			}
		}
	}
}
