---
id: MAC-syos
title: "Context-bounded design artifacts: size budget, sharded documents with generated index, id-based addressing"
status: open
priority: 1
type: epic
labels: [context-budget, small-models, docset]
created_at: 2026-09-25T19:39:57Z
created_by: ramirosalas
updated_at: 2026-09-25T19:39:57Z
content_hash: "sha256:8dd5eae6a8adabccd44ccee58492f640c8e96eb91226d2feeb40b8e5a3713749"
related: [MAC-cup9, MAC-n87x]
---

## Description
Goal: every design artifact an agent reads fits a small model's context, and an agent finds what it needs through an index instead of reading whole files. Serves the small/local-model use case (see MAC-cup9) beyond what execution packets already give implementers: designers, reviewers, conductors, and anyone editing a design source today must open multi-megabyte files.

Evidence (H2 design/, 2026-09-25, bytes and approx tokens at 4 B/token): DECISIONS.md 1.97 MB (~500k), attestations.yaml 1.4 MB (~360k), domain.modelith.md 1.05 MB (~270k), domain.modelith.yaml 920 KB (~235k), STATE.md 880 KB (~225k), BUILD/assess.md 706 KB, BUILD.md 647 KB, BUILD/core.md 629 KB, ARCHITECTURE.md 562 KB, BUILD/answer.md 453 KB, BUILD/ops.md 434 KB. Execution packets are already capped at 64 KiB (Gw-packet); their sources are not.

Principles (owner-approved direction 2026-09-25):
1. One byte budget for every machinery-read design file, equal to the packet budget (64 KiB, about 16k tokens), so the rule is one number.
2. Large logical documents become a directory of entry files plus a GENERATED index (one line per entry: stable id, title, file, bytes). Generated and drift-checked like oracles, never hand-kept, so it cannot go stale. An index over budget becomes an index of indexes.
3. Citations address stable ids, not file paths or byte positions: row:, section:, file:, idcite, attestation covers and ledger reads resolve through the index, so moving an entry between shards never breaks a citation.
4. Gates keep parsing one logical document: a single reader layer maps a logical name to its ordered shards, so gate semantics do not fork per layout.
5. Existing designs migrate by a mechanical converter, never by hand, and a ratchet baselines today's oversized files (they may shrink, not grow) while new files must comply.
6. Splitting reduces what an agent must read, not total content. Retirement is part of it: superseded decisions and closed state move to an archive shard outside the default index.

Scope: files machinery reads or writes under design/ (DECISIONS.md and STATE.md are read by ledger.go, prosecounts.go, clauses.go, idcite.go; attestations.yaml by attest.go; ARCHITECTURE.md and BUILD.md by many gates). Out of scope: implementation trees and project corpora (e.g. H2 sba_mock/priv/corpus). Third-party formats (Modelith YAML, Structurizr DSL) are handled in their own story: DSL already supports !include; Modelith needs an upstream decision.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Related: [[MAC-cup9]], [[MAC-n87x]]

## Comments
