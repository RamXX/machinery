package alloy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot() string {
	p, _ := filepath.Abs("../..")
	return p
}

// --- the real fixture: the go-crm example ---

func TestGenerateGoCRM(t *testing.T) {
	als, stats, err := Generate(
		filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml"),
		filepath.Join(repoRoot(), "examples/go-crm/design/formal/policy.relational.yaml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Roles != 4 || stats.Resources != 5 || stats.Rules != 4 {
		t.Errorf("stats = %+v; want 4 roles, 5 resources, 4 rules", stats)
	}
	for _, want := range []string{
		"module Policy",
		"abstract sig Role {}",
		"sig Record { owner: one User }",
		"pred sameTeam[a, b: User]",
		"pred canRead[u: User, r: Record]",
		"pred canReassign[u: User, r: Record, t: User]",
		"run SomeWorld",
		"check WriteImpliesRead",
		"check CapableWritesOwn",
		"check ReassignRetainsAuthority",
		"run Possible_ReadOnly_read",
		"DO NOT EDIT",
	} {
		if !strings.Contains(als, want) {
			t.Errorf("generated model missing %q", want)
		}
	}
	if strings.Contains(als, "Possible_ReadOnly_update") {
		t.Error("ReadOnly must not get a write exercisability run")
	}
}

func TestDeterminism(t *testing.T) {
	dm := filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml")
	an := filepath.Join(repoRoot(), "examples/go-crm/design/formal/policy.relational.yaml")
	a1, _, err := Generate(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	a2, _, err := Generate(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Error("generation is not deterministic")
	}
}

// --- synthetic fixtures ---

const miniDomain = `
enums:
  R:
    values:
      - {name: A, definition: a}
      - {name: B, definition: b}
      - {name: C, definition: c}
entities:
  U:
    attributes:
      - {name: role, type: R}
      - {name: nick, type: string}
    invariants:
      - {id: u-team, statement: s}
  T:
    relationships:
      - {entity: U, cardinality: "1:n"}
  Rec:
    relationships:
      - {entity: U, cardinality: "n:1"}
    invariants:
      - {id: rec-owned, statement: s}
  Loose:
    attributes:
      - {name: x, type: string}
invariants:
  - {id: top-read, statement: s}
  - {id: top-write, statement: s}
`

const miniAnnotation = `
subjects:
  entity: U
  role_attr: role
  team: {entity: T, membership: lone, invariant: u-team}
resources: [Rec]
owned_invariants: [rec-owned]
rules:
  - invariant: top-read
    verbs: [read]
    scope: {A: all, "*": own | team}
  - invariant: top-write
    verbs: [update, delete]
    scope: {A: all, B: own, C: none}
`

func write(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func gen(t *testing.T, domain, annotation string) (string, Stats, error) {
	t.Helper()
	dir := t.TempDir()
	dm := write(t, dir, "domain.modelith.yaml", domain)
	an := write(t, dir, "policy.relational.yaml", annotation)
	return Generate(dm, an)
}

func TestMiniModel(t *testing.T) {
	als, stats, err := gen(t, miniDomain, miniAnnotation)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rules != 2 || stats.Roles != 3 || stats.Resources != 1 {
		t.Errorf("stats = %+v", stats)
	}
	// wildcard expanded to the remaining roles in enum order
	if !strings.Contains(als, "u.role in (B + C) and (r.owner = u or sameTeam[u, r.owner])") {
		t.Errorf("wildcard expansion wrong:\n%s", als)
	}
	// C: none appears as a comment, not a branch
	if !strings.Contains(als, "// C: none (no update scope)") {
		t.Error("none scope should be documented as a comment")
	}
	// no grants rule: canUpdate has no grants gate
	if strings.Contains(als, "grantsUpdate") {
		t.Error("no grants rule declared, yet a grants pred was emitted")
	}
	// no reassign rule: no reassign pred or check
	if strings.Contains(als, "canReassign") || strings.Contains(als, "ReassignRetainsAuthority") {
		t.Error("no reassign rule declared, yet reassign artifacts were emitted")
	}
	if !strings.Contains(als, "check WriteImpliesRead") || !strings.Contains(als, "check CapableWritesOwn") {
		t.Error("standard checks missing")
	}
}

func TestNoTeam(t *testing.T) {
	domain := `
enums:
  R:
    values:
      - {name: A, definition: a}
entities:
  U:
    attributes:
      - {name: role, type: R}
  Rec:
    relationships:
      - {entity: U, cardinality: "n:1"}
invariants:
  - {id: top-read, statement: s}
`
	annotation := `
subjects:
  entity: U
  role_attr: role
resources: [Rec]
rules:
  - invariant: top-read
    verbs: [read]
    scope: {A: own}
`
	als, _, err := gen(t, domain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(als, "sameTeam") || strings.Contains(als, "team:") {
		t.Error("teamless design must emit no team artifacts")
	}
	// read-only policy: no write verbs, so no write checks
	if strings.Contains(als, "WriteImpliesRead") || strings.Contains(als, "CapableWritesOwn") {
		t.Error("no write verbs scoped, yet write checks were emitted")
	}
}

func TestScopeOverride(t *testing.T) {
	als, _, err := gen(t, miniDomain, miniAnnotation+"scope: 4\n")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "} for 4") || strings.Contains(als, "} for 6") {
		t.Error("scope override not honored")
	}
}

func TestRequiredFor(t *testing.T) {
	annotation := strings.Replace(miniAnnotation,
		"team: {entity: T, membership: lone, invariant: u-team}",
		"team: {entity: T, membership: lone, invariant: u-team, required_for: [B]}", 1)
	als, _, err := gen(t, miniDomain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "fact RequiredTeams") || !strings.Contains(als, "u.role in B implies some u.team") {
		t.Errorf("required_for fact missing:\n%s", als)
	}
}

func TestMembershipOne(t *testing.T) {
	annotation := strings.Replace(miniAnnotation, "membership: lone", "membership: one", 1)
	als, _, err := gen(t, miniDomain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "team: one T") {
		t.Error("membership one not emitted")
	}
}

// --- reconciliation error paths: a drifted or malformed annotation dies ---

func TestErrors(t *testing.T) {
	cases := []struct {
		name       string
		domain     string
		annotation string
		wantErr    string
	}{
		{"unknown root key", miniDomain,
			miniAnnotation + "bogus: 1\n",
			"unsupported key 'bogus'"},
		{"subject not entity", miniDomain,
			strings.Replace(miniAnnotation, "entity: U", "entity: Nope", 1),
			"'Nope' is not a Modelith entity"},
		{"role attr missing", miniDomain,
			strings.Replace(miniAnnotation, "role_attr: role", "role_attr: nope", 1),
			"'nope' is not an attribute"},
		{"role attr not enum", miniDomain,
			strings.Replace(miniAnnotation, "role_attr: role", "role_attr: nick", 1),
			"not an enum"},
		{"membership missing", miniDomain,
			strings.Replace(miniAnnotation, ", membership: lone", "", 1),
			"membership must be 'lone' or 'one'"},
		{"required_for with one", miniDomain,
			strings.Replace(miniAnnotation, "membership: lone", "membership: one, required_for: [B]", 1),
			"redundant with membership 'one'"},
		{"resource not owned", miniDomain,
			strings.Replace(miniAnnotation, "resources: [Rec]", "resources: [Loose]", 1),
			"no n:1 relationship"},
		{"resource is subject", miniDomain,
			strings.Replace(miniAnnotation, "resources: [Rec]", "resources: [U]", 1),
			"is the subject or team entity"},
		{"unknown verb", miniDomain,
			strings.Replace(miniAnnotation, "verbs: [read]", "verbs: [peek]", 1),
			"cannot carry a scope"},
		{"create cannot be scoped", miniDomain,
			strings.Replace(miniAnnotation, "verbs: [read]", "verbs: [create]", 1),
			"cannot carry a scope"},
		{"duplicate verb coverage", miniDomain,
			strings.Replace(miniAnnotation, "verbs: [update, delete]", "verbs: [read]", 1),
			"already scoped"},
		{"unknown scope term", miniDomain,
			strings.Replace(miniAnnotation, "scope: {A: all, \"*\": own | team}", "scope: {A: galaxy}", 1),
			"unknown scope term"},
		{"all in union", miniDomain,
			strings.Replace(miniAnnotation, "\"*\": own | team", "\"*\": own | all", 1),
			"must stand alone"},
		{"unknown role", miniDomain,
			strings.Replace(miniAnnotation, "scope: {A: all,", "scope: {Z: all,", 1),
			"not a value of the role enum"},
		{"uncovered invariant", miniDomain,
			strings.Replace(miniAnnotation, "  - invariant: top-write\n    verbs: [update, delete]\n    scope: {A: all, B: own, C: none}\n", "", 1),
			"neither compiled by a rule nor waived"},
		{"unknown invariant id", miniDomain,
			strings.Replace(miniAnnotation, "invariant: top-read", "invariant: made-up", 1),
			"which the domain model does not declare"},
		{"double claim", miniDomain,
			strings.Replace(miniAnnotation, "invariant: top-write", "invariant: top-read", 1),
			"claimed twice"},
		{"residual needs reason", miniDomain,
			strings.Replace(miniAnnotation, "  - invariant: top-write\n    verbs: [update, delete]\n    scope: {A: all, B: own, C: none}\n", "", 1) +
				"residuals:\n  - {invariant: top-write}\n",
			"needs both an invariant id and a reason"},
		{"scope bound", miniDomain,
			miniAnnotation + "scope: 40\n",
			"between 2 and 12"},
		{"team term without team", strings.Replace(miniDomain, "  T:\n    relationships:\n      - {entity: U, cardinality: \"1:n\"}\n", "", 1),
			strings.Replace(strings.Replace(miniAnnotation,
				"  team: {entity: T, membership: lone, invariant: u-team}\n", "", 1),
				"own | team", "own", 1),
			""}, // valid: exercised for no-crash; team removed cleanly
		{"no rules", miniDomain,
			strings.Split(miniAnnotation, "rules:")[0] + "rules: []\n",
			"declares no rules"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := gen(t, c.domain, c.annotation)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got none", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantErr)
			}
		})
	}
}

func TestReassignErrors(t *testing.T) {
	base := miniAnnotation + `  - invariant: u-team
    reassign:
      scope: {A: all}
      target: {A: any}
`
	// u-team is already claimed by subjects.team.invariant -> double claim
	if _, _, err := gen(t, miniDomain, base); err == nil || !strings.Contains(err.Error(), "claimed twice") {
		t.Errorf("want double-claim error, got %v", err)
	}

	ann := strings.Replace(miniAnnotation, ", invariant: u-team", "", 1)
	valid := ann + `  - invariant: u-team
    reassign:
      scope: {A: all, B: team}
      target: {A: any, B: team}
`
	als, _, err := gen(t, miniDomain, valid)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "canReassign") || !strings.Contains(als, "check ReassignRetainsAuthority") {
		t.Error("reassign artifacts missing")
	}

	noTarget := ann + `  - invariant: u-team
    reassign:
      scope: {A: all}
`
	if _, _, err := gen(t, miniDomain, noTarget); err == nil || !strings.Contains(err.Error(), "reassign.target is required") {
		t.Errorf("want missing-target error, got %v", err)
	}

	undecided := ann + `  - invariant: u-team
    reassign:
      scope: {A: all, B: team}
      target: {A: any}
`
	if _, _, err := gen(t, miniDomain, undecided); err == nil || !strings.Contains(err.Error(), "is undecided") {
		t.Errorf("want undecided-target error, got %v", err)
	}

	badTarget := ann + `  - invariant: u-team
    reassign:
      scope: {A: all}
      target: {A: sideways}
`
	if _, _, err := gen(t, miniDomain, badTarget); err == nil || !strings.Contains(err.Error(), "'any' or 'team'") {
		t.Errorf("want bad-target error, got %v", err)
	}
}

func TestPaths(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Paths(dir); err == nil {
		t.Error("want error for missing model")
	}
	write(t, dir, "a.modelith.yaml", "x: 1")
	dm, ann, err := Paths(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dm) != "a.modelith.yaml" || filepath.Base(ann) != AnnotationName {
		t.Errorf("paths = %s, %s", dm, ann)
	}
	write(t, dir, "b.modelith.yaml", "x: 1")
	if _, _, err := Paths(dir); err == nil || !strings.Contains(err.Error(), "multiple modelith models") {
		t.Errorf("want multiple-models error, got %v", err)
	}
}

func TestRunEndToEnd(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "domain.modelith.yaml", miniDomain)
	if err := os.MkdirAll(filepath.Join(dir, "formal"), 0o755); err != nil {
		t.Fatal(err)
	}
	// annotation absent: opt-in error
	if err := Run(dir, ""); err == nil || !strings.Contains(err.Error(), "opt-in") {
		t.Errorf("want opt-in error, got %v", err)
	}
	write(t, filepath.Join(dir, "formal"), AnnotationName, miniAnnotation)
	if err := Run(dir, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "formal", OutputName)); err != nil {
		t.Errorf("Policy.als not written: %v", err)
	}
}
