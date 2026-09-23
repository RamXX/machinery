// Package gates implements deterministic design checks. This file owns the
// Gy-rules gate (docs/consistency-layer-proposal.md, section 3.3, Stage 3):
// the shipped methodology rules under rules/consistency, evaluated by the
// in-process internal/datalog engine over the design's projected facts.
//
// The facts are built in process by the projection v2 reader; nothing is
// written. Each rule file is loaded once, checked against the relation
// catalog, and run with derivation recording on. Its output relations are
// findings by name:
//
//	finding_<code>   an ERROR of the gate.
//	warn_<code>      a warning.
//
// The first attribute of an output relation names the kind of its subject id
// (unit, action, subject, type), and the gate resolves the id to a design
// location through the projection's sources, so a finding reads
// `machines/Order.matrix.md:12: row 'Order.persistOrder': <code>`.
package gates

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/RamXX/machinery/internal/checker"
	"github.com/RamXX/machinery/internal/datalog"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/rules"
)

// RulesGateTitle is the Gy-rules gate header.
const RulesGateTitle = "Gy-rules  consistency rules over projected facts"

// Output relation prefixes and the tier each selects.
const (
	findingPrefix = "finding_"
	warnPrefix    = "warn_"
)

// ruleSubjectSources maps the first attribute name of an output relation to
// the relations whose first column locates that subject, in the order they
// are tried. It is the whole of the gate's knowledge about subjects; a rule
// file declaring any other first attribute is rejected at load. A unit
// subject falls back to the declaration rows: a declaration on a matrix row
// that names no unit (a consumed-event row) has the matrix id as its subject,
// and those rows carry its location.
var ruleSubjectSources = map[string][]string{
	"unit":    {"unit", "unit_declares", "unit_derived"},
	"action":  {"action"},
	"subject": {"admission", "no_authorization", "action"},
	"type":    {"type_owner", "supersedes"},
}

// ruleLimits bounds every rule evaluation explicitly. A design projection is
// orders of magnitude below both; a hit is reported as an ERROR naming the
// rule file. Tests shrink them to exercise the limit path.
var ruleLimits = struct{ maxTuples, maxIterations int }{maxTuples: 2_000_000, maxIterations: 100_000}

type ruleOutput struct {
	relation string
	code     string
	warn     bool
	attrs    []string
}

type ruleFile struct {
	name    string // file name inside rules/consistency
	prog    *datalog.Program
	outputs []ruleOutput
}

type ruleSet struct {
	files []ruleFile
}

// loadRuleSet parses every *.dl file directly inside dir of fsys and checks
// it against the gate's contract: every .input relation is a catalog relation
// of the same arity; every .output relation is named finding_<code> or
// warn_<code>, and every relation so named is an .output; the first attribute
// of an output names a known subject kind; no two files emit one relation.
func loadRuleSet(fsys fs.FS, dir string) (*ruleSet, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	catalog := map[string]int{}
	for _, spec := range checker.RelationCatalog() {
		catalog[spec.Name] = len(spec.Columns)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".dl") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("%s holds no .dl rule file", dir)
	}
	emitted := map[string]string{}
	set := &ruleSet{}
	var errs []error
	for _, name := range names {
		src, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, err
		}
		prog, err := datalog.Parse(string(src), name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		rf := ruleFile{name: name, prog: prog}
		for _, in := range prog.InputRelations() {
			d, _ := prog.Decl(in)
			arity, ok := catalog[in]
			if !ok {
				errs = append(errs, fmt.Errorf("%s: .input %s is not a projected relation", name, in))
				continue
			}
			if arity != len(d.Attrs) {
				errs = append(errs, fmt.Errorf("%s: .input %s declares %d attributes, the projection has %d", name, in, len(d.Attrs), arity))
			}
		}
		outputs := map[string]bool{}
		for _, out := range prog.OutputRelations() {
			outputs[out] = true
			d, _ := prog.Decl(out)
			o := ruleOutput{relation: out}
			switch {
			case strings.HasPrefix(out, findingPrefix):
				o.code = strings.TrimPrefix(out, findingPrefix)
			case strings.HasPrefix(out, warnPrefix):
				o.code, o.warn = strings.TrimPrefix(out, warnPrefix), true
			}
			if o.code == "" {
				errs = append(errs, fmt.Errorf("%s: .output %s is not named finding_<code> or warn_<code>", name, out))
				continue
			}
			if prior, dup := emitted[out]; dup {
				errs = append(errs, fmt.Errorf("%s: relation %s is already emitted by %s; one relation, one rule file", name, out, prior))
				continue
			}
			emitted[out] = name
			for _, a := range d.Attrs {
				if a.Type != datalog.Symbol {
					errs = append(errs, fmt.Errorf("%s: %s attribute %s must be a symbol", name, out, a.Name))
				}
				o.attrs = append(o.attrs, a.Name)
			}
			if _, ok := ruleSubjectSources[o.attrs[0]]; !ok {
				errs = append(errs, fmt.Errorf("%s: %s's first attribute %q names no subject kind; use one of %s", name, out, o.attrs[0], strings.Join(subjectKinds(), ", ")))
				continue
			}
			rf.outputs = append(rf.outputs, o)
		}
		for _, rel := range prog.Relations() {
			named := strings.HasPrefix(rel, findingPrefix) || strings.HasPrefix(rel, warnPrefix)
			if named && !outputs[rel] {
				errs = append(errs, fmt.Errorf("%s: %s is named like a finding but is not an .output", name, rel))
			}
		}
		set.files = append(set.files, rf)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return set, nil
}

