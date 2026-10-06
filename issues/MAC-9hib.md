---
id: MAC-9hib
title: "OpenCode v2 governance adapter: machinery.js fails to load under OpenCode 2.x, governance silently off"
status: closed
priority: 1
type: bug
labels: [opencode, adapter, governance, hook]
created_at: 2026-09-28T15:19:06Z
created_by: ramirosalas
updated_at: 2026-10-06T05:53:39Z
content_hash: "sha256:131c6dfd6d5369331098fe3f6c2a256b4e66e29cf8c63dad9a1da0f0c986aa09"
closed_at: 2026-10-06T05:53:39Z
close_reason: "Fixed in 0.11.1 (5c8b5873, a9e9c24c RED, bafd5018 GREEN): v2 plugin entrypoint, durable Stop, session location; doctor reports incompatible adapters. Proven against a real OpenCode 2.0.18."
---

## Description
## Problem

OpenCode 2.x (installed via `brew install anomalyco/tap/opencode-v2`, tested on 2.0.18) does not load V1 plugins. The shipped governance adapter `adapters/opencode/plugins/machinery.js` only exports the V1 shape (`export const MachineryPlugin = async ({client, directory, worktree}, options) => ({ "tool.execute.before", "tool.execute.after", event })`). OpenCode v2 rejects it at load:

```
level=WARN message="failed to load plugin" target=~/.config/opencode/plugins/machinery.js
cause=PluginModule.LoadError: Plugin must export a default definition with an id and an effect or setup function. (SchemaError(Missing key at ["default"]))
```

Consequence: under OpenCode v2 machinery governance is SILENTLY OFF. OpenCode keeps running; PreToolUse / PostToolUse / PostToolUseFailure / Stop never reach `machinery hook`. This fails open, which violates the adapter's own fail-closed contract.

A local, untested patch is currently applied to the INSTALLED copy only (`~/.config/opencode/plugins/machinery.js`, appended `export default { id: "machinery.governance", setup(ctx) {...} }`). The next `machinery install --target opencode` overwrites it with the V1-only repo version and governance goes dark again. Use that patch as a reference, not as the answer. Pristine pre-patch copy: `~/opencode-v1-backup-20260928/plugins/machinery.js` (identical to repo HEAD at time of writing).

## V2 plugin contract (verified against anomalyco/opencode tag v2.0.18)

Authoritative sources (fetch with `gh api repos/anomalyco/opencode/contents/<path>?ref=v2.0.18`):
- Migration guide: `services/www/src/docs/content/build/plugins/migrate-v1.mdx`
- Plugin API guide: `services/www/src/docs/content/build/plugins/index.mdx` (sections Hooks > Permissions / Tools, Events)
- Loader / schema: `packages/core/src/plugin/module.ts` (Module schema: default = {id, effect} | {id, setup})
- Promise API types: `packages/plugin/src/promise/{plugin,tool,session,permission,registration,event}.ts`
- Promise adapter: `packages/plugin/src/promise/adapter.ts` (wraps callbacks in Effect.promise)
- Tool hook failure channel: `packages/core/src/plugin/hooks.ts` ("Only tool execute.before may fail")
- Events: `packages/schema/src/event.ts` (payload envelope), `packages/schema/src/session-status-event.ts`, `packages/schema/src/permission.ts`
- Built-in tools: `packages/core/src/tool/plugin/{edit,write,patch,shell,grep,glob}.ts`

