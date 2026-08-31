package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/oracle"
	machversion "github.com/RamXX/machinery/internal/version"
)

func withCapturedIO(t *testing.T) (*bytes.Buffer, *bytes.Buffer, *[]int) {
	t.Helper()
	var out, errB bytes.Buffer
	var codes []int
	stdoutW, stderrW = &out, &errB
	exitFunc = func(c int) { codes = append(codes, c) }
	t.Cleanup(func() {
		stdoutW, stderrW = os.Stdout, os.Stderr
		exitFunc = os.Exit
	})
	return &out, &errB, &codes
}

func TestCheckGateG4RequiresImplCaseInsensitive(t *testing.T) {
	// Regression: `--gate G4` (uppercase) used to skip the requires-impl error
	// AND every gate, exiting 0 having verified nothing.
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "G4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "--gate g4 requires --impl") {
		t.Fatalf("stderr %q", errB.String())
	}
}

// A design path that exists but is a FILE must say so, not claim it does not
// exist (GATE-11 cosmetics).
func TestCheckDesignPathIsAFile(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	file := filepath.Join(t.TempDir(), "design.md")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := newCheckCmd()
	cmd.SetArgs([]string{file})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "is not a directory") {
		t.Fatalf("stderr %q, want 'is not a directory'", errB.String())
	}
}

// `--gate "g2,"` once yielded `unknown gate(s): ` with an empty name.
func TestCheckEmptyGateTokenErrorsClearly(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "g2,"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "empty gate name") {
		t.Fatalf("stderr %q, want an empty-gate-name error", errB.String())
	}
}

