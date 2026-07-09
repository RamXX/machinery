package alloy

import (
	"path/filepath"
	"strings"
	"testing"
)

// --- real fixture: go-crm ---

func TestGenerateIsolationGoCRM(t *testing.T) {
	als, oracle, stats, err := GenerateIsolation(
		filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml"),
		filepath.Join(repoRoot(), "examples/go-crm/design/formal/isolation.relational.yaml"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Records != 4 || stats.References != 2 || stats.Carried != 2 {
		t.Errorf("stats = %+v; want 4 records, 2 references, 2 carried", stats)
	}
	if stats.OracleRows != 8 {
		t.Errorf("oracle rows = %d; want 8 (2 references x 4 tenant cases)", stats.OracleRows)
	}
	for _, want := range []string{
		"module Isolation",
		"sig Team {}",
		"team: lone Team",
		"deal: lone Deal",
		"contact: lone Contact",
		"pred sameTenant[a, b: User] { some a.team and a.team = b.team }",
		"fact Isolation_Task_Deal_Deal {",
		"all x: Task | some x.deal implies sameTenant[x.owner, x.deal.owner]",
		"run SomeWorld {",
		"check SharedReferent_Task_Deal_Deal {",
		"check SharedReferent_Activity_Contact_Contact {",
		"run Possible_Task_Deal_Deal {",
		"an Activity", // article helper
		"DO NOT EDIT",
	} {
		if !strings.Contains(als, want) {
			t.Errorf("generated isolation model missing %q", want)
		}
	}
	for _, want := range []string{
		"tenant-scoping oracle",
		"| Task.deal -> Deal | same-tenant | allow |",
		"| Task.deal -> Deal | cross-tenant | deny |",
		"| Task.deal -> Deal | source-teamless | deny |",
		"| Activity.contact -> Contact | target-teamless | deny |",
	} {
		if !strings.Contains(oracle, want) {
			t.Errorf("oracle missing %q", want)
		}
	}
}

func TestIsolationDeterminism(t *testing.T) {
	dm := filepath.Join(repoRoot(), "examples/go-crm/design/domain.modelith.yaml")
	an := filepath.Join(repoRoot(), "examples/go-crm/design/formal/isolation.relational.yaml")
	a1, o1, _, err := GenerateIsolation(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	a2, o2, _, err := GenerateIsolation(dm, an)
	if err != nil {
		t.Fatal(err)
	}
	if a1 != a2 || o1 != o2 {
		t.Error("isolation generation is not deterministic")
	}
}

// --- synthetic fixture ---

const isoDomain = `
entities:
  Org:
    relationships:
      - {entity: Member, cardinality: "1:n"}
  Member:
    relationships:
      - {entity: Org, cardinality: "n:1"}
  Doc:
    relationships:
      - {entity: Member, cardinality: "n:1"}
      - {entity: Folder, cardinality: "n:1"}
    invariants:
      - {id: doc-folder-tenant, statement: s}
  Folder:
    relationships:
      - {entity: Member, cardinality: "n:1"}
`

const isoAnnotation = `
tenant:
  entity: Org
subject:
  entity: Member
  tenant_attr: org
  membership: lone
references:
  - {from: Doc, to: Folder, field: folder, invariant: doc-folder-tenant}
`

func genIso(t *testing.T, domain, annotation string) (string, string, IsolationStats, error) {
	t.Helper()
	dir := t.TempDir()
	dm := write(t, dir, "domain.modelith.yaml", domain)
	an := write(t, dir, "isolation.relational.yaml", annotation)
	return GenerateIsolation(dm, an)
}

func TestIsolationSynthetic(t *testing.T) {
	als, _, stats, err := genIso(t, isoDomain, isoAnnotation)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Records != 2 || stats.References != 1 {
		t.Errorf("stats = %+v; want 2 records, 1 reference", stats)
	}
	// n:1 reference -> SharedReferent check present (many Doc can share one Folder)
	if !strings.Contains(als, "check SharedReferent_Doc_Folder_Folder {") {
		t.Error("n:1 reference should get a SharedReferent check")
	}
	if !strings.Contains(als, "org: lone Org") {
		t.Errorf("tenant attr not rendered:\n%s", als)
	}
}

func TestIsolationOneToOneNoSharedReferent(t *testing.T) {
	// a 1:1 reference cannot be shared, so no SharedReferent check
	domain := strings.Replace(isoDomain, `{entity: Folder, cardinality: "n:1"}`, `{entity: Folder, cardinality: "1:1"}`, 1)
	als, _, _, err := genIso(t, domain, isoAnnotation)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(als, "SharedReferent") {
		t.Error("1:1 reference must not get a SharedReferent check (no sharing possible)")
	}
	// The isolation model carries the inverse bound itself rather than assuming
	// another relational model is present.
	if !strings.Contains(als, "fact Cardinality_Doc_Folder_Folder {") ||
		!strings.Contains(als, "all target: Folder | lone target.~folder") {
		t.Error("1:1 reference must enforce inverse exclusivity")
	}
	// but the isolation fact and Possible run still hold
	if !strings.Contains(als, "fact Isolation_Doc_Folder_Folder {") || !strings.Contains(als, "run Possible_Doc_Folder_Folder {") {
		t.Error("1:1 reference still needs its fact and non-vacuity run")
	}
}

func TestIsolationReferenceIdentityIncludesField(t *testing.T) {
	domain := strings.Replace(isoDomain,
		`      - {entity: Folder, cardinality: "n:1"}`,
		`      - {entity: Folder, cardinality: "n:1"}
      - {entity: Folder, cardinality: "n:1"}`, 1)
	domain = strings.Replace(domain,
		`      - {id: doc-folder-tenant, statement: s}`,
		`      - {id: doc-folder-tenant, statement: s}
      - {id: doc-archive-tenant, statement: s}`, 1)
	annotation := strings.Replace(isoAnnotation,
		`  - {from: Doc, to: Folder, field: folder, invariant: doc-folder-tenant}`,
		`  - {from: Doc, to: Folder, field: folder, invariant: doc-folder-tenant}
  - {from: Doc, to: Folder, field: archiveFolder, invariant: doc-archive-tenant}`, 1)

	als, oracle, stats, err := genIso(t, domain, annotation)
	if err != nil {
		t.Fatal(err)
	}
	if stats.References != 2 || stats.OracleRows != 8 {
		t.Fatalf("stats = %+v; want two distinct references and eight rows", stats)
	}
	for _, want := range []string{
		"fact Isolation_Doc_Folder_Folder {",
		"fact Isolation_Doc_Folder_ArchiveFolder {",
		"check SharedReferent_Doc_Folder_Folder {",
		"check SharedReferent_Doc_Folder_ArchiveFolder {",
		"run Possible_Doc_Folder_Folder {",
		"run Possible_Doc_Folder_ArchiveFolder {",
		"| Doc.folder -> Folder | same-tenant | allow |",
		"| Doc.archiveFolder -> Folder | same-tenant | allow |",
	} {
		if !strings.Contains(als+oracle, want) {
			t.Errorf("field-qualified output missing %q", want)
		}
	}
	seen := map[string]bool{}
	for _, line := range strings.Split(oracle, "\n") {
		if !strings.HasPrefix(line, "| O-TENANT-") {
			continue
		}
		cells := strings.Split(line, "|")
		id := strings.TrimSpace(cells[2])
		if seen[id] {
			t.Fatalf("duplicate stable id %s in oracle:\n%s", id, oracle)
		}
		seen[id] = true
	}
}

// --- reconciliation error paths ---

func TestIsolationErrors(t *testing.T) {
	cases := []struct {
		name       string
		domain     string
		annotation string
		wantErr    string
	}{
		{"unknown root key", isoDomain, isoAnnotation + "bogus: 1\n", "unsupported key 'bogus'"},
		{"tenant not entity", isoDomain,
			strings.Replace(isoAnnotation, "entity: Org", "entity: Nope", 1),
			"tenant.entity 'Nope' is not a Modelith entity"},
		{"subject not entity", isoDomain,
			strings.Replace(isoAnnotation, "entity: Member", "entity: Nope", 1),
			"is not a Modelith entity"},
		{"membership missing", isoDomain,
			strings.Replace(isoAnnotation, "\n  membership: lone", "", 1),
			"membership must be 'lone' or 'one'"},
		{"reference not owned", isoDomain,
			strings.Replace(isoAnnotation, "{from: Doc, to: Folder, field: folder, invariant: doc-folder-tenant}", "{from: Org, to: Folder, field: folder, invariant: doc-folder-tenant}", 1),
			"is the subject or tenant entity"},
		{"reference no domain edge", isoDomain,
			strings.Replace(isoAnnotation, "{from: Doc, to: Folder, field: folder, invariant: doc-folder-tenant}", "{from: Folder, to: Doc, field: doc, invariant: doc-folder-tenant}", 1),
			"declares no relationship from Folder to Doc"},
		{"reference no invariant", isoDomain,
			strings.Replace(isoAnnotation, ", invariant: doc-folder-tenant", "", 1),
			"declares no invariant id"},
		{"unknown invariant", isoDomain,
			strings.Replace(isoAnnotation, "invariant: doc-folder-tenant", "invariant: made-up", 1),
			"does not declare"},
		{"no references", isoDomain,
			strings.Split(isoAnnotation, "references:")[0] + "references: []\n",
			"declares no references"},
		{"scope out of range", isoDomain, isoAnnotation + "scope: 1\n", "between 2 and 12"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, _, err := genIso(t, c.domain, c.annotation)
			if err == nil {
				t.Fatalf("expected error containing %q, got none", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), c.wantErr)
			}
		})
	}
}

func TestCarriedIsolationIDs(t *testing.T) {
	ids := CarriedIsolationIDs(filepath.Join(repoRoot(), "examples/go-crm/design/formal/isolation.relational.yaml"))
	for _, want := range []string{"task-deal-same-tenant", "activity-contact-same-tenant"} {
		if !ids[want] {
			t.Errorf("carried ids missing %q", want)
		}
	}
	if len(ids) != 2 {
		t.Errorf("carried ids = %v; want 2", ids)
	}
}
