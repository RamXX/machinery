// The static relational ISOLATION model. The third relational algebra, and the
// one the other two cannot express: multi-tenant isolation across the reference
// graph. The policy layer checks role x verb x direct ownership; it never looks
// at what a record REFERENCES. But a record the actor may read can carry a link
// to a record in another tenant, and following that link is a leak the access
// rules never see. This layer carries the meaning of the cross-entity
// tenant-consistency invariants: a record and everything it references belong to
// one tenant, so no link crosses a tenant boundary.
//
// tenant(record) = owner's tenant. A reference From.field -> To is
// tenant-consistent when the two owners share a tenant. The annotation names
// which references must be tenant-consistent (binding a domain invariant each),
// and the layer emits both a bounded relational model (Isolation.als) whose
// checks verify the isolation is coherent -- most sharply, that two records in
// different tenants can never share a referent -- and an isolation oracle
// (Isolation.oracle.md), the tenant-scoping decision table the implementation's
// link-authorization test consumes, exactly as the policy oracle holds the
// access code.

package alloy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// IsolationAnnotationName is the isolation annotation file under design/formal/.
const IsolationAnnotationName = "isolation.relational.yaml"

// IsolationOutputName is the generated model file under design/formal/.
const IsolationOutputName = "Isolation.als"

// IsolationOracleName is the generated tenant-scoping oracle under design/formal/.
const IsolationOracleName = "Isolation.oracle.md"

// --- annotation model (parsed + reconciled) ---

// IsoRef is one cross-entity reference the isolation layer holds to
// tenant-consistency: a field on the From sig pointing at the To sig.
type IsoRef struct {
	From        string
	To          string
	Field       string
	Card        string // domain cardinality From -> To
	Mult        string // lone | set (a reference may be absent)
	ManyToOne   bool   // many From can point at one To: SharedReferent applies
	InverseLone bool   // 1:1 / 1:n: every To is referenced by at most one From
	Invariants  []string
}

// Isolation is the fully reconciled isolation annotation, ready for emission.
type Isolation struct {
	TenantEntity  string
	SubjectEntity string
	TenantAttr    string // the subject attribute/relationship carrying tenant membership
	Membership    string // lone | one
	Records       []string
	Refs          []IsoRef
	Residuals     [][2]string
	Scope         int

	DomainFile     string
	AnnotationFile string
}

// IsolationStats summarizes a generation for the CLI line and the gate counts.
type IsolationStats struct {
	Records    int
	References int
	Residuals  int
	Commands   []Command
	Carried    int
	OracleRows int
}

var isolationRootKeys = map[string]bool{
	"tenant": true, "subject": true, "references": true,
	"residuals": true, "scope": true, "_comment": true,
}
var isolationTenantKeys = map[string]bool{"entity": true}
var isolationSubjectKeys = map[string]bool{"entity": true, "tenant_attr": true, "membership": true}
var isolationRefKeys = map[string]bool{"from": true, "to": true, "field": true, "invariant": true}

