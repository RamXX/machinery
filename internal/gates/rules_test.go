package gates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/RamXX/machinery/internal/checker"
)

// The shipped rule files load at test time, so a rule that stops parsing, or
// breaks the gate's contract, fails this package's tests instead of a user's
// check.
func TestShippedRulesLoad(t *testing.T) {
	set, err := shippedRules()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{
		"authz.dl":        {"finding_authz_missing", "finding_authz_orphan", "finding_authz_unknown_capability", "finding_produces_unknown_action"},
		"carriers.dl":     {"finding_effect_uncarried", "finding_carrier_misplaced"},
		"facts.dl":        {"finding_fact_unresolved"},
		"payload.dl":      {"finding_payload_twin"},
		"supersession.dl": {"finding_duplicate_owner", "finding_supersession_cycle", "finding_dangling_replacement"},
		"values.dl":       {"finding_values_disagree"},
	}
	if len(set.files) != len(want) {
		t.Fatalf("%d rule files loaded, want %d", len(set.files), len(want))
	}
	for _, rf := range set.files {
		var got []string
		for _, o := range rf.outputs {
			got = append(got, o.relation)
			if o.warn != strings.HasPrefix(o.relation, warnPrefix) {
				t.Fatalf("%s: %s has the wrong tier", rf.name, o.relation)
			}
		}
		if strings.Join(got, ",") != strings.Join(want[rf.name], ",") {
			t.Fatalf("%s emits %v, want %v", rf.name, got, want[rf.name])
		}
	}
}

func ruleFS(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["r/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

const okRule = `.decl unit(id:symbol, machine:symbol, name:symbol, kind:symbol)
.input unit
.decl finding_x(unit:symbol)
.output finding_x
finding_x(U) :- unit(U, _, _, "actor").
`

func TestRuleSetLoadRejections(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"parse error", map[string]string{"bad.dl": ".decl r(a:symbol)\nr(X) :- .\n"}, "bad.dl:2"},
		{"duplicate relation across files", map[string]string{"a.dl": okRule, "b.dl": okRule},
			"b.dl: relation finding_x is already emitted by a.dl"},
		{"output without a finding or warn prefix", map[string]string{"a.dl": strings.ReplaceAll(okRule, "finding_x", "result_x")},
			"a.dl: .output result_x is not named finding_<code> or warn_<code>"},
		{"bare prefix", map[string]string{"a.dl": strings.ReplaceAll(okRule, "finding_x", "warn_")},
			".output warn_ is not named"},
		{"finding named relation that is not output", map[string]string{"a.dl": strings.ReplaceAll(okRule, ".output finding_x\n", ".decl finding_y(unit:symbol)\nfinding_y(U) :- finding_x(U).\n.output finding_x\n")},
			"a.dl: finding_y is named like a finding but is not an .output"},
		{"input outside the catalog", map[string]string{"a.dl": strings.ReplaceAll(strings.ReplaceAll(okRule, "unit(", "units("), ".input unit\n", ".input units\n")},
			"a.dl: .input units is not a projected relation"},
		{"input arity differs", map[string]string{"a.dl": ".decl unit(id:symbol)\n.input unit\n.decl finding_x(unit:symbol)\n.output finding_x\nfinding_x(U) :- unit(U).\n"},
			"a.dl: .input unit declares 1 attributes, the projection has 4"},
		{"unknown subject kind", map[string]string{"a.dl": strings.ReplaceAll(okRule, "finding_x(unit:symbol)", "finding_x(thing:symbol)")},
			`first attribute "thing" names no subject kind`},
		{"no rule files", map[string]string{"notes.txt": "x"}, "holds no .dl rule file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadRuleSet(ruleFS(tc.files), "r")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error containing %q", err, tc.want)
			}
		})
	}
	if _, err := loadRuleSet(ruleFS(map[string]string{"a.dl": okRule}), "r"); err != nil {
		t.Fatalf("the control rule must load: %v", err)
	}
}

