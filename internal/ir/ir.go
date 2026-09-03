package ir

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RamXX/machinery/internal/dirscan"
	"github.com/RamXX/machinery/internal/portablepath"
)

const machineInventoryMaxEntries = 100_000

// IdentPattern is machine_lint.IDENT: [A-Za-z_][A-Za-z0-9_]*.
const IdentPattern = `[A-Za-z_][A-Za-z0-9_]*`

var identRe = regexp.MustCompile(IdentPattern)

var tlaModuleIdentRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// TLAModuleName returns the canonical TLA+ module and artifact basename for a
// machine. Machine ids cross two trust boundaries: they are emitted as TLA+
// identifiers and joined to an output directory as filenames. Accepting only
// the portable TLA+ identifier subset makes both uses safe; in particular, a
// path separator or dot segment can never escape the requested output tree.
func TLAModuleName(root *Value) (string, error) {
	if root == nil || root.Kind != KindObject {
		return "", fmt.Errorf("machine is not an object")
	}
	idv := root.AsObject().Get2("id")
	if idv == nil || idv.Kind != KindString || idv.AsString() == "" {
		return "", fmt.Errorf("machine id must be a non-empty string matching [A-Za-z][A-Za-z0-9_]*")
	}
	id := idv.AsString()
	if !tlaModuleIdentRe.MatchString(id) {
		return "", fmt.Errorf("machine id %s is not a safe TLA+ identifier (expected [A-Za-z][A-Za-z0-9_]*)", goRepr(idv))
	}
	module := Title(id)
	if err := portablepath.ValidateBase(module + ".tla"); err != nil {
		return "", fmt.Errorf("machine id %s does not produce a portable artifact name: %w", goRepr(idv), err)
	}
	return module, nil
}

// ValidateTLAModuleInventory requires every machine in dir to be a regular
// source and to produce a basename unique under case-folding. Title-casing
// makes foo and Foo an exact collision; FOO and Foo alias on APFS/NTFS.
func ValidateTLAModuleInventory(dir string) error {
	entries, err := dirscan.Read(dir, machineInventoryMaxEntries)
	if err != nil {
		return err
	}
	owners := map[string]string{}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".machine.json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("machine source %s must be a regular file, not a symlink or special file", entry.Name())
		}
		machine, err := LoadMachineJSON(path)
		if err != nil {
			return err
		}
		module, err := TLAModuleName(machine)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name(), err)
		}
		stem := strings.TrimSuffix(entry.Name(), ".machine.json")
		if !strings.EqualFold(stem, module) {
			return fmt.Errorf("machine source %s must use a case-fold-equivalent canonical filename %s.machine.json for machine id", entry.Name(), module)
		}
		fold := strings.ToLower(module)
		if prior, ok := owners[fold]; ok {
			return fmt.Errorf("machine sources %s and %s produce TLA artifact names that alias on case-insensitive filesystems", prior, entry.Name())
		}
		owners[fold] = entry.Name()
	}
	return nil
}

// StateEntry is a walked state: (path, simpleName, node).
type StateEntry struct {
	Path string
	Name string
	Node *Value
}

// WalkStates yields every state depth-first, mirroring machine_lint.walk_states.
// `states` is the machine's "states" object value (or nil).
func WalkStates(states *Value, prefix string) []StateEntry {
	var out []StateEntry
	if states == nil || states.Kind != KindObject {
		return out
	}
	obj := states.AsObject()
	for _, name := range obj.Keys() {
		node, _ := obj.Get(name)
		path := prefix + name
		out = append(out, StateEntry{Path: path, Name: name, Node: node})
		if node != nil && node.Kind == KindObject {
			if child := node.AsObject().Get2("states"); child != nil {
				out = append(out, WalkStates(child, path+".")...)
			}
		}
	}
	return out
}

// Get2 is a convenience: object key lookup returning *Value (nil if absent or not object).
func (o *Object) Get2(key string) *Value {
	v, _ := o.Get(key)
	return v
}

