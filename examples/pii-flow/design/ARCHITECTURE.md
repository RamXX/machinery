# Architecture: PII Flow

The minimal architecture for the external-checker reference design: one
processing pipeline whose only path to the outside world is the export
adapter, and an external analytics destination behind it. The property this
architecture cannot express (no sensitive attribute reaches the sink
unredacted, a dataflow claim over the model's relationship graph) is exactly
the one the Datalog checker carries; see ../README.md. What the architecture
CAN express is asserted below rather than narrated: the export layer never
reaches back into subject data.

## Architecture Contract

```yaml
contract_version: 2
boundaries:
  - id: pii.pipeline
    kind: container
    element: pipeline
    code: [ "internal/pipeline/**" ]
  - id: pii.export
    kind: container
    element: export
    code: [ "internal/export/**" ]
externals:
  - id: external.analytics
    element: analytics
    imports: [ "example.com/analytics-sdk" ]
dependency_rules:
  allow:
    - pii.pipeline -> pii.export
    - pii.export -> external.analytics
  deny:
    - "pii.pipeline -> external.analytics"
  assert:
    # The export layer never reaches back into subject data: redaction
    # happens inside the pipeline, so a reverse path would let export code
    # read unredacted records. Proven over the transitive closure of the
    # allow graph on every G2 run. The flow-level half of the claim (no
    # sensitive attribute reaches the sink unredacted) is the Datalog
    # checker's, over the domain model's relationship graph.
    - no_path: pii.export -> pii.pipeline
  notes:
    - "The pipeline never talks to the analytics destination directly: the deny pins it, and the only allowed route is through pii.export, which holds no subject records."
```

## Dependency mitigation posture

| dependency | failure modes | mitigation | residual | bound |
|---|---|---|---|---|
| `external.analytics` (`analytics`) | down, slow, rejects a batch | spool exports and retry with backoff; deliveries are idempotent by export id | delayed delivery only; no data loss, and no retry path carries unredacted data | retry <= 5, spool 24h |

## Persistence and placement

| component | machine placement | persistence | concurrency |
|---|---|---|---|
| `DataSubject` | pii.pipeline | db row | single writer per subject id |
| `ProcessingActivity` (no machine: a purpose record with no lifecycle) | pii.pipeline | db row per activity | single writer per activity id |
| `AnalyticsExport` (no machine: a spooled delivery record) | pii.export | spool row per export, retained until the destination acknowledges | single writer per export id |
| `Redactor` (not placed: a pure transform inside pii.pipeline; its method is configuration, not a stored record) | n/a | n/a | n/a |

## NFR record

- Security: subject records hold PII and live inside pii.pipeline only; the
  export boundary ships redacted output only. Out of scope beyond that,
  recorded as such.
- Capacity: toy example; out of scope, recorded as such.
- Observability: log every erasure and every export delivery with their ids.
