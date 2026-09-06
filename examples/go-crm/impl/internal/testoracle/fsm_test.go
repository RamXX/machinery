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
	"flag"
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

var nativeProofParent = flag.String("fsm-native-proof-dir", "", "Retain native proof under this existing absolute directory (default: clean test temporary output)")

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
	proof := t.TempDir()
	retained := *nativeProofParent != ""
	if retained {
		if !filepath.IsAbs(*nativeProofParent) {
			t.Fatal("fsm-native-proof-dir must be an existing absolute directory")
		}
		proof, err = os.MkdirTemp(*nativeProofParent, "crm-fsm-native-proof-")
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("native proof directory: %s; retained=%t; toolchain=%s %s/%s", proof, retained, runtime.Version(), runtime.GOOS, runtime.GOARCH)
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

var candidateDealLeaves = []string{
	"T-DEAL-01_DEAL-eb0c40",
	"DEAL-eb0c40a_/_notWritable_/_DEAL-38ba11",
	"DEAL-eb0c40b_/_negAmount_/_DEAL-38ba11",
	"T-DEAL-03_DEAL-1fe825",
	"DEAL-1fe825a_/_noCloseDate_/_DEAL-e786d8",
	"DEAL-1fe825b_/_notWritable_/_DEAL-e786d8",
	"DEAL-1fe825c_/_negAmount_/_DEAL-e786d8",
	"T-DEAL-05_DEAL-b76457",
	"DEAL-b76457a_/_notWritable_/_DEAL-fdf795",
	"DEAL-b76457b_/_negAmount_/_DEAL-fdf795",
	"T-DEAL-07_DEAL-1d9aa0",
	"T-DEAL-08_DEAL-a14020",
	"DEAL-a14020a_/_notWritable_/_DEAL-0c4c47",
	"DEAL-a14020b_/_negAmount_/_DEAL-0c4c47",
	"T-DEAL-10_DEAL-492234",
	"DEAL-492234a_/_noCloseDate_/_DEAL-81d0ab",
	"DEAL-492234b_/_notWritable_/_DEAL-81d0ab",
	"DEAL-492234c_/_negAmount_/_DEAL-81d0ab",
	"T-DEAL-12_DEAL-f7d8b2",
	"DEAL-f7d8b2a_/_notWritable_/_DEAL-9f48af",
	"DEAL-f7d8b2b_/_negAmount_/_DEAL-9f48af",
	"T-DEAL-14_DEAL-990c3b",
	"T-DEAL-15_DEAL-388687",
	"DEAL-388687a_/_notWritable_/_DEAL-5df488",
	"DEAL-388687b_/_negAmount_/_DEAL-5df488",
	"T-DEAL-17_DEAL-7e1e9b",
	"DEAL-7e1e9ba_/_noCloseDate_/_DEAL-df4442",
	"DEAL-7e1e9bb_/_notWritable_/_DEAL-df4442",
	"DEAL-7e1e9bc_/_negAmount_/_DEAL-df4442",
	"T-DEAL-19_DEAL-fde084",
	"DEAL-fde084a_/_notWritable_/_DEAL-e16eea",
	"DEAL-fde084b_/_negAmount_/_DEAL-e16eea",
	"T-DEAL-21_DEAL-44482d",
	"T-DEAL-22_DEAL-708606",
	"T-DEAL-23_DEAL-38140e",
	"DEAL-38140ea_/_noCloseDate_/_DEAL-3bbe10",
	"DEAL-38140eb_/_notWritable_/_DEAL-3bbe10",
	"DEAL-38140ec_/_negAmount_/_DEAL-3bbe10",
	"T-DEAL-25_DEAL-8fde14",
	"DEAL-8fde14a_/_notWritable_/_DEAL-b5154b",
	"DEAL-8fde14b_/_negAmount_/_DEAL-b5154b",
	"T-DEAL-27_DEAL-69312c",
	"T-DEAL-28_DEAL-99392a",
	"T-DEAL-29a_repNotAuthority_DEAL-5746cc",
	"T-DEAL-29b_mgrOutOfScope_DEAL-5746cc",
	"T-DEAL-30_DEAL-e0bdaf",
	"T-DEAL-31_DEAL-d27905",
	"T-DEAL-32_DEAL-a45f13",
	"T-DEAL-33_DEAL-0fef3d",
	"T-DEAL-34a_repNotAuthority_DEAL-7bb594",
	"T-DEAL-34b_mgrOutOfScope_DEAL-7bb594",
	"T-DEAL-35_DEAL-0a25a2",
	"T-DEAL-36_DEAL-0ec705",
	"T-DEAL-37_DEAL-e9e60a",
	"T-DEAL-38_DEAL-5abbd2",
	"T-DEAL-39_DEAL-da0ce2",
	"T-DEAL-40_DEAL-47ce0d",
	"T-DEAL-41_DEAL-e5d58e",
	"T-DEAL-42_DEAL-03d4fb",
	"T-DEAL-43_DEAL-92b688",
	"T-DEAL-44_DEAL-809c09",
	"T-DEAL-45_DEAL-cf2596",
	"T-DEAL-46_DEAL-daae59",
	"T-DEAL-47_DEAL-41c002",
	"T-DEAL-48_DEAL-7d1911",
	"T-DEAL-49_DEAL-24f320",
	"T-DEAL-50_DEAL-8c9948",
	"T-DEAL-51_DEAL-450b55",
	"T-DEAL-52_DEAL-210c14",
	"T-DEAL-53_DEAL-793e1f",
	"T-DEAL-54_DEAL-97c3ea",
	"T-DEAL-55_DEAL-8a4caf",
	"T-DEAL-56_DEAL-9b6ee7",
	"T-DEAL-57_DEAL-21905a",
	"DEAL-b48a23_/_fail-closed_rollback_routing",
}
var candidateTaskLeaves = []string{
	"T-TASK-01_TASK-db41f8",
	"T-TASK-02_notWritable_TASK-2a7cdb",
	"T-TASK-03_TASK-2019ec",
	"TASK-2019eca_/_notWritable_/_TASK-84d702",
	"T-TASK-05_TASK-b819d1",
	"TASK-b819d1a_/_notWritable_/_TASK-36d38a",
	"T-TASK-07_TASK-7ab0ac",
	"TASK-7ab0aca_/_assigneeOutOfScope_/_TASK-b179c7",
	"TASK-7ab0acb_/_callerNotAuthority_/_TASK-b179c7",
	"TASK-7ab0acc_/_sourceOutOfWriteScope_/_TASK-b179c7",
	"T-TASK-09_TASK-173f61",
	"T-TASK-10_TASK-72ad76",
	"TASK-72ad76a_/_notWritable_/_TASK-7d91c2",
	"T-TASK-12_TASK-cdda50",
	"TASK-cdda50a_/_notWritable_/_TASK-d159c9",
	"T-TASK-14_TASK-2f2bc8",
	"TASK-2f2bc8a_/_assigneeOutOfScope_/_TASK-91fb4d",
	"TASK-2f2bc8b_/_callerNotAuthority_/_TASK-91fb4d",
	"TASK-2f2bc8c_/_sourceOutOfWriteScope_/_TASK-91fb4d",
	"T-TASK-18_TASK-6d5eb1",
	"T-TASK-19_TASK-ae4260",
	"T-TASK-20_TASK-c56bd7",
	"T-TASK-21_TASK-67b0ff",
	"T-TASK-22_TASK-d5bcc8",
	"T-TASK-23_TASK-8d6955",
	"T-TASK-24_TASK-376b22",
	"T-TASK-25_TASK-dc5fe1",
	"T-TASK-26_TASK-21e793",
	"T-TASK-27_TASK-be8721",
	"T-TASK-28_TASK-b4999d",
	"T-TASK-29_TASK-0dd646",
	"T-TASK-30_TASK-168d9b",
	"T-TASK-31_TASK-3f585f",
	"T-TASK-32_TASK-98c3ba",
	"TASK-754183_/_fail-closed_rollback_routing",
}
var candidateCommandLeaves = []string{
	"T-CMD-01_COMM-44671c",
	"T-CMD-02_COMM-6f50f3",
	"T-CMD-03_COMM-5bc5e0",
	"T-CMD-04_COMM-ea53cf",
	"T-CMD-05_COMM-b9aee2",
	"T-CMD-06_COMM-8a2a55",
	"T-CMD-07_COMM-343b9c",
	"T-CMD-08_COMM-5e3106",
	"T-CMD-09_COMM-00c530",
	"T-CMD-10_COMM-215300",
	"T-CMD-11_COMM-71162c",
	"T-CMD-12_COMM-968d17",
	"T-CMD-13_COMM-ed4f93",
	"T-CMD-14_COMM-cc7919",
	"T-CMD-15_COMM-22d79f",
	"T-CMD-16_COMM-2151b8",
	"T-CMD-17_COMM-35a500",
	"T-CMD-18_COMM-8c204a",
	"T-CMD-19_COMM-7f1685",
	"T-CMD-20_COMM-5d7be9",
	"T-CMD-21_COMM-ec7aeb",
	"T-CMD-22_COMM-d6cfde",
	"T-CMD-23_COMM-8be203",
	"T-CMD-24_COMM-40743b",
	"T-CMD-25_COMM-cb11e8",
	"T-CMD-26_COMM-0b53b2",
	"T-CMD-27_COMM-84ddf1",
	"T-CMD-28_COMM-121e81",
}

// The following contracts are an additive expansion of the frozen semantic
// RED file. ErrScaffold failures establish no new qualifying behavioral RED.

func probeID(source, trigger, guard string) string {
	sum := sha256.Sum256([]byte("PROB|" + source + "|" + trigger + "|" + guard))
	return fmt.Sprintf("PROB-%x", sum[:3])
}

func probeFixture(t *testing.T, states []State, rows []Row) ([]byte, []byte) {
	t.Helper()
	md := "# Generated transition oracle: `probe`\n\n## State entry / exit actions\n\n| state | kind | entry | exit |\n|---|---|---|---|\n"
	cell := func(actions []string) string {
		if len(actions) == 0 {
			return "-"
		}
		return strings.Join(actions, ", ")
	}
	jsonStates := map[string]any{}
	for _, state := range states {
		md += fmt.Sprintf("| %s | %s | %s | %s |\n", state.Name, state.Kind, cell(state.Entry), cell(state.Exit))
		object := map[string]any{}
		if state.Kind == "final" {
			object["type"] = "final"
		}
		if len(state.Entry) > 0 {
			object["entry"] = state.Entry
		}
		if len(state.Exit) > 0 {
			object["exit"] = state.Exit
		}
		jsonStates[state.Name] = object
	}
	md += "\n## Transitions\n\n| test id | stable id | source | trigger | guard | target | actions |\n|---|---|---|---|---|---|---|\n"
	for i, row := range rows {
		guard := row.Guard
		if guard == "" {
			guard = "-"
		}
		md += fmt.Sprintf("| T-PROB-%02d | %s | %s | %s | %s | %s | %s |\n", i+1, row.ID, row.Source, row.Trigger, guard, row.Target, cell(row.Actions))
		tr := map[string]any{}
		if row.Target != "(internal)" {
			tr["target"] = row.Target
		}
		if row.Guard != "" {
			tr["guard"] = row.Guard
		}
		if len(row.Actions) > 0 {
			tr["actions"] = row.Actions
		}
		state := jsonStates[row.Source].(map[string]any)
		kind, key, _ := strings.Cut(row.Trigger, ":")
		if kind == "always" {
			old, _ := state["always"].([]any)
			state["always"] = append(old, tr)
			continue
		}
		if kind == "onDone" || kind == "onError" {
			invoke, ok := state["invoke"].(map[string]any)
			if !ok {
				invoke = map[string]any{"src": key, "input": map[string]any{}}
				state["invoke"] = invoke
			}
			old, _ := invoke[kind].([]any)
			invoke[kind] = append(old, tr)
			continue
		}
		events, ok := state[kind].(map[string]any)
		if !ok {
			events = map[string]any{}
			state[kind] = events
		}
		old, _ := events[key].([]any)
		events[key] = append(old, tr)
	}
	md += fmt.Sprintf("\nTotal transitions (test cases): %d\n", len(rows))
	raw, err := json.Marshal(map[string]any{"id": "probe", "initial": states[0].Name, "context": map[string]any{}, "states": jsonStates})
	if err != nil {
		t.Fatal(err)
	}
	return []byte(md), raw
}

func probeRow(source, trigger, guard, target string, actions ...string) Row {
	return Row{Identity: Identity{ID: probeID(source, trigger, guard), Source: source, Trigger: trigger, Guard: guard, Target: target}, Actions: actions}
}
func parsedProbe(t *testing.T, md, raw []byte) *Suite {
	t.Helper()
	suite, err := Parse(md, raw)
	if err != nil {
		t.Fatalf("valid oracle fixture must parse: %v", err)
	}
	return suite
}
func requireOracleError(t *testing.T, err error, prefix string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), prefix) {
		t.Fatalf("want diagnostic %q, got %v", prefix, err)
	}
}

