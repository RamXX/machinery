# User machine - contract, failure catalog, and transition oracle

Component: `crm.domain` User aggregate (status lifecycle only). Machine: `User.machine.json`.
Placement (ARCHITECTURE.md 7): ephemeral in-process, load-act-save inside the one write Tx; status persists as the `status` field on its table. Concurrency: read-modify-write in one write Tx; cross-process serialized in SurrealDB's transaction engine (transient contention as ErrLocked -> CommandExecution/DBLocked).

States trace to enum `UserStatus`. Events trace to `User` actions `disable`/`enable` (both actor Admin). SCOPE: `register`/`changePassword`/`assignRole` are create/update paths owned by `crm.session` and the repo, not this machine; `login`/`logout` are the Session machine. Disabling a User has a downstream effect on live Sessions, enforced by `session-active-user` in the Session machine.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveUser` | actor | `(input{userId,status,actor}) -> UserRow \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: guard passed, tx open. post: row `status` atomically, or unchanged on err | C4 `crm.domain -> crm.repo` then `crm.repo -> store` (SurrealQL SaveUser) |
| `guardAdminAuthority` | guard | `(ctx,evt) -> bool` | true iff `actor.role == Admin` (the verb is granted) | inv `rbac-crud-verbs` (disable/enable are Admin actions) |
| `pendingIsActive` / `pendingIsDisabled` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) |
| `priorIsActive` / `priorIsDisabled` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status | - (rollback routing) |
| `isErrLocked` / `isErrConstraint` / `isErrDiskFull` / `isErrTimeout` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 section 6 failure classes |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 section 6 bound (retry <= 3, ~1.5s) |
| `setPendingDisable` | action | `(ctx) -> ctx` | `priorStatus:=status; pendingStatus:=Disabled` | - |
| `setPendingEnable` | action | `(ctx) -> ctx` | `priorStatus:=status; pendingStatus:=Active` | - |
| `commitStatus` | action | `(ctx) -> ctx` | `status:=pendingStatus` | - |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries:=retries+1` | - |
| `recordError` / `recordConstraint` / `recordDiskFull` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError:=classified error` for surfacing | maps repo errors to a domain error message |
| `recordAuthorityDenied` / `recordAlreadyActive` / `recordAlreadyDisabled` | action | `(ctx,evt) -> ctx` | set a rejection/no-op reason; no state change | surfaces `rbac-crud-verbs` denial or idempotent no-op |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Constraint violation on write | `saveUser` onError `isErrConstraint` | `persisting -> rolledBack -> priorStatus` | surface as a domain validation error; store unchanged | C4 6: one write Tx, no partial write. Residual: none |
| Disk full on write | `saveUser` onError `isErrDiskFull` | `persisting -> rolledBack -> priorStatus` | fail loudly; DB stays consistent | C4 6: atomic. Residual: user must free disk |
| Query/write timeout | `saveUser` onError `isErrTimeout` OR `after persistTimeout` (10s) | `persisting -> rolledBack -> priorStatus` | abort, surface, roll back | C4 6: SetTimeout 10s. Residual: none |
| Store locked by another writer | `saveUser` onError `isErrLocked` | `persisting -> persistRetry -> persisting` then `rolledBack` when `retriesExhausted` | bounded retry, then surface | C4 6: retry <= 3, ~1.5s. Residual: refused after 3 tries |
| Non-admin attempts disable/enable | guard `guardAdminAuthority` false | `Active`/`Disabled` internal, `recordAuthorityDenied` | reject; no write attempted | inv `rbac-crud-verbs`. Residual: none |

## (c) Transition matrix (hard-TDD oracle - one row per transition and per guard branch)

| # | source | event / after / always | guard | target | actions | derived-from |
|---|---|---|---|---|---|---|
| 1 | Active | disable | guardAdminAuthority | persisting | setPendingDisable | disable / rbac-crud-verbs |
| 2 | Active | disable | !guardAdminAuthority | Active (internal) | recordAuthorityDenied | guard false branch |
| 3 | Active | enable | - | Active (internal) | recordAlreadyActive | idempotent no-op |
| 4 | Disabled | enable | guardAdminAuthority | persisting | setPendingEnable | enable / rbac-crud-verbs |
| 5 | Disabled | enable | !guardAdminAuthority | Disabled (internal) | recordAuthorityDenied | guard false branch |
| 6 | Disabled | disable | - | Disabled (internal) | recordAlreadyDisabled | idempotent no-op |
| 7 | persisting | invoke onDone | pendingIsActive | Active | commitStatus | enable success routing |
| 8 | persisting | invoke onDone | pendingIsDisabled | Disabled | commitStatus | disable success routing |
| 9 | persisting | invoke onDone | (else) | rolledBack | recordRoutingError | defensive |
| 10 | persisting | invoke onError | isErrLocked | persistRetry | recordError | C4 6 store-locked |
| 11 | persisting | invoke onError | isErrConstraint | rolledBack | recordConstraint | C4 6 constraint |
| 12 | persisting | invoke onError | isErrDiskFull | rolledBack | recordDiskFull | C4 6 disk-full |
| 13 | persisting | invoke onError | isErrTimeout | rolledBack | recordTimeout | C4 6 timeout |
| 14 | persisting | invoke onError | (else) | rolledBack | recordUnknownError | catch-all |
| 15 | persisting | after persistTimeout | - | rolledBack | recordTimeout | C4 6 timeout 10s |
| 16 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted | C4 6 bound retry<=3 |
| 17 | persistRetry | after persistRetryBackoff | - | persisting | incrementRetries | C4 6 backoff ~0.5s |
| 18 | rolledBack | always | priorIsActive | Active | - | atomic rollback |
| 19 | rolledBack | always | priorIsDisabled | Disabled | - | atomic rollback |

Ignored-by-design: none. `disable` and `enable` are both handled in each resting state (a guarded persist or an explicit idempotent no-op / authority reject).
