// Package gates implements deterministic design checks. This file is the
// fact reader behind projection v2 and `machinery project --facts`
// (docs/consistency-layer-proposal.md, section 3.2). It lives in gates, and
// not in internal/checker, because it reuses the design readers the gates
// already own (the matrix and contract declaration parsers, the event-table
// and authorization-table column rules, the build-plan milestone parser) and
// gates already imports checker; the reverse import would be a cycle.
//
// Nothing here is a gate. The reader projects what the design states and
// validates only what it must to project it: the grammar of a declaration
// group, stable-id uniqueness, and symbol safety. Whether a projected fact is
// coherent (an unknown oracle id cited by a milestone, a unit writing an
// undeclared attribute) is for the gates and the rules to decide.
//
// Prose never becomes a fact. A column is an identifier or an enumerated
// value: a Modelith definition or description, an invariant statement, the
// reason of a derived: waiver or of a (no authorization: ...) waiver, and a
// payload cell that states no closed field set are never projected.
package gates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/oracle"
)

// factBuilder accumulates the facts of one design and every problem found
// reading them. Problems are collected rather than returned at the first one,
// so a single run names every unprojectable declaration.
type factBuilder struct {
	design string
	facts  *checker.DesignFacts
	errs   []string
	// oracleRows is the committed oracle corpus, shared by the oracles and
	// milestones layers.
	oracleRows []projectedOracleRow
}

type projectedOracleRow struct {
	rel, machine, testID, stableID string
	line                           int
}

// LoadDesignFacts reads every fact layer the design has. It fails when the
// design cannot be projected: an unreadable or malformed source, a malformed
// declaration group, a duplicate stable id (both sources named), or a value a
// fact symbol cannot hold.
func LoadDesignFacts(design string) (*checker.DesignFacts, error) {
	abs, err := filepath.Abs(design)
	if err != nil {
		return nil, err
	}
	b := &factBuilder{design: abs, facts: checker.NewDesignFacts()}
	b.model()
	b.oracles()
	b.machines()
	b.matrices()
	b.architecture()
	b.workspace()
	b.authorization()
	b.milestones()
	if len(b.errs) > 0 {
		return nil, errors.New(strings.Join(b.errs, "; "))
	}
	return b.facts, nil
}

func (b *factBuilder) fail(format string, args ...any) {
	b.errs = append(b.errs, fmt.Sprintf(format, args...))
}

func (b *factBuilder) mark(layer string) {
	if err := b.facts.MarkLayer(layer); err != nil {
		b.fail("%s", err.Error())
	}
}

// add records one tuple located at rel:line.
func (b *factBuilder) add(rel string, line int, relation, stableID string, values ...string) {
	src := checker.Source{Path: checker.PortableSourcePath(rel), Line: line}
	if err := b.facts.Add(relation, stableID, src, values...); err != nil {
		b.fail("%s", err.Error())
	}
}

// exists reports whether a design-relative regular file is present.
func (b *factBuilder) exists(rel string) bool {
	has, err := probeRegularFile(b.design, rel)
	if err != nil {
		b.fail("%s: %s", rel, err.Error())
		return false
	}
	return has
}

// read returns a design-relative file's text with CRLF normalized.
func (b *factBuilder) read(rel string) (string, bool) {
	body, err := readDesignFile(b.design, filepath.Join(b.design, filepath.FromSlash(rel)))
	if err != nil {
		b.fail("%s is unreadable: %s", rel, err.Error())
		return "", false
	}
	return normalizeNewlines(body), true
}

// glob lists design-relative files of a directory matching pattern, sorted.
// A matching entry that is not a regular file is a problem, as it is for the
// gates' own inventories.
func (b *factBuilder) glob(dir, pattern string) []string {
	g := NewGate("projection inventory")
	paths, _ := strictSortedGlob(g, filepath.Join(b.design, dir), pattern, "projection source")
	b.errs = append(b.errs, g.Errs...)
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, dir+"/"+filepath.Base(path))
	}
	return out
}

// lineOf returns the 1-based line of byte offset off in text.
func lineOf(text string, off int) int {
	if off > len(text) {
		off = len(text)
	}
	return strings.Count(text[:off], "\n") + 1
}

// ---------------------------------------------------------------- model

// yamlMap returns the value node of key in a mapping node, or nil.
func yamlMap(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func yamlScalar(node *yaml.Node, key string) string {
	if v := yamlMap(node, key); v != nil && v.Kind == yaml.ScalarNode {
		return strings.TrimSpace(v.Value)
	}
	return ""
}

func yamlItems(node *yaml.Node) []*yaml.Node {
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil
	}
	return node.Content
}

func parseYAMLDocument(rel, text string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("%s: empty YAML document", rel)
	}
	return doc.Content[0], nil
}

