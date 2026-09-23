package datalog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Default resource limits. Both are far above what a design projection
// produces; they exist so a runaway program fails instead of exhausting memory.
const (
	DefaultMaxTuples     = 10_000_000
	DefaultMaxIterations = 1_000_000
)

// ErrLimit is wrapped by the error Run returns when a resource limit is hit.
var ErrLimit = errors.New("datalog: resource limit exceeded")

// Inputs maps an input relation name to its tuples. Order and duplicates do
// not matter: relations are sets and inputs are sorted before loading.
type Inputs map[string][][]string

// Options controls a Run. Zero values select the defaults.
type Options struct {
	// Explain records the first derivation of every derived tuple.
	Explain bool
	// MaxTuples bounds the total number of tuples held across all relations,
	// input facts included.
	MaxTuples int
	// MaxIterations bounds the total number of evaluation rounds across all
	// strata (each stratum takes at least one).
	MaxIterations int
}

type ref struct {
	rel int32
	idx int32
}

type prov struct {
	rule int // 0 for an input fact
	kids []ref
}

type relation struct {
	arity  int
	tuples [][]string
	set    map[string]int32
	idx    map[uint64]map[string][]int32
	prov   []prov
}

func newRelation(arity int) *relation {
	return &relation{arity: arity, set: map[string]int32{}, idx: map[uint64]map[string][]int32{}}
}

func appendKey(b []byte, s string) []byte {
	b = binary.AppendUvarint(b, uint64(len(s)))
	return append(b, s...)
}

func tupleKey(t []string) string {
	var b []byte
	for _, s := range t {
		b = appendKey(b, s)
	}
	return string(b)
}

func maskKey(t []string, mask uint64) string {
	var b []byte
	for j, s := range t {
		if mask&(1<<uint(j)) != 0 {
			b = appendKey(b, s)
		}
	}
	return string(b)
}

// index returns the index over the columns in mask, building it on first use.
func (r *relation) index(mask uint64) map[string][]int32 {
	ix, ok := r.idx[mask]
	if !ok {
		ix = map[string][]int32{}
		for i, t := range r.tuples {
			k := maskKey(t, mask)
			ix[k] = append(ix[k], int32(i))
		}
		r.idx[mask] = ix
	}
	return ix
}

// insert adds t if absent and reports whether it was added.
func (r *relation) insert(t []string, p prov, explain bool) bool {
	k := tupleKey(t)
	if _, ok := r.set[k]; ok {
		return false
	}
	i := int32(len(r.tuples))
	r.set[k] = i
	r.tuples = append(r.tuples, t)
	for mask, ix := range r.idx {
		mk := maskKey(t, mask)
		ix[mk] = append(ix[mk], i)
	}
	if explain {
		r.prov = append(r.prov, p)
	}
	return true
}

type span struct{ lo, hi int }

type evaluator struct {
	prog     *Program
	rels     []*relation
	explain  bool
	total    int
	maxTup   int
	maxIter  int
	iters    int
	memo     []map[string]aggValue
	curStrat int
}

type aggValue struct {
	ok  bool
	val string
}

func validSymbol(s string) bool {
	return !strings.ContainsAny(s, "\t\n\r")
}

// Run evaluates the program over inputs. It never returns a partial result:
// on any error the Result is nil.
func (p *Program) Run(in Inputs, opt Options) (*Result, error) {
	ev := &evaluator{
		prog:    p,
		explain: opt.Explain,
		maxTup:  opt.MaxTuples,
		maxIter: opt.MaxIterations,
		memo:    make([]map[string]aggValue, p.naggs),
	}
	if ev.maxTup <= 0 {
		ev.maxTup = DefaultMaxTuples
	}
	if ev.maxIter <= 0 {
		ev.maxIter = DefaultMaxIterations
	}
	for i := range ev.memo {
		ev.memo[i] = map[string]aggValue{}
	}
	ev.rels = make([]*relation, len(p.decls))
	for i, d := range p.decls {
		ev.rels[i] = newRelation(len(d.Attrs))
	}
	if err := ev.load(in); err != nil {
		return nil, err
	}
	for si, st := range p.strata {
		ev.curStrat = si
		if err := ev.stratum(st); err != nil {
			return nil, err
		}
	}
	return &Result{prog: p, rels: ev.rels, explain: opt.Explain}, nil
}

