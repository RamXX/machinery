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
package gates

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var readsDecl = regexp.MustCompile("READS\\{([^}]*)\\}")
var leadingEventName = regexp.MustCompile("^\\s*\\|\\s*`([a-z][a-z0-9_]*\\.[a-z][a-z0-9_.]*)`")

type readsClaim struct {
	event  string
	fields []string
	where  string
}

// collectReadsDecls finds READS declarations on matrix lines that name an
// event in backticks.
func collectReadsDecls(design string) []readsClaim {
	var out []readsClaim
	eventToken := regexp.MustCompile("`([a-z][a-z0-9_]*\\.[a-z][a-z0-9_.]*)`")
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		base := filepath.Base(path)
		for lineNo, line := range strings.Split(string(body), "\n") {
			m := readsDecl.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			ev := eventToken.FindStringSubmatch(line)
			if ev == nil {
				continue
			}
			out = append(out, readsClaim{
				event:  ev[1],
				fields: splitClauses(m[1]),
				where:  base + ":" + strconv.Itoa(lineNo+1),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].where < out[j].where })
	return out
}

// payloadTextFor gathers the full text of every event-table row for an
// event across the hand-written design: rows whose FIRST cell is the
// backticked event name and that have at least six cells (the event-table
// shape, so a mention in narrow tables does not count as a payload).
func payloadTextFor(design string) map[string][]string {
	out := map[string][]string{}
	_ = filepath.Walk(design, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(design, path)
		if rerr != nil {
			rel = path
		}
		if idciteSkips(rel) || strings.ToLower(filepath.Ext(fi.Name())) != ".md" {
			return nil
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for _, line := range strings.Split(string(body), "\n") {
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
