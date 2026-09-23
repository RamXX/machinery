package datalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Result holds the relations computed by one Run. It is read-only.
type Result struct {
	prog    *Program
	rels    []*relation
	explain bool
}

// Derivation is one node of a derivation tree: a tuple, the rule that first
// derived it and the body tuples that rule matched. Input facts are leaves
// with Rule 0.
type Derivation struct {
	Relation string
	Tuple    []string
	Rule     int // 1-based rule index in source order; 0 for an input fact
	Pos      Pos // position of the rule; zero for an input fact
	Children []*Derivation
}

// compareTuples orders tuples column by column: number columns numerically,
// everything else bytewise. types may be nil (all symbol).
func compareTuples(a, b []string, types []Type) int {
	for j := range a {
		if types != nil && types[j] == Number {
			x, _ := strconv.ParseInt(a[j], 10, 64)
			y, _ := strconv.ParseInt(b[j], 10, 64)
			if x != y {
				if x < y {
					return -1
				}
				return 1
			}
			continue
		}
		if c := strings.Compare(a[j], b[j]); c != 0 {
			return c
		}
	}
	return 0
}

func (r *Result) types(ri int) []Type {
	d := r.prog.decls[ri]
	ts := make([]Type, len(d.Attrs))
	for j, a := range d.Attrs {
		ts[j] = a.Type
	}
	return ts
}

// Output returns the tuples of the named relation, sorted (see the package
// documentation). It returns nil for an undeclared relation. Any declared
// relation may be read, not only .output ones.
func (r *Result) Output(name string) [][]string {
	ri, ok := r.prog.relIdx[name]
	if !ok {
		return nil
	}
	rel := r.rels[ri]
	out := make([][]string, len(rel.tuples))
	for i, t := range rel.tuples {
		out[i] = append([]string(nil), t...)
	}
	ts := r.types(ri)
	sort.Slice(out, func(a, b int) bool { return compareTuples(out[a], out[b], ts) < 0 })
	return out
}

// OutputRelations returns the names of the .output relations in
// declaration order.
func (r *Result) OutputRelations() []string { return r.prog.OutputRelations() }

// FormatCSV renders a relation in Soufflé's default output format: sorted
// rows, tab-separated columns, one newline-terminated line per row.
func (r *Result) FormatCSV(name string) string {
	var b strings.Builder
	for _, t := range r.Output(name) {
		b.WriteString(strings.Join(t, "\t"))
		b.WriteByte('\n')
	}
	return b.String()
}

// WriteCSV writes <name>.csv into dir for every .output relation.
func (r *Result) WriteCSV(dir string) error {
	for _, name := range r.OutputRelations() {
		if err := os.WriteFile(filepath.Join(dir, name+".csv"), []byte(r.FormatCSV(name)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Explain returns the derivation tree of a tuple. The Run must have used
// Options.Explain. Shared subtrees are shared pointers; the structure is
// acyclic because a tuple's first derivation only uses earlier tuples.
func (r *Result) Explain(relation string, tuple []string) (*Derivation, error) {
	if !r.explain {
		return nil, fmt.Errorf("datalog: explain needs a Run with Options.Explain")
	}
	ri, ok := r.prog.relIdx[relation]
	if !ok {
		return nil, fmt.Errorf("datalog: relation %q is not declared", relation)
	}
	idx, ok := r.rels[ri].set[tupleKey(tuple)]
	if !ok {
		return nil, fmt.Errorf("datalog: %s(%s) is not in the result", relation, strings.Join(quoteAll(tuple), ", "))
	}
	memo := map[ref]*Derivation{}
	return r.derivation(ref{rel: int32(ri), idx: idx}, memo), nil
}

func (r *Result) derivation(at ref, memo map[ref]*Derivation) *Derivation {
	if d, ok := memo[at]; ok {
		return d
	}
	rel := r.rels[at.rel]
	p := rel.prov[at.idx]
	d := &Derivation{
		Relation: r.prog.decls[at.rel].Name,
		Tuple:    append([]string(nil), rel.tuples[at.idx]...),
		Rule:     p.rule,
	}
	memo[at] = d
	if p.rule > 0 {
		d.Pos = r.prog.rules[p.rule-1].src.pos
		for _, k := range p.kids {
			d.Children = append(d.Children, r.derivation(k, memo))
		}
	}
	return d
}

func quoteAll(t []string) []string {
	q := make([]string, len(t))
	for i, s := range t {
		q[i] = strconv.Quote(s)
	}
	return q
}

// String renders the tree one tuple per line, children indented, with the
// rule that derived each tuple.
func (d *Derivation) String() string {
	var b strings.Builder
	d.write(&b, 0)
	return b.String()
}

func (d *Derivation) write(b *strings.Builder, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(b, "%s(%s)", d.Relation, strings.Join(quoteAll(d.Tuple), ", "))
	if d.Rule == 0 {
		b.WriteString("  [input]\n")
	} else {
		fmt.Fprintf(b, "  [rule %d at %s]\n", d.Rule, d.Pos)
	}
	for _, c := range d.Children {
		c.write(b, depth+1)
	}
}
