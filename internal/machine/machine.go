// Package machine loads machine files and derives their transition cases and
// runtime tables (docs/formats.md sections 5.2, 7.2 and 7.3).
package machine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/project"
)

// Suffix of a machine file.
const Suffix = ".machine.json"

// Machine is one loaded machine file.
type Machine struct {
	Path        string // project-relative
	Base        string // file name without the suffix
	ID          string
	Tag         string
	LifecycleOf string // from _lifecycle_of ("" when absent)
	Role        string // from _role ("" when absent)
	Root        *ir.Value
	Bytes       []byte
	Resolver    *ir.Resolver
	Table       *Table
}

// State is a row of the table's states list.
type State struct {
	Path    string   `json:"path"`
	Kind    string   `json:"kind"`
	Resting bool     `json:"resting"`
	Entry   []string `json:"entry"`
	Exit    []string `json:"exit"`
	Invoke  []string `json:"invoke"`
}

// Transition is a transition case.
type Transition struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Kind    string   `json:"kind"`
	Event   string   `json:"event"`
	Guard   string   `json:"guard"`
	Target  string   `json:"target"`
	Actions []string `json:"actions"`
	Effects []string `json:"effects"`
	// Requires lists who may cause the transition (_requires), sorted;
	// absent when the branch names nobody.
	Requires []string `json:"requires,omitempty"`
}

// Ignore is an ignore case.
type Ignore struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Event  string `json:"event"`
	Reason string `json:"reason"`
}

// Refusal is a refusal case.
type Refusal struct {
	ID          string `json:"id"`
	State       string `json:"state"`
	Handler     string `json:"handler"`
	Disposition string `json:"disposition"`
}

// Clause is a guard clause case.
type Clause struct {
	ID         string `json:"id"`
	Transition string `json:"transition"`
	Guard      string `json:"guard"`
	Clause     string `json:"clause"`
}

// Units lists the named units a machine uses.
type Units struct {
	Guards  []string `json:"guards"`
	Actions []string `json:"actions"`
	Actors  []string `json:"actors"`
}

// Table is the runtime table of section 7.3.
type Table struct {
	Version      int            `json:"version"`
	Machine      string         `json:"machine"`
	Tag          string         `json:"tag"`
	Source       string         `json:"source"`
	SourceSHA256 string         `json:"source_sha256"`
	Initial      string         `json:"initial"`
	Entries      []string       `json:"entries,omitempty"`
	States       []State        `json:"states"`
	Transitions  []Transition   `json:"transitions"`
	Ignores      []Ignore       `json:"ignores"`
	Refusals     []Refusal      `json:"refusals"`
	Clauses      []Clause       `json:"clauses"`
	Units        Units          `json:"units"`
	Delays       map[string]int `json:"delays"`
}

// Case is one transition case of any kind, as listed in obligations.json.
type Case struct {
	ID      string
	Kind    string // transition | ignore | refusal | clause
	Title   string
	Effects []string
}

// Trigger renders the trigger of a transition row (section 7.2).
func Trigger(kind, event string) string {
	switch kind {
	case "always", "stateDone":
		return kind
	}
	return kind + ":" + event
}

// LoadDir loads every machine file of the project's machines directory, sorted
// by path. A file that cannot be parsed is reported in problems and skipped.
func LoadDir(p *project.Project) (ms []*Machine, problems []string, err error) {
	dir := p.Cfg.Design.Machines
	if dir == "" {
		return nil, nil, nil
	}
	files, err := p.Glob([]string{strings.TrimSuffix(dir, "/") + "/*" + Suffix})
	if err != nil {
		return nil, nil, err
	}
	for _, f := range files {
		data, rerr := p.Read(f)
		if rerr != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", f, rerr))
			continue
		}
		m, perr := Parse(f, data)
		if perr != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", f, perr))
			continue
		}
		ms = append(ms, m)
	}
	return ms, problems, nil
}

// Parse reads one machine document and builds its table.
func Parse(rel string, data []byte) (*Machine, error) {
	root, err := ir.LoadMachineJSONBytes(rel, data)
	if err != nil {
		return nil, err
	}
	ro := root.AsObject()
	if ro == nil {
		return nil, fmt.Errorf("machine must be a JSON object")
	}
	base := strings.TrimSuffix(path.Base(rel), Suffix)
	m := &Machine{Path: rel, Base: base, Root: root, Bytes: data}
	m.ID = ro.GetString("id")
	if m.ID == "" {
		m.ID = base
	}
	m.Tag = ro.GetString("_oracle_tag")
	if m.Tag == "" {
		tag := strings.ToUpper(m.ID)
		if r := []rune(tag); len(r) > 4 {
			tag = string(r[:4])
		}
		m.Tag = tag
	}
	m.LifecycleOf = ro.GetString("_lifecycle_of")
	m.Role = ro.GetString("_role")
	m.Resolver = ir.NewResolver(root)
	m.Table = buildTable(m)
	return m, nil
}

