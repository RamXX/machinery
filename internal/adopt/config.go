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

// editor changes machinery.yaml through its yaml.Node tree. Every change adds
// at the end of a mapping or a list, so an existing file is written as it was
// with the additions spliced in: its comments, blank lines, spacing, order and
// styles stay byte for byte (section 17.2).
type editor struct {
	doc     *yaml.Node
	root    *yaml.Node
	changes []string
	src     []byte // the existing file, nil for a new one
	// replaced: a key of the existing file whose empty value adopt replaced
	// with a list, and that value as it was
	replaced map[*yaml.Node]*yaml.Node
}

func newEditor(existing []byte) (*editor, error) {
	e := &editor{doc: &yaml.Node{}}
	if existing == nil {
		e.root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		e.doc = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{e.root}}
		e.set(e.root, "version", "2", "version")
		return e, nil
	}
	e.src, e.replaced = existing, map[*yaml.Node]*yaml.Node{}
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
// spliced in, and must say what the tree says.
func (e *editor) bytes() ([]byte, error) {
	if len(e.changes) == 0 {
		return nil, nil
	}
	tree, err := encode(e.doc)
	if err != nil || e.src == nil {
		return tree, err
	}
	out, err := e.splice()
	if err == nil && !sameYAML(out, tree) {
		err = fmt.Errorf("the additions do not fit the layout of the file")
	}
	if err != nil {
		return nil, project.Errf("%s: adopt cannot add %s without changing what the file has (%v): add it by hand", project.ConfigName, strings.Join(e.changes, "; "), err)
	}
	return out, nil
}

