---
id: MAC-o3p2
title: "Derive membership/mandatory from bounded cardinality; annotation fields become optional and cross-checked"
status: open
priority: 3
type: feature
labels: [modelith, relational, alloy, simplification]
parent: MAC-0p5d
created_at: 2026-09-25T20:11:08Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:3cf4e97422ac6f3f5785bbaf08423e414d69037e7e661c4780580f60b2bce8c2"
blocked_by: [MAC-g7d4]
---

## Description
Use bounded cardinality to derive what the relational annotations currently restate because 0.4 "cannot express" it: policy/isolation `membership: lone|one` and integrity `mandatory`.

Normalize each relationship side to (lo, hi): n:0..1 means lone, not mandatory; n:1..1 (and explicit ranges) means one, mandatory. The annotation field becomes optional when the model states an explicit range, and is cross-checked when both are present (disagreement is an error). Replace exact-string `hasRelationship(..., "n:1")` (alloy.go:500, isolation.go:185) and the four-value switches (integrity.go:132-156,249; isolation.go:207-217) with to-one (hi == 1) logic. Exact counts > 1 have no Alloy field multiplicity: fail clearly.

Caution: 0.5 defines bare `1` as exactly one, but existing models use `n:1` with membership lone (go-crm Team->User). Derive only from explicit ranges at first; bare `1` keeps reading the annotation, with a warning when they disagree.

Acceptance criteria:
1. A model with explicit n:0..1 / n:1..1 needs no membership/mandatory field; generated Policy/Isolation/Integrity .als and oracles are byte-identical to the annotated equivalent.
2. Model and annotation disagreeing fails naming both.
3. Remove the "cannot express" wording (alloy.go:468, isolation.go:148, docs/policy-layer.md:121, isolation-layer.md:91, integrity-layer.md:98); CHANGELOG notes existing annotations stay valid.
4. verify-formal green on all examples.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Normalize cardinality to (lo,hi), make membership/mandatory optional and cross-checked, drop the wording, regenerate/verify-formal. Evidence: Annotation-only membership still in force: internal/alloy/alloy.go:468 and isolation.go:148 still say Modelith cardinality 'cannot express' membership; hasRelationship exact-string checks remain (alloy.go:220). Notes: Pure simplification. Dependency on g7d4 is real (pin 0.5 and loud refusal define which forms are legal). Also should coordinate with gscc on cardinality vocabulary.

## History
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4

## Links
- Parent: [[MAC-0p5d]]
- Blocked by: [[MAC-g7d4]]

## Comments
