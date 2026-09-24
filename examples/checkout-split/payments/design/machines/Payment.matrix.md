# Payment machine: named-unit contracts and failure catalog

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `markPaid` | action | `(ctx) -> publish` | on capture, enqueue markPaid in the outbox, same transaction as the status write | bus relationship; dedupe `Payment.id` | integration | real outbox table + fake broker (contract-tested) |
| `markDeclined` | action | `(ctx) -> publish` | on decline, enqueue markDeclined in the outbox, same transaction | bus relationship; dedupe `Payment.id` | integration | real outbox table + fake broker (contract-tested) |
| `recordRefund` | action | `(ctx) -> ctx` | stamps the refund; only reachable from Captured, which is `payment-single-capture` | inv `payment-single-capture` (structural) | unit | none |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation |
|---|---|---|---|---|
| duplicate `request` redelivery | dedupe by `Payment.orderId` | none (creation dedupe; `_ignores` on Requested) | drop and log | idempotent consumer |
| duplicate `capture` redelivery | dedupe by `Payment.id` | none (`_ignores` on Captured) | drop and log | idempotent consumer |
| `paydb` unavailable | write error | none (command rejected, caller retries) | retry with backoff | retry <= 3 |