func (ev *evaluator) load(in Inputs) error {
	for _, ri := range ev.prog.inputs {
		d := ev.prog.decls[ri]
		rows, ok := in[d.Name]
		if !ok {
			return fmt.Errorf("datalog: no input supplied for .input relation %q", d.Name)
		}
		for n, row := range rows {
			if len(row) != len(d.Attrs) {
				return fmt.Errorf("datalog: input %q row %d has %d columns, relation arity is %d", d.Name, n+1, len(row), len(d.Attrs))
			}
			for _, s := range row {
				if !validSymbol(s) {
					return fmt.Errorf("datalog: input %q row %d: a symbol may not contain a tab, newline or carriage return", d.Name, n+1)
				}
			}
		}
		sorted := make([][]string, len(rows))
		copy(sorted, rows)
		sort.Slice(sorted, func(a, b int) bool { return compareTuples(sorted[a], sorted[b], nil) < 0 })
		for _, row := range sorted {
			t := append([]string(nil), row...)
			if ev.rels[ri].insert(t, prov{}, ev.explain) {
				if err := ev.count(ri); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ev *evaluator) count(ri int) error {
	ev.total++
	if ev.total > ev.maxTup {
		return fmt.Errorf("%w: more than %d tuples (while filling %q)", ErrLimit, ev.maxTup, ev.prog.decls[ri].Name)
	}
	return nil
}

// stratum runs one component to its fixed point, semi-naively: after the
// first round, each rule is re-run once per recursive body atom, with that
// atom reading only the previous round's new tuples, earlier recursive atoms
// reading only older tuples, and later ones reading everything.
func (ev *evaluator) stratum(st *stratum) error {
	full := func() []span {
		sp := make([]span, len(ev.rels))
		for i, r := range ev.rels {
			sp[i] = span{0, len(r.tuples)}
		}
		return sp
	}
	if err := ev.round(); err != nil {
		return err
	}
	start := full()
	for _, r := range st.rules {
		ranges := make([]span, len(r.body.atoms))
		for k, a := range r.body.atoms {
			ranges[k] = start[a.rel]
		}
		if err := ev.fire(r, ranges); err != nil {
			return err
		}
	}
	if !st.recursive {
		return nil
	}
	prev := start
	for {
		cur := full()
		changed := false
		for _, ri := range st.rels {
			if cur[ri].hi > prev[ri].hi {
				changed = true
			}
		}
		if !changed {
			return nil
		}
		if err := ev.round(); err != nil {
			return err
		}
		for _, r := range st.rules {
			for k, a := range r.body.atoms {
				if !st.inStratum[a.rel] || cur[a.rel].hi == prev[a.rel].hi {
					continue
				}
				ranges := make([]span, len(r.body.atoms))
				for j, b := range r.body.atoms {
					switch {
					case !st.inStratum[b.rel]:
						ranges[j] = cur[b.rel]
					case j < k:
						ranges[j] = span{0, prev[b.rel].hi}
					case j == k:
						ranges[j] = span{prev[b.rel].hi, cur[b.rel].hi}
					default:
						ranges[j] = cur[b.rel]
					}
				}
				if err := ev.fire(r, ranges); err != nil {
					return err
				}
			}
		}
		prev = cur
	}
}

func (ev *evaluator) round() error {
	ev.iters++
	if ev.iters > ev.maxIter {
		return fmt.Errorf("%w: more than %d iterations", ErrLimit, ev.maxIter)
	}
	return nil
}

// fire evaluates one rule with the given per-atom ranges and inserts the
// head tuples it derives.
func (ev *evaluator) fire(r *crule, ranges []span) error {
	binding := make([]string, r.body.nslots)
	matched := make([]ref, len(r.body.atoms))
	head := ev.rels[r.head]
	return ev.join(r.body, ranges, binding, matched, 0, func() error {
		t := make([]string, len(r.headArgs))
		for j, a := range r.headArgs {
			if a.kind == argConst {
				t[j] = a.val
			} else {
				t[j] = binding[a.slot]
			}
		}
		var p prov
		if ev.explain {
			p = prov{rule: r.src.index, kids: append([]ref(nil), matched...)}
		}
		if head.insert(t, p, ev.explain) {
			return ev.count(r.head)
		}
		return nil
	})
}

func (ev *evaluator) filtersPass(fs []filter, binding []string) bool {
	for _, f := range fs {
		if f.neg {
			if ev.exists(f.atom, binding) {
				return false
			}
			continue
		}
		eq := value(f.l, binding) == value(f.r, binding)
		if eq == f.neq {
			return false
		}
	}
	return true
}

func value(a argRef, binding []string) string {
	if a.kind == argConst {
		return a.val
	}
	return binding[a.slot]
}

func (ev *evaluator) lookupKey(a catom, binding []string) string {
	var b []byte
	for j, arg := range a.args {
		if a.mask&(1<<uint(j)) != 0 {
			b = appendKey(b, value(arg, binding))
		}
	}
	return string(b)
}

func (ev *evaluator) exists(a catom, binding []string) bool {
	r := ev.rels[a.rel]
	if a.mask == 0 {
		return len(r.tuples) > 0
	}
	return len(r.index(a.mask)[ev.lookupKey(a, binding)]) > 0
}

// join enumerates the bindings of plan step k onward. Steps are the plan's
// positive atoms followed by its aggregates; filters run at the earliest
// stage where their variables are bound.
func (ev *evaluator) join(pl *plan, ranges []span, binding []string, matched []ref, k int, emit func() error) error {
	if !ev.filtersPass(pl.filters[k], binding) {
		return nil
	}
	if k < len(pl.atoms) {
		a := pl.atoms[k]
		r := ev.rels[a.rel]
		sp := ranges[k]
		visit := func(i int) error {
			t := r.tuples[i]
			for j, arg := range a.args {
				switch arg.kind {
				case argBind:
					binding[arg.slot] = t[j]
				case argCheck:
					// The index guarantees equality for masked columns;
					// a variable repeated within one atom is checked here.
					if binding[arg.slot] != t[j] {
						return nil
					}
				}
			}
			matched[k] = ref{rel: int32(a.rel), idx: int32(i)}
			return ev.join(pl, ranges, binding, matched, k+1, emit)
		}
		if a.mask == 0 {
			for i := sp.lo; i < sp.hi; i++ {
				if err := visit(i); err != nil {
					return err
				}
			}
			return nil
		}
		ids := r.index(a.mask)[ev.lookupKey(a, binding)]
		from := sort.Search(len(ids), func(x int) bool { return int(ids[x]) >= sp.lo })
		for x := from; x < len(ids) && int(ids[x]) < sp.hi; x++ {
			if err := visit(int(ids[x])); err != nil {
				return err
			}
		}
		return nil
	}
	if ag := k - len(pl.atoms); ag < len(pl.aggs) {
		g := pl.aggs[ag]
		v := ev.aggregate(g, binding)
		if !v.ok {
			return nil
		}
		if g.resultBound {
			if binding[g.resultSlot] != v.val {
				return nil
			}
		} else {
			binding[g.resultSlot] = v.val
		}
		return ev.join(pl, ranges, binding, matched, k+1, emit)
	}
	return emit()
}

// aggregate computes one aggregate for the grouping values in binding. The
// relations it reads belong to earlier strata, so results are memoized for
// the whole run.
func (ev *evaluator) aggregate(g *caggr, outer []string) aggValue {
	inner := make([]string, g.inner.nslots)
	var kb []byte
	for i, s := range g.groupOuter {
		inner[i] = outer[s]
		kb = appendKey(kb, outer[s])
	}
	key := string(kb)
	if v, ok := ev.memo[g.id][key]; ok {
		return v
	}
	ranges := make([]span, len(g.inner.atoms))
	for k, a := range g.inner.atoms {
		ranges[k] = span{0, len(ev.rels[a.rel].tuples)}
	}
	matched := make([]ref, len(g.inner.atoms))
	var n int64
	var best int64
	found := false
	// The inner emit never fails, so neither does the join.
	_ = ev.join(g.inner, ranges, inner, matched, 0, func() error {
		n++
		if g.kind == aggMin {
			x, _ := strconv.ParseInt(inner[g.targetSlot], 10, 64)
			if !found || x < best {
				best = x
			}
			found = true
		}
		return nil
	})
	var v aggValue
	if g.kind == aggCount {
		v = aggValue{ok: true, val: strconv.FormatInt(n, 10)}
	} else if found {
		v = aggValue{ok: true, val: strconv.FormatInt(best, 10)}
	}
	ev.memo[g.id][key] = v
	return v
}
