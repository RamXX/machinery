---
id: MAC-r3b0
title: "Project Modelith derived entities/attributes; WRITES to derived is a finding; derived entities need no placement row"
status: open
priority: 2
type: feature
labels: [modelith, projection, simplification]
parent: MAC-0p5d
created_at: 2026-09-25T20:11:08Z
created_by: ramirosalas
updated_at: 2026-09-25T20:11:08Z
content_hash: "sha256:c525817c292ed45c291be12580ce175e8d556be089cdbd18041bb5d5b5e8424c"
blocked_by: [MAC-g7d4]
---

## Description
Use Modelith derived entities/attributes to simplify machinery. Derived attributes existed in 0.4 but machinery never projected them; 0.5 adds derived entities.

Acceptance criteria:
1. Projection v2 emits attr_derived and entity_derived facts.
2. A WRITES{} naming a derived fact is a finding (derived values are computed, not written).
3. A derived entity no longer owes a persistence/placement row (gates.go:2242), auto-waived with the reason shown.
4. Row-local `derived:` waivers remain for non-model facts (log signals); docs say when to use which.

## Acceptance Criteria


## Design


## Notes


## History
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4

## Links
- Parent: [[MAC-0p5d]]
- Blocked by: [[MAC-g7d4]]

## Comments
