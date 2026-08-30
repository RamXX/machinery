---
id: MAC-rbje
title: "Deny agent writes to the wave sentinel (human-only .machinery-wave)"
status: in_progress
priority: 1
type: feature
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:40:18Z
content_hash: "sha256:d37b563892d8ecb872ba29e6c83f4c9e9c02c8be16a32d5ddf8f8f1d5eab8326"
assignee: ramirosalas
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
PROOF (commit c594b09, branch story/MAC-rbje-wave-sentinel, built off f1dc685)

Tests (make test == go test ./...), all 16 packages ok:
  ok github.com/RamXX/machinery/cmd/machinery 7.320s
  ok github.com/RamXX/machinery/internal/alloy 0.634s
  ok github.com/RamXX/machinery/internal/checker 0.346s
  ok github.com/RamXX/machinery/internal/compose 0.792s
  ok github.com/RamXX/machinery/internal/experiments 2.502s
  ok github.com/RamXX/machinery/internal/formal 5.360s
  ok github.com/RamXX/machinery/internal/gates 4.292s
  ok github.com/RamXX/machinery/internal/hook 3.966s
  ok github.com/RamXX/machinery/internal/install 4.702s
  ok github.com/RamXX/machinery/internal/ir 2.311s
  ok github.com/RamXX/machinery/internal/lint 2.085s
  ok github.com/RamXX/machinery/internal/oracle 2.669s
  ok github.com/RamXX/machinery/internal/pack 3.071s
  ok github.com/RamXX/machinery/internal/refine 3.252s
  ok github.com/RamXX/machinery/internal/tla 2.541s
  ok github.com/RamXX/machinery/internal/version 3.466s
  0 failures, 0 skips.

Lint: golangci-lint 2.13.2 (matches .golangci-version v2.13.2), 'golangci-lint run ./...' -> 0 issues. gofmt -l . -> clean.

New tests (all PASS): TestPreDeniesWaveSentinelWrites (8 subcases: Write/Edit/MultiEdit on design/.machinery-wave, nested design/children/billing/.machinery-wave, ops/.machinery-wave outside the design dir, plus 3 controls: .machinery-wave.bak, wave.md, Bash tool) and TestCodexPatchWaveSentinel (apply_patch Add denied, apply_patch Delete allowed). Regression: TestPreDeniesGeneratedArtifacts unchanged and green (16 subcases), TestWaveSentinel and TestSessionStartAnnouncesGovernance green.

End-to-end through the built binary (go build ./cmd/machinery, real PreToolUse JSON on stdin, temp managed repo):
  Write design/.machinery-wave -> {"permissionDecision":"deny","permissionDecisionReason":"design/.machinery-wave is the wave sentinel, and it is operator-created: ..."}
  apply_patch Delete File: design/.machinery-wave -> no output (allowed; deleting closes the wave)
  Write design/machines/Deal.machine.json -> no output (allowed)

AC verification:
  AC1 PreToolUse denies file-tool create/edit of .machinery-wave with a clear reason; existing denials unchanged -> MET (hook.go pre(); TestPreDeniesGeneratedArtifacts unchanged and green)
  AC2 New hook tests cover deny + control; go test ./internal/hook/... green -> MET
  AC3 make test and golangci green at repo root -> MET
  AC4 Docs mention the human-only rule -> MET (README.md, docs/claude-plugin.md PreToolUse + Stop rows, skills/machinery/SKILL.md wave-sentinel bullet, and the SessionStart governance announcement in hook.go)
  AC5 No em dashes or emojis in added text -> MET (regex scan over added diff lines found none)

Non-goals honored: TTL/cap/stop-block logic untouched; Bash residual not addressed; SKILL.md frontmatter version untouched (TestPluginManifests green).

Design note: deletion of the sentinel stays ALLOWED. The stop message itself instructs 'Delete <design>/.machinery-wave to close the wave and gate', and deletion re-arms gating, so denying it would contradict the documented remedy. pre() collects deletedPaths first and skips the sentinel deny for a path the same patch deletes.

LEARNINGS:
- editedPaths() also matches '*** Delete File:' lines in an apply_patch, so a naive basename deny would have blocked wave CLOSE as well as wave open. Any new deny keyed on editedPaths needs to decide explicitly what it means for a delete.
- Denying by base name anywhere in the repo (not scoped under <design>/) is the right net here: child designs and decomposed packs each have their own design dir, and the sentinel is a fixed dotfile name with no legitimate agent-authored twin.
- A deny that is not announced at SessionStart reads as a tool malfunction to the agent that hits it. Adding the rule to the governance contract in sessionStart() cost one line and makes the wall self-documenting.
- ENVIRONMENT HAZARD: the shared checkout at /Users/ramirosalas/workspace/machinery had another agent's uncommitted, non-compiling edits (internal/gates/ledger.go, then internal/gates/ledger_test.go) appearing mid-session, and a second story branch story/MAC-v16q-prompt-dedup exists. My first 'go test' failed on THEIR broken build, not mine. I moved to a git worktree and restored the main checkout to 'main' with their edits intact. Concurrent developers on this repo need worktrees; the dispatcher should provision them.

## History
- 2026-08-30T08:35:01Z status: open -> in_progress
- 2026-08-30T08:35:01Z claimed by ramirosalas

## Links


## Comments
