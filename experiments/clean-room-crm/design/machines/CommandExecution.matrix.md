# CommandExecution machine: named-unit contracts and failure catalog

Operational envelope (`_role: operational`), one instance per CLI invocation. Transitions are covered
by the generated `CommandExecution.oracle.md`. This file carries the units and the failure catalog.
Two invariants are enforced here as guards: `deactivated-cannot-login` (via `sessionActive`) and
`access-scope` (via `permitted`).

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `isCorrupt` | guard | `(err) -> bool` | true iff the open error is a failed integrity check (corruption), not a transient IO error | store corruption is fatal | unit | synthetic CorruptError |
| `sessionActive` | guard | `(session, user) -> bool` | true iff the session exists, is unexpired, and the user's `active` is true | invariant `deactivated-cannot-login` | unit | fake session and user |
| `permitted` | guard | `(user, target, verb) -> bool` | true iff the user's role, ownership, and team scope permit the verb on the target | invariant `access-scope` | unit | fake user, target, role matrix |
| `isConflict` | guard | `(err) -> bool` | true iff the execute error is an exhausted optimistic-lock conflict (the aggregate refused) | store conflict surfaced | unit | synthetic ConflictError |
| `isRejected` | guard | `(err) -> bool` | true iff the execute error is a domain RejectedError (an illegal move), not an infrastructure error | domain rejection | unit | synthetic RejectedError |
| `refreshSession` | action | `(ctx) -> ()` | extends the session file's expiry on a successful command | session survives invocations | unit | temp session file |
| `emitRejection` | action | `(ctx) -> ()` | writes the rejection reason to stderr and sets the authz/validation exit code | operator signal | unit | captured stderr |
| `emitBusyNotice` | action | `(ctx) -> ()` | writes "another command changed this record, please retry" to stderr, non-zero exit | operator signal for a refused conflict | unit | captured stderr |
| `emitCorruptAlert` | action | `(ctx) -> ()` | writes "database corrupted, restore from backup with `crm restore <file>`" to stderr, distinct exit | operator signal for corruption | unit | captured stderr |
| `emitError` | action | `(ctx) -> ()` | writes the internal error to stderr and sets the internal-error exit code | operator signal | unit | captured stderr |
| `openDb` | actor | `() -> Db` | opens the local database file and runs the integrity check; read-safe to retry | C4 rel: crm.repo to store | integration | real store plus a deliberately corrupted fixture file |
| `loadSession` | actor | `() -> Session` | reads and validates the local session file | C4 rel: crm.app reads the session file | integration | temp session file |
| `checkScope` | actor | `(target) -> Scope` | reads the target record's owner and team for the scope decision | C4 rel: crm.app to crm.repo | integration | contract-tested LadybugDB fake |
| `execute` | actor | `(cmd) -> Ack` | runs the domain operation, including the aggregate's own bounded persist retries; idempotency is the aggregate's version guard | C4 rel: crm.app to crm.domain and crm.repo | integration | contract-tested LadybugDB fake plus one real-store test |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| database file corrupted | `openDb` invoke `onError` with guard `isCorrupt` | opening to failedCorrupt | none: abort loudly, make no writes, tell the user to restore from a backup | residual: the FailedCorrupt (final) state; operator signal `emitCorruptAlert` with a distinct exit code |
| open IO error or timeout | `openDb` `onError` fallback, or `after OPEN_TIMEOUT` | opening to failedError | command aborts with an internal error | 3 s open timeout; residual FailedError, operator signal `emitError` |
| deactivated or expired session | `loadSession` `onDone` with guard `sessionActive` false | authenticating to rejected | command refuses; user must log in again | invariant `deactivated-cannot-login`; residual Rejected, operator signal `emitRejection` |
| session file unreadable or timeout | `loadSession` `onError`, or `after AUTH_TIMEOUT` | authenticating to failedError | command aborts | 2 s auth timeout; residual FailedError |
| not authorized for the verb or target | `checkScope` `onDone` with guard `permitted` false | authorizing to rejected | command refuses | invariant `access-scope`; residual Rejected |
| authorization read error or timeout | `checkScope` `onError`, or `after AUTHZ_TIMEOUT` | authorizing to failedError | command aborts | 2 s authz timeout; residual FailedError |
| write conflict the aggregate could not resolve | `execute` `onError` with guard `isConflict` | executing to refused | command politely refuses; the aggregate already rolled back and retried | bounded upstream by the aggregate's MaxRetries; residual Refused, operator signal `emitBusyNotice` |
| illegal domain move | `execute` `onError` with guard `isRejected` | executing to rejected | command returns the rejection reason | residual Rejected, operator signal `emitRejection` |
| execute timeout or unexpected error | `after EXEC_TIMEOUT` (to refused) or `onError` fallback (to failedError) | executing to refused or failedError | refuse or abort per the case | 5 s exec timeout; residual Refused or FailedError |
