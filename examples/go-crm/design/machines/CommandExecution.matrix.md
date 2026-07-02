# CommandExecution machine - contract, failure catalog, and transition oracle

Component: `crm.commands` (the operational envelope). Machine: `CommandExecution.machine.json`.
Placement (ARCHITECTURE.md 7): ephemeral per invocation, no persistence of its own; it owns the single write Tx. Concurrency: one invocation owns the write Tx; cross-process serialization is LadybugDB's single-writer lock (ErrLocked -> bounded-retry `DBLocked`).

This is the home of the LadybugDB open/write/timeout failure rows from ARCHITECTURE.md section 6. `Parsing`, `Authorizing`, and `Rendering` are pure (no I/O) so they use `always`, not invokes; `Authorizing` is the single call site of the pure `crm.authz` decision.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `openDatabase` | actor | `(input{dbPath}) -> Tx \| err{ErrLocked,ErrCorrupt,ErrUnavailable}` | post: DB open for writing, or typed error | C4 `crm.commands -> crm.repo` then `crm.repo -> store` (Repo.Open) |
| `resolveSession` | actor | `(input{argv}) -> Actor \| err{ErrNoSession,ErrExpired,ErrLocked,ErrUnavailable}` | post: current User resolved (delegates to the Session machine) | C4 `crm.commands -> crm.session` (Sessions.Current) |
| `executeInTx` | actor | `(input{verb,entityType,actor}) -> Result \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: authorized, tx begun. post: BEGIN -> domain mutation (aggregate machine + SaveX) -> COMMIT atomically; on err rolled back, no partial write | C4 `crm.commands -> crm.repo` (Tx boundary) with `crm.domain -> crm.repo -> store` |
| `guardParseOk` | guard | `(ctx,evt) -> bool` | true iff argv parses to a valid (verb, entity, flags) | - (input validation) |
| `guardAuthorized` | guard | `(ctx,evt) -> bool` | true iff the pure authz `Decision.Allowed` for (actor,verb,entityType,ownerId,teamId) | inv `rbac-crud-verbs`, `rbac-read-visibility`, `rbac-write-scope`, `rbac-reassign-authority` |
| `phaseIsOpen` / `phaseIsExecute` | guard | `(ctx) -> bool` | true iff `ctx.phase` is that phase (routes the retry to the right step) | - |
| `isErrLocked` / `isErrCorrupt` / `isErrUnavailable` / `isErrNoSession` / `isErrExpired` / `isErrConstraint` / `isErrConflict` / `isErrDiskFull` / `isErrTimeout` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed error | C4 sections 5/6 error types |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 section 6 bound (retry <= 3, ~1.5s) |
| `captureArgs` | action | `(ctx,evt) -> ctx` | `verb,entityType,targetOwnerId,targetTeamId := parsed argv` | - |
| `setPhaseOpen` / `setPhaseExecute` | action | `(ctx) -> ctx` | `phase := open \| execute` (entry action) | - |
| `captureTx` / `captureActor` / `captureResult` | action | `(ctx,evt) -> ctx` | record the tx handle / resolved actor / result for rendering | - |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries:=retries+1` | - |
| `ensureRolledBack` | action | `(ctx) -> ctx` | roll back the write tx (idempotent; store guarantees no partial write) | C4 6 atomicity |
| `renderOutput` | action | `(ctx) -> ctx` | format tables/JSON to stdout (entry of Rendering) | - |
| `recordAllowed` / `recordDenyReason` | action | `(ctx,evt) -> ctx` | record the authz outcome | rbac-* surfacing |
| `recordParseError` / `recordError` / `recordCorrupt` / `recordUnavailable` / `recordOpenError` / `recordNeedLogin` / `recordSessionError` / `recordConstraint` / `recordConflict` / `recordDiskFull` / `recordTimeout` / `recordExecuteError` / `recordLockExhausted` | action | `(ctx,evt) -> ctx` | `lastError:=classified error` | maps parse/repo/session errors |
| `recordSuccessExit` / `recordDeniedExit` / `recordValidationExit` / `recordDBErrorExit` / `recordCorruptExit` | action | `(ctx) -> ctx` | set process `exitCode` (entry of each terminal state) | CLI exit contract |

## (b) Failure catalog (every ARCHITECTURE.md section 6 dependency row lands here or in Session)