func TestFSMParseCommittedMachines(t *testing.T) {
	for _, name := range []string{"Deal", "Task", "User", "Session", "CommandExecution"} {
		t.Run(name, func(t *testing.T) {
			md, err := os.ReadFile("../../../design/machines/" + name + ".oracle.md")
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile("../../../design/machines/" + name + ".machine.json")
			if err != nil {
				t.Fatal(err)
			}
			suite := parsedProbe(t, md, raw)
			loaded, err := Load("../../../design/machines", name)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(suite.Rows, loaded.Rows) || !reflect.DeepEqual(suite.States, loaded.States) {
				t.Fatal("Load must read the same complete current files")
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			if suite.Name != doc["id"] {
				t.Fatal("parsed machine identity mismatch")
			}
			var identities []Identity
			var rowActions [][]string
			stateDeclarations := map[string]State{}
			actionsCell := func(cell string) []string {
				if cell == "-" {
					return nil
				}
				return strings.Split(cell, ", ")
			}
			stateNames := map[string]bool{}
			inStates := false
			for _, line := range strings.Split(string(md), "\n") {
				if line == "## State entry / exit actions" {
					inStates = true
					continue
				}
				if line == "## Transitions" {
					inStates = false
				}
				if strings.HasPrefix(line, "| T-") {
					cells := strings.Split(line, "|")
					for i := range cells {
						cells[i] = strings.TrimSpace(cells[i])
					}
					guard := cells[5]
					if guard == "-" {
						guard = ""
					}
					identities = append(identities, Identity{cells[2], cells[3], cells[4], guard, cells[6]})
					rowActions = append(rowActions, actionsCell(cells[7]))
				} else if inStates && strings.HasPrefix(line, "| ") && !strings.HasPrefix(line, "| state ") {
					cells := strings.Split(line, "|")
					for i := range cells {
						cells[i] = strings.TrimSpace(cells[i])
					}
					stateNames[cells[1]] = true
					stateDeclarations[cells[1]] = State{Name: cells[1], Kind: cells[2], Entry: actionsCell(cells[3]), Exit: actionsCell(cells[4])}
				}
			}
			if len(identities) == 0 || len(suite.Rows) != len(identities) || len(suite.States) != len(stateNames) {
				t.Fatal("parsed current row/state inventory incomplete")
			}
			for i, row := range suite.Rows {
				if row.Identity != identities[i] {
					t.Fatalf("current identity/order mismatch at %d: got=%+v want=%+v", i, row.Identity, identities[i])
				}
				if !reflect.DeepEqual(append([]string{}, row.Actions...), append([]string{}, rowActions[i]...)) {
					t.Fatalf("current actions mismatch at %s", row.ID)
				}
			}
			for _, state := range suite.States {
				if !stateNames[state.Name] {
					t.Fatalf("unknown current state %q", state.Name)
				}
				want := stateDeclarations[state.Name]
				if state.Kind != want.Kind || !reflect.DeepEqual(append([]string{}, state.Entry...), append([]string{}, want.Entry...)) || !reflect.DeepEqual(append([]string{}, state.Exit...), append([]string{}, want.Exit...)) {
					t.Fatalf("current state kind/entry/exit mismatch at %s", state.Name)
				}
				delete(stateNames, state.Name)
			}
			if len(stateNames) != 0 {
				t.Fatal("missing state declarations")
			}
		})
	}
}

func TestFSMEffectsReconcileEntryExitAndInternal(t *testing.T) {
	cases := []struct {
		name, target     string
		transition, want []string
		next             string
	}{
		{"external", "B", []string{"transition"}, []string{"exitA", "transition", "enterB"}, "B"},
		{"internal", "(internal)", []string{"transition"}, []string{"transition"}, "A"},
		{"explicit_self", "A", []string{"transition"}, []string{"exitA", "transition", "enterA"}, "A"},
		{"empty_transition_with_entry", "B", nil, []string{"exitA", "enterB"}, "B"},
		{"empty_internal", "(internal)", nil, nil, "A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := probeRow("A", "on:go", "", ""+tc.target, tc.transition...)
			md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic", Entry: []string{"enterA"}, Exit: []string{"exitA"}}, {Name: "B", Kind: "final", Entry: []string{"enterB"}}}, []Row{row})
			suite := parsedProbe(t, md, raw)
			got, err := suite.Bind(Witness{Name: tc.name, RowID: row.ID, Source: "A", Trigger: "on:go"})
			if err != nil {
				t.Fatal(err)
			}
			if got.Next != tc.next || !reflect.DeepEqual(append([]string{}, got.Actions...), append([]string{}, tc.want...)) {
				t.Fatalf("one-Fire expectation: got=%q/%q want=%q/%q", got.Next, got.Actions, tc.next, tc.want)
			}
			if err := got.Check(tc.next, tc.want); err != nil {
				t.Fatal(err)
			}
			if err := suite.Finish(); err != nil {
				t.Fatal(err)
			}
			requireOracleError(t, got.Check("wrong", tc.want), "oracle-next-state probe "+row.ID+": got=\"wrong\" want=\""+tc.next+"\"")
			variants := [][]string{append(append([]string{}, tc.want...), "unrelated")}
			if len(tc.want) > 0 {
				variants = append(variants, tc.want[1:], append(append([]string{}, tc.want...), tc.want[0]))
			}
			if len(tc.want) > 1 {
				reordered := append([]string{}, tc.want...)
				reordered[0], reordered[1] = reordered[1], reordered[0]
				variants = append(variants, reordered)
			}
			for _, bad := range variants {
				requireOracleError(t, got.Check(tc.next, bad), "oracle-actions probe "+row.ID)
			}
		})
	}
}

