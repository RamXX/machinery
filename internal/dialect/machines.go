// Package dialect reads design sources that a project keeps in its own
// formats and translates them, in memory, into the formats of
// docs/formats.md (sections 5.2.1 and 5.4.1). It never writes a source.
package dialect

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/RamXX/machinery/v2/internal/ir"
	"github.com/RamXX/machinery/v2/internal/project"
)

// Machine is one translated machine file.
type Machine struct {
	Source   string // project-relative path of the source file
	ID       string // the machine id
	Original string // the original id: the root id, or the file name
	Bytes    []byte // the translated machine
	// Listed says, for an event import.machines.annotate added to a root
	// list ("<list key> <event>"), which entry and pattern did.
	Listed      map[string]string
	SourceBytes []byte
}

// FileName is the translated file name of a machine id.
func FileName(id string) string { return id + ".machine.json" }

// Machines translates every file import.machines matches, in path order.
// Problems name the source file; a file with a problem that leaves no
// machine (not JSON, a repeated id) is not returned.
func Machines(p *project.Project) (ms []Machine, problems []string) {
	im := p.Cfg.ImportedMachines()
	if im == nil {
		return nil, nil
	}
	files, err := p.Glob(im.Files)
	if err != nil {
		return nil, []string{fmt.Sprintf("import.machines.files: %v", err)}
	}
	gen := strings.TrimSuffix(p.Cfg.Design.Gen, "/") + "/"
	byID := map[string]string{}
	used := map[string]bool{}      // the bind keys that match a machine
	annotated := map[string]bool{} // the annotate keys that match a machine
	for _, f := range files {
		if strings.HasPrefix(f, gen) || f == im.RegisterFile() {
			continue // generated, or the register: never a machine
		}
		for _, k := range im.BindKeys() {
			if project.Match([]string{k}, f) {
				used[k] = true
			}
		}
		for _, k := range im.AnnotateKeys() {
			if project.Match([]string{k}, f) {
				annotated[k] = true
			}
		}
		data, err := p.Read(f)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: cannot read: %v", f, err))
			continue
		}
		root, err := ir.LoadMachineJSONBytes(f, data)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", f, err))
			continue
		}
		if root.AsObject() == nil {
			problems = append(problems, f+": machine must be a JSON object")
			continue
		}
		t := &translator{src: f, rename: im.Rename, mode: im.States, im: im, used: used, annotated: annotated}
		out := t.machine(root.AsObject())
		problems = append(problems, t.problems...)
		if first, dup := byID[t.id]; dup {
			problems = append(problems, fmt.Sprintf("%s: translates to machine id %s, as %s does; give one of them another id", f, t.id, first))
			continue
		}
		byID[t.id] = f
		b, err := ir.Marshal(ir.ObjectValue(out))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", f, err))
			continue
		}
		ms = append(ms, Machine{Source: f, ID: t.id, Original: t.origID, Bytes: b, SourceBytes: data, Listed: t.listed})
	}
	for _, k := range im.BindKeys() {
		if !used[k] {
			problems = append(problems, fmt.Sprintf("import.machines.bind %s: matches no machine file and no machine id", k))
		}
	}
	for _, k := range im.AnnotateKeys() {
		if !annotated[k] {
			problems = append(problems, fmt.Sprintf("import.machines.annotate %s: matches no machine file and no machine id; correct the key or remove the entry", k))
		}
	}
	return ms, problems
}

// translator turns one source machine into a native one.
type translator struct {
	src    string
	rename project.Rename
	mode   string // the states setting: how state names are written
	im     *project.ImportMachines
	used   map[string]bool // bind keys that matched a machine, shared by all files
	// annotate keys that matched a machine, shared by all files
	annotated map[string]bool
	// sourceOf maps the dotted path of a translated state to its path in
	// source names
	sourceOf map[string]string
	// aliasFrom says which entry of import.machines.annotate gave an event its alias
	aliasFrom map[string]string
	// listed is Machine.Listed
	listed   map[string]string
	origID   string
	id       string
	problems []string
}

func (t *translator) fail(format string, a ...any) {
	t.problems = append(t.problems, t.src+": "+fmt.Sprintf(format, a...))
}

