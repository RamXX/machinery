package hook

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Threat note (MAC-ntvm security review). Paths by which an agent session
// could release, rebind, or ignore its own tokens or obligation without the
// operator, and states where boundary logic and the ledger could disagree
// into allow; each is closed fail-closed and tested here or in
// security_invariants_test.go:
//
//  1. Run 'machinery hook-state release|adopt' through a tool call: denied in
//     PreToolUse by command text, and refused by the operator gate (agent
//     session markers or no terminal on stdin). Release never clears an
//     obligation and is journaled.
//  2. Pipe forged events into 'machinery hook' through a tool call: denied
//     in PreToolUse on the command as the shell joins it (quotes and
//     backslashes removed before splitting). The guard is defense in depth:
//     a forged boundary event only marks tokens of a known owner ended, an
//     ended token withholds discharge from every Stop except the owner's
//     main-thread Stop (which the host fires only after every foreground
//     call resolved), and nothing discharges an obligation without gates.
// 10. Ambiguous hook JSON (duplicate keys, case-folded aliases of a routing
//     key, trailing content) naming a boundary event is refused; one
//     scanner reads the routing keys once, and owners are always derived by
//     the same digest functions from the same decoded ids.
//  3. Read, write, or move the hook store or its marker with a file or shell
//     tool: denied in PreToolUse by path text.
//  4. A token with no recorded owner (older ledger, or armed by an event
//     without a session id): unclassifiable, so it blocks every Stop until
//     an operator releases it; boundaries never close it.
//  5. A Stop without a session id: every token counts as its own and blocks.
//  6. A recorded route whose snapshot is gone: treated as narrowing, so the
//     gates run but the obligation is never discharged without the
//     journaled 'release --routes'.
//  7. A PostToolBatch: closes only the exact calls it lists, in its own lane.
//  8. Any error reading the ledger, a route snapshot, or the store binding:
//     the Stop blocks and PreToolUse denies (pre-existing, unchanged).
//  9. A boundary event that fails to decode or to write: it closes nothing,
//     so the token keeps blocking its own session (the shim exits 0 only so
//     a prompt is never blocked; governance is unaffected).
//
// Residual for host-sent boundaries: if a host fired UserPromptSubmit while
// a main-lane tool were still running, that token would close early; the
// tool's own PostToolUse re-arms the obligation, so only a write whose
// completion is also lost escapes until a later Stop or CI.
//
// Residual, stated rather than hidden: command-text guards cannot see
// dynamically assembled commands, and an agent that scrubs its environment
// and fakes a terminal passes the operator gate. Neither path can clear an
// obligation: release removes tokens or route identities, never the
// design/impl obligation, so the gates still run before anything clears, and
// CI's machinery check remains the outer wall.

func TestOperatorGateRefusesAgentSessionsAndNonInteractiveStdin(t *testing.T) {
	for _, tc := range []struct {
		env         []string
		interactive bool
		ok          bool
	}{
		{[]string{"PATH=/bin", "HOME=/h"}, true, true},
		{[]string{"CLAUDECODE=1"}, true, false},
		{[]string{"CLAUDE_CODE_SESSION_ID=abc"}, true, false},
		{[]string{"CODEX_THREAD_ID=t"}, true, false},
		{[]string{"OPENCODE=1"}, true, false},
		{[]string{"CLAUDECODE="}, true, true},
		{[]string{"PATH=/bin"}, false, false},
	} {
		err := requireOperator(tc.env, tc.interactive)
		if (err == nil) != tc.ok {
			t.Errorf("requireOperator(%v, %v) = %v, want ok=%v", tc.env, tc.interactive, err, tc.ok)
		}
	}
}

func TestReleaseAndAdoptRefuseWithoutTheOperatorGate(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "agent", "toolu_own")
	prior := operatorGate
	operatorGate = func() error { return requireOperator([]string{"CLAUDECODE=1"}, false) }
	t.Cleanup(func() { operatorGate = prior })
	var out bytes.Buffer
	if err := ReleaseStateWith(&out, root, ReleaseOptions{Orphaned: true, Routes: true}); err == nil || !strings.Contains(err.Error(), "operator command") {
		t.Fatalf("release ran inside an agent session: %v", err)
	}
	if err := AdoptStateWith(&out, root, "", AdoptOptions{RebindIdentity: true}); err == nil || !strings.Contains(err.Error(), "operator command") {
		t.Fatalf("adopt ran inside an agent session: %v", err)
	}
	requireArmed(t, root, 1, "refused operator commands change nothing")
	marker, err := stateInitializationMarkerPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker + ".handoffs"); !os.IsNotExist(err) {
		t.Fatalf("a refused release must not journal or mutate: %v", err)
	}
}

