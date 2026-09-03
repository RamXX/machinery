package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/RamXX/machinery/internal/runtimeclosure"
)

type brokenDiagnosticWriter struct{ err error }

func (writer brokenDiagnosticWriter) Write([]byte) (int, error) { return 0, writer.err }

func TestRunCommandReturnsExecutionErrors(t *testing.T) {
	_, err := runCommand(filepath.Join(t.TempDir(), "does-not-exist"), false, "--version")
	if err == nil {
		t.Fatal("runCommand discarded an execution error")
	}
}

func TestDiagnosticVersionIdentityParsersAreCanonical(t *testing.T) {
	for _, tc := range []struct {
		name  string
		parse func(string) (string, error)
		input string
		want  string
	}{
		{name: "modelith", parse: parseModelithVersion, input: "modelith version 0.4.0\n", want: "0.4.0"},
		{name: "modelith CRLF", parse: parseModelithVersion, input: "modelith version v0.4.0\r\n", want: "v0.4.0"},
		{name: "scorecard", parse: parseScorecardVersion, input: "OpenSSF Scorecard\n\nGitVersion:    v5.5.0\nGitCommit: abc\n", want: "v5.5.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.parse(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("parse(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
			}
		})
	}

	for _, tc := range []struct {
		name  string
		parse func(string) (string, error)
		input string
	}{
		{name: "modelith trailing prose", parse: parseModelithVersion, input: "warning modelith version 0.4.0\n"},
		{name: "modelith extra line", parse: parseModelithVersion, input: "modelith version 0.4.0\nextra\n"},
		{name: "scorecard missing identity", parse: parseScorecardVersion, input: "GitCommit: abc\n"},
		{name: "scorecard malformed identity", parse: parseScorecardVersion, input: "GitVersion: present\n"},
		{name: "scorecard malformed plus valid identity", parse: parseScorecardVersion, input: "GitVersion: present\nGitVersion: v5.5.0\n"},
		{name: "scorecard duplicate identity", parse: parseScorecardVersion, input: "GitVersion: v5.5.0\nGitVersion: v5.5.0\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := tc.parse(tc.input); err == nil {
				t.Fatalf("parse(%q) = %q, nil; want error", tc.input, got)
			}
		})
	}
}

func TestPreflightRejectsScorecardWithoutGitVersion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses POSIX scripts")
	}
	dir := t.TempDir()
	writeDiagnosticScript(t, dir, "modelith", "echo 'modelith version 0.4.0'\n")
	writeDiagnosticScript(t, dir, "scorecard", "echo 'GitCommit: abcdef'\n")
	t.Setenv("PATH", dir)
	t.Setenv(runtimeclosure.JavaEnv, "")
	t.Setenv(structurizrEnv, "")
	var output bytes.Buffer
	if err := preflightRunUnlockedTo(&output); err == nil {
		t.Fatal("preflight accepted scorecard without a GitVersion identity")
	}
	if got := output.String(); !strings.Contains(got, "ERROR    scorecard probe returned non-canonical identity: missing canonical GitVersion") || strings.Contains(got, "ok       scorecard") {
		t.Fatalf("scorecard identity diagnostic was fail-open:\n%s", got)
	}
}

func TestPreflightRejectsProbeWarningOnStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses POSIX scripts")
	}
	for _, tool := range []string{"modelith", "scorecard"} {
		t.Run(tool, func(t *testing.T) {
			dir := t.TempDir()
			modelithBody := "echo 'modelith version 0.4.0'\n"
			if tool == "modelith" {
				modelithBody += "echo warning >&2\n"
			}
			writeDiagnosticScript(t, dir, "modelith", modelithBody)
			scorecardBody := "echo 'GitVersion: v5.5.0'\n"
			if tool == "scorecard" {
				scorecardBody += "echo warning >&2\n"
			}
			writeDiagnosticScript(t, dir, "scorecard", scorecardBody)
			t.Setenv("PATH", dir)
			t.Setenv(runtimeclosure.JavaEnv, "")
			t.Setenv(structurizrEnv, "")
			var output bytes.Buffer
			if err := preflightRunUnlockedTo(&output); err == nil {
				t.Fatalf("preflight accepted %s warning on stderr", tool)
			}
			if got := output.String(); !strings.Contains(got, "ERROR    "+tool+" probe failed:") || strings.Contains(got, "ok       "+tool) {
				t.Fatalf("%s warning diagnostic was fail-open:\n%s", tool, got)
			}
		})
	}
}

