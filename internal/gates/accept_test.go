package gates

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/testgit"
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
	return writeAcceptFixtureIn(t, t.TempDir(), files)
}

// writeAcceptFixtureIn is writeAcceptFixture with the design directory chosen
// by the caller, so a test can place the design inside a git repository.
func writeAcceptFixtureIn(t *testing.T, design string, files map[string]string) string {
	t.Helper()
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

func TestAcceptanceRejectsNumericFilenameAlias(t *testing.T) {
	design := writeAcceptFixture(t, map[string]string{"acceptance/M00.yaml": acceptEvidenceM0})
	g := CheckAcceptance(design, acceptedCommit)
	if !hasErr(g, "duplicates milestone M0 acceptance evidence") {
		t.Fatalf("M0.yaml and M00.yaml must not be first-wins: %v", g.Errs)
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

// The reverse direction of the coverage rule: a dod_ids entry that resolves
// to no committed oracle id (a typo, or an id a regeneration left behind) is
// an ERROR naming it. Without this the list could be pure fiction and the
// gate would still report coverage.
func TestCheckAcceptanceDoDIDsMustResolve(t *testing.T) {
	phantom := strings.Replace(acceptEvidenceM0, "  - CMD-abc123\n", "  - CMD-abc123\n  - CMD-999999\n", 1)
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": phantom})
	g := CheckAcceptance(design, acceptedCommit)
	if !strings.Contains(strings.Join(g.Errs, "\n"), "dod_ids names 'CMD-999999', which no committed oracle declares") {
		t.Fatalf("an unresolvable dod_ids entry must be named: %v", g.Errs)
	}
	if g.Counts["DoD ids bound"] != 2 {
		t.Errorf("the ids that DO resolve must still bind: %+v", g.Counts)
	}
}

// A typo in an otherwise complete list is the exact failure the reverse rule
// exists for: forward coverage reports the omission, reverse resolution names
// the id that replaced it, and the reviewer sees both halves of the story.
func TestCheckAcceptanceDoDIDTypoFailsBothWays(t *testing.T) {
	typo := strings.Replace(acceptEvidenceM0, "  - CMD-abc123\n", "  - CMD-abc124\n", 1)
	design := writeAcceptFixture(t, map[string]string{"acceptance/M0.yaml": typo})
	g := CheckAcceptance(design, acceptedCommit)
	joined := strings.Join(g.Errs, "\n")
	for _, want := range []string{"dod_ids omits 'CMD-abc123'", "dod_ids names 'CMD-abc124'"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("want an error containing %q, got %v", want, g.Errs)
		}
	}
}

// --- the default commit binding ------------------------------------------

// gitFixture is a repository with a two-commit main line and one commit on a
// branch that was never merged: root is an ancestor of head, side is not.
type gitFixture struct {
	dir  string
	root string
	head string
	side string
}

// initGitRepo turns dir into a git repository carrying exactly one commit and
// returns its HEAD. Every identity and signing input is passed on the command
// line, so the developer's global git configuration cannot change the result.
func initGitRepo(t *testing.T, dir string) string {
	t.Helper()
	return initGitHistory(t, dir).head
}

// initGitHistory builds the gitFixture history in dir.
func initGitHistory(t *testing.T, dir string) gitFixture {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		base := []string{
			"-c", "user.name=machinery test", "-c", "user.email=test@example.invalid",
		}
		out, err := testgit.Run(t.Context(), dir, append(base, args...)...)
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(name, msg string) string {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(msg+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", name)
		git("commit", "-q", "-m", msg)
		return git("rev-parse", "HEAD")
	}
	git("init", "-q")
	f := gitFixture{dir: dir}
	f.root = commit(".gitkeep", "root commit")
	// the initial branch name is whatever this git installation defaults to;
	// read it rather than assume main or master
	main := git("rev-parse", "--abbrev-ref", "HEAD")
	git("branch", "side")
	f.head = commit("second.txt", "second commit on the main line")
	git("checkout", "-q", "side")
	f.side = commit("side.txt", "a commit on a branch that is never merged")
	git("checkout", "-q", main)
	if got := git("rev-parse", "HEAD"); got != f.head {
		t.Fatalf("fixture left HEAD at %s, want %s", got, f.head)
	}
	return f
}

// The three provenances of the commit under review, plus the one case that
// has none. The derivation resolves from the DESIGN PATH: the test process
// runs inside the machinery repository, so a derivation that read the working
// directory would bind the fixture to machinery's own HEAD.
func TestResolveReviewCommit(t *testing.T) {
	repo := t.TempDir()
	head := initGitRepo(t, repo)
	design := filepath.Join(repo, "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()

	t.Run("flag or env wins over git", func(t *testing.T) {
		got, prov, err := resolveReviewCommit(design, "  "+acceptedCommit+"  ")
		if err != nil || got != acceptedCommit || prov != commitFromCaller {
			t.Fatalf("the caller's commit must win: got %q prov=%d err=%v", got, prov, err)
		}
	})
	t.Run("derived from the design's repository", func(t *testing.T) {
		got, prov, err := resolveReviewCommit(design, "")
		if err != nil || got != head || prov != commitFromGit {
			t.Fatalf("want the fixture repo HEAD %s, got %q prov=%d err=%v", head, got, prov, err)
		}
		if cwdHead := gitHeadAt("."); cwdHead != "" && cwdHead == got {
			t.Fatal("the commit was resolved from the process working directory, not the design path")
		}
	})
	t.Run("no repository leaves it unresolved", func(t *testing.T) {
		got, prov, err := resolveReviewCommit(outside, "")
		if err != nil || got != "" || prov != commitAbsent {
			t.Fatalf("outside a repository nothing may be derived: got %q prov=%d err=%v", got, prov, err)
		}
	})
}

func installFakeGit(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake git executable is a POSIX shell fixture")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "git"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestCheckAcceptanceCompleteCommitResolutionFailsClosed(t *testing.T) {
	t.Run("missing git", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		t.Setenv("PATH", t.TempDir())
		g := checkAcceptance(design, "", true)
		if !hasErr(g, "final handoff requires a commit under review") || !hasErr(g, "executable file not found") {
			t.Fatalf("missing git must block final handoff with its cause: %v", g.Errs)
		}
		if len(g.Notes) != 0 {
			t.Fatalf("complete mode may not downgrade commit failure to a note: %v", g.Notes)
		}
	})

	t.Run("not a repository", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		g := checkAcceptance(design, "", true)
		if !hasErr(g, "final handoff requires a commit under review") || !hasErr(g, "not a git repository") {
			t.Fatalf("a non-repository design must block final handoff: %v", g.Errs)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "exec sleep 10")
		old := gitCommandTimeout
		gitCommandTimeout = 25 * time.Millisecond
		t.Cleanup(func() { gitCommandTimeout = old })
		g := checkAcceptance(design, "", true)
		if !hasErr(g, "timed out") || !hasErr(g, "context deadline exceeded") {
			t.Fatalf("a hung git must block final handoff precisely: %v", g.Errs)
		}
	})

	t.Run("malformed HEAD", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "printf '%s\\n' not-a-commit")
		g := checkAcceptance(design, "", true)
		if !hasErr(g, "returned malformed commit") {
			t.Fatalf("malformed git output must block final handoff: %v", g.Errs)
		}
	})
}

