package gates

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// debtMatrix is a named-unit matrix carrying consistency debt of both kinds:
// a USES{} member that resolves to nothing (Gy-rules fact_unresolved), and
// prose quoting a model attribute and a token naming nothing (Gl-ledger
// undeclared-fact warnings of both classes).
const debtMatrix = "| name | kind | event | pre / post |\n|---|---|---|---|\n" +
	"| `canPay` | guard | - | compares `Order.total` with `ghost_token`. USES{Order.stat} |\n" +
	"| `recordPay` | action | - | stores the payment. WRITES{} |\n"

// debtDesign is a synthetic design whose Gy-rules findings and Gl-ledger
// undeclared-fact warnings are all adoption debt: a System action with no
// admission row (authz_missing), a matrix with no machine and no waiver
// (orphan_matrix), an unresolved USES{} member, and two backticked tokens
// outside every group.
func debtDesign(t *testing.T) string {
	t.Helper()
	return writeFactsDesign(t, t.TempDir(), map[string]string{
		"machines/Order.machine.json": factsMachine,
		"machines/Order.matrix.md":    debtMatrix,
		"machines/Ledger.matrix.md":   "| name | kind | pre / post |\n|---|---|---|\n| `checkLedger` | guard | - |\n",
	})
}

func gateOutput(g *Gate) string {
	var b bytes.Buffer
	g.Emit(&b)
	return b.String()
}

func notesWith(g *Gate, needle string) []string {
	var out []string
	for _, n := range g.Notes {
		if strings.Contains(n, needle) {
			out = append(out, n)
		}
	}
	return out
}

// recordDebt records the design's current Gy and Gl debt into its ratchet,
// as `machinery baseline <design> --gate gy,gl` does.
func recordDebt(t *testing.T, design string, grow bool) []DebtRecord {
	t.Helper()
	prior, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	r := prior
	if r == nil {
		r = &Ratchet{Date: "2026-09-23"}
	}
	recs, err := RecordConsistencyDebt(design, "", r, true, true, grow)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteRatchet(design, r); err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestConsistencyDebtIsRedBeforeTheBaseline(t *testing.T) {
	design := debtDesign(t)
	gy := CheckRules(design, false)
	if len(gy.Errs) != 3 || !hasErr(gy, "authz_missing") || !hasErr(gy, "fact_unresolved") || !hasErr(gy, "orphan_matrix") {
		t.Fatalf("the fixture must carry three Gy findings: %v", gy.Errs)
	}
	gl := CheckLedger(design)
	if len(gl.Warns) != 2 {
		t.Fatalf("the fixture must carry two Gl undeclared-fact warnings: %v", gl.Warns)
	}
}

// Recording the baseline turns every recorded finding into a note, counted on
// the checked line, and the gates stop blocking and stop warning.
func TestConsistencyBaselineMakesRecordedFindingsNotes(t *testing.T) {
	design := debtDesign(t)
	recs := recordDebt(t, design, false)
	if len(recs) != 2 || recs[0].Recorded != 3 || recs[1].Recorded != 2 || !recs[0].First || !recs[1].First {
		t.Fatalf("first recording must record every current finding: %+v", recs)
	}
	r, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	if r.Edges != nil {
		t.Fatalf("a Gy/Gl-only baseline records no G4 edges section: %+v", r.Edges)
	}
	if len(r.Rules) != 3 || len(r.Undeclared) != 2 {
		t.Fatalf("ratchet sections: rules %+v undeclared %+v", r.Rules, r.Undeclared)
	}

	gy := CheckRules(design, false)
	if len(gy.Errs) != 0 || len(gy.Warns) != 0 {
		t.Fatalf("baselined Gy findings must not block or warn: %v %v", gy.Errs, gy.Warns)
	}
	if gy.Counts[BaselinedCount] != 3 || len(notesWith(gy, "baselined: ")) != 3 {
		t.Fatalf("Gy must note and count the three baselined findings: %v %v", gy.Counts, gy.Notes)
	}
	if !strings.Contains(gateOutput(gy), "3 baselined") || !strings.Contains(gateOutput(gy), "  ok\n") {
		t.Fatalf("the checked line carries the baselined count:\n%s", gateOutput(gy))
	}

	gl := CheckLedger(design)
	if len(gl.Warns) != 0 || len(gl.Errs) != 0 {
		t.Fatalf("baselined Gl warnings must not warn: %v %v", gl.Warns, gl.Errs)
	}
	if gl.Counts[BaselinedCount] != 2 || len(notesWith(gl, "baselined: machines/Order.matrix.md:")) != 2 {
		t.Fatalf("Gl must note and count the two baselined warnings: %v %v", gl.Counts, gl.Notes)
	}
	// the class counts stay exact: they describe what the scan found
	if gl.Counts["undeclared fact references"] != 1 || gl.Counts["unresolved backticked tokens"] != 1 {
		t.Fatalf("Gl class counts changed: %v", gl.Counts)
	}
}

// A finding added after the baseline blocks exactly as it does without one,
// beside the notes for the recorded ones.
func TestConsistencyBaselineNewFindingStillBlocks(t *testing.T) {
	design := debtDesign(t)
	recordDebt(t, design, false)
	matrix := filepath.Join(design, "machines", "Order.matrix.md")
	body, err := os.ReadFile(matrix)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, matrix, strings.Replace(string(body), "stores the payment. WRITES{}", "stores the payment for `other_ghost`. USES{Order.totl} WRITES{}", 1))

	gy := CheckRules(design, false)
	if len(gy.Errs) != 1 || !hasErr(gy, "fact_unresolved (fact 'Order.totl')") || gy.Counts[BaselinedCount] != 3 {
		t.Fatalf("the new Gy finding must block beside three baselined ones: errs %v counts %v", gy.Errs, gy.Counts)
	}
	gl := CheckLedger(design)
	if len(gl.Warns) != 1 || !strings.Contains(gl.Warns[0], "`other_ghost`") || gl.Counts[BaselinedCount] != 2 {
		t.Fatalf("the new Gl warning must warn beside two baselined ones: warns %v counts %v", gl.Warns, gl.Counts)
	}
}