// GuardClauses returns the declared clauses of each guard (root _guards).
func (m *Machine) GuardClauses() map[string][]string {
	res := map[string][]string{}
	g := m.Root.AsObject().Get2("_guards")
	if g == nil || g.Kind != ir.KindObject {
		return res
	}
	for _, k := range g.AsObject().Keys() {
		v := g.AsObject().Get2(k)
		if v == nil || v.Kind != ir.KindArray {
			continue
		}
		for _, c := range v.AsArray() {
			if c != nil && c.Kind == ir.KindString {
				res[k] = append(res[k], c.AsString())
			}
		}
	}
	return res
}

var leadingInt = regexp.MustCompile(`^\s*([0-9]+)`)

type rawCase struct {
	hash string
	set  func(id string)
}

func stableHash(tag, source, trig, guard string) string {
	h := sha256.Sum256([]byte(tag + "|" + source + "|" + trig + "|" + guard))
	return hex.EncodeToString(h[:])
}

func stringsOrEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func buildTable(m *Machine) *Table {
	ro := m.Root.AsObject()
	sum := sha256.Sum256(m.Bytes)
	t := &Table{
		Version: 1, Machine: m.ID, Tag: m.Tag, Source: m.Path,
		SourceSHA256: hex.EncodeToString(sum[:]), Initial: ro.GetString("initial"),
		States: []State{}, Transitions: []Transition{}, Ignores: []Ignore{}, Refusals: []Refusal{},
		Clauses: []Clause{}, Delays: map[string]int{}, Entries: sortedCopy(m.Entries()),
	}
	states := ir.WalkStates(ro.Get2("states"), "")
	guards, actions, actors := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, s := range states {
		o := s.Node.AsObject()
		if o == nil {
			continue
		}
		kind := "atomic"
		if o.GetString("type") == "final" {
			kind = "final"
		} else if o.Get2("states") != nil {
			kind = "compound"
		}
		entry := ir.ActionNames(o.Get2("entry"), nil, "")
		exit := ir.ActionNames(o.Get2("exit"), nil, "")
		var invokes []string
		for _, iv := range ir.InvokesOf(s.Node) {
			if iv != nil && iv.Kind == ir.KindObject {
				if src := iv.AsObject().GetString("src"); src != "" {
					invokes = append(invokes, src)
					actors[src] = true
				}
			}
		}
		for _, a := range append(append([]string{}, entry...), exit...) {
			actions[a] = true
		}
		resting := !strings.Contains(s.Path, ".") && kind != "final" && len(invokes) == 0 && o.Get2("always") == nil
		t.States = append(t.States, State{Path: s.Path, Kind: kind, Resting: resting,
			Entry: stringsOrEmpty(entry), Exit: stringsOrEmpty(exit), Invoke: stringsOrEmpty(invokes)})
	}

	clausesOf := m.GuardClauses()
	var raws []rawCase
	type pendingClause struct {
		tIdx          int
		guard, clause string
		idx           int
	}
	var pclauses []pendingClause
	for _, s := range states {
		for _, tr := range ir.TransitionsOf(s.Node, nil, s.Path) {
			target := ""
			if tr.Target != "" {
				if dest, why := m.Resolver.Resolve(tr.Target, s.Path); why == "" {
					target = dest
				} else {
					target = tr.Target // unresolved: the machines stage reports it
				}
			}
			effects := m.Step(activeLeaf(m, s.Path, tr.Kind), s.Path, target, tr.Actions)
			for _, a := range tr.Actions {
				actions[a] = true
			}
			if tr.Guard != "" {
				guards[tr.Guard] = true
			}
			row := Transition{Source: s.Path, Kind: tr.Kind, Event: tr.Event, Guard: tr.Guard, Target: target,
				Actions: stringsOrEmpty(tr.Actions), Effects: stringsOrEmpty(effects), Requires: sortedCopy(tr.Requires)}
			t.Transitions = append(t.Transitions, row)
			idx := len(t.Transitions) - 1
			trig := Trigger(tr.Kind, tr.Event)
			raws = append(raws, rawCase{stableHash(m.Tag, s.Path, trig, tr.Guard), func(id string) { t.Transitions[idx].ID = id }})
			if tr.Guard != "" {
				for _, c := range clausesOf[tr.Guard] {
					t.Clauses = append(t.Clauses, Clause{Guard: tr.Guard, Clause: c})
					ci := len(t.Clauses) - 1
					pclauses = append(pclauses, pendingClause{idx, tr.Guard, c, ci})
					raws = append(raws, rawCase{stableHash(m.Tag, s.Path, trig, tr.Guard+"!"+c), func(id string) { t.Clauses[ci].ID = id }})
				}
			}
		}
	}
	for _, s := range states {
		o := s.Node.AsObject()
		if o == nil {
			continue
		}
		if ig := o.Get2("_ignores"); ig != nil && ig.Kind == ir.KindObject {
			for _, ev := range ig.AsObject().Keys() {
				t.Ignores = append(t.Ignores, Ignore{State: s.Path, Event: ev, Reason: ig.AsObject().GetString(ev)})
				i := len(t.Ignores) - 1
				raws = append(raws, rawCase{stableHash(m.Tag, s.Path, "ignore:"+ev, ""), func(id string) { t.Ignores[i].ID = id }})
			}
		}
		if rf := o.Get2("_refusal"); rf != nil && rf.Kind == ir.KindObject {
			for _, h := range rf.AsObject().Keys() {
				t.Refusals = append(t.Refusals, Refusal{State: s.Path, Handler: h, Disposition: rf.AsObject().GetString(h)})
				i := len(t.Refusals) - 1
				raws = append(raws, rawCase{stableHash(m.Tag, s.Path, "refusal:"+h, ""), func(id string) { t.Refusals[i].ID = id }})
			}
		}
	}
	assignIDs(m.Tag, raws)
	for _, pc := range pclauses {
		t.Clauses[pc.idx].Transition = t.Transitions[pc.tIdx].ID
	}

	t.Units = Units{Guards: sortedKeys(guards), Actions: sortedKeys(actions), Actors: sortedKeys(actors)}
	if d := ro.Get2("_delays"); d != nil && d.Kind == ir.KindObject {
		for _, name := range d.AsObject().Keys() {
			v := d.AsObject().Get2(name)
			text := ""
			if v != nil {
				switch v.Kind {
				case ir.KindString:
					text = v.AsString()
				case ir.KindNumber:
					text = string(v.AsNumber())
				}
			}
			if mm := leadingInt.FindStringSubmatch(text); mm != nil {
				n, _ := strconv.Atoi(mm[1])
				t.Delays[name] = n
			}
		}
	}
	return t
}

