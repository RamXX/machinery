---
id: MAC-ntvm
title: "Orphaned in-flight tool tokens after a killed session block every later Stop in the repo, with no recovery command"
status: open
priority: 1
type: bug
labels: [hook, governance, crash-safety, h2-origin]
created_at: 2026-10-06T15:31:42Z
created_by: ramirosalas
updated_at: 2026-10-06T23:12:16Z
content_hash: "sha256:1319c9e5da0c73c2a651961400eaf1468f3f4247c7521b773c81c653668895cc"
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
RESUME POINT 2026-10-06: fix work is on origin branch fix/hook-identity-device-reattach (e45350e0, 'release: prepare machinery 0.11.2', NOT tagged or published) plus uncommitted agent state at origin wip/hookfix-20261006 (843018a1). Remaining: close the last security-review finding (incomplete validation / parser differential in internal/hook/boundary.go), run make preflight and make ci-linux (Linux arm64 + amd64), then the consolidated v0.11.2 release and update verification (machinery update from v0.11.1 and fresh install, macOS and Linux, interrupted-update rollback, Claude plugin refresh). Full brief:
You are the machinery maintainer agent for this run. Work in ~/workspace/machinery (Go; public OSS repo RamXX/machinery). The owner has given an EXPLICIT GO for one consolidated release at the end of this run (this overrides the usual "commit locally only" rule for this run only). Work fully autonomously; nobody will answer questions. Write in the repo's existing style. Never use emojis or em dashes anywhere (code, docs, commits, changelog).
## Status 2026-10-06 (postponed by owner; nothing tagged or released)

Branch fix/hook-identity-device-reattach pushed at 73e4196c. NOTE: origin/main was fast-forwarded to e45350e0 ("release: prepare machinery 0.11.2", version sites at 0.11.2) before the release was blocked; there is NO v0.11.2 tag or GitHub release. main does not yet carry 73e4196c. PR #21 is open (hosted ci/formal/security all green on e45350e0).

Done (all with tests; field regressions fail on v0.11.1 and pass now, on macOS and linux/arm64):
- MAC-xqu7: device number ignored on every Unix; identity qualifiers must match when both present; adopt --rebind-identity (generation must match marker, prints old -> new); doctor names only a recovery that can succeed.
- Tokens record owner session+lane digests. Own tokens block; other sessions' tokens are orphaned (non-blocking, withhold discharge); owner-less (legacy/session-less) tokens block every Stop until operator release.
- Boundary hooks UserPromptSubmit / PostToolBatch (exact listed ids) / Interrupt / SessionEnd mark tokens ended; only the owner's main-thread Stop may discharge over an ended main-lane token; never block a prompt.
- Route change: gates re-run under current .machinery.json; narrowing (design/impl tree change, strict dropped, staged gates removed or replaced by progressive, no stop-time gate, uncomparable route) never discharges without 'hook-state release --routes'.
- hook-state release (--token/--orphaned/--routes) and adopt are operator-only (agent env markers + TTY gate), journaled, never clear obligations. PreToolUse denies agent access to machinery hook / hook-state / the store (canonical paths, shell-joined words, unresolved expansions mentioning the binary).
- Strict boundary JSON (duplicate/case-variant keys refused; routing fields re-decoded by the enforcing decoder).
- Docs/CHANGELOG [0.11.2] written.
- Evidence: macOS make preflight green on e45350e0 ("preflight OK: all required local CI/formal gates passed"); hosted CI green on e45350e0 (PR #21); linux/arm64 container race suite: internal/hook, internal/install, cmd/machinery ok (runtimeclosure/tdd/integration-lane fail only because linux/arm64 is not a pinned native assurance platform).

Open:
1. Security review findings on the latest pushed commits, reported by the coordinator's automated reviewer: (a) parser-differential / guard bypass in internal/hook/boundary.go (the PreToolUse text guard over shell commands); (b) path-validation-differential in internal/hook/hook.go. 73e4196c addresses both (canonical paths everywhere, unresolved-expansion deny) but has NOT been re-reviewed. Inherent residual: no text guard over a shell is complete (e.g. a name assembled from variables with no 'machin' substring, or a script written by a file tool then executed); the semantic bound is the host Stop contract plus PostToolUse re-arm, and CI.
2. 73e4196c not yet verified by full make preflight, hosted CI, or Linux runs.
3. Release not cut: no tag, no GitHub release, no update verification (macOS update from v0.11.1, Claude plugin refresh, Linux install.sh v0.11.1 -> update, fresh install, interrupted-update rollback). Linux verify script drafted (not in repo).
4. Close MAC-xqu7 and MAC-ntvm after release.

Next steps:
1. Get the reviewer's verdict on 73e4196c; fix anything left with tests; commit + push each step.
2. make preflight (macOS, run alone: concurrent heavy load caused one timing flake) and Linux: hosted CI via PR #21 (linux/amd64 race + integration) plus a linux/arm64 container race run with --init and the repo copied into the container (Docker Desktop bind-mounted TMPDIR breaks identity-sensitive tests; make ci-linux on this Mac is not usable as-is).
3. Fast-forward main to the final commit, push, wait for ci/formal/security push runs on that exact SHA, tag v0.11.2, push tag (release.yml publishes).
4. Verify updates on macOS and Linux, then close MAC-xqu7 and MAC-ntvm and nd sync.

# Goal
Fix the governance hook's two field-blocking bugs, prove the fixes on macOS AND Linux, and ship ONE patch release (next version after v0.11.1, i.e. v0.11.2 unless the repo's release policy says otherwise) whose update path works flawlessly on both macOS and Linux. The owner will test the update themselves on both platforms afterwards.

