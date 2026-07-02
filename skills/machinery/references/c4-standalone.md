# C4 architecture (standalone)

Phase 2 fixes how the system is built and deployed, and what each dependency does when it fails.
This is the standalone C4 technique: Structurizr DSL plus a machine-checkable Architecture Contract,
with no dependency on any project settings or external tracker. Two decisions here feed Phase 3
directly: the **dependency mitigation posture** and the **persistence and placement** of each stateful
component.

## workspace.dsl (Structurizr)

```
workspace "Project" "One-line description" {
  model {
    user = person "User" "Who uses it"

    sys = softwareSystem "System" "What it does" {
      api = container "API" "Business logic" "Elixir/Phoenix"
      db  = container "Database" "State of record" "PostgreSQL" "Database"
      q   = container "Queue" "Async work" "RabbitMQ"
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

## ARCHITECTURE.md must carry the Architecture Contract

Embed a parseable YAML block. It is the machine-checkable twin of the narrative.

```yaml
contract_version: 1
boundaries:
  - id: order.service
    kind: container
    code: [ "services/order/**" ]
    exposes: [ "services/order/api/**" ]
  - id: shared.persistence
    kind: component
    code: [ "libs/persistence/**" ]
dependency_rules:
  allow:
    - order.service -> shared.persistence
  deny:
    - order.service -> "*.database_direct"
  notes:
    - "All DB access goes through shared.persistence"
```

Each boundary `id` matches an element in `workspace.dsl`. `code` maps a boundary to file globs.
`exposes` declares the public interface other boundaries may import from. `allow` is the whitelist,
`deny` the blacklist. Bump `contract_version` on any change.

## Interface / boundary contracts (feed the hard-TDD contract tests)

Domain contracts (invariants) come from Modelith. Phase 2 adds **interface contracts** at each boundary,
which is what the test-writer needs for contract tests. For every relationship crossing a boundary, pin:

- **shape**: request and response schema (JSON Schema, OpenAPI fragment, or protobuf message).
- **errors**: the enumerated error responses (these become `onError` branches in Phase 3).
- **idempotency**: is the call safe to retry, and keyed by what.

## Dependency mitigation posture (drives Phase 3 failure transitions)

For every external dependency, fill one row. This is what reclassifies failures rather than deleting them.

| dependency | failure modes | deployment mitigation | residual behavior the FSM must handle | bound |
|---|---|---|---|---|
| PostgreSQL | unavailable, slow, conflict | K8s + operator, HA failover, PgBouncer | transient unavailable during failover; serialization conflicts | retry <= 3, ~5s window |
| Payments API | 5xx, timeout, duplicate | none (third party) | timeout, partial charge, must be idempotent | timeout 10s, idempotency key |
| Queue | unavailable, redeliver | clustered, at-least-once | duplicate delivery, must dedupe | dedupe by message id |

## Persistence and placement (the C4 to FSM bridge)

For every **stateful** component, decide and record. This determines how the Phase 3 machine is realized
and how concurrent events are serialized.

| component | machine placement | persistence | concurrency serialization |
|---|---|---|---|
| Order aggregate | in-memory actor (Elixir GenServer per id via Registry) | event-sourced to Postgres, rehydrate on start | actor mailbox (one process per order) |
| (Go/Rust/Python alt) | none; load-act-save | `state` column + `version` | optimistic lock (`WHERE version = ?`) or `SELECT ... FOR UPDATE` |

Elixir maps almost 1:1 to a supervised process per aggregate. Go, Rust, and Python need the explicit
persisted-state plus lock pattern, or an event-sourced log, because there is no cheap per-entity process.

## Gate 2 checklist

- Every Modelith action maps to an owning component in `workspace.dsl`.
- Every external dependency has a filled mitigation-posture row.
- Every boundary crossing has an interface contract (shape, errors, idempotency).
- Every stateful component has a persistence-and-placement decision.
- The Architecture Contract `allow` / `deny` rules are stated and `contract_version` is set.

## Diagram export (optional)

`structurizr-cli export -workspace design/workspace.dsl -format mermaid -output design/diagrams/`
(needs Java 17+). If unavailable, hand-write the equivalent Mermaid `C4Container` block into
ARCHITECTURE.md; the DSL is still the source of truth.
