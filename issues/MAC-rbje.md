---
id: MAC-rbje
title: "Deny agent writes to the wave sentinel (human-only .machinery-wave)"
status: in_progress
priority: 1
type: feature
created_at: 2026-08-30T08:34:37Z
created_by: ramirosalas
updated_at: 2026-08-30T08:50:33Z
content_hash: "sha256:c7e1f34e224631d7f5e91e5621e2da7354b90fead69a913c2d0415762c98fe84"
assignee: ramirosalas
labels: [delivered]
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
REDELIVERY (rework after PM rejection). Commit 9c81aa6 on story/MAC-rbje-wave-sentinel (parent c594b09). Not merged, not pushed.

ROOT CAUSE (as the PM found it): pre() built a `dropped` set from deletedPaths(in) and skipped the wave-sentinel deny for ANY editedPaths entry whose rel was in that set. editedPaths() deduplicates by path and its regex matched Add|Update|Delete alike, so one apply_patch carrying both '*** Delete File: design/.machinery-wave' and '*** Add File: design/.machinery-wave' collapsed to a single entry that the delete had already exempted. Result: no deny, fresh full-TTL sentinel in one governed call.

FIX (per the prescribed shape): the exemption is now per operation, not per path.
- patchPathLine now captures the operation keyword (Add|Update|Delete) as well as the path.
- New editedOps(in) []editedPath returns (Path, Op) pairs, deduplicated by path AND op, in patch order; file tools report opWrite, move lines report opMove.
- pre() iterates editedOps and skips the sentinel deny only when that entry's own Op == opDelete. The `dropped` map is gone; the config/marker deletion deny loop is unchanged.
- editedPaths() is retained as the path-only, path-deduplicated view of editedOps, so the stop-time touched-class caller keeps its contract.

