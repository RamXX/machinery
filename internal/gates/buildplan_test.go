package gates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// planOracleMD is a minimal committed transition oracle: the citation corpus
// for the skeleton check reads BOTH id columns from the file.
const planOracleMD = `# Generated transition oracle: thing

## Transitions

| test id | stable id | source | trigger | guard | target | actions |
|---|---|---|---|---|---|---|
| T-CMD-01 | CMD-abc123 | A | on:go | - | B | - |
| T-CMD-02 | CMD-def456 | B | on:stop | - | A | - |
`

// goCrmStylePlan replicates go-crm's paragraph-style plan: the skeleton
// citation appears only inside a comma run ("T-CMD-01,03,12"), which still
// contains T-CMD-01 as a whole token because "," is a boundary.
const goCrmStylePlan = `# BUILD: thing

Mode: full (single BUILD.md)

## 9. Build plan

Walking skeleton first, then vertical slices.

**M0 - Walking skeleton (thinnest end-to-end thread).** Implement exactly one
path through every boundary once. DoD: green for T-CMD-01,03,12; the token is
written and re-resolved on the next command.

**M1 - Breadth slice.** Everything else. DoD: all rows green.

## 10. Language realization notes

Prose.
`

// checkoutStylePlan replicates checkout-orders' bullet-style plan: the
// skeleton cites a stable id in prose and again in the DoD line.
const checkoutStylePlan = `# BUILD

Mode: full

## 9. Build plan

Walking skeleton first, then vertical slices, each fully green before the next.

- **M0 - Walking skeleton.** One real transition through one real boundary
  (stable id CMD-abc123). DoD: C-DB-01, C-BUS-01, and CMD-abc123 green.
- **M1 - Settlement slice.** DoD: all oracle rows green by stable id.
`

func writeBuildPlanFixture(t *testing.T, build string, extra map[string]string) string {
	t.Helper()
	design := t.TempDir()
	files := map[string]string{
		"BUILD.md":                 build,
		"machines/Thing.oracle.md": planOracleMD,
	}
	for name, content := range extra {
		if content == "" {
			delete(files, name)
			continue
		}
		files[name] = content
	}
	for name, content := range files {
		writeSuiteFile(t, filepath.Join(design, name), content)
	}
	return design
}

func TestCheckBuildPlanGoCrmShape(t *testing.T) {
	design := writeBuildPlanFixture(t, goCrmStylePlan, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || len(g.Drift) != 0 {
		t.Fatalf("Gb not clean: errs=%v drift=%v", g.Errs, g.Drift)
	}
	want := map[string]int{"plans": 1, "milestones": 2, "DoD-bearing milestones": 2, "skeleton citations": 1}
	for count, n := range want {
		if g.Counts[count] != n {
			t.Errorf("Gb counted %s=%d, want %d: %+v", count, g.Counts[count], n, g.Counts)
		}
	}
}

func TestCheckBuildPlanCheckoutShape(t *testing.T) {
	design := writeBuildPlanFixture(t, checkoutStylePlan, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("Gb not clean on the bullet shape: %v", g.Errs)
	}
	if g.Counts["skeleton citations"] != 1 {
		t.Errorf("skeleton citations = %d, want 1 (CMD-abc123): %+v", g.Counts["skeleton citations"], g.Counts)
	}
}

func TestCheckBuildPlanMutations(t *testing.T) {
	cases := []struct {
		name  string
		build string
		want  string
	}{
		{"missing section",
			strings.Replace(goCrmStylePlan, "## 9. Build plan", "## 9. Rollout", 1),
			"no Build plan section"},
		{"bare N/A waiver",
			"# B\n\nMode: full\n\n## Build plan\n\nN/A\n",
			"bare N/A"},
		{"no milestones",
			"# B\n\nMode: full\n\n## Build plan\n\nJust do the work in order.\n",
			"no milestone markers"},
		{"duplicate milestone numbers",
			strings.Replace(goCrmStylePlan, "**M1 - Breadth slice.**", "**M0 - Breadth slice.**", 1),
			"milestone M0 is declared 2 times"},
		{"missing DoD",
			strings.Replace(goCrmStylePlan, "Everything else. DoD: all rows green.", "Everything else, make it green.", 1),
			"milestone M1 (Breadth slice) states no definition of done"},
		{"skeleton not first",
			strings.Replace(goCrmStylePlan, "Walking skeleton (thinnest end-to-end thread)", "Data layer", 1),
			"is not the walking skeleton"},
		{"skeleton cites no oracle id",
			strings.Replace(goCrmStylePlan, "green for T-CMD-01,03,12", "green for the skeleton tests", 1),
			"cites no committed oracle id"},
		{"hyphen does not create a citation",
			strings.Replace(goCrmStylePlan, "green for T-CMD-01,03,12", "green for X-CMD-abc123", 1),
			"cites no committed oracle id"},
		{"manifest with nothing behind it",
			"# B\n\nMode: manifest\n",
			"a manifest with nothing behind it"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeBuildPlanFixture(t, tc.build, nil)
			g := CheckBuildPlan(design)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestCheckBuildPlanSectionWaiver(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\nN/A - the plan lives in the parent manifest\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a waived plan with a reason must pass: %v", g.Errs)
	}
	if g.Counts["waived plans"] != 1 || g.Counts["milestones"] != 0 {
		t.Errorf("waiver must skip the structural checks: %+v", g.Counts)
	}
}

func TestCheckBuildPlanSkeletonWaiver(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"walking skeleton: N/A - single pure library, no topology to prove\n\n" +
		"**M1 - Core slice.** DoD: all rows green.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a waived skeleton with a reason must pass: %v", g.Errs)
	}
	if g.Counts["skeleton waivers"] != 1 {
		t.Errorf("skeleton waivers = %d, want 1: %+v", g.Counts["skeleton waivers"], g.Counts)
	}
	// the citation check is skipped entirely, and says so in the checked line
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "skeleton citation skipped (skeleton waived)") {
		t.Errorf("checked line must record the skipped citation check: %v", g.checkedExtra)
	}
}

