package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const sampleModel = `kind: DomainModel
version: v1
title: T
enums:
  Status:
    values:
      - {name: Active}
      - {name: Closed}
entities:
  DataSubject:
    attributes:
      - {name: email, type: string}
      - {name: status, type: Status}
    relationships:
      - {entity: Export, cardinality: 1:n}
    invariants:
      - {id: priv-consent, statement: "Consent required."}
      - {id: priv-retention, statement: "Retention bounded."}
  Export:
    attributes:
      - {name: name, type: string}
`

// entities in the opposite YAML order; a well-behaved projection sorts by
// stable_id, so this must hash identically to sampleModel.
const sampleModelReordered = `kind: DomainModel
version: v1
title: T
enums:
  Status:
    values:
      - {name: Active}
      - {name: Closed}
entities:
  Export:
    attributes:
      - {name: name, type: string}
  DataSubject:
    relationships:
      - {entity: Export, cardinality: 1:n}
    attributes:
      - {name: status, type: Status}
      - {name: email, type: string}
    invariants:
      - {id: priv-retention, statement: "Retention bounded."}
      - {id: priv-consent, statement: "Consent required."}
`

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func manifestWith(include []string, claim []string) *Manifest {
	m := &Manifest{}
	m.Checker.ID = "test"
	m.Projection.Include = include
	m.Coverage.Claim = claim
	m.Evidence.ProjectionOut = "checkers/test/projection.json"
	m.Evidence.EvidenceIn = "checkers/test/evidence.json"
	return m
}

func TestLoadModel(t *testing.T) {
	m, err := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entities) != 2 {
		t.Fatalf("entities: got %d want 2", len(m.Entities))
	}
	if len(m.Invariants) != 2 {
		t.Fatalf("invariants: got %d want 2", len(m.Invariants))
	}
	if len(m.Relationships) != 1 {
		t.Fatalf("relationships: got %d want 1", len(m.Relationships))
	}
	// the status attribute resolves to the Status enum lifecycle
	var ds *Entity
	for i := range m.Entities {
		if m.Entities[i].Name == "DataSubject" {
			ds = &m.Entities[i]
		}
	}
	if ds == nil || len(ds.StatusEnum) != 2 {
		t.Fatalf("DataSubject lifecycle enum not detected: %+v", ds)
	}
}

func TestLoadModelErrors(t *testing.T) {
	if _, err := LoadModel(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("expected error on missing file")
	}
	if _, err := LoadModel(writeTemp(t, "empty.yaml", "kind: DomainModel\n")); err == nil {
		t.Fatal("expected error on model with no entities")
	}
}

func TestGenerateDeterministicAndOrderIndependent(t *testing.T) {
	man := manifestWith([]string{"model", "invariants", "relationships"}, []string{"priv-*"})

	mA, _ := LoadModel(writeTemp(t, "a.modelith.yaml", sampleModel))
	mB, _ := LoadModel(writeTemp(t, "b.modelith.yaml", sampleModelReordered))

	pA, err := Generate(mA, man, "sha256:x", "v0")
	if err != nil {
		t.Fatal(err)
	}
	pB, err := Generate(mB, man, "sha256:x", "v0")
	if err != nil {
		t.Fatal(err)
	}

	hA, _ := pA.InputHash()
	hB, _ := pB.InputHash()
	if hA != hB {
		t.Fatalf("hash differs for reordered but equal models:\n A=%s\n B=%s", hA, hB)
	}

	// rendering is stable across calls
	r1, _ := pA.Render()
	r2, _ := pA.Render()
	if string(r1) != string(r2) {
		t.Fatal("Render is not deterministic")
	}

	// machinery_version does not move the binding hash
	pV, _ := Generate(mA, man, "sha256:x", "v99")
	hV, _ := pV.InputHash()
	if hV != hA {
		t.Fatal("machinery_version leaked into the binding hash")
	}

	eq, _ := ContentEqual(pA, pV)
	if !eq {
		t.Fatal("ContentEqual should ignore machinery_version")
	}
}

func TestGenerateStableIDsAndInclude(t *testing.T) {
	man := manifestWith([]string{"relationships", "model", "invariants"}, nil)
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	p, err := Generate(m, man, "sha256:x", "v0")
	if err != nil {
		t.Fatal(err)
	}
	// include is emitted in canonical order regardless of manifest order
	want := []string{"model", "invariants", "relationships"}
	if len(p.Include) != len(want) {
		t.Fatalf("include: got %v want %v", p.Include, want)
	}
	for i := range want {
		if p.Include[i] != want[i] {
			t.Fatalf("include order: got %v want %v", p.Include, want)
		}
	}
	// the relationship stable_id joins entity ids
	if len(p.Model.Relationships) != 1 || p.Model.Relationships[0].From != "entity:DataSubject" || p.Model.Relationships[0].To != "entity:Export" {
		t.Fatalf("relationship join wrong: %+v", p.Model.Relationships)
	}
}

func TestGenerateRejectsUnsupportedLayer(t *testing.T) {
	man := manifestWith([]string{"model", "machines"}, nil)
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	if _, err := Generate(m, man, "sha256:x", "v0"); err == nil {
		t.Fatal("expected error requesting an unsupported layer")
	}
}

func TestRenderRoundTripAndMirror(t *testing.T) {
	man := manifestWith([]string{"model", "invariants", "relationships"}, nil)
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	p, _ := Generate(m, man, "sha256:x", "v0")
	rendered, _ := p.Render()

	back, err := ParseProjection(rendered)
	if err != nil {
		t.Fatal(err)
	}
	eq, _ := ContentEqual(p, back)
	if !eq {
		t.Fatal("Render -> ParseProjection lost content")
	}
	// the generated block mirrors the input hash for adapters
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rendered, &raw); err != nil {
		t.Fatal(err)
	}
	var gen map[string]string
	if err := json.Unmarshal(raw["generated"], &gen); err != nil {
		t.Fatal(err)
	}
	h, _ := p.InputHash()
	if gen["input_hash"] != h {
		t.Fatalf("generated.input_hash mirror wrong: %s vs %s", gen["input_hash"], h)
	}
}

func TestLoadManifestValidation(t *testing.T) {
	ok := `checker: {id: c}
projection: {include: [model]}
evidence: {projection_out: p, evidence_in: e}
`
	if _, err := LoadManifest(writeTemp(t, "a.checker.yaml", ok)); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	for name, body := range map[string]string{
		"no-id":       "projection: {include: [model]}\nevidence: {projection_out: p, evidence_in: e}\n",
		"no-include":  "checker: {id: c}\nevidence: {projection_out: p, evidence_in: e}\n",
		"no-evidence": "checker: {id: c}\nprojection: {include: [model]}\n",
	} {
		if _, err := LoadManifest(writeTemp(t, "bad.checker.yaml", body)); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func TestLoadEvidenceRejectsBadVerdict(t *testing.T) {
	body := `{"evidence_schema":"1.0","checker":{"id":"c","version":"1"},"input_hash":"sha256:x","verdict":"maybe","coverage":[]}`
	if _, err := LoadEvidence(writeTemp(t, "e.json", body)); err == nil {
		t.Fatal("expected error on unknown verdict token")
	}
}