// Fixing a baselined finding yields a resolved note, never an error; the next
// baseline shrinks the ratchet, and it does not absorb a new finding unless
// --grow is given.
func TestConsistencyBaselineResolvesAndShrinks(t *testing.T) {
	design := debtDesign(t)
	recordDebt(t, design, false)
	matrix := filepath.Join(design, "machines", "Order.matrix.md")
	body, err := os.ReadFile(matrix)
	if err != nil {
		t.Fatal(err)
	}
	fixed := strings.Replace(string(body), "USES{Order.stat}", "USES{Order.status}", 1)
	fixed = strings.Replace(fixed, " with `ghost_token`", " with nothing", 1)
	// and one new finding of each kind, which a shrinking baseline must not absorb
	fixed = strings.Replace(fixed, "stores the payment. WRITES{}", "stores `late_ghost`. USES{Order.totl} WRITES{}", 1)
	mustWrite(t, matrix, fixed)

	gy := CheckRules(design, false)
	if len(notesWith(gy, "resolved; run machinery baseline to shrink the ratchet")) != 1 || gy.Counts[BaselinedCount] != 2 {
		t.Fatalf("the fixed Gy finding must be a resolved note: %v %v", gy.Notes, gy.Counts)
	}
	gl := CheckLedger(design)
	if len(notesWith(gl, "resolved; run machinery baseline to shrink the ratchet")) != 1 || gl.Counts[BaselinedCount] != 1 {
		t.Fatalf("the fixed Gl warning must be a resolved note: %v %v", gl.Notes, gl.Counts)
	}
	for _, g := range []*Gate{gy, gl} {
		for _, e := range append(append([]string(nil), g.Errs...), g.Warns...) {
			if strings.Contains(e, "shrink the ratchet") {
				t.Fatalf("%s: a resolved entry must never be an error or a warning: %s", g.Title, e)
			}
		}
	}

	recs := recordDebt(t, design, false)
	gyRec, glRec := recs[0], recs[1]
	if gyRec.First || gyRec.Observed != 3 || gyRec.Recorded != 2 || gyRec.NotRecorded != 1 || gyRec.Dropped != 1 {
		t.Fatalf("a Gy re-baseline must shrink and refuse growth: %+v", gyRec)
	}
	if glRec.First || glRec.Observed != 2 || glRec.Recorded != 1 || glRec.NotRecorded != 1 || glRec.Dropped != 1 {
		t.Fatalf("a Gl re-baseline must shrink and refuse growth: %+v", glRec)
	}
	r, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Rules) != 2 || len(r.Undeclared) != 1 {
		t.Fatalf("the ratchet must shrink to the surviving entries: %+v %+v", r.Rules, r.Undeclared)
	}
	if g := CheckRules(design, false); len(g.Errs) != 1 || len(notesWith(g, "resolved")) != 0 {
		t.Fatalf("after the shrink the new finding still blocks and nothing is stale: %v %v", g.Errs, g.Notes)
	}

	recs = recordDebt(t, design, true)
	if recs[0].Recorded != 3 || recs[1].Recorded != 2 || recs[0].NotRecorded != 0 || recs[1].NotRecorded != 0 {
		t.Fatalf("--grow records the new findings: %+v", recs)
	}
	if g := CheckRules(design, false); len(g.Errs) != 0 {
		t.Fatalf("a grown baseline tolerates the new finding: %v", g.Errs)
	}
}

