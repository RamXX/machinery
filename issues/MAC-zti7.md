---
id: MAC-zti7
title: "Design readiness report with assumption/evidence inventory and explicit evidence limits"
status: open
priority: 2
type: feature
labels: [readiness, evidence, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:53Z
content_hash: "sha256:57cd6bac3fe863826c79a1b236c6e8fed3f7d3d9c06c3bd0578b5f865a9a592d"
blocks: [MAC-ik8q, MAC-b27n]
related: [MAC-wbxq]
---

## Description
Design readiness report with an assumption/evidence inventory and explicit evidence limits

Problem: structural consistency, control-flow model checking, semantic review, static test attribution and successful runtime verification can all read as "green" while proving different things. H2's restored-database key recovery problem survived a design that consistently named key destruction. Reviewers cannot see which external assumptions remain untested before calling a design build-ready. docs/consistency-layer-proposal.md section 7 leaves readiness (entry 9) outside the rule layer; nothing in the code implements it. Owner priority from the 2026-09-14 H2 lessons: 9 first, then 10/11, then 12.

Proposed fix: a documented, versioned assumption/evidence inventory linked to existing invariants, dependency contracts and BUILD obligations. Each load-bearing assumption has an owner, affected scope, falsifying scenario, required evidence layer, current subject-bound evidence (or an explicit unresolved state) and the handoff it must satisfy. Integrate with existing attestations (Class C, MAC-ug4h) and external checkers; do not create a second incompatible evidence framework. Add a readiness report that separately reports model consistency, semantic review, executable dependency proof and implementation acceptance. A bounded or selected gate run must name unchecked obligations; no blanket completeness claim from selected green checks. Machinery does not promise to discover every missing assumption.

Acceptance criteria:
1. A synthetic model passes structural and formal checks while the readiness report still refuses readiness for a declared, unproved restore-safety assumption.
2. A pure comparison fixture cannot satisfy a declared native restore experiment.
3. A changed dependency assumption or evidence subject invalidates the affected readiness claim only.
4. A selected-gate run lists its unchecked obligations and never reports overall readiness.
5. Legacy migration is explicit and does not silently reclassify historical evidence.
6. Fixtures are sanitized and synthetic; no private H2 runtime evidence is copied.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Design and build the versioned assumption/evidence inventory and a readiness report that never claims overall readiness from selected gates; reuse attestations (MAC-ug4h class C). Evidence: Verified at HEAD 8c620d8f (v0.11.0). grep -ril readiness over Go sources finds only an example test; docs/consistency-layer-proposal.md:708 still lists readiness (entry 9) as outside the rule layer; no assumption/evidence inventory or readiness report exists. 0.11.0 threat ledger (docs/threat-driven-verification.md) covers adversaries, not assumption evidence. Notes: Sizeable design item; it is the root of blocks MAC-kyh6, MAC-ik8q, MAC-q1kf, MAC-b27n (blocked edges are real). Overlap risk with the 0.11.0 threat ledger: reuse its evidence vocabulary rather than create a second framework.

## History
- 2026-09-24T21:33:57Z dep_added: blocks MAC-kyh6
- 2026-09-24T21:33:57Z dep_added: blocks MAC-ik8q
- 2026-09-24T21:33:57Z dep_added: blocks MAC-q1kf
- 2026-09-24T21:33:58Z dep_added: blocks MAC-b27n
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-kyh6
- 2026-10-06T04:03:40Z dep_removed: no_longer_blocks MAC-q1kf

## Links
- Blocks: [[MAC-ik8q]], [[MAC-b27n]]
- Related: [[MAC-wbxq]]

## Comments
