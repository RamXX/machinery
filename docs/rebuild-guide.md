# Rebuilding an existing system with machinery

This guide covers the case between ordinary greenfield design and in-place brownfield adoption:
an existing platform works and contains valuable behavior or assets, but its production foundation
must be rebuilt. machinery calls this **rebuild** mode. **Hybrid** mode uses the same contract when
legacy and target will coexist for an extended period rather than only during a cutover.

The core rule is simple: do not blur the current and intended systems into one optimistic model.
Keep three explicit truths:

1. `design/legacy/domain.modelith.yaml` describes the behavior and data that exist.
2. `design/domain.modelith.yaml` describes the production target.
3. `design/migration.yaml` is the checked transition between them.

The first two models answer different questions. The legacy model is evidence for compatibility,
salvage, and migration. The target model is normative for new implementation. The transition
contract prevents "save what can be saved" from becoming an unreviewed mixture of old assumptions
and new architecture.

A fourth artifact guards the three: `design/legacy/surface.yaml`, the capability disposition
ledger (gate: Gs-surface). Gm proves every entity in the declared legacy model is disposed, but
its coverage universe is that declaration; a subsystem the excavation missed never enters the
model and Gm passes green. The ledger anchors coverage to the legacy system's mechanically
enumerable surface instead: routes, CLI commands, tables, jobs, events, and integrations, each
mapped to a target design element or explicitly dropped or deferred. It is independent of
`migration.yaml` so a clean-break run that drops the migration machinery keeps its completeness
anchor. Full guide: [surface-ledger.md](surface-ledger.md).

## When to choose each mode

| mode | use it when | governing artifact |
|---|---|---|
| greenfield | no production behavior or data must survive | target design only |
| brownfield | the current implementation remains the implementation and is brought under gates incrementally | one as-is design plus the G4 baseline/ratchet |
| rebuild | a new implementation replaces the old one after a bounded coexistence and cutover | legacy model + target model + `migration.yaml` |
| hybrid | old and new components or data paths coexist for a material period | the same three artifacts, with longer-lived phases |

Use brownfield archaeology first when the current truth is poorly understood. Rebuild does not
remove that work; it makes the archaeology an explicit input to a different target. The
[brownfield team guide](brownfield-team-guide.md) remains the right process for discovering and
adjudicating current behavior.

## Output layout

```text
design/
  legacy/
    domain.modelith.yaml    # current truth; keep focused on what must migrate or be retired
    surface.yaml            # capability disposition ledger; source for Gs-surface
  domain.modelith.yaml      # target truth
  migration.yaml            # source for Gm-transition
  ARCHITECTURE.md           # includes a "Transition architecture" section
  BUILD.md                  # includes a "Migration implementation plan" section
  ...                       # the ordinary target architecture, machines, and formal artifacts
```

The target goes through the complete machinery pipeline. `migration.yaml` is not a replacement for
the architecture, state machines, generated oracles, or tests. It is the coverage and safety
contract for moving between two independently coherent systems.

## The Gm-transition gate

`migration.yaml` opts the design into `Gm-transition`. It runs automatically in the default suite
and from progressive hooks when the file exists. Run it alone while authoring:

```bash
machinery check design --gate gm
```

Gm is deterministic and has two responsibilities:

- **Coverage:** every legacy entity has exactly one disposition; every target entity is mapped or
  declared new; every replaced legacy and target attribute is mapped, derived, or dropped; every
  replaced legacy lifecycle value is mapped or drained.
- **Transition safety:** every phase names its source of truth, read path, write path, backfill,
  entry/exit criteria, rollback, and observable signals; shadow and dual-write phases carry their
  additional obligations; cutover and transitional dependency risks are explicit.

Gm deliberately does not execute a migration, inspect a database, or prove transformation code.
Those are implementation concerns held by the test plan in BUILD.md. It checks that no required
decision disappeared between the two domain truths and that the build plan has enough information
to test the transition.

## Contract reference

The root is strict: unknown keys fail the gate because a misspelling would otherwise weaken the
contract silently.

```yaml
contract_version: 1
mode: rebuild                  # rebuild | hybrid
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

Both model paths are relative to `design/` and may not escape it.

### Entity dispositions

Every legacy entity appears exactly once:

```yaml
dispositions:
  - legacy: LegacyCustomer
    target: Customer
    strategy: replace          # reuse | wrap | replace | retire
    rationale: Replace the prototype record with the production aggregate.
