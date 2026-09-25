---
id: MAC-gscc
title: "1.0 projection rejects Modelith's valid n:n cardinality (accepts non-Modelith n:m)"
status: open
priority: 1
type: bug
labels: [modelith, projection, gk]
parent: MAC-0p5d
created_at: 2026-09-25T20:11:07Z
created_by: ramirosalas
updated_at: 2026-09-25T20:11:07Z
content_hash: "sha256:898caa60dd3dc70ae3b0052d816bf3464cd6461981b25933a56b37e970ec8c16"
blocks: [MAC-g7d4]
---

## Description
Pre-existing, independent of Modelith 0.5: the 1.0 projection (Gk, machinery project, verify-checkers) accepts only `1:1|1:n|n:1|n:m` (internal/checker/model.go:199-203; schemas/projection.schema.json:83). Modelith never allowed `n:m`, and its valid `n:n` is rejected. Verified on examples/pii-flow copy: `unsupported cardinality "n:n"` blocks project and check.

Acceptance criteria:
1. `n:n` accepted; either projected as `n:m` for 1.0 compatibility or the schema enum changes, with a CHANGELOG compatibility note on whichever is chosen.
2. Any other Modelith-valid cardinality (bounded forms) fails naming the v2 projection as the supported path.
3. Regression test with a real n:n model through machinery project and check --gate gk.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-25T20:11:08Z dep_added: blocks MAC-g7d4

## Links
- Parent: [[MAC-0p5d]]
- Blocks: [[MAC-g7d4]]

## Comments
