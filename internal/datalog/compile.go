package datalog

import (
	"fmt"
	"sort"
	"strings"
)

// maxArity bounds a relation's attribute count; index masks are 64-bit.
const maxArity = 64

// Program is a parsed, checked and stratified program. It is immutable, so
// concurrent Run calls on one Program are safe.
type Program struct {
	name    string
	decls   []*Decl
	relIdx  map[string]int
	rules   []*crule
	strata  []*stratum
	outputs []int // relation indices, declaration order
	inputs  []int
	naggs   int
}

type stratum struct {
	rels      []int
	inStratum map[int]bool
	rules     []*crule
	recursive bool
}

type argKind int

const (
	argWild  argKind = iota
	argConst         // compare with val
	argBind          // bind slot
	argCheck         // compare with bound slot
)

type argRef struct {
	kind argKind
	slot int
	val  string
}

type catom struct {
	rel  int
	args []argRef
	mask uint64 // columns fixed before the lookup (const or check)
}

type filter struct {
	neg  bool
	atom catom // neg: every arg is wild, const or check
	neq  bool  // comparison
	l, r argRef
}

type plan struct {
	nslots  int
	atoms   []catom
	aggs    []*caggr
	filters [][]filter // by stage: 0 before any step, k after step k
}

type caggr struct {
	id          int // index into the per-run memo table
	kind        aggKind
	resultSlot  int
	resultBound bool  // the result variable is already bound: compare
	groupOuter  []int // outer slots of the grouping variables
	targetSlot  int   // min: inner slot of the target variable
	inner       *plan // inner slots 0..len(groupOuter)-1 hold the grouping values
}

type crule struct {
	src      *rule
	head     int
	headArgs []argRef // argConst or argCheck
	body     *plan
}

type edgeKind int

const (
	edgePos edgeKind = iota
	edgeNeg
	edgeAgg
)

type depEdge struct {
	from, to int // from: body relation, to: head relation
	kind     edgeKind
	pos      Pos
}

func (p *Program) names(idx []int) []string {
	var names []string
	for _, ri := range idx {
		names = append(names, p.decls[ri].Name)
	}
	return names
}

// InputRelations returns the names of the .input relations in declaration
// order.
func (p *Program) InputRelations() []string { return p.names(p.inputs) }

// OutputRelations returns the names of the .output relations in declaration
// order.
func (p *Program) OutputRelations() []string { return p.names(p.outputs) }

// Relations returns the names of every declared relation in declaration
// order.
func (p *Program) Relations() []string {
	names := make([]string, len(p.decls))
	for i, d := range p.decls {
		names[i] = d.Name
	}
	return names
}

// Decl returns a copy of a relation's declaration: its attribute names and
// types, its IO directives and its position. ok is false for an undeclared
// relation.
func (p *Program) Decl(name string) (d Decl, ok bool) {
	ri, ok := p.relIdx[name]
	if !ok {
		return Decl{}, false
	}
	d = *p.decls[ri]
	d.Attrs = append([]Attr(nil), d.Attrs...)
	return d, true
}

// Parse parses and checks a program. name labels positions in errors.
func Parse(src, name string) (*Program, error) {
	ps, err := parse(name, src)
	if err != nil {
		return nil, err
	}
	c := &compiler{name: name, prog: &Program{name: name, relIdx: map[string]int{}}}
	if err := c.compile(ps); err != nil {
		return nil, err
	}
	return c.prog, nil
}

type compiler struct {
	name  string
	prog  *Program
	edges []depEdge
}

func (c *compiler) errAt(p Pos, format string, args ...any) *Error {
	return &Error{Name: c.name, Pos: p, Msg: fmt.Sprintf(format, args...)}
}