// model projects the Modelith model: the model, invariants, relationships and
// actions layers. Values are projected as the model states them (a
// relationship cardinality Modelith accepts, such as n:n, is projected even
// though the v1 projection's four-value vocabulary would refuse it); the
// Modelith linter owns the model's validity, and this reader requires only
// the names it projects.
func (b *factBuilder) model() {
	paths := checker.ModelPaths(b.design)
	if len(paths) != 1 {
		b.fail("expected exactly one *.modelith.yaml in the design, found %d", len(paths))
		return
	}
	rel := filepath.Base(paths[0])
	text, ok := b.read(rel)
	if !ok {
		return
	}
	root, err := parseYAMLDocument(rel, text)
	if err != nil {
		b.fail("%s", err.Error())
		return
	}
	for _, layer := range []string{"model", "invariants", "relationships", "actions"} {
		b.mark(layer)
	}
	enums := map[string][]string{}
	if en := yamlMap(root, "enums"); en != nil && en.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(en.Content); i += 2 {
			for _, v := range yamlItems(yamlMap(en.Content[i+1], "values")) {
				if name := yamlScalar(v, "name"); name != "" {
					enums[en.Content[i].Value] = append(enums[en.Content[i].Value], name)
				}
			}
		}
	}
	need := func(line int, what, value string) bool {
		if value == "" {
			b.fail("%s:%d: %s must be non-empty", rel, line, what)
			return false
		}
		return true
	}
	for _, iv := range yamlItems(yamlMap(root, "invariants")) {
		id := yamlScalar(iv, "id")
		if need(iv.Line, "invariant id", id) {
			b.add(rel, iv.Line, "invariant", "inv:"+id, id)
		}
	}
	entities := yamlMap(root, "entities")
	if entities == nil || entities.Kind != yaml.MappingNode || len(entities.Content) == 0 {
		b.fail("%s: declares no entities", rel)
		return
	}
	for i := 0; i+1 < len(entities.Content); i += 2 {
		name, line, body := entities.Content[i].Value, entities.Content[i].Line, entities.Content[i+1]
		if !need(line, "entity name", name) {
			continue
		}
		b.add(rel, line, "entity", "entity:"+name, name)
		for _, attr := range yamlItems(yamlMap(body, "attributes")) {
			an, at := yamlScalar(attr, "name"), yamlScalar(attr, "type")
			if !need(attr.Line, "entity "+name+" attribute name", an) || !need(attr.Line, "entity "+name+" attribute type", at) {
				continue
			}
			id := name + "." + an
			b.add(rel, attr.Line, "attr", "attr:"+id, id, name, at)
			for _, v := range enums[at] {
				b.add(rel, attr.Line, "enum_member", "enum:"+id+"."+v, id+"."+v, at, v)
			}
		}
		for _, iv := range yamlItems(yamlMap(body, "invariants")) {
			id := yamlScalar(iv, "id")
			if !need(iv.Line, "entity "+name+" invariant id", id) {
				continue
			}
			b.add(rel, iv.Line, "invariant", "inv:"+id, id)
			b.add(rel, iv.Line, "invariant_owner", "inv:"+id, id, name)
		}
		for _, rv := range yamlItems(yamlMap(body, "relationships")) {
			to, card := yamlScalar(rv, "entity"), yamlScalar(rv, "cardinality")
			if !need(rv.Line, "entity "+name+" relationship entity", to) || !need(rv.Line, "entity "+name+" relationship cardinality", card) {
				continue
			}
			role := yamlScalar(rv, "role")
			if role == "" {
				role = yamlScalar(rv, "name")
			}
			id := name + "->" + to + ":" + card
			if role != "" {
				id += ":" + role
			}
			b.add(rel, rv.Line, "relationship", "rel:"+id, id, name, to, card)
		}
		for _, av := range yamlItems(yamlMap(body, "actions")) {
			an := yamlScalar(av, "name")
			if !need(av.Line, "entity "+name+" action name", an) {
				continue
			}
			id := name + "." + an
			b.add(rel, av.Line, "action", "action:"+id, id, name, an, yamlScalar(av, "actor"))
		}
	}
}

// ---------------------------------------------------------------- machines

