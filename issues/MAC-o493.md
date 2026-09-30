---
id: MAC-o493
title: "Seal gate for human acts: recorded persona walkthrough in the acceptance evidence"
status: open
priority: 1
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-09-30T23:26:33Z
content_hash: "sha256:d97977ce03217e961698430fd82e28d020ba0afd62f0e427e9387a3fbf0c73a0"
---

## Description
Add a seal gate for human acts: a milestone whose obligations name human acts is not acceptable until a recorded walkthrough of each act by its persona is in the seal directory and the acceptance file names it.

MOTIVATION (H2, 2026-09-30, ruling 426). All four defects reached the owner on surfaces that had passed their milestone reviews; the fresh-context acceptance reviewer read tests and gate output and never saw a page. H2 now adds a line to BUILD.md section 9: a milestone naming a human act is not ACCEPTED until a browser walkthrough of each act by its persona (screenshots or a recording per act, the commit and element ids named) is in the seal directory, produced by the demo lane, and the reviewer states they viewed it.

SCOPE
- Ga-accept extension: derive the set of human acts a milestone's DoD covers (from screen-contract controls bound to acts whose milestone is this one, or from cited oracle ids whose rows are carried by a control).
- Acceptance file schema gains a `walkthroughs` list: act, persona, commit, element ids, and artifact paths (images or recording) inside a declared seal directory; plus a reviewer statement field that the walkthrough was viewed.
- Checks: every human act of the milestone has exactly one walkthrough row; artifact paths exist in the tree at the acceptance commit and are digested (a later change makes the acceptance stale); the walkthrough commit equals or is an ancestor of the acceptance commit; element ids resolve to the act's screen contract. An ACCEPTED verdict missing any of these is an ERROR.
- Separation of duties recorded as a judgment attestation: the walkthrough is produced by the milestone's demo lane, never by the acceptance reviewer.
- BUILD template section 9 text updated ("The human-act walkthrough at every seal").

ACCEPTANCE CRITERIA
- docs/acceptance-gate.md and build-md-template.md updated with the rule and an example acceptance file.
- Tests: milestone with human acts and no walkthroughs is an ERROR on ACCEPTED; missing artifact path is an ERROR; unknown element id is an ERROR; walkthrough commit not an ancestor is an ERROR; milestone with no human acts is unaffected.
- Activation only when screen contracts exist, so current acceptance files stay valid.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes


## History


## Links
- Parent: [[MAC-lnp2]]

## Comments
