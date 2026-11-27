package adopt

import (
	"bytes"
	"fmt"
	"strings"

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

// editor changes machinery.yaml through its yaml.Node tree, so comments and the
// order of the keys the team wrote are kept.
type editor struct {
	doc     *yaml.Node
	root    *yaml.Node
	changes []string
}

func newEditor(existing []byte) (*editor, error) {
	e := &editor{doc: &yaml.Node{}}
	if existing == nil {
		e.root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		e.doc = &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{e.root}}
		e.set(e.root, "version", "2", "version")
		return e, nil
	}
	if err := yaml.Unmarshal(existing, e.doc); err != nil {
		return nil, project.Errf("%s: %v", project.ConfigName, err)
	}
	if e.doc.Kind != yaml.DocumentNode || len(e.doc.Content) != 1 || e.doc.Content[0].Kind != yaml.MappingNode {
		return nil, project.Errf("%s: the top level must be a mapping", project.ConfigName)
	}
	e.root = e.doc.Content[0]
	return e, nil
}

// bytes renders the document, or returns nil when nothing changed.
func (e *editor) bytes() ([]byte, error) {
	if len(e.changes) == 0 {
		return nil, nil
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(e.doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	if seq == nil || (seq.Kind == yaml.ScalarNode && seq.Tag == "!!null") {
		fresh := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		if seq == nil {
			e.root.Content = append(e.root.Content, scalar("waivers", false), fresh)
		} else {
			*seq = *fresh
		}
		seq = get(e.root, "waivers")
	}
	if seq.Kind != yaml.SequenceNode {
		return
	}
	if len(seq.Content) == 0 {
		seq.Style = 0 // an empty flow list becomes a block list
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