// An undeclared-fact key can repeat (two rows of one name in one file: an
// event on several consumer rows); the ratchet records its count and
// tolerates no more than that many.
func TestConsistencyBaselineCountsRepeatedKeys(t *testing.T) {
	design := debtDesign(t)
	matrix := filepath.Join(design, "machines", "Order.matrix.md")
	twice := debtMatrix + "\n| event | consumer | payload notes |\n|---|---|---|\n" +
		"| `order.paid` | ledger | carries `ghost_token` |\n| `order.paid` | audit | carries `ghost_token` |\n"
	mustWrite(t, matrix, twice)
	recordDebt(t, design, false)
	r, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range r.Undeclared {
		if u.Token == "ghost_token" {
			found = found || (u.Count == 2 && u.Unit == "order.paid" && u.File == "machines/Order.matrix.md")
		}
	}
	if !found {
		t.Fatalf("a repeated key must be recorded with its count: %+v", r.Undeclared)
	}
	mustWrite(t, matrix, twice+"| `order.paid` | billing | carries `ghost_token` |\n")
	gl := CheckLedger(design)
	if len(gl.Warns) != 1 || !strings.Contains(gl.Warns[0], "`ghost_token`") {
		t.Fatalf("a third occurrence exceeds the recorded count and warns: %v", gl.Warns)
	}
}

// Projection errors are a broken design, not debt: the baseline refuses to
// record while any exists, and names them.
func TestConsistencyBaselineRefusesProjectionErrors(t *testing.T) {
	design := debtDesign(t)
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), strings.Replace(debtMatrix, "USES{Order.stat}", "USES{Order.stat} USES{Order.total}", 1))
	if g := CheckRules(design, false); !hasErr(g, "projection error") {
		t.Fatalf("the fixture must carry a projection error: %v", g.Errs)
	}
	r := &Ratchet{Date: "2026-09-23"}
	_, err := RecordConsistencyDebt(design, "", r, true, true, false)
	if err == nil || !strings.Contains(err.Error(), "projection error") || !strings.Contains(err.Error(), "not debt") {
		t.Fatalf("baseline must refuse while Gy has projection errors: %v", err)
	}
	if r.Rules != nil || r.Undeclared != nil {
		t.Fatalf("a refused baseline records nothing: %+v", r)
	}
}

// A ratchet written before this change (edges only) renders byte for byte as
// it did, and G4's output is identical whether or not Gy/Gl sections sit
// beside the edges.
func TestRatchetEdgesOnlyFormatAndG4OutputUnchanged(t *testing.T) {
	old := "{\n  \"date\": \"2026-07\",\n  \"edges\": {\n    \"alpha -\\u003e beta\": [\n      \"alpha/a.go\"\n    ]\n  }\n}\n"
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]", secondFile: true})
	mustWrite(t, filepath.Join(design, RatchetFile), old)
	r, err := LoadRatchet(design)
	if err != nil {
		t.Fatal(err)
	}
	if r.Rules != nil || r.Undeclared != nil {
		t.Fatalf("an edges-only ratchet has no consistency sections: %+v", r)
	}
	body, err := RenderRatchet(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != old {
		t.Fatalf("an edges-only ratchet must render unchanged:\n%s\nwant\n%s", body, old)
	}
	before := gateOutput(CheckImports(design, impl))

	r.Rules = []RuleDebt{{Relation: "finding_orphan_matrix", Tuple: []string{"Ledger"}}}
	r.Undeclared = []UndeclaredDebt{{File: "machines/Order.matrix.md", Unit: "canPay", Token: "ghost_token", Count: 1}}
	if err := WriteRatchet(design, r); err != nil {
		t.Fatal(err)
	}
	if after := gateOutput(CheckImports(design, impl)); after != before {
		t.Fatalf("G4 output changed with consistency sections beside the edges:\n%s\nwas\n%s", after, before)
	}
}

// A ratchet that records only Gy/Gl debt carries no G4 snapshot: G4 treats it
// as no ratchet, so a Gy baseline never arms or amnesties import findings.
func TestRatchetWithoutEdgesIsNoG4Snapshot(t *testing.T) {
	design, impl := writeFixture(t, fixtureOpts{rules: "  baseline: [\"alpha -> beta\"]"})
	mustWrite(t, filepath.Join(design, RatchetFile), "{\n  \"date\": \"2026-07\",\n  \"rule_findings\": []\n}\n")
	g := CheckImports(design, impl)
	if !hasErr(g, "records no G4 edges") || hasNote(g, "ratchet snapshot") {
		t.Fatalf("an edges-less ratchet is no G4 snapshot: errs %v notes %v", g.Errs, g.Notes)
	}
	if RatchetArmsImports(design) {
		t.Fatal("an edges-less ratchet must not arm import blocking")
	}
	mustWrite(t, filepath.Join(design, RatchetFile), "{\"date\":\"2026-07\",\"edges\":{}}")
	if !RatchetArmsImports(design) {
		t.Fatal("an edges section arms import blocking")
	}
}

