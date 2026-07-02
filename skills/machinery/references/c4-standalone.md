# C4 architecture (standalone)

Phase 2 fixes how the system is built and deployed, and what each dependency does when it fails.
This is the standalone C4 technique: Structurizr DSL plus a machine-checkable Architecture Contract,
with no dependency on any project settings or external tracker. Two decisions here feed Phase 3
directly: the **dependency mitigation posture** and the **persistence and placement** of each stateful
component. `machinery_check.py --gate g2` verifies the contract deterministically; run it before
calling Gate 2.

## workspace.dsl (Structurizr)

```
workspace "Project" "One-line description" {
  model {
    user = person "User" "Who uses it"

    sys = softwareSystem "System" "What it does" {
      api = container "API" "Business logic" "Elixir/Phoenix"
      db  = container "Database" "State of record" "PostgreSQL" "Database"
      q   = container "Queue" "Async work" "RabbitMQ" "Queue"
    }
    pay = softwareSystem "Payments" "Third-party charges" "External"

    user -> api "Uses" "HTTPS"
    api -> db  "Reads/writes" "SQL"
    api -> q   "Publishes" "AMQP"
    api -> pay "Charges" "REST"
  }
  views {
    systemContext sys { include *; autoLayout }
    container sys { include *; autoLayout }
    deployment sys "production" { include *; autoLayout }
    styles {
      element "Database" { shape Cylinder }
      element "External" { background #999999 }
    }
  }
}
```

Elements: `person`, `softwareSystem`, `container`, `component`. Relationships:
`source -> dest "Description" "Technology"`. Add a `deployment` view when placement matters
(pods, replicas, operators). Prefer the singular, canonical names already used in the domain model.
Tag datastores `Database`, queues `Queue`, and third-party systems `External`: G2 derives the
required mitigation coverage from these tags.

## ARCHITECTURE.md must carry the Architecture Contract (v2)

Embed a parseable YAML block under a heading containing "Architecture Contract", as a yaml code
fence starting with `contract_version`. It is the machine-checkable twin of the narrative. The
shape, from the go-crm example:

```yaml
contract_version: 2
boundaries:
  - id: crm.domain
    kind: component
    element: domain                       # workspace.dsl identifier this boundary binds to;
                                          # defaults to the last segment of the id
    code: [ "internal/domain/**" ]        # required: file globs mapping code to the boundary
    exposes: [ "internal/domain/service.go" ]  # optional public interface
  - id: crm.repo
    kind: component
    element: repo
    code: [ "internal/repo/**" ]
    exposes: [ "internal/repo/repo.go" ]
  - id: crm.model
    kind: component
    element: model
    code: [ "internal/model/**" ]         # no exposes list: all of it is API
externals:
  - id: external.ladybug
    element: store                        # optional: the dsl element it corresponds to
    imports: [ "github.com/LadybugDB/go-ladybug" ]   # import-path prefixes
    # modules: [ "Ladybug" ]              # module-name prefixes (Elixir)
ignore:
  - "internal/testsupport/**"             # source exempt from boundary mapping (test scaffolding)
dependency_rules:
  allow:
    - crm.domain -> crm.repo
    - crm.domain -> crm.model
    - crm.repo   -> crm.model
    - crm.repo   -> external.ladybug      # the sole importer of the embedded store
  deny:
    - "crm.* -> external.ladybug"         # an explicit allow overrides a matching deny
  notes:
    - "All graph access goes through crm.repo."
```

Field semantics:

- **boundary**: `id` (unique), `kind`, `element` (the `workspace.dsl` identifier it binds to;
  defaults to the last segment of the id, so set it explicitly when they differ), `code` (globs,
  required; G4 cannot map the boundary without them), `exposes` (optional: a file entry exposes
  exactly its package directory, a glob entry matches imports), `modules` (Elixir: module-name
  prefixes belonging to the boundary).
- **externals**: `{id, element (optional dsl element), imports: [import-path prefixes],
  modules: [module-name prefixes, for Elixir]}`. Any `dependency_rules` reference to `external.*`
  must be declared here.
- **ignore**: globs for source paths exempt from boundary mapping (test scaffolding, generated code).
- `contract_version: 2` names this format.

G2 verifies: boundaries bind to `workspace.dsl` elements, no duplicate ids, no edge both allowed
and denied, no rule referencing an undeclared boundary or external, and the mitigation coverage
below. G4-import later enforces the rules against the code.

## Interface / boundary contracts (feed the hard-TDD contract tests)

Domain contracts (invariants) come from Modelith. Phase 2 adds **interface contracts** at each boundary,
which is what the test-writer needs for contract tests. For every relationship crossing a boundary, pin:

- **shape**: request and response schema (JSON Schema, OpenAPI fragment, or protobuf message).
- **errors**: the enumerated error responses (these become `onError` branches in Phase 3).
- **idempotency**: is the call safe to retry, and keyed by what.

## Dependency mitigation posture (drives Phase 3 failure transitions)

