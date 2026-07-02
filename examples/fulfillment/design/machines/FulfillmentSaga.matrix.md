# FulfillmentSaga machine - named-unit contracts and failure catalog

Component: `order.service` Saga Orchestrator. Machine: `FulfillmentSaga.machine.json`.
Placement (ARCHITECTURE.md 4): a persistent `gen_statem` per order under a `Registry`, state journaled
to the Order DB. Transitions are the generated oracle (`FulfillmentSaga.oracle.md`); this document is
the named-unit contract table and the failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `reserveInventory` | actor | `(orderId) -> ok \| err` | emits the reserve command via the outbox; ok when the reservation is Held. Idempotent by orderId | C4 `saga -> outbox -> bus -> inventorySvc` | integration | real bus + inventory service (docker compose) |
| `capturePayment` | actor | `(orderId) -> ok \| err` | emits capture via the outbox; ok when payment is Captured. Charges once: idempotency key = orderId | C4 `saga -> outbox -> bus -> paymentSvc`; inv `payment-idempotent`, `capture-matches-total` | integration + property | real bus + payment service with gateway fake (contract-tested against stripe sandbox) |
| `dispatchShipment` | actor | `(orderId) -> ok \| err` | emits dispatch via the outbox; ok when the shipment is Dispatched | C4 `saga -> outbox -> bus -> shippingSvc`; inv `no-ship-before-pay` | integration | real bus + shipping service with carrier fake |
| `compensate` | actor | `(orderId) -> ok \| err` | single idempotent step: refund if captured, release if reserved; ok only when every held obligation is undone | inv `saga-compensation`, `refund-within-capture` | integration + property | real bus + both services; property: compensate twice = compensate once |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 compensation bound | unit | pure |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | pure |
| `markReserved` / `markPaid` / `markShipped` | action | `(ctx,evt) -> ctx` | record the completed forward step (drives the Order aggregate's saga events) | Order actions of the same name | unit | pure |
| `recordReserveFailed` / `recordReserveTimeout` / `recordPayFailed` / `recordPayTimeout` / `recordShipFailed` / `recordShipTimeout` | action | `(ctx,evt) -> ctx` | `lastError := classified step failure` | C4 3 step timeouts | unit | pure |
| `recordCompensated` / `recordCompensateError` / `recordCompensateTimeout` / `recordCompensationIncomplete` | action | `(ctx,evt) -> ctx` | record the compensation outcome; `recordCompensationIncomplete` marks the FailedDirty residual and MUST page an operator (ARCHITECTURE.md 6) | inv `saga-compensation` | unit | pure |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Reserve fails or times out | `reserveInventory` onError, or `after reserveTimeout` (5s) | `Reserving -> Failed` | nothing held yet, terminal Failed | C4 3: nothing to compensate. Residual: none |
| Payment fails or times out | `capturePayment` onError, or `after payTimeout` (8s) | `Paying -> Compensating` | release the reservation | C4 3: refund/release idempotent. Residual: none if compensation completes |
| Dispatch fails or times out | `dispatchShipment` onError, or `after shipTimeout` (8s) | `Shipping -> Compensating` | refund the capture, release the reservation | C4 3: idempotency key. Residual: none if compensation completes |
| Compensation fails or times out | `compensate` onError, or `after compensateTimeout` (8s) | `Compensating -> compensateRetry -> Compensating` (backoff 0.5s) | bounded retry | retry <= 3 |
| Compensation exhausted | `retriesExhausted` at 3 | `compensateRetry -> FailedDirty` | none automatic; explicit residual | FailedDirty MUST page an operator; money or stock may be held pending manual action |

Formal note: `formal/FulfillmentSagaData.tla` (generated from `FulfillmentSaga.semantics.yaml`)
proves that a terminal saga never silently holds an obligation: refunds what it captured and
releases what it reserved, or ends FailedDirty explicitly.