Facts:
1. Entry: `export default { id: "<stable id>", setup(ctx) { ...; return cleanup } }`. `Plugin.define` from `@opencode/plugin` is an identity helper; a plain object works and avoids a package dependency for a locally installed file. Named exports are ignored by v2.
2. Dual support from one file is documented: the default object may carry both `setup` (v2) and `server()` (v1 object entrypoint, OpenCode >= 1.18.29). Decide the supported v1 floor; if older v1 must still work, keep the named export too and verify v1 does not double-register (named export + default.server).
3. Context mapping: `directory` -> `ctx.location.directory`; `worktree` -> check `Location.Info` (the local patch uses `ctx.location.worktree ?? ctx.location.directory`, UNVERIFIED that `worktree` exists); options arg -> `ctx.options` (keep the injectable `runner` there for tests); `client.tui.showToast` / `client.app.log` have NO v2 equivalent for plugins (local patch uses console.warn -> server log). Find the v2 way to surface a user-visible warning, or document the loss.
4. Hooks: `await ctx.tool.hook("execute.before", (event) => ...)` and `"execute.after"`. ONE mutable event object, not (input, output).
   - execute.before event: `{ tool, sessionID, agent, messageID, id /* call id */, input }`. Throwing rejects the call (documented blocking path in migrate-v1.mdx). This is the PreToolUse deny path.
   - execute.after event: same ids + `input` + either `{status:"completed", result}` or `{status:"error", error: Tool.Error}`. `result = { output?, content?: string | Array<{type:"text",text}|{type:"file",...}>, metadata? }`; for grep the content is an array. After-hooks have NO failure channel (hooks.ts NoFailures): a throw becomes an Effect defect, not a clean rejection. PostToolUse "block" must therefore be surfaced by rewriting `event.result` (the local patch replaces content with the denial reason). Verify what the model actually sees (content vs output) for edit/write/patch/shell.
   - Permission rejection: the permission check runs INSIDE tool execution (edit.ts / patch.ts call permission with action "edit"), so a user reject should arrive as execute.after status "error". Also emitted as event `permission.replied`.
5. Tool IDs are lowercase and differ from v1: `edit`, `write`, `patch`, `shell` (NOT `bash`), `grep`, `glob`. Input fields: edit {path, oldString, newString, replaceAll}; write {path, content}; patch {patchText}; shell {command, workdir, timeout, background}. The existing `toolNames` map and `toolInput()` already cover `shell`, `patch`, `path`, `patchText`, but the test suite must pin the v2 names and field shapes.
6. Events: `for await (const event of ctx.event.subscribe({ signal }))`; abort from the cleanup returned by setup. Payload is on `event.data` (v1 used `event.properties`).
   - Idle: `session.status` with `data.status.type === "idle"` and `data.sessionID`. `session.idle` still exists but is marked deprecated; handle both without double-firing Stop (local patch: use session.idle only until a session.status has been seen).
   - `permission.asked` data: `{ id, sessionID, action, resources, source?: {type:"tool", messageID, id /* call id */} }`.
   - `permission.replied` data: `{ sessionID, requestID, reply: "once"|"always"|"reject" }` (v1 used permissionID/response).
   - `message.part.updated` tool-error path from v1: find the v2 equivalent or drop it if execute.after status "error" is sufficient; justify either way.
7. Stop semantics: v1 threw from the idle event to "fail the idle event". In v2 a subscription callback cannot fail anything. Decide the v2 behavior for a Stop `block` (options: warn only; or re-prompt the session via `ctx.session.prompt` / `ctx.session.synthetic` so the agent must address red checks). Document the choice in docs/agent-portability.md.
8. v2 runs one shared background service (`opencode service restart`) across projects; the plugin process cwd is NOT the project. Anything relying on process.cwd() is wrong; always use the event/session location. (Same bug hit the codebase-memory-mcp OpenCode plugin: fixed locally by passing cwd explicitly.)
9. Plugins hot-reload on file change under ~/.config/opencode/plugins; the first `opencode run` right after an edit can return empty output (reload race). Not a plugin bug, but relevant when writing e2e checks.

## Scope

