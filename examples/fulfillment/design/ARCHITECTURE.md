# Architecture: Order Fulfillment

The narrative twin of `workspace.dsl`. Data shapes are the single source of truth in
`fulfillment.modelith.yaml`; this fixes the how, and above all what each dependency does when it fails.

## 1. Shape and deployment

Four services (order, inventory, payment, shipping), each with its own PostgreSQL, communicating only
asynchronously over a message bus. The Order Service owns the fulfillment saga, a persistent process
per order. Reliable delivery is a transactional outbox (events written in the same transaction as the
state change) plus idempotent consumers, so at-least-once delivery yields an exactly-once effect. There
are no synchronous service-to-service calls; the contract forbids them.

Target language: Elixir/OTP. The saga is a `gen_statem` per order under a `Registry`, its state journaled
to the Order DB, which is a near 1:1 realization of the FulfillmentSaga machine.

## 2. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: order.service
    kind: container
    element: orderSvc
    code: [ "apps/order/**" ]
    exposes: [ "apps/order/lib/order_web/**" ]
    modules: [ "Order", "OrderWeb" ]
  - id: inventory.service
    kind: container
    element: inventorySvc
    code: [ "apps/inventory/**" ]
    modules: [ "Inventory" ]
  - id: payment.service
    kind: container
    element: paymentSvc
    code: [ "apps/payment/**" ]
    modules: [ "Payment" ]
  - id: shipping.service
    kind: container
    element: shippingSvc
    code: [ "apps/shipping/**" ]
    modules: [ "Shipping" ]
  - id: shared.contracts
    kind: component
    element: contracts
    code: [ "libs/contracts/**" ]
    modules: [ "Contracts" ]
dependency_rules:
  allow:
    - order.service -> shared.contracts
    - inventory.service -> shared.contracts
    - payment.service -> shared.contracts
    - shipping.service -> shared.contracts
  deny:
    - "order.service -> inventory.service"
    - "order.service -> payment.service"
    - "order.service -> shipping.service"
  notes:
    - "Cross-service communication is asynchronous over the message bus, never a direct call."
    - "Reliable delivery is the transactional outbox plus idempotent consumers (exactly-once effect)."
