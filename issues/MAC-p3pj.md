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
updated_at: 2026-09-25T19:53:32Z
content_hash: "sha256:8384e56ded143cde1452903bb8c5134eaadfb272b1ccfa4a6a1f11f1921bdf6c"
blocks: [MAC-4ah8, MAC-p46v]
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


## History
- 2026-09-25T19:53:33Z dep_added: blocks MAC-4ah8
- 2026-09-25T19:53:33Z dep_added: blocks MAC-p46v

## Links
- Parent: [[MAC-9dai]]
- Blocks: [[MAC-4ah8]], [[MAC-p46v]]

## Comments
