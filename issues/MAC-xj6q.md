---
id: MAC-xj6q
title: "D&F mapping: Designer output as the source of screen-contract judgment fields"
status: open
priority: 4
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:55Z
content_hash: "sha256:c1b450aef20b196ba0b5c1932fc4a0528f8d8a3fcc982f6347d99371daed4039"
blocked_by: [MAC-m7bh]
---

## Description
Document the mapping from a D&F design layer (for example Paivot's Designer role and DESIGN.md) to machinery screen contracts, as an optional input source, without adding any product dependency on Paivot.

WHY THIS IS A SEPARATE STORY. machinery's own docs do not document a D&F or Designer mapping: docs/test-assurance-contract.md states there are no product dependencies on Paivot, and the only Paivot reference in the skill is the nd_contract evidence note. The mapping lives in the paivot-graph plugin, whose domain-model skill assigns the BA and Designer only to feeding Phase 1 interrogation (EARS sweep); nothing routes the Designer's UX output into the machinery corpus. H2 never ran a UX layer at all, which is part of why the human surface had no artifact. The part of this epic that belongs to the D&F layer is the judgment content of a screen contract: persona goals, what each persona needs in front of them to decide (decision_evidence), readable forms, and journey order across screens. machinery should own the schema, the resolution and the gates; the D&F Designer is the natural author of those judgment fields.

SCOPE
- machinery side: a section in docs/screen-contracts.md (and docs/agent-portability.md or docs/claude-plugin.md as fits) describing screen contracts' `sources:` field accepting a D&F design document, which fields a Designer is expected to supply and which machinery derives (persona resolution, act binding, element ids), and that the contract is authored in the Phase 2 persona walk with the Designer's input when one exists.
- Paivot side (out of this repo, recorded here as a follow-up for paivot-graph): the domain-model or c4 skill maps Designer to screen-contract authorship and DESIGN.md sections to the `shows` and `decision_evidence` fields. File there separately; do not change paivot-graph from this story.

ACCEPTANCE CRITERIA
- Mapping documented stack- and methodology-neutral, with Paivot as one example and a plain-team example (a UX lead) beside it.
- No code path in machinery reads or requires DESIGN.md or any Paivot artifact.
- Follow-up for paivot-graph recorded in the Notes of this issue with the fields to map.
- Depends on the screen-contract story.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes
paivot-graph follow-up (file in that repo, not here): map the Designer role to screen-contract authorship in the domain-model or c4 skill; map DESIGN.md persona goals to screen.persona context, per-screen content to shows (with readable form), and decision points to decision_evidence; journeys to screen order and milestone. Captured 2026-09-30 from H2 rulings 424 to 428.
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Docs-only mapping once screen contracts exist, plus the recorded paivot-graph follow-up (already in Notes). Evidence: No docs/screen-contracts.md exists in docs/; the screen-contract story MAC-m7bh is still open. Only Paivot mention in docs is docs/test-assurance-contract.md:52 (no product dependency). Notes: Real dependency: documents a field set that MAC-m7bh defines. Could be folded into MAC-m7bh's docs pass.

## History
- 2026-09-30T23:26:37Z dep_added: blocked_by MAC-m7bh

## Links
- Parent: [[MAC-lnp2]]
- Blocked by: [[MAC-m7bh]]

## Comments
