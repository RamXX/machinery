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
//   - otherwise some matrix line naming the event whole-token declares
//     READS{field, ...}, or the row is an ERROR.
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

// readsDeclsNaming returns the READS-declaring matrix lines that name ev
// whole-token. The locator is deliberately the design's own spelling of the
// event (backticked in prose, bare in the pack format), so a diagnosis lands
// on a row rather than on a naming convention.
func readsDeclsNaming(lines []readsLine, ev string) []readsLine {
	var out []readsLine
	for _, l := range lines {
		if tokenIn(ev, l.line) {
			out = append(out, l)
		}
	}
	return out
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
	decls := collectReadsLines(g, design)
	for _, r := range eventContractRows(archText) {
		if m := noReadsWaiverRe.FindStringSubmatch(r.Cell("consumer")); m != nil {
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
			naming := readsDeclsNaming(decls, ev)
			if len(naming) == 0 {
				g.Count("event-contract consumer reads missing")
				g.Errs = append(g.Errs, r.Where()+": no consumer READS declaration for event "+ir.Repr(ev)+"; this design arms the consumer-READS completeness tier, so the consuming matrix row states what it reads (READS{field, ...}) beside the event name, or the consumer cell waives with '(no reads: <reason>)'")
				continue
			}
			g.Count("event-contract consumer reads declared")
			for _, d := range naming {
				for _, f := range d.fields {
					if tokenIn(f, payload) {
						g.Count("declared read fields carried")
						continue
					}
					g.Errs = append(g.Errs, d.where+": reads "+f+" from event "+ir.Repr(ev)+", but "+r.Where()+" carries no such field in its payload cell as a whole token; the payload-sufficiency drift (widen the payload, or fix the declaration)")
				}
			}
		}
	}
}
