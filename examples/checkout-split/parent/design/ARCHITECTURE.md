# Architecture: Checkout Split

Mode note: this is the PARENT of a recursive decomposition. It fixes the domain
model, the container topology, the boundary contracts, and the per-subsystem
contract machines; the executable state machines live in the child designs
(../../orders/design, ../../payments/design), each held to its pack.

## 1. Shape

Two services, one broker, one database each. All coupling crosses the bus under
the event contracts in section 5; there are no shared tables.

## 4. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: orders.svc
    kind: container
    element: orders
    code: [ "orders/**" ]
  - id: payments.svc
    kind: container
    element: payments
    code: [ "payments/**" ]
externals:
  - id: external.bus
    element: bus
    imports: [ "example.com/busdriver" ]
  - id: external.ordersdb
    element: ordersdb
    imports: [ "example.com/pgdriver" ]
  - id: external.paydb
    element: paydb
    imports: [ "example.com/pgdriver" ]
dependency_rules:
  allow:
    - orders.svc -> external.bus
    - orders.svc -> external.ordersdb
    - payments.svc -> external.bus
    - payments.svc -> external.paydb
  deny:
    - "orders.svc -> payments.svc"
    - "payments.svc -> orders.svc"
    - "orders.svc -> external.paydb"
    - "payments.svc -> external.ordersdb"
```

## 5. Event contracts (the governing artifact for the bus)

| event | producer | consumer | payload | delivery | ordering | dedupe |
|---|---|---|---|---|---|---|
| request | orders | payments | Payment.orderId, Payment.amount | at-least-once | none | Payment.orderId |
| markPaid | payments | orders | Payment.orderId | at-least-once | none | Payment.id |
| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |

## 6. Dependency mitigation posture

| dependency | failure modes | mitigation | residual | bound |
|---|---|---|---|---|
| `bus` | down, redelivery, reorder | outbox + idempotent consumers (dedupe keys above) | duplicate delivery is reclassified to an `_ignores` on every resting state | ack window |
| `ordersdb` | unavailable, corrupt | retry with backoff, PITR restore | transient unavailability surfaces after retries | retry <= 3 |
| `paydb` | unavailable, corrupt | retry with backoff, PITR restore | transient unavailability surfaces after retries | retry <= 3 |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Order` | orders service (no machine: authored in the orders child design) | db row | single writer per order id |
| `Payment` | payments service (no machine: authored in the payments child design) | db row | single writer per payment id |

## 8. NFR record

- Security: service-to-service auth via broker credentials; no direct
  service-to-service calls (denied above). Out of scope: end-user auth
  (recorded as such).
- Capacity: toy example; no volume targets. Out of scope, recorded as such.
- Observability: every markDeclined and every dedupe drop is logged with the
  order id; no residual FailedDirty-style state exists in this design.
