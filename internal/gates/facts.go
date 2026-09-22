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
	factReference     = regexp.MustCompile("`([A-Za-z][A-Za-z0-9_.]*)`")
	derivedFact       = regexp.MustCompile(`(?i)^derived:\s*([A-Za-z][A-Za-z0-9_.]*)\s*\(([^)]*)\)`)
	derivedWord       = regexp.MustCompile(`(?i)\bderived\s*:`)
	valueRole         = regexp.MustCompile(`(?i)(?:reason class|failure class|closure basis|failing clause|disposition|prior status|subject kind|kind|marker|outcome|only for|marks?(?:\s+\w+){0,3}|classifies?\s+a)\s*(?:is|of|,|:)?\s*$`)
	valueSequence     = regexp.MustCompile(`(?i)(?:reason|failure) class[^.;]{0,160}$`)
	valueSuffix       = regexp.MustCompile(`(?i)^\s*(?:derivation\b|with no basis\b)`)
	producerRole      = regexp.MustCompile(`(?i)(?:closed\s+|only\s+)?producer\s*(?:is|:)?\s*$`)
	negativeFact      = regexp.MustCompile(`(?i)(?:there is no|never|not|without)\s*$`)
	prosePayloadField = regexp.MustCompile(`\b[a-z][a-z0-9]*(?:_[a-z0-9]+)+\b`)
	knobKey           = regexp.MustCompile(`\bkey:\s*([a-z][a-z0-9_]*)\s*,\s*class:\s*C\b`)
	verticalField     = regexp.MustCompile(`(?m)^\s+([a-z][a-z0-9]*(?:_[a-z0-9]+)+):`)
	joinPair          = regexp.MustCompile("(?i)\\b(?:keyed by|the) pair\\s*\\(`([a-z][a-z0-9_]*)`,\\s*`([a-z][a-z0-9_]*)`\\)")
)

type factUniverse struct {
	declared             map[string]bool
	entities             map[string]bool
	untypedMaps          map[string]map[string]bool
	snake, camel, single bool
}