func (c *compiler) compile(ps *parsed) error {
	pr := c.prog
	for _, d := range ps.decls {
		if prev, ok := pr.relIdx[d.Name]; ok {
			return c.errAt(d.Pos, "relation %q is declared twice (first at %s)", d.Name, pr.decls[prev].Pos)
		}
		if len(d.Attrs) > maxArity {
			return c.errAt(d.Pos, "relation %q has %d attributes; at most %d are supported", d.Name, len(d.Attrs), maxArity)
		}
		pr.relIdx[d.Name] = len(pr.decls)
		pr.decls = append(pr.decls, d)
	}
	for _, dv := range ps.directives {
		i, ok := pr.relIdx[dv.rel]
		if !ok {
			kind := "input"
			if dv.output {
				kind = "output"
			}
			return c.errAt(dv.pos, ".%s names undeclared relation %q", kind, dv.rel)
		}
		d := pr.decls[i]
		if dv.output {
			if d.Output {
				return c.errAt(dv.pos, "relation %q is marked .output twice", d.Name)
			}
			d.Output = true
		} else {
			if d.Input {
				return c.errAt(dv.pos, "relation %q is marked .input twice", d.Name)
			}
			for _, a := range d.Attrs {
				if a.Type != Symbol {
					return c.errAt(dv.pos, "input relation %q has number attribute %q; number is only for aggregate results", d.Name, a.Name)
				}
			}
			d.Input = true
		}
	}
	for i, d := range pr.decls {
		if d.Output {
			pr.outputs = append(pr.outputs, i)
		}
		if d.Input {
			pr.inputs = append(pr.inputs, i)
		}
	}
	for _, r := range ps.rules {
		cr, err := c.rule(r)
		if err != nil {
			return err
		}
		pr.rules = append(pr.rules, cr)
	}
	return c.stratify()
}

// varInfo tracks one variable while a body is compiled.
type varInfo struct {
	slot    int
	typ     Type
	bound   bool
	byAtom  bool // bound by a positive atom (not by an aggregate result)
	stage   int  // stage after which it is bound
	firstAt Pos
}

type scope struct {
	c    *compiler
	vars map[string]*varInfo
	n    int
}

func (s *scope) get(name string) *varInfo { return s.vars[name] }

func (s *scope) bind(name string, typ Type, stage int, byAtom bool, p Pos) *varInfo {
	v := &varInfo{slot: s.n, typ: typ, bound: true, byAtom: byAtom, stage: stage, firstAt: p}
	s.n++
	s.vars[name] = v
	return v
}

func (c *compiler) decl(a atom) (int, *Decl, error) {
	i, ok := c.prog.relIdx[a.rel]
	if !ok {
		return 0, nil, c.errAt(a.pos, "relation %q is not declared", a.rel)
	}
	d := c.prog.decls[i]
	if len(a.args) != len(d.Attrs) {
		return 0, nil, c.errAt(a.pos, "relation %q has arity %d, used here with %d arguments", a.rel, len(d.Attrs), len(a.args))
	}
	return i, d, nil
}

// positiveAtom compiles a positive atom at the given stage, binding its
// fresh variables.
func (c *compiler) positiveAtom(s *scope, a atom, stage int) (catom, error) {
	ri, d, err := c.decl(a)
	if err != nil {
		return catom{}, err
	}
	ca := catom{rel: ri, args: make([]argRef, len(a.args))}
	for j, t := range a.args {
		col := d.Attrs[j].Type
		switch t.kind {
		case termWild:
			ca.args[j] = argRef{kind: argWild}
		case termConst:
			if col != Symbol {
				return catom{}, c.errAt(t.pos, "string constant in number attribute %q of %q", d.Attrs[j].Name, d.Name)
			}
			ca.args[j] = argRef{kind: argConst, val: t.text}
			ca.mask |= 1 << uint(j)
		case termVar:
			if v := s.get(t.text); v != nil {
				if v.typ != col {
					return catom{}, c.errAt(t.pos, "variable %s is %s but attribute %q of %q is %s", t.text, v.typ, d.Attrs[j].Name, d.Name, col)
				}
				ca.args[j] = argRef{kind: argCheck, slot: v.slot}
				// A variable repeated within this atom is not bound when the
				// lookup key is built, so it is checked per tuple instead.
				if !(v.byAtom && v.stage == stage) {
					ca.mask |= 1 << uint(j)
				}
			} else {
				v := s.bind(t.text, col, stage, true, t.pos)
				ca.args[j] = argRef{kind: argBind, slot: v.slot}
			}
		}
	}
	return ca, nil
}

