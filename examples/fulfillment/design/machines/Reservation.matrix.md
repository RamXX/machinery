# Reservation machine - named-unit contracts and failure catalog

Component: `inventory.service` Reservation aggregate. Machine: `Reservation.machine.json`.
Placement (ARCHITECTURE.md 4): a row in the Inventory DB, optimistic lock. Transitions are the
generated oracle (`Reservation.oracle.md`); this document is the named-unit contract table and the
failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `persistReservation` | actor | `(reservationId, status) -> row \| err{ErrUnavailable,ErrConflict}` | writes the status row, the stock counters, and the outbox event in one transaction, or nothing: committing moves `reserved -> onHand` deduction, releasing returns the hold | C4 `inventorySvc -> inventoryDb`; inv `reserved-within-stock`, `available-nonneg`, `exactly-once-effect` | integration + property | real Postgres; property: concurrent holds never oversell the last unit |
| `pendingIsCommitted` / `pendingIsReleased` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) | unit | pure |
| `priorIsHeld` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus = Held` (the only overlay entry point) | - (rollback routing) | unit | pure |
| `isErrUnavailable` / `isErrConflict` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 3 DB failure classes | unit | pure |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 bound | unit | pure |
| `setPendingCommitted` / `setPendingReleased` | action | `(ctx,evt) -> ctx` | `priorStatus := status; pendingStatus := <that status>` | inv `reservation-terminal` (only legal successors of Held) | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` | - | unit | pure |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified error` | maps repo errors | unit | pure |

Structural: `reservation-terminal` is enforced by Committed and Released being final;
`reservation-quantity-positive` is a creation-time (hold) validation, tested on the aggregate factory.

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Inventory DB unavailable / conflict | `persistReservation` onError | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry; the saga command is redelivered by the bus | C4 3. Residual: reservation lags until redelivery lands |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> Held` | abort; nothing committed | C4 3. Residual: none |
