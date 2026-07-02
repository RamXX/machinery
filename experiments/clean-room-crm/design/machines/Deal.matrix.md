# Deal machine: named-unit contracts and failure catalog

Transitions are covered by the generated `Deal.oracle.md` (do not restate them here). This file
carries the units the coding agent implements and the failure catalog. Every guard, action, and
actor the machine fires has a row below; the last rows cover units used by the non-lifecycle Deal
actions (create, updateAmount) that enforce invariants outside the state graph.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `canReopen` | guard | `(ctx, evt) -> bool` | true iff the acting user is a Manager or Admin | invariant `deal-reopen-role` | unit | fake actor with role |
| `pendingIsQualification` | guard | `(ctx) -> bool` | true iff `pending == Qualification` | routes the persisted advance | unit | none |
| `pendingIsProposal` | guard | `(ctx) -> bool` | true iff `pending == Proposal` | routes the persisted advance | unit | none |
| `pendingIsNegotiation` | guard | `(ctx) -> bool` | true iff `pending == Negotiation` | routes the persisted advance or reopen | unit | none |
| `pendingIsWon` | guard | `(ctx) -> bool` | true iff `pending == Won` | routes the persisted win | unit | none |
| `pendingIsLost` | guard | `(ctx) -> bool` | true iff `pending == Lost` | routes the persisted lose | unit | none |
| `isRetriable` | guard | `(ctx, evt) -> bool` | true iff the persist error is a transient conflict or busy/locked store | store conflict is transient | unit | synthetic ConflictError |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `retries >= MaxRetries` | bounds the retry loop | unit | none |
| `priorIsProspecting` | guard | `(ctx) -> bool` | true iff `prior == Prospecting` | routes rollback to the departed stage | unit | none |
| `priorIsQualification` | guard | `(ctx) -> bool` | true iff `prior == Qualification` | routes rollback to the departed stage | unit | none |
| `priorIsProposal` | guard | `(ctx) -> bool` | true iff `prior == Proposal` | routes rollback to the departed stage | unit | none |
| `priorIsNegotiation` | guard | `(ctx) -> bool` | true iff `prior == Negotiation` | routes rollback to the departed stage | unit | none |
| `priorIsWon` | guard | `(ctx) -> bool` | true iff `prior == Won` | routes rollback of a failed reopen | unit | none |
| `priorIsLost` | guard | `(ctx) -> bool` | true iff `prior == Lost` | routes rollback of a failed reopen | unit | none |
| `setPendingAdvance` | action | `(ctx) -> ctx` | sets `pending := next stage of current`, `prior := current`; enforces forward order | invariant `deal-forward-only` | unit | none |
| `setPendingWin` | action | `(ctx) -> ctx` | sets `pending := Won`, `prior := current` | invariant `deal-forward-only` | unit | none |
| `setPendingLose` | action | `(ctx) -> ctx` | sets `pending := Lost`, `prior := current` | invariant `deal-forward-only` | unit | none |
| `setPendingReopen` | action | `(ctx) -> ctx` | sets `pending := Negotiation`, `prior := current` | invariant `deal-reopen-to-negotiation` | unit | none |
| `commit` | action | `(ctx) -> ctx` | sets `stage := pending`, clears `pending` and `prior`, resets `retries` | invariant `deal-forward-only` | unit | none |
| `recordClose` | action | `(ctx) -> ctx` | sets `closedAt := now` when committing to Won or Lost | invariant `deal-won-has-close-date` | unit | fake clock |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | bounds the retry loop | unit | none |
| `persist` | actor | `(dealId, pending) -> Ack` | writes the deal under its version guard; idempotent by `(dealId, version)`; a side-effect and idempotency contract, not derivable from transition tests | C4 rel: crm.app to crm.repo to store | integration | contract-tested LadybugDB fake plus one real-store test |
| `amountNonNegative` | guard | `(amount) -> bool` | true iff `amount >= 0`; used by the create and updateAmount actions, not by the state graph | invariant `deal-amount-non-negative` | property | none |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| optimistic-lock conflict on write | `persist` invoke `onError` with a retriable ConflictError (guard `isRetriable`) | persisting to persistRetry | back off `RETRY_BACKOFF`, `incrementRetries`, retry `persist` | bounded by `retriesExhausted` (<= MaxRetries); then persistRetry to rolledBack |
| retries exhausted | guard `retriesExhausted` true in persistRetry | persistRetry to rolledBack | roll the in-memory transition back to `prior`; command reports a polite refusal | residual: the deal is unchanged; operator signal is the CommandExecution `refused` state (busy notice) |
| non-retriable write error | `persist` invoke `onError` unguarded fallback | persisting to rolledBack | roll back to `prior`; surface the error | residual: unchanged deal; surfaced as a repo IOError up the stack |
| write timeout | `after PERSIST_TIMEOUT` fires in persisting | persisting to persistRetry | same bounded retry path | bounded by MaxRetries; 5 s write timeout per attempt |
| illegal transition attempt | no guarded `on` transition matches (e.g. advance from Won) | none (event ignored per `_ignores`, or rejected upstream) | command returns a rejection | structural: the graph makes the backward or terminal-illegal move impossible (`deal-forward-only`) |
