---
id: MAC-jr21
title: "Per-slice packet projection: generate bounded executor packets from a design without splitting its sources"
status: open
priority: 0
type: feature
labels: [consumer, h2, build, packets, blocks-h2]
created_at: 2026-09-08T21:59:19Z
created_by: ramirosalas
updated_at: 2026-09-10T14:45:25Z
content_hash: "sha256:b221a043202d40f5f063b5dfb22b313fa63e09ecb4baa6df2d6407f004278133"
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

### 2026-09-10T14:45:25Z ramirosalas
2026-09-10 DESIGN RESOLVED (owner decisions taken this session). The Acceptance Criteria and Design sections above are still empty; this comment is the content for them.

FINDING THAT BLOCKED THE STORY. H2 declares stable slice ids bound to exactly one shard each, but only in prose: BUILD.md, STATE.md, DECISIONS.md, and inside quoted strings in attestations.yaml and acceptance/M1.yaml. Nothing machine-readable binds a slice to the elements it cites, so a byte-reproducible projection could not be derived from the design as it stands. H2's own attestation records the same gap (attestations.yaml:2315).

DECISION 1, slice binding: an AUTHORED SLICE MAP. The design gains a machine-readable artifact binding slice id to its shard and the element ids it cites. The projector reads only that; it never parses prose. Rejected: deriving from shard membership alone (whole-shard granularity cannot reach the budget, since it cannot drop within-shard content) and parsing the prose (not deterministic under ordinary edits).

DECISION 2, budget metric: a BYTE PROXY with a fixed documented divisor. Deterministic, no dependency, no network, consistent with every other machinery gate. The declared budget therefore carries a safety margin against any real tokenizer. Rejected: vendoring a tokenizer (pins the gate to one vendor's tokenization) and a dual byte-gate-plus-estimate (more surface than the problem needs).

SHAPE.
- Authored input: design/slices.yaml. Per milestone, a list of slices: id (stable, of the form <milestone>-S<n>), shard (exactly one BUILD shard path), cites (the element ids the slice binds: oracle rows, matrices, Architecture Contract rows, invariants), and budget (bytes).
- Command: machinery packet <design> --milestone <id> --slice <id> --out <dir>. Projects only what the slice binds: the cited BUILD shard sections, the oracle rows and matrices it binds, the Architecture Contract rows reachable from those elements (boundaries, externals, import rules, mitigation and interface rows), the invariants those rows enforce, and the acceptance entry shape.
- Every excerpt carries its stable id and source path:line, so a developer can navigate back to the authority.
- Output is generated, never hand-edited, and byte-for-byte reproducible for the same design bytes.
- Size report per packet against the declared budget.

GATE (Gb-style, fails closed).
- A packet that exceeds its declared budget fails.
- A projection that drops an obligation the milestone owes fails: every obligation the milestone declares must be claimed by exactly one slice, or carry an explicit recorded waiver. This is the coverage half and it is what stops the projection from being "small because it forgot something".
- The slice map must agree with the design it describes: every cited element id must resolve, and every shard named must exist.

ACCEPTANCE CRITERIA.
1. machinery packet produces, for each of H2 M1-S1..M1-S6, a packet under the declared budget, with the byte size reported per packet.
2. Running the command twice over the same design bytes produces byte-identical output.
3. Every excerpt in a packet carries a stable id and a source path:line that resolves in the design.
4. The gate fails when a packet exceeds its budget, proven by a negative control that raises the projection above it.
5. The gate fails when an obligation the milestone owes is claimed by no slice, proven by a negative control that removes one claim.
6. The gate fails when a slice cites an element id that does not resolve, or a shard that does not exist.
7. The projection never reads BUILD.md or ARCHITECTURE.md wholesale into a packet: a packet's content is bounded by what its slice cites.
8. No prose parsing: removing or re-wording the prose slice narrative in BUILD.md does not change any packet's bytes.

FIRST CONSUMER STEP. H2 authors design/slices.yaml for M1-S1..M1-S6 from the prose that already assigns them. That authoring is the one manual step; everything downstream is generated.