func TestCheckAcceptanceStagedCommitResolutionFailsClosedOperationally(t *testing.T) {
	t.Run("missing git", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		t.Setenv("PATH", t.TempDir())
		g := checkAcceptance(design, "", false)
		if !hasErr(g, "commit binding could not resolve") || !hasErr(g, "executable file not found") {
			t.Fatalf("missing git was downgraded in staged mode: %v", g.Errs)
		}
		if len(g.Notes) != 0 {
			t.Fatalf("operational failure must not claim semantic non-repository absence: %v", g.Notes)
		}
	})

	t.Run("permission failure", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "echo 'fatal: permission denied reading repository' >&2; exit 126")
		g := checkAcceptance(design, "", false)
		if !hasErr(g, "permission denied") {
			t.Fatalf("Git permission failure was downgraded in staged mode: %v", g.Errs)
		}
	})

	t.Run("corrupt repository", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "echo 'fatal: bad object HEAD' >&2; exit 128")
		g := checkAcceptance(design, "", false)
		if !hasErr(g, "bad object HEAD") {
			t.Fatalf("corrupt repository was downgraded in staged mode: %v", g.Errs)
		}
	})

	t.Run("malformed HEAD", func(t *testing.T) {
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "printf '%s\\n' not-a-commit")
		g := checkAcceptance(design, "", false)
		if !hasErr(g, "returned malformed commit") {
			t.Fatalf("malformed Git output was downgraded in staged mode: %v", g.Errs)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		if _, err := os.Stat("/bin/sleep"); err != nil {
			t.Skip("POSIX sleep fixture unavailable")
		}
		design := writeAcceptFixture(t, nil)
		installFakeGit(t, "exec /bin/sleep 10")
		old := gitCommandTimeout
		gitCommandTimeout = 25 * time.Millisecond
		t.Cleanup(func() { gitCommandTimeout = old })
		g := checkAcceptance(design, "", false)
		if !hasErr(g, "timed out") {
			t.Fatalf("Git timeout was downgraded in staged mode: %v", g.Errs)
		}
	})
}

