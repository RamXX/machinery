package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPendingTokenRecordsItsOwningSessionAndLane(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "owner-session", "toolu_main")
	sub := Input{SessionID: "owner-session", AgentID: "agent-7", ToolUseID: "toolu_sub", Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, sub); out != "" {
		t.Fatalf("subagent PreToolUse: %s", out)
	}
	state, err := readStateRecord(root, "owner-session")
	if err != nil || len(state.pending) != 2 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	lanes := map[string]bool{}
	for _, token := range state.pending {
		owner := state.pendingOwners[token]
		if owner.session != hookSessionDigest("owner-session") {
			t.Fatalf("token %s owner session = %q", token, owner.session)
		}
		lanes[owner.lane] = true
	}
	if !lanes[hookLaneDigest("owner-session", "")] || !lanes[hookLaneDigest("owner-session", "agent-7")] {
		t.Fatalf("main and subagent lanes were not distinguished: %v", lanes)
	}
	raw, err := os.ReadFile(statePath(root, ""))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "owner-session") || strings.Contains(string(raw), "agent-7") {
		t.Fatalf("the ledger must store digests, never raw session or agent ids: %s", raw)
	}
	reparsed, err := parseHookStateRecord(raw)
	if err != nil || !bytes.Equal(formatHookStateRecord(reparsed), raw) {
		t.Fatalf("owner lines are not canonical: err=%v\n%s", err, raw)
	}
}

func TestOwnSessionTokenStillBlocksItsStop(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "foreign", "toolu_foreign")
	armShell(t, root, "self", "toolu_self")
	out := stopOutput(t, root, "self")
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "1 in-flight tool") {
		t.Fatalf("a session's own unfinished tool call must still block its Stop: %s", out)
	}
	requireArmed(t, root, 2, "own token block")
}

func TestOrphanedTokensKeepObligationArmedThroughRedAndGreenStops(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "dead", "toolu_dead")
	oracle := filepath.Join(root, "design", "machines", "Deal.oracle.md")
	original, err := os.ReadFile(oracle)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, oracle, string(original)+"\nhand edit\n")
	out := stopOutput(t, root, "live")
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "DRIFT") || !strings.Contains(out, "--orphaned") {
		t.Fatalf("red gates must still block, naming the orphaned token: %s", out)
	}
	writeFile(t, oracle, string(original))
	out = stopOutput(t, root, "live")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "stays armed") {
		t.Fatalf("green gates with an orphaned token must report and retain: %s", out)
	}
	requireArmed(t, root, 1, "green with orphan")
}

func TestBoundaryClosesOnlyItsOwnLane(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "s1", "toolu_main")
	sub := Input{SessionID: "s1", AgentID: "background-agent", ToolUseID: "toolu_sub", Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, sub); out != "" {
		t.Fatal(out)
	}
	armShell(t, root, "s2", "toolu_other_session")
	prompt := map[string]any{"hook_event_name": "UserPromptSubmit", "session_id": "s1", "cwd": root, "user_prompt": "next"}
	if out, err := runHookPayload(t, root, prompt); err != nil || out != "" {
		t.Fatalf("prompt boundary: %s %v", out, err)
	}
	requireArmed(t, root, 2, "a main-lane prompt must keep the background subagent's and the other session's tokens")
	batch := map[string]any{"hook_event_name": "PostToolBatch", "session_id": "s1", "agent_id": "background-agent", "cwd": root, "tool_calls": []any{map[string]any{"tool_use_id": "toolu_sub"}, map[string]any{"tool_use_id": "toolu_main"}}}
	if out, err := runHookPayload(t, root, batch); err != nil || out != "" {
		t.Fatalf("batch boundary: %s %v", out, err)
	}
	requireArmed(t, root, 1, "the subagent lane batch closes only that lane")
	end := map[string]any{"hook_event_name": "SessionEnd", "session_id": "s1", "cwd": root, "reason": "other"}
	if out, err := runHookPayload(t, root, end); err != nil || out != "" {
		t.Fatalf("end boundary: %s %v", out, err)
	}
	requireArmed(t, root, 1, "another session's token survives this session's end")
}

