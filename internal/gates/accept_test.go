package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// acceptOracleMD is the committed oracle Ga's DoD-coverage rule reads its id
// corpus from (both columns, exactly as Gb reads it).
const acceptOracleMD = `# Generated transition oracle: thing

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-CMD-01 | CMD-abc123 | A | on:go | - | B | - |
| T-CMD-02 | CMD-def456 | B | on:stop | - | A | - |
`

// acceptPlan closes M0 and leaves M1 open.
const acceptPlan = `# BUILD: thing

Mode: full

## 9. Build plan

**M0 - Walking skeleton.** One real transition through one real boundary.
DoD: T-CMD-01 and CMD-abc123 green.
Status: closed

**M1 - Breadth slice.** Everything else. DoD: T-CMD-02 green.
`

const acceptEvidenceM0 = `milestone: 0
commit: 9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345
verdict: ACCEPTED
dod_ids:
  - T-CMD-01
  - CMD-abc123
attestations:
  - integration tests ran against the real datastore, no mocks
  - every guard branch has its falsifying test
findings: []
reviewer: acceptance review, conductor
date: 2026-08-27
`

const acceptedCommit = "9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345"

// writeAcceptFixture lays down a design that closes M0 with committed
// evidence. An entry with an empty value removes that file.
func writeAcceptFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	design := t.TempDir()
	all := map[string]string{
		"BUILD.md":                 acceptPlan,
		"machines/Thing.oracle.md": acceptOracleMD,
		"acceptance/M0.yaml":       acceptEvidenceM0,
	}
	for name, content := range files {
		if content == "" {
			delete(all, name)
			continue
		}
		all[name] = content
	}
	for name, content := range all {
		writeSuiteFile(t, filepath.Join(design, name), content)
	}
	return design
}

func TestCheckAcceptanceGreenPath(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Ga not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	want := map[string]int{
		"plan documents": 1, "declared milestones": 2, "closed milestones": 1,
		"acceptance files": 1, "DoD ids bound": 2,
		"closed milestones with accepted evidence": 1, "commit bindings verified": 1,
	}
	for count, n := range want {
		if g.Counts[count] != n {
			t.Errorf("Ga counted %s=%d, want %d: %+v", count, g.Counts[count], n, g.Counts)
		}
	}
	if len(g.Notes) != 0 {
		t.Errorf("a bound commit needs no note: %v", g.Notes)
	}
}

// Without a commit the gate still runs; the unchecked binding is a
// non-blocking note, never a silent pass.
func TestCheckAcceptanceNoCommitNotes(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	g := CheckAcceptance(design, "")
	if len(g.Errs) != 0 {
		t.Fatalf("Ga not clean without a commit: %v", g.Errs)
	}
	if g.Counts["commit bindings verified"] != 0 {
		t.Errorf("nothing may be reported as bound: %+v", g.Counts)
	}
	if !strings.Contains(strings.Join(g.Notes, "\n"), "commit binding not checked") {
		t.Errorf("the unchecked binding must state itself: %v", g.Notes)
	}
}

func TestCheckAcceptanceCommitBinding(t *testing.T) {
	cases := []struct {
		name     string
		evidence string
		given    string
		wantErr  bool
	}{
		{"exact", acceptedCommit, acceptedCommit, false},
		{"evidence is a prefix of the commit", "9f3c1a2b", acceptedCommit, false},
		{"commit is a prefix of the evidence", acceptedCommit, "9f3c1a2b", false},
		{"case-insensitive", strings.ToUpper(acceptedCommit), acceptedCommit, false},
		{"mismatch", "dead0000beef1111222233334444555566667777", acceptedCommit, true},
		{"prefix too short to bind", "9f3c1a", acceptedCommit, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			evidence := strings.Replace(acceptEvidenceM0, "commit: "+acceptedCommit, "commit: "+tc.evidence, 1)
			design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": evidence})
			g := CheckAcceptance(design, tc.given)
			joined := strings.Join(g.Errs, "\n")
			switch {
			case tc.wantErr && !strings.Contains(joined, "does not name the commit under review"):
				t.Fatalf("want a commit-binding error, got %v", g.Errs)
			case !tc.wantErr && len(g.Errs) != 0:
				t.Fatalf("commit must bind: %v", g.Errs)
			}
		})
	}
}

func TestCheckAcceptanceClosedWithoutEvidence(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": ""})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "acceptance/M0.yaml is not committed") {
		t.Fatalf("a closed milestone with no evidence must fail: %v", g.Errs)
	}
}

