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

## 2. Glossary

The ubiquitous language, from the Modelith glossary and entity names; the reader has no other
source for these words.

- **Customer** - the person who places an `Order`; identified by a unique email
  (`customer-email-unique`).
- **Product** - a catalog item with a SKU and a non-negative price (`product-price-nonneg`).
- **Inventory** - per-product stock counters in the Inventory Service (on-hand and reserved).
- **Order** - one customer purchase: line items, a total, an address, and a lifecycle
  (`OrderStatus`) driven by the saga.
- **LineItem** - one product-quantity row of an `Order`; quantity strictly positive.
- **Reservation** - a hold on stock for one order (`ReservationStatus`: Held, Committed, Released).
- **Payment** - the capture of the order total via the gateway (`PaymentStatus`), idempotent per
  `Payment.idempotencyKey`.
- **Refund** - the compensating return of captured funds, capped by the captured amount.
- **Shipment** - the carrier handoff and tracking for one order (`ShipmentStatus`), 1:1 with the
  `Order`.
- **Address** - the delivery address captured with the order; country required
  (`address-country-present`).
- **FulfillmentSaga (saga)** - the per-order orchestrator in the Order Service (`gen_statem`):
  reserve, then capture, then dispatch, compensating on failure (`SagaStatus`).
- **Compensation** - the saga's failure path: release what was reserved, refund what was captured;
  bounded retries, then the explicit `FailedDirty` residual.
- **FailedDirty** - the saga's explicit dirty terminal state: compensation could not complete
  within the retry bound; pages an operator for manual reconciliation.
- **OutboxMessage** - one row of a service's transactional outbox (`MessageStatus`: Pending,
  Published, Consumed); written in the producing transaction, republished until Consumed.
- **Idempotent consumer** - every bus consumer dedupes by message id (or idempotency key), so
  at-least-once delivery has exactly-once effect (`exactly-once-effect`).
- **Gateway / Carrier** - the external payment processor and shipping carrier, reached only from
  the Payment and Shipping services, bounded-retried per the mitigation posture.
- **Walking skeleton** - the thinnest end-to-end slice exercising one real transition through one
  real boundary, built first to prove the topology.

## 3. Domain model (the what)

Source of truth: `fulfillment.modelith.yaml` (lints clean; 12 entities, 6 enums, 25 invariants, 8
scenarios). Render it with `modelith render`. The data dictionary lives there and is not restated here.

## 4. Architecture (the how)

Source of truth: `workspace.dsl` and `ARCHITECTURE.md`. Four Elixir services, each with its own
PostgreSQL, communicating only over a message bus (no direct service-to-service calls, enforced by the
Architecture Contract). Reliable delivery is a transactional outbox plus idempotent consumers. The
Order Service owns the saga as a `gen_statem` per order. The distributed mitigation posture (message
bus, gateway, per-service DB, carrier, partition) is in ARCHITECTURE.md section 3. The event-contract
table governing every bus-crossing message (producer, consumer, payload, delivery, ordering, dedupe)
is ARCHITECTURE.md section 5.

### Migration implementation plan

N/A - greenfield design: no `design/migration.yaml` exists and there is no legacy/target
transition to plan.

### Neighbor stand-ins and test environment

N/A - not a pack child: no `design/pack/` exists. The four services are designed together in this
one run, and the integration fixtures run the real bus and services via docker compose (see the
Toolchain section), so no contract stand-in is needed.

## 5. Behavior and formal verification

The FulfillmentSaga is `machines/FulfillmentSaga.machine.json`; its oracle is generated
(`machines/FulfillmentSaga.oracle.md`). The behavior is not just documented, it is proven by TLC
(`machinery verify-formal design`):

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

## 6. Traceability matrix

Every invariant: its enforcement point, its owning component (per `workspace.dsl`), the interface
contract that carries it (the ARCHITECTURE.md section 5 event rows for bus-crossing rules, the
service's own write path otherwise), and its test ids. Formal entries are TLC-checked. Transition
tests are cited by the oracle's `T-<MACHINE>-NN` ids (the stable-id column is the key the tests
use, section 8); `P-<invariant>` names the property test for that invariant.

