# Payment machine - named-unit contracts and failure catalog

Component: `payment.service` Payment aggregate. Machine: `Payment.machine.json`.
Placement (ARCHITECTURE.md 4): a status row in the Payment DB, optimistic lock; gateway calls carry an
idempotency key. Transitions are the generated oracle (`Payment.oracle.md`); this document is the
named-unit contract table and the failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `gatewayAuthorize` / `gatewayCapture` / `gatewayRefund` | actor | `(amountCents, idempotencyKey) -> ok \| err{retryable, rejected}` | charges/refunds ONCE per idempotency key regardless of retries; capture amount equals the order total; refund never exceeds the capture | C4 `paymentSvc -> stripe`; inv `payment-idempotent`, `capture-matches-total`, `refund-within-capture` | integration + property | gateway fake contract-tested against the stripe sandbox; property: N retries with one key = one charge |
| `persistPayment` | actor | `(paymentId, status) -> row \| err{ErrUnavailable,ErrConflict}` | writes the status row and the outbox event in one transaction, or neither | C4 `paymentSvc -> paymentDb`; inv `exactly-once-effect` | integration | real Postgres |
| `guardAmountNonneg` | guard | `(ctx,evt) -> bool` | true iff `amountCents >= 0` | inv `payment-amount-nonneg` | unit | pure |
| `gatewayForAuthorize` / `gatewayForCapture` / `gatewayForRefund` | guard | `(ctx) -> bool` | true iff `ctx.gatewayOp` is that operation (routes the retry back to the in-flight call) | - | unit | pure |
| `gatewayRetriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.gatewayRetries >= 3` | C4 3 gateway bound | unit | pure |
| `pendingIsAuthorized` / `pendingIsCaptured` / `pendingIsRefunded` / `pendingIsFailed` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) | unit | pure |
| `priorIsPending` / `priorIsAuthorized` / `priorIsCaptured` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status | - (rollback routing) | unit | pure |
| `isErrRetryable` | guard | `(ctx,evt) -> bool` | true iff the gateway error is a 5xx or a timeout (never a rejection) | C4 3 gateway failure classes | unit | pure |
| `isErrUnavailable` / `isErrConflict` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 3 DB failure classes | unit | pure |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 bound | unit | pure |
| `setGatewayAuthorize` / `setGatewayCapture` / `setGatewayRefund` | action | `(ctx,evt) -> ctx` | `priorStatus := status; gatewayOp := <op>; gatewayRetries := 0` | - | unit | pure |
| `setPendingAuthorized` / `setPendingCaptured` / `setPendingRefunded` / `setPendingFailed` | action | `(ctx,evt) -> ctx` | `pendingStatus := <that status>` | inv `payment-terminal` (only legal successors) | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` | - | unit | pure |
| `incrementRetries` / `incrementGatewayRetries` | action | `(ctx) -> ctx` | increment the respective counter | - | unit | pure |
| `recordAuthorizeDenied` | action | `(ctx,evt) -> ctx` | rejection reason naming `payment-amount-nonneg` | surfaces the invariant | unit | pure |
| `recordGatewayError` / `recordGatewayTimeout` / `recordGatewayRejected` / `recordGatewayExhausted` / `recordRefundFailed` | action | `(ctx,evt) -> ctx` | `lastError := classified gateway outcome`; `recordRefundFailed` surfaces to the saga, which owns the FailedDirty residual | C4 3 gateway posture | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified repo error` | maps repo errors | unit | pure |

Structural: `payment-terminal` is enforced by Failed and Refunded being final states.

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Gateway 5xx or timeout | `gateway*` onError `isErrRetryable`, or `after gatewayTimeout` (8s) | `authorizing/capturing/refunding -> gatewayRetry -> gatewayResume -> <same call>` | bounded retry with the SAME idempotency key | C4 3: idempotency key, retry <= 3. Residual: none (one charge per key) |
| Gateway rejection (card declined) | `gateway*` onError, not retryable | `-> persisting` with `pendingStatus := Failed` (authorize/capture) | terminal Failed; the saga compensates | Residual: none |
| Refund exhausted | `gatewayRetriesExhausted` in refunding path | `gatewayRetry -> rolledBack -> Captured`, `recordRefundFailed` | the payment stays truthfully Captured; the saga retries compensation and owns FailedDirty | inv `refund-within-capture` holds; residual owned by the saga |
| Payment DB unavailable / conflict | `persistPayment` onError | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry; the gateway outcome is re-persisted on the bus redelivery | C4 3. Residual: gateway state ahead of the row until redelivery lands |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> priorStatus` | abort; outbox guarantees no half-published event | C4 3. Residual: none |
