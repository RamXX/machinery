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
updated_at: 2026-09-25T19:53:32Z
content_hash: "sha256:70a3925819c5ee8929f6844c3085f334b3162c162633a90e3fce6128daff87f6"
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


## History


## Links
- Parent: [[MAC-9dai]]

## Comments
