# Task machine: named-unit contracts and failure catalog

Transitions are covered by the generated `Task.oracle.md`. This file carries the units the coding
agent implements and the failure catalog. The last rows cover units used by the non-lifecycle Task
create action that enforce invariants outside the state graph.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `pendingIsInProgress` | guard | `(ctx) -> bool` | true iff `pending == InProgress` | routes the persisted start | unit | none |
| `pendingIsDone` | guard | `(ctx) -> bool` | true iff `pending == Done` | routes the persisted complete | unit | none |
| `pendingIsAbandoned` | guard | `(ctx) -> bool` | true iff `pending == Abandoned` | routes the persisted abandon | unit | none |
| `isRetriable` | guard | `(ctx, evt) -> bool` | true iff the persist error is a transient conflict or busy/locked store | store conflict is transient | unit | synthetic ConflictError |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `retries >= MaxRetries` | bounds the retry loop | unit | none |
| `priorIsOpen` | guard | `(ctx) -> bool` | true iff `prior == Open` | routes rollback to the departed state | unit | none |
| `priorIsInProgress` | guard | `(ctx) -> bool` | true iff `prior == InProgress` | routes rollback to the departed state | unit | none |
| `setPendingStart` | action | `(ctx) -> ctx` | sets `pending := InProgress`, `prior := Open` | task-terminal-closed (only open tasks start) | unit | none |
| `setPendingComplete` | action | `(ctx) -> ctx` | sets `pending := Done`, `prior := InProgress` | invariant `task-terminal-closed` | unit | none |
| `setPendingAbandon` | action | `(ctx) -> ctx` | sets `pending := Abandoned`, `prior := current` | invariant `task-terminal-closed` | unit | none |
| `commit` | action | `(ctx) -> ctx` | sets `status := pending`, clears `pending` and `prior`, resets `retries` | invariant `task-terminal-closed` | unit | none |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | bounds the retry loop | unit | none |
| `persist` | actor | `(taskId, pending) -> Ack` | writes the task under its version guard; idempotent by `(taskId, version)`; a side-effect and idempotency contract, not derivable from transition tests | C4 rel: crm.app to crm.repo to store | integration | contract-tested LadybugDB fake plus one real-store test |
| `taskHasAssignee` | guard | `(task) -> bool` | true iff the task references exactly one assignee; used by the create action | invariant `task-has-assignee` | property | none |
| `taskHasDueDate` | guard | `(task) -> bool` | true iff the task has a due date; used by the create action | invariant `task-has-due-date` | property | none |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| optimistic-lock conflict on write | `persist` invoke `onError` with a retriable ConflictError (guard `isRetriable`) | persisting to persistRetry | back off `RETRY_BACKOFF`, `incrementRetries`, retry `persist` | bounded by `retriesExhausted` (<= MaxRetries); then persistRetry to rolledBack |
| retries exhausted | guard `retriesExhausted` true in persistRetry | persistRetry to rolledBack | roll the in-memory transition back to `prior`; command reports a polite refusal | residual: the task is unchanged; operator signal is CommandExecution `refused` |
| non-retriable write error | `persist` invoke `onError` unguarded fallback | persisting to rolledBack | roll back to `prior`; surface the error | residual: unchanged task; surfaced as a repo IOError |
| write timeout | `after PERSIST_TIMEOUT` fires in persisting | persisting to persistRetry | same bounded retry path | bounded by MaxRetries; 5 s write timeout per attempt |
| action on a terminal task | Done and Abandoned are `final` | none (structurally rejected) | command returns a rejection | structural: `task-terminal-closed` holds because final states accept no events |
