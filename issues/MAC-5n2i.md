---
id: MAC-5n2i
title: "Gt falsifier kind: through the rendered surface as the named persona"
status: open
priority: 1
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-09-30T23:26:33Z
content_hash: "sha256:74220591c89ff7c568f3631b1e2516a2a854a1d43698c1797d49528c096021d5"
blocked_by: [MAC-m7bh]
blocks: [MAC-o493]
---

## Description
Add a falsifier kind "through the rendered surface as the named persona" to the Gt vocabulary, require it as the evidence kind for page obligations, and define the attestation re-judgement rule.

MOTIVATION (H2, 2026-09-30, ruling 424). The obligation "the tenant admin approves on /app/support" was discharged by a command-API test plus a row-render assertion, so the missing approve and deny controls passed every gate. Today Gt credits a test by the stable ids it cites (Gt-tests discovery, CLAUSES{} falsifying-clause coverage) and gt.conformance-test-shape judges the shape of the conformance test; neither distinguishes the surface a test drives.

SCOPE
- Evidence kinds: at least `command` (command or API surface), `rendered-surface` (drives the rendered page as the named persona, locates the control by the stable element id from the screen contract, observes the outcome on the page), and `walkthrough` (browser-level recorded run, see the seal story). Declared per test binding (e.g. a KIND{} or persona annotation beside the stable id, or a column in the BUILD.md oracle-bindings table); grammar to be settled in design.
- Rule: an obligation that a screen contract marks as a page obligation (an act carried by a control) is credited only by rendered-surface evidence that names the persona and the element id. A command-kind test is credited to the surface contract (the command, refusal and authorization rows) and never to the page obligation. Gt reports a page obligation whose only evidence is command-kind as uncovered, naming the missing kind.
- Element ids cited by rendered-surface tests must exist in the generated required-control list; a cited id that no screen contract declares is an ERROR.
- Attestation re-judgement: a gt.conformance-test-shape (or successor) row of kind current becomes STALE when any page obligation it covers has only command-kind evidence, or when the screen contract it covers changes digest; the Gv output names the re-judgement reason. Document it in docs/attestation-evidence.md and skills/machinery/references/verification-evidence.md.

ACCEPTANCE CRITERIA
- Vocabulary and annotation grammar documented in the BUILD template section 7 and in verification-evidence.md, with the H2 support-page example (a command test for the consent row plus a rendered-surface test for approve and deny).
- Tests: page obligation with only command evidence is an ERROR; same obligation with rendered-surface evidence citing persona and element id passes; element id not in the generated list is an ERROR; command evidence still credits the surface-contract rows.
- Attestation staleness: a current row goes STALE on a screen-contract digest change and on loss of rendered-surface evidence; tested.
- Opt-in: a design with no screen contracts sees no behaviour change in Gt.
- Stack-neutral wording: LiveViewTest, Playwright, Cypress and similar are examples, not requirements.
- No release, tag or push.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-30T23:26:37Z dep_added: blocked_by MAC-m7bh
- 2026-09-30T23:26:37Z dep_added: blocks MAC-o493

## Links
- Parent: [[MAC-lnp2]]
- Blocks: [[MAC-o493]]
- Blocked by: [[MAC-m7bh]]

## Comments
