// The static relational INTEGRITY model. A sibling of the policy layer one
// concern over: where policy carries the meaning of the access invariants,
// integrity carries the meaning of the STRUCTURAL invariants (cardinality,
// mandatory relationships, attribute uniqueness, singleton flags) that the
// Modelith linter only sees as well-formed prose.
//
// The rung is admissibility, not safety. Policy asks "is anything bad
// permitted?" and answers with UNSAT checks (a counterexample is a hole).
// Integrity asks "are the intended structures admissible, and do they scale?"
// and answers with SAT runs (no instance is the finding: the constraint set
// contradicts itself, or cannot be populated within the bound). A green
// integrity model proves the whole constraint set is jointly satisfiable and
// each constraint is non-vacuous; it goes red the moment an edit makes two
// constraints incompatible, which the linter cannot see.
//
// The annotation (design/formal/integrity.relational.yaml) names the entities
// to model, the domain relationships to enforce (multiplicity read from the
// domain model, the mandatory/optional decision stated here because Modelith
// cardinality cannot express it), the unique attributes, and the singleton
// flags. Every constraint binds a domain invariant id; a drifted binding fails
// generation instead of proving a stale twin, the same rule the policy layer
// and the data-refinement annotations follow.

package alloy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/version"
)

// IntegrityAnnotationName is the integrity annotation file under design/formal/.
const IntegrityAnnotationName = "integrity.relational.yaml"

// IntegrityOutputName is the generated model file under design/formal/.
const IntegrityOutputName = "Integrity.als"

// --- annotation model (parsed + reconciled) ---

// IntRel is one modeled relationship: a field on the `From` sig pointing at
// the `To` sig, with a multiplicity derived from the domain cardinality plus
// the annotation's mandatory decision.
type IntRel struct {
	From       string
	To         string
	Field      string // Alloy field name (defaults to lowercased To)
	Card       string // domain cardinality: n:1 | 1:1 | 1:n | n:n
	Mandatory  bool   // lower bound >= 1 on the From side
	Mult       string // resolved Alloy multiplicity: one | lone | some | set
	Invariants []string
}

// IntUnique is one attribute-uniqueness constraint: no two `Entity` records
// share `Attr`. DomainSize is the cardinality of the attribute's value domain
// when it is bounded (boolean = 2, enum = value count) and 0 when it is
// unbounded (string, integer, timestamp). A bounded domain caps how many
// records can hold distinct values, so uniqueness over a small domain that the
// design nonetheless needs to populate is a real defect the solver surfaces.
type IntUnique struct {
	Entity     string
	Attr       string
	DomainSize int
	Invariants []string
}

// IntSingleton is one exactly-one-flagged constraint: exactly one `Entity`
// record has the boolean `Flag` set.
type IntSingleton struct {
	Entity     string
	Flag       string
	Invariants []string
}

// Integrity is the fully reconciled integrity annotation, ready for emission.
type Integrity struct {
	Entities   []string
	Rels       []IntRel
	Uniques    []IntUnique
	Singletons []IntSingleton
	Residuals  [][2]string
	Scope      int

	DomainFile     string
	AnnotationFile string
}

// IntegrityStats summarizes a generation for the CLI line and the gate counts.
type IntegrityStats struct {
	Entities      int
	Relationships int
	Uniques       int
	Singletons    int
	Residuals     int
	Commands      []Command
	Carried       int
}

// --- domain-view helpers (added to the shared domain type) ---

// relationshipCard returns the declared cardinality of the relationship from
// entity `from` to entity `to` ("" when there is no such relationship).
func (d *domain) relationshipCard(from, to string) string {
	e, ok := d.entities.Get(from)
	if !ok {
		return ""
	}
	for _, r := range listOf(e.AsObject().Get2("relationships")) {
		ro := r.AsObject()
		if ro.GetString("entity") == to {
			return ro.GetString("cardinality")
		}
	}
	return ""
}

// --- annotation keys ---

