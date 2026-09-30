---
id: MAC-lnp2
title: "Human-surface governance"
status: open
priority: 1
type: epic
labels: [human-surface, h2-origin]
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-09-30T23:26:44Z
content_hash: "sha256:aca6d48b3884d2bf36308443651cb82f5f7f00f514a03800bc2a6c1ca927ebf9"
---

## Description
Human-surface governance: give the design corpus an artifact for what a human persona sees and does, and make the gates, the conformance-shape vocabulary, the seal protocol, the packets and the review prompts hold the implementation to it.

ORIGIN (H2, machinery's first real consumer, 2026-09-30). The owner used the platform as a human for the first time and hit four defects in one afternoon that every gate had passed:
1. A second platform-scope human seat could not be provisioned: an ORM eager uniqueness check ran before the contact digest was computed, and every suite seeded the bootstrap seat instead of running the real provisioning action.
2. The support page rendered a pending consent request with no approve or deny control, although the approve command existed and was tested through the command API.
3. The attest page was a raw form (hash table, JSON textarea, a field to type a 64-character image digest) that satisfied its obligation "the page accepts the sealed record" perfectly.
4. The review offered no page image beside the items, while a local generated index did.

DIAGNOSIS (H2 conductor, /Users/ramirosalas/workspace/h2-conductor/human-surfaces-diagnosis.md):
(a) The design describes data contracts, state machines, refusals and authorization, and has no artifact for the human surface, so no gate can object to an unusable page. Gu-surfaces (design/surfaces.yaml, MAC-6ylh) names WHICH surface carries an act, as free text; it says nothing about what the surface must show or which controls it may not ship without.
(b) The conformance-shape gate accepts the shape of a test, not the surface it exercises: a command-API test discharged a page obligation, and the required-control list of the DOM sweep was hand-maintained.
(c) Fixtures seed what the design says is provisioned, and lane hand-backs report inference as observation.

H2 fixes this by convention under owner rulings 424 to 428 (/Users/ramirosalas/workspace/h2-conductor/pending-rulings.md): 424 human-surface falsifier kind; 425 screen contracts generating the sweep; 426 recorded browser walkthrough as seal evidence; 427 no seeded personas; 428 the conductor drives every surface before handing it on. This epic generalises those conventions into machinery so future projects get them from the tool.

STORIES: screen-contract artifact and gate; rendered-surface falsifier kind in Gt; seal gate for human acts; persona provisioning declarations; hand-back and packet vocabulary; design-review prompt item; D&F mapping note (Designer as source for screen contracts).

BOUNDARY (owner): this work must not hinder current H2 development. H2 keeps running its conventions on its pinned machinery; nothing here changes H2's pin, gates or corpus until H2 adopts a release on its own schedule. The work is queued for a later consolidated release under machinery release discipline: accumulate locally, and nothing is tagged, pushed as a release or published without the owner's explicit go. New gates and schema must be opt-in by artifact presence (the Gu/Gs activation convention) or behind the ratchet so an existing corpus does not turn red on upgrade.

## Acceptance Criteria


## Design


## Notes
Related prior work: MAC-6ylh (Gu-surfaces, design/surfaces.yaml) is the act-to-surface ledger this epic extends; its gate and docs (docs/target-surfaces.md) are present in the tree while the issue still reads in_progress. Captured 2026-09-30 from the H2 conductor; H2 pins machinery v0.10.1 and is unaffected until it adopts a release.

## History


## Links


## Comments
