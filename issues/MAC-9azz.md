---
id: MAC-9azz
title: "Table oracles: a declared fixture set a matrix row cites and Gt binds like transition ids"
status: open
priority: 2
type: feature
labels: [self-design, table-oracle, gt, gw, gv]
parent: MAC-cup9
created_at: 2026-09-24T02:38:21Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:ebb5a783fc9e281b98cad39b0119ba45b19be3a461e47b83f5a26525d7e471d4"
blocks: [MAC-cup9]
was_blocked_by: [MAC-va30, MAC-qxaa]
---

## Description
First story of MAC-cup9; blocks the rest. A non-machine unit (a parser, a hasher, a generator, the fact projection) has a contract row but no oracle a test can bind to. Add a declared table oracle: a design artifact (design/oracles/<name>.table.md or .yaml, decide) listing cases as (stable case id, input reference, expected output reference, content hashes), cited from a matrix row (an ORACLESET-like citation), bound by Gt exactly as transition ids are (a test names the case id whole-token; skipped suites credit nothing), carried in Gw packets, covered by Gv attestation rows, and refreshed by a machinery command that recomputes the hashes from the referenced files (never silently). The bundled examples' golden corpus, the --facts outputs and the Soufflé parity programs are the first three table oracles machinery's own design cites. Acceptance: a synthetic design with one contracted unit and a three-case table oracle; a test binding two of three cases fails Gt naming the third; a changed expected file stales the oracle and Gv; the packet carries the cases; no bundled example changes. Nothing consumer-specific.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Whole story: table oracle format, matrix row citation, Gt binding by whole-token case id, Gw packet carriage, Gv staleness, refresh command, synthetic design tests. Evidence: No table-oracle construct exists: ORACLESET citations only reference machine oracle files (cmd/machinery/attest_implementation_test.go:25); no .table.md/yaml, no Gt binding of case ids from a table. Notes: qxaa (Gt credits ids from self-skipping suites) is a real prerequisite: the new binding must reuse the corrected credit logic. va30 (native test adapters) is an owner-ordering ruling, not a technical dependency of the table-oracle design.

## History
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-va30
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-qxaa
- 2026-10-06T04:03:34Z dep_added: blocks MAC-cup9
- 2026-10-06T04:03:38Z dep_removed: was_blocked_by MAC-va30
- 2026-10-06T05:53:39Z dep_removed: was_blocked_by MAC-qxaa

## Links
- Parent: [[MAC-cup9]]
- Blocks: [[MAC-cup9]]
- Was blocked by: [[MAC-va30]], [[MAC-qxaa]]

## Comments
