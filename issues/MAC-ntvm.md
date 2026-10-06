---
id: MAC-ntvm
title: "Orphaned in-flight tool tokens after a killed session block every later Stop in the repo, with no recovery command"
status: open
priority: 1
type: bug
labels: [hook, governance, crash-safety, h2-origin]
created_at: 2026-10-06T15:31:42Z
created_by: ramirosalas
updated_at: 2026-10-06T16:07:35Z
content_hash: "sha256:3d811828926f7454ab875170a337ae375f70f945b1ab3a29b1f9c87819539cab"
related: [MAC-xqu7]
---

## Description
## Summary
PreToolUse durably records a `pending <tool_use_id>` token in the PROJECT-wide state record; only the matching PostToolUse/PostToolUseFailure removes it, and Stop refuses to discharge while any token remains (`internal/hook/hook.go` stop(), "in-flight tool operation(s)"). If the host process is killed between PreToolUse and PostToolUse (SIGKILL, OOM, pod eviction, power loss, laptop sleep that kills the session), the token can never be closed: the session that owned it is gone. Every later session in that repository then blocks at every Stop, forever. There is no operator command to release an orphaned token.

## Field reproduction (H2 dev box, 2026-10-06, machinery v0.11.1 + local fix for MAC-xqu7)
- The conductor's Claude Code session was SIGKILLed by a Kubernetes eviction at 07:28 while tool calls were in flight.
- The project record (`~/.config/machinery-hook-state-<id>/<scope>.state`) still holds `pending 09bd6073...` and `pending 102b4634...` with `design` and `impl` armed.
- A fresh session runs one read-only `git log`; its own token closes, but Stop blocks with "machinery governance has 2 in-flight tool operation(s) whose PostToolUse completion or host denial was not durably recorded" on every turn, and Claude Code loops to its stop-hook cap.
- `machinery hook-state adopt` preserves pending tokens by design, so it cannot help.

## Expected
Crash safety must not become a permanent lockout. Options, in order of preference:
1. Bind each pending token to its session (the token already carries the immutable session route). A Stop in a NEW session should not be blocked by another session's token; it should instead report the orphan and keep the project obligation armed (so the gates still run at the next discharge).
2. Provide an operator command, e.g. `machinery hook-state release --root <root> --token <id>|--orphaned`, that records the release (who, when, which tokens) in the ledger and keeps the design/impl obligation armed so the gates still run.
3. Detect liveness: a token whose owning session's host process is gone can be marked orphaned.
In every option the obligation must remain armed: releasing a token must never skip `machinery check`.

## Related interaction worth documenting
With the obligation armed, Stop runs the full `machinery check` at every turn end. Repositories that intentionally run red between batched checkpoints (H2's cadence) are then blocked on every conductor turn unless a human writes `.machinery-wave` = `open`. The docs should say this plainly next to the wave sentinel, because it is the first thing a long-running agent hits after any crash.

## Is machinery usable by others with this bug?
Usable, with a sharp edge: any crash, kill or eviction during a tool call permanently blocks the repository's sessions until someone hand-edits the hook state. Desktop users rarely kill a session mid-tool-call; unattended agents in pods, CI runners and laptops that sleep hit it routinely. Same severity class as MAC-xqu7 for unattended use.

## Acceptance Criteria


## Design


## Notes
Second field case 2026-10-06 (owner laptop, macOS, machinery v0.11.1, plugin re-enabled after being disabled): the H2 project ledger (revision 1159) holds 13 pending tokens from earlier sessions AND two route bindings (1dee6b9c..., b38873d4...). Every Stop in the repo now blocks with 'dirty obligation was armed under a different routing configuration; refusing to clear it using fallback or changed configuration'. Cause chain: plugin disable/enable cycles and binary upgrades leave PreToolUse tokens without PostToolUse, and a route recorded under an older binary/config can never match again. Needs the same operator recovery (audited release/rebind that keeps the obligation armed and re-runs the gates under the current route). Also: toggling the plugin off mid-project should not strand tokens silently; doctor should report stranded tokens and foreign routes per project.

## History


## Links
- Related: [[MAC-xqu7]]

## Comments
