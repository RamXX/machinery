# BUILD: PII Flow

Mode: full (self-contained).

Single deliverable. A coding agent with zero prior context can build the processing pipeline from
this document under the hard-TDD protocol in section 11. The checked source artifacts under
`design/` remain authoritative for exact detail; disagreement with this document is a design
defect that must be fixed before implementation.

## 1. Purpose and scope

Build a privacy-sensitive processing pipeline for Controllers: register a data subject, process
subject data for a stated purpose, redact sensitive attributes, export only the redacted record,
and honor terminal erasure. The design exists both as an executable system specification and as
the reference integration for Machinery's external-checker contract.

In scope: subject registration and erasure, purpose records, deterministic redaction, spooled
analytics export, lifecycle persistence, and the hermetic fixed-point dataflow conformance check. Out of scope:
the legal basis and consent-record workflow, deciding whether collection is minimal, user-facing
consent UX, and a complete privacy program; the first two are explicit checker residuals rather
than silently unverified claims.

## 2. Glossary

- **DataSubject** - the person whose personal attributes are held; the only lifecycle entity.
- **Controller** - the operator who registers subjects and executes verified erasure requests.
- **ProcessingActivity** - a purpose-bearing record describing one use of subject data.
- **Redactor** - a pure transform that strips or masks sensitive attributes before handoff.
- **AnalyticsExport** - a spooled, idempotent delivery of a redacted record to the external sink.
- **Sensitive attribute** - `DataSubject.email` or `DataSubject.nationalId` in this design.
- **Export id** - the stable identifier used to deduplicate spooling and destination delivery.
- **External checker** - the committed manifest, projection, fixed-point rules, adapter, and evidence
  that decide a property Machinery's built-in graph gates cannot express.
- **Residual** - an invariant deliberately outside a checker's decision procedure, named with a
  specific enforcement boundary rather than counted as mechanically proven.

## 3. Domain model (the what)

Source of truth: `design/pii-flow.modelith.yaml` (Modelith v1; linted by G1). The relationship
pipeline is `DataSubject (1:n) -> ProcessingActivity (1:1) -> Redactor (1:1) -> AnalyticsExport`.

| entity | attribute | type | meaning |
|---|---|---|---|
| `DataSubject` | `email` | string | sensitive contact identifier |
| `DataSubject` | `nationalId` | string | sensitive government identifier |
| `DataSubject` | `status` | SubjectStatus | `Active` or terminal `Erased` |
| `ProcessingActivity` | `purpose` | string | stated reason for processing |
| `Redactor` | `method` | string | deterministic mask/strip policy identifier |
| `AnalyticsExport` | `destination` | string | configured analytics destination |

Actions are `DataSubject.register` and `DataSubject.erase`, both owned by the Processing Pipeline
and exposed to the Controller through the surfaces in section 4. The closed invariant set is:

| invariant id | statement | owner |
|---|---|---|
| `priv-consent-required` | personal data is processed only with recorded consent | Controller process control; checker residual |
| `priv-minimal-collection` | only attributes necessary for the purpose are collected | design review; checker residual |
| `subject-erased-terminal` | Erased accepts no lifecycle action and is never processed again | DataSubject lifecycle + processing repository |
| `priv-no-unredacted-export` | no sensitive attribute reaches an export sink without redaction | pipeline dataflow + external checker |
| `priv-purpose-present` | every processing activity has a non-empty stated purpose before subject data is read | ProcessingActivity validation |
| `priv-redaction-effective` | Redactor output contains neither email nor national identifier data | Redactor transform |
| `priv-export-idempotent` | a stable export id has at most one destination effect | AnalyticsExport spool and destination contract |

## 4. Architecture (the how)

Source of truth: `design/workspace.dsl` and `design/ARCHITECTURE.md`. Implement two Go containers:
the Processing Pipeline owns subject/activity rows and redaction; the Export Adapter owns the
spool and analytics SDK. One deployment unit of each is sufficient for this reference system;
PostgreSQL row locks serialize writes, and export workers claim spool rows with `FOR UPDATE SKIP
LOCKED`. The external destination is never reachable from pipeline code.

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
    - no_path: pii.export -> pii.pipeline