func TestBoundaryNeverClosesLegacyOwnerlessTokens(t *testing.T) {
	root := greenFieldRoot(t)
	if err := appendState(root, "legacy", "design"); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	ledger := "revision 3\n" + hookStateRootLine(canonical) + "\ndesign\npending " + strings.Repeat("c", 64) + "\n"
	if err := os.WriteFile(statePath(canonical, ""), []byte(ledger), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"UserPromptSubmit", "SessionEnd", "PostToolBatch", "Interrupt"} {
		if out, err := runHookPayload(t, root, map[string]any{"hook_event_name": event, "session_id": "any", "cwd": root}); err != nil || out != "" {
			t.Fatalf("%s: %s %v", event, out, err)
		}
	}
	requireArmed(t, root, 1, "owner-less legacy token")
}

func TestBoundaryIsSilentInUnmanagedProjectsAndToleratesNewHostFields(t *testing.T) {
	isolateHookState(t)
	root := t.TempDir()
	for _, event := range []string{"UserPromptSubmit", "SessionEnd", "PostToolBatch", "Interrupt"} {
		payload := map[string]any{"hook_event_name": event, "session_id": "s", "cwd": root, "a_field_from_the_future": map[string]any{"x": 1}}
		if out, err := runHookPayload(t, root, payload); err != nil || out != "" {
			t.Fatalf("%s in an unmanaged project: out=%q err=%v", event, out, err)
		}
	}
	if _, err := os.Lstat(stateDirPath()); !os.IsNotExist(err) {
		t.Fatalf("a boundary event must never create the durable store: %v", err)
	}
	if _, err := runHookPayload(t, root, map[string]any{"hook_event_name": "SessionEnd", "cwd": root}); err == nil {
		t.Fatal("a boundary event without session_id must be refused, not guessed")
	}
}

func TestCodexTurnIDIsAccepted(t *testing.T) {
	root := greenFieldRoot(t)
	payload := map[string]any{"hook_event_name": "PreToolUse", "session_id": "codex", "turn_id": "turn-1", "tool_use_id": "call_1", "cwd": root, "tool_name": "Bash", "tool_input": map[string]any{"command": "true"}, "model": "gpt", "permission_mode": "default"}
	if out, err := runHookPayload(t, root, payload); err != nil || out != "" {
		t.Fatalf("Codex PreToolUse with turn_id: out=%s err=%v", out, err)
	}
}

func TestReleaseOrphanedKeepsObligationAndJournalsTheOperator(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "evicted", "toolu_a")
	armShell(t, root, "evicted", "toolu_b")
	var out bytes.Buffer
	if err := ReleaseState(&out, root, nil, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Released 2 in-flight tool token(s)") || !strings.Contains(out.String(), "stays armed") {
		t.Fatalf("release output: %s", out.String())
	}
	requireArmed(t, root, 0, "release keeps the obligation")
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
		t.Fatalf("journal: %v %s", err, journal)
	}
	tokens, _ := event["tokens"].([]any)
	if event["event"] != "release" || len(tokens) != 2 || event["operator"] == nil || event["time"] == nil || event["host"] == nil {
		t.Fatalf("release was not journaled with who, when and which: %s", journal)
	}
	// The gates still run before the obligation can clear.
	if out := stopOutput(t, root, "next"); out != "" {
		t.Fatalf("green Stop after release: %s", out)
	}
	if state, err := readStateRecord(root, "next"); err != nil || state.design || state.impl {
		t.Fatalf("green Stop after release must discharge: %+v %v", state, err)
	}
}

func TestReleaseSingleTokenByPrefix(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "x", "toolu_a")
	armShell(t, root, "y", "toolu_b")
	state, err := readStateRecord(root, "x")
	if err != nil {
		t.Fatal(err)
	}
	target := state.pending[0]
	var out bytes.Buffer
	for _, bad := range [][]string{{target[:8]}, {strings.Repeat("f", 64)}, {"not-hex-at-all-zz"}} {
		if err := ReleaseState(&out, root, bad, false); err == nil {
			t.Fatalf("release accepted %v", bad)
		}
	}
	if err := ReleaseState(&out, root, nil, false); err == nil {
		t.Fatal("release without a selector was accepted")
	}
	if err := ReleaseState(&out, root, []string{target}, true); err == nil {
		t.Fatal("release with both selectors was accepted")
	}
	requireArmed(t, root, 2, "refused releases change nothing")
	if err := ReleaseState(&out, root, []string{target[:12]}, false); err != nil {
		t.Fatal(err)
	}
	after, err := readStateRecord(root, "x")
	if err != nil || len(after.pending) != 1 || after.pending[0] == target || !after.design {
		t.Fatalf("prefix release: %+v %v", after, err)
	}
}

