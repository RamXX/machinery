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
path through every boundary once. NFR: error envelope, logging. DoD: green for T-CMD-01,03,12; the token is
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
  (stable id CMD-abc123). NFR: error envelope. DoD: C-DB-01, C-BUS-01, and CMD-abc123 green.
- **M1 - Settlement slice.** DoD: all oracle rows green by stable id.
`

// templateSectionsStub carries every template section heading (except Build
// plan, which each fixture states itself) plus the verbatim disclaimer, so a
// fixture exercising one check is not red on the section-presence and
// disclaimer obligations that are not under test.
func templateSectionsStub() string {
	return "\n## Purpose and scope\nstub\n## Glossary\nstub\n## Domain model\nstub\n" +
		"## Architecture\nstub\n## Behavior\nstub\n## Traceability matrix\nstub\n" +
		"## Test specification\nstub\n## State migration\nstub\n" +
		"## Language realization notes\nstub\n## Hard-TDD protocol\nstub\n" +
		"## Open questions and residual risks\nstub\n### What the gates do not verify\n" +
		gatesDisclaimerText + "\n"
}

func executionPacket(num, title string) string {
	return "# M" + num + " - " + title + "\n\n" +
		"## Outcome\nObservable result.\n\n" +
		"## Domain context\nExact milestone domain slice.\n\n" +
		"## Architecture context\nOwned boundary and dependencies.\n\n" +
		"## Behavior and oracles\nCommitted transition rows.\n\n" +
		"## TDD and implementation\nRED then GREEN.\n\n" +
		"## Risks and recovery\nBounded failure and rollback.\n\n" +
		"## Acceptance\nExecutable evidence.\n"
}

func manifestPlan() string {
	return "# B\n\nMode: manifest\n\n## 9. Build plan\n\n" +
		"**M0 - Walking skeleton.**\nStatus: open\nPacket: [M0 execution packet](BUILD/M0-walking-skeleton.md)\n" +
		"Demo: Exercise one real transition through one real boundary.\nNFR: error envelope, logging.\nDoD: T-CMD-01 green.\n\n" +
		"**M1 - Breadth slice.**\nStatus: open\nPacket: [M1 execution packet](BUILD/M1-breadth.md)\n" +
		"Demo: Exercise the complete public workflow.\nDoD: all rows green.\n"
}

func writeBuildPlanFixture(t *testing.T, build string, extra map[string]string) string {
	t.Helper()
	design := t.TempDir()
	files := map[string]string{
		"BUILD.md":                 build + templateSectionsStub(),
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

func TestCheckBuildPlanDuplicateSectionsError(t *testing.T) {
	// two Build plan sections: the first-match locator once made the second
	// section's milestones invisible to Gb and Ga (a closed milestone there
	// never owed evidence). The ambiguity is refused loudly instead.
	build := goCrmStylePlan +
		"\n## Build plan (phase 2)\n\n**M7 - Later work.** DoD: T-CMD-02 green.\nStatus: closed\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	if !hasErr(g, "'Build plan' sections") {
		t.Fatalf("a second Build plan section must be an ERROR: %v", g.Errs)
	}
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
			"manifest mode requires the root to carry the single Build plan"},
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

func TestCheckBuildPlanManifestPackets(t *testing.T) {
	design := writeBuildPlanFixture(t, manifestPlan(), map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("Gb not clean on milestone packets: %v", g.Errs)
	}
	if g.Counts["plans"] != 1 || g.Counts["milestones"] != 2 || g.Counts["execution packets"] != 2 || g.Counts["packet sections"] != 14 {
		t.Errorf("one root plan and two complete packets: %+v", g.Counts)
	}
}

func TestCheckBuildPlanManifestPacketInventoryIsExact(t *testing.T) {
	root := strings.Replace(manifestPlan(), "BUILD/M1-breadth.md", "BUILD/M0-walking-skeleton.md", 1)
	design := writeBuildPlanFixture(t, root, map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/orphan.md":              executionPacket("1", "Orphan"),
	})
	g := CheckBuildPlan(design)
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "linked by 2 milestones") || !strings.Contains(joined, "BUILD/orphan.md: packet is not linked") {
		t.Fatalf("duplicate ownership and orphan inventory must both fail: %v", g.Errs)
	}
}

func TestCheckBuildPlanManifestPacketContract(t *testing.T) {
	cases := []struct {
		name, root, packet, want string
	}{
		{"missing packet line", strings.Replace(manifestPlan(), "Packet: [M0 execution packet](BUILD/M0-walking-skeleton.md)\n", "", 1), executionPacket("0", "Walking skeleton"), "has no Packet line"},
		{"invalid packet path", strings.Replace(manifestPlan(), "BUILD/M0-walking-skeleton.md", "BUILD/../escape.md", 1), executionPacket("0", "Walking skeleton"), "invalid packet path"},
		{"missing demo", strings.Replace(manifestPlan(), "Demo: Exercise one real transition through one real boundary.\n", "", 1), executionPacket("0", "Walking skeleton"), "has no Demo line"},
		{"wrong packet owner", manifestPlan(), executionPacket("7", "Wrong owner"), "exactly one first-level title identifying its owner"},
		{"missing packet section", manifestPlan(), strings.Replace(executionPacket("0", "Walking skeleton"), "## Risks and recovery\nBounded failure and rollback.\n\n", "", 1), "missing required '## Risks and recovery' section"},
		{"packet contains plan", manifestPlan(), executionPacket("0", "Walking skeleton") + "\n## Build plan\nNot allowed.\n", "keep milestone declarations and the Build plan only in BUILD.md"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			design := writeBuildPlanFixture(t, tc.root, map[string]string{
				"BUILD/M0-walking-skeleton.md": tc.packet,
				"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
			})
			g := CheckBuildPlan(design)
			if !strings.Contains(strings.Join(g.Errs, "\n"), tc.want) {
				t.Fatalf("want error containing %q, got %v", tc.want, g.Errs)
			}
		})
	}
}

func TestCheckBuildPlanManifestPacketBound(t *testing.T) {
	packet := executionPacket("0", "Walking skeleton") + strings.Repeat("x", maxExecutionPacketBytes)
	design := writeBuildPlanFixture(t, manifestPlan(), map[string]string{
		"BUILD/M0-walking-skeleton.md": packet,
		"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
	})
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "maximum is 65536 bytes") {
		t.Fatalf("an oversized packet must fail: %v", g.Errs)
	}
}

// README.md and index.md under BUILD/ are navigation for humans, not plan
// shards: they carry no plan obligation, the exemption is visible in the
// checked line, and the real shard is still fully checked.
func TestCheckBuildPlanManifestExemptsIndexFiles(t *testing.T) {
	design := writeBuildPlanFixture(t, manifestPlan(), map[string]string{
		"BUILD/README.md":              "# Shard index\n\nWho builds what.\n",
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("an index file must carry no plan obligation: %v", g.Errs)
	}
	if g.Counts["plans"] != 1 || g.Counts["execution packets"] != 2 {
		t.Errorf("the real packets must still be fully checked: %+v", g.Counts)
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
		"**M0 - Walking skeleton.** NFR: error envelope. Run the smoke script:\n\n" +
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
		"**M0 - Walking skeleton.** NFR: error envelope, logging. DoD: T-CMD-01 green.\n\n" +
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
		"**M0 - Walking skeleton.** NFR: error envelope, logging. DoD: T-CMD-01 green.\n\n" +
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
		"**M0 - Walking skeleton.** NFR: error envelope. Proves T-CMD-01 and CMD-abc123 end to end. DoD: it works.\n\n" +
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
		if !strings.Contains(strings.Join(g.Errs, "\n"), "manifest mode requires the root to carry the single Build plan") {
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
		"**M0 - Walking skeleton.** NFR: error envelope, logging. DoD: T-CMD-01 green.\n\n" +
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
		"**M0 - Walking skeleton.** NFR: error envelope, logging. DoD: T-CMD-01 green.\n\n" +
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

// The manifest root is the sole plan authority. Packets are context, not
// competing plans, so acceptance remains unambiguous for small executors.
func TestCheckBuildPlanManifestRootPlanIsChecked(t *testing.T) {
	design := writeBuildPlanFixture(t, manifestPlan(), map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("Gb not clean on a root-plan manifest: %v", g.Errs)
	}
	if g.Counts["plans"] != 1 || g.Counts["milestones"] != 2 || g.Counts["DoD-bearing milestones"] != 2 || g.Counts["skeleton citations"] != 1 {
		t.Errorf("the root plan must get the full structural check: %+v", g.Counts)
	}
}

// The same shape, with a defective root plan: the finding names BUILD.md.
func TestCheckBuildPlanManifestRootPlanFindingsNameTheRoot(t *testing.T) {
	root := "# B\n\nMode: manifest\n\n## 9. Build plan\n\n" +
		"**M0 - Walking skeleton.**\nPacket: [packet](BUILD/M0-walking-skeleton.md)\nDemo: run it.\nNFR: error envelope. No definition here.\n"
	design := writeBuildPlanFixture(t, root, map[string]string{"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton")})
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "BUILD.md: milestone M0") {
		t.Fatalf("the root plan's findings must name BUILD.md: %v", g.Errs)
	}
}

// Manifest milestones cannot exist without their execution packets.
func TestCheckBuildPlanManifestRootPlanWithoutShards(t *testing.T) {
	design := writeBuildPlanFixture(t, manifestPlan(), nil)
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "is not a regular non-index file in BUILD/") {
		t.Fatalf("a linked packet missing from disk must fail: %v", g.Errs)
	}
}

// A packet cannot become a second scheduling authority when the root omits
// its plan; the missing root plan fails directly.
func TestCheckBuildPlanManifestRootRequiresPlanSection(t *testing.T) {
	packet := executionPacket("0", "Walking skeleton")
	design := writeBuildPlanFixture(t, "# B\n\nMode: manifest\n\n## 9. Milestone map\n\nSee the shards.\n",
		map[string]string{"BUILD/M0-walking-skeleton.md": packet})
	g := CheckBuildPlan(design)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "manifest mode requires the root to carry the single Build plan") {
		t.Fatalf("manifest mode without a root plan must fail: %v", g.Errs)
	}
}

// The milestone status line is optional, parses in the decorated forms the
// markers themselves use, and counts. A typo in it is an ERROR: reading an
// unrecognized value as "not closed" would silently disarm Ga-accept.
func TestCheckBuildPlanMilestoneStatusLine(t *testing.T) {
	closed := strings.Replace(goCrmStylePlan,
		"**M1 - Breadth slice.** Everything else. DoD: all rows green.",
		"**M1 - Breadth slice.** Everything else. DoD: all rows green.\n\n**Status:** closed", 1)
	g := CheckBuildPlan(writeBuildPlanFixture(t, closed, nil))
	if len(g.Errs) != 0 {
		t.Fatalf("a bold status line must parse: %v", g.Errs)
	}
	if g.Counts["closed milestones"] != 1 {
		t.Errorf("closed milestones = %d, want 1: %+v", g.Counts["closed milestones"], g.Counts)
	}

	open := strings.Replace(closed, "**Status:** closed", "Status: open", 1)
	if g := CheckBuildPlan(writeBuildPlanFixture(t, open, nil)); len(g.Errs) != 0 || g.Counts["closed milestones"] != 0 {
		t.Errorf("an explicit open status is clean and closes nothing: errs=%v counts=%+v", g.Errs, g.Counts)
	}

	typo := strings.Replace(closed, "**Status:** closed", "Status: cloesd", 1)
	if g := CheckBuildPlan(writeBuildPlanFixture(t, typo, nil)); !strings.Contains(strings.Join(g.Errs, "\n"), "unrecognized status 'cloesd'") {
		t.Errorf("a typo in the status line must fail loudly: %v", g.Errs)
	}
}

// The plan heading carries a section number and whatever decoration the
// design adds; the phrase "Build plan" is what names the section. A real
// design titled it "## 9. Build plan (sealed trust layers; user directive
// 2026-08-04)": under an exact-title match the manifest root's plan was
// found by nothing, and a standalone design would have been told its plan
// section does not exist. One matcher holds both paths.
func TestCheckBuildPlanDecoratedHeadingStandalone(t *testing.T) {
	build := strings.Replace(goCrmStylePlan, "## 9. Build plan",
		"## 9. Build plan (sealed trust layers; user directive 2026-08-04)", 1)
	g := CheckBuildPlan(writeBuildPlanFixture(t, build, nil))
	if len(g.Errs) != 0 {
		t.Fatalf("a decorated plan heading must still be the plan section: %v", g.Errs)
	}
	want := map[string]int{"plans": 1, "milestones": 2, "DoD-bearing milestones": 2, "skeleton citations": 1}
	for count, n := range want {
		if g.Counts[count] != n {
			t.Errorf("Gb counted %s=%d, want %d: %+v", count, g.Counts[count], n, g.Counts)
		}
	}
}

func TestCheckBuildPlanDecoratedHeadingManifestRoot(t *testing.T) {
	root := strings.Replace(manifestPlan(), "## 9. Build plan", "## 9. Build plan (sealed trust layers; user directive 2026-08-04)", 1)
	design := writeBuildPlanFixture(t, root, map[string]string{
		"BUILD/M0-walking-skeleton.md": executionPacket("0", "Walking skeleton"),
		"BUILD/M1-breadth.md":          executionPacket("1", "Breadth slice"),
	})
	g := CheckBuildPlan(design)
	if len(g.Errs) != 0 {
		t.Fatalf("Gb not clean on a decorated root plan heading: %v", g.Errs)
	}
	if g.Counts["plans"] != 1 || g.Counts["execution packets"] != 2 || g.Counts["milestones"] != 2 {
		t.Errorf("the decorated root plan must be counted and checked: %+v", g.Counts)
	}
}

// The phrase must be whole, and only headings name a section: a "Build
// planning" heading, or the words in body prose, create no phantom plan.
func TestCheckBuildPlanHeadingPhraseIsWhole(t *testing.T) {
	cases := []struct {
		name  string
		build string
	}{
		{"Build planning heading",
			strings.Replace(goCrmStylePlan, "## 9. Build plan", "## 9. Build planning notes", 1)},
		{"Rebuild plan heading",
			strings.Replace(goCrmStylePlan, "## 9. Build plan", "## 9. Rebuild plan", 1)},
		{"prose only",
			strings.Replace(goCrmStylePlan, "## 9. Build plan",
				"## 9. Rollout\n\nBuild planning happens weekly; the build plan lives elsewhere.\n\n### Notes", 1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := CheckBuildPlan(writeBuildPlanFixture(t, tc.build, nil))
			if !strings.Contains(strings.Join(g.Errs, "\n"), "no Build plan section") {
				t.Fatalf("a near-miss heading must not become the plan section: %v", g.Errs)
			}
			if g.Counts["milestones"] != 0 {
				t.Errorf("no milestones may be parsed from a phantom section: %+v", g.Counts)
			}
		})
	}
}

// The skeleton milestone names the NFR mechanisms it instantiates on an
// 'NFR:' line; a missing or empty line is a plan defect (the template stated
// the requirement and, until now, that Gb did not check it).
func TestCheckBuildPlanSkeletonNFRLine(t *testing.T) {
	base := "# BUILD\n\nMode: full\n\n## 9. Build plan\n\n"
	design := writeBuildPlanFixture(t, base+"**M0 - Walking skeleton.** DoD: T-CMD-01 green.\n", nil)
	g := CheckBuildPlan(design)
	if !hasErr(g, "has no 'NFR:' line") {
		t.Fatalf("a skeleton without an NFR line must error: %v", g.Errs)
	}
	design = writeBuildPlanFixture(t, base+"**M0 - Walking skeleton.** NFR:\nDoD: T-CMD-01 green.\n", nil)
	g = CheckBuildPlan(design)
	if !hasErr(g, "empty 'NFR:' line") {
		t.Fatalf("an empty NFR line must error: %v", g.Errs)
	}
	design = writeBuildPlanFixture(t, base+"**M0 - Walking skeleton.** NFR: none - the record is all out of scope. DoD: T-CMD-01 green.\n", nil)
	g = CheckBuildPlan(design)
	if hasErr(g, "NFR") {
		t.Fatalf("a reasoned 'NFR: none' must pass: %v", g.Errs)
	}
	if g.Counts["skeleton NFR lines"] != 1 {
		t.Fatalf("NFR line not counted: %+v", g.Counts)
	}
}

// The gatesDisclaimerText constant must stay verbatim-equal (whitespace
// collapsed) to the block in references/build-md-template.md: the template is
// what authors copy, the constant is what the gate compares, and silent drift
// between them would fail every conforming design.
func TestGatesDisclaimerMatchesTemplate(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "skills", "machinery", "references", "build-md-template.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(collapseWS(string(body)), collapseWS(gatesDisclaimerText)) {
		t.Fatal("references/build-md-template.md no longer carries the exact disclaimer block gatesDisclaimerText compares against; update both together")
	}
}

// A reworded disclaimer names the canonical source, prints the first word
// that diverges with a window of each side, and says the check is the root's
// alone: the reader can find and fix the wording without reading Go source.
func TestCheckBuildPlanDisclaimerDivergenceIsReported(t *testing.T) {
	reworded := strings.Replace(gatesDisclaimerText, "the RIGHT invariants", "the CORRECT invariants", 1)
	if reworded == gatesDisclaimerText {
		t.Fatal("fixture no longer rewords the disclaimer")
	}
	build := strings.Replace(goCrmStylePlan+templateSectionsStub(), gatesDisclaimerText, reworded, 1)
	design := writeBuildPlanFixture(t, "", map[string]string{"BUILD.md": build})
	g := CheckBuildPlan(design)
	msg := ""
	for _, e := range g.Errs {
		if strings.Contains(e, "What the gates do not verify") {
			msg = e
		}
	}
	if msg == "" {
		t.Fatalf("a reworded disclaimer must be an ERROR: %v", g.Errs)
	}
	for _, want := range []string{
		"references/build-md-template.md",
		"pinned in this binary",
		"first divergence at word 16:",
		`expected "RIGHT invariants`,
		`found "CORRECT invariants`,
		"only the root BUILD.md is checked, packets are not",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("disclaimer finding omits %q: %s", want, msg)
		}
	}
}

