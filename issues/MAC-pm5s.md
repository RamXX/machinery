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
updated_at: 2026-10-06T04:03:55Z
content_hash: "sha256:9988c41877ee095b8a2b5449d511b6ad402c4f2f22246b02969027cf1a5a407a"
related: [MAC-m7bh]
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add the verbatim item to SKILL.md, both agent prompts, both target-surfaces docs and the BUILD template attestations, and keep adapter mirrors in sync. Evidence: skills/machinery/SKILL.md has no persona-can-do-it-on-the-page review item. The five-question phase-exit self-review text lives in agents/machinery-build-writer.md:222 and agents/machinery-fsm-author.md:198; docs/target-surfaces.md has the persona-walk sweep at line 40 with no such question. Notes: Independent prose change with no code dependency; Related m7bh only adds a pointer to screen contracts when they exist. The SKILL.md anchor named in the story (the five-question self-review) is in the agent files, so find the SKILL.md equivalent when editing. Cheapest, safest win in the epic; could ship first.

## History


## Links
- Parent: [[MAC-lnp2]]
- Related: [[MAC-m7bh]]

## Comments
