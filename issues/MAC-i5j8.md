---
id: MAC-i5j8
title: "Gl undeclared-fact warning fires on action names, qualified unit names, enum members and file names"
status: open
priority: 1
type: bug
labels: [consistency-layer, gl, warnings, h2]
created_at: 2026-09-23T23:56:54Z
created_by: ramirosalas
updated_at: 2026-09-23T23:56:54Z
content_hash: "sha256:15be40ac8f569e3568a2a4e8a32d9d81c94dcc250bdd990b47b2a4909530de69"
---

## Description
v0.10.0 on H2: 428 Gl warnings over 66 matrices. Classified against H2's own vocabularies: 209 name real model attributes (the warning working as designed), 73 model actions (Entity.action), 15 qualified named units (Machine.unit), 49 enum members, 82 none of those (prose words, derived values, and file names: ARCHITECTURE.md matches the Entity.attr regex). Fix: before warning, resolve the token against model actions, named units (bare and machine-qualified), enum members (and their snake_case form), machine context keys, event names, invariant ids, and design file names; warn only on tokens that resolve to a model attribute or to nothing, and word the two cases differently (declare it in USES/WRITES, versus drop the backticks). Consider a per-file summary line instead of one warning per token when a design has not migrated at all.

## Acceptance Criteria


## Design


## Notes


## History


## Links


## Comments