var integrityRootKeys = map[string]bool{
	"entities": true, "relationships": true, "unique": true,
	"singleton": true, "residuals": true, "scope": true, "_comment": true,
}
var integrityRelKeys = map[string]bool{
	"from": true, "to": true, "field": true, "mandatory": true, "invariant": true,
}
var integrityUniqueKeys = map[string]bool{"entity": true, "attribute": true, "invariant": true}
var integritySingletonKeys = map[string]bool{"entity": true, "flag": true, "invariant": true}

// resolveMult maps a domain cardinality plus the mandatory flag to an Alloy
// field multiplicity on the From side.
func resolveMult(card string, mandatory bool) (string, bool) {
	switch card {
	case "n:1", "1:1":
		if mandatory {
			return "one", true
		}
		return "lone", true
	case "1:n", "n:n":
		if mandatory {
			return "some", true
		}
		return "set", true
	default:
		return "", false
	}
}

// inverseLone reports whether the left side of the domain cardinality is one.
// Alloy field multiplicities constrain only the source -> target direction;
// 1:1 and 1:n additionally require every target to be referenced by at most
// one source. The lower bound remains an annotation decision on the source
// side, so the inverse is lone rather than one.
func inverseLone(card string) bool { return card == "1:1" || card == "1:n" }

// LoadIntegrity parses and reconciles the integrity annotation against the
// domain model. Every disagreement dies (the refine_gen rule).
func LoadIntegrity(domainPath, annotationPath string) *Integrity {
	d := loadDomain(domainPath)
	data, err := os.ReadFile(annotationPath)
	if err != nil {
		die("%s: %v", filepath.Base(annotationPath), err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		die("%s: not a yaml mapping", filepath.Base(annotationPath))
	}
	root := v.AsObject()
	checkKeys(root, integrityRootKeys, filepath.Base(annotationPath))

	p := &Integrity{
		Scope:          6,
		DomainFile:     filepath.Base(domainPath),
		AnnotationFile: filepath.Base(annotationPath),
	}

	// entities: the explicit modeling surface. Every from/to and every
	// unique/singleton entity must be listed here, so the sig set is bounded
	// and nothing is modeled by accident.
	p.Entities = strList(root.Get2("entities"), "entities")
	if len(p.Entities) == 0 {
		die("entities must list at least one entity")
	}
	inModel := map[string]bool{}
	for _, e := range p.Entities {
		if inModel[e] {
			die("entities lists %s twice", ir.Repr(e))
		}
		inModel[e] = true
		if _, ok := d.entities.Get(e); !ok {
			die("entities names %s, which is not a Modelith entity", ir.Repr(e))
		}
		if e == "Bool" {
			die("entity name %s collides with the generated Bool signature; it cannot be modeled here", ir.Repr(e))
		}
	}
	requireModeled := func(name, where string) {
		if !inModel[name] {
			die("%s references %s, which is not in the entities list; add it to model it", where, ir.Repr(name))
		}
	}

	// invariant binding: every referenced id must exist in the domain model,
	// and no id may be claimed twice across the integrity constraints.
	claimed := map[string]string{}
	claim := func(ids []string, where string) []string {
		for _, id := range ids {
			if !d.allInvs[id] {
				die("%s references invariant %s, which the domain model does not declare", where, ir.Repr(id))
			}
			if prev, dup := claimed[id]; dup {
				die("invariant %s is claimed twice (%s and %s); every id maps to one integrity constraint", ir.Repr(id), prev, where)
			}
			claimed[id] = where
		}
		return ids
	}

	// relationships
	fieldSeen := map[string]bool{} // "Entity.field" collision guard
	for i, rv := range listOf(root.Get2("relationships")) {
		ro := rv.AsObject()
		if ro == nil {
			die("relationships[%d] is not a mapping", i)
		}
		checkKeys(ro, integrityRelKeys, fmt.Sprintf("relationships[%d]", i))
		rel := IntRel{From: ro.GetString("from"), To: ro.GetString("to")}
		where := fmt.Sprintf("relationships[%d] (%s -> %s)", i, rel.From, rel.To)
		if rel.From == "" || rel.To == "" {
			die("%s needs both from and to", where)
		}
		requireModeled(rel.From, where+" from")
		requireModeled(rel.To, where+" to")
		rel.Card = d.relationshipCard(rel.From, rel.To)
		if rel.Card == "" {
			die("%s: the domain model declares no relationship from %s to %s", where, rel.From, rel.To)
		}
		if mv := ro.Get2("mandatory"); mv != nil {
			bv, ok := mv.AsBool()
			if !ok {
				die("%s: mandatory must be true or false", where)
			}
			rel.Mandatory = bv
		}
		mult, ok := resolveMult(rel.Card, rel.Mandatory)
		if !ok {
			die("%s: unsupported cardinality %s (expected n:1, 1:1, 1:n, or n:n)", where, ir.Repr(rel.Card))
		}
		rel.Mult = mult
		rel.Field = ro.GetString("field")
		if rel.Field == "" {
			rel.Field = lowerFirst(rel.To)
		}
		fkey := rel.From + "." + rel.Field
		if fieldSeen[fkey] {
			die("%s: field %s is already used on %s; set a distinct field: name", where, ir.Repr(rel.Field), rel.From)
		}
		fieldSeen[fkey] = true
		rel.Invariants = claim(strList(ro.Get2("invariant"), where+" invariant"), where)
		p.Rels = append(p.Rels, rel)
	}

	// unique
	for i, uv := range listOf(root.Get2("unique")) {
		uo := uv.AsObject()
		if uo == nil {
			die("unique[%d] is not a mapping", i)
		}
		checkKeys(uo, integrityUniqueKeys, fmt.Sprintf("unique[%d]", i))
		u := IntUnique{Entity: uo.GetString("entity"), Attr: uo.GetString("attribute")}
		where := fmt.Sprintf("unique[%d] (%s.%s)", i, u.Entity, u.Attr)
		if u.Entity == "" || u.Attr == "" {
			die("%s needs both entity and attribute", where)
		}
		requireModeled(u.Entity, where)
		at := d.attrType(u.Entity, u.Attr)
		if at == "" {
			die("%s: %s is not an attribute of %s", where, ir.Repr(u.Attr), u.Entity)
		}
		switch {
		case at == "boolean":
			u.DomainSize = 2
		case len(d.enums[at]) > 0:
			u.DomainSize = len(d.enums[at])
		}
		fkey := u.Entity + "." + u.Attr
		if fieldSeen[fkey] {
			die("%s: field %s is already used on %s", where, ir.Repr(u.Attr), u.Entity)
		}
		fieldSeen[fkey] = true
		u.Invariants = claim(strList(uo.Get2("invariant"), where+" invariant"), where)
		p.Uniques = append(p.Uniques, u)
	}

	// singleton
	for i, sv := range listOf(root.Get2("singleton")) {
		so := sv.AsObject()
		if so == nil {
			die("singleton[%d] is not a mapping", i)
		}
		checkKeys(so, integritySingletonKeys, fmt.Sprintf("singleton[%d]", i))
		s := IntSingleton{Entity: so.GetString("entity"), Flag: so.GetString("flag")}
		where := fmt.Sprintf("singleton[%d] (%s.%s)", i, s.Entity, s.Flag)
		if s.Entity == "" || s.Flag == "" {
			die("%s needs both entity and flag", where)
		}
		requireModeled(s.Entity, where)
		ft := d.attrType(s.Entity, s.Flag)
		if ft == "" {
			die("%s: %s is not an attribute of %s", where, ir.Repr(s.Flag), s.Entity)
		}
		if ft != "boolean" {
			die("%s: singleton flag %s has type %s, not boolean; a singleton marks exactly one record via a boolean flag", where, ir.Repr(s.Flag), ir.Repr(ft))
		}
		fkey := s.Entity + "." + s.Flag
		if fieldSeen[fkey] {
			die("%s: field %s is already used on %s", where, ir.Repr(s.Flag), s.Entity)
		}
		fieldSeen[fkey] = true
		s.Invariants = claim(strList(so.Get2("invariant"), where+" invariant"), where)
		p.Singletons = append(p.Singletons, s)
	}

	if len(p.Rels)+len(p.Uniques)+len(p.Singletons) == 0 {
		die("annotation compiles no constraints; with nothing to model there is nothing to check")
	}

	// residuals: optional, for documenting integrity-shaped invariants the
	// algebra cannot carry. Same shape and drift rule as the policy layer.
	for i, rv := range listOf(root.Get2("residuals")) {
		ro := rv.AsObject()
		if ro == nil {
			die("residuals[%d] is not a mapping", i)
		}
		checkKeys(ro, residualKeys, fmt.Sprintf("residuals[%d]", i))
		id := ro.GetString("invariant")
		reason := ro.GetString("reason")
		if id == "" || reason == "" {
			die("residuals[%d] needs both an invariant id and a reason; an unexplained waiver is a hole", i)
		}
		if !d.allInvs[id] {
			die("residuals[%d] waives invariant %s, which the domain model does not declare", i, ir.Repr(id))
		}
		if prev, dup := claimed[id]; dup {
			die("invariant %s is claimed twice (%s and a residual)", ir.Repr(id), prev)
		}
		claimed[id] = "a residual"
		p.Residuals = append(p.Residuals, [2]string{id, reason})
	}

	// scope bound
	if sv := root.Get2("scope"); sv != nil {
		n, err := sv.AsNumber().Int64()
		if sv.Kind != ir.KindNumber || err != nil || n < 3 || n > 12 {
			die("scope must be an integer between 3 and 12 (the Populatable witness requires three records)")
		}
		p.Scope = int(n)
	}
	return p
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// GenerateIntegrity emits the integrity model. Deterministic: every ordering
// comes from the annotation (entity order, relationship/unique/singleton order).
func GenerateIntegrity(domainPath, annotationPath string) (als string, stats IntegrityStats, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
				return
			}
			panic(r)
		}
	}()
	p := LoadIntegrity(domainPath, annotationPath)
	als, stats = p.emit()
	return als, stats, nil
}

