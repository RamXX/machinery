---
id: MAC-giok
title: "Gx-trace collects bare READS prose as a consumer declaration"
status: open
priority: 1
type: bug
labels: [regression, gates, gx-trace]
created_at: 2026-09-08T08:33:58Z
created_by: ramirosalas
updated_at: 2026-09-08T08:33:58Z
content_hash: "sha256:5b11f17cb9ef2674c2a1605fccaec78ec7a5c15ee6b988e461a7af7f8da94fd8"
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
