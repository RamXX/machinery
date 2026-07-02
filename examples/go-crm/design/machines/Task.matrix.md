# Task machine - contract, failure catalog, and transition oracle

Component: `crm.domain` Task aggregate. Machine: `Task.machine.json`.
Placement (ARCHITECTURE.md 7): ephemeral in-process, load-act-save inside the one write Tx; status persists as the graph node `status` attribute. Concurrency: read-modify-write in one write Tx; cross-process serialized by LadybugDB's single-writer lock (ErrLocked -> CommandExecution/DBLocked).

States trace to enum `TaskStatus`. Events trace to `Task` actions. `Done`/`Cancelled` are `final`, so `task-terminal` is enforced structurally (no `reopen` exists for Task). `reassign` changes the owner and lands back on the same status.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveTask` | actor | `(input{taskId,status,ownerId,newAssigneeId,actor}) -> TaskRow \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: guard passed, tx open. post: node `status`(+`owner` on reassign) atomically, or unchanged on err | C4 `crm.domain -> crm.repo` then `crm.repo -> store` (Cypher SaveTask) |
| `guardCanStart` | guard | `(ctx,evt) -> bool` | true iff actor may write the task (owner/manager/admin in scope) | inv `rbac-write-scope` |
| `guardCanComplete` | guard | `(ctx,evt) -> bool` | true iff source is non-terminal AND actor may write | inv `task-terminal`, `rbac-write-scope` |
| `guardCanCancel` | guard | `(ctx,evt) -> bool` | true iff source is non-terminal AND actor may write | inv `task-terminal`, `rbac-write-scope` |
| `guardCanReassign` | guard | `(ctx,evt) -> bool` | true iff new assignee is inside the assigner's VisibilityScope AND actor is Manager/Admin in scope | inv `task-assignee-visible`, `rbac-reassign-authority`, `rbac-write-scope` |
| `pendingIsOpen` / `pendingIsInProgress` / `pendingIsDone` / `pendingIsCancelled` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) |
| `priorIsOpen` / `priorIsInProgress` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status | - (rollback routing; only non-terminal states persist) |
| `isErrLocked` / `isErrConstraint` / `isErrDiskFull` / `isErrTimeout` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 section 6 failure classes |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 section 6 bound (retry <= 3, ~1.5s) |
| `setPendingStart` | action | `(ctx) -> ctx` | `priorStatus:=status; pendingStatus:=InProgress` | - |
| `setPendingComplete` | action | `(ctx) -> ctx` | `priorStatus:=status; pendingStatus:=Done` | - |
| `setPendingCancel` | action | `(ctx) -> ctx` | `priorStatus:=status; pendingStatus:=Cancelled` | - |
| `setPendingReassign` | action | `(ctx,evt) -> ctx` | `priorStatus:=status; pendingStatus:=status; newAssigneeId:=evt.assigneeId` | supports `task-owned` |
| `commitStatus` | action | `(ctx) -> ctx` | `status:=pendingStatus` (owner if reassign) | - |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries:=retries+1` | - |
| `recordError` / `recordConstraint` / `recordDiskFull` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError:=classified error` for surfacing | maps repo errors to a domain error message |
| `recordStartDenied` / `recordCompleteDenied` / `recordCancelDenied` / `recordReassignDenied` / `recordAlreadyStarted` | action | `(ctx,evt) -> ctx` | set a rejection reason; no state change | surfaces the violated invariant to the caller |
| `recordTaskClosed` | action | `(ctx) -> ctx` | entry marker on a terminal status | - |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Constraint violation on write | `saveTask` onError `isErrConstraint` | `persisting -> rolledBack -> priorStatus` | surface as a domain validation error; store unchanged | C4 6: one write Tx, no partial write. Residual: none |
| Disk full on write | `saveTask` onError `isErrDiskFull` | `persisting -> rolledBack -> priorStatus` | fail loudly; DB stays consistent | C4 6: atomic. Residual: user must free disk |
| Query/write timeout | `saveTask` onError `isErrTimeout` OR `after persistTimeout` (10s) | `persisting -> rolledBack -> priorStatus` | abort, surface, roll back | C4 6: SetTimeout 10s. Residual: none |
| Store locked by another writer | `saveTask` onError `isErrLocked` | `persisting -> persistRetry -> persisting` then `rolledBack` when `retriesExhausted` | bounded retry, then surface | C4 6: retry <= 3, ~1.5s. Residual: refused after 3 tries |
| Reassign to out-of-scope user | guard `guardCanReassign` false | `Open/InProgress` internal, `recordReassignDenied` | reject; no write attempted | inv `task-assignee-visible` enforced before invoke. Residual: none |
| Mutate a terminal task | event arrives at `Done`/`Cancelled` (final) | none (structurally rejected) | task is closed; no-op | inv `task-terminal` structural (final states). Residual: none |

