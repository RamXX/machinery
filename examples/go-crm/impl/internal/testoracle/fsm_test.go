package testoracle

// These tests invoke the complete existing native User and Session suites.
// Controls and mutants use identical commands. They deliberately fail while
// unsafe oracle changes and extra effects are accepted by those assertions.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

var nativeUserLeaves = []string{
	"T-USER-01_USER-e20d04",
	"T-USER-02_notAdmin_USER-2b2218",
	"T-USER-03_USER-0ef83a",
	"T-USER-04_USER-e59219",
	"T-USER-05_notAdmin_USER-ffd41a",
	"T-USER-06_USER-799d7d",
	"T-USER-07_USER-930b15",
	"T-USER-08_USER-dd6c98",
	"T-USER-09_USER-7b324b",
	"T-USER-10_USER-dde0a6",
	"T-USER-11_USER-d2cfe6",
	"T-USER-12_USER-8e0d4c",
	"T-USER-13_USER-388821",
	"T-USER-14_USER-838e85",
	"T-USER-15_USER-a986a8",
	"T-USER-16_USER-1c13da",
	"T-USER-17_USER-081a5d",
	"T-USER-18_USER-adccd9",
	"T-USER-19_USER-7cf0fc",
	"USER-453743_/_fail-closed_rollback_routing",
}
var nativeSessionLeaves = []string{
	"T-SESS-01_SESS-ee5c17",
	"T-SESS-02_SESS-f3cc5e",
	"T-SESS-03_SESS-27e1b6",
	"T-SESS-04_SESS-fd60bd",
	"T-SESS-05_SESS-abcf53",
	"T-SESS-06_SESS-c78e53",
	"T-SESS-07_SESS-fbf094",
	"T-SESS-08_SESS-e12d58",
	"T-SESS-09_SESS-2dee87",
	"T-SESS-10_SESS-9c998d",
	"T-SESS-11_SESS-63844a",
	"T-SESS-12_SESS-aaa861",
	"T-SESS-13_SESS-316622",
	"T-SESS-14_SESS-b95638",
	"T-SESS-15_SESS-7b8023",
	"T-SESS-16_SESS-402e07",
	"T-SESS-17_SESS-e01d44",
	"T-SESS-18_SESS-e6484d",
	"T-SESS-19_SESS-4f7245",
	"T-SESS-20_SESS-dddfd5",
	"T-SESS-21_SESS-12a601",
	"T-SESS-22_SESS-c552bf",
	"T-SESS-23_SESS-85613f",
	"T-SESS-24_SESS-55a08c",
	"T-SESS-25_SESS-10b95d",
	"T-SESS-26_SESS-54c2ea",
	"T-SESS-27_SESS-a337ab",
	"T-SESS-28_SESS-aafc2f",
	"T-SESS-29_SESS-69da3c",
	"T-SESS-30_SESS-448752",
	"T-SESS-31_SESS-fd231f",
	"T-SESS-32_SESS-7fed3c",
	"T-SESS-33_SESS-87f3f3",
	"T-SESS-34_SESS-706156",
	"T-SESS-35_SESS-51ded9",
	"T-SESS-36_SESS-927687",
	"T-SESS-37_SESS-90aa72",
	"T-SESS-38_SESS-5e1b28",
	"T-SESS-39_SESS-2587f3",
	"T-SESS-40_SESS-25fc00",
	"T-SESS-41_SESS-bb2221",
	"T-SESS-42_SESS-3cee4a",
	"T-SESS-43_SESS-656e49",
	"T-SESS-44_SESS-f09ff1",
	"T-SESS-45_SESS-61d379",
	"T-SESS-46_SESS-f821b7",
	"T-SESS-47_SESS-65b326",
	"T-SESS-48_SESS-aca1a6",
	"T-SESS-49_SESS-5a47c2",
	"T-SESS-50_SESS-22bc20",
	"T-SESS-51_SESS-3e386e",
	"T-SESS-52_SESS-75e8c1",
	"T-SESS-53_SESS-f59ab7",
	"T-SESS-54_SESS-134dea",
	"T-SESS-55_SESS-de6aa7",
	"T-SESS-56_SESS-19fd5c",
	"T-SESS-57_SESS-e7bef7",
	"T-SESS-58_SESS-0488b7",
	"T-SESS-59_SESS-f6a536",
	"T-SESS-60_SESS-b830a0",
}

type nativeMutation struct{ path, before, after string }

func TestFSMNativeOracleSensitivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
	defer cancel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Clean(filepath.Join(cwd, "../../.."))
	proof, err := os.MkdirTemp("", "crm-fsm-native-proof-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained native proof directory: %s; toolchain=%s %s/%s", proof, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	cases := []struct {
		name, pkg, suite, target, diagnostic string
		leaves                               []string
		mutations                            []nativeMutation
	}{
		{
			name: "user_expected_next_state", pkg: "domain", suite: "TestUserTransitions",
			target: "T-USER-01_USER-e20d04", leaves: nativeUserLeaves,
			diagnostic: "oracle-next-state user USER-e20d04: got=\"persisting\" want=\"Disabled\"",
			mutations: []nativeMutation{
				{"design/machines/User.oracle.md",
					"| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |",
					"| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | Disabled | setPendingDisable |"},
				{"design/machines/User.machine.json",
					`{ "target": "persisting", "guard": "guardAdminAuthority", "actions": "setPendingDisable" }`,
					`{ "target": "Disabled", "guard": "guardAdminAuthority", "actions": "setPendingDisable" }`},
			},
		},
		{
			name: "user_expected_actions", pkg: "domain", suite: "TestUserTransitions",
			target: "T-USER-01_USER-e20d04", leaves: nativeUserLeaves,
			diagnostic: "oracle-actions user USER-e20d04: got=[\"setPendingDisable\"] want=[\"setPendingEnable\"]",
			mutations: []nativeMutation{
				{"design/machines/User.oracle.md",
					"| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |",
					"| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingEnable |"},
				{"design/machines/User.machine.json",
					`{ "target": "persisting", "guard": "guardAdminAuthority", "actions": "setPendingDisable" }`,
					`{ "target": "persisting", "guard": "guardAdminAuthority", "actions": "setPendingEnable" }`},
			},
		},
		{
			name: "session_empty_effect_extra_action", pkg: "session", suite: "TestSessionTransitions",
			target: "T-SESS-02_SESS-f3cc5e", leaves: nativeSessionLeaves,
			diagnostic: "oracle-actions session SESS-f3cc5e: got=[\"unexpectedOracleEffect\"] want=[]",
			mutations: []nativeMutation{
				{"impl/internal/session/machine.go",
					"func (m *SessionMachine) fireAnonymous(evt SessionEvent) model.Effect {\n\tswitch evt.Kind {\n\tcase SEvLogin:\n\t\tm.setCredentials(evt)\n\t\tm.State = SAuthenticating\n\t\treturn sEffect(\"setCredentials\")\n\tcase SEvResume:\n\t\tm.State = SResolving\n\t\treturn sEffect()",
					"func (m *SessionMachine) fireAnonymous(evt SessionEvent) model.Effect {\n\tswitch evt.Kind {\n\tcase SEvLogin:\n\t\tm.setCredentials(evt)\n\t\tm.State = SAuthenticating\n\t\treturn sEffect(\"setCredentials\")\n\tcase SEvResume:\n\t\tm.State = SResolving\n\t\treturn sEffect(\"unexpectedOracleEffect\")"},
			},
		},
	}
	original := nativeManifest(t, source)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			nativeCopy(t, source, fixture)
			before := nativeManifest(t, fixture)
			if !reflect.DeepEqual(before, original) {
				t.Fatal("fixture hash inventory differs from source")
			}
			caseProof := filepath.Join(proof, tc.name)
			if err := os.Mkdir(caseProof, 0700); err != nil {
				t.Fatal(err)
			}
			nativeWriteJSON(t, filepath.Join(caseProof, "control-manifest.json"), before)
			control := nativeRun(t, ctx, fixture, caseProof, "control", tc.pkg, tc.suite)
			nativeCheckRun(t, control, tc.suite, tc.leaves, "", "")
			if !reflect.DeepEqual(nativeManifest(t, fixture), before) {
				t.Fatal("control modified fixture source")
			}
			expected := make(map[string]string, len(before))
			for path, hash := range before {
				expected[path] = hash
			}
			for _, mutation := range tc.mutations {
				path := filepath.Join(fixture, filepath.FromSlash(mutation.path))
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.Count(raw, []byte(mutation.before)) != 1 {
					t.Fatalf("mutation anchor must match exactly once: %s", mutation.path)
				}
				changed := bytes.Replace(raw, []byte(mutation.before), []byte(mutation.after), 1)
				if bytes.Equal(raw, changed) {
					t.Fatalf("mutation did not change %s", mutation.path)
				}
				if err := os.WriteFile(path, changed, 0600); err != nil {
					t.Fatal(err)
				}
				expected[mutation.path] = fmt.Sprintf("%x", sha256.Sum256(changed))
				t.Logf("mutation %s: before=%s after=%s; old=%q new=%q", mutation.path, before[mutation.path], expected[mutation.path], mutation.before, mutation.after)
			}
			after := nativeManifest(t, fixture)
			if !reflect.DeepEqual(after, expected) {
				t.Fatal("mutation changed an unrelated file or missed intended content")
			}
			nativeWriteJSON(t, filepath.Join(caseProof, "mutant-manifest.json"), after)
			nativeWriteJSON(t, filepath.Join(caseProof, "mutations.json"), func() []map[string]string {
				var out []map[string]string
				for _, m := range tc.mutations {
					out = append(out, map[string]string{"path": m.path, "before": m.before, "after": m.after})
				}
				return out
			}())
			mutant := nativeRun(t, ctx, fixture, caseProof, "mutant", tc.pkg, tc.suite)
			if !reflect.DeepEqual(nativeManifest(t, fixture), after) {
				t.Fatal("mutant test modified fixture source")
			}
			nativeCheckRun(t, mutant, tc.suite, tc.leaves, tc.target, tc.diagnostic)
		})
	}
	if !reflect.DeepEqual(nativeManifest(t, source), original) {
		t.Fatal("shared source changed during native sensitivity execution")
	}
}

