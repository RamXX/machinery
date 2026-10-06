---
id: MAC-7y8d
title: "machinery shard: mechanical converter plus archive retirement; migrate DECISIONS, STATE, ARCHITECTURE, BUILD, attestations"
status: open
priority: 2
type: feature
labels: [context-budget, docset, migration]
parent: MAC-syos
created_at: 2026-09-25T19:39:57Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:6a44c29aa6a961909241d46f5312338ce2ec18d11038bb9b11b16121db501e10"
blocked_by: [MAC-w44m, MAC-snrm]
related: [MAC-6mzy]
---

## Description
`machinery shard <logical-document>` converts a single-file document into the docset layout mechanically (split on entry boundaries, generate INDEX, rewrite nothing else), and `--check` reports what it would do. Includes the retirement split: entries marked superseded/closed go to an archive shard excluded from the default INDEX but still resolvable by id.

Migrations, each proven on H2 and the bundled examples:
- DECISIONS.md: one file per decision.
- STATE.md: one file per milestone or ledger period; append-only history rotates.
- ARCHITECTURE.md: one file per section, contract tables kept whole.
- BUILD.md and BUILD/<shard>.md: one file per milestone/slice (packets unchanged).
- attestations.yaml: one file per claim; coordinate with MAC-6mzy (root-digest manifest in a sidecar).

Acceptance criteria:
1. Conversion is byte-lossless: concatenating shards in INDEX order reproduces the original's parsed content; gate verdicts are identical before and after on every example and on a copy of H2.
2. Every resulting file is under budget, or the command fails naming the entry that cannot be split (a single entry over budget is a content problem, reported as such).
3. Idempotent: running it twice changes nothing.
4. H2 migration measured: largest file and total bytes an agent must read to locate one decision, before and after, recorded in the story notes.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Implement machinery shard with --check, byte-lossless split, archive retirement, idempotence, and H2 measurement. Evidence: No shard command in cmd/machinery or internal; the converter does not exist. Notes: Blockers real: needs the docset reader (w44m) to define layout and the budget (snrm) as the compliance target. MAC-6mzy coordination is soft (attestations sharding only).

## History
- 2026-09-25T19:39:57Z dep_added: blocked_by MAC-w44m
- 2026-09-25T19:39:58Z dep_added: blocked_by MAC-snrm

## Links
- Parent: [[MAC-syos]]
- Blocked by: [[MAC-w44m]], [[MAC-snrm]]
- Related: [[MAC-6mzy]]

## Comments