func TestFSMBindPriorityAndClosedInventory(t *testing.T) {
	rows := []Row{probeRow("A", "on:go", "first", "B", "firstAction"), probeRow("A", "on:go", "second", "B", "secondAction"), probeRow("A", "on:go", "", "(internal)", "fallback")}
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, rows)
	cases := []struct {
		name  string
		index int
		facts map[string]bool
		bad   string
	}{
		{"first_with_both_true", 0, map[string]bool{"first": true, "second": true}, ""},
		{"second_with_first_false", 1, map[string]bool{"first": false, "second": true}, ""},
		{"fallback_all_false", 2, map[string]bool{"first": false, "second": false}, ""},
		{"wrong_priority", 1, map[string]bool{"first": true, "second": true}, "oracle-witness"},
		{"wrong_fallback", 2, map[string]bool{"first": false, "second": true}, "oracle-witness"},
		{"missing_fact", 0, map[string]bool{"first": true}, "oracle-witness"},
		{"unknown_fact", 0, map[string]bool{"first": true, "second": false, "invented": true}, "oracle-witness"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			suite := parsedProbe(t, md, raw)
			w := Witness{Name: tc.name, RowID: rows[tc.index].ID, Source: "A", Trigger: "on:go", Guards: tc.facts}
			exp, err := suite.Bind(w)
			if tc.bad != "" {
				requireOracleError(t, err, tc.bad)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if exp.Row.Identity != rows[tc.index].Identity {
				t.Fatal("selected wrong ordered alternative")
			}
			requireOracleError(t, suite.Finish(), "oracle-unused-row")
			_, err = suite.Bind(w)
			requireOracleError(t, err, "oracle-witness")
		})
	}
	for _, field := range []string{"row", "source", "trigger", "name"} {
		t.Run("bad_"+field, func(t *testing.T) {
			suite := parsedProbe(t, md, raw)
			w := Witness{Name: "w", RowID: rows[0].ID, Source: "A", Trigger: "on:go", Guards: map[string]bool{"first": true, "second": false}}
			switch field {
			case "row":
				w.RowID = "PROB-ffffff"
			case "source":
				w.Source = "B"
			case "trigger":
				w.Trigger = "on:unknown"
			case "name":
				w.Name = ""
			}
			_, err := suite.Bind(w)
			requireOracleError(t, err, "oracle-witness")
		})
	}
	suite := parsedProbe(t, md, raw)
	for i, row := range rows {
		_, err := suite.Bind(Witness{Name: fmt.Sprint(i), RowID: row.ID, Source: "A", Trigger: "on:go", Guards: map[string]bool{"first": i == 0, "second": i == 1}})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := suite.Finish(); err != nil {
		t.Fatal(err)
	}
}

