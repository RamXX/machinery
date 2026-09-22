// Package gates implements deterministic design checks. This file owns the
// Gx fact-resolution join from named-unit prose to machine-readable sources.
package gates

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

var (
	factReference = regexp.MustCompile("`([a-z][A-Za-z0-9_]*_[A-Za-z0-9_]*)`")
	derivedFact   = regexp.MustCompile(`(?i)derived:\s*([a-z][A-Za-z0-9_]*_[A-Za-z0-9_]*)\s*\(([^)]*)\)`)
	derivedWord   = regexp.MustCompile(`(?i)\bderived\s*:`)
)

func declaredFacts(dm *ir.Value, design, archText string) map[string]bool {
	out := map[string]bool{}
	entities := dm.AsObject().GetObject("entities")
	for _, ename := range entities.Keys() {
		for _, attr := range objSlice(entities.Get2(ename).AsObject().Get2("attributes")) {
			if name := attr.AsObject().GetString("name"); name != "" {
				out[name] = true
				out[ename+"."+name] = true
			}
		}
	}
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.machine.json") {
		machine, err := loadDesignMachine(design, path)
		if err != nil {
			continue
		}
		if ctx := machine.AsObject().GetObject("context"); ctx != nil {
			for _, key := range ctx.Keys() {
				out[key] = true
			}
		}
	}
	for _, row := range eventContractRows(archText) {
		fields, why := architecturePayloadFields(row.Cell("payload"))
		if why != "" {
			continue
		}
		for _, field := range fields {
			out[field] = true
			if dot := strings.LastIndex(field, "."); dot >= 0 && dot+1 < len(field) {
				out[field[dot+1:]] = true
			}
		}
	}
	return out
}

func factColumns(header []string) []int {
	var out []int
	for i, raw := range header {
		h := strings.ToLower(strings.TrimSpace(raw))
		if strings.Contains(h, "contract") || strings.Contains(h, "pre / post") || strings.Contains(h, "pre/post") || strings.Contains(h, "payload") || strings.Contains(h, "clause") {
			out = append(out, i)
		}
	}
	return out
}

func checkFactResolution(g *Gate, design, archText string, dm *ir.Value) {
	declared := declaredFacts(dm, design, archText)
	paths, _ := strictSortedGlob(g, filepath.Join(design, "machines"), "*.matrix.md", "fact matrix")
	for _, path := range paths {
		body, err := readDesignFile(design, path)
		if err != nil {
			g.Errs = append(g.Errs, filepath.Base(path)+": unreadable fact matrix: "+err.Error())
			continue
		}
		var cols []int
		for lineNo, line := range strings.Split(string(body), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "|") {
				cols = nil
				continue
			}
			cells := splitTableRow(line)
			candidate := factColumns(cells)
			if ir.FindCol(cells, "name") >= 0 && ir.FindCol(cells, "kind") >= 0 && len(candidate) > 0 {
				cols = candidate
				continue
			}
			if cols == nil || isMarkdownSeparator(cells) {
				continue
			}
			where := filepath.Base(path) + ":" + strconv.Itoa(lineNo+1)
			waived := map[string]bool{}
			for _, col := range cols {
				cell := cellAt(cells, col)
				matches := derivedFact.FindAllStringSubmatch(cell, -1)
				if derivedWord.MatchString(cell) && len(matches) == 0 {
					g.Errs = append(g.Errs, where+": malformed derived waiver; write derived: fact_name (<reason>)")
				}
				for _, match := range matches {
					fact := match[1]
					if strings.TrimSpace(match[2]) == "" {
						g.Errs = append(g.Errs, where+": derived waiver for "+ir.Repr(fact)+" names no reason")
						continue
					}
					waived[fact] = true
					g.Count("derived facts waived")
				}
			}
			seen := map[string]bool{}
			for _, col := range cols {
				for _, match := range factReference.FindAllStringSubmatch(cellAt(cells, col), -1) {
					fact := match[1]
					if seen[fact] {
						continue
					}
					seen[fact] = true
					if declared[fact] {
						g.Count("unit facts resolved")
						continue
					}
					if waived[fact] {
						continue
					}
					g.Errs = append(g.Errs, where+": unresolved fact "+ir.Repr(fact)+"; declare a Modelith attribute, machine context key, or event payload field, or add derived: "+fact+" (<reason>) on this row")
				}
			}
		}
	}
}

func isMarkdownSeparator(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if strings.Trim(cell, ":-") != "" || !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}
