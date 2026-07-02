# Portfolio machine: named-unit contracts and failure catalog

Transitions are covered by the generated `Portfolio.oracle.md`. Review lifecycle (linear-lifecycle
pattern) with a commit overlay named committing/commitRetry/reverted for this domain. Every guard,
action, and actor the machine fires has a row below.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `canDecide` | guard | `(ctx, evt) -> bool` | true iff the acting role is Manager or Admin | invariant `portfolio-accept-role` | unit | fake actor role |
| `canReopen` | guard | `(ctx, evt) -> bool` | true iff the acting role is Manager or Admin | invariant `portfolio-reopen-role` | unit | fake actor role |
| `pendingIsUnderReview` | guard | `(ctx) -> bool` | true iff `pending == UnderReview` | routes the persisted advance | unit | none |
| `pendingIsAccepted` | guard | `(ctx) -> bool` | true iff `pending == Accepted` | routes the persisted accept | unit | none |
| `pendingIsRejected` | guard | `(ctx) -> bool` | true iff `pending == Rejected` | routes the persisted reject | unit | none |
| `isRetriable` | guard | `(ctx, evt) -> bool` | true iff the commit error is a transient conflict or busy store | store conflict is transient | unit | synthetic ConflictError |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `retries >= MaxRetries` | bounds the commit retry loop | unit | none |
| `priorIsProposed` | guard | `(ctx) -> bool` | true iff `prior == Proposed` | routes rollback to the departed stage | unit | none |
| `priorIsUnderReview` | guard | `(ctx) -> bool` | true iff `prior == UnderReview` | routes rollback to the departed stage | unit | none |
| `priorIsAccepted` | guard | `(ctx) -> bool` | true iff `prior == Accepted` | routes rollback of a failed reopen | unit | none |
| `priorIsRejected` | guard | `(ctx) -> bool` | true iff `prior == Rejected` | routes rollback of a failed reopen | unit | none |
| `setPendingAdvance` | action | `(ctx) -> ctx` | sets `pending := UnderReview`, `prior := Proposed` | invariant `portfolio-review-forward` | unit | none |
| `setPendingAccept` | action | `(ctx) -> ctx` | sets `pending := Accepted`, `prior := current` | invariant `portfolio-review-forward` | unit | none |
| `setPendingReject` | action | `(ctx) -> ctx` | sets `pending := Rejected`, `prior := current` | invariant `portfolio-review-forward` | unit | none |
| `setPendingReopen` | action | `(ctx) -> ctx` | sets `pending := UnderReview`, `prior := current` | invariant `portfolio-review-forward` | unit | none |
| `commit` | action | `(ctx) -> ctx` | sets `status := pending`, clears `pending`/`prior`, resets `retries` | invariant `portfolio-review-forward` | unit | none |
| `recordAccepted` | action | `(ctx) -> ctx` | sets `acceptedAt := now` when committing to Accepted | invariant `portfolio-accepted-has-date` | unit | fake clock |
| `incRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | bounds the commit retry loop | unit | none |
| `persistDecision` | actor | `(portfolioId, pending) -> Ack` | writes the portfolio under its version guard; idempotent by `(portfolioId, version)`; a side-effect and idempotency contract, not derivable from transition tests | C4 rel: pf.app to pf.repo to store | integration | contract-tested DuckDB fake plus one real-store test |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| optimistic-lock conflict on write | `persistDecision` invoke `onError` with a retriable ConflictError (guard `isRetriable`) | committing to commitRetry | back off `RETRY_BACKOFF`, `incRetries`, retry the write | bounded by `retriesExhausted` (<= MaxRetries); then commitRetry to reverted |
| retries exhausted | guard `retriesExhausted` true in commitRetry | commitRetry to reverted | roll the in-memory transition back to `prior`; command reports a polite refusal | residual: the portfolio is unchanged; operator signal is "another reviewer changed this portfolio, please retry" |
| non-retriable write error | `persistDecision` invoke `onError` unguarded fallback | committing to reverted | roll back to `prior`; surface the error | residual: unchanged portfolio; surfaced as a repo IOError |
| write timeout | `after COMMIT_TIMEOUT` fires in committing | committing to commitRetry | same bounded retry path | bounded by MaxRetries; 5 s write timeout per attempt |
| illegal review action | no guarded `on` transition matches (e.g. accept on an Accepted portfolio) | none (event ignored per `_ignores`, or rejected upstream) | command returns a rejection | structural: the graph makes the backward or decided-illegal move impossible (`portfolio-review-forward`) |
