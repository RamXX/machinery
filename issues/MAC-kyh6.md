---
id: MAC-kyh6
title: "Cross-boundary review protocol with concrete failure traces and a seeded benchmark corpus"
status: open
priority: 3
type: feature
labels: [review-protocol, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:51Z
content_hash: "sha256:1bc3dc3f034dd8576205f3bf87d78d4df720024bdce9606c35986c89bd8d0264"
blocks: [MAC-f15w]
was_blocked_by: [MAC-zti7]
---

## Description
Cross-boundary design review protocol with concrete failure traces and a seeded benchmark corpus

Problem: repeated LLM prose reviews can share assumptions. Locally consistent contracts can be impossible together: admission needing future commit evidence, authorization confused with worker identity, a database plus cloud operation treated as one atomic transaction. H2 exposed incompatible uniqueness, retry and erasure guarantees only when their interactions were traced. Not implemented; docs/consistency-layer-proposal.md section 7 keeps failure-trace reviews outside the rule layer.

Proposed fix: a portable review protocol plus a synthetic benchmark corpus. For each selected high-risk workflow, reviewers trace real actors, inputs, authority, state stores, transaction boundaries, external effects, observable outcomes and recovery after each boundary, including restore rollback, crash after an external effect but before acknowledgement, duplicate or concurrent requests, revocation races, unavailable evidence, retained data and unrelated shared-key content. Output counterexample traces and unresolved premises, not just VALIDATED labels. A declarative producer/consumer ordering contract is added only if a bounded prototype shows value; any automated cycle check must distinguish impossible future-proof dependencies from valid iterative or recovery protocols. No keyword heuristics advertised as semantic verification.

Acceptance criteria:
1. Independent reviewers without the expected-answer file identify seeded restore resurrection, future-result authorization, premature erasure, retention/key coupling and unreadable uniqueness-reservation defects.
2. The corpus includes correct near-neighbours and the report measures false positives.
3. Each review records exactly what was reviewed; a delta approval never implies whole-system approval.
4. Tool validation checks trace structure and references only, and says so; it does not claim the truth of arbitrary prose.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Write the portable failure-trace review protocol and seeded synthetic benchmark corpus with near-neighbours and false-positive measurement. Evidence: No cross-boundary review protocol or corpus in repo; docs/consistency-layer-proposal.md section 7 explicitly excludes it. 0.11.0 threat-driven verification (docs/threat-driven-verification.md) covers adversary rows, not restore/crash/external-effect traces, so it does not supersede this. Notes: The protocol is a review procedure; it does not need the readiness inventory to exist. Priority P2 in nd looks high given no consumer breakage; P3 suggested. It is the prerequisite for MAC-f15w.

## History
- 2026-09-24T21:33:57Z dep_added: blocked_by MAC-zti7
- 2026-09-24T21:33:58Z dep_added: blocks MAC-f15w
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-zti7

## Links
- Blocks: [[MAC-f15w]]
- Was blocked by: [[MAC-zti7]]

## Comments