func TestRunGitExactBoundsOutputAndProcessTree(t *testing.T) {
	t.Run("output", func(t *testing.T) {
		installFakeGit(t, `i=0
while [ "$i" -lt 10000 ]; do
  printf '1234567890'
  i=$((i + 1))
done
exit 2`)
		_, err := runGitExact(t.TempDir(), "rev-parse", "HEAD")
		if err == nil || !strings.Contains(err.Error(), "output truncated at 65536 bytes") {
			t.Fatalf("noisy git must fail with bounded, explicit truncation: %v", err)
		}
		if len(err.Error()) > gitOutputLimit+1000 {
			t.Fatalf("git diagnostic exceeded its capture bound: %d bytes", len(err.Error()))
		}
	})

	t.Run("successful output overflow", func(t *testing.T) {
		installFakeGit(t, `i=0
while [ "$i" -lt 10000 ]; do
  printf '1234567890'
  i=$((i + 1))
done
exit 0`)
		_, err := runGitExact(t.TempDir(), "rev-parse", "HEAD")
		if err == nil || !strings.Contains(err.Error(), "exceeded the 65536-byte success-output limit on stdout") {
			t.Fatalf("successful output overflow must fail closed: %v", err)
		}
		if len(err.Error()) > 1000 {
			t.Fatalf("success-overflow diagnostic was not constant-size: %d bytes", len(err.Error()))
		}
	})

	t.Run("descendant holding pipes", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("the fake git executable is a POSIX shell fixture")
		}
		sentinel := filepath.Join(t.TempDir(), "survived")
		t.Setenv("MACHINERY_GIT_DESCENDANT_SENTINEL", sentinel)
		installFakeGit(t, `(
  /bin/sleep 1
  printf survived > "$MACHINERY_GIT_DESCENDANT_SENTINEL"
) &
		/bin/sleep 10`)
		old := gitCommandTimeout
		gitCommandTimeout = 50 * time.Millisecond
		t.Cleanup(func() { gitCommandTimeout = old })
		started := time.Now()
		if _, err := runGitExact(t.TempDir(), "rev-parse", "HEAD"); err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("timed-out git process tree was not reported: %v", err)
		}
		if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
			t.Fatalf("descendant held the git query open for %s", elapsed)
		}
		time.Sleep(1100 * time.Millisecond)
		if _, err := os.Lstat(sentinel); !os.IsNotExist(err) {
			t.Fatalf("git descendant survived process-tree cleanup: %v", err)
		}
	})
}