func TestFSMParseRejectsMalformedOrIncomplete(t *testing.T) {
	row := probeRow("A", "on:go", "ready", "B", "act")
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, []Row{row})
	// Each malformed counterpart has this matched valid parser control first.
	parsedProbe(t, md, raw)
	variants := []struct{ name, old, new string }{
		{"missing_state_table", "## State entry / exit actions", "## Removed"},
		{"wrong_transition_columns", "| test id | stable id | source | trigger | guard | target | actions |", "| test id | stable id | source | trigger | target | actions |"},
		{"bad_separator", "|---|---|---|---|---|---|---|", "|---|---|bad|---|---|---|---|"},
		{"bad_footer", "Total transitions (test cases): 1", "Total transitions (test cases): 2"},
		{"blank_guard", "| ready |", "|  |"},
		{"blank_action_member", "| act |", "| act,  |"},
		{"unknown_target", "| B | act |", "| Missing | act |"},
		{"unknown_source", "| A | on:go |", "| Missing | on:go |"},
		{"bad_trigger", "| on:go |", "| invoke:go |"},
		{"bad_kind", "| A | atomic |", "| A | parallel |"},
		{"bad_stable_id", row.ID, "PROB-xyz"},
		{"duplicate_states", "| A | atomic | - | - |", "| A | atomic | - | - |\n| A | atomic | - | - |"},
		{"duplicate_section", "## Transitions", "## Transitions\n\n## Transitions"},
		{"extra_column", "| B | act |", "| B | act | extra |"},
		{"unterminated_row", "| B | act |", "| B | act"},
	}
	for _, tc := range variants {
		t.Run(tc.name, func(t *testing.T) {
			if bytes.Count(md, []byte(tc.old)) != 1 {
				t.Fatal("malformed fixture anchor is not unique")
			}
			_, err := Parse(bytes.Replace(md, []byte(tc.old), []byte(tc.new), 1), raw)
			requireOracleError(t, err, "oracle-parse")
		})
	}
	line := strings.Split(string(md), "\n")
	var rowLine string
	for _, l := range line {
		if strings.HasPrefix(l, "| T-PROB-") {
			rowLine = l
		}
	}
	for _, tc := range []struct{ name, doc string }{
		{"duplicate_id", strings.Replace(string(md), rowLine, rowLine+"\n"+rowLine, 1)},
		{"missing_row", strings.Replace(string(md), rowLine+"\n", "", 1)},
		{"malformed_trailing_row", string(md) + "| malformed |\n"},
	} {
		t.Run(tc.name, func(t *testing.T) { _, err := Parse([]byte(tc.doc), raw); requireOracleError(t, err, "oracle-parse") })
	}
	for _, tc := range []struct{ name, old, new string }{
		{"target_identity", "| B | act |", "| A | act |"}, {"actions_identity", "| B | act |", "| B | changed |"},
		{"guard_identity", "| ready |", "| changed |"}, {"stable_identity", row.ID, "PROB-ffffff"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(bytes.Replace(md, []byte(tc.old), []byte(tc.new), 1), raw)
			requireOracleError(t, err, "oracle-identity")
		})
	}
}

func TestFSMJSONDialectIsClosed(t *testing.T) {
	row := probeRow("A", "on:go", "ready", "B", "act")
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, []Row{row})
	parsedProbe(t, md, raw)
	mutate := func(change func(map[string]any)) []byte {
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		change(doc)
		out, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	allowed := mutate(func(d map[string]any) {
		d["_role"] = "operational"
		d["_comment"] = "plain metadata"
		d["_delays"] = map[string]any{"delay": "10 ms"}
		d["_counters"] = map[string]any{"retries": "per command"}
		d["context"] = map[string]any{"arbitraryData": map[string]any{"states": "data only"}, "number": 1, "nil": nil}
		d["states"].(map[string]any)["A"].(map[string]any)["_refusal"] = map[string]any{"on:other": "declared refusal"}
	})
	parsedProbe(t, md, allowed)
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"root_unknown", func(d map[string]any) { d["unknown"] = map[string]any{} }},
		{"unknown_metadata", func(d map[string]any) { d["_unknown"] = "data" }},
		{"state_nested", func(d map[string]any) {
			d["states"].(map[string]any)["A"].(map[string]any)["states"] = map[string]any{}
		}},
		{"state_unknown", func(d map[string]any) { d["states"].(map[string]any)["A"].(map[string]any)["unknown"] = true }},
		{"transition_reenter", func(d map[string]any) { probeTransition(d)["reenter"] = true }},
		{"transition_internal", func(d map[string]any) { probeTransition(d)["internal"] = true }},
		{"transition_unknown", func(d map[string]any) { probeTransition(d)["unknown"] = "x" }},
		{"action_object", func(d map[string]any) { probeTransition(d)["actions"] = map[string]any{"type": "act"} }},
		{"empty_target", func(d map[string]any) { probeTransition(d)["target"] = "" }},
		{"bad_context", func(d map[string]any) { d["context"] = []any{} }},
		{"bad_comment", func(d map[string]any) { d["_comment"] = map[string]any{} }},
		{"bad_delays", func(d map[string]any) { d["_delays"] = map[string]any{"delay": 1} }},
		{"final_outgoing", func(d map[string]any) { d["states"].(map[string]any)["A"].(map[string]any)["type"] = "final" }},
	} {
		t.Run(tc.name, func(t *testing.T) { _, err := Parse(md, mutate(tc.change)); requireOracleError(t, err, "oracle-parse") })
	}
	for _, bad := range [][]byte{append(append([]byte{}, raw...), []byte(" {}")...), bytes.Replace(raw, []byte(`"id":"probe"`), []byte(`"id":"probe","id":"probe"`), 1)} {
		_, err := Parse(md, bad)
		requireOracleError(t, err, "oracle-parse")
	}
	scalar := mutate(func(d map[string]any) {
		state := d["states"].(map[string]any)["A"].(map[string]any)
		on := state["on"].(map[string]any)
		tr := on["go"].([]any)[0].(map[string]any)
		tr["actions"] = "act"
		on["go"] = tr
	})
	parsedProbe(t, md, scalar)
}
func probeTransition(doc map[string]any) map[string]any {
	return doc["states"].(map[string]any)["A"].(map[string]any)["on"].(map[string]any)["go"].([]any)[0].(map[string]any)
}

func TestFSMInvokeAndPriorityIdentities(t *testing.T) {
	rows := []Row{probeRow("A", "after:wait", "", "B"), probeRow("A", "onDone:save", "ready", "B", "capture"), probeRow("A", "onDone:save", "", "(internal)", "deny"), probeRow("A", "onError:save", "", "B", "error")}
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, rows)
	parsedProbe(t, md, raw)
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	inv := doc["states"].(map[string]any)["A"].(map[string]any)["invoke"].(map[string]any)
	inv["input"] = map[string]any{"id": "context.id"}
	valid, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	parsedProbe(t, md, valid)
	inv["unknown"] = true
	bad, _ := json.Marshal(doc)
	_, err = Parse(md, bad)
	requireOracleError(t, err, "oracle-parse")
	delete(inv, "unknown")
	inv["input"] = []any{}
	bad, _ = json.Marshal(doc)
	_, err = Parse(md, bad)
	requireOracleError(t, err, "oracle-parse")
	inv["input"] = map[string]any{}
	alternatives := inv["onDone"].([]any)
	alternatives[0], alternatives[1] = alternatives[1], alternatives[0]
	bad, _ = json.Marshal(doc)
	_, err = Parse(md, bad)
	requireOracleError(t, err, "oracle-identity")
}

