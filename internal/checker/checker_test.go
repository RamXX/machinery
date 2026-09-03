package checker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const checkerTestRuntime = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

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

var validTestDesignID = "sha256:" + strings.Repeat("a", 64)

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

func TestLoadModelRejectsMalformedProjectionRows(t *testing.T) {
	for name, model := range map[string]string{
		"non-object entity":            "kind: DomainModel\nentities: {Thing: nope}\n",
		"non-object attribute":         "kind: DomainModel\nentities:\n  Thing:\n    attributes: [nope]\n",
		"empty attribute type":         "kind: DomainModel\nentities:\n  Thing:\n    attributes: [{name: value, type: ''}]\n",
		"duplicate attribute":          "kind: DomainModel\nentities:\n  Thing:\n    attributes: [{name: value, type: string}, {name: value, type: string}]\n",
		"non-object invariant":         "kind: DomainModel\nentities:\n  Thing:\n    invariants: [nope]\n",
		"empty invariant statement":    "kind: DomainModel\nentities:\n  Thing:\n    invariants: [{id: inv-one, statement: ''}]\n",
		"duplicate invariant identity": "kind: DomainModel\ninvariants: [{id: inv-one, statement: top}]\nentities:\n  Thing:\n    invariants: [{id: inv-one, statement: entity}]\n",
		"unknown relationship target":  "kind: DomainModel\nentities:\n  Thing:\n    relationships: [{entity: Missing, cardinality: '1:n'}]\n",
		"unknown cardinality":          "kind: DomainModel\nentities:\n  Thing:\n    relationships: [{entity: Other, cardinality: many}]\n  Other: {}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadModel(writeTemp(t, "bad.modelith.yaml", model)); err == nil {
				t.Fatal("malformed projection source was accepted")
			}
		})
	}
}

