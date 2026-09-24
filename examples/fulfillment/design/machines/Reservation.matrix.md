# Reservation machine - named-unit contracts and failure catalog

Component: `inventory.service` Reservation aggregate. Machine: `Reservation.machine.json`.
Placement (ARCHITECTURE.md 5): a row in the Inventory DB, optimistic lock. Transitions are the
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
| `resetRetries` | action | `(ctx) -> ctx` | `retries := 0` before each independent commit or release persistence operation | C4 3 per-operation DB retry bound | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` | - | unit | pure |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified error` | maps repo errors | unit | pure |
| `validateProductPrice` | invariant | `(priceCents) -> bool` | accept a catalog write only when `priceCents >= 0` | inv `product-price-nonneg`; Product create/reprice validation | property | generated integer prices at the Inventory Service API boundary |
| `validateReservationQuantity` | invariant | `(quantity) -> bool` | construct a Held reservation only when `quantity > 0` | inv `reservation-quantity-positive`; Reservation hold factory | property | generated integer quantities |

Structural: `reservation-terminal` is enforced by Committed and Released being final;
`reservation-quantity-positive` is a creation-time (hold) validation, tested on the aggregate factory.

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Inventory DB unavailable / conflict | `persistReservation` onError | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry; the saga command is redelivered by the bus | C4 3. Residual: reservation lags until redelivery lands |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> Held` | abort; nothing committed | C4 3. Residual: none |
| Corrupt or absent rollback route | `priorIsHeld` is false after a persistence failure | `rolledBack -> routingFault` | stop and alert; never manufacture a terminal reservation state | explicit terminal residual; stock and row stay available for reconciliation |

## (c) Consumed events

The event-contract table (ARCHITECTURE.md 7) arms the consumer-READS completeness tier, so each
command this aggregate consumes states which payload fields it reads; Gx-trace holds every declared
field to the payload cell of the row that carries the event.

| event | reacting unit | payload reads |
|---|---|---|
| `reserve` command | receipt creates the hold, then `persistReservation` | READS{LineItem.quantity, Product.sku} |
| `release` command | `setPendingReleased`, then `persistReservation` | READS{Reservation} |
