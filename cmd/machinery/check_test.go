package main

import (
	"bytes"
	"os"
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