func TestGenerateDeterministicAndOrderIndependent(t *testing.T) {
	man := manifestWith([]string{"model", "invariants", "relationships"}, []string{"priv-*"})

	mA, _ := LoadModel(writeTemp(t, "a.modelith.yaml", sampleModel))
	mB, _ := LoadModel(writeTemp(t, "b.modelith.yaml", sampleModelReordered))

	pA, err := Generate(mA, man, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	pB, err := Generate(mB, man, validTestDesignID, "v0")
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
	pV, _ := Generate(mA, man, validTestDesignID, "v99")
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
	p, err := Generate(m, man, validTestDesignID, "v0")
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
	if _, err := Generate(m, man, validTestDesignID, "v0"); err == nil {
		t.Fatal("expected error requesting an unsupported layer")
	}
}

func TestGenerateRejectsUnknownAndEmptyLayerVocabulary(t *testing.T) {
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	for _, include := range [][]string{nil, {"model", "future-layer"}} {
		if _, err := Generate(m, manifestWith(include, nil), validTestDesignID, "v0"); err == nil {
			t.Fatalf("include %v: expected closed/non-empty vocabulary error", include)
		}
	}
}

func TestInputHashBindsCanonicalManifestSemanticsIncludingConfig(t *testing.T) {
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	base := manifestWith([]string{"model", "invariants"}, []string{"priv-*"})
	base.Config = map[string]any{"sinks": []any{"Export"}, "nested": map[string]any{"enabled": true}}
	p1, err := Generate(m, base, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	h1, _ := p1.InputHash()

	changed := *base
	changed.Config = map[string]any{"sinks": []any{"Export", "Archive"}, "nested": map[string]any{"enabled": true}}
	p2, err := Generate(m, &changed, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := p2.InputHash()
	if h1 == h2 {
		t.Fatal("config mutation did not invalidate projection input_hash")
	}

	reordered := *base
	reordered.Projection.Include = []string{"invariants", "model"}
	p3, err := Generate(m, &reordered, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	h3, _ := p3.InputHash()
	if h1 != h3 {
		t.Fatalf("set-like include reordering moved input_hash: %s != %s", h1, h3)
	}
}

func TestParallelRelationshipsRequireDistinctStableIdentity(t *testing.T) {
	model := strings.Replace(sampleModel,
		"      - {entity: Export, cardinality: 1:n}",
		"      - {entity: Export, cardinality: 1:n}\n      - {entity: Export, cardinality: 1:n}", 1)
	m, _ := LoadModel(writeTemp(t, "collision.modelith.yaml", model))
	if _, err := Generate(m, manifestWith([]string{"relationships"}, nil), validTestDesignID, "v0"); err == nil || !strings.Contains(err.Error(), "stable identity collision") {
		t.Fatalf("expected hard relationship collision, got %v", err)
	}

	model = strings.Replace(model,
		"      - {entity: Export, cardinality: 1:n}\n      - {entity: Export, cardinality: 1:n}",
		"      - {entity: Export, cardinality: 1:n, role: primary}\n      - {entity: Export, cardinality: 1:n, role: archive}", 1)
	m, _ = LoadModel(writeTemp(t, "distinct.modelith.yaml", model))
	p, err := Generate(m, manifestWith([]string{"relationships"}, nil), validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Model.Relationships) != 2 || p.Model.Relationships[0].StableID == p.Model.Relationships[1].StableID {
		t.Fatalf("parallel roles did not produce distinct identities: %+v", p.Model.Relationships)
	}
}

func TestRenderRoundTripAndMirror(t *testing.T) {
	man := manifestWith([]string{"model", "invariants", "relationships"}, nil)
	m, _ := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	p, _ := Generate(m, man, validTestDesignID, "v0")
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
	ok := "checker: {id: c, runtime_closure: " + checkerTestRuntime + "}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n"
	if _, err := LoadManifest(writeTemp(t, "a.checker.yaml", ok)); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
	for name, body := range map[string]string{
		"no-id":       "projection: {include: [model]}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n",
		"no-include":  "checker: {id: c}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n",
		"no-evidence": "checker: {id: c}\nprojection: {include: [model]}\n",
	} {
		if _, err := LoadManifest(writeTemp(t, "bad.checker.yaml", body)); err == nil {
			t.Fatalf("%s: expected validation error", name)
		}
	}
}

func TestLoadManifestUsesClosedUnambiguousYAML(t *testing.T) {
	base := "checker: {id: c, runtime_closure: " + checkerTestRuntime + "}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n"
	for name, body := range map[string]string{
		"unknown top field":    base + "cheker: {id: typo}\n",
		"unknown nested field": strings.Replace(base, "id: c", "id: c, descriptin: typo", 1),
		"duplicate field":      strings.Replace(base, "checker: {id: c,", "checker: {id: c, id: other,", 1),
		"non scalar key":       "? [checker]\n: {id: c}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n",
		"multiple documents":   base + "---\nchecker: {id: other}\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadManifest(writeTemp(t, "bad.checker.yaml", body)); err == nil {
				t.Fatal("ambiguous or open manifest was accepted")
			}
		})
	}
}

func TestParseProjectionStrictContractAndSupportedCombinations(t *testing.T) {
	m, err := LoadModel(writeTemp(t, "d.modelith.yaml", sampleModel))
	if err != nil {
		t.Fatal(err)
	}
	validDesignID := "sha256:" + strings.Repeat("a", 64)
	for _, include := range [][]string{{"model"}, {"invariants"}, {"relationships"}, {"model", "invariants", "relationships"}} {
		p, err := Generate(m, manifestWith(include, nil), validDesignID, "v-test")
		if err != nil {
			t.Fatalf("include %v: %v", include, err)
		}
		rendered, err := p.Render()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseProjection(rendered)
		if err != nil {
			t.Fatalf("include %v did not conform to parser/schema: %v\n%s", include, err, rendered)
		}
		if parsed.Model.Entities == nil {
			t.Fatalf("include %v rendered model.entities as null", include)
		}
	}

	p, _ := Generate(m, manifestWith([]string{"model"}, nil), validDesignID, "v-test")
	rendered, _ := p.Render()
	base := strings.TrimSpace(string(rendered))
	for name, raw := range map[string]string{
		"unknown field":     strings.Replace(base, `"projection_schema": "1.0",`, `"projection_schema": "1.0", "mystery": true,`, 1),
		"duplicate field":   strings.Replace(base, `"checker_id":`, `"checker_id": "duplicate", "checker_id":`, 1),
		"trailing document": base + "\n{}",
		"null entities":     strings.Replace(base, `"entities": [`, `"entities": null, "discarded": [`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseProjection([]byte(raw)); err == nil {
				t.Fatal("invalid projection was accepted")
			}
		})
	}
}