Read first: `nd show MAC-xqu7`, `nd show MAC-ntvm` (full field reports, reproductions and expected behaviour), docs/claude-plugin.md (hook contract), internal/hook/, docs/release-notes.md, release-policy.json, the Makefile (preflight, release targets), CHANGELOG.md, install.sh and the `machinery update` implementation.

# Starting point
Branch fix/hook-identity-device-reattach (local only) has commit 6553172b: on every Unix, sameHookNativeIdentity treats an equal inode as the same store directory (macOS already did), so a Linux block volume that reattaches with a new device number no longer bricks the store. Review it critically, keep it if sound, and finish it (MAC-xqu7).

# Required fixes
1. MAC-xqu7: as above, plus `machinery hook-state adopt` must be able to rebind an operator-verified store (generation matches the independent marker) to a new native identity, printing old and new identity. `doctor` must never prescribe a command that is guaranteed to refuse.
2. MAC-ntvm, stranded in-flight tokens. Field triggers seen 2026-10-06: (a) host process SIGKILLed mid tool call (pod eviction); (b) the user pressing Esc to interrupt while a tool runs (one stranded token observed after Esc then /exit); (c) the plugin disabled and re-enabled across sessions (13 stranded tokens on a Mac); (d) obligations recorded under two different routing configurations after binary/plugin upgrades or .machinery.json changes, so Stop refuses forever with "dirty obligation was armed under a different routing configuration". Required behaviour:
   - A Stop must not be blocked forever by tokens it can never close. Tokens belong to the session that armed them (the immutable session route is already recorded); a different or dead session's tokens are reported as orphaned and do NOT block, while the project design/impl obligation STAYS ARMED so the gates still run at discharge. Same-session in-flight tokens still block, as today.
   - Determine, by reading Claude Code's documented hook events (and Codex/OpenCode adapters in this repo), what is emitted on a user interrupt; if no PostToolUse/PostToolUseFailure arrives, handle the interrupt so the token does not strand.
   - An obligation armed under an older routing configuration must be re-evaluated under the CURRENT configuration (re-run the gates) instead of refusing forever; never discharge it without running the gates.
   - Add an audited operator command, e.g. `machinery hook-state release --root <root> (--token <id> | --orphaned)`, that records who/when/which tokens in the ledger and keeps the obligation armed.
   - `machinery doctor` reports stranded tokens and foreign-route obligations per project, with the exact recovery command.
3. Security posture: none of this may let an agent silently skip `machinery check` or forge a store. Explain the threat-model trade-offs in the changelog entry and in docs/claude-plugin.md.

# Tests (TDD; real behaviour, no weakening of existing tests)
- Regression tests reproducing each field case: device renumbering on Linux; SIGKILL mid tool call followed by a new session; interrupt without PostToolUse; plugin toggling; route change after an armed obligation. Each must fail on v0.11.1 behaviour and pass after the fix.
- Run the FULL test suite and `make preflight` (or the repo's CI-equivalent) on macOS (this machine) AND on Linux amd64. For Linux use a container (`docker run --rm -v "$PWD":/src -w /src golang:<the go.mod version> ...`) or the repo's Dagger setup if it has one. Both must be green. Paste the summary lines into the release notes evidence.

# Release (only after everything above is green on both platforms)
- Follow the repo's documented release process exactly (release-policy.json, docs/release-notes.md, Makefile release targets, tags, GitHub release workflow, checksums, plugin/marketplace version bumps for Claude Code and Codex). One version bump, one CHANGELOG entry covering everything. Merge to main, tag, push, publish.
- Verify the published release end to end on BOTH platforms before you finish:
  - macOS (this machine): `machinery update` from the installed v0.11.1 to the new version; `machinery version`, `machinery doctor` clean; the Claude Code plugin refreshes to the new version (`claude plugin marketplace update machinery`, `claude plugin update machinery@machinery`).
  - Linux amd64: in a fresh container, install v0.11.1 via install.sh, then `machinery update` to the new version, then `machinery doctor`; also a fresh install of the new version directly. Both must succeed with no manual steps.
  - The install receipt, rollback journal and checksums must verify; a failed or interrupted update must roll back cleanly (test it).
- If anything in the update path is not flawless, fix it before publishing; if already published, ship the fix in the same release cycle only if the policy allows, otherwise stop and report precisely.
- Close MAC-xqu7 and MAC-ntvm with the release version and evidence; `nd sync`.

# Do not
- Do not touch any repository other than ~/workspace/machinery (H2 and the dev box are out of scope; the owner will update them).
- Do not use `--no-verify`, skip hooks, or weaken tests or gates.

# Final report (stdout, short, plain text)
Version released, tag and commit, what changed (one line per fix), test evidence per platform (suite and preflight summary lines), update verification results per platform, anything the owner must do or know, and anything that failed.

## History


## Links
- Related: [[MAC-xqu7]]

## Comments
