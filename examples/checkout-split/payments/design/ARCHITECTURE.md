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
  # the peer subsystem is a declared participant, not an undeclared name in the
  # event rows: it originates every settlement request and its outage is a
  # failure mode this design has to hold a posture on
  - id: external.orders
    element: orders
dependency_rules:
  allow:
    - payments.svc -> external.bus
    - payments.svc -> external.paydb
  deny: []
```

## 4b. Interface contracts

| edge | shape | errors | idempotency |
|---|---|---|---|
| `payments.svc -> external.bus` | busdriver publish and subscribe on the payments topic; payloads are exactly the section 5 event rows | `ErrBusUnavailable`, `ErrPublishRejected`, mapped here so no driver type escapes the service | publishes ride the outbox, so a retried publish repeats one message id and the consumer dedupes it |
| `payments.svc -> external.paydb` | pgdriver SQL over the payments schema: load the payment row, write it back together with its outbox rows in one transaction | `ErrDbUnavailable`, `ErrConflict`, `ErrCorrupt` | the state change and its outbox rows commit together, so a retried transaction repeats the whole unit or none of it |

## 5. Event contracts (from the pack; do not widen)

The rows are copied from the parent's table, and the copy is checked rather than promised: every row
here is byte-identical to a parent row, and every parent row that names this subsystem is here.

<!-- machinery:embed from="../../parent/design/ARCHITECTURE.md" table="event,producer,consumer,delivery" where="producer|consumer=payments" claims="subset,complete" -->

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
| `orders` | down, slow, requests replayed | requests ride the peer's outbox, so a replay repeats one `Payment.orderId` and is deduped into the existing payment | a settled payment whose reply was lost is re-settled idempotently, never twice | ack window |

## 7. Persistence and placement

| component | placement | persistence | concurrency |
|---|---|---|---|
| `Payment` | payments service | db row | single writer per payment id |

## 8. NFR record

- Security: broker credentials; processor API key in the secret store. Out of
  scope beyond that, recorded as such.
- Capacity: toy example; out of scope, recorded as such.
- Observability: log every decline and every dedupe drop with the order id.