// LoadIsolation parses and reconciles the isolation annotation against the
// domain model. Every disagreement dies (the refine_gen rule).
func LoadIsolation(domainPath, annotationPath string) *Isolation {
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
	checkKeys(root, isolationRootKeys, filepath.Base(annotationPath))

	p := &Isolation{
		Scope:          6,
		DomainFile:     filepath.Base(domainPath),
		AnnotationFile: filepath.Base(annotationPath),
	}

	// tenant
	tenant := root.GetObject("tenant")
	if tenant.Len() == 0 {
		die("annotation declares no tenant")
	}
	checkKeys(tenant, isolationTenantKeys, "tenant")
	p.TenantEntity = tenant.GetString("entity")
	if p.TenantEntity == "" {
		die("tenant.entity is required")
	}
	if _, ok := d.entities.Get(p.TenantEntity); !ok {
		die("tenant.entity %s is not a Modelith entity", ir.Repr(p.TenantEntity))
	}

	// subject
	subj := root.GetObject("subject")
	if subj.Len() == 0 {
		die("annotation declares no subject")
	}
	checkKeys(subj, isolationSubjectKeys, "subject")
	p.SubjectEntity = subj.GetString("entity")
	if p.SubjectEntity == "" {
		die("subject.entity is required")
	}
	if _, ok := d.entities.Get(p.SubjectEntity); !ok {
		die("subject.entity %s is not a Modelith entity", ir.Repr(p.SubjectEntity))
	}
	if !d.hasRelationship(p.TenantEntity, p.SubjectEntity, "") && !d.hasRelationship(p.SubjectEntity, p.TenantEntity, "") {
		die("the domain model declares no relationship between %s and %s; tenant membership has nothing to bind to", p.TenantEntity, p.SubjectEntity)
	}
	p.TenantAttr = subj.GetString("tenant_attr")
	if p.TenantAttr == "" {
		die("subject.tenant_attr is required (the subject field that carries tenant membership)")
	}
	p.Membership = subj.GetString("membership")
	if p.Membership != "lone" && p.Membership != "one" {
		die("subject.membership must be 'lone' or 'one' (Modelith cardinality cannot express which; the annotation must decide)")
	}

	// invariant binding
	claimed := map[string]string{}
	claim := func(ids []string, where string) []string {
		for _, id := range ids {
			if !d.allInvs[id] {
				die("%s references invariant %s, which the domain model does not declare", where, ir.Repr(id))
			}
			if prev, dup := claimed[id]; dup {
				die("invariant %s is claimed twice (%s and %s); every id maps to one isolation reference", ir.Repr(id), prev, where)
			}
			claimed[id] = where
		}
		return ids
	}

	// references
	refs := listOf(root.Get2("references"))
	if len(refs) == 0 {
		die("annotation declares no references; with nothing to hold to a tenant there is nothing to check")
	}
	recordSet := map[string]bool{}
	addRecord := func(e string) {
		if !recordSet[e] {
			recordSet[e] = true
			p.Records = append(p.Records, e)
		}
	}
	requireOwned := func(e, where string) {
		if _, ok := d.entities.Get(e); !ok {
			die("%s names %s, which is not a Modelith entity", where, ir.Repr(e))
		}
		if e == p.SubjectEntity || e == p.TenantEntity {
			die("%s names %s, which is the subject or tenant entity, not a tenant-scoped record", where, ir.Repr(e))
		}
		if !d.hasRelationship(e, p.SubjectEntity, "n:1") {
			die("%s: record %s declares no n:1 ownership relationship to %s; its tenant is undefined", where, ir.Repr(e), p.SubjectEntity)
		}
	}
	fieldSeen := map[string]bool{}
	for i, rv := range refs {
		ro := rv.AsObject()
		if ro == nil {
			die("references[%d] is not a mapping", i)
		}
		checkKeys(ro, isolationRefKeys, fmt.Sprintf("references[%d]", i))
		ref := IsoRef{From: ro.GetString("from"), To: ro.GetString("to")}
		where := fmt.Sprintf("references[%d] (%s -> %s)", i, ref.From, ref.To)
		if ref.From == "" || ref.To == "" {
			die("%s needs both from and to", where)
		}
		requireOwned(ref.From, where+" from")
		requireOwned(ref.To, where+" to")
		ref.Card = d.relationshipCard(ref.From, ref.To)
		if ref.Card == "" {
			die("%s: the domain model declares no relationship from %s to %s to hold to a tenant", where, ref.From, ref.To)
		}
		switch ref.Card {
		case "n:1", "1:1":
			ref.Mult = "lone"
			ref.ManyToOne = ref.Card == "n:1"
			ref.InverseLone = ref.Card == "1:1"
		case "1:n", "n:n":
			ref.Mult = "set"
			ref.ManyToOne = ref.Card == "n:n"
			ref.InverseLone = ref.Card == "1:n"
		default:
			die("%s: unsupported cardinality %s", where, ir.Repr(ref.Card))
		}
		ref.Field = ro.GetString("field")
		if ref.Field == "" {
			ref.Field = lowerFirst(ref.To)
		}
		fkey := ref.From + "." + ref.Field
		if fieldSeen[fkey] {
			die("%s: field %s is already used on %s; set a distinct field: name", where, ir.Repr(ref.Field), ref.From)
		}
		fieldSeen[fkey] = true
		ref.Invariants = claim(strList(ro.Get2("invariant"), where+" invariant"), where)
		if len(ref.Invariants) == 0 {
			die("%s declares no invariant id; every reference carries the tenant-consistency invariant it holds", where)
		}
		addRecord(ref.From)
		addRecord(ref.To)
		p.Refs = append(p.Refs, ref)
	}

	// sig-name collisions with the generated skeleton
	reserved := map[string]bool{}
	for _, n := range append([]string{p.SubjectEntity, p.TenantEntity}, p.Records...) {
		if reserved[n] {
			die("entity %s collides with a reserved signature name", ir.Repr(n))
		}
	}

	// residuals
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
		if sv.Kind != ir.KindNumber || err != nil || n < 2 || n > 12 {
			die("scope must be an integer between 2 and 12")
		}
		p.Scope = int(n)
	}
	return p
}

