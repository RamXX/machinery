---
id: MAC-94n5
title: "Supersession residual: section/file citations, active definition in packets, migration diagnostics"
status: open
priority: 3
type: feature
labels: [consistency-layer, packets, supersession, from-next]
created_at: 2026-09-24T21:32:34Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:34Z
content_hash: "sha256:8a432eaea20f9a386c97dfad3974336fa85a74cd355e28014fe3a835fa3b1135"
blocks: [MAC-f15w]
---

## Description
Source supersession residual: authoritative definition in packets, section/file citation gap, migration diagnostics

Delivered in 0.10.0 (commits 5f9df519, 66243d6e; rules/consistency/supersession.dl): `SUPERSEDES{type:Old}` and `RESERVED{type:Name}` on Architecture Contract rows, `migration.yaml` type ownership, and findings `duplicate_owner`, `supersession_cycle`, `dangling_replacement`, `stale_reservation`, `superseded_in_packet`, with fixtures in internal/experiments/rules_test.go.

Residual (from NEXT-recovered entry 15, H2 PublicationManifest/ReadbackReceipt stale-reservation case):
- `superseded_in_packet` joins on the `row:<path>#<section>#<key>` key only; a packet reaching the old definition through a `section:` or `file:` citation is not caught (docs/consistency-layer-proposal.md:530-532).
- Review/execution packets do not carry the authoritative definition and its current constructor restrictions, and old reservation text is not visibly marked historical in the packet.
- Value-shape closure is not kept visibly separate from source authority, implementation, native adoption and acceptance: a defined type can still have unavailable constructors, and unresolved native authority must remain unresolved after schema closure.
- No migration diagnostics for existing multi-draft designs (never rewriting authored decisions, deleting older evidence or auto-promoting a proposal).

Acceptance criteria:
1. A packet that cites a superseded definition by `section:` or `file:` is flagged, or the limitation is closed by a documented citation rule that makes it impossible; near-neighbour citing the active owner passes.
2. Generated packets for a slice citing a type carry the active owner's definition and its constructor restrictions, and label any superseded or reserved text as historical.
3. A synthetic case shows unresolved native authority stays unresolved after the type is defined.
4. A multi-draft design gets migration diagnostics listing candidate SUPERSEDES/RESERVED declarations without editing any authored file.
5. Docs state these checks do not establish that the chosen schema is semantically correct or that requirements are complete.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:58Z dep_added: blocks MAC-f15w

## Links
- Blocks: [[MAC-f15w]]

## Comments
