// Payload-sufficiency, stage one (S2 of the dogfood systemic findings, as
// amended). The second most recurrent defect class: a consumer's handler
// writes fields the event payload does not carry, across an asserted
// no_path, caught only by hand sweeps four passes running. The full fix is
// a structured payload schema per event row; this is the staged first
// version the amendment prescribes: a consumer's matrix event row may
// declare what it reads,
//
//	| `order.settled` (consumed) ... READS{settlementClass, declineCause} ... |
//
// and every declared field must appear as a whole token in some event-table
// row for that event somewhere in the hand-written design. Opt-in: an
// undeclared consumption is untouched; a declared read that no payload cell
// carries is the drift, and warns.
//
// A design that wants the obligation the other way round (every consumer row
// MUST declare) arms the completeness tier in readscomplete.go, which judges
// the same declarations at ERROR strength in Gx-trace. This tier stands down
// for the events an armed contract names, so one defect earns one finding.

package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var readsDecl = regexp.MustCompile(`READS\{([^}]*)\}`)
var leadingEventName = regexp.MustCompile("^\\s*\\|\\s*`([a-z][a-z0-9_]*\\.[a-z][a-z0-9_.]*)`")
var backtickedEventName = regexp.MustCompile("`([a-z][a-z0-9_]*\\.[a-z][a-z0-9_.]*)`")

type readsClaim struct {
	event  string
	fields []string
	where  string
}

// readsLine is one matrix line carrying a READS declaration, with the fields
// it declares and where it sits. The line itself is kept because the two
// tiers key declarations differently: this tier by the backticked dotted
// event name in it, the completeness tier by whichever event name the row
// under judgment spells (see readscomplete.go).
type readsLine struct {
	line   string
	fields []string
	where  string
}

// collectReadsLines finds every READS declaration under design/machines.
func collectReadsLines(design string) []readsLine {
	var out []readsLine
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, ok := readTextOK(path)
		if !ok {
			continue
		}
		base := filepath.Base(path)
		for lineNo, line := range strings.Split(body, "\n") {
			m := readsDecl.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			out = append(out, readsLine{
				line:   line,
				fields: splitClauses(m[1]),
				where:  base + ":" + strconv.Itoa(lineNo+1),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].where < out[j].where })
	return out
}

// collectReadsDecls keeps the READS declarations on matrix lines that name an
// event in backticks, which is what this tier can resolve payloads for.
func collectReadsDecls(design string) []readsClaim {
	var out []readsClaim
	for _, l := range collectReadsLines(design) {
		ev := backtickedEventName.FindStringSubmatch(l.line)
		if ev == nil {
			continue
		}
		out = append(out, readsClaim{event: ev[1], fields: l.fields, where: l.where})
	}
	return out
}

// payloadTextFor gathers the full text of every event-table row for an
// event across the hand-written design: rows whose FIRST cell is the
// backticked event name and that have at least six cells (the event-table
// shape, so a mention in narrow tables does not count as a payload).
func payloadTextFor(design string) map[string][]string {
	out := map[string][]string{}
	_ = filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil //nolint:nilerr // keep walking; the audit covers what is readable
		}
		if fi.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if idciteSkips(rel) || strings.ToLower(filepath.Ext(fi.Name())) != ".md" {
			return nil
		}
		body, ok := readTextOK(path)
		if !ok {
			return nil
		}
		for _, line := range strings.Split(body, "\n") {
			m := leadingEventName.FindStringSubmatch(line)
			if m == nil || strings.Count(line, "|") < 7 {
				continue
			}
			out[m[1]] = append(out[m[1]], line)
		}
		return nil
	})
	return out
}

// checkPayloadReads warns for every declared read no payload row carries.
func checkPayloadReads(g *Gate, design string) {
	decls := collectReadsDecls(design)
	if len(decls) == 0 {
		return
	}
	// A design that arms the completeness tier hands those events to Gx-trace,
	// which judges the same declarations against the same payloads at ERROR
	// strength (readscomplete.go). Standing down here is per event, not
	// wholesale: a declaration naming an event the armed contract never lists
	// is nobody else's, and keeps its opt-in warn.
	if armed := armedReadsEvents(design); len(armed) > 0 {
		var kept []readsClaim
		for _, d := range decls {
			if !armed[d.event] {
				kept = append(kept, d)
			}
		}
		decls = kept
		if len(decls) == 0 {
			return
		}
	}
	g.Count("payload READS declarations", len(decls))
	payloads := payloadTextFor(design)
	for _, d := range decls {
		rows := payloads[d.event]
		if len(rows) == 0 {
			g.Warns = append(g.Warns, d.where+": declares reads from `"+d.event+"`, but no event-table row for it exists in the hand-written design")
			continue
		}
		for _, f := range d.fields {
			carried := false
			for _, r := range rows {
				if tokenIn(f, r) {
					carried = true
				}
			}
			if !carried {
				g.Warns = append(g.Warns, d.where+": reads "+f+" from `"+d.event+"`, but no event-table row for it carries that field as a whole token; the payload-sufficiency drift (widen the payload, or fix the declaration)")
			}
		}
	}
}
