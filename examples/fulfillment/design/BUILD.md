# BUILD: Order Fulfillment

Mode: full (self-contained).

Single deliverable; there is no `design/BUILD/` shard directory. A coding agent builds from this
document, and the `design/` files it references (the domain model, the architecture contract, the
machines, and the generated oracles) are the source of truth for full detail. When this document
and a source file disagree, the source file wins and this document is a defect: stop and fix it.

## 1. Purpose and scope

Take a customer order to delivery across four services (order, inventory, payment, shipping),
coordinated by a saga that reserves stock, captures payment, and dispatches a shipment, compensating
on failure. In scope: the fulfillment happy path, compensation, reliable messaging. Out of scope:
catalog UX, returns after delivery, tax and promotions.

## 2. Domain model (the what)

Source of truth: `fulfillment.modelith.yaml` (lints clean; 12 entities, 6 enums, 25 invariants, 8
scenarios). Render it with `modelith render`. The data dictionary lives there and is not restated here.

## 3. Architecture (the how)

Source of truth: `workspace.dsl` and `ARCHITECTURE.md`. Four Elixir services, each with its own
PostgreSQL, communicating only over a message bus (no direct service-to-service calls, enforced by the
Architecture Contract). Reliable delivery is a transactional outbox plus idempotent consumers. The
Order Service owns the saga as a `gen_statem` per order. The distributed mitigation posture (message
bus, gateway, per-service DB, carrier, partition) is in ARCHITECTURE.md section 3. The event-contract
table governing every bus-crossing message (producer, consumer, payload, delivery, ordering, dedupe)
is ARCHITECTURE.md section 5.

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

Elixir umbrella with four apps plus a shared contracts lib. Walking skeleton first (prove the
topology through one real boundary), then one vertical slice per service, then the real gateway and
carrier adapters. The saga is built against the verified machine: its `gen_statem` states and
transitions mirror `FulfillmentSaga.machine.json` one to one. Definition of done (DoD) is stated per
milestone; the global bar is section 7 (one transition test per oracle row keyed by stable id, one
named-unit test per matrix row, every section 5 invariant enforced where the matrix says). This is a
design-only example: the milestones bind the first implementer.

**M0 - Walking skeleton (thinnest end-to-end thread).** `place order -> saga reserves (stub
inventory) -> saga captures (stub gateway) -> saga dispatches (stub carrier) -> Delivered`,
exercising one real message-bus round trip and the transactional outbox. DoD: green for the saga
forward path `FULF-ee2ed2`, `FULF-bba0be`, `FULF-6ec4e1` (T-FULF-02,05,08); the Order happy chain
T-ORDE-01,05,09,11,13 with its persist commits T-ORDE-16..20; one outbox row driven
Pending -> Published -> Consumed (T-OUTB-01,04,07 then T-OUTB-02,08); the delivered order durably
persisted (a re-read sees Delivered); the bus round trip and the outbox write are real (docker
compose), the three step actors are stubs.

**M1 - Order service slice (aggregate, saga, outbox).** Complete the Order lifecycle (cancel, fail,
denial rows) and its persist overlay, every saga compensation path, and the outbox failure rows via
the Oban poller. DoD: all 33 T-ORDE, 14 T-FULF, and 16 T-OUTB rows green; `order-forward`,
`order-owned-by-customer`, `order-total-matches-items` (property over generated line-item lists),
`order-delivered-terminal`, `saga-terminal`, `outbox-at-least-once`, `exactly-once-effect`
(idempotent consumers keyed by message id), `customer-email-unique`, `product-price-nonneg`, and
`line-item-quantity-positive` enforced per section 5; the FailedDirty residual pages an operator;
`make verify-formal` still green (saga termination, `saga-compensation`).

**M2 - Inventory service slice.** Reservation lifecycle end to end against the real Inventory
Postgres. DoD: all 12 T-RESE rows green; `reserved-within-stock` guard tests green (DB check
included) and `available-nonneg` guard tests green; `reservation-quantity-positive` validation and
`reservation-terminal` structural tests green; `reserve-before-pay` still proven (`Checkout.tla`).

**M3 - Payment service slice.** Payment lifecycle against the gateway fake (contract-tested per the
matrix fixture). DoD: all 37 T-PAYM rows green including the gatewayRetry bound and gatewayResume
routing; `payment-amount-nonneg`, `payment-terminal`, `refund-amount-positive`,
`payment-idempotent` (idempotency key unique per capture), `capture-matches-total`, and
`refund-within-capture` enforced per section 5.

**M4 - Shipping service slice.** Shipment lifecycle against the carrier fake. DoD: all 26 T-SHIP
rows green including the carrierRetry bound; `shipment-terminal` structural and
`address-country-present` validation tests green; `no-ship-before-pay` still proven
(`Checkout.tla`).

**M5 - Real gateway and carrier adapters.** Swap the stub gateway and carrier for the real adapters
behind the same ports. DoD: the `capturePayment` and `dispatchShipment` actor contracts green
against the sandbox fixtures named in the matrices; a capture retried twice charges once
(idempotency key end to end); the full compensation path green against the real adapters; no
transition test changes (the machines are adapter-agnostic).

### Toolchain and versions

Design-only example: no `impl/` exists yet. This pins the environment for the first implementer so
two implementing agents cannot diverge.

- Target language: Elixir/OTP, an umbrella of four apps plus `libs/contracts` (per `workspace.dsl`
  and ARCHITECTURE.md section 1): Phoenix for the order API, Ecto for persistence, Oban for the
  outbox poller, `gen_statem` for the saga. Pin exact Elixir/OTP and library versions in `mix.lock`
  at project start; that lockfile then becomes the source of truth for pins.
- Infrastructure: one PostgreSQL per service and RabbitMQ as the bus; integration fixtures run the
  real bus and services via docker compose (per the `machines/*.matrix.md` fixture columns), no mocks.
- Tests: ExUnit; transition tests are derived from the generated oracles, named-unit and property
  tests from the matrix tables.
- The two design-gate commands an implementer runs, from the example root:
  `machinery oracle design/machines` (regenerate and commit the oracles after any machine change)
  and `machinery check design` (all design gates; add `--impl <dir>` once code exists). Formal
  proofs re-run with `make verify-formal` (TLC; needs Java 11+).

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