// doctor reports the hook wiring wherever a plugin layout exists: manifest
// present, shim present and executable (GATE-11 doctor check).
func TestReportHookWiring(t *testing.T) {
	plugin := t.TempDir()
	writeWiring := func(rel, content string, mode os.FileMode) {
		p := filepath.Join(plugin, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
	}
	writeWiring(".claude-plugin/plugin.json", `{"name":"machinery"}`, 0o644)
	writeWiring("hooks/hooks.json", "{}", 0o644)
	writeWiring("hooks/machinery-hook.sh", "#!/bin/sh\nexit 0\n", 0o644) // NOT executable
	t.Setenv("CLAUDE_PLUGIN_ROOT", plugin)

	var out bytes.Buffer
	reportHookWiring(&out)
	got := out.String()
	if !strings.Contains(got, "ok       hook manifest at "+filepath.Join(plugin, "hooks", "hooks.json")) {
		t.Errorf("manifest not reported ok:\n%s", got)
	}
	if !strings.Contains(got, "WARN     hook shim at ") || !strings.Contains(got, "not executable") {
		t.Errorf("non-executable shim must WARN:\n%s", got)
	}

	if err := os.Chmod(filepath.Join(plugin, "hooks", "machinery-hook.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(plugin, "hooks", "hooks.json")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	reportHookWiring(&out)
	got = out.String()
	if !strings.Contains(got, "MISSING  hook manifest") {
		t.Errorf("missing hooks.json must be MISSING:\n%s", got)
	}
	if !strings.Contains(got, "ok       hook shim at ") {
		t.Errorf("executable shim must be ok:\n%s", got)
	}
}

func TestCheckUnknownGateStillErrors(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "g9"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 {
		t.Fatalf("exit codes %v, want [1]", *codes)
	}
	if !strings.Contains(errB.String(), "unknown gate") {
		t.Fatalf("stderr %q", errB.String())
	}
}

// --- P-F10: the version-skew INFO line ---

const skewNoteMachine = `{"id":"widget","initial":"Draft",
  "_delays":{"persistTimeout":"3000 ms - test bound"},
  "states":{
  "Draft":{"on":{"publish":[{"target":"persisting","guard":"guardCanPublish","actions":"setPending"},{"actions":"recordDenied"}]}},
  "Published":{"type":"final"},
  "persisting":{"invoke":{"src":"saveWidget","onDone":{"target":"Published","actions":"commit"},"onError":{"target":"Draft","actions":"recordError"}},"after":{"persistTimeout":{"target":"Draft","actions":"recordTimeout"}}}}}`

// writeSkewDesign builds a one-machine design whose committed oracle carries
// the given transform of a fresh generation.
func writeSkewDesign(t *testing.T, mutate func(string) string) string {
	t.Helper()
	design := t.TempDir()
	mdir := filepath.Join(design, "machines")
	if err := os.MkdirAll(mdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mp := filepath.Join(mdir, "Widget.machine.json")
	if err := os.WriteFile(mp, []byte(skewNoteMachine), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := ir.LoadMachineJSON(mp)
	if err != nil {
		t.Fatal(err)
	}
	text := oracle.Render(m, mp)
	if mutate != nil {
		text = mutate(text)
	}
	if err := os.WriteFile(filepath.Join(mdir, "Widget.oracle.md"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return design
}

func runCheckG3(t *testing.T, design string) string {
	t.Helper()
	out, _, _ := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{design, "--gate", "g3"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// A committed artifact stamped by another machinery version prints exactly one
// non-blocking note; the run stays green.
func TestCheckPrintsVersionSkewNote(t *testing.T) {
	design := writeSkewDesign(t, func(text string) string {
		return strings.Replace(text, machversion.MarkdownStamp(), "<!-- machinery-version: v0.0.1 -->", 1)
	})
	got := runCheckG3(t, design)
	want := "note: artifacts generated by machinery v0.0.1, running " + machversion.Version + "; regenerate on upgrade"
	if !strings.Contains(got, want) {
		t.Fatalf("skew note missing:\n%s\nwant %q", got, want)
	}
	if strings.Count(got, "note: artifacts generated by machinery") != 1 {
		t.Fatalf("more than one skew note:\n%s", got)
	}
	// the note carries the regeneration command for the family this design
	// holds, plus the stamp-only-commit reminder: neither is left to guess
	if !strings.Contains(got, "machinery oracle "+design+"/machines") {
		t.Fatalf("skew note omits the regeneration command:\n%s", got)
	}
	if !strings.Contains(got, "the regeneration lands as its own dedicated stamp-only commit, never mixed with a design change") {
		t.Fatalf("skew note omits the stamp-only-commit reminder:\n%s", got)
	}
	if !strings.Contains(got, "0 blocking (ERROR/DRIFT) finding(s)") {
		t.Fatalf("version-only skew must stay non-blocking:\n%s", got)
	}
}

// Same version: no note. Missing stamp (pre-stamp artifact): no note either.
func TestCheckOmitsVersionSkewNote(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"current stamp": nil,
		"missing stamp": func(text string) string {
			return strings.Replace(text, machversion.MarkdownStamp()+"\n", "", 1)
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := runCheckG3(t, writeSkewDesign(t, mutate))
			if strings.Contains(got, "note: artifacts generated by machinery") {
				t.Fatalf("unexpected skew note:\n%s", got)
			}
			if !strings.Contains(got, "0 blocking (ERROR/DRIFT) finding(s)") {
				t.Fatalf("fixture not green:\n%s", got)
			}
		})
	}
}

// `machinery baseline` stamps the full date so the age note reads 0 days on
// the day the snapshot is taken (a YYYY-MM stamp aged from the first of the
// month).
func TestBaselineDefaultStampIsFullDate(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	root := t.TempDir()
	design := filepath.Join(root, "design")
	impl := filepath.Join(root, "impl")
	writeText(t, filepath.Join(design, "ARCHITECTURE.md"), "# A\n\n## Architecture Contract\n\n```yaml\ncontract_version: 2\nboundaries:\n"+
		"  - id: alpha\n    code: [\"alpha/**\"]\n  - id: beta\n    code: [\"beta/**\"]\ndependency_rules:\n  allow: []\n  deny: []\n```\n")
	writeText(t, filepath.Join(impl, "go.mod"), "module example.com/m\n")
	writeText(t, filepath.Join(impl, "alpha", "a.go"), "package alpha\n\nimport \"example.com/m/beta\"\n")
	writeText(t, filepath.Join(impl, "beta", "b.go"), "package beta\n")
	cmd := newBaselineCmd()
	cmd.SetArgs([]string{design, "--impl", impl})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(out.String(), "# "+today+" seen in") {
		t.Fatalf("rule comment must carry today's full date %s, got:\n%s", today, out.String())
	}
	data, err := os.ReadFile(filepath.Join(design, "ratchet.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"date": "`+today+`"`) && !strings.Contains(string(data), `"date":"`+today+`"`) {
		t.Fatalf("ratchet.json must stamp %s, got %s", today, data)
	}
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// --- Ga-accept: the commit under review ---

// writeAcceptanceDesign builds the smallest design that closes a milestone
// with committed acceptance evidence naming commit.
func writeAcceptanceDesign(t *testing.T, commit string) string {
	t.Helper()
	design := t.TempDir()
	files := map[string]string{
		"BUILD.md": "# B\n\nMode: full\n\n## Build plan\n\n" +
			"**M0 - Walking skeleton.** DoD: green end to end.\nStatus: closed\n",
		"acceptance/M0.yaml": "milestone: 0\ncommit: " + commit + "\nverdict: ACCEPTED\n" +
			"dod_ids: []\nattestations:\n  - the suite ran against real dependencies\n" +
			"findings: []\nreviewer: conductor\ndate: 2026-08-27\n",
	}
	for name, content := range files {
		p := filepath.Join(design, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return design
}

func runCheckGa(t *testing.T, design string, args ...string) (string, int) {
	t.Helper()
	out, _, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs(append([]string{design, "--gate", "ga"}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	code := 0
	if len(*codes) > 0 {
		code = (*codes)[0]
	}
	return out.String(), code
}

// The reviewed commit reaches Ga from --commit and from MACHINERY_COMMIT,
// and the flag wins over the environment.
func TestCheckCommitFlagAndEnvironmentReachGa(t *testing.T) {
	const reviewed = "9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345"
	const other = "dead0000beef1111222233334444555566667777"
	design := writeAcceptanceDesign(t, reviewed)

	if out, code := runCheckGa(t, design, "--commit", reviewed); code != 0 || !strings.Contains(out, "1 commit bindings verified") {
		t.Fatalf("--commit must bind: code=%d out=%s", code, out)
	}

	t.Setenv("MACHINERY_COMMIT", reviewed)
	if out, code := runCheckGa(t, design); code != 0 || !strings.Contains(out, "1 commit bindings verified") {
		t.Fatalf("MACHINERY_COMMIT must bind: code=%d out=%s", code, out)
	}

	t.Setenv("MACHINERY_COMMIT", other)
	out, code := runCheckGa(t, design, "--commit", reviewed)
	if code != 0 || !strings.Contains(out, "1 commit bindings verified") {
		t.Fatalf("the flag must win over the environment: code=%d out=%s", code, out)
	}

	t.Setenv("MACHINERY_COMMIT", "")
	out, code = runCheckGa(t, design, "--commit", other)
	if code != 1 || !strings.Contains(out, "does not name the commit under review") {
		t.Fatalf("a wrong commit must block: code=%d out=%s", code, out)
	}

	out, code = runCheckGa(t, design)
	if code != 0 || !strings.Contains(out, "commit binding not checked") {
		t.Fatalf("no commit must state the non-check: code=%d out=%s", code, out)
	}
}

// With neither flag nor environment, a design that sits inside a git
// repository binds to that repository's history instead of printing the
// non-check note: a local run then proves something. Derived mode is ancestry
// (dispatcher QC adjudication, 2026-08-30), so the ancestor case is the one
// that matters at the CLI: it is the shape every real design has, the evidence
// having been committed after the commit it names. The environment is cleared
// explicitly, because the flag-and-env test above sets it.
func TestCheckDefaultsCommitToGitHistoryOfTheDesignRepo(t *testing.T) {
	t.Setenv("MACHINERY_COMMIT", "")
	repo := t.TempDir()
	root, head, side := initTestGitRepo(t, repo)
	design := filepath.Join(repo, "design")
	writeText(t, filepath.Join(design, "BUILD.md"),
		"# B\n\nMode: full\n\n## Build plan\n\n**M0 - Walking skeleton.** DoD: green end to end.\nStatus: closed\n")
	evidence := func(commit string) {
		t.Helper()
		writeText(t, filepath.Join(design, "acceptance", "M0.yaml"),
			"milestone: 0\ncommit: "+commit+"\nverdict: ACCEPTED\ndod_ids: []\n"+
				"attestations:\n  - the suite ran against real dependencies\n"+
				"findings: []\nreviewer: conductor\ndate: 2026-08-27\n")
	}

	for _, commit := range []struct{ name, sha string }{{"an ancestor of HEAD", root}, {"HEAD itself", head}} {
		evidence(commit.sha)
		out, code := runCheckGa(t, design)
		if code != 0 || !strings.Contains(out, "1 commit bindings verified") {
			t.Fatalf("%s must bind: code=%d out=%s", commit.name, code, out)
		}
		if !strings.Contains(out, "derived from git HEAD of the repository holding the design; evidence commit bound by ancestry") {
			t.Errorf("%s: the provenance and its rule must be visible: %s", commit.name, out)
		}
		if strings.Contains(out, "commit binding not checked") {
			t.Errorf("%s: a derived binding is a checked binding: %s", commit.name, out)
		}
	}

	evidence(side)
	if out, code := runCheckGa(t, design); code != 1 || !strings.Contains(out, "is not an ancestor of the commit under review") {
		t.Fatalf("a commit from an unmerged branch must block: code=%d out=%s", code, out)
	}
	evidence("9f3c1a2b7d4e5f60718293a4b5c6d7e8f9012345")
	if out, code := runCheckGa(t, design); code != 1 || !strings.Contains(out, "names no commit in the repository holding the design") {
		t.Fatalf("a sha this repository does not hold must block: code=%d out=%s", code, out)
	}
}

// initTestGitRepo makes dir a git repository with a two-commit main line and
// one commit on a branch that is never merged, returning root (an ancestor of
// head), head, and side (not an ancestor of head). Identity and signing are
// passed on the command line so the developer's global git configuration
// cannot change the result.
func initTestGitRepo(t *testing.T, dir string) (root, head, side string) {
	t.Helper()
	git := func(args ...string) string {
		t.Helper()
		base := []string{"-C", dir,
			"-c", "user.name=machinery test", "-c", "user.email=test@example.invalid",
			"-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null"}
		out, err := exec.CommandContext(t.Context(), "git", append(base, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(name, msg string) string {
		t.Helper()
		writeText(t, filepath.Join(dir, name), msg+"\n")
		git("add", name)
		git("commit", "-q", "-m", msg)
		return git("rev-parse", "HEAD")
	}
	git("init", "-q")
	root = commit(".gitkeep", "root commit")
	main := git("rev-parse", "--abbrev-ref", "HEAD")
	git("branch", "side")
	head = commit("second.txt", "second commit on the main line")
	git("checkout", "-q", "side")
	side = commit("side.txt", "a commit on a branch that is never merged")
	git("checkout", "-q", main)
	return root, head, side
}

// The green-summary line: a default full run earns platform-green with an
// impl, design-green without one; an explicit --gate subset claims neither.
func TestCheckGreenSummaryLines(t *testing.T) {
	out, _, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--impl", "../../examples/go-crm/impl"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("green run must exit 0: %v\n%s", *codes, out.String())
	}
	if !strings.Contains(out.String(), "platform-green: design gates, G4-import, and Gt-tests all green") {
		t.Fatalf("platform-green line missing:\n%s", out.String())
	}

	out2, _, codes2 := withCapturedIO(t)
	cmd = newCheckCmd()
	cmd.SetArgs([]string{"../../examples/fulfillment/design"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes2) != 0 {
		t.Fatalf("green run must exit 0: %v\n%s", *codes2, out2.String())
	}
	if !strings.Contains(out2.String(), "design-green: all applicable design gates green") {
		t.Fatalf("design-green line missing:\n%s", out2.String())
	}

	out3, _, _ := withCapturedIO(t)
	cmd = newCheckCmd()
	cmd.SetArgs([]string{"../../examples/fulfillment/design", "--gate", "g2"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out3.String(), "-green:") {
		t.Fatalf("an explicit subset must claim no green summary:\n%s", out3.String())
	}
}

// verify-c4 fails loudly when the engine is absent and succeeds through a
// stand-in binary, so the engine contract is pinned without Java in CI.
func TestVerifyC4(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	design := t.TempDir()
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errB.String(), "does not exist") {
		t.Fatalf("missing workspace.dsl must fail: codes=%v stderr=%q", *codes, errB.String())
	}

	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errB2, codes2 := withCapturedIO(t)
	t.Setenv("MACHINERY_STRUCTURIZR_CLI", filepath.Join(t.TempDir(), "missing-binary"))
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes2) == 0 || (*codes2)[0] != 1 {
		t.Fatalf("a missing engine must fail: codes=%v stderr=%q", *codes2, errB2.String())
	}

	fake := filepath.Join(t.TempDir(), "structurizr-cli")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out3, _, codes3 := withCapturedIO(t)
	t.Setenv("MACHINERY_STRUCTURIZR_CLI", fake)
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes3) != 0 || !strings.Contains(out3.String(), "verify-c4: ok") {
		t.Fatalf("a passing export must report ok: codes=%v out=%q", *codes3, out3.String())
	}

	failing := filepath.Join(t.TempDir(), "structurizr-cli")
	if err := os.WriteFile(failing, []byte("#!/bin/sh\necho 'parse error at line 1'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out4, errB4, codes4 := withCapturedIO(t)
	t.Setenv("MACHINERY_STRUCTURIZR_CLI", failing)
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if len(*codes4) == 0 || (*codes4)[0] != 1 ||
		!strings.Contains(out4.String(), "parse error at line 1") ||
		!strings.Contains(errB4.String(), "does not compile") {
		t.Fatalf("a failing export must pass the engine output through: codes=%v out=%q stderr=%q", *codes4, out4.String(), errB4.String())
	}
}
