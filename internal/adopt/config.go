package adopt

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/RamXX/machinery/internal/project"
)

// Default file names adopt writes (section 17.2).
const (
	DefaultE2E       = "e2e/**"
	DesignDir        = "design/" // what adopt writes lives here, or in the gen directory
	TestsBaseline    = "design/tests-baseline.yaml"
	ArchBaseline     = "design/arch-baseline.yaml"
	defaultScopeGlob = "**"
)

// editor changes machinery.yaml through its yaml.Node tree. A new file is the
// encoded tree. An existing file is written as it was, with the additions
// spliced in, and only in two shapes (section 17.2): a key that is absent
// gets a block at the end of the mapping that holds it, and a key that holds
// a block list gets items after its last one, each in the indentation and
// the line endings of the file. Every other shape is refused before anything
// is written, with the lines to add by hand.
type editor struct {
	doc     *yaml.Node
	root    *yaml.Node
	changes []string
	src     []byte // the existing file, nil for a new one
	// stand: an existing value that is not the list or mapping adopt adds
	// to -> the stand-in that collects what adopt would add; it is refused
	stand map[*yaml.Node]*yaml.Node
}

func newEditor(existing []byte) (*editor, error) {
	e := &editor{doc: &yaml.Node{}, stand: map[*yaml.Node]*yaml.Node{}}
	if existing == nil {
		e.root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		e.doc = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{e.root}}
		e.set(e.root, "version", "2", "version")
		return e, nil
	}
	e.src = existing
	if err := yaml.Unmarshal(existing, e.doc); err != nil {
		return nil, project.Errf("%s: %v", project.ConfigName, err)
	}
	if e.doc.Kind != yaml.DocumentNode || len(e.doc.Content) != 1 || e.doc.Content[0].Kind != yaml.MappingNode {
		return nil, project.Errf("%s: the top level must be a mapping", project.ConfigName)
	}
	e.root = e.doc.Content[0]
	return e, nil
}

// bytes renders the document, or returns nil when nothing changed. A new file
// is the encoded tree; an existing one is its own text with the additions
// spliced in, which must say what the tree says.
func (e *editor) bytes() ([]byte, error) {
	if len(e.changes) == 0 {
		return nil, nil
	}
	tree, err := encode(e.doc, 2)
	if err != nil || e.src == nil {
		return tree, err
	}
	s := newSplicer(e)
	s.walk(e.root, "", 0, "", 0)
	s.standIns()
	if shape := docShape(e.src); shape != "" {
		for _, in := range s.ins {
			s.refuse(in.refs, shape)
		}
		s.ins = nil
	}
	var kept []insertion
	for _, in := range s.ins {
		if s.keeps(in) {
			kept = append(kept, in)
			continue
		}
		shape := "an addition here would change what the file has"
		if in.blockScalar {
			shape = fmt.Sprintf("a block scalar ends %s, and an addition after it would change its value", in.ends)
		}
		s.refuse(in.refs, shape)
	}
	out := apply(e.src, kept, s.eol)
	if len(s.refused) == 0 && (docShape(out) != "" || !sameYAML(out, tree)) {
		for _, in := range kept {
			s.refuse(in.refs, "an addition here would change what the file has")
		}
	}
	if len(s.refused) > 0 {
		return nil, s.refusal()
	}
	return out, nil
}

func encode(n *yaml.Node, indent int) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indent)
	if err := enc.Encode(n); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// sameYAML reports whether two documents hold the same values.
func sameYAML(a, b []byte) bool {
	var x, y any
	if yaml.Unmarshal(a, &x) != nil || yaml.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}