func TestFSMOneFireDoesNotFollowAlwaysOrEnterSource(t *testing.T) {
	rows := []Row{probeRow("A", "on:go", "", "B", "move"), probeRow("B", "always", "", "C", "finish")}
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic", Entry: []string{"sourceEntry"}}, {Name: "B", Kind: "atomic", Entry: []string{"enterB"}}, {Name: "C", Kind: "final", Entry: []string{"enterC"}}}, rows)
	suite := parsedProbe(t, md, raw)
	exp, err := suite.Bind(Witness{Name: "single", RowID: rows[0].ID, Source: "A", Trigger: "on:go"})
	if err != nil {
		t.Fatal(err)
	}
	if exp.Next != "B" || !reflect.DeepEqual(exp.Actions, []string{"move", "enterB"}) {
		t.Fatalf("must represent one Fire, got=%q/%q", exp.Next, exp.Actions)
	}
}

func TestFSMLoadRequiresCurrentBothSources(t *testing.T) {
	dir := t.TempDir()
	row := probeRow("A", "on:go", "", "B", "move")
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, []Row{row})
	if err := os.WriteFile(filepath.Join(dir, "Probe.oracle.md"), md, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir, "Probe")
	requireOracleError(t, err, "oracle-load")
	if err := os.WriteFile(filepath.Join(dir, "Probe.machine.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir, "Probe"); err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte(`"actions":["move"]`), []byte(`"actions":["changed"]`), 1)
	if bytes.Equal(changed, raw) {
		t.Fatal("load fixture mutation anchor missing")
	}
	if err := os.WriteFile(filepath.Join(dir, "Probe.machine.json"), changed, 0600); err != nil {
		t.Fatal(err)
	}
	_, err = Load(dir, "Probe")
	requireOracleError(t, err, "oracle-identity")
}

type candidateSuite struct {
	Machine, ID, Package, Test string
	Mandatory, Supplements     []string
}

func candidateDefinitions() []candidateSuite {
	return []candidateSuite{
		{"Deal", "deal", "domain", "TestDealTransitions", candidateDealLeaves, nil},
		{"Task", "task", "domain", "TestTaskTransitions", candidateTaskLeaves, []string{"TestTaskTerminalRejectsEverything/TASK-95f75f", "TestTaskTerminalRejectsEverything/TASK-841a9c", "TestTaskRollbackNonAlwaysPreservesContext/Open", "TestTaskRollbackNonAlwaysPreservesContext/InProgress", "TestTaskRollbackNonAlwaysPreservesContext/bogus"}},
		{"User", "user", "domain", "TestUserTransitions", nativeUserLeaves, nil},
		{"Session", "session", "session", "TestSessionTransitions", nativeSessionLeaves, nil},
		{"CommandExecution", "commandExecution", "cli", "TestCommandExecutionTransitions", candidateCommandLeaves, []string{"TestCommandExecutionTerminalExits/T-CMD-29", "TestCommandExecutionTerminalExits/T-CMD-30", "TestCommandExecutionTerminalExits/T-CMD-31", "TestCommandExecutionTerminalExits/T-CMD-32", "TestCommandExecutionTerminalExits/T-CMD-33"}},
	}
}

func candidateProofDir(t *testing.T) string {
	t.Helper()
	proof := t.TempDir()
	if *nativeProofParent != "" {
		if !filepath.IsAbs(*nativeProofParent) {
			t.Fatal("fsm-native-proof-dir must be an existing absolute directory")
		}
		var err error
		proof, err = os.MkdirTemp(*nativeProofParent, "crm-fsm-candidate-proof-")
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("candidate proof directory: %s; retained=%t", proof, *nativeProofParent != "")
	return proof
}

// candidateInventory reads actual child output. It is never supplied synthetic
// events or a fake process result by these tests. It returns an error so the
// same verifier can reject real copied-source output corruption variants.
func candidateInventory(result nativeResult, dir, pkg string) error {
	if result.exit != 0 {
		return fmt.Errorf("oracle-inventory: native candidate exit=%d", result.exit)
	}
	definitions := map[string]candidateSuite{}
	expectedRows := map[string]Identity{}
	expectedActions := map[string][]string{}
	expectedNext := map[string]string{}
	expectedNative := map[string]bool{}
	mandatory := map[string]bool{}
	for _, def := range candidateDefinitions() {
		if def.Package != pkg {
			continue
		}
		definitions[def.ID] = def
		expectedNative[def.Test] = true
		for _, leaf := range def.Mandatory {
			mandatory[def.Test+"/"+leaf] = true
		}
		for _, name := range def.Supplements {
			expectedNative[name] = true
			top, _, _ := strings.Cut(name, "/")
			expectedNative[top] = true
		}
		suite, err := Load(filepath.Join(dir, "design/machines"), def.Machine)
		if err != nil {
			return err
		}
		states := map[string]State{}
		for _, state := range suite.States {
			states[state.Name] = state
		}
		for _, row := range suite.Rows {
			key := def.ID + "/" + row.ID
			if _, ok := expectedRows[key]; ok {
				return fmt.Errorf("oracle-inventory: duplicate current row %s", key)
			}
			expectedRows[key] = row.Identity
			if row.Target == "(internal)" {
				expectedNext[key] = row.Source
				expectedActions[key] = append([]string{}, row.Actions...)
			} else {
				expectedNext[key] = row.Target
				actions := append([]string{}, states[row.Source].Exit...)
				actions = append(actions, row.Actions...)
				expectedActions[key] = append(actions, states[row.Target].Entry...)
			}
		}
	}
	if len(expectedRows) == 0 {
		return errors.New("oracle-inventory: empty committed rows")
	}
	registrations := map[string]Registration{}
	observations := map[string]Observation{}
	usedRows := map[string]bool{}
	runs := map[string]int{}
	ends := map[string]string{}
	started, completed := 0, 0
	decoder := json.NewDecoder(bytes.NewReader(result.raw))
	for {
		var ev nativeEvent
		err := decoder.Decode(&ev)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("oracle-inventory: invalid actual JSON: %w", err)
		}
		if ev.Package != "crm/internal/"+pkg {
			return fmt.Errorf("oracle-inventory: unexpected package %q", ev.Package)
		}
		switch ev.Action {
		case "start":
			started++
		case "run":
			runs[ev.Test]++
		case "pass", "fail", "skip":
			if ev.Action != "pass" {
				return fmt.Errorf("oracle-inventory: native %s %s", ev.Action, ev.Test)
			}
			if ev.Test == "" {
				completed++
			} else {
				if ends[ev.Test] != "" {
					return fmt.Errorf("oracle-inventory: duplicate terminal %s", ev.Test)
				}
				ends[ev.Test] = ev.Action
			}
		case "output":
			for _, bad := range []string{"panic:", "test timed out", "[build failed]", "no tests to run"} {
				if strings.Contains(ev.Output, bad) {
					return fmt.Errorf("oracle-inventory: invalid native outcome %s", bad)
				}
			}
			if at := strings.Index(ev.Output, "oracle-registered "); at >= 0 {
				var reg Registration
				if err := json.Unmarshal([]byte(strings.TrimSpace(ev.Output[at+len("oracle-registered "):])), &reg); err != nil {
					return fmt.Errorf("oracle-inventory: malformed registration: %w", err)
				}
				def, known := definitions[reg.Machine]
				key := reg.Machine + "/" + reg.Identity.ID
				identity, rowExists := expectedRows[key]
				if !known || !rowExists || reg.Identity != identity || !strings.HasPrefix(reg.Native, def.Test+"/") || ev.Test != def.Test {
					return fmt.Errorf("oracle-inventory: unknown registration %s", reg.Native)
				}
				if _, duplicate := registrations[reg.Native]; duplicate {
					return fmt.Errorf("oracle-inventory: duplicate registration %s", reg.Native)
				}
				if runs[reg.Native] != 0 {
					return fmt.Errorf("oracle-inventory: registration after witness began %s", reg.Native)
				}
				registrations[reg.Native] = reg
				expectedNative[reg.Native] = true
			}
			if at := strings.Index(ev.Output, "oracle-executed "); at >= 0 {
				var obs Observation
				if err := json.Unmarshal([]byte(strings.TrimSpace(ev.Output[at+len("oracle-executed "):])), &obs); err != nil {
					return fmt.Errorf("oracle-inventory: malformed execution: %w", err)
				}
				reg, known := registrations[obs.Registration.Native]
				key := reg.Machine + "/" + reg.Identity.ID
				if !known || reg != obs.Registration || ev.Test != reg.Native || runs[reg.Native] != 1 || ends[reg.Native] != "" {
					return fmt.Errorf("oracle-inventory: unregistered or misplaced execution %s", obs.Registration.Native)
				}
				if _, duplicate := observations[reg.Native]; duplicate {
					return fmt.Errorf("oracle-inventory: duplicate execution %s", reg.Native)
				}
				if obs.Next != expectedNext[key] || !reflect.DeepEqual(append([]string{}, obs.Actions...), append([]string{}, expectedActions[key]...)) {
					return fmt.Errorf("oracle-inventory: incorrect actual state/actions %s", reg.Native)
				}
				observations[reg.Native] = obs
				usedRows[key] = true
			}
		default:
			return fmt.Errorf("oracle-inventory: unexpected actual action %q", ev.Action)
		}
	}
	if started != 1 || completed != 1 {
		return errors.New("oracle-inventory: missing/duplicate package lifecycle")
	}
	if len(runs) != len(expectedNative) || len(ends) != len(expectedNative) {
		return errors.New("oracle-inventory: actual native identity set differs from registered witnesses and supplements")
	}
	for name := range expectedNative {
		if runs[name] != 1 || ends[name] != "pass" {
			return fmt.Errorf("oracle-inventory: missing native terminal %s", name)
		}
	}
	for name := range mandatory {
		if _, ok := registrations[name]; !ok {
			return fmt.Errorf("oracle-inventory: missing preserved guard witness %s", name)
		}
	}
	for name := range registrations {
		if _, ok := observations[name]; !ok {
			return fmt.Errorf("oracle-inventory: missing successful execution %s", name)
		}
	}
	for key := range expectedRows {
		if !usedRows[key] {
			return fmt.Errorf("oracle-inventory: unused committed row %s", key)
		}
	}
	if len(observations) != len(registrations) || len(usedRows) != len(expectedRows) {
		return errors.New("oracle-inventory: bidirectional row/witness mismatch")
	}
	return nil
}

func candidateSelection(pkg string) string {
	switch pkg {
	case "domain":
		return "(TestDealTransitions|TestTaskTransitions|TestUserTransitions|TestTaskTerminalRejectsEverything|TestTaskRollbackNonAlwaysPreservesContext)"
	case "session":
		return "TestSessionTransitions"
	case "cli":
		return "(TestCommandExecutionTransitions|TestCommandExecutionTerminalExits)"
	}
	return "invalid-candidate-package"
}

func TestFSMCandidateExecutedInventory(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
	defer cancel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Clean(filepath.Join(cwd, "../../.."))
	original := nativeManifest(t, source)
	proof := candidateProofDir(t)
	for _, pkg := range []string{"domain", "session", "cli"} {
		t.Run(pkg, func(t *testing.T) {
			fixture := t.TempDir()
			nativeCopy(t, source, fixture)
			before := nativeManifest(t, fixture)
			if !reflect.DeepEqual(before, original) {
				t.Fatal("candidate copy differs from source")
			}
			nativeWriteJSON(t, filepath.Join(proof, pkg+"-manifest.json"), before)
			result := nativeRun(t, ctx, fixture, proof, pkg, pkg, candidateSelection(pkg))
			if err := candidateInventory(result, fixture, pkg); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, nativeManifest(t, fixture)) {
				t.Fatal("candidate native test modified source")
			}
			t.Logf("candidate bidirectional committed-row / successful-witness / native-leaf inventory passed: %s", pkg)
		})
	}
	if !reflect.DeepEqual(original, nativeManifest(t, source)) {
		t.Fatal("shared source changed")
	}
}