// Transition is a flattened transition (the dicts from machine_lint._norm +
// kind/event metadata from transitions_of).
type Transition struct {
	Kind     string // on | after | always | stateDone | onDone | onError
	Event    string
	Target   string // "" means absent (caller treats nil as internal)
	HasTgt   bool   // distinguishes target:"" from missing? Python uses it.get("target")
	Guard    string // guard name or "" (empty) when guard is None
	HasGuard bool   // true when a guard key was present and string
	Actions  []string
}

// ActionNames mirrors machine_lint.action_names: a transition/actions/entry/exit
// value (string | {type} | list of those) normalized to a name slice. Problems
// encountered are appended to *problems (matching the Python error text).
func ActionNames(v *Value, problems *[]string, where string) []string {
	var names []string
	if v == nil {
		return names
	}
	items := []*Value{v}
	if v.Kind == KindArray {
		items = v.AsArray()
	}
	for _, a := range items {
		if a == nil {
			continue
		}
		switch a.Kind {
		case KindString:
			names = append(names, a.AsString())
		case KindObject:
			o := a.AsObject()
			if t := o.Get2("type"); t != nil && t.Kind == KindString {
				names = append(names, t.AsString())
				// unknown keys (params, typos) are hard errors, as documented:
				// an action object carries a name only; parameters belong in
				// the implementation, not the design
				if problems != nil {
					for _, k := range o.Keys() {
						if k != "type" {
							*problems = append(*problems, fmt.Sprintf("unsupported key %s in action object %s%s (only 'type' is supported)",
								pyReprStr(k), goRepr(a), whereSuffix(where)))
						}
					}
				}
			} else if problems != nil {
				*problems = append(*problems, fmt.Sprintf("unsupported action value %s%s (use a name string or {\"type\": name})",
					goRepr(a), whereSuffix(where)))
			}
		default:
			if problems != nil {
				*problems = append(*problems, fmt.Sprintf("unsupported action value %s%s (use a name string or {\"type\": name})",
					goRepr(a), whereSuffix(where)))
			}
		}
	}
	return names
}

func whereSuffix(where string) string {
	if where == "" {
		return ""
	}
	return " in " + where
}

// transitionKeys whitelists transition-object members (lint.TransitionKeys
// mirrors this list for its docs); validated here so every consumer that
// passes a problems slice sees the finding.
var transitionKeys = map[string]bool{
	"target": true, "guard": true, "actions": true, "description": true, "_comment": true,
}

// normTransitions mirrors machine_lint._norm: normalize a transition value into
// a list of {target, guard, actions}. Problems (array target, non-string guard,
// unknown keys, unsupported value) are recorded, matching Python text.
type normBranch struct {
	Target   string
	HasTgt   bool
	Guard    string
	HasGuard bool
	Actions  []string
}