func TestCheckAcceptanceClosedWithRejectedEvidence(t *testing.T) {
	rejected := strings.Replace(acceptEvidenceM0, "verdict: ACCEPTED", "verdict: REJECTED", 1)
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": rejected})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "records verdict REJECTED") {
		t.Fatalf("a rejected review must not close a milestone: %v", g.Errs)
	}
}

func TestCheckAcceptanceEvidenceMutations(t *testing.T) {
	cases := []struct {
		name     string
		evidence string
		want     string
	}{
		{"invalid YAML", "milestone: 0\n  commit: x\n", "invalid YAML"},
		{"not a mapping", "- milestone: 0\n", "not a yaml mapping"},
		{"unknown key", acceptEvidenceM0 + "notes: extra\n", "unknown key 'notes'"},
		{"missing fields",
			strings.Replace(acceptEvidenceM0, "reviewer: acceptance review, conductor\n", "", 1),
			"missing required field(s): reviewer"},
		{"lower-case verdict",
			strings.Replace(acceptEvidenceM0, "verdict: ACCEPTED", "verdict: accepted", 1),
			"it is exactly ACCEPTED or REJECTED"},
		{"milestone disagrees with the file name",
			strings.Replace(acceptEvidenceM0, "milestone: 0", "milestone: 1", 1),
			"but the file names M0"},
		{"milestone is not an integer",
			strings.Replace(acceptEvidenceM0, "milestone: 0", "milestone: zero", 1),
			"milestone must be an integer"},
		{"empty commit",
			strings.Replace(acceptEvidenceM0, "commit: "+acceptedCommit, `commit: ""`, 1),
			"commit must name the single VCS commit"},
		{"empty reviewer",
			strings.Replace(acceptEvidenceM0, "reviewer: acceptance review, conductor", `reviewer: ""`, 1),
			"reviewer must name who or what produced the review"},
		{"malformed date",
			strings.Replace(acceptEvidenceM0, "date: 2026-08-27", "date: 27-08-2026", 1),
			"is not YYYY-MM-DD"},
		{"impossible date",
			strings.Replace(acceptEvidenceM0, "date: 2026-08-27", "date: 2026-02-31", 1),
			"is not a real calendar date"},
		{"accepted with no attestations",
			strings.Replace(acceptEvidenceM0,
				"attestations:\n  - integration tests ran against the real datastore, no mocks\n  - every guard branch has its falsifying test\n",
				"attestations: []\n", 1),
			"attests nothing"},
		{"dod_ids is not a list",
			strings.Replace(acceptEvidenceM0, "dod_ids:\n  - T-CMD-01\n  - CMD-abc123\n", "dod_ids: T-CMD-01\n", 1),
			"dod_ids must be a list of strings"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": tc.evidence})
			g := CheckAcceptance(design, acceptedCommit)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want an error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

// The DoD-id coverage rule is the deterministic proof the review looked at
// the right obligations: an id the DoD cites and the evidence omits is an
// ERROR naming the id.
func TestCheckAcceptanceDoDIDCoverage(t *testing.T) {
	partial := strings.Replace(acceptEvidenceM0, "  - CMD-abc123\n", "", 1)
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": partial})
	g := CheckAcceptance(design, acceptedCommit)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "dod_ids omits 'CMD-abc123'") {
		t.Fatalf("want the omitted id named, got %v", g.Errs)
	}
	if g.Counts["DoD ids bound"] != 1 {
		t.Errorf("the id that WAS bound must still count: %+v", g.Counts)
	}
}

// Only ids cited at or after the DoD token count, exactly as in Gb's
// skeleton-citation rule: pre-DoD prose is not an obligation.
func TestCheckAcceptanceIgnoresIDsBeforeTheDoD(t *testing.T) {
	plan := strings.Replace(acceptPlan,
		"**M0 - Walking skeleton.** One real transition through one real boundary.\nDoD: T-CMD-01 and CMD-abc123 green.",
		"**M0 - Walking skeleton.** Covers T-CMD-02 in passing.\nDoD: T-CMD-01 and CMD-abc123 green.", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan})
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("an id cited before the DoD is not an obligation: %v", g.Errs)
	}
}

func TestCheckAcceptanceUnknownMilestone(t *testing.T) {
	evidence := strings.Replace(acceptEvidenceM0, "milestone: 0", "milestone: 9", 1)
	design := writeAcceptFixture(t, map[string]string{
		"acceptance/M0.yaml": "",
		"acceptance/M9.yaml": evidence,
	})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "names milestone M9, which no build-plan document declares") {
		t.Fatalf("evidence for a phantom milestone must fail: %v", g.Errs)
	}
}

