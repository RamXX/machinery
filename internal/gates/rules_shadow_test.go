package gates

// Shadow agreement: the subjects the 0.9.0 Gx heuristics report for classes A
// (authorization inventory), B (fact resolution), C (closed vocabularies) and
// D (payload twins) against the subjects the corresponding finding_ relations
// report. Where the two differ the rules are NOT tuned toward the heuristics:
// the difference is written down below as a named discrepancy saying which
// side is right and why, and the test asserts exactly that documented
// difference, so a new one fails until it is written down.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/checker"
)

// shadowDiscrepancy is one documented difference. Subjects are class-tagged:
// "A:missing Order.pay", "A:orphan X", "A:unknown X", "B Order.x|fact",
// "C Order.x", "D Order.x".
type shadowDiscrepancy struct {
	name    string // the named case
	design  string // example path, or "fixture:<name>"
	subject string // class-tagged subject
	side    string // "heuristic-only" or "rules-only"
	right   string // which side is right, and why
}

// documentedDiscrepancies is the complete list. The bundled examples agree
// exactly (both sides report nothing for A through D); every entry below is a
// fixture that isolates one structural difference.
var documentedDiscrepancies = []shadowDiscrepancy{
	{
		name: "read-only System action stated in prose", design: "fixture:prose-read-only",
		subject: "A:missing Order.recompute", side: "rules-only",
		right: "rules: the 0.9.0 heuristic drops the obligation because the action description says it writes nothing; prose never declares, so the rule keeps it until the unit declares WRITES{}",
	},
	{
		name: "fact named only in prose", design: "fixture:prose-fact",
		subject: "B Order.recompute|missing_fact", side: "heuristic-only",
		right: "rules: a backticked token in a contract cell is prose (Gl-ledger warns on it as an undeclared reference); only USES{}/WRITES{} members are facts to resolve",
	},
	{
		name: "VALUES group bound to an enum by a normalized name", design: "fixture:values-normalized-name",
		subject: "C Order.orderState", side: "heuristic-only",
		right: "heuristic, on intent: VALUES{...} on unit orderState is meant as enum OrderState, which the heuristic matches case-insensitively; the rule compares names exactly and has no string functions, so the fix is a normalized group key in the projection (Stage 4), or the explicit VALUES OrderState{...}",
	},
}

var (
	shadowMissing   = regexp.MustCompile(`^(?:System action|matrix producer) '([^']+)' has no authorization row`)
	shadowOrphan    = regexp.MustCompile(`: '([^']+)' names no System action or matrix producer`)
	shadowUnknown   = regexp.MustCompile(`^(\S+):(\d+) authorization row \d+: admission subject '[^']+' is not declared`)
	shadowMatrixAt  = regexp.MustCompile(`^(\S+\.matrix\.md):(\d+): `)
	shadowUnresolve = regexp.MustCompile(`unresolved fact '([^']+)'`)
)

// heuristicSubjects runs the four 0.9.0 checks and maps their findings to
// class-tagged subjects. A finding the mapping cannot place is returned as
// unplaced, so a new message shape fails the test instead of vanishing.
func heuristicSubjects(t *testing.T, design string, facts *checker.DesignFacts) (map[string]bool, []string) {
	t.Helper()
	unitsAt := map[string][]string{}
	for _, row := range facts.Rows("unit") {
		key := filepath.Base(row.Source.Path) + ":" + strconv.Itoa(row.Source.Line)
		unitsAt[key] = append(unitsAt[key], row.Values[0])
	}
	admissionAt := map[string]string{}
	for _, row := range facts.Rows("admission") {
		admissionAt[row.Source.String()] = row.Values[0]
	}
	load := NewGate("load")
	dm := loadModelith(design, load)
	if dm == nil {
		t.Fatalf("%s: %v", design, load.Errs)
	}
	archText := readDesignOrEmpty(design, filepath.Join(design, "ARCHITECTURE.md"))
	out := map[string]bool{}
	var unplaced []string
	a := NewGate("A")
	checkAuthorizationInventory(a, design, dm)
	for _, e := range a.Errs {
		switch {
		case shadowMissing.MatchString(e):
			out["A:missing "+shadowMissing.FindStringSubmatch(e)[1]] = true
		case shadowOrphan.MatchString(e):
			out["A:orphan "+shadowOrphan.FindStringSubmatch(e)[1]] = true
		case shadowUnknown.MatchString(e):
			m := shadowUnknown.FindStringSubmatch(e)
			if s, ok := admissionAt[m[1]+":"+m[2]]; ok {
				out["A:unknown "+s] = true
			} else {
				unplaced = append(unplaced, e)
			}
		default:
			unplaced = append(unplaced, e)
		}
	}
	matrixClass := func(class string, errs []string) {
		for _, e := range errs {
			m := shadowMatrixAt.FindStringSubmatch(e)
			units := []string(nil)
			if m != nil {
				units = unitsAt[m[1]+":"+m[2]]
			}
			if len(units) == 0 {
				unplaced = append(unplaced, class+": "+e)
				continue
			}
			for _, u := range units {
				if class == "B" {
					if f := shadowUnresolve.FindStringSubmatch(e); f != nil {
						out["B "+u+"|"+f[1]] = true
						continue
					}
				}
				out[class+" "+u] = true
			}
		}
	}
	b := NewGate("B")
	checkFactResolution(b, design, archText, dm)
	matrixClass("B", b.Errs)
	c := NewGate("C")
	checkClosedVocabularies(c, design, dm)
	matrixClass("C", c.Errs)
	d := NewGate("D")
	checkPayloadTwins(d, design, archText)
	matrixClass("D", d.Errs)
	return out, unplaced
}

