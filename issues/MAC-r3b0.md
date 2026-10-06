---
id: MAC-r3b0
title: "Project Modelith derived entities/attributes; WRITES to derived is a finding; derived entities need no placement row"
status: open
priority: 3
type: feature
labels: [modelith, projection, simplification]
parent: MAC-0p5d
created_at: 2026-09-25T20:11:08Z
created_by: ramirosalas
updated_at: 2026-10-06T04:03:57Z
content_hash: "sha256:06c50831ff19626c69f53f07e72409fadaa73e9a4d098026356c8d9e6e6d1c5a"
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
Revalidated 2026-10-06 against v0.11.0: valid. Remaining: Project derived entities/attributes in v2, make WRITES to a derived fact a finding, auto-waive placement rows for derived entities, document derived: waiver split. Evidence: No attr_derived or entity_derived facts anywhere in code or schemas (grep). Placement check in internal/gates/gates.go:~2242 still requires a row per entity. Notes: Adds new fact relations and a rule; affects projection schema/Datalog parity, so more than cosmetic simplification.

## History
- 2026-09-25T20:11:08Z dep_added: blocked_by MAC-g7d4

## Links
- Parent: [[MAC-0p5d]]
- Blocked by: [[MAC-g7d4]]

## Comments
