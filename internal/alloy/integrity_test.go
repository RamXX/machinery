package alloy

import (
	"path/filepath"
	"strings"
	"testing"
)

// --- real fixtures ---

func TestGenerateIntegrityGoCRM(t *testing.T) {
	als, stats, err := GenerateIntegrity(
		filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml"),
		filepath.Join(repoRoot(), "examples/go-crm/design/formal/integrity.relational.yaml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entities != 4 || stats.Uniques != 3 || stats.Singletons != 1 || stats.Relationships != 0 {
		t.Errorf("stats = %+v; want 4 entities, 3 uniques, 1 singleton, 0 rels", stats)
	}
	if stats.Carried != 4 {
		t.Errorf("carried = %d; want 4", stats.Carried)
	}
	for _, want := range []string{
		"module Integrity",
		"sig User {",
		"username: one Val_User_Username",
		"abstract sig Bool {}",
		"one sig True, False extends Bool {}",
		"fact Unique_User_Username {",
		"all disj x, y: User | x.username != y.username",
		"fact Singleton_Pipeline_IsDefault {",
		"one x: Pipeline | x.isDefault = True",
		"run SomeWorld {",
		"run Populatable {",
		"run Distinct_User_Username {",
		"DO NOT EDIT",
	} {
		if !strings.Contains(als, want) {
			t.Errorf("generated integrity model missing %q", want)
		}
	}
}

func TestGenerateIntegrityFulfillment(t *testing.T) {
	als, stats, err := GenerateIntegrity(
		filepath.Join(repoRoot(), "examples/fulfillment/design/fulfillment.modelith.yaml"),
		filepath.Join(repoRoot(), "examples/fulfillment/design/formal/integrity.relational.yaml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Entities != 3 || stats.Relationships != 2 || stats.Uniques != 1 {
		t.Errorf("stats = %+v; want 3 entities, 2 rels, 1 unique", stats)
	}
	// n:1 mandatory -> one; carried invariant on the customer edge
	if !strings.Contains(als, "customer: one Customer") {
		t.Errorf("mandatory n:1 not rendered as 'one':\n%s", als)
	}
	// 1:1 mandatory -> one, structural (no invariant)
	if !strings.Contains(als, "payment: one Payment") {
		t.Errorf("mandatory 1:1 not rendered as 'one':\n%s", als)
	}
	if !strings.Contains(als, "fact Cardinality_Order_Payment {") ||
		!strings.Contains(als, "all target: Payment | lone target.~payment") ||
		!strings.Contains(als, "check Exclusive_Order_Payment") {
		t.Errorf("1:1 inverse cardinality is not enforced and checked:\n%s", als)
	}
	if !strings.Contains(als, "fact Unique_Customer_Email {") {
		t.Error("email uniqueness fact missing")
	}
}

func TestIntegrityDeterminism(t *testing.T) {
	dm := filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml")
	an := filepath.Join(repoRoot(), "examples/go-crm/design/formal/integrity.relational.yaml")
	a1, _, err := GenerateIntegrity(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	a2, _, err := GenerateIntegrity(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 {
		t.Error("integrity generation is not deterministic")
	}
}

// --- synthetic fixture exercising every multiplicity ---

const intDomain = `
entities:
  A:
    attributes:
      - {name: code, type: string}
      - {name: flag, type: boolean}
    invariants:
      - {id: a-code-unique, statement: s}
      - {id: a-one-flag, statement: s}
  B:
    relationships:
      - {entity: A, cardinality: "n:1"}
      - {entity: C, cardinality: "1:1"}
    invariants:
      - {id: b-owned, statement: s}
  C:
    relationships:
      - {entity: A, cardinality: "1:n"}
      - {entity: B, cardinality: "n:n"}
`

const intAnnotation = `
entities: [A, B, C]
relationships:
  - {from: B, to: A, field: owner, mandatory: true, invariant: b-owned}
  - {from: B, to: C, mandatory: false}
  - {from: C, to: A, mandatory: true}
  - {from: C, to: B, mandatory: false}
unique:
  - {entity: A, attribute: code, invariant: a-code-unique}
singleton:
  - {entity: A, flag: flag, invariant: a-one-flag}
`

func genInt(t *testing.T, domain, annotation string) (string, IntegrityStats, error) {
	t.Helper()
	dir := t.TempDir()
	dm := write(t, dir, "domain.modelith.yaml", domain)
	an := write(t, dir, "integrity.relational.yaml", annotation)
	return GenerateIntegrity(dm, an)
}

func TestIntegrityMultiplicities(t *testing.T) {
	als, stats, err := genInt(t, intDomain, intAnnotation)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Relationships != 4 || stats.Uniques != 1 || stats.Singletons != 1 {
		t.Errorf("stats = %+v", stats)
	}
	for _, want := range []string{
		"owner: one A",           // n:1 mandatory
		"c: lone C",              // 1:1 optional -> lone (default field name = lowercased To)
		"a: some A",              // 1:n mandatory -> some
		"b: set B",               // n:n optional -> set
		"fact Cardinality_B_C {", // 1:1 inverse exclusivity
		"all target: C | lone target.~c",
		"check Exclusive_B_C",
		"fact Cardinality_C_A {", // 1:n inverse exclusivity
		"all target: A | lone target.~a",
		"check Exclusive_C_A",
	} {
		if !strings.Contains(als, want) {
			t.Errorf("multiplicity rendering missing %q:\n%s", want, als)
		}
	}
}

// --- reconciliation error paths ---

func TestIntegrityErrors(t *testing.T) {
	cases := []struct {
		name       string
		domain     string
		annotation string
		wantErr    string
	}{
		{"unknown root key", intDomain, intAnnotation + "bogus: 1\n", "unsupported key 'bogus'"},
		{"entity not in model", intDomain,
			strings.Replace(intAnnotation, "entities: [A, B, C]", "entities: [A, B, Nope]", 1),
			"not a Modelith entity"},
		{"rel from not modeled", intDomain,
			strings.Replace(intAnnotation, "entities: [A, B, C]", "entities: [A, C]", 1),
			"not in the entities list"},
		{"rel has no domain edge", intDomain,
			strings.Replace(intAnnotation, "{from: B, to: A, field: owner, mandatory: true, invariant: b-owned}", "{from: A, to: B, invariant: b-owned}", 1),
			"declares no relationship from A to B"},
		{"unique attr missing", intDomain,
			strings.Replace(intAnnotation, "attribute: code", "attribute: nope", 1),
			"is not an attribute of A"},
		{"singleton not boolean", intDomain,
			strings.Replace(intAnnotation, "flag: flag", "flag: code", 1),
			"not boolean"},
		{"unknown invariant", intDomain,
			strings.Replace(intAnnotation, "invariant: a-code-unique", "invariant: made-up", 1),
			"does not declare"},
		{"double claim", intDomain,
			strings.Replace(intAnnotation, "invariant: a-code-unique", "invariant: b-owned", 1),
			"claimed twice"},
		{"no constraints", intDomain,
			"entities: [A]\n",
			"compiles no constraints"},
		{"scope out of range", intDomain, intAnnotation + "scope: 99\n", "between 3 and 12"},
		{"scope too small for population witness", intDomain, intAnnotation + "scope: 2\n", "between 3 and 12"},
		{"residual needs reason", intDomain,
			intAnnotation + "residuals:\n  - {invariant: a-code-unique}\n",
			"needs both an invariant id and a reason"},
		{"residual double claim", intDomain,
			intAnnotation + "residuals:\n  - {invariant: a-code-unique, reason: x}\n",
			"claimed twice"}, // a-code-unique already claimed by unique
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, err := genInt(t, c.domain, c.annotation)
			if err == nil {
				t.Fatalf("expected error containing %q, got none", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantErr)
			}
		})
	}
}

// TestIntegrityUnsatisfiable is the mutation check: a singleton flag on an
// entity whose uniqueness would require two distinct records is still
// satisfiable (two records, one flagged), but forcing the entity to a single
// atom via a contradictory annotation must surface as a model whose SomeWorld
// could not hold. Here we prove the generator emits the joint-satisfiability
// run so the solver can catch such contradictions.
func TestIntegritySatisfiabilityRunPresent(t *testing.T) {
	als, _, err := genInt(t, intDomain, intAnnotation)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "run SomeWorld {") || !strings.Contains(als, "run Populatable {") {
		t.Error("admissibility runs missing; the solver has nothing to falsify")
	}
	// Populatable must require a population target of every modeled entity
	for _, e := range []string{"A", "B", "C"} {
		if !strings.Contains(als, "#"+e+" >= 3") {
			t.Errorf("Populatable does not require a population target for %s", e)
		}
	}
}

// TestIntegrityBoundedDomain locks the mutation mechanism: a uniqueness
// constraint on a boolean attribute produces a value domain of exactly two
// atoms, which with the Populatable target of 3 is UNSAT at the solver -- the
// integrity analog of the policy layer's teamless-Manager catch. Verified
// against the live solver during development; here we lock the generated shape.
func TestIntegrityBoundedDomain(t *testing.T) {
	domain := `
entities:
  Widget:
    attributes:
      - {name: enabled, type: boolean}
    invariants:
      - {id: w-unique, statement: s}
`
	annotation := `
entities: [Widget]
unique:
  - {entity: Widget, attribute: enabled, invariant: w-unique}
`
	als, _, err := genInt(t, domain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "one sig Val_Widget_Enabled_0, Val_Widget_Enabled_1 extends Val_Widget_Enabled {}") {
		t.Errorf("boolean unique domain not bounded to two atoms:\n%s", als)
	}
	if !strings.Contains(als, "#Widget >= 3") {
		t.Error("Populatable target should exceed the bounded domain so starvation surfaces")
	}
}

// TestIntegrityEnumBoundedDomain: a unique enum attribute is bounded to the
// enum's value count.
func TestIntegrityEnumBoundedDomain(t *testing.T) {
	domain := `
enums:
  Color:
    values:
      - {name: Red, definition: r}
      - {name: Green, definition: g}
      - {name: Blue, definition: b}
entities:
  Thing:
    attributes:
      - {name: color, type: Color}
    invariants:
      - {id: t-unique, statement: s}
`
	annotation := `
entities: [Thing]
unique:
  - {entity: Thing, attribute: color, invariant: t-unique}
`
	als, _, err := genInt(t, domain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(als, "Val_Thing_Color_2 extends Val_Thing_Color {} // bounded: 3 value(s)") {
		t.Errorf("enum unique domain not bounded to three atoms:\n%s", als)
	}
}

func TestCarriedIntegrityIDs(t *testing.T) {
	ids := CarriedIntegrityIDs(filepath.Join(repoRoot(), "examples/go-crm/design/formal/integrity.relational.yaml"))
	for _, want := range []string{"username-unique", "team-name-unique", "tag-name-unique", "one-default-pipeline"} {
		if !ids[want] {
			t.Errorf("carried ids missing %q", want)
		}
	}
	if len(ids) != 4 {
		t.Errorf("carried ids = %v; want 4", ids)
	}
}
