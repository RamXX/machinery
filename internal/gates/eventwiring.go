// Event-wiring reconciliation (Gx-trace). The pack gate already holds a
// subsystem's boundary events to its machines (checkBoundaryEvents in pack.go):
// a consumed event no machine handles or ignores is an ERROR, and a produced
// event no machine action and no matrix row fires is an ERROR. That check runs
// only where a pack exists. On an ORDINARY design the same event-contract table
// and the same machines sit side by side in the same tree and nothing held them
// to each other: the FSM author attested the wiring and the gate counted rows.
//
// HOST GATE: Gx-trace, not G2.
//
// The event table is G2's artifact, the machines are G3's, and this check needs
// both. Gx-trace is where every other check that joins an ARCHITECTURE.md table
// to the committed machines already lives: the placement table's machine
// obligation and the mitigation table's residual-handler column are both here
// (see checkPlacement and checkResidualHandling), and the skill documents the
// mitigation join as running "in Gx-trace once machines exist" for exactly this
// reason. Hosting it in G2 would break phase ordering: a Phase 2 design has an
// authored architecture and no machines yet, and G2 must stay green there.
// Gx also narrows itself away on a decomposed parent with no machines, which is
// precisely where the pack side owns the question instead.
//
// SEMANTICS, mirroring the pack side cell for cell:
//
//   - consumed: the event is handled (`on:`) or explicitly ignored (`_ignores`)
//     by some machine in the design;
//   - produced: the event appears whole-token in some machine action position
//     (entry, exit, transition, invoke src) or some matrix cell.
//
// WHERE THE TWO DISAGREE, the pack side wins and the difference is stated here:
//
//  1. Direction. The pack reads an explicit `direction` column, because a pack
//     is written from one subsystem's point of view. An ordinary design has no
//     such column and needs none: every participant in its table is a component
//     of this design, so the design owes BOTH the emission and the reaction for
//     every row. Applying both obligations to every row is the pack rule with
//     its point of view set to "all of it".
//  2. Event names. The pack reads the event column with ir.CleanCell, which is
//     exact because its format contract forbids anything but a bare event name.
//     An ordinary table's event cell is prose that names its events backticked
//     ("`reserve` command", "`reserved` / `released` events"), the same idiom
//     the mitigation and placement rows use, so backticked tokens are read when
//     the cell has any and CleanCell is the fallback. On a pack-format cell
//     (no backticks, one bare name) this reduces to exactly the pack's rule.
//  3. Machine scope. The pack asks whether ANY machine in the design handles
//     the event, never which one; so does this. Requiring the machine NAMED by
//     the consumer cell would be a stricter rule the pack does not have.
//
// THE REVERSE SWEEP is opt-in by marker: a machine's `_external_events` array
// declares which of its events arrive from outside its component, and every
// declared event owes an event-contract row (an event crossing into the
// design is coupling the table exists to govern). Without the marker the
// reverse rule stays attested, exactly as before: inferring external sourcing
// (an event no local action fires) would demand a contract row for every
// human-initiated event, a CLI command included. Lint holds the marker itself
// (a declared event no state handles is stale).
//
// A row whose obligation the design cannot meet says so in the row, with the
// house waiver token and a reason: `(no machine: <reason>)` in the consumer
// cell waives the reaction, in the producer cell the emission. It is the same
// token, with the same meaning, that a placement row uses for a component with
// no machine; an empty reason is an unanswered question, not a waiver.

package gates

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// machineEventCorpus is what the design's machines say about events: the set
// of event names any machine handles or explicitly ignores, and the corpus of
// action-position text plus matrix cells a produced event must appear in.
// Machines that fail to load contribute nothing (G3 reports the parse error).
func machineEventCorpus(design string, g *Gate) (handled map[string]bool, fired string, external map[string]bool) {
	handled = map[string]bool{}
	external = map[string]bool{}
	var cells []string
	mdir := filepath.Join(design, "machines")
	for _, mf := range sortedGlobExt(mdir, ".machine.json") {
		m, err := ir.LoadMachineJSON(mf)
		if err != nil {
			continue
		}
		if mo := m.AsObject(); mo != nil {
			if extV := mo.Get2("_external_events"); extV != nil {
				for _, e := range extV.AsArray() {
					if name := strings.TrimSpace(e.AsString()); name != "" {
						external[name] = true
					}
				}
			}
		}
		for _, s := range ir.WalkStates(m.AsObject().Get2("states"), "") {
			so := s.Node.AsObject()
			if so == nil {
				continue
			}
			for _, k := range so.GetObject("on").Keys() {
				handled[k] = true
			}
			for _, k := range so.GetObject("_ignores").Keys() {
				handled[k] = true
			}
			// every action position counts as emission evidence: entry and
			// exit actions and invoke actors emit too, not just transitions
			for a := range ir.ActionsOf(s.Node, nil, s.Path) {
				cells = append(cells, a)
			}
			for _, inv := range ir.InvokesOf(s.Node) {
				if io := inv.AsObject(); io != nil {
					if src := io.GetString("src"); src != "" {
						cells = append(cells, src)
					}
				}
			}
		}
	}
	for _, f := range sortedGlobExt(mdir, ".matrix.md") {
		for _, tbl := range ir.ParseMdTables(readFileOrErr(f, g)) {
			for _, r := range tbl.Rows {
				cells = append(cells, r...)
			}
		}
	}
	return handled, strings.Join(cells, "\n"), external
}