func declaredFacts(dm *ir.Value, design, archText string) factUniverse {
	out := map[string]bool{}
	universe := factUniverse{declared: out, entities: map[string]bool{}, untypedMaps: map[string]map[string]bool{}}
	entities := dm.AsObject().GetObject("entities")
	for _, ename := range entities.Keys() {
		universe.entities[ename] = true
		for _, attr := range objSlice(entities.Get2(ename).AsObject().Get2("attributes")) {
			if attr.AsObject().GetString("type") == "string" {
				desc := attr.AsObject().GetString("description")
				const intro = "KEY NAMES ARE CLOSED AND STATED HERE:"
				if at := strings.Index(desc, intro); at >= 0 {
					segment := desc[at+len(intro):]
					if end := strings.Index(segment, ". A key"); end >= 0 {
						segment = segment[:end]
					}
					keys := map[string]bool{}
					for _, match := range factReference.FindAllStringSubmatch(segment, -1) {
						keys[match[1]] = true
					}
					if len(keys) > 0 {
						universe.untypedMaps[ename+"."+attr.AsObject().GetString("name")] = keys
					}
				}
			}
			if name := attr.AsObject().GetString("name"); name != "" {
				out[name] = true
				out[ename+"."+name] = true
				if strings.Contains(name, "_") {
					universe.snake = true
				} else if strings.ToLower(name) != name {
					universe.camel = true
				} else {
					universe.single = true
				}
			}
		}
		for _, action := range objSlice(entities.Get2(ename).AsObject().Get2("actions")) {
			if name := action.AsObject().GetString("name"); name != "" {
				out[name] = true
				out[ename+"."+name] = true
			}
		}
		for _, relation := range objSlice(entities.Get2(ename).AsObject().Get2("relationships")) {
			target := relation.AsObject().GetString("entity")
			if target == "" || entities.Get2(target) == nil {
				continue
			}
			name := strings.ToLower(target[:1]) + target[1:]
			out[ename+"."+name] = true
			if relation.AsObject().GetString("cardinality") == "n:1" {
				out[name+"_id"] = true
				out[ename+"."+name+"_id"] = true
			}
		}
	}
	for _, members := range modelEnumValues(dm) {
		for _, member := range members {
			out[member] = true
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
		collectMachineStateNames(machine, out)
	}
	for _, row := range eventContractRows(archText) {
		for _, event := range eventNamesOf(row.Cell("event")) {
			out[event] = true
		}
		fields, why := architecturePayloadFields(row.Cell("payload"))
		if why != "" {
			// A contract may spell payload members as prose rather than a
			// tokenized group. Snake-case names in that payload cell still
			// declare fields; prose in other cells does not.
			fields = prosePayloadField.FindAllString(row.Cell("payload"), -1)
		}
		for _, field := range fields {
			out[field] = true
			if dot := strings.LastIndex(field, "."); dot >= 0 && dot+1 < len(field) {
				out[field[dot+1:]] = true
			}
		}
	}
	if body, err := readDesignFile(design, filepath.Join(design, "content", "knob-register.yaml")); err == nil {
		for _, match := range knobKey.FindAllStringSubmatch(string(body), -1) {
			out[match[1]] = true
		}
	}
	for _, path := range sortedGlob(filepath.Join(design, "content", "verticals"), "*.vertical.yaml") {
		body, err := readDesignFile(design, path)
		if err != nil {
			continue
		}
		for _, match := range verticalField.FindAllStringSubmatch(string(body), -1) {
			out[match[1]] = true
		}
	}
	for _, match := range joinPair.FindAllStringSubmatch(archText, -1) {
		out[match[1]], out[match[2]] = true, true
	}
	for _, path := range sortedGlob(filepath.Join(design, "machines"), "*.matrix.md") {
		body, err := readDesignFile(design, path)
		if err != nil {
			continue
		}
		owner := strings.TrimSuffix(filepath.Base(path), ".matrix.md")
		for _, table := range ir.ParseMdTables(string(body)) {
			failureCol := ir.FindCol(table.Header, "failure")
			nameCol, kindCol := ir.FindCol(table.Header, "name"), ir.FindCol(table.Header, "kind")
			for _, row := range table.Rows {
				if nameCol >= 0 && kindCol >= 0 {
					kind := strings.ToLower(ir.CleanCell(cellAt(row, kindCol)))
					if kind == "action" || kind == "actor" || kind == "guard" {
						for _, match := range factReference.FindAllStringSubmatch(cellAt(row, nameCol), -1) {
							name := match[1]
							out[name] = true
							out[owner+"."+name] = true
						}
					}
				}
				if failureCol >= 0 {
					for _, cell := range row {
						for _, match := range factReference.FindAllStringSubmatch(cell, -1) {
							out[match[1]] = true
						}
					}
					if name := ir.CleanCell(cellAt(row, failureCol)); valueMember.MatchString(name) {
						out[name] = true
					}
				}
			}
		}
	}
	return universe
}

func collectMachineStateNames(machine *ir.Value, out map[string]bool) {
	if machine == nil || machine.Kind != ir.KindObject {
		return
	}
	states := machine.AsObject().GetObject("states")
	if states == nil {
		return
	}
	for _, name := range states.Keys() {
		out[name] = true
		collectMachineStateNames(states.Get2(name), out)
	}
}

func (u factUniverse) candidate(name, cell string, start int) bool {
	if valueOrProducerReference(cell, start, name) {
		return false
	}
	if negativeFactReference(cell, start) {
		return false
	}
	if dot := strings.IndexByte(name, '.'); dot >= 0 {
		return dot+1 < len(name) && u.entities[name[:dot]] && !strings.Contains(name[dot+1:], ".") &&
			name[dot+1] >= 'a' && name[dot+1] <= 'z'
	}
	if name[0] < 'a' || name[0] > 'z' {
		return false
	}
	if strings.Contains(name, "_") {
		return true // keep the established snake-case grammar beside model-derived styles
	}
	if strings.ToLower(name) != name {
		return u.camel
	}
	if !u.single {
		return false
	}
	// Single-word names overlap ordinary labels. Require an explicit fact verb
	// immediately before the quoted name to keep those labels out of scope.
	prefix := strings.ToLower(strings.TrimSpace(cell[:start]))
	for _, verb := range []string{"persists", "stores", "writes", "records", "reads", "uses", "consumes", "emits", "requires", "pins"} {
		if strings.HasSuffix(prefix, verb) {
			return true
		}
	}
	return false
}

func negativeFactReference(cell string, start int) bool {
	if negativeFact.MatchString(strings.TrimSpace(cell[:start])) {
		return true
	}
	left := strings.LastIndexAny(cell[:start], ".;") + 1
	right := len(cell)
	if next := strings.IndexAny(cell[start:], ".;"); next >= 0 {
		right = start + next
	}
	segment := strings.ToLower(cell[left:right])
	return strings.Contains(segment, "weighed and refused") ||
		(strings.Contains(segment, "earlier reading") && strings.Contains(segment, "proves"))
}

func valueOrProducerReference(cell string, start int, name string) bool {
	before := strings.TrimSpace(cell[:start])
	if valueRole.MatchString(before) || producerRole.MatchString(before) || valueSequence.MatchString(before) {
		return true
	}
	end := start + len(name) + 2
	if end <= len(cell) && valueSuffix.MatchString(cell[end:]) {
		return true
	}
	open, close := strings.LastIndex(before, "("), strings.LastIndex(before, ")")
	if open > close && valueRole.MatchString(strings.TrimSpace(before[:open])) {
		return true
	}
	return false
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
	universe := declaredFacts(dm, design, archText)
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
			g.Count("named-unit fact rows scanned")
			waived := map[string]bool{}
			rowFacts := map[string]bool{}
			rowValues := map[string]bool{}
			for _, col := range cols {
				for _, group := range valuesGroup.FindAllStringSubmatch(cellAt(cells, col), -1) {
					if values, why := parseValues(group[2]); why == "" {
						for _, value := range values {
							rowValues[value] = true
						}
					}
				}
			}
			for _, col := range cols {
				cell := cellAt(cells, col)
				for _, loc := range factReference.FindAllStringSubmatchIndex(cell, -1) {
					fact := cell[loc[2]:loc[3]]
					if universe.candidate(fact, cell, loc[0]) {
						rowFacts[fact] = true
					}
				}
			}
			for _, col := range cols {
				cell := cellAt(cells, col)
				starts := derivedWord.FindAllStringIndex(cell, -1)
				for _, at := range starts {
					match := derivedFact.FindStringSubmatch(cell[at[0]:])
					if match == nil {
						g.Errs = append(g.Errs, where+": malformed derived waiver; write derived: fact_name (<reason>)")
						continue
					}
					fact := match[1]
					if strings.TrimSpace(match[2]) == "" {
						g.Errs = append(g.Errs, where+": derived waiver for "+ir.Repr(fact)+" names no reason")
						continue
					}
					if !rowFacts[fact] {
						g.Errs = append(g.Errs, where+": derived waiver for "+ir.Repr(fact)+" names no fact on this row")
						continue
					}
					waived[fact] = true
					g.Count("derived facts waived")
				}
			}
			seen := map[string]bool{}
			mapGap := ""
			rowText := strings.Join(cells, " ")
			for _, col := range cols {
				cell := cellAt(cells, col)
				for _, loc := range factReference.FindAllStringSubmatchIndex(cell, -1) {
					fact := cell[loc[2]:loc[3]]
					if !universe.candidate(fact, cell, loc[0]) {
						continue
					}
					if seen[fact] {
						continue
					}
					seen[fact] = true
					if universe.declared[fact] || rowValues[fact] {
						g.Count("unit facts resolved")
						continue
					}
					if waived[fact] {
						continue
					}
					for attr, keys := range universe.untypedMaps {
						if keys[fact] && strings.Contains(rowText, "`"+strings.TrimPrefix(attr, strings.SplitN(attr, ".", 2)[0]+".")+"`") {
							mapGap = attr
							break
						}
					}
					if mapGap != "" && universe.untypedMaps[mapGap][fact] {
						continue
					}
					g.Errs = append(g.Errs, where+": unresolved fact "+ir.Repr(fact)+"; declare a Modelith attribute, enum member, action, machine context key or unit, event name or payload field, same-row VALUES member, or add derived: "+fact+" (<reason>) on this row")
				}
			}
			if mapGap != "" {
				g.Errs = append(g.Errs, where+": "+mapGap+" needs a typed map for its closed keys; a string attribute description is not a structured member declaration")
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
