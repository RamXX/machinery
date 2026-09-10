---
id: MAC-jr21
title: "Per-slice packet projection: generate bounded executor packets from a design without splitting its sources"
status: open
priority: 0
type: feature
labels: [consumer, h2, build, packets]
created_at: 2026-09-08T21:59:19Z
created_by: ramirosalas
updated_at: 2026-09-10T14:36:38Z
content_hash: "sha256:0e80b947fde4973b1d8628de545c9df378315dd650bbd1a35ff790c943ea8dcd"
---

## Description
## Intent
M1 packets for H2 exceed the 200K-token executor budget even after binding each slice to one shard (284K to 467K per slice, with ARCHITECTURE.md alone at 109K). The owner ruled: generate per-slice packets automatically rather than splitting sources by hand or moving to a larger-context executor.

## Scope
A machinery command (working name: machinery packet <design> --milestone <id> --slice <id> --out <dir>) that projects, deterministically, only what a slice binds: the BUILD shard sections the slice cites, the oracle rows and matrices it binds, the Architecture Contract rows (boundaries, externals, import rules, mitigation and interface rows) reachable from those elements, the invariants those rows enforce, and the acceptance entry shape; every excerpt carries the stable id and source path:line so a developer can navigate back; a size report per packet against a declared budget; a Gb-style gate that fails when a packet exceeds its budget or when the projection drops an obligation the milestone owes. Output is generated, never hand-edited, and reproducible byte-for-byte.

## Consumer
H2 M1 slices M1-S1..M1-S6 (design/BUILD.md:1878-1955). Needed before the first RED test.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments

### 2026-09-10T14:36:38Z ramirosalas
2026-09-10: escalated. The H2 agent reports this now blocks H2 M1 from moving forward, so it is a cross-project blocker rather than only a P0 in this backlog. Note for whoever picks it up: the Acceptance Criteria and Design sections are still empty, so the story is not executable as written.
