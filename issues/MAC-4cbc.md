---
id: MAC-4cbc
title: "preflight.sh heavy tier: route through the Dagger module so the gate has one definition"
status: open
priority: 3
type: task
labels: [dagger, ci, gates]
created_at: 2026-09-24T21:33:44Z
created_by: ramirosalas
updated_at: 2026-09-24T21:33:44Z
content_hash: "sha256:63495e5b872d1e3cf75673c5c633b1a18cebaf6bc52eda98108a3d64038d720e"
blocked_by: [MAC-n35t]
related: [MAC-y8lj]
---

## Description
Residual of MAC-zafm (da1226e8, a9dd4b53 in 0.8.0 moved every Linux CI job to `dagger call`; make dagger-ci / dagger-job exist). The heavy tier of scripts/preflight.sh is still a native second definition of the gate: only datalog-parity goes through dagger-job. Two definitions of one policy drift.

Acceptance criteria:
1. Every heavy-tier step in scripts/preflight.sh runs through the Dagger module (make dagger-job or dagger call), so the gate has one owner per policy.
2. The Dagger lane is declared hosted-equivalent only after the container ABA residuals (MAC-n35t) are fixed.
3. release.yml conversion stays tracked separately in MAC-y8lj.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-n35t

## Links
- Blocked by: [[MAC-n35t]]
- Related: [[MAC-y8lj]]

## Comments