func writeDiagnosticScript(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDoctorDefaultRoutesAllOutputToCustomSink(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("MACHINERY_CONFIG_DIR", privateTestConfigDir(t))
	t.Setenv(runtimeclosure.JavaEnv, "")
	t.Setenv(structurizrEnv, "")
	var global, custom bytes.Buffer
	oldStdout := stdoutW
	stdoutW = &global
	t.Cleanup(func() { stdoutW = oldStdout })
	_ = doctorRunUnlockedTo(nil, &custom)
	if global.Len() != 0 {
		t.Fatalf("default doctor bypassed custom sink:\n%s", global.String())
	}
	for _, marker := range []string{"machinery prerequisites:", "install status:"} {
		if !strings.Contains(custom.String(), marker) {
			t.Fatalf("custom doctor sink missing %q:\n%s", marker, custom.String())
		}
	}
}

func TestDoctorPropagatesOutputFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CLAUDE_PLUGIN_ROOT", "")
	t.Setenv("MACHINERY_CONFIG_DIR", privateTestConfigDir(t))
	t.Setenv(runtimeclosure.JavaEnv, "")
	t.Setenv(structurizrEnv, "")
	sentinel := errors.New("closed diagnostic sink")
	err := doctorRunUnlockedTo(nil, brokenDiagnosticWriter{err: sentinel})
	if !errors.Is(err, sentinel) {
		t.Fatalf("doctor discarded output failure: %v", err)
	}
}

func TestPreflightFailsWhenRequiredProbeFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX script")
	}
	dir := t.TempDir()
	t.Setenv("MACHINERY_CONFIG_DIR", privateTestConfigDir(t))
	modelith := filepath.Join(dir, "modelith")
	if err := os.WriteFile(modelith, []byte("#!/bin/sh\nexit 17\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	var output bytes.Buffer
	oldStdout := stdoutW
	stdoutW = &output
	t.Cleanup(func() { stdoutW = oldStdout })

	if err := preflightRun(); err == nil {
		t.Fatal("preflight succeeded after the required modelith probe failed")
	}
	if got := output.String(); !strings.Contains(got, "ERROR    modelith probe failed") || strings.Contains(got, "ok       modelith") {
		t.Fatalf("preflight reported a false success:\n%s", got)
	}
}

func TestPreflightRejectsArguments(t *testing.T) {
	if err := newPreflightCmd().Args(newPreflightCmd(), []string{"ignored"}); err == nil {
		t.Fatal("preflight accepted an ignored positional argument")
	}
}

func TestPreflightReportsUnsafeErrDotResolutionForEveryTool(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ErrDot PATH fixture uses Unix executable modes")
	}
	dir := t.TempDir()
	for _, name := range []string{"modelith", "java", "scorecard"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not executed\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	t.Setenv("PATH", ".")
	t.Setenv("GODEBUG", "execerrdot=1")
	t.Setenv(runtimeclosure.JavaEnv, "")
	t.Setenv(structurizrEnv, "")
	var output bytes.Buffer
	oldStdout := stdoutW
	stdoutW = &output
	t.Cleanup(func() { stdoutW = oldStdout })

	if err := preflightRunUnlocked(); err == nil {
		t.Fatal("preflight accepted current-directory executable resolution")
	}
	got := output.String()
	for _, name := range []string{"modelith", "java", "scorecard"} {
		want := "  ERROR    " + name + " executable resolution failed: resolve " + name + " on PATH: exec: \"" + name + "\": cannot run executable found relative to current directory\n"
		if !strings.Contains(got, want) {
			t.Errorf("preflight diagnostic missing exact ErrDot line %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "auto     verify-c4 provisions the checksum-pinned Structurizr CLI; no ambient executable is trusted") {
		t.Errorf("preflight did not report hermetic Structurizr provisioning:\n%s", got)
	}
	for _, misleading := range []string{"MISSING  modelith", "optional Temurin Java", "optional structurizr-cli", "optional scorecard"} {
		if strings.Contains(got, misleading) {
			t.Errorf("unsafe PATH resolution was also reported as %q:\n%s", misleading, got)
		}
	}
}

func TestPreflightReportsPermissionCandidateInsteadOfOptional(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission fixture")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "scorecard")
	if err := os.WriteFile(path, []byte("not executable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv(runtimeclosure.JavaEnv, "")
	t.Setenv(structurizrEnv, "")
	var output bytes.Buffer
	oldStdout := stdoutW
	stdoutW = &output
	t.Cleanup(func() { stdoutW = oldStdout })

	if err := preflightRunUnlocked(); err == nil {
		t.Fatal("preflight treated a non-executable PATH candidate as absent")
	}
	want := "  ERROR    scorecard executable resolution failed: resolve scorecard on PATH: PATH candidate " + path + " exists but is not an executable regular file (-rw-r--r--)\n"
	if got := output.String(); !strings.Contains(got, want) || strings.Contains(got, "optional scorecard") {
		t.Fatalf("permission diagnostic is not exact or remained optional; want %q:\n%s", want, got)
	}
}

func TestPreflightProbesOnlyPinnedStructurizrCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper uses a POSIX script")
	}
	for _, tc := range []struct {
		name   string
		body   string
		marker string
	}{
		{"wrong version", "#!/bin/sh\necho 'structurizr-cli: 0.0.1'\n", "ERROR    structurizr-cli probe failed"},
		{"broken probe", "#!/bin/sh\nexit 19\n", "ERROR    structurizr-cli probe failed"},
		{"pinned", "#!/bin/sh\n" + fakeStructurizrVersionBranch, "ok       structurizr-cli 2025.11.09"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setSupportedJava(t)
			fake := filepath.Join(t.TempDir(), "structurizr-cli")
			if err := os.WriteFile(fake, []byte(tc.body), 0o755); err != nil {
				t.Fatal(err)
			}
			setStructurizrOverride(t, fake)
			var output bytes.Buffer
			oldStdout := stdoutW
			stdoutW = &output
			t.Cleanup(func() { stdoutW = oldStdout })
			_ = preflightRunUnlocked()
			if !strings.Contains(output.String(), tc.marker) {
				t.Fatalf("Structurizr probe output missing %q:\n%s", tc.marker, output.String())
			}
		})
	}
}

func TestJavaMajor(t *testing.T) {
	for _, tc := range []struct {
		line string
		want int
	}{
		{`openjdk version "25.0.2" 2026-01-20`, 25},
		{`java version "11.0.24" 2024-07-16 LTS`, 11},
		{`java version "1.8.0_442"`, 8},
	} {
		got, err := javaMajor(tc.line)
		if err != nil || got != tc.want {
			t.Errorf("javaMajor(%q) = %d, %v; want %d", tc.line, got, err, tc.want)
		}
	}
	if _, err := javaMajor("not actually java"); err == nil {
		t.Fatal("javaMajor accepted unrecognized output")
	}
}

func TestDoctorRejectsCorruptPluginIdentityAndSymlinkShim(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{"name":"not-machinery"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), []byte(`{"hooks":{"Stop":[]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "real-shim")
	if err := os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	shim := filepath.Join(root, "hooks", "machinery-hook.sh")
	if err := os.Symlink(target, shim); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
	var output bytes.Buffer
	if reportHookWiring(&output) {
		t.Fatalf("doctor accepted corrupt plugin layout:\n%s", output.String())
	}
	if got := output.String(); !strings.Contains(got, "wrong identity") || !strings.Contains(got, "not a real regular file") {
		t.Fatalf("doctor diagnostics incomplete:\n%s", got)
	}
}

func TestDoctorDoesNotFilterCorruptPluginCandidates(t *testing.T) {
	t.Run("manifest without hooks", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".claude-plugin"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".claude-plugin", "plugin.json"), []byte(`{"name":"machinery"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CLAUDE_PLUGIN_ROOT", root)
		var output bytes.Buffer
		if reportHookWiring(&output) || !strings.Contains(output.String(), "hook manifest") || !strings.Contains(output.String(), "hook shim") {
			t.Fatalf("missing hook layout was filtered or accepted:\n%s", output.String())
		}
	})

	t.Run("plugin enumeration error", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("CLAUDE_PLUGIN_ROOT", "")
		if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".claude", "plugins"), []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		var output bytes.Buffer
		if reportHookWiring(&output) || !strings.Contains(output.String(), "plugin discovery failed") {
			t.Fatalf("plugin enumeration error was hidden:\n%s", output.String())
		}
	})
}

