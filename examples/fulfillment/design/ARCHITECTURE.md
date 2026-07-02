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
contract_version: 1
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

## 4. Persistence and placement

| component | placement | persistence | concurrency serialization |
|---|---|---|---|
| `FulfillmentSaga` | a persistent process per order (`gen_statem` under a `Registry`) | saga state row plus the outbox in the Order DB | one process per order (actor mailbox) |
| `Order` `Payment` `Reservation` `Shipment` | per-service DB rows | status columns per aggregate | optimistic lock per row |
| `OutboxMessage` | a table in each service DB, drained by a poller | rows marked Published then Consumed; the machine models one publish attempt, the poller supplies the at-least-once re-drive | at-least-once; consumers dedupe by message id |

## 5. Gate 2 result

Every Modelith action maps to an owning service; every external dependency has a mitigation-posture row;
the contract is consistent and the services are forbidden from calling each other directly; persistence
and placement are decided per component. PASS.