func TestAcceptanceAncestryQueriesDistinguishSemanticAndOperationalFailure(t *testing.T) {
	t.Run("missing commit is semantic", func(t *testing.T) {
		installFakeGit(t, "exit 1")
		full, err := gitCommitOf(t.TempDir(), "deadbeef")
		if err != nil || full != "" {
			t.Fatalf("exit 1 from quiet rev-parse means no such commit: full=%q err=%v", full, err)
		}
	})

	t.Run("commit query failure is operational", func(t *testing.T) {
		installFakeGit(t, "echo 'fatal: corrupt object database' >&2; exit 128")
		_, err := gitCommitOf(t.TempDir(), "deadbeef")
		if err == nil || !strings.Contains(err.Error(), "corrupt object database") {
			t.Fatalf("corrupt rev-parse must retain its operational error: %v", err)
		}
	})

	t.Run("malformed commit output is operational", func(t *testing.T) {
		installFakeGit(t, "printf '%s\\n' malformed")
		_, err := gitCommitOf(t.TempDir(), "deadbeef")
		if err == nil || !strings.Contains(err.Error(), "malformed commit") {
			t.Fatalf("malformed rev-parse output must not mean absent: %v", err)
		}
	})

	t.Run("not ancestor is semantic", func(t *testing.T) {
		installFakeGit(t, "exit 1")
		ok, err := gitIsAncestor(t.TempDir(), "deadbeef", "cafebabe")
		if err != nil || ok {
			t.Fatalf("merge-base exit 1 means not ancestor: ok=%v err=%v", ok, err)
		}
	})

	t.Run("ancestry query failure is operational", func(t *testing.T) {
		installFakeGit(t, "echo 'fatal: cannot read commit graph' >&2; exit 128")
		_, err := gitIsAncestor(t.TempDir(), "deadbeef", "cafebabe")
		if err == nil || !strings.Contains(err.Error(), "cannot read commit graph") {
			t.Fatalf("merge-base operational failure was collapsed to not-ancestor: %v", err)
		}
	})

	t.Run("ancestry success stdout is operational", func(t *testing.T) {
		installFakeGit(t, "printf 'unexpected success output\\n'; exit 0")
		_, err := gitIsAncestor(t.TempDir(), "deadbeef", "cafebabe")
		if err == nil || !strings.Contains(err.Error(), "emitted stdout on success") || !strings.Contains(err.Error(), "unexpected success output") {
			t.Fatalf("merge-base success stdout must fail closed: %v", err)
		}
	})
}