type laterNativeCase struct {
	name, pkg, suite, target, diagnostic, kind string
	leaves                                     []string
	mutations                                  []nativeMutation
}

func laterApply(t *testing.T, fixture, proof string, mutations []nativeMutation) map[string]string {
	t.Helper()
	before := nativeManifest(t, fixture)
	expected := map[string]string{}
	for p, h := range before {
		expected[p] = h
	}
	nativeWriteJSON(t, filepath.Join(proof, "control-manifest.json"), before)
	var evidence []map[string]string
	for _, mutation := range mutations {
		p := filepath.Join(fixture, filepath.FromSlash(mutation.path))
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Count(raw, []byte(mutation.before)) != 1 {
			t.Fatalf("later mutation anchor must match once: %s", mutation.path)
		}
		changed := bytes.Replace(raw, []byte(mutation.before), []byte(mutation.after), 1)
		if bytes.Equal(changed, raw) {
			t.Fatal("later mutation has no effect")
		}
		if err := os.WriteFile(p, changed, 0600); err != nil {
			t.Fatal(err)
		}
		expected[mutation.path] = fmt.Sprintf("%x", sha256.Sum256(changed))
		evidence = append(evidence, map[string]string{"path": mutation.path, "before": mutation.before, "after": mutation.after})
	}
	after := nativeManifest(t, fixture)
	if !reflect.DeepEqual(after, expected) {
		t.Fatal("later mutation changed unrelated bytes/files")
	}
	nativeWriteJSON(t, filepath.Join(proof, "mutations.json"), evidence)
	nativeWriteJSON(t, filepath.Join(proof, "mutant-manifest.json"), after)
	return after
}

func laterStructuralOutcome(result nativeResult, pkg, suite, diagnostic string) error {
	if result.exit != 1 {
		return fmt.Errorf("structural variant exit=%d want1", result.exit)
	}
	decoder := json.NewDecoder(bytes.NewReader(result.raw))
	runs, ends, starts, packages := 0, 0, 0, 0
	output := ""
	for {
		var ev nativeEvent
		err := decoder.Decode(&ev)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if ev.Package != "crm/internal/"+pkg {
			return errors.New("structural variant wrong package")
		}
		if ev.Test != "" && ev.Test != suite {
			return fmt.Errorf("structural failure must precede Fire leaves, got %s", ev.Test)
		}
		switch ev.Action {
		case "start":
			starts++
		case "run":
			if ev.Test != suite {
				return errors.New("structural run missing suite")
			}
			runs++
		case "fail":
			if ev.Test == suite {
				ends++
			} else {
				packages++
			}
		case "output":
			if ev.Test == suite {
				output += ev.Output
			}
			for _, bad := range []string{"panic:", "test timed out", "[build failed]", "oracle-scaffold"} {
				if strings.Contains(ev.Output, bad) {
					return fmt.Errorf("nonqualifying structural outcome %s", bad)
				}
			}
		default:
			return fmt.Errorf("unexpected structural native event %s", ev.Action)
		}
	}
	if starts != 1 || runs != 1 || ends != 1 || packages != 1 || !strings.Contains(output, diagnostic) {
		return fmt.Errorf("wrong structural failure lifecycle/diagnostic: %q", output)
	}
	return nil
}

func TestFSMRemainingNativeSensitivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
	defer cancel()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Clean(filepath.Join(cwd, "../../.."))
	original := nativeManifest(t, source)
	proof := candidateProofDir(t)
	for _, tc := range laterNativeCases() {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			nativeCopy(t, source, fixture)
			if !reflect.DeepEqual(original, nativeManifest(t, fixture)) {
				t.Fatal("later control copy differs from source")
			}
			caseProof := filepath.Join(proof, tc.name)
			if err := os.Mkdir(caseProof, 0700); err != nil {
				t.Fatal(err)
			}
			control := nativeRun(t, ctx, fixture, caseProof, "control", tc.pkg, tc.suite)
			nativeCheckRun(t, control, tc.suite, tc.leaves, "", "")
			if tc.kind == "inventory" {
				if err := candidateInventory(control, fixture, tc.pkg); err != nil {
					t.Fatalf("unchanged candidate inventory control failed: %v", err)
				}
			}
			if !reflect.DeepEqual(original, nativeManifest(t, fixture)) {
				t.Fatal("later control modified fixture source")
			}
			after := laterApply(t, fixture, caseProof, tc.mutations)
			mutant := nativeRun(t, ctx, fixture, caseProof, "mutant", tc.pkg, tc.suite)
			if !reflect.DeepEqual(after, nativeManifest(t, fixture)) {
				t.Fatal("later mutant run modified fixture source")
			}
			switch tc.kind {
			case "semantic":
				nativeCheckRun(t, mutant, tc.suite, tc.leaves, tc.target, tc.diagnostic)
			case "structural":
				if err := laterStructuralOutcome(mutant, tc.pkg, tc.suite, tc.diagnostic); err != nil {
					t.Fatal(err)
				}
			case "inventory":
				// Native machine assertions must still pass: only actual emitted
				// inventory data is corrupted by these copied-test variants.
				nativeCheckRun(t, mutant, tc.suite, tc.leaves, "", "")
				err := candidateInventory(mutant, fixture, tc.pkg)
				requireOracleError(t, err, tc.diagnostic)
			default:
				t.Fatalf("unknown later sensitivity kind %s", tc.kind)
			}
		})
	}
	if !reflect.DeepEqual(original, nativeManifest(t, source)) {
		t.Fatal("shared source changed during later sensitivity")
	}
}

func TestFSMParserPreservesRepeatedExpectedActionsAndWhitespace(t *testing.T) {
	row := probeRow("A", "on:go", "", "B", "same", "same")
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, []Row{row})
	for _, doc := range [][]byte{md, bytes.ReplaceAll(md, []byte("\n"), []byte("\r\n")), bytes.TrimSuffix(md, []byte("\n"))} {
		suite := parsedProbe(t, doc, raw)
		exp, err := suite.Bind(Witness{Name: "duplicates", RowID: row.ID, Source: "A", Trigger: "on:go"})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(exp.Actions, []string{"same", "same"}) {
			t.Fatal("expected action multiplicity must be preserved")
		}
		requireOracleError(t, exp.Check("B", []string{"same"}), "oracle-actions probe "+row.ID)
	}
}

func TestFSMNestedDuplicateKeysAndInvalidMetadata(t *testing.T) {
	row := probeRow("A", "on:go", "ready", "B", "act")
	md, raw := probeFixture(t, []State{{Name: "A", Kind: "atomic"}, {Name: "B", Kind: "final"}}, []Row{row})
	parsedProbe(t, md, raw)
	duplicate := bytes.Replace(raw, []byte(`"guard":"ready"`), []byte(`"guard":"ready","guard":"ready"`), 1)
	if bytes.Equal(raw, duplicate) {
		t.Fatal("nested duplicate anchor missing")
	}
	_, err := Parse(md, duplicate)
	requireOracleError(t, err, "oracle-parse")
	for _, name := range []string{"counters", "role", "refusal", "parallel", "initial", "mixed_actions", "empty_action", "missing_id"} {
		t.Run(name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			state := doc["states"].(map[string]any)["A"].(map[string]any)
			switch name {
			case "counters":
				doc["_counters"] = []any{}
			case "role":
				doc["_role"] = false
			case "refusal":
				state["_refusal"] = map[string]any{"on:go": []any{}}
			case "parallel":
				state["type"] = "parallel"
			case "initial":
				doc["initial"] = "Missing"
			case "mixed_actions":
				probeTransition(doc)["actions"] = []any{"act", 1}
			case "empty_action":
				probeTransition(doc)["actions"] = []any{""}
			case "missing_id":
				delete(doc, "id")
			}
			bad, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Parse(md, bad)
			requireOracleError(t, err, "oracle-parse")
		})
	}
}

