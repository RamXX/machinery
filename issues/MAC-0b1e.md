---
id: MAC-0b1e
title: "README residuals: isolation is not noninterference; possibility and statistical properties are not proven"
status: open
priority: 2
type: task
labels: [docs, isolation]
created_at: 2026-09-24T21:27:28Z
created_by: ramirosalas
updated_at: 2026-09-24T21:27:28Z
content_hash: "sha256:c50f7d386bb145e0ce5bd30f5dc73366555a22d822f1fe22f8f2a7dd06bfd442"
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


## History


## Links


## Comments
