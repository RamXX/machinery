# Formal artifacts for the surreal-crm design

The complete suite, held to the same standard as every other example (`make verify-formal` runs it
in CI): nothing here is asserted by inspection.

- **Rung 3 (control-flow TLA+, generated):** `Deal.tla`, `Task.tla`, `User.tla`, `Session.tla`,
  `CommandExecution.tla` and their `.cfg` files, regenerated from the machine JSON on every run.
- **Rung 4 (data refinement + composition):** `Deal.semantics.yaml` is the source;
  `DealData.tla`, `DealContract.tla`, and `DealRefinement.tla` are generated after the annotation
  reconciles against the machine. `System.tla` INSTANCEs the generated `DealContract`, so the
  assumption it makes is the module the refinement proves.
- **Static relational layers:** `policy.relational.yaml` -> `Policy.als` + `Policy.oracle.md`
  (Gp-policy), `integrity.relational.yaml` -> `Integrity.als` (Gi-integrity),
  `isolation.relational.yaml` -> `Isolation.als` + `Isolation.oracle.md` (Gn-isolation).
  Regenerate with `machinery alloy design`; the gates diff committed artifacts against a fresh
  generation, so a stale copy is DRIFT.

Run everything with `machinery verify-formal design` from the example root (needs Java; 32 proofs).

Every source annotation and every generated module is byte-identical to
`examples/go-crm/design/formal`, and that is the example's point, not an accident: the store swap
changed the architecture and the failure classes, and the behavior layer's proofs did not move.
The modules are still committed and checked here in their own right, so a future edit to a
surreal-crm machine is caught by this design's own suite, not by trusting its twin.
