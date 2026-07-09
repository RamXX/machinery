// Package alloy generates opted-in static relational proofs and decision
// oracles from the Modelith domain model plus policy, integrity, and isolation
// annotations, after reconciling each annotation against the domain model.
//
// The annotation (design/formal/policy.relational.yaml) is written in a small
// closed algebra: a subject entity with a role enum, optional team scoping,
// resource entities under single-owner ownership, and per-verb scope rules
// (all | own | team | none). Everything the algebra cannot express is a named
// residual; every top-level invariant must be one or the other, so nothing
// falls between the rungs silently.
package alloy

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RamXX/machinery/internal/ir"
)

// AnnotationName is the policy annotation file name under design/formal/.
// Deliberately NOT *.semantics.yaml: verify-formal feeds every
// formal/*.semantics.yaml to refine_gen, and this file is not a machine
// annotation.
const AnnotationName = "policy.relational.yaml"

// OutputName is the generated model file name under design/formal/.
const OutputName = "Policy.als"

// OracleName is the generated authorization-oracle file name under
// design/formal/.
const OracleName = "Policy.oracle.md"

// ExitError carries a hard-error message (mirrors tla/refine).
type ExitError struct{ Msg string }

func (e *ExitError) Error() string { return e.Msg }

func die(format string, args ...interface{}) {
	panic(&ExitError{Msg: "alloy_gen: " + fmt.Sprintf(format, args...)})
}

// Verbs is the closed v1 verb vocabulary for grants and scope rules.
// Reassign is not in it: changing an owner has an actor scope AND a target
// constraint, so it is its own rule form.
var Verbs = []string{"create", "read", "update", "delete"}

// scopedVerbs are the verbs a scope rule may cover (create has no object).
var scopedVerbs = map[string]bool{"read": true, "update": true, "delete": true}

var verbSet = func() map[string]bool {
	m := map[string]bool{}
	for _, v := range Verbs {
		m[v] = true
	}
	return m
}()

// --- annotation model (parsed + reconciled) ---

// ScopeExpr is a parsed scope expression: all | none | union of own/team.
type ScopeExpr struct {
	All  bool
	None bool
	Own  bool
	Team bool
}

// ScopeEntry binds one role group (explicit roles or the "*" remainder) to
// one scope expression, in annotation order.
type ScopeEntry struct {
	Roles []string // expanded: "*" is resolved to the remaining roles
	Star  bool
	Expr  ScopeExpr
}

// Rule is one reconciled annotation rule.
type Rule struct {
	Invariants []string
	// exactly one of the three shapes is populated
	Grants        map[string][]string // role -> verbs (grants shape)
	GrantRoles    []string            // grants iteration order (enum order)
	ScopeVerbs    []string            // scope shape: verbs covered
	Scope         []ScopeEntry        // scope shape: role groups in order
	Reassign      []ScopeEntry        // reassign shape: actor scope
	ReassignAny   map[string]bool     // reassign shape: role -> target any
	ReassignRoles []string            // reassign role order (enum order)
}

// Policy is the fully reconciled annotation, ready for emission.
type Policy struct {
	SubjectEntity string
	RoleAttr      string
	RoleEnum      string
	Roles         []string // enum order
	TeamEntity    string   // "" when the design has no team scoping
	Membership    string   // lone | one
	RequiredFor   []string
	TeamInvs      []string
	Resources     []string
	OwnedInvs     []string
	Rules         []Rule
	Residuals     [][2]string // (invariant id, reason)
	Scope         int         // Alloy bound

	DomainFile     string // base name, for the generated header
	AnnotationFile string // base name, for the generated header

	grantsRule   *Rule // at most one
	scopeByVerb  map[string]*Rule
	reassignRule *Rule
}

// Command is one emitted Alloy command. Kind decides the pass semantics:
// a check passes on UNSAT (no counterexample), a run passes on SAT (the
// possibility it asserts exists).
type Command struct {
	Kind string // check | run
	Name string
}

// Stats summarizes a generation for the CLI line and the gate counts.
type Stats struct {
	Roles      int
	Resources  int
	Rules      int
	Residuals  int
	Commands   []Command
	Carried    int // invariant ids carried by the model (rules + structure)
	OracleRows int
}

// --- domain-model view ---

type domain struct {
	obj      *ir.Object
	enums    map[string][]string // enum name -> ordered values
	entities *ir.Object
	topInvs  []string        // top-level invariant ids, source order
	allInvs  map[string]bool // every invariant id (top + entity)
}

func listOf(v *ir.Value) []*ir.Value {
	if v == nil || v.Kind != ir.KindArray {
		return nil
	}
	return v.AsArray()
}

func loadDomain(path string) *domain {
	data, err := os.ReadFile(path)
	if err != nil {
		die("%s: %v", filepath.Base(path), err)
	}
	v, err := ir.LoadYAML(data)
	if err != nil || v.AsObject() == nil {
		die("%s: not a yaml mapping", filepath.Base(path))
	}
	d := &domain{obj: v.AsObject(), enums: map[string][]string{}, allInvs: map[string]bool{}}
	if ev := d.obj.Get2("enums"); ev != nil && ev.Kind == ir.KindObject {
		for _, k := range ev.AsObject().Keys() {
			var vals []string
			for _, vv := range listOf(ev.AsObject().Get2(k).AsObject().Get2("values")) {
				vals = append(vals, vv.AsObject().GetString("name"))
			}
			d.enums[k] = vals
		}
	}
	d.entities = d.obj.GetObject("entities")
	if d.entities.Len() == 0 {
		die("%s: declares no entities", filepath.Base(path))
	}
	for _, i := range listOf(d.obj.Get2("invariants")) {
		id := i.AsObject().GetString("id")
		if id != "" {
			d.topInvs = append(d.topInvs, id)
			d.allInvs[id] = true
		}
	}
	for _, ename := range d.entities.Keys() {
		for _, i := range listOf(d.entities.Get2(ename).AsObject().Get2("invariants")) {
			if id := i.AsObject().GetString("id"); id != "" {
				d.allInvs[id] = true
			}
		}
	}
	return d
}

