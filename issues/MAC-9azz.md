---
id: MAC-9azz
title: "Table oracles: a declared fixture set a matrix row cites and Gt binds like transition ids"
status: open
priority: 1
type: feature
labels: [self-design, table-oracle, gt, gw, gv]
parent: MAC-cup9
created_at: 2026-09-24T02:38:21Z
created_by: ramirosalas
updated_at: 2026-09-24T02:38:21Z
content_hash: "sha256:8b2fe9e68288bc40de5afa233a275aeeebc3d91d8a6066976d05fa234d78e52c"
---

## Description
First story of MAC-cup9; blocks the rest. A non-machine unit (a parser, a hasher, a generator, the fact projection) has a contract row but no oracle a test can bind to. Add a declared table oracle: a design artifact (design/oracles/<name>.table.md or .yaml, decide) listing cases as (stable case id, input reference, expected output reference, content hashes), cited from a matrix row (an ORACLESET-like citation), bound by Gt exactly as transition ids are (a test names the case id whole-token; skipped suites credit nothing), carried in Gw packets, covered by Gv attestation rows, and refreshed by a machinery command that recomputes the hashes from the referenced files (never silently). The bundled examples' golden corpus, the --facts outputs and the Soufflé parity programs are the first three table oracles machinery's own design cites. Acceptance: a synthetic design with one contracted unit and a three-case table oracle; a test binding two of three cases fails Gt naming the third; a changed expected file stales the oracle and Gv; the packet carries the cases; no bundled example changes. Nothing consumer-specific.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-cup9]]

## Comments
