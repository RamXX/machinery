# ReferenceDataCommand machine: named-unit contracts and failure catalog

Transitions are covered by the generated `ReferenceDataCommand.oracle.md`. This operational
envelope gives each analyst reference-data command a bounded, explicit success or failure outcome
without inventing a lifecycle for Index, Security, or CandidateSet.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `selectEligibleConstituents` | actor | `(rankedConstituents, limits: ReferenceLimits) -> candidates` | admits the closed row shape `RankedConstituent{rank, ticker, name, sector}` (each exactly once; rank lexeme `0\|-?[1-9][0-9]*`) against limits and maxScalarBytes BEFORE conversion or filtering; then filters ranks 1..30, sorts by (rank, canonicalTicker), dedupes canonical identities keeping the first sorted occurrence, and takes the first 30 | invariant `index-top-30`; Index refresh boundary | property | generated ranked lists, including ties and more than 30 rows |
| `upsertSecurityByTicker` | actor | `(security, limits) -> storedSecurity` | admits raw ticker, then name, then sector UTF-8 byte lengths against maxScalarBytes BEFORE pf.domain ticker normalization (failure is ValidationError(LIMIT_EXCEEDED) with no domain/repository effect); then resolves by canonical ticker under the unique store constraint, updating the existing row instead of inserting a duplicate | invariant `ticker-unique`; Security upsert boundary | integration + property | duplicate and case-normalized ticker inputs against real DuckDB |
| `buildCandidateSet` | actor | `(sourceIndices) -> CandidateSet` | unions only each source's eligible top-30 securities and deduplicates by normalized ticker | invariants `candidate-from-top-30`, `candidate-deduped`; CandidateSet build boundary | property | overlapping generated index membership lists |
| `recordReferenceError` | action | `(ctx, evt) -> ctx` | records the typed provider, validation, conflict, or I/O failure of a CONFIRMED-nonpublication outcome and preserves the prior store state as witnessed for THIS attempt (independent writers are not frozen); only resolved Unpublished failures enter this row | command failure signal | unit | one typed error per boundary contract |
| `cancelReferenceOperation` | action | `(ctx) -> ctx` | closes publication permission, cancels the provider request or rolls back the open DuckDB transaction, and completes drain before the timeout row is delivered | timeout atomicity | integration | cancellable HTTP request and real transaction rollback |
| `recordReferenceTimeout` | action | `(ctx) -> ctx` | records COMMAND_TIMEOUT after cancellation/rollback has completed | command timeout signal | unit | fake clock |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| provider, validation, conflict, or I/O failure | invoked actor `onError` | refreshing/upserting/building to failed, `recordReferenceError` | print the typed cause; do not publish or persist a partial result | provider reads are idempotent; repository writes are atomic and version-guarded |
| command stalls | `after COMMAND_TIMEOUT` wins before publication admission | refreshing/upserting/building to failed, `cancelReferenceOperation`, `recordReferenceTimeout` | deny publication, cancel/drain, and only then record the timeout and return InternalError(REFERENCE_TIMEOUT) | 20000 ms is the publication-admission deadline shared by preparation and publication (preparation never resets it), not a total command return bound; retry remains safe under the idempotent-read/versioned-write contract |
| unresolved publication | drained Unresolved outcome | NO failed-row delivery; no lifecycle transition | report `IOError(PUBLICATION_OUTCOME_UNKNOWN)` exit 12 with operation identity; one read-only Reconcile permitted | unknown publication is an operational residual outside this envelope; never rewritten as rollback or ordinary failure |
