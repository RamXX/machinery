---
id: MAC-wcrd
title: "Gx-trace READS field grammar narrowed to identifiers without notice; diagnostic misreports multi-word names as empty"
status: closed
priority: 1
type: bug
labels: [regression, gates, gx-trace, diagnostics]
parent: MAC-ui8a
created_at: 2026-09-08T07:54:52Z
created_by: ramirosalas
updated_at: 2026-09-08T09:27:49Z
content_hash: "sha256:c7141d0f9d17155d825861382f3f36d5aa167566c977d546d46145f0a11f1bea"
closed_at: 2026-09-08T09:27:49Z
close_reason: "Landed on main (release 0.7.0 line) at 5c35463: 5f2ef53 READS member grammar restored, empty-member diagnostic accurate"
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
- 2026-09-08T09:27:49Z status: open -> closed

## Links
- Parent: [[MAC-ui8a]]

## Comments

### 2026-09-08T08:32:18Z ramirosalas
Fixed on branch fix/MAC-3iga-wcrd at fb045c1 (worktree /Users/ramirosalas/workspace/machinery-worktrees/fix-gates). Not merged, not pushed. Left open.

RULING: REGRESSION, restored. No story narrowed the member grammar. The identifier pattern arrived with 6131abf, fix(MAC-p8ce) "bind complete READS sets to exact event consumers" (closed, accepted). That story's AC #4 asks only that "duplicate/conflicting overrides ... and empty/malformed overrides" be handled deterministically; nothing in it, and nothing in the CHANGELOG 0.7.0 entry for per-edge reads, says a member must be a code identifier. v0.6.11 parsed members with splitClauses (git show v0.6.11:internal/gates/payloadreads.go): TrimSpace, drop empties, no grammar at all, and readscomplete.go used exactly those fields. So `occurrence time`, `PAIR KEY` and `unsupported-element accounting` were valid members and are again.

Change (internal/gates/readscomplete.go): readFieldName is gone. A member is trimmed, optionally backtick-wrapped (the backticks are stripped and the result re-trimmed), and must be non-empty; duplicates are still rejected. The reconciliation against the row's payload cell is whole-token containment, which needs no identifier rule.

Diagnostic: the one message that covered three rejections is split. An empty member now reads `empty READS member '<member>'; name each field explicitly, and drop the stray separator`. A non-empty member is never called empty. The unchanged rejections keep their own wording (duplicate READS field; expected exactly one complete READS{field, ...} declaration per row; malformed READS declaration suffix).

KEPT (documented, CHANGELOG 0.7.0 Changed ~line 134 "Event-contract consumer reads bind per edge", MAC-p8ce AC #1/#2/#3): per-edge binding itself. H2 still receives, and must adopt: 45 "no consumer READS declaration for event/consumer" (was 61 before the fix; 16 edges now bind because their multi-word members parse), 28 "legacy READS ... ambiguous consumer ownership; add an explicit consumer column", 4 "conflicting READS ... exact field set differs from <other matrix>" (was 1; the 3 new ones are edges that previously failed on the grammar error and now parse into genuinely disagreeing field sets across matrices).

Regression tests (internal/gates/reads_consumer_supplemental_test.go, both RED before the fix):
- TestReadsConsumerAcceptsMultiWordFieldNames: READS{Order.id, occurrence time, PAIR KEY, unsupported-element accounting} against a payload carrying all four; asserts zero findings and 4 "declared read fields carried".
- TestReadsConsumerEmptyMemberDiagnosticIsAccurate: READS{Order.id, } still blocks, and the finding says "empty READS member".
TestReadsConsumerMalformedDeclarationsFailClosed (frozen MAC-p8ce RED) passes unchanged, including empty_set, empty_members, trailing_empty_member and duplicate_field.

Verification: go test -race -count=1 ./internal/gates -timeout 15m PASS 89s; go test -count=1 -run 'Golden|Check|Gate|Clause|Reads' ./cmd/machinery PASS 55s (golden corpus unchanged); gofmt clean; go vet clean; golangci-lint run --config .golangci.yml 0 issues.

H2 consumer check (read-only, commit c59bc81): the 19 "empty or malformed READS field" errors are gone. Total blocking 163 -> 88.

SEPARATE DEFECT FOUND, NOT FIXED HERE (needs its own issue): the same "prose is not syntax" shape exists in collectConsumerReads. It collects any matrix line containing the bare word READS, so a contract cell reading "this machine only READS those rows through recomputeMandateConsumed" is judged as a declaration and reported as "expected exactly one complete READS{field, ...} declaration per row". v0.6.11's collectReadsLines required the full READS{...} group, so such prose was never a declaration. 9 occurrences in H2: AgentMandate.matrix.md:82, DeclaredOperation.matrix.md:62, DocumentVersion.matrix.md:130, Identity.matrix.md:60, ModelAsset.matrix.md:29, Principal.matrix.md:85, QualificationSubmission.matrix.md:76, Schedule.matrix.md:107, Trial.matrix.md:108. Same policy verdict as MAC-3iga symptom 1; left untouched to keep this change narrow.
