---
id: MAC-giok
title: "Gx-trace collects bare READS prose as a consumer declaration"
status: open
priority: 1
type: bug
labels: [regression, gates, gx-trace]
created_at: 2026-09-08T08:33:58Z
created_by: ramirosalas
updated_at: 2026-09-08T08:39:42Z
content_hash: "sha256:dc6dbe2db81e9e7c528ede22f5c356bbfe9bca1f36714e6c8cc78db8bff70d21"
---

## Description
## Symptom

Gx-trace's consumer-READS collector treats ANY matrix line containing the bare word `READS` as a
consumer declaration. A contract cell that merely uses the verb, "this machine only READS those
rows through `recomputeMandateConsumed`", is then judged as a declaration and reported as

```
  ERROR  AgentMandate.matrix.md:82: event 'usage.metered', consumer 'core': expected exactly one
         complete READS{field, ...} declaration per row
```

This is the same "prose is not syntax" shape as MAC-3iga, in the sibling collector.

## Reproduction

Design: H2 (/Users/ramirosalas/workspace/H2) at commit c59bc81, unchanged between the two runs.

```
machinery check design --warnings-as-errors   # v0.6.11 -> these rows are not declarations
machinery check design --warnings-as-errors   # v0.7.0  -> 9 of these
```

Nine occurrences, every one an English use of the verb in a contract or residual cell:

- design/machines/AgentMandate.matrix.md:82
- design/machines/DeclaredOperation.matrix.md:62
- design/machines/DocumentVersion.matrix.md:130
- design/machines/Identity.matrix.md:60
- design/machines/ModelAsset.matrix.md:29
- design/machines/Principal.matrix.md:85
- design/machines/QualificationSubmission.matrix.md:76
- design/machines/Schedule.matrix.md:107
- design/machines/Trial.matrix.md:108

## Root cause (code reading)

`internal/gates/readscomplete.go`, in `collectConsumerReads`:

```go
add := func(line string, lineNo int, cells, header []string) {
    if !tokenIn("READS", line) {
        return
    }
    ...
}
```

Every collected line then reaches `parseConsumerReadSet`, whose first rejection is

```go
if len(matches) != 1 || len(readsWord.FindAllStringIndex(text, -1)) != 1 {
    return nil, "expected exactly one complete READS{field, ...} declaration per row"
}
```

so a line with the word and no group can only be reported as malformed.

## v0.6.11 behavior

`git show v0.6.11:internal/gates/payloadreads.go`, `collectReadsLines`:

```go
m := readsDecl.FindStringSubmatch(line)
if m == nil {
    continue
}
```

A declaration was a `READS{...}` GROUP. A line carrying only the word was never collected, by
either tier: v0.6.11's readscomplete.go consumed exactly these `readsLine` values. The nine H2
rows above were therefore silent.

## Policy verdict

Undocumented narrowing. No nd story asked for it (MAC-p8ce bound complete READS sets to exact
event consumers; nothing in it, or in the CHANGELOG 0.7.0 per-edge entry, says the bare word
arms an obligation), and the CHANGELOG documents no such change. Restore the v0.6.11 acceptance:
`collectConsumerReads` treats a line as a declaration only when it carries a full `READS{...}`
group, so prose using the verb is ignored.

What must NOT be relaxed: a line that DOES carry a group is still held to exactly one complete
declaration per row, to the declaration suffix rule, and to non-empty unique members. A row
carrying `READS{a}` and a second bare `READS` is still ambiguous and still fails.

## Acceptance Criteria

1. A matrix contract cell using the verb READS with no `READS{...}` group produces no finding, and
   arms no consumer obligation.
2. A row carrying one complete `READS{...}` group is collected and judged exactly as now,
   including the payload reconciliation and the per-edge ownership rules.
3. A row carrying a group plus a second bare `READS`, or two groups, still fails as ambiguous.
4. Regression test in internal/gates over the H2 shape; the existing frozen MAC-p8ce cases stay
   green.
5. CHANGELOG entry under 0.7.0 Fixed.

## Notes

Found while fixing MAC-3iga and MAC-wcrd. Scope was kept narrow there; this is the third member
of the same family. Fixing it takes the H2 blocking count from 88 to 79.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-08T08:39:42Z ramirosalas
Fixed on branch fix/MAC-3iga-wcrd at bd67c04 (worktree /Users/ramirosalas/workspace/machinery-worktrees/fix-gates), alongside 3e5ab40 (MAC-3iga) and fb045c1 (MAC-wcrd). Not merged, not pushed. Left open.

RULING: regression, restored. `git show v0.6.11:internal/gates/payloadreads.go` collected a declaration only from a full `READS{...}` group (`readsDecl.FindStringSubmatch(line); if m == nil { continue }`), and v0.6.11's readscomplete.go consumed exactly those values. 0.7.0's `collectConsumerReads` took the bare word (`tokenIn("READS", line)`), so a sentence using the verb became a declaration that `parseConsumerReadSet` could only report as incomplete. No story asked for the word (MAC-p8ce bound complete sets to exact consumers) and no CHANGELOG entry documented it.

Change (internal/gates/readscomplete.go): the collector requires `READS{` on the line. Prose using the verb is ignored. A row that does carry a group is judged exactly as before, including the one-complete-declaration-per-row rule, the declaration-suffix rule, the per-edge ownership rules and the payload reconciliation.

Tests (internal/gates/reads_consumer_supplemental_test.go):
- TestReadsConsumerBareWordProseIsNotADeclaration: a residual row naming `markPaid` and reading "this machine only READS those rows and never appends one" beside a valid sibling declaration. RED before the fix with exactly the reported message; green after.
- TestReadsConsumerGroupBesideBareWordStillFails: `READS{Order.id} and separately READS the ledger` still blocks. Passed before and after, pinning the retained tightening.
The frozen MAC-p8ce cases stay green, including unclosed_set, empty_members and duplicate_field.

Verification: go test -race -count=1 ./internal/gates -timeout 15m PASS 88s; go test -count=1 -run 'Golden|Check|Gate|Clause|Reads' ./cmd/machinery PASS 49s (golden corpus unchanged); gofmt clean; go vet clean; golangci-lint run --config .golangci.yml 0 issues.

H2 (read-only, commit c59bc81, machinery check design --warnings-as-errors): 88 blocking before this commit, 78 after.
- "expected exactly one complete READS{field, ...} declaration per row" 9 -> 1
- "legacy READS has ambiguous consumer ownership" 28 -> 26 (ReviewTask.matrix.md:13 and WatchedSource.matrix.md:90 were bare-word prose lines and are no longer collected at all)

The one remaining case is NOT prose and is correctly reported: design/machines/DeclaredOperation.matrix.md:62 opens `READS{Tenant,` and closes the group two lines later, so the row's declaration is genuinely incomplete. v0.6.11 was silent on it because its regex is single-line, but the author did write a group and means it as a declaration; the diagnostic names the real problem and the repair is to join the lines. Kept deliberately, and the CHANGELOG 0.7.0 Fixed entry states it.
