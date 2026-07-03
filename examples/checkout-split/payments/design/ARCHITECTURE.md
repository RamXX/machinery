# Architecture: Payments Subsystem

A child design of examples/checkout-split/parent. The pack under design/pack/
is the frozen interface; the Payment machine must refine PaymentsContract
(see packmap.yaml and formal/).

## 4. Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: payments.svc
    kind: container
    element: payments
    code: [ "cmd/**", "internal/**" ]
externals:
  - id: external.bus
    element: bus
    imports: [ "example.com/busdriver" ]
  - id: external.paydb
    element: paydb
    imports: [ "example.com/pgdriver" ]
dependency_rules:
  allow:
    - payments.svc -> external.bus
    - payments.svc -> external.paydb
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
| `bus` | down, redelivery, reorder | outbox + idempotent consumers (dedupe by `Payment.orderId`) | duplicate `request` creates no second payment | ack window |
| `paydb` | unavailable, corrupt | retry with backoff, PITR restore | transient unavailability surfaces after retries | retry <= 3 |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Payment` | payments service | db row | single writer per payment id |

## 8. NFR record

- Security: broker credentials; processor API key in the secret store. Out of
  scope beyond that, recorded as such.
- Capacity: toy example; out of scope, recorded as such.
- Observability: log every decline and every dedupe drop with the order id.