// A body that is a different text entirely (the block simply absent under a
// present heading) still gets a divergence report, from word 1.
func TestCheckBuildPlanDisclaimerWhollyDifferentBodyReportsFromWordOne(t *testing.T) {
	build := strings.Replace(goCrmStylePlan+templateSectionsStub(), gatesDisclaimerText, "Nothing to declare here.", 1)
	design := writeBuildPlanFixture(t, "", map[string]string{"BUILD.md": build})
	g := CheckBuildPlan(design)
	if !hasErr(g, "first divergence at word 1:") {
		t.Fatalf("a wholly different body must report from word 1: %v", g.Errs)
	}
	if !hasErr(g, `found "Nothing to declare here."`) {
		t.Fatalf("the found window must quote the body: %v", g.Errs)
	}
}

// The missing-section finding names the template file too: a reader who never
// had the block cannot be told only that it is missing.
func TestCheckBuildPlanDisclaimerMissingSectionNamesTemplate(t *testing.T) {
	build := strings.Replace(goCrmStylePlan+templateSectionsStub(),
		"### What the gates do not verify\n"+gatesDisclaimerText+"\n", "", 1)
	design := writeBuildPlanFixture(t, "", map[string]string{"BUILD.md": build})
	g := CheckBuildPlan(design)
	if !hasErr(g, "no 'What the gates do not verify' section") {
		t.Fatalf("a missing section must be an ERROR: %v", g.Errs)
	}
	if !hasErr(g, "references/build-md-template.md") {
		t.Fatalf("the missing-section finding must name the template file: %v", g.Errs)
	}
}