func TestAcceptanceGitIgnoresAmbientRepositoryRedirectionAndSuccessWarnings(t *testing.T) {
	repoA := t.TempDir()
	headA := initGitRepo(t, repoA)
	repoB := t.TempDir()
	_ = initGitRepo(t, repoB)
	t.Setenv("GIT_DIR", filepath.Join(repoB, ".git"))
	t.Setenv("GIT_WORK_TREE", repoB)
	t.Setenv("GIT_TRACE", "1")
	designA := filepath.Join(repoA, "design")
	if err := os.MkdirAll(designA, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := gitHeadAtExact(designA)
	if err != nil || got != headA {
		t.Fatalf("ambient Git redirection changed acceptance repository: got=%q want=%q err=%v", got, headA, err)
	}

	installFakeGit(t, "printf '%s\\n' "+acceptedCommit+"; echo 'warning: injected config' >&2; exit 0")
	if _, err := gitHeadAtExact(t.TempDir()); err == nil || !strings.Contains(err.Error(), "emitted stderr on success") || !strings.Contains(err.Error(), "injected config") {
		t.Fatalf("successful Git warning was discarded: %v", err)
	}
}

func TestCheckAcceptanceStagedUnbornRepositoryIsSemanticAbsence(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	if out, err := testgit.Run(t.Context(), repo, "init", "-q"); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	design := writeAcceptFixtureIn(t, filepath.Join(repo, "design"), nil)
	g := checkAcceptance(design, "", false)
	if len(g.Errs) != 0 || !strings.Contains(strings.Join(g.Notes, "\n"), "commit binding not checked") {
		t.Fatalf("an unborn but readable repository should remain an explicit staged note: errs=%v notes=%v", g.Errs, g.Notes)
	}
}

func TestCheckAcceptanceCompleteResolvesValidImplicitCommit(t *testing.T) {
	repo := t.TempDir()
	head := initGitRepo(t, repo)
	design := writeAncestryFixture(t, repo, head)
	g := checkAcceptance(design, "", true)
	if len(g.Errs) != 0 {
		t.Fatalf("a readable repository HEAD must bind final handoff: %v", g.Errs)
	}
	if g.Counts["commit bindings verified"] != 1 {
		t.Fatalf("implicit final-handoff commit was not verified: %+v", g.Counts)
	}
	if len(g.Notes) != 0 {
		t.Fatalf("a verified final-handoff binding needs no note: %v", g.Notes)
	}
}

func TestRunSelectedCompleteUsesFailClosedCommitResolution(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	selection := Selection{Run: map[string]bool{"ga": true}, Explicit: true}
	var acceptance *Gate
	for _, gate := range RunSelected(design, "", selection, RunOptions{Complete: true}) {
		if strings.HasPrefix(gate.Title, "Ga-accept") {
			acceptance = gate
			break
		}
	}
	if acceptance == nil || !hasErr(acceptance, "final handoff requires a commit under review") {
		t.Fatalf("complete suite did not route Ga through strict commit resolution: %+v", acceptance)
	}
}

// --- derived mode: ancestry ----------------------------------------------
//
// Dispatcher QC adjudication, 2026-08-30. The derived lane binds by ANCESTRY,
// not identity. Identity is right when a caller names the commit under review;
// it is wrong when the gate went looking for one, because the commit that adds
// the evidence file already differs from the commit the evidence names, so an
// identity rule would go red one commit later and stay red. Ancestry still
// catches what the note tier let through: a sha that resolves to nothing, and
// a sha from a history this tree never took.

// writeAncestryFixture lays the standard design inside repo's design/ with its
// evidence naming evidenceCommit.
func writeAncestryFixture(t *testing.T, repo, evidenceCommit string) string {
	t.Helper()
	return writeAcceptFixtureIn(t, filepath.Join(repo, "design"), map[string]string{
		// The sha is quoted: a live abbreviation can be purely numeric (about
		// (10/16)^10 of repos), and unquoted YAML would read it as a number,
		// which the schema check rejects by design. Quoting is exactly what
		// the gate's own finding tells a user to do, and it makes this test
		// deterministic instead of flaky at that probability.
		"acceptance/M0.yaml": strings.Replace(acceptEvidenceM0, "commit: "+acceptedCommit, "commit: \""+evidenceCommit+"\"", 1),
	})
}

// The failure mode the ancestry fixture quotes its way around, pinned
// deterministically: an UNQUOTED purely numeric revision is read by YAML as a
// number, and the schema check must answer with the quoting guidance instead
// of a bare type error.
func TestCheckAcceptanceNumericRevisionGuidance(t *testing.T) {
	repo := t.TempDir()
	initGitHistory(t, repo)
	design := writeAcceptFixtureIn(t, filepath.Join(repo, "design"), map[string]string{
		"acceptance/M0.yaml": strings.Replace(acceptEvidenceM0, "commit: "+acceptedCommit, "commit: 1234567890", 1),
	})
	g := CheckAcceptance(design, "")
	joined := strings.Join(g.Errs, "\n")
	if !strings.Contains(joined, "quote a purely numeric revision") {
		t.Fatalf("an unquoted numeric revision must earn the quoting guidance, got %v", g.Errs)
	}
	if g.Counts["commit bindings verified"] != 0 {
		t.Errorf("a rejected revision may not be counted as verified: %+v", g.Counts)
	}
}

func TestCheckAcceptanceDerivedModeAncestry(t *testing.T) {
	cases := []struct {
		name     string
		evidence func(gitFixture) string
		wantErr  string // "" means the binding must hold
	}{
		{
			name:     "equal to HEAD",
			evidence: func(f gitFixture) string { return f.head },
		},
		{
			name:     "an ancestor of HEAD",
			evidence: func(f gitFixture) string { return f.root },
		},
		{
			name: "an abbreviated ancestor",
			// (a) is resolution, not string matching: an abbreviation that
			// names one commit in this repository is that commit
			evidence: func(f gitFixture) string { return f.root[:10] },
		},
		{
			name:     "a sha no object answers to",
			evidence: func(gitFixture) string { return acceptedCommit },
			wantErr:  "names no commit in the repository holding the design",
		},
		{
			name:     "a commit on an unmerged branch",
			evidence: func(f gitFixture) string { return f.side },
			wantErr:  "is not an ancestor of the commit under review",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			f := initGitHistory(t, repo)
			g := CheckAcceptance(writeAncestryFixture(t, repo, tc.evidence(f)), "")
			joined := strings.Join(g.Errs, "\n")
			if tc.wantErr != "" {
				if !strings.Contains(joined, tc.wantErr) {
					t.Fatalf("want an error containing %q, got %v", tc.wantErr, g.Errs)
				}
				if g.Counts["commit bindings verified"] != 0 {
					t.Errorf("a failed binding may not be counted as verified: %+v", g.Counts)
				}
				return
			}
			if len(g.Errs) != 0 {
				t.Fatalf("the reviewed commit is in this history: %v", g.Errs)
			}
			if g.Counts["commit bindings verified"] != 1 {
				t.Errorf("the derived commit must be bound, not merely resolved: %+v", g.Counts)
			}
			if len(g.Notes) != 0 {
				t.Errorf("a derived binding is a checked binding, not an unchecked one: %v", g.Notes)
			}
			if want := "derived from git HEAD of the repository holding the design; evidence commit bound by ancestry"; !strings.Contains(checkedLine(g), want) {
				t.Errorf("the provenance and its rule must be visible: %q", checkedLine(g))
			}
		})
	}
}

