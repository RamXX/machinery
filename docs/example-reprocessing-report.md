# Example reprocessing report (2026-07-09)

This report records the full regeneration and verification of the three standalone examples after
the policy, integrity, isolation, and rebuild/hybrid changes. "Before" means the pre-fix branch
tip; "after" means the regenerated working tree recorded with this report.

## Process

For each of `examples/go-crm/design`, `examples/fulfillment/design`, and
`examples/portfolio-engine/design`:

1. lint the Modelith source;
2. render the Modelith markdown and apply the repository's no-em-dash generated-artifact rule;
3. regenerate every machine oracle;
4. regenerate every present relational layer with `machinery alloy`;
5. regenerate the TLA+/refinement/composition artifacts with `verify-formal --gen-only`;
6. run the full deterministic gate suite (with the Go CRM implementation for G4);
7. run `verify-formal` with TLC and Alloy.

The repository golden corpus was then intentionally recaptured and byte-checked, followed by the
full Go suite.

## Before and after

| example | before | after | assessment |
|---|---|---|---|
| Go CRM | One target model; policy, integrity, and isolation artifacts; 8 TLC + 24 Alloy commands | Separate legacy and target models plus a Gm transition contract; target machine/TLA/policy/integrity outputs unchanged; isolation artifacts are field-qualified; 32 solver checks pass | Better transition coverage without behavioral churn in the target. Stable tenant-oracle ids intentionally change once because field identity is now part of the case. |
| Fulfillment | Integrity encoded the forward `Order.payment` field multiplicity but did not prevent two orders sharing one payment; 8 TLC + 3 Alloy commands | `Cardinality_Order_Payment` and `Exclusive_Order_Payment` enforce and check the inverse side; 8 TLC + 4 Alloy commands, all pass | The declared 1:1 relationship is now represented completely. This is the only generated proof-semantic change in this example. |
| Portfolio engine | 3 machine oracles and 6 TLA+/refinement checks | Generated Modelith, machine, oracle, and formal outputs are byte-identical; the same 6 checks pass | No churn: an example outside the changed relational/rebuild surfaces remains stable. |

Across the three examples, solver checks increase from 49 to 50 because fulfillment gains the
missing exclusivity check. Go CRM's solver count stays constant while isolation command names and
oracle identities become collision-safe (`Task.deal -> Deal`, not merely `Task -> Deal`).

## Go CRM transition result

Gm-transition reports:

```text
4 legacy entities
9 target entities
4 dispositions
5 new target entities
4 salvage decisions
16 data mappings
9 state mappings
4 transition phases
1 cutover contract
3 transition risks
```

The contract preserves characterized prototype behavior and the export reader, replaces the
prototype schema and production foundations, retires disposable seed data, and holds baseline,
shadow, dual-write, and cutover phases with explicit authority, parity, idempotency,
reconciliation, rollback, observability, and maximum-data-loss obligations.

## Verification result

- Modelith: all three target models and the legacy Go CRM model lint with zero errors and zero
  warnings.
- Deterministic gates: zero ERROR or DRIFT findings in all three examples.
- Solver checks: Go CRM 32/32, fulfillment 12/12, portfolio engine 6/6.
- Generated machine oracles and behavioral formal models: no unintended diffs.
- Golden and Go test suites: green after reviewing and accepting only the intended output changes.

G3 continues to print the existing `_exhaustive` liveness warnings with an adjacent design-specific
note proving each guard set's codomain coverage. Those are explicit abstraction limits rather than
new failures; no ERROR, DRIFT, solver counterexample, or unaccounted output change remains.

## Conclusion

The after state is suitable to merge. It closes the reviewed proof holes, adds a first-class and
non-vacuous rebuild/hybrid contract, demonstrates that contract on the production-rebuild scenario,
and leaves the unrelated behavioral outputs stable.
