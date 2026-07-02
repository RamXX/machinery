package ir

import (
	"fmt"
	"regexp"
	"strings"
)

// IdentPattern is machine_lint.IDENT: [A-Za-z_][A-Za-z0-9_]*.
const IdentPattern = `[A-Za-z_][A-Za-z0-9_]*`

var identRe = regexp.MustCompile(IdentPattern)

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

// normTransitions mirrors machine_lint._norm: normalize a transition value into
// a list of {target, guard, actions}. Problems (array target, non-string guard,
// unsupported value) are recorded, matching Python text.
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
				}
			}
			var guard string
			hasGuard := false
			if gv := o.Get2("guard"); gv != nil {
				if gv.Kind == KindString {
					guard = gv.AsString()
					hasGuard = true
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

// TransitionsOf mirrors machine_lint.transitions_of: all transitions on a state
// node, flattened. kind ∈ {on, after, always, stateDone, onDone, onError}.
func TransitionsOf(node *Value, problems *[]string, state string) []Transition {
	if node == nil || node.Kind != KindObject {
		return nil
	}
	o := node.AsObject()
	var res []Transition

	if on := o.Get2("on"); on != nil {
		for _, ev := range on.AsObject().Keys() {
			for _, b := range normTransition(on.AsObject().Get2(ev), problems, state+" on:"+ev) {
				res = append(res, Transition{Kind: "on", Event: ev,
					Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
			}
		}
	}
	if after := o.Get2("after"); after != nil {
		for _, delay := range after.AsObject().Keys() {
			for _, b := range normTransition(after.AsObject().Get2(delay), problems, state+" after:"+delay) {
				res = append(res, Transition{Kind: "after", Event: delay,
					Target: b.Target, HasTgt: b.HasTgt, Guard: b.Guard, HasGuard: b.HasGuard, Actions: b.Actions})
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
		for _, iv := range invokesRaw(inv) {
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
}

var parenRe = regexp.MustCompile(`\([^)]*\)`)

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
			s := strings.TrimSpace(r)
			s = strings.TrimSuffix(strings.TrimPrefix(s, "|"), "|")
			cells := strings.Split(s, "|")
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
		if isSep {
			data = rows[2:]
		} else {
			data = rows[1:]
		}
		tables = append(tables, MdTable{Header: rows[0], Rows: data})
	}
	return tables
}

// FindCol mirrors machine_lint.find_col: first header cell whose lowercased
// text contains any of names.
func FindCol(header []string, names ...string) int {
	for i, h := range header {
		hl := strings.ToLower(h)
		for _, n := range names {
			if strings.Contains(hl, n) {
				return i
			}
		}
	}
	return -1
}

// CleanCell mirrors machine_lint._clean_cell: strip backticks + parentheticals.
func CleanCell(cell string) string {
	cell = strings.ReplaceAll(cell, "`", "")
	cell = parenRe.ReplaceAllString(cell, "")
	return strings.TrimSpace(cell)
}

// FindAllIdent is a helper wrapping the IDENT regex (returns all matches).
func FindAllIdent(s string) []string {
	return identRe.FindAllString(s, -1)
}

// FindAllIdentSubexpr returns the captured groups of IDENT matches.
func FindAllIdentSubexpr(s string) []string {
	matches := identRe.FindAllStringSubmatch(s, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m[1])
	}
	return out
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
