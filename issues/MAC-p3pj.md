---
id: MAC-p3pj
title: "Milestone E2E block: declared end-to-end behaviors with entry points and bound e2e test ids; Demo cites them"
status: open
priority: 1
type: feature
labels: [e2e, dod, gb-plan, gt]
parent: MAC-9dai
created_at: 2026-09-25T19:53:32Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:56Z
content_hash: "sha256:6a36a71795c1026e23d5183a1e9a66ef855248a7f1dd024a5f97ab6147a37f44"
blocks: [MAC-4ah8, MAC-p46v, MAC-qax4]
---

## Description
Every BUILD milestone declares the end-to-end behaviors it proves, and each is bound to an e2e test id.

Format (exact grammar decided in this story): an `E2E:` block per milestone listing behavior ids, each with the entry point it drives (CLI, HTTP route, UI flow, message consumer), the Modelith scenario or Demo it proves, and its e2e test id. The walking skeleton must declare at least one. A milestone with no e2e behavior needs `E2E: N/A - <reason>` (pure internal refactor), which Ga reviewers see.

Acceptance criteria:
1. Gb-plan fails a milestone with no E2E block and no reasoned N/A; the skeleton cannot use N/A.
2. The `Demo:` line must reference at least one declared e2e behavior id, so the demo is the e2e test, not prose.
3. Gt binds e2e test ids statically like oracle ids (bound / unbound per id), reported as a separate e2e category, never pooled with unit coverage.
4. The projection's milestones layer gains e2e rows (behavior, milestone, entry point, test id) so rules and checkers can read them.
5. Template, build-writer agent and example designs updated; CHANGELOG compatibility note with a ratchet so existing designs are not broken on upgrade.

## Acceptance Criteria


## Design


## Notes
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Decide grammar, Gb-plan check, Demo must cite behavior ids, Gt e2e category, projection rows, template and build-writer updates, ratchet and compat note. Evidence: No E2E block grammar anywhere: grep 'E2E:' and 'E2E block' finds nothing; build-md-template.md:242-243 still defines only the prose Demo line; projection milestones layer has no e2e rows. Notes: Root of the e2e chain. Needs a ratchet so existing designs and examples do not break (AC5).

## History
- 2026-09-25T19:53:33Z dep_added: blocks MAC-4ah8
- 2026-09-25T19:53:33Z dep_added: blocks MAC-p46v
- 2026-09-25T19:53:33Z dep_added: blocks MAC-qax4

## Links
- Parent: [[MAC-9dai]]
- Blocks: [[MAC-4ah8]], [[MAC-p46v]], [[MAC-qax4]]

## Comments
