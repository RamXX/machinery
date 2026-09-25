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
content_hash: "sha256:df22beab0df9d91a3dcf7f9f96fe35e5a90ea2db660ad45d29915161c14e9107"
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


## Links
- Parent: [[MAC-9dai]]

## Comments
