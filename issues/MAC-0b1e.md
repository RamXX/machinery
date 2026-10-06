---
id: MAC-0b1e
title: "README residuals: isolation is not noninterference; possibility and statistical properties are not proven"
status: open
priority: 3
type: task
labels: [docs, isolation]
created_at: 2026-09-24T21:27:28Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:71f8d234e18b3b96f0637236a278bfda32ec1d7f78e205396ee8f6ca72352de7"
related: [MAC-2f7g]
---

## Description
README "What machinery does not verify" (README.md ~941) omits three limits a reader could otherwise assume are covered.

1. Isolation is not information-flow security. The isolation layer proves tenant consistency of the reference graph (a record and what it references share a tenant), an access-control property checked per state. It does not prove noninterference (a hyperproperty over two traces): a tenant learning about another tenant through a differing error code for exists-vs-forbidden, counts, aggregates, or timing is outside it. Say so in README and docs/isolation-layer.md.
2. Possibility properties ("a user can always cancel") are not proven: TLC checks linear-time properties over single behaviors, and guard erasure makes any positive possibility result an over-approximation.
3. Statistical properties (latency percentiles, throughput) are outside every engine machinery runs.

Source: Hillel Wayne on TLA+ limits (2026-09).

Acceptance criteria: the three bullets appear in README's residual list and the isolation guide; wording matches the eventual property-class vocabulary if that story lands first; no em dashes.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add the three bullets to README residual list and docs/isolation-layer.md; align wording with property-class vocabulary if MAC-2f7g lands first. Evidence: README.md 'What machinery does not verify' (about lines 957-975) lists seven residuals; none mention noninterference, possibility or statistical properties (grep for noninterference/possibility/statistical/hyperprop in README.md and docs/isolation-layer.md returns nothing). Only the 0.11.0 threat-ledger residual was added. Notes: Pure docs, no dependency. Can ship now; vocabulary can be adjusted later. Priority P2 is high for a docs-only residual but the claim-honesty angle justifies P2-P3.

## History


## Links
- Related: [[MAC-2f7g]]

## Comments
