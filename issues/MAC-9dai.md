---
id: MAC-9dai
title: "End-to-end proof in the definition of done: declared e2e behaviors, external-dependency postures, executed evidence at acceptance"
status: open
priority: 1
type: epic
labels: [e2e, dod, acceptance]
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-09-25T19:53:32Z
content_hash: "sha256:a97676aea92da62338292db581f68437825d90e8c49dd2868e5cb6017bd40967"
related: [MAC-ui8a]
---

## Description
Owner direction 2026-09-25: hard TDD stays, but it buys functional correctness at unit level, which matters less with capable models. The missing piece is proof that the system works. A milestone is done only when its behaviors are proven end to end, through the real entry points, including when external systems are involved.

Current state (machinery 0.10.1):
- BUILD milestones carry `DoD:` lines citing oracle ids (mostly per-machine transition rows, unit-level) and a prose `Demo:` line that is only attested (skills/machinery/references/build-md-template.md:242-263).
- Gt credits static discovery of test references; no machinery gate executes tests (README "Inside what is checked").
- Stand-ins replace external neighbors; end-to-end latency, cross-contract liveness and unmodeled channels are deferred to a "parent assembly suite" nothing enforces (build-md-template.md:143).
- The projection reserves but never emits Modelith `scenarios` (docs/external-checkers.md), which are the model's own statement of user-visible behavior.
- Ga acceptance records that someone looked at a commit; it does not require executed evidence.

Target: every milestone declares its end-to-end behaviors; each is bound to an e2e test; every external dependency has an explicit e2e posture; and acceptance requires executed e2e evidence at the reviewed commit. No mocks in e2e. An unprovable behavior is a named residual that blocks closure unless the owner waives it with a reason.

Phase 1 (declarative, deterministic gates) lands without the executable-assurance chain. Phase 2 moves e2e evidence onto executed runtime obligations (MAC-ui8a, MAC-u4oo).

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Related: [[MAC-ui8a]]

## Comments