| invariant | enforced by (guard / structural) | in component | interface contract | test id(s) |
|---|---|---|---|---|
| `customer-email-unique` | DB unique constraint on `Customer.email` | `orderSvc` (Order Repository, Order DB unique index) | order API customer write path | P-customer-email-unique |
| `product-price-nonneg` | validation in the order service's product write path (catalog UX is out of scope; the catalog data lives in the Order DB per `workspace.dsl`) | `orderSvc` (API + Order Repository) | order API product write path | P-product-price-nonneg |
| `reserved-within-stock` | guard in the Inventory reserve action; DB check constraint | `inventorySvc` | `reserve` command row (ARCHITECTURE.md section 5) | P-reserved-within-stock (incl. the `persistReservation` concurrent-hold property, matrix fixture) |
| `available-nonneg` | guard in the Inventory reserve/release actions | `inventorySvc` | `reserve` / `release` command rows | P-available-nonneg |
| `order-owned-by-customer` | structural (an order is created with its customer; `guardCanCancel` checks the owner) | `orderSvc` (API + Order machine) | order API place | P-order-owned-by-customer |
| `order-total-matches-items` | guard `guardCanConfirm` (has line items AND total equals their sum); computed on place | `orderSvc` (Order machine) | order API place/confirm | T-ORDE-01, P-order-total-matches-items |
| `order-forward` | `Order.machine.json` (each state exposes only its forward saga event; `setPending*` named units) | `orderSvc` (Order machine) | consumed saga reply rows (`reserved`, `captured`, `dispatched`, `delivered`) | T-ORDE-01,05,09,11,13, P-order-forward |
| `order-delivered-terminal` | structural (terminal states have no outgoing transition) | `orderSvc` (Order machine) | - (internal state graph) | P-order-delivered-terminal |
| `line-item-quantity-positive` | validation on add | `orderSvc` (API) | order API add-item | P-line-item-quantity-positive |
| `reservation-quantity-positive` | validation on hold | `inventorySvc` | `reserve` command row | P-reservation-quantity-positive |
| `reservation-terminal` | structural (Committed/Released are final) | `inventorySvc` (Reservation machine) | `release` command row (no-op on Released) | P-reservation-terminal |
| `payment-amount-nonneg` | guard `guardAmountNonneg` on authorize | `paymentSvc` (Payment machine) | `capture` command row | T-PAYM-01, P-payment-amount-nonneg |
| `payment-idempotent` | idempotency key unique per capture; the gateway is called with the same key | `paymentSvc` | `capture` command row (dedupe `Payment.idempotencyKey`) | P-payment-idempotent |
| `payment-terminal` | structural (Failed/Refunded are final) | `paymentSvc` (Payment machine) | - (internal state graph) | P-payment-terminal |
| `refund-amount-positive` | validation on issue | `paymentSvc` | `refund` command row | P-refund-amount-positive |
| `shipment-terminal` | structural (Delivered/Lost are final) | `shippingSvc` (Shipment machine) | - (internal state graph) | P-shipment-terminal |
| `saga-terminal` | FulfillmentSaga control flow (final states); TLC termination proof | `orderSvc` (Saga Orchestrator) | - (internal state graph) | T-FULF-02,05,08, P-saga-terminal |
| `address-country-present` | validation when the address is captured with the order | `orderSvc` (API); consumed by `shippingSvc` | order API place; `dispatch` command row payload | P-address-country-present |
| `outbox-at-least-once` | outbox poller republishes until Consumed (`priorIsPending` re-drive) | every producing service's Outbox (reference component: `orderSvc` `outbox`) | every event row (all ride the outbox) | T-OUTB-01,04,07, P-outbox-at-least-once |
| `no-ship-before-pay` | **formal**: `Checkout.tla` composition; runtime: the saga emits `dispatch` only after `captured` | `orderSvc` (Saga Orchestrator) | `dispatch` command row ordering | TLC `Checkout.tla`, T-FULF-08 |
| `capture-matches-total` | payment service asserts amount equals order total | `paymentSvc` | `capture` command row payload | P-capture-matches-total |
| `refund-within-capture` | payment service caps refund at captured amount | `paymentSvc` | `refund` command row payload | P-refund-within-capture |
| `reserve-before-pay` | **formal**: `Checkout.tla` composition; runtime: the saga emits `capture` only after `reserved` | `orderSvc` (Saga Orchestrator) | `capture` command row ordering | TLC `Checkout.tla`, T-FULF-05 |
| `saga-compensation` | **formal**: `FulfillmentSagaData.tla` (no silent loss; FailedDirty is the explicit residual) | `orderSvc` (Saga Orchestrator) | `release` / `refund` compensation rows | TLC `FulfillmentSagaData.tla`, T-FULF-14 |
| `exactly-once-effect` | idempotent consumers keyed by message id | every consuming service (`orderSvc`, `inventorySvc`, `paymentSvc`, `shippingSvc`) | the dedupe column of every event row | P-exactly-once-effect |

## 7. Build plan

Elixir umbrella with four apps plus a shared contracts lib. Walking skeleton first (prove the
topology through one real boundary), then one vertical slice per service, then the real gateway and
carrier adapters. The saga is built against the verified machine: its `gen_statem` states and
transitions mirror `FulfillmentSaga.machine.json` one to one. Definition of done (DoD) is stated per
milestone; the global bar is section 8 (one transition test per oracle row keyed by stable id, one
named-unit test per matrix row, every section 6 invariant enforced where the matrix says). This is a
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
`line-item-quantity-positive` enforced per section 6; the FailedDirty residual pages an operator;
`machinery verify-formal design` still green (saga termination, `saga-compensation`).