func TestPreToolUseDeniesOperatorSurfacesToAgents(t *testing.T) {
	root := greenFieldRoot(t)
	store := stateDirPath()
	for _, command := range []string{
		"machinery hook-state release --root . --orphaned",
		"MACHINERY=1 machinery hook-state adopt --root . --rebind-identity",
		`echo '{"hook_event_name":"SessionEnd","session_id":"x"}' | machinery hook --root .`,
		`printf x | ~/.local/bin/machinery hook`,
		"rm -f " + filepath.Join(store, "x.state"),
		"cat ~/.machinery-hook-state-abc.initialized",
	} {
		pre := Input{SessionID: "agent", ToolUseID: "t-" + command, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: command}}
		if out := runEvent(t, root, pre); !strings.Contains(out, `"permissionDecision":"deny"`) {
			t.Fatalf("agent shell reached an operator surface: %q -> %s", command, out)
		}
	}
	for _, allowed := range []string{"machinery check design", "git log --oneline", "machinery doctor"} {
		pre := Input{SessionID: "agent", ToolUseID: "ok-" + allowed, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: allowed}}
		if out := runEvent(t, root, pre); out != "" {
			t.Fatalf("ordinary command denied: %q -> %s", allowed, out)
		}
	}
	edit := editEvent("PreToolUse", "Write", "agent", filepath.Join(store, "forged.state"))
	if out := runEvent(t, root, edit); !strings.Contains(out, `"permissionDecision":"deny"`) {
		t.Fatalf("file tool reached the hook store: %s", out)
	}
}

func TestUnclassifiedTokensBlockEveryStopUntilReleased(t *testing.T) {
	root := greenFieldRoot(t)
	sessionless := Input{ToolUseID: "toolu_no_session", Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: "true"}}
	if out := runEvent(t, root, sessionless); out != "" {
		t.Fatal(out)
	}
	state, err := readStateRecord(root, "x")
	if err != nil || len(state.pending) != 1 || state.pendingOwners[state.pending[0]].known() {
		t.Fatalf("a session-less token must be recorded without an owner: %+v %v", state, err)
	}
	for _, event := range []string{"SessionEnd", "UserPromptSubmit"} {
		if out, err := runHookPayload(t, root, map[string]any{"hook_event_name": event, "session_id": "", "cwd": root}); err == nil || out != "" {
			t.Fatalf("a boundary without a session id must be refused: %s %v", out, err)
		}
	}
	for _, session := range []string{"a", "b"} {
		if out := stopOutput(t, root, session); !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "no recorded owning session") {
			t.Fatalf("unclassified token did not block session %s: %s", session, out)
		}
	}
	var released bytes.Buffer
	if err := ReleaseStateWith(&released, root, ReleaseOptions{Orphaned: true}); err != nil {
		t.Fatal(err)
	}
	requireArmed(t, root, 0, "release keeps the obligation")
	if out := stopOutput(t, root, "a"); out != "" {
		t.Fatalf("after release the gates run and clear: %s", out)
	}
}

func TestStopWithoutSessionIDTreatsEveryTokenAsItsOwn(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "someone", "toolu_x")
	out := runEvent(t, root, Input{Cwd: root, HookEventName: "Stop"})
	if !strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "in-flight tool") {
		t.Fatalf("a session-less Stop must fail closed on any token: %s", out)
	}
}

func TestPostToolBatchClosesOnlyListedCalls(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "s", "toolu_listed")
	armShell(t, root, "s", "toolu_unlisted")
	batch := map[string]any{"hook_event_name": "PostToolBatch", "session_id": "s", "cwd": root, "tool_calls": []any{map[string]any{"tool_use_id": "toolu_listed"}}}
	if out, err := runHookPayload(t, root, batch); err != nil || out != "" {
		t.Fatalf("%s %v", out, err)
	}
	requireLive(t, root, 1, "an unlisted call survives the batch")
	if out := stopOutput(t, root, "s"); !strings.Contains(out, `"decision":"block"`) {
		t.Fatalf("the unlisted own call must still block: %s", out)
	}
	malformed := map[string]any{"hook_event_name": "PostToolBatch", "session_id": "s", "cwd": root, "tool_calls": "not-a-list"}
	if _, err := runHookPayload(t, root, malformed); err == nil {
		t.Fatal("a malformed batch must be refused, not treated as resolving anything")
	}
	requireLive(t, root, 1, "a malformed batch closes nothing")
}

func TestUncomparableRouteNeverDischargesWithoutAcceptance(t *testing.T) {
	root := greenFieldRoot(t)
	armUnder(t, root, `{"design":"design"}`, "s", "toolu_1")
	paths, err := routeStatePaths(root)
	if err != nil || len(paths) == 0 {
		t.Fatalf("route snapshots: %v %v", paths, err)
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design","dialog":"plain"}`)
	out := stopOutput(t, root, "s")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "can no longer be compared") {
		t.Fatalf("an uncomparable route must run the gates and retain: %s", out)
	}
	requireArmed(t, root, 0, "uncomparable route")
}

