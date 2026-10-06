package hook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// B8: a crashed or interrupted hook-state write (a full disk, a killed hook)
// must never block every later command. An empty or provably stale temp is
// removed by itself; any other temp is moved aside as crash evidence, the
// project is marked dirty for design and impl, that one command is refused
// with a one-line recovery note, and the next command proceeds.

func crashShellPre(t *testing.T, root, sid, operation string) string {
	t.Helper()
	return runEvent(t, root, Input{
		SessionID: sid, HookEventName: "PreToolUse", ToolName: "Bash",
		ToolInput: toolInput{Command: "ls"}, ToolUseID: operation,
	})
}

func stateTempPath(root, sid, suffix string) string {
	p := statePath(root, sid)
	return filepath.Join(filepath.Dir(p), "."+filepath.Base(p)+".tmp-"+suffix)
}

func routeTempPath(root, sid, suffix string) string {
	p := routeStatePath(root, sid)
	return filepath.Join(filepath.Dir(p), "."+filepath.Base(p)+".tmp-"+suffix)
}

func crashEvidence(t *testing.T, root string) []string {
	t.Helper()
	p := statePath(root, "")
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(p), "."+filepath.Base(p)+".crashed-*"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func writeCrashTemp(t *testing.T, path string, body []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, body, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func assertNoCrashTemps(t *testing.T, root string) {
	t.Helper()
	if temps, err := hookStateTemps(statePath(root, "")); err != nil || len(temps) != 0 {
		t.Fatalf("hook state temps left behind: %v %v", temps, err)
	}
	if temps, err := routeStateTemps(root); err != nil || len(temps) != 0 {
		t.Fatalf("hook route temps left behind: %v %v", temps, err)
	}
}

func assertAllowed(t *testing.T, out, why string) {
	t.Helper()
	if strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("%s: the command was refused: %s", why, out)
	}
}

// assertRefusedOnce checks the one refusal, the preserved evidence, the dirty
// ledger, and that the next command proceeds.
func assertRefusedOnce(t *testing.T, root, sid, out, kind string) {
	t.Helper()
	var got preOut
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("pre output is not JSON: %v (%q)", err, out)
	}
	reason := got.HookSpecificOutput.PermissionDecisionReason
	if got.HookSpecificOutput.PermissionDecision != "deny" {
		t.Fatalf("an unprovable crash temp must refuse this one command: %s", out)
	}
	if strings.Contains(reason, "\n") {
		t.Fatalf("the refusal must be a single line: %q", reason)
	}
	evidence := crashEvidence(t, root)
	if len(evidence) != 1 {
		t.Fatalf("want exactly one preserved crash file, got %v", evidence)
	}
	if !strings.Contains(reason, "incomplete hook "+kind+" transaction") || !strings.Contains(reason, "rm '"+evidence[0]+"'") {
		t.Fatalf("the refusal must name the preserved file and its recovery command: %q", reason)
	}
	info, err := os.Lstat(evidence[0])
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		t.Fatalf("preserved crash evidence must be a private regular file: %v %v", info, err)
	}
	assertNoCrashTemps(t, root)
	design, impl, err := readStateErr(root, sid)
	if err != nil || !design || !impl {
		t.Fatalf("crash recovery must mark the project dirty for design and impl: design=%v impl=%v err=%v", design, impl, err)
	}
	assertAllowed(t, crashShellPre(t, root, sid, "after-crash-"+kind), "the next command after a crash refusal")
	if after := crashEvidence(t, root); len(after) != 1 || after[0] != evidence[0] {
		t.Fatalf("crash evidence must survive the next command: %v", after)
	}
}

func TestCrashStateTempEmptyOrStaleIsRecoveredAutomatically(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func(t *testing.T, root, sid string) []byte
	}{
		{"empty", func(*testing.T, string, string) []byte { return nil }},
		{"lower revision", func(*testing.T, string, string) []byte { return []byte("revision 1\ndesign\n") }},
		{"identical to the live ledger", func(t *testing.T, root, sid string) []byte {
			raw, err := os.ReadFile(statePath(root, sid))
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := managedRoot(t)
			sid := "crash-stale-state"
			t.Cleanup(func() { clearState(root, sid) })
			for i := 0; i < 2; i++ {
				if err := appendState(root, sid, "design"); err != nil {
					t.Fatal(err)
				}
			}
			writeCrashTemp(t, stateTempPath(root, sid, "stale"), tc.body(t, root, sid), 0o600)
			assertAllowed(t, crashShellPre(t, root, sid, "stale-state"), "an empty or stale state temp")
			assertNoCrashTemps(t, root)
			if evidence := crashEvidence(t, root); len(evidence) != 0 {
				t.Fatalf("a provably stale temp is removed, not preserved: %v", evidence)
			}
		})
	}
}

