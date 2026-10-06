---
id: MAC-b27n
title: "Composition check for assembled bounded values and exact source-type pairing"
status: open
priority: 3
type: feature
labels: [composition, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:50Z
content_hash: "sha256:9db1f8e9c2f938e6e1f3a0ad91798f3beea202c388d3f6118aa8eb029862feb6"
blocked_by: [MAC-zti7]
was_blocked_by: [MAC-q1kf]
---

## Description
Composition check for assembled bounded values and exact source-type pairing

Problem (H2, 2026-09-14): independently reviewed row/source products each respected their bounds, but assembling the whole nine-row retained transaction result produced a 57,772-byte synthetic case under a 40,960-byte ceiling; removing duplicated row views alone could not make it fit (nested base64, repeated native representations, source metadata and genuine late read witnesses all contributed). Separately, a protocol union extension allowed a specialized expectation inside a generic result, and a proposed prior-source link required its own future registration reference. Not implemented; docs/consistency-layer-proposal.md section 7 leaves assembled-value bounds outside the rule layer.

Proposed fix: an early composition experiment/check contract for bounded wire and local result designs. Require one complete assembled value at the actual consuming root, all physical effects and mandatory read witnesses, exact canonical encoding and nested-encoding expansion, and explicit evidence for each chosen parameter bound. Validate source-specific union pairings and distinguish prior references, same-result local leaves and future/self references. Prefer a public bounded prototype over a universal schema compiler; report unsupported semantics plainly. Lossless representation changes must reconstruct every original native field and byte, retain authority/currentness observations and preserve existing limits. Feed findings into the readiness report (entry 9) and revision-impact work (12).

Acceptance criteria:
1. Sanitized synthetic fixtures reproduce: individual products pass while the whole value overflows; an extra-wrapper depth overflow; an accidental cross-product of generic and specialized protocol arms; a prior-source self-reference cycle.
2. Correct near-neighbours pass, including retained late witnesses and exact acyclic local deduplication.
3. Bootstrap regressions: the complete signed work wrapper around a six-field readiness value (nested depth 9, lossless flattening depth 8); a plan claiming to bind a future result without encoded result identity must carry exact preallocated logical target/result fields and later equality to the registered execution, without demanding a native backend/xid before planning; initial input-manifest verification resolves the independently current frozen source, and using the future work record for initial verification is reported as a cycle.
4. Evidence identifies actual assembled bytes and depth, parameter premises, unresolved native applicability and exact field reconstruction.
5. Truncated rows, raised budgets, assumed provider maximums or shape-only fixtures are never reported as runnable GREEN or native proof; structural cases stay separate from authenticated source, transaction and native behavior evidence.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Build the early composition experiment and fixtures per ACs, after the readiness report exists to feed. Evidence: No composition check exists: grep for assembled value or composition check in cmd, internal, rules, docs finds nothing; no readiness or revision-impact report in internal/gates. Notes: Soft dependency: findings 'feed into' the readiness report (zti7); revision-impact (q1kf) is an output consumer, not a prerequisite. Design-heavy, speculative; keep P3.

## History
- 2026-09-24T21:33:58Z dep_added: blocked_by MAC-zti7
- 2026-09-24T21:33:58Z dep_added: blocked_by MAC-q1kf
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-q1kf

## Links
- Blocked by: [[MAC-zti7]]
- Was blocked by: [[MAC-q1kf]]

## Comments
