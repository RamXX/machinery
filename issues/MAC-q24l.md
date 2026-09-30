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
updated_at: 2026-09-30T23:26:33Z
content_hash: "sha256:ca1afd2d7455fc0890a33978c33a98a3dd028580d290b503da83f6492f0c8e4f"
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


## History


## Links
- Parent: [[MAC-lnp2]]

## Comments