**M2 - Inventory service slice.** Reservation lifecycle end to end against the real Inventory
Postgres. DoD: all 12 T-RESE rows green; `reserved-within-stock` guard tests green (DB check
included) and `available-nonneg` guard tests green; `reservation-quantity-positive` validation and
`reservation-terminal` structural tests green; `reserve-before-pay` still proven (`Checkout.tla`).

**M3 - Payment service slice.** Payment lifecycle against the gateway fake (contract-tested per the
matrix fixture). DoD: all 37 T-PAYM rows green including the gatewayRetry bound and gatewayResume
routing; `payment-amount-nonneg`, `payment-terminal`, `refund-amount-positive`,
`payment-idempotent` (idempotency key unique per capture), `capture-matches-total`, and
`refund-within-capture` enforced per section 6.

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

- Target language: Elixir 1.17 on OTP 27 (`elixir 1.17.x-otp-27` in `.tool-versions`), an umbrella
  of four apps plus `libs/contracts` (per `workspace.dsl` and ARCHITECTURE.md section 1): Phoenix
  for the order API, Ecto for persistence, Oban for the outbox poller, `gen_statem` for the saga.
  Pin exact library versions in `mix.lock` at project start; that lockfile then becomes the source
  of truth for library pins.
- Infrastructure: one PostgreSQL per service and RabbitMQ as the bus; integration fixtures run the
  real bus and services via docker compose (per the `machines/*.matrix.md` fixture columns), no mocks.
- Tests: ExUnit; transition tests are derived from the generated oracles, named-unit and property
  tests from the matrix tables.
- The two design-gate commands an implementer runs, from the example root:
  `machinery oracle design/machines` (regenerate and commit the oracles after any machine change)
  and `machinery check design` (all design gates; add `--impl <dir>` once code exists). Formal
  proofs re-run with `machinery verify-formal design` (TLC; needs Java 11+).

## 8. Hard-TDD protocol

RED precondition: `machinery check design` reports ZERO blocking findings before any test is
derived. The oracles are the test spec; a red design means the spec itself cannot be trusted, so
fix the design first, never the tests.

A test-writer derives the suite from the six generated oracles (`machines/*.oracle.md`, one row per
transition, keyed by STABLE id, never by row number), the named-unit contract tables (each row states
its test type and fixture), the traceability matrix, and the formal properties (the saga terminates,
money is never silently lost, no-ship-before-pay). A runtime that cannot spawn a fresh-context
test-writer runs RED then GREEN sequentially with the same single agent; the derivation rule is
unchanged (tests come from the oracles, the matrices, and section 6, never from implementation
intentions), and the gate runs before and after are what separate the phases.

RED exits only when all three deterministic checks hold: every oracle stable id appears whole-token
in the suite (Gt-tests holds this once `machinery check design --impl <dir>` points at the code
tree), that same check is green over the compile skeleton and stubs the tests stand on (G4-import
skips test files but checks everything they import), and the suite runs red on failing assertions,
never on its own compile errors. The tests are then locked; the implementer makes them pass without
editing them. GREEN is accepted only when the locked suite passes AND
`machinery check design --impl <dir>` is green again: code that passes the tests by crossing a
boundary fails the gate; code that respects the boundaries but fails a test is not done. A wrong
test is a design defect that returns to the design and the formal model. Generated transition tests
live apart from hand-written tests so a design revision can regenerate them without clobbering
anything.

### Guard-branch completeness

One test per falsifying clause of each conjunction guard; a guard-false row is not covered until
every clause has its own case expecting the rejection path.

- `guardCanConfirm` (Order) = (the order has line items) AND (`totalCents` equals their sum). Two
  falsifying tests: an empty order with a matching zero total, and a non-empty order whose total
  mismatches. Covers `order-total-matches-items`.
- `guardCanReassign`-class role guards do not exist here; the remaining guards are single-clause
  (`guardAmountNonneg`, `guardCanCancel`, the `pendingIs*` / `priorIs*` routers) or classifier
  disjunctions: `isErrRetryable` (gateway/carrier 5xx OR timeout, never a rejection) needs one test
  per accepted class plus one rejection that must NOT retry; each `retriesExhausted` bound needs
  one test below the bound (retries) and one at the bound (routes to the failure or FailedDirty
  path).

## 9. State migration

Machine states are persisted (saga rows, aggregate status columns, outbox rows; see the
ARCHITECTURE.md placement table). No persisted instances exist yet (design only). From the first
deployment onward, any revision that renames, splits, or removes a state of a persisted machine MUST
ship a mapping table from old persisted values to new states (or an explicit drain rule for in-flight
instances) in this section before the revision is implemented.

## 10. Residual risks

`FailedDirty` is the explicit residual when compensation cannot complete within the retry bound; it
requires manual reconciliation and must page an operator. Exactly-once effect depends on every consumer
being idempotent; a non-idempotent consumer is a defect the outbox cannot mask.