// The directory holds exactly M<n>.yaml files (plus a human index): anything
// else is a finding, never quietly ignored.
func TestCheckAcceptanceStrayDirectoryEntries(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0-round2.yaml": acceptEvidenceM0})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "acceptance/M0-round2.yaml is not acceptance evidence") {
		t.Fatalf("a stray file must be named: %v", g.Errs)
	}

	indexed := writeAcceptFixture(t, map[string]string{"acceptance/README.md": "# acceptance evidence\n"})
	gi := CheckAcceptance(indexed, acceptedCommit)
	if len(gi.Errs) != 0 {
		t.Fatalf("a README index is exempt: %v", gi.Errs)
	}
	if !strings.Contains(strings.Join(gi.checkedExtra, ", "), "1 index files exempt") {
		t.Errorf("the exemption must stay visible in the checked line: %v", gi.checkedExtra)
	}
}

// A design with no BUILD.md has no milestones to discharge; the gate says so
// instead of reporting a vacuous pass.
func TestCheckAcceptanceWithoutBuildDoc(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": ""})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "no BUILD.md in the design") {
		t.Fatalf("want the missing-plan error, got %v", g.Errs)
	}
}

// Forcing the gate on a design that closed nothing and committed no evidence
// is an error naming both missing things, the way forcing gp/gi/gn is.
func TestCheckAcceptanceForcedWithoutArtifacts(t *testing.T) {
	plan := strings.Replace(acceptPlan, "Status: closed\n", "", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan, "acceptance/M0.yaml": ""})
	if AcceptanceActive(design) {
		t.Fatal("neither artifact exists; Ga must not auto-activate")
	}
	g := CheckAcceptance(design, "")
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"no acceptance/ directory", "no milestone marked 'Status: closed'"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the forced-gate error must name %q, got %v", want, g.Errs)
		}
	}
}

// An acceptance directory with nothing in it, on a design that closed
// nothing, is an empty check: a failure, not a pass.
func TestCheckAcceptanceEmptyDirectoryFails(t *testing.T) {
	plan := strings.Replace(acceptPlan, "Status: closed\n", "", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan, "acceptance/M0.yaml": ""})
	if err := os.MkdirAll(filepath.Join(design, "acceptance"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !AcceptanceActive(design) {
		t.Fatal("the acceptance directory alone must activate Ga")
	}
	g := CheckAcceptance(design, "")
	if !strings.Contains(strings.Join(g.Errs, "\n"), "an empty check is a failure") {
		t.Fatalf("want the empty-check error, got %v", g.Errs)
	}
}

// Evidence for an OPEN milestone is legitimate (a review that has not closed
// anything yet) and still gets its parse and coverage checks.
func TestCheckAcceptanceEvidenceForOpenMilestone(t *testing.T) {
	plan := strings.Replace(acceptPlan, "Status: closed\n", "", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan})
	if !AcceptanceActive(design) {
		t.Fatal("the acceptance directory alone must activate Ga")
	}
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("evidence for an open milestone is not a finding: %v", g.Errs)
	}
	if g.Counts["closed milestones"] != 0 || g.Counts["acceptance files"] != 1 {
		t.Errorf("counts must show evidence with nothing closed: %+v", g.Counts)
	}
}

// --- manifest mode -------------------------------------------------------

const acceptManifestRoot = `# BUILD: thing

Mode: manifest

## 9. Build plan

**M0 - Walking skeleton.** One real transition through one real boundary.
DoD: T-CMD-01 and CMD-abc123 green.
Status: closed

**M1 - Orders slice.** DoD: T-CMD-02 green.
`

const acceptShardWaived = `# BUILD: orders

## 5. Behavior

Prose.

## 9. Build plan

N/A - the build plan is the root BUILD.md section 9; this shard carries sections 5 to 8.
`

// The production shape that went unchecked: every shard waives its Build plan
// toward the root, and the root carries the real milestones. Ga must bind
// evidence to the ROOT's milestones.
func TestCheckAcceptanceManifestRootPlan(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{
		"BUILD.md":             acceptManifestRoot,
		"BUILD/orders.md":      acceptShardWaived,
		"BUILD/payments.md":    strings.Replace(acceptShardWaived, "orders", "payments", 1),
		"acceptance/M0.yaml":   acceptEvidenceM0,
		"machines/T.oracle.md": acceptOracleMD,
	})
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("Ga not clean on a root-plan manifest: %v", g.Errs)
	}
	if g.Counts["plan documents"] != 1 || g.Counts["declared milestones"] != 2 || g.Counts["closed milestones"] != 1 {
		t.Errorf("the root plan is the plan-bearing document: %+v", g.Counts)
	}
}