func TestCheckBuildPlanUnderscoreCitations(t *testing.T) {
	// a Go subtest name literal cites both ids: underscore is a boundary
	build := strings.Replace(goCrmStylePlan,
		"green for T-CMD-01,03,12", "green for T-CMD-01_CMD-abc123", 1)
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("Gb not clean: %v", g.Errs)
	}
	if g.Counts["skeleton citations"] != 2 {
		t.Errorf("skeleton citations = %d, want 2 (test id + stable id): %+v", g.Counts["skeleton citations"], g.Counts)
	}
}

func TestCheckBuildPlanNoCommittedOracles(t *testing.T) {
	design := writeBuildPlanFixture(t, goCrmStylePlan, map[string]string{"machines/Thing.oracle.md": ""})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("no committed oracles must skip the citation check, not fail it: %v", g.Errs)
	}
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "no committed oracles") {
		t.Errorf("checked line must record why the citation check was skipped: %v", g.checkedExtra)
	}
}

func TestCheckBuildPlanNoModeLineIsFullMode(t *testing.T) {
	build := strings.Replace(goCrmStylePlan, "Mode: full (single BUILD.md)\n", "", 1)
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 || g.Counts["milestones"] != 2 {
		t.Fatalf("an absent Mode line is full mode (Gx owns the Mode finding): errs=%v counts=%+v", g.Errs, g.Counts)
	}
}

func TestCheckBuildPlanManifestShards(t *testing.T) {
	shardOK := "# Shard\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n"
	shardBad := "# Shard\n\n## Build plan\n\n**M0 - Walking skeleton.** No definition here.\n"
	design := writeBuildPlanFixture(t, "# B\n\nMode: manifest\n", map[string]string{
		"BUILD/core.md": shardOK,
		"BUILD/edge.md": shardBad,
	})
	g := CheckBuildPlan(design)
	if g.Counts["plans"] != 2 {
		t.Errorf("plans = %d, want 2 (one per shard): %+v", g.Counts["plans"], g.Counts)
	}
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "edge.md: milestone M0") {
		t.Errorf("shard findings must carry the shard name: %v", g.Errs)
	}
	if strings.Contains(joined, "core.md") {
		t.Errorf("the clean shard must not be flagged: %v", g.Errs)
	}
}