// ruleSubjects maps the class A through D finding_ relations to the same
// class-tagged subjects.
func ruleSubjects(t *testing.T, facts *checker.DesignFacts) map[string]bool {
	t.Helper()
	set, err := shippedRules()
	if err != nil {
		t.Fatal(err)
	}
	findings, errs := evaluateRules(set, facts)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	tag := map[string]string{
		"finding_authz_missing": "A:missing", "finding_authz_orphan": "A:orphan",
		"finding_authz_unknown_capability": "A:unknown",
		"finding_fact_unresolved":          "B", "finding_values_disagree": "C", "finding_payload_twin": "D",
	}
	out := map[string]bool{}
	for _, f := range findings {
		prefix, ok := tag[f.output.relation]
		if !ok {
			continue
		}
		subject := f.tuple[0]
		if prefix == "B" {
			subject += "|" + f.tuple[1]
		}
		out[prefix+" "+subject] = true
	}
	return out
}

func shadowDiff(heuristic, rules map[string]bool) []string {
	var out []string
	for s := range heuristic {
		if !rules[s] {
			out = append(out, s+" heuristic-only")
		}
	}
	for s := range rules {
		if !heuristic[s] {
			out = append(out, s+" rules-only")
		}
	}
	sort.Strings(out)
	return out
}

func documentedFor(design string) []string {
	var out []string
	for _, d := range documentedDiscrepancies {
		if d.design == design {
			out = append(out, d.subject+" "+d.side)
		}
	}
	sort.Strings(out)
	return out
}

// shadowFixtures are the designs that isolate each documented discrepancy.
var shadowFixtures = map[string]map[string]string{
	"prose-read-only": {
		"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n" +
			"    actions: [{name: recompute, actor: System, description: \"Recompute the displayed total; writes nothing.\"}]\n",
		"machines/Order.machine.json": `{"id":"Order","initial":"Open","states":{"Open":{"type":"final"}}}`,
		"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | computes the total |\n",
	},
	"prose-fact": {
		"domain.modelith.yaml":        "kind: DomainModel\nversion: v1\nentities:\n  Order:\n    attributes: [{name: total, type: integer}]\n    actions: [{name: recompute}]\n",
		"machines/Order.machine.json": `{"id":"Order","initial":"Open","states":{"Open":{"type":"final"}}}`,
		"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `recompute` | action | reads `missing_fact` / records result |\n",
	},
	"values-normalized-name": {
		"domain.modelith.yaml": "kind: DomainModel\nversion: v1\nenums:\n  OrderState:\n    values: [{name: Open}, {name: Closed}]\n" +
			"entities:\n  Order:\n    attributes: [{name: state, type: OrderState}]\n    actions: [{name: classify}]\n",
		"machines/Order.machine.json": `{"id":"Order","initial":"Open","states":{"Open":{"type":"final"}}}`,
		"machines/Order.matrix.md":    "| name | kind | contract (pre / post) |\n|---|---|---|\n| `orderState` | guard | VALUES{Open, Closed, Voided} |\n",
	},
}

func TestRulesShadowAgreement(t *testing.T) {
	type target struct{ name, path string }
	var targets []target
	for _, rel := range bundledDesigns(t) {
		design := filepath.Join("..", "..", filepath.FromSlash(rel))
		if !RulesActive(design) {
			continue // Gy-rules does not run here; Gx narrows away too
		}
		targets = append(targets, target{rel, design})
	}
	var names []string
	for name := range shadowFixtures {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		dir := t.TempDir()
		for rel, body := range shadowFixtures[name] {
			path := filepath.Join(dir, filepath.FromSlash(rel))
			must(t, os.MkdirAll(filepath.Dir(path), 0o755))
			must(t, os.WriteFile(path, []byte(body), 0o644))
		}
		targets = append(targets, target{"fixture:" + name, dir})
	}
	for _, tg := range targets {
		t.Run(tg.name, func(t *testing.T) {
			facts, err := LoadDesignFacts(tg.path)
			if err != nil {
				t.Fatal(err)
			}
			heuristic, unplaced := heuristicSubjects(t, tg.path, facts)
			if len(unplaced) > 0 {
				t.Fatalf("heuristic findings the comparison cannot place: %v", unplaced)
			}
			ruled := ruleSubjects(t, facts)
			t.Logf("heuristic %d subject(s), rules %d subject(s)", len(heuristic), len(ruled))
			got := shadowDiff(heuristic, ruled)
			if want := documentedFor(tg.name); strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("heuristic and rule subjects differ in an undocumented way\n got: %v\nwant: %v", got, want)
			}
		})
	}
}

// Every documented discrepancy names a fixture that exists, a side and a
// verdict, so the list cannot rot into unexplained entries.
func TestDocumentedDiscrepanciesAreComplete(t *testing.T) {
	for _, d := range documentedDiscrepancies {
		name := strings.TrimPrefix(d.design, "fixture:")
		if _, ok := shadowFixtures[name]; !ok && strings.HasPrefix(d.design, "fixture:") {
			t.Fatalf("%s: no fixture %s", d.name, name)
		}
		if d.side != "heuristic-only" && d.side != "rules-only" || d.right == "" || d.name == "" {
			t.Fatalf("incomplete discrepancy %+v", d)
		}
	}
}