// attrType returns the declared type of an attribute on an entity ("" if absent).
func (d *domain) attrType(entity, attr string) string {
	e, ok := d.entities.Get(entity)
	if !ok {
		return ""
	}
	for _, a := range listOf(e.AsObject().Get2("attributes")) {
		ao := a.AsObject()
		if ao.GetString("name") == attr {
			return ao.GetString("type")
		}
	}
	return ""
}

// hasRelationship reports whether entity `from` declares a relationship to
// entity `to`, optionally requiring a cardinality.
func (d *domain) hasRelationship(from, to, cardinality string) bool {
	e, ok := d.entities.Get(from)
	if !ok {
		return false
	}
	for _, r := range listOf(e.AsObject().Get2("relationships")) {
		ro := r.AsObject()
		if ro.GetString("entity") != to {
			continue
		}
		if cardinality == "" || ro.GetString("cardinality") == cardinality {
			return true
		}
	}
	return false
}

// --- annotation parsing + reconciliation ---

func strList(v *ir.Value, what string) []string {
	if v == nil {
		return nil
	}
	var out []string
	items := []*ir.Value{v}
	if v.Kind == ir.KindArray {
		items = v.AsArray()
	}
	for _, it := range items {
		if it == nil || it.Kind != ir.KindString {
			die("%s must be a string or list of strings", what)
		}
		out = append(out, it.AsString())
	}
	return out
}

func checkKeys(o *ir.Object, allowed map[string]bool, where string) {
	for _, k := range o.Keys() {
		if !allowed[k] {
			die("unsupported key %s in %s (a typo here silently weakens the policy)", ir.Repr(k), where)
		}
	}
}

var rootKeys = map[string]bool{
	"subjects": true, "resources": true, "owned_invariants": true,
	"rules": true, "residuals": true, "scope": true, "_comment": true,
}
var subjectKeys = map[string]bool{"entity": true, "role_attr": true, "team": true}
var teamKeys = map[string]bool{"entity": true, "membership": true, "required_for": true, "invariant": true}
var ruleKeys = map[string]bool{"invariant": true, "grants": true, "verbs": true, "scope": true, "reassign": true, "_comment": true}
var reassignKeys = map[string]bool{"scope": true, "target": true}
var residualKeys = map[string]bool{"invariant": true, "reason": true}

// parseScopeExpr parses "all", "none", or a union of own/team.
func parseScopeExpr(s, where string, hasTeam bool) ScopeExpr {
	var e ScopeExpr
	terms := strings.Split(s, "|")
	for i := range terms {
		terms[i] = strings.TrimSpace(terms[i])
	}
	if len(terms) == 1 && (terms[0] == "all" || terms[0] == "none") {
		e.All = terms[0] == "all"
		e.None = terms[0] == "none"
		return e
	}
	for _, t := range terms {
		switch t {
		case "own":
			e.Own = true
		case "team":
			if !hasTeam {
				die("%s uses scope term 'team' but the annotation declares no subjects.team", where)
			}
			e.Team = true
		case "all", "none":
			die("%s: '%s' must stand alone, not in a union", where, t)
		default:
			die("%s: unknown scope term %s (the v1 algebra is all, none, own, team and unions of own|team)", where, ir.Repr(t))
		}
	}
	return e
}

// parseScopeMap parses a {role-or-*: expr} object into ordered entries with
// the "*" wildcard expanded to the roles not explicitly named.
func parseScopeMap(o *ir.Object, roles []string, where string, hasTeam bool) []ScopeEntry {
	if o == nil || o.Len() == 0 {
		die("%s declares an empty scope map", where)
	}
	roleSet := map[string]bool{}
	for _, r := range roles {
		roleSet[r] = true
	}
	named := map[string]bool{}
	sawStar := false
	var entries []ScopeEntry
	for _, k := range o.Keys() {
		v := o.Get2(k)
		if v == nil || v.Kind != ir.KindString {
			die("%s: scope for %s must be a string expression", where, ir.Repr(k))
		}
		expr := parseScopeExpr(v.AsString(), where+" role "+ir.Repr(k), hasTeam)
		if k == "*" {
			if sawStar {
				die("%s: duplicate '*' entry", where)
			}
			sawStar = true
			entries = append(entries, ScopeEntry{Star: true, Expr: expr})
			continue
		}
		if !roleSet[k] {
			die("%s names role %s, which is not a value of the role enum", where, ir.Repr(k))
		}
		if named[k] {
			die("%s: duplicate role %s", where, ir.Repr(k))
		}
		named[k] = true
		entries = append(entries, ScopeEntry{Roles: []string{k}, Expr: expr})
	}
	// expand "*" to the remaining roles in enum order
	for i := range entries {
		if entries[i].Star {
			var rest []string
			for _, r := range roles {
				if !named[r] {
					rest = append(rest, r)
				}
			}
			if len(rest) == 0 {
				die("%s: '*' matches no remaining role", where)
			}
			entries[i].Roles = rest
		}
	}
	return entries
}