func laterNativeCases() []laterNativeCase {
	return []laterNativeCase{
		{name: "missing_current_row", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-parse", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |\n", ""},
		}},
		{name: "duplicate_current_row", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-parse", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |\n| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |"},
		}},
		{name: "malformed_current_row", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-parse", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |", "| T-USER-01 | USER-e20d04 | Active on:disable | guardAdminAuthority | persisting | setPendingDisable |"},
		}},
		{name: "new_unmapped_current_row", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-unused-row user USER-bfc3b1", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-20 | USER-453743 | rolledBack | always | - | Disabled | recordRoutingError |", "| T-USER-20 | USER-453743 | rolledBack | always | - | Disabled | recordRoutingError |\n| T-USER-21 | USER-bfc3b1 | Active | on:oracleUnmapped | - | (internal) | recordAlreadyActive |"},
			{"design/machines/User.oracle.md", "Total transitions (test cases): 20", "Total transitions (test cases): 21"},
			{"design/machines/User.machine.json", "\"enable\": { \"actions\": \"recordAlreadyActive\" }", "\"enable\": { \"actions\": \"recordAlreadyActive\" },\n        \"oracleUnmapped\": { \"actions\": \"recordAlreadyActive\" }"},
		}},
		{name: "unused_current_row", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-unused-row user USER-453743", mutations: []nativeMutation{
			{"impl/internal/domain/user_test.go", "\t\t{\"USER-453743 / fail-closed rollback routing\", userPrior(domain.UserState(\"bogus\")), domain.UserEvent{Kind: domain.UEvAlways}, \"USER-453743\"},\n", ""},
		}},
		{name: "duplicate_actual_witness", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-witness user USER-e20d04", mutations: []nativeMutation{
			{"impl/internal/domain/user_test.go", "\t\t{\"T-USER-01_USER-e20d04\", newUserAgg(domain.USActive, uAdmin), disable, \"USER-e20d04\"},", "\t\t{\"T-USER-01_USER-e20d04\", newUserAgg(domain.USActive, uAdmin), disable, \"USER-e20d04\"},\n\t\t{\"T-USER-01_USER-e20d04\", newUserAgg(domain.USActive, uAdmin), disable, \"USER-e20d04\"},"},
		}},
		{name: "coherent_trigger_changes_actual_witness", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-witness user USER-e20d04", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |", "| T-USER-01 | USER-4c2fb2 | Active | on:otherDisable | guardAdminAuthority | persisting | setPendingDisable |"},
			{"design/machines/User.oracle.md", "| T-USER-02 | USER-2b2218 | Active | on:disable | - | (internal) | recordAuthorityDenied |", "| T-USER-02 | USER-db965d | Active | on:otherDisable | - | (internal) | recordAuthorityDenied |"},
			{"design/machines/User.machine.json", "\"disable\": [", "\"otherDisable\": ["},
		}},
		{name: "coherent_guard_changes_actual_witness", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-witness user USER-e20d04", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |", "| T-USER-01 | USER-9080f8 | Active | on:disable | newAuthority | persisting | setPendingDisable |"},
			{"design/machines/User.machine.json", "{ \"target\": \"persisting\", \"guard\": \"guardAdminAuthority\", \"actions\": \"setPendingDisable\" }", "{ \"target\": \"persisting\", \"guard\": \"newAuthority\", \"actions\": \"setPendingDisable\" }"},
		}},
		{name: "coherent_source_changes_actual_witness", kind: "structural", pkg: "domain", suite: "TestUserTransitions", leaves: nativeUserLeaves, target: "", diagnostic: "oracle-witness user USER-e20d04", mutations: []nativeMutation{
			{"design/machines/User.oracle.md", "| T-USER-01 | USER-e20d04 | Active | on:disable | guardAdminAuthority | persisting | setPendingDisable |", "| T-USER-01 | USER-43ab94 | Disabled | on:disable | guardAdminAuthority | persisting | setPendingDisable |"},
			{"design/machines/User.machine.json", "          { \"target\": \"persisting\", \"guard\": \"guardAdminAuthority\", \"actions\": \"setPendingDisable\" },\n", ""},
			{"design/machines/User.machine.json", "\"disable\": { \"actions\": \"recordAlreadyDisabled\" }", "\"disable\": [\n          { \"target\": \"persisting\", \"guard\": \"guardAdminAuthority\", \"actions\": \"setPendingDisable\" },\n          { \"actions\": \"recordAlreadyDisabled\" }\n        ]"},
		}},
		{name: "cli_nonempty_entry_extra_effect", kind: "semantic", pkg: "cli", suite: "TestCommandExecutionTransitions", leaves: candidateCommandLeaves, target: "T-CMD-01_COMM-44671c", diagnostic: "oracle-actions commandExecution COMM-44671c: got=[\"captureArgs\" \"setPhaseOpen\" \"unexpectedOracleEffect\"] want=[\"captureArgs\" \"setPhaseOpen\"]", mutations: []nativeMutation{
			{"impl/internal/cli/command.go", "return c.enter(COpening, \"captureArgs\")", "got := c.enter(COpening, \"captureArgs\")\n\t\tgot.Actions = append(got.Actions, \"unexpectedOracleEffect\")\n\t\treturn got"},
		}},
		{name: "task_entry_name_only_forgery", kind: "semantic", pkg: "domain", suite: "TestTaskTransitions", leaves: candidateTaskLeaves, target: "TASK-754183_/_fail-closed_rollback_routing", diagnostic: "oracle-entry-context TASK-754183: got=\"task: unroutable pending status\" want=\"\"", mutations: []nativeMutation{
			{"impl/internal/domain/task.go", "\tdefault:\n\t\tt.recordRoutingError()\n\t\tt.State = TSCancelled\n\t\tt.recordTaskClosed()\n\t\treturn effect(\"recordRoutingError\", \"recordTaskClosed\")", "\tdefault:\n\t\tt.recordRoutingError()\n\t\tt.State = TSCancelled\n\t\treturn effect(\"recordRoutingError\", \"recordTaskClosed\")"},
		}},
		{name: "task_missing_entry_call_and_name", kind: "semantic", pkg: "domain", suite: "TestTaskTransitions", leaves: candidateTaskLeaves, target: "TASK-754183_/_fail-closed_rollback_routing", diagnostic: "oracle-actions task TASK-754183: got=[\"recordRoutingError\"] want=[\"recordRoutingError\" \"recordTaskClosed\"]", mutations: []nativeMutation{
			{"impl/internal/domain/task.go", "\tdefault:\n\t\tt.recordRoutingError()\n\t\tt.State = TSCancelled\n\t\tt.recordTaskClosed()\n\t\treturn effect(\"recordRoutingError\", \"recordTaskClosed\")", "\tdefault:\n\t\tt.recordRoutingError()\n\t\tt.State = TSCancelled\n\t\treturn effect(\"recordRoutingError\")"},
		}},
		{name: "duplicate_registration", kind: "inventory", pkg: "session", suite: "TestSessionTransitions", leaves: nativeSessionLeaves, target: "", diagnostic: "oracle-inventory: duplicate registration", mutations: []nativeMutation{
			{"impl/internal/session/machine_test.go", "t.Logf(\"oracle-registered %s\", raw)", "t.Logf(\"oracle-registered %s\", raw)\n\t\tt.Logf(\"oracle-registered %s\", raw)"},
		}},
		{name: "missing_registration", kind: "inventory", pkg: "session", suite: "TestSessionTransitions", leaves: nativeSessionLeaves, target: "", diagnostic: "oracle-inventory: unregistered or misplaced execution", mutations: []nativeMutation{
			{"impl/internal/session/machine_test.go", "t.Logf(\"oracle-registered %s\", raw)", "t.Logf(\"registration-omitted %s\", raw)"},
		}},
		{name: "unknown_registration", kind: "inventory", pkg: "session", suite: "TestSessionTransitions", leaves: nativeSessionLeaves, target: "", diagnostic: "oracle-inventory: unknown registration", mutations: []nativeMutation{
			{"impl/internal/session/machine_test.go", "t.Logf(\"oracle-registered %s\", raw)", "t.Logf(\"oracle-registered %s\", strings.Replace(string(raw), \"TestSessionTransitions/\", \"UnknownNative/\", 1))"},
		}},
		{name: "missing_execution", kind: "inventory", pkg: "session", suite: "TestSessionTransitions", leaves: nativeSessionLeaves, target: "", diagnostic: "oracle-inventory: missing successful execution", mutations: []nativeMutation{
			{"impl/internal/session/machine_test.go", "t.Logf(\"oracle-executed %s\", raw)", "t.Logf(\"execution-omitted %s\", raw)"},
		}},
		{name: "duplicate_execution", kind: "inventory", pkg: "session", suite: "TestSessionTransitions", leaves: nativeSessionLeaves, target: "", diagnostic: "oracle-inventory: duplicate execution", mutations: []nativeMutation{
			{"impl/internal/session/machine_test.go", "t.Logf(\"oracle-executed %s\", raw)", "t.Logf(\"oracle-executed %s\", raw)\n\t\t\tt.Logf(\"oracle-executed %s\", raw)"},
		}},
	}
}