// jsonLines maps each JSON pointer of a document (object members by key,
// array elements by index) to the 1-based line its key or element starts on.
func jsonLines(data []byte) (map[string]int, error) {
	newlines := []int{}
	for i, c := range data {
		if c == '\n' {
			newlines = append(newlines, i)
		}
	}
	lineAt := func(off int64) int {
		return sort.SearchInts(newlines, int(off)) + 1
	}
	skip := func(off int64) int64 {
		for off < int64(len(data)) {
			switch data[off] {
			case ' ', '\t', '\r', '\n', ',', ':':
				off++
				continue
			}
			break
		}
		return off
	}
	escape := strings.NewReplacer("~", "~0", "/", "~1")
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	lines := map[string]int{}
	var walk func(path string) error
	walk = func(path string) error {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			for dec.More() {
				start := skip(dec.InputOffset())
				keyTok, err := dec.Token()
				if err != nil {
					return err
				}
				key, _ := keyTok.(string)
				p := path + "/" + escape.Replace(key)
				lines[p] = lineAt(start)
				if err := walk(p); err != nil {
					return err
				}
			}
		case '[':
			for i := 0; dec.More(); i++ {
				p := path + "/" + strconv.Itoa(i)
				lines[p] = lineAt(skip(dec.InputOffset()))
				if err := walk(p); err != nil {
					return err
				}
			}
		}
		_, err = dec.Token()
		return err
	}
	if err := walk(""); err != nil {
		return nil, err
	}
	return lines, nil
}

// statePointer is the JSON pointer of a dotted state path.
func statePointer(path string) string {
	var b strings.Builder
	for _, part := range strings.Split(path, ".") {
		b.WriteString("/states/")
		b.WriteString(strings.NewReplacer("~", "~0", "/", "~1").Replace(part))
	}
	return b.String()
}

// resolveTarget renders a transition target as a state id of the machine:
// "#<machine id>.a.b" is absolute, ".a" is a child of the source, and a bare
// name is a sibling of the source. An internal transition has no target and
// resolves to "".
func resolveTarget(machine, machineID, source, target string) string {
	switch {
	case target == "":
		return ""
	case strings.HasPrefix(target, "#"):
		rest := strings.TrimPrefix(target, "#")
		if machineID != "" && strings.HasPrefix(rest, machineID+".") {
			return machine + "." + strings.TrimPrefix(rest, machineID+".")
		}
		return target
	case strings.HasPrefix(target, "."):
		return machine + "." + source + target
	}
	if at := strings.LastIndex(source, "."); at >= 0 {
		return machine + "." + source[:at+1] + target
	}
	return machine + "." + target
}

// oracleStableIDs renders the oracle for a machine and returns the stable id
// of each transition row in order, so a transition's projected id is the id
// its generated oracle row carries.
func oracleStableIDs(root *ir.Value, sourceName string) []string {
	var ids []string
	for _, tbl := range ir.ParseMdTables(oracle.Render(root, sourceName)) {
		si := ir.FindCol(tbl.Header, "stable id")
		if si < 0 || ir.FindCol(tbl.Header, "trigger") < 0 {
			continue
		}
		for _, row := range tbl.Rows {
			ids = append(ids, strings.TrimSpace(cellAt(row, si)))
		}
	}
	return ids
}

func (b *factBuilder) machines() {
	paths := b.glob("machines", "*.machine.json")
	if len(paths) == 0 {
		return
	}
	b.mark("machines")
	for _, rel := range paths {
		text, ok := b.read(rel)
		if !ok {
			continue
		}
		root, err := ir.LoadMachineJSONBytes(rel, []byte(text))
		if err != nil {
			b.fail("%s", err.Error())
			continue
		}
		lines, err := jsonLines([]byte(text))
		if err != nil {
			b.fail("%s: %s", rel, err.Error())
			continue
		}
		at := func(pointer string) int {
			if l := lines[pointer]; l > 0 {
				return l
			}
			return 1
		}
		m := strings.TrimSuffix(filepath.Base(rel), ".machine.json")
		obj := root.AsObject()
		if obj == nil {
			b.fail("%s: machine is not a JSON object", rel)
			continue
		}
		b.add(rel, 1, "machine", "machine:"+m, m)
		if ctx := obj.GetObject("context"); ctx != nil {
			for _, key := range ctx.Keys() {
				b.add(rel, at("/context/"+key), "context_key", "machine:"+m, m, key)
			}
		}
		states := ir.WalkStates(obj.Get2("states"), "")
		sids := oracleStableIDs(root, filepath.Base(rel))
		n := 0
		for _, s := range states {
			so := s.Node.AsObject()
			kind := ""
			if so != nil {
				kind = so.GetString("type")
				if so.Get2("states") != nil && kind == "" {
					kind = "compound"
				}
			}
			if kind == "" {
				kind = "atomic"
			}
			sp := statePointer(s.Path)
			stateID := m + "." + s.Path
			b.add(rel, at(sp), "state", "state:"+stateID, stateID, m, kind)
			for _, inv := range ir.InvokesOf(s.Node) {
				if inv == nil || inv.Kind != ir.KindObject {
					continue
				}
				if src := inv.AsObject().GetString("src"); src != "" {
					b.add(rel, at(sp+"/invoke"), "invoke", "state:"+stateID, stateID, src)
				}
			}
			for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
				if n >= len(sids) {
					b.fail("%s: transition count disagrees with the generated oracle", rel)
					break
				}
				sid := sids[n]
				n++
				trig := tr.Kind
				if tr.Event != "" {
					trig = tr.Kind + ":" + tr.Event
				}
				pointer := sp
				switch tr.Kind {
				case "on":
					pointer += "/on/" + tr.Event
				case "after":
					pointer += "/after/" + tr.Event
				case "always":
					pointer += "/always"
				case "stateDone":
					pointer += "/onDone"
				default:
					pointer += "/invoke"
				}
				trID := m + "." + sid
				dst := resolveTarget(m, obj.GetString("id"), s.Path, tr.Target)
				b.add(rel, at(pointer), "transition", "tr:"+trID, trID, stateID, trig, dst)
				if tr.Guard != "" {
					b.add(rel, at(pointer), "guard_on", "tr:"+trID, trID, tr.Guard)
				}
				for _, action := range tr.Actions {
					b.add(rel, at(pointer), "action_on", "tr:"+trID, trID, action)
				}
			}
		}
		if n != len(sids) {
			b.fail("%s: %d transitions read, the generated oracle has %d", rel, n, len(sids))
		}
	}
}

