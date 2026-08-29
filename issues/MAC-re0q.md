---
id: MAC-re0q
title: "W1-A: fix IR/lint/oracle findings (IR-F01..F30) with TDD"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:31Z
created_by: ramirosalas
updated_at: 2026-07-22T06:59:11Z
content_hash: "sha256:18cab0ffa68d0f5281306583372b144f801e8645062e7cc5fcf31bd852515a63"
closed_at: 2026-07-22T06:59:11Z
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T06:33:02Z status: open -> in_progress
- 2026-07-22T06:59:11Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]

## Comments

### 2026-07-22T06:59:11Z ramirosalas
DONE: all 26 IR/lint/oracle findings fixed TDD. Zero example fallout; example oracles byte-identical. New surface: _oracle_tag root key, _delays mandatory, name pattern ^[A-Za-z][A-Za-z0-9_]*$, sibling-only bare targets, strict #id.path, onDone requires compound+final. Handoffs -> MAC-qssg: xstate-format.md updates + fsm-author marker-row wording; -> MAC-5w16: golden lint-output shifts, gates G3 output shifts, unreadable-matrix sibling in internal/gates covered by W1-C.
