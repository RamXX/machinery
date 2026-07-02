# Order machine - named-unit contracts and failure catalog

Component: `order.service` Order aggregate. Machine: `Order.machine.json`.
Placement (ARCHITECTURE.md 4): a status row in the Order DB, optimistic lock, mutations persist in one
transaction with the outbox event. Transitions are the generated oracle (`Order.oracle.md`); this
document is the named-unit contract table and the failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `persistOrder` | actor | `(orderId, status) -> row \| err{ErrUnavailable,ErrConflict}` | writes the status row AND the outbox event in one transaction, or neither (the transactional outbox) | C4 `orderRepo -> orderDb`; inv `exactly-once-effect` | integration | real Postgres (docker compose) |
| `guardCanConfirm` | guard | `(ctx,evt) -> bool` | true iff the order has line items and `totalCents` matches their sum | inv `order-total-matches-items` | unit + property | pure; property over generated line-item lists |
| `guardCanCancel` | guard | `(ctx,evt) -> bool` | true iff the caller owns the order | inv `order-owned-by-customer` | unit | pure |
| `pendingIsConfirmed` / `pendingIsReserved` / `pendingIsPaid` / `pendingIsShipped` / `pendingIsDelivered` / `pendingIsCancelled` / `pendingIsFailed` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) | unit | pure |
| `priorIsPending` / `priorIsConfirmed` / `priorIsReserved` / `priorIsPaid` / `priorIsShipped` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status | - (rollback routing) | unit | pure |
| `isErrUnavailable` / `isErrConflict` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 3 per-service DB failure classes | unit | pure |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 bound | unit | pure |
| `setPendingConfirmed` / `setPendingReserved` / `setPendingPaid` / `setPendingShipped` / `setPendingDelivered` / `setPendingCancelled` / `setPendingFailed` | action | `(ctx,evt) -> ctx` | `priorStatus := status; pendingStatus := <that status>`; enforces the forward order | inv `order-forward` | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` (mirrors the committed row) | - | unit | pure |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | pure |
| `recordConfirmDenied` / `recordCancelDenied` | action | `(ctx,evt) -> ctx` | set the rejection reason naming the violated invariant | surfaces `order-total-matches-items`, `order-owned-by-customer` | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified error` | maps repo errors | unit | pure |

Structural: `order-delivered-terminal` is enforced by Delivered, Cancelled, and Failed being final
states (any event there is rejected structurally); `order-forward` is enforced by each domain state
exposing only its forward saga event plus cancel/fail.

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Order DB unavailable | `persistOrder` onError `isErrUnavailable` | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry, then surface; the saga event is redelivered by the bus | C4 3: HA Postgres, retry <= 3. Residual: order lags the saga until redelivery lands |
| Optimistic-lock conflict | `persistOrder` onError `isErrConflict` | as above | reload and retry within the bound | C4 3. Residual: none (single winner) |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> priorStatus` | abort the attempt; the outbox guarantees no half-published event | C4 3: one transaction. Residual: none |
| Out-of-order or stale saga event | `_ignores` entry on the resting state | none (explicit ignore, logged) | the bus redelivers until the preceding event lands | dedupe by message id; ordering converges |
