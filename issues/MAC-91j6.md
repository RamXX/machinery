---
id: MAC-91j6
title: "machinery check --complete --milestone <id>: a final-handoff check a seal can pass"
status: open
priority: 2
type: feature
labels: [check, gates, h2, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:50Z
content_hash: "sha256:a3402129a141120e9d72244be4739d6929c576c9527d7d6cb3a0c082e1f3062d"
---

## Description
machinery check --complete --milestone <id>: a final-handoff check a milestone seal can pass

Problem (H2): the project's protocol runs `machinery check --complete` at every seal. On a ten-milestone design with two closed it prints nine `G!-complete` ERRORs (the open milestones) plus the whole-design Gt findings the ratchet accounts for, exits 1, and the count (479 at the M2 seal) means nothing. A seal has no check that can be green.

Evidence: `check` has `--complete` (cmd/machinery/check.go:31, "require all phase artifacts, --impl, closed milestones, and zero warnings") and no `--milestone` scope.

Proposed fix: `--complete --milestone <id>` asks the final-handoff question of one milestone: its acceptance bound, its DoD ids bound, its shards' claims attested current, no warnings inside its scope. Findings outside the milestone scope are not reported (or are reported as informational). Whole-design `--complete` keeps its meaning for the last milestone.

Acceptance criteria:
1. On a fixture with several milestones, one closed and fully bound, `check --complete --milestone <closed>` exits 0 with 0 blocking.
2. The same with an unbound DoD id or a stale shard claim inside that milestone fails naming it.
3. A warning inside the milestone scope fails; a warning outside it does not.
4. `--milestone` naming an unknown milestone fails loudly.
5. `check --complete` without `--milestone` is unchanged.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add --milestone scope to --complete as specified in the ACs. Evidence: cmd/machinery/check.go:33 defines --complete with no --milestone flag; internal/gates/complete.go:61-79 requires every declared milestone closed. 0.11.0 --landing (check.go:32) skips checkpoint gates but is a different question and is refused with --complete (check.go:51). Notes: Partly relieved by --landing in 0.11.0 for per-landing runs, but a seal-level check that can be green is still absent.

## History


## Links


## Comments