// The two modes are genuinely different rules on the same tree: the commit
// that merely FOLLOWS the evidence passes derived (it is a descendant) and
// fails explicit (it is not that commit). This is the adjudication, pinned.
func TestCheckAcceptanceModesDifferOnADescendantHead(t *testing.T) {
	repo := t.TempDir()
	f := initGitHistory(t, repo)
	design := writeAncestryFixture(t, repo, f.root)

	derived := CheckAcceptance(design, "")
	if len(derived.Errs) != 0 {
		t.Fatalf("derived mode must accept an ancestor: %v", derived.Errs)
	}

	explicit := CheckAcceptance(design, f.head)
	if !strings.Contains(strings.Join(explicit.Errs, "\n"), "does not name the commit under review") {
		t.Fatalf("explicit mode must still demand identity: %v", explicit.Errs)
	}
	if want := "supplied by --commit or MACHINERY_COMMIT; evidence commit bound by identity"; !strings.Contains(checkedLine(explicit), want) {
		t.Errorf("the explicit rule must be named on the checked line: %q", checkedLine(explicit))
	}
}

// An explicit commit still wins inside a repository, whatever HEAD says, and
// is held to identity against the value supplied rather than to the history.
func TestCheckAcceptanceExplicitCommitWinsInsideRepo(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	design := writeAcceptFixtureIn(t, filepath.Join(repo, "design"), nil)
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("the supplied commit must win over the repository HEAD: %v", g.Errs)
	}
	if want := "commit under review supplied by --commit or MACHINERY_COMMIT"; !strings.Contains(checkedLine(g), want) {
		t.Errorf("the provenance must be visible: %q", checkedLine(g))
	}
	if g.Counts["commit bindings verified"] != 1 {
		t.Errorf("identity against the supplied commit must bind: %+v", g.Counts)
	}
}