```

- `reuse`: the entity contract remains valid and is adopted directly.
- `wrap`: the entity remains behind a target boundary or adapter.
- `replace`: target implementation and shape are new; complete attribute and lifecycle mappings
  are mandatory.
- `retire`: the entity has no target and must not name one.

`reuse`, `wrap`, and `replace` require an existing target entity. A target entity that receives no
legacy disposition must appear in `new_entities`. A target may not be both mapped and new.

The strategy is about the domain shape. The separate asset inventory decides what implementation
material is worth preserving.

### Asset salvage inventory

At least one implementation asset decision is required:

```yaml
assets:
  - name: legacy characterization suite
    kind: test                 # module | service | schema | data | test
    strategy: reuse            # reuse | wrap | replace | retire
    target: target adapter compatibility suite
    rationale: The observable cases are evidence even though the internals are discarded.
    verification: Run the cases against legacy and target adapters and review every delta.
```

`reuse`, `wrap`, and `replace` require `target`; `retire` forbids it. `verification` is the concrete
proof that the decision is safe. Typical candidates are characterization tests, export readers,
schemas, seed data, integration adapters, and operational runbooks.

### Attribute mappings

Every source and target attribute of a `replace` disposition must be covered. Use `-` on exactly
one side for a drop or derivation:

```yaml
data_mappings:
  - source: LegacyUser.password
    target: User.passwordHash
    transform: Verify then hash with argon2id; never persist plaintext.
    validation: Login succeeds and plaintext is absent from storage and logs.
    rollback: Remove the imported user and restore legacy authentication.
  - source: "-"
    target: User.createdAt
    transform: Derive from the earliest source audit timestamp or the import timestamp.
    validation: The value is non-null and carries derivation provenance.
    rollback: Discard the imported target row.
  - source: LegacyThing.obsoleteFlag
    target: "-"
    transform: Drop after recording it in the signed migration manifest.
    validation: The manifest contains every source value.
    rollback: Restore the source snapshot.
```

References are `Entity.attribute` and must bind to their respective model. A mapping between two
entities must agree with the entity disposition. `validation` should be machine-executable where
possible: counts, hashes, invariant checks, normalized result equality, or property tests.

### Lifecycle mappings

For each replaced legacy entity whose `status`, `stage`, or `state` attribute uses an enum, every
legacy value must be mapped or explicitly drained:

```yaml
state_mappings:
  - source: LegacyDeal.Working
    target: Deal.Qualified
    reason: Working means active qualification; later target stages require explicit evidence.
  - source: LegacyJob.Unknown
    target: drain
    reason: The target intentionally has no equivalent; complete or cancel these jobs before cutover.
```

The target value must be on the disposition target. `drain` means no target instance may enter
cutover in that state; BUILD.md must turn that decision into a query, test, and exit criterion.

### Transition phases

Phases are ordered. The first keeps `legacy` as `source_of_truth`; the last makes `target` the
source of truth. Once target becomes authoritative, a later normal phase may not return authority
to legacy; the separate cutover rollback path handles that contingency. At least two phases are
required.

```yaml
phases:
  - id: shadow
    source_of_truth: legacy    # legacy | target
    read_path: shadow          # legacy | target | shadow
    write_path: legacy         # legacy | target | dual
    backfill: Import the snapshot and apply the ordered change log.
    entry_criteria: Repeatable export and target invariant suite are green.
    exit_criteria: Zero unexplained parity or reconciliation drift for seven days.
    rollback: Disable shadow reads and rebuild target from the signed snapshot.
    observability: [read parity mismatch, replication lag, reconciliation drift]
    parity: Compare normalized results and authorization decisions; serve legacy results only.
```

A `shadow` read path additionally requires `parity`. A `dual` write path additionally requires:

```yaml
    idempotency: Use one stable operation id as the unique write key in both systems.
    conflict_resolution: Fail closed and keep the named source of truth authoritative.
    reconciliation: Compare manifests, counts, field hashes, states, and authorization outcomes.
```

Do not declare dual writes without these three answers. "Write both and log errors" is not a
recoverable protocol.

### Cutover

```yaml
cutover:
  phase: cutover
  rollback_phase: dual-write
  decision_criteria: Zero unexplained drift and target durability, authorization, and latency SLOs green.
  rollback_window: 72h
  max_data_loss: zero acknowledged writes