// GenerateIsolation emits both the Alloy model and the tenant-scoping oracle
// from one reconciliation, so they can never disagree.
func GenerateIsolation(domainPath, annotationPath string) (als, oracle string, stats IsolationStats, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
				return
			}
			panic(r)
		}
	}()
	p := LoadIsolation(domainPath, annotationPath)
	als, stats = p.emit()
	oracle, stats.OracleRows = p.generateOracle()
	return als, oracle, stats, nil
}

func (p *Isolation) emit() (string, IsolationStats) {
	var b strings.Builder
	w := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, format+"\n", args...)
	}
	stats := IsolationStats{Records: len(p.Records), References: len(p.Refs), Residuals: len(p.Residuals)}
	carried := map[string]bool{}
	for _, r := range p.Refs {
		for _, id := range r.Invariants {
			carried[id] = true
		}
	}
	stats.Carried = len(carried)

	w("// Code generated from %s + %s by machinery alloy. DO NOT EDIT.", p.DomainFile, p.AnnotationFile)
	w("//")
	w("// Static relational model of multi-tenant ISOLATION: the tenant of a record")
	w("// is its owner's tenant, and every reference the annotation names must stay")
	w("// inside one tenant. The policy layer checks who may touch a record; this")
	w("// layer checks that the records a record REFERENCES cannot belong to another")
	w("// tenant, so following a link never crosses a tenant boundary.")
	w("//")
	w("// ASSUMPTIONS (what this abstraction erases; results are conditional on them):")
	w("//   1. Bounded search: exhaustive only within scope %d (at most %d atoms per", p.Scope, p.Scope)
	w("//      signature). PASS on a check means no counterexample within the bound.")
	w("//   2. Statics only: this rung checks admissible configurations, never how")
	w("//      the system moves between them; lifecycle properties are the TLC rung.")
	w("//   3. tenant(record) = owner's tenant; a teamless owner has no tenant, and a")
	w("//      reference from or to a teamless owner is never same-tenant. Only the")
	w("//      named references are held; every other relationship is unmodeled.")
	if len(p.Residuals) > 0 {
		w("//")
		w("// RESIDUALS (tenant invariants the algebra does not carry; enforced elsewhere):")
		for _, r := range p.Residuals {
			w("//   - %s: %s", r[0], r[1])
		}
	}
	w("")
	w("module Isolation")
	w("")
	w("// the tenant")
	w("sig %s {}", p.TenantEntity)
	w("")
	w("// subjects hold a tenant (membership %s)", p.Membership)
	w("sig %s {", p.SubjectEntity)
	w("  %s: %s %s", p.TenantAttr, p.Membership, p.TenantEntity)
	w("}")

	// record signatures: owner plus any reference fields
	for _, e := range p.Records {
		var fields, comments []string
		fields = append(fields, "owner: one "+p.SubjectEntity)
		comments = append(comments, "tenant = owner's "+p.TenantAttr)
		for _, r := range p.Refs {
			if r.From == e {
				fields = append(fields, fmt.Sprintf("%s: %s %s", r.Field, r.Mult, r.To))
				comments = append(comments, fmt.Sprintf("reference -> %s (%s)", r.To, r.Card))
			}
		}
		w("")
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

	w("")
	w("// two subjects share a tenant: both hold one and it is the same. A teamless")
	w("// subject shares a tenant with nobody, so a link touching one is never")
	w("// same-tenant.")
	w("pred sameTenant[a, b: %s] { some a.%s and a.%s = b.%s }", p.SubjectEntity, p.TenantAttr, p.TenantAttr, p.TenantAttr)

	// isolation facts: each named reference stays in-tenant
	for _, r := range p.Refs {
		w("")
		w("// %s: %s %s and the %s it references are owned in the same tenant", strings.Join(r.Invariants, ", "), article(r.From), r.From, r.To)
		w("fact Isolation_%s_%s_%s {", r.From, r.To, upperFirst(r.Field))
		w("  all x: %s | some x.%s implies sameTenant[x.owner, x.%s.owner]", r.From, r.Field, r.Field)
		w("}")
		if r.InverseLone {
			w("fact Cardinality_%s_%s_%s {", r.From, r.To, upperFirst(r.Field))
			w("  all target: %s | lone target.~%s", r.To, r.Field)
			w("}")
		}
	}

	// --- meta-checks ---
	w("")
	w("// --- Generated meta-checks: the standard suite, identical for every design ---")
	w("")
	w("// PASS = instance found: a genuinely multi-tenant world (two inhabited")
	w("// tenants) with at least one cross-entity link still exists. No instance =")
	w("// the isolation facts collapse tenancy (over-isolation: links force one")
	w("// tenant), which is as wrong as leaking across tenants.")
	w("run SomeWorld {")
	w("  (some disj ta, tb: %s | (some u: %s | u.%s = ta) and (some u: %s | u.%s = tb))",
		p.TenantEntity, p.SubjectEntity, p.TenantAttr, p.SubjectEntity, p.TenantAttr)
	var links []string
	for _, r := range p.Refs {
		links = append(links, fmt.Sprintf("(some x: %s | some x.%s)", r.From, r.Field))
	}
	w("  and (%s)", strings.Join(links, " or "))
	w("} for %d", p.Scope)
	stats.Commands = append(stats.Commands, Command{Kind: "run", Name: "SomeWorld"})

	// SharedReferent: two records of different tenants cannot reference the same
	// target. A non-trivial consequence of the single-hop facts (transitivity
	// of sameTenant through the shared referent); the sharp isolation guarantee.
	for _, r := range p.Refs {
		if !r.ManyToOne {
			continue
		}
		name := fmt.Sprintf("SharedReferent_%s_%s_%s", r.From, r.To, upperFirst(r.Field))
		w("")
		w("// PASS = no counterexample: two %s records that reference the same %s are", r.From, r.To)
		w("// owned in the same tenant. A counterexample would be a %s referenced from", r.To)
		w("// two tenants at once -- a shared referent bridging the boundary.")
		w("check %s {", name)
		w("  all x, y: %s | (some x.%s and x.%s = y.%s) implies sameTenant[x.owner, y.owner]", r.From, r.Field, r.Field, r.Field)
		w("} for %d", p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "check", Name: name})
	}

	// per-reference non-vacuity: a same-tenant link is actually constructible
	for _, r := range p.Refs {
		name := fmt.Sprintf("Possible_%s_%s_%s", r.From, r.To, upperFirst(r.Field))
		w("")
		w("// PASS = instance found: %s %s referencing %s %s exists, so the isolation", article(r.From), r.From, article(r.To), r.To)
		w("// fact is not vacuously satisfied by forbidding the link entirely.")
		w("run %s { some x: %s | some x.%s } for %d", name, r.From, r.Field, p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "run", Name: name})
	}

	return b.String(), stats
}

