---
id: MAC-f15w
title: "Prerequisite-review contract: challenge invented prerequisites and incompatible proof reuse"
status: open
priority: 3
type: task
labels: [review-protocol, docs, h2-lessons, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:651b09538c85c75e6ecc356bfcc5c6b621999a672f6db6ce526f7cfda1a20890"
blocked_by: [MAC-kyh6]
was_blocked_by: [MAC-94n5, MAC-q1kf]
---

## Description
Prerequisite-review contract: challenge invented prerequisites and incompatible proof reuse

Problem (H2, 2026-09-14): a publication proposal treated permanent shutdown of future observations as a prerequisite to ordinary immutable publication and proposed a new closure registry; tracing the actual W/T/C consumers showed they need exact original bytes/readback/retention, truthful inspected-cut history and complete current copy accounting, and the registry was rejected. A bootstrap completion proposal borrowed a ScopedWork closure whose receipt requires erasure Closing and Barrier facts that routine provisioning cannot produce. Both were proposal/review errors. Machinery recorded (commit 7222c075, docs/consistency-layer-proposal.md:676-687) that entry 16 is judgment and ships no Datalog rule; the fact-shaped parts are covered by supersession.dl and facts.dl. The documented review contract itself does not exist yet.

Proposed fix: a documented prerequisite-review contract for conductor and reviewers (skill plus docs), tied to actual consumers:
- Every newly blocking source dependency names the concrete consumer claim, authoritative requirement, exact applicability preconditions and a failure trace showing why it is necessary (reuse the entry 10 trace format).
- Distinguish historical/original result, current readiness, invocation completion and final scoped disposal; a stronger proof may not silently become a prerequisite to a weaker claim, while existing effects, uncertainty and later-copy duties still apply.
- Check receipt/source-family compatibility before reuse: matching field shapes or names do not import the producer's authority, lifecycle preconditions or scope; unsupported reuse needs a recorded disposition rather than new entities, stores or transitions.
- Report rejected or invented prerequisites separately from discovered defects and from missing implementation evidence, linked to revision impact (12) and supersession (15), so a private proposal cannot become an unquestioned blocker through repetition in progress records or packets.

Acceptance criteria:
1. Sanitized positive/negative fixtures distinguish ordinary immutable publication from final erasure, and ordinary native completion from an erasure-only closure.
2. The wrong receipt or prerequisite is challenged with its exact incompatible precondition; a genuine unresolved original create or unaccounted selected copy still blocks the claim that depends on it.
3. No assertion, retention requirement, current-denial check or acceptance gate is weakened.
4. Output states where semantic necessity needs human/conductor adjudication and never claims a type/reference check proves it.
5. No private native material is copied into machinery fixtures.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:58Z dep_added: blocked_by MAC-kyh6
- 2026-09-24T21:33:58Z dep_added: blocked_by MAC-q1kf
- 2026-09-24T21:33:58Z dep_added: blocked_by MAC-94n5
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-94n5
- 2026-10-06T04:03:39Z dep_removed: was_blocked_by MAC-q1kf

## Links
- Blocked by: [[MAC-kyh6]]
- Was blocked by: [[MAC-94n5]], [[MAC-q1kf]]

## Comments