func nativeSelectedPaths(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(filepath.Join(root, "impl"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("fixture refuses symlink %s", path)
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", ".vault", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("fixture refuses nonregular file %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, machine := range []string{"Deal", "Task", "User", "Session", "CommandExecution"} {
		for _, suffix := range []string{".oracle.md", ".machine.json"} {
			paths = append(paths, "design/machines/"+machine+suffix)
		}
	}
	return paths
}

func nativeManifest(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, path := range nativeSelectedPaths(t, root) {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		out[path] = fmt.Sprintf("%x", sha256.Sum256(raw))
	}
	return out
}

func nativeCopy(t *testing.T, source, dest string) {
	t.Helper()
	for _, path := range nativeSelectedPaths(t, source) {
		raw, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(dest, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func nativeWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

type nativeResult struct {
	raw  []byte
	exit int
}
type nativeEvent struct{ Action, Package, Test, Output string }

func nativeRun(t *testing.T, parent context.Context, root, proof, label, pkg, suite string) nativeResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, 125*time.Second)
	defer cancel()
	args := []string{"test", "-count=1", "-timeout=120s", "./internal/" + pkg, "-run", "^" + suite + "$", "-json"}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = filepath.Join(root, "impl")
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if key != "GOWORK" && key != "GOPROXY" && key != "GOTOOLCHAIN" {
			cmd.Env = append(cmd.Env, item)
		}
	}
	cmd.Env = append(cmd.Env, "GOWORK=off", "GOPROXY=off", "GOTOOLCHAIN=local")
	cmd.WaitDelay = 5 * time.Second
	started := time.Now()
	raw, err := cmd.CombinedOutput()
	elapsed := time.Since(started)
	if writeErr := os.WriteFile(filepath.Join(proof, label+".jsonl"), raw, 0600); writeErr != nil {
		t.Fatal(writeErr)
	}
	exit := 0
	if err != nil {
		var exited *exec.ExitError
		if !errors.As(err, &exited) {
			t.Fatalf("native setup/process failure: %v; log=%s", err, proof)
		}
		exit = exited.ExitCode()
	}
	nativeWriteJSON(t, filepath.Join(proof, label+"-execution.json"), map[string]any{
		"command": append([]string{"go"}, args...), "cwd": cmd.Dir, "exit": exit,
		"duration": elapsed.String(), "sha256": fmt.Sprintf("%x", sha256.Sum256(raw)),
		"toolchain": runtime.Version(), "goos": runtime.GOOS, "goarch": runtime.GOARCH,
	})
	t.Logf("native %s: command=go %s cwd=%s exit=%d duration=%s sha256=%x log=%s", label, strings.Join(args, " "), cmd.Dir, exit, elapsed, sha256.Sum256(raw), filepath.Join(proof, label+".jsonl"))
	if ctx.Err() != nil {
		t.Fatalf("native timeout is not semantic sensitivity: %v", ctx.Err())
	}
	return nativeResult{raw, exit}
}

func nativeCheckRun(t *testing.T, result nativeResult, suite string, leaves []string, target, diagnostic string) {
	t.Helper()
	expected := map[string]bool{suite: true}
	for _, leaf := range leaves {
		name := suite + "/" + leaf
		if expected[name] {
			t.Fatalf("duplicate expected native identity %s", name)
		}
		expected[name] = true
	}
	if len(leaves) == 0 {
		t.Fatal("native suite inventory must not be empty")
	}
	runs, terminals := map[string]int{}, map[string]string{}
	output := map[string]string{}
	packageStart, packageEnd := 0, ""
	packageName := ""
	decoder := json.NewDecoder(bytes.NewReader(result.raw))
	for {
		var ev nativeEvent
		err := decoder.Decode(&ev)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid native JSON/setup output: %v", err)
		}
		if ev.Package == "" {
			t.Fatal("native event missing package")
		}
		if packageName == "" {
			packageName = ev.Package
		}
		if ev.Package != packageName {
			t.Fatalf("unexpected native package %s", ev.Package)
		}
		if ev.Test != "" && !expected[ev.Test] {
			t.Fatalf("unexpected native test identity %s", ev.Test)
		}
		switch ev.Action {
		case "run":
			runs[ev.Test]++
		case "pass", "fail", "skip":
			if ev.Action == "skip" {
				t.Fatalf("native skip is not sensitivity: %s", ev.Test)
			}
			if ev.Test == "" {
				if packageEnd != "" {
					t.Fatal("duplicate package completion")
				}
				packageEnd = ev.Action
			} else {
				if terminals[ev.Test] != "" {
					t.Fatalf("duplicate native completion %s", ev.Test)
				}
				terminals[ev.Test] = ev.Action
			}
		case "start":
			packageStart++
		case "output":
			output[ev.Test] += ev.Output
		case "pause", "cont":
			t.Fatalf("unexpected asynchronous native event %s", ev.Action)
		default:
			t.Fatalf("unexpected native event action %q", ev.Action)
		}
	}
	if packageStart != 1 || packageEnd == "" {
		t.Fatal("native package did not start and terminate exactly once")
	}
	for name := range expected {
		if runs[name] != 1 || terminals[name] == "" {
			t.Fatalf("native run did not terminate exactly once: %s runs=%d terminal=%q", name, runs[name], terminals[name])
		}
	}
	if len(runs) != len(expected) || len(terminals) != len(expected) {
		t.Fatal("native run/terminal inventory mismatch")
	}
	for _, text := range output {
		for _, bad := range []string{"panic:", "test timed out", "[build failed]", "no tests to run", "oracle-parse", "oracle-identity", "oracle-witness", "oracle-unused-row"} {
			if strings.Contains(text, bad) {
				t.Fatalf("native setup/panic/timeout outcome: %s", bad)
			}
		}
	}
	if target == "" {
		if result.exit != 0 || packageEnd != "pass" {
			t.Fatalf("unchanged native control failed: exit=%d package=%s", result.exit, packageEnd)
		}
		for name, outcome := range terminals {
			if outcome != "pass" {
				t.Fatalf("unchanged native control failed: %s", name)
			}
		}
		t.Logf("native control inventory: %d leaves PASS, 0 FAIL, 0 SKIP", len(leaves))
		return
	}
	targetName := suite + "/" + target
	if !expected[targetName] {
		t.Fatalf("target absent from frozen native inventory: %s", targetName)
	}
	if result.exit == 0 && packageEnd == "pass" {
		for name, outcome := range terminals {
			if outcome != "pass" {
				t.Fatalf("inconsistent successful native exit: %s=%s", name, outcome)
			}
		}
		t.Errorf("unsafe native variant was accepted: %s; all %d leaves PASS; required diagnostic %q", targetName, len(leaves), diagnostic)
		return
	}
	if result.exit != 1 || packageEnd != "fail" {
		t.Fatalf("unexpected native failure exit=%d package=%s", result.exit, packageEnd)
	}
	for name, outcome := range terminals {
		want := "pass"
		if name == suite || name == targetName {
			want = "fail"
		}
		if outcome != want {
			t.Fatalf("native failure attribution %s: got %s want %s", name, outcome, want)
		}
	}
	if !strings.Contains(output[targetName], diagnostic) {
		t.Fatalf("native target failed without intended conformance diagnostic %q; output=%s", diagnostic, output[targetName])
	}
	t.Logf("native mutant inventory: %d leaves PASS, 1 intended FAIL, 0 SKIP", len(leaves)-1)
}