func TestReleaseWithoutObligationIsANoop(t *testing.T) {
	root := greenFieldRoot(t)
	var out bytes.Buffer
	if err := ReleaseState(&out, root, nil, true); err != nil || !strings.Contains(out.String(), "nothing to release") {
		t.Fatalf("out=%s err=%v", out.String(), err)
	}
}

func TestDoctorReportsStrandedTokensAndForeignRoutesWithRecovery(t *testing.T) {
	root := greenFieldRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design"}`)
	armShell(t, root, "gone", "toolu_gone")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","strict":true}`)
	var report bytes.Buffer
	ok, err := StateReport(&report, false)
	if err != nil || !ok {
		t.Fatalf("stranded tokens alone must not fail doctor: ok=%v err=%v %s", ok, err, report.String())
	}
	text := report.String()
	canonical, _ := filepath.EvalSymlinks(root)
	for _, want := range []string{"1 in-flight tool token", "machinery hook-state release --root '" + canonical + "' --orphaned", "differ from the current .machinery.json", "machinery check '" + filepath.Join(canonical, "design") + "'"} {
		if !strings.Contains(text, want) {
			t.Fatalf("doctor report is missing %q:\n%s", want, text)
		}
	}
}

func TestNativeIdentityQualifiersStillDistinguishStores(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Windows compares its native witness exactly")
	}
	cases := []struct {
		a, b string
		same bool
	}{
		{"unix:850:80e50:gen:7", "unix:840:80e50:gen:7", true},
		{"unix:850:80e50:gen:7", "unix:850:80e50:gen:8", false},
		{"unix:1:80e50:birth:5:6", "unix:0:80e50", true},
		{"unix:1:80e50:birth:5:6", "unix:2:80e50:birth:5:7", false},
		{"unix:1:80e50", "unix:1:80e51", false},
	}
	for _, tc := range cases {
		if got := sameHookNativeIdentity(tc.a, tc.b); got != tc.same {
			t.Errorf("sameHookNativeIdentity(%s, %s) = %v, want %v", tc.a, tc.b, got, tc.same)
		}
	}
}