func normTransition(t *Value, problems *[]string, where string) []normBranch {
	var items []*Value
	if t == nil {
		return nil
	}
	if t.Kind == KindArray {
		items = t.AsArray()
		if len(items) == 0 && problems != nil {
			*problems = append(*problems, fmt.Sprintf("empty transition branch list%s (the trigger would be silently swallowed)", whereSuffix(where)))
		}
	} else {
		items = []*Value{t}
	}
	var out []normBranch
	for _, it := range items {
		if it == nil {
			continue
		}
		switch it.Kind {
		case KindString:
			out = append(out, normBranch{Target: it.AsString(), HasTgt: true})
		case KindObject:
			o := it.AsObject()
			for _, k := range o.Keys() {
				if !transitionKeys[k] {
					if problems != nil {
						*problems = append(*problems, fmt.Sprintf("unsupported key %s in transition%s (a typo here silently becomes an internal self-transition)",
							goRepr(StringValue(k)), whereSuffix(where)))
					}
				}
			}
			var tgt string
			hasTgt := false
			if tv := o.Get2("target"); tv != nil && tv.Kind != KindNull {
				switch tv.Kind {
				case KindArray:
					arr := tv.AsArray()
					if problems != nil {
						*problems = append(*problems, fmt.Sprintf("array transition target %s%s (parallel targets are unsupported)",
							goRepr(tv), whereSuffix(where)))
					}
					if len(arr) > 0 && arr[0].Kind == KindString {
						tgt = arr[0].AsString()
						hasTgt = true
					}
				case KindString:
					tgt = tv.AsString()
					hasTgt = true
				default:
					if problems != nil {
						*problems = append(*problems, fmt.Sprintf("non-string transition target %s%s", goRepr(tv), whereSuffix(where)))
					}
				}
			}
			var guard string
			hasGuard := false
			if gv := o.Get2("guard"); gv != nil {
				if gv.Kind == KindString && gv.AsString() != "" {
					guard = gv.AsString()
					hasGuard = true
				} else if gv.Kind == KindString {
					if problems != nil {
						*problems = append(*problems, fmt.Sprintf("empty guard string%s (write an unguarded branch instead)", whereSuffix(where)))
					}
				} else if problems != nil {
					*problems = append(*problems, fmt.Sprintf("non-string guard %s%s", goRepr(gv), whereSuffix(where)))
				}
			}
			acts := ActionNames(o.Get2("actions"), problems, where)
			out = append(out, normBranch{Target: tgt, HasTgt: hasTgt, Guard: guard, HasGuard: hasGuard, Actions: acts})
		default:
			if problems != nil {
				*problems = append(*problems, fmt.Sprintf("unsupported transition value %s%s", goRepr(it), whereSuffix(where)))
			}
		}
	}
	return out
}

// TransitionProblems runs the shared transition normalizer over every state
// and returns every admissibility finding in source order. Generators use this
// before emitting anything so malformed branches cannot be silently narrowed
// into a different machine. Lint uses the same TransitionsOf path directly.
func TransitionProblems(root *Value) []string {
	if root == nil || root.Kind != KindObject {
		return nil
	}
	var problems []string
	for _, s := range WalkStates(root.AsObject().Get2("states"), "") {
		TransitionsOf(s.Node, &problems, s.Path)
	}
	return problems
}

