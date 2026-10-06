package hook

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Field regressions for the stranded-token and foreign-route lockouts
// reported on 2026-10-06 (MAC-ntvm). Each test drives the hook only through
// Run with the payloads the hosts send, so the same file reproduces the
// lockout against a binary that predates the fix.

// runHookPayload pipes a raw host payload through Run.
func runHookPayload(t *testing.T, root string, payload map[string]any) (string, error) {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = Run(bytes.NewReader(raw), &out, root)
	return out.String(), err
}

func greenFieldRoot(t *testing.T) string {
	t.Helper()
	isolateHookState(t)
	root := t.TempDir()
	copyTree(t, crmDesign, filepath.Join(root, "design"))
	return root
}

func armShell(t *testing.T, root, session, toolUseID string) {
	t.Helper()
	pre := Input{SessionID: session, ToolUseID: toolUseID, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, pre); out != "" {
		t.Fatalf("PreToolUse for %s denied: %s", toolUseID, out)
	}
}

func completeShell(t *testing.T, root, session, toolUseID string) {
	t.Helper()
	post := Input{SessionID: session, ToolUseID: toolUseID, Cwd: root, HookEventName: "PostToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, post); out != "" {
		t.Fatalf("PostToolUse for %s: %s", toolUseID, out)
	}
}

func stopOutput(t *testing.T, root, session string) string {
	t.Helper()
	return runEvent(t, root, Input{SessionID: session, Cwd: root, HookEventName: "Stop"})
}

func requireNotWedged(t *testing.T, out, context string) {
	t.Helper()
	if strings.Contains(out, "in-flight tool") || strings.Contains(out, "different routing configuration") {
		t.Fatalf("%s: Stop is wedged on state it can never clear: %s", context, out)
	}
}

func requireArmed(t *testing.T, root string, pending int, context string) {
	t.Helper()
	state, err := readStateRecord(root, "inspector")
	if err != nil || !state.design || len(state.pending) != pending {
		t.Fatalf("%s: want the obligation armed with %d pending token(s), got state=%+v err=%v", context, pending, state, err)
	}
}

// Field case (a): the host was SIGKILLed (pod eviction) between PreToolUse
// and PostToolUse. A fresh session must not be blocked forever by that token,
// and the obligation must stay armed so the gates keep running.
func TestFieldSIGKILLMidToolCallDoesNotWedgeNextSession(t *testing.T) {
	if os.Getenv("MACHINERY_FIELD_KILLED_HOST") == "1" {
		root := os.Getenv("MACHINERY_FIELD_KILLED_ROOT")
		pre := Input{SessionID: "evicted-conductor", ToolUseID: "toolu_inflight", Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "sleep 600"}}
		raw, _ := json.Marshal(pre)
		if err := Run(bytes.NewReader(raw), os.Stdout, root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		fmt.Println("ARMED")
		time.Sleep(10 * time.Minute) // the tool call the host never finishes
		os.Exit(0)
	}
	root := greenFieldRoot(t)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFieldSIGKILLMidToolCallDoesNotWedgeNextSession$")
	cmd.Env = append(os.Environ(), "MACHINERY_FIELD_KILLED_HOST=1", "MACHINERY_FIELD_KILLED_ROOT="+root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	armed := make(chan bool, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "ARMED") {
				armed <- true
				return
			}
		}
		armed <- false
	}()
	select {
	case ok := <-armed:
		if !ok {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatal("host child exited before arming its tool call")
		}
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("host child never armed its tool call")
	}
	if err := cmd.Process.Kill(); err != nil { // SIGKILL on Unix
		t.Fatal(err)
	}
	_ = cmd.Wait()
	requireArmed(t, root, 1, "after the kill")

	armShell(t, root, "fresh-session", "toolu_git_log")
	completeShell(t, root, "fresh-session", "toolu_git_log")
	for i := 0; i < 2; i++ {
		out := stopOutput(t, root, "fresh-session")
		requireNotWedged(t, out, "fresh session after SIGKILL")
		if strings.Contains(out, `"decision":"block"`) {
			t.Fatalf("green fresh session blocked: %s", out)
		}
		requireArmed(t, root, 1, "fresh session Stop")
	}
}

// Field case (b): Esc interrupts a running tool. The host sends neither
// PostToolUse nor PostToolUseFailure and no Stop; the next prompt, a resolved
// tool batch, a Codex Interrupt, or the session end proves the call is over.
func TestFieldInterruptWithoutPostToolUseDoesNotStrandToken(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload map[string]any
	}{
		{"claude-next-prompt", map[string]any{"hook_event_name": "UserPromptSubmit", "user_prompt": "continue", "permission_mode": "default", "prompt_id": "p2"}},
		{"codex-next-prompt", map[string]any{"hook_event_name": "UserPromptSubmit", "prompt": "continue", "turn_id": "t2", "model": "gpt"}},
		{"codex-interrupt", map[string]any{"hook_event_name": "Interrupt", "turn_id": "t1", "permission_mode": "default"}},
		{"claude-denied-in-batch", map[string]any{"hook_event_name": "PostToolBatch", "tool_calls": []any{map[string]any{"tool_name": "Bash", "tool_use_id": "toolu_interrupted", "tool_input": map[string]any{"command": "sleep 600"}, "tool_result": "Interrupted by user"}}}},
		{"esc-then-exit", map[string]any{"hook_event_name": "SessionEnd", "reason": "prompt_input_exit"}},
	} {
		boundary, name := tc.payload, tc.payload["hook_event_name"].(string)
		t.Run(tc.name, func(t *testing.T) {
			root := greenFieldRoot(t)
			session := "interrupted-session"
			armShell(t, root, session, "toolu_interrupted")
			boundary["session_id"] = session
			boundary["cwd"] = root
			boundary["transcript_path"] = filepath.Join(root, "transcript.jsonl")
			if out, err := runHookPayload(t, root, boundary); err != nil || out != "" {
				t.Fatalf("%s boundary failed: out=%s err=%v", name, out, err)
			}
			out := stopOutput(t, root, session)
			requireNotWedged(t, out, name)
			if out != "" {
				t.Fatalf("green Stop after %s should be silent: %s", name, out)
			}
			if state, err := readStateRecord(root, session); err != nil || state.design || state.impl {
				t.Fatalf("green Stop after the gates ran must discharge: %+v %v", state, err)
			}
		})
	}
}

