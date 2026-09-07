# M5 - Reference-data and operations slice

## Outcome

An analyst refreshes the top-30 index, upserts securities, builds a deduplicated candidate set,
backs up the store, restores it, and gets a loud abort on corrupt backup evidence.

## Domain context

Enforce `index-top-30`, `ticker-unique`, `candidate-deduped`, and `candidate-from-top-30`.
ReferenceDataCommand is an execution envelope, not a persistent domain lifecycle. Backup/restore
must preserve all persisted entities and versions exactly. Reference rows are the closed shape
`RankedConstituent{rank, ticker, name, sector}` with rank lexeme `0|-?[1-9][0-9]*`; every
scalar (including the rank) is byte-bounded by maxScalarBytes BEFORE conversion, filtering or
normalization; ineligible-but-valid ranks (31, -1) are filtered, not errors; refresh/upsert/
build carry their explicit ReferenceLimits (upsert admits raw ticker, then name, then sector
bytes BEFORE pf.domain normalization).

## Architecture context

CLI operations enter `pf.app`; reference actors use repository and provider ports. Each command has
one invoke under the 20000 ms publication-admission deadline shared by preparation and
publication (preparation never resets it) and stable error mapping; failed rows require
confirmed nonpublication, timeout rows act only after denial and drain, and an unresolved
publication is reported as an unknown-outcome residual with no failed-row delivery. Backup
reads one immutable consistent DuckDB snapshot as a RAW file copy under
`BackupLimits{maxBytes, deadlineMs}` and returns `BackupReceipt{byteLength, sha256,
schemaVersion:1}`; restore stages to an isolated path and validates structural receipt/limits,
budgets, content length then digest, schema, and integrity before atomic replacement under the
same operation-handle contract.

## Behavior and oracles

Parse `machines/ReferenceDataCommand.oracle.md` and assert `REFE-33a2ae`, `REFE-6e63db`,
`REFE-62d231`, `REFE-cb1193`, `REFE-c51bb5`, `REFE-6b30a5`, `REFE-8c9720`, `REFE-cc283e`,
`REFE-cc10bd`, `REFE-f67aa3`, `REFE-074a67`, and `REFE-c60610`, including timeout cancellation and
error recording actions.

## TDD and implementation

Write all 12 oracle cases, four invariant properties, provider/repository contract tests, and a
byte-exact backup/restore round trip before implementation. Include truncation, checksum mismatch,
wrong schema version, and interrupted staging tests; include destination-exists refusal
(ValidationError DESTINATION_EXISTS, no write), over-budget size refusal (LIMIT_EXCEEDED
before content), existing-store metadata refusal (CorruptError, never stamped), and the
separate closed-byte vs reopened-projection equality observations. Lock RED, then implement
actors and operations.

## Risks and recovery

Bound input inventory, file sizes, output, and command time. Never overwrite the live store before
the staged restore is fully validated. Preserve the live store and isolated staging evidence on
failure; remove only a staging directory still proven to belong to this operation.

## Acceptance

Run all 12 stable ids, four properties, exact round trip, corrupt-evidence abort, and interruption
recovery with real DuckDB. Record evidence and implementation checks in `acceptance/M5.yaml` for the
reviewed commit.
