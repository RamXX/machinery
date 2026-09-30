---
id: MAC-m7bh
title: "Screen-contract artifact, required-control generator and screen-coverage gate"
status: open
priority: 1
type: feature
labels: [story, human-surface, h2-origin]
parent: MAC-lnp2
created_at: 2026-09-30T23:26:33Z
created_by: ramirosalas
updated_at: 2026-09-30T23:26:33Z
content_hash: "sha256:97b056daf1437532e0ed4b88ab6453b81b6b0dd1e92731df04a39cbc21ec91f2"
blocks: [MAC-5n2i, MAC-o493, MAC-q24l, MAC-xj6q]
---

## Description
Add a screen-contract artifact to the design corpus, a generator that derives a DOM-sweep required-control list from it, and a gate that fails when a human act has no screen contract.

MOTIVATION (H2, 2026-09-30, rulings 425 and 424). The support page rendered a pending consent request with no approve or deny control; the attest page was a raw JSON form that met its obligation text; the review showed no page image. Gu-surfaces already maps each human act to a named surface (free text in design/surfaces.yaml), but nothing states what that surface must show, in what readable form, or which controls it may not ship without. H2's DOM-bound sweep kept a hand-maintained control list that never named approve or deny.

SCOPE
- Schema (strict, unknown keys are errors, mirroring surfaces.yaml conventions): design/screens.yaml or design/screens/<Screen>.yaml. Per screen: id, persona (must resolve to a model actor), route or surface reference (must resolve to a surfaces.yaml acts row), shows (facts with their readable form, e.g. "rendered page image", "diff", never a raw hash or JSON dump unless declared as such), controls (stable element ids, each bound to an act), acts (Entity.action or machine event, with outcome on the page and each refusal and how it is shown), decision_evidence (what the persona needs in front of them to decide; for a review, the thing reviewed, not a digest of it), milestone (optional, resolved like Gu).
- Generator: `machinery generate screens` (or a regen step) emits a deterministic required-control list per screen (element id, act, persona) in a machine-readable form a DOM sweep in any stack can load. Generated, never hand-edited, covered by the drift check.
- Gate (working name Gh-screens, letter to be assigned without colliding with the existing --gate vocabulary): closed set = every human act (actor not System) in the model plus every machine event whose actor is a persona. Each must be carried by a control in some screen contract or by an explicit deferral with a reason. Controls must resolve to acts; personas must resolve to actors; screens must resolve to surfaces.yaml rows; a surfaces.yaml row whose surface is a screen with no contract is a finding. checked: line prints screens, controls, acts covered, deferred, and human acts with no contract.
- Activation by artifact presence, as Gu and Gs do; an explicit --gate with no artifact errors.

ACCEPTANCE CRITERIA
- Schema documented (docs/screen-contracts.md and a skills/machinery/references page) with a worked example that reproduces the H2 support-page case (pending consent request, approve and deny controls with element ids, refusal shown on the page).
- Strict parsing: unknown key, dangling persona, dangling act, dangling surface row and duplicate element id are each an ERROR with a table-driven test.
- Completeness: a human act with neither a control nor a deferral is an ERROR naming the act; test covers model action and machine event sources.
- Generator output is deterministic, golden-tested, and drift-checked; a hand edit is DRIFT.
- Absent artifact: gate skipped in the default run; explicit gate letter with no artifact errors.
- Gate lists swept consistently (cmd help, SKILL.md, docs, plugin adapters, README), as MAC-6ylh did for gu.
- Existing corpora with no screens artifact see no new finding on upgrade.
- Coverage floor 80 percent for new code; no release, tag or push.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-30T23:26:37Z dep_added: blocks MAC-5n2i
- 2026-09-30T23:26:37Z dep_added: blocks MAC-o493
- 2026-09-30T23:26:37Z dep_added: blocks MAC-q24l
- 2026-09-30T23:26:38Z dep_added: blocks MAC-xj6q

## Links
- Parent: [[MAC-lnp2]]
- Blocks: [[MAC-5n2i]], [[MAC-o493]], [[MAC-q24l]], [[MAC-xj6q]]

## Comments
