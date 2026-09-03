# ReferenceDataCommand machine: named-unit contracts and failure catalog

Transitions are covered by the generated `ReferenceDataCommand.oracle.md`. This operational
envelope gives each analyst reference-data command a bounded, explicit success or failure outcome
without inventing a lifecycle for Index, Security, or CandidateSet.

## Named-unit contracts

| name | kind | signature | contract (pre / post) | maps to | test type | fixture |
|---|---|---|---|---|---|---|
| `selectEligibleConstituents` | actor | `(rankedConstituents) -> candidates` | sorts by provider rank with a stable ticker tie-break and returns only ranks 1 through 30 | invariant `index-top-30`; Index refresh boundary | property | generated ranked lists, including ties and more than 30 rows |
| `upsertSecurityByTicker` | actor | `(security) -> storedSecurity` | resolves by normalized ticker under a unique store constraint, updating the existing row instead of inserting a duplicate | invariant `ticker-unique`; Security upsert boundary | integration + property | duplicate and case-normalized ticker inputs against real DuckDB |
| `buildCandidateSet` | actor | `(sourceIndices) -> CandidateSet` | unions only each source's eligible top-30 securities and deduplicates by normalized ticker | invariants `candidate-from-top-30`, `candidate-deduped`; CandidateSet build boundary | property | overlapping generated index membership lists |
| `recordReferenceError` | action | `(ctx, evt) -> ctx` | records the typed provider, validation, conflict, or I/O failure and preserves the unchanged prior store state | command failure signal | unit | one typed error per boundary contract |
| `cancelReferenceOperation` | action | `(ctx) -> ctx` | cancels the provider request or rolls back the open DuckDB transaction, and waits for cancellation before the command returns | timeout atomicity | integration | cancellable HTTP request and real transaction rollback |
| `recordReferenceTimeout` | action | `(ctx) -> ctx` | records COMMAND_TIMEOUT after cancellation/rollback has completed | command timeout signal | unit | fake clock |

## Failure catalog

| failure | detection | transition | recovery | bounding mitigation or residual risk |
|---|---|---|---|---|
| provider, validation, conflict, or I/O failure | invoked actor `onError` | refreshing/upserting/building to failed, `recordReferenceError` | print the typed cause; do not publish or persist a partial result | provider reads are idempotent; repository writes are atomic and version-guarded |
| command stalls | `after COMMAND_TIMEOUT` | refreshing/upserting/building to failed, `cancelReferenceOperation`, `recordReferenceTimeout` | cancel the provider request or roll back the store transaction before returning a distinct non-zero exit | bounded at 20 s; retry remains safe under the idempotent-read/versioned-write contract |
