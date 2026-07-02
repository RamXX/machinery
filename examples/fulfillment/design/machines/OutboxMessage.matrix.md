# OutboxMessage machine - named-unit contracts and failure catalog

Component: the Outbox in every service (the canonical instance lives in `order.service`).
Machine: `OutboxMessage.machine.json`. Placement (ARCHITECTURE.md 4): a table in each service DB,
drained by a poller; the machine models ONE publish attempt, and the poller loop supplies the
unbounded at-least-once re-drive. Transitions are the generated oracle (`OutboxMessage.oracle.md`);
this document is the named-unit contract table and the failure catalog.

## (a) Named-unit contract table

| name | kind | signature | pre / post | maps to | test | fixture |
|---|---|---|---|---|---|---|
| `publishToBus` | actor | `(messageId, messageType) -> ok \| err` | publishes the payload to the bus; the same messageId may be published more than once across attempts (consumers dedupe), never zero times for a row that stays Pending | C4 `outbox -> bus`; inv `outbox-at-least-once` | integration | real RabbitMQ (docker compose) |
| `persistOutboxRow` | actor | `(messageId, status) -> row \| err{ErrUnavailable,ErrConflict}` | marks the row Published or Consumed in its own transaction | C4 `outbox -> orderDb` | integration | real Postgres |
| `pendingIsPublished` / `pendingIsConsumed` | guard | `(ctx) -> bool` | true iff `ctx.pendingStatus` equals that status | - (persist success routing) | unit | pure |
| `priorIsPending` / `priorIsPublished` | guard | `(ctx) -> bool` | true iff `ctx.priorStatus` equals that status; a rollback to Pending is the at-least-once re-drive point | inv `outbox-at-least-once` | unit | pure |
| `isErrUnavailable` / `isErrConflict` | guard | `(ctx,evt) -> bool` | true iff `evt.error` is that typed repo error | C4 3 DB failure classes | unit | pure |
| `retriesExhausted` | guard | `(ctx) -> bool` | true iff `ctx.retries >= 3` | C4 3 bound | unit | pure |
| `loadPayload` | action | `(ctx,evt) -> ctx` | load the event payload for the publish attempt | - | unit | pure |
| `setPendingPublished` / `setPendingConsumed` | action | `(ctx,evt) -> ctx` | `priorStatus := status; pendingStatus := <that status>` | - | unit | pure |
| `commitStatus` | action | `(ctx) -> ctx` | `status := pendingStatus` | - | unit | pure |
| `incrementRetries` | action | `(ctx) -> ctx` | `retries := retries + 1` | - | unit | pure |
| `recordPublishError` / `recordPublishTimeout` | action | `(ctx,evt) -> ctx` | `lastError := classified bus outcome`; the row stays Pending for the next sweep | C4 3 bus posture | unit | pure |
| `recordError` / `recordConflict` / `recordTimeout` / `recordUnknownError` / `recordRetriesExhausted` / `recordRoutingError` | action | `(ctx,evt) -> ctx` | `lastError := classified repo error` | maps repo errors | unit | pure |

## (b) Failure catalog

| failure | detection | transition | recovery | bounding mitigation / residual risk |
|---|---|---|---|---|
| Bus unavailable or publish timeout | `publishToBus` onError, or `after busTimeout` (5s) | `publishing -> rolledBack -> Pending` | the row stays Pending; the next poller sweep re-drives publish (at-least-once) | C4 3: outbox. Residual: delivery delayed until the bus recovers; monitor sweep-queue depth |
| Duplicate publish | poller races its own sweep, or redelivery | `_ignores` on Published | consumers dedupe by message id (exactly-once effect) | inv `exactly-once-effect` |
| Service DB unavailable / conflict on mark | `persistOutboxRow` onError | `persisting -> persistRetry -> persisting` then `rolledBack` at 3 | bounded retry per attempt; the sweep retries the marking too | C4 3. Residual: a Published row may be re-published (safe, deduped) |
| Write timeout | `after persistTimeout` (5s) | `persisting -> rolledBack -> priorStatus` | abort the attempt | C4 3. Residual: none |
