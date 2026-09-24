// Package gates implements deterministic design checks. This file owns
// event-payload twin reconciliation in Gx-trace.
//
// A matrix may restate an architecture event payload only as one row-local
// declaration:
//
//	payload {Order.id, Order.paidAt}
//	payload is exactly {Order.id, Order.paidAt}
//
// The declaration is a closed set. It binds to the one event named by that
// matrix row and must equal the Architecture Contract event row's payload
// field set. Ordinary payload prose remains prose and creates no twin.
package gates

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	payloadDeclaration = regexp.MustCompile(`(?i)payload(?:\s+is\s+exactly)?\s*\{([^}]*)\}`)
	payloadOpening     = regexp.MustCompile(`(?i)\bpayload(?:\s+is\s+exactly)?\s*\{`)
	payloadField       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.]*$`)
	payloadBacktick    = regexp.MustCompile("`([A-Za-z][A-Za-z0-9_.]*)`")
)

type payloadTwinDeclaration struct {
	event, where string
	fields       []string
}

func parseClosedFieldSet(body string) ([]string, string) {
	if strings.TrimSpace(body) == "" {
		return nil, "payload declaration has no fields"
	}
	seen := map[string]bool{}
	var fields []string
	for _, raw := range strings.Split(body, ",") {
		field := strings.TrimSpace(raw)
		if len(field) >= 2 && field[0] == '`' && field[len(field)-1] == '`' {
			field = strings.TrimSpace(field[1 : len(field)-1])
		}
		if field == "" {
			return nil, "payload declaration contains an empty member"
		}
		if !payloadField.MatchString(field) {
			return nil, "payload declaration field " + ir.Repr(field) + " is not an identifier; use dot-qualified ubiquitous-language names"
		}
		if seen[field] {
			return nil, "payload declaration has duplicate field " + ir.Repr(field)
		}
		seen[field] = true
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields, ""
}

func architecturePayloadFields(cell string) ([]string, string) {
	var toks []string
	for _, match := range payloadBacktick.FindAllStringSubmatch(cell, -1) {
		toks = append(toks, match[1])
	}
	if len(toks) > 0 {
		return uniqueSortedPayloadFields(toks)
	}
	parts := strings.FieldsFunc(cell, func(r rune) bool { return r == ',' || r == ';' })
	var fields []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || !payloadField.MatchString(part) {
			return nil, "payload cell contains non-identifier field " + ir.Repr(part) + "; backtick each field or use a comma-separated identifier list"
		}
		fields = append(fields, part)
	}
	if len(fields) == 0 {
		return nil, "payload cell states no closed field set; backtick each field or use a comma-separated identifier list"
	}
	return uniqueSortedPayloadFields(fields)
}

func uniqueSortedPayloadFields(fields []string) ([]string, string) {
	seen := map[string]bool{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			return nil, "payload field set contains an empty member"
		}
		if seen[field] {
			return nil, "payload field set has duplicate field " + ir.Repr(field)
		}
		seen[field] = true
		out = append(out, field)
	}
	sort.Strings(out)
	return out, ""
}

func collectPayloadTwinDeclarations(g *Gate, design string) []payloadTwinDeclaration {
	var out []payloadTwinDeclaration
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "payload twin matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable payload twin matrix: "+err.Error())
			continue
		}
		lines := strings.Split(string(body), "\n")
		for lineNo, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") || !payloadOpening.MatchString(line) {
				continue
			}
			matches := payloadDeclaration.FindAllStringSubmatch(line, -1)
			if len(matches) != 1 {
				g.Errs = append(g.Errs, filepath.Base(path)+":"+strconv.Itoa(lineNo+1)+": payload twin row must carry exactly one complete payload {field, ...} or payload is exactly {field, ...} declaration")
				continue
			}
			fields, why := parseClosedFieldSet(matches[0][1])
			if why != "" {
				g.Errs = append(g.Errs, filepath.Base(path)+":"+strconv.Itoa(lineNo+1)+": "+why)
				continue
			}
			var eventCell string
			for _, tbl := range ir.ParseMdTables(string(body)) {
				ei := ir.FindCol(tbl.Header, "event")
				if ei < 0 {
					continue
				}
				for i, row := range tbl.Rows {
					if strings.TrimSpace(tbl.RowLines[i]) == strings.TrimSpace(line) {
						eventCell = cellAt(row, ei)
					}
				}
			}
			events := eventNamesOf(eventCell)
			where := filepath.Base(path) + ":" + strconv.Itoa(lineNo+1)
			if len(events) != 1 {
				g.Errs = append(g.Errs, where+": payload declaration names "+strconv.Itoa(len(events))+" events; one closed payload set must bind to exactly one event row")
				continue
			}
			out = append(out, payloadTwinDeclaration{event: events[0], where: where, fields: fields})
		}
	}
	return out
}

func checkPayloadTwins(g *Gate, design, archText string) {
	decls := collectPayloadTwinDeclarations(g, design)
	if len(decls) == 0 {
		return
	}
	arch := map[string][]string{}
	archWhere := map[string]string{}
	for _, row := range eventContractRows(archText) {
		for _, event := range eventNamesOf(row.Cell("event")) {
			fields, why := architecturePayloadFields(row.Cell("payload"))
			if why != "" {
				g.Errs = append(g.Errs, row.Where()+": cannot reconcile a matrix payload declaration: "+why+" (architecture spelling "+ir.Repr(row.Cell("payload"))+")")
				continue
			}
			if prior, ok := arch[event]; ok && strings.Join(prior, "\x00") != strings.Join(fields, "\x00") {
				g.Errs = append(g.Errs, row.Where()+": event "+ir.Repr(event)+" has conflicting architecture payload twins "+fmt.Sprint(prior)+" and "+fmt.Sprint(fields))
				continue
			}
			arch[event] = fields
			archWhere[event] = row.Where()
		}
	}
	seen := map[string][]string{}
	for _, decl := range decls {
		if prior, ok := seen[decl.event]; ok && strings.Join(prior, "\x00") != strings.Join(decl.fields, "\x00") {
			g.Errs = append(g.Errs, decl.where+": event "+ir.Repr(decl.event)+" has conflicting matrix payload declarations "+fmt.Sprint(prior)+" and "+fmt.Sprint(decl.fields))
			continue
		}
		seen[decl.event] = decl.fields
		fields, ok := arch[decl.event]
		if !ok {
			g.Errs = append(g.Errs, decl.where+": payload declaration for event "+ir.Repr(decl.event)+" has no architecture event-contract payload twin")
			continue
		}
		if strings.Join(fields, "\x00") != strings.Join(decl.fields, "\x00") {
			g.Errs = append(g.Errs, decl.where+": payload twin mismatch for event "+ir.Repr(decl.event)+": matrix spells "+fmt.Sprint(decl.fields)+", "+archWhere[decl.event]+" spells "+fmt.Sprint(fields))
			continue
		}
		g.Count("event payload twins reconciled")
	}
}
