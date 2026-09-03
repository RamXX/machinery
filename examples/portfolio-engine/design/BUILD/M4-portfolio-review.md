# M4 - Portfolio review slice

## Outcome

A Manager or Admin reviews, accepts or rejects, persists, conflict-retries, and reopens a portfolio;
an Analyst is refused, and corrupt prior-stage routing fails loudly.

## Domain context

Portfolio persists status, acceptedAt, and version. Enforce `portfolio-review-forward`,
`portfolio-accept-role`, `portfolio-reopen-role`, and `portfolio-accepted-has-date`. The commit
overlay states are not persisted. Writes are idempotent by `(portfolioId, version)`.

## Architecture context

`pf.domain` computes transitions; `pf.app` invokes the `pf.repo` optimistic write and bounded
backoff; the DuckDB adapter updates only at the expected version. Inject role, clock, and retry
policy. No domain package imports DuckDB.

## Behavior and oracles

Parse every row of `machines/Portfolio.oracle.md`: `PORT-27d66f`, `PORT-2bf44c`, `PORT-a41039`,
`PORT-ddb44c`, `PORT-351dec`, `PORT-db3bb9`, `PORT-9facf7`, `PORT-5e6be0`, `PORT-f43140`,
`PORT-d1647b`, `PORT-fb8c92`, `PORT-40b6e7`, `PORT-c4a186`, `PORT-f6e220`, `PORT-cba032`,
`PORT-8c0400`, `PORT-3cb0b6`, `PORT-53d34b`, `PORT-3390a7`, and `PORT-3bd579`. Assert ordered actions.

## TDD and implementation

Write conformance, role-refusal, retry-bound, idempotency, and real DuckDB optimistic-conflict tests.
Cover null and invalid prior-stage fallbacks. Lock RED after formatter/linter/import checks, then
implement transition, app overlay, and repository compare-and-update without changing locked tests.

## Risks and recovery

Bound retries and backoff. On non-retriable failure restore the exact prior domain stage; if no
valid witness exists enter routingFault and persist no guessed value. After a crash, reload version
and status before retrying; compare-and-update prevents duplicate decisions.

## Acceptance

Exercise all 20 rows, both authorized roles, Analyst refusal, forced conflict, exhausted retry,
idempotent replay, and routing fault against real DuckDB. Record evidence in `acceptance/M4.yaml`
against one reviewed commit.
