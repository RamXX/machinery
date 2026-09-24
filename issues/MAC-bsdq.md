---
id: MAC-bsdq
title: "Closed group vocabulary rejects a design's own private groups although the 0.10.0 migration note says the design keeps its notation"
status: closed
priority: 0
type: bug
labels: [consistency-layer, declarations, gx, gy, docs, h2, blocks-h2]
created_at: 2026-09-23T23:56:54Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:43Z
content_hash: "sha256:d48534c6f14bb6b547e04909a24deb18dc8a4e3582e936776689c0bf646a827e"
closed_at: 2026-09-24T21:33:43Z
close_reason: "Released in 0.10.1 (private_groups:). Triage 2026-09-24."
---

## Description
v0.10.0 on H2: Gx reports 27 errors (MACHINE-WRITTEN 17, MACHINE-WRITTEN-BY 9, GUARD-HELD 1) and Gy's projection fails on the same parse errors. The 0.10.0 CHANGELOG contradicts itself: 'A design carrying a private group now fails until it maps that notation to the public grammar' versus 'A design that carries its own authorization notation ... keeps it for its own tooling, but machinery no longer reads it'; the skill's migration section says the same as the second. H2's compile-time Elixir reader requires those marks in the matrices. Fix without losing typo protection: an explicit private-group declaration (a private_groups: list in the Architecture Contract v2 fence or .machinery.json, names in the same NAME{ grammar); declared names are skipped by the declaration parser, the projection and the unknown-group rule, never projected, and never satisfy an obligation; an undeclared unknown name stays an error. Correct the CHANGELOG and skill text. Also: the unknown-group message omits RETIRED from the known list though the set accepts it.

## Acceptance Criteria


## Design


## Notes
Fixed on release/0.10.1: private_groups: in the contract fence; projection honors it. H2 scratch tree with the declaration: Gx 27 -> 0.
Released in v0.10.1 (4bb85b78).

## History
- 2026-09-24T21:33:43Z status: open -> closed

## Links


## Comments
