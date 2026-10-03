---
id: MAC-2ei9
title: "Stop hook blocks every turn while background tasks run; defer instead of block"
status: open
priority: 1
type: bug
labels: [governance, hook, claude-code, stop]
created_at: 2026-10-03T19:42:12Z
created_by: ramirosalas
updated_at: 2026-10-03T19:42:12Z
content_hash: "sha256:ed2857fa132558709e8d422dd5b64cbe0f2854e4bf9dcb4f2b3fe6852c5cafbb"
---

## Description
## Problem

The Claude Code Stop/SubagentStop governance hook blocks every turn of a session that keeps long-lived background tasks (event listeners, background agents, long builds) while a governed project is open. `internal/hook/hook.go` `stop()`:

- with `len(state.pending) > 0` and `in.BackgroundTasks > 0` it returns `decision: block` ("sees N background task(s) still running; refusing to discharge or clear the project gate obligation");
- with touched design/impl state and `in.BackgroundTasks > 0` it returns the same block.

A background Bash call that never finishes inside the turn leaves its PreToolUse token pending, so the block repeats on every Stop until the host's stop-hook block cap overrides it (Claude Code 2.1.288: 9 consecutive blocks, then "overriding and ending turn"). An orchestrating session that waits on background work therefore burns up to 9 forced extra model turns per wake-up, indefinitely. The block cannot change the outcome: the background tasks are, by design, still running when the turn ends.

Observed 2026-10-03 with machinery v0.10.1 (plugin and binary), Claude Code 2.1.288, Linux amd64, in an orchestrator session holding an event-drain background Bash call plus several background agents.

## Expected

A Stop that arrives while background tasks run should DEFER, not block: allow the turn to end, keep every pending token and touched flag exactly as recorded (no discharge, no clear), and run the gates at the next Stop whose input reports no background tasks (or when the pending tokens are closed by their PostToolUse/PostToolUseFailure events). Fail-closed is preserved because nothing is discharged; only the forced re-prompt loop goes away. Consider also honoring `stop_hook_active` so a single deferral cannot loop.

## Interim local workaround (not the fix)

On one host the installed plugin shim `hooks/machinery-hook.sh` (cache copy, v0.10.1) is patched to exit 0 before invoking `machinery hook` when `hook_event_name` is Stop/SubagentStop and `background_tasks` is non-empty. The original is kept beside it as `machinery-hook.sh.orig`. A plugin update overwrites the patch. Use it as a behavioral reference only.

## Acceptance criteria

1. RED tests in `internal/hook` (public `runEvent` surface): Stop and SubagentStop with pending tokens and non-empty `background_tasks` return no block, and the state record (pending tokens, design/impl flags, route bindings) is byte-identical afterwards.
2. Same for touched design/impl state with no pending tokens.
3. A later Stop with no background tasks runs the gates and blocks or discharges exactly as today; existing stop tests unchanged.
4. Session-retention and pruning behavior unchanged for deferred sessions.
5. Docs: hook contract section in the README/skill states the deferral rule.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