func TestLoadEvidenceStrictContract(t *testing.T) {
	valid := `{"evidence_schema":"1.0","checker":{"id":"c","version":"1"},"input_hash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","runtime_closure":"sha256:1111111111111111111111111111111111111111111111111111111111111111","verdict":"pass","coverage":[{"element":"inv:x","verdict":"pass"}]}`
	for name, raw := range map[string]string{
		"unknown field":         strings.Replace(valid, `"verdict":"pass"`, `"unknown":true,"verdict":"pass"`, 1),
		"duplicate field":       strings.Replace(valid, `"checker":{"id":"c"`, `"checker":{"id":"c","id":"d"`, 1),
		"trailing document":     valid + `{}`,
		"wrong schema":          strings.Replace(valid, `"1.0"`, `"2.0"`, 1),
		"empty checker id":      strings.Replace(valid, `"id":"c"`, `"id":""`, 1),
		"empty version":         strings.Replace(valid, `"version":"1"`, `"version":""`, 1),
		"bad hash":              strings.Replace(valid, `sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`, `sha256:ABC`, 1),
		"null coverage":         strings.Replace(valid, `[{"element":"inv:x","verdict":"pass"}]`, `null`, 1),
		"duplicate coverage":    strings.Replace(valid, `[{"element":"inv:x","verdict":"pass"}]`, `[{"element":"inv:x","verdict":"pass"},{"element":"inv:x","verdict":"fail"}]`, 1),
		"bad finding severity":  strings.Replace(valid, `"coverage":[`, `"findings":[{"severity":"warning","message":"x"}],"coverage":[`, 1),
		"empty finding message": strings.Replace(valid, `"coverage":[`, `"findings":[{"severity":"info","message":""}],"coverage":[`, 1),
		"bad signature":         strings.Replace(valid, `"coverage":[`, `"input_signature":{"scheme":"rsa","value":"x"},"coverage":[`, 1),
		"nonobject generated":   strings.Replace(valid, `"coverage":[`, `"generated":[],"coverage":[`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := LoadEvidence(writeTemp(t, "bad.json", raw)); err == nil {
				t.Fatal("invalid evidence was accepted")
			}
		})
	}
}

