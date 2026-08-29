---
id: MAC-pg7j
title: "W2-E: generator version stamp + freshness stamp-skew (P-F10) with TDD"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:32Z
created_by: ramirosalas
updated_at: 2026-07-22T07:28:28Z
content_hash: "sha256:0bbe65d242774a8e0723535246cef2715c8bef1d19cc3f75cfeff0ed8131009c"
follows: [MAC-r466]
closed_at: 2026-07-22T07:28:28Z
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T07:07:46Z status: open -> in_progress
- 2026-07-22T07:07:47Z auto-follows: linked to predecessor MAC-r466
- 2026-07-22T07:28:28Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]
- Follows: [[MAC-r466]]

## Comments

### 2026-07-22T07:28:28Z ramirosalas
DONE: version stamp (internal/version, all generators, freshness strip, skew note in check), gates/pack.go readFileOrErr, install.go ReadDir, 14 pack experiments registered. TestGoldenOracle now joins wave-3 golden set (stamp line). Stamp formats documented in agent report for docs wave.