```

`phase` must be a declared target-only phase: target source of truth, target reads, target writes.
`rollback_phase` must exist and precede it. The criteria must be evidence, not a calendar date.

### Transitional dependency risks

The transition topology introduces dependencies the final architecture will not have. Inventory
each one:

```yaml
risks:
  - dependency: ordered change log and dual-write adapter
    detection: Replication lag, missing operation ids, or source/target hash divergence.
    mitigation: Stop cutover, retain legacy authority, and replay from the last reconciled id.
    residual: An unacknowledged in-flight write may require operator review.
    owner: migration platform owner
```

At least one risk is required. Each needs a detection signal, mitigation, honest residual, and
named owner. Put the temporary topology and failure posture under a `Transition architecture`
heading in ARCHITECTURE.md. Put the ordered implementation and regression work under a `Migration
implementation plan` heading in BUILD.md. Gm requires both bridges.

## End-to-end workflow

1. **Archaeology and opening sweep:** model current behavior and data from code, schema, runtime
   evidence, and owner interviews. Write characterization tests before trusting a legacy behavior
   as intentional. Seed `legacy/surface.yaml` by mechanical enumeration (route tables, command
   registrations, schema catalogs, cron and worker lists, outbound calls); most rows start
   deferred, and the ledger becomes the interrogation's work list.
2. **Target design:** run the ordinary four phases without inheriting legacy topology by default.
   The target model and architecture are normative.
3. **Disposition:** decide every legacy entity and implementation asset. A missing decision is a
   Gm error, not a future TODO.
4. **Mapping:** cover all replaced data and lifecycle values. Design stable identifiers and signed,
   repeatable manifests before writing import code.
5. **Closing sweep:** once the design stands, re-mine the legacy system against it and settle
   every ledger row to covered, dropped, or a deliberate deferred. Whatever the docs-first pass
   missed surfaces here as a row that cannot be honestly disposed; run
   `machinery check design --gate gs` until it is green with no placeholder rationales.
6. **Coexistence:** describe source of truth, reads, writes, reconciliation, observability, and
   rollback per phase. Model any stateful transition controller as an ordinary machine when it has
   retries, timeouts, or partial-failure behavior.
7. **Hard TDD:** derive locked mapping, adapter, reconciliation, fault-injection, rollback, and
   cutover tests from BUILD.md. The target's ordinary oracle, boundary, invariant, and formal tests
   stay mandatory.
8. **Advance by evidence:** a phase changes only when its checked entry and exit criteria are true.
   Keep the previous rollback path operational through the declared window. `migration.yaml` has
   an operational twin: the **migration log**, a dated log recording each phase transition as it
   happens, with the evidence that satisfied the entry/exit criteria (the parity report, the
   reconciliation run, the drift query) and who made the call. The Migration implementation plan
   in BUILD.md names the log's location and its owner; a phase transition with no log entry did
   not happen, whatever the calendar says.
9. **Retire deliberately:** remove transition code and legacy dependencies only after cutover exit
   criteria, reconciliation, backup retention, and owner approval are satisfied.

## Regression test checklist

- One table test covers every `data_mappings` row, including invalid and malformed source values.
- One table test covers every `state_mappings` row and proves drained values cannot cross cutover.
- Characterization cases run against legacy and target adapters; intentional deltas are recorded.
- Import and replay are idempotent under duplicate, reordered, interrupted, and resumed delivery.
- Reconciliation detects missing rows, extra rows, field drift, lifecycle drift, ownership drift,
  authorization drift, and manifest tampering.
- Shadow reads never serve target results before authority changes.
- Dual-write tests fault either side before, during, and after acknowledgement.
- Rollback is rehearsed with live writes and proves the declared maximum data loss.
- Cutover tests enforce the decision criteria and refuse stale or unexplained reconciliation.
- Target domain, architecture, machine, relational, implementation, and solver gates stay green.

## Worked example

`examples/go-crm/design` is the complete reference. Its legacy model describes a small working CRM
prototype; the existing full design is the production target. Its contract disposes four legacy
entities, declares five genuinely new target entities, inventories four salvage decisions, covers
all 16 replaced attributes and nine lifecycle values, and defines baseline, shadow, dual-write,
and cutover phases with three transitional dependency risks.
