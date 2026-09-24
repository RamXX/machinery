---
id: MAC-mkh1
title: "machinery attest --merge-into <file>: replace a claim's row in a hand-kept record"
status: open
priority: 3
type: feature
labels: [attest, ux, from-next]
created_at: 2026-09-24T21:32:50Z
created_by: ramirosalas
updated_at: 2026-09-24T21:32:50Z
content_hash: "sha256:b41c8513682a0695cde4bdf2795812d7f77c7df101e07801eb848df2d7e965ea"
---

## Description
machinery attest --merge-into <file>: replace a claim's row in a hand-kept attestation record

Problem (H2): `machinery attest --design --impl --claim ... --kind current` emits a row with the generator's key order (`attestor` first) and four-space list indent; merging it into a hand-kept `attestations.yaml` needed a script.

Evidence: no merge flag in cmd/machinery/attest.go:92-99.

Proposed fix: `--merge-into <file>` replaces the claim's row in place (or appends if absent), preserving the rest of the file byte-for-byte and emitting the row in the record's existing key order and indentation (or a documented canonical form the record already uses). Write atomically.

Acceptance criteria:
1. Merging a regenerated row into a record containing that claim replaces only that row; all other bytes are unchanged.
2. Merging a claim absent from the record appends it.
3. A malformed target record fails without writing.
4. The merged record passes Gv exactly as a hand-merged one does.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
