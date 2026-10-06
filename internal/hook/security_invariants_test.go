package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Security invariants of the stranded-token and route-change fixes. Each test
// names the invariant it holds; together they show that no new path
// discharges an obligation without its gates, lets a session's own unfinished
// call stop blocking it, clears an obligation by release, or accepts a
// replaced store.

func armUnder(t *testing.T, root, config, session, toolUseID string) {
	t.Helper()
	writeFile(t, filepath.Join(root, ConfigName), config)
	armShell(t, root, session, toolUseID)
	completeShell(t, root, session, toolUseID)
}

// Invariant: a changed configuration that selects no stop-time gate cannot
// clear an obligation armed under the earlier one.
func TestInvariantRouteChangeToNoGatesNeverDischarges(t *testing.T) {
	root := greenFieldRoot(t)
	armUnder(t, root, `{"design":"design"}`, "s", "toolu_1")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","gates":"gv"}`)
	for i := 0; i < 2; i++ {
		out := stopOutput(t, root, "s")
		if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "retained") {
			t.Fatalf("empty selection after a route change must retain, not clear or wedge: %s", out)
		}
		requireArmed(t, root, 0, "empty selection")
	}
}

// Invariant: a configuration that narrows what the gates check (here, a
// different design tree) runs the gates but never discharges; only an
// audited operator acceptance lets the next Stop clear it.
func TestInvariantNarrowedConfigurationNeedsAuditedAcceptance(t *testing.T) {
	root := greenFieldRoot(t)
	copyTree(t, crmDesign, filepath.Join(root, "decoy"))
	armUnder(t, root, `{"design":"design","strict":true}`, "s", "toolu_1")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"decoy"}`)
	out := stopOutput(t, root, "s")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "narrows what the gates check") || !strings.Contains(out, "--routes") ||
		!strings.Contains(out, `design directory \"design\" is now \"decoy\"`) || !strings.Contains(out, "strict mode was switched off") {
		t.Fatalf("narrowing was not detected and named: %s", out)
	}
	requireArmed(t, root, 0, "narrowed configuration")
	var report bytes.Buffer
	if _, err := StateReport(&report, false); err != nil || !strings.Contains(report.String(), "--routes") {
		t.Fatalf("doctor must name the acceptance command: %v %s", err, report.String())
	}
	var released bytes.Buffer
	if err := ReleaseStateWith(&released, root, ReleaseOptions{Routes: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(released.String(), "forgot") {
		t.Fatalf("release --routes output: %s", released.String())
	}
	requireArmed(t, root, 0, "release --routes keeps the obligation armed")
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	journal, err := os.ReadFile(marker + ".handoffs")
	if err != nil {
		t.Fatal(err)
	}
	var event map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(journal), &event); err != nil {
		t.Fatal(err)
	}
	if routes, _ := event["routes"].([]any); event["event"] != "release" || len(routes) == 0 {
		t.Fatalf("route acceptance was not journaled: %s", journal)
	}
	if out := stopOutput(t, root, "s"); out != "" {
		t.Fatalf("after an audited acceptance the gates run and clear: %s", out)
	}
	if state, err := readStateRecord(root, "s"); err != nil || state.design || state.impl {
		t.Fatalf("not discharged after acceptance: %+v %v", state, err)
	}
}

// Invariant: a widening change (strict switched on) is not narrowing; the
// gates run under it and a green Stop discharges.
func TestInvariantWideningConfigurationDischargesAfterGates(t *testing.T) {
	root := greenFieldRoot(t)
	armUnder(t, root, `{"design":"design"}`, "s", "toolu_1")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","strict":true}`)
	out := stopOutput(t, root, "s")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "re-evaluated") {
		t.Fatalf("widened route: %s", out)
	}
	if state, err := readStateRecord(root, "s"); err != nil || state.design {
		t.Fatalf("widened route was not discharged after green gates: %+v %v", state, err)
	}
}

// Invariant: a Stop forged with another session id (an agent piping its own
// event into 'machinery hook') cannot discharge while the agent's own call is
// still recorded in flight.
func TestInvariantForgedForeignStopCannotDischargeInFlightCall(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "agent", "toolu_running_forger")
	out := stopOutput(t, root, "forged-session")
	if strings.Contains(out, `"decision":"block"`) {
		t.Fatalf("a foreign token must not block: %s", out)
	}
	requireArmed(t, root, 1, "forged foreign Stop")
	out = stopOutput(t, root, "agent")
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "in-flight tool") {
		t.Fatalf("the owning session's Stop must still block: %s", out)
	}
}

// Invariant: a subagent lane's unfinished call blocks the main thread's Stop
// of the same session, and the main thread's next prompt does not close it.
func TestInvariantSameSessionSubagentTokenStillBlocks(t *testing.T) {
	root := greenFieldRoot(t)
	sub := Input{SessionID: "s", AgentID: "worker", ToolUseID: "toolu_sub", Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, sub); out != "" {
		t.Fatal(out)
	}
	if out, err := runHookPayload(t, root, map[string]any{"hook_event_name": "UserPromptSubmit", "session_id": "s", "cwd": root}); err != nil || out != "" {
		t.Fatalf("%s %v", out, err)
	}
	if out := stopOutput(t, root, "s"); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "in-flight tool") {
		t.Fatalf("same-session subagent call must block: %s", out)
	}
}

// Invariant: a store directory replaced by a new one carrying a copied
// identity record is refused by governed events and by plain adoption.
func TestInvariantReplacedStoreWithCopiedIdentityIsRefused(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix native identity")
	}
	isolateHookState(t)
	root := managedRoot(t)
	event := editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md"))
	runEvent(t, root, event)
	dir := stateDirPath()
	identity, err := os.ReadFile(filepath.Join(dir, stateDirectoryIdentityName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(dir, dir+".parked"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, stateDirectoryIdentityName), identity, 0o600); err != nil {
		t.Fatal(err)
	}
	if out := runEvent(t, root, event); !strings.Contains(out, `"permissionDecision":"deny"`) || !strings.Contains(out, "replacement store") {
		t.Fatalf("a replaced store was accepted by PreToolUse: %s", out)
	}
	if out := stopOutput(t, root, "seat"); !strings.Contains(out, `"decision":"block"`) {
		t.Fatalf("a replaced store was accepted by Stop: %s", out)
	}
	var output bytes.Buffer
	if err := AdoptState(&output, root, ""); err == nil {
		t.Fatal("plain adoption accepted a replaced store")
	}
}