// MachineID replaces every character outside [A-Za-z0-9_] with "_".
func MachineID(orig string) string {
	var b strings.Builder
	for _, r := range orig {
		if r < 128 && (r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

// OracleTag is the tag of an imported machine without _oracle_tag: the first
// four letters or digits of the id, upper-cased, and the first four hex
// digits of the SHA-256 of the original id, upper-cased.
func OracleTag(id, orig string) string {
	var b strings.Builder
	for _, r := range id {
		if b.Len() == 4 {
			break
		}
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	sum := sha256.Sum256([]byte(orig))
	return b.String() + strings.ToUpper(hex.EncodeToString(sum[:])[:4])
}

// machine translates the root of a source machine. With
// import.machines.terminal, the reason its root key holds becomes _terminal
// on every absorbing state of the result; with none, it stays as _comment,
// as a rename to _comment would keep it (section 5.2.1).
func (t *translator) machine(root *ir.Object) *ir.Object {
	key := t.im.TerminalKey()
	if key == "" || !root.Has(key) {
		return t.translate(root, t.rename.Root)
	}
	reason := ir.StringValue(textOf(root.Get2(key)))
	without := ir.NewObject()
	for _, k := range root.Keys() {
		if k != key {
			without.Set(k, root.Get2(k))
		}
	}
	before := len(t.problems)
	out := t.translate(without, t.rename.Root)
	absorbing := ir.Absorbing(ir.ObjectValue(out))
	if len(absorbing) == 0 {
		t.problems = t.problems[:before]
		names := map[string]string{key: "_comment"}
		for k, v := range t.rename.Root {
			names[k] = v
		}
		return t.translate(root, names)
	}
	nodes := ir.NewResolver(ir.ObjectValue(out)).NodeOf
	for _, p := range absorbing {
		nodes[p].AsObject().Set("_terminal", reason)
	}
	return out
}

// translate renames the root keys with names, then translates the rest.
func (t *translator) translate(root *ir.Object, names map[string]string) *ir.Object {
	out := t.keys(root, names, "root", nil)
	t.origID = out.GetString("id")
	if t.origID == "" {
		base := path.Base(t.src)
		t.origID = strings.TrimSuffix(base, path.Ext(base))
	}
	t.id = MachineID(t.origID)
	if !out.Has("id") {
		withID := ir.NewObject()
		withID.Set("id", ir.StringValue(t.id))
		for _, k := range out.Keys() {
			withID.Set(k, out.Get2(k))
		}
		out = withID
	}
	out.Set("id", ir.StringValue(t.id))
	t.capitalizeValue(out, "initial")
	if e := out.Get2("_entries"); e != nil && e.Kind == ir.KindArray {
		var names []*ir.Value
		for _, v := range e.AsArray() {
			if v != nil && v.Kind == ir.KindString {
				v = ir.StringValue(t.stateName(v.AsString()))
			}
			names = append(names, v)
		}
		out.Set("_entries", ir.ArrayValue(names))
	}
	t.sourceOf = map[string]string{}
	if sv := out.Get2("states"); sv != nil {
		out.Set("states", t.states(sv, "", ""))
	}
	t.bind(out)
	t.annotate(out)
	if !out.Has("_oracle_tag") {
		out.Set("_oracle_tag", ir.StringValue(OracleTag(t.id, t.origID)))
	}
	return out
}

// bind adds the binding import.machines.bind gives a machine that has none
// of its own (section 5.2.1). It never invents one.
func (t *translator) bind(out *ir.Object) {
	var keys []string
	for _, k := range t.im.BindKeys() {
		if k == t.id || project.Match([]string{k}, t.src) {
			keys = append(keys, k)
			t.used[k] = true
		}
	}
	switch {
	case len(keys) == 0:
		return
	case out.Has("_lifecycle_of") || out.Has("_role") || out.Has("_serves"):
		t.fail("bound in the source and by import.machines.bind %s; keep one", keys[0])
		return
	case len(keys) > 1:
		t.fail("import.machines.bind %s and %s both match; keep one", keys[0], keys[1])
		return
	}
	b := t.im.Bind[keys[0]]
	if strings.TrimSpace(b.LifecycleOf) != "" {
		if b.Aspect != "" && out.Has("_aspect") {
			t.fail("_aspect is set in the source and by import.machines.bind %s; keep one", keys[0])
			return
		}
		out.Set("_lifecycle_of", ir.StringValue(b.LifecycleOf))
		if b.Aspect != "" {
			out.Set("_aspect", ir.StringValue(b.Aspect))
		}
		return
	}
	ids := make([]*ir.Value, len(b.Serves))
	for i, id := range b.Serves {
		ids[i] = ir.StringValue(id)
	}
	out.Set("_role", ir.StringValue("operational"))
	out.Set("_serves", ir.ArrayValue(ids))
}

type landed struct {
	key     string
	renamed bool
	value   *ir.Value
}

// keys renames the keys of one node. Values that land on _comment are
// joined; a renamed key that lands on _refusal is handed to hoist when hoist
// is set; any other two keys that land on one key are a problem.
func (t *translator) keys(o *ir.Object, names map[string]string, node string, hoist func(key string, v *ir.Value)) *ir.Object {
	out := ir.NewObject()
	from := map[string]string{}
	var comments []landed
	for _, k := range o.Keys() {
		v := o.Get2(k)
		nk, renamed := names[k]
		if !renamed {
			nk = k
		}
		switch {
		case nk == "_comment":
			if len(comments) == 0 {
				out.Set("_comment", v) // keeps the place of the first
			}
			comments = append(comments, landed{k, renamed, v})
			continue
		case renamed && nk == "_refusal" && hoist != nil:
			hoist(k, v)
			continue
		}
		if first, dup := from[nk]; dup {
			t.fail("%s: keys %s and %s both become %s", node, first, k, nk)
			continue
		}
		from[nk] = k
		out.Set(nk, v)
	}
	if len(comments) == 1 && !comments[0].renamed {
		return out
	}
	if len(comments) > 0 {
		var parts []string
		for _, c := range comments {
			text := textOf(c.value)
			if c.renamed {
				text = c.key + ": " + text
			}
			parts = append(parts, text)
		}
		out.Set("_comment", ir.StringValue(strings.Join(parts, " | ")))
	}
	return out
}

// textOf is a string as written and any other value as compact JSON.
func textOf(v *ir.Value) string {
	if v != nil && v.Kind == ir.KindString {
		return v.AsString()
	}
	return ir.Compact(v)
}

func capitalize(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if n == 0 || r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[n:]
}

// pascal reads a name as words separated by "_" or "-" and upper-cases the
// first letter of each, dropping the separators: on_hold becomes OnHold. A name of separators only is kept as written.
func pascal(s string) string {
	var b strings.Builder
	for _, w := range strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' }) {
		b.WriteString(capitalize(w))
	}
	if b.Len() == 0 {
		return s
	}
	return b.String()
}

// StateName writes a state name as the states setting of import.machines
// says (section 5.2.1).
func StateName(mode, s string) string {
	switch mode {
	case project.StatesCapitalize:
		return capitalize(s)
	case project.StatesPascal:
		return pascal(s)
	}
	return s
}

func (t *translator) stateName(s string) string { return StateName(t.mode, s) }

func (t *translator) capitalizeValue(o *ir.Object, key string) {
	if v := o.Get2(key); v != nil && v.Kind == ir.KindString {
		o.Set(key, ir.StringValue(t.stateName(v.AsString())))
	}
}

// target rewrites a transition target: the head of a #<original id> target
// becomes the machine id, and state names are written as the states setting
// says.
func (t *translator) target(s string) string {
	rest, ok := strings.CutPrefix(s, "#")
	if !ok {
		return t.stateName(s)
	}
	head, tail := rest, ""
	switch {
	case rest == t.origID || strings.HasPrefix(rest, t.origID+"."):
		head, tail = t.id, strings.TrimPrefix(rest, t.origID)
	case strings.Contains(rest, "."):
		i := strings.Index(rest, ".")
		head, tail = rest[:i], rest[i:]
	}
	if tail != "" {
		segs := strings.Split(tail[1:], ".")
		for i := range segs {
			segs[i] = t.stateName(segs[i])
		}
		tail = "." + strings.Join(segs, ".")
	}
	return "#" + head + tail
}

// states translates a states object: names, then each state.
func (t *translator) states(sv *ir.Value, parent, translated string) *ir.Value {
	o := sv.AsObject()
	if o == nil {
		return sv // malformed: the lint reports it
	}
	out := ir.NewObject()
	from := map[string]string{}
	for _, name := range o.Keys() {
		nn := t.stateName(name)
		if first, dup := from[nn]; dup {
			t.fail("states %s and %s both become %s with states: %s", first, name, nn, t.mode)
			continue
		}
		from[nn] = name
		p, tp := name, nn
		if parent != "" {
			p, tp = parent+"."+name, translated+"."+nn
		}
		t.sourceOf[tp] = p
		out.Set(nn, t.state(o.Get2(name), p, tp))
	}
	return ir.ObjectValue(out)
}

type disposition struct {
	handler string
	value   *ir.Value
}

// state translates one state node; p is its dotted path in source names.
func (t *translator) state(v *ir.Value, p, translated string) *ir.Value {
	o := v.AsObject()
	if o == nil {
		return v
	}
	node := "state " + p
	out := t.keys(o, t.rename.State, node, nil)
	var hoisted []disposition
	trig := func(container, trigger, handler string) {
		c := out.Get2(container)
		if c == nil || c.Kind != ir.KindObject {
			return
		}
		co := c.AsObject()
		nc := ir.NewObject()
		for _, k := range co.Keys() {
			nc.Set(k, t.branches(co.Get2(k), node+" "+trigger+k, handler+k, &hoisted))
		}
		out.Set(container, ir.ObjectValue(nc))
	}
	trig("on", "on:", "")
	trig("after", "after:", "after:")
	if a := out.Get2("always"); a != nil {
		out.Set("always", t.branches(a, node+" always", "always", &hoisted))
	}
	if d := out.Get2("onDone"); d != nil {
		out.Set("onDone", t.branches(d, node+" stateDone", "onDone", &hoisted))
	}
	if iv := out.Get2("invoke"); iv != nil {
		out.Set("invoke", t.invokes(iv, node, &hoisted))
	}
	t.capitalizeValue(out, "initial")
	if sv := out.Get2("states"); sv != nil {
		out.Set("states", t.states(sv, p, translated))
	}
	t.refusals(out, node, hoisted)
	return ir.ObjectValue(out)
}

// invokes translates the onDone and onError branches of a state's invoke.
func (t *translator) invokes(iv *ir.Value, node string, hoisted *[]disposition) *ir.Value {
	one := func(v *ir.Value) *ir.Value {
		o := v.AsObject()
		if o == nil {
			return v
		}
		out := ir.NewObject()
		src := o.GetString("src")
		for _, k := range o.Keys() {
			val := o.Get2(k)
			if k == "onDone" || k == "onError" {
				val = t.branches(val, node+" "+k+":"+src, src+"."+k, hoisted)
			}
			out.Set(k, val)
		}
		return ir.ObjectValue(out)
	}
	if iv.Kind == ir.KindArray {
		var items []*ir.Value
		for _, v := range iv.AsArray() {
			items = append(items, one(v))
		}
		return ir.ArrayValue(items)
	}
	return one(iv)
}

// branches translates a transition value: a target, a branch or a list.
func (t *translator) branches(v *ir.Value, node, handler string, hoisted *[]disposition) *ir.Value {
	if v == nil {
		return v
	}
	if v.Kind == ir.KindArray {
		var items []*ir.Value
		for i, b := range v.AsArray() {
			items = append(items, t.branch(b, fmt.Sprintf("%s branch %d", node, i+1), handler, hoisted))
		}
		return ir.ArrayValue(items)
	}
	return t.branch(v, node, handler, hoisted)
}

func (t *translator) branch(v *ir.Value, node, handler string, hoisted *[]disposition) *ir.Value {
	if v == nil {
		return v
	}
	if v.Kind == ir.KindString {
		return ir.StringValue(t.target(v.AsString()))
	}
	o := v.AsObject()
	if o == nil {
		return v
	}
	out := t.keys(o, t.rename.Transition, node, func(_ string, val *ir.Value) {
		*hoisted = append(*hoisted, disposition{handler, val})
	})
	if tg := out.Get2("target"); tg != nil && tg.Kind == ir.KindString {
		out.Set("target", ir.StringValue(t.target(tg.AsString())))
	}
	return ir.ObjectValue(out)
}

// refusals adds the hoisted dispositions to the state's _refusal, keyed by
// handler; dispositions of one handler are joined.
func (t *translator) refusals(out *ir.Object, node string, hoisted []disposition) {
	if len(hoisted) == 0 {
		return
	}
	rv := out.Get2("_refusal")
	if rv != nil && rv.Kind != ir.KindObject {
		t.fail("%s: _refusal is not an object, so the dispositions of its transitions cannot be added to it", node)
		return
	}
	ro := ir.NewObject()
	if rv != nil {
		for _, k := range rv.AsObject().Keys() {
			ro.Set(k, rv.AsObject().Get2(k))
		}
	}
	for _, h := range hoisted {
		have := ro.Get2(h.handler)
		switch {
		case have == nil:
			ro.Set(h.handler, h.value)
		case have.Kind == ir.KindString && h.value != nil && h.value.Kind == ir.KindString:
			ro.Set(h.handler, ir.StringValue(have.AsString()+" | "+h.value.AsString()))
		default:
			t.fail("%s: two dispositions for %s cannot be joined: one is not a string", node, h.handler)
		}
	}
	out.Set("_refusal", ir.ObjectValue(ro))
}