// Review finding F1, sequence B: a SessionEnd forged for another live session
// marks its tokens ended, but they still withhold discharge from every other
// session, so the forger cannot clear the obligation while that session's
// call may still be writing.
func TestForgedSessionEndForAnotherSessionCannotLetOthersDischarge(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "victim", "toolu_victim_writing")
	forged := map[string]any{"hook_event_name": "SessionEnd", "session_id": "victim", "cwd": root}
	if out, err := runHookPayload(t, root, forged); err != nil || out != "" {
		t.Fatalf("%s %v", out, err)
	}
	out := stopOutput(t, root, "forger")
	if strings.Contains(out, `"decision":"block"`) || !strings.Contains(out, "stays armed") {
		t.Fatalf("forger's Stop: %s", out)
	}
	requireArmed(t, root, 1, "an ended foreign token still withholds discharge")
	// The owning session's own Stop is backed by its host: it may discharge.
	if out := stopOutput(t, root, "victim"); out != "" {
		t.Fatalf("owner's Stop after its own session end boundary: %s", out)
	}
	if state, err := readStateRecord(root, "victim"); err != nil || state.design {
		t.Fatalf("owner's green Stop must discharge: %+v %v", state, err)
	}
}

// Review finding F1: literal spellings that a plain regular expression missed.
func TestPreToolUseDeniesLiteralHookInvocationVariants(t *testing.T) {
	root := greenFieldRoot(t)
	for _, command := range []string{
		`machinery "hook" <<<'{}'`,
		`machinery hook<<<'{"hook_event_name":"SessionEnd"}'`,
		`machinery hook<f`,
		`machinery hook>o`,
		`machinery\ hook`,
		`/usr/local/bin/machinery --root . hook`,
		`"$plugin"/hooks/machinery-hook.sh boundary`,
		`(machinery hook)`,
	} {
		pre := Input{SessionID: "agent", ToolUseID: "v-" + command, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: command}}
		if out := runEvent(t, root, pre); !strings.Contains(out, `"permissionDecision":"deny"`) {
			t.Fatalf("hook invocation variant was allowed: %q -> %s", command, out)
		}
	}
	for _, allowed := range []string{"machinery check design --gate g2", "git commit -m 'hooks: tidy'", "machinery doctor --repair"} {
		pre := Input{SessionID: "agent", ToolUseID: "a-" + allowed, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: allowed}}
		if out := runEvent(t, root, pre); out != "" {
			t.Fatalf("ordinary command denied: %q -> %s", allowed, out)
		}
	}
}

// Review finding F2: a staged gate list replaced by progressive selection
// may run fewer gates, so it is narrowing.
func TestStagedToProgressiveSelectionIsNarrowing(t *testing.T) {
	root := greenFieldRoot(t)
	armUnder(t, root, `{"design":"design","gates":"g2,g3,gx"}`, "s", "toolu_1")
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design"}`)
	out := stopOutput(t, root, "s")
	if !strings.Contains(out, "became progressive selection") {
		t.Fatalf("staged to progressive was not treated as narrowing: %s", out)
	}
	requireArmed(t, root, 0, "staged to progressive")
}

