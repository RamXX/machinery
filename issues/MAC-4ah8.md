---
id: MAC-4ah8
title: "Ga requires executed e2e evidence at the reviewed commit for every declared behavior (phase 2: replayed via assurance)"
status: open
priority: 1
type: feature
labels: [e2e, ga, acceptance]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:ec741ae5a32e020091fcd8763d4f95ae268b472d76ffbb754cd5bb8e6c3e762f"
blocked_by: [MAC-p3pj, MAC-lioz]
blocks: [MAC-qax4]
related: [MAC-u4oo]
---

## Description
Acceptance (Ga) requires executed e2e evidence. Today an acceptance entry binds a reviewer to a commit; nothing requires that the milestone's e2e behaviors ran and passed there.

Phase 1: each acceptance entry cites an e2e evidence record for the reviewed commit: the command, environment recipe id, per-behavior pass/fail with e2e test ids, dependency postures used, and a digest of the run log. Ga checks the record is bound to the reviewed commit, covers every declared e2e behavior of the milestone (pass, or residual with owner waiver), and that no behavior is silently missing.
Phase 2: the record is produced by machinery's executable assurance (runtime obligations of kind `runtime`, MAC-u4oo) instead of being submitted, so the pass is replayed, not claimed.

Acceptance criteria:
1. A milestone with a declared e2e behavior and no passing evidence at the reviewed commit fails Ga (Phase 1).
2. Evidence for a different commit, a missing behavior, or a failed behavior fails Ga naming it.
3. A residual behavior passes only with an owner waiver line in the acceptance entry, and the report lists it as unproven.
4. Output wording distinguishes submitted evidence (Phase 1) from replayed evidence (Phase 2).
5. Phase 2 tracked as a follow-up dependent on MAC-u4oo.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Define the e2e evidence record, Ga binding to the reviewed commit and declared behaviors, waiver line, submitted-vs-replayed wording; Phase 2 follow-up on MAC-u4oo. Evidence: No e2e evidence record in Ga: no E2E block or e2e evidence symbols in internal/gates; CHANGELOG 0.11.0 does not mention it. Acceptance still binds a reviewer to a commit only. Notes: Blockers are real: needs declared behaviors (p3pj) and postures (lioz, for dependency postures used in the record). Phase 2 should be filed as its own issue blocked by MAC-u4oo (AC5).

## History
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-p3pj
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-lioz
- 2026-09-25T19:53:33Z dep_added: blocks MAC-qax4

## Links
- Parent: [[MAC-9dai]]
- Blocks: [[MAC-qax4]]
- Blocked by: [[MAC-p3pj]], [[MAC-lioz]]
- Related: [[MAC-u4oo]]

## Comments