// activeLeaf is the state taken as active when a row of the given kind fires
// from source: the source's initial leaf, or for a state onDone the source's
// first final child, since only a final child completes it.
func activeLeaf(m *Machine, source, kind string) string {
	if kind == "stateDone" {
		if n := m.Resolver.NodeOf[source]; n != nil && n.Kind == ir.KindObject {
			if ch := n.AsObject().Get2("states"); ch != nil && ch.Kind == ir.KindObject {
				for _, name := range ch.AsObject().Keys() {
					c := ch.AsObject().Get2(name)
					if c != nil && c.Kind == ir.KindObject && c.AsObject().GetString("type") == "final" {
						return source + "." + name
					}
				}
			}
		}
	}
	return m.Resolver.Leaf(source)
}

// Step returns the named actions XState 5 executes, in order and deduplicated,
// when a transition defined on source, with the given target ("" for an
// internal transition) and actions, fires while active is the active leaf:
// the exit actions of the states left, innermost first; the transition's
// actions; the entry actions of the states entered, outermost first, down
// through initial children; and, when a top-level final state is entered, its
// exit actions, because the machine stops. A target that is the source or one
// of its descendants does not exit and re-enter the source.
func (m *Machine) Step(active, source, target string, actions []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(names []string) {
		for _, n := range names {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	if target == "" {
		add(actions)
		return stringsOrEmpty(out)
	}
	domain := source
	if target != source && !IsDescendant(target, source) {
		domain = ""
		for anc := Parent(target); anc != ""; anc = Parent(anc) {
			if IsDescendant(source, anc) {
				domain = anc
				break
			}
		}
	}
	for p := active; p != ""; p = Parent(p) {
		if IsDescendant(p, domain) {
			add(m.actionsOf(p, "exit"))
		}
	}
	add(actions)
	leaf := m.Resolver.Leaf(target)
	var chain []string
	for p := leaf; p != ""; p = Parent(p) {
		chain = append([]string{p}, chain...)
	}
	for _, p := range chain {
		if IsDescendant(p, domain) {
			add(m.actionsOf(p, "entry"))
		}
	}
	if !strings.Contains(leaf, ".") && m.isFinal(leaf) {
		add(m.actionsOf(leaf, "exit"))
	}
	return stringsOrEmpty(out)
}

func (m *Machine) actionsOf(path, key string) []string {
	n := m.Resolver.NodeOf[path]
	if n == nil || n.Kind != ir.KindObject {
		return nil
	}
	return ir.ActionNames(n.AsObject().Get2(key), nil, "")
}

func (m *Machine) isFinal(path string) bool {
	n := m.Resolver.NodeOf[path]
	return n != nil && n.Kind == ir.KindObject && n.AsObject().GetString("type") == "final"
}

// Parent returns the dotted path of a state's parent, "" for a top-level state.
func Parent(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}

// IsDescendant reports whether path is a proper descendant of anc; every state
// descends from the root, written "".
func IsDescendant(path, anc string) bool {
	if anc == "" {
		return path != ""
	}
	return strings.HasPrefix(path, anc+".")
}

// assignIDs gives every case its tc id: the 6-hex prefix, extended two
// characters at a time where different cases share a prefix. Identical
// stimuli (a lint error) get a positional suffix so ids stay unique.
func assignIDs(tag string, raws []rawCase) {
	byPrefix := map[string]map[string]bool{}
	for _, r := range raws {
		p := r.hash[:6]
		if byPrefix[p] == nil {
			byPrefix[p] = map[string]bool{}
		}
		byPrefix[p][r.hash] = true
	}
	seen := map[string]int{}
	for _, r := range raws {
		n := 6
		for n < len(r.hash) {
			unique := true
			for other := range byPrefix[r.hash[:6]] {
				if other != r.hash && other[:n] == r.hash[:n] {
					unique = false
					break
				}
			}
			if unique {
				break
			}
			n += 2
		}
		id := "tc:" + tag + "-" + r.hash[:n]
		if c, ok := seen[id]; ok {
			seen[id] = c + 1
			id = fmt.Sprintf("%s.%d", id, c+1)
		} else {
			seen[id] = 1
		}
		r.set(id)
	}
}

// sortedCopy returns the names sorted, or nil when there are none.
func sortedCopy(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	res := append([]string(nil), names...)
	sort.Strings(res)
	return res
}

func sortedKeys(m map[string]bool) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

// Cases lists every case of the table with its title, in table order:
// transitions, ignores, refusals, clauses.
func (t *Table) Cases() []Case {
	var cs []Case
	byID := map[string]Transition{}
	for _, tr := range t.Transitions {
		byID[tr.ID] = tr
		cs = append(cs, Case{ID: tr.ID, Kind: "transition", Title: transitionTitle(tr, ""), Effects: tr.Effects})
	}
	for _, ig := range t.Ignores {
		cs = append(cs, Case{ID: ig.ID, Kind: "ignore", Title: fmt.Sprintf("%s ignore:%s -> (unchanged)", ig.State, ig.Event), Effects: []string{}})
	}
	for _, rf := range t.Refusals {
		cs = append(cs, Case{ID: rf.ID, Kind: "refusal", Title: fmt.Sprintf("%s refusal:%s -> (unchanged)", rf.State, rf.Handler), Effects: []string{}})
	}
	for _, cl := range t.Clauses {
		cs = append(cs, Case{ID: cl.ID, Kind: "clause", Title: transitionTitle(byID[cl.Transition], cl.Clause), Effects: []string{}})
	}
	return cs
}

func transitionTitle(tr Transition, clause string) string {
	var b strings.Builder
	b.WriteString(tr.Source + " " + Trigger(tr.Kind, tr.Event))
	if clause != "" {
		fmt.Fprintf(&b, " [%s!%s] -> (branch does not fire)", tr.Guard, clause)
		return b.String()
	}
	if tr.Guard != "" {
		fmt.Fprintf(&b, " [%s]", tr.Guard)
	}
	if tr.Target == "" {
		b.WriteString(" -> (internal)")
	} else {
		b.WriteString(" -> " + tr.Target)
	}
	return b.String()
}

// Events returns the distinct event names of on transitions, sorted.
func (t *Table) Events() []string {
	set := map[string]bool{}
	for _, tr := range t.Transitions {
		if tr.Kind == "on" {
			set[tr.Event] = true
		}
	}
	return sortedKeys(set)
}
