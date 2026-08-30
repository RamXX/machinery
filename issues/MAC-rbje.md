---
id: MAC-rbje
title: "Deny agent writes to the wave sentinel (human-only .machinery-wave)"
status: open
priority: 1
type: feature
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:34:37Z
content_hash: "sha256:80a447eaafa2c8d4048078c7744e74e4f8f7e9bd70471fa0ae698c0125350bd0"
---

## Description
# Deny agent writes to the wave sentinel (human-only .machinery-wave)

## Context (all you need; verified at HEAD f1dc685)

The stop hook honors a wave sentinel: while `<design>/.machinery-wave` is younger than its TTL (first line = minutes, default 45, cap 240), red gates surface as a message instead of blocking the stop (internal/hook/hook.go:265-287 and :401-411). The 2026-08-30 audit found a bypass: nothing denies an agent writing or touching `.machinery-wave` (it is not in the generated-artifact deny list at hook.go:212-246), so an agent can re-touch the sentinel every turn and defer stop-blocking indefinitely at up to 240 minutes per touch.

Decision (user, 2026-08-30): the sentinel is human/operator-created only. Agent file-tool writes to it must be denied at PreToolUse, exactly like edits to ratchet.json / *.oracle.md / formal artifacts.

## Scope

- internal/hook/hook.go: add a case to the PreToolUse deny path (the `pre()` / `generatedReason` area, hook.go:212-246) denying create/edit/touch of any path whose basename is `.machinery-wave`, for all governed file tools. Deny reason must state the sentinel is operator-created and name the file, in the same register as the existing deny reasons.
- internal/hook/hook_test.go: table cases for deny on Write/Edit/apply_patch-style tools targeting `.machinery-wave` (root design dir and nested design dirs), and a non-denied control case.
- Docs: wherever the wave sentinel is documented (search SKILL.md and docs/ for "machinery-wave" / "wave sentinel"), add one sentence: the sentinel is created by the operator, not by agents; the hook denies agent writes to it.

## Non-goals

- Do NOT change TTL semantics, cap, or stop-block logic.
- The Bash escape (shell can still touch the file) is an accepted residual (hook.go:183 documents the class); do not try to close it.
- Do not touch skills/machinery/SKILL.md frontmatter `version:` (pinned by TestPluginManifests, hook_test.go:793-799).

## Acceptance criteria

1. PreToolUse denies file-tool create/edit of `.machinery-wave` with a clear reason; existing generated-artifact denials unchanged.
2. New hook tests cover deny + control; `go test ./internal/hook/...` green.
3. `make test` and `make lint` (golangci) green at repo root.
4. Docs mention the human-only rule wherever the sentinel is described.
5. No em dashes or emojis in any added text.

## Proof required on delivery

Paste the go test summary line(s) and lint result into the story notes.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