// --- oracle ---

// isoCase is one tenant-relationship case between the source owner and the
// target owner of a reference. The sameTenant predicate decides the link from
// exactly these, so the cases are the complete semantics boundary.
type isoCase struct {
	key         string
	expectation string
}

var isoCases = []isoCase{
	{"same-tenant", "allow"},    // both owners hold the same tenant
	{"cross-tenant", "deny"},    // both hold a tenant, different ones
	{"source-teamless", "deny"}, // the source owner holds no tenant
	{"target-teamless", "deny"}, // the target owner holds no tenant
}

// article returns the indefinite article for word, by its first letter.
func article(word string) string {
	if word == "" {
		return "a"
	}
	switch word[0] {
	case 'A', 'E', 'I', 'O', 'U', 'a', 'e', 'i', 'o', 'u':
		return "an"
	default:
		return "a"
	}
}

func stableIsoID(key string) string {
	h := sha256.Sum256([]byte("TENANT|" + key))
	return hex.EncodeToString(h[:])
}

// generateOracle renders the tenant-scoping decision table for the impl's
// link-authorization test: one row per (reference, tenant case).
func (p *Isolation) generateOracle() (string, int) {
	type row struct {
		ref  IsoRef
		c    isoCase
		key  string
		hash string
	}
	var rows []row
	for _, r := range p.Refs {
		for _, c := range isoCases {
			key := fmt.Sprintf("%s.%s->%s|%s", r.From, r.Field, r.To, c.key)
			rows = append(rows, row{ref: r, c: c, key: key, hash: stableIsoID(key)})
		}
	}
	byPrefix := map[string]map[string]bool{}
	for _, rr := range rows {
		pfx := rr.hash[:6]
		if byPrefix[pfx] == nil {
			byPrefix[pfx] = map[string]bool{}
		}
		byPrefix[pfx][rr.hash] = true
	}
	stable := func(h string) string {
		n := 6
		group := byPrefix[h[:6]]
		for n < len(h) {
			clash := false
			for other := range group {
				if other != h && other[:n] == h[:n] {
					clash = true
					break
				}
			}
			if !clash {
				break
			}
			n += 2
		}
		return "TENANT-" + h[:n]
	}

	var b strings.Builder
	w := func(format string, args ...interface{}) { fmt.Fprintf(&b, format+"\n", args...) }
	w("# Generated tenant-scoping oracle: isolation")
	w("")
	w("Generated from `%s` + `%s` by `machinery alloy`. DO NOT EDIT BY HAND.", p.DomainFile, p.AnnotationFile)
	w("Single source of truth for the link-authorization test: one row is one decision")
	w("case for the pure tenant-scoping function that decides whether a reference may")
	w("be established. Key tests on the STABLE id, not the row number; row numbers")
	w("renumber when the design changes, stable ids do not.")
	w("")
	w("A link From -> To is allowed exactly when the source owner and the target owner")
	w("share a tenant. tenant(record) = owner's %s; a teamless owner has no tenant, so", p.TenantAttr)
	w("a link touching one is refused. This table is the COMPLETE semantics: the pure")
	w("function decides every case from whether the two owners share a tenant.")
	w("")
	w("## Case vocabulary")
	w("")
	w("Tenant cases (relationship between the source owner and the target owner):")
	w("")
	w("- `same-tenant`: both owners hold a tenant and it is the same one.")
	w("- `cross-tenant`: both owners hold a tenant and they differ.")
	w("- `source-teamless`: the source record's owner holds no tenant.")
	w("- `target-teamless`: the target record's owner holds no tenant.")
	w("")
	w("## Decisions")
	w("")
	w("| test id | stable id | reference | tenant case | expectation | invariants |")
	w("|---|---|---|---|---|---|")
	for i, rr := range rows {
		w("| O-TENANT-%02d | %s | %s.%s -> %s | %s | %s | %s |",
			i+1, stable(rr.hash), rr.ref.From, rr.ref.Field, rr.ref.To, rr.c.key, rr.c.expectation, strings.Join(rr.ref.Invariants, ", "))
	}
	return b.String(), len(rows)
}

// CarriedIsolationIDs extracts the invariant ids the isolation annotation
// claims to carry, WITHOUT validating them; the Gn gate owns validation.
// Gx-trace uses this to credit the isolation model as an enforcement artifact.
func CarriedIsolationIDs(annotationPath string) map[string]bool {
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
	for _, rv := range listOf(root.Get2("references")) {
		if ro := rv.AsObject(); ro != nil {
			iv := ro.Get2("invariant")
			if iv == nil {
				continue
			}
			items := []*ir.Value{iv}
			if iv.Kind == ir.KindArray {
				items = iv.AsArray()
			}
			for _, it := range items {
				if it != nil && it.Kind == ir.KindString {
					ids[it.AsString()] = true
				}
			}
		}
	}
	return ids
}