## (c) Transition matrix (hard-TDD oracle - one row per transition and per guard branch)

| # | source | event / after / always | guard | target | actions | derived-from |
|---|---|---|---|---|---|---|
| 1 | Open | start | guardCanStart | persisting | setPendingStart | start / rbac-write-scope |
| 2 | Open | start | !guardCanStart | Open (internal) | recordStartDenied | guard false branch |
| 3 | Open | complete | guardCanComplete | persisting | setPendingComplete | complete / task-terminal,rbac-write-scope |
| 4 | Open | complete | !guardCanComplete | Open (internal) | recordCompleteDenied | guard false branch |
| 5 | Open | cancel | guardCanCancel | persisting | setPendingCancel | cancel / task-terminal,rbac-write-scope |
| 6 | Open | cancel | !guardCanCancel | Open (internal) | recordCancelDenied | guard false branch |
| 7 | Open | reassign | guardCanReassign | persisting | setPendingReassign | reassign / task-assignee-visible,rbac-reassign-authority,rbac-write-scope |
| 8 | Open | reassign | !guardCanReassign | Open (internal) | recordReassignDenied | guard false branch |
| 9 | InProgress | start | - | InProgress (internal) | recordAlreadyStarted | idempotent no-op (already started) |
| 10 | InProgress | complete | guardCanComplete | persisting | setPendingComplete | complete / task-terminal,rbac-write-scope |
| 11 | InProgress | complete | !guardCanComplete | InProgress (internal) | recordCompleteDenied | guard false branch |
| 12 | InProgress | cancel | guardCanCancel | persisting | setPendingCancel | cancel / task-terminal,rbac-write-scope |
| 13 | InProgress | cancel | !guardCanCancel | InProgress (internal) | recordCancelDenied | guard false branch |
| 14 | InProgress | reassign | guardCanReassign | persisting | setPendingReassign | reassign / task-assignee-visible,rbac-reassign-authority,rbac-write-scope |
| 15 | InProgress | reassign | !guardCanReassign | InProgress (internal) | recordReassignDenied | guard false branch |
| 16 | Done | (any event) | - | none (final) | - | task-terminal (structural) |
| 17 | Cancelled | (any event) | - | none (final) | - | task-terminal (structural) |
| 18 | persisting | invoke onDone | pendingIsOpen | Open | commitStatus | reassign-in-Open success routing |
| 19 | persisting | invoke onDone | pendingIsInProgress | InProgress | commitStatus | start / reassign-in-InProgress routing |
| 20 | persisting | invoke onDone | pendingIsDone | Done | commitStatus | complete success routing |
| 21 | persisting | invoke onDone | pendingIsCancelled | Cancelled | commitStatus | cancel success routing |
| 22 | persisting | invoke onDone | (else) | rolledBack | recordRoutingError | defensive |
| 23 | persisting | invoke onError | isErrLocked | persistRetry | recordError | C4 6 store-locked |
| 24 | persisting | invoke onError | isErrConstraint | rolledBack | recordConstraint | C4 6 constraint |
| 25 | persisting | invoke onError | isErrDiskFull | rolledBack | recordDiskFull | C4 6 disk-full |
| 26 | persisting | invoke onError | isErrTimeout | rolledBack | recordTimeout | C4 6 timeout |
| 27 | persisting | invoke onError | (else) | rolledBack | recordUnknownError | catch-all |
| 28 | persisting | after persistTimeout | - | rolledBack | recordTimeout | C4 6 timeout 10s |
| 29 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted | C4 6 bound retry<=3 |
| 30 | persistRetry | after persistRetryBackoff | - | persisting | incrementRetries | C4 6 backoff ~0.5s |
| 31 | rolledBack | always | priorIsOpen | Open | - | atomic rollback |
| 32 | rolledBack | always | priorIsInProgress | InProgress | - | atomic rollback |

Ignored-by-design: `Done` and `Cancelled` are `final`; every event {start, complete, cancel, reassign} is structurally rejected there (this is `task-terminal`). All events are handled in `Open` and `InProgress`.
