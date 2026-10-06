---
id: MAC-q24l
title: "Packet and hand-back vocabulary for surface obligations"
status: open
priority: 2
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:55Z
content_hash: "sha256:36018895798cc81a96389d5b5fc6aaffcb188df73233b00a27d111c8daae9085"
blocked_by: [MAC-m7bh]
was_blocked_by: [MAC-5n2i]
---

## Description
Extend the packet projection and the hand-back vocabulary so an executor is told that a surface obligation is discharged only by rendering it as the persona, and a hand-back must name the persona rendered.

MOTIVATION (H2, 2026-09-30, ruling 428). The attest-prep lane handed back "the consent card is ready" without having rendered it as the tenant admin; the conductor accepted the hand-back without driving the surface. The hand-back reported inference as observation, and nothing in the packet told the executor that observation was required.

SCOPE
- Packet projection (machinery packet, Gw-packet): when a slice claims an obligation carried by a screen-contract control, the packet includes that screen contract (or its projection: persona, controls with element ids, acts, decision evidence) and a fixed rule line stating that the obligation is discharged only by rendering the surface as the named persona. Budget accounting includes the added bytes.
- Hand-back shape: document a machine-readable hand-back block (for example in docs/packet-projection.md and execution-packets.md) with, per surface obligation claimed ready: persona rendered, route, element ids driven, observed outcome, evidence path, commit. A hand-back claiming readiness of a surface obligation without these fields is to be returned by the conductor; optionally a `machinery handback check <file> --packet <dir>` validator that flags missing fields against the packet's surface obligations.
- The machinery skill's conductor text states the return rule.

ACCEPTANCE CRITERIA
- Packet golden tests show the screen-contract projection and the rule line for a slice claiming a control-carried obligation, and no change for slices without one.
- Hand-back shape documented with the H2 consent-card example; if the validator is built, tests cover a complete block, a missing persona and an element id outside the packet.
- Depends on the screen-contract story and the rendered-surface falsifier story.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Project the screen contract and the rule line into packets, document the hand-back block and optional `handback check` validator, and update the conductor text. Evidence: machinery packet (internal/gates/packet.go and internal/pack) has no screen-contract projection or hand-back block; no handback command exists. Notes: m7bh is a real dependency (the contract being projected). 5n2i is only needed for the 'rendered-surface' wording; it is a soft dependency that could be dropped by wording the rule line against the screen contract alone, so list it as a stale blocker only if the owner accepts that. The validator is optional and could be split into its own story.

## History
- 2026-09-30T23:26:37Z dep_added: blocked_by MAC-m7bh
- 2026-09-30T23:26:37Z dep_added: blocked_by MAC-5n2i
- 2026-10-06T04:03:40Z dep_removed: was_blocked_by MAC-5n2i

## Links
- Parent: [[MAC-lnp2]]
- Blocked by: [[MAC-m7bh]]
- Was blocked by: [[MAC-5n2i]]

## Comments