- `adapters/opencode/plugins/machinery.js`: add the v2 default entrypoint reusing the existing transport (defaultRunner, runMachinery, responseProblem, denial, toolNames, toolInput). Keep the fail-closed guarantees (deadline, capture ceilings, SIGTERM->SIGKILL, unrecognized response blocks).
- `adapters/opencode/plugins/machinery.test.mjs`: extend with a v2 lane that drives `default.setup(ctx)` against a fake ctx exposing `tool.hook`, `event.subscribe` (async iterable), `location`, `options.runner`. Cover: before-deny throws; before-allow records pending; after-completed PostToolUse; after-block rewrites result; after-error -> PostToolUseFailure; permission.replied reject -> PostToolUseFailure once (idempotent with after-error); session.status idle -> Stop once; session.idle fallback without double Stop; cleanup aborts the subscription; v2 tool IDs (shell/patch) map to Bash/apply_patch.
- `internal/install/targets.go` + `cmd/machinery/repository_contract_test.go`: nothing may assume a V1-only plugin shape; confirm install still writes `plugins/machinery.js` (v2 discovers `~/.config/opencode/plugins/*.js`). Also verify v2 still loads the installed OpenCode agents (`~/.config/opencode/agents/*.md`) and commands (`commands/`); file a separate bug if not.
- `machinery doctor`: detect OpenCode major version and flag a V1-only adapter under v2 (the silent fail-open that caused this story).
- `docs/agent-portability.md` + CHANGELOG: document v2 support, the supported v1 floor, and Stop semantics under v2.

## Acceptance criteria

1. With OpenCode 2.0.18, `machinery install --target opencode` produces a plugin that loads with no "failed to load plugin" WARN in `~/.local/share/opencode/log/opencode.log`.
2. Real-binary e2e under OpenCode v2 (no mocks): in a governed repo, a write/edit/patch/shell that machinery denies is rejected before execution and the agent sees the denial reason; an allowed call runs and produces PostToolUse; a user-rejected permission produces exactly one PostToolUseFailure; going idle produces exactly one Stop.
3. The v2 test lane above passes, alongside the existing v1 lane (or the v1 lane is removed with an explicit, documented v1 support drop).
4. `machinery doctor` reports a clear error when the installed adapter cannot load under the installed OpenCode major version.
5. Re-running `machinery install --target opencode` over the local patch replaces it with the repo version and governance stays active (verify with AC2).

## Environment at time of filing (2026-09-28)

- OpenCode v2.0.18 (brew formula anomalyco/tap/opencode-v2; v1 formula anomalyco/tap/opencode 1.18.33 uninstalled; the two conflict).
- machinery v0.10.1.
- Backups of v1 config/state/plugins: `~/opencode-v1-backup-20260928/`.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add v2 default entrypoint, v2 test lane, real-binary e2e, doctor version/adapter check, docs/agent-portability.md and CHANGELOG. Evidence: adapters/opencode/plugins/machinery.js has only the V1 named export (line 309 export const MachineryPlugin); no default export, ctx.tool.hook or setup(); machinery.test.mjs has no v2 lane; cmd/machinery/diag.go (doctor) has no OpenCode version check; CHANGELOG OpenCode entries (0.8.x lines 644, 1200) are V1 only. Notes: Fails open (governance silently off) for any user on OpenCode 2.x, violating the adapter fail-closed contract. Large story; consider splitting doctor detection (cheap, closes silent failure first) from the adapter port.

## History
- 2026-10-06T05:53:39Z status: open -> closed

## Links


## Comments

### 2026-09-28T15:50:10Z ramirosalas
Idle-event finding (2026-09-28 smoke test, OpenCode 2.0.18): session.status and session.idle are ephemeral events (packages/schema/src/session-status-event.ts) and are NOT delivered to plugin ctx.event.subscribe() subscribers. A traced opencode run showed only durable events; the turn ends with session.execution.succeeded (also session.execution.failed / session.execution.interrupted, packages/schema/src/session-event.ts Execution namespace). Consequence: any v2 port that waits on session.status {idle} or session.idle never runs the Stop hook, so the Stop gate is silently off. The installed local patch now keys Stop on the three session.execution.* events; this is applied but NOT yet verified to reach machinery hook. Verified in the same run: PreToolUse and PostToolUse fire via ctx.tool.hook with the correct root. Also observed: plugin console.warn output does not appear in ~/.local/share/opencode/log/opencode.log, so the patch's warning path is invisible.