// covers reports whether every value of a is in b as it was: the keys of a
// mapping, the first items of a list, and every scalar equal.
func covers(a, b any) bool {
	switch x := a.(type) {
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range x {
			if w, ok := y[k]; !ok || !covers(v, w) {
				return false
			}
		}
		return true
	case map[any]any:
		y, ok := b.(map[any]any)
		if !ok {
			return false
		}
		for k, v := range x {
			if w, ok := y[k]; !ok || !covers(v, w) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(y) < len(x) {
			return false
		}
		for i := range x {
			if !covers(x[i], y[i]) {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(a, b)
}

// docShape names what makes a file more than one YAML document, or "".
func docShape(src []byte) string {
	dec := yaml.NewDecoder(bytes.NewReader(src))
	docs := 0
	for {
		var n yaml.Node
		if dec.Decode(&n) != nil {
			break
		}
		docs++
	}
	content := false
	for i, l := range strings.SplitAfter(string(src), "\n") {
		t := strings.TrimRight(l, "\r\n")
		if i == 0 {
			t = strings.TrimPrefix(t, "\ufeff")
		}
		marker := func(m string) bool {
			return t == m || strings.HasPrefix(t, m+" ") || strings.HasPrefix(t, m+"\t")
		}
		switch trimmed := strings.TrimSpace(t); {
		case marker("..."):
			return "the file has a document end marker (...)"
		case marker("---") && content:
			docs = max(docs, 2)
		case trimmed != "" && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(t, "%") && !marker("---"):
			content = true
		}
	}
	if docs > 1 {
		return "the file holds more than one YAML document"
	}
	return ""
}

// insertion is text spliced into the existing file at off; at one offset
// the deeper container goes first.
type insertion struct {
	off, depth, seq int
	text            string
	refs            []refusal // what adopt prints when it refuses the insertion
	// blockScalar: the text follows a block scalar, the last value of ends
	blockScalar bool
	ends        string
}

// refusal is an addition adopt does not make, and the lines to make it by
// hand (section 17.2).
type refusal struct {
	key, shape, place string
	lines             []string
}

// apply writes src with the insertions; text that follows a line without a
// line break begins with one.
func apply(src []byte, ins []insertion, eol string) []byte {
	sorted := append([]insertion{}, ins...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.off != b.off {
			return a.off < b.off
		}
		if a.depth != b.depth {
			return a.depth > b.depth
		}
		return a.seq < b.seq
	})
	var out bytes.Buffer
	pos := 0
	for _, in := range sorted {
		out.Write(src[pos:in.off])
		if b := out.Bytes(); len(b) > 0 && b[len(b)-1] != '\n' {
			out.WriteString(eol)
		}
		out.WriteString(in.text)
		pos = in.off
	}
	out.Write(src[pos:])
	return out.Bytes()
}

type splicer struct {
	e       *editor
	src     []byte
	lines   []string // each with its line ending
	eol     string   // the line ending of the file
	step    int      // its indentation step
	ins     []insertion
	refused []refusal
}

func newSplicer(e *editor) *splicer {
	s := &splicer{e: e, src: e.src, lines: strings.SplitAfter(string(e.src), "\n"), eol: "\n", step: 2}
	if i := bytes.IndexByte(e.src, '\n'); i > 0 && e.src[i-1] == '\r' {
		s.eol = "\r\n"
	}
	// the step of the first block nested under a key of the top level
	for i := 0; i+1 < len(e.root.Content); i += 2 {
		k, v := e.root.Content[i], e.root.Content[i+1]
		if k.Line == 0 || v.Line == 0 || v.Style&yaml.FlowStyle != 0 || len(v.Content) == 0 {
			continue
		}
		var col int
		switch v.Kind {
		case yaml.MappingNode:
			col = v.Content[0].Column - 1
		case yaml.SequenceNode:
			col = s.dashes(v)
		default:
			continue
		}
		if d := col - (k.Column - 1); d >= 2 && d <= 9 {
			s.step = d
			break
		}
	}
	return s
}

func (s *splicer) refuse(refs []refusal, shape string) {
	for _, r := range refs {
		r.shape = shape
		s.refused = append(s.refused, r)
	}
}

// refusal is the error that lists every addition adopt does not make.
func (s *splicer) refusal() error {
	sort.SliceStable(s.refused, func(i, j int) bool { return s.refused[i].key < s.refused[j].key })
	var b strings.Builder
	fmt.Fprintf(&b, "%s: adopt adds to a key only where it is absent or holds a block list, and writes nothing: add these lines by hand, then run adopt again", project.ConfigName)
	for _, r := range s.refused {
		fmt.Fprintf(&b, "\n%s: %s; %s:", r.key, r.shape, r.place)
		for _, l := range r.lines {
			b.WriteString("\n" + l)
		}
	}
	return project.Errf("%s", b.String())
}

// render returns the lines of a node in block style, at col.
func (s *splicer) render(n *yaml.Node, col int) []string {
	text, err := encode(n, s.step)
	if err != nil {
		return nil
	}
	var res []string
	for _, l := range strings.Split(strings.TrimRight(string(text), "\n"), "\n") {
		if l != "" {
			l = strings.Repeat(" ", col) + l
		}
		res = append(res, l)
	}
	return res
}

func (s *splicer) text(lines []string) string {
	return strings.Join(lines, s.eol) + s.eol
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// subject names the container at path in a shape.
func subject(path string) string {
	if path == "" {
		return "the top level"
	}
	return path
}

// walk finds what adopt added under the existing container n: path is its
// key path ("" for the top level), col the column of the key that holds it,
// and shared why an addition into it would change another value too.
func (s *splicer) walk(n *yaml.Node, path string, col int, shared string, depth int) {
	if n.Anchor != "" && shared == "" {
		shared = fmt.Sprintf("%s has the anchor &%s", subject(path), n.Anchor)
	}
	var added []*yaml.Node
	switch n.Kind {
	case yaml.MappingNode:
		merge := false
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			switch {
			case k.Line == 0:
				added = append(added, k, v)
			case k.Tag == "!!merge":
				merge = true
			case v.Kind != yaml.AliasNode:
				s.walk(v, join(path, k.Value), k.Column-1, shared, depth+1)
			}
		}
		if len(added) > 0 {
			s.addKeys(n, path, col, shared, merge, added, depth)
		}
	case yaml.SequenceNode:
		for _, c := range n.Content {
			if c.Line == 0 {
				added = append(added, c)
			}
		}
		if len(added) > 0 {
			s.addItems(n, path, col, shared, added, depth)
		}
	}
}

// addKeys splices the keys adopt added to an existing mapping: as a block
// after its last line, at the column of its keys (at the end of the file for
// the top level), or a refusal for every other shape.
func (s *splicer) addKeys(n *yaml.Node, path string, col int, shared string, merge bool, added []*yaml.Node, depth int) {
	flow := n.Style&yaml.FlowStyle != 0
	kcol := col + s.step
	switch {
	case path == "" && (flow || len(n.Content) == len(added)):
		kcol = 0
	case !flow && len(n.Content) > len(added):
		kcol = n.Content[0].Column - 1
	}
	place := "under " + path
	if path == "" {
		place = "at the top level"
	}
	var refs []refusal
	var lines []string
	for i := 0; i+1 < len(added); i += 2 {
		pair := s.render(&yaml.Node{Kind: yaml.MappingNode, Content: added[i : i+2]}, kcol)
		refs = append(refs, refusal{key: join(path, added[i].Value), place: place, lines: pair})
		lines = append(lines, pair...)
	}
	switch {
	case flow:
		s.refuse(refs, subject(path)+" is a flow mapping")
		return
	case shared != "":
		s.refuse(refs, shared)
		return
	case merge:
		s.refuse(refs, subject(path)+" has a merge key (<<)")
		return
	}
	off := len(s.src)
	if path != "" {
		off = s.after(s.lastLine(n), kcol)
	}
	s.insert(n, off, depth, lines, refs, path)
}

// addItems splices the items adopt added to an existing list: after its last
// item, at the column of its dashes, or a refusal for every other shape.
func (s *splicer) addItems(n *yaml.Node, path string, col int, shared string, added []*yaml.Node, depth int) {
	flow := n.Style&yaml.FlowStyle != 0
	dcol := col + s.step
	if !flow && len(n.Content) > len(added) {
		dcol = s.dashes(n)
	}
	at := dcol
	if at < 0 {
		at = col + s.step
	}
	lines := s.render(&yaml.Node{Kind: yaml.SequenceNode, Content: added}, at)
	refs := []refusal{{key: path, place: "as a block list under " + path, lines: lines}}
	switch {
	case flow:
		s.refuse(refs, path+" is a flow list")
		return
	case shared != "":
		s.refuse(refs, shared)
		return
	case dcol < 0:
		s.refuse(refs, "an addition here would change what the file has")
		return
	}
	s.insert(n, s.after(s.lastLine(n), dcol), depth, lines, refs, path)
}

func (s *splicer) insert(n *yaml.Node, off, depth int, lines []string, refs []refusal, path string) {
	in := insertion{off: off, depth: depth, seq: len(s.ins), text: s.text(lines), refs: refs, ends: path}
	if path == "" {
		in.ends = "the file"
	}
	if last := lastNode(n); last.Kind == yaml.ScalarNode && last.Style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		in.blockScalar = true
	}
	s.ins = append(s.ins, in)
}

// dashes returns the column of the dashes of an existing block list, read
// from the line of its first item (the node itself starts at its anchor or
// tag when it has one), or -1 when that line does not begin with a dash.
func (s *splicer) dashes(n *yaml.Node) int {
	if len(n.Content) == 0 || n.Content[0].Line < 1 || n.Content[0].Line > len(s.lines) {
		return -1
	}
	l := s.lines[n.Content[0].Line-1]
	if n.Content[0].Line == 1 {
		l = strings.TrimPrefix(l, "\ufeff")
	}
	ind := len(l) - len(strings.TrimLeft(l, " "))
	if ind >= len(l) || l[ind] != '-' {
		return -1
	}
	return ind
}

// standIns refuses what adopt would add to an existing value that is not the
// list or mapping it adds to: a scalar, an alias, a list for a mapping or a
// mapping for a list.
func (s *splicer) standIns() {
	for orig, st := range s.e.stand {
		if len(st.Content) == 0 {
			continue
		}
		path, col := s.e.pathOf(orig)
		var shape string
		switch {
		case orig.Kind == yaml.AliasNode:
			shape = fmt.Sprintf("%s is the alias *%s", path, orig.Value)
		case orig.Kind == yaml.ScalarNode && orig.Tag == "!!null" && orig.Value == "":
			shape = path + " has no value"
		case orig.Kind == yaml.ScalarNode && orig.Tag == "!!null":
			shape = fmt.Sprintf("%s holds the scalar %s", path, orig.Value)
		case orig.Kind == yaml.ScalarNode:
			shape = fmt.Sprintf("%s holds the scalar %q", path, orig.Value)
		case orig.Kind == yaml.SequenceNode:
			shape = path + " is a list"
		default:
			shape = path + " is a mapping"
		}
		if st.Kind == yaml.SequenceNode {
			s.refuse([]refusal{{key: path, place: "as a block list under " + path, lines: s.render(st, col+s.step)}}, shape)
			continue
		}
		for i := 0; i+1 < len(st.Content); i += 2 {
			lines := s.render(&yaml.Node{Kind: yaml.MappingNode, Content: st.Content[i : i+2]}, col+s.step)
			s.refuse([]refusal{{key: join(path, st.Content[i].Value), place: "under " + path, lines: lines}}, shape)
		}
	}
}

// pathOf returns the key path of an existing value and the column of its key.
func (e *editor) pathOf(target *yaml.Node) (string, int) {
	var find func(m *yaml.Node, path string) (string, int, bool)
	find = func(m *yaml.Node, path string) (string, int, bool) {
		if m.Kind != yaml.MappingNode {
			return "", 0, false
		}
		for i := 0; i+1 < len(m.Content); i += 2 {
			k, v := m.Content[i], m.Content[i+1]
			p := join(path, k.Value)
			if v == target {
				return p, k.Column - 1, true
			}
			if res, col, ok := find(v, p); ok {
				return res, col, true
			}
		}
		return "", 0, false
	}
	p, col, _ := find(e.root, "")
	return p, col
}

// keeps reports whether the file with the one insertion still holds every
// value it had.
func (s *splicer) keeps(in insertion) bool {
	var before, after any
	if yaml.Unmarshal(s.src, &before) != nil || yaml.Unmarshal(apply(s.src, []insertion{in}, s.eol), &after) != nil {
		return false
	}
	return covers(before, after)
}

// lastNode is the last existing node in a node's text: its last descendant.
func lastNode(n *yaml.Node) *yaml.Node {
	for n.Style&yaml.FlowStyle == 0 {
		var next *yaml.Node
		for i := len(n.Content) - 1; i >= 0 && next == nil; i-- {
			if n.Content[i].Line > 0 {
				next = n.Content[i]
			}
		}
		if next == nil {
			break
		}
		n = next
	}
	return n
}

// offset is the byte offset of a 1-based line and column (in characters, a
// byte order mark not counted).
func (s *splicer) offset(line, col int) (int, error) {
	if line < 1 || line > len(s.lines) {
		return 0, fmt.Errorf("no line %d", line)
	}
	off := 0
	for _, l := range s.lines[:line-1] {
		off += len(l)
	}
	l := s.lines[line-1]
	if line == 1 && strings.HasPrefix(l, "\ufeff") {
		off += len("\ufeff")
		l = l[len("\ufeff"):]
	}
	for i := 1; i < col && l != ""; i++ {
		_, w := utf8.DecodeRuneInString(l)
		off += w
		l = l[w:]
	}
	return off, nil
}

// after returns the offset right after the block that starts on line (1-based)
// and holds every following line indented deeper than col, blank lines inside
// it included and trailing ones not.
func (s *splicer) after(line, col int) int {
	last := line - 1
	for i := line; i < len(s.lines); i++ {
		l := strings.TrimRight(s.lines[i], "\r\n")
		if strings.TrimSpace(l) == "" {
			continue
		}
		if len(l)-len(strings.TrimLeft(l, " ")) <= col {
			break
		}
		last = i
	}
	off := 0
	for _, l := range s.lines[:last+1] {
		off += len(l)
	}
	return off
}

// lastLine is the last line a node of the existing file takes: the closing
// bracket of a flow collection, or else the last line of its last descendant,
// so that a list written without indentation under its key is inside.
func (s *splicer) lastLine(n *yaml.Node) int {
	if n.Kind == yaml.SequenceNode || n.Kind == yaml.MappingNode {
		if n.Style&yaml.FlowStyle != 0 {
			if start, err := s.offset(n.Line, n.Column); err == nil {
				if end, _, ok := flowEnd(s.src, start); ok {
					return bytes.Count(s.src[:end], []byte("\n")) + 1
				}
			}
		}
	}
	line := n.Line
	for _, c := range n.Content {
		if c.Line > 0 {
			line = max(line, s.lastLine(c))
		}
	}
	return line
}

// flowEnd scans a flow collection that starts at start: it returns the offset
// of its closing bracket and the offset right after its last content, comments
// and white space left out.
func flowEnd(src []byte, start int) (end, last int, ok bool) {
	depth, last := 0, start+1
	for i := start; i < len(src); i++ {
		switch c := src[i]; {
		case c == '"' || c == '\'':
			j := i + 1
			for ; j < len(src); j++ {
				if c == '"' && src[j] == '\\' {
					j++
					continue
				}
				if src[j] == c {
					if c == '\'' && j+1 < len(src) && src[j+1] == '\'' {
						j++
						continue
					}
					break
				}
			}
			if j >= len(src) {
				return 0, 0, false
			}
			i, last = j, j+1
		case c == '#' && i > 0 && (src[i-1] == ' ' || src[i-1] == '\t' || src[i-1] == '\n'):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '[' || c == '{':
			depth++
			if i > start {
				last = i + 1
			}
		case c == ']' || c == '}':
			if depth--; depth == 0 {
				return i, last, true
			}
			last = i + 1
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		default:
			last = i + 1
		}
	}
	return 0, 0, false
}

func get(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func scalar(v string, quoted bool) *yaml.Node {
	n := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v}
	if quoted {
		n.Style = yaml.DoubleQuotedStyle
	}
	return n
}

func (e *editor) note(format string, a ...any) {
	e.changes = append(e.changes, fmt.Sprintf(format, a...))
}

// container returns n when it is of the kind adopt adds to, or else the
// stand-in that collects what adopt would add to it, which is refused
// (section 17.2).
func (e *editor) container(n *yaml.Node, kind yaml.Kind) *yaml.Node {
	if n.Kind == kind {
		return n
	}
	st := e.stand[n]
	if st == nil {
		st = &yaml.Node{Kind: kind}
		e.stand[n] = st
	}
	return st
}

// set adds key: value when the key is absent.
func (e *editor) set(m *yaml.Node, key, value, what string) bool {
	m = e.container(m, yaml.MappingNode)
	if get(m, key) != nil {
		return false
	}
	val := scalar(value, false)
	if value == "2" {
		val.Tag = "!!int"
	}
	m.Content = append(m.Content, scalar(key, false), val)
	e.note("%s", what)
	return true
}

// mapping returns the mapping under key, creating it when absent.
func (e *editor) mapping(m *yaml.Node, key string) *yaml.Node {
	m = e.container(m, yaml.MappingNode)
	if n := get(m, key); n != nil {
		return e.container(n, yaml.MappingNode)
	}
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, scalar(key, false), n)
	return n
}

func seqValues(n *yaml.Node) []string {
	var res []string
	if n != nil && n.Kind == yaml.SequenceNode {
		for _, c := range n.Content {
			res = append(res, c.Value)
		}
	}
	return res
}

// addSeq appends the values that are not in the list yet, creating the key
// when it is absent. Globs are quoted, and a new list is a block list, so
// that a later adopt adds to it (section 17.2).
func (e *editor) addSeq(m *yaml.Node, key string, values []string, glob bool, what string) {
	m = e.container(m, yaml.MappingNode)
	seq := get(m, key)
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		m.Content = append(m.Content, scalar(key, false), seq)
	}
	seq = e.container(seq, yaml.SequenceNode)
	have := map[string]bool{}
	for _, v := range seqValues(seq) {
		have[v] = true
	}
	var added []string
	for _, v := range values {
		if !have[v] {
			have[v] = true
			seq.Content = append(seq.Content, scalar(v, glob))
			added = append(added, v)
		}
	}
	if len(added) > 0 {
		e.note("%s +%s", what, strings.Join(added, ", "))
	}
}

