---
id: MAC-kyh6
title: "Cross-boundary review protocol with concrete failure traces and a seeded benchmark corpus"
status: open
priority: 2
type: feature
labels: [review-protocol, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:8b9be716ba38b866262a261ae0ec633315a24a00366c66057f4ae8bd597ebac8"
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


## History


## Links


## Comments
