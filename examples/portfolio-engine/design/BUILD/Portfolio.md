# BUILD shard: Portfolio (review lifecycle and commit overlay)

Shard of the manifest root `design/BUILD.md` (Mode: manifest). The root carries the glossary, the
domain model, the Architecture Contract, the traceability matrix, the shared toolchain, the
state-migration protocol, and the hard-TDD protocol; this shard carries the Portfolio component:
its behavior, its test specification, and its build-plan milestone (M4 of the root milestone map).

## 5. Behavior

### Portfolio (`design/machines/Portfolio.machine.json`)

A review lifecycle: Proposed advances to UnderReview, then a Manager or Admin accepts or rejects
it; a decided portfolio may be reopened to UnderReview. Every state change is written through the
commit overlay (committing invokes the versioned write; a retriable conflict retries with backoff
up to MaxRetries via commitRetry, then rolls back to the prior stage via reverted). Accepting
records acceptedAt. Named-unit contracts and failure catalog: `design/machines/Portfolio.matrix.md`
(11 guards, 7 actions, 1 actor). `canDecide` enforces `portfolio-accept-role`; `canReopen` enforces
`portfolio-reopen-role`; `recordAccepted` enforces `portfolio-accepted-has-date`. The
`persistDecision` actor is an integration/side-effect contract (writes once per
`(portfolioId, version)`), not derivable from transition tests.

## 7. Test specification

The transition test spec IS the generated `design/machines/Portfolio.oracle.md` (19 rows). Do not
restate the table; tests key on the STABLE id, never the row number:

| stable id | transition |
|---|---|
| PORT-27d66f | Proposed, advance, to committing (setPendingAdvance) |
| PORT-2bf44c | Proposed, accept, canDecide, to committing |
| PORT-a41039 | Proposed, reject, canDecide, to committing |
| PORT-ddb44c | UnderReview, accept, canDecide, to committing |
| PORT-351dec | UnderReview, reject, canDecide, to committing |
| PORT-db3bb9 | Accepted, reopen, canReopen, to committing |
| PORT-9facf7 | Rejected, reopen, canReopen, to committing |
| PORT-5e6be0 | committing, commit timeout, to commitRetry |
| PORT-f43140 | committing, persist done, pendingIsUnderReview, to UnderReview |
| PORT-d1647b | committing, persist done, pendingIsAccepted, to Accepted (recordAccepted) |
| PORT-fb8c92 | committing, persist done, pendingIsRejected, to Rejected |
| PORT-40b6e7 | committing, persist error, isRetriable, to commitRetry |
| PORT-c4a186 | committing, persist error, to reverted |
| PORT-f6e220 | commitRetry, backoff elapsed, to committing (incRetries) |
| PORT-cba032 | commitRetry, retriesExhausted, to reverted |
| PORT-8c0400 | reverted, priorIsProposed, to Proposed |
| PORT-3cb0b6 | reverted, priorIsUnderReview, to UnderReview |
| PORT-53d34b | reverted, priorIsAccepted, to Accepted |
| PORT-3390a7 | reverted, priorIsRejected, to Rejected |

### Guard-branch completeness

The guards are single-clause or disjunctions, not conjunctions, so there are no A-AND-B-AND-C
falsifying triples; the falsifying tests are:

- `canDecide` = (role is Manager) OR (role is Admin). Falsifying: an Analyst attempts accept or
  reject; the guard is false and the decision does not fire (AuthzError). Covers
  `portfolio-accept-role`.
- `canReopen` = (role is Manager) OR (role is Admin). Falsifying: an Analyst attempts reopen on a
  decided portfolio; refused. Covers `portfolio-reopen-role`.
- `retriesExhausted` = (retries >= MaxRetries). One test below the bound retries (PORT-f6e220), one
  at the bound routes to reverted (PORT-cba032).

### Named-unit test plan

Per the matrix: guards and pending/prior/commit actions are unit tests over context;
`recordAccepted` uses a fake clock; `canDecide`/`canReopen` use fake roles. The `persistDecision`
actor is an integration test: idempotency (writes once per `(portfolioId, version)`) against a
contract-tested DuckDB fake plus one real-store test, and a forced version conflict exercising the
overlay end to end.

## 8. State migration

`Portfolio` persists `status`, `acceptedAt`, and `version`; no persisted instances yet. The
protocol is the root's section 8; the committing/commitRetry/reverted overlay states are never
persisted, so renaming them needs no migration.

## 9. Build plan

The design-wide walking skeleton is M0 in `BUILD/RecommendationRun.md`; this shard's milestone
extends the already-proven topology. Walking skeleton: N/A - the design-wide skeleton lives in the
RecommendationRun shard (M0); it crosses this shard's repo boundary (one real DuckDB write) before
M4 begins.

**M4 - Portfolio review slice.** All Portfolio transitions, `portfolio-review-forward`,
`portfolio-accept-role`, `portfolio-reopen-role`, `portfolio-accepted-has-date`, and the commit
overlay under a forced version conflict. DoD: all 19 Portfolio oracle rows covered by stable id
(PORT-27d66f through PORT-3390a7 above), the four listed invariants property-tested, the commit
overlay verified under a forced version conflict, its contract tests green, G4-import clean, formal
suite still green.
