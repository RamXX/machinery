---
id: MAC-cup9
title: "Machinery designed in machinery: a self-design a zero-context agent can rebuild from, judged by machinery's own gates"
status: open
priority: 3
type: epic
labels: [self-design, dogfood, table-oracle, after-va30]
created_at: 2026-09-24T02:38:02Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:7108e273445315f5b7ce803d0941fdd5af14504858bbaea3cab28d61b25dc8a8"
blocked_by: [MAC-9azz]
related: [MAC-syos]
was_blocked_by: [MAC-va30]
---

## Description
Goal: describe machinery itself with Modelith, C4, XState machines, the relational layers and the consistency rules, hold the current implementation to that design, and only then have a zero-context coding agent produce a fresh implementation under hard TDD that passes the same gates. Owner ruling 2026-09-23: this workstream starts after MAC-va30.

Why: only the envelope of machinery is a state machine (the pipeline, the gate lifecycle, the hook's stop obligation, install and update, cache provisioning with its locks and witnesses). Parsers, hashing, custody checks, generators and the fact projection are contracted units with no oracle a test can bind to. The v0.10.0 regressions (MAC-j39j, MAC-bsdq, MAC-i5j8) lived exactly there: the event-table contract and the group grammar were assumptions in briefs, not rows with oracles. A self-design forces those contracts into tables the gates read.

Acceptance for the epic: (1) design/ at the machinery repo root passes the full gate suite with --impl . (2) every machine-shaped subsystem has a machine, a matrix, generated oracles and TLC proofs; every non-machine unit has a matrix row bound to a table oracle. (3) the bundled examples' golden corpus, the fact projections and the Soufflé parity harness are cited as table oracles from the design, so Gt holds them. (4) a zero-context agent, given only the design and the packets, produces an implementation that passes Gt, Gk, the golden corpus byte for byte on the bundled examples, and rules parity, without reading the current source. (5) the design finds at least one real defect in the current implementation before the rebuild starts, or documents that it found none.

Stories, in order:
1. Table oracles: a declared fixture set (input, expected output, content-hashed) that a matrix row cites and Gt binds like transition ids; Gw packets carry them; Gv covers them. First story, blocks the rest.
2. Phase 0 and 1 for machinery itself: the domain model (designs, artifacts, gates, findings, attestations, receipts, caches, locks) with invariants and scenarios; Gc and Gl green.
3. C4 for machinery: components (CLI, gates, projection, rules, evaluator, generators, hook, installer, provisioners), the Architecture Contract with dependency rules mirroring the Go package boundaries, reads: rows for every design artifact the code parses (the skill, the schemas), G2 and G4 green against the current tree.
4. Machines for the envelope: pipeline and gate lifecycle, the stop hook obligation, install and update, provisioning with the provision lock and the stage witness; oracles, TLC, matrices with CLAUSES; G3, Gd, Gx green.
5. Relational layers for machinery: which gate may write which artifact (policy), one owner per generated artifact and per gate (integrity), custody boundaries (isolation); Alloy via verify-formal.
6. Consistency rules for machinery: every gate has one owner, every generated artifact has a regeneration command, every finding class has a fixture, every declaration group has a parser and a projection; shipped as rule files under rules/self/.
7. Hold the current code to the design: run everything with --impl ., record findings, fix code or design, record the defects the design found (acceptance criterion 5).
8. BUILD.md and slices.yaml for a rebuild; packets under budget; Gb and Gw green.
9. The clean-room rebuild: a zero-context agent in a fresh repository from the packets only, hard TDD, judged by criterion 4; keep the rebuild under experiments/ like clean-room-crm.
10. Retrospective: what the design missed, what the rebuild could not reproduce, what the table oracle needs next.

## Acceptance Criteria


## Design


## Notes
Ordering: starts after MAC-va30 (owner ruling 2026-09-23). Story 1 filed as a child; stories 2 to 10 are enumerated in the epic body and get filed when story 1 lands.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Entire self-design workstream; stories 2-10 to be filed after story 1 lands. Evidence: No design/ at repo root; no self-design artifacts or rules/self/. Epic body states stories 2-10 not yet filed; only story 1 (MAC-9azz) exists. Notes: Epic still makes sense as dogfooding. Child: MAC-9azz. The va30 blocker is an owner scheduling ruling (2026-09-23), not a technical dependency; keep as ordering note, not a hard blockedBy. Large and speculative; P1 overstated.

## History
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-va30
- 2026-10-06T04:03:34Z dep_added: blocked_by MAC-9azz
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-va30

## Links
- Blocked by: [[MAC-9azz]]
- Was blocked by: [[MAC-va30]]
- Related: [[MAC-syos]]

## Comments