func TestCrashStateTempNewerOrGarbageIsPreservedAndRefusedOnce(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ledger bool
		body   []byte
		mode   os.FileMode
	}{
		{"newer revision", true, []byte("revision 99\ndesign\n"), 0o600},
		{"garbage with no live ledger", false, []byte("\x00not a ledger"), 0o644},
		{"unparseable next to a live ledger", true, []byte("revision x\n"), 0o600},
		{"oversize", true, bytes.Repeat([]byte("x"), int(hookStateMaxBytes)+1), 0o600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := managedRoot(t)
			sid := "crash-newer-state"
			t.Cleanup(func() { clearState(root, sid) })
			if tc.ledger {
				if err := appendState(root, sid, "design"); err != nil {
					t.Fatal(err)
				}
			} else if err := ensureStateDir(); err != nil {
				t.Fatal(err)
			}
			writeCrashTemp(t, stateTempPath(root, sid, "crash"), tc.body, tc.mode)
			assertRefusedOnce(t, root, sid, crashShellPre(t, root, sid, "newer-state"), "state")
		})
	}
}

func TestCrashRouteTempEmptyOrStaleIsRecoveredAutomatically(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func(t *testing.T, root, sid string) []byte
	}{
		{"empty", func(*testing.T, string, string) []byte { return nil }},
		{"identical to the live route snapshot", func(t *testing.T, root, sid string) []byte {
			raw, err := os.ReadFile(routeStatePath(root, sid))
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}},
		{"route the live ledger does not reference", func(t *testing.T, _, _ string) []byte {
			raw, err := routeSnapshotBody(Config{Design: "elsewhere"})
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := managedRoot(t)
			sid := "crash-stale-route"
			t.Cleanup(func() { clearState(root, sid) })
			assertAllowed(t, crashShellPre(t, root, sid, "arm-route"), "arming")
			writeCrashTemp(t, routeTempPath(root, sid, "stale"), tc.body(t, root, sid), 0o600)
			assertAllowed(t, crashShellPre(t, root, sid, "stale-route"), "an empty or stale route temp")
			assertNoCrashTemps(t, root)
			if evidence := crashEvidence(t, root); len(evidence) != 0 {
				t.Fatalf("a provably stale route temp is removed, not preserved: %v", evidence)
			}
		})
	}
}

func TestCrashRouteTempReferencedOrGarbageIsPreservedAndRefusedOnce(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func(t *testing.T, root string) []byte
	}{
		{"route the live ledger references", func(t *testing.T, root string) []byte {
			cfg, ok, _ := Load(root)
			if !ok {
				t.Fatal("managed root did not load")
			}
			raw, err := routeSnapshotBody(cfg)
			if err != nil {
				t.Fatal(err)
			}
			return raw
		}},
		{"garbage", func(*testing.T, string) []byte { return []byte("{not json") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := managedRoot(t)
			sid := "crash-newer-route"
			t.Cleanup(func() { clearState(root, sid) })
			assertAllowed(t, crashShellPre(t, root, sid, "arm-route"), "arming")
			// a second session whose route publish crashed: no live snapshot
			writeCrashTemp(t, routeTempPath(root, "crashed-session", "crash"), tc.body(t, root), 0o600)
			assertRefusedOnce(t, root, sid, crashShellPre(t, root, sid, "newer-route"), "route")
		})
	}
}