// eventNamesOf reads the event names a row's event cell states: the backticked
// tokens when the cell has any (the prose form), otherwise the cleaned cell as
// one name (the pack format).
func eventNamesOf(cell string) []string {
	if toks := backtickTokens(cell); len(toks) > 0 {
		return toks
	}
	if name := ir.CleanCell(cell); name != "" {
		return []string{name}
	}
	return nil
}

// checkEventWiring holds every event-contract row to the committed machines,
// and every declared externally-sourced machine event to the table.
func checkEventWiring(g *Gate, design, archText string) {
	rows := eventContractRows(archText)
	handled, fired, external := machineEventCorpus(design, g)
	// the reverse sweep, armed per event by the machine-side declaration; it
	// runs even with no table at all, because a declared external event with
	// no contract row anywhere is exactly the finding
	if len(external) > 0 {
		rowEvents := map[string]bool{}
		for _, r := range rows {
			if r.Cols["event"] >= 0 {
				for _, ev := range eventNamesOf(r.Cell("event")) {
					rowEvents[ev] = true
				}
			}
		}
		var names []string
		for ev := range external {
			names = append(names, ev)
		}
		sort.Strings(names)
		for _, ev := range names {
			if rowEvents[ev] {
				g.Count("externally-sourced events with contract rows")
			} else {
				g.Errs = append(g.Errs, "machine event "+ir.Repr(ev)+" is declared externally sourced (_external_events) but no event-contract row names it; an event crossing into this design is coupling the table exists to govern")
			}
		}
	}
	if len(rows) == 0 {
		return
	}
	for _, r := range rows {
		if r.Cols["event"] < 0 {
			// no event column: the row names no event to reconcile. G2 does
			// not require the column (the pack does, on the designs where
			// generation reads the table), so this is a stated non-check
			// rather than a finding here.
			g.Count("event-contract rows without an event column")
			continue
		}
		names := eventNamesOf(r.Cell("event"))
		if len(names) == 0 {
			// an empty event cell is G2's finding (checkEventCells); nothing
			// to reconcile, and one defect never earns a second finding
			continue
		}
		for _, side := range []struct{ col, obligation string }{
			{"consumer", "reaction"},
			{"producer", "emission"},
		} {
			cell := r.Cell(side.col)
			if m := noMachineWaiverRe.FindStringSubmatch(cell); m != nil {
				if strings.TrimSpace(m[1]) == "" {
					g.Errs = append(g.Errs, r.Where()+": the "+side.col+" cell's waiver names no reason; write '(no machine: <reason>)'")
				} else {
					g.Count("event-contract " + side.obligation + "s waived")
				}
				continue
			}
			for _, ev := range names {
				switch {
				case side.col == "consumer" && !handled[ev]:
					g.Errs = append(g.Errs, r.Where()+": event "+ir.Repr(ev)+" is handled or ignored by no machine, and the consumer cell carries no '(no machine: <reason>)' waiver; the table says a component reacts to it and the behavior layer says nothing does")
				case side.col == "producer" && !tokenIn(ev, fired):
					g.Errs = append(g.Errs, r.Where()+": event "+ir.Repr(ev)+" appears in no machine action (entry, exit, transition, invoke src) and no matrix cell, and the producer cell carries no '(no machine: <reason>)' waiver; if the emitting action is named differently, name the event whole-token in that action's matrix row")
				default:
					g.Count("event-contract " + side.obligation + "s traced")
				}
			}
		}
	}
}
