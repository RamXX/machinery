---
id: MAC-pm5s
title: "Design-review prompt item: can the persona do it on the page, with the decision evidence in front of them"
status: open
priority: 2
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-09-30T23:26:33Z
content_hash: "sha256:ebb6861ab059357fbbd86ba81347cf0b1186023ac94ab3faf3c5ff640b8cd588"
---

## Description
Add a design-review prompt item to the machinery skill so the zero-context reviewer asks, for every human act, whether the persona can actually do it on the page and whether what they need to decide is in front of them.

MOTIVATION (H2, 2026-09-30). Ten-plus review passes read the model, machines and refusals and never asked the persona's question. The attest page satisfied "the page accepts the sealed record" with a JSON textarea and a typed 64-character digest; the review page showed no image of the thing reviewed. Neither is a data-contract defect, so no existing review question caught them.

SCOPE
- Add to the five-question phase-exit self-review (reality, depth, scope, coverage, consistency) in skills/machinery/SKILL.md and the agents/machinery-build-writer.md and agents/machinery-fsm-author.md copies, under reality or coverage: "For every human act, can the persona actually do it on the page (the control exists, is reachable, and is labelled for them), and is what they need to decide in front of them in readable form (the thing itself, not a hash or raw record)?"
- Add the same item to the persona-walk sweep text in docs/target-surfaces.md and skills/machinery/references/target-surfaces.md, and to the milestone acceptance attestations suggested in the BUILD template.
- Keep the skill's voice; no gate change.

ACCEPTANCE CRITERIA
- The item appears verbatim in SKILL.md, both agent prompts and the target-surfaces docs, and the plugin adapters that mirror them stay in sync (any sync check green).
- Where screen contracts exist, the item points the reviewer at them; where they do not, it still asks the question against surfaces.yaml.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-lnp2]]

## Comments