// A design that has only the model and an authorization inventory (no
// machines/, so no matrices, machines or oracles layer): every rule reading
// an absent layer gets it empty and the gate runs clean.
func TestRulesSupplyAbsentLayersEmpty(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"AUTHORIZATION.md": "<!-- machinery:authorization-inventory -->\n\n| authorization subject | admission |\n|---|---|\n| Order.pay | (no authorization: batch job) |\n",
	})
	if !RulesActive(design) {
		t.Fatal("AUTHORIZATION.md alone activates Gy-rules")
	}
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	if facts.HasLayer("matrices") {
		t.Fatal("the fixture must have no matrices layer")
	}
	g := CheckRules(design, false)
	if len(g.Errs)+len(g.Warns)+len(g.Shadow) != 0 {
		t.Fatalf("absent layers must be empty inputs, not errors: %+v", g)
	}
	if g.Counts["rule files evaluated"] != 6 {
		t.Fatalf("every rule file must run: %v", g.Counts)
	}
}

func TestRulesSubjectWithoutSourcePrintsIDVerbatim(t *testing.T) {
	set, err := loadRuleSet(ruleFS(map[string]string{"ghost.dl": `.decl entity(id:symbol)
.input entity
.decl finding_ghost(unit:symbol, fact:symbol)
.output finding_ghost
finding_ghost("Nowhere.unit", E) :- entity(E).
`}), "r")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := LoadDesignFacts(writeFactsDesign(t, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate(RulesGateTitle)
	checkRulesOver(g, set, facts, true)
	if len(g.Shadow) != 1 || g.Shadow[0] != "Nowhere.unit (no source): ghost (fact 'Order')" {
		t.Fatalf("shadow = %q", g.Shadow)
	}
	var out bytes.Buffer
	if blocking := g.Emit(&out); blocking != 0 {
		t.Fatalf("a SHADOW finding never blocks, got %d", blocking)
	}
	want := `== Gy-rules  consistency rules over projected facts ==
  SHADOW Nowhere.unit (no source): ghost (fact 'Order')
         finding_ghost("Nowhere.unit", "Order")  [ghost.dl rule 1]
           entity("Order")  [fact domain.modelith.yaml:11]
  checked: 1 rule files evaluated, `
	if !strings.HasPrefix(out.String(), want) || !strings.Contains(out.String(), ", 1 shadow finding(s)\n  ok\n") {
		t.Fatalf("emitted:\n%s", out.String())
	}
}

func TestRulesTupleLimitIsAnErrorNamingTheRuleFile(t *testing.T) {
	saved := ruleLimits
	t.Cleanup(func() { ruleLimits = saved })
	ruleLimits.maxTuples = 50
	g := CheckRules(filepath.Join("..", "..", "examples", "fulfillment", "design"), false)
	if len(g.Errs) == 0 {
		t.Fatal("a tuple limit hit must be an ERROR")
	}
	for _, e := range g.Errs {
		if !strings.HasPrefix(e, "rules/consistency/") || !strings.Contains(e, "resource limit") || !strings.Contains(e, "MaxTuples 50") {
			t.Fatalf("limit error must name the rule file and the limit: %q", e)
		}
	}
	ruleLimits = saved
	ruleLimits.maxIterations = 1
	g = CheckRules(filepath.Join("..", "..", "examples", "fulfillment", "design"), false)
	if len(g.Errs) == 0 || !strings.Contains(strings.Join(g.Errs, "\n"), "supersession.dl") {
		t.Fatalf("an iteration limit hit in the recursive rule must name supersession.dl: %v", g.Errs)
	}
}

// --explain on a design with zero findings adds nothing to the output.
func TestRulesExplainWithZeroFindingsPrintsNothingExtra(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"AUTHORIZATION.md": "<!-- machinery:authorization-inventory -->\n\n| authorization subject | admission |\n|---|---|\n| Order.pay | (no authorization: batch job) |\n",
	})
	var plain, explained bytes.Buffer
	CheckRules(design, false).Emit(&plain)
	CheckRules(design, true).Emit(&explained)
	if plain.String() != explained.String() {
		t.Fatalf("explain changed a zero-finding run:\n%s\nvs\n%s", plain.String(), explained.String())
	}
	if !strings.Contains(plain.String(), "0 shadow finding(s)") {
		t.Fatalf("the shadow count is always on the checked: line:\n%s", plain.String())
	}
}

