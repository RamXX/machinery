---
id: MAC-ris5
title: "W1-D: fix formal-chain findings (FORMAL F1..F8) with TDD"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:32Z
created_by: ramirosalas
updated_at: 2026-07-22T06:55:20Z
content_hash: "sha256:22eb39de2a49b700928227081e2aeb3cfe3cb7305210dc853e3b09a0bed69f39"
closed_at: 2026-07-22T06:55:20Z
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T06:33:02Z status: open -> in_progress
- 2026-07-22T06:55:20Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]

## Comments

### 2026-07-22T06:55:20Z ramirosalas
DONE: FORMAL F1-F8 fixed TDD. Mutations now die at generation (forceWin, saga abort); isolation probes UNSAT + check falsifiable. New counts: fulfillment 11 (was 12), others unchanged. Wave-3 regen: TLA headers (assumptions 5-6) + Isolation.als + Integrity.als across go-crm/surreal/fulfillment/portfolio; checkout-split zero changes. Docs handoffs -> MAC-qssg: example-reprocessing-report counts, integrity-layer.md Exclusive wording, isolation overlap wording. Lint wishlist noted for MAC-re0q scope owner: flag transitions on final states at lint time.