// Load parses and reconciles the annotation against the domain model.
// Every disagreement dies: a drifted annotation must fail generation instead
// of proving a stale twin (the refine_gen rule, applied here).
func Load(domainPath, annotationPath string) *Policy {
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
	checkKeys(root, rootKeys, filepath.Base(annotationPath))

	l := &policyLoader{
		d:       d,
		root:    root,
		claimed: map[string]string{},
		p: &Policy{
			Scope:          6,
			DomainFile:     filepath.Base(domainPath),
			AnnotationFile: filepath.Base(annotationPath),
			scopeByVerb:    map[string]*Rule{},
		},
	}
	l.loadSubjects()
	l.loadResources()
	l.validateNames()
	l.seedStructuralClaims()
	l.loadRules()
	l.validateGrantScopeClosure()
	l.loadResiduals()
	l.validateCoverage()
	l.loadScope()
	return l.p
}

type policyLoader struct {
	d       *domain
	root    *ir.Object
	p       *Policy
	claimed map[string]string
}

func (l *policyLoader) loadSubjects() {
	subj := l.root.GetObject("subjects")
	if subj.Len() == 0 {
		die("annotation declares no subjects")
	}
	checkKeys(subj, subjectKeys, "subjects")
	l.p.SubjectEntity = subj.GetString("entity")
	if l.p.SubjectEntity == "" {
		die("subjects.entity is required")
	}
	if _, ok := l.d.entities.Get(l.p.SubjectEntity); !ok {
		die("subjects.entity %s is not a Modelith entity", ir.Repr(l.p.SubjectEntity))
	}
	l.p.RoleAttr = subj.GetString("role_attr")
	if l.p.RoleAttr == "" {
		die("subjects.role_attr is required")
	}
	l.p.RoleEnum = l.d.attrType(l.p.SubjectEntity, l.p.RoleAttr)
	if l.p.RoleEnum == "" {
		die("subjects.role_attr %s is not an attribute of %s", ir.Repr(l.p.RoleAttr), l.p.SubjectEntity)
	}
	roles, ok := l.d.enums[l.p.RoleEnum]
	if !ok || len(roles) == 0 {
		die("subjects.role_attr %s has type %s, which is not an enum with values", ir.Repr(l.p.RoleAttr), ir.Repr(l.p.RoleEnum))
	}
	l.p.Roles = roles
	if tv := subj.Get2("team"); tv != nil {
		l.loadTeam(tv)
	}
}

func (l *policyLoader) loadTeam(v *ir.Value) {
	team := v.AsObject()
	if team == nil {
		die("subjects.team must be a mapping")
	}
	checkKeys(team, teamKeys, "subjects.team")
	l.p.TeamEntity = team.GetString("entity")
	if l.p.TeamEntity == "" {
		die("subjects.team.entity is required")
	}
	if _, ok := l.d.entities.Get(l.p.TeamEntity); !ok {
		die("subjects.team.entity %s is not a Modelith entity", ir.Repr(l.p.TeamEntity))
	}
	if !l.d.hasRelationship(l.p.TeamEntity, l.p.SubjectEntity, "") && !l.d.hasRelationship(l.p.SubjectEntity, l.p.TeamEntity, "") {
		die("the domain model declares no relationship between %s and %s; team scoping has nothing to bind to", l.p.TeamEntity, l.p.SubjectEntity)
	}
	l.p.Membership = team.GetString("membership")
	if l.p.Membership != "lone" && l.p.Membership != "one" {
		die("subjects.team.membership must be 'lone' or 'one' (Modelith cardinality cannot express which; the annotation must decide)")
	}
	l.p.RequiredFor = strList(team.Get2("required_for"), "subjects.team.required_for")
	if len(l.p.RequiredFor) > 0 && l.p.Membership == "one" {
		die("subjects.team.required_for is redundant with membership 'one'; drop one of them")
	}
	roleSet := setOf(l.p.Roles)
	for _, role := range l.p.RequiredFor {
		if !roleSet[role] {
			die("subjects.team.required_for names role %s, which is not a value of enum %s", ir.Repr(role), l.p.RoleEnum)
		}
	}
	l.p.TeamInvs = strList(team.Get2("invariant"), "subjects.team.invariant")
}

func (l *policyLoader) loadResources() {
	l.p.Resources = strList(l.root.Get2("resources"), "resources")
	if len(l.p.Resources) == 0 {
		die("resources must list at least one entity")
	}
	seen := map[string]bool{}
	for _, resource := range l.p.Resources {
		if seen[resource] {
			die("resources lists %s twice", ir.Repr(resource))
		}
		seen[resource] = true
		if _, ok := l.d.entities.Get(resource); !ok {
			die("resource %s is not a Modelith entity", ir.Repr(resource))
		}
		if resource == l.p.SubjectEntity || resource == l.p.TeamEntity {
			die("resource %s is the subject or team entity; a resource is a record under ownership", ir.Repr(resource))
		}
		if !l.d.hasRelationship(resource, l.p.SubjectEntity, "n:1") {
			die("resource %s declares no n:1 relationship to %s (ownership); add the relationship or drop the resource", ir.Repr(resource), l.p.SubjectEntity)
		}
	}
	l.p.OwnedInvs = strList(l.root.Get2("owned_invariants"), "owned_invariants")
}