func TestRulesGateNotActivatedWhenExplicitlySelected(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"ARCHITECTURE.md": "# Architecture\n",
	})
	sel, run, _, err := SelectRunAndNote(design, "", "gy", RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["gy"] || len(run) != 1 {
		t.Fatalf("gy must be selected and emit one gate: %+v", run)
	}
	var out bytes.Buffer
	run[0].Emit(&out)
	want := "== Gy-rules  consistency rules over projected facts ==\n  note   not activated: the design has no machines/ and no AUTHORIZATION.md, the sources the consistency rules read\n  checked: nothing\n  ok\n"
	if out.String() != want {
		t.Fatalf("got:\n%s", out.String())
	}
	_, run, _, err = SelectRunAndNote(design, "", "", RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range run {
		if g.Title == RulesGateTitle {
			t.Fatal("the default suite skips Gy-rules silently on a design it does not apply to")
		}
	}
}

func TestRulesGateRunsInTheDefaultSuiteAndHonorsExplain(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), everyRuleFires)
	_, run, _, err := SelectRunAndNote(design, "", "gy", RunOptions{Explain: true})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if blocking := run[0].Emit(&out); blocking != 0 {
		t.Fatalf("shadow findings must not block:\n%s", out.String())
	}
	text := out.String()
	for _, want := range []string{
		"  SHADOW ARCHITECTURE.md:12: row 'TypeD': dangling_replacement (old 'Ghost')\n",
		"         finding_dangling_replacement(\"TypeD\", \"Ghost\")  [supersession.dl rule 6]\n",
		"           supersedes(\"TypeD\", \"Ghost\")  [fact ARCHITECTURE.md:12]\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

// Every bundled example is clean under the shipped rules.
func TestRulesBundledExamplesAreClean(t *testing.T) {
	for _, rel := range bundledDesigns(t) {
		design := filepath.Join("..", "..", filepath.FromSlash(rel))
		if !RulesActive(design) {
			continue
		}
		g := CheckRules(design, false)
		if len(g.Errs)+len(g.Warns)+len(g.Shadow) != 0 {
			t.Fatalf("%s: errs=%v warns=%v findings=%v", rel, g.Errs, g.Warns, g.Shadow)
		}
	}
}

// A warn_ relation is a real warning: the shipped rules carry none today,
// so a rule set of one warn_ rule pins the tier.
func TestRulesWarnTierIsARealWarning(t *testing.T) {
	design := writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md": "| name | kind | signature | pre / post |\n|---|---|---|---|\n" +
			"| `recordPay` | action | `(ctx) -> ctx` | WRITES{Order.status} |\n",
	})
	set, err := loadRuleSet(ruleFS(map[string]string{"w.dl": `.decl unit_writes(unit:symbol, fact:symbol)
.input unit_writes
.decl warn_writer(unit:symbol)
.output warn_writer
warn_writer(U) :- unit_writes(U, _).
`}), "r")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := LoadDesignFacts(design)
	if err != nil {
		t.Fatal(err)
	}
	g := NewGate(RulesGateTitle)
	checkRulesOver(g, set, facts, false)
	if len(g.Warns) != 1 || g.Warns[0] != "machines/Order.matrix.md:3: row 'Order.recordPay': writer" || len(g.Errs) != 0 {
		t.Fatalf("warns = %q errs = %q", g.Warns, g.Errs)
	}
	var out bytes.Buffer
	g.Emit(&out)
	if strings.Contains(out.String(), "\n  ok\n") {
		t.Fatalf("a warning is not ok:\n%s", out.String())
	}
}

func TestRulesActiveProbe(t *testing.T) {
	dir := t.TempDir()
	if RulesActive(dir) {
		t.Fatal("an empty design does not activate Gy-rules")
	}
	must(t, os.Mkdir(filepath.Join(dir, "machines"), 0o755))
	if !RulesActive(dir) {
		t.Fatal("a machines/ directory activates Gy-rules, even without machine JSON")
	}
}

// checkRulesOver indexes every relation a subject kind names, so a subject
// kind whose relations leave the catalog would silently stop resolving.
func TestRuleSubjectSourcesNameCatalogRelations(t *testing.T) {
	known := map[string]bool{}
	for _, spec := range checker.RelationCatalog() {
		known[spec.Name] = true
	}
	for kind, rels := range ruleSubjectSources {
		for _, rel := range rels {
			if !known[rel] {
				t.Fatalf("subject kind %s names %s, which the catalog lacks", kind, rel)
			}
		}
	}
}