// boundTerm compiles a term of a negation or comparison; variables must be
// bound. want, when non-nil, is the attribute type the term must have.
func (c *compiler) boundTerm(s *scope, t term, want *Type, what string) (argRef, Type, int, error) {
	switch t.kind {
	case termWild:
		return argRef{kind: argWild}, Symbol, 0, nil
	case termConst:
		if want != nil && *want != Symbol {
			return argRef{}, 0, 0, c.errAt(t.pos, "string constant where a number is expected")
		}
		return argRef{kind: argConst, val: t.text}, Symbol, 0, nil
	}
	v := s.get(t.text)
	if v == nil {
		return argRef{}, 0, 0, c.errAt(t.pos, "variable %s in %s is not bound by a positive body atom", t.text, what)
	}
	if want != nil && v.typ != *want {
		return argRef{}, 0, 0, c.errAt(t.pos, "variable %s is %s where %s is expected", t.text, v.typ, *want)
	}
	return argRef{kind: argCheck, slot: v.slot}, v.typ, v.stage, nil
}

func (c *compiler) filterLit(s *scope, l literal) (filter, int, error) {
	stage := 0
	if l.kind == litNeg {
		ri, d, err := c.decl(l.atom)
		if err != nil {
			return filter{}, 0, err
		}
		f := filter{neg: true, atom: catom{rel: ri, args: make([]argRef, len(l.atom.args))}}
		for j, t := range l.atom.args {
			col := d.Attrs[j].Type
			ar, _, st, err := c.boundTerm(s, t, &col, "a negated atom")
			if err != nil {
				return filter{}, 0, err
			}
			if ar.kind != argWild {
				f.atom.mask |= 1 << uint(j)
			}
			f.atom.args[j] = ar
			stage = max(stage, st)
		}
		return f, stage, nil
	}
	lr, lt, ls, err := c.boundTerm(s, l.left, nil, "a comparison")
	if err != nil {
		return filter{}, 0, err
	}
	rr, rt, rs, err := c.boundTerm(s, l.right, nil, "a comparison")
	if err != nil {
		return filter{}, 0, err
	}
	if lr.kind == argWild || rr.kind == argWild {
		return filter{}, 0, c.errAt(l.pos, "the wildcard _ cannot be compared")
	}
	if lt != rt {
		return filter{}, 0, c.errAt(l.pos, "comparison between %s and %s", lt, rt)
	}
	return filter{neq: l.neq, l: lr, r: rr}, max(ls, rs), nil
}

func termVars(ts []term, into map[string]bool) {
	for _, t := range ts {
		if t.kind == termVar {
			into[t.text] = true
		}
	}
}

func litVars(l literal, into map[string]bool) {
	switch l.kind {
	case litAtom, litNeg:
		termVars(l.atom.args, into)
	case litCmp:
		termVars([]term{l.left, l.right}, into)
	case litAgg:
		into[l.result] = true
		if l.agg == aggMin {
			into[l.target.text] = true
		}
		for _, b := range l.body {
			litVars(b, into)
		}
	}
}