func (l *policyLoader) validateNames() {
	reserved := map[string]bool{"Role": true, "Record": true}
	for _, role := range l.p.Roles {
		if reserved[role] || role == l.p.SubjectEntity || role == l.p.TeamEntity {
			die("role value %s collides with a signature name; rename it in the enum", ir.Repr(role))
		}
		for _, resource := range l.p.Resources {
			if role == resource {
				die("role value %s collides with resource entity %s; rename one", ir.Repr(role), ir.Repr(resource))
			}
		}
	}
	for _, name := range append([]string{l.p.SubjectEntity, l.p.TeamEntity}, l.p.Resources...) {
		if reserved[name] {
			die("entity %s collides with a reserved signature name (Role, Record)", ir.Repr(name))
		}
	}
}

func (l *policyLoader) claim(id, by, where string) {
	if !l.d.allInvs[id] {
		die("%s references invariant %s, which the domain model does not declare", where, ir.Repr(id))
	}
	if prev, dup := l.claimed[id]; dup {
		die("invariant %s is claimed twice (%s and %s); every id maps to exactly one rule or residual", ir.Repr(id), prev, by)
	}
	l.claimed[id] = by
}

func (l *policyLoader) seedStructuralClaims() {
	for _, inv := range l.p.TeamInvs {
		l.claim(inv, "subjects.team", "subjects.team.invariant")
	}
	for _, inv := range l.p.OwnedInvs {
		l.claim(inv, "owned_invariants", "owned_invariants")
	}
}

func (l *policyLoader) loadRules() {
	rules := listOf(l.root.Get2("rules"))
	if len(rules) == 0 {
		die("annotation declares no rules; with nothing to compile there is nothing to check")
	}
	for i, rv := range rules {
		ro := rv.AsObject()
		if ro == nil {
			die("rules[%d] is not a mapping", i)
		}
		checkKeys(ro, ruleKeys, fmt.Sprintf("rules[%d]", i))
		rule := Rule{Invariants: strList(ro.Get2("invariant"), fmt.Sprintf("rules[%d].invariant", i))}
		if len(rule.Invariants) == 0 {
			die("rules[%d] declares no invariant id; every rule carries the id it compiles", i)
		}
		where := fmt.Sprintf("rules[%d] (%s)", i, rule.Invariants[0])
		for _, inv := range rule.Invariants {
			l.claim(inv, "a rule", where)
		}
		shapes := 0
		for _, key := range []string{"grants", "scope", "reassign"} {
			if ro.Get2(key) != nil {
				shapes++
			}
		}
		if shapes != 1 {
			die("%s must have exactly one shape: grants, verbs+scope, or reassign", where)
		}
		switch {
		case ro.Get2("grants") != nil:
			l.loadGrantRule(ro, rule, where)
		case ro.Get2("scope") != nil:
			l.loadScopeRule(ro, rule, where)
		default:
			l.loadReassignRule(ro, rule, where)
		}
	}
}

func (l *policyLoader) loadGrantRule(ro *ir.Object, rule Rule, where string) {
	if l.p.grantsRule != nil {
		die("%s: a second grants rule; v1 supports exactly one verb-capability map", where)
	}
	g := ro.GetObject("grants")
	rule.Grants = map[string][]string{}
	roleSet := setOf(l.p.Roles)
	for _, role := range g.Keys() {
		if !roleSet[role] {
			die("%s grants to %s, which is not a value of enum %s", where, ir.Repr(role), l.p.RoleEnum)
		}
		verbs := strList(g.Get2(role), where+" grants."+role)
		for _, verb := range verbs {
			if !verbSet[verb] {
				die("%s grants unknown verb %s (the v1 vocabulary is %s; reassign is its own rule)", where, ir.Repr(verb), strings.Join(Verbs, ", "))
			}
		}
		rule.Grants[role] = verbs
	}
	for _, role := range l.p.Roles {
		if _, ok := rule.Grants[role]; ok {
			rule.GrantRoles = append(rule.GrantRoles, role)
		}
	}
	l.p.Rules = append(l.p.Rules, rule)
	l.p.grantsRule = &l.p.Rules[len(l.p.Rules)-1]
}

func (l *policyLoader) loadScopeRule(ro *ir.Object, rule Rule, where string) {
	rule.ScopeVerbs = strList(ro.Get2("verbs"), where+" verbs")
	if len(rule.ScopeVerbs) == 0 {
		die("%s: a scope rule needs a verbs list", where)
	}
	for _, verb := range rule.ScopeVerbs {
		if !scopedVerbs[verb] {
			die("%s: verb %s cannot carry a scope (scoped verbs: read, update, delete)", where, ir.Repr(verb))
		}
		if l.p.scopeByVerb[verb] != nil {
			die("%s: verb %s is already scoped by another rule", where, ir.Repr(verb))
		}
	}
	rule.Scope = parseScopeMap(ro.GetObject("scope"), l.p.Roles, where, l.p.TeamEntity != "")
	l.p.Rules = append(l.p.Rules, rule)
	for _, verb := range rule.ScopeVerbs {
		l.p.scopeByVerb[verb] = &l.p.Rules[len(l.p.Rules)-1]
	}
}

