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
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:835e56f220b16f2c3c7972975eb0398fdf594d4294dd75d034bedb46979e9020"
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Accept n:n in v1 (map to n:m or extend enum), reject bounded forms naming v2 projection, regression test through project and check --gate gk with a real n:n model. Evidence: internal/checker/model.go:199-203 still accepts only 1:1,1:n,n:1,n:m and schemas/projection.schema.json:83 enum unchanged; v2 reader (internal/gates/projection_readers.go:281-285) projects n:n but v1 path refuses, as projection_readers_test.go:139 comments. Alloy isolation.go:212 accepts n:n. No CHANGELOG entry. Notes: Consumers with valid Modelith n:n models cannot run project/Gk: blocks a valid input. Hence P1; not P0 since v2 path works.

## History
- 2026-09-25T20:11:08Z dep_added: blocks MAC-g7d4
- 2026-10-06T04:03:39Z dep_removed: no_longer_blocks MAC-g7d4

## Links
- Parent: [[MAC-0p5d]]

## Comments