func encode(n *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
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

// insertion is text spliced into the existing file at off, after deleting
// del bytes there; at one offset the deeper container goes first.
type insertion struct {
	off, del, depth, seq int
	text                 string
}

// splice writes the existing file with the nodes adopt added: those without a
// position, which are always the last children of their container.
func (e *editor) splice() ([]byte, error) {
	src := e.src
	if len(src) > 0 && src[len(src)-1] != '\n' {
		src = append(append([]byte{}, src...), '\n')
	}
	s := &splicer{src: src, lines: strings.SplitAfter(string(src), "\n"), replaced: e.replaced}
	if err := s.walk(e.root, 0, true); err != nil {
		return nil, err
	}
	sort.SliceStable(s.ins, func(i, j int) bool {
		a, b := s.ins[i], s.ins[j]
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
	for _, in := range s.ins {
		out.Write(src[pos:in.off])
		out.WriteString(in.text)
		pos = in.off + in.del
	}
	out.Write(src[pos:])
	return out.Bytes(), nil
}

type splicer struct {
	src      []byte
	lines    []string // each with its newline
	replaced map[*yaml.Node]*yaml.Node
	ins      []insertion
}

func (s *splicer) add(off, del, depth int, text string) {
	s.ins = append(s.ins, insertion{off: off, del: del, depth: depth, seq: len(s.ins), text: text})
}

// offset is the byte offset of a 1-based line and column (in characters).
func (s *splicer) offset(line, col int) (int, error) {
	if line < 1 || line > len(s.lines) {
		return 0, fmt.Errorf("no line %d", line)
	}
	off := 0
	for _, l := range s.lines[:line-1] {
		off += len(l)
	}
	l := s.lines[line-1]
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

// walk finds the added children of every container of the existing file.
func (s *splicer) walk(n *yaml.Node, depth int, root bool) error {
	switch n.Kind {
	case yaml.MappingNode:
		first := len(n.Content)
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			switch {
			case key.Line == 0:
				if first == len(n.Content) {
					first = i
				}
			case first < len(n.Content):
				return fmt.Errorf("an addition before the key %s", key.Value)
			case val.Line == 0:
				if err := s.replace(key, val, depth); err != nil {
					return err
				}
			default:
				if err := s.walk(val, depth+1, false); err != nil {
					return err
				}
			}
		}
		return s.added(n, n.Content[first:], depth, root)
	case yaml.SequenceNode:
		first := len(n.Content)
		for i, c := range n.Content {
			switch {
			case c.Line == 0:
				if first == len(n.Content) {
					first = i
				}
			case first < len(n.Content):
				return fmt.Errorf("an addition before line %d", c.Line)
			default:
				if err := s.walk(c, depth+1, false); err != nil {
					return err
				}
			}
		}
		return s.added(n, n.Content[first:], depth, root)
	}
	return nil
}

// added splices the added children of a container: before the closing
// bracket of a flow collection, after the last line of a block one, at the
// end of the file for the top-level mapping.
func (s *splicer) added(n *yaml.Node, kids []*yaml.Node, depth int, root bool) error {
	if len(kids) == 0 {
		return nil
	}
	orig := len(n.Content) - len(kids)
	if n.Style&yaml.FlowStyle != 0 {
		start, err := s.offset(n.Line, n.Column)
		if err != nil {
			return err
		}
		_, last, ok := flowEnd(s.src, start)
		if !ok {
			return fmt.Errorf("no end to the collection on line %d", n.Line)
		}
		text, err := flowText(n.Kind, kids)
		if err != nil {
			return err
		}
		switch {
		case orig == 0:
		case s.src[last-1] == ',':
			text = " " + text
		default:
			text = ", " + text
		}
		s.add(last, 0, depth, text)
		return nil
	}
	col := n.Column - 1 // of the keys of a mapping, of the dashes of a list
	temp := &yaml.Node{Kind: n.Kind, Tag: n.Tag, Content: kids}
	text, err := encode(temp)
	if err != nil {
		return err
	}
	off := len(s.src)
	if !root {
		off = s.after(s.lastLine(n.Content[orig-1]), col)
	}
	s.add(off, 0, depth, indent(string(text), col))
	return nil
}

// replace splices the list adopt put in place of an empty value: the value
// as written goes, and the list follows the key's line.
func (s *splicer) replace(key, val *yaml.Node, depth int) error {
	old := s.replaced[key]
	if old == nil {
		return fmt.Errorf("a value replaced under %s", key.Value)
	}
	if old.Kind == yaml.SequenceNode || old.Value != "" { // [] or ~ or null: the token goes
		start, err := s.offset(old.Line, old.Column)
		if err != nil {
			return err
		}
		end := start + len(old.Value)
		if old.Kind == yaml.SequenceNode {
			close, _, ok := flowEnd(s.src, start)
			if !ok {
				return fmt.Errorf("no end to the list on line %d", old.Line)
			}
			end = close + 1
		}
		s.add(start, end-start, depth, "")
	}
	text, err := encode(&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{key, val}})
	if err != nil {
		return err
	}
	_, body, _ := strings.Cut(string(text), "\n") // the lines under the key
	s.add(s.after(key.Line, key.Column-1), 0, depth+1, indent(body, key.Column-1))
	return nil
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

// indent prefixes every line that is not empty with col spaces.
func indent(text string, col int) string {
	pad := strings.Repeat(" ", col)
	lines := strings.SplitAfter(text, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) != "" {
			lines[i] = pad + l
		}
	}
	return strings.Join(lines, "")
}

// flowText renders children the way they go inside a flow collection.
func flowText(kind yaml.Kind, kids []*yaml.Node) (string, error) {
	text, err := encode(&yaml.Node{Kind: kind, Style: yaml.FlowStyle, Content: kids})
	if err != nil {
		return "", err
	}
	t := strings.TrimSpace(string(text))
	if len(t) < 2 {
		return "", fmt.Errorf("cannot render %q in a flow collection", t)
	}
	return t[1 : len(t)-1], nil
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

// set adds key: value when the key is absent.
func (e *editor) set(m *yaml.Node, key, value, what string) bool {
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
	if n := get(m, key); n != nil {
		return n
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

// addSeq appends the values that are not in the sequence yet, creating the key
// when it is absent. Globs are quoted and the sequence is a flow sequence.
func (e *editor) addSeq(m *yaml.Node, key string, values []string, glob bool, what string) {
	seq := get(m, key)
	if seq == nil {
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		if glob {
			seq.Style = yaml.FlowStyle
		}
		m.Content = append(m.Content, scalar(key, false), seq)
	}
	if seq.Kind != yaml.SequenceNode {
		return
	}
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
	empty := seq != nil && ((seq.Kind == yaml.ScalarNode && seq.Tag == "!!null") || (seq.Kind == yaml.SequenceNode && len(seq.Content) == 0))
	switch {
	case seq == nil:
		seq = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		e.root.Content = append(e.root.Content, scalar("waivers", false), seq)
	case empty && seq.Line == 0: // replaced already, by an earlier step
	case empty: // no value, or an empty flow list: a block list in its place
		for i := 0; i+1 < len(e.root.Content); i += 2 {
			if e.root.Content[i+1] == seq {
				fresh := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
				e.replaced[e.root.Content[i]] = seq
				e.root.Content[i+1] = fresh
				seq = fresh
				break
			}
		}
	}
	if seq.Kind != yaml.SequenceNode {
		return
	}
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
