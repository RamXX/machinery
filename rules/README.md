# Shipped consistency rules

The files under `consistency/` are machinery's methodology rules: Datalog in the subset
`internal/datalog` and Soufflé both accept, embedded at build time and evaluated by the Gy-rules
gate (`machinery check --gate gy`) over the design's projected facts
([docs/consistency-layer-proposal.md](../docs/consistency-layer-proposal.md), section 3.3). Every
`finding_<code>` output is an ERROR with code `<code>`; a `warn_<code>` output would be a warning
(none ships). The first attribute of each output names its subject kind, which the gate resolves to
a design location. `TestRulesParity` (internal/gates) runs every file over every bundled example
and two synthetic designs (one on which every output fires, one with a fan-out event) under both
engines and requires identical outputs; `make dagger-job JOB=datalog-parity` runs it against a
pinned Soufflé.

| File | Findings | Declarations and sources it reads |
|---|---|---|
| `authz.dl` | `authz_missing`, `authz_orphan`, `authz_unknown_capability`, `produces_unknown_action` | Modelith actions (`actor: System`), `PRODUCES{}` and `WRITES{}` on matrix rows, the `AUTHORIZATION.md` inventory table, `workspace.dsl` elements |
| `bindings.dl` | `milestone_binding_stale`, `milestone_binding_phantom` | BUILD.md's oracle binding table (`oracle` and `bound at` columns); under `--impl` only, the test files Gt scans and the oracle rows they bind |
| `carriers.dl` | `effect_uncarried`, `carrier_misplaced` | `CARRIES{}` and `WRITES{}` on matrix rows, the unit kind column |
| `facts.dl` | `fact_unresolved` | `USES{}` and `WRITES{}` members against Modelith attributes and enum members, machine context keys, event payload fields, and the row's own `derived:` and `VALUES` declarations |
| `payload.dl` | `payload_twin`, `payload_no_edge`, `payload_unknown_event` | `payload {}` on matrix event rows against the closed payload set of each event-contract edge whose producer or consumer is the unit's component (the ARCHITECTURE.md action-ownership table's owner of an action of the matrix's entity; every edge of the event when the unit has no owner) |
| `records.dl` | `orphan_matrix`, `waived_machine_present` | `machines/*.matrix.md` and `*.machine.json` stems, the `(no machine: <reason>)` waiver on ARCHITECTURE.md persistence-and-placement rows |
| `supersession.dl` | `duplicate_owner`, `supersession_cycle`, `dangling_replacement`, `stale_reservation`, `superseded_in_packet` | `SUPERSEDES{type:...}` and `RESERVED{type:...}` on Architecture Contract rows, `migration.yaml` dispositions, `slices.yaml` `row:` citations |
| `values.dl` | `values_disagree`, `values_conflict` | `VALUES{}` and `VALUES name{}` on matrix rows, Modelith enums |

A new rule is a file here plus a fixture that fails without it and a near-neighbour that must not
fire (`internal/experiments`), plus, where it needs one, a relation in the catalog
(`internal/checker/relations.go`) and its reader (`internal/gates/projection_readers.go`).
