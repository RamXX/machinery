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
updated_at: 2026-09-25T19:53:32Z
content_hash: "sha256:96ad7908cec35199fb3994dda52ecae52e2cdb2d506dd57b1b85e081fcc4f4f9"
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


## History
- 2026-09-25T19:53:33Z dep_added: blocked_by MAC-p3pj
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-g7d4

## Links
- Parent: [[MAC-9dai]]
- Blocked by: [[MAC-p3pj]]
- Was blocked by: [[MAC-g7d4]]

## Comments
