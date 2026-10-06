---
id: MAC-4cbc
title: "preflight.sh heavy tier: route through the Dagger module so the gate has one definition"
status: open
priority: 3
type: task
labels: [dagger, ci, gates]
created_at: 2026-09-24T21:33:44Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:50Z
content_hash: "sha256:61d91c4f378ef00646dd857720fc1f24d40d1ea6800da3835e34b839a922c10e"
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Route every heavy-tier step through make dagger-job or dagger call; declare the lane hosted-equivalent only after MAC-n35t. Evidence: scripts/preflight.sh:79 is the only dagger-job call (JOB=datalog-parity); the race sweep, integration lane and other heavy steps remain native (header lines 9-13). make dagger-job exists (Makefile:100). Notes: MAC-n35t blocker is real per AC2 only for the 'hosted-equivalent' declaration, not for the routing itself; routing could start earlier and the declaration wait.

## History
- 2026-09-24T21:33:56Z dep_added: blocked_by MAC-n35t

## Links
- Blocked by: [[MAC-n35t]]
- Related: [[MAC-y8lj]]

## Comments
