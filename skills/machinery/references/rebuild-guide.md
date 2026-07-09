# Rebuild/hybrid transition reference

Keep current and intended truth separate:

```text
design/legacy/domain.modelith.yaml  # current truth
design/domain.modelith.yaml         # target truth
design/migration.yaml               # checked bridge
```

The target follows the ordinary four phases. `migration.yaml` activates Gm-transition and must use
this strict root shape (unknown keys fail):

```yaml
contract_version: 1
mode: rebuild                       # rebuild | hybrid
legacy: {model: legacy/domain.modelith.yaml}
target: {model: domain.modelith.yaml}
dispositions: []
new_entities: []
assets: []
data_mappings: []
state_mappings: []
phases: []
cutover: {}
risks: []
```

Paths are relative to `design/` and may not escape it.

## Coverage rules

- Every legacy entity has exactly one disposition: `reuse`, `wrap`, `replace`, or `retire`.
  The first three require an existing target; `retire` forbids one.
- Every target entity is a disposition target or appears once in `new_entities`, never both.
- At least one implementation asset (`module | service | schema | data | test`) has a salvage
  strategy, rationale, verification, and (except `retire`) a target.
- Every source and target attribute of a `replace` disposition is covered by `data_mappings`.
  References are `Entity.attribute`; use `-` on one side for derive or drop. Every row requires
  `transform`, `validation`, and `rollback`.
- Every value of a replaced legacy lifecycle enum on `status`, `stage`, or `state` appears in
  `state_mappings` as `LegacyEntity.Value -> TargetEntity.Value` or `drain`, with a reason.

## Phase rules

At least two ordered phases are required. Each has:

```yaml
- id: shadow
  source_of_truth: legacy            # legacy | target
  read_path: shadow                  # legacy | target | shadow
  write_path: legacy                 # legacy | target | dual
  backfill: <repeatable movement plan>
  entry_criteria: <evidence>
  exit_criteria: <evidence>
  rollback: <executable return path>
  observability: [<signal>]
  parity: <required for shadow reads>
  idempotency: <required for dual writes>
  conflict_resolution: <required for dual writes>
  reconciliation: <required for dual writes>
```

The first phase keeps legacy authoritative; the final phase makes target authoritative, and normal
phase order may not return to legacy after that switch. Cutover
must reference a declared target-only phase (target source/read/write), an earlier rollback phase,
decision criteria, rollback window, and maximum data loss.

Every temporary dependency gets a risk row with `dependency`, `detection`, `mitigation`, `residual`,
and `owner`. ARCHITECTURE.md must contain a `Transition architecture` heading; BUILD.md must contain
a `Migration implementation plan` heading.

## Required implementation regressions

- One table test per field mapping and lifecycle mapping, including malformed input and drain rules.
- Characterization against legacy and target adapters with adjudicated deltas.
- Stable operation ids and signed manifests; duplicate, reorder, interruption, and replay tests.
- Reconciliation tests for missing/extra rows, field/state/ownership/authz drift, and tampering.
- Shadow tests prove target results are not served before authority changes.
- Dual-write fault injection on either side before, during, and after acknowledgement.
- Rollback rehearsal proves the declared maximum data loss under live writes.
- Cutover refuses stale or unexplained reconciliation and advances only on declared evidence.
- Target domain, architecture, machine, relational, implementation, and solver gates remain green.

The repository's expanded guide is `docs/rebuild-guide.md`; the worked contract is
`examples/go-crm/design/migration.yaml`.
