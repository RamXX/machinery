# BUILD: Order Fulfillment

MODE: manifest. This document is the entry point over the `design/` tree, not a self-contained
blueprint: a coding agent builds from this document PLUS the artifacts it references (the domain
model, the architecture contract, the machines, and the generated oracles), which are the source of
truth. The zero-context claim applies to the design tree as a whole.

## 1. Purpose and scope

Take a customer order to delivery across four services (order, inventory, payment, shipping),
coordinated by a saga that reserves stock, captures payment, and dispatches a shipment, compensating
on failure. In scope: the fulfillment happy path, compensation, reliable messaging. Out of scope:
catalog UX, returns after delivery, tax and promotions.

## 2. Domain model (the what)

Source of truth: `fulfillment.modelith.yaml` (lints clean; 12 entities, 6 enums, 28 invariants, 8
scenarios). Render it with `modelith render`. The data dictionary lives there and is not restated here.

## 3. Architecture (the how)

Source of truth: `workspace.dsl` and `ARCHITECTURE.md`. Four Elixir services, each with its own
PostgreSQL, communicating only over a message bus (no direct service-to-service calls, enforced by the
Architecture Contract). Reliable delivery is a transactional outbox plus idempotent consumers. The
Order Service owns the saga as a `gen_statem` per order. The distributed mitigation posture (message
bus, gateway, per-service DB, carrier, partition) is in ARCHITECTURE.md section 3.

## 4. Behavior and formal verification

The FulfillmentSaga is `machines/FulfillmentSaga.machine.json`; its oracle is generated
(`machines/FulfillmentSaga.oracle.md`). The behavior is not just documented, it is proven by TLC
(`make verify-formal`):

- `formal/FulfillmentSaga.tla` (control flow): the saga always terminates (Completed, Failed, or the
  explicit residual FailedDirty), retries are bounded, and there is no deadlock.
- `formal/FulfillmentSagaData.tla` (data, generated from `FulfillmentSaga.semantics.yaml`): money and
  stock are never silently lost. A compensated saga refunds what it captured and releases what it
  reserved, or ends FailedDirty as an explicit residual for a human to resolve.
- `formal/Checkout.tla` (composition, generated from `checkout.composition.yaml`): the cross-aggregate
  invariants `reserve-before-pay` and `no-ship-before-pay` hold over the composed contracts.

Every lifecycle entity has its machine, each with a generated oracle and a named-unit matrix, and
each with a generated TLC control-flow proof (termination of every overlay excursion, bounded
retries, no deadlock):

| machine | realization | proof |
|---|---|---|
| `machines/FulfillmentSaga.machine.json` | `gen_statem` per order (Order Service) | `formal/FulfillmentSaga.tla` + `formal/FulfillmentSagaData.tla` |
| `machines/Order.machine.json` | Ecto schema + transition function, Order DB row | `formal/Order.tla` |
| `machines/Reservation.machine.json` | Ecto schema, Inventory DB row | `formal/Reservation.tla` |
| `machines/Payment.machine.json` | Ecto schema, Payment DB row; gateway calls carry an idempotency key | `formal/Payment.tla` |
| `machines/Shipment.machine.json` | Ecto schema, Shipping DB row; carrier calls bounded-retried | `formal/Shipment.tla` |
| `machines/OutboxMessage.machine.json` | outbox table row; one machine activation per poller attempt | `formal/OutboxMessage.tla` |

Each machine's `.matrix.md` carries the named-unit contracts (every guard, action, and actor with
its test type and fixture) and the failure catalog mapping ARCHITECTURE.md section 3 to transitions.

## 5. Traceability matrix

Every invariant with where it is enforced. Formal entries are TLC-checked.

| invariant | enforced by |
|---|---|
| `customer-email-unique` | DB unique constraint (order/customer service) |
| `product-price-nonneg` | validation in the catalog write |
| `reserved-within-stock` | guard in the Inventory reserve action; DB check |
| `available-nonneg` | guard in the Inventory reserve/release actions |
| `order-owned-by-customer` | structural (order created with a customer) |
| `order-total-matches-items` | computed on place; property test |
| `order-forward` | `Order.machine.json` (each state exposes only its forward saga event; `setPending*` named units) |
| `order-delivered-terminal` | structural (terminal states have no outgoing transition) |
| `line-item-quantity-positive` | validation on add |
| `reservation-quantity-positive` | validation on hold |
| `reservation-terminal` | structural (Committed/Released are final) |
| `payment-amount-nonneg` | validation on authorize |
| `payment-idempotent` | idempotency key unique per capture |
| `payment-terminal` | structural (Failed/Refunded are final) |
| `refund-amount-positive` | validation on issue |
| `shipment-terminal` | structural (Delivered/Lost are final) |
| `saga-terminal` | FulfillmentSaga control flow (final states) |
| `address-country-present` | validation on capture |
| `outbox-at-least-once` | outbox poller republishes until Consumed |
| `no-ship-before-pay` | **formal**: `Checkout.tla` composition |
| `capture-matches-total` | payment service asserts amount equals order total |
| `refund-within-capture` | payment service caps refund at captured amount |
| `reserve-before-pay` | **formal**: `Checkout.tla` composition |
| `saga-compensation` | **formal**: `FulfillmentSagaData.tla` (no silent loss) |
| `exactly-once-effect` | idempotent consumers keyed by message id |

## 6. Build plan

Elixir umbrella with four apps plus a shared contracts lib. Walking skeleton: `place order -> saga
reserves (stub inventory) -> saga captures (stub gateway) -> saga dispatches (stub carrier) ->
Delivered`, exercising one real message-bus round trip and the outbox. Then vertical slices per service,
then the real gateway and carrier adapters. The saga is built against the verified machine: its
`gen_statem` states and transitions mirror `FulfillmentSaga.machine.json` one to one.

## 7. Hard-TDD protocol

A test-writer derives the suite from the six generated oracles (`machines/*.oracle.md`, one row per
transition, keyed by STABLE id, never by row number), the named-unit contract tables (each row states
its test type and fixture), the traceability matrix, and the formal properties (the saga terminates,
money is never silently lost, no-ship-before-pay). The tests are then locked; the implementer makes
them pass without editing them. A wrong test is a design defect that returns to the design and the
formal model. Generated transition tests live apart from hand-written tests so a design revision can
regenerate them without clobbering anything.

## 8. State migration

Machine states are persisted (saga rows, aggregate status columns, outbox rows; see the
ARCHITECTURE.md placement table). No persisted instances exist yet (design only). From the first
deployment onward, any revision that renames, splits, or removes a state of a persisted machine MUST
ship a mapping table from old persisted values to new states (or an explicit drain rule for in-flight
instances) in this section before the revision is implemented.

## 9. Residual risks

`FailedDirty` is the explicit residual when compensation cannot complete within the retry bound; it
requires manual reconciliation and must page an operator. Exactly-once effect depends on every consumer
being idempotent; a non-idempotent consumer is a defect the outbox cannot mask.