func TestRatchetConsistencySectionsRejectMalformedEntries(t *testing.T) {
	tests := []struct {
		name, body, want string
	}{
		{"no section at all", `{"date":"2026-09"}`, "missing required root key"},
		{"rules not an array", `{"date":"2026-09","rule_findings":{}}`, "rule_findings"},
		{"rule unknown field", `{"date":"2026-09","rule_findings":[{"relation":"finding_x","tuple":["a"],"line":3}]}`, "unknown field"},
		{"rule empty tuple", `{"date":"2026-09","rule_findings":[{"relation":"finding_x","tuple":[]}]}`, "tuple"},
		{"rule bad relation", `{"date":"2026-09","rule_findings":[{"relation":"x","tuple":["a"]}]}`, "finding_<code> or warn_<code>"},
		{"rule duplicate", `{"date":"2026-09","rule_findings":[{"relation":"finding_x","tuple":["a"]},{"relation":"finding_x","tuple":["a"]}]}`, "duplicate"},
		{"undeclared zero count", `{"date":"2026-09","undeclared_facts":[{"file":"m.md","unit":"u","token":"t_x","count":0}]}`, "count"},
		{"undeclared missing token", `{"date":"2026-09","undeclared_facts":[{"file":"m.md","unit":"u","count":1}]}`, "token"},
		{"undeclared duplicate", `{"date":"2026-09","undeclared_facts":[{"file":"m.md","unit":"u","token":"t_x","count":1},{"file":"m.md","unit":"u","token":"t_x","count":2}]}`, "duplicate"},
		{"duplicate section", `{"date":"2026-09","rule_findings":[],"rule_findings":[]}`, "duplicate root key"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			design := t.TempDir()
			mustWrite(t, filepath.Join(design, RatchetFile), tc.body)
			_, err := LoadRatchet(design)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("LoadRatchet error = %v, want %q", err, tc.want)
			}
		})
	}
}

// Final handoff refuses while baselined consistency debt remains, and prints
// the count, the way it refuses open milestones.
func TestFinalHandoffRefusesBaselinedConsistencyDebt(t *testing.T) {
	design := debtDesign(t)
	recordDebt(t, design, false)
	sel := Selection{Run: map[string]bool{"gy": true, "gl": true}, Explicit: true}
	var final *Gate
	for _, g := range RunSelected(design, "", sel, RunOptions{Complete: true}) {
		if strings.HasPrefix(g.Title, "G!-complete") {
			final = g
		}
	}
	if final == nil {
		t.Fatal("--complete must run the final-handoff gate")
	}
	if !hasErr(final, "5 baselined consistency finding(s) remain (Gy-rules 3, Gl-ledger 2)") {
		t.Fatalf("final handoff must refuse baselined debt and print the count: %v", final.Errs)
	}
	for _, g := range RunSelected(design, "", sel, RunOptions{}) {
		if strings.HasPrefix(g.Title, "G!-complete") {
			t.Fatal("the final-handoff gate runs only under --complete")
		}
	}
}

// The Gl summary threshold applies to baselined notes as it does to warnings:
// past it, one note per file unless --verbose.
func TestConsistencyBaselineSummarizesManyGlNotes(t *testing.T) {
	design := debtDesign(t)
	var rows strings.Builder
	rows.WriteString(debtMatrix)
	for i := 0; i < UndeclaredSummaryThreshold+1; i++ {
		rows.WriteString("| `unit" + string(rune('a'+i)) + "` | guard | - | mentions `ghost_" + string(rune('a'+i)) + "_x`. USES{Order.status} |\n")
	}
	mustWrite(t, filepath.Join(design, "machines", "Order.matrix.md"), rows.String())
	recordDebt(t, design, false)
	gl := CheckLedgerWith(design, LedgerOptions{})
	if len(gl.Warns) != 0 || len(notesWith(gl, "baselined: ")) != 1 || !hasNote(gl, "23 undeclared-fact warning(s) recorded in "+RatchetFile) {
		t.Fatalf("past the threshold the baselined notes summarize per file: %v", gl.Notes)
	}
	if v := CheckLedgerWith(design, LedgerOptions{Verbose: true}); len(notesWith(v, "baselined: ")) != 23 {
		t.Fatalf("--verbose lists every baselined line: %d", len(notesWith(v, "baselined: ")))
	}
}
