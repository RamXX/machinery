package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func writeMachine(t *testing.T, dir, name, src string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOracleRefusesLintFailingMachine(t *testing.T) {
	// IR-F14: `machinery oracle` generated from machines that fail lint
	// (silently narrowing array targets to their first element, among others).
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = oracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
	if leftovers, _ := filepath.Glob(filepath.Join(d, "*.oracle.md")); len(leftovers) != 0 {
		t.Fatalf("oracle written for a lint-failing machine: %v", leftovers)
	}
}

func TestOracleTagCollisionIsError(t *testing.T) {
	// IR-F12: Deal and DealAggregate both derive the tag DEAL; identical
	// stable ids across machines in one design must be a hard error.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
}

func TestOracleTagOverrideDisambiguates(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","_oracle_tag":"DEALAGG","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	if err := oracleRun(d, false, ""); err != nil {
		t.Fatalf("oracleRun: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	body, err := os.ReadFile(filepath.Join(d, "DealAggregate.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "DEALAGG-") {
		t.Fatalf("override tag missing from generated oracle:\n%s", body)
	}
}

const validMachineA = `{"id":"alpha","initial":"Lead","states":{
	"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
	"Won":{"type":"final"}}}`

const validMachineB = `{"id":"beta","initial":"Open","states":{
	"Open":{"on":{"close":{"target":"Closed","guard":"canClose"}},"_refusal":{"close":"fixture: the command boundary refuses when canClose is false"}},
	"Closed":{"type":"final"}}}`

func TestOracleDirectoryRegeneratesValidPastBroken(t *testing.T) {
	// S12: one half-written machine must not block the whole directory. The
	// valid machines regenerate, the broken one is named, the run exits 1.
	out, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	writeMachine(t, d, "Broken.machine.json", `{"id":"broken","initial":`)
	_ = oracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	for _, want := range []string{"Alpha.oracle.md", "Beta.oracle.md"} {
		if _, err := os.Stat(filepath.Join(d, want)); err != nil {
			t.Fatalf("%s not generated past the broken sibling (stdout %q, stderr %q)", want, out.String(), errB.String())
		}
	}
	if _, err := os.Stat(filepath.Join(d, "Broken.oracle.md")); err == nil {
		t.Fatal("oracle written for an unparseable machine")
	}
	if !strings.Contains(errB.String(), "Broken") && !strings.Contains(errB.String(), "broken") {
		t.Fatalf("broken machine not named on stderr: %q", errB.String())
	}
	if !strings.Contains(errB.String(), "1 machine(s) failed") {
		t.Fatalf("failure summary missing: %q", errB.String())
	}
}

func TestOracleSingleFileMode(t *testing.T) {
	// S12: a named file regenerates exactly that file, leaving siblings alone.
	out, _, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Beta.machine.json", validMachineB)
	if err := oracleRunFiles([]string{fa}, false, ""); err != nil {
		t.Fatalf("oracleRunFiles: %v (stdout %q)", err, out.String())
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Fatal("named file's oracle not generated")
	}
	if _, err := os.Stat(filepath.Join(d, "Beta.oracle.md")); err == nil {
		t.Fatal("unnamed sibling's oracle generated by a per-file run")
	}
}

func TestOracleSingleFileReservesSiblingTags(t *testing.T) {
	// S12: a per-file run must not silently mint a stable-id tag a directory
	// sibling already owns; both ids share the derived tag here.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	fa := writeMachine(t, d, "Deal.machine.json", `{"id":"deal","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Won","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Won":{"type":"final"}}}`)
	writeMachine(t, d, "DealAggregate.machine.json", `{"id":"dealAggregate","initial":"Lead","states":{
		"Lead":{"on":{"advance":{"target":"Closed","guard":"canAdvance"}},"_refusal":{"advance":"fixture: the command boundary refuses when canAdvance is false"}},
		"Closed":{"type":"final"}}}`)
	_ = oracleRunFiles([]string{fa}, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "_oracle_tag") {
		t.Fatalf("stderr should point at _oracle_tag, got %q", errB.String())
	}
	if _, err := os.Stat(filepath.Join(d, "Deal.oracle.md")); err == nil {
		t.Fatal("oracle generated for a machine whose tag a sibling contests")
	}
}

func TestOracleLintFailureSkipsOnlyThatMachine(t *testing.T) {
	// S12: a lint-failing machine is refused without blocking a valid sibling.
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Alpha.machine.json", validMachineA)
	writeMachine(t, d, "Bad.machine.json", `{"id":"m14","initial":"Idle","states":{
		"Idle":{"on":{"GO":{"target":["Done","Other"]}}},
		"Done":{"type":"final"},
		"Other":{"type":"final"}}}`)
	_ = oracleRun(d, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if _, err := os.Stat(filepath.Join(d, "Alpha.oracle.md")); err != nil {
		t.Fatal("valid sibling not generated past the lint-failing machine")
	}
	if _, err := os.Stat(filepath.Join(d, "Bad.oracle.md")); err == nil {
		t.Fatal("oracle written for a lint-failing machine")
	}
	if !strings.Contains(errB.String(), "fails lint") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestOracleNonMachineFileArgIsError(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	p := writeMachine(t, d, "notes.txt", "hello")
	_ = oracleRunFiles([]string{p}, false, "")
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "not a *.machine.json") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestOracleDiffClassifiesChurn(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"stop":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	_ = oracleRun(d, false, "")
	if len(*codes) != 0 {
		t.Fatalf("generation failed: %v", *codes)
	}
	outB.Reset()
	_ = oracleRun(d, true, "")
	if !strings.Contains(outB.String(), "no churn") {
		t.Fatalf("a fresh oracle must diff clean, got %q", outB.String())
	}
	committed, err := os.ReadFile(filepath.Join(d, "Thing.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	// drop one transition, add another: the diff owes a deleted and a new id
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"pause":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	outB.Reset()
	_ = oracleRun(d, true, "")
	out := outB.String()
	if !strings.Contains(out, "new") || !strings.Contains(out, "deleted") {
		t.Fatalf("churn must classify as new+deleted, got %q", out)
	}
	after, err := os.ReadFile(filepath.Join(d, "Thing.oracle.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(committed) {
		t.Fatal("--diff must not rewrite the committed oracle")
	}
}

func TestOracleDiffRenameShapedChurn(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	base := `{"id":"thing",%s"initial":"A","states":{
		"A":{"on":{"go":{"target":"B"}}},
		"B":{"type":"final"}}}`
	writeMachine(t, d, "Thing.machine.json", strings.Replace(base, "%s", "", 1))
	_ = oracleRun(d, false, "")
	if len(*codes) != 0 {
		t.Fatalf("generation failed: %v", *codes)
	}
	// an _oracle_tag change churns every stable id with identical row content:
	// exactly the rename-shaped class the revision protocol maps, never
	// processes as delete-all-plus-new
	writeMachine(t, d, "Thing.machine.json", strings.Replace(base, "%s", `"_oracle_tag":"WIDG",`, 1))
	outB.Reset()
	_ = oracleRun(d, true, "")
	if !strings.Contains(outB.String(), "rename-shaped") {
		t.Fatalf("identical-content id churn must classify as rename-shaped, got %q", outB.String())
	}
}

func TestTokensEqual(t *testing.T) {
	outB, _, codes := withCapturedIO(t)
	d := t.TempDir()
	a := filepath.Join(d, "a.md")
	b := filepath.Join(d, "b.md")
	c := filepath.Join(d, "c.md")
	if err := os.WriteFile(a, []byte("one two\nthree   four\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("  one\ntwo three\nfour"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c, []byte("one two\nthree five\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := tokensEqualRun(a, b); err != nil {
		t.Fatalf("reflow is formatting-only: %v", err)
	}
	if !strings.Contains(outB.String(), "token-identical: 4 tokens") {
		t.Fatalf("got %q", outB.String())
	}
	outB.Reset()
	_ = tokensEqualRun(a, c)
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("a wording change must exit 1: %v", *codes)
	}
	if !strings.Contains(outB.String(), "NOT token-identical") {
		t.Fatalf("got %q", outB.String())
	}
}

// gitInit makes dir a repository with one commit of everything in it. It
// skips the test when git is unavailable, since --against is a git feature.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "t@example.invalid"},
		{"config", "user.name", "t"},
		{"add", "-A"},
		{"-c", "commit.gpgsign=false", "commit", "-q", "-m", "baseline"},
	} {
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestOracleDiffAgainstRef(t *testing.T) {
	outB, errB, codes := withCapturedIO(t)
	d := t.TempDir()
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"stop":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	if err := oracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	gitInit(t, d)

	// the churn: one transition renamed, and the regeneration ALREADY written,
	// which is exactly the state where plain --diff reports nothing.
	writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
		"A":{"on":{"go":{"target":"B"},"pause":{"target":"C"}}},
		"B":{"type":"final"},
		"C":{"type":"final"}}}`)
	outB.Reset()
	if err := oracleRun(d, false, ""); err != nil {
		t.Fatal(err)
	}
	outB.Reset()
	if err := oracleRun(d, true, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(outB.String(), "no churn") {
		t.Fatalf("after a regeneration, plain --diff sees nothing; got %q", outB.String())
	}
	// --against HEAD recovers the affected-test list
	outB.Reset()
	if err := oracleRun(d, true, "HEAD"); err != nil {
		t.Fatalf("oracleRun --against: %v (stderr %q)", err, errB.String())
	}
	out := outB.String()
	if !strings.Contains(out, "new ") || !strings.Contains(out, "deleted ") {
		t.Fatalf("--against HEAD must classify new+deleted, got %q", out)
	}
	if len(*codes) != 0 {
		t.Fatalf("a clean --against run must not exit non-zero: %v", *codes)
	}
}

func TestOracleDiffAgainstRefFailsLoudly(t *testing.T) {
	cases := []struct {
		name    string
		ref     string
		commit  bool
		wantSub string
	}{
		{name: "unknown ref", ref: "no-such-ref", commit: true, wantSub: "does not resolve"},
		{name: "path absent at the ref", ref: "HEAD", commit: false, wantSub: "at HEAD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, errB, codes := withCapturedIO(t)
			d := t.TempDir()
			writeMachine(t, d, "Thing.machine.json", `{"id":"thing","initial":"A","states":{
				"A":{"on":{"go":{"target":"B"}}},
				"B":{"type":"final"}}}`)
			if tc.commit {
				if err := oracleRun(d, false, ""); err != nil {
					t.Fatal(err)
				}
			}
			gitInit(t, d)
			if !tc.commit {
				// the oracle exists in the working tree but not at the ref
				if err := oracleRun(d, false, ""); err != nil {
					t.Fatal(err)
				}
			}
			if err := oracleRun(d, true, tc.ref); err == nil {
				t.Fatal("a baseline that cannot be read must fail loudly")
			}
			if len(*codes) == 0 || (*codes)[0] != 1 {
				t.Fatalf("exit codes %v, want [1]", *codes)
			}
			if !strings.Contains(errB.String(), tc.wantSub) {
				t.Fatalf("stderr %q does not name %q", errB.String(), tc.wantSub)
			}
		})
	}
}

func TestOracleAgainstWithoutDiffIsRefused(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	c := newOracleCmd()
	c.SilenceUsage, c.SilenceErrors = true, true
	c.SetArgs([]string{t.TempDir(), "--against", "HEAD"})
	if err := c.Execute(); err == nil {
		t.Fatal("--against without --diff must be refused")
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "pass --diff too") {
		t.Fatalf("stderr %q", errB.String())
	}
}

func TestGitHelpersDegradeCleanly(t *testing.T) {
	t.Run("realPath falls back to the path it was given", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "no-such-dir")
		if got := realPath(missing); got != missing {
			t.Fatalf("realPath(%q) = %q, want the input back", missing, got)
		}
	})
	t.Run("gitMessage renders a non-exec error", func(t *testing.T) {
		if got := gitMessage(errors.New("  boom  ")); got != "boom" {
			t.Fatalf("gitMessage = %q, want %q", got, "boom")
		}
	})
	t.Run("gitShowOracle refuses a path outside the repository", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "Thing.oracle.md")
		if _, err := gitShowOracle(root, "HEAD", outside); err == nil {
			t.Fatal("a path outside the repository must fail loudly")
		}
	})
	t.Run("gitRootOf refuses a directory outside a repository", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git not available")
		}
		// a temp dir is not inside the machinery checkout
		if _, err := gitRootOf(t.TempDir()); err == nil {
			t.Skip("the temp directory happens to sit inside a repository")
		}
	})
}
