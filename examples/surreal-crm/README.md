# surreal-crm: a store-swap rebuild with a legacy surface ledger

The worked example for **Gs-surface** and the bookend sweeps (`docs/surface-ledger.md`), and for a
rebuild whose domain does not change: the running go-crm system (see `examples/go-crm`) rebuilt over
SurrealDB in a local Docker container, replacing the embedded LadybugDB graph store.

What it demonstrates, and where to look:

- **The legacy surface ledger** (`design/legacy/surface.yaml`): 56 surface items enumerated from the
  running system, one `crm <noun> <verb>` command per domain action plus the store shape, every row
  `covered` against the target design except the two that make the point: `crm backup` is `dropped`
  (superseded by the SurrealDB export procedure) and `crm restore` is `deferred` (to the operations
  iteration, tracked in BUILD.md section 12). Four classes are waived with reasons (a local CLI has
  no routes, jobs, events, or integrations).
- **An all-reuse migration contract** (`design/migration.yaml`): every entity disposition is `reuse`
  and there are no field or lifecycle mappings, because a store swap changes no domain shape. Gm's
  work moves to the asset inventory (the repository adapter and schema are `replace`, the domain
  modules and test suites are `reuse`) and the baseline/shadow/cutover phases.
- **Mitigations reclassify failures**: the five machines and their generated oracles are
  byte-identical to go-crm's. What changed is ARCHITECTURE.md section 6: the embedded store's
  file-lock failure class becomes container-start/transaction-conflict contention, absorbed by the
  same `DBLocked` retry overlay with new detection and bounds. The machines did not need to change,
  which is exactly why the reused oracle suites are valid migration evidence.
- **The formal layer travels with the domain**: the TLA+ suite (rung 3 control-flow models, the
  Deal data refinement, the System composition) and the three relational annotations are unchanged
  from go-crm and regenerate byte-identical modules, all committed and CI-checked here in their own
  right (`design/formal/README.md`).

Run the gates and the proofs:

```sh
machinery check design
machinery verify-formal design   # needs Java; 32 proofs
```