```

<!-- machinery:embed from="ARCHITECTURE.md" table="edge,shape,errors,idempotency" claims="subset,complete" -->
| edge | shape | errors | idempotency |
|---|---|---|---|
| `pii.pipeline -> pii.export` | `Export(redacted RedactedRecord) -> ExportId`; the argument type carries redacted fields only, so the type system, not a review, keeps subject data on the pipeline side | `ErrSpoolFull`, `ErrRejected` (the destination refused the batch) | idempotent by export id: re-exporting the same id is a no-op |
| `pii.export -> external.analytics` | analytics-sdk batch delivery over HTTPS, one call per spooled export | transport failures and destination rejections mapped here onto `ErrRejected`; no SDK type escapes `pii.export` | at-least-once with the export id as the dedupe key; the destination collapses repeats |

The Controller surfaces are `POST /subjects` for registration and `DELETE /subjects/{id}` for
erasure. Both endpoints authenticate a Controller principal, use request ids for replay safety,
and return only `400 InvalidSubject`, `404 NotFound`, `409 IllegalState`, or `503 Unavailable`.
The target-surface ledger at `design/surfaces.yaml` binds both human acts.

<!-- machinery:embed from="ARCHITECTURE.md" table="component,machine placement,persistence" claims="subset,complete" -->
| component | machine placement | persistence | concurrency |
|---|---|---|---|
| `DataSubject` | pii.pipeline | db row | single writer per subject id |
| `ProcessingActivity` (no machine: a purpose record with no lifecycle) | pii.pipeline | db row per activity | single writer per activity id |
| `AnalyticsExport` (no machine: a spooled delivery record) | pii.export | spool row per export, retained until the destination acknowledges | single writer per export id |
| `Redactor` (not placed: a pure transform inside pii.pipeline; its method is configuration, not a stored record) | n/a | n/a | n/a |

Security: encrypt subject rows at rest, keep PII inside `pii.pipeline`, log identifiers but never
attribute values, and load analytics credentials from the secret store. Capacity: a bounded spool
retains 24 hours and rejects before data loss. Observability: emit erasure audit ids, spool depth,
oldest-spool age, retry count, and rejected-delivery count. The analytics failure posture is retry
with exponential backoff, maximum five attempts per dispatch activation; delayed delivery is the
bounded residual and never creates an unredacted retry route.

### Migration implementation plan

N/A - greenfield design with no legacy/target transition or `migration.yaml`.

### Test environment (not a pack child)

N/A - not a pack child; integration tests use a real PostgreSQL container and a contract-tested
fake analytics HTTPS service in one disposable compose environment.

## 5. Behavior: the state machines (the logic)

`design/machines/DataSubject.machine.json` is the source for the one lifecycle. Registration
creates an `Active` row; it is not a transition and duplicates are rejected by subject id. An
authenticated `erase` destroys the subject attributes and writes erasure evidence in the same
transaction as `Active -> Erased`. `Erased` is final, and every processing repository query
requires `status = Active` so terminal control flow and data access enforce the same rule.

<!-- machinery:embed from="machines/DataSubject.matrix.md" table="name,kind,signature" claims="subset,complete" -->
| name | kind | signature | pre / post | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `recordErasure` | action | `(ctx) -> ctx` | on erase, destroy the subject's stored attributes and write the erasure evidence in the same transaction as the status flip; Erased is final, so processing after erasure is structurally impossible | inv `subject-erased-terminal` (structural) | unit | none |

<!-- machinery:embed from="machines/DataSubject.matrix.md" table="failure,detection,recovery" claims="subset,complete" -->
| failure | detection | transition | recovery | bounding mitigation |
|---|---|---|---|---|
| duplicate `erase` delivery | subject already Erased | none (final state accepts nothing) | drop and log | idempotent by state |
| evidence write fails | write error | none (command rejected, caller retries) | retry with backoff | retry <= 3 |

The lifecycle's refinement declaration is intentionally `control-flow-only`: privacy dataflow is
not falsely claimed as a lifecycle algebra. `machinery verify-formal design` regenerates and TLC
checks `formal/Datasubject.tla`; the external checker independently decides the dataflow invariant.

## 6. Traceability matrix

| invariant id | enforced by (guard / structural) | in component | interface contract | test id(s) |
|---|---|---|---|---|
| `priv-consent-required` | declared checker residual; Controller consent-record workflow must supply runtime evidence before production use | pii.pipeline intake policy | `POST /subjects` process precondition | P-priv-consent-required |
| `priv-minimal-collection` | declared checker residual; privacy review holds the schema to each purpose | pii.pipeline request validation | registration schema review | P-priv-minimal-collection |
| `subject-erased-terminal` | final `Erased` state plus `recordErasure` transaction and Active-only repository filter | DataSubject machine in pii.pipeline | `DELETE /subjects/{id}` | DATA-f61f71, P-subject-erased-terminal |
| `priv-no-unredacted-export` | digest-pinned `pii-flow` fixed-point checker over model, invariants, and relationships; `Redactor` is the taint boundary | pii.pipeline -> pii.export | `Export(RedactedRecord)` | Gk-pii-flow, P-priv-no-unredacted-export |
| `priv-purpose-present` | checker residual carried by non-empty request validation before the repository can read subject data | ProcessingActivity in pii.pipeline | processing intake schema | P-priv-purpose-present |
| `priv-redaction-effective` | checker residual carried by a pure transform property over generated sensitive values | Redactor in pii.pipeline | `Export(RedactedRecord)` construction | P-priv-redaction-effective |
| `priv-export-idempotent` | checker residual carried by spool uniqueness and destination replay property | AnalyticsExport in pii.export | export-id dedupe contract | P-priv-export-idempotent |

The five residuals are known production and implementation obligations, not proof claims: M2 cannot
close until the process controls exist, and M0 cannot close until the purpose, redaction, and
idempotency properties pass. The checker carrier is the fresh projection and evidence bound by
`input_hash`; the lifecycle carrier is the concrete machine unit above.

## 7. Test specification (the hard-TDD oracle)

The transition specification is the committed `design/machines/DataSubject.oracle.md`. The
conformance test opens that path as a string, parses the Markdown `|` table at runtime, and asserts
the next state and actions for every row keyed by stable id. It must cover `DATA-f61f71` and reject
any unrecognized oracle column. There are no guards, so falsifying-clause tests are N/A.

Named-unit test: `recordErasure` runs against a real transaction repository, injects an evidence
write failure to prove rollback, verifies that successful erasure clears both sensitive fields,
and proves repeated erase is state-idempotent. Boundary contract tests prove pipeline packages
cannot construct the analytics SDK type, spool failures map to the enumerated errors, and duplicate
export ids create one destination effect (`P-priv-export-idempotent`). Property tests generate
relationship graphs and require the checker to fail after a deliberately introduced unredacted
bypass; `P-priv-redaction-effective` generates sensitive input and asserts neither sensitive field
survives, while process tests require `P-priv-purpose-present`, a consent record, and rejection of
purpose schemas containing fields not approved for that purpose.

The external-checker engine test copies `checkers.local.example.yaml` to the ignored local
registry, runs `machinery project design`, then `machinery verify-checkers design`. It requires
checker id `pii-flow`, the exact manifest-bound OCI runtime closure, a passing verdict, complete
claimed coverage or a specific residual, zero findings, and byte-identical committed
projection/evidence after a second run.

## 8. State migration

No persisted instances exist yet. Any future rename, split, or removal of `Active` or `Erased`
ships an explicit old-to-new status mapping or a drain rule. `Erased` may never map to a
nonterminal state; migration rejects unknown persisted values instead of coercing them.

## 9. Build plan

**M0 - Walking skeleton.** Implement Controller registration into a real PostgreSQL transaction,
run one purpose record through the real Redactor into the spool, and deliver one redacted export
to the contract-tested HTTPS fake. NFR: structured request/export ids, PII-free logs, secret-loaded
destination credentials, spool-depth and oldest-age metrics. DoD: DATA-f61f71 conformance harness
parses the committed oracle; P-priv-no-unredacted-export, P-priv-purpose-present,
P-priv-redaction-effective, P-priv-export-idempotent, and the two boundary contract tests are green;
`machinery check design --impl <dir> --warnings-as-errors` reports zero findings.

**M1 - Terminal erasure slice.** Implement the Active-only repository filter and atomic erasure
evidence transaction, including injected rollback and duplicate delivery. NFR: erasure audit ids
and failure counters follow M0's structured logging. DoD: DATA-f61f71, P-subject-erased-terminal,
and every `recordErasure` named-unit case are green; no post-erasure processing query returns data.

**M2 - Process-control closure.** Add consent-record lookup and purpose-to-approved-field policy,
with evidence owned by the Controller workflow. NFR: consent/purpose denials emit PII-free reason
codes and counters. DoD: P-priv-consent-required and P-priv-minimal-collection are green against
real repositories; the checker residual reasons still accurately identify the non-static layer.

**M3 - Deterministic checker and release gate.** Pin the checker OCI image and declared input closure
and run projection, offline reproduction, mutation rejection, and idempotence in CI. NFR: checker duration and version
are recorded in build evidence. DoD: `machinery verify-formal design`, `machinery verify-checkers
design`, and `machinery check design --impl <dir> --warnings-as-errors` pass twice with no byte
changes, errors, drift, or warnings.

## 10. Language realization notes

Use Go 1.27.0. Represent status as a closed typed string, centralize transitions in one pure
function, and persist with optimistic version plus row locking. `internal/pipeline` defines
`RedactedRecord`; `internal/export` accepts that type through a narrow port and alone imports the
analytics SDK. PostgreSQL schema migrations are ordered and transactional. HTTP uses generated
request ids and an explicit error envelope; no framework type crosses either component boundary.

### Toolchain and versions

Pin Go 1.27.0, the checker Python userspace by immutable OCI digest plus the exact `linux/amd64` platform,
PostgreSQL 18 by immutable image digest, and the analytics fake by repository commit. Commit `go.mod`/`go.sum` and
the container lock/digest file before M0. Tests use the Go 1.27 standard `testing` package and a
pinned testcontainers dependency. Generate with `machinery oracle design/machines`, `machinery tla
design/machines`, and `machinery project design`; verify with `machinery check design
--warnings-as-errors`, `machinery verify-formal design`, and `machinery verify-checkers design`.

## 11. Hard-TDD protocol (read this before writing any code)

RED begins only after `machinery check design --warnings-as-errors` and both proof commands are
green. A test-writer derives the locked suite from sections 6 and 7, the committed oracle, and the
checker mutation case. Every stable id and property id must appear as a whole token. Before lock,
the suite must compile, obey G4 boundaries, pass format/vet/static analysis, and run red only on
missing-behavior assertions. The implementer may not edit locked tests. GREEN requires the locked
suite plus all design, formal, checker-engine, architecture, and import gates. A wrong test is a
design round trip: edit the source artifacts, regenerate, inspect the stable-id diff, and regenerate
only affected tests. Generated conformance tests live apart from hand-written property and fault
injection tests.

## 12. Open questions and residual risks

- Consent validity and collection minimality require organizational policy and evidence that a
  static relationship checker cannot decide; M2 owns their runtime closure.
- Destination uptime can delay export for at most the 24-hour spool window; exhaustion rejects new
  exports rather than dropping or bypassing redaction.
- Legal retention periods and data-residency regions are not specified by this reference example
  and must be decided before a real deployment.

### What the gates do not verify

Not covered by any deterministic check or proof, by construction: whether the interrogation
extracted the RIGHT invariants (a shallow domain model gates clean); guard and action semantics in
code (the named-unit contracts carry them into tests; a wrong implementation of a correctly-named
guard is caught by tests, not proofs); races between concurrent machine instances, and message
loss, duplication, or reordering between machines (the models are single-instance; the
event-contract table and the idempotency contracts govern those seams, and the tests exercise
them); whether migration transformations preserve real production data (Gm proves decision
coverage, not the implementation or a database run); coupling through shared database tables or
bus topics (invisible to import analysis; the event-contract table governs it); and security,
capacity, and observability beyond what the Phase 2 NFR record captures.