// Manifest shards get the full check including the skeleton citation, held
// against the design-wide committed oracle ids (the same corpus as full mode).
func TestCheckBuildPlanManifestShardsRequireSkeletonCitation(t *testing.T) {
	shardCiting := "# Core\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n"
	shardBare := "# Edge\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: skeleton green.\n"
	design := writeBuildPlanFixture(t, "# B\n\nMode: manifest\n", map[string]string{
		"BUILD/core.md": shardCiting,
		"BUILD/edge.md": shardBare,
	})
	g := CheckBuildPlan(design)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "edge.md") || !strings.Contains(joined, "cites no committed oracle id") {
		t.Fatalf("a shard whose skeleton DoD cites no oracle id must fail: %v", g.Errs)
	}
	if strings.Contains(joined, "core.md") {
		t.Errorf("the citing shard must pass: %v", g.Errs)
	}
	if g.Counts["skeleton citations"] != 1 {
		t.Errorf("skeleton citations = %d, want 1 (core.md): %+v", g.Counts["skeleton citations"], g.Counts)
	}
}

// README.md and index.md under BUILD/ are navigation for humans, not plan
// shards: they carry no plan obligation, the exemption is visible in the
// checked line, and the real shard is still fully checked.
func TestCheckBuildPlanManifestExemptsIndexFiles(t *testing.T) {
	shard := "# Core\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n"
	design := writeBuildPlanFixture(t, "# B\n\nMode: manifest\n", map[string]string{
		"BUILD/README.md": "# Shard index\n\nWho builds what.\n",
		"BUILD/core.md":   shard,
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("an index file must carry no plan obligation: %v", g.Errs)
	}
	if g.Counts["plans"] != 1 || g.Counts["milestones"] != 1 {
		t.Errorf("the real shard must still be fully checked: %+v", g.Counts)
	}
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "1 index files exempt") {
		t.Errorf("checked line must show the exemption: %v", g.checkedExtra)
	}
}

func TestCheckBuildPlanManifestParentWithoutShards(t *testing.T) {
	design := writeBuildPlanFixture(t, "# B\n\nMode: manifest\n", map[string]string{
		"decomposition.yaml":       "decomposition_version: 1\n",
		"machines/Thing.oracle.md": "",
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a decomposed manifest parent carries no local plan and must pass: %v", g.Errs)
	}
	if !strings.Contains(strings.Join(g.checkedExtra, ", "), "0 local plans") {
		t.Errorf("checked line must show the 0 local plans explicitly: %v", g.checkedExtra)
	}
}

func TestCheckBuildPlanExplicitWithoutBuildDoc(t *testing.T) {
	design := t.TempDir()
	if HasBuildDoc(design) {
		t.Fatal("HasBuildDoc on an empty design must be false")
	}
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "no BUILD.md") {
		t.Fatalf("a requested gate with no artifact is an error, not a pass: %v", g.Errs)
	}
}

// Fenced code blocks are opaque to every Gb scan: a bash "# comment" is not
// a heading that truncates the section, and a fenced fake milestone or DoD
// line is not plan structure.
func TestCheckBuildPlanFenceAware(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** Run the smoke script:\n\n" +
		"```bash\n# comment that is not a heading\n**M9 - fake milestone**\nDoD: fake\n```\n\n" +
		"DoD: T-CMD-01 green.\n\n" +
		"**M1 - Breadth slice.** DoD: all rows green.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("fence content must be invisible to the plan scans: %v", g.Errs)
	}
	if g.Counts["milestones"] != 2 {
		t.Errorf("milestones = %d, want 2 (the fenced M9 is not one): %+v", g.Counts["milestones"], g.Counts)
	}
	if g.Counts["skeleton citations"] != 1 {
		t.Errorf("skeleton citations = %d, want 1 (T-CMD-01 after the real DoD): %+v", g.Counts["skeleton citations"], g.Counts)
	}
	// a DoD that lives only inside a fence is no DoD
	fencedDoD := strings.Replace(build, "\nDoD: T-CMD-01 green.\n", "\nStill no definition outside the fence.\n", 1)
	g = CheckBuildPlan(writeBuildPlanFixture(t, fencedDoD, nil))
	if !strings.Contains(strings.Join(g.Errs, "\n"), "milestone M0 (Walking skeleton) states no definition of done") {
		t.Fatalf("a fenced DoD must not satisfy the DoD check: %v", g.Errs)
	}
}

// A bold cross-reference mid-prose ("built by **M0 - Walking skeleton**
// above") is not a milestone declaration: markers are anchored per line.
func TestCheckBuildPlanBoldCrossReferenceIsNotAMilestone(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n" +
		"**M1 - Breadth slice.** Reuses the fixtures built by **M0 - Walking skeleton** above. DoD: all rows green.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("a mid-prose cross-reference must not declare a phantom milestone: %v", g.Errs)
	}
	if g.Counts["milestones"] != 2 {
		t.Errorf("milestones = %d, want 2: %+v", g.Counts["milestones"], g.Counts)
	}
}

