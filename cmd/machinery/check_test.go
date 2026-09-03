package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/testgit"

	"github.com/spf13/cobra"

	"github.com/RamXX/machinery/internal/gates"
	"github.com/RamXX/machinery/internal/ir"
	"github.com/RamXX/machinery/internal/oracle"
	"github.com/RamXX/machinery/internal/runtimeclosure"
	machversion "github.com/RamXX/machinery/internal/version"
)

// capturedExitCodes is the test process boundary. Commands now return typed
// exit statuses so every deferred cleanup and snapshot revalidation runs; the
// production main translates those statuses into os.Exit. Tests translate at
// this boundary instead of installing a deep exit hook.
var capturedExitCodes *[]int

func executeCaptured(err error) error {
	if capturedExitCodes == nil {
		return err
	}
	code, remaining := commandResult(err)
	if code != 0 {
		*capturedExitCodes = append(*capturedExitCodes, code)
	}
	return remaining
}

func executeDirectCaptured(err error) error {
	if capturedExitCodes == nil {
		return err
	}
	code, remaining := commandResult(err)
	if code != 0 {
		*capturedExitCodes = append(*capturedExitCodes, code)
	}
	if remaining != nil {
		return remaining
	}
	var status *exitStatusError
	if errors.As(err, &status) {
		return status.cause
	}
	return nil
}

func executeCapturedCommand(cmd *cobra.Command) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return executeCaptured(cmd.Execute())
}

func capturedOracleRun(mdir string, diff bool, against string) error {
	return executeDirectCaptured(oracleRun(mdir, diff, against))
}

func capturedEmbedRefreshRun(design string, dryRun bool) error {
	return executeDirectCaptured(embedRefreshRun(design, dryRun))
}

func capturedTokensEqualRun(oldPath, newPath string) error {
	return executeDirectCaptured(tokensEqualRun(oldPath, newPath))
}

func withCapturedIO(t *testing.T) (*bytes.Buffer, *bytes.Buffer, *[]int) {
	t.Helper()
	var out, errB bytes.Buffer
	var codes []int
	priorCodes := capturedExitCodes
	capturedExitCodes = &codes
	stdoutW, stderrW = &out, &errB
	t.Cleanup(func() {
		stdoutW, stderrW = os.Stdout, os.Stderr
		capturedExitCodes = priorCodes
	})
	return &out, &errB, &codes
}

