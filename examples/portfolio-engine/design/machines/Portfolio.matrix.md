# Portfolio machine: named-unit contracts and failure catalog

Transitions are covered by the generated `Portfolio.oracle.md`. Review lifecycle (linear-lifecycle
pattern) with a commit overlay named committing/commitRetry/reverted and an explicit routingFault
terminal for corrupt rollback context. Every guard,
action, and actor the machine fires has a row below.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `canDecide` | guard | `(ctx, evt) -> bool` | true iff the acting role is in the authorized set `{Manager, Admin}`. `CLAUSES{actor-role-authorized}` | invariant `portfolio-accept-role` | unit | Manager, Admin, and Analyst role cases |
| `canReopen` | guard | `(ctx, evt) -> bool` | true iff the acting role is in the authorized set `{Manager, Admin}`. `CLAUSES{actor-role-authorized}` | invariant `portfolio-reopen-role` | unit | Manager, Admin, and Analyst role cases |
| `pendingIsUnderReview` | guard | `(ctx) -> bool` | true iff `pending == UnderReview` | routes the persisted advance | unit | none |
| `pendingIsAccepted` | guard | `(ctx) -> bool` | true iff `pending == Accepted` | routes the persisted accept | unit | none |
| `pendingIsRejected` | guard | `(ctx) -> bool` | true iff `pending == Rejected` | routes the persisted reject | unit | none |
| `isRetriable` | guard | `(ctx, evt) -> bool` | true iff the typed error arrives inside a confirmed-Unpublished outcome and belongs to the retriable class `{ConflictError, BusyError}` (plus timeout-won `IOError(COMMIT_TIMEOUT)`). `CLAUSES{error-class-retriable}` | store conflict or busy state is transient; generic IOError stays nonretriable; Unresolved never reaches this guard | unit | ConflictError, BusyError, confirmed-Unpublished IOError, and Unresolved contrast cases |
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
| `recordRoutingError` | action | `(ctx) -> ctx` | records that `prior` was absent or outside the four persisted `PortfolioStatus` values before entering terminal `routingFault`; never guesses a review state | defensive rollback-route integrity | unit | null and invalid prior values |
| `persistDecision` | actor | `(portfolioId, pending) -> Ack` | captures expectedVersion, the frozen complete Save value, an independent OperationId and its OperationHandle, then calls the bound `Save(value, expectedVersion, operation, expectedId)`; the repo compares both identities and exact captured arguments before mutation; a retry creates a NEW attempt/id/handle; idempotent by `(portfolioId, version)`; a side-effect and idempotency contract, not derivable from transition tests CARRIES{column:Portfolio.status, column:Portfolio.acceptedAt} | C4 rel: pf.app to pf.repo to store | integration | contract-tested DuckDB fake plus one real-store test |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| optimistic-lock conflict on write | confirmed-Unpublished ConflictError from the bound Save (guard `isRetriable`) | committing to commitRetry | back off `RETRY_BACKOFF`, `incRetries`, retry via a NEW operation handle | bounded by `retriesExhausted` (<= MaxRetries); then commitRetry to reverted; this attempt performed no publication |
| retries exhausted | guard `retriesExhausted` true in commitRetry | commitRetry to reverted | roll the in-memory transition back to the witnessed `prior`; command reports a polite refusal | residual: THIS attempt caused no publication (independent writers are not frozen); operator signal is "another reviewer changed this portfolio, please retry" |
| non-retriable write error | confirmed-Unpublished nonretriable error from the bound Save, unguarded fallback | committing to reverted | revert the in-memory stage to the witnessed prior and return the final typed error | residual: this attempt performed no publication; surfaced as the typed repo error class |
| write timeout | timeout wins BEFORE publication admission, then nonpublication is confirmed after drain | committing to commitRetry | deny publication, drain, then the same bounded retry path; if exhausted, revert and return `IOError(COMMIT_TIMEOUT)` | 5000 ms is the publication-admission deadline per attempt; an admitted commit finishing later stays authoritative |
| unresolved publication | drained Unresolved outcome of the bound Save | NO modeled transition; `reverted` is not entered on an unknown outcome | report `IOError(PUBLICATION_OUTCOME_UNKNOWN)` exit 12 with operation identity; one read-only Reconcile permitted; no automatic retry | unknown publication is a separate operational residual; never rewritten as rollback or ordinary failure |
| absent or corrupt rollback route | no `priorIs*` guard admits in reverted | reverted to terminal routingFault, `recordRoutingError` | stop and surface the corrupt command context; never guess a persisted review stage | explicit terminal residual; the stored Portfolio remains unchanged and available for repair |
| illegal review action | no guarded `on` transition matches (e.g. accept on an Accepted portfolio) | none (event ignored per `_ignores`, or rejected upstream) | command returns a rejection | structural: the graph makes the backward or decided-illegal move impossible (`portfolio-review-forward`) |