| failure (section 6 row) | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| DB open: file locked by another `crm` | `openDatabase` onError `isErrLocked` | `Opening -> DBLocked -> Opening` (backoff) then `DBError` when `retriesExhausted` | bounded retry then clear message | C4 6: retry <= 3, ~1.5s. Residual: exit with "database busy" |
| DB open: corrupt / version-incompatible | `openDatabase` onError `isErrCorrupt` | `Opening -> Corrupt` (final) | fail loudly; tell user to `crm restore` | C4 6: fatal, no auto-recovery. Residual: restore from backup |
| DB open: unavailable / open timeout | `openDatabase` onError `isErrUnavailable` / `after openTimeout` | `Opening -> DBError` (final) | fail loudly | Residual: environment/permissions issue |
| Session file missing / expired | `resolveSession` onError `isErrNoSession` / `isErrExpired` | `ResolvingSession -> Denied` (final) | require `crm login` | C4 6: user re-authenticates. Residual: none |
| Store locked during session resolve | `resolveSession` onError `isErrLocked` | `ResolvingSession -> DBLocked` (phase=open) | bounded retry | C4 6: retry <= 3. Residual: as open-lock |
| Session resolve unavailable / timeout | `resolveSession` onError (else) / `after sessionResolveTimeout` | `ResolvingSession -> DBError` (final) | fail loudly | Residual: none |
| Authorization denied | `guardAuthorized` false | `Authorizing -> Denied` (final) | none; caller lacks the verb/scope | inv `rbac-*`. Residual: none |
| DB write: constraint / Cypher violation | `executeInTx` onError `isErrConstraint` | `Executing -> ValidationFailed` (final), `ensureRolledBack` | surface as a domain validation error | C4 6: one write Tx, no partial write. Residual: none |
| DB write: disk full | `executeInTx` onError `isErrDiskFull` | `Executing -> DBError` (final), `ensureRolledBack` | fail loudly; DB stays consistent | C4 6: atomic. Residual: free disk |
| DB query/write: runaway / timeout | `executeInTx` onError `isErrTimeout` / `after queryTimeout` (10s) | `Executing -> DBError` (final), `ensureRolledBack` | abort, surface, roll back | C4 6: SetTimeout + Interrupt, 10s. Residual: none |
| DB write: locked / conflict mid-tx | `executeInTx` onError `isErrLocked` / `isErrConflict` | `Executing -> DBLocked` (phase=execute) -> retry `Executing` | bounded retry of the whole tx | C4 6: retry <= 3. Residual: refused after 3 |
| Bad CLI args | `guardParseOk` false | `Parsing -> ValidationFailed` (final) | show usage/help | Residual: none |

## (c) Transition matrix (hard-TDD oracle - one row per transition and per guard branch)

| # | source | event / after / always | guard | target | actions | derived-from |
|---|---|---|---|---|---|---|
| 1 | Parsing | always | guardParseOk | Opening | captureArgs | arg validation ok |
| 2 | Parsing | always | !guardParseOk | ValidationFailed | recordParseError | bad args |
| 3 | Opening | invoke onDone | - | ResolvingSession | captureTx | db open ok |
| 4 | Opening | invoke onError | isErrLocked | DBLocked | recordError | C4 6 open-lock |
| 5 | Opening | invoke onError | isErrCorrupt | Corrupt | recordCorrupt | C4 6 corrupt (fatal) |
| 6 | Opening | invoke onError | isErrUnavailable | DBError | recordUnavailable | C4 6 unavailable |
| 7 | Opening | invoke onError | (else) | DBError | recordOpenError | catch-all |
| 8 | Opening | after openTimeout | - | DBError | recordTimeout | open timeout 5s |
| 9 | DBLocked | always | retriesExhausted | DBError | recordLockExhausted | C4 6 bound retry<=3 |
| 10 | DBLocked | after dbRetryBackoff | phaseIsOpen | Opening | incrementRetries | retry the open (~0.5s) |
| 11 | DBLocked | after dbRetryBackoff | phaseIsExecute | Executing | incrementRetries | retry the write tx (~0.5s) |
| 12 | ResolvingSession | invoke onDone | - | Authorizing | captureActor | session resolved |
| 13 | ResolvingSession | invoke onError | isErrNoSession | Denied | recordNeedLogin | C4 6 no session |
| 14 | ResolvingSession | invoke onError | isErrExpired | Denied | recordNeedLogin | C4 6 expired |
| 15 | ResolvingSession | invoke onError | isErrLocked | DBLocked | recordError | C4 6 open-lock (session hits repo) |
| 16 | ResolvingSession | invoke onError | (else) | DBError | recordSessionError | unavailable |
| 17 | ResolvingSession | after sessionResolveTimeout | - | DBError | recordTimeout | session resolve timeout 5s |
| 18 | Authorizing | always | guardAuthorized | Executing | recordAllowed | rbac-crud-verbs, rbac-read-visibility, rbac-write-scope, rbac-reassign-authority |
| 19 | Authorizing | always | !guardAuthorized | Denied | recordDenyReason | authz deny |
| 20 | Executing | invoke onDone | - | Rendering | captureResult | commit ok |
| 21 | Executing | invoke onError | isErrConstraint | ValidationFailed | ensureRolledBack, recordConstraint | C4 6 constraint |
| 22 | Executing | invoke onError | isErrLocked | DBLocked | ensureRolledBack, recordError | C4 6 write-lock |
| 23 | Executing | invoke onError | isErrConflict | DBLocked | ensureRolledBack, recordConflict | C4 6 conflict |
| 24 | Executing | invoke onError | isErrDiskFull | DBError | ensureRolledBack, recordDiskFull | C4 6 disk-full |
| 25 | Executing | invoke onError | isErrTimeout | DBError | ensureRolledBack, recordTimeout | C4 6 timeout |
| 26 | Executing | invoke onError | (else) | DBError | ensureRolledBack, recordExecuteError | catch-all |
| 27 | Executing | after queryTimeout | - | DBError | ensureRolledBack, recordTimeout | C4 6 query timeout 10s |
| 28 | Rendering | always | - | Done | (entry renderOutput) | render + exit 0 |
| 29 | Done | (final) | - | - | recordSuccessExit | success exit |
| 30 | Denied | (final) | - | - | recordDeniedExit | authn/authz exit |
| 31 | ValidationFailed | (final) | - | - | recordValidationExit | validation exit |
| 32 | DBError | (final) | - | - | recordDBErrorExit | db-error exit |
| 33 | Corrupt | (final) | - | - | recordCorruptExit | fatal exit (restore) |

Event completeness: this machine is driven entirely by `always` / `invoke` outcomes / `after` (no external user events), so every non-final state auto-advances and there are no unhandled events. The five terminal states are `final` (process exit).