// TransitionsOf mirrors machine_lint.transitions_of: all transitions on a state
// node, flattened. kind ∈ {on, after, always, stateDone, onDone, onError}.
func TransitionsOf(node *Value, problems *[]string, state string) []Transition {
	if node == nil || node.Kind != KindObject {
		return nil
	}
	o := node.AsObject()
	var res []Transition

	if on := o.Get2("on"); on != nil {
		if on.Kind != KindObject {
			if problems != nil {
				*problems = append(*problems, fmt.Sprintf("transition container 'on' must be an object%s", whereSuffix(state)))
			}
		} else {
			for _, ev := range on.AsObject().Keys() {
				for _, b := range normTransition(on.AsObject().Get2(ev), problems, state+" on:"+ev) {
					res = append(res, Transition{Kind: "on", Event: ev,
						Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
				}
			}
		}
	}
	if after := o.Get2("after"); after != nil {
		if after.Kind != KindObject {
			if problems != nil {
				*problems = append(*problems, fmt.Sprintf("transition container 'after' must be an object%s", whereSuffix(state)))
			}
		} else {
			for _, delay := range after.AsObject().Keys() {
				for _, b := range normTransition(after.AsObject().Get2(delay), problems, state+" after:"+delay) {
					res = append(res, Transition{Kind: "after", Event: delay,
						Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
				}
			}
		}
	}
	if always := o.Get2("always"); always != nil {
		for _, b := range normTransition(always, problems, state+" always") {
			res = append(res, Transition{Kind: "always", Event: "",
				Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
		}
	}
	if od := o.Get2("onDone"); od != nil {
		for _, b := range normTransition(od, problems, state+" onDone") {
			res = append(res, Transition{Kind: "stateDone", Event: "",
				Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
		}
	}
	if inv := o.Get2("invoke"); inv != nil {
		if inv.Kind != KindObject && inv.Kind != KindArray {
			if problems != nil {
				*problems = append(*problems, fmt.Sprintf("transition source 'invoke' must be an object or array%s", whereSuffix(state)))
			}
			return res
		}
		for i, iv := range invokesRaw(inv) {
			if iv == nil || iv.Kind != KindObject {
				if problems != nil {
					*problems = append(*problems, fmt.Sprintf("invoke entry %d must be an object%s", i+1, whereSuffix(state)))
				}
				continue
			}
			ivObj := iv.AsObject()
			for _, key := range []string{"onDone", "onError"} {
				if ivObj.Get2(key) != nil {
					src := ""
					if s := ivObj.Get2("src"); s != nil && s.Kind == KindString {
						src = s.AsString()
					}
					for _, b := range normTransition(ivObj.Get2(key), problems, state+" invoke."+key) {
						res = append(res, Transition{Kind: key, Event: src,
							Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
					}
				}
			}
		}
	}
	return res
}

// InvokesOf mirrors machine_lint.invokes_of: invoke as a list (or wrapped).
func InvokesOf(node *Value) []*Value {
	if node == nil || node.Kind != KindObject {
		return nil
	}
	inv := node.AsObject().Get2("invoke")
	if inv == nil {
		return nil
	}
	if inv.Kind == KindArray {
		return inv.AsArray()
	}
	return []*Value{inv}
}

// invokesRaw is the same as InvokesOf but tolerates non-object elements.
func invokesRaw(inv *Value) []*Value {
	if inv.Kind == KindArray {
		return inv.AsArray()
	}
	return []*Value{inv}
}

// ActionsOf mirrors machine_lint.actions_of: every action name on entry/exit
// plus transition actions (a set).
func ActionsOf(node *Value, problems *[]string, state string) map[string]struct{} {
	acc := map[string]struct{}{}
	if node == nil || node.Kind != KindObject {
		return acc
	}
	o := node.AsObject()
	for _, k := range []string{"entry", "exit"} {
		for _, n := range ActionNames(o.Get2(k), problems, state+" "+k) {
			acc[n] = struct{}{}
		}
	}
	for _, tr := range TransitionsOf(node, nil, state) {
		for _, a := range tr.Actions {
			acc[a] = struct{}{}
		}
	}
	return acc
}

// --- markdown table parsing (machine_lint.parse_md_tables / find_col / _clean_cell) ---

// MdTable is a parsed markdown table: header cells + data rows.
type MdTable struct {
	Header []string
	Rows   [][]string
	// RowLines holds the source line of each data row, index-aligned with
	// Rows (leading and trailing whitespace trimmed, the cell text otherwise
	// untouched). A tool that REWRITES a table copies a row from here, so the
	// copy is byte-identical to its source rather than re-rendered from the
	// parsed cells, which would normalize the author's spacing.
	RowLines []string
}

// stripAnnotations removes every parenthesized annotation from a cell,
// BALANCING the parentheses: an annotation that itself contains parentheses
// ("core (custody registration (Oban-delivered))") is one annotation, not a
// prefix of one plus a stray ")" left behind to make the participant name
// unresolvable. An annotation that is never closed swallows the rest of the
// cell, which is the same reading: everything from the first top-level "(" is
// annotation. A ")" with no "(" open is ordinary text and survives.
func stripAnnotations(cell string) string {
	var b strings.Builder
	depth := 0
	for _, r := range cell {
		switch {
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case depth == 0:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// annotationGroups returns the CONTENT of each top-level parenthesized
// annotation, balanced the same way. An unclosed group yields the rest of the
// cell, so a caller sees the text it would otherwise silently ignore.
func annotationGroups(cell string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range cell {
		switch r {
		case '(':
			if depth == 0 {
				start = i + 1
			}
			depth++
		case ')':
			if depth == 1 {
				out = append(out, cell[start:i])
			}
			if depth > 0 {
				depth--
			}
		}
	}
	if depth > 0 {
		out = append(out, cell[start:])
	}
	return out
}

// splitRowCells splits a markdown table row into cells, honoring the GFM
// `\|` escape (a literal pipe inside a cell). Mirrors the previous
// TrimPrefix("|") / TrimSuffix("|") / Split("|") semantics otherwise: one
// leading delimiter pipe and one trailing delimiter pipe are consumed, and a
// trailing `\|` belongs to the last cell instead of being eaten as the
// closing delimiter.
func splitRowCells(s string) []string {
	s = strings.TrimPrefix(s, "|")
	var cells []string
	var b strings.Builder
	endsWithDelim := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' && i+1 < len(s) && s[i+1] == '|' {
			b.WriteByte('|')
			i++
			endsWithDelim = false
			continue
		}
		if c == '|' {
			cells = append(cells, b.String())
			b.Reset()
			endsWithDelim = i == len(s)-1
			continue
		}
		b.WriteByte(c)
	}
	cells = append(cells, b.String())
	if n := len(cells); n > 1 && endsWithDelim {
		cells = cells[:n-1] // the closing delimiter pipe, not an empty cell
	}
	return cells
}

// ParseMdTables mirrors machine_lint.parse_md_tables.
func ParseMdTables(text string) []MdTable {
	var blocks [][]string
	var cur []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "|") {
			cur = append(cur, strings.TrimSpace(line))
		} else if len(cur) > 0 {
			blocks = append(blocks, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		blocks = append(blocks, cur)
	}
	var tables []MdTable
	for _, b := range blocks {
		var rows [][]string
		for _, r := range b {
			cells := splitRowCells(strings.TrimSpace(r))
			for i := range cells {
				cells[i] = strings.TrimSpace(cells[i])
			}
			rows = append(rows, cells)
		}
		if len(rows) < 2 {
			continue
		}
		sep := strings.Join(rows[1], "")
		sepSet := map[rune]bool{}
		for _, c := range sep {
			sepSet[c] = true
		}
		isSep := true
		for c := range sepSet {
			if c != '-' && c != ':' && c != ' ' {
				isSep = false
				break
			}
		}
		var data [][]string
		var dataLines []string
		if isSep {
			data, dataLines = rows[2:], b[2:]
		} else {
			data, dataLines = rows[1:], b[1:]
		}
		tables = append(tables, MdTable{Header: rows[0], Rows: data, RowLines: dataLines})
	}
	return tables
}

// --- Architecture Contract locator ---
//
// The single source of truth for finding the contract fence in an
// ARCHITECTURE.md: a yaml fence under a heading containing "architecture
// contract", falling back to the first yaml fence starting with
// contract_version. Both internal/gates (G2) and internal/pack read the
// contract; a second, laxer locator once made G5 reject boundaries on a
// design G2 passed (prose mentioning contract_version above the heading).
var (
	contractHeadingRe  = regexp.MustCompile(`(?ims)^#+[^\n]*architecture contract[^\n]*\n.*?` + "```yaml\n(.*?)\n```")
	contractFallbackRe = regexp.MustCompile("(?s)```yaml\n(contract_version:.*?)\n```")
)

// ContractFence extracts the Architecture Contract yaml fence body from a
// markdown document. Returns ok=false when no contract fence is present.
func ContractFence(text string) (string, bool) {
	m := contractHeadingRe.FindStringSubmatch(text)
	if m == nil {
		m = contractFallbackRe.FindStringSubmatch(text)
	}
	if m == nil {
		return "", false
	}
	return m[1], true
}

// FindCol locates the first header cell labeled with any of names. Matching is
// per-cell label-prefix (cell == name, or cell starts with "name "), after
// stripping backticks and parentheticals, never substring-in-cell: "resource"
// is not a "source" column and "retarget" is not a "target" column. A prose
// table whose cells merely contain the words must not be mistaken for a
// transition table (same hardening as the lint's raw-header detector).
func FindCol(header []string, names ...string) int {
	for i, h := range header {
		cell := strings.ToLower(strings.TrimSpace(CleanCell(h)))
		for _, n := range names {
			if cell == n || strings.HasPrefix(cell, n+" ") {
				return i
			}
		}
	}
	return -1
}

// CleanCell mirrors machine_lint._clean_cell: strip backticks + parentheticals.
func CleanCell(cell string) string {
	cell = strings.ReplaceAll(cell, "`", "")
	cell = stripAnnotations(cell)
	return strings.TrimSpace(cell)
}

// CellNames resolves the COMPONENT NAMES a table cell denotes, under the one
// cell grammar the pack extractor and the Ge where-filter share (S15 of the
// dogfood systemic findings: Ge once matched by whole-token containment over the
// full cell, so prose inside an annotation, "the batch_rows site MOVED to
// ops", made an ingest row match producer=ops and every complete-claim embed
// under that filter failed). Candidates are: the first identifier of each
// "+"-separated part of the CleanCell head (backticks and parentheticals
// stripped), plus the content of any parenthetical that holds EXACTLY ONE
// identifier (the owning-domain marker form: "`AuditEvent` (trust) (no
// machine: ...)" denotes trust). A multi-word parenthetical is annotation
// prose and denotes nothing.
func CellNames(cell string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	for _, part := range strings.Split(CleanCell(cell), "+") {
		if ids := identRe.FindAllString(part, 1); len(ids) > 0 {
			add(ids[0])
		}
	}
	for _, inner := range annotationGroups(cell) {
		ids := identRe.FindAllString(inner, -1)
		if len(ids) == 1 && strings.TrimSpace(inner) == ids[0] {
			add(ids[0])
		}
	}
	return out
}

// FindAllIdent is a helper wrapping the IDENT regex (returns all matches).
func FindAllIdent(s string) []string {
	return identRe.FindAllString(s, -1)
}

// Simple mirrors refine/compose _simple: "#a.b" -> "b", "" -> "".
func Simple(t string) string {
	if t == "" {
		return ""
	}
	t = strings.TrimLeft(t, "#")
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

// Title mirrors _title/_t: capitalize first rune.
func Title(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// goRepr produces a Python-like repr for use in error messages, to match the
// exact strings machine_lint emits (so the differential harness passes).
//
//	Python repr('foo')  == "'foo'"
//	Python repr(42)     == "42"
//	Python repr([1,2])  == "[1, 2]"
//	Python repr({'a':1})== "{'a': 1}"
//	Python repr(None)   == "None"
//	Python repr(True)   == "True"
func goRepr(v *Value) string {
	if v == nil {
		return "None"
	}
	switch v.Kind {
	case KindString:
		return pyReprStr(v.AsString())
	case KindNumber:
		return string(v.AsNumber())
	case KindBool:
		if b, _ := v.AsBool(); b {
			return "True"
		}
		return "False"
	case KindNull:
		return "None"
	case KindArray:
		parts := make([]string, 0, len(v.AsArray()))
		for _, e := range v.AsArray() {
			parts = append(parts, goRepr(e))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case KindObject:
		o := v.AsObject()
		parts := make([]string, 0, o.Len())
		for _, k := range o.Keys() {
			parts = append(parts, pyReprStr(k)+": "+goRepr(o.Get2(k)))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return "None"
}

// pyReprStr mimics Python repr() for a string: single quotes by default,
// switching to double quotes if the string contains a single quote but no
// double quote, and escaping backslashes/newlines/tabs minimally.
func pyReprStr(s string) string {
	hasSingle := strings.Contains(s, "'")
	hasDouble := strings.Contains(s, "\"")
	quote := "'"
	if hasSingle && !hasDouble {
		quote = "\""
	}
	var b strings.Builder
	b.WriteString(quote)
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r == rune(quote[0]) {
				b.WriteString("\\" + quote)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteString(quote)
	return b.String()
}
