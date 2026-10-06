---
id: MAC-q1kf
title: "Revision-impact report: one repair's blast radius and design-only delivery drift"
status: open
priority: 3
type: feature
labels: [traceability, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:d3ed9fa67c0124a4fc663ca0fab297c07fcde6fc96300855e5e1c2cc9c02b3a1"
blocked_by: [MAC-zti7]
blocks: [MAC-f15w]
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


## History
- 2026-09-24T21:33:57Z dep_added: blocked_by MAC-zti7
- 2026-09-24T21:33:58Z dep_added: blocks MAC-b27n
- 2026-09-24T21:33:58Z dep_added: blocks MAC-f15w
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-b27n

## Links
- Blocks: [[MAC-f15w]]
- Blocked by: [[MAC-zti7]]

## Comments
