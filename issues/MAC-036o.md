---
id: MAC-036o
title: "packet: row:<path>#<section>#<key> cannot address a table row whose first-cell key repeats (two ARCHITECTURE section 3 rows keyed pg)"
status: open
priority: 2
type: feature
labels: [packet, gw, h2]
created_at: 2026-09-10T19:13:54Z
created_by: ramirosalas
updated_at: 2026-09-10T19:13:54Z
content_hash: "sha256:a18a5e8f457bad8d92ad38242c48d822f8d4e3ba50079f9db09d1f351e210229"
---

## Description
H2's ARCHITECTURE.md section 3 mitigation table carries two rows whose first backticked key is pg (the record-store row and the retrieval-and-graph-projections row); G2 binds both to the pg DSL element, so renaming a key is not available. Gw reports '2 table rows ... have first-cell key pg; a row citation must be unique', so no slice can cite the Postgres mitigation posture by row although Postgres is the M1 system of record. Proposal: accept an ordinal or a second-cell discriminator (row:ARCHITECTURE.md#3#pg#2, or key on the whole first cell text), keeping uniqueness explicit.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
