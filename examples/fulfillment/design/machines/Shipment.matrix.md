# Shipment machine - named-unit contracts and failure catalog

Component: `shipping.service` Shipment aggregate. Machine: `Shipment.machine.json`.
Placement (ARCHITECTURE.md 4): a row in the Shipping DB, optimistic lock; the carrier is called with
bounded retry. Transitions are the generated oracle (`Shipment.oracle.md`); this document is the
named-unit contract table and the failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `carrierDispatch` | actor | `(shipmentId, orderId) -> trackingId \| err{retryable, rejected}` | books the parcel once (carrier keyed by shipmentId); returns the tracking id | C4 `shippingSvc -> carrier` | integration | carrier fake contract-tested against the carrier sandbox |
| `persistShipment` | actor | `(shipmentId, status) -> row \| err{ErrUnavailable,ErrConflict}` | writes the status row and the outbox event in one transaction, or neither | C4 `shippingSvc -> shippingDb`; inv `exactly-once-effect` | integration | real Postgres |
| `pendingIsDispatched` / `pendingIsInTransit` / `pendingIsDelivered` / `pendingIsLost` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) | unit | pure |
| `priorIsPending` / `priorIsDispatched` / `priorIsInTransit` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status | - (rollback routing) | unit | pure |
| `isErrRetryable` | guard | `(ctx,evt) -> bool` | true iff the carrier error is a 5xx or a timeout | C4 3 carrier failure classes | unit | pure |
| `carrierRetriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.carrierRetries >= 3` | C4 3 carrier bound | unit | pure |
| `isErrUnavailable` / `isErrConflict` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 3 DB failure classes | unit | pure |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 bound | unit | pure |
| `setCarrierDispatch` | action | `(ctx,evt) -> ctx` | `priorStatus := status; carrierRetries := 0` | - | unit | pure |
| `captureTrackingId` | action | `(ctx,evt) -> ctx` | `trackingId := evt.trackingId` | - | unit | pure |
| `setPendingDispatched` / `setPendingInTransit` / `setPendingDelivered` / `setPendingLost` | action | `(ctx,evt) -> ctx` | `pendingStatus := <that status>` | inv `shipment-terminal` (only legal successors) | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` | - | unit | pure |
| `incrementRetries` / `incrementCarrierRetries` | action | `(ctx) -> ctx` | increment the respective counter | - | unit | pure |
| `recordCarrierError` / `recordCarrierTimeout` / `recordCarrierExhausted` / `recordDispatchFailed` | action | `(ctx,evt) -> ctx` | `lastError := classified carrier outcome`; a failed dispatch leaves the shipment truthfully Pending for the saga to compensate | C4 3 carrier posture | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified repo error` | maps repo errors | unit | pure |

Structural: `shipment-terminal` is enforced by Delivered and Lost being final states.

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Carrier 5xx or timeout | `carrierDispatch` onError `isErrRetryable`, or `after carrierTimeout` (8s) | `dispatching -> carrierRetry -> dispatching` | bounded retry, same shipmentId | C4 3: retry <= 3. Residual: none (carrier keyed by shipmentId) |
| Carrier rejection or retries exhausted | `carrierDispatch` onError, or `carrierRetriesExhausted` | `-> rolledBack -> Pending`, `recordDispatchFailed` / `recordCarrierExhausted` | shipment stays Pending; the saga times out the step and compensates | C4 3: saga step timeout drives compensation |
| Lost parcel | tracking event `markLost` | `Dispatched/InTransit -> persisting -> Lost` | terminal Lost; the order receives fail | C4 3: explicit terminal. Residual: manual claim with the carrier |
| Shipping DB unavailable / conflict | `persistShipment` onError | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry; tracking events are redelivered by the bus | C4 3. Residual: row lags until redelivery lands |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> priorStatus` | abort; outbox guarantees no half-published event | C4 3. Residual: none |