// A milestone block ends at the next heading of any level, so a trailing
// subsection cannot donate a DoD, and the mid-sentence phrase "definition of
// done" without a colon is not a DoD token.
func TestCheckBuildPlanDoDTokenAndBlockEnd(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n" +
		"**M1 - Breadth slice.** Everything else, following the team's definition of done conventions.\n\n" +
		"### Notes\n\nDoD: this subsection is not part of M1.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "milestone M1 (Breadth slice) states no definition of done") {
		t.Fatalf("neither the prose phrase nor the trailing subsection is a DoD: %v", g.Errs)
	}
	// the long form counts when the colon makes it a token
	withColon := strings.Replace(build,
		"following the team's definition of done conventions.",
		"Definition of done: all rows green.", 1)
	g = CheckBuildPlan(writeBuildPlanFixture(t, withColon, nil))
	if len(g.Errs) != 0 {
		t.Fatalf("'Definition of done:' is a valid DoD token: %v", g.Errs)
	}
}

// The skeleton citation must appear at or after the DoD token: ids cited only
// in pre-DoD prose with a vacuous DoD are exactly what the docs promise Gb
// rejects.
func TestCheckBuildPlanSkeletonCitationMustFollowDoD(t *testing.T) {
	bad := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** Proves T-CMD-01 and CMD-abc123 end to end. DoD: it works.\n\n" +
		"**M1 - Breadth slice.** DoD: all rows green.\n"
	design := writeBuildPlanFixture(t, bad, nil)
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "cites no committed oracle id") {
		t.Fatalf("ids only in pre-DoD prose must not satisfy the citation: %v", g.Errs)
	}
	good := strings.Replace(bad, "DoD: it works.", "DoD: T-CMD-01 green.", 1)
	g = CheckBuildPlan(writeBuildPlanFixture(t, good, nil))
	if len(g.Errs) != 0 {
		t.Fatalf("an id after the DoD token must pass: %v", g.Errs)
	}
	if g.Counts["skeleton citations"] != 1 {
		t.Errorf("skeleton citations = %d, want 1: %+v", g.Counts["skeleton citations"], g.Counts)
	}
}

// The mode sniff runs on fence-masked text: a fenced example "Mode: manifest"
// line must not override the real declaration (NG-4, both directions).
func TestCheckBuildPlanModeSniffIsFenceMasked(t *testing.T) {
	t.Run("fenced manifest does not hide a full plan", func(t *testing.T) {
		build := "# B\n\nAn example of the sharded declaration:\n\n" +
			"```text\nMode: manifest (shards under BUILD/)\n```\n\n" +
			"Mode: full (single BUILD.md)\n\n## 9. Build plan\n\n" +
			"**M0 - Data layer.** No definition of done here and not a walking skeleton.\n"
		design := writeBuildPlanFixture(t, build, map[string]string{
			"decomposition.yaml": "decomposition_version: 1\n",
		})
		g := CheckBuildPlan(design)
		if g.Counts["milestones"] != 1 {
			t.Fatalf("the real Mode: full plan must be structurally checked: %+v", g.Counts)
		}
		joined := strings.Join(g.Errs, "\n")
		if !strings.Contains(joined, "states no definition of done") || !strings.Contains(joined, "is not the walking skeleton") {
			t.Fatalf("the broken full plan must fail its structural checks: %v", g.Errs)
		}
	})
	t.Run("fenced full does not hide a manifest declaration", func(t *testing.T) {
		build := "# B\n\n```text\nMode: full (single BUILD.md)\n```\n\nMode: manifest\n"
		design := writeBuildPlanFixture(t, build, nil)
		g := CheckBuildPlan(design)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "a manifest with nothing behind it") {
			t.Fatalf("the real Mode: manifest must govern: %v", g.Errs)
		}
	})
}