func TestCheckGateG4RequiresImplCaseInsensitive(t *testing.T) {
	// Regression: `--gate G4` (uppercase) used to skip the requires-impl error
	// AND every gate, exiting 0 having verified nothing.
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "G4"})
	if err := executeCapturedCommand(cmd); err != nil {
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
	if err := executeCapturedCommand(cmd); err != nil {
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
	if err := executeCapturedCommand(cmd); err != nil {
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
	readCanonical := func(rel string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	writeWiring(".claude-plugin/plugin.json", readCanonical(".claude-plugin/plugin.json"), 0o644)
	writeWiring("hooks/hooks.json", readCanonical("hooks/hooks.json"), 0o644)
	writeWiring("hooks/machinery-hook.sh", readCanonical("hooks/machinery-hook.sh"), 0o644) // canonical but NOT executable
	t.Setenv("CLAUDE_PLUGIN_ROOT", plugin)

	var out bytes.Buffer
	reportHookWiring(&out)
	got := out.String()
	if !strings.Contains(got, "ok       hook manifest at "+filepath.Join(plugin, "hooks", "hooks.json")) {
		t.Errorf("manifest not reported ok:\n%s", got)
	}
	if !strings.Contains(got, "ERROR    hook shim at ") || !strings.Contains(got, "not executable") {
		t.Errorf("non-executable shim must ERROR:\n%s", got)
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
	if !strings.Contains(got, "ERROR    hook manifest") {
		t.Errorf("missing hooks.json must be ERROR:\n%s", got)
	}
	if !strings.Contains(got, "ok       hook shim at ") || !strings.Contains(got, "canonical digest") {
		t.Errorf("canonical executable shim must be ok:\n%s", got)
	}
}

func TestCheckUnknownGateStillErrors(t *testing.T) {
	_, errB, codes := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{"../../examples/go-crm/design", "--gate", "g9"})
	if err := executeCapturedCommand(cmd); err != nil {
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
	matrix := "| name | kind | contract |\n|---|---|---|\n" +
		"| `guardCanPublish` | guard | decides publish |\n" +
		"| `setPending` | action | records pending |\n" +
		"| `recordDenied` | action | records refusal |\n" +
		"| `commit` | action | commits |\n" +
		"| `recordError` | action | records error |\n" +
		"| `recordTimeout` | action | records timeout |\n" +
		"| `saveWidget` | actor | persists widget |\n"
	if err := os.WriteFile(filepath.Join(mdir, "Widget.matrix.md"), []byte(matrix), 0o644); err != nil {
		t.Fatal(err)
	}
	return design
}

func runCheckG3(t *testing.T, design string) string {
	t.Helper()
	out, _, _ := withCapturedIO(t)
	cmd := newCheckCmd()
	cmd.SetArgs([]string{design, "--gate", "g3"})
	if err := executeCapturedCommand(cmd); err != nil {
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

// `machinery baseline` derives its full date from the reproducible-build
// epoch, never from the host clock (which changes output across midnight).
func TestBaselineSourceDateEpochStampIsFullDate(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1788393600") // 2026-09-03T00:00:00Z
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
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("exit codes %v, want none", *codes)
	}
	today := "2026-09-03"
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

func TestResolveBaselineDateRequiresDeterministicSourceAndReusesExisting(t *testing.T) {
	design := t.TempDir()
	if _, err := resolveBaselineDate(design, "", ""); err == nil || !strings.Contains(err.Error(), "deterministic snapshot date is required") {
		t.Fatalf("missing deterministic date source did not fail: %v", err)
	}
	if got, err := resolveBaselineDate(design, "2026-09-04", "bad"); err != nil || got != "2026-09-04" {
		t.Fatalf("explicit date must win: got=%q err=%v", got, err)
	}
	if _, err := resolveBaselineDate(design, "2026-02-30", ""); err == nil || !strings.Contains(err.Error(), "canonical YYYY-MM-DD") {
		t.Fatalf("impossible explicit date did not fail: %v", err)
	}
	if got, err := resolveBaselineDate(design, "", "1788393600"); err != nil || got != "2026-09-03" {
		t.Fatalf("SOURCE_DATE_EPOCH was not resolved in UTC: got=%q err=%v", got, err)
	}
	if err := gates.WriteRatchet(design, &gates.Ratchet{Date: "2026-08", Edges: map[string][]string{}}); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveBaselineDate(design, "", ""); err != nil || got != "2026-08" {
		t.Fatalf("existing ratchet date was not reused: got=%q err=%v", got, err)
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
	if err := executeCapturedCommand(cmd); err != nil {
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
	if code != 1 || !strings.Contains(out, "does not name the supplied review target") {
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
	if out, code := runCheckGa(t, design); code != 1 || !strings.Contains(out, "is not an ancestor of history anchor") {
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
	if err := executeCapturedCommand(cmd); err != nil {
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
	if err := executeCapturedCommand(cmd); err != nil {
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
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out3.String(), "-green:") {
		t.Fatalf("an explicit subset must claim no green summary:\n%s", out3.String())
	}
}

// verify-c4 fails loudly when the engine is absent and succeeds through a
// stand-in binary, so the engine contract is pinned without Java in CI.
func setSupportedJava(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"bin", "conf", "lib"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "modules"), []byte("modules"), 0o644); err != nil {
		t.Fatal(err)
	}
	java := filepath.Join(root, "bin", "java")
	body := "#!/bin/sh\nif [ \"$1\" = '-XshowSettings:properties' ]; then\n  java_home=$(CDPATH= cd -- \"$(dirname -- \"$0\")/..\" && pwd -P)\n  echo \"    java.home = $java_home\" >&2\n  echo '    java.runtime.version = 21.0.12.1+1-LTS' >&2\n  echo '    java.vendor = Eclipse Adoptium' >&2\n  echo '    java.version = 21.0.12.1' >&2\n  echo '    java.vm.name = OpenJDK 64-Bit Server VM' >&2\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(java, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	digest, err := runtimeclosure.JavaClosureDigest(java)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MACHINERY_JAVA", java)
	t.Setenv(runtimeclosure.JavaClosureSHAEnv, digest)
	return java
}

func setStructurizrOverride(t *testing.T, launcher string) {
	t.Helper()
	real, err := filepath.EvalSymlinks(launcher)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := fingerprintStructurizrTree(filepath.Dir(real))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(structurizrEnv, launcher)
	t.Setenv(structurizrClosureSHAEnv, fmt.Sprintf("%x", digest))
}

const fakeStructurizrVersionBranch = `if [ "$1" = version ]; then
  echo 'structurizr-cli: 2025.11.09'
  echo 'structurizr-java: 5.0.2'
  echo 'Java: 21.0.12.1/Eclipse Adoptium (/fixture/java-home)'
  echo 'OS: Fixture OS (fixture)'
  exit 0
fi
`

const fakeStructurizrExportProgress = `echo "Exporting workspace from $3"
echo ' - exporting with MermaidDiagramExporter'
for view in "$7"/*.mmd; do
  [ -f "$view" ] || continue
  echo " - writing $view"
done
echo ' - finished'
`

func TestVerifyC4(t *testing.T) {
	setSupportedJava(t)
	_, errB, codes := withCapturedIO(t)
	design := t.TempDir()
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
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
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes2) == 0 || (*codes2)[0] != 1 {
		t.Fatalf("a missing engine must fail: codes=%v stderr=%q", *codes2, errB2.String())
	}

	fake := filepath.Join(t.TempDir(), "structurizr-cli")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"+fakeStructurizrVersionBranch+"printf 'graph TD\\n' > \"$7/view.mmd\"\n"+fakeStructurizrExportProgress+"exit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out3, _, codes3 := withCapturedIO(t)
	setStructurizrOverride(t, fake)
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes3) != 0 || !strings.Contains(out3.String(), "verify-c4: ok") {
		t.Fatalf("a passing export must report ok: codes=%v out=%q", *codes3, out3.String())
	}

	failing := filepath.Join(t.TempDir(), "structurizr-cli")
	if err := os.WriteFile(failing, []byte("#!/bin/sh\n"+fakeStructurizrVersionBranch+"echo 'parse error at line 1'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	out4, errB4, codes4 := withCapturedIO(t)
	setStructurizrOverride(t, failing)
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes4) == 0 || (*codes4)[0] != 1 ||
		!strings.Contains(out4.String(), "parse error at line 1") ||
		!strings.Contains(errB4.String(), "does not compile") {
		t.Fatalf("a failing export must pass the engine output through: codes=%v out=%q stderr=%q", *codes4, out4.String(), errB4.String())
	}
}

func TestVerifyC4PinsVersionAndExportsPrivateSnapshot(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(t.TempDir(), "args")
	fake := filepath.Join(t.TempDir(), "structurizr-cli")
	script := "#!/bin/sh\n" + fakeStructurizrVersionBranch + "printf '%s\\n' \"$@\" > \"" + logPath + "\"\nprintf 'graph TD\\n' > \"$7/view.mmd\"\n" + fakeStructurizrExportProgress
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, fake)
	oldAfterVersion := verifyC4AfterVersion
	verifyC4AfterVersion = func(string) {
		if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 99\n"), 0o755); err != nil {
			t.Errorf("replace ambient launcher: %v", err)
		}
	}
	t.Cleanup(func() { verifyC4AfterVersion = oldAfterVersion })
	_, _, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("snapshot export failed: %v", *codes)
	}
	args, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(args), filepath.Join(design, "workspace.dsl")) || !strings.Contains(string(args), "machinery-design-source-") {
		t.Fatalf("Structurizr received ambient design path instead of private snapshot: %s", args)
	}
	verifyC4AfterVersion = func(string) {}

	mismatch := filepath.Join(t.TempDir(), "structurizr-cli")
	mismatchScript := "#!/bin/sh\necho 'structurizr-cli: 0.0.1'\necho 'structurizr-java: 5.0.2'\necho 'Java: 21.0.12.1/Eclipse Adoptium (/fixture/java-home)'\necho 'OS: Fixture OS (fixture)'\n"
	if err := os.WriteFile(mismatch, []byte(mismatchScript), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, mismatch)
	_, errB, mismatchCodes := withCapturedIO(t)
	cmd = newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*mismatchCodes) == 0 || (*mismatchCodes)[0] != 1 || !strings.Contains(errB.String(), "does not match supported embedded version") {
		t.Fatalf("version mismatch passed: codes=%v stderr=%q", *mismatchCodes, errB.String())
	}
}

func TestVerifyC4SnapshotsOfficialLauncherDistribution(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	distribution := t.TempDir()
	if err := os.Mkdir(filepath.Join(distribution, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distribution, "lib", "version"), []byte("2025.11.09\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(distribution, "structurizr.sh")
	script := `#!/bin/sh
base=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
version=$(cat "$base/lib/version")
if [ "$1" = version ]; then
  echo "structurizr-cli: $version"
  echo 'structurizr-java: 5.0.2'
  echo 'Java: 21.0.12.1/Eclipse Adoptium (/fixture/java-home)'
  echo 'OS: Fixture OS (fixture)'
  exit 0
fi
test -f "$base/lib/version"
printf 'graph TD\n' > "$7/view.mmd"
echo "Exporting workspace from $3"
echo ' - exporting with MermaidDiagramExporter'
echo " - writing $7/view.mmd"
echo ' - finished'
`
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	_, errOut, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("official-layout snapshot failed: codes=%v stderr=%s", *codes, errOut)
	}
}

func TestVerifyC4SnapshotsWrapperLibThroughLauncherSymlink(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	distribution := t.TempDir()
	if err := os.Mkdir(filepath.Join(distribution, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distribution, "lib", "version"), []byte("2025.11.09\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(distribution, "structurizr-cli")
	script := "#!/bin/sh\nbase=$(CDPATH= cd -- \"$(dirname -- \"$0\")\" && pwd)\nversion=$(cat \"$base/lib/version\")\n" + fakeStructurizrVersionBranch + "test -f \"$base/lib/version\"\nprintf 'graph TD\\n' > \"$7/view.mmd\"\n" + fakeStructurizrExportProgress
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	linkDir := t.TempDir()
	link := filepath.Join(linkDir, "structurizr-cli")
	if err := os.Symlink(launcher, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	setStructurizrOverride(t, link)
	_, errOut, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) != 0 {
		t.Fatalf("wrapper/lib symlink snapshot failed: codes=%v stderr=%s", *codes, errOut)
	}
}

func TestVerifyC4FailureDiagnosticRedactsScratchAndIsStable(t *testing.T) {
	setSupportedJava(t)
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/hostile.jar")
	t.Setenv("JDK_JAVA_OPTIONS", "-Dhostile=true")
	t.Setenv("CLASSPATH", "/hostile/classpath")
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	script := "#!/bin/sh\n" + fakeStructurizrVersionBranch + "printf 'tool=%s\\nargs=%s\\nHOME=%s JAVA_TOOL_OPTIONS=%s JDK_JAVA_OPTIONS=%s CLASSPATH=%s TZ=%s LC_ALL=%s\\n' \"$0\" \"$*\" \"$HOME\" \"$JAVA_TOOL_OPTIONS\" \"$JDK_JAVA_OPTIONS\" \"$CLASSPATH\" \"$TZ\" \"$LC_ALL\"\nexit 9\n"
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	var want string
	for i := 0; i < 10; i++ {
		out, errOut, codes := withCapturedIO(t)
		cmd := newVerifyC4Cmd()
		cmd.SetArgs([]string{design})
		if err := executeCapturedCommand(cmd); err != nil {
			t.Fatal(err)
		}
		got := out.String() + errOut.String()
		if len(*codes) == 0 || (*codes)[0] != 1 || strings.Contains(got, "machinery-structurizr-tool-") || strings.Contains(got, "machinery-design-reader-") || strings.Contains(got, "machinery-verify-c4") {
			t.Fatalf("run %d leaked scratch or passed: codes=%v diagnostic=%q", i, *codes, got)
		}
		for _, forbidden := range []string{"hostile.jar", "hostile=true", "/hostile/classpath"} {
			if strings.Contains(got, forbidden) {
				t.Fatalf("hostile environment reached Structurizr: %s", got)
			}
		}
		for _, required := range []string{"TZ=UTC", "LC_ALL=C.UTF-8"} {
			if !strings.Contains(got, required) {
				t.Fatalf("deterministic environment lacks %s: %s", required, got)
			}
		}
		if i == 0 {
			want = got
		} else if got != want {
			t.Fatalf("run %d diagnostic changed:\nwant %s\n got %s", i, want, got)
		}
	}
}

func TestVerifyC4RejectsJavaPathSwapAfterProbe(t *testing.T) {
	java := setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\n"+fakeStructurizrVersionBranch), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	oldHook := verifyC4AfterJavaProbe
	verifyC4AfterJavaProbe = func(string) {
		if err := os.Rename(java, java+".old"); err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(java, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { verifyC4AfterJavaProbe = oldHook })
	_, errOut, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errOut.String(), "changed identity") {
		t.Fatalf("Java swap passed C4: codes=%v stderr=%s", *codes, errOut)
	}
}

func TestVerifyC4RejectsSelfMutatingPinnedDistribution(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	distribution := t.TempDir()
	if err := os.Mkdir(filepath.Join(distribution, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distribution, "lib", "state"), []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(distribution, "structurizr-cli")
	script := "#!/bin/sh\nif [ \"$1\" = version ]; then\n  echo 'structurizr-cli: 2025.11.09'\n  echo 'structurizr-java: 5.0.2'\n  echo 'Java: 21.0.12.1/Eclipse Adoptium (/fixture/java-home)'\n  echo 'OS: Fixture OS (fixture)'\n  echo mutated > \"$(dirname \"$0\")/lib/state\"\nfi\n"
	if err := os.WriteFile(launcher, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	_, errOut, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errOut.String(), "snapshot changed between version verification and export") {
		t.Fatalf("self-mutating same-version distribution passed: codes=%v stderr=%s", *codes, errOut)
	}
}

func TestVerifyC4RejectsModifiedSameVersionExplicitClosure(t *testing.T) {
	setSupportedJava(t)
	design := t.TempDir()
	if err := os.WriteFile(filepath.Join(design, "workspace.dsl"), []byte("workspace \"W\" \"d\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "structurizr-cli")
	body := "#!/bin/sh\n" + fakeStructurizrVersionBranch
	if err := os.WriteFile(launcher, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	setStructurizrOverride(t, launcher)
	if err := os.WriteFile(launcher, []byte(body+"# modified same-version bytes\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, errOut, codes := withCapturedIO(t)
	cmd := newVerifyC4Cmd()
	cmd.SetArgs([]string{design})
	if err := executeCapturedCommand(cmd); err != nil {
		t.Fatal(err)
	}
	if len(*codes) == 0 || (*codes)[0] != 1 || !strings.Contains(errOut.String(), "matching exact lowercase closure sha256") {
		t.Fatalf("modified explicit Structurizr closure passed: codes=%v stderr=%s", *codes, errOut)
	}
}

func TestStructurizrFingerprintExcludesOnlyRootReceipt(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	rootReceipt := filepath.Join(root, ".machinery-structurizr-receipt")
	nestedReceipt := filepath.Join(root, "lib", ".machinery-structurizr-receipt")
	if err := os.WriteFile(rootReceipt, []byte("root-a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nestedReceipt, []byte("nested-a"), 0o600); err != nil {
		t.Fatal(err)
	}
	first, err := fingerprintStructurizrTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootReceipt, []byte("root-b"), 0o600); err != nil {
		t.Fatal(err)
	}
	second, err := fingerprintStructurizrTree(root)
	if err != nil || second != first {
		t.Fatalf("root receipt affected closure: %x vs %x, %v", first, second, err)
	}
	if err := os.WriteFile(nestedReceipt, []byte("nested-b"), 0o600); err != nil {
		t.Fatal(err)
	}
	third, err := fingerprintStructurizrTree(root)
	if err != nil || third == second {
		t.Fatalf("nested receipt-named file was excluded: %x vs %x, %v", second, third, err)
	}
}

func TestProvisionOfficialStructurizrArchiveAndReuseCache(t *testing.T) {
	if os.Getenv("MACHINERY_TEST_OFFICIAL_STRUCTURIZR") != "1" {
		t.Skip("set MACHINERY_TEST_OFFICIAL_STRUCTURIZR=1 for the official archive/cache contract")
	}
	first, err := provisionStructurizr()
	if err != nil {
		t.Fatal(err)
	}
	firstDigest, err := fingerprintStructurizrTree(filepath.Dir(first))
	if err != nil {
		t.Fatal(err)
	}
	second, err := provisionStructurizr()
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := fingerprintStructurizrTree(filepath.Dir(second))
	if err != nil {
		t.Fatal(err)
	}
	if first != second || firstDigest != secondDigest {
		t.Fatalf("cache reuse changed Structurizr identity:\nfirst %s %x\nsecond %s %x", first, firstDigest, second, secondDigest)
	}
}
