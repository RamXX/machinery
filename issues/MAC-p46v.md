---
id: MAC-p46v
title: "Project Modelith scenarios; each scenario proven by an e2e behavior or named as a residual"
status: open
priority: 2
type: feature
labels: [e2e, modelith, projection]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:d865674bd6399875c1dd75d8a344c727df42adaa8e3a7fcddee173cc865f5579"
blocked_by: [MAC-p3pj]
was_blocked_by: [MAC-g7d4]
---

## Description
Project Modelith scenarios and require each to be proven end to end or named as a residual. Scenarios are the domain model's own statement of user-visible behavior; the projection reserves `scenarios` but never emits it (docs/external-checkers.md), and the phase-1 attested half only checks "scenario coverage" by review.

Acceptance criteria:
1. The projection emits a scenarios layer (scenario id, entity/actions involved) with stable ids.
2. Every scenario maps to at least one declared e2e behavior in some milestone, or to a reasoned residual; an unmapped scenario is a finding.
3. Coordinate with the Modelith 0.5.0 adoption (scenario format unchanged or updated) before implementing.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Emit the scenarios layer with stable ids and gate that every scenario maps to an e2e behavior or a reasoned residual. Evidence: Projection still reserves scenarios: docs/external-checkers.md:110 and :272 say scenarios is reserved and fails loudly. CHANGELOG.md:401 repeats 'scenarios stays reserved'. Notes: MAC-g7d4 (pin bump to Modelith 0.5.0) is largely done: the pin is v0.5.0 since machinery 0.10.3 (CHANGELOG.md:141). The AC3 coordination note is satisfied by the pin; g7d4 only adds loud refusal of imports and is not needed to emit scenarios. Verify scenario format in 0.5.0 before implementing.

## History
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-p3pj
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-g7d4

## Links
- Parent: [[MAC-9dai]]
- Blocked by: [[MAC-p3pj]]
- Was blocked by: [[MAC-g7d4]]

## Comments
