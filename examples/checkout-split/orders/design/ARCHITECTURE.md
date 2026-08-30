# Architecture: Orders Subsystem

A child design of examples/checkout-split/parent. The pack under design/pack/
is the frozen interface: the Order entity's public shape, the three boundary
event contracts, the delegated invariant, and the contract machine this
design's Order machine must refine (see packmap.yaml and formal/).

## 4. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: orders.svc
    kind: container
    element: orders
    code: [ "cmd/**", "internal/**" ]
externals:
  - id: external.bus
    element: bus
    imports: [ "example.com/busdriver" ]
  - id: external.ordersdb
    element: ordersdb
    imports: [ "example.com/pgdriver" ]
  # the peer subsystem is a declared participant, not an undeclared name in the
  # event rows: it settles this subsystem's orders and its outage is a failure
  # mode this design has to hold a posture on
  - id: external.payments
    element: payments
dependency_rules:
  allow:
    - orders.svc -> external.bus
    - orders.svc -> external.ordersdb
  deny: []
```

## 4b. Interface contracts

| edge | shape | errors | idempotency |
|---|---|---|---|
| `orders.svc -> external.bus` | busdriver publish and subscribe on the orders topic; payloads are exactly the section 5 event rows | `ErrBusUnavailable`, `ErrPublishRejected`, mapped here so no driver type escapes the service | publishes ride the outbox, so a retried publish repeats one message id and the consumer dedupes it |
| `orders.svc -> external.ordersdb` | pgdriver SQL over the orders schema: load the order row, write it back together with its outbox rows in one transaction | `ErrDbUnavailable`, `ErrConflict`, `ErrCorrupt` | the state change and its outbox rows commit together, so a retried transaction repeats the whole unit or none of it |

## 5. Event contracts (from the pack; do not widen)

The rows are copied from the parent's table, and the copy is checked rather than promised: every row
here is byte-identical to a parent row, and every parent row that names this subsystem is here.

<!-- machinery:embed from="../../parent/design/ARCHITECTURE.md" table="event,producer,consumer,delivery" where="producer|consumer=orders" claims="subset,complete" -->

| event | producer | consumer | payload | delivery | ordering | dedupe |
|---|---|---|---|---|---|---|
| request | orders | payments | Payment.orderId, Payment.amount | at-least-once | none | Payment.orderId |
| markPaid | payments | orders | Payment.orderId | at-least-once | none | Payment.id |
| markDeclined | payments | orders | Payment.orderId | at-least-once | none | Payment.id |

## 6. Dependency mitigation posture

| dependency | failure modes | mitigation | residual | bound |
|---|---|---|---|---|
| `bus` | down, redelivery, reorder | outbox + idempotent consumers (dedupe keys above) | duplicates land as `_ignores` on every resting state | ack window |
| `ordersdb` | unavailable, corrupt | retry with backoff, PITR restore | transient unavailability surfaces after retries | retry <= 3 |
| `payments` | down, slow, replies lost | the order waits in its awaiting-settlement state; the bus re-drives `request` until a reply lands | an order can sit unsettled for as long as the peer is down; nothing times it out | ack window |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Order` | orders service | db row | single writer per order id |

## 8. NFR record

- Security: broker credentials only; no inbound calls except the customer API.
  End-user auth out of scope, recorded as such.
- Capacity: toy example; out of scope, recorded as such.
- Observability: log every markDeclined and every dedupe drop with the order id.