For every external dependency, fill one row. This is what reclassifies failures rather than deleting
them. Format rules, checked by G2:

- The table header must contain **failure** and **mitigation** columns.
- The **first column** of each row names the dependency by its backticked `workspace.dsl` element id
  or contract external id (e.g. `` `db` ``, `` `q` ``, `` `store` ``). A backticked name that matches
  neither is an error (typo catch).
- **Required coverage**: every contract external plus every DSL element tagged Database, Queue, or
  External must have a row (an external may be covered via its bound dsl element).
- Every residual failure state, in particular any FailedDirty-style one, must say how an operator
  learns about it: add a detection/alerting column, or an operator-signal note in the residual column
  (log line, metric, alert).

| dependency | failure modes | deployment mitigation | residual behavior the FSM must handle | bound | operator signal |
|---|---|---|---|---|---|
| `db` (PostgreSQL) | unavailable, slow, conflict | K8s + operator, HA failover, PgBouncer | transient unavailable during failover; serialization conflicts | retry <= 3, ~5s window | `db_retry_exhausted` metric + alert |
| `pay` (Payments API) | 5xx, timeout, duplicate | none (third party) | timeout, partial charge, must be idempotent | timeout 10s, idempotency key | `payment_failed_dirty` alert per stuck order |
| `q` (Queue) | unavailable, redeliver | clustered, at-least-once | duplicate delivery, must dedupe | dedupe by message id | dedupe-drop counter, redelivery log line |

## NFR record (part of the Architecture Contract conversation)

Record these during Phase 2, even when the answer is "out of scope, recorded as such":

- **security posture**: authn/authz approach, secret handling.
- **capacity assumptions**: expected volume, latency budget where relevant.
- **observability**: what must be logged, metered, alerted; in particular, every FailedDirty-style
  residual state needs a stated operator signal (see the mitigation table rule above).

## Persistence and placement (the C4 to FSM bridge)

For every **stateful** component, decide and record. This determines how the Phase 3 machine is realized
and how concurrent events are serialized. Format rules, checked by Gx-trace once machines exist:

- The table header must contain **placement** and **persistence**.
- The **first column** names each stateful component in backticks.
- Every named component must have a `machines/<Name>.machine.json`, or the row must contain the
  waiver text `(no machine: <reason>)`.

| component | machine placement | persistence | concurrency serialization |
|---|---|---|---|
| `Order` aggregate | in-memory actor (Elixir GenServer per id via Registry) | event-sourced to Postgres, rehydrate on start | actor mailbox (one process per order) |
| `Order` (Go/Rust/Python alt) | none; load-act-save | `state` column + `version` | optimistic lock (`WHERE version = ?`) or `SELECT ... FOR UPDATE` |
| `Pricing` (no machine: pure transform, contract spec instead) | n/a | none | n/a |

Elixir maps almost 1:1 to a supervised process per aggregate. Go, Rust, and Python need the explicit
persisted-state plus lock pattern, or an event-sourced log, because there is no cheap per-entity process.

## Event-contract table (required for multi-component designs)

Coupling through shared DB tables or bus topics is **invisible to G4-import**; this table is the
governing artifact for it. One row per event that crosses a component boundary (every external event
a machine consumes in a choreography must appear here; see the xstate reference for the redelivery
rule). Columns:

- **producer**: `machine.event` or component.
- **consumer**: `machine.event`.
- **payload**: by Modelith attribute reference, never redefined shapes.
- **delivery**: at-least-once / at-most-once / exactly-once-effect, and the mechanism.
- **ordering assumption**.
- **dedupe key**.

| producer | consumer | payload | delivery | ordering | dedupe key |
|---|---|---|---|---|---|
| `Order.ORDER_PAID` | `Shipment.PREPARE` | Order.id, Order.total | at-least-once (outbox -> `q`) | per-order FIFO (partition by Order.id) | Order.id + event type |

## Gate 2 checklist

Deterministic (run `tools/machinery_check.py <design> --gate g2`; needs PyYAML):

- The contract parses, boundaries bind to `workspace.dsl` elements, ids are unique, no edge is both
  allowed and denied, no rule references an undeclared boundary or external.
- Every contract external and every Database/Queue/External-tagged element has a mitigation row
  naming it backticked in the first column.
- Read the `checked:` counts; an empty check is an ERROR, never a silent pass.

LLM-attested (you verify; the tool cannot):

- Every Modelith action maps to an owning component in `workspace.dsl`.
- Every boundary crossing has an interface contract (shape, errors, idempotency).
- Every stateful component has a persistence-and-placement decision (the machine-per-row check runs
  in Gx-trace once machines exist).
- The event-contract table exists for multi-component designs and covers every cross-component event.
- The NFR record is filled (security, capacity, observability).

## Diagram export (optional)

`structurizr-cli export -workspace design/workspace.dsl -format mermaid -output design/diagrams/`
(needs Java 17+). If unavailable, hand-write the equivalent Mermaid `C4Container` block into
ARCHITECTURE.md; the DSL is still the source of truth.
