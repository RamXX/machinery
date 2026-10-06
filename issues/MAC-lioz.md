---
id: MAC-lioz
title: "External dependency e2e posture (real, verified-stand-in, residual) in the Architecture Contract; mocks never count"
status: open
priority: 1
type: feature
labels: [e2e, c4, g2, external-deps]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:49bf8a24565f778d509c10323ea877d3fdf9ff494311cc14ce3c8e653c94e39c"
blocks: [MAC-4ah8, MAC-qax4]
---

## Description
Every external dependency in the Architecture Contract declares how end-to-end tests exercise it. Today dependencies carry a failure posture; they carry no e2e posture, so stand-ins silently become the only proof.

Postures (closed vocabulary):
- `real`: e2e runs against the real system or its official sandbox/test tenant; the recipe names how credentials are provisioned in CI.
- `verified-stand-in`: an emulator or stand-in is allowed in e2e only if a contract suite runs the same interactions against the real system, and that run's evidence is fresh (bound to a date or commit window the design states).
- `residual`: cannot be exercised yet; named, with reason and owner. A milestone whose e2e behaviors touch a residual dependency cannot close without an owner waiver recorded in acceptance.
Mocks inside the process under test are never an e2e posture.

Acceptance criteria:
1. G2 fails an external dependency with no e2e posture, and an unknown posture.
2. verified-stand-in requires a named contract suite id; Gt binds it; stale contract evidence is reported.
3. Each declared e2e behavior (see sibling story) lists the dependencies it touches; the gate derives which postures apply and flags behaviors that touch a residual.
4. Docs (c4 references, build-md-template section on stand-ins) rewritten: stand-ins cover neighbors for fast suites, but e2e proof follows the posture.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add posture vocabulary to the Architecture Contract, G2 checks, Gt binding of the contract-suite id, stale-evidence report, docs rewrite. Evidence: Architecture Contract has no e2e posture field; grep 'verified-stand-in' and 'e2e posture' across repo finds nothing. G2 unchanged in CHANGELOG through 0.11.0. Notes: Independent of MAC-p3pj for the posture field itself; only AC3 (behaviors list touched dependencies) needs the E2E block, so land the posture field first or accept a soft ordering on MAC-p3pj.

## History
- 2026-09-25T19:53:33Z dep_added: blocks MAC-4ah8
- 2026-09-25T19:53:33Z dep_added: blocks MAC-qax4

## Links
- Parent: [[MAC-9dai]]
- Blocks: [[MAC-4ah8]], [[MAC-qax4]]

## Comments
