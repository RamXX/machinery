# Stress test: what held and what strained

A second, deliberately different design run through the toolchain to find its limits: a
distributed order-fulfillment platform (microservices, a saga orchestrator, compensation,
transactional outbox), in contrast to go-crm's single-binary CLI over an embedded store.
Target language: Elixir/OTP.

## What held up

- **Modelith scaled.** The larger domain (12 entities, 6 enums, 28 invariants, 8 scenarios,
  distributed failure semantics) lints clean, same as the small one. Gate 1 is not size-sensitive.
- **Oracle generation generalized.** `oracle_gen` produced the saga's transition oracle (17 rows)
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
   hand-authored matrix. Now that `oracle_gen` produces the oracle from the machine, G3 should diff
   the committed oracle against the freshly generated one, making the check a pure generate-and-compare.

## Status

Phase 1 (domain model) complete and clean. The saga machine is authored and its control-flow model is
TLC-verified. The remaining phases (C4, the other machines, BUILD.md, the multi-contract composition)
are the continuation, and items 1 and 2 above are the tooling work that continuation will drive.