func (l *policyLoader) loadReassignRule(ro *ir.Object, rule Rule, where string) {
	if l.p.reassignRule != nil {
		die("%s: a second reassign rule", where)
	}
	ra := ro.GetObject("reassign")
	checkKeys(ra, reassignKeys, where+" reassign")
	rule.Reassign = parseScopeMap(ra.GetObject("scope"), l.p.Roles, where+" reassign.scope", l.p.TeamEntity != "")
	inScope := map[string]bool{}
	for _, entry := range rule.Reassign {
		if entry.Expr.None {
			die("%s: reassign scope 'none' is the default; omit the role instead", where)
		}
		for _, role := range entry.Roles {
			inScope[role] = true
		}
	}
	target := ra.GetObject("target")
	if target.Len() == 0 {
		die("%s: reassign.target is required; where a record may go must be stated, not implied", where)
	}
	rule.ReassignAny = map[string]bool{}
	for _, role := range target.Keys() {
		if !inScope[role] {
			die("%s: reassign.target names %s, which has no reassign scope", where, ir.Repr(role))
		}
		value := target.Get2(role)
		if value == nil || value.Kind != ir.KindString {
			die("%s: reassign.target.%s must be 'any' or 'team'", where, role)
		}
		switch value.AsString() {
		case "any":
			rule.ReassignAny[role] = true
		case "team":
			if l.p.TeamEntity == "" {
				die("%s: reassign.target 'team' needs subjects.team", where)
			}
		default:
			die("%s: reassign.target.%s must be 'any' or 'team', got %s", where, role, ir.Repr(value.AsString()))
		}
	}
	for role := range inScope {
		if _, ok := target.Get(role); !ok {
			die("%s: reassign.target must decide every role with reassign scope; %s is undecided", where, ir.Repr(role))
		}
	}
	for _, role := range l.p.Roles {
		if inScope[role] {
			rule.ReassignRoles = append(rule.ReassignRoles, role)
		}
	}
	l.p.Rules = append(l.p.Rules, rule)
	l.p.reassignRule = &l.p.Rules[len(l.p.Rules)-1]
}

func (l *policyLoader) validateGrantScopeClosure() {
	if l.p.grantsRule == nil {
		return
	}
	for _, verb := range []string{"read", "update", "delete"} {
		roles := l.p.rolesWith(verb)
		scope := l.p.scopeByVerb[verb]
		switch {
		case len(roles) > 0 && scope == nil:
			die("grants rule grants %s to %s but no scope rule defines which records they may access", verb, strings.Join(roles, ", "))
		case len(roles) == 0 && scope != nil:
			die("scope rule defines %s authority but the grants rule grants %s to no role", verb, verb)
		}
	}
}

func (l *policyLoader) loadResiduals() {
	for i, rv := range listOf(l.root.Get2("residuals")) {
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
		l.claim(id, "a residual", fmt.Sprintf("residuals[%d]", i))
		l.p.Residuals = append(l.p.Residuals, [2]string{id, reason})
	}
}

func (l *policyLoader) validateCoverage() {
	var uncovered []string
	for _, id := range l.d.topInvs {
		if _, ok := l.claimed[id]; !ok {
			uncovered = append(uncovered, id)
		}
	}
	if len(uncovered) > 0 {
		die("top-level invariant(s) %s are neither compiled by a rule nor waived by a residual; the relational layer must account for every cross-cutting invariant", strings.Join(uncovered, ", "))
	}
}

func (l *policyLoader) loadScope() {
	if sv := l.root.Get2("scope"); sv != nil {
		n, err := sv.AsNumber().Int64()
		if sv.Kind != ir.KindNumber || err != nil || n < 2 || n > 12 {
			die("scope must be an integer between 2 and 12")
		}
		l.p.Scope = int(n)
	}
}

func setOf(xs []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// --- emission ---

func roleUnion(roles []string) string {
	if len(roles) == 1 {
		return roles[0]
	}
	return "(" + strings.Join(roles, " + ") + ")"
}

// scopeCond renders a scope expression over owner expression ownerExpr for
// subject u. Returns "" for all (no extra condition).
func scopeCond(e ScopeExpr, ownerExpr string) string {
	if e.All {
		return ""
	}
	var terms []string
	if e.Own {
		terms = append(terms, ownerExpr+" = u")
	}
	if e.Team {
		terms = append(terms, "sameTeam[u, "+ownerExpr+"]")
	}
	return strings.Join(terms, " or ")
}

// branch renders one role-group disjunct: role test plus scope condition.
func branch(roles []string, e ScopeExpr, ownerExpr string) string {
	test := "u.role in " + roleUnion(roles)
	if len(roles) == 1 {
		test = "u.role = " + roles[0]
	}
	cond := scopeCond(e, ownerExpr)
	if cond == "" {
		return test
	}
	if strings.Contains(cond, " or ") {
		cond = "(" + cond + ")"
	}
	return test + " and " + cond
}

func verbTitle(v string) string { return strings.ToUpper(v[:1]) + v[1:] }

// rolesWith returns the roles that hold verb v: from the grants rule when one
// exists, else every role with a non-none scope for v.
func (p *Policy) rolesWith(v string) []string {
	if p.grantsRule != nil {
		var out []string
		for _, role := range p.grantsRule.GrantRoles {
			for _, g := range p.grantsRule.Grants[role] {
				if g == v {
					out = append(out, role)
					break
				}
			}
		}
		return out
	}
	rule := p.scopeByVerb[v]
	if rule == nil {
		return nil
	}
	var out []string
	inRule := map[string]bool{}
	for _, e := range rule.Scope {
		if e.Expr.None {
			continue
		}
		for _, r := range e.Roles {
			inRule[r] = true
		}
	}
	for _, r := range p.Roles {
		if inRule[r] {
			out = append(out, r)
		}
	}
	return out
}

// Generate emits the model. Deterministic: every ordering comes from the
// source YAML (enum order, resource order, rule order).
func Generate(domainPath, annotationPath string) (als string, stats Stats, err error) {
	als, _, stats, err = GenerateAll(domainPath, annotationPath)
	return als, stats, err
}

// GenerateAll emits both compiled forms of the annotation: the Alloy model
// (for the solver) and the authorization oracle (for the implementation
// tests). One reconciliation, two artifacts, so they can never disagree.
func GenerateAll(domainPath, annotationPath string) (als, oracle string, stats Stats, err error) {
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(*ExitError); ok {
				err = ee
				return
			}
			panic(r)
		}
	}()
	p := Load(domainPath, annotationPath)
	als, stats = p.emit()
	oracle, stats.OracleRows = p.GenerateOracle()
	return als, oracle, stats, nil
}

