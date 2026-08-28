package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func namerotDesign(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	md := filepath.Join(d, "machines")
	if err := os.MkdirAll(md, 0755); err != nil {
		t.Fatal(err)
	}
	deal := `{"id":"deal","initial":"Lead","_comment":"see ` + "`Deal.Paused`" + ` for the hold story","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: refused; compare ` + "`Deal.Won`" + `"}},
		"Won":{"type":"final"}}}`
	if err := os.WriteFile(filepath.Join(md, "Deal.machine.json"), []byte(deal), 0644); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestNameRotFlagsMissingState(t *testing.T) {
	d := namerotDesign(t)
	g := NewGate("t")
	CheckNameRot(g, d)
	found := false
	for _, w := range g.Warns {
		if strings.Contains(w, "`Deal.Paused`") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing-state ref not flagged: %v", g.Warns)
	}
	for _, w := range g.Warns {
		if strings.Contains(w, "Deal.Won") {
			t.Fatalf("existing state flagged: %v", g.Warns)
		}
	}
}

func TestNameRotFileRefsAndUnknownMachinesIgnored(t *testing.T) {
	d := namerotDesign(t)
	mx := "see `Deal.machine.json` and `Deal.matrix.md` and `Elixir.Module.name` and `Deal.canAdvance`\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(mx), 0644); err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckNameRot(g, d)
	for _, w := range g.Warns {
		if strings.Contains(w, "matrix.md:1") {
			t.Fatalf("file ref, unknown machine, or unit ref flagged: %v", g.Warns)
		}
	}
}

func TestNameRotSnakeCaseIsNote(t *testing.T) {
	d := namerotDesign(t)
	mx := "the pair `Deal.route_review` names the requesting act\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(mx), 0644); err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckNameRot(g, d)
	for _, w := range g.Warns {
		if strings.Contains(w, "matrix.md") {
			t.Fatalf("snake_case miss must be a note, got warn: %v", g.Warns)
		}
	}
	found := false
	for _, n := range g.Notes {
		if strings.Contains(n, "Deal.route_review") {
			found = true
		}
	}
	if !found {
		t.Fatalf("snake_case miss not noted: %v", g.Notes)
	}
}

func TestNameRotModelVocabularyAdmitted(t *testing.T) {
	d := namerotDesign(t)
	model := `kind: modelith
version: 1
entities:
  Deal:
    definition: d
    attributes:
      - {name: trace_ref, type: string}
    actions:
      - name: export_library
        actor: Admin
        description: d
`
	if err := os.WriteFile(filepath.Join(d, "domain.modelith.yaml"), []byte(model), 0644); err != nil {
		t.Fatal(err)
	}
	mx := "reads `Deal.trace_ref` and `Deal.export_library`\n"
	if err := os.WriteFile(filepath.Join(d, "machines", "Deal.matrix.md"), []byte(mx), 0644); err != nil {
		t.Fatal(err)
	}
	g := NewGate("t")
	CheckNameRot(g, d)
	for _, x := range append(append([]string{}, g.Warns...), g.Notes...) {
		if strings.Contains(x, "matrix.md") {
			t.Fatalf("model vocabulary flagged: warns %v notes %v", g.Warns, g.Notes)
		}
	}
}
