---
id: MAC-s3b7
title: "Persona provisioning declarations and a real-path conformance obligation"
status: open
priority: 2
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:55Z
content_hash: "sha256:e0bf7456eba528ce15b40aeed60727dd37dbf4cecfd61fd217c50fd87a245b2e"
related: [MAC-5n2i]
---

## Description
Add persona provisioning declarations to the model, and a conformance row requiring every provisioned persona to be exercised through its real provisioning path at least once.

MOTIVATION (H2, 2026-09-30, ruling 427, instance ruling 417). A second platform-scope human seat could not be provisioned: an ORM eager uniqueness check ran before the contact digest was computed. No suite ran the real provisioning action for a second seat; every suite seeded the bootstrap seat, so the defect was invisible to every gate.

SCOPE
- Model schema: each persona (actor) declares `provisioning: bootstrap` (created by a named bootstrap step, may be seeded by fixtures) or `provisioning: action <Entity.action>` (or a chain of actions, e.g. registration, IdP readback, principal provisioning, consent). Coordinate with Modelith for where the field lives (actor or glossary entry) and keep it strict.
- Lint: the named actions exist; the bootstrap set is explicit; a human actor with no provisioning declaration is a finding once any persona declares one (so adoption is visible, like Gu's actorless count).
- Conformance obligation: for every persona provisioned by action, the plan owes at least one test that provisions that persona through the declared action path (a stable-id or evidence binding Gt can check), and at least one that provisions a second instance where the model admits more than one per scope. Fixtures may seed only bootstrap personas; document this as a BUILD template rule and a verification-evidence judgment claim.
- Gt reports a provisioned persona with no real-path test as uncovered.

ACCEPTANCE CRITERIA
- Schema and rule documented (model reference, BUILD template, verification-evidence) with the H2 platform-seat example.
- Tests: persona with action provisioning and no binding test is an ERROR; with a binding test passes; bootstrap persona needs none; dangling provisioning action is an ERROR; undeclared persona counted and reported.
- Opt-in by first declaration; existing models unaffected.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Add the provisioning field (coordinate with Modelith), lint for dangling actions and undeclared humans, Gt binding for real-path tests, plus docs/template rules. Evidence: grep for 'provisioning:' declarations, persona or actor provisioning in internal/, schemas/, skills/ finds nothing; CHANGELOG through 0.11.0 has no persona provisioning rule (only unrelated JVM/custody provisioning at CHANGELOG.md:556,1048). Model schema, lint and Gt obligation absent. Notes: Related MAC-5n2i is not a hard dependency: this is model plus Gt-binding work and does not need the screen-contract artifact (MAC-m7bh). Parent epic MAC-lnp2 (human-surface governance) still makes sense. Cost of a model-schema change depends on upstream Modelith, which owns the actor format.

## History


## Links
- Parent: [[MAC-lnp2]]
- Related: [[MAC-5n2i]]

## Comments