func (p *Policy) emit() (string, Stats) {
	var b strings.Builder
	w := func(format string, args ...interface{}) {
		fmt.Fprintf(&b, format+"\n", args...)
	}
	stats := Stats{Roles: len(p.Roles), Resources: len(p.Resources), Rules: len(p.Rules), Residuals: len(p.Residuals)}
	carried := map[string]bool{}
	for _, r := range p.Rules {
		for _, id := range r.Invariants {
			carried[id] = true
		}
	}
	for _, id := range p.TeamInvs {
		carried[id] = true
	}
	for _, id := range p.OwnedInvs {
		carried[id] = true
	}
	stats.Carried = len(carried)

	w("// Code generated from %s + %s by machinery alloy. DO NOT EDIT.", p.DomainFile, p.AnnotationFile)
	w("//")
	w("// Static relational model of the policy invariants: which configurations of")
	w("// subjects, teams, and record ownership the invariant set admits. Alloy")
	w("// searches every configuration within the bound, so a green check is")
	w("// exhaustive there and silent beyond it.")
	w("//")
	w("// ASSUMPTIONS (what this abstraction erases; results are conditional on them):")
	w("//   1. Bounded search: exhaustive only within scope %d (at most %d atoms per", p.Scope, p.Scope)
	w("//      signature). PASS on a check means no counterexample within the bound.")
	w("//   2. Statics only: this rung checks admissible configurations, never how")
	w("//      the system moves between them; lifecycle and temporal properties are")
	w("//      the TLC rung (machinery tla / machinery refine).")
	w("//   3. Only what the annotation names is modeled: roles, team membership,")
	w("//      ownership. Sessions, verb semantics in code, and every other")
	w("//      attribute are carried by the machines and the implementation tests.")
	w("//   4. Every resource entity is policy-equivalent: one Record signature")
	w("//      with exactly one owner stands for all of them (the policy treats")
	w("//      them alike). A single sig, not subsignatures, so the solver's")
	w("//      receipt renders the owner relation in every counterexample.")
	if len(p.Residuals) > 0 {
		w("//")
		w("// RESIDUALS (invariants the algebra does not carry; enforced elsewhere):")
		for _, r := range p.Residuals {
			w("//   - %s: %s", r[0], r[1])
		}
	}
	w("")
	w("module Policy")
	w("")
	w("// role enum %s", p.RoleEnum)
	w("abstract sig Role {}")
	w("one sig %s extends Role {}", strings.Join(p.Roles, ", "))
	w("")
	if p.TeamEntity != "" {
		inv := ""
		if len(p.TeamInvs) > 0 {
			inv = " (" + strings.Join(p.TeamInvs, ", ") + ")"
		}
		w("// team scoping: membership %s%s", p.Membership, inv)
		w("sig %s {}", p.TeamEntity)
		w("")
		w("// subjects: %s", p.SubjectEntity)
		w("sig %s {", p.SubjectEntity)
		w("  role: one Role,")
		w("  team: %s %s", p.Membership, p.TeamEntity)
		w("}")
	} else {
		w("// subjects: %s", p.SubjectEntity)
		w("sig %s {", p.SubjectEntity)
		w("  role: one Role")
		w("}")
	}
	if len(p.RequiredFor) > 0 {
		w("")
		w("// roles that must hold a team (required_for)")
		w("fact RequiredTeams {")
		w("  all u: %s | u.role in %s implies some u.team", p.SubjectEntity, roleUnion(p.RequiredFor))
		w("}")
	}
	w("")
	inv := ""
	if len(p.OwnedInvs) > 0 {
		inv = " (" + strings.Join(p.OwnedInvs, ", ") + ")"
	}
	w("// resources: %s", strings.Join(p.Resources, ", "))
	w("// every record has exactly one owner%s", inv)
	w("sig Record { owner: one %s }", p.SubjectEntity)
	if p.TeamEntity != "" {
		w("")
		w("// a member of the subject's team: both hold a team and it is the same one;")
		w("// a teamless subject is nobody's teammate")
		w("pred sameTeam[a, b: %s] { some a.team and a.team = b.team }", p.SubjectEntity)
	}

	// grants preds
	if p.grantsRule != nil {
		w("")
		w("// %s: verb capability by role", strings.Join(p.grantsRule.Invariants, ", "))
		for _, verb := range Verbs {
			roles := p.rolesWith(verb)
			if len(roles) == 0 {
				w("// no role is granted %s", verb)
				continue
			}
			w("pred grants%s[u: %s] { u.role in %s }", verbTitle(verb), p.SubjectEntity, roleUnion(roles))
		}
	}

	// scope rules -> per-verb canX preds
	for i := range p.Rules {
		r := &p.Rules[i]
		if len(r.ScopeVerbs) == 0 {
			continue
		}
		w("")
		w("// %s: %s scope by role", strings.Join(r.Invariants, ", "), strings.Join(r.ScopeVerbs, "/"))
		for _, verb := range r.ScopeVerbs {
			var branches []string
			for _, e := range r.Scope {
				if e.Expr.None {
					continue
				}
				branches = append(branches, branch(e.Roles, e.Expr, "r.owner"))
			}
			for _, e := range r.Scope {
				if e.Expr.None {
					w("// %s: none (no %s scope)", strings.Join(e.Roles, ", "), verb)
				}
			}
			if len(branches) == 0 {
				die("scope rule for %s grants nothing to any role; an all-none rule checks nothing", verb)
			}
			gate := ""
			if p.grantsRule != nil {
				gate = fmt.Sprintf("grants%s[u] and ", verbTitle(verb))
			}
			if len(branches) == 1 && gate == "" {
				w("pred can%s[u: %s, r: Record] { %s }", verbTitle(verb), p.SubjectEntity, branches[0])
			} else {
				w("pred can%s[u: %s, r: Record] {", verbTitle(verb), p.SubjectEntity)
				w("  %s(", gate)
				for j, br := range branches {
					sep := ""
					if j < len(branches)-1 {
						sep = ""
					}
					prefix := "    "
					if j > 0 {
						prefix = "    or "
					}
					w("%s%s%s", prefix, br, sep)
				}
				w("  )")
				w("}")
			}
		}
	}

	// reassign rule
	if p.reassignRule != nil {
		r := p.reassignRule
		w("")
		w("// %s: who may change an owner, and where the record may go", strings.Join(r.Invariants, ", "))
		w("pred canReassign[u: %s, r: Record, t: %s] {", p.SubjectEntity, p.SubjectEntity)
		first := true
		for _, e := range r.Reassign {
			for _, role := range e.Roles {
				cond := branch([]string{role}, e.Expr, "r.owner")
				if r.ReassignAny[role] {
					cond += " and t in " + p.SubjectEntity + " // target: any"
				} else {
					cond += " and sameTeam[u, t] // target: team"
				}
				prefix := "  "
				if !first {
					prefix = "  or "
				}
				w("%s%s", prefix, cond)
				first = false
			}
		}
		w("}")
	}

	// --- meta-checks ---
	w("")
	w("// --- Generated meta-checks: the standard suite, identical for every design ---")
	w("")
	w("// PASS = instance found: the invariants admit at least one world with every")
	w("// role inhabited and a record present. No instance = the policy contradicts")
	w("// itself (vacuity guard: everything below would pass emptily).")
	w("run SomeWorld {")
	w("  (all ro: Role | some u: %s | u.role = ro) and some Record", p.SubjectEntity)
	w("} for %d", p.Scope)
	stats.Commands = append(stats.Commands, Command{Kind: "run", Name: "SomeWorld"})

	readRule := p.scopeByVerb["read"]
	var writeVerbs []string
	for _, v := range []string{"update", "delete"} {
		if p.scopeByVerb[v] != nil {
			writeVerbs = append(writeVerbs, v)
		}
	}
	if readRule != nil && len(writeVerbs) > 0 {
		var disj []string
		for _, v := range writeVerbs {
			disj = append(disj, fmt.Sprintf("can%s[u, r]", verbTitle(v)))
		}
		w("")
		w("// PASS = no counterexample: anyone who may write a record may also read it")
		w("// (write scope stays inside read scope).")
		w("check WriteImpliesRead {")
		w("  all u: %s, r: Record | (%s) implies canRead[u, r]", p.SubjectEntity, strings.Join(disj, " or "))
		w("} for %d", p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "check", Name: "WriteImpliesRead"})
	}
	if len(writeVerbs) > 0 {
		w("")
		w("// PASS = no counterexample: every role granted a write verb can write the")
		w("// records it owns. A counterexample here is an unstated assumption: some")
		w("// write-capable subject exists whose own records are outside its scope.")
		w("check CapableWritesOwn {")
		for _, v := range writeVerbs {
			roles := p.rolesWith(v)
			if len(roles) == 0 {
				continue
			}
			w("  all u: %s, r: Record | (u.role in %s and r.owner = u) implies can%s[u, r]", p.SubjectEntity, roleUnion(roles), verbTitle(v))
		}
		w("} for %d", p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "check", Name: "CapableWritesOwn"})
	}
	if p.reassignRule != nil {
		w("")
		w("// PASS = no counterexample: a legal reassign leaves the record inside the")
		w("// actor's reassign authority (no one-step escape: the actor cannot hand a")
		w("// record somewhere it could not touch again).")
		w("check ReassignRetainsAuthority {")
		var disj []string
		for _, e := range p.reassignRule.Reassign {
			for _, role := range e.Roles {
				disj = append(disj, branch([]string{role}, e.Expr, "t"))
			}
		}
		w("  all u, t: %s, r: Record | canReassign[u, r, t] implies", p.SubjectEntity)
		w("    (%s)", strings.Join(disj, " or "))
		w("} for %d", p.Scope)
		stats.Commands = append(stats.Commands, Command{Kind: "check", Name: "ReassignRetainsAuthority"})
	}
	// per-role exercisability: a granted verb must be usable in some world
	var possible []string
	var possibleNames []string
	for _, role := range p.Roles {
		for _, verb := range []string{"read", "update", "delete"} {
			if p.scopeByVerb[verb] == nil {
				continue
			}
			held := false
			for _, r := range p.rolesWith(verb) {
				if r == role {
					held = true
					break
				}
			}
			if !held {
				continue
			}
			name := fmt.Sprintf("Possible_%s_%s", role, verb)
			possibleNames = append(possibleNames, name)
			possible = append(possible, fmt.Sprintf("run %s { some u: %s, r: Record | u.role = %s and can%s[u, r] } for %d",
				name, p.SubjectEntity, role, verbTitle(verb), p.Scope))
		}
	}
	if len(possible) > 0 {
		w("")
		w("// PASS = instance found, one per (role, granted verb): the grant is")
		w("// exercisable somewhere. No instance = the grant is vacuous (granted a")
		w("// verb whose scope is empty in every admissible world).")
		for i, cmd := range possible {
			w("%s", cmd)
			stats.Commands = append(stats.Commands, Command{Kind: "run", Name: possibleNames[i]})
		}
	}
	return b.String(), stats
}