// Review finding F3: when the configuration moves the design tree between
// PreToolUse and PostToolUse, the completion still closes its own token.
func TestCompletionClosesItsTokenAfterTheDesignTreeMoved(t *testing.T) {
	root := greenFieldRoot(t)
	copyTree(t, crmDesign, filepath.Join(root, "blueprint"))
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"design"}`)
	target := filepath.Join(root, "design", "notes.txt")
	pre := editEvent("PreToolUse", "Write", "s", target)
	pre.ToolUseID = "toolu_write"
	if out := runEvent(t, root, pre); out != "" {
		t.Fatal(out)
	}
	writeFile(t, filepath.Join(root, ConfigName), `{"design":"blueprint"}`)
	post := pre
	post.HookEventName = "PostToolUse"
	if out := runEvent(t, root, post); out != "" {
		t.Fatal(out)
	}
	requireArmed(t, root, 0, "completion after the design tree moved")
	if out := stopOutput(t, root, "s"); strings.Contains(out, "in-flight tool") {
		t.Fatalf("own completed call still blocks: %s", out)
	}
}

// Review finding N1/N2 impact: a boundary event forged for the agent's own
// session (through any shell spelling a text guard misses) must not let a
// SubagentStop discharge while the main lane's call may still be writing.
// Only the owner's main-thread Stop, which the host fires after every
// foreground call has resolved, may discharge over an ended token.
func TestForgedOwnSessionEndLetsOnlyTheMainThreadStopDischarge(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "s", "toolu_main_writer")
	if out, err := runHookPayload(t, root, map[string]any{"hook_event_name": "SessionEnd", "session_id": "s", "cwd": root}); err != nil || out != "" {
		t.Fatalf("%s %v", out, err)
	}
	sub := runEvent(t, root, Input{SessionID: "s", AgentID: "worker", Cwd: root, HookEventName: "SubagentStop"})
	if strings.Contains(sub, `"decision":"block"`) {
		t.Fatalf("an ended token must not block: %s", sub)
	}
	requireArmed(t, root, 1, "a subagent Stop cannot discharge over the main lane's ended token")
	if out := stopOutput(t, root, "s"); out != "" {
		t.Fatalf("main-thread Stop: %s", out)
	}
	if state, err := readStateRecord(root, "s"); err != nil || state.design {
		t.Fatalf("main-thread Stop must discharge after green gates: %+v %v", state, err)
	}
}

// Parser differential: the guard must read a command the way the shell joins
// it (quotes and backslashes removed before word splitting), across case,
// whitespace, aliases of the binary path, and redirections.
func TestOperatorGuardReadsWordsAsTheShellJoinsThem(t *testing.T) {
	root := greenFieldRoot(t)
	for _, command := range []string{
		`mach''inery hook`,
		`machinery ho''ok`,
		`machin\ery hook`,
		`"machinery" "hook"`,
		`MACHINERY HOOK`,
		"machinery\thook",
		"machinery\n hook",
		`./bin/../bin/machinery hook`,
		`$HOME/.local/bin/machinery hook`,
		`machinery-hook''.sh boundary`,
		`machinery hook-st''ate release --orphaned`,
		`machinery hook\-state adopt`,
		`ca''t ~/.config/machinery-hook-st"a"te-x/y.state`,
		"`machinery hook`",
		`x=$(machinery hook)`,
	} {
		pre := Input{SessionID: "agent", ToolUseID: "d-" + command, Cwd: root, HookEventName: "PreToolUse", ToolName: "Bash", ToolInput: toolInput{Command: command}}
		if out := runEvent(t, root, pre); !strings.Contains(out, `"permissionDecision":"deny"`) {
			t.Fatalf("guard read %q differently from the shell: %s", command, out)
		}
	}
}

// Parser differential: a boundary payload with duplicate keys, case-folded
// aliases of a routing key, or trailing content is refused, never routed.
func TestBoundaryPayloadAmbiguityIsRefused(t *testing.T) {
	root := greenFieldRoot(t)
	armShell(t, root, "victim", "toolu_v")
	for _, raw := range []string{
		`{"hook_event_name":"SessionEnd","session_id":"attacker","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"}`,
		`{"hook_event_name":"SessionEnd","session_id":"attacker","SESSION_ID":"victim","cwd":"` + filepath.ToSlash(root) + `"}`,
		`{"hook_event_name":"Stop","Hook_Event_Name":"SessionEnd","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"}`,
		`{"hook_event_name":"SessionEnd","hook_event_name":"SessionEnd","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"}`,
		`{"hook_event_name":"SessionEnd","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"} {}`,
		`{"hook_event_name":"SessionEnd","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"`,
		`{"hook_event_name":"SessionEnd","hook_event_name":"Stop","session_id":"victim","cwd":"` + filepath.ToSlash(root) + `"}`,
		`{"hook_event_name":"SessionEnd","session_id":7,"cwd":"` + filepath.ToSlash(root) + `"}`,
	} {
		var out bytes.Buffer
		if err := Run(strings.NewReader(raw), &out, root); err == nil {
			t.Fatalf("ambiguous payload was routed: %s -> %s", raw, out.String())
		}
	}
	requireLive(t, root, 1, "no ambiguous payload marked the victim's token")
}

// Single source of truth: the owner a PreToolUse records and the owner a
// boundary or Stop computes come from the same decoded session and agent
// ids, including non-ASCII and escaped spellings of the same id.
func TestOwnerDigestsAgreeAcrossDecoders(t *testing.T) {
	root := greenFieldRoot(t)
	session := "sessión-☃"
	armShell(t, root, session, "toolu_u")
	raw := `{"hook_event_name":"UserPromptSubmit","session_id":"sessión-☃","cwd":"` + filepath.ToSlash(root) + `","user_prompt":"x"}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(raw), &out, root); err != nil || out.Len() != 0 {
		t.Fatalf("%s %v", out.String(), err)
	}
	requireLive(t, root, 0, "an escaped spelling of the same session id is the same owner")
	if out := stopOutput(t, root, "sessiÓn-☃"); !strings.Contains(out, "stays armed") {
		t.Fatalf("a case-different session id is another session: %s", out)
	}
}