func subjectKinds() []string {
	var kinds []string
	for k := range ruleSubjectSources {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

// shippedRules is the embedded rule set, loaded once. A load failure is a
// build defect the rules tests catch; at run time it is an ERROR of the gate.
var shippedRules = sync.OnceValues(func() (*ruleSet, error) {
	return loadRuleSet(rules.Consistency, rules.ConsistencyDir)
})

// ruleFinding is one output tuple with its rule file and derivation.
type ruleFinding struct {
	file   string
	output ruleOutput
	tuple  []string
	tree   *datalog.Derivation
}

// ruleInputs renders every catalog relation's rows as evaluator inputs. A
// relation of an absent layer is supplied empty: a rule that reads a layer
// the design does not have sees no facts, never a missing-input error.
func ruleInputs(facts *checker.DesignFacts) datalog.Inputs {
	in := datalog.Inputs{}
	for _, spec := range checker.RelationCatalog() {
		rows := facts.Rows(spec.Name)
		tuples := make([][]string, 0, len(rows))
		for _, row := range rows {
			tuples = append(tuples, row.Values)
		}
		in[spec.Name] = tuples
	}
	return in
}

// evaluateRules runs every rule file over the facts. A file that fails (a
// limit hit, a run error) is reported by name and the other files still run.
func evaluateRules(set *ruleSet, facts *checker.DesignFacts) ([]ruleFinding, []string) {
	in := ruleInputs(facts)
	var findings []ruleFinding
	var errs []string
	for _, rf := range set.files {
		res, err := rf.prog.Run(in, datalog.Options{Explain: true, MaxTuples: ruleLimits.maxTuples, MaxIterations: ruleLimits.maxIterations})
		if err != nil {
			if errors.Is(err, datalog.ErrLimit) {
				errs = append(errs, "rules/consistency/"+rf.name+": evaluation stopped at a resource limit (MaxTuples "+fmt.Sprint(ruleLimits.maxTuples)+", MaxIterations "+fmt.Sprint(ruleLimits.maxIterations)+"): "+err.Error())
			} else {
				errs = append(errs, "rules/consistency/"+rf.name+": "+err.Error())
			}
			continue
		}
		for _, o := range rf.outputs {
			for _, tuple := range res.Output(o.relation) {
				tree, err := res.Explain(o.relation, tuple)
				if err != nil {
					errs = append(errs, "rules/consistency/"+rf.name+": "+err.Error())
					continue
				}
				findings = append(findings, ruleFinding{file: rf.name, output: o, tuple: tuple, tree: tree})
			}
		}
	}
	return findings, errs
}

// factSources indexes the facts for the gate's two lookups: a subject's
// location by (relation, first column), and an input tuple's location by
// (relation, tuple).
type factSources struct {
	byFirst map[string]checker.Source
	byTuple map[string]checker.Source
}

func indexFactSources(facts *checker.DesignFacts) factSources {
	idx := factSources{byFirst: map[string]checker.Source{}, byTuple: map[string]checker.Source{}}
	for _, spec := range checker.RelationCatalog() {
		for _, row := range facts.Rows(spec.Name) {
			if len(row.Values) == 0 {
				continue
			}
			first := spec.Name + "\x00" + row.Values[0]
			if _, ok := idx.byFirst[first]; !ok {
				idx.byFirst[first] = row.Source
			}
			idx.byTuple[spec.Name+"\x00"+strings.Join(row.Values, "\x00")] = row.Source
		}
	}
	return idx
}

// locate renders a subject as `path:line: row 'X'`, or `X (no source)` when
// no projected row carries it.
func (s factSources) locate(kind, subject string) string {
	for _, rel := range ruleSubjectSources[kind] {
		if src, ok := s.byFirst[rel+"\x00"+subject]; ok {
			return src.String() + ": row " + ir.Repr(subject)
		}
	}
	return subject + " (no source)"
}

// message renders one finding: its location, its code, and every column
// after the subject as `name 'value'`.
func (s factSources) message(f ruleFinding) string {
	msg := s.locate(f.output.attrs[0], f.tuple[0]) + ": " + f.output.code
	if len(f.tuple) > 1 {
		var parts []string
		for i := 1; i < len(f.tuple); i++ {
			parts = append(parts, f.output.attrs[i]+" "+ir.Repr(f.tuple[i]))
		}
		msg += " (" + strings.Join(parts, ", ") + ")"
	}
	return msg
}

// explainLines renders a derivation tree, one tuple per line, children
// indented two spaces: a derived tuple names its rule file and 1-based rule
// index, an input fact names its design source. Negated and aggregated body
// literals hold by absence and have no witness, so they are not printed.
func (s factSources) explainLines(file string, d *datalog.Derivation) []string {
	var out []string
	var walk func(n *datalog.Derivation, depth int)
	walk = func(n *datalog.Derivation, depth int) {
		quoted := make([]string, len(n.Tuple))
		for i, v := range n.Tuple {
			quoted[i] = fmt.Sprintf("%q", v)
		}
		line := strings.Repeat("  ", depth) + n.Relation + "(" + strings.Join(quoted, ", ") + ")"
		if n.Rule > 0 {
			line += fmt.Sprintf("  [%s rule %d]", file, n.Rule)
		} else if src, ok := s.byTuple[n.Relation+"\x00"+strings.Join(n.Tuple, "\x00")]; ok {
			line += "  [fact " + src.String() + "]"
		} else {
			line += "  [fact]"
		}
		out = append(out, line)
		for _, c := range n.Children {
			walk(c, depth+1)
		}
	}
	walk(d, 0)
	return out
}

// RulesActive reports whether Gy-rules applies to a design: it has a
// machines/ directory or an AUTHORIZATION.md, the two sources its rules read
// that no other artifact supplies.
func RulesActive(design string) bool {
	has, err := probeRulesActive(design)
	return has || err != nil
}

func probeRulesActive(design string) (bool, error) {
	machines, err := probeRealDir(design, "machines")
	if err != nil || machines {
		return machines, err
	}
	return probeRegularFile(design, "AUTHORIZATION.md")
}

// rulesNotActivated is the gate an explicit --gate gy prints for a design
// the rules do not apply to.
func rulesNotActivated() *Gate {
	g := NewGate(RulesGateTitle)
	g.Notes = append(g.Notes, "not activated: the design has no machines/ and no AUTHORIZATION.md, the sources the consistency rules read")
	return g
}

// CheckRules runs Gy-rules on a design. With explain, every finding carries
// its derivation tree, printed under it by Emit.
func CheckRules(design string, explain bool) *Gate {
	g := NewGate(RulesGateTitle)
	set, err := shippedRules()
	if err != nil {
		g.Errs = append(g.Errs, "the shipped consistency rules do not load: "+err.Error())
		return g
	}
	facts, err := LoadDesignFacts(design)
	if err != nil {
		g.Errs = append(g.Errs, "cannot project the design's facts for the consistency rules: "+err.Error())
		return g
	}
	checkRulesOver(g, set, facts, explain)
	return g
}

func checkRulesOver(g *Gate, set *ruleSet, facts *checker.DesignFacts, explain bool) {
	findings, errs := evaluateRules(set, facts)
	g.Errs = append(g.Errs, errs...)
	sources := indexFactSources(facts)
	for _, f := range findings {
		msg := sources.message(f)
		if f.output.warn {
			g.Warns = append(g.Warns, msg)
		} else {
			g.Errs = append(g.Errs, msg)
		}
		if explain {
			g.addExplain(msg, sources.explainLines(f.file, f.tree))
		}
	}
	g.Count("rule files evaluated", len(set.files))
	inputs := 0
	for _, rows := range ruleInputs(facts) {
		inputs += len(rows)
	}
	g.Count("facts", inputs)
}