// CarriedIDs extracts the invariant ids the annotation claims to carry
// (rules, team multiplicity, ownership), WITHOUT validating them; Gp-policy
// owns validation. Gx-trace uses this to credit the relational model as an
// enforcement artifact. A missing or malformed annotation yields nil.
func CarriedIDs(annotationPath string) map[string]bool {
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
	if subj := root.GetObject("subjects"); subj != nil {
		if team := subj.GetObject("team"); team != nil {
			add(team.Get2("invariant"))
		}
	}
	add(root.Get2("owned_invariants"))
	for _, rv := range listOf(root.Get2("rules")) {
		if ro := rv.AsObject(); ro != nil {
			add(ro.Get2("invariant"))
		}
	}
	return ids
}

// --- CLI entrypoints ---

// Paths resolves the domain model and annotation for a design dir.
// The annotation path is returned even when absent (callers stat it).
func Paths(design string) (domainPath, annotationPath string, err error) {
	entries, _ := os.ReadDir(design)
	var models []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".modelith.yaml") {
			models = append(models, filepath.Join(design, e.Name()))
		}
	}
	if len(models) == 0 {
		return "", "", fmt.Errorf("alloy_gen: no *.modelith.yaml in %s", design)
	}
	if len(models) > 1 {
		var names []string
		for _, p := range models {
			names = append(names, filepath.Base(p))
		}
		sort.Strings(names)
		return "", "", fmt.Errorf("alloy_gen: multiple modelith models: %s", strings.Join(names, ", "))
	}
	return models[0], filepath.Join(design, "formal", AnnotationName), nil
}

func statFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

// Run is the `machinery alloy <design> [out-dir]` entrypoint. One command
// emits every relational layer the design opted into (policy, integrity, and isolation), one
// artifact set per present annotation, in a fixed order. At least one
// annotation must be present; each layer is independently skippable.
func Run(design, outdir string) error {
	domainPath, policyAnn, err := Paths(design)
	if err != nil {
		return err
	}
	if outdir == "" {
		outdir = filepath.Join(design, "formal")
	}
	formalDir := filepath.Join(design, "formal")
	integrityAnn := filepath.Join(formalDir, IntegrityAnnotationName)
	isolationAnn := filepath.Join(formalDir, IsolationAnnotationName)

	havePolicy := statFile(policyAnn)
	haveIntegrity := statFile(integrityAnn)
	haveIsolation := statFile(isolationAnn)
	if !havePolicy && !haveIntegrity && !haveIsolation {
		return fmt.Errorf("alloy_gen: no relational annotation under %s (looked for %s, %s, %s); the relational layer is opt-in, author an annotation first (see the machinery skill, Phase 1)",
			formalDir, AnnotationName, IntegrityAnnotationName, IsolationAnnotationName)
	}
	if err := os.MkdirAll(outdir, 0755); err != nil {
		return err
	}

	if havePolicy {
		als, oracle, stats, err := GenerateAll(domainPath, policyAnn)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outdir, OutputName), []byte(als), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outdir, OracleName), []byte(oracle), 0644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "alloy_gen: reconciled %s against %s: %d roles, %d resources, %d rule(s), %d residual(s), %d invariant(s) carried\n",
			AnnotationName, filepath.Base(domainPath), stats.Roles, stats.Resources, stats.Rules, stats.Residuals, stats.Carried)
		fmt.Fprintf(os.Stdout, "wrote %s (%d commands) and %s (%d decision rows) to %s\n",
			OutputName, len(stats.Commands), OracleName, stats.OracleRows, outdir)
	}

	if haveIntegrity {
		als, stats, err := GenerateIntegrity(domainPath, integrityAnn)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outdir, IntegrityOutputName), []byte(als), 0644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "alloy_gen: reconciled %s against %s: %d entities, %d relationship(s), %d unique, %d singleton, %d residual(s), %d invariant(s) carried\n",
			IntegrityAnnotationName, filepath.Base(domainPath), stats.Entities, stats.Relationships, stats.Uniques, stats.Singletons, stats.Residuals, stats.Carried)
		fmt.Fprintf(os.Stdout, "wrote %s (%d commands) to %s\n", IntegrityOutputName, len(stats.Commands), outdir)
	}

	if haveIsolation {
		als, oracle, stats, err := GenerateIsolation(domainPath, isolationAnn)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outdir, IsolationOutputName), []byte(als), 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(outdir, IsolationOracleName), []byte(oracle), 0644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "alloy_gen: reconciled %s against %s: %d record(s), %d reference(s), %d residual(s), %d invariant(s) carried\n",
			IsolationAnnotationName, filepath.Base(domainPath), stats.Records, stats.References, stats.Residuals, stats.Carried)
		fmt.Fprintf(os.Stdout, "wrote %s (%d commands) and %s (%d decision rows) to %s\n",
			IsolationOutputName, len(stats.Commands), IsolationOracleName, stats.OracleRows, outdir)
	}
	return nil
}