func TestLoadManifestRejectsEscapingAndSymlinkedEvidencePaths(t *testing.T) {
	design := t.TempDir()
	checkers := filepath.Join(design, "checkers")
	if err := os.MkdirAll(checkers, 0o755); err != nil {
		t.Fatal(err)
	}
	base := func(proj string) string {
		return "checker: {id: c, runtime_closure: " + checkerTestRuntime + "}\nprojection: {include: [model]}\nevidence: {projection_out: " + proj + ", evidence_in: checkers/c/evidence.json}\n"
	}
	for name, rel := range map[string]string{"absolute": filepath.Join(design, "out.json"), "escape": "../out.json"} {
		p := filepath.Join(checkers, name+".checker.yaml")
		if err := os.WriteFile(p, []byte(base(rel)), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadManifest(p); err == nil {
			t.Fatalf("%s path accepted", name)
		}
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(checkers, "c")); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(checkers, "symlink.checker.yaml")
	if err := os.WriteFile(p, []byte(base("checkers/c/projection.json")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(p); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("symlink component accepted: %v", err)
	}
}

func TestConfinedPathRejectsHostIndependentNonportablePaths(t *testing.T) {
	for _, rel := range []string{
		`C:\temp\projection.json`,
		`\\server\share\evidence.json`,
		"checkers/NUL.json",
		"checkers/con.txt",
		"checkers/COM1.proof",
		"checkers/name./projection.json",
		"checkers/name /projection.json",
		"checkers/café/projection.json",
	} {
		if _, err := ConfinedPath(t.TempDir(), rel); err == nil {
			t.Errorf("nonportable path %q was accepted", rel)
		}
	}
}

func TestLoadManifestRequiresCheckerOwnedOutputDirectory(t *testing.T) {
	body := "checker: {id: c, runtime_closure: " + checkerTestRuntime + "}\nprojection: {include: [model]}\nevidence: {projection_out: projection.json, evidence_in: evidence.json}\n"
	if _, err := LoadManifest(writeTemp(t, "c.checker.yaml", body)); err == nil || !strings.Contains(err.Error(), "owned under checkers/c/") {
		t.Fatalf("root-level checker outputs were accepted: %v", err)
	}
}

func TestLoadManifestRejectsNonportableCheckerID(t *testing.T) {
	for _, id := range []string{"café", "NUL", "name ", "a/b"} {
		body := "checker: {id: '" + id + "'}\nprojection: {include: [model]}\nevidence: {projection_out: checkers/c/projection.json, evidence_in: checkers/c/evidence.json}\n"
		if _, err := LoadManifest(writeTemp(t, "bad.checker.yaml", body)); err == nil {
			t.Errorf("nonportable checker id %q was accepted", id)
		}
	}
}

func TestLoadEvidenceRejectsBadVerdict(t *testing.T) {
	body := `{"evidence_schema":"1.0","checker":{"id":"c","version":"1"},"input_hash":"sha256:x","verdict":"maybe","coverage":[]}`
	if _, err := LoadEvidence(writeTemp(t, "e.json", body)); err == nil {
		t.Fatal("expected error on unknown verdict token")
	}
}

func TestReadConfinedFileBoundedRejectsOversizedArtifact(t *testing.T) {
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "trace.bin"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfinedFileBounded(design, "trace.bin", 4); err == nil || !strings.Contains(err.Error(), "exceeds 4-byte limit") {
		t.Fatalf("oversized confined artifact diagnostic = %v", err)
	}
	got, err := ReadConfinedFileBounded(design, "trace.bin", 5)
	if err != nil || string(got) != "12345" {
		t.Fatalf("bounded confined artifact = %q, %v", got, err)
	}
}

func TestStructuredCheckerReadersRejectOversizedFilesBeforeParsing(t *testing.T) {
	tests := []struct {
		name string
		load func(string) error
	}{
		{"model", func(path string) error { _, err := LoadModel(path); return err }},
		{"manifest", func(path string) error { _, err := LoadManifest(path); return err }},
		{"evidence", func(path string) error { _, err := LoadEvidence(path); return err }},
		{"registry", func(path string) error { _, err := LoadRegistry(path); return err }},
		{"design-id", func(path string) error { _, err := DesignID(path); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "oversized")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(checkerStructuredFileMaxBytes + 1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			if err := test.load(path); err == nil || !strings.Contains(err.Error(), "byte limit") {
				t.Fatalf("oversized %s diagnostic = %v", test.name, err)
			}
		})
	}

	design := t.TempDir()
	path := filepath.Join(design, "projection.json")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(checkerStructuredFileMaxBytes + 1); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfinedFile(design, "projection.json"); err == nil || !strings.Contains(err.Error(), "byte limit") {
		t.Fatalf("oversized confined projection diagnostic = %v", err)
	}
}

func TestCheckerDirectoryReadersRejectEntryOverflow(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := readDirectoryBounded(dir, 2); err == nil || !strings.Contains(err.Error(), "2-entry limit") {
		t.Fatalf("direct directory overflow = %v", err)
	}
	root, err := openDesignRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.close()
	if _, err := root.readDirBounded(".", 2); err == nil || !strings.Contains(err.Error(), "2-entry limit") {
		t.Fatalf("rooted directory overflow = %v", err)
	}
}
