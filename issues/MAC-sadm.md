---
id: MAC-sadm
title: "W1-B: fix pack/recursion findings (PACK-1..10, SCALE-1..2, pack_revision, strict flag) with TDD"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:31Z
created_by: ramirosalas
updated_at: 2026-07-22T06:52:10Z
content_hash: "sha256:08517d34ca4045a0f52aab219eb65e57272fb07168ce91e393fa88c64bb4b104"
closed_at: 2026-07-22T06:52:00Z
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T06:33:02Z status: open -> in_progress
- 2026-07-22T06:52:00Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]

## Comments

### 2026-07-22T06:52:10Z ramirosalas
DONE by fix agent: all 12 findings fixed TDD (24 tests in internal/experiments/pack_review_experiments_test.go). Handoffs: (1) G2 locator merge-all -> MAC-r466 agent already instructed; (2) docs items (retained:, revision:, waiver single-line, hash-laundering limit) -> MAC-qssg; (3) experiments.go registration of 14 new classes + checkout-split pack regeneration (pack_revision + new hashes) -> MAC-5w16. 3 pre-existing experiments red until regen.
