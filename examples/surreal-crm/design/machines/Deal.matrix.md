# Deal machine - contract, failure catalog, and transition oracle

Component: `crm.domain` Deal aggregate. Machine: `Deal.machine.json`.
Placement (ARCHITECTURE.md 7): ephemeral in-process, load-act-save inside the one write Tx; stage persists as the `stage` field on the `deal` table. Concurrency: read-modify-write in one write Tx; cross-process serialized in SurrealDB's transaction engine (transient contention as ErrLocked -> CommandExecution/DBLocked).

States trace to enum `DealStage`. Events trace to `Deal` actions. Operational states (`persisting`, `persistRetry`, `rolledBack`) are the C4 persist overlay; the persist is atomic so failure returns to `context.priorStage`.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to |
|---|---|---|---|---|
| `saveDeal` | actor | `(input{dealId,stage,amountCents,closeDate,ownerId,actor}) -> DealRow \| err{ErrConstraint,ErrConflict,ErrDiskFull,ErrTimeout,ErrLocked}` | pre: guard passed, tx open. post: row `stage` = pendingStage atomically, or store unchanged on err | C4 `crm.domain -> crm.repo` then `crm.repo -> store` (SurrealQL SaveDeal) |
| `guardCanAdvance` | guard | `(ctx,evt) -> bool` | true iff pendingStage is the next forward stage AND actor may write AND amountCents>=0 | inv `deal-stage-forward`, `rbac-write-scope`, `deal-amount-nonneg` |
| `guardCanWin` | guard | `(ctx,evt) -> bool` | true iff evt supplies a closeDate AND actor may write AND amountCents>=0 | inv `deal-won-has-closedate`, `rbac-write-scope`, `deal-amount-nonneg` |
| `guardCanLose` | guard | `(ctx,evt) -> bool` | true iff actor may write AND amountCents>=0 | inv `rbac-write-scope`, `deal-amount-nonneg` |
| `guardCanReopen` | guard | `(ctx,evt) -> bool` | true iff actor is Manager/Admin acting in scope (the sanctioned backward move) | inv `rbac-reassign-authority`, `rbac-write-scope`; sanctioned exception to `deal-stage-forward` |
| `pendingIsQualified` / `pendingIsProposal` / `pendingIsNegotiation` / `pendingIsWon` / `pendingIsLost` | guard | `(ctx) -> bool` | true iff `ctx.pendingStage` equals that stage | - (persist success routing) |
| `priorIsLead` / `priorIsQualified` / `priorIsProposal` / `priorIsNegotiation` / `priorIsWon` / `priorIsLost` | guard | `(ctx) -> bool` | true iff `ctx.priorStage` equals that stage | - (rollback routing) |
| `isErrLocked` / `isErrConstraint` / `isErrDiskFull` / `isErrTimeout` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 section 6 failure classes |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 section 6 bound (retry <= 3, ~1.5s) |
| `setPendingAdvance` | action | `(ctx,evt) -> ctx` | `priorStage:=stage; pendingStage:=next(stage)` | - |
| `setPendingWin` | action | `(ctx,evt) -> ctx` | `priorStage:=stage; pendingStage:=Won; pendingCloseDate:=evt.closeDate` | - |
| `setPendingLose` | action | `(ctx,evt) -> ctx` | `priorStage:=stage; pendingStage:=Lost` | - |
| `setPendingReopen` | action | `(ctx,evt) -> ctx` | `priorStage:=stage; pendingStage:=Negotiation` | - |
| `commitStage` | action | `(ctx) -> ctx` | `stage:=pendingStage` (mirrors committed row) | - |
| `commitCloseDate` | action | `(ctx) -> ctx` | `closeDate:=pendingCloseDate` | supports `deal-won-has-closedate` |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries:=retries+1` | - |
| `recordError` / `recordConstraint` / `recordDiskFull` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError:=classified error` for surfacing to the CLI | maps repo errors to a domain validation/error message |
| `recordAdvanceDenied` / `recordWinDenied` / `recordLoseDenied` / `recordReopenDenied` / `recordReopenNotTerminal` / `recordTerminalRejected` | action | `(ctx,evt) -> ctx` | set a rejection reason; no state change | surfaces the violated invariant to the caller |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Constraint violation on write | `saveDeal` onError `isErrConstraint` | `persisting -> rolledBack -> priorStage` | surface as a domain validation error; store unchanged (atomic) | C4 6: one write Tx, no partial write. Residual: none |
| Disk full on write | `saveDeal` onError `isErrDiskFull` | `persisting -> rolledBack -> priorStage` | fail loudly; DB stays consistent | C4 6: atomic. Residual: user must free disk |
| Query/write timeout | `saveDeal` onError `isErrTimeout` OR `after persistTimeout` (10s) | `persisting -> rolledBack -> priorStage` | abort, surface, roll back | C4 6: `Connection.SetTimeout` 10s. Residual: none |
| Store locked by another writer | `saveDeal` onError `isErrLocked` | `persisting -> persistRetry -> persisting` (backoff) then `rolledBack` when `retriesExhausted` | bounded retry, then surface | C4 6: retry <= 3, ~1.5s. Residual: second concurrent writer refused after 3 tries |
| Write conflict | `saveDeal` onError (catch-all `recordUnknownError`) | `persisting -> rolledBack -> priorStage` | surface; user re-runs | Residual: rare in single-writer store; envelope also retries `ErrConflict` |
| Illegal domain move (backward, terminal, unauthorized, negative amount, missing closeDate) | guard `guardCan*` false, or terminal-state reject handler | internal self-transition, `record*Denied` / `recordTerminalRejected` | reject with the violated invariant id; no write attempted | structural: `deal-terminal`, `deal-stage-forward` enforced before any invoke |

