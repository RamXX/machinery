---
id: MAC-5w16
title: "W3-G: integration: regenerate artifacts, golden-update, full preflight + verify-formal proof"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:32Z
created_by: ramirosalas
updated_at: 2026-07-22T07:48:39Z
content_hash: "sha256:0f581e4323574e8f70119d71b5ac423b019c1dc42cfdc439b8a7b320de07d1d2"
follows: [MAC-fa79]
closed_at: 2026-07-22T07:48:39Z
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T07:31:10Z status: open -> in_progress
- 2026-07-22T07:31:10Z auto-follows: linked to predecessor MAC-fa79
- 2026-07-22T07:48:39Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]
- Follows: [[MAC-fa79]]

## Comments

### 2026-07-22T07:48:39Z ramirosalas
DONE: regen (21 oracles stamp-only, formal 85/85, packs rev 1), stale fixtures fixed, goldens 99 hunks all classified (1 outside-class hunk root-caused to fulfillment BUILD rewrite), numeric claims corrected (362 tests, coverage table refreshed, overall 79%), preflight green (after fixing 4 lint findings from prior waves), go test ./... 0 failures, make check 7x0 blocking. Nothing committed.