const acceptShardPlanned = `# BUILD: orders

## 9. Build plan

Walking skeleton: N/A - the design-wide skeleton lives in the payments shard.

**M2 - Orders slice.** DoD: T-CMD-02 green.
Status: closed
`

// Shards may carry the plans instead; milestone numbers must then be unique
// across every plan-bearing document, because evidence is keyed by number.
func TestCheckAcceptanceManifestShardPlans(t *testing.T) {
	root := "# BUILD: thing\n\nMode: manifest\n\n## 1. Purpose\n\nProse.\n"
	evidence := strings.Replace(
		strings.Replace(acceptEvidenceM0, "milestone: 0", "milestone: 2", 1),
		"dod_ids:\n  - T-CMD-01\n  - CMD-abc123\n", "dod_ids:\n  - T-CMD-02\n", 1)
	design := writeAcceptFixture(t, map[string]string{
		"BUILD.md":           root,
		"BUILD/orders.md":    acceptShardPlanned,
		"acceptance/M0.yaml": "",
		"acceptance/M2.yaml": evidence,
	})
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("Ga not clean on a shard-plan manifest: %v", g.Errs)
	}
	if g.Counts["closed milestones with accepted evidence"] != 1 {
		t.Errorf("the shard's closed milestone must be discharged: %+v", g.Counts)
	}
}

func TestCheckAcceptanceManifestDuplicateMilestoneNumbers(t *testing.T) {
	root := "# BUILD: thing\n\nMode: manifest\n\n## 1. Purpose\n\nProse.\n"
	design := writeAcceptFixture(t, map[string]string{
		"BUILD.md":           root,
		"BUILD/orders.md":    acceptShardPlanned,
		"BUILD/payments.md":  strings.Replace(acceptShardPlanned, "Orders slice", "Payments slice", 1),
		"acceptance/M0.yaml": "",
		"acceptance/M2.yaml": strings.Replace(acceptEvidenceM0, "milestone: 0", "milestone: 2", 1),
	})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "declared in both orders.md and payments.md") {
		t.Fatalf("a repeated milestone number must be ambiguous, got %v", g.Errs)
	}
}

// --- suite wiring --------------------------------------------------------

func TestSelectAndRunSelectedWireGa(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["ga"] {
		t.Fatalf("ga must be in the default vocabulary: %v", sel.Run)
	}
	var titles []string
	for _, g := range RunSelected(design, "", sel, RunOptions{Commit: acceptedCommit}) {
		titles = append(titles, g.Title)
	}
	joined := strings.Join(titles, "\n")
	if !strings.Contains(joined, "Ga-accept") {
		t.Fatalf("Ga must auto-activate on committed acceptance evidence:\n%s", joined)
	}
	if gb, ga := strings.Index(joined, "Gb-plan"), strings.Index(joined, "Ga-accept"); gb < 0 || ga < gb {
		t.Errorf("Ga must run after Gb:\n%s", joined)
	}
}

func TestRunSelectedSkipsGaWithoutAcceptance(t *testing.T) {
	plan := strings.Replace(acceptPlan, "Status: closed\n", "", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan, "acceptance/M0.yaml": ""})
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range RunSelected(design, "", sel, RunOptions{}) {
		if strings.Contains(g.Title, "Ga-accept") {
			t.Fatal("Ga must not run on a design that has discharged nothing")
		}
	}
}

// The closed marker alone activates the gate, with no acceptance directory in
// the tree: that is exactly the case the gate exists to fail.
func TestRunSelectedActivatesGaOnClosedMarkerAlone(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": ""})
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, g := range RunSelected(design, "", sel, RunOptions{}) {
		if strings.Contains(g.Title, "Ga-accept") {
			found = true
			if len(g.Errs) == 0 {
				t.Error("a closed milestone with no evidence must fail")
			}
		}
	}
	if !found {
		t.Fatal("the closed marker alone must activate Ga")
	}
}

// The machine-less decomposed-parent narrowing keeps Ga when the parent
// carries acceptance artifacts, and names it in the note.
func TestSelectNarrowingKeepsGaWhenAcceptanceExists(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	writeSuiteFile(t, filepath.Join(design, "decomposition.yaml"), "decomposition_version: 1\n")
	sel, err := Select(design, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !sel.Run["ga"] {
		t.Errorf("narrowing dropped ga although acceptance evidence exists: %v", sel.Run)
	}
	if !strings.Contains(sel.Note, "gb,ga,g5") {
		t.Errorf("the note must list ga after gb: %q", sel.Note)
	}
}