// ---------------------------------------------------------------- matrices

var (
	unitNameToken = regexp.MustCompile("`([^`]+)`")
	unitNameShape = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)
)

// unitNames returns the unit names a named-unit row declares: every
// backticked name in the name cell (a row may name several units joined by
// " / "), or the cleaned cell when it carries no backticks.
func unitNames(cell string) []string {
	var out []string
	for _, m := range unitNameToken.FindAllStringSubmatch(cell, -1) {
		out = append(out, strings.TrimSpace(m[1]))
	}
	if len(out) == 0 {
		if name := ir.CleanCell(cell); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// matrixRow is one data row of a matrix table with the subjects its
// declarations attach to: the unit ids of a named-unit row, or the matrix id
// itself for any other row (a consumed-event row, say).
type matrixRow struct {
	rel, matrix string
	row         tableRow
	subjects    []string
	unitNames   []string
	stableIDs   []string
	decls       []Declaration
}

func (b *factBuilder) matrices() {
	paths := b.glob("machines", "*.matrix.md")
	if len(paths) == 0 {
		return
	}
	b.mark("matrices")
	for _, rel := range paths {
		text, ok := b.read(rel)
		if !ok {
			continue
		}
		matrix := strings.TrimSuffix(filepath.Base(rel), ".matrix.md")
		machine := ""
		if b.exists("machines/" + matrix + ".machine.json") {
			machine = matrix
		}
		decls, derrs := ParseMatrixDeclarations(rel, []byte(text))
		for _, e := range derrs {
			b.fail("%s", e.String())
		}
		byLine := map[int][]Declaration{}
		for _, d := range decls {
			byLine[d.Line] = append(byLine[d.Line], d)
		}
		for _, r := range walkTableRows(text) {
			mr := matrixRow{rel: rel, matrix: matrix, row: r, decls: byLine[r.line]}
			if ni, ki, _, ok := namedUnitCols(r.header); ok {
				kind := strings.ToLower(ir.CleanCell(cellAt(r.cells, ki)))
				for _, name := range unitNames(cellAt(r.cells, ni)) {
					if !unitNameShape.MatchString(name) {
						b.fail("%s:%d: unit name %q is not an identifier", rel, r.line, name)
						continue
					}
					id := matrix + "." + name
					b.add(rel, r.line, "unit", "unit:"+id, id, machine, name, kind)
					mr.subjects = append(mr.subjects, id)
					mr.unitNames = append(mr.unitNames, name)
					mr.stableIDs = append(mr.stableIDs, "unit:"+id)
				}
			} else {
				mr.subjects = []string{matrix}
				mr.unitNames = []string{""}
				mr.stableIDs = []string{"matrix:" + matrix}
			}
			b.matrixRowFacts(mr)
		}
	}
}

// matrixRowFacts projects the declaration groups on one matrix row into the
// matrices relations. It is the one place a new matrix declaration becomes a
// fact: CLAUSES{}/RETIRED{} (unit_clauses), READS{} (unit_reads), VALUES{}
// (unit_values), derived: (unit_derived), payload {} (unit_payload), and the
// Stage 1 groups parsed by ParseMatrixDeclarations, WRITES{} (unit_writes),
// USES{} (unit_uses) and CARRIES{} (unit_carries). A new group adds a case
// here and a relation to the checker catalog, and nothing else moves.
func (b *factBuilder) matrixRowFacts(mr matrixRow) {
	r := mr.row
	where := fmt.Sprintf("%s:%d", mr.rel, r.line)
	var events []string
	if ei := ir.FindCol(r.header, "event"); ei >= 0 {
		events = eventNamesOf(cellAt(r.cells, ei))
	}
	if len(events) == 0 {
		events = []string{""}
	}
	each := func(fn func(i int, subject, stableID string)) {
		for i, subject := range mr.subjects {
			fn(i, subject, mr.stableIDs[i])
		}
	}
	for _, cell := range r.cells {
		for _, m := range clauseDecl.FindAllStringSubmatch(cell, -1) {
			for _, status := range []struct {
				name, body string
			}{{"active", m[1]}, {"retired", m[2]}} {
				for _, clause := range splitClauses(status.body) {
					each(func(_ int, subject, sid string) {
						b.add(mr.rel, r.line, "unit_clauses", sid, subject, clause, status.name)
					})
				}
			}
		}
		if strings.Count(cell, "CLAUSES{") != len(clauseDecl.FindAllString(cell, -1)) {
			b.fail("%s: malformed CLAUSES declaration", where)
		}
		for _, m := range readsDecl.FindAllStringSubmatch(cell, -1) {
			for _, field := range splitClauses(m[1]) {
				for _, event := range events {
					each(func(_ int, subject, sid string) {
						b.add(mr.rel, r.line, "unit_reads", sid, subject, event, field)
					})
				}
			}
		}
		groups := valuesGroup.FindAllStringSubmatch(cell, -1)
		if len(groups) == 0 && valuesOpening.MatchString(cell) {
			b.fail("%s: malformed VALUES declaration; write VALUES{a, b, c}", where)
		}
		for _, g := range groups {
			values, why := parseValues(g[2])
			if why != "" {
				b.fail("%s: %s", where, why)
				continue
			}
			each(func(i int, subject, sid string) {
				group := g[1]
				if group == "" {
					group = mr.unitNames[i]
				}
				if group == "" {
					b.fail("%s: VALUES declaration on a row that names no unit needs a group name (VALUES name{...})", where)
					return
				}
				for _, v := range values {
					b.add(mr.rel, r.line, "unit_values", sid, subject, group, v)
				}
			})
		}
		for _, at := range derivedWord.FindAllStringIndex(cell, -1) {
			m := derivedFact.FindStringSubmatch(cell[at[0]:])
			if m == nil || strings.TrimSpace(m[2]) == "" {
				b.fail("%s: malformed derived waiver; write derived: fact_name (<reason>)", where)
				continue
			}
			each(func(_ int, subject, sid string) {
				b.add(mr.rel, r.line, "unit_derived", sid, subject, m[1])
			})
		}
		if payloadOpening.MatchString(cell) {
			matches := payloadDeclaration.FindAllStringSubmatch(cell, -1)
			if len(matches) != 1 {
				b.fail("%s: a payload declaration must be exactly one complete payload {field, ...}", where)
			} else if fields, why := parseClosedFieldSet(matches[0][1]); why != "" {
				b.fail("%s: %s", where, why)
			} else {
				for _, field := range fields {
					for _, event := range events {
						each(func(_ int, subject, sid string) {
							b.add(mr.rel, r.line, "unit_payload", sid, subject, event, field)
						})
					}
				}
			}
		}
	}
	for _, d := range mr.decls {
		switch d.Group {
		case GroupWrites, GroupUses:
			relation := "unit_writes"
			if d.Group == GroupUses {
				relation = "unit_uses"
			}
			for _, fact := range d.Members {
				each(func(_ int, subject, sid string) {
					b.add(mr.rel, r.line, relation, sid, subject, fact)
				})
			}
		case GroupCarries:
			for _, p := range d.Pairs {
				each(func(_ int, subject, sid string) {
					b.add(mr.rel, r.line, "unit_carries", sid, subject, p.Kind, p.Target)
				})
			}
		}
	}
}

// ---------------------------------------------------------------- oracles

// oracleSets are the committed oracle files besides machines/*.oracle.md.
var formalOracleSets = []string{"formal/Policy.oracle.md", "formal/Isolation.oracle.md"}

func (b *factBuilder) oracles() {
	var paths []string
	paths = append(paths, b.glob("machines", "*.oracle.md")...)
	for _, rel := range formalOracleSets {
		if b.exists(rel) {
			paths = append(paths, rel)
		}
	}
	if len(paths) == 0 {
		return
	}
	b.mark("oracles")
	for _, rel := range paths {
		text, ok := b.read(rel)
		if !ok {
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(rel), ".oracle.md")
		machineOracle := strings.HasPrefix(rel, "machines/")
		for _, r := range walkTableRows(text) {
			ti, si := ir.FindCol(r.header, "test id"), ir.FindCol(r.header, "stable id")
			if si < 0 {
				continue
			}
			sid := strings.TrimSpace(cellAt(r.cells, si))
			if sid == "" || sid == "-" {
				continue
			}
			transition := ""
			if machineOracle {
				transition = stem + "." + sid
			}
			b.add(rel, r.line, "oracle_row", "orc:"+sid, sid, stem, transition)
			b.oracleRows = append(b.oracleRows, projectedOracleRow{
				rel: rel, machine: stem, testID: strings.TrimSpace(cellAt(r.cells, ti)), stableID: sid, line: r.line,
			})
		}
	}
}

// ---------------------------------------------------------------- architecture

// architecture projects ARCHITECTURE.md: the event-contract tables (events),
// the Architecture Contract (the contract half of c4) and SUPERSEDES
// declarations (supersession).
func (b *factBuilder) architecture() {
	const rel = "ARCHITECTURE.md"
	if !b.exists(rel) {
		return
	}
	text, ok := b.read(rel)
	if !ok {
		return
	}
	b.events(rel, text)
	b.contract(rel, text)
	b.supersession(rel, text)
}

func (b *factBuilder) events(rel, text string) {
	for _, r := range walkTableRows(text) {
		hl := strings.ToLower(strings.Join(r.header, " "))
		if !strings.Contains(hl, "producer") || !strings.Contains(hl, "consumer") || !strings.Contains(hl, "delivery") {
			continue
		}
		b.mark("events")
		ei := colContaining(r.header, "event")
		if ei < 0 {
			continue
		}
		producers := ir.CellNames(cellAt(r.cells, colContaining(r.header, "producer")))
		consumers := ir.CellNames(cellAt(r.cells, colContaining(r.header, "consumer")))
		fields, why := architecturePayloadFields(cellAt(r.cells, colContaining(r.header, "payload")))
		if why != "" {
			fields = nil // a payload cell stating no closed field set is prose, not facts
		}
		for _, ev := range eventNamesOf(cellAt(r.cells, ei)) {
			sid := "event:" + ev
			if len(producers) == 0 {
				b.add(rel, r.line, "event", sid, ev, "")
			}
			for _, p := range producers {
				b.add(rel, r.line, "event", sid, ev, p)
				b.add(rel, r.line, "event_participant", sid, ev, p)
			}
			for _, c := range consumers {
				b.add(rel, r.line, "event_consumer", sid, ev, c)
				b.add(rel, r.line, "event_participant", sid, ev, c)
			}
			for _, f := range fields {
				b.add(rel, r.line, "event_payload_field", sid, ev, f)
			}
		}
	}
}

// contract projects the Architecture Contract fence: boundaries and
// externals (boundary), dependency_rules.allow (allowed_edge) and the
// declared reads list (reads_row). The contract is validated by the G2
// loader first; a contract G2 rejects is not projected.
func (b *factBuilder) contract(rel, text string) {
	fence, ok := ir.ContractFence(text)
	if !ok {
		return
	}
	g := NewGate("projection contract")
	if loadContract(b.design, filepath.Join(b.design, rel), g) == nil || len(g.Errs) > 0 {
		for _, e := range g.Errs {
			b.fail("%s: %s", rel, e)
		}
		return
	}
	b.mark("c4")
	first := lineOf(text, strings.Index(text, fence))
	root, err := parseYAMLDocument(rel, fence)
	if err != nil {
		b.fail("%s", err.Error())
		return
	}
	at := func(n *yaml.Node) int { return first + n.Line - 1 }
	for _, spec := range []struct{ key, role string }{{"boundaries", "boundary"}, {"externals", "external"}} {
		for _, item := range yamlItems(yamlMap(root, spec.key)) {
			id := yamlScalar(item, "id")
			b.add(rel, at(item), "boundary", spec.role+":"+id, id, yamlScalar(item, "element"), spec.role)
		}
	}
	for _, item := range yamlItems(yamlMap(yamlMap(root, "dependency_rules"), "allow")) {
		if m := edgeRuleRe.FindStringSubmatch(item.Value); m != nil {
			b.add(rel, at(item), "allowed_edge", "edge:"+m[1]+"->"+m[2], m[1], m[2])
		}
	}
	for _, item := range yamlItems(yamlMap(root, "reads")) {
		artifact := yamlScalar(item, "artifact")
		b.add(rel, at(item), "reads_row", "reads:"+artifact, artifact, yamlScalar(item, "reader"))
	}
}

// supersession projects SUPERSEDES{type:Old} declarations. The replacing type
// is the row's subject: the first name of a table row's first cell, or the
// contract item id of a fence line. type_owner records the artifact that
// declares the type (ARCHITECTURE.md here; a schema catalog would add its own
// path), so a rule can find two artifacts claiming one type.
func (b *factBuilder) supersession(rel, text string) {
	b.mark("supersession")
	decls, errs := ParseContractDeclarations(rel, []byte(text))
	for _, e := range errs {
		b.fail("%s", e.String())
	}
	rows := map[int]tableRow{}
	for _, r := range walkTableRows(text) {
		rows[r.line] = r
	}
	for _, d := range decls {
		subject := d.Row
		if d.Column != "contract" {
			if names := ir.CellNames(cellAt(rows[d.Line].cells, 0)); len(names) > 0 {
				subject = names[0]
			}
		}
		if !unitNameShape.MatchString(subject) {
			b.fail("%s:%d: SUPERSEDES row subject %q is not an identifier", rel, d.Line, subject)
			continue
		}
		b.add(rel, d.Line, "type_owner", "type:"+subject, subject, checker.PortableSourcePath(rel))
		for _, p := range d.Pairs {
			b.add(rel, d.Line, "supersedes", "type:"+subject, subject, p.Target)
		}
	}
}

// ---------------------------------------------------------------- workspace

// workspace projects workspace.dsl: every declared element with its kind and
// enclosing element, and every drawn relationship.
func (b *factBuilder) workspace() {
	const rel = "workspace.dsl"
	if !b.exists(rel) {
		return
	}
	text, ok := b.read(rel)
	if !ok {
		return
	}
	b.mark("c4")
	type open struct {
		name  string
		depth int
	}
	var stack []open
	depth := 0
	for i, line := range strings.Split(text, "\n") {
		if m := dslDeclRe.FindStringSubmatch(line); m != nil {
			parent := ""
			if len(stack) > 0 {
				parent = stack[len(stack)-1].name
			}
			b.add(rel, i+1, "c4_element", "c4:"+m[1], m[1], m[2], parent)
			if strings.HasSuffix(strings.TrimSpace(line), "{") {
				stack = append(stack, open{name: m[1], depth: depth})
			}
		}
		bare := dslQuotedRe.ReplaceAllString(line, `""`)
		depth += strings.Count(bare, "{") - strings.Count(bare, "}")
		for len(stack) > 0 && depth <= stack[len(stack)-1].depth {
			stack = stack[:len(stack)-1]
		}
	}
	for _, r := range dslRelationships(text) {
		b.add(rel, r.Line, "c4_relationship", "c4:"+r.Src+"->"+r.Dst, r.Src, r.Dst)
	}
}

// ---------------------------------------------------------------- authorization

// authorization projects the marked authorization inventory: one admission
// or no_authorization row per subject. The reader is the table rule of
// authz.go (the authorization subject and admission columns, the backticked
// admission, the (no authorization: reason) waiver); no prose heuristic and
// no consumer-private notation is read. The waiver reason is prose and is
// not projected.
func (b *factBuilder) authorization() {
	g := NewGate("projection authorization")
	documents, markers := authorizationInventoryCorpus(g, b.design)
	b.errs = append(b.errs, g.Errs...)
	if markers == 0 {
		return
	}
	b.mark("authorization")
	for _, doc := range documents {
		for _, r := range walkTableRows(normalizeNewlines([]byte(doc.text))) {
			si, ai := colContaining(r.header, "authorization subject"), colContaining(r.header, "admission")
			if si < 0 || ai < 0 {
				continue
			}
			where := fmt.Sprintf("%s:%d", doc.path, r.line)
			subject := ir.CleanCell(cellAt(r.cells, si))
			if !authorizationSubject.MatchString(subject) {
				b.fail("%s: authorization subject %q is not one Entity.action identifier", where, subject)
				continue
			}
			admission := strings.TrimSpace(cellAt(r.cells, ai))
			if w := noAuthorization.FindStringSubmatch(admission); w != nil {
				b.add(doc.path, r.line, "no_authorization", "action:"+subject, subject)
				continue
			}
			m := authorizationAdmission.FindStringSubmatch(admission)
			if m == nil {
				b.fail("%s: admission must be one backticked identifier or (no authorization: <reason>)", where)
				continue
			}
			b.add(doc.path, r.line, "admission", "action:"+subject, subject, m[1])
		}
	}
}

// ---------------------------------------------------------------- milestones

var (
	oracleIDShape = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9]`)
	oracleSetRef  = regexp.MustCompile(`ORACLESET\{([^}]*)\}`)
)

// citedOracleIDs returns the oracle ids a text cites, in first-seen order:
// every committed oracle id (test or stable) occurring as a whole token,
// normalized to its stable id; every ORACLESET{path} expanded to the stable
// ids of that committed file; and, projected as stated, every token shaped
// like an oracle id (TAG-hex6 or T-TAG-nn) that no committed oracle declares.
// Whether a cited id exists is Gb's question, not the projection's.
func (b *factBuilder) citedOracleIDs(text string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(id string) {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	known := map[string]string{}
	for _, row := range b.oracleRows {
		known[row.stableID] = row.stableID
		if row.testID != "" && row.testID != "-" {
			known[row.testID] = row.stableID
		}
	}
	for _, tok := range oracleIDShape.FindAllString(text, -1) {
		if sid, ok := known[tok]; ok {
			add(sid)
		} else if packetStableIDRe.MatchString(tok) || packetTestIDRe.MatchString(tok) {
			add(tok)
		}
	}
	for _, row := range b.oracleRows {
		if !seen[row.stableID] && idTokenIn(row.stableID, text) {
			add(row.stableID)
		}
	}
	for _, m := range oracleSetRef.FindAllStringSubmatch(text, -1) {
		target := strings.TrimSpace(m[1])
		for _, row := range b.oracleRows {
			if row.rel == target {
				add(row.stableID)
			}
		}
	}
	return out
}

func (b *factBuilder) milestones() {
	hasPlan := b.exists("BUILD.md")
	hasSlices := b.exists(SliceMapFile)
	if !hasPlan && !hasSlices {
		return
	}
	b.mark("milestones")
	if hasPlan {
		b.planMilestones("BUILD.md")
	}
	if hasSlices {
		b.sliceClaims(SliceMapFile)
	}
}

func (b *factBuilder) planMilestones(rel string) {
	text, ok := b.read(rel)
	if !ok {
		return
	}
	ms, ok := planMilestonesOf(text)
	if !ok {
		return
	}
	masked := maskFences(text)
	lines := strings.Split(masked, "\n")
	bodyLine := 0
	var body string
	for i, line := range lines {
		level, title := headingText(line)
		if (level != 2 && level != 3) || !planHeadingRe.MatchString(title) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l, _ := headingText(lines[j]); l > 0 && l <= level {
				end = j
				break
			}
		}
		bodyLine, body = i+2, strings.Join(lines[i+1:end], "\n")
		break
	}
	offsets := milestoneRe.FindAllStringIndex(body, -1)
	if len(offsets) != len(ms) {
		b.fail("%s: milestone markers disagree with the plan parser", rel)
		return
	}
	for i, m := range ms {
		line := bodyLine + strings.Count(body[:offsets[i][0]], "\n")
		id := "M" + m.numRaw
		b.add(rel, line, "milestone", "ms:"+id, id, m.status)
		for _, orc := range b.citedOracleIDs(m.dodText()) {
			b.add(rel, line, "dod_id", "ms:"+id, id, orc)
		}
	}
}

// sliceClaims projects slices.yaml: each slice's oracle citations (a bare
// oracle id, or oracleset:<path> expanded) as slice_claim rows. Gw-packet owns
// the slice map's validation; this reader needs only ids and cites.
func (b *factBuilder) sliceClaims(rel string) {
	text, ok := b.read(rel)
	if !ok {
		return
	}
	root, err := parseYAMLDocument(rel, text)
	if err != nil {
		b.fail("%s", err.Error())
		return
	}
	for _, ms := range yamlItems(yamlMap(root, "milestones")) {
		for _, sl := range yamlItems(yamlMap(ms, "slices")) {
			id := yamlScalar(sl, "id")
			if id == "" {
				b.fail("%s:%d: slice has no id", rel, sl.Line)
				continue
			}
			for _, cite := range yamlItems(yamlMap(sl, "cites")) {
				c := strings.TrimSpace(cite.Value)
				var ids []string
				switch {
				case packetStableIDRe.MatchString(c) || packetTestIDRe.MatchString(c):
					ids = b.citedOracleIDs(c)
				case strings.HasPrefix(c, "oracleset:"):
					ids = b.citedOracleIDs("ORACLESET{" + strings.TrimPrefix(c, "oracleset:") + "}")
				}
				for _, orc := range ids {
					b.add(rel, cite.Line, "slice_claim", "slice:"+id, id, orc)
				}
			}
		}
	}
}

// LoadDesignFacts reads the design's facts inside the held snapshot, so the
// facts come from the same immutable view every other reader uses and a
// design with crash residue is refused before anything is read.
func (s *Snapshot) LoadDesignFacts() (*checker.DesignFacts, error) {
	if err := validateDesignInventory(s.design); err != nil {
		return nil, s.LogicalError(err)
	}
	facts, err := LoadDesignFacts(s.design)
	if err != nil {
		return nil, s.LogicalError(err)
	}
	return facts, nil
}