func TestFirstWordDivergence(t *testing.T) {
	cases := []struct {
		name             string
		want, got        string
		pos              int
		wantWin, gotWin  string
		expectDivergence bool
	}{
		{"identical", "a b c", "a  b\nc", 0, "", "", false},
		{"mid-run", "a b c d", "a b x d", 3, "c d", "x d", true},
		{"first word", "a b", "z b", 1, "a b", "z b", true},
		{"got runs out", "a b c", "a b", 3, "c", "(nothing; the text ends here)", true},
		{"want runs out", "a b", "a b c", 3, "(nothing; the text ends here)", "c", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pos, wantWin, gotWin, ok := firstWordDivergence(c.want, c.got)
			if ok != c.expectDivergence {
				t.Fatalf("ok = %v, want %v", ok, c.expectDivergence)
			}
			if !ok {
				return
			}
			if pos != c.pos || wantWin != c.wantWin || gotWin != c.gotWin {
				t.Fatalf("got (%d, %q, %q), want (%d, %q, %q)", pos, wantWin, gotWin, c.pos, c.wantWin, c.gotWin)
			}
		})
	}
}

// The window stops at divergenceWindow words and says more follow.
func TestWordWindowTruncates(t *testing.T) {
	words := strings.Fields("one two three four five six seven eight nine ten")
	if got, want := wordWindow(words, 0), "one two three four five six seven eight ..."; got != want {
		t.Fatalf("wordWindow = %q, want %q", got, want)
	}
	if got, want := wordWindow(words, 9), "ten"; got != want {
		t.Fatalf("wordWindow tail = %q, want %q", got, want)
	}
}