func TestDoctorRejectsCorruptNonemptyHookManifestAndArbitraryExecutable(t *testing.T) {
	root := t.TempDir()
	copyDoctorFixture(t, root, ".claude-plugin/plugin.json", filepath.Join("..", "..", ".claude-plugin", "plugin.json"), 0o644)
	if err := os.MkdirAll(filepath.Join(root, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Both files are non-empty and the shim is executable; neither is the
	// shipped governance contract.
	if err := os.WriteFile(filepath.Join(root, "hooks", "hooks.json"), []byte(`{"description":"looks plausible","hooks":{"Stop":[]},"unknown":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "hooks", "machinery-hook.sh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_PLUGIN_ROOT", root)
	var output bytes.Buffer
	if reportHookWiring(&output) {
		t.Fatalf("doctor accepted corrupt nonempty hook assets:\n%s", output.String())
	}
	got := output.String()
	if !strings.Contains(got, "unknown field") || !strings.Contains(got, "wrong identity") {
		t.Fatalf("doctor did not report both strict failures:\n%s", got)
	}
}

func TestValidateDoctorHookManifestRejectsAmbiguousAndDriftedJSON(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "..", "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"duplicate", `{"description":"x","description":"y","hooks":{}}`, "duplicate JSON field"},
		{"trailing", string(canonical) + ` {}`, "trailing"},
		{"async", strings.Replace(string(canonical), `"timeout": 15`, `"timeout": 15, "async": true`, 1), "async override"},
		{"explicit async false", strings.Replace(string(canonical), `"timeout": 15`, `"timeout": 15, "async": false`, 1), "no async override"},
		{"timeout", strings.Replace(string(canonical), `"timeout": 180`, `"timeout": 179`, 1), "canonical shim contract"},
		{"command", strings.Replace(string(canonical), `machinery-hook.sh`, `other-hook.sh`, 1), "canonical shim contract"},
		{"event inventory", strings.Replace(string(canonical), `"PreToolUse"`, `"BeforeTool"`, 1), "event PreToolUse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "hooks.json")
			if err := os.WriteFile(path, []byte(tt.raw), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := validateDoctorHookManifest(path); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateDoctorHookManifest error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestCanonicalDoctorHookAssetsMatchRepository(t *testing.T) {
	manifest := filepath.Join("..", "..", "hooks", "hooks.json")
	if err := validateDoctorHookManifest(manifest); err != nil {
		t.Fatalf("shipped hook manifest is not canonical: %v", err)
	}
	shim := filepath.Join("..", "..", "hooks", "machinery-hook.sh")
	if err := validateDoctorHookShim(shim); err != nil {
		t.Fatalf("shipped hook shim digest is stale: %v", err)
	}
}

func copyDoctorFixture(t *testing.T, root, rel, source string, mode os.FileMode) {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, body, mode); err != nil {
		t.Fatal(err)
	}
}
