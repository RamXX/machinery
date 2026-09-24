package checker

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/datalog"
)

func src(line int) Source { return Source{Path: "machines/Order.machine.json", Line: line} }

func sampleFacts(t *testing.T) *DesignFacts {
	t.Helper()
	f := NewDesignFacts()
	for _, layer := range []string{"model", "invariants", "relationships", "actions", "machines"} {
		if err := f.MarkLayer(layer); err != nil {
			t.Fatal(err)
		}
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(f.Add("machine", "machine:Order", src(1), "Order"))
	must(f.Add("state", "state:Order.Paid", src(9), "Order.Paid", "Order", "final"))
	must(f.Add("state", "state:Order.Placed", src(8), "Order.Placed", "Order", "atomic"))
	must(f.Add("context_key", "machine:Order", src(4), "Order", "total"))
	must(f.Add("entity", "entity:Order", Source{Path: "d.modelith.yaml", Line: 3}, "Order"))
	return f
}

func TestDesignFactsRejectsValuesAFactCannotHold(t *testing.T) {
	f := NewDesignFacts()
	for _, bad := range []string{"a\tb", "a\nb", "a\rb"} {
		err := f.Add("entity", "entity:X", src(1), bad)
		if err == nil || !strings.Contains(err.Error(), "tab, carriage return or newline") {
			t.Fatalf("value %q must be refused, got %v", bad, err)
		}
	}
	if err := f.Add("entity", "entity:\tX", src(1), "X"); err == nil {
		t.Fatal("a stable id with a tab must be refused")
	}
	if err := f.Add("state", "state:X", src(1), "X"); err == nil || !strings.Contains(err.Error(), "3 columns") {
		t.Fatalf("an arity mismatch must be refused, got %v", err)
	}
	if err := f.Add("nope", "x:y", src(1), "y"); err == nil {
		t.Fatal("an unknown relation must be refused")
	}
	if err := f.Add("entity", "entity:X", Source{Path: "x", Line: 0}, "X"); err == nil {
		t.Fatal("a zero line must be refused")
	}
}

func TestDesignFactsDuplicateDefinitionNamesBothSources(t *testing.T) {
	f := NewDesignFacts()
	if err := f.Add("state", "state:Order.Paid", src(9), "Order.Paid", "Order", "final"); err != nil {
		t.Fatal(err)
	}
	// The same element from the same place (several tuples of one row) is fine.
	if err := f.Add("state", "state:Order.Paid", src(9), "Order.Paid", "Order", "atomic"); err != nil {
		t.Fatal(err)
	}
	err := f.Add("state", "state:Order.Paid", Source{Path: "machines/Other.machine.json", Line: 2}, "Order.Paid", "Order", "final")
	if err == nil || !strings.Contains(err.Error(), "machines/Order.machine.json:9") || !strings.Contains(err.Error(), "machines/Other.machine.json:2") {
		t.Fatalf("a duplicate definition must name both sources, got %v", err)
	}
	// A non-defining relation repeats its owner's id freely.
	for i := 0; i < 2; i++ {
		if err := f.Add("context_key", "machine:Order", src(4+i), "Order", "k"+string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	// An identical tuple is a set member once.
	if err := f.Add("context_key", "machine:Order", src(7), "Order", "ka"); err != nil {
		t.Fatal(err)
	}
	if got := len(f.Rows("context_key")); got != 2 {
		t.Fatalf("identical tuples must collapse, got %d rows", got)
	}
}

func TestFactsFilesAreTheDatalogInputFormat(t *testing.T) {
	files, err := sampleFacts(t).FactsFiles()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(files["state.facts"]); got != "Order.Paid\tOrder\tfinal\nOrder.Placed\tOrder\tatomic\n" {
		t.Fatalf("state.facts = %q", got)
	}
	if got := string(files["guard_on.facts"]); got != "" {
		t.Fatalf("an empty relation of a present layer is an empty file, got %q", got)
	}
	if _, ok := files["unit.facts"]; ok {
		t.Fatal("an absent layer writes no files")
	}
	for name, body := range files {
		if name == RelationsIndexFile {
			continue
		}
		rows, err := datalog.ParseFacts(string(body), name)
		if err != nil {
			t.Fatal(err)
		}
		spec := relationByName[strings.TrimSuffix(name, ".facts")]
		for _, row := range rows {
			if len(row) != len(spec.Columns) {
				t.Fatalf("%s: row %v has %d columns, want %d", name, row, len(row), len(spec.Columns))
			}
		}
	}
	index := string(files[RelationsIndexFile])
	if !strings.HasPrefix(index, "entity\t1\tmodel\tid\n") || !strings.Contains(index, "transition\t4\tmachines\tid,src,event,dst\n") {
		t.Fatalf("relations.txt = %q", index)
	}
}

func TestRelationCatalogNamesAreValidDatalog(t *testing.T) {
	var b strings.Builder
	for _, spec := range RelationCatalog() {
		b.WriteString(".decl " + spec.Name + "(")
		for i, c := range spec.Columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(c + ":symbol")
		}
		b.WriteString(")\n.input " + spec.Name + "\n")
	}
	if _, err := datalog.Parse(b.String(), "catalog.dl"); err != nil {
		t.Fatalf("every relation and column name must be legal in the Souffle subset: %v", err)
	}
}

func TestWriteFactsDirReplacesOnlyFactsOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "facts dir")
	facts := sampleFacts(t)
	if err := WriteFactsDir(dir, facts); err != nil {
		t.Fatal(err)
	}
	first := readDirBytes(t, dir)
	if err := WriteFactsDir(dir, facts); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, readDirBytes(t, dir)) {
		t.Fatal("a second write of the same facts must be byte-identical")
	}
	entries, err := os.ReadDir(filepath.Dir(dir))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("no staging directory may survive a successful write: %v", entries)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFactsDir(dir, facts); err == nil || !strings.Contains(err.Error(), "notes.md") {
		t.Fatalf("a directory holding anything but facts output must be refused, got %v", err)
	}
	file := filepath.Join(t.TempDir(), "plain")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFactsDir(file, facts); err == nil {
		t.Fatal("a regular file must never be replaced")
	}
}