// Field case (c): the plugin was disabled and re-enabled across sessions,
// leaving one PreToolUse token per session with no PostToolUse (13 observed).
func TestFieldPluginToggleStrandedTokensDoNotWedgeLaterSessions(t *testing.T) {
	root := greenFieldRoot(t)
	for i := 0; i < 13; i++ {
		armShell(t, root, fmt.Sprintf("toggled-session-%02d", i), fmt.Sprintf("toolu_toggled_%02d", i))
	}
	armShell(t, root, "re-enabled", "toolu_now")
	completeShell(t, root, "re-enabled", "toolu_now")
	out := stopOutput(t, root, "re-enabled")
	requireNotWedged(t, out, "re-enabled plugin")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "13 tool operation(s)") {
		t.Fatalf("stranded tokens were not reported as orphaned: %s", out)
	}
	requireArmed(t, root, 13, "re-enabled plugin")
}

// Field case (d): the obligation was armed under one routing configuration
// and .machinery.json changed before the Stop. The gates must run under the
// current configuration instead of refusing forever.
func TestFieldRouteChangeAfterArmedObligationReRunsGates(t *testing.T) {
	root := greenFieldRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design"}`)
	armShell(t, root, "routed", "toolu_before_change")
	completeShell(t, root, "routed", "toolu_before_change")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","dialog":"plain"}`)
	out := stopOutput(t, root, "routed")
	requireNotWedged(t, out, "operator route change")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "re-evaluated") {
		t.Fatalf("route change was not re-evaluated under the current configuration: %s", out)
	}
	if state, err := readStateRecord(root, "routed"); err != nil || state.design || state.impl {
		t.Fatalf("green gates under the current route must discharge: %+v %v", state, err)
	}
}

// Field case (c)+(d) as found on the laptop: a ledger written by v0.11.1
// carries two route identities and thirteen owner-less pending tokens.
func TestFieldLegacyLedgerWithForeignRoutesAndTokensDoesNotWedge(t *testing.T) {
	root := greenFieldRoot(t)
	if err := appendState(root, "legacy-writer", "design"); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	var ledger strings.Builder
	ledger.WriteString("revision 1159\n" + hookStateRootLine(canonical) + "\ndesign\nimpl\n")
	ledger.WriteString("route 1dee6b9c" + strings.Repeat("0", 56) + "\n")
	ledger.WriteString("route b38873d4" + strings.Repeat("0", 56) + "\n")
	for i := 0; i < 13; i++ {
		fmt.Fprintf(&ledger, "pending %02x%s\n", i, strings.Repeat("a", 62))
	}
	if err := os.WriteFile(statePath(canonical, ""), []byte(ledger.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	// Owner-less tokens cannot be proven foreign, so the Stop fails closed,
	// but it names the audited operator recovery instead of wedging silently.
	out := stopOutput(t, root, "laptop-session")
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "no recorded owning session") || !strings.Contains(out, "hook-state release") || !strings.Contains(out, "--orphaned") {
		t.Fatalf("legacy ledger must block with its recovery named: %s", out)
	}
	state, err := readStateRecord(root, "inspector")
	if err != nil || !state.design || !state.impl || len(state.pending) != 13 {
		t.Fatalf("legacy obligation was not retained: %+v %v", state, err)
	}
}

// Field case (d), mid-call variant: the operator edits .machinery.json while
// a tool runs. Its completion must still close the token, and the Stop must
// not wedge.
func TestFieldRouteChangeDuringToolCallStillCompletes(t *testing.T) {
	root := greenFieldRoot(t)
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design"}`)
	armShell(t, root, "mid-call", "toolu_running")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","dialog":"plain"}`)
	completeShell(t, root, "mid-call", "toolu_running")
	// The completion closed the token. The earlier route's snapshot was
	// replaced by the completion's, so the change cannot be compared: the
	// gates run, nothing blocks, and the obligation waits for an operator
	// acceptance instead of being cleared on an unknown comparison.
	out := stopOutput(t, root, "mid-call")
	requireNotWedged(t, out, "route change during a tool call")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "--routes") {
		t.Fatalf("an uncomparable route change must run the gates and name the acceptance: %s", out)
	}
	requireArmed(t, root, 0, "uncomparable route change")
}