func (c *compiler) rule(r *rule) (*crule, error) {
	hi, hd, err := c.decl(r.head)
	if err != nil {
		return nil, err
	}
	s := &scope{c: c, vars: map[string]*varInfo{}}
	pl := &plan{}
	// Positive atoms first, in source order.
	for _, l := range r.body {
		if l.kind != litAtom {
			continue
		}
		ca, err := c.positiveAtom(s, l.atom, len(pl.atoms)+1)
		if err != nil {
			return nil, err
		}
		pl.atoms = append(pl.atoms, ca)
		c.edges = append(c.edges, depEdge{from: ca.rel, to: hi, kind: edgePos, pos: l.pos})
	}
	// Aggregates next. A variable is a grouping variable of an aggregate
	// when it occurs both inside and outside it.
	for ai, l := range r.body {
		if l.kind != litAgg {
			continue
		}
		outside := map[string]bool{}
		termVars(r.head.args, outside)
		for bj, o := range r.body {
			if bj != ai {
				litVars(o, outside)
			}
		}
		ag, err := c.aggregate(s, l, outside, hi, len(pl.atoms)+len(pl.aggs)+1)
		if err != nil {
			return nil, err
		}
		ag.id = c.prog.naggs
		c.prog.naggs++
		pl.aggs = append(pl.aggs, ag)
	}
	pl.filters = make([][]filter, len(pl.atoms)+len(pl.aggs)+1)
	for _, l := range r.body {
		if l.kind != litNeg && l.kind != litCmp {
			continue
		}
		f, stage, err := c.filterLit(s, l)
		if err != nil {
			return nil, err
		}
		pl.filters[stage] = append(pl.filters[stage], f)
		if f.neg {
			c.edges = append(c.edges, depEdge{from: f.atom.rel, to: hi, kind: edgeNeg, pos: l.pos})
		}
	}
	cr := &crule{src: r, head: hi, headArgs: make([]argRef, len(r.head.args))}
	for j, t := range r.head.args {
		col := hd.Attrs[j].Type
		switch t.kind {
		case termWild:
			return nil, c.errAt(t.pos, "the wildcard _ cannot appear in a rule head")
		case termConst:
			if col != Symbol {
				return nil, c.errAt(t.pos, "string constant in number attribute %q of %q", hd.Attrs[j].Name, hd.Name)
			}
			cr.headArgs[j] = argRef{kind: argConst, val: t.text}
		case termVar:
			v := s.get(t.text)
			if v == nil {
				return nil, c.errAt(t.pos, "variable %s in the head is not bound by a positive body atom", t.text)
			}
			if v.typ != col {
				return nil, c.errAt(t.pos, "variable %s is %s but attribute %q of %q is %s", t.text, v.typ, hd.Attrs[j].Name, hd.Name, col)
			}
			cr.headArgs[j] = argRef{kind: argCheck, slot: v.slot}
		}
	}
	pl.nslots = s.n
	cr.body = pl
	return cr, nil
}

func (c *compiler) aggregate(s *scope, l literal, outside map[string]bool, head, stage int) (*caggr, error) {
	inside := map[string]bool{}
	for _, b := range l.body {
		litVars(b, inside)
	}
	if inside[l.result] {
		return nil, c.errAt(l.pos, "aggregate result %s also occurs inside the aggregate", l.result)
	}
	var group []string
	for v := range inside {
		if outside[v] {
			group = append(group, v)
		}
	}
	sort.Strings(group)
	ag := &caggr{kind: l.agg}
	is := &scope{c: c, vars: map[string]*varInfo{}}
	for _, g := range group {
		ov := s.get(g)
		if ov == nil || !ov.byAtom {
			return nil, c.errAt(l.pos, "variable %s is shared with the aggregate but not bound by a positive atom outside it", g)
		}
		ag.groupOuter = append(ag.groupOuter, ov.slot)
		is.bind(g, ov.typ, 0, true, ov.firstAt)
	}
	if l.agg == aggMin && outside[l.target.text] {
		return nil, c.errAt(l.target.pos, "the min target %s must occur only inside the aggregate", l.target.text)
	}
	inner := &plan{}
	for _, b := range l.body {
		if b.kind != litAtom {
			continue
		}
		ca, err := c.positiveAtom(is, b.atom, len(inner.atoms)+1)
		if err != nil {
			return nil, err
		}
		inner.atoms = append(inner.atoms, ca)
		c.edges = append(c.edges, depEdge{from: ca.rel, to: head, kind: edgeAgg, pos: b.pos})
	}
	if len(inner.atoms) == 0 {
		return nil, c.errAt(l.pos, "an aggregate body needs at least one positive atom")
	}
	inner.filters = make([][]filter, len(inner.atoms)+1)
	for _, b := range l.body {
		if b.kind != litNeg && b.kind != litCmp {
			continue
		}
		f, st, err := c.filterLit(is, b)
		if err != nil {
			return nil, err
		}
		inner.filters[st] = append(inner.filters[st], f)
		if f.neg {
			c.edges = append(c.edges, depEdge{from: f.atom.rel, to: head, kind: edgeAgg, pos: b.pos})
		}
	}
	if l.agg == aggMin {
		tv := is.get(l.target.text)
		if tv == nil {
			return nil, c.errAt(l.target.pos, "the min target %s is not bound inside the aggregate", l.target.text)
		}
		if tv.typ != Number {
			return nil, c.errAt(l.target.pos, "min over symbol variable %s is not supported (Soufflé rejects it); min needs a number attribute", l.target.text)
		}
		ag.targetSlot = tv.slot
	}
	inner.nslots = is.n
	ag.inner = inner
	if v := s.get(l.result); v != nil {
		if v.typ != Number {
			return nil, c.errAt(l.pos, "aggregate result %s is already bound to a symbol", l.result)
		}
		ag.resultBound = true
		ag.resultSlot = v.slot
	} else {
		ag.resultSlot = s.bind(l.result, Number, stage, false, l.pos).slot
	}
	return ag, nil
}

