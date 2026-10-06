---
id: MAC-036o
title: "packet: row:<path>#<section>#<key> cannot address a table row whose first-cell key repeats (two ARCHITECTURE section 3 rows keyed pg)"
status: open
priority: 2
type: feature
labels: [packet, gw, h2]
created_at: 2026-09-10T19:13:54Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:49Z
content_hash: "sha256:f7976a061f1a07dde8d0a40e4bfa4c345ef63fecf55d28b9462ed531da12a023"
related: [MAC-94n5, MAC-n87x]
---

## Description
H2's ARCHITECTURE.md section 3 mitigation table carries two rows whose first backticked key is pg (the record-store row and the retrieval-and-graph-projections row); G2 binds both to the pg DSL element, so renaming a key is not available. Gw reports '2 table rows ... have first-cell key pg; a row citation must be unique', so no slice can cite the Postgres mitigation posture by row although Postgres is the M1 system of record. Proposal: accept an ordinal or a second-cell discriminator (row:ARCHITECTURE.md#3#pg#2, or key on the whole first cell text), keeping uniqueness explicit.

## Acceptance Criteria


## Design


## Notes
Triage 2026-09-24: valid. packet.go:1432 rejects duplicate first-cell keys; H2 ARCHITECTURE.md:616-617 still has two pg rows.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Accept an ordinal or second-cell discriminator in row: citations while keeping uniqueness explicit; add tests and doc. Evidence: internal/gates/packet.go:1432 still rejects duplicate first-cell keys ('a row citation must be unique'); rowKey at packet.go:1396 takes the first backticked span. No ordinal or discriminator support; not in CHANGELOG through 0.11.0. Notes: H2 data not in this repo; the code-side limitation is verified. Related MAC-94n5 and MAC-n87x are packet follow-ups, not blockers.

## History


## Links
- Related: [[MAC-94n5]], [[MAC-n87x]]

## Comments
