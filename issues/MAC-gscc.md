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
content_hash: "sha256:63d0eef4790d88077549d46e53b257b54f90848cae61b85ce71b4355b690972b"
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


## Links
- Parent: [[MAC-0p5d]]

## Comments