// stratify computes the strongly connected components of the dependency
// graph, rejects negation or aggregation inside a component, and orders the
// components so every relation is evaluated after the relations it reads.
func (c *compiler) stratify() error {
	pr := c.prog
	n := len(pr.decls)
	adj := make([][]int, n)
	for _, e := range c.edges {
		adj[e.from] = append(adj[e.from], e.to)
	}
	comp := make([]int, n)
	for i := range comp {
		comp[i] = -1
	}
	var comps [][]int
	index := make([]int, n)
	low := make([]int, n)
	onStack := make([]bool, n)
	for i := range index {
		index[i] = -1
	}
	var stack []int
	counter := 0
	var strong func(v int)
	strong = func(v int) {
		index[v], low[v] = counter, counter
		counter++
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range adj[v] {
			if index[w] < 0 {
				strong(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var cc []int
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp[w] = len(comps)
				cc = append(cc, w)
				if w == v {
					break
				}
			}
			sort.Ints(cc)
			comps = append(comps, cc)
		}
	}
	for v := 0; v < n; v++ {
		if index[v] < 0 {
			strong(v)
		}
	}
	for _, e := range c.edges {
		if e.kind != edgePos && comp[e.from] == comp[e.to] {
			return c.errAt(e.pos, "%s", c.cycleMessage(e, comp))
		}
	}
	// Tarjan emits a component after every component reachable from it, so
	// with edges pointing from dependency to dependent the reverse of the
	// emission order is a topological order.
	strata := make([]*stratum, len(comps))
	for k := range comps {
		cc := comps[len(comps)-1-k]
		st := &stratum{rels: cc, inStratum: map[int]bool{}}
		for _, r := range cc {
			st.inStratum[r] = true
		}
		strata[k] = st
	}
	byComp := map[int]*stratum{}
	for k, st := range strata {
		byComp[len(comps)-1-k] = st
	}
	for _, r := range pr.rules {
		st := byComp[comp[r.head]]
		st.rules = append(st.rules, r)
		for _, a := range r.body.atoms {
			if st.inStratum[a.rel] {
				st.recursive = true
			}
		}
	}
	for _, st := range strata {
		if len(st.rules) > 0 {
			pr.strata = append(pr.strata, st)
		}
	}
	return nil
}

// cycleMessage names the relations on one cycle through the offending edge.
func (c *compiler) cycleMessage(bad depEdge, comp []int) string {
	decls := c.prog.decls
	word := "negation"
	if bad.kind == edgeAgg {
		word = "aggregation"
	}
	// Shortest path from the edge's head back to its body relation within
	// the component, following dependency edges.
	prev := map[int]int{bad.to: -1}
	queue := []int{bad.to}
	for len(queue) > 0 && bad.to != bad.from {
		v := queue[0]
		queue = queue[1:]
		if v == bad.from {
			break
		}
		for _, e := range c.edges {
			if e.from == v && comp[e.to] == comp[v] {
				if _, seen := prev[e.to]; !seen {
					prev[e.to] = v
					queue = append(queue, e.to)
				}
			}
		}
	}
	var path []int
	for v := bad.from; v != -1; v = prev[v] {
		path = append(path, v)
		if v == bad.to {
			break
		}
	}
	// path runs from body relation back to head; print head first.
	names := []string{decls[bad.from].Name}
	for i := len(path) - 1; i >= 0; i-- {
		names = append(names, decls[path[i]].Name)
	}
	if bad.from == bad.to {
		names = []string{decls[bad.from].Name, decls[bad.from].Name}
	}
	return fmt.Sprintf("relations are not stratifiable: cycle through %s %s (%s depends on %s through %s)",
		word, strings.Join(names, " -> "), decls[bad.to].Name, decls[bad.from].Name, word)
}