// A PostToolUse cannot refuse a tool that already ran. It still recovers,
// completes its own operation (so the in-flight token never strands the
// stop), and reports the preserved evidence once.
func TestCrashTempAtPostCompletesTrackingAndReports(t *testing.T) {
	root := managedRoot(t)
	sid := "crash-post"
	t.Cleanup(func() { clearState(root, sid) })
	assertAllowed(t, crashShellPre(t, root, sid, "post-operation"), "arming")
	writeCrashTemp(t, stateTempPath(root, sid, "crash"), []byte("revision 99\ndesign\n"), 0o600)
	raw, err := json.Marshal(Input{SessionID: sid, HookEventName: "PostToolUse", ToolName: "Bash",
		ToolInput: toolInput{Command: "ls"}, ToolUseID: "post-operation"})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	runErr := Run(bytes.NewReader(raw), &out, root)
	evidence := crashEvidence(t, root)
	if len(evidence) != 1 || runErr == nil || !strings.Contains(runErr.Error(), "rm '"+evidence[0]+"'") {
		t.Fatalf("post must report the preserved evidence once: evidence=%v err=%v", evidence, runErr)
	}
	record, err := readStateRecord(root, sid)
	if err != nil || !record.design || !record.impl || len(record.pending) != 0 {
		t.Fatalf("post must complete its operation on a dirty ledger: %+v err=%v", record, err)
	}
	stop := runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})
	if strings.Contains(stop, "in-flight") || strings.Contains(stop, "incomplete hook") {
		t.Fatalf("the stop after a recovered crash must run the checks, not strand: %s", stop)
	}
}

// Stop stays conservative: a temp present at stop time is never read as an
// untouched project. Its block names the automatic recovery.
func TestCrashTempAtStopStillBlocksAndNamesRecovery(t *testing.T) {
	root := managedRoot(t)
	sid := "crash-stop"
	t.Cleanup(func() {
		for _, temp := range []string{stateTempPath(root, sid, "crash")} {
			_ = os.Remove(temp)
		}
		clearState(root, sid)
	})
	if err := ensureStateDir(); err != nil {
		t.Fatal(err)
	}
	writeCrashTemp(t, stateTempPath(root, sid, "crash"), nil, 0o600)
	var got stopOut
	if err := json.Unmarshal([]byte(runEvent(t, root, Input{SessionID: sid, HookEventName: "Stop"})), &got); err != nil {
		t.Fatal(err)
	}
	if got.Decision != "block" || !strings.Contains(got.Reason, "untouched") || !strings.Contains(got.Reason, "next governed shell or file command recovers it") {
		t.Fatalf("a crash temp at stop must block and name the recovery: %+v", got)
	}
}

// The preserved evidence is bounded per project, and a dead project's evidence
// is reclaimed with its generation instead of freezing it.
func TestCrashEvidenceIsBoundedAndReclaimable(t *testing.T) {
	dir := isolateHookRetention(t)
	root := managedRoot(t)
	sid := "crash-bound"
	t.Cleanup(func() { clearState(root, sid) })
	if err := appendState(root, sid, "design"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < hookCrashEvidenceRetention+3; i++ {
		writeCrashTemp(t, stateTempPath(root, sid, fmt.Sprintf("crash%d", i)), []byte("revision 999\ndesign\n"), 0o600)
		out := crashShellPre(t, root, sid, fmt.Sprintf("bound-%d", i))
		if !strings.Contains(out, `"permissionDecision":"deny"`) {
			t.Fatalf("crash %d was not refused: %s", i, out)
		}
	}
	if evidence := crashEvidence(t, root); len(evidence) != hookCrashEvidenceRetention {
		t.Fatalf("crash evidence must be bounded at %d per project, got %d", hookCrashEvidenceRetention, len(evidence))
	}

	dead := filepath.Join(t.TempDir(), "root-that-no-longer-exists")
	base := fmt.Sprintf("%064x.state", 7)
	ledger := filepath.Join(dir, base)
	if err := os.WriteFile(ledger, []byte("revision 1\n"+hookStateRootLine(dead)+"\ndesign\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	evidence := filepath.Join(dir, "."+base+".crashed-0123456789abcdef")
	if err := os.WriteFile(evidence, []byte("revision 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if owner, kind := classifyHookStateEntry(filepath.Base(evidence)); owner != base || kind != hookStateEntryCrashed {
		t.Fatalf("preserved evidence must belong to its generation: owner=%q kind=%v", owner, kind)
	}
	if _, err := compactHookStateDir(dir, 0); err != nil {
		t.Fatalf("compaction: %v", err)
	}
	for _, p := range []string{ledger, evidence} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a dead generation and its crash evidence must be reclaimed together: %s %v", p, err)
		}
	}
}
