---
id: MAC-wcrd
title: "Gx-trace READS field grammar narrowed to identifiers without notice; diagnostic misreports multi-word names as empty"
status: open
priority: 1
type: bug
labels: [regression, gates, gx-trace, diagnostics]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:52Z
created_by: ramirosalas
updated_at: 2026-09-08T07:54:52Z
content_hash: "sha256:254db252df4311523af6509f23bbd833d529b93483161806d5f1fef0b2b180de"
---

## Description
## Symptom

Gx-trace's consumer READS parser silently narrowed its field-name grammar in v0.7.0 to a
code-identifier pattern. Natural-language field names, which every existing declaration in a real
consumer design uses, are now rejected with a diagnostic that says the member is "empty".

## Reproduction

Design: H2 (/Users/ramirosalas/workspace/H2) at commit c59bc81, worktree copy. The file is
unchanged between the two runs.

```
machinery check design    # v0.6.11 -> exit 0; "97 event-contract consumer reads declared, 705 declared read fields carried"
machinery check design    # v0.7.0  -> exit 1; 19 of these
```

Offending source, design/machines/ComplianceTask.matrix.md:118:

```
- `schedule.fired` READS{Schedule.name, action, occurrence time, tenant}: ...
```

## Expected

Either the multi-word field name is accepted (as in v0.6.11), or it is rejected with a diagnostic
that names the actual rule ("field names must match `[A-Za-z_][A-Za-z0-9_-]*(\.…)*`; rewrite
`occurrence time` as `occurrence_time`") and the grammar change is documented in the release notes
with a migration command.

## Actual

```
  ERROR  ComplianceTask.matrix.md:118: event 'schedule.fired', consumer 'assess': empty or malformed READS field ' occurrence time'; name each field explicitly, without empty members
```

The member is neither empty nor missing. The message sends the reader hunting for a trailing
comma. Nineteen occurrences across the design, every one a multi-word field name:
`occurrence time`, `governing version`, `recommended action text`, `unsupported-element accounting`,
`version key`, `PAIR KEY`, `Connector ref`, `DocumentDistribution ref`, `regression set`,
`crossing kind`, `item counts`, `official record identity`, `stated effective date`.

## Root cause (code reading)

`internal/gates/readscomplete.go:190`

```go
var readFieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\.[A-Za-z_][A-Za-z0-9_-]*)*$`)
```

and line 211

```go
return nil, "empty or malformed READS field " + ir.Repr(member) + "; name each field explicitly, without empty members"
```

One message covers three distinct rejections (empty member, whitespace inside a name, any other
non-identifier shape) and describes only the first.

## Two asks

1. Decide and state the intended grammar. READS members in this corpus name domain fields in the
   ubiquitous language, not code symbols; an identifier-only grammar is a real semantic narrowing
   of what the matrix can say, not a formatting nit.
2. If the narrowing is intended, split the diagnostic so it names the rule that failed, and add a
   Compatibility/migration entry per docs/release-notes.md rule 3.

## H2 impact

MAJOR. 19 of the 163 new blocking findings. Mechanically fixable (rename to snake_case) but it
touches the ubiquitous-language vocabulary that the domain model, the event-contract table and the
BUILD shards all quote, so the rename is a design revision with its own re-attestation, not a sed.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-ui8a]]

## Comments