// An evidence value that could be read as a git option is data, and must
// never reach git as a flag.
func TestCheckAcceptanceDerivedModeRefusesOptionShapedCommits(t *testing.T) {
	repo := t.TempDir()
	initGitHistory(t, repo)
	g := CheckAcceptance(writeAncestryFixture(t, repo, "--output=/tmp/machinery-gate-escape"), "")
	if !strings.Contains(strings.Join(g.Errs, "\n"), "names no commit in the repository holding the design") {
		t.Fatalf("an option-shaped commit must be refused as data: %v", g.Errs)
	}
	if _, err := os.Stat("/tmp/machinery-gate-escape"); err == nil {
		t.Fatal("the value reached git as an option")
	}
}

// Outside a repository the binding still degrades to the stated non-check:
// the default adds a lane, it does not remove the honest note.
func TestCheckAcceptanceOutsideRepoKeepsTheNote(t *testing.T) {
	design := writeAcceptFixture(t, nil)
	if head := gitHeadAt(design); head != "" {
		t.Fatalf("the fixture must sit outside a git repository, got HEAD %s", head)
	}
	g := CheckAcceptance(design, "")
	if !strings.Contains(strings.Join(g.Notes, "\n"), "not inside a git repository") {
		t.Fatalf("the unchecked binding must state itself: %v", g.Notes)
	}
	if strings.Contains(checkedLine(g), "commit under review") {
		t.Errorf("nothing was resolved, so nothing may claim provenance: %q", checkedLine(g))
	}
}

// A design with nothing closed resolves no commit at all: the gate must not
// shell out, and must not claim a provenance for a binding it never made.
func TestCheckAcceptanceNoClosedMilestoneResolvesNoCommit(t *testing.T) {
	repo := t.TempDir()
	initGitRepo(t, repo)
	plan := strings.Replace(acceptPlan, "Status: closed\n", "", 1)
	design := writeAcceptFixtureIn(t, filepath.Join(repo, "design"), map[string]string{"BUILD.md": plan})
	g := CheckAcceptance(design, "")
	if len(g.Errs) != 0 {
		t.Fatalf("evidence for an open milestone is not a finding: %v", g.Errs)
	}
	if strings.Contains(checkedLine(g), "commit under review") || len(g.Notes) != 0 {
		t.Errorf("nothing is bound, so nothing is said: checked=%q notes=%v", checkedLine(g), g.Notes)
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
	if !strings.Contains(sel.Note, "gb,ga,gv,g5") {
		t.Errorf("the note must list ga after gb and owed Gv after ga: %q", sel.Note)
	}
}

// Ga reads the plan through the same heading matcher: a decorated section
// heading must not leave a closed milestone unbindable.
func TestCheckAcceptanceDecoratedPlanHeading(t *testing.T) {
	plan := strings.Replace(acceptPlan, "## 9. Build plan",
		"## 9. Build plan (sealed trust layers; user directive 2026-08-04)", 1)
	design := writeAcceptFixture(t, map[string]string{"BUILD.md": plan})
	if !AcceptanceActive(design) {
		t.Fatal("a closed milestone under a decorated heading must still activate Ga")
	}
	g := CheckAcceptance(design, acceptedCommit)
	if len(g.Errs) != 0 {
		t.Fatalf("Ga not clean under a decorated plan heading: %v", g.Errs)
	}
	if g.Counts["declared milestones"] != 2 || g.Counts["closed milestones with accepted evidence"] != 1 {
		t.Errorf("the decorated plan's milestones must bind: %+v", g.Counts)
	}
}
