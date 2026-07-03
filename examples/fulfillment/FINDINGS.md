# Stress test: what held and what strained

A second, deliberately different design run through the toolchain to find its limits: a
distributed order-fulfillment platform (microservices, a saga orchestrator, compensation,
transactional outbox), in contrast to go-crm's single-binary CLI over an embedded store.
Target language: Elixir/OTP.

## What held up

- **Modelith scaled.** The larger domain (12 entities, 6 enums, 25 invariants, 8 scenarios,
  distributed failure semantics) lints clean, same as the small one. Gate 1 is not size-sensitive.
- **Oracle generation generalized.** the oracle generator (now `machinery oracle`) produced the saga's transition oracle (17 rows)
  with no special casing.
- **Control-flow model checking generalized to a genuinely new shape.** The saga is not a lifecycle:
  it is a forward path (reserve, pay, ship) with a compensation reversal (refund, release) and a
  bounded compensation retry. `tla_gen` + TLC checked it unchanged and proved, over 15 states, that
  the saga always terminates (reaches Completed or Failed, never stuck mid-orchestration), the retry
  counter is bounded, and there is no deadlock. The same tool that proved the Deal lifecycle proved
  the saga. That is the load-bearing result: the generated formal layer is not tied to one pattern.

## Where it strained (the roadmap)

1. **`refine_gen` is linear-lifecycle only.** It cannot generate the data model for the saga
   (an orchestration, not an ordered lifecycle) or for a branching aggregate like Payment
   (Authorized, then Captured or Failed or Refunded). The control-flow proof still works; only the
   data-refined domain invariants are blocked. Next: a `saga` pattern (forward plus compensation,
   proving that a failure after capture issues a refund and releases reservations) and a
   `branching-lifecycle` pattern, or the general effect-weaver that reads action effects per
   transition rather than assuming a shape.

2. **The strongest invariants are cross-aggregate.** `no-ship-before-pay`, `capture-matches-total`,
   `reserve-before-pay`, and `saga-compensation` span Order, Payment, Reservation, and Shipment, so
   no single aggregate's data model can prove them. They are exactly what the assume-guarantee
   composition is for, but `System.tla` is currently hand-written for one composition. Next: a
   composition generator that instances each subsystem's contract and checks the cross-aggregate
   invariants against the composed contracts, so the recursion runs itself across N subsystems.

3. **The distributed dependency posture is richer.** Message bus, exactly-once via outbox plus
   idempotent consumers, network partitions, and saga compensation replace go-crm's single embedded
   store. The C4 mitigation-posture modeling and `machinery check` G2/G4 gates should be exercised on
   the multi-service contract; that phase of this design is not yet authored.

4. **Close the oracle loop in the gate.** `machinery check` still reconciles the machine against a
   hand-authored matrix. Now that the generator produces the oracle from the machine, G3 should diff
   the committed oracle against the freshly generated one, making the check a pure generate-and-compare.

## Status

Phase 1 (domain model) complete and clean. The saga machine is authored and its control-flow model is
TLC-verified. The remaining phases (C4, the other machines, BUILD.md) are the continuation.

## Update: strains 1 and 2 resolved, and a real bug caught

- **Strain 1 is closed.** `refine_gen` gained a `saga` pattern. `FulfillmentSagaData` is generated
  from a six-line annotation and TLC proves that money and stock are never silently lost.
- **Strain 2 is closed.** `compose_gen` generates a composition spec from a `*.composition.yaml`.
  `Checkout` proves the cross-aggregate invariants `reserve-before-pay` and `no-ship-before-pay`
  over the composed contracts, which no single aggregate could.
- **The data-refined saga model earned its keep.** It caught a money-losing bug in the saga as first
  drawn: a single refund attempt could leave a captured payment un-refunded during compensation. TLC
  produced the exact six-step counterexample. The fix (one idempotent, retried compensate step with an
  explicit FailedDirty residual for exhausted retries) is proven.

`make verify-formal` now checks 11 proofs across both examples, all green.

## Update 2: strains 3 and 4 resolved; the hardened gates finished the design

A design review of the toolchain (four adversarial tracks: the formal layer, the gates, the
methodology, and the claims) found that several green checks were weaker than they read, and the
fixes landed here first:

- **Strain 3 is closed.** The distributed C4 is now fully exercised: the Architecture Contract is
  v2 (elements bound to `workspace.dsl`, boundary `modules:` for the Elixir target, mitigation rows
  keyed by backticked dependency ids covering the bus, the four DBs, the gateway, and the carrier),
  and G2 verifies it non-vacuously (`checked:` counts printed).
- **Strain 4 is closed.** G3 regenerates each oracle in memory and diffs it against the committed
  copy; a stale oracle is DRIFT. The oracle rows carry content-derived STABLE ids for the tests.
- **The hardened Gx gate found real drift in this design.** SagaStatus said Running while the
  machine said Reserving/Paying/Shipping, and FailedDirty was missing from the enum entirely; the
  enum is now the fine-grained truth. The same gate demanded the five missing lifecycle machines,
  and they now exist (Order, Payment, Reservation, Shipment, OutboxMessage), each with a generated
  oracle, a named-unit matrix with test types and fixtures, and a generated TLC control-flow proof.
- **The annotations are no longer trust points.** `refine_gen` and `compose_gen` reconcile
  `FulfillmentSaga.semantics.yaml` and `checkout.composition.yaml` against the machines before
  emitting anything; a drifted annotation fails generation. The saga data model now compensates PER
  OBLIGATION (partial compensation is representable), and `Checkout` models the full branching
  (step failures, compensation in any order, the FailedDirty stall) with an auto-generated
  clean-compensation invariant, its step order validated against the saga machine's forward chain.

`make verify-formal` now checks 16 proofs across both examples, all green, and `machinery check`
passes both designs with every count non-zero.