// An inode change (the store moved to another filesystem) is refused by the
// hook, doctor prescribes the one adoption that succeeds, and adoption with
// the explicit rebind prints the old and new identity.
func TestAdoptRebindsVerifiedStoreToNewNativeIdentity(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix native identity")
	}
	isolateHookState(t)
	root := managedRoot(t)
	event := editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md"))
	runEvent(t, root, event)
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	_, _, binding, err := readStateInitializationMarker(marker)
	if err != nil {
		t.Fatal(err)
	}
	moved := "unix:abc:fffffff"
	binding.native = moved
	if err := os.WriteFile(filepath.Join(stateDirPath(), stateDirectoryIdentityName), binding.identityBody(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, binding.markerBody(), 0o600); err != nil {
		t.Fatal(err)
	}
	denied := runEvent(t, root, event)
	if !strings.Contains(denied, "replacement store") || !strings.Contains(denied, "--rebind-identity") {
		t.Fatalf("hook refusal does not name the recovery: %s", denied)
	}
	var report bytes.Buffer
	if ok, _ := StateReport(&report, false); ok || !strings.Contains(report.String(), "--rebind-identity") {
		t.Fatalf("doctor must prescribe the rebind: %s", report.String())
	}
	var output bytes.Buffer
	if err := AdoptState(&output, root, ""); err == nil || !strings.Contains(err.Error(), "--rebind-identity") {
		t.Fatalf("plain adoption must refuse an inode change and name the flag: %v", err)
	}
	if err := AdoptStateWith(&output, root, "", AdoptOptions{RebindIdentity: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "rebound store identity: "+moved+" -> ") || !strings.Contains(output.String(), "pending=1") {
		t.Fatalf("adoption must print old and new identity and retained obligations: %s", output.String())
	}
	if out := runEvent(t, root, event); out != "" {
		t.Fatalf("rebound store still denied: %s", out)
	}
	report.Reset()
	if ok, _ := StateReport(&report, false); !ok {
		t.Fatalf("doctor after rebind: %s", report.String())
	}
}

// A store whose identity AND generation differ is not the recorded store:
// doctor must not prescribe an adoption that is guaranteed to refuse.
func TestDoctorNeverPrescribesAGuaranteedRefusal(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Unix native identity")
	}
	isolateHookState(t)
	root := managedRoot(t)
	runEvent(t, root, editEvent("PreToolUse", "Write", "seat", filepath.Join(root, "design", "BUILD.md")))
	identityPath := filepath.Join(stateDirPath(), stateDirectoryIdentityName)
	raw, err := os.ReadFile(identityPath)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := parseStateDirectoryBinding(raw, "machinery-hook-state-directory-v1")
	if err != nil {
		t.Fatal(err)
	}
	stored.native = "unix:abc:fffffff"
	stored.generation = strings.Repeat("1", 64)
	if err := os.WriteFile(identityPath, stored.identityBody(), 0o600); err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	if ok, _ := StateReport(&report, false); ok {
		t.Fatalf("doctor accepted a different store: %s", report.String())
	}
	text := report.String()
	if strings.Contains(text, "adopt --root <root>\n") || strings.Contains(text, "--rebind-identity") || !strings.Contains(text, "--from <copy>") {
		t.Fatalf("doctor prescribed a command that must refuse: %s", text)
	}
	var output bytes.Buffer
	for _, opts := range []AdoptOptions{{}, {RebindIdentity: true}} {
		if err := AdoptStateWith(&output, root, "", opts); err == nil {
			t.Fatalf("a different store was adopted with %+v", opts)
		}
	}
}

func TestBoundaryHooksAreWiredNonBlocking(t *testing.T) {
	raw, err := os.ReadFile(repoPath("hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
				Timeout int    `json:"timeout"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	for event := range boundaryEvents {
		entries := doc.Hooks[event]
		if len(entries) != 1 || len(entries[0].Hooks) != 1 || entries[0].Hooks[0].Command != "${CLAUDE_PLUGIN_ROOT}/hooks/machinery-hook.sh boundary" || entries[0].Hooks[0].Timeout <= 0 {
			t.Fatalf("boundary event %s is not wired to the non-blocking shim mode: %+v", event, entries)
		}
	}
}

// A boundary notice must never block a prompt or a session end, even in a
// managed project whose binary is missing or failing.
func TestShimBoundaryModeNeverBlocks(t *testing.T) {
	root := managedRoot(t)
	shim, err := filepath.Abs(repoPath("hooks", "machinery-hook.sh"))
	if err != nil {
		t.Fatal(err)
	}
	failing := t.TempDir()
	if err := os.WriteFile(filepath.Join(failing, "machinery"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, path := range map[string]string{"missing": t.TempDir(), "failing": failing} {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{{shim, "boundary"}, {shim}} {
				cmd := exec.CommandContext(t.Context(), "/bin/sh", args...)
				cmd.Dir = root
				cmd.Env = []string{"CLAUDE_PROJECT_DIR=" + root, "HOME=" + t.TempDir(), "PATH=" + path + string(os.PathListSeparator) + "/usr/bin:/bin"}
				cmd.Stdin = strings.NewReader(`{"hook_event_name":"UserPromptSubmit","session_id":"s"}`)
				err := cmd.Run()
				boundary := len(args) == 2
				if boundary && err != nil {
					t.Fatalf("boundary mode blocked with a %s binary: %v", name, err)
				}
				if !boundary && err == nil {
					t.Fatalf("governing mode must still fail closed with a %s binary", name)
				}
			}
		})
	}
}
