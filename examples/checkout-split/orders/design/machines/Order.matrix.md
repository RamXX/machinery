# Order machine: named-unit contracts and failure catalog

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `request` | action | `(ctx) -> publish` | on entry to Placed, enqueue the payment request in the outbox, same transaction as the insert | bus relationship; dedupe `Payment.orderId` | integration | real outbox table + fake broker (contract-tested) |
| `recordShipment` | action | `(ctx) -> ctx` | stamps carrier handoff; only reachable from Paid, which is `no-ship-without-capture` | inv `no-ship-without-capture` (structural) | unit | none |
| `recordCancel` | action | `(ctx) -> ctx` | stamps cancellation reason | - | unit | none |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation |
|---|---|---|---|---|
| bus down while publishing `request` | outbox dispatcher error | none (outbox retries outside the machine) | dispatcher backoff | outbox + retry, ARCHITECTURE.md section 6 |
| duplicate `markPaid` redelivery | dedupe by `Payment.id` | none (`_ignores` on Paid) | drop and log | idempotent consumer |
| `ordersdb` unavailable | row-lock/write error | none (command rejected, caller retries) | retry with backoff | retry <= 3 |