## (c) Transition matrix (hard-TDD oracle - one row per transition and per guard branch)

| # | source | event / after / always | guard | target | actions | derived-from |
|---|---|---|---|---|---|---|
| 1 | Lead | advanceStage | guardCanAdvance | persisting | setPendingAdvance | advanceStage / deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| 2 | Lead | advanceStage | !guardCanAdvance | Lead (internal) | recordAdvanceDenied | guard false branch |
| 3 | Lead | win | guardCanWin | persisting | setPendingWin | win / deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| 4 | Lead | win | !guardCanWin | Lead (internal) | recordWinDenied | guard false branch |
| 5 | Lead | lose | guardCanLose | persisting | setPendingLose | lose / rbac-write-scope,deal-amount-nonneg |
| 6 | Lead | lose | !guardCanLose | Lead (internal) | recordLoseDenied | guard false branch |
| 7 | Lead | reopen | - | Lead (internal) | recordReopenNotTerminal | deal-terminal (reopen only on terminal) |
| 8 | Qualified | advanceStage | guardCanAdvance | persisting | setPendingAdvance (-> Proposal) | deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| 9 | Qualified | advanceStage | !guardCanAdvance | Qualified (internal) | recordAdvanceDenied | guard false branch |
| 10 | Qualified | win | guardCanWin | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| 11 | Qualified | win | !guardCanWin | Qualified (internal) | recordWinDenied | guard false branch |
| 12 | Qualified | lose | guardCanLose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| 13 | Qualified | lose | !guardCanLose | Qualified (internal) | recordLoseDenied | guard false branch |
| 14 | Qualified | reopen | - | Qualified (internal) | recordReopenNotTerminal | deal-terminal |
| 15 | Proposal | advanceStage | guardCanAdvance | persisting | setPendingAdvance (-> Negotiation) | deal-stage-forward,rbac-write-scope,deal-amount-nonneg |
| 16 | Proposal | advanceStage | !guardCanAdvance | Proposal (internal) | recordAdvanceDenied | guard false branch |
| 17 | Proposal | win | guardCanWin | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| 18 | Proposal | win | !guardCanWin | Proposal (internal) | recordWinDenied | guard false branch |
| 19 | Proposal | lose | guardCanLose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| 20 | Proposal | lose | !guardCanLose | Proposal (internal) | recordLoseDenied | guard false branch |
| 21 | Proposal | reopen | - | Proposal (internal) | recordReopenNotTerminal | deal-terminal |
| 22 | Negotiation | advanceStage | - | Negotiation (internal) | recordAdvanceDenied | deal-stage-forward (no forward stage; win/lose only) |
| 23 | Negotiation | win | guardCanWin | persisting | setPendingWin | deal-won-has-closedate,rbac-write-scope,deal-amount-nonneg |
| 24 | Negotiation | win | !guardCanWin | Negotiation (internal) | recordWinDenied | guard false branch |
| 25 | Negotiation | lose | guardCanLose | persisting | setPendingLose | rbac-write-scope,deal-amount-nonneg |
| 26 | Negotiation | lose | !guardCanLose | Negotiation (internal) | recordLoseDenied | guard false branch |
| 27 | Negotiation | reopen | - | Negotiation (internal) | recordReopenNotTerminal | deal-terminal |
| 28 | Won | reopen | guardCanReopen | persisting | setPendingReopen (-> Negotiation) | reopen / rbac-reassign-authority,rbac-write-scope |
| 29 | Won | reopen | !guardCanReopen | Won (internal) | recordReopenDenied | guard false branch |
| 30 | Won | advanceStage | - | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| 31 | Won | win | - | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| 32 | Won | lose | - | Won (internal) | recordTerminalRejected | deal-terminal (structural) |
| 33 | Lost | reopen | guardCanReopen | persisting | setPendingReopen (-> Negotiation) | reopen / rbac-reassign-authority,rbac-write-scope |
| 34 | Lost | reopen | !guardCanReopen | Lost (internal) | recordReopenDenied | guard false branch |
| 35 | Lost | advanceStage | - | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| 36 | Lost | win | - | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| 37 | Lost | lose | - | Lost (internal) | recordTerminalRejected | deal-terminal (structural) |
| 38 | persisting | invoke onDone | pendingIsQualified | Qualified | commitStage | persist success routing |
| 39 | persisting | invoke onDone | pendingIsProposal | Proposal | commitStage | persist success routing |
| 40 | persisting | invoke onDone | pendingIsNegotiation | Negotiation | commitStage | persist success routing |
| 41 | persisting | invoke onDone | pendingIsWon | Won | commitStage, commitCloseDate | persist success; deal-won-has-closedate |
| 42 | persisting | invoke onDone | pendingIsLost | Lost | commitStage | persist success routing |
| 43 | persisting | invoke onDone | (else) | rolledBack | recordRoutingError | defensive |
| 44 | persisting | invoke onError | isErrLocked | persistRetry | recordError | C4 6 store-locked |
| 45 | persisting | invoke onError | isErrConstraint | rolledBack | recordConstraint | C4 6 constraint |
| 46 | persisting | invoke onError | isErrDiskFull | rolledBack | recordDiskFull | C4 6 disk-full |
| 47 | persisting | invoke onError | isErrTimeout | rolledBack | recordTimeout | C4 6 timeout |
| 48 | persisting | invoke onError | (else) | rolledBack | recordUnknownError | catch-all |
| 49 | persisting | after persistTimeout | - | rolledBack | recordTimeout | C4 6 timeout 10s |
| 50 | persistRetry | always | retriesExhausted | rolledBack | recordRetriesExhausted | C4 6 bound retry<=3 |
| 51 | persistRetry | after persistRetryBackoff | - | persisting | incrementRetries | C4 6 backoff ~0.5s |
| 52 | rolledBack | always | priorIsLead | Lead | - | atomic rollback |
| 53 | rolledBack | always | priorIsQualified | Qualified | - | atomic rollback |
| 54 | rolledBack | always | priorIsProposal | Proposal | - | atomic rollback |
| 55 | rolledBack | always | priorIsNegotiation | Negotiation | - | atomic rollback |
| 56 | rolledBack | always | priorIsWon | Won | - | atomic rollback |
| 57 | rolledBack | always | priorIsLost | Lost | - | atomic rollback |

Ignored-by-design: none. Every event in {advanceStage, win, lose, reopen} is handled in every resting stage (either a guarded persist or an explicit `record*` reject). `persisting`/`persistRetry`/`rolledBack` are transient (invoke/after/always), so domain events cannot arrive there within one command.