BYPASS-DENY EVIDENCE (end to end, built binaries, real PreToolUse JSON on stdin, temp managed repo with design/domain.modelith.yaml):
  patch = '*** Begin Patch\n*** Delete File: design/.machinery-wave\n*** Add File: design/.machinery-wave\n+240\n*** End Patch'
  binary built from c594b09 (pre-fix) -> empty stdout (ALLOW). Bypass reproduced.
  binary built from 9c81aa6 (fixed)   -> {"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"design/.machinery-wave is the wave sentinel, and it is operator-created: ..."}}
PURE-DELETE-ALLOWED EVIDENCE (same fixed binary):
  patch = '*** Begin Patch\n*** Delete File: design/.machinery-wave\n*** End Patch' -> empty stdout (ALLOW). Deleting still closes the wave, matching the stop hook's own remedy text at hook.go:424.

NEW COMMITTED TESTS (all PASS):
  TestCodexPatchWaveSentinelDeleteDoesNotLaunderRewrite, 4 subcases, each expecting deny: 'delete then add', 'add then delete', 'delete then update', 'delete one sentinel, add another'.
  TestEditedOpsReportsOperationPerPath: pins that one patch deleting and re-adding a path reports BOTH ops in patch order, that editedPaths stays deduplicated by path (2 paths from 3 ops), and that a file-tool write reports opWrite and never opDelete.
PINNED REGRESSIONS, unchanged and green: TestCodexPatchWaveSentinel (add denied, pure delete allowed), TestPreDeniesWaveSentinelWrites (8 subcases incl. 3 controls), TestPreDeniesGeneratedArtifacts (16 subcases), TestPreDeniesGovernanceConfigEdits, TestCodexDeleteOfGovernanceMarkerDenied, TestEditedPathsParsesCodexPatchOperations, TestCodexApplyPatchRecordsAllTouchedClasses, TestWaveSentinel, TestSessionStartAnnouncesGovernance.

TEST SUITE: 'go test ./...' at 9c81aa6, 16/16 packages ok, 0 failures, 0 skips:
  cmd/machinery, internal/alloy, internal/checker, internal/compose, internal/experiments, internal/formal, internal/gates, internal/hook, internal/install, internal/ir, internal/lint, internal/oracle, internal/pack, internal/refine, internal/tla, internal/version.
LINT: golangci-lint 2.13.2 (matches .golangci-version v2.13.2), 'golangci-lint run ./...' -> 0 issues. 'gofmt -l .' -> clean.
pvg gates -> GATES: PASS (37 warn, 2 skipped); all warns are pre-existing file_loc, lizard/jscpd skipped (not installed).
pvg verify internal/hook/hook.go internal/hook/hook_test.go --include-tests -> 6 'stub' hits, all 'return \"\"' in hook.go, identical count to c594b09 and f1dc685 (PM already adjudicated these as pre-existing not-applicable sentinel returns in generatedReason()/relToRoot()). No new ones introduced.

DOCS: added the per-operation clarification in the two places that state the delete carve-out, so the rule reads the same in code and prose: skills/machinery/SKILL.md wave-sentinel bullet and docs/claude-plugin.md PreToolUse row. SKILL.md frontmatter version untouched (TestPluginManifests green).
Non-goals still honored: TTL/cap/stop-block logic untouched; the Bash residual not addressed.
No em dashes or emojis: regex scan over every added diff line (U+2014 plus emoji blocks) found none.

AC verification at 9c81aa6:
  AC1 PreToolUse denies file-tool create/edit of .machinery-wave with a clear reason; existing denials unchanged -> MET, now including the delete+add-in-one-patch shape that the rejection identified.
  AC2 New hook tests cover deny + control; go test ./internal/hook/... green -> MET.
  AC3 make test and golangci green at repo root -> MET.
  AC4 Docs mention the human-only rule -> MET, and now the per-operation nuance too.
  AC5 No em dashes or emojis in added text -> MET.

LEARNINGS (full history across both rounds):
- The original round got the policy right and the parsing wrong. The bug was not in the deny rule but in the shape of the data it consumed: a set of paths cannot express 'this path was deleted AND re-added', so any exemption keyed on a path set is a blanket amnesty for every other operation on that path in the same call. Exemptions belong on operations.
- editedPaths() deduplicating by path actively hid the bypass: the delete and the add of the same file collapsed into one entry, so even reading pre() carefully the second operation was invisible. Deduplication upstream of a security decision is a trap; dedupe by the full key the decision uses.
- Protocol-level proof beat unit-level proof here. The new unit test cannot compile against the old code (it names editedOps), so it could not by itself demonstrate the regression. Building the binary at c594b09 and at 9c81aa6 and feeding both the same PreToolUse JSON showed allow-then-deny with no code coupling at all, which is the evidence that actually settles it.
- A carve-out documented as 'deleting it stays allowed' is under-specified prose. Saying 'allowed as a delete on its own' in SKILL.md and docs/claude-plugin.md costs one clause and removes the reading that let the bypass look legitimate.
- ENVIRONMENT: the dispatcher-provisioned worktree removed the concurrent-checkout hazard reported in round one. Worktree isolation worked; the whole ladder ran clean without touching the shared checkout.

## History
- 2026-08-30T08:35:01Z status: open -> in_progress
- 2026-08-30T08:35:01Z claimed by ramirosalas
- 2026-08-30T08:44:50Z status: in_progress -> open
- 2026-08-30T08:46:11Z status: open -> in_progress
- 2026-08-30T08:46:11Z claimed by ramirosalas

## Links


## Comments

### 2026-08-30T08:40:34Z ramirosalas
DELIVERED for acceptance review. Branch story/MAC-rbje-wave-sentinel at c594b09 (not merged, not pushed). Proof in Notes. Status left in_progress: this vault has no 'delivered' status (open/in_progress/blocked/deferred/closed only), so the 'delivered' label carries the signal.

### 2026-08-30T08:45:18Z ramirosalas
EXPECTED: PreToolUse denies every agent file-tool create/edit of any path whose basename is .machinery-wave (AC1), with deletion staying allowed only because it closes the wave and re-arms gating (the developer's documented judgment call, which I confirmed is sound: the stop-hook's own remedy text at hook.go:424 literally instructs "Delete <design>/.machinery-wave to close the wave and gate", so denying deletion would contradict the hook's own documented remedy, and deletion cannot be used to defer gating since it re-arms it immediately -- ACCEPTED as correct reasoning).

DELIVERED: internal/hook/hook.go pre() (lines ~184-210) builds a `dropped` map from deletedPaths(in) (paths that appear on a "*** Delete File:" line anywhere in an apply_patch/OpenCode patch), then for every path in editedPaths(in) skips the wave-sentinel deny whenever `dropped[rel]` is true for that path -- regardless of what kind of edit produced that editedPaths entry.

GAP: This exempts more than deletion. editedPaths() is built from patchPathLine, which matches "*** Add File:", "*** Update File:", AND "*** Delete File:" lines (hook.go:670). So a single apply_patch tool call containing BOTH a "*** Delete File: design/.machinery-wave" line and an "*** Add File: design/.machinery-wave" line (with a fresh TTL body) for the SAME path sails through with NO deny at all: the delete populates dropped[rel]=true, and the add's editedPaths entry for that same rel is then skipped because dropped[rel] is true. I reproduced this live in the worktree with a probe test (not committed, removed after verification): a patch shaped
  *** Begin Patch
  *** Delete File: design/.machinery-wave
  *** Add File: design/.machinery-wave
  +240
  *** End Patch
produces empty stdout (allow), confirmed by unmarshal failure / assertion in the probe. This is precisely the threat model the story exists to close: a single governed tool call lets an agent open a fresh wave sentinel (TTL up to 240) and defer stop-time gating indefinitely, using the exact tool (apply_patch) this story's own tests (TestCodexPatchWaveSentinel) exercise. AC1 ("PreToolUse denies file-tool create/edit of .machinery-wave with a clear reason") is not met for this input shape. Not a Bash-escape-class residual (that's a different, explicitly accepted class per the story's Non-goals) -- this is a bypass of the governed apply_patch tool itself.

FIX: Stop treating "this path was deleted somewhere in the patch" as license to allow every other edit to that path in the same patch. Track the operation type per editedPaths entry (Add/Update/Delete) instead of reusing the deletedPaths()-derived path set as a blanket exemption -- deny should be skipped only for the editedPaths entry that IS itself the delete operation on that path, never for a co-occurring Add/Update of the same path in the same patch. Concretely: extend editedPaths (or add a sibling helper) to report each match's operation kind from patchPathLine's own capture group (it already distinguishes Add|Update|Delete), and in pre() only exempt the wave-sentinel deny when the specific edited-path entry's operation is Delete. Add a regression test for the delete+add (and delete+update) combo in one apply_patch call targeting .machinery-wave, expecting deny. Re-run the full verification ladder (go test ./..., golangci-lint run ./...) after the fix and re-paste PROOF with the new test included.

Everything else in this delivery checked out: go test ./... (16/16 packages ok, matches proof), golangci-lint run ./... (0 issues) and gofmt -l . (clean) both reproduced independently; pvg gates PASS (only pre-existing file_loc WARNs, lizard/jscpd skipped); pvg verify's 6 "stub" hits on `return ""` in hook.go are pre-existing (identical count at f1dc685 and c594b09), not introduced by this diff, and are legitimate not-applicable sentinel returns in generatedReason()/relToRoot(), not incomplete implementation. Docs (README.md, docs/claude-plugin.md, skills/machinery/SKILL.md) all correctly describe the human-only rule; SKILL.md frontmatter version untouched. No em dashes or emojis found in any added line. Please rework the dropped/exemption logic per FIX above and redeliver.