// The data-dictionary finding says what it actually keys on (heading titles)
// and offers a remedy that fits a hand-derived slice, not only a byte copy.
func TestCheckBuildPlanDataDictionaryFindingNamesHeadings(t *testing.T) {
	build := goCrmStylePlan + "\n## Data dictionary\n\nrows\n\n## Data dictionary (orders slice)\n\nrows\n"
	design := writeBuildPlanFixture(t, build, nil)
	g := CheckBuildPlan(design)
	msg := ""
	for _, e := range g.Errs {
		if strings.Contains(e, "data dictionary") {
			msg = e
		}
	}
	if msg == "" {
		t.Fatalf("two data-dictionary headings must be an ERROR: %v", g.Errs)
	}
	for _, want := range []string{
		"2 headings contain the phrase 'data dictionary'",
		"BUILD.md:",
		"retitle derived or per-packet slices",
		"schema slice",
		"machinery:embed marker on the line before a byte-identical copy",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("data-dictionary finding omits %q: %s", want, msg)
		}
	}
}

// One heading passes and counts; zero headings stay silent. The wording change
// must not move either boundary.
func TestCheckBuildPlanDataDictionaryCountsUnchanged(t *testing.T) {
	one := writeBuildPlanFixture(t, goCrmStylePlan+"\n## Data dictionary\n\nrows\n", nil)
	g := CheckBuildPlan(one)
	if hasErr(g, "data dictionary") {
		t.Fatalf("one dictionary must pass: %v", g.Errs)
	}
	if g.Counts["data dictionary unique"] != 1 {
		t.Fatalf("one dictionary not counted: %+v", g.Counts)
	}
	none := writeBuildPlanFixture(t, goCrmStylePlan, nil)
	g = CheckBuildPlan(none)
	if hasErr(g, "data dictionary") || g.Counts["data dictionary unique"] != 0 {
		t.Fatalf("no dictionary must be silent: errs=%v counts=%+v", g.Errs, g.Counts)
	}
}