// valSig is the abstract value-domain signature name for a unique attribute.
func valSig(entity, attr string) string { return "Val_" + entity + "_" + upperFirst(attr) }

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (p *Integrity) emit() (string, IntegrityStats) {
	var b strings.Builder
	w := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, format+"\n", args...)
	}
	stats := IntegrityStats{
		Entities: len(p.Entities), Relationships: len(p.Rels),
		Uniques: len(p.Uniques), Singletons: len(p.Singletons), Residuals: len(p.Residuals),
	}
	carried := map[string]bool{}
	for _, r := range p.Rels {
		for _, id := range r.Invariants {
			carried[id] = true
		}
	}
	for _, u := range p.Uniques {
		for _, id := range u.Invariants {
			carried[id] = true
		}
	}
	for _, s := range p.Singletons {
		for _, id := range s.Invariants {
			carried[id] = true
		}
	}
	stats.Carried = len(carried)

	w("// Code generated from %s + %s by machinery alloy. DO NOT EDIT.", p.DomainFile, p.AnnotationFile)
	w("%s", version.AlloyStamp())
	w("//")
	w("// Static relational model of the STRUCTURAL invariants: which configurations")
	w("// of entities, relationships, unique keys, and singleton flags the constraint")
	w("// set admits. This rung checks ADMISSIBILITY, not safety: the commands are")
	w("// runs (a satisfying instance must exist), and no instance is the finding.")
	w("//")
	w("// ASSUMPTIONS (what this abstraction erases; results are conditional on them):")
	w("//   1. Bounded search: exhaustive only within scope %d (at most %d atoms per", p.Scope, p.Scope)
	w("//      signature). A run that finds no instance within the bound is a FAIL:")
	w("//      the constraint set contradicts itself, or cannot populate to scale.")
	w("//   2. Statics only: this rung checks admissible configurations, never how")
	w("//      the system moves between them; lifecycle properties are the TLC rung.")
	w("//   3. Only the named constraints are modeled: cardinality of the listed")
	w("//      relationships, uniqueness of the listed attributes, and the listed")
	w("//      singleton flags. Attribute values are abstract atoms (an injective")
	w("//      relation into an unconstrained domain), never their concrete types.")
	hasInverse := false
	for _, r := range p.Rels {
		if inverseLone(r.Card) {
			hasInverse = true
		}
	}
	if hasInverse {
		w("//   4. Inverse exclusivity of the declared 1:1 / 1:n edges is enforced as a")
		w("//      fact (Cardinality_*), an axiom of the model. No check command restates")
		w("//      it: a check identical to a fact is a tautology that can never fail.")
	}
	if len(p.Residuals) > 0 {
		w("//")
		w("// RESIDUALS (structural invariants the algebra does not carry; enforced elsewhere):")
		for _, r := range p.Residuals {
			w("//   - %s: %s", r[0], r[1])
		}
	}
	w("")
	w("module Integrity")

	// entity signatures with their fields (relationships, unique attrs, flags)
	for _, e := range p.Entities {
		var fields, comments []string
		for _, r := range p.Rels {
			if r.From == e {
				fields = append(fields, fmt.Sprintf("%s: %s %s", r.Field, r.Mult, r.To))
				comments = append(comments, fmt.Sprintf("%s -> %s (%s%s)", r.Field, r.To, r.Card, mandatoryNote(r.Mandatory)))
			}
		}
		for _, u := range p.Uniques {
			if u.Entity == e {
				fields = append(fields, fmt.Sprintf("%s: one %s", u.Attr, valSig(e, u.Attr)))
				comments = append(comments, "unique key")
			}
		}
		for _, s := range p.Singletons {
			if s.Entity == e {
				fields = append(fields, fmt.Sprintf("%s: one Bool", s.Flag))
				comments = append(comments, "singleton flag")
			}
		}
		w("")
		if len(fields) == 0 {
			w("sig %s {}", e)
			continue
		}
		w("sig %s {", e)
		for i, f := range fields {
			sep := ","
			if i == len(fields)-1 {
				sep = " "
			}
			w("  %s%s  // %s", f, sep, comments[i])
		}
		w("}")
	}

	// A field multiplicity describes only how many targets one source may
	// reference. Cardinalities with a left-side one also need the inverse bound
	// or a declared 1:1/1:n edge still admits two sources sharing one target.
	// The bound is a FACT (an axiom of the model); no check command restates
	// it: a check byte-identical to a fact is a tautology that can never fail
	// and would inflate the pass count without proving anything.
	for _, r := range p.Rels {
		if !inverseLone(r.Card) {
			continue
		}
		w("")
		w("// %s: the %s side of %s -> %s is exclusive (enforced as a fact)", r.Card, r.From, r.From, r.To)
		w("fact Cardinality_%s_%s {", r.From, upperFirst(r.Field))
		w("  all target: %s | lone target.~%s", r.To, r.Field)
		w("}")
	}

	// Bool enum, if any singleton needs it
	if len(p.Singletons) > 0 {
		w("")
		w("// boolean domain for singleton flags")
		w("abstract sig Bool {}")
		w("one sig True, False extends Bool {}")
	}

	// value domains for unique attributes: unbounded (abstract) for open types
	// like string, bounded to the exact cardinality for boolean and enum types
	// so uniqueness over a small domain genuinely caps the record count.
	if len(p.Uniques) > 0 {
		w("")
		w("// value domains: one atom per distinct attribute value (bounded types")
		w("// get exactly as many atoms as the type admits)")
		for _, u := range p.Uniques {
			sig := valSig(u.Entity, u.Attr)
			if u.DomainSize == 0 {
				w("sig %s {}", sig)
				continue
			}
			var atoms []string
			for i := 0; i < u.DomainSize; i++ {
				atoms = append(atoms, fmt.Sprintf("%s_%d", sig, i))
			}
			w("abstract sig %s {}", sig)
			w("one sig %s extends %s {} // bounded: %d value(s)", strings.Join(atoms, ", "), sig, u.DomainSize)
		}
	}

	// uniqueness facts
	for _, u := range p.Uniques {
		w("")
		w("// %s: no two %s records share %s", strings.Join(u.Invariants, ", "), u.Entity, u.Attr)
		w("fact Unique_%s_%s {", u.Entity, upperFirst(u.Attr))
		w("  all disj x, y: %s | x.%s != y.%s", u.Entity, u.Attr, u.Attr)
		w("}")
	}

	// singleton facts
	for _, s := range p.Singletons {
		w("")
		w("// %s: exactly one %s record has %s set", strings.Join(s.Invariants, ", "), s.Entity, s.Flag)
		w("fact Singleton_%s_%s {", s.Entity, upperFirst(s.Flag))
		w("  one x: %s | x.%s = True", s.Entity, s.Flag)
		w("}")
	}

	// --- meta-checks: admissibility runs ---
	w("")
	w("// --- Generated meta-checks: the admissibility suite, identical for every design ---")
	w("")
	w("// PASS = instance found: the whole constraint set is jointly satisfiable with")
	w("// every modeled entity inhabited. No instance = the constraints contradict each")
	w("// other (a structural over-specification the linter cannot see).")
	w("run SomeWorld {")
	var some []string
	for _, e := range p.Entities {
		some = append(some, "some "+e)
	}
	w("  %s", strings.Join(some, " and "))
	w("} for %d", p.Scope)
	stats.Commands = append(stats.Commands, Command{Kind: "run", Name: "SomeWorld"})

	// Populatable: every modeled entity can hold `target` records at once under
	// all constraints. Catches cardinality/uniqueness starvation that SomeWorld
	// (one atom each) misses: a mandatory chain or a too-small unique value
	// domain shows up here, never at a single atom. The target exceeds a
	// boolean domain (2) so uniqueness declared on a boolean or a small enum
	// -- a record count that cannot exist -- fails here.
	target := 3
	if p.Scope < target {
		target = p.Scope
	}
	w("")
	w("// PASS = instance found: every modeled entity can hold %d records", target)
	w("// simultaneously under all constraints. No instance = a cardinality or")
	w("// uniqueness constraint starves the model (it admits a token world but cannot")
	w("// scale), which is a structural defect masquerading as a valid design.")
	w("run Populatable {")
	for i, e := range p.Entities {
		sep := " and"
		if i == len(p.Entities)-1 {
			sep = ""
		}
		w("  #%s >= %d%s", e, target, sep)
	}
	w("} for %d", p.Scope)
	stats.Commands = append(stats.Commands, Command{Kind: "run", Name: "Populatable"})

	// per-unique non-vacuity witnesses
	for _, u := range p.Uniques {
		name := fmt.Sprintf("Distinct_%s_%s", u.Entity, upperFirst(u.Attr))
		w("")
		w("// PASS = instance found: two %s records with different %s coexist, so the", u.Entity, u.Attr)
		w("// unique key is a real constraint over a non-trivial value domain, not a")
		w("// fact that vacuously forces at most one record.")
		w("run %s { some disj x, y: %s | x.%s != y.%s } for %d", name, u.Entity, u.Attr, u.Attr, p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "run", Name: name})
	}

	return b.String(), stats
}

func mandatoryNote(mandatory bool) string {
	if mandatory {
		return ", mandatory"
	}
	return ", optional"
}

// CarriedIntegrityIDs extracts the invariant ids the integrity annotation
// claims to carry, WITHOUT validating them; the Gi gate owns validation.
// Gx-trace uses this to credit the integrity model as an enforcement artifact.
func CarriedIntegrityIDs(annotationPath string) map[string]bool {
	data, err := os.ReadFile(annotationPath)
	if err != nil {
		return nil
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		return nil
	}
	root := v.AsObject()
	ids := map[string]bool{}
	add := func(v *ir.Value) {
		if v == nil {
			return
		}
		items := []*ir.Value{v}
		if v.Kind == ir.KindArray {
			items = v.AsArray()
		}
		for _, it := range items {
			if it != nil && it.Kind == ir.KindString {
				ids[it.AsString()] = true
			}
		}
	}
	for _, key := range []string{"relationships", "unique", "singleton"} {
		for _, rv := range listOf(root.Get2(key)) {
			if ro := rv.AsObject(); ro != nil {
				add(ro.Get2("invariant"))
			}
		}
	}
	return ids
}