// Fences follow the CommonMark run-length rule: a fence opened with N
// backticks closes only on a line of >= N backticks of the same character, so
// a 4-backtick documentation fence swallows its inner ``` lines instead of
// leaking phantom DoD lines (NG-5).
func TestMaskFencesRunLength(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n" +
		"**M1 - Breadth slice.** This milestone has NO real DoD. Example snippet:\n\n" +
		"````markdown\n```text\nDoD: example text inside a documentation fence, not a commitment.\n```\n````\n\n" +
		"More prose, still no definition of done for M1.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "milestone M1 (Breadth slice) states no definition of done") {
		t.Fatalf("the fenced example DoD must not satisfy M1: %v", g.Errs)
	}
	masked := maskFences("````\ncontent\n```\nstill content\n````\nvisible\n")
	if strings.Contains(masked, "content") {
		t.Errorf("inner ``` must not close a 4-backtick fence: %q", masked)
	}
	if !strings.Contains(masked, "visible") {
		t.Errorf("text after the true closer must survive: %q", masked)
	}
}

// The section waiver is the documented literal form 'N/A - <reason>': prose
// that merely starts with N/A must not waive the whole structural check
// (NG-6).
func TestCheckBuildPlanNAWaiverLiteralFormOnly(t *testing.T) {
	cases := []struct{ name, first string }{
		{"prose starting with N/A", "N/A rows in the oracle table are excluded from milestone scoping."},
		{"colon separator", "N/A: the plan lives in the parent manifest"},
		{"lowercase", "n/a - the plan lives in the parent manifest"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := "# B\n\nMode: full\n\n## Build plan\n\n" + tc.first + "\n"
			design := writeBuildPlanFixture(t, build, nil)
			g := CheckBuildPlan(design)
			if len(g.Errs) == 0 {
				t.Fatalf("a non-literal waiver form must not pass silently: %+v", g.Counts)
			}
			if g.Counts["waived plans"] != 0 {
				t.Errorf("no waiver may be counted for %q: %+v", tc.first, g.Counts)
			}
		})
	}
}

// Milestone numbers are compared numerically: M1 and M01 are the same
// milestone declared twice (NG-8).
func TestCheckBuildPlanDuplicateMilestoneNumbersNumeric(t *testing.T) {
	build := "# B\n\nMode: full\n\n## Build plan\n\n" +
		"**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n\n" +
		"**M1 - First slice.** DoD: rows green.\n\n" +
		"**M01 - Also the first slice.** DoD: rows green.\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "milestone M1 is declared 2 times") {
		t.Fatalf("M1 and M01 must collide numerically: %v", g.Errs)
	}
}

// An artifact that exists but cannot be read is a hard ERROR naming the
// file, never silently treated as empty (NG-9).
func TestUnreadableArtifactsAreHardErrors(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not deny reads")
	}
	t.Run("Gb BUILD.md", func(t *testing.T) {
		design := writeBuildPlanFixture(t, goCrmStylePlan, nil)
		path := filepath.Join(design, "BUILD.md")
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
		g := CheckBuildPlan(design)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "BUILD.md is unreadable") {
			t.Fatalf("an unreadable BUILD.md must be a hard error naming the file: %v", g.Errs)
		}
	})
	t.Run("Gt committed oracle", func(t *testing.T) {
		design, impl := writeCovFixture(t, map[string]string{
			"machines/Thing.oracle.md": covOracleMD,
			"impl/thing_test.go":       "package thing\n\n// THIN-aaa111 THIN-bbb222\n",
		})
		path := filepath.Join(design, "machines", "Thing.oracle.md")
		if err := os.Chmod(path, 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
		g := CheckOracleCoverage(design, impl)
		if !strings.Contains(strings.Join(g.Errs, "\n"), "Thing.oracle.md is unreadable") {
			t.Fatalf("an unreadable oracle must be a hard error naming the file: %v", g.Errs)
		}
	})
}

func TestIDTokenIn(t *testing.T) {
	cases := []struct {
		token, text string
		want        bool
	}{
		// underscore IS a boundary: a Go subtest literal cites both ids
		{"T-DEAL-01", `t.Run("T-DEAL-01_DEAL-eb0c40", nil)`, true},
		{"DEAL-eb0c40", `t.Run("T-DEAL-01_DEAL-eb0c40", nil)`, true},
		// hyphen is NOT a boundary: a prefixed lookalike is a different id
		{"DEAL-eb0c40", "X-DEAL-eb0c40", false},
		{"T-CMD-01", "T-CMD-01,03,12", true},
		{"inv-1", "inv-12", false},
	}
	for _, tc := range cases {
		if got := idTokenIn(tc.token, tc.text); got != tc.want {
			t.Errorf("idTokenIn(%q, %q) = %v, want %v", tc.token, tc.text, got, tc.want)
		}
	}
}