// covered drops the globs an existing glob already matches.
func covered(existing, globs []string) []string {
	var res []string
	for _, g := range globs {
		if !project.Match(existing, g) {
			res = append(res, g)
		}
	}
	return res
}

func (e *editor) addContract(path string) {
	seq := get(e.root, "contracts")
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		e.root.Content = append(e.root.Content, scalar("contracts", false), seq)
	}
	seq = e.container(seq, yaml.SequenceNode)
	item := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
		scalar("kind", false), scalar("openapi", false), scalar("path", false), scalar(path, false)}}
	seq.Content = append(seq.Content, item)
	e.note("contracts +%s", path)
}

// addWaivers appends one waiver per entry, each with the reason, to the
// waivers list (section 17.2), and notes how many. It never removes or
// rewrites a waiver.
func (e *editor) addWaivers(ws []project.Waiver, reason, note string) {
	seq := get(e.root, "waivers")
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		e.root.Content = append(e.root.Content, scalar("waivers", false), seq)
	}
	seq = e.container(seq, yaml.SequenceNode)
	for _, w := range ws {
		seq.Content = append(seq.Content, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle, Content: []*yaml.Node{
			scalar("rule", false), scalar(w.Rule, false),
			scalar("id", false), scalar(w.ID, true),
			scalar("reason", false), scalar(reason, true)}})
	}
	e.note(note, len(ws))
}

func (e *editor) contractPaths() map[string]bool {
	res := map[string]bool{}
	if seq := get(e.root, "contracts"); seq != nil && seq.Kind == yaml.SequenceNode {
		for _, c := range seq.Content {
			if p := get(c, "path"); p != nil {
				res[p.Value] = true
			}
		}
	}
	return res
}
