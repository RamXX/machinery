---
id: MAC-r466
title: "W1-C: fix gate orchestration findings (NG-*, GATE-*, Gt citation, surface as_of) with TDD"
status: closed
priority: 0
type: task
parent: MAC-40jl
created_at: 2026-07-22T06:28:32Z
created_by: ramirosalas
updated_at: 2026-07-22T07:05:40Z
content_hash: "sha256:62a810628d58e2958a6a14d59d3fdaf37fd0e8d4c152bcc05e18a6350cadbe11"
closed_at: 2026-07-22T07:05:40Z
led_to: [MAC-pg7j, MAC-qssg, MAC-fa79]
---

## Description

## Acceptance Criteria


## Design


## Notes


## History
- 2026-07-22T06:33:02Z status: open -> in_progress
- 2026-07-22T07:05:40Z status: in_progress -> closed

## Links
- Parent: [[MAC-40jl]]
- Led to: [[MAC-pg7j]], [[MAC-qssg]], [[MAC-fa79]]

## Comments

### 2026-07-22T07:05:40Z ramirosalas
DONE: all 17 orchestration findings fixed TDD. Gt citation rule settled (string literal + word boundaries + parse evidence with pipe-splitting). One golden line changes (checkout-split parent narrowing note). Leftovers -> MAC-pg7j: gates/pack.go readOrEmpty x3, install.go:195 Glob, experiments registration. Docs handoffs -> MAC-qssg: Gt rule wording, N/A literal waiver, wildcard-baseline prohibition, as_of key, narrowing-note text, Bash-bypass residual, hook gt vacuous-green asymmetry.