```

## 3. Dependency mitigation posture (drives the saga and aggregate failure transitions)

Distributed, so the failure model is partial failure and message loss, not a single local store. This is
where this design differs most from go-crm, and where the FulfillmentSaga's compensation earns its keep.

| dependency | failure modes | mitigation | residual behavior the machines handle | bound |
|---|---|---|---|---|
| `bus` (message bus) | unavailable, partition, duplicate | transactional outbox (at-least-once), clustered bus | duplicate delivery handled by idempotent consumers; delayed delivery becomes a saga step timeout then compensation | dedupe by message id; step timeout 5-8s |
| `stripe` (payment gateway) | 5xx, timeout, duplicate charge | idempotency key, bounded retry | a timeout compensates (refund is idempotent and retried); a partial capture is reconciled by the refund | idempotency key; capture retried |
| `orderDb` `inventoryDb` `paymentDb` `shippingDb` (per-service DB) | unavailable, conflict | HA Postgres, the outbox | transient unavailability retries; the outbox guarantees no event is lost on a crash | retry <= 3 |
| `carrier` | 5xx, timeout, lost parcel | retry, tracking | a dispatch timeout compensates; a lost parcel is the terminal Shipment Lost | timeout 8s |
| `bus` (network partition, service to bus) | messages delayed | outbox retry, saga step timeout | a step that does not confirm in time drives the saga into compensation | saga step timeout |

The saga's compensation is a single idempotent step (refund if captured, release if reserved), retried;
if it cannot complete within the bound the saga ends in the explicit residual FailedDirty. The data-refined
model `formal/FulfillmentSagaData.tla` proves that money and stock are never silently lost.

## 4. Interface contracts

The contract allows exactly four crossings, all into the shared contracts library. Everything else
between services is asynchronous and governed by the event-contract table in section 6: a bus message
is not an edge in the dependency graph, which is precisely why that table exists.

| edge | shape | errors | idempotency |
|---|---|---|---|
| `order.service -> shared.contracts`, `inventory.service -> shared.contracts`, `payment.service -> shared.contracts`, `shipping.service -> shared.contracts` | type-only: the `Contracts` modules holding the event structs and their encode/decode functions, built from `fulfillment.modelith.yaml` attributes; no I/O and no process | a decode failure surfaces at the consuming service's own poison-message path, never inside the library | pure: encoding the same event twice yields the same bytes, so a redelivered message decodes identically |

## 5. Persistence and placement

| component | placement | persistence | concurrency serialization |
|---|---|---|---|
| `FulfillmentSaga` | a persistent process per order (`gen_statem` under a `Registry`) | saga state row plus the outbox in the Order DB | one process per order (actor mailbox) |
| `Order` `Payment` `Reservation` `Shipment` | per-service DB rows | status columns per aggregate | optimistic lock per row |
| `OutboxMessage` | a table in each service DB, drained by a poller | rows marked Published then Consumed; the machine models one publish attempt, the poller supplies the at-least-once re-drive | at-least-once; consumers dedupe by message id |
| `Customer` (no machine: a registered party record with no lifecycle) | none; rows in the Order Service DB | one row per customer, unique by email (`customer-email-unique`) | last write wins per customer row |
| `Product` `Inventory` (no machine: stock is a counter pair held by invariants, not by states) | none; rows in the Inventory Service DB | catalog row per product; onHand and reserved counters per stock position | optimistic lock per inventory row |
| `LineItem` (no machine: rows owned by their `Order`, written with it) | none; rows in the Order Service DB | rows keyed by order id | written inside the order's own write |
| `Refund` (no machine: the outcome record of a compensating capture) | none; rows in the Payment Service DB | one row per issued refund | written inside the payment's compensation step |
| `Address` (not placed: a value object stored inline in the `Shipment` row it belongs to) | n/a | n/a | n/a |

## 6. Event-contract table

Bus coupling is invisible to import-level checking, so this table is the governing artifact for
every message that crosses a service boundary. Every row rides the transactional outbox of the
producing service (`OutboxMessage`: Pending, Published, Consumed; the poller re-drives until
Consumed, `outbox-at-least-once`), so delivery is at least once everywhere and every consumer
dedupes by message id (`exactly-once-effect`). Payloads name `fulfillment.modelith.yaml`
attributes. Order placement itself does not cross the bus: the customer reaches the API over HTTPS
and the API starts the saga in-process; the saga's first outbox emission is the `reserve` command.

This contract arms the consumer-READS completeness tier: every row below states what its consumer
reads off the payload, in section (c) of the consuming machine's matrix, and Gx-trace holds each
declared field to the payload cell of the row that carries it.

<!-- machinery:reads-complete -->

Source: enumerated from the saga's outbox emissions (`FulfillmentSaga` invoke actors) and the
consumers' machine events; the broker lane is the single outbox topic per service, no other
subscriptions exist to sweep.

| event | producer | consumer | payload (Modelith attributes) | delivery | ordering | dedupe |
|---|---|---|---|---|---|---|
| `reserve` command | `orderSvc` (saga, via its outbox) | `inventorySvc` (no machine: receipt CREATES the Reservation, whose machine starts in Held; creation is not one of its events) | order id; per line: `LineItem.quantity`, `Product.sku`; the hold creates `Reservation.quantity`, `Reservation.status` | at-least-once via outbox | first saga step; no cross-order ordering assumed | consumer dedupes by message id; hold idempotent per order and line item |
| `release` command (compensation) | `orderSvc` (saga, via its outbox) | `inventorySvc` | order id; the `Reservation` records to release | at-least-once via outbox; re-driven by the `compensate` step | after `reserved`; releasing a Released reservation is a no-op (`reservation-terminal`) | message id; release is idempotent |
| `capture` command | `orderSvc` (saga, via its outbox) | `paymentSvc` | `Payment.amountCents` (equals `Order.totalCents`, `capture-matches-total`), `Payment.idempotencyKey` | at-least-once via outbox | only after `reserved` (`reserve-before-pay`, proven in `formal/Checkout.tla`) | `Payment.idempotencyKey` (`payment-idempotent`); the gateway is called with the same key |
| `refund` command (compensation) | `orderSvc` (saga, via its outbox) | `paymentSvc` | `Refund.amountCents` (capped by `refund-within-capture`), `Payment.idempotencyKey` | at-least-once via outbox; re-driven by the `compensate` step | only after a successful capture | idempotency key; refund is idempotent |
| `dispatch` command | `orderSvc` (saga, via its outbox) | `shippingSvc` | order id; `Address.line1`, `Address.city`, `Address.postalCode`, `Address.country` (`address-country-present`); line quantities | at-least-once via outbox | only after `captured` (`no-ship-before-pay`, proven in `formal/Checkout.tla`) | message id; one `Shipment` per `Order` (1:1) |
| `reserved` / `released` events | `inventorySvc` (via its outbox) (no machine: the reply is an outbox row the service writes after the Reservation transition, not a machine action) | `orderSvc` (saga) (no machine: the saga consumes the reply through its `reserveInventory` invoke, not as an event; the Order aggregate then reacts as `markReserved`) | order id, `Reservation.status`, `Reservation.quantity` | at-least-once via outbox | per order; the saga reacts only to the reply for its current step, stale replies are ignored | message id |
| `captured` / `refunded` / `failed` events | `paymentSvc` (via its outbox) (no machine: the reply is an outbox row the service writes after the Payment transition, not a machine action) | `orderSvc` (saga) (no machine: the saga consumes the reply through its `capturePayment` invoke, not as an event; the Order aggregate then reacts as `markPaid` or `fail`) | order id, `Payment.status`, `Payment.amountCents` | at-least-once via outbox | per order; as above | message id |
| `dispatched` / `delivered` / `lost` events | `shippingSvc` (via its outbox) (no machine: the reply is an outbox row the service writes after the Shipment transition, not a machine action) | `orderSvc` (saga) (no machine: the saga consumes the reply through its `dispatchShipment` invoke, not as an event; the Order aggregate then reacts as `markShipped`, `markDelivered`, or `fail`) | order id, `Shipment.status`, `Shipment.trackingId` | at-least-once via outbox | per order; as above; `delivered` / `lost` drive the Order's `markDelivered` / `fail` | message id |

## 7. NFR record

- **Security posture**: the customer-facing API and the operator's stock management are HTTPS. The
  payment gateway credential lives only in the Payment Service and is never logged; the platform
  stores no card data, only `Payment.amountCents` and `Payment.idempotencyKey` (the gateway holds
  the instrument). Each service connects only to its own database and to the bus; the Architecture
  Contract forbids direct service-to-service calls. Out of scope: end-user authentication and
  storefront hardening (catalog UX is out of scope for this design).
- **Capacity assumptions**: one `gen_statem` per in-flight order; thousands of orders per day, not
  millions. Each service scales against its own PostgreSQL; the bus carries a bounded handful of
  messages per order. Step timeouts (5-8s) and retry bounds (<= 3) cap in-flight work; correctness
  of compensation is chosen over throughput.
- **Observability and alerting**: every service logs state transitions keyed by order id and
  message id. A `FulfillmentSaga` that enters `FailedDirty` MUST page an operator: money or stock
  may still be held and no automatic recovery exists (this is the paging requirement the
  FulfillmentSaga matrix cites). Monitor outbox lag (age of the oldest Pending `OutboxMessage`)
  and bus redelivery counts; sustained growth in either means delivery or an idempotent consumer
  is broken. Metrics and log tooling choices are deferred to implementation; the signals above are
  not.

## 8. Gate 2 result

Every Modelith action maps to an owning service; every external dependency has a mitigation-posture row;
the contract is consistent and the services are forbidden from calling each other directly; persistence
and placement are decided per component; every bus-crossing message has an event-contract row. PASS.