func readDirBytes(t *testing.T, dir string) []byte {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	for _, e := range entries {
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString("== " + e.Name() + "\n")
		b.Write(body)
	}
	return b.Bytes()
}

func TestV2ProjectionRoundTripsAndValidates(t *testing.T) {
	model, err := LoadModel(writeTemp(t, "v2.modelith.yaml", sampleModel))
	if err != nil {
		t.Fatal(err)
	}
	facts := sampleFacts(t)
	proj, err := GenerateWithFacts(model, facts, manifestWith([]string{"machines", "model"}, nil), validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	if proj.ProjectionSchema != ProjectionSchemaV2 || proj.Model == nil {
		t.Fatalf("a v2 include with model keeps the v1 model block under 2.0: %+v", proj)
	}
	rendered, err := proj.Render()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"projection_schema": "2.0"`, `"layers": {`, `"stable_id": "state:Order.Paid"`, `"source": {`, `"kind": "final"`} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("rendered v2 projection lacks %s:\n%s", want, rendered)
		}
	}
	parsed, err := ParseProjection(rendered)
	if err != nil {
		t.Fatal(err)
	}
	eq, err := ContentEqual(parsed, proj)
	if err != nil || !eq {
		t.Fatalf("a parsed v2 projection must equal its source (eq=%v err=%v)", eq, err)
	}
	h1, _ := proj.InputHash()
	proj.MachineryVersion = "v9"
	h2, _ := proj.InputHash()
	if h1 != h2 {
		t.Fatal("machinery_version must not bind the v2 input hash")
	}
	onlyMachines, err := GenerateWithFacts(model, facts, manifestWith([]string{"machines"}, nil), validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	if onlyMachines.Model != nil {
		t.Fatal("a v2 include with no v1 layer carries no model block")
	}
	if _, err := Generate(model, manifestWith([]string{"machines"}, nil), validTestDesignID, "v0"); err == nil {
		t.Fatal("a v2 layer without a fact reader must fail loudly")
	}
}

func TestV2ProjectionParseRejectsMalformedLayers(t *testing.T) {
	model, err := LoadModel(writeTemp(t, "v2.modelith.yaml", sampleModel))
	if err != nil {
		t.Fatal(err)
	}
	proj, err := GenerateWithFacts(model, sampleFacts(t), manifestWith([]string{"machines"}, nil), validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := proj.Render()
	if err != nil {
		t.Fatal(err)
	}
	good := string(rendered)
	cases := map[string]string{
		"v1 schema with layers":  strings.Replace(good, `"projection_schema": "2.0"`, `"projection_schema": "1.0"`, 1),
		"unknown relation":       strings.Replace(good, `"guard_on": [`, `"guard_onn": [`, 1),
		"missing column":         strings.Replace(good, `"kind": "final"`, `"kindx": "final"`, 1),
		"backslash source":       strings.Replace(good, `"path": "machines/Order.machine.json"`, `"path": "machines\\Order.machine.json"`, 1),
		"zero line":              strings.Replace(good, `"line": 9`, `"line": 0`, 1),
		"layer not included":     strings.Replace(good, `"include": [`+"\n    \"machines\"", `"include": [`+"\n    \"model\",\n    \"machines\"", 1),
		"non-string column":      strings.Replace(good, `"kind": "final"`, `"kind": 3`, 1),
		"empty stable id":        strings.Replace(good, `"stable_id": "state:Order.Paid"`, `"stable_id": ""`, 1),
		"relation not an array":  strings.Replace(good, `"guard_on": []`, `"guard_on": null`, 1),
		"tab inside a fact cell": strings.Replace(good, `"kind": "final"`, `"kind": "fi\tnal"`, 1),
	}
	for name, body := range cases {
		if body == good {
			t.Fatalf("%s: the mutation did not apply", name)
		}
		if _, err := ParseProjection([]byte(body)); err == nil {
			t.Errorf("%s: a malformed v2 projection must be rejected", name)
		}
	}
}

func TestV1OnlyManifestNeverSelectsV2(t *testing.T) {
	model, err := LoadModel(writeTemp(t, "v1.modelith.yaml", sampleModel))
	if err != nil {
		t.Fatal(err)
	}
	man := manifestWith([]string{"model", "invariants", "relationships"}, nil)
	withFacts, err := GenerateWithFacts(model, sampleFacts(t), man, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	without, err := Generate(model, man, validTestDesignID, "v0")
	if err != nil {
		t.Fatal(err)
	}
	a, err := withFacts.Render()
	if err != nil {
		t.Fatal(err)
	}
	b, err := without.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) || withFacts.ProjectionSchema != SchemaVersion || bytes.Contains(a, []byte(`"layers"`)) {
		t.Fatalf("a v1-only manifest must render the 1.0 bytes whether or not facts are supplied:\n%s", a)
	}
}
