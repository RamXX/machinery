---
id: MAC-6ylh
title: "Gu-surfaces: target-side surface ledger gate"
status: in_progress
priority: 1
type: feature
labels: [story]
created_at: 2026-08-29T23:49:02Z
created_by: ramirosalas
updated_at: 2026-08-29T23:49:33Z
content_hash: "sha256:5f6caef5602744b1b8c9600be99aa4f6d08ab4b7a5ed25438fc57a28f8af660f"
assignee: ramirosalas
---

## Description
Implement Gu-surfaces: the target-side surface ledger gate. Origin: H2 root-cause finding 2026-08-30 (eight admin-gated acts survived ten deep reviews with no named interface; cause: the human-act-to-surface mapping had no artifact, so no gate and no closed review list). This is the forward twin of Gs (legacy/surface.yaml).

ARTIFACT: design/surfaces.yaml (target design). Strict schema, unknown keys are errors (mirror internal/gates/surface.go conventions exactly):
- surface_version: 1 (required), _comment optional, sources: optional list of strings (where the act list was enumerated from).
- acts: required list. Row keys: act (required: "Entity.action" or "knob:<key>"), actor (required), surface (required, non-empty free text naming the screen/admin command/API route/config release), milestone (optional), _comment (optional).
- deferrals: optional list. Row keys: act (required: "Entity.action", "knob:<key>", or "actor:<Name>" to defer one persona wholesale), reason (required, non-empty), _comment (optional).

GATE Gu-surfaces ("Gu-surfaces  target surface ledger"):
- Activation: file presence (HasTargetSurfaces), same convention as Gs/Gp: auto-runs when design/surfaces.yaml exists; an explicit --gate gu with no file errors on the missing artifact, never skips silently.
- Closed set: parse every *.modelith.yaml at the design root with the SAME loading approach the Gs target model uses, extended to read each action's actor: field. Obligated set = every action whose actor is present and not "System".
- Checks, all deterministic: (1) schema strict; (2) COMPLETENESS: every obligated action is covered by exactly one acts row (exact Entity.action match) or by a deferral (act-level, or actor-level via "actor:<Name>"); a missing action is an ERROR naming it; (3) RESOLUTION: an acts row in Entity.action shape must name an existing entity and action (dangling = ERROR) and its actor must equal the model action's actor (mismatch = ERROR: the ledger must not misattribute an act); (4) knob: rows resolve against nothing (open set by design) but duplicate act values anywhere are ERRORS; (5) an actor-level deferral must name an actor that actually appears in the model; (6) actions with NO actor field carry no obligation, but their count prints in the checked line so partial actor adoption stays visible.
- checked line prints: obligated actions, covered, deferred acts, deferred personas, knob rows, actorless actions.

WIRING: internal/gates/suite.go: add {"gu", HasTargetSurfaces} beside gs in both the activation table and the run block (gu runs after gs). Sweep the repo for every enumerated gate list that names gs (cmd help text, docs, skills/machinery/SKILL.md gate lists, .claude-plugin and .codex-plugin adapters, README) and add gu consistently.

TESTS (standard TDD, table-driven, mirror surface_test.go): happy complete ledger; missing obligated action; dangling Entity.action; actor mismatch; unknown key at each level; actor-level deferral covering a persona; duplicate act row; knob rows accepted and counted; absent file = gate skipped in default run; explicit gu with absent file = error; actorless actions counted not obligated. Run the full repo test suite and lint (make targets / golangci per .golangci.yml) green.

DOCS + SKILL (same commit): docs/target-surfaces.md, a short guide in the docs/surface-ledger.md voice: the artifact, the gate, the persona-walk authoring sweep. skills/machinery/SKILL.md: (a) add surfaces.yaml to the output layout listing; (b) add gu to the gate enumerations; (c) in the Phase 2 section add the PERSONA-WALK sweep: for every human persona in the glossary, walk their complete action list into named surfaces before Gate 2, author design/surfaces.yaml, and extend the phase-exit self-review coverage question to include it; (d) mirror any per-reference gate list that would otherwise go stale. Keep the skill's voice; no em dashes anywhere.

CONSTRAINTS: no version bump, no tag, no push (the maintainer will land more changes before releasing). Coverage floor 80 percent for the new code. Do not modify unrelated gates.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-08-29T23:49:33Z status: open -> in_progress
- 2026-08-29T23:49:33Z claimed by ramirosalas

## Links


## Comments
