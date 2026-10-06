---
id: MAC-q1kf
title: "Revision-impact report: one repair's blast radius and design-only delivery drift"
status: open
priority: 3
type: feature
labels: [traceability, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:52Z
content_hash: "sha256:b2b60faf5233a60d02e30ee139d396e463e52c6d7d7eba95fe6bf4e655bee548"
was_blocked_by: [MAC-zti7]
---

## Description
Revision-impact report: bound one source repair's blast radius and flag design-only delivery drift

Problem: one source repair can require many downstream updates, making ordinary derivation look like many new defects; conversely, review loops can expand into later milestones without delivering the current slice. H2 records (design/DECISIONS.md and .claude/plans/m1-s5/reviews/rev18-domain-checkpoint.md, 2026-09-14) contain both genuine safety repairs and a redirected M4 authoring exploration during M1 cleanup. Nothing in machinery reports revision impact today.

Proposed fix: a revision-impact report built on the public traceability model. Classify the initiating change (discovered defect, owner decision, planned implementation detail, deferred expansion); enumerate affected invariants, contracts, generated artifacts, locked tests and sanctions, packets and milestone obligations; separate direct semantic changes from generated propagation; report introduced, retained and resolved findings without confusing raw findings with normalized ratchet groups. Show the shortest dependency-closed candidate increment and remaining gates where derivable; never invent a gate waiver or claim an independently committable subset because files can be separated.

Acceptance criteria:
1. A one-invariant repair with many generated changes reports one causal revision plus its impact.
2. Unchanged oracle ids whose contract meaning changed remain visible.
3. A later-milestone-only proposal is flagged for explicit scope disposition, not made a current blocker.
4. The report distinguishes source progress, locked RED, runnable GREEN and accepted/pushed delivery.
5. Existing baselines (design/ratchet.json), owner authority, locked-test sanctions and acceptance requirements are preserved.
6. Fixtures are sanitized synthetic reproductions; no private H2 runtime evidence, credentials or custody material.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Design and build the revision-impact report per the 6 ACs, with sanitized synthetic fixtures. Evidence: No revision-impact code anywhere: grep for 'revision-impact' and 'revision impact' across cmd, internal, docs, skills and CHANGELOG finds nothing. Notes: Listed blockedBy MAC-zti7 (readiness report) is a soft conceptual dependency (shared evidence-limits vocabulary), not shared code; it can start on the public traceability model alone. Treat as soft, not a hard blocker. Its blocks on MAC-b27n and MAC-f15w are likewise soft.

## History
- 2026-09-24T21:33:57Z dep_added: blocked_by MAC-zti7
- 2026-09-24T21:33:58Z dep_added: blocks MAC-b27n
- 2026-09-24T21:33:58Z dep_added: blocks MAC-f15w
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-b27n
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-f15w
- 2026-10-06T04:03:40Z dep_removed: was_blocked_by MAC-zti7

## Links
- Was blocked by: [[MAC-zti7]]

## Comments
