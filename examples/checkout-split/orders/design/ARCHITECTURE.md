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
dependency_rules:
  allow:
    - orders.svc -> external.bus
    - orders.svc -> external.ordersdb
  deny: []
```

## 5. Event contracts (from the pack; do not widen)

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

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Order` | orders service | db row | single writer per order id |

## 8. NFR record

- Security: broker credentials only; no inbound calls except the customer API.
  End-user auth out of scope, recorded as such.
- Capacity: toy example; out of scope, recorded as such.
- Observability: log every markDeclined and every dedupe drop with the order id.
